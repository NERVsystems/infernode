implement GitTest;

#
# Git module integration tests
#
# Tests the core git library: hashing, pkt-line protocol,
# object parsing, delta application, and optionally
# live clone + git/fs operations.
#

include "sys.m";
	sys: Sys;
	sprint: import sys;
include "draw.m";
include "testing.m";
	testing: Testing;
	T: import testing;

include "git.m";
	git: Git;
	Hash, Commit, TreeEntry, Tag, PackIdx, Repo: import git;

GitTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

SRCFILE: con "/tests/git_test.b";

passed := 0;
failed := 0;
skipped := 0;

run(name: string, testfn: ref fn(t: ref T))
{
	t := testing->newTsrc(name, SRCFILE);
	{
		testfn(t);
	} exception {
	"fail:fatal" =>
		;
	"fail:skip" =>
		;
	"*" =>
		t.failed = 1;
	}

	if(testing->done(t))
		passed++;
	else if(t.skipped)
		skipped++;
	else
		failed++;
}

# --- Hash Tests ---

testNullHash(t: ref T)
{
	h := git->nullhash();
	t.assert(h.isnil(), "null hash should be nil");
	t.assertseq(h.hex(), "0000000000000000000000000000000000000000", "null hash hex");
}

testParseHash(t: ref T)
{
	hexstr := "da39a3ee5e6b4b0d3255bfef95601890afd80709";
	(h, err) := git->parsehash(hexstr);
	t.assertnil(err, "parsehash should succeed");
	t.assertseq(h.hex(), hexstr, "round-trip hex");
	t.assert(!h.isnil(), "parsed hash should not be nil");
}

testParseHashBad(t: ref T)
{
	(nil, err) := git->parsehash("too-short");
	t.assertnotnil(err, "parsehash should fail on short string");

	(nil, err2) := git->parsehash("zz39a3ee5e6b4b0d3255bfef95601890afd80709");
	t.assertnotnil(err2, "parsehash should fail on bad hex");
}

testHashEquality(t: ref T)
{
	h1 := git->nullhash();
	h2 := git->nullhash();
	t.assert(h1.eq(h2), "two null hashes should be equal");

	hexstr := "da39a3ee5e6b4b0d3255bfef95601890afd80709";
	(h3, nil) := git->parsehash(hexstr);
	(h4, nil) := git->parsehash(hexstr);
	t.assert(h3.eq(h4), "same hashes should be equal");
	t.assert(!h1.eq(h3), "different hashes should not be equal");
}

# --- Object Hashing Tests ---

testHashBlob(t: ref T)
{
	# SHA-1 of empty blob: "blob 0\0" = e69de29bb2d1d6434b8b29ae775ad8c2e48c5391
	data := array [0] of byte;
	h := git->hashobj(git->OBJ_BLOB, data);
	t.assertseq(h.hex(), "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", "empty blob hash");
}

testHashBlobContent(t: ref T)
{
	# "hello\n" blob: "blob 6\0hello\n" = ce013625030ba8dba906f756967f9e9ca394464a
	data := array of byte "hello\n";
	h := git->hashobj(git->OBJ_BLOB, data);
	t.assertseq(h.hex(), "ce013625030ba8dba906f756967f9e9ca394464a", "hello blob hash");
}

# --- Type Name Tests ---

testTypename(t: ref T)
{
	t.assertseq(git->typename(git->OBJ_COMMIT), "commit", "commit type name");
	t.assertseq(git->typename(git->OBJ_TREE), "tree", "tree type name");
	t.assertseq(git->typename(git->OBJ_BLOB), "blob", "blob type name");
	t.assertseq(git->typename(git->OBJ_TAG), "tag", "tag type name");
}

testTypenum(t: ref T)
{
	t.asserteq(git->typenum("commit"), git->OBJ_COMMIT, "commit type num");
	t.asserteq(git->typenum("tree"), git->OBJ_TREE, "tree type num");
	t.asserteq(git->typenum("blob"), git->OBJ_BLOB, "blob type num");
	t.asserteq(git->typenum("tag"), git->OBJ_TAG, "tag type num");
	t.asserteq(git->typenum("unknown"), 0, "unknown type num");
}

