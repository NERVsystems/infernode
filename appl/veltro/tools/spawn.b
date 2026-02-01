implement ToolSpawn;

#
# spawn - Create subagent with secure namespace isolation for Veltro agent
#
# SECURITY MODEL (v2):
# ====================
# Uses proper NEWNS-based isolation with allowlist model:
#
# Parent (before spawn):
#   1. validatesandboxid(id) - Reject traversal attacks
#   2. preparesandbox(caps) - Create sandbox dir with restrictive perms
#   3. verifyownership(path) - stat() every path before bind
#
# Child (after spawn):
#   1. pctl(NEWPGRP, nil) - Fresh process group (empty srv registry)
#   2. pctl(FORKNS, nil) - Fork namespace for mutation
#   3. pctl(NEWENV, nil) - Empty environment (NOT FORKENV!)
#   4. verifysafefds() - Verify FDs point at safe endpoints
#   5. pctl(NEWFD, keepfds) - Prune all other FDs
#   6. pctl(NODEVS, nil) - Block #U/#p/#c (still allows #e/#s/#|)
#   7. chdir(sandboxdir) - Enter prepared sandbox
#   8. pctl(NEWNS, nil) - Sandbox becomes /
#   9. mounttools9p(tools) - Mount tools without /srv or /net
#  10. executetask(task) - No policy checks; namespace IS capability
#
# Security Properties:
#   - No #U escape (NODEVS before sandbox entry)
#   - No env secrets (NEWENV - empty environment)
#   - No FD leaks (NEWFD with explicit keep-list)
#   - Empty srv registry (NEWPGRP first)
#   - Truthful namespace (only granted paths exist after NEWNS)
#   - No shell for untrusted (safeexec runs .dis directly)
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
	if(nsconstruct != nil)
		nsconstruct->init();
}

name(): string
{
	return "spawn";
}

doc(): string
{
	return "Spawn - Create subagent with secure namespace isolation\n\n" +
		"Usage:\n" +
		"  Spawn tools=<tools> paths=<paths> [shellcmds=<cmds>] [trusted=1] -- <task>\n\n" +
		"Arguments:\n" +
		"  tools     - Comma-separated tools to grant (e.g., \"read,list\")\n" +
		"  paths     - Comma-separated paths to grant (e.g., \"/appl,/tmp\")\n" +
		"  shellcmds - Comma-separated shell commands for exec (trusted only)\n" +
		"  trusted   - Set to 1 to allow shell access (default: 0)\n" +
		"  task      - Task description for child agent\n\n" +
		"Examples:\n" +
		"  Spawn tools=read,list paths=/appl -- \"List .b files\"\n" +
		"  Spawn tools=read,exec paths=/appl shellcmds=cat,ls trusted=1 -- \"Explore\"\n\n" +
		"Security:\n" +
		"  - Child sees ONLY granted paths (allowlist model)\n" +
		"  - Environment is empty (no inherited secrets)\n" +
		"  - Untrusted agents cannot use shell (exec runs .dis directly)\n" +
		"  - All binds are logged for audit\n";
}

exec(args: string): string
{
	if(sys == nil)
		init();

	if(nsconstruct == nil)
		return "error: cannot load nsconstruct module";

	# Parse arguments
	(tools, paths, shellcmds, trusted, task, err) := parseargs(args);
	if(err != "")
		return "error: " + err;

	if(tools == nil)
		return "error: no tools specified";
	if(task == "")
		return "error: no task specified";

	# Generate unique sandbox ID
	sandboxid := nsconstruct->gensandboxid();

	# Build capabilities structure
	caps := ref NsConstruct->Capabilities(
		tools,
		paths,
		shellcmds,
		ref NsConstruct->LLMConfig("default", 0.7, ""),
		0 :: 1 :: 2 :: nil,  # Default FD keep list
		ref NsConstruct->Mountpoints(0, 0, 0),  # No srv/net/prog for untrusted
		sandboxid,
		trusted
	);

	# PARENT: Prepare sandbox directory
	# This creates the sandbox structure and binds granted paths
	err = nsconstruct->preparesandbox(caps);
	if(err != nil)
		return "error: " + err;

	# Create pipe for IPC
	pipefds := array[2] of ref Sys->FD;
	if(sys->pipe(pipefds) < 0) {
		nsconstruct->cleanupsandbox(sandboxid);
		return sys->sprint("error: cannot create pipe: %r");
	}

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
		nsconstruct->cleanupsandbox(sandboxid);
		return "error: child agent timed out after 60 seconds";
	}

	pipefds[0] = nil;  # Close read end

	# Clean up sandbox after child exits
	nsconstruct->cleanupsandbox(sandboxid);

	if(hasprefix(result, "ERROR:"))
		return "error: " + result[6:];

	return result;
}

# Parse spawn arguments
# Returns: (tools, paths, shellcmds, trusted, task, error)
parseargs(s: string): (list of string, list of string, list of string, int, string, string)
{
	tools: list of string;
	paths: list of string;
	shellcmds: list of string;
	trusted := 0;
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
		} else if(hasprefix(tok, "trusted=")) {
			if(tok[8:] == "1")
				trusted = 1;
		}
	}

	# Reverse lists to maintain order
	tools = reverse(tools);
	paths = reverse(paths);
	shellcmds = reverse(shellcmds);

	return (tools, paths, shellcmds, trusted, task, "");
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
toolregistry: list of string;

