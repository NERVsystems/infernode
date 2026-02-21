implement Veltro;

#
# veltro - Veltro Agent Loop
#
# A minimal agent where namespace IS the capability system.
#
# Design principles:
#   - Namespace = capability (constructed, not filtered)
#   - Agent operates freely within its world
#   - Everything visible is usable
#
# Usage:
#   veltro "task description"
#   veltro -v "task description"     # verbose mode
#   veltro -n 100 "task description" # max 100 steps
#
# Requires:
#   - /tool mounted (via tools9p)
#   - /n/llm mounted (LLM interface)
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "arg.m";

include "string.m";
	str: String;

include "nsconstruct.m";
	nsconstruct: NsConstruct;

include "agentlib.m";
	agentlib: AgentLib;

Veltro: module {
	init: fn(ctxt: ref Draw->Context, argv: list of string);
};

# Defaults and limits
DEFAULT_MAX_STEPS: con 50;
MAX_MAX_STEPS: con 100;

# Large result: chars of preview to include inline before referring to scratch file
TRUNC_PREVIEW: con 2000;

# Configuration
verbose := 0;
maxsteps := DEFAULT_MAX_STEPS;

stderr: ref Sys->FD;

usage()
{
	sys->fprint(stderr, "Usage: veltro [-v] [-n maxsteps] <task>\n");
	sys->fprint(stderr, "\nOptions:\n");
	sys->fprint(stderr, "  -v          Verbose output\n");
	sys->fprint(stderr, "  -n steps    Maximum steps (default: %d, max: %d)\n",
		DEFAULT_MAX_STEPS, MAX_MAX_STEPS);
	sys->fprint(stderr, "\nRequires /tool and /n/llm to be mounted.\n");
	raise "fail:usage";
}

nomod(s: string)
{
	sys->fprint(stderr, "veltro: can't load %s: %r\n", s);
	raise "fail:load";
}

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	stderr = sys->fildes(2);

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
	args = arg->argv();
	arg = nil;

	if(args == nil)
		usage();

	agentlib->setverbose(verbose);

	# Join remaining args as task
	task := "";
	for(; args != nil; args = tl args) {
		if(task != "")
			task += " ";
		task += hd args;
	}

	# Check required mounts
	if(!agentlib->pathexists("/tool"))
		sys->fprint(stderr, "warning: /tool not mounted (run tools9p first)\n");
	if(!agentlib->pathexists("/n/llm"))
		sys->fprint(stderr, "warning: /n/llm not mounted (LLM unavailable)\n");

	# Namespace restriction (v3): FORKNS + bind-replace
	# Load nsconstruct module (must happen while /dis is unrestricted)
	nsconstruct = load NsConstruct NsConstruct->PATH;
	if(nsconstruct != nil) {
		nsconstruct->init();

		# Fork namespace so caller is unaffected
		sys->pctl(Sys->FORKNS, nil);

		parent_caps := ref NsConstruct->Capabilities(
			nil, nil, nil, nil, nil, nil, 0, 0
		);

		# Apply namespace restrictions
		nserr := nsconstruct->restrictns(parent_caps);
		if(nserr != nil)
			sys->fprint(stderr, "veltro: namespace restriction failed: %s\n", nserr);
		else if(verbose)
			sys->fprint(stderr, "veltro: namespace restricted\n");
	}

	# Run agent
	runagent(task);
}

# Main agent loop
runagent(task: string)
{
	if(verbose)
		sys->fprint(stderr, "veltro: starting with task: %s\n", task);

	# Create LLM session — clone pattern: read /n/llm/new returns session ID
	sessionid := agentlib->createsession();
	if(sessionid == "") {
		sys->fprint(stderr, "veltro: cannot create LLM session\n");
		return;
	}
	if(verbose)
		sys->fprint(stderr, "veltro: session %s\n", sessionid);

	# Set prefill to keep model in character
	prefillpath := "/n/llm/" + sessionid + "/prefill";
	agentlib->setprefillpath(prefillpath, "[Veltro]\n");

	# Open session's ask file
	askpath := "/n/llm/" + sessionid + "/ask";
	llmfd := sys->open(askpath, Sys->ORDWR);
	if(llmfd == nil) {
		sys->fprint(stderr, "veltro: cannot open %s: %r\n", askpath);
		return;
	}

	# Discover namespace — this IS our capability set
	ns := agentlib->discovernamespace();
	if(verbose)
		sys->fprint(stderr, "veltro: namespace:\n%s\n", ns);

	# Assemble initial prompt: system + namespace + task (single write to ask)
	prompt := agentlib->buildsystemprompt(ns) +
		"\n\n== Task ==\n" + task +
		"\n\nRespond with a tool invocation or DONE if complete.";

	retries := 0;
	for(step := 0; step < maxsteps; step++) {
		if(verbose)
			sys->fprint(stderr, "veltro: step %d\n", step + 1);

		# Query LLM using persistent fd for conversation history
		response := agentlib->queryllmfd(llmfd, prompt);
		if(response == "") {
			sys->fprint(stderr, "veltro: LLM returned empty response\n");
			break;
		}

		if(verbose)
			sys->fprint(stderr, "veltro: LLM: %s\n", response);

		# Parse action from response
		(tool, toolargs) := agentlib->parseaction(response);

		# Check for completion
		if(str->tolower(tool) == "done") {
			if(verbose)
				sys->fprint(stderr, "veltro: task completed\n");
			break;
		}

		# No tool found — LLM output conversational text; retry
		if(tool == "") {
			retries++;
			if(retries > 2)
				break;
			prompt = "INVALID OUTPUT. Respond with exactly one tool invocation (tool name as first word) or DONE.";
			continue;
		}

		retries = 0;

		if(verbose)
			sys->fprint(stderr, "veltro: tool=%s args=%s\n", tool, toolargs);

		# Execute tool
		result := agentlib->calltool(tool, toolargs);

		if(verbose)
			sys->fprint(stderr, "veltro: result: %s\n", agentlib->truncate(result, 500));

		# Check for large result — write to scratch, include preview inline
		if(len result > AgentLib->STREAM_THRESHOLD) {
			scratchfile := agentlib->writescratch(result, step);
			trunc := result[0:TRUNC_PREVIEW];
			result = sys->sprint("[TRUNCATED — first %d of %d chars]\n%s\n\nFull output: read %s",
				TRUNC_PREVIEW, len result, trunc, scratchfile);
		}

		# Feed result back for next iteration, always including original task for orientation
		haserr := len result >= 6 && result[0:6] == "error:";
		if(str->tolower(tool) == "spawn") {
			prompt = sys->sprint("Task: %s\n\nStep %d/%d. Tool %s completed:\n%s\n\nSubagent finished. Report result with say then DONE.",
				task, step+1, maxsteps, tool, result);
		} else if(haserr) {
			prompt = sys->sprint("Task: %s\n\nStep %d/%d. ERROR: Tool %s failed:\n%s\n\nDo NOT retry the same call. Choose a different approach or DONE if impossible.",
				task, step+1, maxsteps, tool, result);
		} else {
			prompt = sys->sprint("Task: %s\n\nStep %d/%d. Tool %s returned:\n%s\n\nNext tool invocation or DONE.",
				task, step+1, maxsteps, tool, result);
		}
	}

	if(verbose && maxsteps > 0)
		sys->fprint(stderr, "veltro: completed (max steps: %d)\n", maxsteps);
}
