implement XenithConcurrencyTest;

#
# Tests for Xenith concurrency improvements:
# - Buffered channel behavior
# - Async I/O module
# - Non-blocking operations
#
# To run: emu -r../.. sh -c 'cd /tests; limbo xenith_concurrency_test.b && xenith_concurrency_test'
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

XenithConcurrencyTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

# Source file path for clickable error addresses
SRCFILE: con "/tests/xenith_concurrency_test.b";

passed := 0;
failed := 0;
skipped := 0;

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

#
# Test 1: Buffered channel sends don't block
#
testBufferedChannelNonBlocking(t: ref T)
{
	# Create buffered channel like xenith uses
	ch := chan[8] of string;

	# Should be able to send 8 messages without blocking
	for(i := 0; i < 8; i++) {
		alt {
			ch <-= sys->sprint("msg%d", i) =>
				;  # sent successfully
			* =>
				t.error(sys->sprint("buffered send %d blocked unexpectedly", i));
		}
	}

	# 9th send should block (channel full)
	blocked := 0;
	alt {
		ch <-= "overflow" =>
			;  # shouldn't happen
		* =>
			blocked = 1;
	}
	t.assert(blocked == 1, "9th send to buffer[8] channel should block");

	# Drain the channel
	for(i = 0; i < 8; i++)
		<-ch;

	t.log("buffered channel behaves correctly");
}

#
# Test 2: Unbuffered channel blocks immediately
#
testUnbufferedChannelBlocks(t: ref T)
{
	ch := chan of string;

	# Send should block immediately on unbuffered channel
	blocked := 0;
	alt {
		ch <-= "test" =>
			;  # shouldn't happen without receiver
		* =>
			blocked = 1;
	}
	t.assert(blocked == 1, "unbuffered channel send should block without receiver");
}

#
# Test 3: Channel with buffer of 1 (like cedit)
#
testSingleBufferChannel(t: ref T)
{
	ch := chan[1] of int;

	# First send should succeed
	sent1 := 0;
	alt {
		ch <-= 1 =>
			sent1 = 1;
		* =>
			;
	}
	t.assert(sent1 == 1, "first send to buffer[1] should succeed");

	# Second send should block
	sent2 := 0;
	alt {
		ch <-= 2 =>
			sent2 = 1;
		* =>
			;
	}
	t.assert(sent2 == 0, "second send to buffer[1] should block");

	# Receive should unblock
	val := <-ch;
	t.asserteq(val, 1, "received value should be 1");
}

#
# Test 4: Spawned task can send to buffered channel without blocking main
#
testSpawnedTaskBufferedSend(t: ref T)
{
	ch := chan[4] of int;
	done := chan of int;

	# Spawn a task that sends 4 values
	spawn sendtask(ch, done);

	# Wait for task to complete (it shouldn't block since channel is buffered)
	timeout := chan of int;
	spawn timeouttask(timeout, 1000);  # 1 second timeout

	alt {
		<-done =>
			t.log("spawned task completed without blocking");
		<-timeout =>
			t.error("spawned task blocked on buffered channel send");
	}

	# Verify all 4 values were sent
	for(i := 0; i < 4; i++) {
		val := <-ch;
		t.asserteq(val, i, sys->sprint("value %d", i));
	}
}

sendtask(ch: chan of int, done: chan of int)
{
	for(i := 0; i < 4; i++)
		ch <-= i;
	done <-= 1;
}

timeouttask(ch: chan of int, ms: int)
{
	sys->sleep(ms);
	alt {
		ch <-= 1 =>
			;
		* =>
			;  # channel might be gone
	}
}

#
# Test 5: Alt with multiple buffered channels
#
testAltMultipleBuffered(t: ref T)
{
	ch1 := chan[2] of int;
	ch2 := chan[2] of string;
	ch3 := chan[2] of int;

	# Pre-fill ch2
	ch2 <-= "ready";

	# Alt should select ch2 since it has data
	selected := 0;
	alt {
		v := <-ch1 =>
			selected = 1;
			t.log(sys->sprint("ch1: %d", v));
		s := <-ch2 =>
			selected = 2;
			t.assertseq(s, "ready", "ch2 value");
		v := <-ch3 =>
			selected = 3;
			t.log(sys->sprint("ch3: %d", v));
		* =>
			selected = 0;
	}
	t.asserteq(selected, 2, "alt should select ch2 with ready data");
}

#
# Test 6: Non-blocking send pattern (like in asynccancel)
#
testNonBlockingSendPattern(t: ref T)
{
	ctl := chan[1] of int;

	# Non-blocking send should succeed on empty buffered channel
	sent := 0;
	alt {
		ctl <-= 1 =>
			sent = 1;
		* =>
			;
	}
	t.assert(sent == 1, "non-blocking send to empty buffer[1] should succeed");

	# Second non-blocking send should fall through
	sent = 0;
	alt {
		ctl <-= 2 =>
			sent = 1;
		* =>
			;  # expected path
	}
	t.assert(sent == 0, "non-blocking send to full buffer[1] should fall through");
}

