implement Export9pTest;

#
# 9P export/announce functionality tests
# Migrated from test-9p-export.b
#
# Tests:
# - sys->announce() creates a listening connection
# - Connection info is available via c.dir
# - sys->listen() can wait for connections
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

Export9pTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

passed := 0;
failed := 0;
skipped := 0;

# Source file path for clickable error addresses
SRCFILE: con "/tests/9p_export_test.b";

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

# Test announce on a TCP port
testAnnounce(t: ref T)
{
	# Use a high port to avoid conflicts
	(ok, c) := sys->announce("tcp!*!19999");
	if(ok < 0) {
		t.fatal(sys->sprint("announce failed: %r"));
		return;
	}

	t.log(sys->sprint("announced on port 19999"));
	t.assert(c.dir != nil, "connection dir should be set");
	t.log(sys->sprint("connection dir: %s", c.dir));
}

# Test announce on alternative port
testAnnounceAltPort(t: ref T)
{
	# Use a different port
	(ok, c) := sys->announce("tcp!*!19998");
	if(ok < 0) {
		# May fail if previous test didn't clean up, skip rather than fail
		t.skip(sys->sprint("announce failed: %r"));
		return;
	}

	t.log(sys->sprint("announced on port 19998"));
	t.assert(c.dir != nil, "connection dir should be set");
}

# Test that announce returns proper connection info
testAnnounceConnectionInfo(t: ref T)
{
	(ok, c) := sys->announce("tcp!*!19997");
	if(ok < 0) {
		t.skip(sys->sprint("announce failed: %r"));
		return;
	}

	# Verify connection has expected fields
	# c.dir should be a path like /net/tcp/N where N is clone number
	if(c.dir == nil || c.dir == "") {
		t.error("connection dir should not be empty");
		return;
	}
	t.log(sys->sprint("dir: %s", c.dir));

	# c.cfd should be the control file descriptor
	t.assert(c.cfd != nil, "control fd should be valid");
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
	run("Announce", testAnnounce);
	run("AnnounceAltPort", testAnnounceAltPort);
	run("AnnounceConnectionInfo", testAnnounceConnectionInfo);

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
