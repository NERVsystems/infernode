implement VeltroConcurrentTest;

#
# Veltro Concurrent Spawn Test
#
# Tests that multiple concurrent spawn operations don't crash.
# This specifically tests the race condition fix in nsconstruct.b
# and spawn.b initialization.
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

VeltroConcurrentTest: module {
	init: fn(nil: ref Draw->Context, args: list of string);
};

include "nsconstruct.m";
	nsconstruct: NsConstruct;

SRCFILE: con "/tests/veltro_concurrent_test.b";

passed := 0;
failed := 0;
skipped := 0;

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
# Test 1: Concurrent nsconstruct init
# Spawns multiple threads that all call init() simultaneously
# ============================================================================
testConcurrentInit(t: ref T)
{
	done := chan of int;
	errors := chan of string;
	nthreads := 10;

	# Spawn threads that all call init
	for(i := 0; i < nthreads; i++)
		spawn initworker(done, errors);

	# Collect results
	errs: list of string;
	for(i = 0; i < nthreads; i++) {
		alt {
		e := <-errors =>
			errs = e :: errs;
		<-done =>
			;
		}
	}

	t.assert(errs == nil, "all init calls should succeed");
	for(; errs != nil; errs = tl errs)
		t.log(hd errs);
}

initworker(done: chan of int, errors: chan of string)
{
	# Small random delay to increase chance of race
	sys->sleep(sys->millisec() % 10);

	nsconstruct->init();
	done <-= 1;
}

# ============================================================================
# Test 2: Concurrent sandbox creation
# Creates multiple sandboxes concurrently to test preparesandbox
# ============================================================================
testConcurrentSandboxCreate(t: ref T)
{
	done := chan of int;
	errors := chan of string;
	nsandboxes := 5;
	sandboxids: list of string;

	# Create sandboxes concurrently
	for(j := 0; j < nsandboxes; j++) {
		sandboxid := nsconstruct->gensandboxid();
		sandboxids = sandboxid :: sandboxids;
		spawn sandboxworker(sandboxid, done, errors);
		# Small delay to get different IDs
		sys->sleep(15);
	}

	# Collect results
	succeeded := 0;
	errs: list of string;
	for(k := 0; k < nsandboxes; k++) {
		alt {
		e := <-errors =>
			errs = e :: errs;
		<-done =>
			succeeded++;
		}
	}

	t.assert(succeeded == nsandboxes,
		sys->sprint("all %d sandboxes should be created (got %d)", nsandboxes, succeeded));

	for(; errs != nil; errs = tl errs)
		t.log(hd errs);

	# Cleanup all sandboxes
	for(ids := sandboxids; ids != nil; ids = tl ids)
		nsconstruct->cleanupsandbox(hd ids);
}

sandboxworker(sandboxid: string, done: chan of int, errors: chan of string)
{
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
	if(err != nil)
		errors <-= sys->sprint("sandbox %s failed: %s", sandboxid, err);
	else
		done <-= 1;
}

# ============================================================================
# Test 3: Concurrent sandbox cleanup
# Tests that cleanup doesn't race with other operations
# ============================================================================
testConcurrentCleanup(t: ref T)
{
	# Create a sandbox
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
		t.error(sys->sprint("preparesandbox failed: %s", err));
		return;
	}

	# Cleanup from multiple threads (only one should actually do work)
	done := chan of int;
	nthreads := 5;

	for(m := 0; m < nthreads; m++)
		spawn cleanupworker(sandboxid, done);

	# Wait for all to complete
	for(n := 0; n < nthreads; n++)
		<-done;

	# Verify sandbox is gone
	sandboxdir := nsconstruct->sandboxpath(sandboxid);
	(ok, nil) := sys->stat(sandboxdir);
	t.assert(ok < 0, "sandbox should be removed after cleanup");
}

cleanupworker(sandboxid: string, done: chan of int)
{
	nsconstruct->cleanupsandbox(sandboxid);
	done <-= 1;
}

# ============================================================================
# Main
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

	for(a := args; a != nil; a = tl a) {
		if(hd a == "-v")
			testing->verbose(1);
	}

	run("ConcurrentInit", testConcurrentInit);
	run("ConcurrentSandboxCreate", testConcurrentSandboxCreate);
	run("ConcurrentCleanup", testConcurrentCleanup);

	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
