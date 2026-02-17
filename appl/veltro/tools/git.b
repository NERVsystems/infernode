implement ToolGit;

#
# git - Read-only git repository access for Veltro agents
#
# Mounts git/fs at /n/git during init() (before namespace restriction),
# then reads from the 9P filesystem in exec().
#
# git/fs presents the repository as:
#   /n/git/ctl              current branch
#   /n/git/HEAD/            commit dir (hash, author, msg, parent, tree/)
#   /n/git/branch/heads/    local branches → commit dirs
#   /n/git/branch/remotes/  remote tracking branches
#   /n/git/tag/             tags → commit dirs
#   /n/git/object/<hex>/    any object by hash
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

Gitfs: module {
	init: fn(nil: ref Draw->Context, args: list of string);
};

gitavail := 0;

init(): string
{
	sys = load Sys Sys->PATH;
	if(sys == nil)
		return "cannot load Sys";
	str = load String String->PATH;
	if(str == nil)
		return "cannot load String";

	# Check for git repo
	(ok, nil) := sys->stat("/.git");
	if(ok < 0)
		return nil;  # No repo; exec() will return errors

	# Mount git/fs at /n/git before namespace restriction
	ready := chan of int;
	spawn mountgitfs(ready);
	result := <-ready;
	if(result > 0)
		gitavail = 1;

	return nil;
}

mountgitfs(ready: chan of int)
{
	gitfs := load Gitfs "/dis/cmd/git/fs.dis";
	if(gitfs == nil) {
		ready <-= 0;
		return;
	}

	{
		gitfs->init(nil, "git/fs" :: "-m" :: "/n/git" :: "/.git" :: nil);
		ready <-= 1;
	} exception {
	"*" =>
		ready <-= 0;
	}
}

name(): string
{
	return "git";
}

doc(): string
{
	return "Git - Read-only git repository access\n\n" +
		"Subcommands:\n" +
		"  git status              Current branch and HEAD commit\n" +
		"  git log [n]             Last n commits (default 10, max 50)\n" +
		"  git show <hash>         Show commit details by hash or branch name\n" +
		"  git branch              List local and remote branches\n" +
		"  git tag                 List tags\n" +
		"  git cat <path>          Show file content at HEAD\n" +
		"  git cat <path> <ref>    Show file content at branch or hash\n";
}

exec(args: string): string
{
	if(sys == nil)
		init();
	if(!gitavail)
		return "error: no git repository available";

	args = strip(args);
	if(args == "")
		return "error: usage: git <status|log|show|branch|tag|cat> [args]";

	(cmd, rest) := splitword(args);
	rest = strip(rest);

	case cmd {
	"status" =>
		return gitstatus();
	"log" =>
		n := 10;
		if(rest != "")
			n = int rest;
		if(n < 1)
			n = 1;
		if(n > 50)
			n = 50;
		return gitlog(n);
	"show" =>
		if(rest == "")
			return "error: usage: git show <hash|branch>";
		return gitshow(rest);
	"branch" or "branches" =>
		return gitbranch();
	"tag" or "tags" =>
		return gittag();
	"cat" =>
		return gitcat(rest);
	* =>
		return "error: unknown subcommand: " + cmd +
			"\nAvailable: status, log, show, branch, tag, cat";
	}
}

gitstatus(): string
{
	branch := strip(readfile("/n/git/ctl"));
	if(branch == "")
		branch = "(unknown)";

	headhash := strip(readfile("/n/git/HEAD/hash"));
	if(headhash == "")
		return "On branch " + branch + "\n(no commits)";

	headmsg := strip(readfile("/n/git/HEAD/msg"));
	(firstline, nil) := splitline(headmsg);

	return "On branch " + branch + "\n" +
		"HEAD " + shorthash(headhash) + " " + firstline;
}

gitlog(n: int): string
{
	result := "";

	# First commit from HEAD
	hash := strip(readfile("/n/git/HEAD/hash"));
	if(hash == "")
		return "(no commits)";

	author := strip(readfile("/n/git/HEAD/author"));
	msg := strip(readfile("/n/git/HEAD/msg"));
	(firstline, nil) := splitline(msg);
	result = shorthash(hash) + " " + firstline + "\n";
	result += "  Author: " + author + "\n";

	parent := strip(readfile("/n/git/HEAD/parent"));

	# Follow parent chain via object directory
	for(i := 1; i < n && parent != "" && parent != "nil"; i++) {
		objdir := "/n/git/object/" + parent;

		author = strip(readfile(objdir + "/author"));
		msg = strip(readfile(objdir + "/msg"));
		(firstline, nil) = splitline(msg);

		result += "\n" + shorthash(parent) + " " + firstline + "\n";
		result += "  Author: " + author + "\n";

		parent = strip(readfile(objdir + "/parent"));
	}

	return result;
}

