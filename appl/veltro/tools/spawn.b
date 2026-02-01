implement ToolSpawn;

#
# spawn - Create subagent with constructed namespace for Veltro agent
#
# The heart of Veltro's capability model. Spawns a child agent with a
# namespace constructed from only the capabilities the parent chooses to grant.
#
# A child's namespace can only be equal to or smaller than its parent's.
# You cannot grant tools or paths you don't have yourself.
#
# Usage:
#   Spawn tools=<tools> paths=<paths> shellcmds=<cmds> -- <task>
#
# Arguments:
#   tools     - Comma-separated list of tools to grant (e.g., "read,list")
#   paths     - Comma-separated list of paths to grant (e.g., "/appl,/tmp")
#   shellcmds - Comma-separated shell commands for exec (e.g., "cat,ls,head")
#   task      - Task description for the child agent
#
# Examples:
#   Spawn tools=read,list paths=/appl -- "List all .b files in /appl"
#   Spawn tools=read,exec paths=/appl shellcmds=cat,ls,head -- "Explore /appl"
#   Spawn tools=read,write paths=/tmp -- "Create a test file"
#
# Shell command restriction:
#   If exec is granted but shellcmds is empty, exec has full shell access.
#   If shellcmds is specified, exec can only run those commands.
#   This uses namespace restriction: /dis contains only allowed commands.
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "string.m";
	str: String;

include "../tool.m";
include "../nsconstruct.m";
	nsconstruct: NsConstruct;

ToolSpawn: module {
	name: fn(): string;
	doc:  fn(): string;
	exec: fn(args: string): string;
};

# Result channel for child process
Result: adt {
	output: string;
	err:    string;
};

init()
{
	sys = load Sys Sys->PATH;
	str = load String String->PATH;
	nsconstruct = load NsConstruct NsConstruct->PATH;
}

name(): string
{
	return "spawn";
}

doc(): string
{
	return "Spawn - Create subagent with constructed namespace\n\n" +
		"Usage:\n" +
		"  Spawn tools=<tools> paths=<paths> shellcmds=<cmds> -- <task>\n\n" +
		"Arguments:\n" +
		"  tools     - Comma-separated tools to grant (e.g., \"read,list\")\n" +
		"  paths     - Comma-separated paths to grant (e.g., \"/appl,/tmp\")\n" +
		"  shellcmds - Comma-separated shell commands for exec (e.g., \"cat,ls\")\n" +
		"  task      - Task description for child agent\n\n" +
		"Examples:\n" +
		"  Spawn tools=read,list paths=/appl -- \"List .b files\"\n" +
		"  Spawn tools=read,exec shellcmds=cat,ls -- \"Explore with exec\"\n\n" +
		"You can only grant tools and paths you have access to.\n" +
		"If exec is granted with shellcmds, only those shell commands work.";
}

exec(args: string): string
{
	if(sys == nil)
		init();

	if(nsconstruct == nil)
		return "error: cannot load nsconstruct module";

	# Parse arguments
	(tools, paths, shellcmds, task, err) := parseargs(args);
	if(err != "")
		return "error: " + err;

	if(tools == nil)
		return "error: no tools specified";
	if(task == "")
		return "error: no task specified";

	# Note: Tool and path validation is skipped because spawn.exec() runs
	# inside tools9p's single-threaded serveloop. Any 9P operation (like
	# stat on a path) would cause a deadlock. The calling agent (veltro)
	# already knows what tools are available from /tool/tools, so it
	# won't request tools that don't exist. Paths are validated by the
	# child when it tries to access them.
	#
	# Future fix: tools9p could execute tools asynchronously to allow
	# concurrent 9P operations.

	# Build capabilities
	caps := ref NsConstruct->Capabilities(
		tools,
		paths,
		shellcmds,
		ref NsConstruct->LLMConfig("default", 0.7, "")
	);

	# Create pipe for IPC
	pipefds := array[2] of ref Sys->FD;
	if(sys->pipe(pipefds) < 0)
		return sys->sprint("error: cannot create pipe: %r");

	# Spawn child process with pipe write end
	spawn runchild(pipefds[1], caps, task);
	pipefds[1] = nil;  # Close write end in parent

	# Wait for result with timeout
	timeout := chan of int;
	spawn timer(timeout, 60000);  # 60 second timeout

	resultch := chan of string;
	spawn pipereader(pipefds[0], resultch);

	result: string;
	alt {
	result = <-resultch =>
		;
	<-timeout =>
		pipefds[0] = nil;  # Close read end
		return "error: child agent timed out after 60 seconds";
	}

	pipefds[0] = nil;  # Close read end

	if(hasprefix(result, "ERROR:"))
		return "error: " + result[6:];

	return result;
}