#
# Test 7: Channel receive with timeout (async pattern)
#
testReceiveWithTimeout(t: ref T)
{
	ch := chan of int;
	timeout := chan of int;

	spawn timeouttask(timeout, 100);  # 100ms timeout

	result := -1;
	timedout := 0;
	alt {
		v := <-ch =>
			result = v;
		<-timeout =>
			timedout = 1;
	}

	t.assert(timedout == 1, "should timeout when no sender");
	t.asserteq(result, -1, "result should be unchanged on timeout");
}

#
# Test 8: Large buffer stress test
#
testLargeBuffer(t: ref T)
{
	BUFSIZE: con 32;
	ch := chan[BUFSIZE] of int;

	# Fill the buffer completely
	for(i := 0; i < BUFSIZE; i++) {
		alt {
			ch <-= i =>
				;
			* =>
				t.error(sys->sprint("send %d blocked unexpectedly", i));
		}
	}

	# Verify buffer is full
	blocked := 0;
	alt {
		ch <-= 999 =>
			;
		* =>
			blocked = 1;
	}
	t.assert(blocked == 1, sys->sprint("buffer[%d] should be full after %d sends", BUFSIZE, BUFSIZE));

	# Drain and verify order (FIFO)
	for(i = 0; i < BUFSIZE; i++) {
		val := <-ch;
		if(!t.asserteq(val, i, sys->sprint("FIFO order at %d", i)))
			break;
	}
}

#
# Test 9: Concurrent producers with buffered channel
#
testConcurrentProducers(t: ref T)
{
	NPRODUCERS: con 4;
	NITEMS: con 4;
	TOTAL: con 16;  # NPRODUCERS * NITEMS
	# Buffer must be larger than total items to avoid blocking producers
	ch := chan[32] of int;
	done := chan[NPRODUCERS] of int;  # Buffered to avoid blocking on done

	# Spawn multiple producers
	for(i := 0; i < NPRODUCERS; i++)
		spawn producer(ch, done, i, NITEMS);

	# Wait for all producers to complete
	for(i = 0; i < NPRODUCERS; i++)
		<-done;

	# Count received items - we know exactly how many to expect
	count := 0;
	for(j := 0; j < TOTAL; j++) {
		<-ch;
		count++;
	}

	t.asserteq(count, TOTAL, sys->sprint("%d producers x %d items", NPRODUCERS, NITEMS));
}

producer(ch: chan of int, done: chan of int, id: int, n: int)
{
	for(i := 0; i < n; i++)
		ch <-= id * 100 + i;
	done <-= 1;
}

#
# Test 10: Pick type pattern (like AsyncMsg)
#
AsyncTestMsg: adt {
	pick {
		Data =>
			value: int;
		Error =>
			msg: string;
		Done =>
			dummy: int;  # pick variants need at least one field
	}
};

testPickTypeChannel(t: ref T)
{
	ch := chan[4] of ref AsyncTestMsg;

	# Send different pick variants
	ch <-= ref AsyncTestMsg.Data(42);
	ch <-= ref AsyncTestMsg.Error("test error");
	ch <-= ref AsyncTestMsg.Done;

	# Receive and dispatch
	dataCount := 0;
	errorCount := 0;
	doneCount := 0;

	for(i := 0; i < 3; i++) {
		msg := <-ch;
		pick m := msg {
			Data =>
				dataCount++;
				t.asserteq(m.value, 42, "data value");
			Error =>
				errorCount++;
				t.assertseq(m.msg, "test error", "error message");
			Done =>
				doneCount++;
		}
	}

	t.asserteq(dataCount, 1, "data message count");
	t.asserteq(errorCount, 1, "error message count");
	t.asserteq(doneCount, 1, "done message count");
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
	run("BufferedChannelNonBlocking", testBufferedChannelNonBlocking);
	run("UnbufferedChannelBlocks", testUnbufferedChannelBlocks);
	run("SingleBufferChannel", testSingleBufferChannel);
	run("SpawnedTaskBufferedSend", testSpawnedTaskBufferedSend);
	run("AltMultipleBuffered", testAltMultipleBuffered);
	run("NonBlockingSendPattern", testNonBlockingSendPattern);
	run("ReceiveWithTimeout", testReceiveWithTimeout);
	run("LargeBuffer", testLargeBuffer);
	run("ConcurrentProducers", testConcurrentProducers);
	run("PickTypeChannel", testPickTypeChannel);

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
