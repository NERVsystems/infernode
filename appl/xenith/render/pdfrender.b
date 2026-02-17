implement Renderer;

#
# PDF renderer - renders PDF pages via host-side pdftoppm, extracts
# text for the body buffer so the AI can read the document content.
#
# Rendering pipeline:
#   1. Host-side: pdftoppm rasterizes PDF pages to PPM images via /cmd device
#   2. Limbo-side: imgload converts PPM to Draw->Image for display
#   3. Text extraction: PDF parser + ToUnicode CMap decoding for body buffer
#
# Falls back to text-only rlayout rendering if pdftoppm is unavailable.
#

include "sys.m";
	sys: Sys;

include "draw.m";
	drawm: Draw;
	Display, Image, Font, Rect, Point: import drawm;

include "filter.m";
	filtermod: Filter;

include "renderer.m";
include "rlayout.m";

# Image loader for PPM decode (loaded dynamically like imgrender does)
Imgload: module {
	PATH: con "/dis/xenith/imgload.dis";

	ImgProgress: adt {
		image: ref Draw->Image;
		rowsdone: int;
		rowstotal: int;
	};

	init: fn(d: ref Draw->Display);
	readimagedata: fn(data: array of byte, hint: string): (ref Draw->Image, string);
};

rlayout: Rlayout;
imgload: Imgload;
display: ref Display;
DocNode: import rlayout;

# Font paths (for text-only fallback)
PROPFONT: con "/fonts/vera/Vera/unicode.14.font";
MONOFONT: con "/fonts/vera/VeraMono/VeraMono.14.font";

propfont: ref Font;
monofont: ref Font;

# Page navigation state
curpage := 1;
totalpages := 0;
curdpi := 150;

# Max PDF size for in-memory text extraction
MAXPARSE: con 8*1024*1024;

# ---- PDF internal types ----

# PDF object types
Onull, Obool, Oint, Oreal, Ostring, Oname,
Oarray, Odict, Ostream, Oref: con iota;

PdfObj: adt {
	kind: int;
	ival: int;           # Obool, Oint
	rval: real;          # Oreal
	sval: string;        # Ostring, Oname
	aval: list of ref PdfObj;  # Oarray
	dval: list of ref DictEntry; # Odict, Ostream
	stream: array of byte;    # Ostream (raw, possibly compressed)
	# For Oref: generation in rval (as int), objnum in ival
};

DictEntry: adt {
	key: string;
	val: ref PdfObj;
};

# Cross-reference entry
XrefEntry: adt {
	offset: int;
	gen: int;
	inuse: int;
};

# CMap entry: CIDs lo..hi map to Unicode starting at unicode
CMapEntry: adt {
	lo: int;
	hi: int;
	unicode: int;
};

# Per-font CMap: name is "F1"/"F2" etc., twobyte flags 2-byte CID encoding
FontMapEntry: adt {
	name: string;
	twobyte: int;
	entries: list of ref CMapEntry;
};

# Parsed PDF document
PdfDoc: adt {
	data: array of byte;
	xref: array of ref XrefEntry;
	trailer: ref PdfObj;     # trailer dictionary
	nobjs: int;
};

init(d: ref Draw->Display)
{
	sys = load Sys Sys->PATH;
	drawm = load Draw Draw->PATH;
	display = d;

	rlayout = load Rlayout Rlayout->PATH;
	if(rlayout != nil)
		rlayout->init(d);

	imgload = load Imgload Imgload->PATH;
	if(imgload != nil)
		imgload->init(d);

	propfont = Font.open(d, PROPFONT);
	monofont = Font.open(d, MONOFONT);
	if(propfont == nil)
		propfont = Font.open(d, "*default*");
	if(monofont == nil)
		monofont = propfont;
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
	# Check for %PDF- magic
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
	# Read file from path for text extraction (size-limited)
	text := "";
	totalpages = 0;
	perr: string;
	pdfdata := readpdffile(hint, MAXPARSE);
	if(pdfdata != nil){
		doc: ref PdfDoc;
		(doc, perr) = parsepdf(pdfdata);
		if(doc != nil){
			totalpages = countpages(doc);
			(text, nil) = extracttext(doc);
		}
		pdfdata = nil;  # Free before render
	}
	if(text == nil || len text == 0)
		text = "[No extractable text in PDF]";

	curpage = 1;

	# Try host-side rendering via pdftoppm (streams from path)
	(im, rerr) := hostrender(hint, curpage);
	if(im != nil){
		progress <-= nil;
		return (im, text, nil);
	}

	# Fallback: text-only rendering via rlayout
	if(rlayout == nil || propfont == nil){
		progress <-= nil;
		if(perr != nil)
			return (nil, nil, "PDF parse error: " + perr);
		return (nil, text, "host render failed: " + rerr);
	}

	docnodes := texttodom(text);

	if(width <= 0)
		width = 800;

	fgcolor := display.color(drawm->Black);
	bgcolor := display.color(drawm->White);
	linkcolor := display.newimage(Rect(Point(0,0), Point(1,1)), drawm->RGB24, 1, 16r2255AA);
	codebg := display.newimage(Rect(Point(0,0), Point(1,1)), drawm->RGB24, 1, 16rF0F0F0);

	style := ref Rlayout->Style(
		width,
		12,
		propfont,
		monofont,
		fgcolor,
		bgcolor,
		linkcolor,
		codebg,
		150
	);

	(img, nil) := rlayout->render(docnodes, style);

	progress <-= nil;
	return (img, text, nil);
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
	* =>
		return (nil, "unknown command: " + cmd);
	}

	(im, err) := hostrender(hint, curpage);
	if(im == nil)
		return (nil, "render page " + string curpage + ": " + err);
	return (im, nil);
}

# ---- Host-Side PDF Rendering ----

# Render a single PDF page via host-side pdftoppm.
# Streams PDF from file path through stdin, reads PNG from stdout.
hostrender(path: string, page: int): (ref Draw->Image, string)
{
	if(imgload == nil)
		return (nil, "image loader not available");

	# Bind #C device if needed
	if(sys->stat("/cmd/clone").t0 == -1)
		if(sys->bind("#C", "/", Sys->MBEFORE) < 0)
			return (nil, sys->sprint("cannot bind #C: %r"));

	cfd := sys->open("/cmd/clone", sys->ORDWR);
	if(cfd == nil)
		return (nil, sys->sprint("cannot open /cmd/clone: %r"));

	buf := array[32] of byte;
	n := sys->read(cfd, buf, len buf);
	if(n <= 0)
		return (nil, sys->sprint("cannot read /cmd/clone: %r"));

	dir := "/cmd/" + string buf[0:n];

	# pdftoppm reads PDF from stdin, writes PNG to stdout
	pgstr := string page;
	dpistr := string curdpi;
	cmd := "exec /bin/sh -c '"
		+ "PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:$PATH; "
		+ "pdftoppm -png -f " + pgstr + " -l " + pgstr
		+ " -r " + dpistr + " -singlefile'";

	if(sys->fprint(cfd, "%s", cmd) < 0)
		return (nil, sys->sprint("cannot exec: %r"));

	# Open data fd for writing (stdin) and reading (stdout) separately
	tocmd := sys->open(dir+"/data", sys->OWRITE);
	if(tocmd == nil)
		return (nil, sys->sprint("cannot open data for write: %r"));

	fromcmd := sys->open(dir+"/data", sys->OREAD);
	if(fromcmd == nil){
		tocmd = nil;
		return (nil, sys->sprint("cannot open data for read: %r"));
	}

	# Spawn writer to stream PDF file to stdin
	wdone := chan of int;
	spawn pdffilewriter(path, tocmd, wdone);
	tocmd = nil;

	# Read PNG output from stdout
	chunks: list of array of byte;
	totallen := 0;
	readbuf := array[65536] of byte;
	for(;;){
		r := sys->read(fromcmd, readbuf, len readbuf);
		if(r <= 0)
			break;
		chunk := array[r] of byte;
		chunk[0:] = readbuf[0:r];
		chunks = chunk :: chunks;
		totallen += r;
	}
	fromcmd = nil;

	# Wait for writer to finish
	<-wdone;

	# Wait for command exit
	wfd := sys->open(dir+"/wait", Sys->OREAD);
	if(wfd != nil){
		wbuf := array[1024] of byte;
		sys->read(wfd, wbuf, len wbuf);
	}
	cfd = nil;

	if(totallen == 0)
		return (nil, "pdftoppm produced no output");

	# Assemble PNG data
	png := array[totallen] of byte;
	pos := totallen;
	for(; chunks != nil; chunks = tl chunks){
		chunk := hd chunks;
		pos -= len chunk;
		png[pos:] = chunk;
	}
	chunks = nil;

	# Decode PNG → Draw->Image
	(im, ierr) := imgload->readimagedata(png, "page.png");
	png = nil;
	if(im == nil)
		return (nil, "cannot decode PNG: " + ierr);

	return (im, nil);
}

