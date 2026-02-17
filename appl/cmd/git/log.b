implement Gitlog;

include "sys.m";
	sys: Sys;
	sprint: import sys;
include "draw.m";
include "arg.m";
	arg: Arg;

include "git.m";
	git: Git;
	Hash, Repo, Commit: import git;

Gitlog: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

stderr: ref Sys->FD;

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	stderr = sys->fildes(2);

	arg = load Arg Arg->PATH;
	git = load Git Git->PATH;
	if(git == nil)
		fail(sprint("load Git: %r"));

	err := git->init();
	if(err != nil)
		fail("git init: " + err);

	arg->init(args);
	arg->setusage(arg->progname() + " [-n count] [-1] [dir]");

	limit := 0;
	compact := 0;
	while((ch := arg->opt()) != 0)
		case ch {
		'n' =>
			limit = int arg->earg();
		'1' =>
			limit = 1;
			compact = 1;
		* =>
			arg->usage();
		}

	argv := arg->argv();
	dir := ".";
	if(len argv >= 1)
		dir = hd argv;

	gitdir := findgitdir(dir);
	if(gitdir == nil)
		fail("not a git repository");

	(repo, oerr) := git->openrepo(gitdir);
	if(oerr != nil)
		fail("openrepo: " + oerr);

	# Resolve HEAD
	(refname, herr) := repo.head();
	if(herr != nil)
		fail("HEAD: " + herr);
	(hash, rrerr) := repo.readref(refname);
	if(rrerr != nil)
		fail("readref: " + rrerr);

	count := 0;
	while(!hash.isnil()) {
		if(limit > 0 && count >= limit)
			break;

		(otype, data, rerr) := repo.readobj(hash);
		if(rerr != nil) {
			sys->fprint(stderr, "git/log: read %s: %s\n", hash.hex(), rerr);
			break;
		}
		if(otype != git->OBJ_COMMIT)
			break;

		(commit, cperr) := git->parsecommit(data);
		if(cperr != nil) {
			sys->fprint(stderr, "git/log: parse: %s\n", cperr);
			break;
		}

		if(compact) {
			firstline := commit.msg;
			for(i := 0; i < len firstline; i++)
				if(firstline[i] == '\n') {
					firstline = firstline[:i];
					break;
				}
			sys->print("%s %s\n", hash.hex()[:7], firstline);
		} else {
			sys->print("commit %s\n", hash.hex());
			sys->print("Author: %s\n", commit.author);
			sys->print("\n");
			# Indent message
			msg := commit.msg;
			s := msg;
			for(;;) {
				(line, rest) := splitline(s);
				if(line == nil && rest == nil)
					break;
				sys->print("    %s\n", line);
				s = rest;
				if(s == nil || len s == 0)
					break;
			}
			sys->print("\n");
		}

		# Follow first parent
		if(commit.parents == nil)
			break;
		hash = hd commit.parents;
		count++;
	}
}

splitline(s: string): (string, string)
{
	for(i := 0; i < len s; i++) {
		if(s[i] == '\n')
			return (s[:i], s[i+1:]);
	}
	if(len s > 0)
		return (s, nil);
	return (nil, nil);
}

findgitdir(dir: string): string
{
	for(depth := 0; depth < 20; depth++) {
		gitdir := dir + "/.git";
		(n, nil) := sys->stat(gitdir);
		if(n >= 0)
			return gitdir;
		dir = dir + "/..";
	}
	return nil;
}

fail(msg: string)
{
	sys->fprint(stderr, "git/log: %s\n", msg);
	raise "fail:" + msg;
}
