implement SpawnTest;

#
# spawn_test - Tests for the spawn result passing mechanism
#
# Tests:
# - Result directory structure
# - File writing/reading helpers
# - Status tracking
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

SpawnTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

passed := 0;
failed := 0;
skipped := 0;

# Test base directory
TEST_BASE := "/tmp/spawn_test";

# Source file path for clickable error addresses
SRCFILE: con "/tests/spawn_test.b";

# Helper to run a test and track results
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

# ============================================================
# Helper functions (from spawn.b)
# ============================================================

# Write content to a file
writefile(path, content: string): int
{
	fd := sys->create(path, Sys->OWRITE, 8r644);
	if(fd == nil)
		return -1;
	data := array of byte content;
	n := sys->write(fd, data, len data);
	return n;
}

# Read file content
readfile(path: string): string
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return "";
	buf := array[8192] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return "";
	return string buf[0:n];
}

# Ensure directory exists
ensuredir(path: string)
{
	fd := sys->open(path, Sys->OREAD);
	if(fd != nil)
		return;
	fd = sys->create(path, Sys->OREAD, Sys->DMDIR | 8r755);
}

# Cleanup test directory
cleanup()
{
	# Remove test files
	sys->remove(TEST_BASE + "/status");
	sys->remove(TEST_BASE + "/output");
	sys->remove(TEST_BASE + "/error");
	sys->remove(TEST_BASE);
}

# ============================================================
# Tests
# ============================================================

# Test ensuredir creates directory
testEnsuredir(t: ref T)
{
	testdir := TEST_BASE + "/subdir";

	# Remove if exists
	sys->remove(testdir);

	# Create directory
	ensuredir(testdir);

	# Verify it exists
	fd := sys->open(testdir, Sys->OREAD);
	t.assert(fd != nil, "ensuredir should create directory");

	# Cleanup
	sys->remove(testdir);
}

# Test ensuredir is idempotent
testEnsuredirIdempotent(t: ref T)
{
	testdir := TEST_BASE + "/idem";

	# Create twice
	ensuredir(testdir);
	ensuredir(testdir);  # Should not fail

	# Verify still exists
	fd := sys->open(testdir, Sys->OREAD);
	t.assert(fd != nil, "ensuredir should be idempotent");

	# Cleanup
	sys->remove(testdir);
}

# Test writefile creates file with content
testWritefile(t: ref T)
{
	testfile := TEST_BASE + "/write_test";

	n := writefile(testfile, "hello world");
	t.assert(n > 0, "writefile should return bytes written");
	t.asserteq(n, 11, "writefile should write 11 bytes");

	# Verify content
	content := readfile(testfile);
	t.assertseq(content, "hello world", "content should match");

	# Cleanup
	sys->remove(testfile);
}

# Test writefile with empty content
testWritefileEmpty(t: ref T)
{
	testfile := TEST_BASE + "/empty_test";

	n := writefile(testfile, "");
	t.asserteq(n, 0, "writefile should write 0 bytes for empty");

	# Cleanup
	sys->remove(testfile);
}

# Test writefile to bad path
testWritefileBadPath(t: ref T)
{
	n := writefile("/nonexistent/path/file", "test");
	t.asserteq(n, -1, "writefile to bad path should return -1");
}

# Test readfile returns content
testReadfile(t: ref T)
{
	testfile := TEST_BASE + "/read_test";

	# Write some content
	writefile(testfile, "test content");

	# Read it back
	content := readfile(testfile);
	t.assertseq(content, "test content", "readfile should return content");

	# Cleanup
	sys->remove(testfile);
}

# Test readfile nonexistent file
testReadfileNonexistent(t: ref T)
{
	content := readfile("/nonexistent/file");
	t.assertseq(content, "", "readfile of nonexistent file should return empty");
}

