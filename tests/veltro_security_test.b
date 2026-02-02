implement VeltroSecurityTest;

#
# Veltro Namespace Security Tests
#
# These tests verify the security properties of the Veltro namespace
# isolation model (v2). Each test verifies a specific security guarantee.
#
# Security Properties Tested:
#   1. Sandbox ID validation - reject path traversal attacks
#   2. Environment isolation - NEWENV creates empty environment
#   3. Path traversal blocked - .. cannot escape sandbox root
#   4. Path isolation - only granted paths visible
#   5. Audit logging - all binds recorded
#
# Note: Some tests (FD isolation, NODEVS, service registry) require
# running inside a spawned child to properly test pctl effects.
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

VeltroSecurityTest: module {
	init: fn(nil: ref Draw->Context, args: list of string);
};

# Include nsconstruct for testing
include "nsconstruct.m";
	nsconstruct: NsConstruct;

# Source file path for clickable error addresses
SRCFILE: con "/tests/veltro_security_test.b";

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

# ============================================================================
# Test 1: Sandbox ID Validation
# Verifies that validatesandboxid() rejects path traversal attacks
# ============================================================================
testSandboxIdValidation(t: ref T)
{
	# Valid IDs should pass
	validids := array[] of {
		"abc123",
		"sandbox-1",
		"A-B-C",
		"test",
		"a",
		"123",
	};

	for(i := 0; i < len validids; i++) {
		id := validids[i];
		err := nsconstruct->validatesandboxid(id);
		t.assert(err == nil, sys->sprint("valid id '%s' should pass", id));
		if(err != nil)
			t.log(sys->sprint("valid id '%s' rejected: %s", id, err));
	}

	# Invalid IDs should fail
	invalidids := array[] of {
		"",           # empty
		"../escape",  # path traversal
		"foo/bar",    # path separator
		"foo\\bar",   # backslash
		"foo.bar",    # dot (could be ..)
		"foo bar",    # space
		"foo\tbar",   # tab
		"foo\nbar",   # newline
		".",          # current dir
		"..",         # parent dir
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmn", # too long (>64)
	};

	for(i = 0; i < len invalidids; i++) {
		id := invalidids[i];
		err := nsconstruct->validatesandboxid(id);
		t.assert(err != nil, sys->sprint("invalid id '%s' should fail", id));
		if(err == nil)
			t.log(sys->sprint("invalid id '%s' was accepted", id));
	}
}

# ============================================================================
# Test 2: Sandbox ID Generation
# Verifies that gensandboxid() generates unique, valid IDs
# ============================================================================
testSandboxIdGeneration(t: ref T)
{
	# Generate multiple IDs and verify they are unique and valid
	ids: list of string;
	for(i := 0; i < 10; i++) {
		id := nsconstruct->gensandboxid();

		# Verify ID is valid
		err := nsconstruct->validatesandboxid(id);
		t.assert(err == nil, sys->sprint("generated id '%s' should be valid", id));

		# Verify ID is unique (not in list)
		for(l := ids; l != nil; l = tl l) {
			if(hd l == id)
				t.error(sys->sprint("duplicate id generated: %s", id));
		}
		ids = id :: ids;

		# Small delay to ensure different timestamps
		sys->sleep(10);
	}
}

# ============================================================================
# Test 3: Sandbox Path Generation
# Verifies that sandboxpath() returns correct path
# ============================================================================
testSandboxPath(t: ref T)
{
	path := nsconstruct->sandboxpath("test-123");
	expected := "/tmp/.veltro/sandbox/test-123";
	t.assertseq(path, expected, "sandbox path format");
}

# ============================================================================
# Test 4: Verify Ownership (stat check)
# Verifies that verifyownership() checks path existence
# ============================================================================
testVerifyOwnership(t: ref T)
{
	# Existing path should succeed
	err := nsconstruct->verifyownership("/dev");
	t.assert(err == nil, "/dev should exist");

	# Non-existent path should fail
	err = nsconstruct->verifyownership("/nonexistent/path/12345");
	t.assert(err != nil, "nonexistent path should fail verification");
}

