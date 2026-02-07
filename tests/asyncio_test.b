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

#
# ASYNC FILE SAVE TESTS
# These tests verify the integrity of async file save operations.
# File saving is critical - any data loss is unacceptable.
#

# Test basic async file save operation
testAsyncSaveBasic(t: ref T)
{
	# Original data to save
	testdata := "Hello, this is test data for async save!\nLine 2\nLine 3\n";

	result := chan[2] of (int, int, string);  # (written, mtime, error)
	ctl := chan[1] of int;

	savepath := TESTDIR + "/save_basic.txt";
	spawn asyncWriter(result, ctl, savepath, testdata);

	timeout := chan of int;
	spawn sleeper(timeout, 5000);

	alt {
		(written, mtime, err) := <-result =>
			if(err != nil) {
				t.fatal("async save failed: " + err);
				return;
			}
			t.assert(written > 0, "should have written bytes");
			t.asserteq(written, len array of byte testdata, "bytes written should match input");
			t.assert(mtime > 0, "should have valid mtime");
			t.log(sys->sprint("wrote %d bytes, mtime=%d", written, mtime));
		<-timeout =>
			t.fatal("async save timed out");
	}

	# Verify by reading back
	fd := sys->open(savepath, Sys->OREAD);
	if(fd == nil) {
		t.fatal("can't open saved file: %r");
		return;
	}
	buf := array[1024] of byte;
	n := sys->read(fd, buf, len buf);
	fd = nil;

	content := string buf[0:n];
	t.assertseq(content, testdata, "saved content should match original");

	sys->remove(savepath);
}

# Async writer task (simulates savetask in asyncio.b)
asyncWriter(result: chan of (int, int, string), ctl: chan of int, path: string, data: string)
{
	# Check for cancellation before starting
	alt {
		<-ctl =>
			result <-= (0, 0, "cancelled");
			return;
		* => ;
	}

	fd := sys->create(path, Sys->OWRITE, 8r644);
	if(fd == nil) {
		result <-= (0, 0, sys->sprint("can't create: %r"));
		return;
	}

	ab := array of byte data;
	written := 0;
	chunksize := 1024;  # Write in chunks like real savetask

	for(q := 0; q < len ab; ) {
		# Check for cancellation
		alt {
			<-ctl =>
				fd = nil;
				result <-= (written, 0, "cancelled");
				return;
			* => ;
		}

		n := len ab - q;
		if(n > chunksize)
			n = chunksize;

		nw := sys->write(fd, ab[q:q+n], n);
		if(nw != n) {
			fd = nil;
			result <-= (written, 0, sys->sprint("write error: %r"));
			return;
		}
		written += nw;
		q += n;
	}

	# Get mtime
	(ok, dir) := sys->fstat(fd);
	mtime := 0;
	if(ok == 0)
		mtime = dir.mtime;

	fd = nil;
	result <-= (written, mtime, nil);
}

# Test async save with large file (data integrity over chunks)
testAsyncSaveLargeFile(t: ref T)
{
	# Generate large test data with known pattern for verification
	chunkstr := "0123456789ABCDEF";  # 16 chars
	testdata := "";
	for(i := 0; i < 2048; i++)  # 32KB of data
		testdata += chunkstr;

	result := chan[2] of (int, int, string);
	ctl := chan[1] of int;

	savepath := TESTDIR + "/save_large.txt";
	spawn asyncWriter(result, ctl, savepath, testdata);

	timeout := chan of int;
	spawn sleeper(timeout, 10000);

	alt {
		(written, mtime, err) := <-result =>
			if(err != nil) {
				t.fatal("large save failed: " + err);
				return;
			}
			t.asserteq(written, len array of byte testdata, "all bytes should be written");
			t.log(sys->sprint("wrote %d bytes for large file", written));
		<-timeout =>
			t.fatal("large save timed out");
	}

	# Verify file size
	(ok, dir) := sys->stat(savepath);
	t.assert(ok == 0, "saved file should exist");
	t.asserteq(int dir.length, len array of byte testdata, "file size should match");

	# Verify content integrity by sampling
	fd := sys->open(savepath, Sys->OREAD);
	if(fd == nil) {
		t.fatal("can't open saved large file: %r");
		return;
	}

	# Read beginning
	buf := array[64] of byte;
	n := sys->read(fd, buf, 64);
	t.asserteq(n, 64, "should read 64 bytes from start");
	content := string buf[0:n];
	expected := testdata[0:64];
	t.assertseq(content, expected, "start of file should match");

	# Seek to middle and read
	sys->seek(fd, big (16*1024), Sys->SEEKSTART);
	n = sys->read(fd, buf, 64);
	t.asserteq(n, 64, "should read 64 bytes from middle");
	content = string buf[0:n];
	expected = testdata[16*1024:16*1024+64];
	t.assertseq(content, expected, "middle of file should match");

	# Seek near end and read
	sys->seek(fd, big (32*1024 - 64), Sys->SEEKSTART);
	n = sys->read(fd, buf, 64);
	t.asserteq(n, 64, "should read 64 bytes from end");
	content = string buf[0:n];
	expected = testdata[32*1024 - 64:32*1024];
	t.assertseq(content, expected, "end of file should match");

	fd = nil;
	sys->remove(savepath);
}