# --- Commit Parsing ---

testParseCommit(t: ref T)
{
	commitdata := array of byte (
		"tree 4b825dc642cb6eb9a060e54bf899d69f7c355415\n" +
		"parent da39a3ee5e6b4b0d3255bfef95601890afd80709\n" +
		"author Test User <test@example.com> 1234567890 +0000\n" +
		"committer Test User <test@example.com> 1234567890 +0000\n" +
		"\n" +
		"Initial commit\n");

	(c, err) := git->parsecommit(commitdata);
	t.assertnil(err, "parsecommit should succeed");
	t.assertseq(c.tree.hex(), "4b825dc642cb6eb9a060e54bf899d69f7c355415", "commit tree");
	t.assertseq(c.msg, "Initial commit\n", "commit message");

	# Check parent
	nparents := 0;
	for(pl := c.parents; pl != nil; pl = tl pl)
		nparents++;
	t.asserteq(nparents, 1, "should have one parent");

	t.assert(c.author != nil && len c.author > 0, "author should be set");
	t.assert(c.committer != nil && len c.committer > 0, "committer should be set");
}

testParseCommitNoParent(t: ref T)
{
	commitdata := array of byte (
		"tree 4b825dc642cb6eb9a060e54bf899d69f7c355415\n" +
		"author Test User <test@example.com> 1234567890 +0000\n" +
		"committer Test User <test@example.com> 1234567890 +0000\n" +
		"\n" +
		"Root commit\n");

	(c, err) := git->parsecommit(commitdata);
	t.assertnil(err, "parsecommit no-parent should succeed");

	nparents := 0;
	for(pl := c.parents; pl != nil; pl = tl pl)
		nparents++;
	t.asserteq(nparents, 0, "root commit should have no parents");
}

# --- Tree Parsing ---

testParseTree(t: ref T)
{
	# Build a tree manually: "100644 hello.txt\0<20-byte-hash>"
	name := "hello.txt";
	mode := "100644";
	hash := array [20] of { * => byte 16raa };

	# mode + space + name + null + hash
	modebs := array of byte mode;
	namebs := array of byte name;
	treedata := array [len modebs + 1 + len namebs + 1 + 20] of byte;
	off := 0;
	treedata[off:] = modebs;
	off += len modebs;
	treedata[off++] = byte ' ';
	treedata[off:] = namebs;
	off += len namebs;
	treedata[off++] = byte 0;
	treedata[off:] = hash;

	(entries, err) := git->parsetree(treedata);
	t.assertnil(err, "parsetree should succeed");
	t.asserteq(len entries, 1, "should have one entry");
	t.assertseq(entries[0].name, "hello.txt", "entry name");
	t.asserteq(entries[0].mode, 8r100644, "entry mode");
}

# --- Delta Application ---

testApplyDelta(t: ref T)
{
	# Simple delta: copy entire source + insert literal
	base := array of byte "Hello, World!";

	# Delta format:
	# srcsize varint: 13
	# tgtsize varint: 20 (13 + 7)
	# copy cmd: 0x80 | 0x01 | 0x10 = 0x91, offset=0, size=13
	# insert cmd: 7, "!\n test"
	srcsize := 13;
	tgtsize := 20;
	delta := array [] of {
		byte srcsize,    # src size varint (13, fits in 7 bits)
		byte tgtsize,    # tgt size varint (20, fits in 7 bits)
		byte 16r91,      # copy: offset(1 byte) + size(1 byte)
		byte 0,          # offset = 0
		byte 13,         # size = 13
		byte 7,          # insert 7 bytes
		byte '!', byte ' ', byte 'e', byte 'x', byte 't', byte 'r', byte 'a',
	};

	(result, err) := git->applydelta(base, delta);
	t.assertnil(err, "applydelta should succeed");
	t.asserteq(len result, tgtsize, "result size");
	t.assertseq(string result, "Hello, World!! extra", "delta result");
}

# --- Tag Parsing ---

