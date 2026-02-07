implement Sdl3Test;

#
# SDL3 Draw module tests
# Migrated from appl/test-sdl3.b
#
# Tests:
# - Draw module loads
# - Display.allocate (requires display - skipped if unavailable)
# - Basic drawing operations (requires display - skipped if unavailable)
#
# Note: Many tests require a graphical display.
# Tests will be skipped if no display is available.
#

include "sys.m";
	sys: Sys;

include "draw.m";
	draw: Draw;
	Display, Image, Rect, Point: import draw;

include "testing.m";
	testing: Testing;
	T: import testing;

Sdl3Test: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

passed := 0;
failed := 0;
skipped := 0;

# Source file path for clickable error addresses
SRCFILE: con "/tests/sdl3_test.b";

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

# Test Draw module loads
testDrawModuleLoads(t: ref T)
{
	# draw is already loaded at init time
	t.assert(draw != nil, "Draw module should be loaded");
}

# Test Display.allocate (requires display)
testDisplayAllocate(t: ref T)
{
	display := Display.allocate(nil);
	if(display == nil) {
		t.skip("no display available (headless mode)");
		return;
	}
	t.log("Display.allocate succeeded");

	# Check display has expected fields
	t.assert(display.image != nil, "display should have image");
}

# Test color allocation (requires display)
testColorAllocation(t: ref T)
{
	display := Display.allocate(nil);
	if(display == nil) {
		t.skip("no display available");
		return;
	}

	red := display.color(Draw->Red);
	t.assert(red != nil, "color allocation should succeed");
	t.log("allocated red color");

	green := display.color(Draw->Green);
	t.assert(green != nil, "green color allocation should succeed");

	blue := display.color(Draw->Blue);
	t.assert(blue != nil, "blue color allocation should succeed");
}

# Test Rect construction
testRectConstruction(t: ref T)
{
	r := Rect((0, 0), (100, 100));
	t.asserteq(r.min.x, 0, "rect min.x");
	t.asserteq(r.min.y, 0, "rect min.y");
	t.asserteq(r.max.x, 100, "rect max.x");
	t.asserteq(r.max.y, 100, "rect max.y");
}

# Test Point construction
testPointConstruction(t: ref T)
{
	p := Point(50, 75);
	t.asserteq(p.x, 50, "point x");
	t.asserteq(p.y, 75, "point y");
}

# Test drawing (requires display) - brief test
testDrawing(t: ref T)
{
	display := Display.allocate(nil);
	if(display == nil) {
		t.skip("no display available");
		return;
	}

	# Draw a small rectangle - no error means success
	red := display.color(Draw->Red);
	display.image.draw(Rect((10, 10), (20, 20)), red, nil, (0, 0));
	t.log("drawing succeeded");
}

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	draw = load Draw Draw->PATH;
	testing = load Testing Testing->PATH;

	if(testing == nil) {
		sys->fprint(sys->fildes(2), "cannot load testing module: %r\n");
		raise "fail:cannot load testing";
	}

	if(draw == nil) {
		sys->fprint(sys->fildes(2), "cannot load draw module: %r\n");
		raise "fail:cannot load draw";
	}

	testing->init();

	# Check for verbose flag
	for(a := args; a != nil; a = tl a) {
		if(hd a == "-v")
			testing->verbose(1);
	}

	# Run tests
	run("DrawModuleLoads", testDrawModuleLoads);
	run("RectConstruction", testRectConstruction);
	run("PointConstruction", testPointConstruction);
	run("DisplayAllocate", testDisplayAllocate);
	run("ColorAllocation", testColorAllocation);
	run("Drawing", testDrawing);

	# Print summary
	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
