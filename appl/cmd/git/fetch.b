implement Gitfetch;

include "sys.m";
	sys: Sys;
	sprint: import sys;
include "draw.m";
include "arg.m";
	arg: Arg;
include "string.m";
	str: String;

include "git.m";
	git: Git;
	Hash, Ref, Repo: import git;

Gitfetch: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

stderr: ref Sys->FD;

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	stderr = sys->fildes(2);

	arg = load Arg Arg->PATH;
	str = load String String->PATH;
	git = load Git Git->PATH;
	if(git == nil)
		fail(sprint("load Git: %r"));

	err := git->init();
	if(err != nil)
		fail("git init: " + err);

	arg->init(args);
	arg->setusage(arg->progname() + " [-v] [remote]");

	verbose := 0;
	while((ch := arg->opt()) != 0)
		case ch {
		'v' =>
			verbose = 1;
		* =>
			arg->usage();
		}

	argv := arg->argv();

	# Find .git directory
	gitdir := findgitdir();
	if(gitdir == nil)
		fail("not a git repository");

	# Get remote URL
	remote := "origin";
	if(len argv >= 1)
		remote = hd argv;

	remoteurl := getremoteurl(gitdir, remote);
	if(remoteurl == nil)
		fail("no url for remote: " + remote);

	if(verbose)
		sys->fprint(stderr, "fetching from %s (%s)\n", remote, remoteurl);

	# Open local repo
	(repo, oerr) := git->openrepo(gitdir);
	if(oerr != nil)
		fail("openrepo: " + oerr);

	# Discover remote refs
	(refs, nil, derr) := git->discover(remoteurl);
	if(derr != nil)
		fail("discover: " + derr);

	if(verbose) {
		skipped := 0;
		for(rl := refs; rl != nil; rl = tl rl) {
			r := hd rl;
			if(r.name == "HEAD"
			   || (len r.name > 11 && r.name[:11] == "refs/heads/")
			   || (len r.name > 10 && r.name[:10] == "refs/tags/"))
				sys->fprint(stderr, "  remote: %s %s\n", r.hash.hex(), r.name);
			else
				skipped++;
		}
		if(skipped > 0)
			sys->fprint(stderr, "  (%d additional refs not shown)\n", skipped);
	}

	# Determine what we need
	want: list of Hash;
	seen: list of string;
	for(rl := refs; rl != nil; rl = tl rl) {
		r := hd rl;
		hexstr := r.hash.hex();
		if(!inlist(hexstr, seen) && !repo.hasobj(r.hash)) {
			want = r.hash :: want;
			seen = hexstr :: seen;
		}
	}

	if(want == nil) {
		if(verbose)
			sys->fprint(stderr, "already up to date\n");
		# Still update refs
		updaterefs(gitdir, remote, refs, verbose);
		return;
	}

	# Collect hashes we have for negotiation
	have: list of Hash;
	localrefs := repo.listrefs();
	for(lr := localrefs; lr != nil; lr = tl lr) {
		(nil, h) := hd lr;
		have = h :: have;
	}

	if(verbose)
		sys->fprint(stderr, "fetching %d new objects...\n", listlen(want));

	# Fetch pack
	packname := "pack-fetch";
	packpath := gitdir + "/objects/pack/" + packname + ".pack";
	ferr := git->fetchpack(remoteurl, want, have, packpath);
	if(ferr != nil)
		fail("fetchpack: " + ferr);

	if(verbose)
		sys->fprint(stderr, "indexing pack...\n");

	# Index the pack
	xerr := git->indexpack(packpath);
	if(xerr != nil)
		fail("indexpack: " + xerr);

	# Rename pack using its SHA-1
	renamepak(gitdir, packpath, packname);

	# Update refs
	updaterefs(gitdir, remote, refs, verbose);

	sys->print("fetch complete\n");
}

updaterefs(gitdir, remote: string, refs: list of Ref, verbose: int)
{
	for(rl := refs; rl != nil; rl = tl rl) {
		r := hd rl;
		name := r.name;

		if(name == "HEAD")
			continue;

		if(len name > 11 && name[:11] == "refs/heads/") {
			branchname := name[11:];
			refname := "refs/remotes/" + remote + "/" + branchname;
			writeref(gitdir, refname, r.hash);
			if(verbose)
				sys->fprint(stderr, "  -> %s\n", refname);
		}

		if(len name > 10 && name[:10] == "refs/tags/") {
			writeref(gitdir, name, r.hash);
			if(verbose)
				sys->fprint(stderr, "  -> %s\n", name);
		}
	}
}

