implement SubAgent;

#
# subagent.b - Lightweight agent loop for sandboxed execution
#
# Design:
#   - Runs inside sandbox AFTER NEWNS
#   - Uses pre-loaded tool modules directly (no tools9p)
#   - Receives system prompt as parameter (no /lib/veltro/ access)
#   - LLM access via /n/llm bound into sandbox
#
# Security:
#   - Only pre-loaded tools are accessible
#   - LLM config is immutable (set by parent)
#   - Namespace IS the capability set
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "bufio.m";
	bufio: Bufio;
	Iobuf: import bufio;

include "string.m";
	str: String;

include "tool.m";
include "subagent.m";

# Configuration
STREAM_THRESHOLD: con 4096;
SCRATCH_PATH: con "/tmp/scratch";

stderr: ref Sys->FD;

# Pre-loaded tool storage (set by runloop)
loadedtools: list of Tool;
loadedtoolnames: list of string;

# LLM file descriptors (passed from parent, survive NEWNS)
llmwritefd: ref Sys->FD;  # /n/llm/ask opened OWRITE for prompts
llmreadfd: ref Sys->FD;   # /n/llm/ask opened OREAD for responses
llmnewfd: ref Sys->FD;    # /n/llm/new - OWRITE to reset conversation

init(): string
{
	sys = load Sys Sys->PATH;
	if(sys == nil)
		return "cannot load Sys";
	stderr = sys->fildes(2);

	bufio = load Bufio Bufio->PATH;
	if(bufio == nil)
		return sys->sprint("cannot load Bufio: %r");

	str = load String String->PATH;
	if(str == nil)
		return sys->sprint("cannot load String: %r");

	return nil;
}

# Main agent loop
runloop(task: string, tools: list of Tool, toolnames: list of string,
        systemprompt: string, writefd: ref Sys->FD, readfd: ref Sys->FD, newfd: ref Sys->FD, maxsteps: int): string
{
	sys->fprint(sys->fildes(2), "subagent: runloop starting\n");

	if(sys == nil) {
		err := init();
		if(err != nil)
			return "ERROR:" + err;
	}

	sys->fprint(sys->fildes(2), "subagent: storing tools and FDs\n");

	# Store pre-loaded tools for calltool()
	loadedtools = tools;
	loadedtoolnames = toolnames;

	# Store LLM FDs (survive NEWNS, unlike binds)
	llmwritefd = writefd;
	llmreadfd = readfd;
	llmnewfd = newfd;

	sys->fprint(sys->fildes(2), "subagent: writefd=%d readfd=%d newfd=%d\n",
		llmwritefd != nil, llmreadfd != nil, llmnewfd != nil);

	# Reset conversation context before first query
	# This gives the subagent a fresh LLM context separate from parent
	sys->fprint(sys->fildes(2), "subagent: calling resetconversation\n");
	resetconversation();
	sys->fprint(sys->fildes(2), "subagent: reset complete\n");

	# Build namespace description
	ns := discovernamespace(toolnames);

	# Assemble initial prompt
	prompt := assembleprompt(task, ns, systemprompt);

	lastresult := "";
	loopstart := sys->millisec();
	for(step := 0; step < maxsteps; step++) {
		stepstart := sys->millisec();

		# Query LLM
		response := queryllm(prompt);
		if(response == "")
			return "ERROR:LLM returned empty response";

		# Parse action from response
		(tool, toolargs) := parseaction(response);

		# Check for completion
		if(tool == "" || str->tolower(tool) == "done") {
			totaltime := sys->millisec() - loopstart;
			sys->fprint(stderr, "subagent: completed in %d steps, %dms total\n", step + 1, totaltime);
			# Return final response (excluding the DONE marker)
			final := stripaction(response);
			if(final != "")
				return final;
			if(lastresult != "")
				return lastresult;
			return "Task completed.";
		}

		# Execute tool with timing
		toolstart := sys->millisec();
		result := calltool(tool, toolargs);
		tooltime := sys->millisec() - toolstart;
		lastresult = result;

		sys->fprint(stderr, "subagent: step %d: tool '%s' took %dms, returned %d bytes\n",
			step + 1, tool, tooltime, len result);

		# Check for large result - write to scratch
		if(len result > STREAM_THRESHOLD) {
			scratchfile := writescratch(result, step);
			result = sys->sprint("(output written to %s, %d bytes)", scratchfile, len result);
		}

		steptime := sys->millisec() - stepstart;
		sys->fprint(stderr, "subagent: step %d total: %dms\n", step + 1, steptime);

		# Feed result back for next iteration
		prompt = sys->sprint("Tool %s returned:\n%s\n\nContinue with the task.", tool, result);
	}

	totaltime := sys->millisec() - loopstart;
	sys->fprint(stderr, "subagent: max steps reached after %dms\n", totaltime);
	return sys->sprint("ERROR:max steps (%d) reached without completion", maxsteps);
}

