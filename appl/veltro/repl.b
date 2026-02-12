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

include "xenithwin.m";
	xenithwin: Xenithwin;
	Win, Event: import xenithwin;

Repl: module {
	init: fn(ctxt: ref Draw->Context, argv: list of string);
};

# Defaults
DEFAULT_MAX_STEPS: con 50;
MAX_MAX_STEPS: con 100;
SCRATCH_PATH: con "/tmp/veltro/scratch";
STREAM_THRESHOLD: con 4096;

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

	# Check required mounts
	if(!pathexists("/tool")) {
		sys->fprint(stderr, "repl: /tool not mounted (run tools9p first)\n");
		raise "fail:no /tool";
	}
	if(!pathexists("/n/llm")) {
		sys->fprint(stderr, "repl: /n/llm not mounted\n");
		raise "fail:no /n/llm";
	}

	# Create LLM session
	initsession();

	# Detect Xenith and choose mode
	if(xenithavail()) {
		xenithwin = load Xenithwin Xenithwin->PATH;
		if(xenithwin == nil)
			nomod(Xenithwin->PATH);
		xenithwin->init();
		xenithmode();
	} else {
		termmode();
	}
}

# Check if Xenith window system is available
xenithavail(): int
{
	fd := sys->open("/chan/new/ctl", Sys->OREAD);
	if(fd == nil)
		return 0;
	fd = nil;
	return 1;
}

# Create LLM session and set up system prompt
initsession()
{
	sessionid = createsession();
	if(sessionid == "") {
		sys->fprint(stderr, "repl: cannot create LLM session\n");
		raise "fail:no LLM session";
	}

	prefillpath := "/n/llm/" + sessionid + "/prefill";
	setprefillpath(prefillpath, "[Veltro]\n");

	systempath := "/n/llm/" + sessionid + "/system";
	ns := discovernamespace();
	sysprompt := buildsystemprompt(ns);
	setsystemprompt(systempath, sysprompt);

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
		line = strip(line);
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
	sessionid = createsession();
	if(sessionid == "") {
		sys->print("[error: cannot create new LLM session]\n");
		return;
	}

	prefillpath := "/n/llm/" + sessionid + "/prefill";
	setprefillpath(prefillpath, "[Veltro]\n");

	systempath := "/n/llm/" + sessionid + "/system";
	ns := discovernamespace();
	sysprompt := buildsystemprompt(ns);
	setsystemprompt(systempath, sysprompt);

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
	ns := discovernamespace();
	prompt := input + "\n\n== Your Namespace ==\n" + ns +
		"\n\nRespond with a tool invocation or DONE if complete.";

	for(step := 0; step < maxsteps; step++) {
		if(verbose)
			sys->fprint(stderr, "repl: step %d\n", step + 1);

		sys->print("[thinking...]\n");

		response := queryllmfd(llmfd, prompt);
		if(response == "") {
			sys->print("[error: LLM returned empty response]\n\n");
			return;
		}

		if(verbose)
			sys->fprint(stderr, "repl: LLM: %s\n", response);

		(tool, toolargs) := parseaction(response);

		if(tool == "" || str->tolower(tool) == "done") {
			final := stripaction(response);
			if(final != "")
				sys->print("%s\n\n", final);
			return;
		}

		# For say, display the full text so user can read it
		if(str->tolower(tool) == "say")
			sys->print("[speaking] %s\n", toolargs);
		else
			sys->print("[%s %s]\n", tool, truncate(toolargs, 80));

		result := calltool(tool, toolargs);

		if(verbose)
			sys->fprint(stderr, "repl: tool result: %s\n", truncate(result, 200));

		if(len result > STREAM_THRESHOLD) {
			scratchfile := writescratch(result, step);
			result = sys->sprint("(output written to %s, %d bytes)", scratchfile, len result);
		}

		if(str->tolower(tool) == "spawn")
			prompt = sys->sprint("Tool %s completed:\n%s\n\nThe subagent has finished. Summarize the result briefly and output DONE.", tool, result);
		else
			prompt = sys->sprint("Tool %s returned:\n%s\n\nContinue with the task.", tool, result);
	}

	sys->print("[max steps reached]\n\n");
}

#
# ==================== Xenith Mode ====================
#

