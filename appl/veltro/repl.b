implement Repl;

#
# repl - Interactive Veltro Agent Chat
#
# A REPL where users have ongoing conversations with Veltro.
# Works in two modes:
#   - Xenith mode: window with Send/Clear/Reset/Delete tag buttons
#   - Terminal mode: line-oriented stdin/stdout when Xenith unavailable
#
# Usage:
#   repl [-v] [-n maxsteps]
#
# Requires:
#   - /tool mounted (via tools9p)
#   - /n/llm mounted (LLM interface)
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "bufio.m";
	bufio: Bufio;
	Iobuf: import bufio;

include "arg.m";

include "string.m";
	str: String;

include "nsconstruct.m";
	nsconstruct: NsConstruct;

include "xenithwin.m";
	xenithwin: Xenithwin;
	Win, Event: import xenithwin;

include "agentlib.m";
	agentlib: AgentLib;

Repl: module {
	init: fn(ctxt: ref Draw->Context, argv: list of string);
};

# Defaults
DEFAULT_MAX_STEPS: con 50;
MAX_MAX_STEPS: con 100;

# Configuration
verbose := 0;
maxsteps := DEFAULT_MAX_STEPS;

stderr: ref Sys->FD;

# LLM session state
sessionid := "";
llmfd: ref Sys->FD;

# Window state (Xenith mode only)
w: ref Win;
hostpt := 0;
busy := 0;

usage()
{
	sys->fprint(stderr, "Usage: repl [-v] [-n maxsteps]\n");
	sys->fprint(stderr, "\nOptions:\n");
	sys->fprint(stderr, "  -v          Verbose output\n");
	sys->fprint(stderr, "  -n steps    Maximum steps per turn (default: %d, max: %d)\n",
		DEFAULT_MAX_STEPS, MAX_MAX_STEPS);
	sys->fprint(stderr, "\nRequires /tool and /n/llm to be mounted.\n");
	raise "fail:usage";
}

nomod(s: string)
{
	sys->fprint(stderr, "repl: can't load %s: %r\n", s);
	raise "fail:load";
}

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	stderr = sys->fildes(2);

	bufio = load Bufio Bufio->PATH;
	if(bufio == nil)
		nomod(Bufio->PATH);

	str = load String String->PATH;
	if(str == nil)
		nomod(String->PATH);

	agentlib = load AgentLib AgentLib->PATH;
	if(agentlib == nil)
		nomod(AgentLib->PATH);
	agentlib->init();

	arg := load Arg Arg->PATH;
	if(arg == nil)
		nomod(Arg->PATH);
	arg->init(args);

	while((o := arg->opt()) != 0)
		case o {
		'v' =>	verbose = 1;
		'n' =>
			n := int arg->earg();
			if(n < 1)
				n = 1;
			if(n > MAX_MAX_STEPS)
				n = MAX_MAX_STEPS;
			maxsteps = n;
		* =>	usage();
		}
	arg = nil;

	agentlib->setverbose(verbose);

	# Check required mounts
	if(!agentlib->pathexists("/tool")) {
		sys->fprint(stderr, "repl: /tool not mounted (run tools9p first)\n");
		raise "fail:no /tool";
	}
	if(!agentlib->pathexists("/n/llm")) {
		sys->fprint(stderr, "repl: /n/llm not mounted\n");
		raise "fail:no /n/llm";
	}

	# Detect Xenith and create window BEFORE namespace restriction.
	# /chan (Xenith 9P) exposes ALL window contents — after restriction
	# it must be hidden. Open FDs before restriction so they survive.
	xmode := 0;
	if(xenithavail()) {
		xenithwin = load Xenithwin Xenithwin->PATH;
		if(xenithwin == nil)
			nomod(Xenithwin->PATH);
		xenithwin->init();
		w = Win.wnew();              # opens ctl, event via /chan
		w.wname("/+Veltro");
		w.wtagwrite(" Send Voice Clear Reset Delete");  # opens tag transiently
		# Eagerly open addr and data — used later by wreplace, wread, readinput.
		# After restriction /chan is gone, but these FDs persist.
		w.addr = w.openfile("addr");
		w.data = w.openfile("data");
		xmode = 1;
	}

	# Namespace restriction (v3): FORKNS + bind-replace
	# Must happen after mount checks and Xenith window creation,
	# but before session creation
	nsconstruct = load NsConstruct NsConstruct->PATH;
	if(nsconstruct != nil) {
		nsconstruct->init();
		sys->pctl(Sys->FORKNS, nil);

		caps := ref NsConstruct->Capabilities(
			nil, nil, nil, nil, nil, nil, 0, 0
		);

		nserr := nsconstruct->restrictns(caps);
		if(nserr != nil)
			sys->fprint(stderr, "repl: namespace restriction failed: %s\n", nserr);
		else if(verbose)
			sys->fprint(stderr, "repl: namespace restricted\n");
	}

	# Create LLM session
	initsession();

	# Enter mode — window already created if Xenith available
	if(xmode)
		xenithmode();
	else
		termmode();
}

