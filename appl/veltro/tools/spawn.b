implement ToolSpawn;

#
# spawn - Create subagent with secure namespace isolation for Veltro agent
#
# SECURITY MODEL (v3):
# ====================
# Uses FORKNS + bind-replace for namespace isolation:
#
# Child (after spawn):
#   1. pctl(NEWPGRP, nil) - Fresh process group (empty srv registry)
#   2. pctl(FORKNS, nil)  - Fork parent's (already restricted) namespace
#   3. pctl(NEWENV, nil)  - Empty environment (NOT FORKENV!)
#   4. Open LLM FDs       - While /n/llm still accessible
#   5. restrictns(caps)   - Further bind-replace restrictions
#   6. verifysafefds()    - Verify FDs point at safe endpoints
#   7. pctl(NEWFD, keep)  - Prune all other FDs
#   8. pctl(NODEVS, nil)  - Block #U/#p/#c (still allows #e/#s/#|)
#   9. subagent->runloop() - Execute task
#
# Security Properties:
#   - No #U escape (NODEVS after all binds)
#   - No env secrets (NEWENV - empty environment)
#   - No FD leaks (NEWFD with explicit keep-list)
#   - Empty srv registry (NEWPGRP first)
#   - Truthful namespace (restrictdir makes only allowed items visible)
#   - Capability attenuation (child forks restricted parent, can only narrow)
#   - No cleanup needed (bind-replace is in-namespace, not physical dirs)
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

# Pre-load subagent and all granted tool modules
# Called BEFORE spawn so modules are loaded while /dis paths exist
preloadmodules(tools: list of string): string
{
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
		"  shellcmds   - Comma-separated shell commands (grants sh + named cmds)\n" +
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
		"  - Child sees ONLY allowed items in each directory (bind-replace)\n" +
		"  - Environment is empty (no inherited secrets)\n" +
		"  - Shell only available if shellcmds specified\n" +
		"  - LLM config is isolated per-agent\n" +
		"  - Capability attenuation: child can only narrow, never widen\n";
}

exec(args: string): string
{
	if(sys == nil)
		init();

	if(nsconstruct == nil)
		return "error: cannot load nsconstruct module";

	# Parse arguments (includes LLM config)
	(tools, paths, shellcmds, llmconfig, task, err) := parseargs(args);
	if(err != "")
		return "error: " + err;

	if(tools == nil)
		return "error: no tools specified";
	if(task == "")
		return "error: no task specified";

	# Build capabilities
	caps := ref NsConstruct->Capabilities(
		tools,
		paths,
		shellcmds,
		llmconfig,
		0 :: 1 :: 2 :: nil,  # Default FD keep list
		nil,  # No mc9p providers by default
		0,    # No memory by default
		0     # No xenith — subagents don't get /chan access
	);

	# Pre-load modules BEFORE spawn
	# These modules will be used by child AFTER namespace restriction
	err = preloadmodules(tools);
	if(err != nil)
		return "error: " + err;

	# Create pipe for IPC
	pipefds := array[2] of ref Sys->FD;
	if(sys->pipe(pipefds) < 0)
		return sys->sprint("error: cannot create pipe: %r");

	# Spawn child process
	spawn runchild(pipefds[1], caps, task);

	# Close write end in parent
	pipefds[1] = nil;

	# Wait for result with timeout
	timeout := chan of int;
	spawn timer(timeout, 120000);  # 2 minute timeout for multi-step subagent

	resultch := chan of string;
	spawn pipereader(pipefds[0], resultch);

	result: string;
	alt {
	result = <-resultch =>
		;
	<-timeout =>
		pipefds[0] = nil;
		return "error: child agent timed out after 2 minutes";
	}

	pipefds[0] = nil;

	if(hasprefix(result, "ERROR:"))
		return "error: " + result[6:];

	return result;
}