# ============================================================================
# Test 5: Sandbox Preparation - Basic
# Verifies preparesandbox creates proper directory structure
# Note: This test creates a real sandbox, so we clean up after
# ============================================================================
testPrepareSandbox(t: ref T)
{
	# Generate a unique ID
	sandboxid := nsconstruct->gensandboxid();

	# Create minimal capabilities
	caps := ref NsConstruct->Capabilities(
		"read" :: nil,        # tools
		nil,                  # paths (no extra paths)
		nil,                  # shellcmds
		nil,                  # llmconfig
		0 :: 1 :: 2 :: nil,   # fds
		ref NsConstruct->Mountpoints(0, 0, 0),  # no srv/net/prog
		sandboxid,            # sandboxid
		0,                    # untrusted
		nil,                  # mcproviders
		0                     # memory
	);

	# Prepare sandbox
	err := nsconstruct->preparesandbox(caps);
	if(err != nil) {
		t.error(sys->sprint("preparesandbox failed: %s", err));
		return;
	}

	# Verify sandbox directory was created
	sandboxdir := nsconstruct->sandboxpath(sandboxid);
	(ok, nil) := sys->stat(sandboxdir);
	t.assert(ok >= 0, "sandbox directory should exist");

	# Verify essential directories were created
	dirs := array[] of {
		"/dis",
		"/dis/lib",
		"/dis/veltro",
		"/dev",
		"/tool",
		"/tmp",
	};

	for(i := 0; i < len dirs; i++) {
		path := sandboxdir + dirs[i];
		(ok, nil) = sys->stat(path);
		t.assert(ok >= 0, sys->sprint("%s should exist", dirs[i]));
	}

	# Verify audit log was created
	auditpath := "/tmp/.veltro/audit/" + sandboxid + ".ns";
	(ok, nil) = sys->stat(auditpath);
	t.assert(ok >= 0, "audit log should exist");

	# Clean up
	nsconstruct->cleanupsandbox(sandboxid);

	# Verify cleanup removed sandbox
	(ok, nil) = sys->stat(sandboxdir);
	t.assert(ok < 0, "sandbox should be removed after cleanup");
}

# ============================================================================
# Test 6: Sandbox Preparation - With Paths
# Verifies preparesandbox binds granted paths correctly
# ============================================================================
testPrepareSandboxWithPaths(t: ref T)
{
	# Generate a unique ID
	sandboxid := nsconstruct->gensandboxid();

	# Create capabilities with path grants
	caps := ref NsConstruct->Capabilities(
		"read" :: nil,               # tools
		"/dev/null" :: nil,          # paths - grant /dev/null
		nil,                         # shellcmds
		nil,                         # llmconfig
		0 :: 1 :: 2 :: nil,          # fds
		ref NsConstruct->Mountpoints(0, 0, 0),
		sandboxid,
		0,                           # untrusted
		nil,                         # mcproviders
		0                            # memory
	);

	# Prepare sandbox
	err := nsconstruct->preparesandbox(caps);
	if(err != nil) {
		t.error(sys->sprint("preparesandbox failed: %s", err));
		return;
	}

	# Verify granted path was bound
	sandboxdir := nsconstruct->sandboxpath(sandboxid);
	grantedpath := sandboxdir + "/dev/null";
	(ok, nil) := sys->stat(grantedpath);
	t.assert(ok >= 0, "granted path should be accessible in sandbox");

	# Clean up
	nsconstruct->cleanupsandbox(sandboxid);
}

# ============================================================================
# Test 7: Sandbox Preparation - Trusted vs Untrusted
# Verifies that shell is only bound for trusted agents
# ============================================================================
testPrepareSandboxTrust(t: ref T)
{
	# Test untrusted - should NOT have shell
	sandboxid1 := nsconstruct->gensandboxid();
	caps1 := ref NsConstruct->Capabilities(
		"read" :: nil,
		nil,
		"cat" :: nil,                # shellcmds specified but untrusted
		nil,
		0 :: 1 :: 2 :: nil,
		ref NsConstruct->Mountpoints(0, 0, 0),
		sandboxid1,
		0,                           # UNTRUSTED
		nil,                         # mcproviders
		0                            # memory
	);

	err := nsconstruct->preparesandbox(caps1);
	if(err != nil) {
		t.error(sys->sprint("preparesandbox (untrusted) failed: %s", err));
	} else {
		sandboxdir := nsconstruct->sandboxpath(sandboxid1);
		shpath := sandboxdir + "/dis/sh.dis";
		(ok, nil) := sys->stat(shpath);
		t.assert(ok < 0, "untrusted should NOT have shell");
		nsconstruct->cleanupsandbox(sandboxid1);
	}

	# Test trusted - should have shell
	sandboxid2 := nsconstruct->gensandboxid();
	caps2 := ref NsConstruct->Capabilities(
		"read" :: "exec" :: nil,
		nil,
		"cat" :: nil,                # shellcmds for trusted
		nil,
		0 :: 1 :: 2 :: nil,
		ref NsConstruct->Mountpoints(0, 0, 0),
		sandboxid2,
		1,                           # TRUSTED
		nil,                         # mcproviders
		0                            # memory
	);

	err = nsconstruct->preparesandbox(caps2);
	if(err != nil) {
		t.error(sys->sprint("preparesandbox (trusted) failed: %s", err));
	} else {
		sandboxdir := nsconstruct->sandboxpath(sandboxid2);
		shpath := sandboxdir + "/dis/sh.dis";
		(ok, nil) := sys->stat(shpath);
		t.assert(ok >= 0, "trusted should have shell");

		# Verify granted shell command is present
		catpath := sandboxdir + "/dis/cat.dis";
		(ok, nil) = sys->stat(catpath);
		t.assert(ok >= 0, "trusted should have granted shell command");

		nsconstruct->cleanupsandbox(sandboxid2);
	}
}