# Stream PDF file to the command's stdin in 8KB chunks.
pdffilewriter(path: string, fd: ref Sys->FD, done: chan of int)
{
	infd := sys->open(path, Sys->OREAD);
	if(infd != nil){
		buf := array[8192] of byte;
		for(;;){
			n := sys->read(infd, buf, len buf);
			if(n <= 0)
				break;
			if(sys->write(fd, buf[:n], n) != n)
				break;
		}
	}
	fd = nil;
	done <-= 1;
}

# Read a PDF file up to maxsize bytes for in-memory parsing.
# Returns nil if file can't be opened or exceeds maxsize.
readpdffile(path: string, maxsize: int): array of byte
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return nil;
	(ok, dir) := sys->fstat(fd);
	if(ok != 0 || int dir.length <= 0)
		return nil;
	fsize := int dir.length;
	if(fsize > maxsize)
		return nil;
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

# Count total pages in the document
countpages(doc: ref PdfDoc): int
{
	root := dictget(doc.trailer.dval, "Root");
	if(root == nil)
		return 0;
	root = resolve(doc, root);
	if(root == nil)
		return 0;
	pages := dictget(root.dval, "Pages");
	if(pages == nil)
		return 0;
	pages = resolve(doc, pages);
	if(pages == nil)
		return 0;
	return countpagenode(doc, pages);
}

countpagenode(doc: ref PdfDoc, node: ref PdfObj): int
{
	if(node == nil)
		return 0;
	typobj := dictget(node.dval, "Type");
	typ := "";
	if(typobj != nil && typobj.kind == Oname)
		typ = typobj.sval;

	if(typ == "Page")
		return 1;

	if(typ == "Pages"){
		count := 0;
		kids := dictget(node.dval, "Kids");
		if(kids != nil && kids.kind == Oarray){
			for(k := kids.aval; k != nil; k = tl k){
				child := resolve(doc, hd k);
				if(child != nil)
					count += countpagenode(doc, child);
			}
		}
		return count;
	}
	return 0;
}

# ---- PDF Parser ----

# Parse PDF file: find xref, parse trailer, build object table
parsepdf(data: array of byte): (ref PdfDoc, string)
{
	if(len data < 20)
		return (nil, "file too small");

	# Verify %PDF- header
	if(data[0] != byte '%' || data[1] != byte 'P' ||
	   data[2] != byte 'D' || data[3] != byte 'F')
		return (nil, "not a PDF file");

	# Find startxref from end of file
	(xrefoff, err) := findstartxref(data);
	if(xrefoff < 0)
		return (nil, "cannot find startxref: " + err);

	# Parse xref table (traditional) or xref stream (PDF 1.5+)
	(xref, nobjs, traileroff, xerr) := parsexref(data, xrefoff);
	if(xref != nil){
		# Traditional xref — parse separate trailer dictionary
		(trailer, nil, terr) := parseobj(data, traileroff);
		if(trailer == nil)
			return (nil, "cannot parse trailer: " + terr);
		doc := ref PdfDoc(data, xref, trailer, nobjs);
		return (doc, nil);
	}

	# Try cross-reference stream (PDF 1.5+)
	trailer: ref PdfObj;
	xserr: string;
	(xref, nobjs, trailer, xserr) = parsexrefstream(data, xrefoff);
	if(xref == nil)
		return (nil, "cannot parse xref: " + xerr + "; xref stream: " + xserr);

	doc := ref PdfDoc(data, xref, trailer, nobjs);
	return (doc, nil);
}

# Find "startxref" near end of file, return the xref offset
findstartxref(data: array of byte): (int, string)
{
	# Search last 1024 bytes for "startxref"
	searchlen := 1024;
	if(searchlen > len data)
		searchlen = len data;
	start := len data - searchlen;

	needle := "startxref";
	pos := -1;
	for(i := start; i <= len data - len needle; i++){
		found := 1;
		for(j := 0; j < len needle; j++){
			if(data[i+j] != byte needle[j]){
				found = 0;
				break;
			}
		}
		if(found){
			pos = i;
			# Don't break — take the last occurrence
		}
	}

	if(pos < 0)
		return (-1, "startxref not found");

	# Skip "startxref" and whitespace, read the offset number
	pos += len needle;
	while(pos < len data && isws(int data[pos]))
		pos++;

	numstr := "";
	while(pos < len data && int data[pos] >= '0' && int data[pos] <= '9'){
		numstr[len numstr] = int data[pos];
		pos++;
	}

	if(len numstr == 0)
		return (-1, "no offset after startxref");

	return (int numstr, nil);
}

# Parse xref table starting at offset
parsexref(data: array of byte, offset: int): (array of ref XrefEntry, int, int, string)
{
	pos := offset;

	# Expect "xref"
	if(pos + 4 > len data)
		return (nil, 0, 0, "truncated xref");

	tag := slicestr(data, pos, 4);
	if(tag != "xref")
		return (nil, 0, 0, "expected 'xref' at offset " + string offset);

	pos += 4;
	pos = skipws(data, pos);

	# Read subsections
	maxobj := 0;
	entries: list of (int, int, array of ref XrefEntry);  # (startobj, count, entries)

	for(;;){
		if(pos >= len data)
			break;

		# Check for "trailer"
		if(pos + 7 <= len data && slicestr(data, pos, 7) == "trailer")
			break;

		# Read start object number and count
		(startobj, p1) := readint(data, pos);
		if(p1 == pos)
			break;
		pos = skipws(data, p1);

		(count, p2) := readint(data, pos);
		if(p2 == pos)
			break;
		pos = skipws(data, p2);

		if(startobj + count > maxobj)
			maxobj = startobj + count;

		# Read entries
		sect := array[count] of ref XrefEntry;
		for(i := 0; i < count; i++){
			(eoff, p3) := readint(data, pos);
			pos = skipws(data, p3);

			(egen, p4) := readint(data, pos);
			pos = skipws(data, p4);

			# Read 'f' or 'n'
			inuse := 0;
			if(pos < len data){
				if(int data[pos] == 'n')
					inuse = 1;
				pos++;
			}
			pos = skipws(data, pos);

			sect[i] = ref XrefEntry(eoff, egen, inuse);
		}
		entries = (startobj, count, sect) :: entries;
	}

	if(maxobj == 0)
		return (nil, 0, 0, "empty xref table");

	# Build flat xref array
	xref := array[maxobj] of ref XrefEntry;
	for(; entries != nil; entries = tl entries){
		(sobj, cnt, sect) := hd entries;
		for(i := 0; i < cnt; i++)
			xref[sobj + i] = sect[i];
	}

	# Find trailer position
	trailerpos := pos;
	if(trailerpos + 7 <= len data && slicestr(data, trailerpos, 7) == "trailer")
		trailerpos += 7;
	trailerpos = skipws(data, trailerpos);

	return (xref, maxobj, trailerpos, nil);
}

