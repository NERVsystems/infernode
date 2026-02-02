implement VeltroToolsTest;

#
# Tests for new Veltro tools (Phase 1c)
#
# Tests: diff, json, memory tools
# Skips: http (requires network), git (requires git), ask (requires console)
#
# To run: cd $ROOT && ./emu/MacOSX/Infernode -r . /tests/veltro_tools_test.dis
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

# Tool interface (same as /appl/veltro/tool.m)
Tool: module {
	init: fn(): string;
	name: fn(): string;
	doc:  fn(): string;
	exec: fn(args: string): string;
};

VeltroToolsTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

SRCFILE: con "/tests/veltro_tools_test.b";

passed := 0;
failed := 0;
skipped := 0;

run(testname: string, testfn: ref fn(t: ref T))
{
	t := testing->newTsrc(testname, SRCFILE);
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

# Load a tool module
loadtool(name: string): Tool
{
	path := "/dis/veltro/tools/" + name + ".dis";
	mod := load Tool path;
	if(mod != nil)
		mod->init();
	return mod;
}

# Test diff tool - basic functionality
testDiffBasic(t: ref T)
{
	# Create two test files
	file1 := "/tmp/test_diff_a.txt";
	file2 := "/tmp/test_diff_b.txt";

	# Write file1
	fd := sys->create(file1, Sys->OWRITE, 8r644);
	if(fd == nil) {
		t.fatal("cannot create test file 1");
		return;
	}
	sys->fprint(fd, "line1\nline2\nline3\n");
	fd = nil;

	# Write file2 (modified)
	fd = sys->create(file2, Sys->OWRITE, 8r644);
	if(fd == nil) {
		t.fatal("cannot create test file 2");
		return;
	}
	sys->fprint(fd, "line1\nline2modified\nline3\n");
	fd = nil;

	# Load and test diff tool
	diff := loadtool("diff");
	if(diff == nil) {
		t.skip("diff tool not available");
		return;
	}

	# Run diff
	result := diff->exec(file1 + " " + file2);
	t.log("diff result: " + result);

	# Should contain diff output (not error)
	if(len result > 5 && result[0:5] == "error") {
		t.error("diff returned error: " + result);
	}

	# Should contain +/- markers for changes
	t.assert(hassubstr(result, "line2modified") || hassubstr(result, "+line2") || hassubstr(result, "-line2"),
		"diff should show change in line2");

	# Cleanup
	sys->remove(file1);
	sys->remove(file2);
}

# Test diff tool - identical files
testDiffIdentical(t: ref T)
{
	file1 := "/tmp/test_diff_same1.txt";
	file2 := "/tmp/test_diff_same2.txt";

	# Create identical files
	content := "same\ncontent\nhere\n";
	fd := sys->create(file1, Sys->OWRITE, 8r644);
	if(fd != nil) {
		sys->fprint(fd, "%s", content);
		fd = nil;
	}
	fd = sys->create(file2, Sys->OWRITE, 8r644);
	if(fd != nil) {
		sys->fprint(fd, "%s", content);
		fd = nil;
	}

	diff := loadtool("diff");
	if(diff == nil) {
		t.skip("diff tool not available");
		return;
	}

	result := diff->exec(file1 + " " + file2);
	t.log("identical files result: " + result);

	# Should indicate files are identical
	t.assert(hassubstr(result, "identical"), "diff of identical files should say 'identical'");

	sys->remove(file1);
	sys->remove(file2);
}

# Test JSON tool - parse and query
testJsonParse(t: ref T)
{
	json := loadtool("json");
	if(json == nil) {
		t.skip("json tool not available");
		return;
	}

	# Create test JSON file
	jsonfile := "/tmp/test.json";
	fd := sys->create(jsonfile, Sys->OWRITE, 8r644);
	if(fd == nil) {
		t.fatal("cannot create test JSON file");
		return;
	}
	sys->fprint(fd, `{"name": "test", "value": 42, "nested": {"key": "val"}}`);
	fd = nil;

	# Query .name
	result := json->exec(jsonfile + " .name");
	t.log(".name result: " + result);
	if(len result > 5 && result[0:5] == "error") {
		t.error("json query failed: " + result);
	} else {
		t.assert(hassubstr(result, "test"), "should find 'test' in name query");
	}

	# Query .value
	result = json->exec(jsonfile + " .value");
	t.log(".value result: " + result);
	t.assert(hassubstr(result, "42"), "should find 42 in value query");

	# Query .nested.key
	result = json->exec(jsonfile + " .nested.key");
	t.log(".nested.key result: " + result);
	t.assert(hassubstr(result, "val"), "should find 'val' in nested key query");

	sys->remove(jsonfile);
}

# Test JSON tool - array access
testJsonArray(t: ref T)
{
	json := loadtool("json");
	if(json == nil) {
		t.skip("json tool not available");
		return;
	}

	jsonfile := "/tmp/test_array.json";
	fd := sys->create(jsonfile, Sys->OWRITE, 8r644);
	if(fd == nil) {
		t.fatal("cannot create test JSON file");
		return;
	}
	sys->fprint(fd, `{"items": ["first", "second", "third"]}`);
	fd = nil;

	# Query .items[0]
	result := json->exec(jsonfile + " .items[0]");
	t.log(".items[0] result: " + result);
	t.assert(hassubstr(result, "first"), "should find 'first' at index 0");

	# Query .items[1]
	result = json->exec(jsonfile + " .items[1]");
	t.log(".items[1] result: " + result);
	t.assert(hassubstr(result, "second"), "should find 'second' at index 1");

	sys->remove(jsonfile);
}

# Test memory tool - save and load
testMemorySaveLoad(t: ref T)
{
	mem := loadtool("memory");
	if(mem == nil) {
		t.skip("memory tool not available");
		return;
	}

	# Save a value
	result := mem->exec("save testkey testvalue123");
	t.log("save result: " + result);
	if(len result > 5 && result[0:5] == "error") {
		t.error("memory save failed: " + result);
		return;
	}

	# Load the value back
	result = mem->exec("load testkey");
	t.log("load result: " + result);
	t.assertseq(result, "testvalue123", "loaded value should match saved value");

	# Clean up
	mem->exec("delete testkey");
}

# Test memory tool - list keys
testMemoryList(t: ref T)
{
	mem := loadtool("memory");
	if(mem == nil) {
		t.skip("memory tool not available");
		return;
	}

	# Save a few values
	mem->exec("save listkey1 value1");
	mem->exec("save listkey2 value2");

	# List keys
	result := mem->exec("list");
	t.log("list result: " + result);

	t.assert(hassubstr(result, "listkey1"), "list should contain listkey1");
	t.assert(hassubstr(result, "listkey2"), "list should contain listkey2");

	# Clean up
	mem->exec("delete listkey1");
	mem->exec("delete listkey2");
}

# Test memory tool - append
testMemoryAppend(t: ref T)
{
	mem := loadtool("memory");
	if(mem == nil) {
		t.skip("memory tool not available");
		return;
	}

	# Save initial value
	mem->exec("save appendkey hello");

	# Append to it
	result := mem->exec("append appendkey world");
	t.log("append result: " + result);

	# Load and verify
	result = mem->exec("load appendkey");
	t.assertseq(result, "helloworld", "appended value should be concatenated");

	# Clean up
	mem->exec("delete appendkey");
}

# Test memory tool - delete
testMemoryDelete(t: ref T)
{
	mem := loadtool("memory");
	if(mem == nil) {
		t.skip("memory tool not available");
		return;
	}

	# Save and verify
	mem->exec("save delkey todelete");
	result := mem->exec("load delkey");
	t.assertseq(result, "todelete", "saved value should exist");

	# Delete
	result = mem->exec("delete delkey");
	t.log("delete result: " + result);

	# Verify deleted
	result = mem->exec("load delkey");
	t.assert(hassubstr(result, "error") || hassubstr(result, "not found"),
		"deleted key should not be found");
}

# Helper: check if s contains substr
hassubstr(s, sub: string): int
{
	if(len sub > len s)
		return 0;
	for(i := 0; i <= len s - len sub; i++) {
		if(s[i:i+len sub] == sub)
			return 1;
	}
	return 0;
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

	for(a := args; a != nil; a = tl a) {
		if(hd a == "-v")
			testing->verbose(1);
	}

	# Diff tests
	run("DiffBasic", testDiffBasic);
	run("DiffIdentical", testDiffIdentical);

	# JSON tests
	run("JsonParse", testJsonParse);
	run("JsonArray", testJsonArray);

	# Memory tests
	run("MemorySaveLoad", testMemorySaveLoad);
	run("MemoryList", testMemoryList);
	run("MemoryAppend", testMemoryAppend);
	run("MemoryDelete", testMemoryDelete);

	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
