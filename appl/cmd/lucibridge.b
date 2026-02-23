implement LuciBridge;

#
# lucibridge - Connects Lucifer UI to an LLM via llm9p
#
# Reads human messages from /mnt/ui/activity/{id}/conversation/input
# (blocking read), sends them to /n/llm/{session}/ask, and writes
# responses back as role=veltro messages.
#
# Usage: lucibridge [-v] [-a actid]
#   -v       verbose logging
#   -a id    activity ID (default: 0)
#
# Prerequisites:
#   - luciuisrv running (serves /mnt/ui/)
#   - llm9p mounted at /n/llm/
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "string.m";
	str: String;

include "arg.m";
	arg: Arg;

LuciBridge: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

verbose := 0;
stderr: ref Sys->FD;

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

readfile(path: string): string
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return nil;
	buf := array[8192] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return nil;
	s := string buf[0:n];
	if(len s > 0 && s[len s - 1] == '\n')
		s = s[0:len s - 1];
	return s;
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

# Query LLM: write prompt, pread response from offset 0
queryllm(fd: ref Sys->FD, prompt: string): string
{
	data := array of byte prompt;
	n := sys->write(fd, data, len data);
	if(n != len data) {
		sys->fprint(stderr, "lucibridge: llm write failed: %r\n");
		return nil;
	}
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

# Escape text for key=value format: the text field is always last,
# so no escaping is needed — luciuisrv parses greedily.
writemsg(actid: int, role, text: string)
{
	path := sys->sprint("/mnt/ui/activity/%d/conversation/ctl", actid);
	msg := "role=" + role + " text=" + text;
	if(writefile(path, msg) < 0)
		sys->fprint(stderr, "lucibridge: write to %s failed: %r\n", path);
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

	actid := 0;

	arg->init(args);
	while((c := arg->opt()) != 0) {
		case c {
		'v' =>
			verbose = 1;
		'a' =>
			s := arg->arg();
			if(s == nil)
				fatal("-a requires activity ID");
			(actid, nil) = str->toint(s, 10);
		* =>
			sys->fprint(stderr, "usage: lucibridge [-v] [-a actid]\n");
			raise "fail:usage";
		}
	}

	# Verify prerequisites
	if(sys->open("/mnt/ui/ctl", Sys->OREAD) == nil)
		fatal("/mnt/ui/ not mounted — start luciuisrv first");
	if(sys->open("/n/llm/new", Sys->OREAD) == nil)
		fatal("/n/llm/ not mounted — mount llm9p first");

	# Create LLM session
	sessionid := readfile("/n/llm/new");
	if(sessionid == nil)
		fatal("cannot create LLM session");
	log("llm session: " + sessionid);

	# Open persistent ask fd (maintains conversation history)
	askpath := "/n/llm/" + sessionid + "/ask";
	askfd := sys->open(askpath, Sys->ORDWR);
	if(askfd == nil)
		fatal("cannot open " + askpath);

	# Set model to haiku for cost efficiency
	writefile("/n/llm/" + sessionid + "/model", "claude-haiku-4-5-20251001");
	log("model: " + readfile("/n/llm/" + sessionid + "/model"));

	# Open blocking input reader
	inputpath := sys->sprint("/mnt/ui/activity/%d/conversation/input", actid);
	inputfd := sys->open(inputpath, Sys->OREAD);
	if(inputfd == nil)
		fatal("cannot open " + inputpath);

	log(sys->sprint("ready — listening on activity %d", actid));

	# Main loop
	for(;;) {
		# Block until human sends a message
		human := blockread(inputfd);
		if(human == nil) {
			log("input closed");
			break;
		}
		log("human: " + human);

		# Record human message in UI
		writemsg(actid, "human", human);

		# Update status
		statuspath := sys->sprint("/mnt/ui/activity/%d/status", actid);
		writefile(statuspath, "working");

		# Query LLM
		response := queryllm(askfd, human);
		if(response == nil) {
			writemsg(actid, "veltro", "(no response from LLM)");
			writefile(statuspath, "idle");
			continue;
		}
		log("veltro: " + response);

		# Write response to UI
		writemsg(actid, "veltro", response);
		writefile(statuspath, "idle");
	}
}