# Parse a cross-reference stream (PDF 1.5+).
# The xref stream is an indirect object whose dictionary contains
# /Type /XRef, /Size, /W, and optionally /Index.  The stream data
# (after decompression) encodes the xref entries.  The dictionary
# itself serves as the trailer (/Root, /Info, etc.).
parsexrefstream(data: array of byte, offset: int): (array of ref XrefEntry, int, ref PdfObj, string)
{
	pos := offset;

	# Skip "N N obj" header
	(nil, p1) := readint(data, pos);
	if(p1 == pos)
		return (nil, 0, nil, "expected object number");
	pos = skipws(data, p1);

	(nil, p2) := readint(data, pos);
	if(p2 == pos)
		return (nil, 0, nil, "expected generation number");
	pos = skipws(data, p2);

	if(pos + 3 > len data || slicestr(data, pos, 3) != "obj")
		return (nil, 0, nil, "expected 'obj' keyword");
	pos += 3;
	pos = skipws(data, pos);

	# Parse the stream object (dict + stream data)
	(obj, nil, perr) := parseobj(data, pos);
	if(obj == nil)
		return (nil, 0, nil, "cannot parse xref stream object: " + perr);
	if(obj.kind != Ostream)
		return (nil, 0, nil, "xref stream object is not a stream");

	# Verify /Type /XRef
	typeobj := dictget(obj.dval, "Type");
	if(typeobj == nil || typeobj.kind != Oname || typeobj.sval != "XRef")
		return (nil, 0, nil, "/Type is not /XRef");

	# Get /Size (total number of objects)
	size := dictgetint(obj.dval, "Size");
	if(size <= 0)
		return (nil, 0, nil, "missing or invalid /Size");

	# Get /W array (field widths)
	wobj := dictget(obj.dval, "W");
	if(wobj == nil || wobj.kind != Oarray)
		return (nil, 0, nil, "missing /W array");
	wvals: list of int;
	for(wl := wobj.aval; wl != nil; wl = tl wl){
		w := hd wl;
		if(w.kind == Oint)
			wvals = w.ival :: wvals;
		else
			wvals = 0 :: wvals;
	}
	# Reverse to get correct order
	w := array[3] of {* => 0};
	i := 0;
	for(wr := wvals; wr != nil; wr = tl wr)
		i++;
	if(i != 3)
		return (nil, 0, nil, sys->sprint("/W has %d entries, expected 3", i));
	i = 0;
	for(wr = wvals; wr != nil; wr = tl wr){
		w[2 - i] = hd wr;
		i++;
	}

	entrysize := w[0] + w[1] + w[2];
	if(entrysize <= 0)
		return (nil, 0, nil, "invalid /W field widths");

	# Get /Index array (optional — defaults to [0 Size])
	idxobj := dictget(obj.dval, "Index");
	subsections: list of (int, int);
	if(idxobj != nil && idxobj.kind == Oarray){
		il := idxobj.aval;
		for(;;){
			if(il == nil)
				break;
			sobj := hd il;
			il = tl il;
			if(il == nil)
				break;
			cobj := hd il;
			il = tl il;
			sv := 0;
			cv := 0;
			if(sobj.kind == Oint)
				sv = sobj.ival;
			if(cobj.kind == Oint)
				cv = cobj.ival;
			subsections = (sv, cv) :: subsections;
		}
		# Reverse
		rev: list of (int, int);
		for(; subsections != nil; subsections = tl subsections)
			rev = (hd subsections) :: rev;
		subsections = rev;
	} else
		subsections = (0, size) :: nil;

	# Decompress stream data
	(sdata, derr) := decompressstream(obj);
	if(sdata == nil)
		return (nil, 0, nil, "cannot decompress xref stream: " + derr);

	# Parse entries from decompressed stream
	xref := array[size] of ref XrefEntry;
	dpos := 0;
	for(sl := subsections; sl != nil; sl = tl sl){
		(startobj, count) := hd sl;
		for(j := 0; j < count; j++){
			if(dpos + entrysize > len sdata)
				break;

			# Read field values as big-endian integers
			f0 := readfield(sdata, dpos, w[0]);
			dpos += w[0];
			f1 := readfield(sdata, dpos, w[1]);
			dpos += w[1];
			f2 := readfield(sdata, dpos, w[2]);
			dpos += w[2];

			# Default type is 1 if w[0]==0
			ftype := f0;
			if(w[0] == 0)
				ftype = 1;

			objnum := startobj + j;
			if(objnum >= size)
				break;

			case ftype {
			0 =>
				# Free entry
				xref[objnum] = ref XrefEntry(0, f2, 0);
			1 =>
				# In-use: f1=offset, f2=generation
				xref[objnum] = ref XrefEntry(f1, f2, 1);
			2 =>
				# Compressed in object stream:
				# offset=stream obj#, gen=index within stream
				# inuse=2 flags this as a compressed entry
				xref[objnum] = ref XrefEntry(f1, f2, 2);
			* =>
				xref[objnum] = ref XrefEntry(0, 0, 0);
			}
		}
	}

	# The xref stream dictionary is the trailer
	trailer := ref PdfObj(Odict, 0, 0.0, nil, nil, obj.dval, nil);
	return (xref, size, trailer, nil);
}

# Read a big-endian integer field of the given width from data.
readfield(data: array of byte, pos, width: int): int
{
	v := 0;
	for(i := 0; i < width && pos + i < len data; i++)
		v = (v << 8) | int data[pos + i];
	return v;
}