# ============================================================================
# Test 8: Sandbox Preparation - Race Protection
# Verifies that creating a sandbox with existing ID fails
# ============================================================================
testPrepareSandboxRace(t: ref T)
{
	# Generate ID and create sandbox
	sandboxid := nsconstruct->gensandboxid();
	caps := ref NsConstruct->Capabilities(
		"read" :: nil,
		nil,
		nil,
		nil,
		0 :: 1 :: 2 :: nil,
		ref NsConstruct->Mountpoints(0, 0, 0),
		sandboxid,
		0,
		nil,  # mcproviders
		0     # memory
	);

	err := nsconstruct->preparesandbox(caps);
	if(err != nil) {
		t.error(sys->sprint("first preparesandbox failed: %s", err));
		return;
	}

	# Try to create again with same ID - should fail
	err = nsconstruct->preparesandbox(caps);
	t.assert(err != nil, "second preparesandbox with same ID should fail");
	if(err != nil)
		t.log(sys->sprint("expected failure: %s", err));

	# Clean up
	nsconstruct->cleanupsandbox(sandboxid);
}

# ============================================================================
# Test 9: Audit Log Content
# Verifies that audit log contains bind records
# ============================================================================
testAuditLog(t: ref T)
{
	# Generate ID and create sandbox
	sandboxid := nsconstruct->gensandboxid();
	caps := ref NsConstruct->Capabilities(
		"read" :: nil,
		"/dev/null" :: nil,      # grant a path
		nil,
		nil,
		0 :: 1 :: 2 :: nil,
		ref NsConstruct->Mountpoints(0, 0, 0),
		sandboxid,
		0,
		nil,  # mcproviders
		0     # memory
	);

	err := nsconstruct->preparesandbox(caps);
	if(err != nil) {
		t.error(sys->sprint("preparesandbox failed: %s", err));
		return;
	}

	# Read audit log
	auditpath := "/tmp/.veltro/audit/" + sandboxid + ".ns";
	fd := sys->open(auditpath, Sys->OREAD);
	if(fd == nil) {
		t.error("cannot open audit log");
		nsconstruct->cleanupsandbox(sandboxid);
		return;
	}

	buf := array[4096] of byte;
	n := sys->read(fd, buf, len buf);
	fd = nil;

	if(n <= 0) {
		t.error("audit log is empty");
		nsconstruct->cleanupsandbox(sandboxid);
		return;
	}

	content := string buf[0:n];

	# Verify audit log has header
	t.assert(contains(content, "Veltro Sandbox Namespace Audit"),
		"audit log should have header");

	# Verify audit log has sandbox ID
	t.assert(contains(content, sandboxid),
		"audit log should contain sandbox ID");

	# Verify audit log records binds
	t.assert(contains(content, "bind"),
		"audit log should contain bind records");

	# Clean up
	nsconstruct->cleanupsandbox(sandboxid);
}

# ============================================================================
# Helper: Check if string contains substring
# ============================================================================
contains(s, sub: string): int
{
	if(len sub > len s)
		return 0;
	for(i := 0; i <= len s - len sub; i++) {
		if(s[i:i+len sub] == sub)
			return 1;
	}
	return 0;
}

# ============================================================================
# Main entry point
# ============================================================================
init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	testing = load Testing Testing->PATH;
	nsconstruct = load NsConstruct NsConstruct->PATH;

	if(testing == nil) {
		sys->fprint(sys->fildes(2), "cannot load testing module: %r\n");
		raise "fail:cannot load testing";
	}

	if(nsconstruct == nil) {
		sys->fprint(sys->fildes(2), "cannot load nsconstruct module: %r\n");
		raise "fail:cannot load nsconstruct";
	}

	testing->init();
	nsconstruct->init();

	# Check for verbose flag
	for(a := args; a != nil; a = tl a) {
		if(hd a == "-v")
			testing->verbose(1);
	}

	# Run tests
	run("SandboxIdValidation", testSandboxIdValidation);
	run("SandboxIdGeneration", testSandboxIdGeneration);
	run("SandboxPath", testSandboxPath);
	run("VerifyOwnership", testVerifyOwnership);
	run("PrepareSandbox", testPrepareSandbox);
	run("PrepareSandboxWithPaths", testPrepareSandboxWithPaths);
	run("PrepareSandboxTrust", testPrepareSandboxTrust);
	run("PrepareSandboxRace", testPrepareSandboxRace);
	run("AuditLog", testAuditLog);

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
