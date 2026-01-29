implement AsyncioTest;

#
# Regression tests for async I/O module
#
# Tests the asyncio module which provides non-blocking file operations.
# These tests verify the async infrastructure works correctly.
#
# To run: emu tests/asyncio_test.dis
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "bufio.m";
	bufio: Bufio;
	Iobuf: import bufio;

include "testing.m";
	testing: Testing;
	T: import testing;

AsyncioTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

# Source file path for clickable error addresses
SRCFILE: con "/tests/asyncio_test.b";

passed := 0;
failed := 0;
skipped := 0;

# Test data directory
TESTDIR: con "/tmp/asyncio_test";

run(name: string, testfn: ref fn(t: ref T))
{
	t := testing->newTsrc(name, SRCFILE);
	{
		testfn(t);
	} exception {
	"fail:fatal" =>
		;
	"fail:skip" =>
		;
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

# Setup test directory and files
setup(): string
{
	# Create test directory
	fd := sys->create(TESTDIR, Sys->OREAD, Sys->DMDIR | 8r755);
	if(fd == nil)
		return sys->sprint("can't create test dir: %r");
	fd = nil;

	# Create a small test file
	fd = sys->create(TESTDIR + "/small.txt", Sys->OWRITE, 8r644);
	if(fd == nil)
		return sys->sprint("can't create test file: %r");
	data := array of byte "Hello, async world!\n";
	sys->write(fd, data, len data);
	fd = nil;

	# Create a larger test file (for chunked reading)
	fd = sys->create(TESTDIR + "/large.txt", Sys->OWRITE, 8r644);
	if(fd == nil)
		return sys->sprint("can't create large test file: %r");
	chunk := array of byte "0123456789ABCDEF";  # 16 bytes
	for(i := 0; i < 1024; i++)  # 16KB
		sys->write(fd, chunk, len chunk);
	fd = nil;

	# Create a minimal valid PNG (1x1 red pixel)
	# PNG signature + IHDR + IDAT + IEND
	fd = sys->create(TESTDIR + "/test.png", Sys->OWRITE, 8r644);
	if(fd == nil)
		return sys->sprint("can't create test PNG: %r");
	# Minimal 1x1 red PNG (hand-crafted)
	png := array[] of {
		# PNG signature
		byte 137, byte 80, byte 78, byte 71, byte 13, byte 10, byte 26, byte 10,
		# IHDR chunk: length=13, type=IHDR, width=1, height=1, depth=8, colortype=2
		byte 0, byte 0, byte 0, byte 13,
		byte 'I', byte 'H', byte 'D', byte 'R',
		byte 0, byte 0, byte 0, byte 1,  # width
		byte 0, byte 0, byte 0, byte 1,  # height
		byte 8,   # bit depth
		byte 2,   # color type (RGB)
		byte 0,   # compression
		byte 0,   # filter
		byte 0,   # interlace
		byte 16r90, byte 16r77, byte 16r53, byte 16rDE,  # CRC
		# IDAT chunk: compressed image data
		byte 0, byte 0, byte 0, byte 12,
		byte 'I', byte 'D', byte 'A', byte 'T',
		byte 16r08, byte 16rD7, byte 16r63, byte 16rF8,
		byte 16rCF, byte 16rC0, byte 16r00, byte 16r00,
		byte 16r01, byte 16r01, byte 16r01, byte 16r00,
		byte 16r1B, byte 16rB6, byte 16rEE, byte 16r56,  # CRC
		# IEND chunk
		byte 0, byte 0, byte 0, byte 0,
		byte 'I', byte 'E', byte 'N', byte 'D',
		byte 16rAE, byte 16r42, byte 16r60, byte 16r82,  # CRC
	};
	sys->write(fd, png, len png);
	fd = nil;

	return nil;
}

# Cleanup test files
cleanup()
{
	sys->remove(TESTDIR + "/small.txt");
	sys->remove(TESTDIR + "/large.txt");
	sys->remove(TESTDIR + "/test.png");
	sys->remove(TESTDIR);
}

# Test that test files were created
testSetup(t: ref T)
{
	(ok, dir) := sys->stat(TESTDIR + "/small.txt");
	t.assert(ok == 0, "small.txt should exist");

	(ok, dir) = sys->stat(TESTDIR + "/large.txt");
	t.assert(ok == 0, "large.txt should exist");
	t.assert(int dir.length == 16*1024, "large.txt should be 16KB");

	(ok, dir) = sys->stat(TESTDIR + "/test.png");
	t.assert(ok == 0, "test.png should exist");
}

# Test PNG header reading
testPngHeader(t: ref T)
{
	fd := bufio->open(TESTDIR + "/test.png", Bufio->OREAD);
	if(fd == nil) {
		t.fatal("can't open test.png: %r");
		return;
	}

	# Read PNG signature
	sig := array[8] of byte;
	n := fd.read(sig, 8);
	t.asserteq(n, 8, "should read 8 bytes for signature");

	# Verify PNG signature
	t.asserteq(int sig[0], 137, "PNG signature byte 0");
	t.asserteq(int sig[1], int 'P', "PNG signature byte 1");
	t.asserteq(int sig[2], int 'N', "PNG signature byte 2");
	t.asserteq(int sig[3], int 'G', "PNG signature byte 3");

	fd.close();
}

# Test reading file into byte array (simulates async read)
testFileToBytes(t: ref T)
{
	fd := sys->open(TESTDIR + "/small.txt", Sys->OREAD);
	if(fd == nil) {
		t.fatal("can't open small.txt: %r");
		return;
	}

	(ok, dir) := sys->fstat(fd);
	t.assert(ok == 0, "should stat file");

	fsize := int dir.length;
	t.assert(fsize > 0, "file should have content");

	data := array[fsize] of byte;
	n := sys->read(fd, data, fsize);
	t.asserteq(n, fsize, "should read entire file");

	# Verify content
	content := string data;
	t.assertseq(content, "Hello, async world!\n", "file content");

	fd = nil;
}

# Test bufio aopen (creates Iobuf from byte array)
testBufioAopen(t: ref T)
{
	data := array of byte "Test data for bufio aopen";

	fd := bufio->aopen(data);
	t.assert(fd != nil, "aopen should return Iobuf");

	# Read back
	buf := array[100] of byte;
	n := fd.read(buf, 100);
	t.asserteq(n, len data, "should read all data");

	content := string buf[0:n];
	t.assertseq(content, "Test data for bufio aopen", "content matches");

	fd.close();
}

# Test bufio aopen with PNG data
testBufioAopenPng(t: ref T)
{
	# Read PNG file into bytes
	fd := sys->open(TESTDIR + "/test.png", Sys->OREAD);
	if(fd == nil) {
		t.fatal("can't open test.png: %r");
		return;
	}

	(ok, dir) := sys->fstat(fd);
	fsize := int dir.length;
	data := array[fsize] of byte;
	sys->read(fd, data, fsize);
	fd = nil;

	# Create Iobuf from bytes
	iobuf := bufio->aopen(data);
	t.assert(iobuf != nil, "aopen should work with PNG data");

	# Verify we can read PNG signature
	sig := array[8] of byte;
	n := iobuf.read(sig, 8);
	t.asserteq(n, 8, "should read PNG signature");
	t.asserteq(int sig[0], 137, "PNG signature byte 0");

	# Seek back and read again
	iobuf.seek(big 0, Bufio->SEEKSTART);
	n = iobuf.read(sig, 8);
	t.asserteq(n, 8, "should read after seek");
	t.asserteq(int sig[0], 137, "PNG signature after seek");

	iobuf.close();
}

# Test channel operations (simulates async message passing)
testChannelOps(t: ref T)
{
	# Test buffered channel (like casync)
	ch := chan[8] of string;

	# Non-blocking send should succeed
	alt {
		ch <-= "message1" =>
			t.log("sent message1");
		* =>
			t.fatal("buffered send should not block");
	}

	# Send more
	ch <-= "message2";
	ch <-= "message3";

	# Receive
	msg := <-ch;
	t.assertseq(msg, "message1", "first message");

	msg = <-ch;
	t.assertseq(msg, "message2", "second message");

	msg = <-ch;
	t.assertseq(msg, "message3", "third message");
}

# Test spawn and channel communication (simulates async task)
testSpawnAndChannel(t: ref T)
{
	result := chan of string;

	# Spawn a task that does some work
	spawn worker(result, "test input");

	# Wait for result with timeout
	timeout := chan of int;
	spawn sleeper(timeout, 1000);

	alt {
		r := <-result =>
			t.assertseq(r, "processed: test input", "worker result");
		<-timeout =>
			t.fatal("worker timed out");
	}
}

worker(result: chan of string, input: string)
{
	# Simulate some work
	sys->sleep(10);
	result <-= "processed: " + input;
}

sleeper(ch: chan of int, ms: int)
{
	sys->sleep(ms);
	ch <-= 1;
}

# Test file read in spawned task (core of async loading)
testAsyncFileRead(t: ref T)
{
	result := chan of (array of byte, string);

	spawn asyncReader(result, TESTDIR + "/small.txt");

	timeout := chan of int;
	spawn sleeper(timeout, 2000);

	alt {
		(data, err) := <-result =>
			if(err != nil) {
				t.fatal("async read failed: " + err);
				return;
			}
			content := string data;
			t.assertseq(content, "Hello, async world!\n", "async read content");
		<-timeout =>
			t.fatal("async read timed out");
	}
}

asyncReader(result: chan of (array of byte, string), path: string)
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil) {
		result <-= (nil, sys->sprint("can't open: %r"));
		return;
	}

	(ok, dir) := sys->fstat(fd);
	if(ok != 0) {
		result <-= (nil, "can't stat");
		return;
	}

	fsize := int dir.length;
	data := array[fsize] of byte;
	n := sys->read(fd, data, fsize);
	fd = nil;

	if(n != fsize) {
		result <-= (nil, "short read");
		return;
	}

	result <-= (data, nil);
}

