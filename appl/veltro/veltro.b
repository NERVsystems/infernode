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
#   veltro -v "task description"          # verbose mode
#   veltro -r last                        # resume most recent session
#   veltro -r <name>                      # resume named session
#   veltro -r <name> "extra instruction"  # resume + redirect
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

# Session storage: persistent across reboots
SESSION_BASE: con "/usr/inferno/veltro/sessions";

# How many log lines to inject into resume context
LOG_RESUME_LINES: con 15;

# Max chars of tool args / result to record per log entry
LOG_PREVIEW: con 200;

# Default thinking token budget (0 = disabled)
THINK_DEFAULT: con 8000;

# Configuration
verbose := 0;
thinkbudget := 0;

# Active session directory (empty = sessions disabled for this run)
sessiondir := "";

stderr: ref Sys->FD;

usage()
{
	sys->fprint(stderr, "Usage: veltro [-v] [-t] <task>\n");
	sys->fprint(stderr, "       veltro [-v] [-t] -r <name> [extra instruction]\n");
	sys->fprint(stderr, "\nOptions:\n");
	sys->fprint(stderr, "  -v          Verbose output\n");
	sys->fprint(stderr, "  -t          Enable extended thinking (%d token budget)\n", THINK_DEFAULT);
	sys->fprint(stderr, "  -r name     Resume session ('last' = most recent)\n");
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

	resumename := "";
	while((o := arg->opt()) != 0)
		case o {
		'v' =>	verbose = 1;
		't' =>	thinkbudget = THINK_DEFAULT;
		'r' =>	resumename = arg->earg();
		* =>	usage();
		}
	args = arg->argv();

	agentlib->setverbose(verbose);

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

	if(resumename != "") {
		# Resume mode: remaining args become optional extra instruction
		extra := "";
		for(; args != nil; args = tl args) {
			if(extra != "")
				extra += " ";
			extra += hd args;
		}
		runresume(resumename, extra);
	} else {
		if(args == nil)
			usage();
		task := "";
		for(; args != nil; args = tl args) {
			if(task != "")
				task += " ";
			task += hd args;
		}
		runagent(task);
	}
}

# ---- Session management ----

# Derive a URL-safe slug from a task string (max ~30 chars)
makeslug(task: string): string
{
	lower := str->tolower(task);
	slug := "";
	prevhyph := 0;
	for(i := 0; i < len lower; i++) {
		c := lower[i];
		if((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			slug += string c;
			prevhyph = 0;
		} else if(c == ' ' || c == '-' || c == '_') {
			if(!prevhyph && len slug > 0) {
				slug += "-";
				prevhyph = 1;
			}
		}
		if(len slug >= 30)
			break;
	}
	# Trim trailing hyphen
	while(len slug > 0 && slug[len slug - 1] == '-')
		slug = slug[0:len slug - 1];
	if(slug == "")
		slug = "task";
	return slug;
}

# Find a free session name: if base exists, try base-2, base-3, ...
findfreeslug(base: string): string
{
	(ok, nil) := sys->stat(SESSION_BASE + "/" + base);
	if(ok < 0)
		return base;
	for(n := 2; n < 1000; n++) {
		candidate := base + "-" + string n;
		(ok2, nil) := sys->stat(SESSION_BASE + "/" + candidate);
		if(ok2 < 0)
			return candidate;
	}
	return base + "-x";
}

# Create path and all missing parent directories (mkdir -p equivalent)
mkdirall(path: string): string
{
	for(i := 1; i < len path; i++) {
		if(path[i] == '/')
			sys->create(path[0:i], Sys->OREAD, 8r755 | Sys->DMDIR);
	}
	fd := sys->create(path, Sys->OREAD, 8r755 | Sys->DMDIR);
	if(fd == nil) {
		# May already exist as a directory — check
		(ok, d) := sys->stat(path);
		if(ok >= 0 && (d.mode & Sys->DMDIR))
			return nil;
		return sys->sprint("cannot create %s: %r", path);
	}
	fd = nil;
	return nil;
}

# Write string content to a file (create or overwrite)
writefile(path, content: string): string
{
	fd := sys->create(path, Sys->OWRITE, 8r644);
	if(fd == nil)
		return sys->sprint("cannot create %s: %r", path);
	data := array of byte content;
	if(sys->write(fd, data, len data) < 0) {
		fd = nil;
		return sys->sprint("write %s failed: %r", path);
	}
	fd = nil;
	return nil;
}

# Read entire file contents; returns "" silently on error
readfile(path: string): string
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return "";
	content := "";
	buf := array[8192] of byte;
	while((n := sys->read(fd, buf, len buf)) > 0)
		content += string buf[0:n];
	fd = nil;
	return content;
}