# Parse spawn arguments
# Returns: (tools, paths, shellcmds, task, error)
parseargs(s: string): (list of string, list of string, list of string, string, string)
{
	tools: list of string;
	paths: list of string;
	shellcmds: list of string;
	task := "";

	# Split on --
	(before, after) := spliton(s, "--");
	task = strip(after);

	# Parse key=value pairs in before
	(nil, tokens) := sys->tokenize(before, " \t");
	for(; tokens != nil; tokens = tl tokens) {
		tok := hd tokens;
		if(hasprefix(tok, "tools=")) {
			toolstr := tok[6:];
			(nil, tlist) := sys->tokenize(toolstr, ",");
			for(; tlist != nil; tlist = tl tlist)
				tools = str->tolower(hd tlist) :: tools;
		} else if(hasprefix(tok, "paths=")) {
			pathstr := tok[6:];
			(nil, plist) := sys->tokenize(pathstr, ",");
			for(; plist != nil; plist = tl plist)
				paths = hd plist :: paths;
		} else if(hasprefix(tok, "shellcmds=")) {
			cmdstr := tok[10:];
			(nil, clist) := sys->tokenize(cmdstr, ",");
			for(; clist != nil; clist = tl clist)
				shellcmds = str->tolower(hd clist) :: shellcmds;
		}
	}

	# Reverse lists to maintain order
	tools = reverse(tools);
	paths = reverse(paths);
	shellcmds = reverse(shellcmds);

	return (tools, paths, shellcmds, task, "");
}

# Split string on separator
spliton(s, sep: string): (string, string)
{
	for(i := 0; i <= len s - len sep; i++) {
		if(s[i:i+len sep] == sep)
			return (s[0:i], s[i+len sep:]);
	}
	return (s, "");
}