# Parse spawn arguments
# Returns: (tools, paths, shellcmds, llmconfig, task, error)
parseargs(s: string): (list of string, list of string, list of string, ref NsConstruct->LLMConfig, string, string)
{
	tools: list of string;
	paths: list of string;
	shellcmds: list of string;
	task := "";

	# LLM configuration with defaults
	llmmodel := "haiku";    # Default to haiku for subagents (faster)
	llmtemp := 0.7;
	llmsystem := "";
	llmthinking := 0;       # Default: thinking off for subagents
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
		} else if(hasprefix(tok, "model=")) {
			llmmodel = str->tolower(tok[6:]);
		} else if(hasprefix(tok, "temperature=")) {
			llmtemp = real tok[12:];
			# Clamp to valid range
			if(llmtemp < 0.0)
				llmtemp = 0.0;
			if(llmtemp > 2.0)
				llmtemp = 2.0;
		} else if(hasprefix(tok, "thinking=")) {
			# Parse thinking: off, max, or a number 0-30000
			thinkval := str->tolower(tok[9:]);
			if(thinkval == "off" || thinkval == "0")
				llmthinking = 0;
			else if(thinkval == "max" || thinkval == "on")
				llmthinking = -1;
			else {
				llmthinking = int thinkval;
				# Clamp to valid range
				if(llmthinking < 0)
					llmthinking = 0;
				if(llmthinking > 30000)
					llmthinking = 30000;
			}
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

	llmconfig := ref NsConstruct->LLMConfig(llmmodel, llmtemp, llmsystem, llmthinking);
	return (tools, paths, shellcmds, llmconfig, task, "");
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
	while(i < len s && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n'))
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

# Run child agent with FORKNS + bind-replace namespace isolation
runchild(pipefd: ref Sys->FD, caps: ref NsConstruct->Capabilities, task: string)
{
	# SECURITY MODEL (v3):
	# ====================
	# Uses FORKNS + bind-replace for true isolation:
	#   1. NEWPGRP - Fresh process group (empty srv registry)
	#   2. FORKNS  - Fork already-restricted parent namespace
	#   3. NEWENV  - Empty environment (no inherited secrets)
	#   4. Open LLM FDs (while /n/llm still accessible)
	#   5. restrictns - Further bind-replace operations
	#   6. verifysafefds - Check FDs 0-2 are safe
	#   7. NEWFD - Prune to keep list only
	#   8. NODEVS - Block device naming (#U/#p/#c)
	#   9. Run subagent loop

	# Step 1: Fresh process group (empty service registry)
	sys->pctl(Sys->NEWPGRP, nil);

	# Step 2: Fork namespace (inherits parent's already-restricted namespace)
	sys->pctl(Sys->FORKNS, nil);

	# Step 3: NEWENV - empty environment, not inherited!
	sys->pctl(Sys->NEWENV, nil);

	# Step 4: Create LLM session using clone pattern
	# Each subagent gets its own session, fully isolated from parent
	llmaskfd: ref Sys->FD;
	sessionid := "";
	if(caps.llmconfig != nil) {
		# Create session - read /n/llm/new returns session ID
		newfd := sys->open("/n/llm/new", Sys->OREAD);
		if(newfd != nil) {
			buf := array[32] of byte;
			n := sys->read(newfd, buf, len buf);
			if(n > 0) {
				sessionid = string buf[:n];
				# Trim newline if present
				if(len sessionid > 0 && sessionid[len sessionid - 1] == '\n')
					sessionid = sessionid[:len sessionid - 1];
			}
			newfd = nil;  # Close
		}
		if(sessionid != "") {
			# Configure session-specific settings
			modelpath := "/n/llm/" + sessionid + "/model";
			modelfd := sys->open(modelpath, Sys->OWRITE);
			if(modelfd != nil) {
				modeldata := array of byte caps.llmconfig.model;
				sys->write(modelfd, modeldata, len modeldata);
				modelfd = nil;  # Close
			}

			thinkingpath := "/n/llm/" + sessionid + "/thinking";
			thinkingfd := sys->open(thinkingpath, Sys->OWRITE);
			if(thinkingfd != nil) {
				thinkstr: string;
				if(caps.llmconfig.thinking == 0)
					thinkstr = "off";
				else if(caps.llmconfig.thinking < 0)
					thinkstr = "max";
				else
					thinkstr = string caps.llmconfig.thinking;
				thinkdata := array of byte thinkstr;
				sys->write(thinkingfd, thinkdata, len thinkdata);
				thinkingfd = nil;  # Close
			}

			# Set system prompt on session
			if(caps.llmconfig.system != "") {
				systempath := "/n/llm/" + sessionid + "/system";
				systemfd := sys->open(systempath, Sys->OWRITE);
				if(systemfd != nil) {
					sysdata := array of byte caps.llmconfig.system;
					sys->write(systemfd, sysdata, len sysdata);
					systemfd = nil;  # Close
				}
			}

			# Open session's ask file
			askpath := "/n/llm/" + sessionid + "/ask";
			llmaskfd = sys->open(askpath, Sys->ORDWR);
		}
	}

	# Step 5: Apply namespace restrictions (FORKNS + bind-replace)
	# This narrows the already-restricted parent namespace further
	err := nsconstruct->restrictns(caps);
	if(err != nil) {
		writeresult(pipefd, sys->sprint("ERROR:namespace restriction failed: %s", err));
		return;
	}

	# Step 6: Verify FDs 0-2 are safe endpoints
	verifysafefds();

	# Step 7: Prune FDs - keep stdin, stdout, stderr, pipe, and LLM ask FD
	keepfds := 0 :: 1 :: 2 :: pipefd.fd :: nil;
	if(llmaskfd != nil)
		keepfds = llmaskfd.fd :: keepfds;

	sys->pctl(Sys->NEWFD, keepfds);

	# Step 8: Block device naming (AFTER all bind operations)
	sys->pctl(Sys->NODEVS, nil);

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

	# Run the agent loop (up to 50 steps)
	result := subagent->runloop(task, toolmods, toolnames, systemprompt, llmaskfd, 50);

	writeresult(pipefd, result);
	pipefd = nil;
}

# Verify FDs 0-2 are safe endpoints
# If in doubt, redirect to /dev/null
verifysafefds()
{
	# Check stdin
	fd0 := sys->fildes(0);
	if(fd0 == nil) {
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
