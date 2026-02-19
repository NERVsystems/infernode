implement Renderer;

#
# PDF renderer - thin wrapper around the PDF module.
#
# Renders PDF pages natively using Inferno's Draw primitives.
# Text extraction provided for the body buffer so the AI can
# read document content.
#

include "sys.m";
	sys: Sys;

include "draw.m";
	drawm: Draw;
	Display, Image, Font, Rect, Point: import drawm;

include "renderer.m";

include "pdf.m";
	pdf: PDF;
	Doc: import pdf;

display: ref Display;
curdoc: ref Doc;
curpage := 1;
totalpages := 0;
curdpi := 150;
stderr: ref Sys->FD;

# Max PDF size for in-memory parsing
MAXPARSE: con 64*1024*1024;

init(d: ref Display)
{
	sys = load Sys Sys->PATH;
	drawm = load Draw Draw->PATH;
	display = d;
	stderr = sys->fildes(2);

	pdf = load PDF PDF->PATH;
	if(pdf == nil)
		sys->fprint(stderr, "pdfrender: cannot load PDF module: %r\n");
	else {
		err := pdf->init(d);
		if(err != nil)
			sys->fprint(stderr, "pdfrender: pdf init: %s\n", err);
		else
			sys->fprint(stderr, "pdfrender: initialized OK\n");
	}
}

info(): ref RenderInfo
{
	return ref RenderInfo(
		"PDF",
		".pdf",
		1  # Has text content
	);
}

canrender(data: array of byte, hint: string): int
{
	if(len data >= 5 &&
	   data[0] == byte '%' && data[1] == byte 'P' &&
	   data[2] == byte 'D' && data[3] == byte 'F' &&
	   data[4] == byte '-')
		return 90;
	return 0;
}

render(data: array of byte, hint: string,
       width, height: int,
       progress: chan of ref RenderProgress): (ref Draw->Image, string, string)
{
	sys->fprint(stderr, "pdfrender: render called, hint=%s datalen=%d\n", hint, len data);

	if(pdf == nil){
		sys->fprint(stderr, "pdfrender: PDF module not available\n");
		progress <-= nil;
		return (nil, nil, "PDF module not available");
	}

	# Read file from path for parsing
	pdfdata := readpdffile(hint, MAXPARSE);
	if(pdfdata == nil){
		sys->fprint(stderr, "pdfrender: readpdffile failed, using data param (%d bytes)\n", len data);
		pdfdata = data;
	} else
		sys->fprint(stderr, "pdfrender: readpdffile OK, %d bytes\n", len pdfdata);

	if(pdfdata == nil || len pdfdata < 5){
		sys->fprint(stderr, "pdfrender: no PDF data\n");
		progress <-= nil;
		return (nil, nil, "no PDF data");
	}

	doc: ref Doc;
	oerr: string;
	{
		(doc, oerr) = pdf->open(pdfdata);
	} exception e {
	"*" =>
		sys->fprint(stderr, "pdfrender: open exception: %s\n", e);
		progress <-= nil;
		return (nil, nil, "PDF open exception: " + e);
	}
	if(doc == nil){
		sys->fprint(stderr, "pdfrender: open failed: %s\n", oerr);
		progress <-= nil;
		return (nil, nil, "PDF parse error: " + oerr);
	}

	curdoc = doc;
	totalpages = doc.pagecount();
	curpage = 1;
	sys->fprint(stderr, "pdfrender: opened OK, %d pages\n", totalpages);

	# Extract text for body buffer
	text := "";
	{
		text = doc.extractall();
	} exception {
	"*" =>
		text = "[text extraction failed]";
	}
	if(text == nil || len text == 0)
		text = "[No extractable text in PDF]";
	sys->fprint(stderr, "pdfrender: text extracted, %d chars\n", len text);

	# Compute DPI to fit the window (avoid enormous images)
	if(width > 0 && height > 0){
		(pw, ph) := doc.pagesize(curpage);
		if(pw > 0.0 && ph > 0.0){
			dpix := real width * 72.0 / pw;
			dpiy := real height * 72.0 / ph;
			fitdpi := int dpix;
			if(int dpiy < fitdpi)
				fitdpi = int dpiy;
			# Use 2x the fit DPI for quality (will be scaled down)
			fitdpi *= 2;
			if(fitdpi < 72) fitdpi = 72;
			if(fitdpi < curdpi)
				curdpi = fitdpi;
			sys->fprint(stderr, "pdfrender: page %.0fx%.0fpt, window %dx%d, dpi=%d\n",
				pw, ph, width, height, curdpi);
		}
	}

	# Render first page
	im: ref Draw->Image;
	rerr: string;
	{
		(im, rerr) = doc.renderpage(curpage, curdpi);
	} exception e {
	"*" =>
		sys->fprint(stderr, "pdfrender: renderpage exception: %s\n", e);
		progress <-= nil;
		return (nil, text, "render exception: " + e);
	}
	pdfdata = nil;  # Free before return

	progress <-= nil;

	if(im == nil && rerr != nil){
		sys->fprint(stderr, "pdfrender: renderpage error: %s\n", rerr);
		return (nil, text, "render: " + rerr);
	}

	if(im != nil)
		sys->fprint(stderr, "pdfrender: renderpage OK, image %dx%d\n",
			im.r.dx(), im.r.dy());
	else
		sys->fprint(stderr, "pdfrender: renderpage returned nil image, no error\n");

	return (im, text, nil);
}

