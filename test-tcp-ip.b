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

	sys->print("Testing TCP/IP with direct IP address...\n");

	# Google DNS: 8.8.8.8:53 (DNS server)
	# Cloudflare DNS: 1.1.1.1:53
	(ok, c) := sys->dial("tcp!8.8.8.8!53", nil);
	if (ok < 0) {
		sys->print("FAIL: Cannot dial 8.8.8.8:53 - %r\n");
		return;
	}
	sys->print("✓ TCP dial works to 8.8.8.8:53\n");
	sys->print("✓ Connection file descriptor: %d\n", c.dfd.fd);

	# Try sending a byte
	buf := array[1] of byte;
	buf[0] = byte 0;
	n := sys->write(c.dfd, buf, 1);
	sys->print("✓ TCP write returned: %d\n", n);

	sys->print("\n✅ TCP/IP stack can establish connections!\n");
}
