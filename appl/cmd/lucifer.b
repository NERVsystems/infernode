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
# Reads events, renders messages, accepts keyboard input.
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

include "bufio.m";

include "imagefile.m";

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
COLHUMAN:	con int 16r1E2028FF;	# human tile bg  (visibly distinct from bg)
COLVELTRO:	con int 16r0E1418FF;	# veltro tile bg
COLINPUT:	con int 16r101010FF;	# input field bg
COLCURSOR:	con int 16rE8553AFF;	# cursor (accent)
COLGREEN:	con int 16r44AA44FF;	# resource: streaming
COLYELLOW:	con int 16rAAAA44FF;	# resource: stale
COLRED:		con int 16rAA4444FF;	# resource: offline/error
COLPROGBG:	con int 16r1A1A1AFF;	# progress bar bg
COLPROGFG:	con int 16r3388CCFF;	# progress bar fill

# --- Markdown constants ---

MD_PLAIN:   con 0;
MD_BOLD:    con 1;
MD_CODE:    con 2;

ML_NORMAL:  con 0;
ML_HEADER:  con 1;
ML_BULLET:  con 2;
ML_CODEBLK: con 3;
ML_BLANK:   con 4;

# One processed markdown line
MdPL: adt {
	kind: int;	# ML_*
	text: string;
	bold: int;
	mono: int;
};

# --- Data model ---

ConvMsg: adt {
	role:	string;		# "human" or "veltro"
	text:	string;
	using:	string;
};

Artifact: adt {
	id:	string;
	atype:	string;
	label:	string;
};

Resource: adt {
	path:	string;
	label:	string;
	rtype:	string;
	status:	string;
};

Gap: adt {
	desc:	string;
	relevance: string;
};

# Used for click hit-testing — records each drawn message tile
TileRect: adt {
	r:   Rect;
	msg: ref ConvMsg;
};

BgTask: adt {
	label:	string;
	status:	string;
	progress: string;
};

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
humancol: ref Image;
veltrocol: ref Image;
inputcol: ref Image;
cursorcol: ref Image;
greencol: ref Image;
yellowcol: ref Image;
redcol: ref Image;
progbgcol: ref Image;
progfgcol: ref Image;

# Fonts
mainfont: ref Font;
boldfont: ref Font;
monofont: ref Font;

# Logo
logoimg: ref Image;

# UI mount point
mountpt: string;

# Activity state
actid := -1;
actlabel: string;
actstatus: string;

# Conversation
messages: list of ref ConvMsg;
nmsg := 0;

# Input buffer
inputbuf: string;

# Presentation
artifacts: list of ref Artifact;
nart := 0;

# Context
resources: list of ref Resource;
gaps: list of ref Gap;
bgtasks: list of ref BgTask;

# Pixel-based scrolling (0 = bottom/newest, positive = scrolled up into history)
scrollpx := 0;
maxscrollpx := 0;
viewport_h := 400;	# message area height; updated by drawconversation each frame

# Username (read from /dev/user at startup)
username := "human";

# Tile layout — populated by drawconversation(), used for click hit-testing
tilelayout: array of ref TileRect;
ntiles := 0;

# Channels
cmouse: chan of ref Pointer;
uievent: chan of int;	# just triggers redraw

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
	humancol = display.color(COLHUMAN);
	veltrocol = display.color(COLVELTRO);
	inputcol = display.color(COLINPUT);
	cursorcol = display.color(COLCURSOR);
	greencol = display.color(COLGREEN);
	yellowcol = display.color(COLYELLOW);
	redcol = display.color(COLRED);
	progbgcol = display.color(COLPROGBG);
	progfgcol = display.color(COLPROGFG);

	# Load fonts (fall back gracefully)
	mainfont = Font.open(display, "/fonts/vera/Vera/unicode.14.font");
	if(mainfont == nil)
		mainfont = Font.open(display, "/fonts/vera/Vera/Vera.14.font");
	if(mainfont == nil)
		mainfont = Font.open(display, "*default*");
	boldfont = Font.open(display, "/fonts/vera/VeraBd/unicode.14.font");
	if(boldfont == nil)
		boldfont = mainfont;
	monofont = Font.open(display, "/fonts/vera/VeraMono/unicode.14.font");
	if(monofont == nil)
		monofont = mainfont;

	# Load logo (22x32 RGBA PNG with transparent background)
	bufio := load Bufio Bufio->PATH;
	if(bufio != nil) {
		readpng := load RImagefile RImagefile->READPNGPATH;
		remap := load Imageremap Imageremap->PATH;
		if(readpng != nil && remap != nil) {
			readpng->init(bufio);
			remap->init(display);
			fd := bufio->open("/lib/lucifer/logo.png", Bufio->OREAD);
			if(fd != nil) {
				(raw, nil) := readpng->read(fd);
				if(raw != nil)
					(logoimg, nil) = remap->remap(raw, display, 0);
			}
		}
	}

	# Find current activity
	s := readfile(mountpt + "/activity/current");
	if(s != nil)
		actid = strtoint(strip(s));

	# Load initial state
	if(actid >= 0) {
		loadlabel();
		loadstatus();
		loadmessages();
		loadpresentation();
		loadcontext();
	}

	inputbuf = "";
	username = readdevuser();
	cmouse = chan of ref Pointer;
	uievent = chan[1] of int;

	# Draw initial frame
	redraw();

	# Spawn event handlers
	spawn eventproc();
	spawn mouseproc();
	spawn kbdproc();
	if(actid >= 0)
		spawn nslistener();

	# Main loop
	mainloop();
}

