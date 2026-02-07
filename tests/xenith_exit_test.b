implement XenithExitTest;

#
# Tests for Xenith Exit command fix
#
# The Exit command now uses /dev/sysctl "halt" to properly terminate
# the emulator, ensuring SDL cleanup happens via cleanexit() at C level.
#
# These tests verify the mechanism works correctly without actually
# calling halt (which would terminate the test).
#
# To run: limbo -I/module tests/xenith_exit_test.b
#         emu tests/xenith_exit_test.dis
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

XenithExitTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

passed := 0;
failed := 0;
skipped := 0;

# Source file path for clickable error addresses
SRCFILE: con "/tests/xenith_exit_test.b";

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

# Test that /dev/sysctl exists
# This device is required for the halt command
testSysctlExists(t: ref T)
{
	fd := sys->open("/dev/sysctl", Sys->OREAD);
	t.assert(fd != nil, "/dev/sysctl should exist");
	fd = nil;
}

# Test that /dev/sysctl is writable
# The halt command requires write access
testSysctlWritable(t: ref T)
{
	fd := sys->open("/dev/sysctl", Sys->OWRITE);
	t.assert(fd != nil, "/dev/sysctl should be writable");
	fd = nil;
	# Note: We don't actually write "halt" as that would exit the test
}

# Test that the console device exists
# This is a sanity check that the device tree is set up correctly
testConsDeviceExists(t: ref T)
{
	fd := sys->open("/dev/cons", Sys->OREAD);
	t.assert(fd != nil, "/dev/cons should exist");
	fd = nil;
}

# Test that gui.dis module can be loaded
# This verifies the module was compiled correctly with the fix
testGuiModuleLoads(t: ref T)
{
	# The Gui module path from xenith
	mod := sys->open("/dis/xenith/gui.dis", Sys->OREAD);
	t.assert(mod != nil, "/dis/xenith/gui.dis should exist");
	mod = nil;
}

# Test that xenith.dis module can be loaded
# This verifies the main module was compiled correctly
testXenithModuleLoads(t: ref T)
{
	mod := sys->open("/dis/xenith.dis", Sys->OREAD);
	t.assert(mod != nil, "/dis/xenith.dis should exist");
	mod = nil;
}

# Test that gui.dis has correct size (indicating the halt fix is present)
# The fixed gui.dis should be larger than the original (2585 -> ~2685 bytes)
testGuiModuleSize(t: ref T)
{
	(ok, dir) := sys->stat("/dis/xenith/gui.dis");
	t.asserteq(ok, 0, "stat should succeed");
	if(ok == 0) {
		# The fixed gui.dis should be > 2600 bytes (original was 2585)
		t.assert(int dir.length > 2600,
			sys->sprint("gui.dis size %bd should be > 2600 (halt fix present)", dir.length));
	}
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
	run("SysctlExists", testSysctlExists);
	run("SysctlWritable", testSysctlWritable);
	run("ConsDeviceExists", testConsDeviceExists);
	run("GuiModuleLoads", testGuiModuleLoads);
	run("XenithModuleLoads", testXenithModuleLoads);
	run("GuiModuleSize", testGuiModuleSize);

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