# Test result directory structure
testResultDirectory(t: ref T)
{
	resultdir := TEST_BASE + "/result";

	# Create result directory structure
	ensuredir(resultdir);
	writefile(resultdir + "/status", "running");
	writefile(resultdir + "/output", "");
	writefile(resultdir + "/error", "");

	# Verify status
	status := readfile(resultdir + "/status");
	t.assertseq(status, "running", "initial status should be 'running'");

	# Update status to completed
	writefile(resultdir + "/status", "completed");
	status = readfile(resultdir + "/status");
	t.assertseq(status, "completed", "status should update to 'completed'");

	# Write output
	writefile(resultdir + "/output", "task result here");
	output := readfile(resultdir + "/output");
	t.assertseq(output, "task result here", "output should be written");

	# Cleanup
	sys->remove(resultdir + "/status");
	sys->remove(resultdir + "/output");
	sys->remove(resultdir + "/error");
	sys->remove(resultdir);
}

# Test error status flow
testErrorStatus(t: ref T)
{
	resultdir := TEST_BASE + "/error_test";

	# Create result directory
	ensuredir(resultdir);
	writefile(resultdir + "/status", "running");
	writefile(resultdir + "/error", "");

	# Simulate error
	writefile(resultdir + "/status", "error");
	writefile(resultdir + "/error", "something went wrong");

	# Verify
	status := readfile(resultdir + "/status");
	t.assertseq(status, "error", "status should be 'error'");

	errmsg := readfile(resultdir + "/error");
	t.assertseq(errmsg, "something went wrong", "error message should be set");

	# Cleanup
	sys->remove(resultdir + "/status");
	sys->remove(resultdir + "/output");
	sys->remove(resultdir + "/error");
	sys->remove(resultdir);
}

# Test timeout status
testTimeoutStatus(t: ref T)
{
	resultdir := TEST_BASE + "/timeout_test";

	ensuredir(resultdir);
	writefile(resultdir + "/status", "timeout");

	status := readfile(resultdir + "/status");
	t.assertseq(status, "timeout", "timeout status should be valid");

	# Cleanup
	sys->remove(resultdir + "/status");
	sys->remove(resultdir);
}

# Test valid status values
testValidStatusValues(t: ref T)
{
	validStatuses := array[] of {"running", "completed", "error", "timeout"};
	resultdir := TEST_BASE + "/valid_status";

	ensuredir(resultdir);

	for(i := 0; i < len validStatuses; i++) {
		status := validStatuses[i];
		writefile(resultdir + "/status", status);
		read := readfile(resultdir + "/status");
		t.assertseq(read, status, "status '" + status + "' should be valid");
	}

	# Cleanup
	sys->remove(resultdir + "/status");
	sys->remove(resultdir);
}

# Test large output handling
testLargeOutput(t: ref T)
{
	resultdir := TEST_BASE + "/large_output";

	ensuredir(resultdir);

	# Create large output (1KB)
	large := "";
	for(i := 0; i < 100; i++)
		large += "0123456789";  # 1000 chars

	writefile(resultdir + "/output", large);
	output := readfile(resultdir + "/output");

	t.asserteq(len output, 1000, "large output should be preserved");

	# Cleanup
	sys->remove(resultdir + "/output");
	sys->remove(resultdir);
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

	# Setup test directory
	ensuredir(TEST_BASE);

	# Run tests
	run("Ensuredir", testEnsuredir);
	run("EnsuredirIdempotent", testEnsuredirIdempotent);
	run("Writefile", testWritefile);
	run("WritefileEmpty", testWritefileEmpty);
	run("WritefileBadPath", testWritefileBadPath);
	run("Readfile", testReadfile);
	run("ReadfileNonexistent", testReadfileNonexistent);
	run("ResultDirectory", testResultDirectory);
	run("ErrorStatus", testErrorStatus);
	run("TimeoutStatus", testTimeoutStatus);
	run("ValidStatusValues", testValidStatusValues);
	run("LargeOutput", testLargeOutput);

	# Cleanup
	cleanup();

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