# Check if Xenith window system is available.
# Use stat on /chan/index — do NOT open /chan/new/ctl, as that creates a window.
xenithavail(): int
{
	(ok, nil) := sys->stat("/chan/index");
	return ok >= 0;
}

# Create LLM session and set up system prompt
initsession()
{
	sessionid = agentlib->createsession();
	if(sessionid == "") {
		sys->fprint(stderr, "repl: cannot create LLM session\n");
		raise "fail:no LLM session";
	}

	prefillpath := "/n/llm/" + sessionid + "/prefill";
	agentlib->setprefillpath(prefillpath, "[Veltro]\n");

	ns := agentlib->discovernamespace();
	sysprompt := agentlib->buildsystemprompt(ns) +
		"\n\nYou are in interactive REPL mode. The user will send messages. " +
		"Each response must be exactly one tool invocation or DONE. No other output.";

	if(verbose) {
		sys->fprint(stderr, "repl: session %s\n", sessionid);
		sys->fprint(stderr, "repl: system prompt: %d bytes\n", len array of byte sysprompt);
		sys->fprint(stderr, "repl: namespace:\n%s\n", ns);
	}

	systempath := "/n/llm/" + sessionid + "/system";
	agentlib->setsystemprompt(systempath, sysprompt);

	askpath := "/n/llm/" + sessionid + "/ask";
	llmfd = sys->open(askpath, Sys->ORDWR);
	if(llmfd == nil) {
		sys->fprint(stderr, "repl: cannot open %s: %r\n", askpath);
		raise "fail:open ask";
	}
}

#
# ==================== Terminal Mode ====================
#

termmode()
{
	sys->print("Veltro REPL (terminal mode)\n");
	sys->print("Type a message, or /voice to speak. /quit to exit, /reset for new session.\n\n");

	stdin := bufio->fopen(sys->fildes(0), Sys->OREAD);
	if(stdin == nil) {
		sys->fprint(stderr, "repl: cannot open stdin\n");
		raise "fail:stdin";
	}

	for(;;) {
		sys->print("veltro> ");

		line := stdin.gets('\n');
		if(line == nil)
			break;

		# Strip trailing newline
		if(len line > 0 && line[len line - 1] == '\n')
			line = line[:len line - 1];
		line = agentlib->strip(line);
		if(line == "")
			continue;

		# Commands
		if(line == "/quit" || line == "/exit")
			break;
		if(line == "/reset") {
			termreset();
			continue;
		}
		if(line == "/clear") {
			sys->print("\n");
			continue;
		}
		if(line == "/voice" || line == "/v") {
			voiceline := voiceinput();
			if(voiceline != "") {
				sys->print("> %s\n", voiceline);
				termagent(voiceline);
			}
			continue;
		}

		# Run agent synchronously
		termagent(line);
	}

	sys->print("\n");
}