testParseTag(t: ref T)
{
	tagdata := array of byte (
		"object da39a3ee5e6b4b0d3255bfef95601890afd80709\n" +
		"type commit\n" +
		"tag v1.0\n" +
		"tagger Test User <test@example.com> 1234567890 +0000\n" +
		"\n" +
		"Release v1.0\n");

	(tag, err) := git->parsetag(tagdata);
	t.assertnil(err, "parsetag should succeed");
	t.assertseq(tag.name, "v1.0", "tag name");
	t.asserteq(tag.otype, git->OBJ_COMMIT, "tag object type");
	t.assertseq(tag.msg, "Release v1.0\n", "tag message");
}

# --- PackIdx Tests ---

testPackIdxFind(t: ref T)
{
	# Build a PackIdx in memory with 3 hashes and verify binary search
	# Hashes (sorted by first byte):
	#   h1: 10aa...  (first byte 0x10)
	#   h2: 80bb...  (first byte 0x80)
	#   h3: ff cc... (first byte 0xff)
	nobj := 3;
	hashes := array [nobj * 20] of { * => byte 0 };

	# h1 at index 0
	hashes[0] = byte 16r10;
	i := 0;
	for(i = 1; i < 20; i++)
		hashes[i] = byte 16raa;

	# h2 at index 1
	hashes[20] = byte 16r80;
	for(i = 1; i < 20; i++)
		hashes[20 + i] = byte 16rbb;

	# h3 at index 2
	hashes[40] = byte 16rff;
	for(i = 1; i < 20; i++)
		hashes[40 + i] = byte 16rcc;

	# Build fanout: fanout[i] = count of hashes with first byte <= i
	fanout := array [256] of int;
	for(i = 0; i < 256; i++) {
		count := 0;
		if(i >= 16r10) count++;
		if(i >= 16r80) count++;
		if(i >= 16rff) count++;
		fanout[i] = count;
	}

	offsets := array [3] of { 100, 200, 300 };

	idx := ref PackIdx(fanout, hashes, offsets, nil, nobj);

	# Build Hash objects matching what we put in
	h1: Hash;
	h1.a = array [20] of byte;
	h1.a[0] = byte 16r10;
	for(i = 1; i < 20; i++) h1.a[i] = byte 16raa;

	h2: Hash;
	h2.a = array [20] of byte;
	h2.a[0] = byte 16r80;
	for(i = 1; i < 20; i++) h2.a[i] = byte 16rbb;

	h3: Hash;
	h3.a = array [20] of byte;
	h3.a[0] = byte 16rff;
	for(i = 1; i < 20; i++) h3.a[i] = byte 16rcc;

	(off1, found1) := idx.find(h1);
	t.assert(found1 != 0, "h1 should be found");
	t.asserteq(int off1, 100, "h1 offset");

	(off2, found2) := idx.find(h2);
	t.assert(found2 != 0, "h2 should be found");
	t.asserteq(int off2, 200, "h2 offset");

	(off3, found3) := idx.find(h3);
	t.assert(found3 != 0, "h3 should be found");
	t.asserteq(int off3, 300, "h3 offset");
}

testPackIdxFindEmpty(t: ref T)
{
	# Empty index should return not-found
	fanout := array [256] of { * => 0 };
	hashes := array [0] of byte;
	offsets := array [0] of int;
	idx := ref PackIdx(fanout, hashes, offsets, nil, 0);

	h: Hash;
	h.a = array [20] of { * => byte 16r42 };
	(nil, found) := idx.find(h);
	t.assert(found == 0, "empty index should not find anything");
}

# --- Extended Hash Tests ---

testHashObjCommit(t: ref T)
{
	data := array of byte (
		"tree 4b825dc642cb6eb9a060e54bf899d69f7c355415\n" +
		"author A <a@b.c> 1 +0000\n" +
		"committer A <a@b.c> 1 +0000\n" +
		"\n" +
		"msg\n");
	h := git->hashobj(git->OBJ_COMMIT, data);
	t.assert(!h.isnil(), "commit hash should not be nil");
	t.asserteq(len h.hex(), 40, "commit hash should be 40 chars");
}

testHashObjTree(t: ref T)
{
	# Empty tree has the well-known SHA-1
	data := array [0] of byte;
	h := git->hashobj(git->OBJ_TREE, data);
	t.assertseq(h.hex(), "4b825dc642cb6eb9a060e54bf8d69288fbee4904", "empty tree hash");
}