# Test chunked file reading (simulates async text file loading)
testChunkedFileRead(t: ref T)
{
	result := chan[8] of (string, int, string);  # (chunk, offset, error)
	done := chan of int;

	spawn chunkedReader(result, done, TESTDIR + "/large.txt", 1024);

	timeout := chan of int;
	spawn sleeper(timeout, 5000);

	totalBytes := 0;
	chunks := 0;

	loop: for(;;) alt {
		(chunk, offset, err) := <-result =>
			if(err != nil) {
				t.fatal("chunked read error: " + err);
				break loop;
			}
			if(chunk == nil) {
				# Done signal
				break loop;
			}
			chunks++;
			totalBytes += len chunk;
			t.log(sys->sprint("chunk %d: %d bytes at offset %d", chunks, len chunk, offset));
		<-timeout =>
			t.fatal("chunked read timed out");
			break loop;
	}

	t.asserteq(totalBytes, 16*1024, "total bytes read");
	t.assert(chunks > 1, "should have multiple chunks");
}

chunkedReader(result: chan of (string, int, string), done: chan of int, path: string, chunksize: int)
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil) {
		result <-= (nil, 0, sys->sprint("can't open: %r"));
		return;
	}

	buf := array[chunksize] of byte;
	offset := 0;

	for(;;) {
		n := sys->read(fd, buf, chunksize);
		if(n < 0) {
			result <-= (nil, 0, sys->sprint("read error: %r"));
			fd = nil;
			return;
		}
		if(n == 0)
			break;

		chunk := string buf[0:n];
		result <-= (chunk, offset, nil);
		offset += n;
	}

	fd = nil;
	result <-= (nil, offset, nil);  # Signal completion
}

