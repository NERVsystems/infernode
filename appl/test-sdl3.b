implement TestSDL3;

include "sys.m";
	sys: Sys;
include "draw.m";
	draw: Draw;
	Display, Screen, Image, Rect, Point: import draw;

TestSDL3: module {
	init: fn(ctxt: ref Draw->Context, argv: list of string);
};

init(ctxt: ref Draw->Context, nil: list of string)
{
	sys = load Sys Sys->PATH;
	draw = load Draw Draw->PATH;

	sys->fprint(sys->fildes(2), "test-sdl3: Starting...\n");

	# Create our own Display if no context provided
	display: ref Display;

	if (ctxt == nil) {
		sys->fprint(sys->fildes(2), "test-sdl3: No context provided, creating our own Display...\n");
		display = Display.allocate(nil);
		if (display == nil) {
			sys->fprint(sys->fildes(2), "test-sdl3: ERROR - Display.allocate failed!\n");
			return;
		}
		sys->fprint(sys->fildes(2), "test-sdl3: Display.allocate succeeded!\n");
	} else {
		display = ctxt.display;
		if (display == nil) {
			sys->fprint(sys->fildes(2), "test-sdl3: ERROR - context.display is nil!\n");
			return;
		}
		sys->fprint(sys->fildes(2), "test-sdl3: Using context.display\n");
	}

	# Draw directly to display.image
	sys->fprint(sys->fildes(2), "test-sdl3: Drawing red rectangle...\n");
	red := display.color(Draw->Red);
	display.image.draw(Rect((200, 200), (400, 400)), red, nil, (0,0));

	sys->fprint(sys->fildes(2), "test-sdl3: Drawing green rectangle...\n");
	green := display.color(Draw->Green);
	display.image.draw(Rect((500, 200), (700, 400)), green, nil, (0,0));

	sys->fprint(sys->fildes(2), "test-sdl3: Drawing done, flushing display...\n");

	# Flush to make drawing visible
	display.image.flush(Draw->Flushnow);

	sys->fprint(sys->fildes(2), "test-sdl3: Flush complete, sleeping 10 seconds...\n");
	sys->sleep(10000);

	sys->fprint(sys->fildes(2), "test-sdl3: Done\n");
}
