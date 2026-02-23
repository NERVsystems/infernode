implement LuciBridge;

#
# lucibridge - Connects Lucifer UI to Veltro agent via llm9p
#
# Reads human messages from /mnt/ui/activity/{id}/conversation/input
# (blocking read), runs the Veltro agent loop (LLM + tools), and writes
# responses and tool activity back to the UI as role=veltro messages.
#
# Usage: lucibridge [-v] [-n maxsteps] [-a actid]
#   -v            verbose logging
#   -n steps      max agent steps per turn (default: 20)
#   -a id         activity ID (default: 0)
#
# Prerequisites:
#   - luciuisrv running (serves /mnt/ui/)
#   - llm9p mounted at /n/llm/
#   - tools9p running (serves /tool/) — optional but enables tool use
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "string.m";
	str: String;

include "arg.m";
	arg: Arg;

include "agentlib.m";
	agentlib: AgentLib;

LuciBridge: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

DEFAULT_MAX_STEPS: con 20;
MAX_MAX_STEPS: con 100;

verbose := 0;
maxsteps := DEFAULT_MAX_STEPS;
stderr: ref Sys->FD;

# LLM session state
sessionid := "";
llmfd: ref Sys->FD;

# Activity state
actid := 0;

BRIDGE_SUFFIX: con "\n\nYou are the AI assistant in a Lucifer activity. " +
	"The user sends messages through the UI. " +
	"Respond conversationally using 'say' for dialogue, questions, and greetings. " +
	"Use tools when the user asks you to do something. Say DONE when finished.";

log(msg: string)
{
	if(verbose)
		sys->fprint(stderr, "lucibridge: %s\n", msg);
}

fatal(msg: string)
{
	sys->fprint(stderr, "lucibridge: %s\n", msg);
	raise "fail:" + msg;
}

writefile(path, data: string): int
{
	fd := sys->open(path, Sys->OWRITE);
	if(fd == nil)
		return -1;
	b := array of byte data;
	return sys->write(fd, b, len b);
}

# Read from a blocking fd, strip trailing newline
blockread(fd: ref Sys->FD): string
{
	buf := array[65536] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return nil;
	s := string buf[0:n];
	if(len s > 0 && s[len s - 1] == '\n')
		s = s[0:len s - 1];
	return s;
}

# Write a message to the activity conversation
writemsg(role, text: string)
{
	path := sys->sprint("/mnt/ui/activity/%d/conversation/ctl", actid);
	msg := "role=" + role + " text=" + text;
	if(writefile(path, msg) < 0)
		sys->fprint(stderr, "lucibridge: write to %s failed: %r\n", path);
}

# Set activity status
setstatus(status: string)
{
	path := sys->sprint("/mnt/ui/activity/%d/status", actid);
	writefile(path, status);
}

# Create LLM session with system prompt
initsession(): string
{
	sessionid = agentlib->createsession();
	if(sessionid == "")
		return "cannot create LLM session";

	# Set prefill so LLM stays in character
	prefillpath := "/n/llm/" + sessionid + "/prefill";
	agentlib->setprefillpath(prefillpath, "[Veltro]\n");

	# Build system prompt from namespace discovery
	ns := agentlib->discovernamespace();
	sysprompt := agentlib->buildsystemprompt(ns);

	# Append bridge suffix, truncating base if needed
	MAXWRITE: con 8000;
	suffixbytes := array of byte BRIDGE_SUFFIX;
	basebytes := array of byte sysprompt;
	if(len basebytes + len suffixbytes > MAXWRITE) {
		room := MAXWRITE - len suffixbytes;
		if(room < 0)
			room = 0;
		sysprompt = string basebytes[0:room];
	}
	sysprompt += BRIDGE_SUFFIX;

	systempath := "/n/llm/" + sessionid + "/system";
	agentlib->setsystemprompt(systempath, sysprompt);

	askpath := "/n/llm/" + sessionid + "/ask";
	llmfd = sys->open(askpath, Sys->ORDWR);
	if(llmfd == nil)
		return sys->sprint("cannot open %s: %r", askpath);

	log(sys->sprint("session %s, prompt %d bytes", sessionid, len array of byte sysprompt));
	return nil;
}

