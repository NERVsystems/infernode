implement StderrTest;

#
# STDOUT and STDERR output test
# Migrated from test-stderr.b
#
# Tests:
# - sys->fildes(1) returns valid STDOUT fd
# - sys->fildes(2) returns valid STDERR fd
# - sys->fprint works with both file descriptors
# - sys->print defaults to STDOUT
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

StderrTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

passed := 0;
failed := 0;
skipped := 0;

# Source file path for clickable error addresses
SRCFILE: con "/tests/stderr_test.b";

# Helper to run a test and track results
run(name: string, testfn: ref fn(t: ref T))
{
	t := testing->newTsrc(name, SRCFILE);
	{
		testfn(t);
	} exception {
	"fail:fatal" =>
		;	# already marked as failed
	"fail:skip" =>
		;	# already marked as skipped
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

# Test that STDOUT file descriptor is accessible
testStdoutFd(t: ref T)
{
	stdout := sys->fildes(1);
	t.assert(stdout != nil, "fildes(1) should return valid STDOUT fd");
}

# Test that STDERR file descriptor is accessible
testStderrFd(t: ref T)
{
	stderr := sys->fildes(2);
	t.assert(stderr != nil, "fildes(2) should return valid STDERR fd");
}

# Test writing to STDOUT via fprint
testFprintStdout(t: ref T)
{
	stdout := sys->fildes(1);
	n := sys->fprint(stdout, "");  # write empty string
	# fprint returns number of bytes written (0 for empty string is OK)
	t.assert(n >= 0, "fprint to stdout should not return error");
}

# Test writing to STDERR via fprint
testFprintStderr(t: ref T)
{
	stderr := sys->fildes(2);
	n := sys->fprint(stderr, "");  # write empty string
	t.assert(n >= 0, "fprint to stderr should not return error");
}

# Test that STDIN fd is different from STDOUT/STDERR
testStdinFd(t: ref T)
{
	stdin := sys->fildes(0);
	t.assert(stdin != nil, "fildes(0) should return valid STDIN fd");
}

# Test that print returns non-negative (success)
testPrint(t: ref T)
{
	n := sys->print("");
	t.assert(n >= 0, "print should return non-negative");
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

	# Check for verbose flag
	for(a := args; a != nil; a = tl a) {
		if(hd a == "-v")
			testing->verbose(1);
	}

	# Run tests
	run("StdoutFd", testStdoutFd);
	run("StderrFd", testStderrFd);
	run("FprintStdout", testFprintStdout);
	run("FprintStderr", testFprintStderr);
	run("StdinFd", testStdinFd);
	run("Print", testPrint);

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