# Write a value to /env/name (Inferno environment variable mechanism)
setenv(name, val: string)
{
	fd := sys->create("/env/" + name, Sys->OWRITE, 8r644);
	if(fd == nil)
		return;
	data := array of byte val;
	sys->write(fd, data, len data);
	fd = nil;
}

# Write thinking token budget to the session's thinking file.
# budget <= 0 means no-op (thinking stays disabled).
setthinking(llmsessionid: string, budget: int)
{
	if(budget <= 0)
		return;
	path := "/n/llm/" + llmsessionid + "/thinking";
	fd := sys->open(path, Sys->OWRITE);
	if(fd == nil) {
		if(verbose)
			sys->fprint(stderr, "veltro: cannot open %s: %r\n", path);
		return;
	}
	val := string budget;
	data := array of byte val;
	sys->write(fd, data, len data);
	fd = nil;
	if(verbose)
		sys->fprint(stderr, "veltro: thinking budget: %d tokens\n", budget);
}

# Replace newlines and tabs with spaces (for single-line log entries)
collapsenl(s: string): string
{
	result := "";
	for(i := 0; i < len s; i++) {
		c := s[i];
		if(c == '\n' || c == '\r' || c == '\t')
			result += " ";
		else
			result += string c;
	}
	return result;
}

# Append one step entry to the session log file
appendlog(step: int, tool, toolargs, result: string)
{
	if(sessiondir == "")
		return;

	apreview := toolargs;
	if(len apreview > LOG_PREVIEW)
		apreview = apreview[0:LOG_PREVIEW] + "...";
	rpreview := result;
	if(len rpreview > LOG_PREVIEW)
		rpreview = rpreview[0:LOG_PREVIEW] + "...";

	line := sys->sprint("step %d: %s %s -> %s\n",
		step, tool, collapsenl(apreview), collapsenl(rpreview));

	logpath := sessiondir + "/log";
	fd := sys->open(logpath, Sys->OWRITE);
	if(fd == nil)
		fd = sys->create(logpath, Sys->OWRITE, 8r644);
	if(fd == nil)
		return;
	sys->seek(fd, big 0, 2);	# append to end
	data := array of byte line;
	sys->write(fd, data, len data);
	fd = nil;
}

# Resolve "last" session name from the pointer file
resolvelast(): string
{
	name := readfile(SESSION_BASE + "/last");
	# Trim whitespace
	i := 0;
	while(i < len name && (name[i] == ' ' || name[i] == '\n' || name[i] == '\r'))
		i++;
	j := len name;
	while(j > i && (name[j-1] == ' ' || name[j-1] == '\n' || name[j-1] == '\r'))
		j--;
	if(i >= j)
		return "";
	return name[i:j];
}

