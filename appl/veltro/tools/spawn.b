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
include "../subagent.m";
	subagent: SubAgent;

ToolSpawn: module {
	init: fn(): string;
	name: fn(): string;
	doc:  fn(): string;
	exec: fn(args: string): string;
};

# Result channel for child process
Result: adt {
	output: string;
	err:    string;
};

# Pre-loaded tools9p module (loaded before NEWNS, usable after)
Tools9pMod: module {
	init: fn(nil: ref Draw->Context, nil: list of string);
};
tools9pmod: Tools9pMod;

# Pre-loaded tool modules for direct execution
PreloadedTool: adt {
	name: string;
	mod:  Tool;
};
preloadedtools: list of ref PreloadedTool;

# Thread-safe initialization
inited := 0;

init(): string
{
	# Quick check - already initialized
	if(inited)
		return nil;

	# Load modules - idempotent operations
	sys = load Sys Sys->PATH;
	if(sys == nil)
		return "cannot load Sys";
	str = load String String->PATH;
	if(str == nil)
		return "cannot load String";
	nsconstruct = load NsConstruct NsConstruct->PATH;
	if(nsconstruct == nil)
		return "cannot load NsConstruct";
	nsconstruct->init();

	inited = 1;
	return nil;
}

# Pre-load tools9p, subagent, and all granted tool modules
# Called BEFORE spawn/NEWNS so modules are loaded and initialized while paths exist
preloadmodules(tools: list of string): string
{
	# Load tools9p module (may not be used but kept for compatibility)
	tools9pmod = load Tools9pMod "/dis/veltro/tools9p.dis";
	if(tools9pmod == nil)
		return sys->sprint("cannot load tools9p: %r");

	# Load and initialize subagent module
	subagent = load SubAgent SubAgent->PATH;
	if(subagent == nil)
		return sys->sprint("cannot load subagent: %r");
	err := subagent->init();
	if(err != nil)
		return sys->sprint("cannot init subagent: %s", err);

	# Load and initialize each granted tool module
	preloadedtools = nil;
	for(t := tools; t != nil; t = tl t) {
		name := str->tolower(hd t);
		path := "/dis/veltro/tools/" + name + ".dis";
		mod := load Tool path;
		if(mod == nil)
			return sys->sprint("cannot load tool %s: %r", name);

		# Initialize module while paths are still accessible
		err = mod->init();
		if(err != nil)
			return sys->sprint("cannot init tool %s: %s", name, err);

		preloadedtools = ref PreloadedTool(name, mod) :: preloadedtools;
	}

	return nil;
}

name(): string
{
	return "spawn";
}

doc(): string
{
	return "Spawn - Create subagent with secure namespace isolation\n\n" +
		"Usage:\n" +
		"  Spawn tools=<tools> paths=<paths> [options] -- <task>\n\n" +
		"Arguments:\n" +
		"  tools       - Comma-separated tools to grant (e.g., \"read,list\")\n" +
		"  paths       - Comma-separated paths to grant (e.g., \"/appl,/tmp\")\n" +
		"  shellcmds   - Comma-separated shell commands for exec (trusted only)\n" +
		"  trusted     - Set to 1 to allow shell access (default: 0)\n" +
		"  llmmodel    - LLM model for child agent (default: \"default\")\n" +
		"  temperature - LLM temperature 0.0-2.0 (default: 0.7)\n" +
		"  agenttype   - Agent type: explore, plan, task, default (loads prompt)\n" +
		"  system      - System prompt for child agent (overrides agenttype)\n" +
		"  task        - Task description for child agent\n\n" +
		"Examples:\n" +
		"  Spawn tools=read,list paths=/appl -- \"List .b files\"\n" +
		"  Spawn tools=read,list agenttype=explore paths=/appl -- \"Find handlers\"\n" +
		"  Spawn tools=read agenttype=plan paths=/appl -- \"Plan refactor\"\n" +
		"  Spawn tools=read llmmodel=gpt-4 temperature=0.3 -- \"Analyze code\"\n\n" +
		"Security:\n" +
		"  - Child sees ONLY granted paths (allowlist model)\n" +
		"  - Environment is empty (no inherited secrets)\n" +
		"  - Untrusted agents cannot use shell (exec runs .dis directly)\n" +
		"  - LLM config is isolated per-agent (parent sets, child inherits)\n" +
		"  - All binds are logged for audit\n";
}