mainloop()
{
	prevbuttons := 0;
	for(;;) alt {
	p := <-cmouse =>
		wasdown := prevbuttons;
		prevbuttons = p.buttons;
		if(p.buttons & M_QUIT) {
			shutdown();
			return;
		}
		if(p.buttons & M_RESIZE) {
			mainwin = win.image;
			redraw();
		}
		# Button-1 just pressed: copy the tapped message tile to snarf
		if(p.buttons == 1 && wasdown == 0) {
			for(ti := 0; ti < ntiles; ti++) {
				if(tilelayout[ti].r.contains(p.xy)) {
					writetosnarf(tilelayout[ti].msg.text);
					break;
				}
			}
		}
	<-uievent =>
		redraw();
	}
}

shutdown()
{
	fd := sys->open("/dev/sysctl", Sys->OWRITE);
	if(fd != nil)
		sys->fprint(fd, "halt");
	wmclient->win.wmctl("exit");
}

# --- Namespace reading ---

loadlabel()
{
	s := readfile(sys->sprint("%s/activity/%d/label", mountpt, actid));
	if(s != nil)
		actlabel = strip(s);
	else
		actlabel = "";
}

loadstatus()
{
	s := readfile(sys->sprint("%s/activity/%d/status", mountpt, actid));
	if(s != nil)
		actstatus = strip(s);
	else
		actstatus = "";
}

loadmessages()
{
	messages = nil;
	nmsg = 0;
	base := sys->sprint("%s/activity/%d/conversation", mountpt, actid);
	for(i := 0; ; i++) {
		s := readfile(sys->sprint("%s/%d", base, i));
		if(s == nil)
			break;
		s = strip(s);
		attrs := parseattrs(s);
		role := getattr(attrs, "role");
		text := getattr(attrs, "text");
		using := getattr(attrs, "using");
		if(role == nil)
			role = "?";
		if(text == nil)
			text = "";
		messages = ref ConvMsg(role, text, using) :: messages;
		nmsg++;
	}
	# Reverse to chronological order
	messages = revmsgs(messages);
}

loadmessage(idx: int)
{
	base := sys->sprint("%s/activity/%d/conversation", mountpt, actid);
	s := readfile(sys->sprint("%s/%d", base, idx));
	if(s == nil)
		return;
	s = strip(s);
	attrs := parseattrs(s);
	role := getattr(attrs, "role");
	text := getattr(attrs, "text");
	using := getattr(attrs, "using");
	if(role == nil)
		role = "?";
	if(text == nil)
		text = "";
	msg := ref ConvMsg(role, text, using);
	# Deduplicate: skip if we already optimistically displayed this human message
	if(role == "human" && messages != nil) {
		last: ref ConvMsg = nil;
		for(l := messages; l != nil; l = tl l)
			last = hd l;
		if(last != nil && last.role == "human" && last.text == text)
			return;
	}
	# Append to list
	messages = appendmsg(messages, msg);
	nmsg++;
	# Auto-scroll to bottom on new message
	scrollpx = 0;
}

loadpresentation()
{
	artifacts = nil;
	nart = 0;
	# Read presentation directory via numbered listing isn't possible
	# since artifacts use string IDs. Read via directory contents.
	# For now, just track what events tell us.
}

loadcontext()
{
	# Resources
	resources = nil;
	base := sys->sprint("%s/activity/%d/context/resources", mountpt, actid);
	for(i := 0; ; i++) {
		s := readfile(sys->sprint("%s/%d", base, i));
		if(s == nil)
			break;
		s = strip(s);
		attrs := parseattrs(s);
		resources = ref Resource(
			getattr(attrs, "path"),
			getattr(attrs, "label"),
			getattr(attrs, "type"),
			getattr(attrs, "status")
		) :: resources;
	}
	resources = revres(resources);

	# Gaps
	gaps = nil;
	base = sys->sprint("%s/activity/%d/context/gaps", mountpt, actid);
	for(i = 0; ; i++) {
		s := readfile(sys->sprint("%s/%d", base, i));
		if(s == nil)
			break;
		s = strip(s);
		attrs := parseattrs(s);
		gaps = ref Gap(
			getattr(attrs, "desc"),
			getattr(attrs, "relevance")
		) :: gaps;
	}
	gaps = revgaps(gaps);

	# Background tasks
	bgtasks = nil;
	base = sys->sprint("%s/activity/%d/context/background", mountpt, actid);
	for(i = 0; ; i++) {
		s := readfile(sys->sprint("%s/%d", base, i));
		if(s == nil)
			break;
		s = strip(s);
		attrs := parseattrs(s);
		bgtasks = ref BgTask(
			getattr(attrs, "label"),
			getattr(attrs, "status"),
			getattr(attrs, "progress")
		) :: bgtasks;
	}
	bgtasks = revbg(bgtasks);
}