renamepak(gitdir, packpath, packname: string)
{
	pfd := sys->open(packpath, Sys->OREAD);
	if(pfd == nil)
		return;
	sys->seek(pfd, big -20, Sys->SEEKEND);
	sha := array [20] of byte;
	sys->read(pfd, sha, 20);
	pfd = nil;

	packhex := "";
	for(i := 0; i < 20; i++)
		packhex += sprint("%02x", int sha[i]);

	newpackpath := gitdir + "/objects/pack/pack-" + packhex + ".pack";
	newidxpath := gitdir + "/objects/pack/pack-" + packhex + ".idx";
	oldidxpath := gitdir + "/objects/pack/" + packname + ".idx";

	copyfile(packpath, newpackpath);
	copyfile(oldidxpath, newidxpath);
	sys->remove(packpath);
	sys->remove(oldidxpath);
}

writeref(gitdir, name: string, h: Hash)
{
	path := gitdir + "/" + name;
	mkdirp(path);
	fd := sys->create(path, Sys->OWRITE, 8r644);
	if(fd == nil)
		return;
	data := array of byte (h.hex() + "\n");
	sys->write(fd, data, len data);
}

mkdirp(filepath: string)
{
	for(i := 1; i < len filepath; i++)
		if(filepath[i] == '/')
			sys->create(filepath[:i], Sys->OREAD, Sys->DMDIR | 8r755);
}

copyfile(src, dst: string)
{
	sfd := sys->open(src, Sys->OREAD);
	if(sfd == nil)
		return;
	dfd := sys->create(dst, Sys->OWRITE, 8r644);
	if(dfd == nil)
		return;
	buf := array [8192] of byte;
	for(;;) {
		n := sys->read(sfd, buf, len buf);
		if(n <= 0)
			break;
		sys->write(dfd, buf[:n], n);
	}
}

findgitdir(): string
{
	# Look for .git in current directory, then parents
	# Start with current working directory
	dir := ".";
	for(depth := 0; depth < 20; depth++) {
		gitdir := dir + "/.git";
		(n, nil) := sys->stat(gitdir);
		if(n >= 0)
			return gitdir;
		dir = dir + "/..";
	}
	return nil;
}

getremoteurl(gitdir, remote: string): string
{
	# Parse .git/config to find remote URL
	# Simple parser: look for [remote "name"] section, then url =
	fd := sys->open(gitdir + "/config", Sys->OREAD);
	if(fd == nil)
		return nil;
	buf := array [8192] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return nil;

	config := string buf[:n];
	target := "[remote \"" + remote + "\"]";
	insection := 0;

	s := config;
	for(;;) {
		(line, rest) := splitline(s);
		if(line == nil && rest == nil)
			break;
		s = rest;

		# Strip whitespace
		line = strtrim(line);

		if(len line > 0 && line[0] == '[') {
			insection = (line == target || line == "[remote \"" + remote + "\"]");
			continue;
		}

		if(insection) {
			(key, val) := splitfirst(line, '=');
			key = strtrim(key);
			val = strtrim(val);
			if(key == "url")
				return val;
		}
	}
	return nil;
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

splitfirst(s: string, sep: int): (string, string)
{
	for(i := 0; i < len s; i++)
		if(s[i] == sep)
			return (s[:i], s[i+1:]);
	return (s, "");
}

strtrim(s: string): string
{
	i := 0;
	while(i < len s && (s[i] == ' ' || s[i] == '\t'))
		i++;
	j := len s;
	while(j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\r' || s[j-1] == '\n'))
		j--;
	return s[i:j];
}

inlist(s: string, l: list of string): int
{
	for(; l != nil; l = tl l)
		if(hd l == s)
			return 1;
	return 0;
}

listlen(l: list of Hash): int
{
	n := 0;
	for(; l != nil; l = tl l)
		n++;
	return n;
}

fail(msg: string)
{
	sys->fprint(stderr, "git/fetch: %s\n", msg);
	raise "fail:" + msg;
}