commands(): list of ref Command
{
	return
		ref Command("NextPage", "b2", "n", nil) ::
		ref Command("PrevPage", "b2", "p", nil) ::
		ref Command("FirstPage", "b2", "^", nil) ::
		ref Command("Zoom+", "b2", "+", nil) ::
		ref Command("Zoom-", "b2", "-", nil) ::
		nil;
}

command(cmd: string, arg: string,
        data: array of byte, hint: string,
        width, height: int): (ref Draw->Image, string)
{
	if(curdoc == nil)
		return (nil, "no document loaded");

	case cmd {
	"NextPage" =>
		if(curpage < totalpages)
			curpage++;
		else
			return (nil, nil);
	"PrevPage" =>
		if(curpage > 1)
			curpage--;
		else
			return (nil, nil);
	"FirstPage" =>
		curpage = 1;
	"Zoom+" =>
		curdpi += 25;
		if(curdpi > 600) curdpi = 600;
	"Zoom-" =>
		curdpi -= 25;
		if(curdpi < 50) curdpi = 50;
	* =>
		return (nil, "unknown command: " + cmd);
	}

	# Compute DPI to fit the window
	renderdpi := curdpi;
	if(width > 0 && height > 0){
		(pw, ph) := curdoc.pagesize(curpage);
		if(pw > 0.0 && ph > 0.0){
			dpix := real width * 72.0 / pw;
			dpiy := real height * 72.0 / ph;
			fitdpi := int dpix;
			if(int dpiy < fitdpi)
				fitdpi = int dpiy;
			fitdpi *= 2;
			if(fitdpi < 72) fitdpi = 72;
			if(fitdpi < renderdpi)
				renderdpi = fitdpi;
		}
	}

	(im, err) := curdoc.renderpage(curpage, renderdpi);
	if(im == nil)
		return (nil, "render page " + string curpage + ": " + err);
	return (im, nil);
}

# Read a PDF file up to maxsize bytes
readpdffile(path: string, maxsize: int): array of byte
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return nil;
	(ok, dir) := sys->fstat(fd);
	if(ok != 0 || int dir.length <= 0)
		return nil;
	fsize := int dir.length;
	if(fsize > maxsize){
		sys->fprint(stderr, "pdfrender: file too large: %d > %d\n", fsize, maxsize);
		return nil;
	}
	data := array[fsize] of byte;
	total := 0;
	while(total < fsize){
		n := sys->read(fd, data[total:], fsize - total);
		if(n <= 0)
			break;
		total += n;
	}
	if(total < fsize)
		return nil;
	return data;
}