# --- Namespace listener ---

nslistener()
{
	evpath := sys->sprint("%s/activity/%d/event", mountpt, actid);
	for(;;) {
		fd := sys->open(evpath, Sys->OREAD);
		if(fd == nil) {
			sys->sleep(1000);
			continue;
		}
		buf := array[4096] of byte;
		n := sys->read(fd, buf, len buf);
		if(n <= 0) {
			sys->sleep(500);
			continue;
		}
		ev := strip(string buf[0:n]);

		# Parse event and update state
		if(hasprefix(ev, "conversation ")) {
			idx := strtoint(ev[len "conversation ":]);
			if(idx >= 0)
				loadmessage(idx);
		} else if(ev == "status") {
			loadstatus();
		} else if(ev == "label") {
			loadlabel();
		} else if(hasprefix(ev, "context")) {
			loadcontext();
		} else if(hasprefix(ev, "presentation")) {
			loadpresentation();
		}

		# Trigger redraw
		alt {
		uievent <-= 1 => ;
		* => ;	# non-blocking
		}
	}
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
		if(wmclient->win.pointer(*p) == 0) {
			# Check for scroll wheel
			if(p.buttons & 8) {
				# Scroll up (older messages) — smooth pixel step
				scrollpx += mainfont.height * 3;
				if(scrollpx > maxscrollpx)
					scrollpx = maxscrollpx;
				alt {
				uievent <-= 1 => ;
				* => ;
				}
			} else if(p.buttons & 16) {
				# Scroll down (newer messages) — smooth pixel step
				scrollpx -= mainfont.height * 3;
				if(scrollpx < 0)
					scrollpx = 0;
				alt {
				uievent <-= 1 => ;
				* => ;
				}
			} else
				cmouse <-= p;
		}
	}
}

# --- Keyboard handling ---

kbdproc()
{
	for(;;) {
		c := <-win.ctxt.kbd;
		case c {
		8 or 127 =>
			# Backspace / Delete
			if(len inputbuf > 0)
				inputbuf = inputbuf[0:len inputbuf - 1];
		'\n' or 13 =>
			# Enter - send input
			if(len inputbuf > 0) {
				sendinput(inputbuf);
				inputbuf = "";
			}
		27 =>
			# Escape - clear buffer
			inputbuf = "";
		16rF00E =>
			# Page Up (Inferno keysym) — half viewport
			scrollpx += viewport_h / 2;
			if(scrollpx > maxscrollpx)
				scrollpx = maxscrollpx;
		16rF00F =>
			# Page Down (Inferno keysym) — half viewport
			scrollpx -= viewport_h / 2;
			if(scrollpx < 0)
				scrollpx = 0;
		* =>
			if(c == 'q' || c == 'Q') {
				if(len inputbuf == 0) {
					p := ref zpointer;
					p.buttons = M_QUIT;
					cmouse <-= p;
					continue;
				}
			}
			# Printable characters
			if(c >= 32 && c < 16rFFFF)
				inputbuf[len inputbuf] = c;
		}
		# Trigger redraw for keyboard changes
		alt {
		uievent <-= 1 => ;
		* => ;
		}
	}
}

sendinput(text: string)
{
	if(actid < 0)
		return;
	# Show the human message immediately without waiting for lucibridge echo
	messages = appendmsg(messages, ref ConvMsg("human", text, nil));
	nmsg++;
	scrollpx = 0;
	alt { uievent <-= 1 => ; * => ; }
	# Send to lucibridge (it will echo back as role=human; loadmessage deduplicates)
	path := sys->sprint("%s/activity/%d/conversation/input", mountpt, actid);
	fd := sys->open(path, Sys->OWRITE);
	if(fd == nil) {
		sys->fprint(stderr, "lucifer: can't open %s: %r\n", path);
		return;
	}
	b := array of byte text;
	sys->write(fd, b, len b);
}

# --- Drawing ---