# Run the agent loop for one human turn.
# Mirrors repl.b:termagent — query LLM, parse tool, execute, repeat.
agentturn(input: string)
{
	hastools := agentlib->pathexists("/tool");
	ns := "";
	if(hastools)
		ns = agentlib->discovernamespace();

	prompt: string;
	if(hastools)
		prompt = input + "\n\n== Your Namespace ==\n" + ns +
			"\n\nRespond with a tool invocation or DONE if complete.";
	else
		prompt = input;

	setstatus("working");
	retries := 0;
	for(step := 0; step < maxsteps; step++) {
		log(sys->sprint("step %d", step + 1));

		response := agentlib->queryllmfd(llmfd, prompt);
		if(response == "") {
			writemsg("veltro", "(no response from LLM)");
			break;
		}

		log("llm: " + agentlib->truncate(response, 200));

		if(!hastools) {
			# No tools — just relay the response
			writemsg("veltro", response);
			break;
		}

		(tool, toolargs) := agentlib->parseaction(response);

		if(str->tolower(tool) == "done")
			break;

		if(tool == "") {
			# LLM responded conversationally without say tool
			text := agentlib->stripaction(response);
			if(text != "") {
				writemsg("veltro", text);
				break;
			}
			# Unparseable — retry
			retries++;
			if(retries > 2) {
				writemsg("veltro", "(could not parse LLM response)");
				break;
			}
			prompt = "INVALID OUTPUT. Respond with exactly one tool invocation (tool name as first word) or DONE.";
			continue;
		}

		retries = 0;

		# say tool: deliver text directly to UI
		if(str->tolower(tool) == "say") {
			writemsg("veltro", toolargs);
			# After say, continue — LLM should follow with DONE
			prompt = "Message delivered. Say DONE.";
			continue;
		}

		# Other tools: show activity in UI, execute, feed result back
		writemsg("veltro", sys->sprint("[%s %s]", tool, agentlib->truncate(toolargs, 80)));

		result := agentlib->calltool(tool, toolargs);
		log("tool result: " + agentlib->truncate(result, 200));

		if(len result > AgentLib->STREAM_THRESHOLD) {
			scratchfile := agentlib->writescratch(result, step);
			result = sys->sprint("(output written to %s, %d bytes)", scratchfile, len result);
		}

		if(str->tolower(tool) == "spawn")
			prompt = sys->sprint("Tool %s completed:\n%s\n\nSubagent finished. Report result with say then DONE.", tool, result);
		else
			prompt = sys->sprint("Tool %s returned:\n%s\n\nNext tool invocation or DONE.", tool, result);
	}

	setstatus("idle");
}

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	stderr = sys->fildes(2);

	str = load String String->PATH;
	if(str == nil)
		fatal("cannot load String");

	arg = load Arg Arg->PATH;
	if(arg == nil)
		fatal("cannot load Arg");

	agentlib = load AgentLib AgentLib->PATH;
	if(agentlib == nil)
		fatal("cannot load agentlib: " + AgentLib->PATH);
	agentlib->init();

	arg->init(args);
	while((c := arg->opt()) != 0) {
		case c {
		'v' =>
			verbose = 1;
		'n' =>
			s := arg->arg();
			if(s == nil)
				fatal("-n requires step count");
			(maxsteps, nil) = str->toint(s, 10);
			if(maxsteps < 1)
				maxsteps = 1;
			if(maxsteps > MAX_MAX_STEPS)
				maxsteps = MAX_MAX_STEPS;
		'a' =>
			s := arg->arg();
			if(s == nil)
				fatal("-a requires activity ID");
			(actid, nil) = str->toint(s, 10);
		* =>
			sys->fprint(stderr, "usage: lucibridge [-v] [-n maxsteps] [-a actid]\n");
			raise "fail:usage";
		}
	}

	agentlib->setverbose(verbose);

	# Verify prerequisites
	if(sys->open("/mnt/ui/ctl", Sys->OREAD) == nil)
		fatal("/mnt/ui/ not mounted — start luciuisrv first");
	if(!agentlib->pathexists("/n/llm"))
		fatal("/n/llm/ not mounted — mount llm9p first");

	# Tools are optional — bridge works as simple chat relay without them
	if(agentlib->pathexists("/tool"))
		log("tools available at /tool");
	else
		log("no /tool mount — running in chat-only mode");

	# Create LLM session
	err := initsession();
	if(err != nil)
		fatal(err);

	inputpath := sys->sprint("/mnt/ui/activity/%d/conversation/input", actid);

	log(sys->sprint("ready — activity %d, session %s, max %d steps", actid, sessionid, maxsteps));

	# Main loop: re-open input fd each iteration because 9P offset
	# advances after read, causing subsequent reads to return EOF.
	for(;;) {
		inputfd := sys->open(inputpath, Sys->OREAD);
		if(inputfd == nil)
			fatal("cannot open " + inputpath);
		human := blockread(inputfd);
		inputfd = nil;
		if(human == nil) {
			log("input closed");
			break;
		}
		log("human: " + human);

		# Record human message in UI
		writemsg("human", human);

		# Run agent turn
		agentturn(human);
	}
}