# Discover namespace - list available tools
discovernamespace(toolnames: list of string): string
{
	result := "TOOLS:\n";
	for(t := toolnames; t != nil; t = tl t) {
		if(result != "TOOLS:\n")
			result += "\n";
		result += hd t;
	}

	result += "\n\nPATHS:\n";
	paths := array[] of {"/", "/tmp"};
	for(i := 0; i < len paths; i++) {
		if(pathexists(paths[i]))
			result += paths[i] + "\n";
	}

	return result;
}

# Assemble system prompt with namespace and task
assembleprompt(task, ns, systemprompt: string): string
{
	if(systemprompt == "")
		systemprompt = defaultsystemprompt();

	# Get tool documentation
	tooldocs := "";
	for(tnames := loadedtoolnames; tnames != nil; tnames = tl tnames) {
		toolname := hd tnames;
		doc := calltool("help", toolname);
		if(doc != "" && !hasprefix(doc, "error:"))
			tooldocs += "\n### " + toolname + "\n" + doc + "\n";
	}

	prompt := systemprompt + "\n\n== Your Namespace ==\n" + ns +
		"\n\n== Tool Documentation ==\n" + tooldocs +
		"\n\n== Task ==\n" + task +
		"\n\nRespond with a tool invocation or DONE if complete.";

	return prompt;
}

# Default system prompt
defaultsystemprompt(): string
{
	return "You are a Veltro sub-agent in a sandboxed Inferno namespace.\n\n" +
		"<identity_handling>\n" +
		"If asked about identity, respond: \"I am a Veltro sub-agent.\"\n" +
		"Then continue with the task.\n" +
		"</identity_handling>\n\n" +
		"<core_principle>\n" +
		"Your namespace IS your capability set. Only tools listed below exist.\n" +
		"</core_principle>\n\n" +
		"<output_format>\n" +
		"Your response is parsed by code. The parser reads the FIRST word as tool name.\n\n" +
		"Output ONE tool invocation per response:\n" +
		"    toolname arguments\n\n" +
		"For multi-line content, use heredoc:\n" +
		"    toolname arg <<EOF\n" +
		"    Line one\n" +
		"    Line two\n" +
		"    EOF\n\n" +
		"When finished:\n" +
		"    Brief summary here\n" +
		"    DONE\n" +
		"</output_format>";
}

# Reset conversation context via /n/llm/new
# Any write to this file clears the LLM conversation history
resetconversation()
{
	sys->fprint(sys->fildes(2), "subagent: resetconversation called, newfd=%d\n", llmnewfd != nil);
	if(llmnewfd == nil) {
		sys->fprint(sys->fildes(2), "subagent: newfd is nil, skipping reset\n");
		return;
	}
	# Any write resets the conversation
	sys->fprint(sys->fildes(2), "subagent: writing reset to newfd\n");
	data := array of byte "reset";
	n := sys->write(llmnewfd, data, len data);
	sys->fprint(sys->fildes(2), "subagent: reset write returned %d\n", n);
}