redraw()
{
	if(mainwin == nil)
		return;

	r := mainwin.r;
	w := r.dx();

	# Fill background
	mainwin.draw(r, bgcol, nil, (0, 0));

	# Header bar (40px)
	headerh := 40;
	headerr := Rect((r.min.x, r.min.y), (r.max.x, r.min.y + headerh));
	mainwin.draw(headerr, headercol, nil, (0, 0));

	# Header text and logo
	if(mainfont != nil) {
		title := "InferNode";
		if(actlabel != nil && actlabel != "")
			title += " | " + actlabel;
		if(actstatus != nil && actstatus != "" && actstatus != "idle")
			title += " [" + actstatus + "]";
		texty := headerr.min.y + (headerh - mainfont.height) / 2;
		# Accent bar (4px left edge)
		mainwin.draw(Rect((r.min.x, r.min.y), (r.min.x + 4, r.min.y + headerh)),
			accentcol, nil, (0, 0));
		# Logo (after accent bar, before title)
		textx := r.min.x + 16;
		if(logoimg != nil) {
			lw := logoimg.r.dx();
			lh := logoimg.r.dy();
			logoy := headerr.min.y + (headerh - lh) / 2;
			logodst := Rect((textx, logoy), (textx + lw, logoy + lh));
			mainwin.draw(logodst, logoimg, nil, (0, 0));
			textx = textx + lw + 8;
		}
		# Title
		mainwin.text((textx, texty), textcol, (0, 0), mainfont, title);
	}

	# Zone layout below header
	zonety := r.min.y + headerh + 1;
	# Draw header/zone separator
	mainwin.draw(Rect((r.min.x, zonety - 1), (r.max.x, zonety)), bordercol, nil, (0, 0));

	# Zone widths: conversation ~30%, presentation ~45%, context ~25%
	convw := w * 30 / 100;
	presw := w * 45 / 100;

	convx := r.min.x;
	presx := convx + convw;
	ctxx := presx + presw;

	# Draw zone separators (1px vertical lines)
	mainwin.draw(Rect((presx, zonety), (presx + 1, r.max.y)), bordercol, nil, (0, 0));
	mainwin.draw(Rect((ctxx, zonety), (ctxx + 1, r.max.y)), bordercol, nil, (0, 0));

	# Draw zone labels
	if(mainfont != nil) {
		labely := zonety + 8;

		drawzonelabel(Rect((convx, zonety), (presx, r.max.y)), "Conversation", labely);
		drawzonelabel(Rect((presx + 1, zonety), (ctxx, r.max.y)), "Presentation", labely);
		drawzonelabel(Rect((ctxx + 1, zonety), (r.max.x, r.max.y)), "Context", labely);

		# Content area starts below zone labels
		contenty := zonety + 32;

		# Draw the three zones
		drawconversation(Rect((convx, contenty), (presx - 1, r.max.y)));
		drawpresentation(Rect((presx + 2, contenty), (ctxx - 1, r.max.y)));
		drawcontext(Rect((ctxx + 2, contenty), (r.max.x, r.max.y)));
	}

	mainwin.flush(Draw->Flushnow);
}

drawzonelabel(r: Rect, label: string, y: int)
{
	tw := mainfont.width(label);
	tx := r.min.x + (r.dx() - tw) / 2;
	mainwin.text((tx, y + 6), labelcol, (0, 0), mainfont, label);
}

# --- Conversation zone ---

drawconversation(zone: Rect)
{
	pad := 8;
	inputh := mainfont.height + 2 * pad;
	msgy := zone.max.y - inputh - 2;	# bottom of message area

	# Draw input field at bottom
	inputr := Rect((zone.min.x + pad, zone.max.y - inputh),
		(zone.max.x - pad, zone.max.y));
	mainwin.draw(inputr, inputcol, nil, (0, 0));

	# Input text
	itext := inputbuf;
	itx := inputr.min.x + pad;
	ity := inputr.min.y + (inputh - mainfont.height) / 2;
	maxitw := inputr.dx() - 2 * pad - 8;	# leave room for cursor

	# Truncate from left if too wide
	while(len itext > 0 && mainfont.width(itext) > maxitw)
		itext = itext[1:];
	mainwin.text((itx, ity), textcol, (0, 0), mainfont, itext);

	# Block cursor after text
	cw := 8;
	ch := mainfont.height;
	cx := itx + mainfont.width(itext);
	cy := ity;
	mainwin.draw(Rect((cx, cy), (cx + cw, cy + ch)), cursorcol, nil, (0, 0));

	# Draw messages bottom-up from msgy
	if(messages == nil) {
		drawcentertext(Rect((zone.min.x, zone.min.y), (zone.max.x, msgy)),
			"No messages yet");
		return;
	}

	# Reset tile layout for this frame
	tilelayout = array[nmsg + 1] of ref TileRect;
	ntiles = 0;

	# Get messages as array for indexed access
	marr := msgstoarray(messages, nmsg);

	# Tile layout parameters
	tilegap := 4;
	tpadv := 3;			# vertical padding only — no horizontal indent
	tilew := zone.dx() - 2 * pad;	# full width, both roles
	maxw  := tilew;
	tilex := zone.min.x + pad;	# same left edge for both roles

	# Pre-compute markdown parse and tile heights for all messages
	mdarr := array[nmsg] of list of ref MdPL;
	harr  := array[nmsg] of int;
	total_h := 0;
	for(pi := 0; pi < nmsg; pi++) {
		mdarr[pi] = parsemd(marr[pi].text);
		texth := mdheight(mdarr[pi], maxw);
		harr[pi] = mainfont.height + texth + 2 * tpadv;
		total_h += harr[pi] + tilegap;
	}

	# Update viewport height and pixel scroll bounds
	viewport_h = msgy - zone.min.y;
	newmax := total_h - viewport_h;
	if(newmax < 0)
		newmax = 0;
	maxscrollpx = newmax;
	if(scrollpx > maxscrollpx)
		scrollpx = maxscrollpx;

	# Draw messages bottom-up using pixel offset
	y := msgy + scrollpx;		# effective viewport floor
	for(i := nmsg - 1; i >= 0; i--) {
		tileh := harr[i];
		tiletop := y - tileh - tilegap;

		# Completely below visible area — skip
		if(tiletop >= msgy) {
			y = tiletop;
			continue;
		}
		# Completely above visible area — stop
		if(tiletop + tileh <= zone.min.y)
			break;

		msg := marr[i];
		human := msg.role == "human";
		tilecol: ref Image;
		rolecol: ref Image;
		if(human) {
			tilecol = humancol;
			rolecol = text2col;
		} else {
			tilecol = veltrocol;
			rolecol = accentcol;
		}

		# Draw tile background clamped to visible area
		drawtop := tiletop;
		if(drawtop < zone.min.y) drawtop = zone.min.y;
		drawbot := tiletop + tileh;
		if(drawbot > msgy) drawbot = msgy;
		if(drawtop < drawbot) {
			tiler := Rect((tilex, drawtop), (tilex + tilew, drawbot));
			mainwin.draw(tiler, tilecol, nil, (0, 0));
		}
		if(ntiles < len tilelayout)
			tilelayout[ntiles++] = ref TileRect(Rect((tilex, tiletop), (tilex + tilew, tiletop + tileh)), msg);

		# Role label (skip if outside visible area)
		ty := tiletop + tpadv;
		rolelabel := msg.role;
		if(human)
			rolelabel = username;
		if(ty >= zone.min.y && ty + mainfont.height <= msgy) {
			if(human)
				mainwin.text((tilex + tilew - mainfont.width(rolelabel), ty), rolecol, (0, 0), mainfont, rolelabel);
			else
				mainwin.text((tilex, ty), rolecol, (0, 0), mainfont, rolelabel);
		}
		ty += mainfont.height;

		# Markdown text with viewport clipping
		drawmdlines(mdarr[i], tilex, ty, maxw, zone.min.y, msgy, textcol, accentcol, human);

		y = tiletop;
	}
}

