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

# Large result: chars of preview to include inline before referring to scratch file
TRUNC_PREVIEW: con 2000;

# Task complexity threshold: tasks at or above this length trigger a planning turn
PLAN_TASK_THRESHOLD: con 80;

# Configuration
verbose := 0;

stderr: ref Sys->FD;

usage()
{
	sys->fprint(stderr, "Usage: veltro [-v] <task>\n");
	sys->fprint(stderr, "\nOptions:\n");
	sys->fprint(stderr, "  -v    Verbose output\n");
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

# Decide whether this task warrants a planning turn before the action loop.
# Triggers on long tasks (>= PLAN_TASK_THRESHOLD chars) or known complex keywords.
shouldplan(task: string): int
{
	if(len task >= PLAN_TASK_THRESHOLD)
		return 1;
	lower := str->tolower(task);
	keywords := array[] of {
		"refactor", "implement", "debug", "analyze", "design", "migrate"
	};
	for(i := 0; i < len keywords; i++) {
		if(agentlib->contains(lower, keywords[i]))
			return 1;
	}
	return 0;
}

# Run a single planning-only LLM turn on llmfd.
# Sends system context + task and asks the model to state a plan via say.
# Returns the plan text, or "" if the turn fails or produces nothing useful.
doplanningturn(llmfd: ref Sys->FD, ns, task: string): string
{
	planprompt := agentlib->buildsystemprompt(ns) +
		"\n\n== Task ==\n" + task +
		"\n\nBefore taking any action, use say to state your plan in 3-5 numbered steps.\n" +
		"Do not invoke any other tool yet.";

	if(verbose)
		sys->fprint(stderr, "veltro: planning turn\n");

	planresponse := agentlib->queryllmfd(llmfd, planprompt);
	if(planresponse == "")
		return "";

	if(verbose)
		sys->fprint(stderr, "veltro: plan response: %s\n",
			agentlib->truncate(planresponse, 500));

	# Expect "say <plan>" — extract the plan text from the say invocation
	(tool, plantext) := agentlib->parseaction(planresponse);
	if(str->tolower(tool) == "say" && plantext != "")
		return plantext;

	# Model didn't use say — extract any prose text as the plan
	plantext = agentlib->stripaction(planresponse);
	return plantext;
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

	# Optional planning turn for complex tasks
	plan := "";
	if(shouldplan(task)) {
		plan = doplanningturn(llmfd, ns, task);
		if(verbose && plan != "")
			sys->fprint(stderr, "veltro: plan:\n%s\n", plan);
	}

	# Assemble initial prompt.
	# If a plan was produced, the system context was already sent during the
	# planning turn; just transition into execution.  Otherwise send full prompt.
	prompt: string;
	if(plan != "") {
		prompt = "Plan:\n" + plan +
			"\n\nNow begin execution. Respond with your first tool invocation or DONE if already complete.";
	} else {
		prompt = agentlib->buildsystemprompt(ns) +
			"\n\n== Task ==\n" + task +
			"\n\nRespond with a tool invocation or DONE if complete.";
	}

	# Reused in every continuation prompt; empty string when no plan was made
	planctx := "";
	if(plan != "")
		planctx = "\n\nPlan:\n" + plan;

	retries := 0;
	for(step := 0; ; step++) {
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

		# Feed result back for next iteration, including original task and plan for orientation
		haserr := len result >= 6 && result[0:6] == "error:";
		if(str->tolower(tool) == "spawn") {
			prompt = sys->sprint("Task: %s%s\n\nStep %d. Tool %s completed:\n%s\n\nSubagent finished. Report result with say then DONE.",
				task, planctx, step+1, tool, result);
		} else if(haserr) {
			prompt = sys->sprint("Task: %s%s\n\nStep %d. ERROR: Tool %s failed:\n%s\n\nDo NOT retry the same call. Choose a different approach or DONE if impossible.",
				task, planctx, step+1, tool, result);
		} else {
			prompt = sys->sprint("Task: %s%s\n\nStep %d. Tool %s returned:\n%s\n\nNext tool invocation or DONE.",
				task, planctx, step+1, tool, result);
		}
	}

	if(verbose)
		sys->fprint(stderr, "veltro: completed after %d steps\n", step);
}