# Extract the last n lines from log content (oldest-first chronological order)
loglines(logcontent: string, n: int): string
{
	if(logcontent == "")
		return "";

	# Parse all lines; build newest-first list by prepending
	newest: list of string;
	nc := len logcontent;
	i := 0;
	while(i < nc) {
		j := i;
		while(j < nc && logcontent[j] != '\n')
			j++;
		if(j > i)
			newest = logcontent[i:j] :: newest;
		i = j + 1;
	}

	# Reverse back to oldest-first
	oldest: list of string;
	l: list of string;
	for(l = newest; l != nil; l = tl l)
		oldest = hd l :: oldest;

	# Count total lines
	total := 0;
	for(l = oldest; l != nil; l = tl l)
		total++;

	# Skip lines before the last n
	skip := total - n;
	if(skip < 0)
		skip = 0;

	result := "";
	cnt := 0;
	for(l = oldest; l != nil; l = tl l) {
		if(cnt >= skip) {
			if(result != "")
				result += "\n";
			result += hd l;
		}
		cnt++;
	}
	return result;
}

# Strip trailing whitespace/newlines from s
trimright(s: string): string
{
	j := len s;
	while(j > 0 && (s[j-1] == ' ' || s[j-1] == '\n' || s[j-1] == '\r' || s[j-1] == '\t'))
		j--;
	return s[0:j];
}

# Build the initial prompt for a resumed session
buildresumecontext(task, plan, logcontent, extra, ns: string): string
{
	# Count total steps from log line count
	nsteps := 0;
	for(i := 0; i < len logcontent; i++) {
		if(logcontent[i] == '\n')
			nsteps++;
	}

	ctx := agentlib->buildsystemprompt(ns) +
		"\n\n== Resuming Task ==\n" + task;

	if(plan != "")
		ctx += "\n\nPlan:\n" + plan;

	if(nsteps > 0) {
		ctx += sys->sprint("\n\nPrevious steps (%d total). Recent actions:\n", nsteps);
		ctx += loglines(logcontent, LOG_RESUME_LINES);
	}

	# Include todo state if the session has one
	todostate := readfile(sessiondir + "/todo.txt");
	if(todostate != "")
		ctx += "\n\nCurrent todo list:\n" + todostate;

	if(extra != "")
		ctx += "\n\nAdditional instruction: " + extra;

	ctx += "\n\nContinue the task. Next tool invocation or DONE.";
	return ctx;
}

# ---- Planning ----

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

# Run a single planning-only LLM turn.
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

# ---- Context compaction ----

# Threshold: compact when estimated context exceeds 75% of 200K limit
COMPACT_THRESHOLD: con 150000;

# checkandcompact reads /n/llm/N/usage and triggers compaction if needed.
# The usage file returns "estimated_tokens/context_limit\n".
# A write to /n/llm/N/compact triggers the summarisation LLM call.
checkandcompact(llmsessionid: string)
{
	usagepath := "/n/llm/" + llmsessionid + "/usage";
	s := readfile(usagepath);
	if(s == "")
		return;
	# s is "estimated/limit\n" — extract the numerator
	n := 0;
	for(i := 0; i < len s && s[i] >= '0' && s[i] <= '9'; i++)
		n = n * 10 + (s[i] - '0');
	if(n < COMPACT_THRESHOLD)
		return;
	if(verbose)
		sys->fprint(stderr, "veltro: context at ~%d tokens, compacting session\n", n);
	compactpath := "/n/llm/" + llmsessionid + "/compact";
	err := writefile(compactpath, "compact");
	if(err != nil)
		sys->fprint(stderr, "veltro: compaction failed: %s\n", err);
	else if(verbose)
		sys->fprint(stderr, "veltro: session compacted\n");
}

# ---- Core action loop (shared by runagent and runresume) ----

