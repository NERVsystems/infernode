implement AgentLib;

#
# agentlib - Shared agent library for Veltro
#
# Extracted from veltro.b and repl.b: LLM session management, prompt building,
# response parsing, tool execution, and utility functions. Each function uses
# the best version from whichever file had it (see plan audit table).
#

include "sys.m";
	sys: Sys;

include "string.m";
	str: String;

include "agentlib.m";

SCRATCH_PATH: con "/tmp/veltro/scratch";

verbose := 0;
stderr: ref Sys->FD;

init()
{
	sys = load Sys Sys->PATH;
	stderr = sys->fildes(2);
	str = load String String->PATH;
}

setverbose(v: int)
{
	verbose = v;
}

#
# ==================== LLM Session Management ====================
#

# Create LLM session using clone pattern
# Returns session ID (e.g., "0") or empty string on error
# (from veltro.b — has verbose logging on failure)
createsession(): string
{
	fd := sys->open("/n/llm/new", Sys->OREAD);
	if(fd == nil) {
		if(verbose)
			sys->fprint(stderr, "agentlib: cannot open /n/llm/new: %r\n");
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
# (from veltro.b — has verbose logging on failure)
setprefillpath(path, prefill: string)
{
	fd := sys->open(path, Sys->OWRITE);
	if(fd == nil) {
		if(verbose)
			sys->fprint(stderr, "agentlib: cannot open %s: %r\n", path);
		return;
	}
	data := array of byte prefill;
	sys->write(fd, data, len data);
}

# Query LLM using persistent fd for conversation history
# The same fd must be used across all steps to maintain session isolation
# (from veltro.b — has verbose logging on write failure)
queryllmfd(fd: ref Sys->FD, prompt: string): string
{
	# Write prompt
	data := array of byte prompt;
	n := sys->write(fd, data, len data);
	if(n != len data) {
		if(verbose)
			sys->fprint(stderr, "agentlib: write to ask failed: %r\n");
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

# Write system prompt to session path
# (from repl.b — only repl had this)
setsystemprompt(path, prompt: string)
{
	fd := sys->open(path, Sys->OWRITE);
	if(fd == nil) {
		sys->fprint(stderr, "agentlib: cannot open %s: %r\n", path);
		return;
	}
	data := array of byte prompt;
	n := sys->write(fd, data, len data);
	if(n != len data)
		sys->fprint(stderr, "agentlib: system prompt write: %d/%d bytes: %r\n", n, len data);
	else if(verbose)
		sys->fprint(stderr, "agentlib: system prompt set: %d bytes\n", n);
}

#
# ==================== Prompt Building ====================
#

# Discover namespace — read /tool/tools and list accessible paths
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

# Build system prompt with namespace and reminders
# Does NOT append mode-specific suffix — callers add their own
# (from repl.b — has MAXPROMPT 8KB guard against 9P write limit)
buildsystemprompt(ns: string): string
{
	# NOTE: The system prompt may be written to /n/llm/{id}/system via a single
	# 9P Twrite. llm9p's MaxMessageSize is 8192 bytes, and each write
	# REPLACES the content (offset is ignored). If the prompt exceeds ~8KB,
	# the kernel splits into multiple Twrites and only the LAST survives.
	MAXPROMPT: con 8000;

	# Read base system prompt
	base := readfile("/lib/veltro/system.txt");
	if(base == "")
		base = defaultsystemprompt();

	# Tool names are already in the namespace section.
	# Full tool docs are too large for the 9P write limit.
	(nil, toollist) := sys->tokenize(readfile("/tool/tools"), "\n");

	# Load context-specific reminders based on available tools
	reminders := loadreminders(toollist);

	prompt := base + "\n\n== Your Namespace ==\n" + ns +
		"\n\nUse 'help <toolname>' to see usage for any tool.";

	if(reminders != "")
		prompt += "\n\n== Reminders ==\n" + reminders;

	# Guard against exceeding 9P write limit
	data := array of byte prompt;
	if(len data > MAXPROMPT) {
		sys->fprint(stderr, "agentlib: WARNING: system prompt %d bytes exceeds %d limit, truncating\n",
			len data, MAXPROMPT);
		prompt = string data[0:MAXPROMPT];
	}

	return prompt;
}

# Load context-specific reminders based on available tools
loadreminders(toollist: list of string): string
{
	reminders := "";

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

#
# ==================== Response Parsing ====================
#

# Parse tool invocation from LLM response
# Supports heredoc syntax for multi-line content and collectsaytext for say
# (from repl.b — has collectsaytext for multi-line say)
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
				# say collects all remaining lines as text
				if(tool == "say")
					args = collectsaytext(args, tl lines);
				else
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
	(delim, nil) := splitfirst(aftermarker);
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

# Collect all remaining lines as say text, stopping at DONE
# Strips markdown formatting for cleaner speech
# (from repl.b — only repl had this)
collectsaytext(first: string, lines: list of string): string
{
	text := first;
	for(; lines != nil; lines = tl lines) {
		line := hd lines;
		cleaned := str->drop(line, " \t");
		if(hasprefix(cleaned, "[Veltro]"))
			cleaned = cleaned[8:];
		cleaned = str->drop(cleaned, " \t");
		lower := str->tolower(cleaned);
		if(lower == "done" || hasprefix(lower, "done"))
			break;
		if(cleaned == "")
			text += " ";  # Preserve paragraph breaks as space
		else
			text += " " + stripmarkdown(cleaned);
	}
	return text;
}

# Strip common markdown formatting for speech
# (from repl.b — only repl had this)
stripmarkdown(s: string): string
{
	result := "";
	for(i := 0; i < len s; i++) {
		c := s[i];
		# Skip ** and * (bold/italic markers)
		if(c == '*')
			continue;
		# Skip # at start of line (headers)
		if(c == '#' && (i == 0 || s[i-1] == '\n'))
			continue;
		# Skip ` (code markers)
		if(c == '`')
			continue;
		result[len result] = c;
	}
	return result;
}

# Strip action line from response
# (from repl.b — strips [Veltro] prefix and empty lines)
stripaction(response: string): string
{
	result := "";
	(nil, lines) := sys->tokenize(response, "\n");
	for(; lines != nil; lines = tl lines) {
		line := hd lines;
		lower := str->drop(str->tolower(str->drop(line, " \t")), "*#`- ");
		if(lower == "done" || hasprefix(lower, "done"))
			continue;
		cleaned := str->drop(line, " \t");
		if(hasprefix(cleaned, "[Veltro]"))
			cleaned = cleaned[8:];
		cleaned = str->drop(cleaned, " \t");
		if(cleaned == "")
			continue;
		if(result != "")
			result += "\n";
		result += cleaned;
	}
	return result;
}

#
# ==================== Tool Execution ====================
#

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

#
# ==================== Utilities ====================
#

# Read entire file contents
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

# Check if path exists
pathexists(path: string): int
{
	(ok, nil) := sys->stat(path);
	return ok >= 0;
}

# Ensure directory exists (recursive)
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

# Check string prefix
hasprefix(s, prefix: string): int
{
	return len s >= len prefix && s[0:len prefix] == prefix;
}

# Split string at first whitespace
splitfirst(s: string): (string, string)
{
	for(i := 0; i < len s; i++) {
		if(s[i] == ' ' || s[i] == '\t')
			return (s[0:i], s[i:]);
	}
	return (s, "");
}

# Truncate string with ellipsis
truncate(s: string, max: int): string
{
	if(len s <= max)
		return s;
	return s[0:max] + "...";
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