# --- Presentation zone ---

drawpresentation(zone: Rect)
{
	if(artifacts == nil && nart == 0) {
		drawcentertext(zone, "No artifacts");
		return;
	}

	pad := 8;
	y := zone.min.y + pad;
	for(a := artifacts; a != nil; a = tl a) {
		art := hd a;
		if(y + mainfont.height > zone.max.y)
			break;
		label := "[" + art.atype + "] " + art.label;
		mainwin.text((zone.min.x + pad, y), text2col, (0, 0), mainfont, label);
		y += mainfont.height + 4;
	}
}

# --- Context zone ---

drawcontext(zone: Rect)
{
	pad := 8;
	y := zone.min.y + pad;
	secgap := 12;
	indw := 10;	# status indicator width
	indh := 10;	# status indicator height

	# --- Resources section ---
	if(resources != nil) {
		mainwin.text((zone.min.x + pad, y), labelcol, (0, 0), mainfont, "Resources");
		y += mainfont.height + 4;

		for(r := resources; r != nil; r = tl r) {
			res := hd r;
			if(y + mainfont.height > zone.max.y)
				break;

			# Status indicator (small filled rect)
			indcol := dimcol;
			if(res.status == "streaming")
				indcol = greencol;
			else if(res.status == "stale")
				indcol = yellowcol;
			else if(res.status == "offline" || res.status == "error")
				indcol = redcol;

			indy := y + (mainfont.height - indh) / 2;
			mainwin.draw(Rect(
				(zone.min.x + pad, indy),
				(zone.min.x + pad + indw, indy + indh)),
				indcol, nil, (0, 0));

			# Label
			label := res.label;
			if(label == nil || label == "")
				label = res.path;
			mainwin.text((zone.min.x + pad + indw + 6, y),
				text2col, (0, 0), mainfont, label);
			y += mainfont.height + 2;
		}
		y += secgap;
	}

	# --- Gaps section ---
	if(gaps != nil) {
		if(y + mainfont.height > zone.max.y)
			return;
		mainwin.text((zone.min.x + pad, y), labelcol, (0, 0), mainfont, "Gaps");
		y += mainfont.height + 4;

		for(g := gaps; g != nil; g = tl g) {
			gap := hd g;
			if(y + mainfont.height > zone.max.y)
				break;
			desc := gap.desc;
			if(gap.relevance != nil && gap.relevance != "")
				desc += " [" + gap.relevance + "]";
			mainwin.text((zone.min.x + pad, y), text2col, (0, 0), mainfont, desc);
			y += mainfont.height + 2;
		}
		y += secgap;
	}

	# --- Background tasks section ---
	if(bgtasks != nil) {
		if(y + mainfont.height > zone.max.y)
			return;
		mainwin.text((zone.min.x + pad, y), labelcol, (0, 0), mainfont, "Background");
		y += mainfont.height + 4;

		barh := 6;
		for(b := bgtasks; b != nil; b = tl b) {
			bg := hd b;
			if(y + mainfont.height + barh + 4 > zone.max.y)
				break;

			# Task label + status
			label := bg.label;
			if(bg.status != nil && bg.status != "")
				label += " [" + bg.status + "]";
			mainwin.text((zone.min.x + pad, y), text2col, (0, 0), mainfont, label);
			y += mainfont.height + 2;

			# Progress bar
			if(bg.progress != nil && bg.progress != "") {
				pct := strtoint(bg.progress);
				if(pct < 0)
					pct = 0;
				if(pct > 100)
					pct = 100;
				barw := zone.dx() - 2 * pad;
				bary := y;
				# Background
				mainwin.draw(Rect(
					(zone.min.x + pad, bary),
					(zone.min.x + pad + barw, bary + barh)),
					progbgcol, nil, (0, 0));
				# Fill
				fillw := barw * pct / 100;
				if(fillw > 0)
					mainwin.draw(Rect(
						(zone.min.x + pad, bary),
						(zone.min.x + pad + fillw, bary + barh)),
						progfgcol, nil, (0, 0));
				y += barh + 4;
			}
		}
	}

	# Empty state
	if(resources == nil && gaps == nil && bgtasks == nil)
		drawcentertext(zone, "No context");
}

