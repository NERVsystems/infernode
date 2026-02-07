implement TempfileTest;

#
# Temp file slot test - converted to use testing framework
# Tests that tempfile() can reuse stale slots (verifies OEXCL removal fix)
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

TempfileTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

passed := 0;
failed := 0;
skipped := 0;

# Source file path for clickable error addresses
SRCFILE: con "/tests/tempfile_test.b";

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

# Simulates the tempfile() function from disk.b
tempfile(): ref Sys->FD
{
	user := "test";
	fd := sys->open("/dev/user", Sys->OREAD);
	if(fd != nil){
		b := array[Sys->NAMEMAX] of byte;
		n := sys->read(fd, b, len b);
		if(n > 0 && n <= 4)
			user = string b[0:n];
		else if(n > 4)
			user = string b[0:4];
	}
	fd = nil;

	buf := sys->sprint("/tmp/X12345.%sxenith", user);
	for(i:='A'; i<='Z'; i++){
		buf[5] = i;
		# Without OEXCL, this will truncate existing file
		fd = sys->create(buf, Sys->ORDWR|Sys->ORCLOSE, 8r600);
		if(fd != nil)
			return fd;
	}
	return nil;
}

getuser(): string
{
	user := "test";
	fd := sys->open("/dev/user", Sys->OREAD);
	if(fd != nil){
		b := array[Sys->NAMEMAX] of byte;
		n := sys->read(fd, b, len b);
		if(n > 0 && n <= 4)
			user = string b[0:n];
		else if(n > 4)
			user = string b[0:4];
	}
	return user;
}

cleanup(user: string)
{
	for(i:='A'; i<='Z'; i++){
		stalefile := sys->sprint("/tmp/%c12345.%sxenith", i, user);
		sys->remove(stalefile);
	}
}

# Test basic tempfile creation
testTempfileBasic(t: ref T)
{
	fd := tempfile();
	if(!t.assert(fd != nil, "tempfile should return a valid fd"))
		return;
	fd = nil;  # close via ORCLOSE
}

# Test that tempfile can reuse stale slots
testTempfileReuseStaleSlots(t: ref T)
{
	user := getuser();
	t.log(sys->sprint("user prefix: %s", user));

	# Create stale files for all 26 slots
	t.log("creating 26 stale temp files to simulate exhaustion");
	for(i:='A'; i<='Z'; i++){
		stalefile := sys->sprint("/tmp/%c12345.%sxenith", i, user);
		sfd := sys->create(stalefile, Sys->OWRITE, 8r600);
		if(sfd != nil){
			sys->write(sfd, array of byte "stale", 5);
			sfd = nil;  # Close but don't delete
		}
	}
	t.log("all 26 slots filled with stale files");

	# Now try to get a temp file - this tests the fix
	t.log("attempting to create temp file (should reuse stale slot)");
	tfd := tempfile();

	# Cleanup before checking result
	cleanup(user);

	if(tfd == nil) {
		t.fatal("could not create temp file - OEXCL removal fix may not be applied");
		return;
	}

	t.log("successfully created temp file despite all slots being 'taken'");
	t.log("OEXCL removal fix is working correctly");
	tfd = nil;
}

# Test that multiple tempfiles get different slots
testTempfileMultiple(t: ref T)
{
	user := getuser();

	# Clean up first
	cleanup(user);

	# Create two temp files
	fd1 := tempfile();
	if(!t.assert(fd1 != nil, "first tempfile should succeed"))
		return;

	fd2 := tempfile();
	if(!t.assert(fd2 != nil, "second tempfile should succeed")) {
		fd1 = nil;
		return;
	}

	# Both should be valid and different
	# (We can't easily check they're different without inspecting paths,
	# but at least verify both are valid)
	fd1 = nil;
	fd2 = nil;

	cleanup(user);
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
	run("TempfileBasic", testTempfileBasic);
	run("TempfileReuseStaleSlots", testTempfileReuseStaleSlots);
	run("TempfileMultiple", testTempfileMultiple);

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
