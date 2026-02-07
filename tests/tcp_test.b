implement TcpTest;

#
# TCP/IP stack tests
# Migrated from test-tcp-simple.b and test-tcp-ip.b
#
# Tests:
# - TCP dial to remote host
# - TCP write to connection
# - TCP read from connection
# - HTTP request/response
#
# Note: These tests require network connectivity.
# Tests will be skipped if network is unavailable.
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

TcpTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

passed := 0;
failed := 0;
skipped := 0;

# Source file path for clickable error addresses
SRCFILE: con "/tests/tcp_test.b";

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

# Test basic TCP dial to an IP address
testTcpDialIp(t: ref T)
{
	# Connect to Google DNS on TCP port 53
	(ok, c) := sys->dial("tcp!8.8.8.8!53", nil);
	if(ok < 0) {
		t.skip(sys->sprint("network unavailable: %r"));
		return;
	}

	t.assert(c.dfd != nil, "data fd should be valid");
	t.log(sys->sprint("connected to 8.8.8.8:53, fd=%d", c.dfd.fd));
}

# Test TCP dial with hostname resolution
testTcpDialHostname(t: ref T)
{
	(ok, c) := sys->dial("tcp!google.com!80", nil);
	if(ok < 0) {
		t.skip(sys->sprint("network unavailable or DNS failed: %r"));
		return;
	}

	t.assert(c.dfd != nil, "data fd should be valid");
	t.log("connected to google.com:80");
}

# Test TCP write
testTcpWrite(t: ref T)
{
	(ok, c) := sys->dial("tcp!8.8.8.8!53", nil);
	if(ok < 0) {
		t.skip(sys->sprint("network unavailable: %r"));
		return;
	}

	# Write a single byte
	buf := array[1] of byte;
	buf[0] = byte 0;
	n := sys->write(c.dfd, buf, 1);
	t.asserteq(n, 1, "write should return 1 byte written");
}

# Test HTTP request and response
testHttpRequest(t: ref T)
{
	(ok, c) := sys->dial("tcp!google.com!80", nil);
	if(ok < 0) {
		t.skip(sys->sprint("network unavailable: %r"));
		return;
	}

	# Send HTTP request
	request := "GET / HTTP/1.0\r\nHost: google.com\r\n\r\n";
	buf := array of byte request;
	n := sys->write(c.dfd, buf, len buf);
	if(n < 0) {
		t.fatal(sys->sprint("write failed: %r"));
		return;
	}
	t.assert(n > 0, "write should succeed");
	t.log(sys->sprint("sent %d bytes", n));

	# Read response
	rbuf := array[512] of byte;
	n = sys->read(c.dfd, rbuf, len rbuf);
	if(n < 0) {
		t.fatal(sys->sprint("read failed: %r"));
		return;
	}
	t.assert(n > 0, "should receive response");
	t.log(sys->sprint("received %d bytes", n));

	# Verify response starts with HTTP
	response := string rbuf[0:n];
	if(len response >= 4 && response[0:4] == "HTTP") {
		t.log("response is HTTP");
	} else {
		t.error("response does not start with HTTP");
	}
}

# Test connection to localhost (may or may not work)
testLocalhostConnect(t: ref T)
{
	# Try to connect to localhost:9999 - typically fails but tests the code path
	(ok, nil) := sys->dial("tcp!127.0.0.1!9999", nil);
	if(ok >= 0) {
		t.log("localhost:9999 was unexpectedly reachable");
	} else {
		# This is expected - no server on 9999
		t.log("localhost:9999 unreachable (expected)");
	}
	# This test always passes - it just exercises the dial code
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
	run("TcpDialIp", testTcpDialIp);
	run("TcpDialHostname", testTcpDialHostname);
	run("TcpWrite", testTcpWrite);
	run("HttpRequest", testHttpRequest);
	run("LocalhostConnect", testLocalhostConnect);

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