xenithmode()
{
	w = Win.wnew();
	w.wname("/+Veltro");
	w.wtagwrite(" Send Voice Clear Reset Delete");

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
	(word, nil) := splitfirst(cmd);
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
	input = strip(input);
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

	result = strip(result);
	if(result == "" || hasprefix(result, "error:")) {
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

	sessionid = createsession();
	if(sessionid == "") {
		appendoutput("[error: cannot create new LLM session]\n");
		return;
	}

	prefillpath := "/n/llm/" + sessionid + "/prefill";
	setprefillpath(prefillpath, "[Veltro]\n");

	systempath := "/n/llm/" + sessionid + "/system";
	ns := discovernamespace();
	sysprompt := buildsystemprompt(ns);
	setsystemprompt(systempath, sysprompt);

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
	ns := discovernamespace();
	prompt := input + "\n\n== Your Namespace ==\n" + ns +
		"\n\nRespond with a tool invocation or DONE if complete.";

	for(step := 0; step < maxsteps; step++) {
		if(verbose)
			sys->fprint(stderr, "repl: step %d\n", step + 1);

		agentout <-= "[thinking...]\n";

		response := queryllmfd(llmfd, prompt);
		if(response == "") {
			agentout <-= "[error: LLM returned empty response]\n\n";
			break;
		}

		if(verbose)
			sys->fprint(stderr, "repl: LLM: %s\n", response);

		(tool, toolargs) := parseaction(response);

		if(tool == "" || str->tolower(tool) == "done") {
			final := stripaction(response);
			if(final != "")
				agentout <-= final + "\n\n";
			else
				agentout <-= "\n";
			break;
		}

		# For say, display the full text so user can read it
		if(str->tolower(tool) == "say")
			agentout <-= "[speaking] " + toolargs + "\n";
		else
			agentout <-= "[" + tool + " " + truncate(toolargs, 80) + "]\n";

		result := calltool(tool, toolargs);

		if(verbose)
			sys->fprint(stderr, "repl: tool result: %s\n", truncate(result, 200));

		if(len result > STREAM_THRESHOLD) {
			scratchfile := writescratch(result, step);
			result = sys->sprint("(output written to %s, %d bytes)", scratchfile, len result);
		}

		if(str->tolower(tool) == "spawn")
			prompt = sys->sprint("Tool %s completed:\n%s\n\nThe subagent has finished. Summarize the result briefly and output DONE.", tool, result);
		else
			prompt = sys->sprint("Tool %s returned:\n%s\n\nContinue with the task.", tool, result);
	}

	busy = 0;
}

#
# ==================== Shared: System Prompt ====================
#

buildsystemprompt(ns: string): string
{
	base := readfile("/lib/veltro/system.txt");
	if(base == "")
		base = defaultsystemprompt();

	tooldocs := "";
	(nil, toollist) := sys->tokenize(readfile("/tool/tools"), "\n");
	for(t := toollist; t != nil; t = tl t) {
		toolname := hd t;
		doc := calltool("help", toolname);
		if(doc != "" && !hasprefix(doc, "error:"))
			tooldocs += "\n### " + toolname + "\n" + doc + "\n";
	}

	reminders := loadreminders(toollist);

	prompt := base + "\n\n== Your Namespace ==\n" + ns +
		"\n\n== Tool Documentation ==\n" + tooldocs;

	if(reminders != "")
		prompt += "\n\n== Reminders ==\n" + reminders;

	prompt += "\n\nYou are in interactive REPL mode. The user will send messages. " +
		"Respond with tool invocations or DONE when you have answered.";

	return prompt;
}

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

setsystemprompt(path, prompt: string)
{
	fd := sys->open(path, Sys->OWRITE);
	if(fd == nil) {
		if(verbose)
			sys->fprint(stderr, "repl: cannot open %s: %r\n", path);
		return;
	}
	data := array of byte prompt;
	sys->write(fd, data, len data);
}

#
# ==================== Shared: LLM & Tools ====================
#

createsession(): string
{
	fd := sys->open("/n/llm/new", Sys->OREAD);
	if(fd == nil)
		return "";
	buf := array[32] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return "";
	id := string buf[:n];
	if(len id > 0 && id[len id - 1] == '\n')
		id = id[:len id - 1];
	return id;
}

setprefillpath(path, prefill: string)
{
	fd := sys->open(path, Sys->OWRITE);
	if(fd == nil)
		return;
	data := array of byte prefill;
	sys->write(fd, data, len data);
}

queryllmfd(fd: ref Sys->FD, prompt: string): string
{
	data := array of byte prompt;
	n := sys->write(fd, data, len data);
	if(n != len data)
		return "";

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

discovernamespace(): string
{
	result := "TOOLS:\n";

	tools := readfile("/tool/tools");
	if(tools != "")
		result += tools;
	else
		result += "(none)";

	result += "\n\nPATHS:\n";
	paths := array[] of {"/", "/tool", "/n", "/tmp"};
	for(i := 0; i < len paths; i++) {
		if(pathexists(paths[i]))
			result += paths[i] + "\n";
	}

	return result;
}

parseaction(response: string): (string, string)
{
	(nil, lines) := sys->tokenize(response, "\n");
	(nil, toollist) := sys->tokenize(readfile("/tool/tools"), "\n");

	for(; lines != nil; lines = tl lines) {
		line := hd lines;
		line = str->drop(line, " \t");
		if(line == "")
			continue;

		if(hasprefix(line, "[Veltro]"))
			line = line[8:];
		line = str->drop(line, " \t");
		if(line == "")
			continue;

		if(str->tolower(line) == "done" || hasprefix(str->tolower(line), "done"))
			return ("DONE", "");

		(first, rest) := splitfirst(line);
		tool := str->tolower(first);

		for(t := toollist; t != nil; t = tl t) {
			if(tool == hd t) {
				args := str->drop(rest, " \t");
				if(tool == "say") {
					# Collect all remaining lines as text
					# LLM output after say often spans multiple lines
					args = collectsaytext(args, tl lines);
				} else
					(args, lines) = parseheredoc(args, tl lines);
				return (first, args);
			}
		}

		if(tool == "tools" || tool == "help") {
			args := str->drop(rest, " \t");
			(args, lines) = parseheredoc(args, tl lines);
			return (first, args);
		}
	}

	return ("", "");
}

parseheredoc(args: string, lines: list of string): (string, list of string)
{
	markerpos := findheredoc(args);
	if(markerpos < 0)
		return (args, lines);

	aftermarker := args[markerpos + 2:];
	aftermarker = str->drop(aftermarker, " \t");
	(delim, nil) := splitfirst(aftermarker);
	if(delim == "")
		delim = "EOF";

	argsbefore := "";
	if(markerpos > 0)
		argsbefore = strip(args[0:markerpos]);

	content := "";
	for(; lines != nil; lines = tl lines) {
		line := hd lines;
		if(strip(line) == delim) {
			lines = tl lines;
			break;
		}
		if(content != "")
			content += "\n";
		content += line;
	}

	result := argsbefore;
	if(result != "" && content != "")
		result += " ";
	result += content;

	return (result, lines);
}

# Collect all remaining lines as say text, stopping at DONE
# Strips markdown formatting for cleaner speech
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

findheredoc(s: string): int
{
	for(i := 0; i < len s - 1; i++) {
		if(s[i] == '<' && s[i+1] == '<') {
			if(i + 2 >= len s || s[i+2] != '<')
				return i;
		}
	}
	return -1;
}

stripaction(response: string): string
{
	result := "";
	(nil, lines) := sys->tokenize(response, "\n");
	for(; lines != nil; lines = tl lines) {
		line := hd lines;
		lower := str->tolower(str->drop(line, " \t"));
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

calltool(tool, args: string): string
{
	path := "/tool/" + str->tolower(tool);

	fd := sys->open(path, Sys->ORDWR);
	if(fd == nil)
		return sys->sprint("error: tool not found: %s", tool);

	if(args != "") {
		data := array of byte args;
		n := sys->write(fd, data, len data);
		if(n < 0)
			return sys->sprint("error: write to %s failed: %r", tool);
	}

	sys->seek(fd, big 0, Sys->SEEKSTART);

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
# ==================== Helpers ====================
#

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

	for(i := len path - 1; i > 0; i--) {
		if(path[i] == '/') {
			ensuredir(path[0:i]);
			break;
		}
	}

	sys->create(path, Sys->OREAD, Sys->DMDIR | 8r755);
}

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