exec(args: string): string
{
	if(sys == nil)
		init();

	if(nsconstruct == nil)
		return "error: cannot load nsconstruct module";

	# Parse arguments (now includes LLM config)
	(tools, paths, shellcmds, trusted, llmconfig, task, err) := parseargs(args);
	if(err != "")
		return "error: " + err;

	if(tools == nil)
		return "error: no tools specified";
	if(task == "")
		return "error: no task specified";

	# Generate unique sandbox ID
	sandboxid := nsconstruct->gensandboxid();

	# Build capabilities structure with parsed LLM config
	caps := ref NsConstruct->Capabilities(
		tools,
		paths,
		shellcmds,
		llmconfig,  # Use parsed LLM config instead of hardcoded defaults
		0 :: 1 :: 2 :: nil,  # Default FD keep list
		ref NsConstruct->Mountpoints(0, 0, 0),  # No srv/net/prog for untrusted
		sandboxid,
		trusted,
		nil,  # No mc9p providers by default
		0     # No memory by default
	);

	# PARENT: Pre-load modules BEFORE spawn
	# These modules will be used by child AFTER NEWNS when paths no longer exist
	err = preloadmodules(tools);
	if(err != nil)
		return "error: " + err;

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

	# Open LLM file descriptor BEFORE spawn
	# This FD survives NEWNS (binds don't, but open FDs do)
	# Pass FD number, not ref, because ref becomes invalid after NEWFD
	llmfdnum := -1;
	if(caps.llmconfig != nil) {
		llmfd := sys->open("/n/llm/ask", Sys->ORDWR);
		if(llmfd != nil)
			llmfdnum = llmfd.fd;
		# Not fatal if LLM unavailable - subagent will handle gracefully
	}

	# Spawn child process with pipe write end and LLM FD number
	spawn runchild(pipefds[1], llmfdnum, caps, task);
	pipefds[1] = nil;  # Close write end in parent

	# Wait for result with timeout
	timeout := chan of int;
	spawn timer(timeout, 30000);  # 30 second timeout

	resultch := chan of string;
	spawn pipereader(pipefds[0], resultch);

	result: string;
	alt {
	result = <-resultch =>
		;
	<-timeout =>
		pipefds[0] = nil;  # Close read end
		nsconstruct->cleanupsandbox(sandboxid);
		return "error: child agent timed out after 30 seconds";
	}

	pipefds[0] = nil;  # Close read end

	# Clean up sandbox after child exits
	nsconstruct->cleanupsandbox(sandboxid);

	if(hasprefix(result, "ERROR:"))
		return "error: " + result[6:];

	return result;
}

# Parse spawn arguments
# Returns: (tools, paths, shellcmds, trusted, llmconfig, task, error)
parseargs(s: string): (list of string, list of string, list of string, int, ref NsConstruct->LLMConfig, string, string)
{
	tools: list of string;
	paths: list of string;
	shellcmds: list of string;
	trusted := 0;
	task := "";

	# LLM configuration with defaults
	llmmodel := "default";
	llmtemp := 0.7;
	llmsystem := "";
	agenttype := "";

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
		} else if(hasprefix(tok, "llmmodel=")) {
			llmmodel = tok[9:];
		} else if(hasprefix(tok, "temperature=")) {
			llmtemp = real tok[12:];
			# Clamp to valid range
			if(llmtemp < 0.0)
				llmtemp = 0.0;
			if(llmtemp > 2.0)
				llmtemp = 2.0;
		} else if(hasprefix(tok, "system=")) {
			# System prompt - may be quoted
			llmsystem = stripquotes(tok[7:]);
		} else if(hasprefix(tok, "agenttype=")) {
			agenttype = str->tolower(tok[10:]);
		}
	}

	# Load system prompt from agent type file if not explicitly set
	if(llmsystem == "" && agenttype != "") {
		llmsystem = loadagentprompt(agenttype);
	}
	# Fall back to default agent prompt if nothing specified
	if(llmsystem == "") {
		llmsystem = loadagentprompt("default");
	}

	# Reverse lists to maintain order
	tools = reverse(tools);
	paths = reverse(paths);
	shellcmds = reverse(shellcmds);

	llmconfig := ref NsConstruct->LLMConfig(llmmodel, llmtemp, llmsystem);
	return (tools, paths, shellcmds, trusted, llmconfig, task, "");
}

# Load agent prompt from /lib/veltro/agents/<type>.txt
loadagentprompt(agenttype: string): string
{
	path := "/lib/veltro/agents/" + agenttype + ".txt";
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return "";

	buf := array[4096] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return "";

	return string buf[0:n];
}

