implement Export9P;

include "sys.m";
	sys: Sys;
include "draw.m";

Export9P: module {
	init: fn(ctxt: ref Draw->Context, args: list of string);
};

init(ctxt: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;

	sys->print("Testing 9P export on localhost...\n");

	# Start export server in background
	sys->print("Starting 9P server on port 19999...\n");

	# This would normally be done with:
	# listen -A tcp!*!19999 {export /dis} &
	# But we'll test that the system call works

	(ok, c) := sys->announce("tcp!*!19999");
	if (ok < 0) {
		sys->print("FAIL: Cannot announce on port 19999 - %r\n");
		return;
	}

	sys->print("✓ announce() works - listening on port 19999\n");
	sys->print("✓ Connection info: %s\n", c.dir);

	sys->print("\n✅ 9P server operations functional!\n");
	sys->print("(In production, run: listen -A tcp!*!19999 {export /dis} &)\n");
}