termreset()
{
	sessionid = agentlib->createsession();
	if(sessionid == "") {
		sys->print("[error: cannot create new LLM session]\n");
		return;
	}

	prefillpath := "/n/llm/" + sessionid + "/prefill";
	agentlib->setprefillpath(prefillpath, "[Veltro]\n");

	ns := agentlib->discovernamespace();
	sysprompt := agentlib->buildsystemprompt(ns) +
		"\n\nYou are in interactive REPL mode. The user will send messages. " +
		"Each response must be exactly one tool invocation or DONE. No other output.";

	systempath := "/n/llm/" + sessionid + "/system";
	agentlib->setsystemprompt(systempath, sysprompt);

	askpath := "/n/llm/" + sessionid + "/ask";
	llmfd = sys->open(askpath, Sys->ORDWR);
	if(llmfd == nil) {
		sys->print("[error: cannot open LLM session]\n");
		return;
	}

	sys->print("[session reset]\n\n");
}

termagent(input: string)
{
	ns := agentlib->discovernamespace();
	prompt := input + "\n\n== Your Namespace ==\n" + ns +
		"\n\nRespond with a tool invocation or DONE if complete.";

	retries := 0;
	for(step := 0; step < maxsteps; step++) {
		if(verbose)
			sys->fprint(stderr, "repl: step %d\n", step + 1);

		sys->print("[thinking...]\n");

		response := agentlib->queryllmfd(llmfd, prompt);
		if(response == "") {
			sys->print("[error: LLM returned empty response]\n\n");
			return;
		}

		if(verbose)
			sys->fprint(stderr, "repl: LLM: %s\n", response);

		(tool, toolargs) := agentlib->parseaction(response);

		if(str->tolower(tool) == "done") {
			return;
		}

		if(tool == "") {
			retries++;
			if(retries > 2)
				return;
			prompt = "INVALID OUTPUT. Respond with exactly one tool invocation (tool name as first word) or DONE.";
			continue;
		}

		retries = 0;

		# For say, display the full text so user can read it
		if(str->tolower(tool) == "say")
			sys->print("[speaking] %s\n", toolargs);
		else
			sys->print("[%s %s]\n", tool, agentlib->truncate(toolargs, 80));

		result := agentlib->calltool(tool, toolargs);

		if(verbose)
			sys->fprint(stderr, "repl: tool result: %s\n", result);

		if(len result > AgentLib->STREAM_THRESHOLD) {
			scratchfile := agentlib->writescratch(result, step);
			result = sys->sprint("(output written to %s, %d bytes)", scratchfile, len result);
		}

		if(str->tolower(tool) == "spawn")
			prompt = sys->sprint("Tool %s completed:\n%s\n\nSubagent finished. Report result with say then DONE.", tool, result);
		else
			prompt = sys->sprint("Tool %s returned:\n%s\n\nNext tool invocation or DONE.", tool, result);
	}

	sys->print("[max steps reached]\n\n");
}

#
# ==================== Xenith Mode ====================
#

xenithmode()
{
	# Window already created in init() before namespace restriction.
	# FDs (ctl, event, addr, data) are open and survive restriction.
	spawn xmainloop();
}

xmainloop()
{
	c := chan of Event;
	agentout := chan[32] of string;

	spawn w.wslave(c);

	loop: for(;;) alt {
	msg := <-agentout =>
		appendoutput(msg);

	e := <-c =>
		case e.c1 {
		'M' or 'K' =>
			case e.c2 {
			'I' =>
				if(e.q0 < hostpt)
					hostpt += e.q1 - e.q0;
			'D' =>
				if(e.q0 < hostpt) {
					if(hostpt < e.q1)
						hostpt = e.q0;
					else
						hostpt -= e.q1 - e.q0;
				}
			'x' or 'X' =>
				s := getexectext(e, c);
				n := doexec(s, agentout);
				if(n == 0)
					w.wwriteevent(ref e);
				else if(n < 0)
					break loop;
			'l' or 'L' =>
				w.wwriteevent(ref e);
			}
		}
	}
	w.wdel(1);
}