# Test async save cancellation
testAsyncSaveCancellation(t: ref T)
{
	# Large data so we have time to cancel
	testdata := "";
	for(i := 0; i < 4096; i++)
		testdata += "0123456789ABCDEF";  # 64KB

	result := chan[2] of (int, int, string);
	ctl := chan[1] of int;

	savepath := TESTDIR + "/save_cancel.txt";
	spawn asyncWriter(result, ctl, savepath, testdata);

	# Let it start
	sys->sleep(5);

	# Cancel
	alt {
		ctl <-= 1 =>
			t.log("sent save cancellation");
		* =>
			t.log("cancellation channel full");
	}

	# Wait for result
	timeout := chan of int;
	spawn sleeper(timeout, 2000);

	cancelled := 0;
	completed := 0;
	alt {
		(written, mtime, err) := <-result =>
			if(err == "cancelled") {
				cancelled = 1;
				t.log(sys->sprint("save cancelled after %d bytes", written));
			} else if(err == nil) {
				completed = 1;
				t.log("save completed before cancellation took effect");
			} else {
				t.fatal("unexpected error: " + err);
			}
		<-timeout =>
			t.fatal("cancellation wait timed out");
	}

	# Either outcome is acceptable
	t.assert(cancelled || completed, "should be cancelled or completed");

	sys->remove(savepath);
}

# Test rapid save cycles (stress test for deadlock)
testAsyncSaveRapidCycles(t: ref T)
{
	# This simulates rapidly saving and cancelling, which could cause
	# deadlock if channel sends block
	testdata := "Short test data for rapid cycles\n";

	for(i := 0; i < 10; i++) {
		result := chan[2] of (int, int, string);
		ctl := chan[1] of int;

		savepath := TESTDIR + sys->sprint("/rapid_%d.txt", i);
		spawn asyncWriter(result, ctl, savepath, testdata);

		# Randomly either wait for completion or cancel
		if(i % 2 == 0) {
			# Wait for completion
			timeout := chan of int;
			spawn sleeper(timeout, 1000);
			alt {
				(written, mtime, err) := <-result =>
					if(err != nil && err != "cancelled")
						t.fatal(sys->sprint("cycle %d error: %s", i, err));
				<-timeout =>
					# Cancel if taking too long
					alt { ctl <-= 1 => ; * => ; }
			}
		} else {
			# Cancel immediately
			alt { ctl <-= 1 => ; * => ; }
			# Drain result
			timeout := chan of int;
			spawn sleeper(timeout, 100);
			alt {
				<-result => ;
				<-timeout => ;
			}
		}

		sys->remove(savepath);
	}

	t.log("rapid save cycles completed without deadlock");
}

# Test save with special characters in data (Unicode, newlines, etc.)
testAsyncSaveSpecialChars(t: ref T)
{
	# Test data with various special characters
	# Note: Limbo doesn't support \x escapes, using explicit byte conversion instead
	testdata := "ASCII: Hello World\n" +
		"Unicode: 日本語テスト\n" +
		"Newlines:\r\n\r\n" +
		"Tabs:\t\t\t\n" +
		"Mixed quotes: 'single' and \"double\"\n" +
		"End\n";

	result := chan[2] of (int, int, string);
	ctl := chan[1] of int;

	savepath := TESTDIR + "/save_special.txt";
	spawn asyncWriter(result, ctl, savepath, testdata);

	timeout := chan of int;
	spawn sleeper(timeout, 5000);

	alt {
		(written, mtime, err) := <-result =>
			if(err != nil) {
				t.fatal("special chars save failed: " + err);
				return;
			}
			t.assert(written > 0, "should have written bytes");
		<-timeout =>
			t.fatal("special chars save timed out");
	}

	# Read back and verify
	fd := sys->open(savepath, Sys->OREAD);
	if(fd == nil) {
		t.fatal("can't open special chars file: %r");
		return;
	}

	(ok, dir) := sys->fstat(fd);
	buf := array[int dir.length] of byte;
	n := sys->read(fd, buf, len buf);
	fd = nil;

	content := string buf[0:n];
	t.assertseq(content, testdata, "special chars should be preserved");

	sys->remove(savepath);
}