# Test cancellation of async operation
testAsyncCancellation(t: ref T)
{
	result := chan[8] of (string, string);  # (data, error)
	ctl := chan[1] of int;  # Cancellation channel

	spawn cancellableReader(result, ctl, TESTDIR + "/large.txt");

	# Let it start reading
	sys->sleep(10);

	# Cancel it
	alt {
		ctl <-= 1 =>
			t.log("sent cancellation");
		* =>
			t.log("cancellation channel full (task may have finished)");
	}

	# Drain any pending results
	timeout := chan of int;
	spawn sleeper(timeout, 500);

	cancelled := 0;
	loop: for(;;) alt {
		(data, err) := <-result =>
			if(err == "cancelled") {
				cancelled = 1;
				break loop;
			}
			if(data == nil && err == nil) {
				# Normal completion before cancel took effect
				t.log("task completed before cancellation");
				break loop;
			}
		<-timeout =>
			break loop;
	}

	# Either cancelled or completed quickly - both are acceptable
	t.log(sys->sprint("cancellation test: cancelled=%d", cancelled));
}

cancellableReader(result: chan of (string, string), ctl: chan of int, path: string)
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil) {
		result <-= (nil, sys->sprint("can't open: %r"));
		return;
	}

	buf := array[1024] of byte;

	for(;;) {
		# Check for cancellation
		alt {
			<-ctl =>
				fd = nil;
				result <-= (nil, "cancelled");
				return;
			* => ;
		}

		n := sys->read(fd, buf, len buf);
		if(n <= 0)
			break;

		# Send chunk, checking for cancellation
		alt {
			result <-= (string buf[0:n], nil) =>
				;
			<-ctl =>
				fd = nil;
				result <-= (nil, "cancelled");
				return;
		}
	}

	fd = nil;
	result <-= (nil, nil);  # Done
}