# Extract command text from execute event, consuming secondary/arg events
getexectext(e: Event, c: chan of Event): string
{
	eq := e;
	na := 0;
	ea: Event;

	if(e.flag & 2)
		eq = <-c;
	if(e.flag & 8) {
		ea = <-c;
		na = ea.nb;
		<-c;	# toss
	}

	s: string;
	if(eq.q1 > eq.q0 && eq.nb == 0)
		s = w.wread(eq.q0, eq.q1);
	else
		s = string eq.b[0:eq.nb];
	if(na)
		s += " " + string ea.b[0:ea.nb];
	return s;
}

# Dispatch tag commands. Returns: 1=handled, 0=pass to xenith, -1=exit
doexec(cmd: string, agentout: chan of string): int
{
	cmd = str->drop(cmd, " \t\n");
	(word, nil) := agentlib->splitfirst(cmd);
	case word {
	"Send" =>
		dosend(agentout);
	"Voice" =>
		dovoice(agentout);
	"Clear" =>
		doclear();
	"Reset" =>
		doreset();
	"Del" or "Delete" =>
		return -1;
	* =>
		return 0;
	}
	return 1;
}

# Harvest user input from below hostpt, dispatch to agent
dosend(agentout: chan of string)
{
	if(busy) {
		appendoutput("[busy -- agent is still working]\n");
		return;
	}

	# Read input from hostpt to end of body
	input := readinput();
	input = agentlib->strip(input);
	if(input == "")
		return;

	# Clear the input area
	w.wreplace(sys->sprint("#%d,$", hostpt), "");

	# Echo the user's message
	appendoutput("> " + input + "\n");

	busy = 1;
	spawn xagentthread(input, agentout);
}

# Read body text from hostpt to end
readinput(): string
{
	addr := sys->sprint("#%d,$", hostpt);
	if(!w.wsetaddr(addr, 1))
		return "";

	# Read from the addr/data pair
	if(w.data == nil)
		w.data = w.openfile("data");

	result := "";
	buf := array[4096] of byte;
	for(;;) {
		n := sys->read(w.data, buf, len buf);
		if(n <= 0)
			break;
		result += string buf[0:n];
	}
	return result;
}

# Voice input for Xenith mode: record, transcribe, send to agent
dovoice(agentout: chan of string)
{
	if(busy) {
		appendoutput("[busy -- agent is still working]\n");
		return;
	}

	appendoutput("[listening...]\n");
	input := voiceinput();
	if(input == "")
		return;

	appendoutput("> " + input + "\n");
	busy = 1;
	spawn xagentthread(input, agentout);
}

# Record and transcribe via /n/speech/hear
voiceinput(): string
{
	SPEECH_HEAR: con "/n/speech/hear";

	(ok, nil) := sys->stat(SPEECH_HEAR);
	if(ok < 0) {
		sys->print("[voice: /n/speech not mounted]\n");
		return "";
	}

	sys->print("[recording 5 seconds...]\n");

	fd := sys->open(SPEECH_HEAR, Sys->ORDWR);
	if(fd == nil) {
		sys->print("[voice: cannot open %s: %r]\n", SPEECH_HEAR);
		return "";
	}

	# Write start command to trigger recording
	cmd := array of byte "start 5000";
	sys->write(fd, cmd, len cmd);

	# Read transcription
	sys->seek(fd, big 0, Sys->SEEKSTART);
	result := "";
	buf := array[8192] of byte;
	for(;;) {
		n := sys->read(fd, buf, len buf);
		if(n <= 0)
			break;
		result += string buf[0:n];
	}

	result = agentlib->strip(result);
	if(result == "" || agentlib->hasprefix(result, "error:")) {
		sys->print("[voice: no speech detected]\n");
		return "";
	}

	return result;
}

