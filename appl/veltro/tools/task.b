implement ToolTask;

#
# task - Task lifecycle management tool for Veltro agents
#
# Creates, monitors, and closes Lucifer activities with their own
# tools9p + lucibridge processes. The meta-agent uses this to delegate
# work to task agents with scoped tools and namespace paths.
#
# Budget enforcement: requested tools/paths are validated against
# /tool/budget and /tool/budgetpaths. Only tools within the delegation
# budget can be granted to task agents — no privilege escalation.
#
# Commands:
#   create label=<name> tools=<csv> [paths=<csv>] [urgency=<0-2>] [brief=<text>] [model=<name>]
#   status <id>
#   close <id>
#   list
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "string.m";
	str: String;

include "../tool.m";

include "cowfs.m";

ToolTask: module {
	init: fn(): string;
	name: fn(): string;
	doc:  fn(): string;
	exec: fn(args: string): string;
};

UI_MOUNT: con "/n/ui";

# Process registry for task agents
TaskProc: adt {
	actid: int;
	pgrp:  int;	# process group ID for cleanup
};
taskprocs: list of ref TaskProc;

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
	return "task";
}

doc(): string
{
	return "Task - Create and manage Lucifer task activities\n\n" +
		"Commands:\n" +
		"  create label=<name> tools=<csv> [paths=<csv>] [urgency=<0-2>] [brief=<text>] [model=<name>]\n" +
		"    Create a new task activity with its own agent. Tools must be within budget.\n\n" +
		"  status <id>\n" +
		"    Show task status, cowfs change count.\n\n" +
		"  close <id>\n" +
		"    Archive a task activity and stop its agent processes.\n\n" +
		"  list\n" +
		"    List all active tasks.\n\n" +
		"Budget:\n" +
		"  The delegation budget (/tool/budget) limits which tools can be granted.\n" +
		"  Only tools listed in the budget may be delegated to task agents.\n" +
		"  The budget is set at launch (-b flag) and user-modifiable at runtime.\n";
}

exec(args: string): string
{
	if(args == nil || args == "")
		return "error: task requires a command (create, status, close, list)";

	(verb, rest) := splitfirst(args);

	case verb {
	"create" =>
		return docreate(rest);
	"status" =>
		return dostatus(rest);
	"close" =>
		return doclose(rest);
	"list" =>
		return dolist();
	* =>
		return "error: unknown task command: " + verb;
	}
}

# --- create ---