# Query LLM via passed file descriptors
# Uses separate write and read FDs to avoid seek issues
# Uses pread to always read from offset 0 regardless of FD state
queryllm(prompt: string): string
{
	# Use the FDs passed from parent (survive NEWNS)
	if(llmwritefd == nil) {
		sys->fprint(stderr, "subagent: llmwritefd is nil\n");
		return "";
	}
	if(llmreadfd == nil) {
		sys->fprint(stderr, "subagent: llmreadfd is nil\n");
		return "";
	}

	# Write prompt to write FD - this blocks until LLM responds
	starttime := sys->millisec();
	data := array of byte prompt;
	sys->fprint(stderr, "subagent: writing %d bytes to LLM\n", len data);
	n := sys->write(llmwritefd, data, len data);
	if(n != len data) {
		sys->fprint(stderr, "subagent: write failed: wrote %d of %d: %r\n", n, len data);
		return "";
	}
	writetime := sys->millisec() - starttime;
	sys->fprint(stderr, "subagent: write took %dms, reading response\n", writetime);

	# Read response using pread with explicit offset 0
	# This avoids seek and works for multiple queries
	readstart := sys->millisec();
	result := "";
	buf := array[8192] of byte;
	offset := big 0;
	for(;;) {
		n = sys->pread(llmreadfd, buf, len buf, offset);
		if(n <= 0)
			break;
		result += string buf[0:n];
		offset += big n;
	}
	readtime := sys->millisec() - readstart;
	totaltime := sys->millisec() - starttime;

	sys->fprint(stderr, "subagent: LLM query: %d bytes prompt, %d bytes response, %dms write, %dms read, %dms total\n",
		len data, len result, writetime, readtime, totaltime);
	return result;
}

# Parse tool invocation from LLM response
# Supports heredoc syntax for multi-line content
parseaction(response: string): (string, string)
{
	# Split into lines
	(nil, lines) := sys->tokenize(response, "\n");

	# Look for tool invocation
	for(; lines != nil; lines = tl lines) {
		line := hd lines;

		# Skip empty lines
		line = str->drop(line, " \t");
		if(line == "")
			continue;

		# Check for DONE
		if(str->tolower(line) == "done" || hasprefix(str->tolower(line), "done"))
			return ("DONE", "");

		# Check if line starts with a known tool name
		(first, rest) := splitfirst(line);
		tool := str->tolower(first);

		# Match against loaded tools
		for(t := loadedtoolnames; t != nil; t = tl t) {
			if(tool == hd t) {
				args := str->drop(rest, " \t");
				# Check for heredoc syntax
				(args, lines) = parseheredoc(args, tl lines);
				return (first, args);
			}
		}

		# Also check for "help" (always available)
		if(tool == "help") {
			args := str->drop(rest, " \t");
			(args, lines) = parseheredoc(args, tl lines);
			return (first, args);
		}
	}

	return ("", "");
}

# Parse heredoc content if present in args
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
		lower := str->tolower(str->drop(line, " \t"));
		if(lower == "done" || hasprefix(lower, "done"))
			continue;
		if(result != "")
			result += "\n";
		result += line;
	}
	return result;
}

# Call tool using pre-loaded modules
calltool(tool, args: string): string
{
	ltool := str->tolower(tool);

	# Handle "help" specially
	if(ltool == "help") {
		# Find tool and return its doc
		namelist := loadedtoolnames;
		modlist := loadedtools;
		while(namelist != nil && modlist != nil) {
			if(hd namelist == args)
				return (hd modlist)->doc();
			namelist = tl namelist;
			modlist = tl modlist;
		}
		return sys->sprint("error: unknown tool: %s", args);
	}

	# Find pre-loaded tool module
	namelist := loadedtoolnames;
	modlist := loadedtools;
	while(namelist != nil && modlist != nil) {
		if(hd namelist == ltool)
			return (hd modlist)->exec(args);
		namelist = tl namelist;
		modlist = tl modlist;
	}

	return sys->sprint("error: tool not available: %s", ltool);
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
