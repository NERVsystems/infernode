implement Gitdiff;

include "sys.m";
	sys: Sys;
	sprint: import sys;
include "draw.m";
include "arg.m";
	arg: Arg;

include "git.m";
	git: Git;
	Hash, Repo, Commit, TreeEntry: import git;

Gitdiff: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

stderr: ref Sys->FD;

# Cap per-file line count to avoid memory blowup
MAXLINES: con 5000;

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

	for(tl0 := treelist; tl0 != nil; tl0 = tl tl0) {
		(path, treehash) := hd tl0;
		filepath := workdir + "/" + path;
		(rc, nil) := sys->stat(filepath);
		if(rc < 0) {
			# Deleted file — show all lines as removed
			(nil, olddata, rerr) := repo.readobj(treehash);
			if(rerr != nil)
				continue;
			oldlines := splitlines(string olddata);
			if(len oldlines > MAXLINES)
				continue;
			sys->print("diff --git a/%s b/%s\n", path, path);
			sys->print("--- a/%s\n", path);
			sys->print("+++ /dev/null\n");
			sys->print("@@ -1,%d +0,0 @@\n", len oldlines);
			for(j := 0; j < len oldlines; j++)
				sys->print("-%s\n", oldlines[j]);
			continue;
		}

		(filehash, fherr) := hashfile(filepath);
		if(fherr != nil)
			continue;
		if(filehash.eq(treehash))
			continue;

		# Modified file — compute diff
		(nil, olddata, rerr) := repo.readobj(treehash);
		if(rerr != nil)
			continue;
		newdata := readfile(filepath);
		if(newdata == nil)
			continue;

		oldlines := splitlines(string olddata);
		newlines := splitlines(string newdata);
		if(len oldlines > MAXLINES || len newlines > MAXLINES) {
			sys->print("diff --git a/%s b/%s\n", path, path);
			sys->print("Binary files differ (too many lines)\n");
			continue;
		}

		showdiff(path, oldlines, newlines);
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
			result = (path, e.hash) :: result;
		}
	}
	return result;
}

# Read a file and compute its git blob hash.
hashfile(filepath: string): (Hash, string)
{
	data := readfile(filepath);
	if(data == nil)
		return (git->nullhash(), sprint("read %s failed", filepath));
	h := git->hashobj(git->OBJ_BLOB, data);
	return (h, nil);
}

readfile(filepath: string): array of byte
{
	fd := sys->open(filepath, Sys->OREAD);
	if(fd == nil)
		return nil;
	(rc, dir) := sys->fstat(fd);
	if(rc < 0)
		return nil;
	size := int dir.length;
	data := array [size] of byte;
	total := 0;
	while(total < size) {
		n := sys->read(fd, data[total:], size - total);
		if(n <= 0)
			break;
		total += n;
	}
	return data[:total];
}

# Split a string into lines (without trailing newlines).
splitlines(s: string): array of string
{
	lines: list of string;
	n := 0;
	start := 0;
	i := 0;
	for(i = 0; i < len s; i++) {
		if(s[i] == '\n') {
			lines = s[start:i] :: lines;
			n++;
			start = i + 1;
		}
	}
	if(start < len s) {
		lines = s[start:] :: lines;
		n++;
	}

	result := array [n] of string;
	i = n - 1;
	for(; lines != nil; lines = tl lines)
		result[i--] = hd lines;
	return result;
}

# Show unified diff for a modified file.
showdiff(path: string, old, new: array of string)
{
	# Compute LCS edit script using O(mn) DP
	m := len old;
	n := len new;

	# dp[i][j] = length of LCS of old[0:i], new[0:j]
	dp := array [m + 1] of { * => array [n + 1] of { * => 0 } };
	i := 0;
	j := 0;
	for(i = 1; i <= m; i++)
		for(j = 1; j <= n; j++) {
			if(old[i-1] == new[j-1])
				dp[i][j] = dp[i-1][j-1] + 1;
			else if(dp[i-1][j] >= dp[i][j-1])
				dp[i][j] = dp[i-1][j];
			else
				dp[i][j] = dp[i][j-1];
		}

	# Backtrack to produce diff lines
	# Tags: ' ' context, '-' removed, '+' added
	dtags: list of int;
	dlines: list of string;
	ndiff := 0;
	i = m;
	j = n;
	while(i > 0 || j > 0) {
		if(i > 0 && j > 0 && old[i-1] == new[j-1]) {
			dtags = ' ' :: dtags;
			dlines = old[i-1] :: dlines;
			i--;
			j--;
		} else if(j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j])) {
			dtags = '+' :: dtags;
			dlines = new[j-1] :: dlines;
			j--;
		} else {
			dtags = '-' :: dtags;
			dlines = old[i-1] :: dlines;
			i--;
		}
		ndiff++;
	}

	# Convert to arrays for easier processing
	tags := array [ndiff] of int;
	lines := array [ndiff] of string;
	k := 0;
	tl0 := dtags;
	ll := dlines;
	for(; tl0 != nil; tl0 = tl tl0) {
		tags[k] = hd tl0;
		lines[k] = hd ll;
		ll = tl ll;
		k++;
	}

	# Check if there are any differences
	hasdiff := 0;
	for(k = 0; k < ndiff; k++)
		if(tags[k] != ' ') {
			hasdiff = 1;
			break;
		}
	if(!hasdiff)
		return;

	sys->print("diff --git a/%s b/%s\n", path, path);
	sys->print("--- a/%s\n", path);
	sys->print("+++ b/%s\n", path);

	# Output hunks with 3 lines of context
	CTX: con 3;
	k = 0;
	while(k < ndiff) {
		# Find next change
		while(k < ndiff && tags[k] == ' ')
			k++;
		if(k >= ndiff)
			break;

		# Start of hunk: back up for context
		hstart := k - CTX;
		if(hstart < 0)
			hstart = 0;

		# Find end of hunk (include changes and context)
		hend := k;
		while(hend < ndiff) {
			if(tags[hend] != ' ') {
				hend++;
				continue;
			}
			# Look ahead for more changes within context distance
			lookahead := hend;
			gap := 0;
			while(lookahead < ndiff && tags[lookahead] == ' ') {
				lookahead++;
				gap++;
			}
			if(lookahead < ndiff && gap <= 2 * CTX) {
				hend = lookahead;
				continue;
			}
			break;
		}
		# Add trailing context
		hend += CTX;
		if(hend > ndiff)
			hend = ndiff;

		# Count old/new lines in hunk
		oldstart := 1;
		newstart := 1;
		# Count lines before hstart
		for(p := 0; p < hstart; p++) {
			if(tags[p] == ' ' || tags[p] == '-')
				oldstart++;
			if(tags[p] == ' ' || tags[p] == '+')
				newstart++;
		}
		oldcount := 0;
		newcount := 0;
		for(p = hstart; p < hend; p++) {
			if(tags[p] == ' ' || tags[p] == '-')
				oldcount++;
			if(tags[p] == ' ' || tags[p] == '+')
				newcount++;
		}

		sys->print("@@ -%d,%d +%d,%d @@\n", oldstart, oldcount, newstart, newcount);
		for(p = hstart; p < hend; p++)
			sys->print("%c%s\n", tags[p], lines[p]);

		k = hend;
	}
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
	sys->fprint(stderr, "git/diff: %s\n", msg);
	raise "fail:" + msg;
}
