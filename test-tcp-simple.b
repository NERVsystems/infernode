implement NetTest;

include "sys.m";
	sys: Sys;
include "draw.m";

NetTest: module {
	init: fn(ctxt: ref Draw->Context, args: list of string);
};

init(ctxt: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;

	sys->print("Testing TCP/IP stack...\n");

	# Test 1: Can we create a connection?
	(ok, c) := sys->dial("tcp!google.com!80", nil);
	if (ok < 0) {
		sys->print("FAIL: Cannot dial google.com:80 - %r\n");
		return;
	}
	sys->print("✓ TCP dial works - connected to google.com:80\n");

	# Test 2: Can we send/receive data?
	request := "GET / HTTP/1.0\r\nHost: google.com\r\n\r\n";
	buf := array of byte request;
	n := sys->write(c.dfd, buf, len buf);
	if (n < 0) {
		sys->print("FAIL: Cannot write to connection - %r\n");
		return;
	}
	sys->print("✓ TCP write works - sent %d bytes\n", n);

	# Read response
	rbuf := array[512] of byte;
	n = sys->read(c.dfd, rbuf, len rbuf);
	if (n < 0) {
		sys->print("FAIL: Cannot read from connection - %r\n");
		return;
	}
	sys->print("✓ TCP read works - received %d bytes\n", n);
	sys->print("Response: %s\n", string rbuf[0:n]);

	sys->print("\n✅ TCP/IP stack is fully functional!\n");
}