drawcentertext(r: Rect, text: string)
{
	tw := mainfont.width(text);
	tx := r.min.x + (r.dx() - tw) / 2;
	ty := r.min.y + (r.dy() - mainfont.height) / 2;
	mainwin.text((tx, ty), dimcol, (0, 0), mainfont, text);
}

# --- Word wrapping ---

wraptext(text: string, maxw: int): list of string
{
	if(text == nil || text == "")
		return "" :: nil;

	lines: list of string;
	line := "";

	i := 0;
	while(i < len text) {
		# Find next word
		while(i < len text && (text[i] == ' ' || text[i] == '\t'))
			i++;
		if(i >= len text)
			break;
		wstart := i;
		while(i < len text && text[i] != ' ' && text[i] != '\t' && text[i] != '\n')
			i++;
		word := text[wstart:i];

		# Handle newline
		if(i < len text && text[i] == '\n') {
			if(line != "")
				line += " " + word;
			else
				line = word;
			lines = line :: lines;
			line = "";
			i++;
			continue;
		}

		# Check if word fits on current line
		candidate: string;
		if(line != "")
			candidate = line + " " + word;
		else
			candidate = word;

		if(mainfont.width(candidate) > maxw && line != "") {
			# Wrap: current line is done, start new with word
			lines = line :: lines;
			line = word;
		} else {
			line = candidate;
		}
	}
	if(line != "")
		lines = line :: lines;
	if(lines == nil)
		return "" :: nil;

	# Reverse to correct order
	rev: list of string;
	for(; lines != nil; lines = tl lines)
		rev = hd lines :: rev;
	return rev;
}

# --- Markdown processing ---

# wraptext variant using an explicit font for width measurement
wraptext_f(text: string, fnt: ref Font, maxw: int): list of string
{
	if(text == nil || text == "")
		return "" :: nil;

	lines: list of string;
	line := "";
	i := 0;
	while(i < len text) {
		while(i < len text && (text[i] == ' ' || text[i] == '\t'))
			i++;
		if(i >= len text)
			break;
		wstart := i;
		while(i < len text && text[i] != ' ' && text[i] != '\t' && text[i] != '\n')
			i++;
		word := text[wstart:i];
		if(i < len text && text[i] == '\n') {
			if(line != "")
				line += " " + word;
			else
				line = word;
			lines = line :: lines;
			line = "";
			i++;
			continue;
		}
		candidate: string;
		if(line != "")
			candidate = line + " " + word;
		else
			candidate = word;
		if(fnt.width(candidate) > maxw && line != "") {
			lines = line :: lines;
			line = word;
		} else {
			line = candidate;
		}
	}
	if(line != "")
		lines = line :: lines;
	if(lines == nil)
		return "" :: nil;
	rev: list of string;
	for(; lines != nil; lines = tl lines)
		rev = hd lines :: rev;
	return rev;
}

# Strip markdown inline markers (**, *, `, _) from a string
stripmarkers(s: string): string
{
	result := "";
	i := 0;
	n := len s;
	while(i < n) {
		c := s[i];
		if(c == '*' || c == '_' || c == '`') {
			i++;
			continue;
		}
		result[len result] = c;
		i++;
	}
	return result;
}