agentloop(llmfd: ref Sys->FD, llmsessionid, task, planctx, initialprompt: string)
{
	prompt := initialprompt;
	retries := 0;
	step := 0;
	for(; ; step++) {
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

		# Check context window; compact if approaching limit
		if(llmsessionid != "")
			checkandcompact(llmsessionid);

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

		# Log step before truncation (so log has the real result preview)
		appendlog(step + 1, tool, toolargs, result);

		# Check for large result — write to scratch, include preview inline
		if(len result > AgentLib->STREAM_THRESHOLD) {
			scratchfile := agentlib->writescratch(result, step);
			trunc := result[0:TRUNC_PREVIEW];
			result = sys->sprint("[TRUNCATED — first %d of %d chars]\n%s\n\nFull output: read %s",
				TRUNC_PREVIEW, len result, trunc, scratchfile);
		}

		# Feed result back for next iteration
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

# ---- New session ----

runagent(task: string)
{
	if(verbose)
		sys->fprint(stderr, "veltro: starting with task: %s\n", task);

	# Create session directory and set environment
	slug := findfreeslug(makeslug(task));
	sdir := SESSION_BASE + "/" + slug;
	if(mkdirall(sdir) != nil) {
		sys->fprint(stderr, "veltro: warning: cannot create session dir — session not saved\n");
		sdir = "";
	}
	if(sdir != "") {
		writefile(sdir + "/task", task);
		writefile(SESSION_BASE + "/last", slug);
		setenv("VELTRO_SESSION", sdir);
		sessiondir = sdir;
		sys->fprint(stderr, "veltro: session %s\n", slug);
	}

	# Create LLM session — clone pattern: read /n/llm/new returns session ID
	llmsessionid := agentlib->createsession();
	if(llmsessionid == "") {
		sys->fprint(stderr, "veltro: cannot create LLM session\n");
		return;
	}
	if(verbose)
		sys->fprint(stderr, "veltro: llm session %s\n", llmsessionid);

	# Set prefill to keep model in character
	prefillpath := "/n/llm/" + llmsessionid + "/prefill";
	agentlib->setprefillpath(prefillpath, "[Veltro]\n");
	setthinking(llmsessionid, thinkbudget);

	# Open session's ask file
	askpath := "/n/llm/" + llmsessionid + "/ask";
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

	# Save plan to session directory
	if(sdir != "" && plan != "")
		writefile(sdir + "/plan", plan);

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

	planctx := "";
	if(plan != "")
		planctx = "\n\nPlan:\n" + plan;

	agentloop(llmfd, llmsessionid, task, planctx, prompt);
}

# ---- Resume session ----

runresume(name, extra: string)
{
	# Resolve "last" to actual session name
	actualname := name;
	if(name == "last") {
		actualname = resolvelast();
		if(actualname == "") {
			sys->fprint(stderr, "veltro: no previous session found\n");
			return;
		}
	}

	sdir := SESSION_BASE + "/" + actualname;
	task := trimright(readfile(sdir + "/task"));
	if(task == "") {
		sys->fprint(stderr, "veltro: session '%s' not found\n", actualname);
		return;
	}

	plan := trimright(readfile(sdir + "/plan"));
	logcontent := readfile(sdir + "/log");

	# Restore session context
	sessiondir = sdir;
	setenv("VELTRO_SESSION", sdir);
	writefile(SESSION_BASE + "/last", actualname);

	sys->fprint(stderr, "veltro: resuming session %s\n", actualname);
	if(extra != "" && verbose)
		sys->fprint(stderr, "veltro: extra instruction: %s\n", extra);

	# Create new LLM session
	llmsessionid := agentlib->createsession();
	if(llmsessionid == "") {
		sys->fprint(stderr, "veltro: cannot create LLM session\n");
		return;
	}

	prefillpath := "/n/llm/" + llmsessionid + "/prefill";
	agentlib->setprefillpath(prefillpath, "[Veltro]\n");
	setthinking(llmsessionid, thinkbudget);

	askpath := "/n/llm/" + llmsessionid + "/ask";
	llmfd := sys->open(askpath, Sys->ORDWR);
	if(llmfd == nil) {
		sys->fprint(stderr, "veltro: cannot open %s: %r\n", askpath);
		return;
	}

	ns := agentlib->discovernamespace();
	if(verbose)
		sys->fprint(stderr, "veltro: namespace:\n%s\n", ns);

	# Build resume context as the initial prompt
	prompt := buildresumecontext(task, plan, logcontent, extra, ns);

	planctx := "";
	if(plan != "")
		planctx = "\n\nPlan:\n" + plan;

	agentloop(llmfd, llmsessionid, task, planctx, prompt);
}