# --- Repo Tests ---

testInitRepo(t: ref T)
{
	tmpdir := "/tmp/git_test_repo_" + string sys->pctl(0, nil);
	sys->create(tmpdir, Sys->OREAD, Sys->DMDIR | 8r755);

	gitdir := tmpdir + "/.git";
	(repo, err) := git->initrepo(gitdir, 0);
	t.assertnil(err, "initrepo should succeed");
	t.assert(repo != nil, "repo should not be nil");

	# Verify HEAD points to main
	(refname, herr) := repo.head();
	t.assertnil(herr, "head should succeed");
	t.assertseq(refname, "refs/heads/main", "HEAD should point to main");

	# Cleanup
	cleanup(tmpdir);
}

testRefReadWrite(t: ref T)
{
	tmpdir := "/tmp/git_test_ref_" + string sys->pctl(0, nil);
	sys->create(tmpdir, Sys->OREAD, Sys->DMDIR | 8r755);

	gitdir := tmpdir + "/.git";
	(repo, err) := git->initrepo(gitdir, 0);
	t.assertnil(err, "initrepo should succeed");

	# Write a ref manually
	refpath := gitdir + "/refs/heads/testbranch";
	fd := sys->create(refpath, Sys->OWRITE, 8r644);
	t.assert(fd != nil, "create ref file");
	hexstr := "da39a3ee5e6b4b0d3255bfef95601890afd80709";
	data := array of byte (hexstr + "\n");
	sys->write(fd, data, len data);
	fd = nil;

	# Read it back
	(h, rerr) := repo.readref("refs/heads/testbranch");
	t.assertnil(rerr, "readref should succeed");
	t.assertseq(h.hex(), hexstr, "readref should return correct hash");

	# Test symref resolution: HEAD -> refs/heads/testbranch
	headfd := sys->create(gitdir + "/HEAD", Sys->OWRITE, 8r644);
	t.assert(headfd != nil, "create HEAD");
	hdata := array of byte "ref: refs/heads/testbranch\n";
	sys->write(headfd, hdata, len hdata);
	headfd = nil;

	(h2, rerr2) := repo.readref("HEAD");
	t.assertnil(rerr2, "readref HEAD should resolve symref");
	t.assertseq(h2.hex(), hexstr, "HEAD should resolve to testbranch hash");

	cleanup(tmpdir);
}

# --- Extended Commit Parsing ---

testParseCommitMultipleParents(t: ref T)
{
	commitdata := array of byte (
		"tree 4b825dc642cb6eb9a060e54bf899d69f7c355415\n" +
		"parent da39a3ee5e6b4b0d3255bfef95601890afd80709\n" +
		"parent e69de29bb2d1d6434b8b29ae775ad8c2e48c5391\n" +
		"author Merger <m@e.x> 1234567890 +0000\n" +
		"committer Merger <m@e.x> 1234567890 +0000\n" +
		"\n" +
		"Merge branch\n");

	(c, err) := git->parsecommit(commitdata);
	t.assertnil(err, "parsecommit merge should succeed");

	nparents := 0;
	for(pl := c.parents; pl != nil; pl = tl pl)
		nparents++;
	t.asserteq(nparents, 2, "merge commit should have 2 parents");

	# Verify order: first parent listed first
	t.assertseq((hd c.parents).hex(), "da39a3ee5e6b4b0d3255bfef95601890afd80709", "first parent");
	t.assertseq((hd tl c.parents).hex(), "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", "second parent");
}

# --- Extended Tree Parsing ---

