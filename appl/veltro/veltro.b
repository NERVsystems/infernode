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
#   - ~150 lines of core logic
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

include "bufio.m";
	bufio: Bufio;
	Iobuf: import bufio;

include "arg.m";

include "string.m";
	str: String;

include "nsconstruct.m";
	nsconstruct: NsConstruct;

Veltro: module {
	init: fn(ctxt: ref Draw->Context, argv: list of string);
};

# Defaults and limits
DEFAULT_MAX_STEPS: con 50;
MAX_MAX_STEPS: con 100;
SCRATCH_PATH: con "/tmp/veltro/scratch";
STREAM_THRESHOLD: con 4096;

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

	bufio = load Bufio Bufio->PATH;
	if(bufio == nil)
		nomod(Bufio->PATH);

	str = load String String->PATH;
	if(str == nil)
		nomod(String->PATH);

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

	# Join remaining args as task
	task := "";
	for(; args != nil; args = tl args) {
		if(task != "")
			task += " ";
		task += hd args;
	}

	# Check required mounts
	if(!pathexists("/tool"))
		sys->fprint(stderr, "warning: /tool not mounted (run tools9p first)\n");
	if(!pathexists("/n/llm"))
		sys->fprint(stderr, "warning: /n/llm not mounted (LLM unavailable)\n");

	# Namespace restriction (v3): FORKNS + bind-replace
	# Load nsconstruct module (must happen while /dis is unrestricted)
	nsconstruct = load NsConstruct NsConstruct->PATH;
	if(nsconstruct != nil) {
		nsconstruct->init();

		# Fork namespace so caller is unaffected
		sys->pctl(Sys->FORKNS, nil);

		# Build parent capabilities (tools are served by tools9p, not restricted here)
		parent_caps := ref NsConstruct->Capabilities(
			nil,   # tools — tools9p handles tool access
			nil,   # paths
			nil,   # shellcmds — no shell for parent
			nil,   # llmconfig — /n/llm preserved by stat check in restrictns
			nil,   # fds
			nil,   # mcproviders
			0      # memory
		);

		# Apply namespace restrictions
		nserr := nsconstruct->restrictns(parent_caps);
		if(nserr != nil)
			sys->fprint(stderr, "veltro: namespace restriction failed: %s\n", nserr);

		# Verify restrictions
		nserr = nsconstruct->verifyns("/dis/lib" :: "/dis/veltro" :: "/dev/cons" :: nil);
		if(nserr != nil)
			sys->fprint(stderr, "veltro: namespace verification warning: %s\n", nserr);

		# Emit audit log
		nsconstruct->emitauditlog(
			sys->sprint("parent-%d", sys->pctl(0, nil)),
			"restrictns applied" :: nil);
	}

	# Run agent
	runagent(task);
}

