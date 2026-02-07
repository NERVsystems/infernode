implement ExampleTest;

#
# Example test file demonstrating the testing framework
#
# To run: limbtest tests/example_test.b
# Or compile first: limbo -I /module tests/example_test.b
# Then run: emu tests/example_test.dis
#
# Note: Use newTsrc() with your source file path to enable clickable
# error addresses in Xenith. On failure, output like:
#     /tests/example_test.b:/testAdd/
# can be right-clicked to navigate directly to the test function.
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

ExampleTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

# Source file path for clickable error addresses
SRCFILE: con "/tests/example_test.b";

passed := 0;
failed := 0;
skipped := 0;

# Helper to run a test and track results
# Uses newTsrc to enable clickable addresses on failure
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

# Simple function to test
add(a, b: int): int
{
	return a + b;
}

# Another function to test
concat(a, b: string): string
{
	return a + b;
}

# Test the add function
testAdd(t: ref T)
{
	t.asserteq(add(1, 2), 3, "1 + 2 should equal 3");
	t.asserteq(add(0, 0), 0, "0 + 0 should equal 0");
	t.asserteq(add(-1, 1), 0, "-1 + 1 should equal 0");
}

# Test the concat function
testConcat(t: ref T)
{
	t.assertseq(concat("hello", " world"), "hello world", "string concatenation");
	t.assertseq(concat("", "test"), "test", "empty + string");
	t.assertseq(concat("test", ""), "test", "string + empty");
}

# Table-driven test example using a loop
testAddTable(t: ref T)
{
	# Test cases: (a, b, expected)
	cases := array[] of {
		(1, 2, 3),
		(0, 0, 0),
		(-1, 1, 0),
		(100, 200, 300),
		(-50, -50, -100),
	};

	for(i := 0; i < len cases; i++) {
		(a, b, want) := cases[i];
		got := add(a, b);
		if(!t.asserteq(got, want, sys->sprint("add(%d, %d)", a, b)))
			t.log(sys->sprint("case %d failed", i));
	}
}

# Example of skipping a test
testSkipExample(t: ref T)
{
	# Skip this test for demonstration
	t.skip("this test is skipped as an example");
	# This line is never reached
	t.assert(0, "should not reach here");
}

# Example of using log for debugging
testWithLogging(t: ref T)
{
	result := add(10, 20);
	t.log(sys->sprint("computed result: %d", result));
	t.asserteq(result, 30, "10 + 20 should equal 30");
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
	run("Add", testAdd);
	run("Concat", testConcat);
	run("AddTable", testAddTable);
	run("SkipExample", testSkipExample);
	run("WithLogging", testWithLogging);

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