# Parse a PDF object at the given offset.
# Returns (object, new position, error)
parseobj(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	if(pos >= len data)
		return (nil, pos, "unexpected end of data");

	pos = skipws(data, pos);
	if(pos >= len data)
		return (nil, pos, "unexpected end of data");

	c := int data[pos];

	# Dictionary or stream
	if(c == '<' && pos+1 < len data && int data[pos+1] == '<')
		return parsedict(data, pos);

	# Hex string
	if(c == '<')
		return parsehexstring(data, pos);

	# Literal string
	if(c == '(')
		return parselitstring(data, pos);

	# Name
	if(c == '/')
		return parsename(data, pos);

	# Array
	if(c == '[')
		return parsearray(data, pos);

	# Boolean
	if(c == 't' && pos+4 <= len data && slicestr(data, pos, 4) == "true")
		return (ref PdfObj(Obool, 1, 0.0, nil, nil, nil, nil), pos+4, nil);
	if(c == 'f' && pos+5 <= len data && slicestr(data, pos, 5) == "false")
		return (ref PdfObj(Obool, 0, 0.0, nil, nil, nil, nil), pos+5, nil);

	# Null
	if(c == 'n' && pos+4 <= len data && slicestr(data, pos, 4) == "null")
		return (ref PdfObj(Onull, 0, 0.0, nil, nil, nil, nil), pos+4, nil);

	# Number (may be indirect reference: N N R)
	if((c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.')
		return parsenumber(data, pos);

	return (nil, pos, "unexpected character: " + string c);
}

# Parse a dictionary << ... >>
parsedict(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	pos += 2;  # skip <<
	pos = skipws(data, pos);

	entries: list of ref DictEntry;

	while(pos < len data){
		pos = skipws(data, pos);
		if(pos >= len data)
			break;

		# Check for >>
		if(int data[pos] == '>' && pos+1 < len data && int data[pos+1] == '>'){
			pos += 2;
			break;
		}

		# Key must be a name
		if(int data[pos] != '/')
			return (nil, pos, "expected name key in dict");

		(keyobj, p1, kerr) := parsename(data, pos);
		if(keyobj == nil)
			return (nil, p1, kerr);
		pos = p1;

		# Value
		(valobj, p2, verr) := parseobj(data, pos);
		if(valobj == nil)
			return (nil, p2, verr);
		pos = p2;

		entries = ref DictEntry(keyobj.sval, valobj) :: entries;
	}

	# Check if followed by "stream"
	spos := skipws(data, pos);
	if(spos + 6 <= len data && slicestr(data, spos, 6) == "stream"){
		return parsestreamdata(data, spos + 6, entries);
	}

	return (ref PdfObj(Odict, 0, 0.0, nil, nil, entries, nil), pos, nil);
}

# Parse stream data after "stream" keyword
parsestreamdata(data: array of byte, pos: int,
                entries: list of ref DictEntry): (ref PdfObj, int, string)
{
	# Skip to start of stream data (after \r\n or \n)
	if(pos < len data && int data[pos] == '\r')
		pos++;
	if(pos < len data && int data[pos] == '\n')
		pos++;

	# Find stream length from dictionary
	slen := dictgetint(entries, "Length");
	if(slen <= 0){
		# Try to find "endstream" by scanning
		(slen, pos) = findendstream(data, pos);
		if(slen < 0)
			return (nil, pos, "cannot determine stream length");
	}

	if(pos + slen > len data)
		slen = len data - pos;

	streamdata := array[slen] of byte;
	streamdata[0:] = data[pos:pos+slen];

	pos += slen;

	# Skip "endstream"
	pos = skipws(data, pos);
	if(pos + 9 <= len data && slicestr(data, pos, 9) == "endstream")
		pos += 9;

	obj := ref PdfObj(Ostream, 0, 0.0, nil, nil, entries, streamdata);
	return (obj, pos, nil);
}

# Find endstream marker by scanning
findendstream(data: array of byte, start: int): (int, int)
{
	needle := "endstream";
	for(i := start; i <= len data - len needle; i++){
		found := 1;
		for(j := 0; j < len needle; j++){
			if(data[i+j] != byte needle[j]){
				found = 0;
				break;
			}
		}
		if(found)
			return (i - start, start);
	}
	return (-1, start);
}

# Parse hex string <...>
parsehexstring(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	pos++;  # skip <
	s := "";
	nibble := -1;
	while(pos < len data){
		c := int data[pos];
		pos++;
		if(c == '>')
			break;
		if(isws(c))
			continue;

		v := hexval(c);
		if(v < 0)
			continue;

		if(nibble < 0)
			nibble = v;
		else {
			s[len s] = nibble * 16 + v;
			nibble = -1;
		}
	}
	if(nibble >= 0)
		s[len s] = nibble * 16;

	return (ref PdfObj(Ostring, 0, 0.0, s, nil, nil, nil), pos, nil);
}

# Parse literal string (...)
parselitstring(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	pos++;  # skip (
	depth := 1;
	s := "";

	while(pos < len data && depth > 0){
		c := int data[pos];
		pos++;

		case c {
		'(' =>
			depth++;
			s[len s] = c;
		')' =>
			depth--;
			if(depth > 0)
				s[len s] = c;
		'\\' =>
			if(pos < len data){
				ec := int data[pos];
				pos++;
				case ec {
				'n' => s[len s] = '\n';
				'r' => s[len s] = '\r';
				't' => s[len s] = '\t';
				'b' => s[len s] = '\b';
				'f' => s[len s] = 16r0c;
				'(' => s[len s] = '(';
				')' => s[len s] = ')';
				'\\' => s[len s] = '\\';
				'0' to '7' =>
					# Octal escape
					oct := ec - '0';
					if(pos < len data && int data[pos] >= '0' && int data[pos] <= '7'){
						oct = oct * 8 + (int data[pos] - '0');
						pos++;
					}
					if(pos < len data && int data[pos] >= '0' && int data[pos] <= '7'){
						oct = oct * 8 + (int data[pos] - '0');
						pos++;
					}
					s[len s] = oct;
				* =>
					s[len s] = ec;
				}
			}
		* =>
			s[len s] = c;
		}
	}

	return (ref PdfObj(Ostring, 0, 0.0, s, nil, nil, nil), pos, nil);
}

# Parse a /Name
parsename(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	pos++;  # skip /
	name := "";

	while(pos < len data){
		c := int data[pos];
		# Name ends at whitespace, delimiters
		if(isws(c) || c == '/' || c == '<' || c == '>' ||
		   c == '[' || c == ']' || c == '(' || c == ')' ||
		   c == '{' || c == '}' || c == '%')
			break;

		if(c == '#' && pos+2 < len data){
			# Hex escape in name
			h1 := hexval(int data[pos+1]);
			h2 := hexval(int data[pos+2]);
			if(h1 >= 0 && h2 >= 0){
				name[len name] = h1 * 16 + h2;
				pos += 3;
				continue;
			}
		}

		name[len name] = c;
		pos++;
	}

	return (ref PdfObj(Oname, 0, 0.0, name, nil, nil, nil), pos, nil);
}

# Parse array [...]
parsearray(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	pos++;  # skip [
	pos = skipws(data, pos);

	items: list of ref PdfObj;
	while(pos < len data){
		pos = skipws(data, pos);
		if(pos >= len data)
			break;
		if(int data[pos] == ']'){
			pos++;
			break;
		}

		(obj, p, err) := parseobj(data, pos);
		if(obj == nil)
			return (nil, p, err);
		items = obj :: items;
		pos = p;
	}

	# Reverse
	rev: list of ref PdfObj;
	for(; items != nil; items = tl items)
		rev = hd items :: rev;

	return (ref PdfObj(Oarray, 0, 0.0, nil, rev, nil, nil), pos, nil);
}

# Parse number, possibly followed by N R (indirect reference)
parsenumber(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	# Read the number string
	numstr := "";
	isreal := 0;
	start := pos;

	if(pos < len data && (int data[pos] == '-' || int data[pos] == '+')){
		numstr[len numstr] = int data[pos];
		pos++;
	}

	while(pos < len data){
		c := int data[pos];
		if(c >= '0' && c <= '9'){
			numstr[len numstr] = c;
			pos++;
		} else if(c == '.' && !isreal){
			isreal = 1;
			numstr[len numstr] = c;
			pos++;
		} else
			break;
	}

	if(len numstr == 0)
		return (nil, start, "expected number");

	if(isreal)
		return (ref PdfObj(Oreal, 0, real numstr, nil, nil, nil, nil), pos, nil);

	num := int numstr;

	# Check for "N N R" (indirect reference)
	svpos := pos;
	pos = skipws(data, pos);
	if(pos < len data && int data[pos] >= '0' && int data[pos] <= '9'){
		genstr := "";
		while(pos < len data && int data[pos] >= '0' && int data[pos] <= '9'){
			genstr[len genstr] = int data[pos];
			pos++;
		}
		pos = skipws(data, pos);
		if(pos < len data && int data[pos] == 'R'){
			pos++;
			gen := int genstr;
			return (ref PdfObj(Oref, num, real gen, nil, nil, nil, nil), pos, nil);
		}
	}

	# Not a reference, return the integer
	return (ref PdfObj(Oint, num, 0.0, nil, nil, nil, nil), svpos, nil);
}

# ---- Object Resolution ----

# Resolve an indirect reference to its object
resolve(doc: ref PdfDoc, obj: ref PdfObj): ref PdfObj
{
	if(obj == nil)
		return nil;
	if(obj.kind != Oref)
		return obj;

	objnum := obj.ival;
	if(objnum < 0 || objnum >= doc.nobjs)
		return nil;

	entry := doc.xref[objnum];
	if(entry == nil || entry.inuse == 0)
		return nil;

	if(entry.inuse == 2)
		return resolveobjstm(doc, entry.offset, entry.gen);

	# Type 1: parse the object at its direct offset
	offset := entry.offset;
	if(offset >= len doc.data)
		return nil;

	# Skip "N N obj"
	pos := offset;
	(nil, p1) := readint(doc.data, pos);  # obj number
	pos = skipws(doc.data, p1);
	(nil, p2) := readint(doc.data, pos);  # generation
	pos = skipws(doc.data, p2);

	# Skip "obj" keyword
	if(pos + 3 <= len doc.data && slicestr(doc.data, pos, 3) == "obj")
		pos += 3;
	pos = skipws(doc.data, pos);

	(parsed, nil, nil) := parseobj(doc.data, pos);
	return parsed;
}

# Resolve an object stored in a compressed object stream.
# stmnum is the object number of the ObjStm, idx is the index within it.
resolveobjstm(doc: ref PdfDoc, stmnum, idx: int): ref PdfObj
{
	if(stmnum < 0 || stmnum >= doc.nobjs)
		return nil;

	# The object stream itself must be a type-1 entry
	stmentry := doc.xref[stmnum];
	if(stmentry == nil || stmentry.inuse != 1)
		return nil;

	# Parse the object stream's indirect object
	offset := stmentry.offset;
	if(offset >= len doc.data)
		return nil;

	pos := offset;
	(nil, p1) := readint(doc.data, pos);
	pos = skipws(doc.data, p1);
	(nil, p2) := readint(doc.data, pos);
	pos = skipws(doc.data, p2);
	if(pos + 3 <= len doc.data && slicestr(doc.data, pos, 3) == "obj")
		pos += 3;
	pos = skipws(doc.data, pos);

	(stmobj, nil, nil) := parseobj(doc.data, pos);
	if(stmobj == nil || stmobj.kind != Ostream)
		return nil;

	# Get /N (number of objects) and /First (byte offset to first object)
	n := dictgetint(stmobj.dval, "N");
	first := dictgetint(stmobj.dval, "First");
	if(n <= 0 || first <= 0 || idx >= n)
		return nil;

	# Decompress the stream
	(sdata, derr) := decompressstream(stmobj);
	if(sdata == nil || derr != nil)
		return nil;

	# Parse the header: N pairs of (objnum offset) integers
	# These offsets are relative to /First
	spos := 0;
	offsets := array[n] of int;
	for(i := 0; i < n; i++){
		spos = skipwsbytes(sdata, spos);
		(nil, sp1) := readint(sdata, spos);  # object number
		spos = skipwsbytes(sdata, sp1);
		(ooff, sp2) := readint(sdata, spos);  # offset relative to /First
		spos = sp2;
		offsets[i] = first + ooff;
	}

	if(idx >= n)
		return nil;

	# Parse the object at the computed offset
	opos := offsets[idx];
	if(opos >= len sdata)
		return nil;

	(parsed, nil, nil) := parseobj(sdata, opos);
	return parsed;
}

# Skip whitespace in a byte array (same as skipws but for decompressed data)
skipwsbytes(data: array of byte, pos: int): int
{
	while(pos < len data){
		c := int data[pos];
		if(c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == 0)
			pos++;
		else
			break;
	}
	return pos;
}

# ---- Stream Decompression ----

# Decompress a stream if needed
decompressstream(obj: ref PdfObj): (array of byte, string)
{
	if(obj == nil || obj.kind != Ostream)
		return (nil, "not a stream");

	raw := obj.stream;
	if(raw == nil)
		return (nil, "empty stream");

	# Check Filter
	filterobj := dictget(obj.dval, "Filter");
	if(filterobj == nil)
		return (raw, nil);  # Not compressed

	filtername := "";
	if(filterobj.kind == Oname)
		filtername = filterobj.sval;
	else if(filterobj.kind == Oarray && filterobj.aval != nil){
		first := hd filterobj.aval;
		if(first != nil && first.kind == Oname)
			filtername = first.sval;
	}

	if(filtername == "FlateDecode" || filtername == "Fl")
		return inflate(raw);
	if(filtername == "ASCIIHexDecode")
		return asciihexdecode(raw);

	# Unsupported filter — return raw
	return (raw, nil);
}

# Inflate (zlib decompress) data
inflate(data: array of byte): (array of byte, string)
{
	filtermod = load Filter Filter->INFLATEPATH;
	if(filtermod == nil)
		return (nil, sys->sprint("cannot load inflate: %r"));

	filtermod->init();
	rqchan := filtermod->start("z");  # zlib mode

	# Wait for Start
	rq := <-rqchan;
	pick r := rq {
	Start =>
		;
	* =>
		return (nil, "inflate: unexpected initial message");
	}

	# Feed data and collect output
	result: list of array of byte;
	resultlen := 0;
	inpos := 0;
	done := 0;

	while(!done){
		rq = <-rqchan;
		pick r := rq {
		Fill =>
			# Copy input data into the buffer
			n := len data - inpos;
			if(n > len r.buf)
				n = len r.buf;
			if(n > 0)
				r.buf[0:] = data[inpos:inpos+n];
			inpos += n;
			r.reply <-= n;
		Result =>
			# Collect output
			chunk := array[len r.buf] of byte;
			chunk[0:] = r.buf;
			result = chunk :: result;
			resultlen += len chunk;
			r.reply <-= 0;
		Info =>
			;  # Ignore info messages (gzip metadata)
		Finished =>
			done = 1;
		Error =>
			return (nil, "inflate error: " + r.e);
		* =>
			done = 1;
		}
	}

	# Assemble result
	out := array[resultlen] of byte;
	pos := resultlen;
	for(; result != nil; result = tl result){
		chunk := hd result;
		pos -= len chunk;
		out[pos:] = chunk;
	}

	return (out, nil);
}

# ASCII hex decode
asciihexdecode(data: array of byte): (array of byte, string)
{
	out := array[len data / 2 + 1] of byte;
	n := 0;
	nibble := -1;

	for(i := 0; i < len data; i++){
		c := int data[i];
		if(c == '>')
			break;
		if(isws(c))
			continue;
		v := hexval(c);
		if(v < 0)
			continue;
		if(nibble < 0)
			nibble = v;
		else {
			out[n++] = byte (nibble * 16 + v);
			nibble = -1;
		}
	}
	if(nibble >= 0)
		out[n++] = byte (nibble * 16);

	return (out[0:n], nil);
}

# ---- Text Extraction ----

# Extract text from all pages in the document
extracttext(doc: ref PdfDoc): (string, string)
{
	# Get root catalog
	root := dictget(doc.trailer.dval, "Root");
	if(root == nil)
		return (nil, "no Root in trailer");
	root = resolve(doc, root);
	if(root == nil)
		return (nil, "cannot resolve Root");

	# Get Pages
	pages := dictget(root.dval, "Pages");
	if(pages == nil)
		return (nil, "no Pages in catalog");
	pages = resolve(doc, pages);
	if(pages == nil)
		return (nil, "cannot resolve Pages");

	# Traverse page tree
	text := "";
	pagenum := 0;
	(text, pagenum) = extractpages(doc, pages, text, pagenum);
	if(pagenum == 0)
		return (nil, "no pages found");

	return (text, nil);
}

# Recursively extract text from page tree node
extractpages(doc: ref PdfDoc, node: ref PdfObj,
             text: string, pagenum: int): (string, int)
{
	if(node == nil)
		return (text, pagenum);

	typobj := dictget(node.dval, "Type");
	typ := "";
	if(typobj != nil && typobj.kind == Oname)
		typ = typobj.sval;

	if(typ == "Pages"){
		# Intermediate node — recurse into Kids
		kids := dictget(node.dval, "Kids");
		if(kids != nil && kids.kind == Oarray){
			for(k := kids.aval; k != nil; k = tl k){
				child := resolve(doc, hd k);
				if(child != nil)
					(text, pagenum) = extractpages(doc, child, text, pagenum);
			}
		}
	} else if(typ == "Page"){
		# Leaf page node
		pagenum++;

		if(len text > 0)
			text += "\n\n";
		if(pagenum > 1)
			text += "--- Page " + string pagenum + " ---\n\n";

		# Build font map for this page's ToUnicode CMaps
		fontmap := buildfontmap(doc, node);

		# Get Contents
		contents := dictget(node.dval, "Contents");
		if(contents != nil){
			pagetext := extractpagetext(doc, contents, fontmap);
			if(pagetext != nil)
				text += pagetext;
		}
	}

	return (text, pagenum);
}

# Extract text from a page's Contents (may be a stream or array of streams)
extractpagetext(doc: ref PdfDoc, contents: ref PdfObj, fontmap: list of ref FontMapEntry): string
{
	if(contents == nil)
		return nil;

	contents = resolve(doc, contents);
	if(contents == nil)
		return nil;

	if(contents.kind == Oarray){
		# Array of content streams — concatenate
		text := "";
		for(a := contents.aval; a != nil; a = tl a){
			stream := resolve(doc, hd a);
			if(stream != nil){
				t := extractstreamtext(doc, stream, fontmap);
				if(t != nil)
					text += t;
			}
		}
		return text;
	}

	if(contents.kind == Ostream)
		return extractstreamtext(doc, contents, fontmap);

	return nil;
}

# Extract text from a content stream
extractstreamtext(doc: ref PdfDoc, stream: ref PdfObj, fontmap: list of ref FontMapEntry): string
{
	(data, err) := decompressstream(stream);
	if(data == nil)
		return nil;

	return parsecontentstream(data, fontmap);
}

# Parse PDF content stream operators to extract text
parsecontentstream(data: array of byte, fontmap: list of ref FontMapEntry): string
{
	text := "";
	pos := 0;
	operands: list of string;  # stack of operand strings
	curfont: ref FontMapEntry;

	while(pos < len data){
		pos = skipws(data, pos);
		if(pos >= len data)
			break;

		c := int data[pos];

		# String operand (...)
		if(c == '('){
			(s, newpos) := readlitstr(data, pos);
			operands = s :: operands;
			pos = newpos;
			continue;
		}

		# Hex string operand <...>
		if(c == '<' && (pos+1 >= len data || int data[pos+1] != '<')){
			(s, newpos) := readhexstr(data, pos);
			operands = s :: operands;
			pos = newpos;
			continue;
		}

		# Array operand [...] (for TJ)
		if(c == '['){
			(s, newpos) := readtjarray(data, pos, curfont);
			operands = s :: operands;
			pos = newpos;
			continue;
		}

		# Skip dict << >> inline images, etc.
		if(c == '<' && pos+1 < len data && int data[pos+1] == '<'){
			pos = skipdict(data, pos);
			continue;
		}

		# Name operand /Foo (for Tf font selection)
		if(c == '/'){
			(tok, newpos) := readcsname(data, pos);
			operands = tok :: operands;
			pos = newpos;
			continue;
		}

		# Number or other token
		if((c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.'){
			(tok, newpos) := readtoken(data, pos);
			operands = tok :: operands;
			pos = newpos;
			continue;
		}

		# Operator (alphabetic)
		if((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		   c == '\'' || c == '"' || c == '*'){
			(op, newpos) := readtoken(data, pos);
			pos = newpos;

			# Text-showing operators
			case op {
			"Tj" =>
				# Show string: (string) Tj
				if(operands != nil){
					s := hd operands;
					if(curfont != nil)
						s = decodecidstr(s, curfont);
					text += cleanpdftext(s);
				}
			"TJ" =>
				# Show strings from array: [...] TJ
				# CID decoding already done per-fragment in readtjarray
				if(operands != nil)
					text += cleanpdftext(hd operands);
			"'" =>
				# Move to next line and show: (string) '
				if(operands != nil){
					s := hd operands;
					if(curfont != nil)
						s = decodecidstr(s, curfont);
					text += "\n" + cleanpdftext(s);
				}
			"\"" =>
				# Set spacing, move, show: aw ac (string) "
				if(operands != nil){
					s := hd operands;
					if(curfont != nil)
						s = decodecidstr(s, curfont);
					text += "\n" + cleanpdftext(s);
				}
			"Td" or "TD" =>
				# Text positioning — check for line breaks vs word spacing
				# operands (reversed): ty, tx
				if(operands != nil && tl operands != nil){
					ty := real (hd operands);
					tx := real (hd tl operands);
					if(ty < -1.5 || ty > 1.5)
						text += "\n";
					else if(tx > 5.0)
						text += " ";
					# Small tx = character advance, no space needed
				}
			"Tm" =>
				# Text matrix — position reset (new text block)
				;
			"T*" =>
				# Move to start of next line
				text += "\n";
			"BT" =>
				# Begin text object
				;
			"ET" =>
				# End text object — don't insert space; positioning
				# operators (Td/Tm) handle whitespace
				;
			"Tf" =>
				# Set font — operands are: /FontName size Tf
				if(operands != nil && tl operands != nil)
					curfont = fontmaplookup(fontmap, hd tl operands);
			"cm" or "q" or "Q" or "re" or "W" or "n" or
			"m" or "l" or "c" or "h" or "f" or "S" or "B" or
			"gs" or "cs" or "CS" or "sc" or "SC" or "rg" or "RG" or
			"g" or "G" or "k" or "K" or "Do" or "w" or "J" or "j" or
			"M" or "d" or "ri" or "i" or "BDC" or "EMC" or
			"BMC" or "MP" or "DP" =>
				# Graphics/state operators — skip
				;
			"BI" =>
				# Begin inline image — skip to EI
				pos = skipinlineimage(data, pos);
			}

			operands = nil;
			continue;
		}

		# Comment
		if(c == '%'){
			while(pos < len data && int data[pos] != '\n')
				pos++;
			continue;
		}

		# Skip unrecognized
		pos++;
	}

	return text;
}

# Read a literal string (...) from content stream
readlitstr(data: array of byte, pos: int): (string, int)
{
	pos++;  # skip (
	depth := 1;
	s := "";

	while(pos < len data && depth > 0){
		c := int data[pos];
		pos++;
		case c {
		'(' =>
			depth++;
			s[len s] = c;
		')' =>
			depth--;
			if(depth > 0)
				s[len s] = c;
		'\\' =>
			if(pos < len data){
				ec := int data[pos];
				pos++;
				case ec {
				'n' => s[len s] = '\n';
				'r' => s[len s] = '\r';
				't' => s[len s] = '\t';
				'(' => s[len s] = '(';
				')' => s[len s] = ')';
				'\\' => s[len s] = '\\';
				'0' to '7' =>
					oct := ec - '0';
					if(pos < len data && int data[pos] >= '0' && int data[pos] <= '7'){
						oct = oct * 8 + (int data[pos] - '0');
						pos++;
					}
					if(pos < len data && int data[pos] >= '0' && int data[pos] <= '7'){
						oct = oct * 8 + (int data[pos] - '0');
						pos++;
					}
					s[len s] = oct;
				* =>
					s[len s] = ec;
				}
			}
		* =>
			s[len s] = c;
		}
	}
	return (s, pos);
}

# Read a hex string <...> from content stream
readhexstr(data: array of byte, pos: int): (string, int)
{
	pos++;  # skip <
	s := "";
	nibble := -1;

	while(pos < len data){
		c := int data[pos];
		pos++;
		if(c == '>')
			break;
		if(isws(c))
			continue;
		v := hexval(c);
		if(v < 0)
			continue;
		if(nibble < 0)
			nibble = v;
		else {
			s[len s] = nibble * 16 + v;
			nibble = -1;
		}
	}
	if(nibble >= 0)
		s[len s] = nibble * 16;

	return (s, pos);
}

# Read TJ array: [ (string1) num (string2) num ... ]
# Extract text from string elements, ignore numbers.
# Each string fragment is decoded through curfont's CMap individually.
readtjarray(data: array of byte, pos: int, curfont: ref FontMapEntry): (string, int)
{
	pos++;  # skip [
	s := "";

	while(pos < len data){
		pos = skipws(data, pos);
		if(pos >= len data)
			break;
		c := int data[pos];

		if(c == ']'){
			pos++;
			break;
		}

		if(c == '('){
			(substr, newpos) := readlitstr(data, pos);
			if(curfont != nil)
				substr = decodecidstr(substr, curfont);
			s += substr;
			pos = newpos;
			continue;
		}

		if(c == '<'){
			(substr, newpos) := readhexstr(data, pos);
			if(curfont != nil)
				substr = decodecidstr(substr, curfont);
			s += substr;
			pos = newpos;
			continue;
		}

		# Skip numbers (kerning adjustments)
		# Large negative numbers indicate word spacing
		if((c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.'){
			numstr := "";
			while(pos < len data){
				nc := int data[pos];
				if((nc >= '0' && nc <= '9') || nc == '-' || nc == '+' || nc == '.')
					numstr[len numstr] = nc;
				else
					break;
				pos++;
			}
			# Large negative kerning = word space
			if(len numstr > 0){
				kern := real numstr;
				if(kern < -100.0)
					s += " ";
			}
			continue;
		}

		pos++;
	}

	return (s, pos);
}

# Read a token (word)
readtoken(data: array of byte, pos: int): (string, int)
{
	tok := "";
	while(pos < len data){
		c := int data[pos];
		if(isws(c) || c == '(' || c == ')' || c == '<' || c == '>' ||
		   c == '[' || c == ']' || c == '{' || c == '}' || c == '/' || c == '%')
			break;
		tok[len tok] = c;
		pos++;
	}
	return (tok, pos);
}

# Skip an inline image (BI ... ID ... EI)
skipinlineimage(data: array of byte, pos: int): int
{
	# Find "ID" marking start of image data
	while(pos < len data - 1){
		if(int data[pos] == 'I' && int data[pos+1] == 'D'){
			pos += 2;
			break;
		}
		pos++;
	}

	# Find "EI" marking end of image data
	while(pos < len data - 1){
		if(int data[pos] == 'E' && int data[pos+1] == 'I'){
			# Make sure preceded by whitespace
			if(pos > 0 && isws(int data[pos-1])){
				pos += 2;
				return pos;
			}
		}
		pos++;
	}
	return pos;
}

# Skip a dictionary << ... >>
skipdict(data: array of byte, pos: int): int
{
	pos += 2;  # skip <<
	depth := 1;
	while(pos < len data - 1 && depth > 0){
		if(int data[pos] == '<' && int data[pos+1] == '<'){
			depth++;
			pos += 2;
		} else if(int data[pos] == '>' && int data[pos+1] == '>'){
			depth--;
			pos += 2;
		} else
			pos++;
	}
	return pos;
}

# Clean up extracted PDF text
cleanpdftext(s: string): string
{
	if(s == nil)
		return nil;

	# Replace control characters with spaces, collapse whitespace
	out := "";
	lastspace := 0;
	for(i := 0; i < len s; i++){
		c := s[i];
		if(c == '\r' || c == '\n'){
			if(!lastspace){
				out[len out] = '\n';
				lastspace = 1;
			}
		} else if(c < ' '){
			if(!lastspace){
				out[len out] = ' ';
				lastspace = 1;
			}
		} else {
			out[len out] = c;
			lastspace = 0;
		}
	}
	return out;
}

# ---- ToUnicode CMap Support ----

# Parse a ToUnicode CMap string.
# Returns (twobyte, entries) where twobyte=1 if codespace uses 2-byte codes.
parsecmap(text: string): (int, list of ref CMapEntry)
{
	entries: list of ref CMapEntry;
	twobyte := 0;
	pos := 0;
	tlen := len text;

	while(pos < tlen){
		# Find next keyword
		if(pos + 19 <= tlen && text[pos:pos+19] == "begincodespacerange"){
			# Check codespace to detect 2-byte encoding
			pos += 19;
			while(pos < tlen && text[pos] != '<')
				pos++;
			if(pos < tlen){
				(nil, np) := parsecmaphex(text, pos);
				# Count hex digits between < and >
				hstart := pos + 1;
				ndigits := 0;
				for(h := hstart; h < tlen && text[h] != '>'; h++)
					ndigits++;
				if(ndigits >= 4)
					twobyte = 1;
				pos = np;
			}
			continue;
		}

		if(pos + 11 <= tlen && text[pos:pos+11] == "beginbfchar"){
			# Single CID-to-Unicode mappings: <XXXX> <YYYY>
			pos += 11;
			for(;;){
				while(pos < tlen && (text[pos] == ' ' || text[pos] == '\n' || text[pos] == '\r' || text[pos] == '\t'))
					pos++;
				if(pos + 9 <= tlen && text[pos:pos+9] == "endbfchar")
					break;
				if(pos >= tlen)
					break;
				if(text[pos] != '<'){
					pos++;
					continue;
				}
				(cid, np1) := parsecmaphex(text, pos);
				pos = np1;
				while(pos < tlen && text[pos] != '<')
					pos++;
				if(pos >= tlen)
					break;
				(uni, np2) := parsecmaphex(text, pos);
				pos = np2;
				entries = ref CMapEntry(cid, cid, uni) :: entries;
			}
			continue;
		}

		if(pos + 12 <= tlen && text[pos:pos+12] == "beginbfrange"){
			# Range mappings: <XXXX> <YYYY> <ZZZZ>
			pos += 12;
			for(;;){
				while(pos < tlen && (text[pos] == ' ' || text[pos] == '\n' || text[pos] == '\r' || text[pos] == '\t'))
					pos++;
				if(pos + 10 <= tlen && text[pos:pos+10] == "endbfrange")
					break;
				if(pos >= tlen)
					break;
				if(text[pos] != '<'){
					pos++;
					continue;
				}
				(lo, np1) := parsecmaphex(text, pos);
				pos = np1;
				while(pos < tlen && text[pos] != '<')
					pos++;
				if(pos >= tlen)
					break;
				(hi, np2) := parsecmaphex(text, pos);
				pos = np2;
				while(pos < tlen && text[pos] != '<')
					pos++;
				if(pos >= tlen)
					break;
				(uni, np3) := parsecmaphex(text, pos);
				pos = np3;
				entries = ref CMapEntry(lo, hi, uni) :: entries;
			}
			continue;
		}

		pos++;
	}

	return (twobyte, entries);
}

# Read <XXXX> hex value from CMap text at pos, return (value, newpos)
parsecmaphex(s: string, pos: int): (int, int)
{
	slen := len s;
	if(pos >= slen || s[pos] != '<')
		return (0, pos);
	pos++;  # skip <

	val := 0;
	while(pos < slen && s[pos] != '>'){
		c := s[pos];
		pos++;
		v := hexval(c);
		if(v >= 0)
			val = (val << 4) | v;
	}
	if(pos < slen && s[pos] == '>')
		pos++;  # skip >
	return (val, pos);
}

# Build font map from page's /Resources/Font dictionary.
# For each font, resolve → get /ToUnicode stream → decompress → parsecmap.
buildfontmap(doc: ref PdfDoc, page: ref PdfObj): list of ref FontMapEntry
{
	if(page == nil)
		return nil;

	resources := dictget(page.dval, "Resources");
	if(resources == nil)
		return nil;
	resources = resolve(doc, resources);
	if(resources == nil)
		return nil;

	fonts := dictget(resources.dval, "Font");
	if(fonts == nil)
		return nil;
	fonts = resolve(doc, fonts);
	if(fonts == nil || (fonts.kind != Odict && fonts.kind != Ostream))
		return nil;

	fontmap: list of ref FontMapEntry;

	for(fl := fonts.dval; fl != nil; fl = tl fl){
		de := hd fl;
		fontname := de.key;
		fontobj := resolve(doc, de.val);
		if(fontobj == nil)
			continue;

		twobyte := 0;
		fentries: list of ref CMapEntry;

		# Check /Encoding for Identity-H
		enc := dictget(fontobj.dval, "Encoding");
		if(enc != nil){
			enc = resolve(doc, enc);
			if(enc != nil && enc.kind == Oname && enc.sval == "Identity-H")
				twobyte = 1;
		}

		# Get /ToUnicode stream
		tounicode := dictget(fontobj.dval, "ToUnicode");
		if(tounicode != nil){
			tounicode = resolve(doc, tounicode);
			if(tounicode != nil && tounicode.kind == Ostream){
				(cmapdata, derr) := decompressstream(tounicode);
				if(cmapdata != nil && derr == nil){
					cmaptext := "";
					for(i := 0; i < len cmapdata; i++)
						cmaptext[len cmaptext] = int cmapdata[i];
					(tb, ent) := parsecmap(cmaptext);
					if(tb)
						twobyte = 1;
					fentries = ent;
				}
			}
		}

		if(fentries != nil || twobyte)
			fontmap = ref FontMapEntry(fontname, twobyte, fentries) :: fontmap;
	}

	return fontmap;
}

# Look up a CID in a CMap entry list. Returns Unicode codepoint.
cmaplookup(entries: list of ref CMapEntry, cid: int): int
{
	for(; entries != nil; entries = tl entries){
		e := hd entries;
		if(cid >= e.lo && cid <= e.hi)
			return e.unicode + (cid - e.lo);
	}
	return cid;  # identity fallback
}

# Decode a CID string through a font's CMap.
# If twobyte: read pairs of chars as big-endian 2-byte CIDs, map each.
decodecidstr(s: string, fm: ref FontMapEntry): string
{
	if(fm == nil || !fm.twobyte)
		return s;

	out := "";
	slen := len s;
	i := 0;
	while(i + 1 < slen){
		cid := (s[i] << 8) | (s[i+1] & 16rFF);
		i += 2;
		if(cid == 0)
			continue;
		uni := cmaplookup(fm.entries, cid);
		if(uni > 0)
			out[len out] = uni;
	}
	return out;
}

# Look up a font by name in the font map.
fontmaplookup(fontmap: list of ref FontMapEntry, name: string): ref FontMapEntry
{
	for(; fontmap != nil; fontmap = tl fontmap){
		fm := hd fontmap;
		if(fm.name == name)
			return fm;
	}
	return nil;
}

# Read /Name from content stream data, return (name, newpos).
readcsname(data: array of byte, pos: int): (string, int)
{
	pos++;  # skip /
	name := "";
	while(pos < len data){
		c := int data[pos];
		if(isws(c) || c == '/' || c == '<' || c == '>' ||
		   c == '[' || c == ']' || c == '(' || c == ')' ||
		   c == '{' || c == '}' || c == '%')
			break;
		name[len name] = c;
		pos++;
	}
	return (name, pos);
}

# ---- Document Tree Construction ----

# Convert extracted text into a document tree for rendering
texttodom(text: string): list of ref DocNode
{
	doc: list of ref DocNode;

	# Split into paragraphs (double newline or page markers)
	lines := splitlines(text);
	i := 0;
	nlines := len lines;

	for(;;){
		if(i >= nlines)
			break;
		line := lines[i];

		# Blank line — skip
		if(isblankstr(line)){
			i++;
			continue;
		}

		# Page separator
		if(len line >= 3 && line[0:3] == "---"){
			doc = ref DocNode(Rlayout->Nhrule, nil, nil, 0) :: doc;
			# The page header line becomes a heading
			if(len line > 4){
				children := ref DocNode(Rlayout->Ntext, line[4:], nil, 0) :: nil;
				doc = ref DocNode(Rlayout->Nheading, nil, children, 2) :: doc;
			}
			i++;
			continue;
		}

		# Default: gather into paragraph
		para := "";
		while(i < nlines){
			l := lines[i];
			if(isblankstr(l))
				break;
			if(len l >= 3 && l[0:3] == "---")
				break;
			if(len para > 0)
				para += " ";
			para += l;
			i++;
		}

		if(len para > 0){
			children := ref DocNode(Rlayout->Ntext, para, nil, 0) :: nil;
			doc = ref DocNode(Rlayout->Npara, nil, children, 0) :: doc;
		}
	}

	# Reverse
	rev: list of ref DocNode;
	for(; doc != nil; doc = tl doc)
		rev = hd doc :: rev;
	return rev;
}

# ---- Utility Functions ----

# Dictionary lookup by key name
dictget(entries: list of ref DictEntry, key: string): ref PdfObj
{
	for(; entries != nil; entries = tl entries){
		e := hd entries;
		if(e.key == key)
			return e.val;
	}
	return nil;
}

# Dictionary lookup returning integer value
dictgetint(entries: list of ref DictEntry, key: string): int
{
	obj := dictget(entries, key);
	if(obj == nil)
		return 0;
	if(obj.kind == Oint)
		return obj.ival;
	return 0;
}

# Extract a string slice from byte array
slicestr(data: array of byte, pos, length: int): string
{
	if(pos + length > len data)
		length = len data - pos;
	s := "";
	for(i := 0; i < length; i++)
		s[len s] = int data[pos + i];
	return s;
}

# Read an integer from data, return (value, new position)
readint(data: array of byte, pos: int): (int, int)
{
	start := pos;
	while(pos < len data && int data[pos] >= '0' && int data[pos] <= '9')
		pos++;
	if(pos == start)
		return (0, start);
	return (int slicestr(data, start, pos - start), pos);
}

# Skip whitespace in byte array
skipws(data: array of byte, pos: int): int
{
	while(pos < len data){
		c := int data[pos];
		if(c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == 0)
			pos++;
		else if(c == '%'){
			# PDF comment — skip to end of line
			while(pos < len data && int data[pos] != '\n')
				pos++;
		} else
			break;
	}
	return pos;
}

# Check if byte is whitespace
isws(c: int): int
{
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == 0;
}

# Hex digit value
hexval(c: int): int
{
	if(c >= '0' && c <= '9')
		return c - '0';
	if(c >= 'a' && c <= 'f')
		return c - 'a' + 10;
	if(c >= 'A' && c <= 'F')
		return c - 'A' + 10;
	return -1;
}

# Split string into lines
splitlines(text: string): array of string
{
	nlines := 1;
	for(i := 0; i < len text; i++)
		if(text[i] == '\n')
			nlines++;

	lines := array[nlines] of string;
	li := 0;
	start := 0;
	for(i = 0; i < len text; i++){
		if(text[i] == '\n'){
			lines[li++] = text[start:i];
			start = i + 1;
		}
	}
	if(start <= len text)
		lines[li] = text[start:];
	return lines;
}

# Check if string is blank
isblankstr(s: string): int
{
	for(i := 0; i < len s; i++)
		if(s[i] != ' ' && s[i] != '\t' && s[i] != '\r')
			return 0;
	return 1;
}