# Insert text at hostpt, advance hostpt
appendoutput(text: string)
{
	addr := sys->sprint("#%d,#%d", hostpt, hostpt);
	w.wreplace(addr, text);
	hostpt += len text;
	w.ctlwrite("show\n");
}

# Clear body, reset hostpt (keep LLM session)
doclear()
{
	w.wreplace(",", "");
	hostpt = 0;
	w.ctlwrite("clean\n");
}

# Create new LLM session, clear body
doreset()
{
	doclear();

	sessionid = agentlib->createsession();
	if(sessionid == "") {
		appendoutput("[error: cannot create new LLM session]\n");
		return;
	}

	prefillpath := "/n/llm/" + sessionid + "/prefill";
	agentlib->setprefillpath(prefillpath, "[Veltro]\n");

	ns := agentlib->discovernamespace();
	sysprompt := agentlib->buildsystemprompt(ns) +
		"\n\nYou are in interactive REPL mode. The user will send messages. " +
		"Each response must be exactly one tool invocation or DONE. No other output.";

	systempath := "/n/llm/" + sessionid + "/system";
	agentlib->setsystemprompt(systempath, sysprompt);

	askpath := "/n/llm/" + sessionid + "/ask";
	llmfd = sys->open(askpath, Sys->ORDWR);
	if(llmfd == nil) {
		appendoutput("[error: cannot open LLM session]\n");
		return;
	}

	appendoutput("[session reset]\n");
}

# Agent thread for Xenith mode: sends display text on agentout channel
xagentthread(input: string, agentout: chan of string)
{
	{
		xagentsteps(input, agentout);
	} exception {
	* =>
		sys->fprint(stderr, "repl: agent exception\n");
		agentout <-= "[error: agent exception]\n";
	}
	busy = 0;
}

xagentsteps(input: string, agentout: chan of string)
{
	ns := agentlib->discovernamespace();
	prompt := input + "\n\n== Your Namespace ==\n" + ns +
		"\n\nRespond with a tool invocation or DONE if complete.";

	retries := 0;
	for(step := 0; step < maxsteps; step++) {
		if(verbose)
			sys->fprint(stderr, "repl: step %d\n", step + 1);

		agentout <-= "[thinking...]\n";

		response := agentlib->queryllmfd(llmfd, prompt);
		if(response == "") {
			agentout <-= "[error: LLM returned empty response]\n\n";
			return;
		}

		if(verbose)
			sys->fprint(stderr, "repl: LLM: %s\n", response);

		(tool, toolargs) := agentlib->parseaction(response);

		if(str->tolower(tool) == "done") {
			agentout <-= "\n";
			return;
		}

		if(tool == "") {
			retries++;
			if(retries > 2) {
				agentout <-= "\n";
				return;
			}
			prompt = "INVALID OUTPUT. Respond with exactly one tool invocation (tool name as first word) or DONE.";
			continue;
		}

		retries = 0;

		# For say, display the full text so user can read it
		if(str->tolower(tool) == "say")
			agentout <-= "[speaking] " + toolargs + "\n";
		else
			agentout <-= "[" + tool + " " + agentlib->truncate(toolargs, 80) + "]\n";

		result := agentlib->calltool(tool, toolargs);

		if(verbose)
			sys->fprint(stderr, "repl: tool result: %s\n", result);

		if(len result > AgentLib->STREAM_THRESHOLD) {
			scratchfile := agentlib->writescratch(result, step);
			result = sys->sprint("(output written to %s, %d bytes)", scratchfile, len result);
		}

		if(str->tolower(tool) == "spawn")
			prompt = sys->sprint("Tool %s completed:\n%s\n\nSubagent finished. Report result with say then DONE.", tool, result);
		else
			prompt = sys->sprint("Tool %s returned:\n%s\n\nNext tool invocation or DONE.", tool, result);
	}
}
