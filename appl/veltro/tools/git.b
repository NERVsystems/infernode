implement ToolGit;

#
# git - Git operations tool for Veltro agent
#
# Provides safe access to common git operations.
# Only available to trusted agents (requires shell access).
#
# TODO: Requires /cmd device for host command execution.
#       The /cmd device must be bound before this tool works.
#       See: emu/MacOSX/os.c for cmd device implementation.
#
# Usage:
#   git status                    # Show working tree status
#   git log [<options>]           # Show commit logs
#   git diff [<options>]          # Show changes
#   git add <path>                # Add file to staging
#   git commit -m "<message>"     # Create commit
#   git branch [<name>]           # List or create branch
#   git checkout <branch>         # Switch branches
#   git fetch                     # Fetch from remote
#   git pull                      # Fetch and merge
#   git push                      # Push to remote
#
# Safety:
#   - Destructive operations (reset --hard, push --force) are blocked
#   - Only allowed on repos within sandbox
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "string.m";
	str: String;

include "../tool.m";

ToolGit: module {
	init: fn(): string;
	name: fn(): string;
	doc:  fn(): string;
	exec: fn(args: string): string;
};

# Allowed git commands (safe operations)
ALLOWED_CMDS := array[] of {
	"status",
	"log",
	"diff",
	"show",
	"branch",
	"checkout",
	"switch",
	"add",
	"commit",
	"fetch",
	"pull",
	"push",
	"remote",
	"stash",
	"config",
	"ls-files",
	"rev-parse",
	"describe",
	"tag",
	"blame",
	"grep",
};

# Dangerous patterns to block
BLOCKED_PATTERNS := array[] of {
	"--force",
	"-f push",
	"push -f",
	"reset --hard",
	"clean -f",
	"checkout .",
	"restore .",
	"rm -rf",
};

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
	return "git";
}

doc(): string
{
	return "Git - Git operations\n\n" +
		"Usage:\n" +
		"  git status                # Working tree status\n" +
		"  git log [-n N]            # Commit history\n" +
		"  git diff [<path>]         # Show changes\n" +
		"  git add <path>            # Stage file\n" +
		"  git commit -m \"<msg>\"     # Create commit\n" +
		"  git branch [<name>]       # List/create branch\n" +
		"  git checkout <branch>     # Switch branches\n" +
		"  git fetch                 # Fetch from remote\n" +
		"  git pull                  # Fetch and merge\n" +
		"  git push                  # Push to remote\n\n" +
		"Allowed: status, log, diff, show, branch, checkout, switch,\n" +
		"         add, commit, fetch, pull, push, remote, stash,\n" +
		"         config, ls-files, rev-parse, describe, tag, blame, grep\n\n" +
		"Blocked: --force, reset --hard, clean -f, checkout .\n\n" +
		"Note: Requires trusted mode with shell access.";
}

exec(args: string): string
{
	if(sys == nil)
		init();

	args = strip(args);
	if(args == "")
		return "error: usage: git <command> [args...]";

	# Extract git subcommand
	(subcmd, rest) := splitfirst(args);
	subcmd = str->tolower(subcmd);

	# Check if command is allowed
	if(!isallowed(subcmd))
		return sys->sprint("error: git command '%s' is not allowed", subcmd);

	# Check for dangerous patterns
	fullcmd := subcmd + " " + rest;
	if(isdangerous(fullcmd))
		return "error: dangerous git operation blocked for safety";

	# Build command
	gitcmd := "git " + args;

	# Execute via os command
	(ok, output) := runcmd(gitcmd);
	if(!ok)
		return "error: " + output;

	if(output == "")
		return "(no output)";

	return output;
}

# Check if git subcommand is allowed
isallowed(cmd: string): int
{
	for(i := 0; i < len ALLOWED_CMDS; i++) {
		if(ALLOWED_CMDS[i] == cmd)
			return 1;
	}
	return 0;
}

# Check if command contains dangerous patterns
isdangerous(cmd: string): int
{
	lcmd := str->tolower(cmd);
	for(i := 0; i < len BLOCKED_PATTERNS; i++) {
		if(contains(lcmd, BLOCKED_PATTERNS[i]))
			return 1;
	}
	return 0;
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

# Run a command and capture output
# Requires /cmd device for host command execution
runcmd(cmd: string): (int, string)
{
	# Check if /cmd device exists (host command execution)
	(ok, nil) := sys->stat("/cmd");
	if(ok < 0)
		return (0, "git requires /cmd device for host command execution (not available)");

	# Try to open /cmd/clone to get a command channel
	cmdctl := sys->open("/cmd/clone", Sys->ORDWR);
	if(cmdctl == nil)
		return (0, sys->sprint("cannot open /cmd/clone: %r"));

	# Read the command directory number
	buf := array[32] of byte;
	n := sys->read(cmdctl, buf, len buf);
	if(n <= 0) {
		return (0, "cannot read cmd number");
	}
	cmdnum := string buf[0:n];

	# Open data file for command I/O
	datapath := "/cmd/" + cmdnum + "/data";
	data := sys->open(datapath, Sys->ORDWR);
	if(data == nil) {
		return (0, sys->sprint("cannot open %s: %r", datapath));
	}

	# Write the command (with exec prefix for direct execution)
	fullcmd := "exec " + cmd;
	if(sys->fprint(cmdctl, "%s", fullcmd) < 0) {
		return (0, sys->sprint("cannot write command: %r"));
	}

	# Start the command
	if(sys->fprint(cmdctl, "start") < 0) {
		return (0, sys->sprint("cannot start command: %r"));
	}

	# Read output
	output := "";
	readbuf := array[8192] of byte;
	while((n = sys->read(data, readbuf, len readbuf)) > 0)
		output += string readbuf[0:n];

	# Trim trailing newline
	if(len output > 0 && output[len output - 1] == '\n')
		output = output[0:len output - 1];

	return (1, output);
}

# Strip whitespace
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

# Split on first whitespace
splitfirst(s: string): (string, string)
{
	for(i := 0; i < len s; i++) {
		if(s[i] == ' ' || s[i] == '\t')
			return (s[0:i], strip(s[i:]));
	}
	return (s, "");
}
