implement Todo9pTest;

#
# todo9p_test - Tests for the todo9p task tracking server
#
# Tests the 9P file server interface for task management.
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

include "sh.m";
	sh: Sh;

Todo9pTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

passed := 0;
failed := 0;
skipped := 0;
ctxt: ref Draw->Context;
mountpoint := "/usr/inferno/todo_test";

# Source file path for clickable error addresses
SRCFILE: con "/tests/todo9p_test.b";

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

# Start todo9p and mount it
setup(): int
{
	# Run todo9p server with mount point argument (it mounts itself)
	err := sh->system(ctxt, "/dis/nerv/todo9p.dis " + mountpoint);
	if(err != nil && err != "") {
		sys->fprint(sys->fildes(2), "todo9p_test: failed to start todo9p: %s\n", err);
		return -1;
	}
	return 0;
}

# Unmount and cleanup
teardown()
{
	sh->system(ctxt, "unmount " + mountpoint);
}

# Helper to read a file
readfile(path: string): string
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return nil;
	buf := array[8192] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return "";
	return string buf[0:n];
}

# Helper to write to a file
writefile(path, data: string): int
{
	fd := sys->open(path, Sys->OWRITE);
	if(fd == nil)
		return -1;
	b := array of byte data;
	n := sys->write(fd, b, len b);
	return n;
}

# Test basic filesystem structure exists
testStructure(t: ref T)
{
	# Check that required files exist
	(ok, nil) := sys->stat(mountpoint + "/new");
	t.assert(ok >= 0, "new file should exist");

	(ok, nil) = sys->stat(mountpoint + "/list");
	t.assert(ok >= 0, "list file should exist");
}

# Test creating a new todo
testCreateTodo(t: ref T)
{
	# Write to new file to create a todo
	fd := sys->open(mountpoint + "/new", Sys->OWRITE);
	t.assert(fd != nil, "should be able to open new file for writing");
	if(fd == nil)
		return;

	content := "Test todo item";
	b := array of byte content;
	n := sys->write(fd, b, len b);
	t.assert(n > 0, "should write to new file");
	fd = nil;

	# Check that the todo appears in list
	listing := readfile(mountpoint + "/list");
	t.assert(listing != nil, "should be able to read list");
	t.assert(hassubstr(listing, "Test todo item"), "list should contain our todo");
}

# Test reading todo content
testReadTodo(t: ref T)
{
	# First create a todo
	writefile(mountpoint + "/new", "Read test item");

	# Find the todo ID from list (use last ID to get most recent)
	listing := readfile(mountpoint + "/list");
	id := extractlastid(listing);
	t.assert(id != "", "should extract todo ID from list");
	if(id == "")
		return;

	# Read the content file
	content := readfile(mountpoint + "/" + id + "/content");
	t.assert(hassubstr(content, "Read test item"),
		"content should match what we wrote");
}

# Test updating todo status
testUpdateStatus(t: ref T)
{
	# Create a todo
	writefile(mountpoint + "/new", "Status test item");

	# Find the todo ID
	listing := readfile(mountpoint + "/list");
	id := extractfirstid(listing);
	t.assert(id != "", "should have a todo ID");
	if(id == "")
		return;

	# Initial status should be pending
	status := readfile(mountpoint + "/" + id + "/status");
	t.assertseq(trim(status), "pending", "initial status should be pending");

	# Update to in_progress
	n := writefile(mountpoint + "/" + id + "/status", "in_progress");
	t.assert(n > 0, "should write to status file");

	# Verify change
	status = readfile(mountpoint + "/" + id + "/status");
	t.assertseq(trim(status), "in_progress", "status should be in_progress");

	# Update to completed
	writefile(mountpoint + "/" + id + "/status", "completed");
	status = readfile(mountpoint + "/" + id + "/status");
	t.assertseq(trim(status), "completed", "status should be completed");
}

