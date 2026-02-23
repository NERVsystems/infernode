implement Lucifer;

#
# lucifer - Lucifer GUI Coordinator
#
# Fullscreen three-zone layout for InferNode:
#   Left (~30%):   Conversation
#   Center (~45%): Presentation
#   Right (~25%):  Context
#
# Connects to /mnt/ui/ namespace served by luciuisrv.
#
# Usage:
#   lucifer                 use /mnt/ui
#   lucifer -m /n/ui        custom mount point
#

include "sys.m";
	sys: Sys;

include "draw.m";
	draw: Draw;
	Font, Point, Rect, Image, Context, Display, Screen, Pointer: import draw;

include "arg.m";

include "wmclient.m";
	wmclient: Wmclient;

Lucifer: module {
	init: fn(ctxt: ref Draw->Context, args: list of string);
};

# --- Color scheme ---
COLBG:		con int 16r080808FF;
COLBORDER:	con int 16r131313FF;
COLHEADER:	con int 16r0A0A0AFF;
COLACCENT:	con int 16rE8553AFF;
COLTEXT:	con int 16rCCCCCCFF;
COLTEXT2:	con int 16r999999FF;
COLDIM:		con int 16r444444FF;
COLLABEL:	con int 16r333333FF;

# --- Globals ---
stderr: ref Sys->FD;
display: ref Display;
win: ref Wmclient->Window;
mainwin: ref Image;

# Colors
bgcol: ref Image;
bordercol: ref Image;
headercol: ref Image;
accentcol: ref Image;
textcol: ref Image;
text2col: ref Image;
dimcol: ref Image;
labelcol: ref Image;

# Font
mainfont: ref Font;

# UI mount point
mountpt: string;

# Activity state read from namespace
actlabel: string;

# Channels
cmouse: chan of ref Pointer;

M_RESIZE: con 1 << 5;
M_QUIT: con 1 << 6;

nomod(s: string)
{
	sys->fprint(stderr, "lucifer: can't load %s: %r\n", s);
	raise "fail:load";
}

usage()
{
	sys->fprint(stderr, "Usage: lucifer [-m mountpoint]\n");
	raise "fail:usage";
}

init(ctxt: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	sys->pctl(Sys->NEWPGRP, nil);
	stderr = sys->fildes(2);

	draw = load Draw Draw->PATH;
	if(draw == nil)
		nomod(Draw->PATH);

	wmclient = load Wmclient Wmclient->PATH;
	if(wmclient == nil)
		nomod(Wmclient->PATH);
	wmclient->init();

	arg := load Arg Arg->PATH;
	if(arg == nil)
		nomod(Arg->PATH);
	arg->init(args);

	mountpt = "/mnt/ui";

	while((o := arg->opt()) != 0)
		case o {
		'm' =>	mountpt = arg->earg();
		* =>	usage();
		}
	arg = nil;

	# Create window
	if(ctxt == nil)
		ctxt = wmclient->makedrawcontext();
	display = ctxt.display;

	buts := Wmclient->Appl;
	if(ctxt.wm == nil)
		buts = Wmclient->Plain;
	win = wmclient->window(ctxt, "Lucifer", buts);
	wmclient->win.reshape(((0, 0), (win.displayr.size())));
	wmclient->win.onscreen("place");
	wmclient->win.startinput("kbd"::"ptr"::nil);
	mainwin = win.image;

	# Allocate colors
	bgcol = display.color(COLBG);
	bordercol = display.color(COLBORDER);
	headercol = display.color(COLHEADER);
	accentcol = display.color(COLACCENT);
	textcol = display.color(COLTEXT);
	text2col = display.color(COLTEXT2);
	dimcol = display.color(COLDIM);
	labelcol = display.color(COLLABEL);

	# Load font
	mainfont = Font.open(display, "/fonts/vera/Vera-Roman/14a.font");
	if(mainfont == nil)
		mainfont = Font.open(display, "*default*");

	# Read activity state from namespace
	actlabel = readfile(mountpt + "/activity/current");
	if(actlabel != nil) {
		# Try to read the label of the current activity
		id := strip(actlabel);
		l := readfile(mountpt + "/activity/" + id + "/label");
		if(l != nil)
			actlabel = strip(l);
		else
			actlabel = nil;
	}

	cmouse = chan of ref Pointer;

	# Draw initial frame
	redraw();

	# Spawn event handlers
	spawn eventproc();
	spawn mouseproc();
	spawn kbdproc();

	# Main loop
	mainloop();
}

mainloop()
{
	for(;;) {
		p := <-cmouse;
		if(p.buttons & M_QUIT) {
			shutdown();
			return;
		}
		if(p.buttons & M_RESIZE) {
			mainwin = win.image;
			redraw();
		}
	}
}

shutdown()
{
	fd := sys->open("/dev/sysctl", Sys->OWRITE);
	if(fd != nil)
		sys->fprint(fd, "halt");
	wmclient->win.wmctl("exit");
}

# --- Event handling ---

zpointer: Pointer;

eventproc()
{
	wmsize := startwmsize();
	for(;;) alt {
	wmsz := <-wmsize =>
		win.image = win.screen.newwindow(wmsz, Draw->Refnone, Draw->Nofill);
		p := ref zpointer;
		mainwin = win.image;
		p.buttons = M_RESIZE;
		cmouse <-= p;
	e := <-win.ctl or
	e = <-win.ctxt.ctl =>
		p := ref zpointer;
		if(e == "exit") {
			p.buttons = M_QUIT;
			cmouse <-= p;
		} else {
			wmclient->win.wmctl(e);
			if(win.image != mainwin) {
				mainwin = win.image;
				p.buttons = M_RESIZE;
				cmouse <-= p;
			}
		}
	}
}