# Parse one raw markdown line into (kind, display-text, use-bold, use-mono).
# codeblk: pass-through flag for ``` blocks (updated by caller).
# Returns (kind, text, bold, mono).
parsemdline(raw: string): (int, string, int, int)
{
	# Strip leading whitespace
	si := 0;
	while(si < len raw && (raw[si] == ' ' || raw[si] == '\t'))
		si++;
	s := raw[si:];

	# Blank line
	if(s == "")
		return (ML_BLANK, "", 0, 0);

	# Header: # text or ## text
	if(s[0] == '#') {
		level := 0;
		while(level < len s && s[level] == '#')
			level++;
		htext := s[level:];
		hi := 0;
		while(hi < len htext && (htext[hi] == ' ' || htext[hi] == '\t'))
			hi++;
		return (ML_HEADER, stripmarkers(htext[hi:]), 1, 0);
	}

	# Bullet: "- " or "* " (not code fences)
	if(len s >= 2 && s[0] == '-' && s[1] == ' ')
		return (ML_BULLET, stripmarkers(s[2:]), 0, 0);
	if(len s >= 2 && s[0] == '*' && s[1] == ' ')
		return (ML_BULLET, stripmarkers(s[2:]), 0, 0);

	# Numbered list: "1. "
	k := 0;
	while(k < len s && s[k] >= '0' && s[k] <= '9')
		k++;
	if(k > 0 && k+1 < len s && s[k] == '.' && s[k+1] == ' ')
		return (ML_BULLET, stripmarkers(s[k+2:]), 0, 0);

	# Whole-line bold: **text**
	if(len s >= 4 && s[0:2] == "**" && s[len s - 2:] == "**")
		return (ML_NORMAL, s[2:len s - 2], 1, 0);

	# Normal line: strip inline markers
	return (ML_NORMAL, stripmarkers(s), 0, 0);
}

# Parse a full message text into a list of MdPL.
# Handles ``` code blocks as a toggle.
parsemd(text: string): list of ref MdPL
{
	result: list of ref MdPL;
	incode := 0;
	i := 0;
	n := len text;
	while(i <= n) {
		j := i;
		while(j < n && text[j] != '\n')
			j++;
		raw := text[i:j];
		if(j >= n)
			i = n + 1;
		else
			i = j + 1;

		# ``` fence toggles code block mode
		si := 0;
		while(si < len raw && (raw[si] == ' ' || raw[si] == '\t'))
			si++;
		s := raw[si:];
		if(len s >= 3 && s[0:3] == "```") {
			if(incode)
				incode = 0;
			else
				incode = 1;
			continue;
		}

		if(incode) {
			result = ref MdPL(ML_CODEBLK, raw, 0, 1) :: result;
			continue;
		}

		(kind, text2, bold, mono) := parsemdline(raw);
		result = ref MdPL(kind, text2, bold, mono) :: result;
	}
	rev: list of ref MdPL;
	for(; result != nil; result = tl result)
		rev = hd result :: rev;
	return rev;
}

# Compute total pixel height of a list of MdPL lines (no drawing).
mdheight(mdlines: list of ref MdPL, maxw: int): int
{
	lineh := mainfont.height;
	h := 0;
	for(; mdlines != nil; mdlines = tl mdlines) {
		pl := hd mdlines;
		fnt := mainfont;
		if(pl.bold || pl.kind == ML_HEADER)
			fnt = boldfont;
		if(pl.mono || pl.kind == ML_CODEBLK)
			fnt = monofont;
		case pl.kind {
		ML_BLANK =>
			h += lineh / 2;
		ML_BULLET =>
			wls := wraptext_f("• " + pl.text, fnt, maxw);
			for(; wls != nil; wls = tl wls)
				h += lineh;
		* =>
			wls := wraptext_f(pl.text, fnt, maxw);
			for(; wls != nil; wls = tl wls)
				h += lineh;
		}
	}
	return h;
}

# Draw MdPL lines inside a tile, respecting viewport clip [clip_top, clip_bot).
# Returns final y after drawing.
drawmdlines(mdlines: list of ref MdPL, x0, ty, maxw, clip_top, clip_bot: int,
            txtcol, hdrcol: ref Image, human: int): int
{
	lineh := mainfont.height;
	for(; mdlines != nil; mdlines = tl mdlines) {
		pl := hd mdlines;
		fnt := mainfont;
		col := txtcol;
		if(pl.bold || pl.kind == ML_HEADER) {
			fnt = boldfont;
			if(pl.kind == ML_HEADER)
				col = hdrcol;
		}
		if(pl.mono || pl.kind == ML_CODEBLK)
			fnt = monofont;
		case pl.kind {
		ML_BLANK =>
			ty += lineh / 2;
		* =>
			drawtext := pl.text;
			if(pl.kind == ML_BULLET)
				drawtext = "• " + pl.text;
			wls := wraptext_f(drawtext, fnt, maxw);
			for(; wls != nil; wls = tl wls) {
				if(ty >= clip_top && ty + lineh <= clip_bot) {
					tw := fnt.width(hd wls);
					lx := x0;
					if(human)
						lx = x0 + maxw - tw;
					mainwin.text((lx, ty), col, (0, 0), fnt, hd wls);
				}
				ty += lineh;
			}
		}
	}
	return ty;
}

# --- Attribute parsing ---
# Same format as luciuisrv: "key1=val1 key2=val2"

Attr: adt {
	key: string;
	val: string;
};

