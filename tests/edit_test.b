implement EditTest;

#
# edit_test - Tests for the edit command
#
# Tests the simple text replacement command.
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

include "sh.m";
	sh: Sh;

EditTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

passed := 0;
failed := 0;
skipped := 0;
ctxt: ref Draw->Context;
testprefix := "/usr/inferno/edittest_";

# Source file path for clickable error addresses
SRCFILE: con "/tests/edit_test.b";

# Helper to run a test and track results
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

# Check /tmp is writable by creating a test file
setup(): int
{
	testfile := testprefix + "setup.tmp";
	fd := sys->create(testfile, Sys->OWRITE, 8r644);
	if(fd == nil)
		return -1;
	sys->remove(testfile);
	return 0;
}

# Cleanup all test files
teardown()
{
	# Remove individual test files
	sys->remove(testprefix + "basic.txt");
	sys->remove(testprefix + "quoted.txt");
	sys->remove(testprefix + "middle.txt");
	sys->remove(testprefix + "notfound.txt");
	sys->remove(testprefix + "ambiguous.txt");
	sys->remove(testprefix + "replaceall.txt");
	sys->remove(testprefix + "emptyold.txt");
	sys->remove(testprefix + "special.txt");
}

# Helper to write a file
writefile(path, content: string): int
{
	fd := sys->create(path, Sys->OWRITE, 8r644);
	if(fd == nil)
		return -1;
	b := array of byte content;
	n := sys->write(fd, b, len b);
	return n;
}

# Helper to read a file
readfile(path: string): string
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return nil;
	buf := array[8192] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return "";
	return string buf[0:n];
}

# Test basic replacement
testBasicReplace(t: ref T)
{
	testfile := testprefix + "basic.txt";
	writefile(testfile, "hello world");

	err := sh->system(ctxt, "edit -f " + testfile + " -old hello -new goodbye");
	t.assert(err == nil || err == "", "edit should succeed");

	content := readfile(testfile);
	t.assertseq(content, "goodbye world", "content should be replaced");
}

# Test replacement with quoted strings
testQuotedStrings(t: ref T)
{
	testfile := testprefix + "quoted.txt";
	writefile(testfile, "the quick brown fox");

	err := sh->system(ctxt, "edit -f " + testfile + " -old 'quick brown' -new 'slow gray'");
	t.assert(err == nil || err == "", "edit with quotes should succeed");

	content := readfile(testfile);
	t.assertseq(content, "the slow gray fox", "quoted content should be replaced");
}

# Test replacement in middle of file
testMiddleReplace(t: ref T)
{
	testfile := testprefix + "middle.txt";
	writefile(testfile, "line1\nTARGET\nline3\n");

	err := sh->system(ctxt, "edit -f " + testfile + " -old TARGET -new REPLACED");
	t.assert(err == nil || err == "", "edit should succeed");

	content := readfile(testfile);
	t.assert(hassubstr(content, "REPLACED"), "TARGET should be replaced");
	t.assert(!hassubstr(content, "TARGET"), "TARGET should not remain");
}

# Test not found error
testNotFound(t: ref T)
{
	testfile := testprefix + "notfound.txt";
	writefile(testfile, "hello world");

	err := sh->system(ctxt, "edit -f " + testfile + " -old nonexistent -new replacement");
	t.assert(err != nil && err != "", "edit should fail when text not found");
}

# Test ambiguous (multiple matches) error
testAmbiguous(t: ref T)
{
	testfile := testprefix + "ambiguous.txt";
	writefile(testfile, "foo bar foo baz foo");

	err := sh->system(ctxt, "edit -f " + testfile + " -old foo -new qux");
	t.assert(err != nil && err != "", "edit should fail with multiple matches");

	# Content should be unchanged
	content := readfile(testfile);
	t.assertseq(content, "foo bar foo baz foo", "content should be unchanged on ambiguous match");
}

# Test -all flag for multiple replacements
testReplaceAll(t: ref T)
{
	testfile := testprefix + "replaceall.txt";
	writefile(testfile, "foo bar foo baz foo");

	err := sh->system(ctxt, "edit -f " + testfile + " -old foo -new qux -all");
	t.assert(err == nil || err == "", "edit -all should succeed");

	content := readfile(testfile);
	t.assertseq(content, "qux bar qux baz qux", "all occurrences should be replaced");
}

# Test missing file error
testMissingFile(t: ref T)
{
	err := sh->system(ctxt, "edit -f " + testprefix + "nonexistent.txt -old foo -new bar");
	t.assert(err != nil && err != "", "edit should fail with missing file");
}

# Test empty old string error
testEmptyOld(t: ref T)
{
	testfile := testprefix + "emptyold.txt";
	writefile(testfile, "hello world");

	err := sh->system(ctxt, "edit -f " + testfile + " -old '' -new replacement");
	t.assert(err != nil && err != "", "edit should fail with empty -old");
}

# Test special characters
testSpecialChars(t: ref T)
{
	testfile := testprefix + "special.txt";
	writefile(testfile, "port=8080");

	err := sh->system(ctxt, "edit -f " + testfile + " -old 'port=8080' -new 'port=9090'");
	t.assert(err == nil || err == "", "edit with special chars should succeed");

	content := readfile(testfile);
	t.assertseq(content, "port=9090", "special chars should be replaced");
}

# Helper: check if string contains substring
hassubstr(s, sub: string): int
{
	if(len sub > len s)
		return 0;
	for(i := 0; i <= len s - len sub; i++) {
		match := 1;
		for(j := 0; j < len sub; j++) {
			if(s[i+j] != sub[j]) {
				match = 0;
				break;
			}
		}
		if(match)
			return 1;
	}
	return 0;
}

init(drawctxt: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	testing = load Testing Testing->PATH;
	sh = load Sh Sh->PATH;
	ctxt = drawctxt;

	if(testing == nil) {
		sys->fprint(sys->fildes(2), "cannot load testing module: %r\n");
		raise "fail:cannot load testing";
	}
	if(sh == nil) {
		sys->fprint(sys->fildes(2), "cannot load sh module: %r\n");
		raise "fail:cannot load sh";
	}

	testing->init();

	# Check for verbose flag
	for(a := args; a != nil; a = tl a) {
		if(hd a == "-v")
			testing->verbose(1);
	}

	# Verify /usr/inferno is writable
	if(setup() < 0) {
		sys->fprint(sys->fildes(2), "cannot write to /usr/inferno\n");
		raise "fail:setup";
	}

	# Run tests
	run("BasicReplace", testBasicReplace);
	run("QuotedStrings", testQuotedStrings);
	run("MiddleReplace", testMiddleReplace);
	run("NotFound", testNotFound);
	run("Ambiguous", testAmbiguous);
	run("ReplaceAll", testReplaceAll);
	run("MissingFile", testMissingFile);
	run("EmptyOld", testEmptyOld);
	run("SpecialChars", testSpecialChars);

	# Cleanup
	teardown();

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