mouseproc()
{
	for(;;) {
		p := <-win.ctxt.ptr;
		if(wmclient->win.pointer(*p) == 0)
			cmouse <-= p;
	}
}

# --- Keyboard handling ---

kbdproc()
{
	for(;;) {
		c := <-win.ctxt.kbd;
		if(c == 'q' || c == 'Q') {
			p := ref zpointer;
			p.buttons = M_QUIT;
			cmouse <-= p;
		}
	}
}

# --- Drawing ---

redraw()
{
	if(mainwin == nil)
		return;

	r := mainwin.r;
	w := r.dx();
	h := r.dy();

	# Fill background
	mainwin.draw(r, bgcol, nil, (0, 0));

	# Header bar (40px)
	headerh := 40;
	headerr := Rect((r.min.x, r.min.y), (r.max.x, r.min.y + headerh));
	mainwin.draw(headerr, headercol, nil, (0, 0));

	# Header text
	title := "InferNode";
	if(actlabel != nil && actlabel != "")
		title += " | " + actlabel;
	if(mainfont != nil) {
		texty := headerr.min.y + (headerh - mainfont.height) / 2;
		# Accent bar (4px left edge)
		mainwin.draw(Rect((r.min.x, r.min.y), (r.min.x + 4, r.min.y + headerh)),
			accentcol, nil, (0, 0));
		# Title
		mainwin.text((r.min.x + 16, texty), textcol, (0, 0), mainfont, title);
	}

	# Zone layout below header
	zonety := r.min.y + headerh + 1;
	# Draw header/zone separator
	mainwin.draw(Rect((r.min.x, zonety - 1), (r.max.x, zonety)), bordercol, nil, (0, 0));

	# Zone widths: conversation ~30%, presentation ~45%, context ~25%
	convw := w * 30 / 100;
	presw := w * 45 / 100;
	# context gets the rest

	convx := r.min.x;
	presx := convx + convw;
	ctxx := presx + presw;

	# Draw zone separators (1px vertical lines)
	mainwin.draw(Rect((presx, zonety), (presx + 1, r.max.y)), bordercol, nil, (0, 0));
	mainwin.draw(Rect((ctxx, zonety), (ctxx + 1, r.max.y)), bordercol, nil, (0, 0));

	# Draw zone labels
	if(mainfont != nil) {
		labely := zonety + 8;

		# Conversation zone label
		drawzonelabel(Rect((convx, zonety), (presx, r.max.y)), "Conversation", labely);
		# Presentation zone label
		drawzonelabel(Rect((presx + 1, zonety), (ctxx, r.max.y)), "Presentation", labely);
		# Context zone label
		drawzonelabel(Rect((ctxx + 1, zonety), (r.max.x, r.max.y)), "Context", labely);

		# Draw placeholder text in each zone
		placey := zonety + 40;
		drawcentertext(Rect((convx, placey), (presx, r.max.y)),
			"Messages will appear here", placey);
		drawcentertext(Rect((presx + 1, placey), (ctxx, r.max.y)),
			"Artifacts will appear here", placey);
		drawcentertext(Rect((ctxx + 1, placey), (r.max.x, r.max.y)),
			"Resources will appear here", placey);
	}

	mainwin.flush(Draw->Flushnow);
}

drawzonelabel(r: Rect, label: string, y: int)
{
	# Draw label header background
	headerh := 28;
	hr := Rect((r.min.x + 8, y), (r.max.x - 8, y + headerh));
	# Label text centered
	tw := mainfont.width(label);
	tx := r.min.x + (r.dx() - tw) / 2;
	mainwin.text((tx, y + 6), labelcol, (0, 0), mainfont, label);
}

drawcentertext(r: Rect, text: string, y: int)
{
	tw := mainfont.width(text);
	tx := r.min.x + (r.dx() - tw) / 2;
	mainwin.text((tx, y), dimcol, (0, 0), mainfont, text);
}

# --- WM size tracking ---

startwmsize(): chan of Rect
{
	rchan := chan of Rect;
	fd := sys->open("/dev/wmsize", Sys->OREAD);
	if(fd == nil)
		return rchan;
	sync := chan of int;
	spawn wmsizeproc(sync, fd, rchan);
	<-sync;
	return rchan;
}

Wmsize: con 1 + 4*12;

wmsizeproc(sync: chan of int, fd: ref Sys->FD, ptr: chan of Rect)
{
	sync <-= sys->pctl(0, nil);
	b := array[Wmsize] of byte;
	while(sys->read(fd, b, len b) > 0) {
		p := bytes2rect(b);
		if(p != nil)
			ptr <-= *p;
	}
}

bytes2rect(b: array of byte): ref Rect
{
	if(len b < Wmsize || int b[0] != 'm')
		return nil;
	x := int string b[1:13];
	y := int string b[13:25];
	return ref Rect((0, 0), (x, y));
}

# --- Helpers ---

readfile(path: string): string
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return nil;
	buf := array[1024] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return nil;
	return string buf[0:n];
}

strip(s: string): string
{
	# Remove trailing whitespace/newline
	while(len s > 0 && (s[len s - 1] == '\n' || s[len s - 1] == ' ' || s[len s - 1] == '\t'))
		s = s[0:len s - 1];
	return s;
}
