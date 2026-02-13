implement ToolXenith;

#
# xenith - Xenith UI control tool for Veltro agent
#
# Provides AI control over Xenith's Acme-style windowing system.
# Windows are exposed via the file server at /mnt/xenith/.
#
# Commands:
#   create [name]              - Create new window, returns ID
#   write <id> body <text>     - Write text to window body
#   write <id> tag <text>      - Write text to window tag
#   read <id> [body|tag]       - Read window content (default: body)
#   append <id> <text>         - Append text to body
#   ctl <id> <commands>        - Send control commands
#   colors <id> <settings>     - Set window colors
#   delete <id>                - Delete window
#   list                       - List all windows
#   status <id> <state>        - Set visual status (ok/warn/error/info)
#
# Status colors (for AI feedback):
#   ok    - Green tag (success)
#   warn  - Yellow tag (warning)
#   error - Red tag (error)
#   info  - Blue tag (information)
#   reset - Default colors
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "string.m";
	str: String;

include "../tool.m";

ToolXenith: module {
	init: fn(): string;
	name: fn(): string;
	doc:  fn(): string;
	exec: fn(args: string): string;
};

XENITH_ROOT: con "/mnt/xenith";

init(): string
{
	sys = load Sys Sys->PATH;
	if(sys == nil)
		return "cannot load Sys";
	str = load String String->PATH;
	if(str == nil)
		return "cannot load String";
	return nil;
}

name(): string
{
	return "xenith";
}

doc(): string
{
	return "Xenith - AI control for Xenith windowing system\n\n" +
		"Commands:\n" +
		"  create [name]              Create new window, returns ID\n" +
		"  write <id> body <text>     Write text to window body\n" +
		"  write <id> tag <text>      Write text to window tag  \n" +
		"  read <id> [body|tag]       Read window content (default: body)\n" +
		"  append <id> <text>         Append text to body\n" +
		"  ctl <id> <commands>        Send control commands\n" +
		"  colors <id> <settings>     Set window colors\n" +
		"  delete <id>                Delete window\n" +
		"  list                       List all windows\n" +
		"  status <id> <state>        Set visual status indicator\n\n" +
		"Status states: ok (green), warn (yellow), error (red), info (blue), reset\n\n" +
		"Control commands (for ctl):\n" +
		"  name <string>    Set window name/title\n" +
		"  clean            Mark as unmodified\n" +
		"  show             Scroll to cursor position\n" +
		"  grow             Moderate growth in column\n" +
		"  growmax          Maximum size in column\n" +
		"  growfull         Full column height\n" +
		"  moveto <y>       Move to Y position\n" +
		"  tocol <n> [y]    Move to column n\n" +
		"  noscroll         Disable auto-scroll on write\n" +
		"  scroll           Enable auto-scroll\n\n" +
		"Examples:\n" +
		"  xenith create output        Create window named 'output'\n" +
		"  xenith write 3 body Hello   Write 'Hello' to window 3 body\n" +
		"  xenith status 3 ok          Set green status on window 3\n" +
		"  xenith ctl 3 growmax        Maximize window 3\n";
}

exec(args: string): string
{
	if(sys == nil)
		init();

	args = strip(args);
	if(args == "")
		return "error: no command specified. Use: create, write, read, append, ctl, colors, delete, list, status";

	# Parse command
	(cmd, rest) := splitfirst(args);
	cmd = str->tolower(cmd);

	case cmd {
	"create" =>
		return docreate(rest);
	"write" =>
		return dowrite(rest);
	"read" =>
		return doread(rest);
	"append" =>
		return doappend(rest);
	"ctl" =>
		return doctl(rest);
	"colors" =>
		return docolors(rest);
	"delete" =>
		return dodelete(rest);
	"list" =>
		return dolist();
	"status" =>
		return dostatus(rest);
	* =>
		return sys->sprint("error: unknown command '%s'", cmd);
	}
}