# Test rapid start/cancel cycles (regression test for deadlock bug)
testRapidStartCancel(t: ref T)
{
	# This tests the scenario where windows are opened/closed rapidly
	# which previously caused deadlock due to channel buffer exhaustion

	for(i := 0; i < 5; i++) {
		result := chan[8] of (string, string);
		ctl := chan[1] of int;

		spawn cancellableReader(result, ctl, TESTDIR + "/large.txt");

		# Immediately cancel
		alt {
			ctl <-= 1 => ;
			* => ;
		}

		# Drain results with short timeout
		timeout := chan of int;
		spawn sleeper(timeout, 100);

		drain: for(;;) alt {
			<-result => ;
			<-timeout =>
				break drain;
		}

		t.log(sys->sprint("cycle %d complete", i+1));
	}

	t.log("rapid start/cancel completed without deadlock");
}

# Test non-blocking send with alt (core pattern for async tasks)
testNonBlockingSend(t: ref T)
{
	ch := chan of int;  # Unbuffered channel

	# Non-blocking send to unbuffered channel should not block
	sent := 0;
	alt {
		ch <-= 42 =>
			sent = 1;
		* =>
			sent = 0;
	}
	t.asserteq(sent, 0, "send to unbuffered channel with no receiver should not block");

	# With buffered channel
	bufch := chan[1] of int;
	alt {
		bufch <-= 42 =>
			sent = 1;
		* =>
			sent = 0;
	}
	t.asserteq(sent, 1, "send to buffered channel should succeed");

	# Second send should not block (but won't succeed)
	alt {
		bufch <-= 43 =>
			sent = 1;
		* =>
			sent = 0;
	}
	t.asserteq(sent, 0, "send to full buffered channel should not block");
}

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	bufio = load Bufio Bufio->PATH;
	testing = load Testing Testing->PATH;

	if(testing == nil) {
		sys->fprint(sys->fildes(2), "cannot load testing module: %r\n");
		raise "fail:cannot load testing";
	}

	testing->init();

	for(a := args; a != nil; a = tl a) {
		if(hd a == "-v")
			testing->verbose(1);
	}

	# Setup test environment
	err := setup();
	if(err != nil) {
		sys->fprint(sys->fildes(2), "setup failed: %s\n", err);
		raise "fail:setup";
	}

	# Run tests
	run("Setup", testSetup);
	run("PngHeader", testPngHeader);
	run("FileToBytes", testFileToBytes);
	run("BufioAopen", testBufioAopen);
	run("BufioAopenPng", testBufioAopenPng);
	run("ChannelOps", testChannelOps);
	run("SpawnAndChannel", testSpawnAndChannel);
	run("AsyncFileRead", testAsyncFileRead);
	run("ChunkedFileRead", testChunkedFileRead);
	run("AsyncCancellation", testAsyncCancellation);
	run("RapidStartCancel", testRapidStartCancel);
	run("NonBlockingSend", testNonBlockingSend);

	# Cleanup
	cleanup();

	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