testParseTreeMultipleEntries(t: ref T)
{
	# Build a 3-entry tree:
	#   100644 file.txt  <hash_aa>
	#   100755 script.sh <hash_bb>
	#   40000  subdir    <hash_cc>
	entries_data: list of (string, string, array of byte);
	hash_aa := array [20] of { * => byte 16raa };
	hash_bb := array [20] of { * => byte 16rbb };
	hash_cc := array [20] of { * => byte 16rcc };
	entries_data = ("100644", "file.txt", hash_aa) :: entries_data;
	entries_data = ("100755", "script.sh", hash_bb) :: entries_data;
	entries_data = ("40000", "subdir", hash_cc) :: entries_data;

	# Calculate total size
	total := 0;
	for(el := entries_data; el != nil; el = tl el) {
		(m, n, nil) := hd el;
		total += len array of byte m + 1 + len array of byte n + 1 + 20;
	}

	treedata := array [total] of byte;
	off := 0;
	for(el = entries_data; el != nil; el = tl el) {
		(m, n, h) := hd el;
		mb := array of byte m;
		nb := array of byte n;
		for(k := 0; k < len mb; k++)
			treedata[off++] = mb[k];
		treedata[off++] = byte ' ';
		for(k = 0; k < len nb; k++)
			treedata[off++] = nb[k];
		treedata[off++] = byte 0;
		for(k = 0; k < 20; k++)
			treedata[off++] = h[k];
	}

	(entries, err) := git->parsetree(treedata);
	t.assertnil(err, "parsetree 3-entry should succeed");
	t.asserteq(len entries, 3, "should have 3 entries");

	# entries_data was built via cons, so order is reversed: subdir, script.sh, file.txt
	t.assertseq(entries[0].name, "subdir", "entry 0 name");
	t.asserteq(entries[0].mode, 8r40000, "entry 0 mode (directory)");
	t.assertseq(entries[1].name, "script.sh", "entry 1 name");
	t.asserteq(entries[1].mode, 8r100755, "entry 1 mode");
	t.assertseq(entries[2].name, "file.txt", "entry 2 name");
	t.asserteq(entries[2].mode, 8r100644, "entry 2 mode");
}

# --- Extended Delta Tests ---

testApplyDeltaCopyOnly(t: ref T)
{
	# Delta that copies entire source with no inserts
	base := array of byte "ABCDEFGH";
	srcsize := 8;
	tgtsize := 8;

	delta := array [] of {
		byte srcsize,    # src size
		byte tgtsize,    # tgt size
		byte 16r91,      # copy: offset(1) + size(1)
		byte 0,          # offset = 0
		byte 8,          # size = 8
	};

	(result, err) := git->applydelta(base, delta);
	t.assertnil(err, "copy-only delta should succeed");
	t.asserteq(len result, tgtsize, "result size");
	t.assertseq(string result, "ABCDEFGH", "copy-only result");
}

testApplyDeltaInsertOnly(t: ref T)
{
	# Delta from empty source, insert-only
	base := array [0] of byte;
	srcsize := 0;
	tgtsize := 5;

	delta := array [] of {
		byte srcsize,    # src size = 0
		byte tgtsize,    # tgt size = 5
		byte 5,          # insert 5 bytes
		byte 'h', byte 'e', byte 'l', byte 'l', byte 'o',
	};

	(result, err) := git->applydelta(base, delta);
	t.assertnil(err, "insert-only delta should succeed");
	t.asserteq(len result, tgtsize, "result size");
	t.assertseq(string result, "hello", "insert-only result");
}

testApplyDeltaSizeMismatch(t: ref T)
{
	# Wrong source size in delta header should fail
	base := array of byte "ABC";  # 3 bytes
	delta := array [] of {
		byte 99,         # src size = 99 (wrong!)
		byte 3,          # tgt size
		byte 16r91,
		byte 0,
		byte 3,
	};

	(nil, err) := git->applydelta(base, delta);
	t.assertnotnil(err, "size mismatch delta should fail");
}

testApplyDeltaTooShort(t: ref T)
{
	# Truncated delta
	base := array of byte "data";
	delta := array [] of {
		byte 4,          # src size
		# missing tgt size and commands
	};

	git->applydelta(base, delta);
	# The above delta is technically valid with tgt size 0 and no commands.
	# Make a truly truncated one: target says 10 but no data
	delta2 := array [] of {
		byte 4,          # src size = 4
		byte 10,         # tgt size = 10
		# no commands to produce 10 bytes
	};
	(nil, err2) := git->applydelta(base, delta2);
	t.assertnotnil(err2, "truncated delta should fail");
}

# --- Hash Symmetry ---