# Test invalid status rejection
testInvalidStatus(t: ref T)
{
	# Create a todo
	writefile(mountpoint + "/new", "Invalid status test");

	listing := readfile(mountpoint + "/list");
	id := extractfirstid(listing);
	if(id == "") {
		t.skip("no todo ID available");
		return;
	}

	# Try to write invalid status
	fd := sys->open(mountpoint + "/" + id + "/status", Sys->OWRITE);
	if(fd == nil) {
		t.skip("cannot open status file");
		return;
	}

	b := array of byte "invalid_status";
	n := sys->write(fd, b, len b);
	# The write should fail or status should remain unchanged
	fd = nil;

	# Verify status is still valid
	status := readfile(mountpoint + "/" + id + "/status");
	valid := trim(status) == "pending" || trim(status) == "in_progress" || trim(status) == "completed";
	t.assert(valid, "status should remain valid after invalid write");
}

# Test deleting a todo
testDeleteTodo(t: ref T)
{
	# Create a todo
	writefile(mountpoint + "/new", "Delete test item");

	listing := readfile(mountpoint + "/list");
	id := extractfirstid(listing);
	if(id == "") {
		t.skip("no todo ID available");
		return;
	}

	# Delete via ctl
	n := writefile(mountpoint + "/" + id + "/ctl", "delete");
	t.assert(n > 0, "should write delete to ctl");

	# Verify todo no longer accessible
	(ok, nil) := sys->stat(mountpoint + "/" + id + "/content");
	t.assert(ok < 0, "deleted todo should not be accessible");
}

# Helper: check if string contains substring
hassubstr(s, sub: string): int
{
	if(len sub > len s)
		return 0;
	for(i := 0; i <= len s - len sub; i++) {
		match := 1;
		for(j := 0; j < len sub; j++) {
			if(s[i+j] != sub[j]) {
				match = 0;
				break;
			}
		}
		if(match)
			return 1;
	}
	return 0;
}

# Helper: extract first ID from list output
# List format: "id\tstatus\tcontent\n"
extractfirstid(listing: string): string
{
	if(listing == nil || listing == "")
		return "";
	# Find first tab or newline
	for(i := 0; i < len listing; i++) {
		if(listing[i] == '\t' || listing[i] == '\n')
			return listing[0:i];
	}
	return listing;
}

# Helper: extract last ID from list output (most recently created)
# List format: "id\tstatus\tcontent\n..."
extractlastid(listing: string): string
{
	if(listing == nil || listing == "")
		return "";
	lastid := "";
	i := 0;
	while(i < len listing) {
		# Skip to next line start or beginning of entry
		start := i;
		# Find the ID (ends at tab)
		for(; i < len listing && listing[i] != '\t' && listing[i] != '\n'; i++)
			;
		if(i > start)
			lastid = listing[start:i];
		# Skip to next line
		for(; i < len listing && listing[i] != '\n'; i++)
			;
		if(i < len listing)
			i++;  # skip the newline
	}
	return lastid;
}

# Helper: trim whitespace
trim(s: string): string
{
	if(s == nil)
		return "";
	start := 0;
	for(; start < len s; start++) {
		c := s[start];
		if(c != ' ' && c != '\t' && c != '\n' && c != '\r')
			break;
	}
	end := len s;
	for(; end > start; end--) {
		c := s[end-1];
		if(c != ' ' && c != '\t' && c != '\n' && c != '\r')
			break;
	}
	if(start >= end)
		return "";
	return s[start:end];
}

init(drawctxt: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	testing = load Testing Testing->PATH;
	sh = load Sh Sh->PATH;
	ctxt = drawctxt;

	if(testing == nil) {
		sys->fprint(sys->fildes(2), "cannot load testing module: %r\n");
		raise "fail:cannot load testing";
	}
	if(sh == nil) {
		sys->fprint(sys->fildes(2), "cannot load sh module: %r\n");
		raise "fail:cannot load sh";
	}

	testing->init();

	# Check for verbose flag
	for(a := args; a != nil; a = tl a) {
		if(hd a == "-v")
			testing->verbose(1);
	}

	# Setup: mount todo9p
	if(setup() < 0) {
		sys->fprint(sys->fildes(2), "skipping todo9p tests: cannot mount server\n");
		raise "fail:skip";
	}

	# Run tests
	run("Structure", testStructure);
	run("CreateTodo", testCreateTodo);
	run("ReadTodo", testReadTodo);
	run("UpdateStatus", testUpdateStatus);
	run("InvalidStatus", testInvalidStatus);
	run("DeleteTodo", testDeleteTodo);

	# Cleanup
	teardown();

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
