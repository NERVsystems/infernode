implement HelloTest;

#
# Basic module loading and output test
# Migrated from test-hello.b
#
# Tests:
# - sys module loads correctly
# - sys->print() works
# - Argument list is accessible
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

HelloTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

passed := 0;
failed := 0;
skipped := 0;

# Source file path for clickable error addresses
SRCFILE: con "/tests/hello_test.b";

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

# Test that Sys module loads
testSysModuleLoads(t: ref T)
{
	# sys is already loaded at init time, but let's verify
	t.assert(sys != nil, "Sys module should be loaded");
}

# Test that sys->print works (implicit - if we get here, it worked)
testPrint(t: ref T)
{
	# Capture would require more infrastructure
	# For now, verify we can call print without error
	sys->print("");  # empty print should work
	t.log("sys->print works");
}

# Test argument list handling
testArguments(t: ref T)
{
	# When run via limbtest, args should include at least the program name
	# This test is mainly about verifying the list handling works
	args := list of {"test", "arg1", "arg2"};

	count := 0;
	for(a := args; a != nil; a = tl a)
		count++;

	t.asserteq(count, 3, "argument list should have 3 elements");
	t.assertseq(hd args, "test", "first argument should be 'test'");
}

# Test string formatting with sprint
testSprint(t: ref T)
{
	result := sys->sprint("Hello %s!", "World");
	t.assertseq(result, "Hello World!", "sprint should format strings");

	result = sys->sprint("Number: %d", 42);
	t.assertseq(result, "Number: 42", "sprint should format integers");
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
	run("SysModuleLoads", testSysModuleLoads);
	run("Print", testPrint);
	run("Arguments", testArguments);
	run("Sprint", testSprint);

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