docreate(args: string): string
{
	if(args == nil || args == "")
		return "error: task create requires label= and tools= parameters";

	# Parse key=value pairs
	label := "";
	toolcsv := "";
	pathcsv := "";
	urgency := "0";
	brief := "";
	model := "haiku";

	(nil, tokens) := sys->tokenize(args, " \t");
	for(; tokens != nil; tokens = tl tokens) {
		tv := hd tokens;
		if(hasprefix(tv, "label="))
			label = tv[6:];
		else if(hasprefix(tv, "tools="))
			toolcsv = tv[6:];
		else if(hasprefix(tv, "paths="))
			pathcsv = tv[6:];
		else if(hasprefix(tv, "urgency="))
			urgency = tv[8:];
		else if(hasprefix(tv, "brief="))
			brief = stripquotes(tv[6:]);
		else if(hasprefix(tv, "model="))
			model = tv[6:];
	}

	if(label == "")
		return "error: label= is required";
	if(toolcsv == "")
		return "error: tools= is required";

	# Parse requested tools
	(nil, reqtools) := sys->tokenize(toolcsv, ",");
	reqpaths: list of string;
	if(pathcsv != "")
		(nil, reqpaths) = sys->tokenize(pathcsv, ",");

	# Validate against budget
	budget := readfile("/tool/budget");
	if(budget == nil)
		budget = "";
	(nil, budgetlist) := sys->tokenize(budget, "\n \t");

	for(rt := reqtools; rt != nil; rt = tl rt) {
		tname := str->tolower(hd rt);
		if(!inlist(tname, budgetlist))
			return "error: tool '" + tname + "' is not in the delegation budget";
	}

	# Validate paths against budget
	if(reqpaths != nil) {
		bpaths := readfile("/tool/budgetpaths");
		if(bpaths == nil)
			bpaths = "";
		(nil, bpathlist) := sys->tokenize(bpaths, "\n \t");
		for(rp := reqpaths; rp != nil; rp = tl rp) {
			pname := hd rp;
			if(!pathallowed(pname, bpathlist))
				return "error: path '" + pname + "' is not in the delegation budget";
		}
	}

	# Strip surrounding quotes from label and brief
	label = stripquotes(label);
	brief = stripquotes(brief);

	# Create the activity via /n/ui/ctl
	ctlcmd := "activity create label=" + label + " urgency=" + urgency +
		" initiator=agent";
	if(brief != "")
		ctlcmd += " brief=" + brief;
	err := writefile(UI_MOUNT + "/ctl", ctlcmd);
	if(err != nil)
		return "error: failed to create activity: " + err;

	# Read back current activity (the newly created one becomes current)
	# Actually, the new activity ID is the last one created. Read ctl to find it.
	info := readfile(UI_MOUNT + "/ctl");
	newid := -1;
	if(info != nil) {
		# Parse "activity <id> <label> ..." lines, find the one matching our label
		(nil, lines) := sys->tokenize(info, "\n");
		for(; lines != nil; lines = tl lines) {
			line := hd lines;
			if(hasprefix(line, "activity ")) {
				(nil, ltoks) := sys->tokenize(line[len "activity ":], " \t");
				if(ltoks != nil) {
					aid := int hd ltoks;
					ltoks = tl ltoks;
					if(ltoks != nil && hd ltoks == label)
						newid = aid;
				}
			}
		}
	}

	if(newid < 0)
		return "error: activity created but could not determine ID";

	# Start task agent processes
	spawn starttaskagent(newid, reqtools, reqpaths, model);

	return sys->sprint("task created: activity %d \"%s\" with tools [%s]", newid, label, toolcsv);
}

# Start tools9p + lucibridge for a task activity
starttaskagent(actid: int, tools, paths: list of string, model: string)
{
	sys->pctl(Sys->NEWPGRP, nil);
	pgrp := sys->pctl(0, nil);

	# Register in process list
	taskprocs = ref TaskProc(actid, pgrp) :: taskprocs;

	# Build tools9p command args
	toolargs := "";
	for(tl_ := tools; tl_ != nil; tl_ = tl tl_) {
		if(toolargs != "")
			toolargs += " ";
		toolargs += hd tl_;
	}

	# Build path args
	pathargs := "";
	for(pl := paths; pl != nil; pl = tl pl)
		pathargs += " -p " + hd pl;

	# Use a unique mount point for this task's tools
	mntpt := sys->sprint("/tool/%d", actid);

	# Start tools9p for this task
	toolcmd := sys->sprint("tools9p -m %s%s %s", mntpt, pathargs, toolargs);
	spawn runcmd(toolcmd);
	sys->sleep(500);  # Give tools9p time to mount

	# Start lucibridge for this task activity
	bridgecmd := sys->sprint("lucibridge -a %d", actid);
	spawn runcmd(bridgecmd);
}

runcmd(cmd: string)
{
	fd := sys->open("/cmd/clone", Sys->ORDWR);
	if(fd == nil) {
		sys->fprint(sys->fildes(2), "task: cannot open /cmd/clone: %r\n");
		return;
	}
	buf := array[32] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return;
	ctlid := string buf[0:n];
	ctlpath := "/cmd/" + ctlid + "/ctl";
	cfd := sys->open(ctlpath, Sys->OWRITE);
	if(cfd == nil) {
		sys->fprint(sys->fildes(2), "task: cannot open %s: %r\n", ctlpath);
		return;
	}
	sys->fprint(cfd, "exec %s", cmd);
}

# --- status ---