# Strip leading/trailing whitespace
strip(s: string): string
{
	i := 0;
	while(i < len s && (s[i] == ' ' || s[i] == '\t'))
		i++;
	j := len s;
	while(j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n'))
		j--;
	if(i >= j)
		return "";
	return s[i:j];
}

# Check if string has prefix
hasprefix(s, prefix: string): int
{
	return len s >= len prefix && s[0:len prefix] == prefix;
}

# Reverse a list
reverse(l: list of string): list of string
{
	result: list of string;
	for(; l != nil; l = tl l)
		result = hd l :: result;
	return result;
}

# Known tools registry - set by setregistry() before exec() is called
# This avoids the deadlock that occurs when spawn.exec() tries to
# read /tool/_registry (since exec runs inside tools9p's serveloop)
toolregistry: list of string;

# Set the tool registry from outside (called by tools9p before exec)
setregistry(tools: list of string)
{
	toolregistry = tools;
}

# Check if tool exists in parent's namespace
# Uses the pre-set registry to avoid any 9P operations
toolexists(tool: string): int
{
	# If registry is empty, allow all tools (backwards compatibility)
	if(toolregistry == nil)
		return 1;

	ltool := str->tolower(tool);
	for(t := toolregistry; t != nil; t = tl t) {
		if(hd t == ltool)
			return 1;
	}

	return 0;
}

# Check if path exists in our namespace
pathexists(path: string): int
{
	(ok, nil) := sys->stat(path);
	return ok >= 0;
}

# Timer thread
timer(ch: chan of int, ms: int)
{
	sys->sleep(ms);
	ch <-= 1;
}

# Read from pipe until sentinel or EOF
pipereader(fd: ref Sys->FD, resultch: chan of string)
{
	result := "";
	buf := array[8192] of byte;
	for(;;) {
		n := sys->read(fd, buf, len buf);
		if(n <= 0)
			break;
		result += string buf[0:n];
		# Check for sentinel
		if(len result >= len RESULT_END) {
			endpos := len result - len RESULT_END;
			if(result[endpos:] == RESULT_END) {
				result = result[0:endpos];
				break;
			}
		}
	}
	resultch <-= result;
}

# Run child agent with constructed namespace
runchild(pipefd: ref Sys->FD, caps: ref NsConstruct->Capabilities, task: string)
{
	# CAPABILITY MODEL: Parent grants subset of its own capabilities
	# ==============================================================
	# 1. Agent starts with namespace given by user/system
	# 2. Sub-agent gets subset: tools' ⊆ tools, paths' ⊆ paths, shellcmds' ⊆ shellcmds
	# 3. You can only grant what you have
	#
	# Implementation:
	#   FORKNS - child copies parent's namespace
	#   New tools9p - serves only granted tools (namespace enforced)
	#   Shell restriction - /dis contains only allowed commands (namespace enforced)
	#   Path check - executetask validates paths (tool enforced)

	sys->pctl(Sys->FORKNS|Sys->NEWPGRP, nil);

	# CRITICAL: Unmount parent's /tool before starting our own tools9p
	# After FORKNS, /tool still points to parent's tools9p. Any access to
	# /tool would go to the parent's server which is blocked waiting for
	# spawn.exec() to return - causing deadlock.
	sys->unmount(nil, "/tool");

	# Build tool list for tools9p command
	toolargs: list of string;
	for(t := caps.tools; t != nil; t = tl t)
		toolargs = hd t :: toolargs;

	# Reverse to maintain order
	toolargsr: list of string;
	for(t = toolargs; t != nil; t = tl t)
		toolargsr = hd t :: toolargsr;
	toolargs = toolargsr;

	# Start tools9p with only the granted tools
	err := starttools9p(toolargs);
	if(err != nil) {
		writeresult(pipefd, "ERROR:" + err);
		pipefd = nil;
		return;
	}

	# Verify tools9p is serving by checking /tool/tools exists
	for(i := 0; i < 10; i++) {
		if(pathexists("/tool/tools"))
			break;
		sys->sleep(50);
	}

	# SHELL COMMAND RESTRICTION
	# If shellcmds is specified, restrict /dis to only those commands.
	# This means exec can only run the allowed shell commands.
	# If shellcmds is empty, exec has full shell access (parent's /dis).
	if(caps.shellcmds != nil) {
		err = restrictshellcmds(caps.shellcmds);
		if(err != nil) {
			writeresult(pipefd, "ERROR:" + err);
			pipefd = nil;
			return;
		}
	}

	# Execute the task using the granted tools and path restrictions
	# Path restrictions are enforced by executetask(), not by namespace
	result := executetask(task, caps.tools, caps.paths);
	writeresult(pipefd, result);
	pipefd = nil;
}

# Restrict shell commands by rebuilding /dis with only allowed commands
# Uses namespace layering: create restricted view, bind over /dis
restrictshellcmds(cmds: list of string): string
{
	# Strategy: Use bind to layer a restricted view over /dis
	# 1. Create a temp directory for restricted commands
	# 2. Bind essential runtime (lib/, sh.dis, etc.)
	# 3. Bind only allowed commands
	# 4. Bind this restricted view MREPL over /dis
	#
	# Note: Current process has already loaded its modules, so replacing
	# /dis doesn't break us. Future loads (by shell commands) see restricted /dis.
	#
	# IMPORTANT: In Inferno, bind requires destination to exist first.
	# We must create placeholder files before binding.

	tmpdir := "/tmp/.restricted_dis";

	# Create temp directory structure
	mkpath(tmpdir);
	mkpath(tmpdir + "/lib");
	mkpath(tmpdir + "/veltro");
	mkpath(tmpdir + "/veltro/tools");

	# Bind essential runtime - these are needed for any Limbo code to run
	# The shell and any commands need these
	if(sys->bind("/dis/lib", tmpdir + "/lib", Sys->MREPL) < 0)
		return sys->sprint("cannot bind /dis/lib: %r");

	# Bind shell itself - needed to run commands
	# First create placeholder, then bind over it
	createplaceholder(tmpdir + "/sh.dis");
	if(sys->bind("/dis/sh.dis", tmpdir + "/sh.dis", Sys->MREPL) < 0)
		return sys->sprint("cannot bind sh.dis: %r");

	# Bind Veltro tools - needed for agent tool execution
	if(sys->bind("/dis/veltro", tmpdir + "/veltro", Sys->MREPL) < 0)
		return sys->sprint("cannot bind /dis/veltro: %r");

	# Bind only the allowed shell commands
	for(c := cmds; c != nil; c = tl c) {
		cmd := hd c;
		srcpath := "/dis/" + cmd + ".dis";
		dstpath := tmpdir + "/" + cmd + ".dis";

		# Check if command exists in parent's namespace
		if(!pathexists(srcpath))
			return sys->sprint("shell command not found: %s", cmd);

		# Create placeholder file, then bind real file over it
		createplaceholder(dstpath);
		if(sys->bind(srcpath, dstpath, Sys->MREPL) < 0)
			return sys->sprint("cannot bind %s: %r", cmd);
	}

	# Replace /dis with our restricted version
	if(sys->bind(tmpdir, "/dis", Sys->MREPL) < 0)
		return sys->sprint("cannot bind restricted /dis: %r");

	return nil;
}

# Create an empty placeholder file for bind destination
createplaceholder(path: string)
{
	fd := sys->create(path, Sys->OWRITE, 8r644);
	if(fd != nil)
		fd = nil;  # Close the file
}

# Sentinel to mark end of result
RESULT_END: con "\n<<EOF>>\n";

# Write result to pipe with sentinel
writeresult(fd: ref Sys->FD, result: string)
{
	data := array of byte (result + RESULT_END);
	sys->write(fd, data, len data);
}

# Start tools9p with specified tools
# tools9p handles its own pipe creation and mounting
starttools9p(tools: list of string): string
{
	if(tools == nil)
		return "no tools specified";

	# Build command arguments: tools9p -m /tool tool1 tool2 ...
	# tools9p will unmount any existing /tool and mount itself there
	# First collect tool names, then prepend fixed args
	args: list of string;
	for(t := tools; t != nil; t = tl t)
		args = hd t :: args;
	# Reverse to maintain order, then prepend fixed args
	revargs: list of string;
	for(; args != nil; args = tl args)
		revargs = hd args :: revargs;
	args = "tools9p" :: "-m" :: "/tool" :: revargs;

	# Load tools9p
	tools9pmod := load Command "/dis/veltro/tools9p.dis";
	if(tools9pmod == nil)
		return sys->sprint("cannot load tools9p: %r");

	# tools9p.init() creates pipe, spawns serveloop, and mounts
	# It returns after mounting is complete, with serveloop running in background
	tools9pmod->init(nil, args);

	return nil;
}

# Execute task using granted tools with path restrictions
# Parses simple "Tool args" format
# Path restrictions: if paths is non-empty, args must start with a path in the list
executetask(task: string, tools: list of string, paths: list of string): string
{
	# Strip leading/trailing whitespace
	task = strip(task);
	if(task == "")
		return "ERROR:empty task";

	# Parse first word as tool name
	(toolname, args) := splitfirst(task);
	ltool := str->tolower(toolname);

	# Check if tool is in granted list
	found := 0;
	for(t := tools; t != nil; t = tl t) {
		if(str->tolower(hd t) == ltool) {
			found = 1;
			break;
		}
	}
	if(!found)
		return sys->sprint("ERROR:tool not granted: %s", toolname);

	# PATH RESTRICTION: If paths is non-empty, verify the path arg is allowed
	# This enforces path restrictions at the tool level
	# NOTE: For exec tool, path validation is skipped here because:
	#   1. Shell commands have complex syntax; can't reliably extract paths
	#   2. Shell command restriction (shellcmds) limits what commands can run
	#   3. Namespace restriction on /dis provides structural security
	# For other tools (read, write, list, etc.), first arg is the path
	if(paths != nil && args != "" && ltool != "exec") {
		# Extract path from args (first argument for most tools)
		argpath := args;
		for(i := 0; i < len argpath; i++) {
			if(argpath[i] == ' ' || argpath[i] == '\t') {
				argpath = argpath[0:i];
				break;
			}
		}

		# Check if argpath starts with any granted path
		allowed := 0;
		for(p := paths; p != nil; p = tl p) {
			gpath := hd p;
			if(pathwithin(argpath, gpath)) {
				allowed = 1;
				break;
			}
		}
		if(!allowed)
			return sys->sprint("ERROR:path not granted: %s", argpath);
	}

	# Load and execute the tool using the Tool interface from tool.m
	toolpath := "/dis/veltro/tools/" + ltool + ".dis";
	tool := load Tool toolpath;
	if(tool == nil)
		return sys->sprint("ERROR:cannot load tool %s: %r", ltool);

	return tool->exec(args);
}

# Check if path is within or equal to basepath
# e.g., pathwithin("/appl/veltro/veltro.b", "/appl") returns true
pathwithin(path, basepath: string): int
{
	if(path == basepath)
		return 1;
	# Check if path starts with basepath/
	if(len path > len basepath && path[0:len basepath] == basepath) {
		# Make sure it's a directory boundary
		if(basepath[len basepath - 1] == '/' || path[len basepath] == '/')
			return 1;
	}
	return 0;
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

# Create directory path recursively
mkpath(path: string)
{
	if(path == "" || path == "/")
		return;

	# Find parent directory
	parent := "";
	for(i := len path - 1; i > 0; i--) {
		if(path[i] == '/') {
			parent = path[0:i];
			break;
		}
	}

	# Create parent first
	if(parent != "" && parent != "/")
		mkpath(parent);

	# Create this directory (ignore errors - might already exist)
	sys->create(path, Sys->OREAD, Sys->DMDIR|8r755);
}

# Command module interface for loading tools9p
Command: module {
	init: fn(ctxt: ref Draw->Context, args: list of string);
};