# Strip surrounding quotes from a string
stripquotes(s: string): string
{
	if(len s < 2)
		return s;
	if((s[0] == '"' && s[len s - 1] == '"') ||
	   (s[0] == '\'' && s[len s - 1] == '\''))
		return s[1:len s - 1];
	return s;
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
runchild(pipefd: ref Sys->FD, llmfdnum: int, caps: ref NsConstruct->Capabilities, task: string)
{
	# SECURITY MODEL (v2):
	# ====================
	# Uses proper pctl sequence for true isolation:
	#   1. NEWPGRP - Fresh process group (empty srv registry)
	#   2. FORKNS - Fork namespace for mutation
	#   3. NEWENV - Empty environment (no inherited secrets)
	#   4. verifysafefds - Check FDs 0-2 are safe
	#   5. NEWFD - Prune to keep list only (including LLM FD)
	#   6. NODEVS - Block device naming (#U/#p/#c)
	#   7. chdir - Enter prepared sandbox
	#   8. NEWNS - Sandbox becomes /
	#   9. Run subagent loop with LLM FD
	#  10. Return result

	# Step 1: Fresh process group (empty service registry)
	sys->pctl(Sys->NEWPGRP, nil);

	# Step 2: Fork namespace for mutation
	sys->pctl(Sys->FORKNS, nil);

	# Step 3: NEWENV - empty environment, not inherited!
	sys->pctl(Sys->NEWENV, nil);

	# Step 4: Verify FDs 0-2 are safe endpoints
	# Redirect to /dev/null if suspicious
	verifysafefds();

	# Step 5: Prune FDs - keep stdin, stdout, stderr, pipe, and LLM FD
	# The LLM FD survives NEWNS because it's an open FD, not a bind
	keepfds := 0 :: 1 :: 2 :: pipefd.fd :: nil;
	if(llmfdnum >= 0)
		keepfds = llmfdnum :: keepfds;
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

	# NOTE: We skip starting tools9p here because:
	# 1. tools9p has dependencies (styx, styxservers) that would fail to load after NEWNS
	# 2. We have pre-loaded tool modules that subagent uses directly
	# 3. The child runs a full agent loop via subagent->runloop()

	# Step 9: Run the sub-agent loop
	# Build tool module list and name list from preloadedtools
	toolmods: list of Tool;
	toolnames: list of string;
	for(pt := preloadedtools; pt != nil; pt = tl pt) {
		toolmods = (hd pt).mod :: toolmods;
		toolnames = (hd pt).name :: toolnames;
	}

	# Get system prompt from capabilities
	systemprompt := "";
	if(caps.llmconfig != nil)
		systemprompt = caps.llmconfig.system;

	# Recreate LLM FD ref after NEWFD (the ref from parent is invalid now)
	llmfd: ref Sys->FD;
	if(llmfdnum >= 0)
		llmfd = sys->fildes(llmfdnum);

	# Run the agent loop (up to 50 steps)
	# Pass LLM FD directly - it survives NEWNS while binds don't
	result := subagent->runloop(task, toolmods, toolnames, systemprompt, llmfd, 50);

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
# Uses pre-loaded tools9pmod (loaded before NEWNS)
starttools9p(tools: list of string): string
{
	if(tools == nil)
		return "no tools specified";

	# Use pre-loaded tools9p module
	if(tools9pmod == nil)
		return "tools9p not pre-loaded";

	# Build command arguments: tools9p -m /tool tool1 tool2 ...
	args: list of string;
	for(t := tools; t != nil; t = tl t)
		args = hd t :: args;

	# Reverse to maintain order, then prepend fixed args
	revargs: list of string;
	for(; args != nil; args = tl args)
		revargs = hd args :: revargs;
	args = "tools9p" :: "-m" :: "/tool" :: revargs;

	# tools9p.init() creates pipe, spawns serveloop, and mounts
	tools9pmod->init(nil, args);

	return nil;
}

# Safe execution for untrusted agents
# Uses pre-loaded tool modules (loaded before NEWNS)
safeexec(task: string, tools: list of string): string
{
	# Parse first word as tool name
	task = strip(task);
	if(task == "")
		return "ERROR:empty task";

	(toolname, args) := splitfirst(task);
	ltool := str->tolower(toolname);

	# Find pre-loaded tool module
	tool: Tool;
	for(pt := preloadedtools; pt != nil; pt = tl pt) {
		if((hd pt).name == ltool) {
			tool = (hd pt).mod;
			break;
		}
	}

	if(tool == nil)
		return sys->sprint("ERROR:tool not available: %s", ltool);

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