parseattrs(s: string): list of ref Attr
{
	kstarts := array[32] of int;
	eqposs := array[32] of int;
	nkp := 0;

	i := 0;
	while(i < len s && (s[i] == ' ' || s[i] == '\t'))
		i++;

	j := i;
	while(j < len s) {
		if(s[j] == '=') {
			kstart := j - 1;
			while(kstart > i && s[kstart - 1] != ' ' && s[kstart - 1] != '\t')
				kstart--;
			if(kstart >= 0 && kstart < j) {
				if(kstart == 0 || kstart == i || s[kstart - 1] == ' ' || s[kstart - 1] == '\t') {
					if(nkp >= len kstarts) {
						nks := array[len kstarts * 2] of int;
						nks[0:] = kstarts[0:nkp];
						kstarts = nks;
						neq := array[len eqposs * 2] of int;
						neq[0:] = eqposs[0:nkp];
						eqposs = neq;
					}
					kstarts[nkp] = kstart;
					eqposs[nkp] = j;
					nkp++;
				}
			}
		}
		j++;
	}

	attrs: list of ref Attr;
	for(k := 0; k < nkp; k++) {
		key := s[kstarts[k]:eqposs[k]];
		vstart := eqposs[k] + 1;
		vend: int;
		if(k + 1 < nkp) {
			vend = kstarts[k + 1];
			while(vend > vstart && (s[vend - 1] == ' ' || s[vend - 1] == '\t'))
				vend--;
		} else
			vend = len s;
		val := "";
		if(vstart < vend)
			val = s[vstart:vend];
		attrs = ref Attr(key, val) :: attrs;
	}

	rev: list of ref Attr;
	for(; attrs != nil; attrs = tl attrs)
		rev = hd attrs :: rev;
	return rev;
}

getattr(attrs: list of ref Attr, key: string): string
{
	for(; attrs != nil; attrs = tl attrs)
		if((hd attrs).key == key)
			return (hd attrs).val;
	return nil;
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

writetosnarf(text: string)
{
	fd := sys->open("/dev/snarf", Sys->OWRITE);
	if(fd == nil)
		return;
	b := array of byte text;
	sys->write(fd, b, len b);
}

readfile(path: string): string
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return nil;
	buf := array[8192] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return nil;
	return string buf[0:n];
}

readdevuser(): string
{
	fd := sys->open("/dev/user", Sys->OREAD);
	if(fd == nil)
		return "human";
	buf := array[64] of byte;
	n := sys->read(fd, buf, len buf);
	if(n <= 0)
		return "human";
	s := string buf[0:n];
	# strip trailing newline/whitespace
	while(len s > 0 && (s[len s - 1] == '\n' || s[len s - 1] == ' '))
		s = s[0:len s - 1];
	if(len s == 0)
		return "human";
	return s;
}

strip(s: string): string
{
	while(len s > 0 && (s[len s - 1] == '\n' || s[len s - 1] == ' ' || s[len s - 1] == '\t'))
		s = s[0:len s - 1];
	return s;
}

strtoint(s: string): int
{
	n := 0;
	for(i := 0; i < len s; i++) {
		c := s[i];
		if(c < '0' || c > '9')
			return -1;
		n = n * 10 + (c - '0');
	}
	if(len s == 0)
		return -1;
	return n;
}

hasprefix(s, prefix: string): int
{
	return len s >= len prefix && s[0:len prefix] == prefix;
}

listlen(l: list of string): int
{
	n := 0;
	for(; l != nil; l = tl l)
		n++;
	return n;
}

# List reversal helpers (Limbo lacks generics)

revmsgs(l: list of ref ConvMsg): list of ref ConvMsg
{
	r: list of ref ConvMsg;
	for(; l != nil; l = tl l)
		r = hd l :: r;
	return r;
}

appendmsg(l: list of ref ConvMsg, m: ref ConvMsg): list of ref ConvMsg
{
	if(l == nil)
		return m :: nil;
	# Reverse, cons, reverse
	r: list of ref ConvMsg;
	for(; l != nil; l = tl l)
		r = hd l :: r;
	r = m :: r;
	result: list of ref ConvMsg;
	for(; r != nil; r = tl r)
		result = hd r :: result;
	return result;
}

msgstoarray(l: list of ref ConvMsg, n: int): array of ref ConvMsg
{
	a := array[n] of ref ConvMsg;
	i := 0;
	for(; l != nil && i < n; l = tl l)
		a[i++] = hd l;
	return a;
}

revres(l: list of ref Resource): list of ref Resource
{
	r: list of ref Resource;
	for(; l != nil; l = tl l)
		r = hd l :: r;
	return r;
}

revgaps(l: list of ref Gap): list of ref Gap
{
	r: list of ref Gap;
	for(; l != nil; l = tl l)
		r = hd l :: r;
	return r;
}

revbg(l: list of ref BgTask): list of ref BgTask
{
	r: list of ref BgTask;
	for(; l != nil; l = tl l)
		r = hd l :: r;
	return r;
}