# Main agent loop
runagent(task: string)
{
	if(verbose)
		sys->print("Veltro Agent starting with task: %s\n", task);

	# Create LLM session - clone pattern: read /n/llm/new returns session ID
	sessionid := createsession();
	if(sessionid == "") {
		sys->fprint(stderr, "veltro: cannot create LLM session\n");
		return;
	}
	if(verbose)
		sys->print("Created LLM session: %s\n", sessionid);

	# Build session-specific paths
	askpath := "/n/llm/" + sessionid + "/ask";
	prefillpath := "/n/llm/" + sessionid + "/prefill";
	ctlpath := "/n/llm/" + sessionid + "/ctl";

	# Set prefill to keep model in character
	# Uses newline so tool invocation starts on its own line for parsing
	setprefillpath(prefillpath, "[Veltro]\n");

	# Open session's ask file
	llmfd := sys->open(askpath, Sys->ORDWR);
	if(llmfd == nil) {
		sys->fprint(stderr, "veltro: cannot open %s: %r\n", askpath);
		return;
	}

	# Discover namespace - this IS our capability set
	ns := discovernamespace();
	if(verbose)
		sys->print("Namespace: %s\n", ns);

	# Assemble initial prompt
	prompt := assembleprompt(task, ns);

	retries := 0;
	for(step := 0; step < maxsteps; step++) {
		if(verbose)
			sys->print("\n=== Step %d ===\n", step + 1);

		# Query LLM using persistent fd for conversation history
		response := queryllmfd(llmfd, prompt);
		if(response == "") {
			sys->fprint(stderr, "veltro: LLM returned empty response\n");
			break;
		}

		if(verbose)
			sys->print("LLM: %s\n", response);

		# Parse action from response
		(tool, toolargs) := parseaction(response);

		# Check for completion
		if(str->tolower(tool) == "done") {
			if(verbose)
				sys->print("Task completed.\n");
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
			sys->print("Tool: %s\nArgs: %s\n", tool, toolargs);

		# Execute tool
		result := calltool(tool, toolargs);

		if(verbose)
			sys->print("Result: %s\n", truncate(result, 500));

		# Check for large result - write to scratch
		if(len result > STREAM_THRESHOLD) {
			scratchfile := writescratch(result, step);
			result = sys->sprint("(output written to %s, %d bytes)", scratchfile, len result);
		}

		# Feed result back for next iteration
		if(str->tolower(tool) == "spawn")
			prompt = sys->sprint("Tool %s completed:\n%s\n\nSubagent finished. Report result with say then DONE.", tool, result);
		else
			prompt = sys->sprint("Tool %s returned:\n%s\n\nNext tool invocation or DONE.", tool, result);
	}

	if(verbose && maxsteps > 0)
		sys->print("Agent completed (max steps: %d)\n", maxsteps);
}

# Discover namespace - read /tool/tools and list accessible paths
discovernamespace(): string
{
	result := "TOOLS:\n";

	# Read available tools
	tools := readfile("/tool/tools");
	if(tools != "")
		result += tools;
	else
		result += "(none)";

	# List accessible paths
	result += "\n\nPATHS:\n";
	paths := array[] of {"/", "/tool", "/n", "/tmp"};
	for(i := 0; i < len paths; i++) {
		if(pathexists(paths[i]))
			result += paths[i] + "\n";
	}

	return result;
}

# Assemble system prompt with namespace and task
assembleprompt(task, ns: string): string
{
	# Read base system prompt
	base := readfile("/lib/veltro/system.txt");
	if(base == "")
		base = defaultsystemprompt();

	# Get tool documentation
	tooldocs := "";
	(nil, toollist) := sys->tokenize(readfile("/tool/tools"), "\n");
	for(; toollist != nil; toollist = tl toollist) {
		toolname := hd toollist;
		doc := calltool("help", toolname);
		if(doc != "" && !hasprefix(doc, "error:"))
			tooldocs += "\n### " + toolname + "\n" + doc + "\n";
	}

	# Load context-specific reminders based on available tools
	reminders := loadreminders(toollist);

	prompt := base + "\n\n== Your Namespace ==\n" + ns +
		"\n\n== Tool Documentation ==\n" + tooldocs;

	if(reminders != "")
		prompt += "\n\n== Reminders ==\n" + reminders;

	prompt += "\n\n== Task ==\n" + task +
		"\n\nRespond with a tool invocation or DONE if complete.";

	return prompt;
}

# Load context-specific reminders based on available tools
loadreminders(toollist: list of string): string
{
	reminders := "";

	# Always include inferno shell reminder if exec is available
	for(t := toollist; t != nil; t = tl t) {
		tool := hd t;
		reminderpath := "";

		case tool {
		"exec" =>
			reminderpath = "/lib/veltro/reminders/inferno-shell.txt";
		"git" =>
			reminderpath = "/lib/veltro/reminders/git.txt";
		"xenith" =>
			reminderpath = "/lib/veltro/reminders/xenith.txt";
		"write" or "edit" =>
			reminderpath = "/lib/veltro/reminders/file-modified.txt";
		"spawn" =>
			reminderpath = "/lib/veltro/reminders/security.txt";
		}

		if(reminderpath != "") {
			content := readfile(reminderpath);
			if(content != "" && !contains(reminders, content)) {
				if(reminders != "")
					reminders += "\n\n";
				reminders += content;
			}
		}
	}

	return reminders;
}

# Check if string contains substring
contains(s, sub: string): int
{
	if(len sub > len s)
		return 0;
	for(i := 0; i <= len s - len sub; i++) {
		match := 1;
		for(j := 0; j < len sub; j++) {
			if(s[i+j] != sub[j]) {
				match = 0;
				break;
			}
		}
		if(match)
			return 1;
	}
	return 0;
}

# Default system prompt if file not found
defaultsystemprompt(): string
{
	return "You are a Veltro agent running in Inferno OS.\n\n" +
		"== Core Principle ==\n" +
		"Your namespace IS your capability set. If a tool isn't in /tool, it doesn't exist.\n\n" +
		"== Tool Invocation ==\n" +
		"Output ONE tool per response:\n" +
		"    toolname arguments\n\n" +
		"== MULTI-LINE CONTENT - REQUIRED ==\n" +
		"For ANY multi-line content, you MUST use heredoc:\n\n" +
		"    xenith write 4 body <<EOF\n" +
		"    Line one\n" +
		"    Line two\n" +
		"    EOF\n\n" +
		"WITHOUT <<EOF, only the first line is captured!\n\n" +
		"== OUTPUT FORMAT - STRICT ==\n" +
		"Your output MUST be a tool invocation. Nothing else.\n\n" +
		"PROHIBITED:\n" +
		"- NO markdown, NO commentary, NO bash commands\n" +
		"- NO multi-line output without heredoc\n\n" +
		"== Completion ==\n" +
		"When done, output DONE on its own line.";
}

# Create LLM session using clone pattern
# Returns session ID (e.g., "0") or empty string on error
createsession(): string
{
	fd := sys->open("/n/llm/new", Sys->OREAD);
	if(fd == nil) {
		if(verbose)
			sys->fprint(stderr, "veltro: cannot open /n/llm/new: %r\n");
		return "";
	}
	buf := array[32] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return "";
	# Trim newline if present
	id := string buf[:n];
	if(len id > 0 && id[len id - 1] == '\n')
		id = id[:len id - 1];
	return id;
}

# Set prefill on session-specific path
setprefillpath(path, prefill: string)
{
	fd := sys->open(path, Sys->OWRITE);
	if(fd == nil) {
		if(verbose)
			sys->fprint(stderr, "veltro: cannot open %s: %r\n", path);
		return;
	}
	data := array of byte prefill;
	sys->write(fd, data, len data);
}

# Query LLM using persistent fd for conversation history
# The same fd must be used across all steps to maintain session isolation
queryllmfd(fd: ref Sys->FD, prompt: string): string
{
	# Write prompt
	data := array of byte prompt;
	n := sys->write(fd, data, len data);
	if(n != len data) {
		if(verbose)
			sys->fprint(stderr, "veltro: write to /n/llm/ask failed: %r\n");
		return "";
	}

	# Read response using pread from offset 0
	result := "";
	buf := array[8192] of byte;
	offset := big 0;
	for(;;) {
		n = sys->pread(fd, buf, len buf, offset);
		if(n <= 0)
			break;
		result += string buf[0:n];
		offset += big n;
	}
	return result;
}

# Parse tool invocation from LLM response
# Supports heredoc syntax for multi-line content:
#   tool arg1 arg2 <<EOF
#   multi-line
#   content
#   EOF
parseaction(response: string): (string, string)
{
	# Split into lines
	(nil, lines) := sys->tokenize(response, "\n");

	# Get available tools for matching
	(nil, toollist) := sys->tokenize(readfile("/tool/tools"), "\n");

	# Look for tool invocation
	for(; lines != nil; lines = tl lines) {
		line := hd lines;

		# Skip empty lines
		line = str->drop(line, " \t");
		if(line == "")
			continue;

		# Strip [Veltro] prefix if present (from prefill)
		if(hasprefix(line, "[Veltro]"))
			line = line[8:];
		line = str->drop(line, " \t");
		if(line == "")
			continue;

		# Check for DONE (strip markdown formatting first)
		stripped := str->drop(str->tolower(line), "*#`- ");
		if(stripped == "done" || hasprefix(stripped, "done"))
			return ("DONE", "");

		# Check if line starts with a known tool name
		(first, rest) := splitfirst(line);
		tool := str->tolower(first);

		# Match against discovered tools
		for(t := toollist; t != nil; t = tl t) {
			if(tool == hd t) {
				args := str->drop(rest, " \t");
				# Check for heredoc syntax
				(args, lines) = parseheredoc(args, tl lines);
				return (first, args);
			}
		}

		# Also check for "tools" and "help" (always available)
		if(tool == "tools" || tool == "help") {
			args := str->drop(rest, " \t");
			(args, lines) = parseheredoc(args, tl lines);
			return (first, args);
		}

		# First non-empty line is not a tool or DONE — reject immediately.
		# Do not scan further; the LLM is being conversational.
		return ("", "");
	}

	return ("", "");
}

# Parse heredoc content if present in args
# Returns (processed_args, remaining_lines)
# Heredoc format: <<DELIM ... DELIM (DELIM defaults to EOF)
parseheredoc(args: string, lines: list of string): (string, list of string)
{
	# Find heredoc marker <<
	markerpos := findheredoc(args);
	if(markerpos < 0)
		return (args, lines);

	# Extract delimiter (word after <<)
	aftermarker := args[markerpos + 2:];
	aftermarker = str->drop(aftermarker, " \t");
	(delim, _) := splitfirst(aftermarker);
	if(delim == "")
		delim = "EOF";

	# Args before the heredoc marker
	argsbefore := "";
	if(markerpos > 0)
		argsbefore = strip(args[0:markerpos]);

	# Collect heredoc content from remaining lines
	content := "";
	for(; lines != nil; lines = tl lines) {
		line := hd lines;
		# Check for end delimiter (must be alone on line, stripped)
		if(strip(line) == delim) {
			lines = tl lines;
			break;
		}
		if(content != "")
			content += "\n";
		content += line;
	}

	# Combine: args_before + heredoc_content
	result := argsbefore;
	if(result != "" && content != "")
		result += " ";
	result += content;

	return (result, lines);
}

# Find heredoc marker << in string, returns position or -1
findheredoc(s: string): int
{
	for(i := 0; i < len s - 1; i++) {
		if(s[i] == '<' && s[i+1] == '<') {
			# Make sure it's not <<< (which would be different)
			if(i + 2 >= len s || s[i+2] != '<')
				return i;
		}
	}
	return -1;
}

# Strip leading/trailing whitespace
strip(s: string): string
{
	i := 0;
	while(i < len s && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n'))
		i++;
	j := len s;
	while(j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n'))
		j--;
	if(i >= j)
		return "";
	return s[i:j];
}

# Strip action line from response
stripaction(response: string): string
{
	result := "";
	(nil, lines) := sys->tokenize(response, "\n");
	for(; lines != nil; lines = tl lines) {
		line := hd lines;
		lower := str->drop(str->tolower(str->drop(line, " \t")), "*#`- ");
		if(lower == "done" || hasprefix(lower, "done"))
			continue;
		if(result != "")
			result += "\n";
		result += line;
	}
	return result;
}

# Call tool via /tool filesystem
calltool(tool, args: string): string
{
	path := "/tool/" + str->tolower(tool);

	# Open tool file
	fd := sys->open(path, Sys->ORDWR);
	if(fd == nil)
		return sys->sprint("error: tool not found: %s", tool);

	# Write arguments
	if(args != "") {
		data := array of byte args;
		n := sys->write(fd, data, len data);
		if(n < 0)
			return sys->sprint("error: write to %s failed: %r", tool);
	}

	# Seek back to start
	sys->seek(fd, big 0, Sys->SEEKSTART);

	# Read result
	result := "";
	buf := array[8192] of byte;
	for(;;) {
		n := sys->read(fd, buf, len buf);
		if(n <= 0)
			break;
		result += string buf[0:n];
	}

	return result;
}

# Write large result to scratch file
writescratch(content: string, step: int): string
{
	ensuredir(SCRATCH_PATH);
	path := sys->sprint("%s/step%d.txt", SCRATCH_PATH, step);

	fd := sys->create(path, Sys->OWRITE, 8r644);
	if(fd == nil)
		return "(cannot create scratch file)";

	data := array of byte content;
	sys->write(fd, data, len data);
	return path;
}

# Helper functions
readfile(path: string): string
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return "";

	result := "";
	buf := array[8192] of byte;
	for(;;) {
		n := sys->read(fd, buf, len buf);
		if(n <= 0)
			break;
		result += string buf[0:n];
	}
	return result;
}

pathexists(path: string): int
{
	(ok, nil) := sys->stat(path);
	return ok >= 0;
}

ensuredir(path: string)
{
	fd := sys->open(path, Sys->OREAD);
	if(fd != nil)
		return;

	# Ensure parent
	for(i := len path - 1; i > 0; i--) {
		if(path[i] == '/') {
			ensuredir(path[0:i]);
			break;
		}
	}

	sys->create(path, Sys->OREAD, Sys->DMDIR | 8r755);
}

hasprefix(s, prefix: string): int
{
	return len s >= len prefix && s[0:len prefix] == prefix;
}

splitfirst(s: string): (string, string)
{
	for(i := 0; i < len s; i++) {
		if(s[i] == ' ' || s[i] == '\t')
			return (s[0:i], s[i:]);
	}
	return (s, "");
}

truncate(s: string, max: int): string
{
	if(len s <= max)
		return s;
	return s[0:max] + "...";
}