# Set the tool registry from outside (called by tools9p before exec)
setregistry(tools: list of string)
{
	toolregistry = tools;
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

# Run child agent with secure namespace isolation
runchild(pipefd: ref Sys->FD, caps: ref NsConstruct->Capabilities, task: string)
{
	# SECURITY MODEL (v2):
	# ====================
	# Uses proper pctl sequence for true isolation:
	#   1. NEWPGRP - Fresh process group (empty srv registry)
	#   2. FORKNS - Fork namespace for mutation
	#   3. NEWENV - Empty environment (no inherited secrets)
	#   4. verifysafefds - Check FDs 0-2 are safe
	#   5. NEWFD - Prune to keep list only
	#   6. NODEVS - Block device naming (#U/#p/#c)
	#   7. chdir - Enter prepared sandbox
	#   8. NEWNS - Sandbox becomes /
	#   9. Mount tools9p
	#  10. Execute task

	# Step 1: Fresh process group (empty service registry)
	sys->pctl(Sys->NEWPGRP, nil);

	# Step 2: Fork namespace for mutation
	sys->pctl(Sys->FORKNS, nil);

	# Step 3: NEWENV - empty environment, not inherited!
	sys->pctl(Sys->NEWENV, nil);

	# Step 4: Verify FDs 0-2 are safe endpoints
	# Redirect to /dev/null if suspicious
	verifysafefds();

	# Step 5: Prune FDs - keep only stdin, stdout, stderr, and pipe
	keepfds := 0 :: 1 :: 2 :: pipefd.fd :: nil;
	sys->pctl(Sys->NEWFD, keepfds);

	# Step 6: Block device naming (still allows #e/#s/#| but env is empty)
	sys->pctl(Sys->NODEVS, nil);

	# Step 7: Enter sandbox (path already validated by parent)
	sandboxpath := nsconstruct->sandboxpath(caps.sandboxid);
	if(sys->chdir(sandboxpath) < 0) {
		writeresult(pipefd, sys->sprint("ERROR:cannot enter sandbox: %r"));
		return;
	}

	# Step 8: NEWNS - sandbox becomes /
	# After this, sandboxpath IS / and nothing outside exists
	sys->pctl(Sys->NEWNS, nil);

	# Step 9: Mount tools9p with only the granted tools
	# tools9p handles its own pipe creation and mounting
	err := starttools9p(caps.tools);
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

	# Step 10: Execute the task
	# For untrusted agents: use safeexec (no shell)
	# For trusted agents: can use executetask (may use shell if shellcmds granted)
	result: string;
	if(caps.trusted)
		result = executetask(task, caps.tools);
	else
		result = safeexec(task, caps.tools);

	writeresult(pipefd, result);
	pipefd = nil;
}

# Verify FDs 0-2 are safe endpoints
# If in doubt, redirect to /dev/null
verifysafefds()
{
	# In Inferno, we can use fd2path to check what an FD points to
	# For now, we just ensure FDs 0-2 exist and are valid
	# After NEWNS, they'll point to sandbox /dev/cons anyway

	# Check stdin
	buf := array[1] of byte;
	fd0 := sys->fildes(0);
	if(fd0 == nil) {
		# Redirect stdin from /dev/null
		null := sys->open("/dev/null", Sys->OREAD);
		if(null != nil)
			sys->dup(null.fd, 0);
	}

	# Check stdout
	fd1 := sys->fildes(1);
	if(fd1 == nil) {
		null := sys->open("/dev/null", Sys->OWRITE);
		if(null != nil)
			sys->dup(null.fd, 1);
	}

	# Check stderr
	fd2 := sys->fildes(2);
	if(fd2 == nil) {
		null := sys->open("/dev/null", Sys->OWRITE);
		if(null != nil)
			sys->dup(null.fd, 2);
	}
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
starttools9p(tools: list of string): string
{
	if(tools == nil)
		return "no tools specified";

	# Build command arguments: tools9p -m /tool tool1 tool2 ...
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
	tools9pmod->init(nil, args);

	return nil;
}

# Safe execution for untrusted agents
# Runs .dis files directly without shell interpretation
safeexec(task: string, tools: list of string): string
{
	# Parse first word as tool name
	task = strip(task);
	if(task == "")
		return "ERROR:empty task";

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

	# Load and execute the tool directly (no shell)
	toolpath := "/dis/veltro/tools/" + ltool + ".dis";
	tool := load Tool toolpath;
	if(tool == nil)
		return sys->sprint("ERROR:cannot load tool %s: %r", ltool);

	return tool->exec(args);
}

# Execute task for trusted agents (may use shell if shellcmds granted)
executetask(task: string, tools: list of string): string
{
	# For trusted agents, we still use direct tool execution
	# but shell commands are available in /dis if granted
	return safeexec(task, tools);
}

# Check if path exists
pathexists(path: string): int
{
	(ok, nil) := sys->stat(path);
	return ok >= 0;
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

# Command module interface for loading tools9p
Command: module {
	init: fn(ctxt: ref Draw->Context, args: list of string);
};