gitshow(gitref: string): string
{
	objdir: string;

	if(len gitref == 40) {
		objdir = "/n/git/object/" + gitref;
	} else {
		# Try as branch name
		hash := strip(readfile("/n/git/branch/heads/" + gitref + "/hash"));
		if(hash != "") {
			objdir = "/n/git/object/" + hash;
			gitref = hash;
		} else {
			# Try as tag
			hash = strip(readfile("/n/git/tag/" + gitref + "/hash"));
			if(hash != "") {
				objdir = "/n/git/object/" + hash;
				gitref = hash;
			} else
				return "error: cannot find ref: " + gitref;
		}
	}

	otype := strip(readfile(objdir + "/type"));
	if(otype == "")
		return "error: object not found: " + gitref;

	case otype {
	"commit" =>
		chash := strip(readfile(objdir + "/hash"));
		cauthor := strip(readfile(objdir + "/author"));
		ccommitter := strip(readfile(objdir + "/committer"));
		cmsg := strip(readfile(objdir + "/msg"));
		cparent := strip(readfile(objdir + "/parent"));

		cresult := "commit " + chash + "\n";
		cresult += "Author: " + cauthor + "\n";
		cresult += "Committer: " + ccommitter + "\n";
		if(cparent != "" && cparent != "nil")
			cresult += "Parent: " + cparent + "\n";
		cresult += "\n" + cmsg;
		return cresult;

	"blob" =>
		bdata := readfile(objdir + "/data");
		if(bdata == "")
			return "(empty blob)";
		return bdata;

	"tree" =>
		return "tree " + gitref + "\n(use 'git cat <path>' to view files)";

	"tag" =>
		ttagger := strip(readfile(objdir + "/tagger"));
		tmsg := strip(readfile(objdir + "/msg"));
		tresult := "tag " + gitref + "\n";
		if(ttagger != "")
			tresult += "Tagger: " + ttagger + "\n";
		if(tmsg != "")
			tresult += "\n" + tmsg;
		return tresult;

	* =>
		return "object " + gitref + " type=" + otype;
	}
}

gitbranch(): string
{
	current := strip(readfile("/n/git/ctl"));

	# Local branches
	entries := listdir("/n/git/branch/heads");
	if(entries == nil)
		return "(no branches)";

	result := "";
	for(; entries != nil; entries = tl entries) {
		bname := hd entries;
		if(bname == current)
			result += "* " + bname + "\n";
		else
			result += "  " + bname + "\n";
	}

	# Remote branches
	remotes := listdir("/n/git/branch/remotes");
	for(; remotes != nil; remotes = tl remotes) {
		remote := hd remotes;
		rbranches := listdir("/n/git/branch/remotes/" + remote);
		for(; rbranches != nil; rbranches = tl rbranches)
			result += "  remotes/" + remote + "/" + hd rbranches + "\n";
	}

	return result;
}

gittag(): string
{
	entries := listdir("/n/git/tag");
	if(entries == nil)
		return "(no tags)";

	result := "";
	for(; entries != nil; entries = tl entries)
		result += hd entries + "\n";

	return result;
}

gitcat(args: string): string
{
	if(args == "")
		return "error: usage: git cat <path> [ref]";

	(fpath, gitref) := splitword(args);
	gitref = strip(gitref);

	treepath: string;
	if(gitref == "") {
		treepath = "/n/git/HEAD/tree/" + fpath;
	} else {
		# Try as branch name
		hash := strip(readfile("/n/git/branch/heads/" + gitref + "/hash"));
		if(hash == "") {
			# Try as full hash
			if(len gitref == 40)
				hash = gitref;
			else
				return "error: cannot find ref: " + gitref;
		}
		treepath = "/n/git/object/" + hash + "/tree/" + fpath;
	}

	content := readfile(treepath);
	if(content == "")
		return "error: file not found: " + fpath;

	return content;
}

# --- Helpers ---

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

listdir(path: string): list of string
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return nil;

	result: list of string;
	for(;;) {
		(n, dirs) := sys->dirread(fd);
		if(n <= 0)
			break;
		for(i := 0; i < n; i++)
			result = dirs[i].name :: result;
	}

	# Reverse to maintain order
	rev: list of string;
	for(; result != nil; result = tl result)
		rev = hd result :: rev;
	return rev;
}

strip(s: string): string
{
	if(len s == 0)
		return "";
	i := 0;
	while(i < len s && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r'))
		i++;
	j := len s;
	while(j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r'))
		j--;
	if(i >= j)
		return "";
	return s[i:j];
}

splitword(s: string): (string, string)
{
	for(i := 0; i < len s; i++) {
		if(s[i] == ' ' || s[i] == '\t')
			return (s[0:i], s[i+1:]);
	}
	return (s, "");
}

splitline(s: string): (string, string)
{
	for(i := 0; i < len s; i++) {
		if(s[i] == '\n')
			return (s[0:i], s[i+1:]);
	}
	return (s, "");
}

shorthash(h: string): string
{
	if(len h >= 8)
		return h[0:8];
	return h;
}