# Create a new window
docreate(args: string): string
{
	winname := strip(args);

	# Create window by writing to new/ctl
	newctl := XENITH_ROOT + "/new/ctl";
	fd := sys->open(newctl, Sys->ORDWR);
	if(fd == nil)
		return sys->sprint("error: cannot open %s: %r (is Xenith running?)", newctl);

	# Write empty string to create, read back ID
	sys->write(fd, array[0] of byte, 0);

	buf := array[64] of byte;
	n := sys->read(fd, buf, len buf);
	fd = nil;

	if(n <= 0)
		return "error: failed to create window";

	winid := strip(string buf[0:n]);

	# Set name if provided
	if(winname != "") {
		ctlpath := sys->sprint("%s/%s/ctl", XENITH_ROOT, winid);
		ctlfd := sys->open(ctlpath, Sys->OWRITE);
		if(ctlfd != nil) {
			namecmd := sys->sprint("name %s\n", winname);
			sys->write(ctlfd, array of byte namecmd, len namecmd);
			ctlfd = nil;
		}
	}

	return winid;
}

# Write to window body or tag
dowrite(args: string): string
{
	(winid, rest) := splitfirst(args);
	if(winid == "")
		return "error: usage: write <id> body|tag <text>";

	(target, text) := splitfirst(rest);
	target = str->tolower(target);

	if(target == "ctl")
		return "error: use 'xenith ctl <id> <command>' instead of write";
	if(target != "body" && target != "tag")
		return "error: target must be 'body' or 'tag'. Use 'xenith ctl' for control, 'xenith delete' to close";

	filepath := sys->sprint("%s/%s/%s", XENITH_ROOT, winid, target);
	fd := sys->open(filepath, Sys->OWRITE | Sys->OTRUNC);
	if(fd == nil)
		return sys->sprint("error: cannot open %s: %r", filepath);

	data := array of byte text;
	n := sys->write(fd, data, len data);
	fd = nil;

	if(n != len data)
		return sys->sprint("error: write failed: %r");

	return sys->sprint("wrote %d bytes to %s/%s", n, winid, target);
}

# Read from window body or tag
doread(args: string): string
{
	(winid, rest) := splitfirst(args);
	if(winid == "")
		return "error: usage: read <id> [body|tag]";

	target := strip(rest);
	if(target == "")
		target = "body";
	target = str->tolower(target);

	if(target == "ctl")
		return "error: use 'xenith ctl <id> <command>' for control commands";
	if(target != "body" && target != "tag")
		return "error: target must be 'body' or 'tag'";

	filepath := sys->sprint("%s/%s/%s", XENITH_ROOT, winid, target);
	fd := sys->open(filepath, Sys->OREAD);
	if(fd == nil)
		return sys->sprint("error: cannot open %s: %r", filepath);

	# Read all content
	result := "";
	buf := array[8192] of byte;
	for(;;) {
		n := sys->read(fd, buf, len buf);
		if(n <= 0)
			break;
		result += string buf[0:n];
	}
	fd = nil;

	return result;
}

# Append to window body
doappend(args: string): string
{
	(winid, text) := splitfirst(args);
	if(winid == "" || text == "")
		return "error: usage: append <id> <text>";

	filepath := sys->sprint("%s/%s/body", XENITH_ROOT, winid);
	fd := sys->open(filepath, Sys->OWRITE);
	if(fd == nil)
		return sys->sprint("error: cannot open %s: %r", filepath);

	# Seek to end
	sys->seek(fd, big 0, Sys->SEEKEND);

	data := array of byte text;
	n := sys->write(fd, data, len data);
	fd = nil;

	if(n != len data)
		return sys->sprint("error: append failed: %r");

	return sys->sprint("appended %d bytes", n);
}