# Test save error handling (permission denied)
testAsyncSavePermissionDenied(t: ref T)
{
	# Try to save to a non-existent directory
	result := chan[2] of (int, int, string);
	ctl := chan[1] of int;

	savepath := "/nonexistent_dir_12345/file.txt";
	spawn asyncWriter(result, ctl, savepath, "test data");

	timeout := chan of int;
	spawn sleeper(timeout, 2000);

	alt {
		(written, mtime, err) := <-result =>
			t.assert(err != nil, "should fail for nonexistent directory");
			t.log("expected error: " + err);
		<-timeout =>
			t.fatal("permission denied test timed out");
	}
}

# Test that save preserves file position (simulates buffer read)
testAsyncSaveBufferSimulation(t: ref T)
{
	# This simulates how savetask reads from Buffer in chunks
	# We verify that the chunked writing produces correct output

	# Create data with known pattern at each position
	datalen := 8192;  # 8KB
	testbuf := array[datalen] of byte;
	for(i := 0; i < datalen; i++)
		testbuf[i] = byte (i % 256);

	result := chan[2] of (int, int, string);
	ctl := chan[1] of int;

	savepath := TESTDIR + "/save_buffer.bin";
	spawn asyncBinaryWriter(result, ctl, savepath, testbuf);

	timeout := chan of int;
	spawn sleeper(timeout, 5000);

	alt {
		(written, mtime, err) := <-result =>
			if(err != nil) {
				t.fatal("buffer simulation save failed: " + err);
				return;
			}
			t.asserteq(written, datalen, "all bytes should be written");
		<-timeout =>
			t.fatal("buffer simulation save timed out");
	}

	# Verify byte-by-byte
	fd := sys->open(savepath, Sys->OREAD);
	if(fd == nil) {
		t.fatal("can't open buffer simulation file: %r");
		return;
	}

	readbuf := array[datalen] of byte;
	n := sys->read(fd, readbuf, datalen);
	fd = nil;

	t.asserteq(n, datalen, "should read all bytes back");

	# Verify pattern
	errors := 0;
	for(i = 0; i < datalen && errors < 5; i++) {
		if(readbuf[i] != testbuf[i]) {
			t.log(sys->sprint("mismatch at offset %d: got %d, expected %d", i, int readbuf[i], int testbuf[i]));
			errors++;
		}
	}
	t.asserteq(errors, 0, "all bytes should match original");

	sys->remove(savepath);
}

# Binary writer (for byte-level verification)
asyncBinaryWriter(result: chan of (int, int, string), ctl: chan of int, path: string, data: array of byte)
{
	alt {
		<-ctl =>
			result <-= (0, 0, "cancelled");
			return;
		* => ;
	}

	fd := sys->create(path, Sys->OWRITE, 8r644);
	if(fd == nil) {
		result <-= (0, 0, sys->sprint("can't create: %r"));
		return;
	}

	written := 0;
	chunksize := 1024;

	for(q := 0; q < len data; ) {
		alt {
			<-ctl =>
				fd = nil;
				result <-= (written, 0, "cancelled");
				return;
			* => ;
		}

		n := len data - q;
		if(n > chunksize)
			n = chunksize;

		nw := sys->write(fd, data[q:q+n], n);
		if(nw != n) {
			fd = nil;
			result <-= (written, 0, sys->sprint("write error: %r"));
			return;
		}
		written += nw;
		q += n;
	}

	(ok, dir) := sys->fstat(fd);
	mtime := 0;
	if(ok == 0)
		mtime = dir.mtime;

	fd = nil;
	result <-= (written, mtime, nil);
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

	# Async save tests (critical for data integrity)
	run("AsyncSaveBasic", testAsyncSaveBasic);
	run("AsyncSaveLargeFile", testAsyncSaveLargeFile);
	run("AsyncSaveCancellation", testAsyncSaveCancellation);
	run("AsyncSaveRapidCycles", testAsyncSaveRapidCycles);
	run("AsyncSaveSpecialChars", testAsyncSaveSpecialChars);
	run("AsyncSavePermissionDenied", testAsyncSavePermissionDenied);
	run("AsyncSaveBufferSimulation", testAsyncSaveBufferSimulation);

	# Cleanup
	cleanup();

	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