testHashEqualitySymmetric(t: ref T)
{
	(h1, nil) := git->parsehash("da39a3ee5e6b4b0d3255bfef95601890afd80709");
	(h2, nil) := git->parsehash("da39a3ee5e6b4b0d3255bfef95601890afd80709");
	(h3, nil) := git->parsehash("e69de29bb2d1d6434b8b29ae775ad8c2e48c5391");

	# Symmetry: h1.eq(h2) <=> h2.eq(h1)
	t.assert(h1.eq(h2) && h2.eq(h1), "equality should be symmetric (equal)");
	t.assert(!h1.eq(h3) && !h3.eq(h1), "inequality should be symmetric (not equal)");
}

# --- Upper Case Hash Parsing ---

testParseHashUpperCase(t: ref T)
{
	upper := "DA39A3EE5E6B4B0D3255BFEF95601890AFD80709";
	(h, err) := git->parsehash(upper);
	t.assertnil(err, "parsehash upper case should succeed");
	# hex() always returns lowercase
	t.assertseq(h.hex(), "da39a3ee5e6b4b0d3255bfef95601890afd80709", "hex should be lowercase");
}

# --- Helpers ---

# Recursively remove a directory tree
cleanup(path: string)
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil) {
		sys->remove(path);
		return;
	}
	for(;;) {
		(n, dirs) := sys->dirread(fd);
		if(n <= 0)
			break;
		for(i := 0; i < n; i++) {
			child := path + "/" + dirs[i].name;
			if(dirs[i].qid.qtype & Sys->QTDIR)
				cleanup(child);
			else
				sys->remove(child);
		}
	}
	fd = nil;
	sys->remove(path);
}

# --- Module Init Test ---

testModuleInit(t: ref T)
{
	# Module was already loaded in init(); verify it's not nil
	t.assert(git != nil, "git module should be loaded");
}

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	testing = load Testing Testing->PATH;

	if(testing == nil) {
		sys->fprint(sys->fildes(2), "cannot load testing module: %r\n");
		raise "fail:cannot load testing";
	}

	testing->init();

	for(a := args; a != nil; a = tl a)
		if(hd a == "-v")
			testing->verbose(1);

	# Load git module
	git = load Git Git->PATH;
	if(git == nil) {
		sys->fprint(sys->fildes(2), "cannot load git module: %r\n");
		raise "fail:cannot load git module";
	}

	err := git->init();
	if(err != nil) {
		sys->fprint(sys->fildes(2), "git init: %s\n", err);
		raise "fail:git init: " + err;
	}

	# Run tests
	run("ModuleInit", testModuleInit);
	run("NullHash", testNullHash);
	run("ParseHash", testParseHash);
	run("ParseHashBad", testParseHashBad);
	run("HashEquality", testHashEquality);
	run("HashBlob", testHashBlob);
	run("HashBlobContent", testHashBlobContent);
	run("Typename", testTypename);
	run("Typenum", testTypenum);
	run("ParseCommit", testParseCommit);
	run("ParseCommitNoParent", testParseCommitNoParent);
	run("ParseTree", testParseTree);
	run("ApplyDelta", testApplyDelta);
	run("ParseTag", testParseTag);
	run("PackIdxFind", testPackIdxFind);
	run("PackIdxFindEmpty", testPackIdxFindEmpty);
	run("HashObjCommit", testHashObjCommit);
	run("HashObjTree", testHashObjTree);
	run("InitRepo", testInitRepo);
	run("RefReadWrite", testRefReadWrite);
	run("ParseCommitMultipleParents", testParseCommitMultipleParents);
	run("ParseTreeMultipleEntries", testParseTreeMultipleEntries);
	run("ApplyDeltaCopyOnly", testApplyDeltaCopyOnly);
	run("ApplyDeltaInsertOnly", testApplyDeltaInsertOnly);
	run("ApplyDeltaSizeMismatch", testApplyDeltaSizeMismatch);
	run("ApplyDeltaTooShort", testApplyDeltaTooShort);
	run("HashEqualitySymmetric", testHashEqualitySymmetric);
	run("ParseHashUpperCase", testParseHashUpperCase);

	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
