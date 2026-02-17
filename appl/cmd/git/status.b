implement Gitstatus;

include "sys.m";
	sys: Sys;
	sprint: import sys;
include "draw.m";
include "arg.m";
	arg: Arg;

include "git.m";
	git: Git;
	Hash, Repo, Commit, TreeEntry: import git;

Gitstatus: module
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
	arg->setusage(arg->progname() + " [dir]");

	while((ch := arg->opt()) != 0)
		case ch {
		* =>
			arg->usage();
		}

	argv := arg->argv();
	workdir := ".";
	if(len argv >= 1)
		workdir = hd argv;

	gitdir := findgitdir(workdir);
	if(gitdir == nil)
		fail("not a git repository");

	(repo, oerr) := git->openrepo(gitdir);
	if(oerr != nil)
		fail("openrepo: " + oerr);

	# Resolve HEAD to commit tree hash
	(refname, herr) := repo.head();
	if(herr != nil)
		fail("HEAD: " + herr);
	(headhash, rrerr) := repo.readref(refname);
	if(rrerr != nil)
		fail("readref: " + rrerr);

	(nil, cdata, cerr) := repo.readobj(headhash);
	if(cerr != nil)
		fail("read HEAD commit: " + cerr);
	(commit, cperr) := git->parsecommit(cdata);
	if(cperr != nil)
		fail("parse commit: " + cperr);

	# Flatten tree into list of (path, hash)
	treelist := walktree(repo, commit.tree, "");

	# Walk working directory
	dirlist := walkdir(workdir, "");

	# Check for modified and deleted files
	for(tl0 := treelist; tl0 != nil; tl0 = tl tl0) {
		(path, treehash) := hd tl0;
		filepath := workdir + "/" + path;
		(rc, nil) := sys->stat(filepath);
		if(rc < 0) {
			sys->print(" D %s\n", path);
			continue;
		}
		(filehash, fherr) := hashfile(filepath);
		if(fherr != nil)
			continue;
		if(!filehash.eq(treehash))
			sys->print(" M %s\n", path);
	}

	# Check for untracked files
	for(dl := dirlist; dl != nil; dl = tl dl) {
		path := hd dl;
		if(!intreelist(path, treelist))
			sys->print("?? %s\n", path);
	}
}

# Recursively flatten a git tree into a list of (path, hash) for blobs.
walktree(repo: ref Repo, treehash: Hash, prefix: string): list of (string, Hash)
{
	(otype, data, err) := repo.readobj(treehash);
	if(err != nil || otype != git->OBJ_TREE)
		return nil;

	(entries, perr) := git->parsetree(data);
	if(perr != nil)
		return nil;

	result: list of (string, Hash);
	for(i := 0; i < len entries; i++) {
		e := entries[i];
		path: string;
		if(prefix == "")
			path = e.name;
		else
			path = prefix + "/" + e.name;

		if(e.mode == 8r40000) {
			sub := walktree(repo, e.hash, path);
			for(; sub != nil; sub = tl sub)
				result = (hd sub) :: result;
		} else if(e.mode != 8r120000) {
			# Regular file (skip symlinks)
			result = (path, e.hash) :: result;
		}
	}
	return result;
}

# Recursively list working directory files, skipping .git.
walkdir(basepath, prefix: string): list of string
{
	dirpath: string;
	if(prefix == "")
		dirpath = basepath;
	else
		dirpath = basepath + "/" + prefix;

	fd := sys->open(dirpath, Sys->OREAD);
	if(fd == nil)
		return nil;

	result: list of string;
	for(;;) {
		(n, dirs) := sys->dirread(fd);
		if(n <= 0)
			break;
		for(i := 0; i < n; i++) {
			name := dirs[i].name;
			if(name == ".git")
				continue;

			path: string;
			if(prefix == "")
				path = name;
			else
				path = prefix + "/" + name;

			if(dirs[i].qid.qtype & Sys->QTDIR) {
				sub := walkdir(basepath, path);
				for(; sub != nil; sub = tl sub)
					result = (hd sub) :: result;
			} else {
				result = path :: result;
			}
		}
	}
	return result;
}

# Read a file and compute its git blob hash.
hashfile(filepath: string): (Hash, string)
{
	fd := sys->open(filepath, Sys->OREAD);
	if(fd == nil)
		return (git->nullhash(), sprint("open %s: %r", filepath));

	(rc, dir) := sys->fstat(fd);
	if(rc < 0)
		return (git->nullhash(), sprint("fstat %s: %r", filepath));

	size := int dir.length;
	data := array [size] of byte;
	total := 0;
	while(total < size) {
		n := sys->read(fd, data[total:], size - total);
		if(n <= 0)
			break;
		total += n;
	}
	data = data[:total];

	h := git->hashobj(git->OBJ_BLOB, data);
	return (h, nil);
}

intreelist(path: string, l: list of (string, Hash)): int
{
	for(; l != nil; l = tl l) {
		(p, nil) := hd l;
		if(p == path)
			return 1;
	}
	return 0;
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
	sys->fprint(stderr, "git/status: %s\n", msg);
	raise "fail:" + msg;
}