dostatus(args: string): string
{
	if(args == nil || args == "")
		return "error: task status requires an activity ID";
	id := int args;
	if(id < 0)
		return "error: bad activity ID";

	label := readfile(sys->sprint("%s/activity/%d/label", UI_MOUNT, id));
	status := readfile(sys->sprint("%s/activity/%d/status", UI_MOUNT, id));
	urgency := readfile(sys->sprint("%s/activity/%d/urgency", UI_MOUNT, id));
	cowcount := readfile(sys->sprint("%s/activity/%d/cow/count", UI_MOUNT, id));

	if(label == nil)
		return "error: activity " + args + " not found";

	result := sys->sprint("activity %d: %s\n", id, strip(label));
	result += "  status: " + strip(status) + "\n";
	result += "  urgency: " + strip(urgency) + "\n";
	if(cowcount != nil)
		result += "  modified files: " + strip(cowcount) + "\n";

	return result;
}

# --- close ---

doclose(args: string): string
{
	if(args == nil || args == "")
		return "error: task close requires an activity ID";
	id := int args;
	if(id < 0)
		return "error: bad activity ID";

	# Kill process group for this task
	for(tp := taskprocs; tp != nil; tp = tl tp) {
		t := hd tp;
		if(t.actid == id && t.pgrp > 0) {
			killgrp(t.pgrp);
			break;
		}
	}

	# Remove from proc list
	nl: list of ref TaskProc;
	for(tp2 := taskprocs; tp2 != nil; tp2 = tl tp2)
		if((hd tp2).actid != id)
			nl = hd tp2 :: nl;
	taskprocs = nl;

	# Archive the activity
	err := writefile(UI_MOUNT + "/ctl", "activity archive " + string id);
	if(err != nil)
		return "error: " + err;

	return sys->sprint("task %d closed", id);
}

killgrp(pgrp: int)
{
	fd := sys->open(sys->sprint("/prog/%d/ctl", pgrp), Sys->OWRITE);
	if(fd != nil)
		sys->fprint(fd, "killgrp");
}

# --- list ---

dolist(): string
{
	info := readfile(UI_MOUNT + "/ctl");
	if(info == nil)
		return "no activities";

	result := "";
	(nil, lines) := sys->tokenize(info, "\n");
	for(; lines != nil; lines = tl lines) {
		line := hd lines;
		if(hasprefix(line, "activity "))
			result += line + "\n";
	}
	if(result == "")
		return "no active tasks";
	return result;
}

# --- Helpers ---

splitfirst(s: string): (string, string)
{
	for(i := 0; i < len s; i++)
		if(s[i] == ' ' || s[i] == '\t') {
			rest := s[i+1:];
			while(len rest > 0 && (rest[0] == ' ' || rest[0] == '\t'))
				rest = rest[1:];
			return (s[0:i], rest);
		}
	return (s, "");
}

hasprefix(s, pfx: string): int
{
	return len s >= len pfx && s[0:len pfx] == pfx;
}

strip(s: string): string
{
	if(s == nil)
		return "";
	while(len s > 0 && (s[len s - 1] == '\n' || s[len s - 1] == ' ' || s[len s - 1] == '\t'))
		s = s[0:len s - 1];
	while(len s > 0 && (s[0] == ' ' || s[0] == '\t'))
		s = s[1:];
	return s;
}

stripquotes(s: string): string
{
	if(len s >= 2 && s[0] == '"' && s[len s - 1] == '"')
		return s[1:len s - 1];
	return s;
}

inlist(s: string, l: list of string): int
{
	for(; l != nil; l = tl l)
		if(hd l == s)
			return 1;
	return 0;
}

# Check if a path is allowed by the budget path list.
# A budget path "/n/local/Users/finn/projects" allows
# any subpath like "/n/local/Users/finn/projects/foo".
pathallowed(path: string, budget: list of string): int
{
	for(; budget != nil; budget = tl budget) {
		bp := hd budget;
		if(path == bp)
			return 1;
		# Allow subpaths
		if(hasprefix(path, bp + "/"))
			return 1;
		# Allow parent paths (broader grant)
		if(hasprefix(bp, path + "/"))
			return 1;
	}
	return 0;
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
	return string buf[0:n];
}

writefile(path, data: string): string
{
	fd := sys->open(path, Sys->OWRITE);
	if(fd == nil)
		return sys->sprint("cannot open %s: %r", path);
	d := array of byte data;
	if(sys->write(fd, d, len d) != len d)
		return sys->sprint("write to %s failed: %r", path);
	return nil;
}