# Send control commands
doctl(args: string): string
{
	(winid, cmds) := splitfirst(args);
	if(winid == "" || cmds == "")
		return "error: usage: ctl <id> <commands>";

	filepath := sys->sprint("%s/%s/ctl", XENITH_ROOT, winid);
	fd := sys->open(filepath, Sys->OWRITE);
	if(fd == nil)
		return sys->sprint("error: cannot open %s: %r", filepath);

	# Ensure commands end with newline
	if(len cmds > 0 && cmds[len cmds - 1] != '\n')
		cmds += "\n";

	data := array of byte cmds;
	n := sys->write(fd, data, len data);
	fd = nil;

	if(n != len data)
		return sys->sprint("error: ctl write failed: %r");

	return "ok";
}

# Set window colors
docolors(args: string): string
{
	(winid, settings) := splitfirst(args);
	if(winid == "" || settings == "")
		return "error: usage: colors <id> <settings>";

	filepath := sys->sprint("%s/%s/colors", XENITH_ROOT, winid);
	fd := sys->open(filepath, Sys->OWRITE);
	if(fd == nil)
		return sys->sprint("error: cannot open %s: %r", filepath);

	# Ensure settings end with newline
	if(len settings > 0 && settings[len settings - 1] != '\n')
		settings += "\n";

	data := array of byte settings;
	n := sys->write(fd, data, len data);
	fd = nil;

	if(n != len data)
		return sys->sprint("error: colors write failed: %r");

	return "ok";
}

# Delete a window
dodelete(args: string): string
{
	winid := strip(args);
	if(winid == "")
		return "error: usage: delete <id>";

	filepath := sys->sprint("%s/%s/ctl", XENITH_ROOT, winid);
	fd := sys->open(filepath, Sys->OWRITE);
	if(fd == nil)
		return sys->sprint("error: cannot open %s: %r", filepath);

	data := array of byte "delete\n";
	sys->write(fd, data, len data);
	fd = nil;

	return "ok";
}

# List all windows
dolist(): string
{
	filepath := XENITH_ROOT + "/index";
	fd := sys->open(filepath, Sys->OREAD);
	if(fd == nil)
		return sys->sprint("error: cannot open %s: %r (is Xenith running?)", filepath);

	result := "";
	buf := array[8192] of byte;
	for(;;) {
		n := sys->read(fd, buf, len buf);
		if(n <= 0)
			break;
		result += string buf[0:n];
	}
	fd = nil;

	if(result == "")
		return "(no windows)";
	return result;
}

# Set visual status indicator via colors
dostatus(args: string): string
{
	(winid, state) := splitfirst(args);
	if(winid == "" || state == "")
		return "error: usage: status <id> ok|warn|error|info|reset";

	state = str->tolower(strip(state));

	# Define status colors (Catppuccin-inspired)
	colorstr: string;
	case state {
	"ok" or "success" or "green" =>
		# Green tag for success
		colorstr = "tagbg #A6E3A1\ntagfg #1E1E2E\n";
	"warn" or "warning" or "yellow" =>
		# Yellow tag for warning
		colorstr = "tagbg #F9E2AF\ntagfg #1E1E2E\n";
	"error" or "fail" or "red" =>
		# Red tag for error
		colorstr = "tagbg #F38BA8\ntagfg #1E1E2E\n";
	"info" or "blue" =>
		# Blue tag for information
		colorstr = "tagbg #89B4FA\ntagfg #1E1E2E\n";
	"reset" or "default" =>
		colorstr = "reset\n";
	* =>
		return sys->sprint("error: unknown status '%s'. Use: ok, warn, error, info, reset", state);
	}

	filepath := sys->sprint("%s/%s/colors", XENITH_ROOT, winid);
	fd := sys->open(filepath, Sys->OWRITE);
	if(fd == nil)
		return sys->sprint("error: cannot open %s: %r", filepath);

	data := array of byte colorstr;
	n := sys->write(fd, data, len data);
	fd = nil;

	if(n != len data)
		return sys->sprint("error: status write failed: %r");

	return "ok";
}

# Helper: strip whitespace
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

# Helper: split on first whitespace
splitfirst(s: string): (string, string)
{
	s = strip(s);
	for(i := 0; i < len s; i++) {
		if(s[i] == ' ' || s[i] == '\t')
			return (s[0:i], strip(s[i:]));
	}
	return (s, "");
}
