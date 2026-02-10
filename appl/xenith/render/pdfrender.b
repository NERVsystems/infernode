implement Renderer;

#
# PDF renderer - parses PDF files, extracts text, renders via rlayout.
#
# Phase 1 implementation:
#   - PDF cross-reference table (xref) and trailer parsing
#   - Indirect object lookup and stream extraction
#   - FlateDecode decompression via Filter module
#   - Content stream text extraction (Tj, TJ, ', " operators)
#   - Rendered output via shared rlayout engine
#
# The AI sees extracted text in the body buffer; the user sees
# a rendered document with headings, paragraphs, and structure.
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

rlayout: Rlayout;
display: ref Display;
DocNode: import rlayout;

# Font paths
PROPFONT: con "/fonts/vera/Vera/unicode.14.font";
MONOFONT: con "/fonts/vera/VeraMono/VeraMono.14.font";

propfont: ref Font;
monofont: ref Font;

# Current page for command dispatch
curpage := 0;

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
	if(rlayout == nil)
		return (nil, nil, "layout module not available");
	if(propfont == nil)
		return (nil, nil, "font not available");

	# Parse PDF
	(doc, err) := parsepdf(data);
	if(doc == nil){
		progress <-= nil;
		return (nil, nil, "PDF parse error: " + err);
	}

	# Extract text from all pages
	(text, texterr) := extracttext(doc);
	if(text == nil && texterr != nil){
		# Still try to render something
		text = "[PDF text extraction failed: " + texterr + "]";
	}

	if(text == nil || len text == 0)
		text = "[No extractable text in PDF]";

	# Build document tree from extracted text
	docnodes := texttodom(text);

	# Set up style
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
		nil;
}

command(cmd: string, arg: string,
        data: array of byte, hint: string,
        width, height: int): (ref Draw->Image, string)
{
	return (nil, "command not yet implemented: " + cmd);
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

	# Parse xref table
	(xref, nobjs, traileroff, xerr) := parsexref(data, xrefoff);
	if(xref == nil)
		return (nil, "cannot parse xref: " + xerr);

	# Parse trailer dictionary
	(trailer, nil, terr) := parseobj(data, traileroff);
	if(trailer == nil)
		return (nil, "cannot parse trailer: " + terr);

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
	if(entry == nil || !entry.inuse)
		return nil;

	# Parse the object at its offset
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

		# Get Contents
		contents := dictget(node.dval, "Contents");
		if(contents != nil){
			pagetext := extractpagetext(doc, contents);
			if(pagetext != nil)
				text += pagetext;
		}
	}

	return (text, pagenum);
}

# Extract text from a page's Contents (may be a stream or array of streams)
extractpagetext(doc: ref PdfDoc, contents: ref PdfObj): string
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
				t := extractstreamtext(doc, stream);
				if(t != nil)
					text += t;
			}
		}
		return text;
	}

	if(contents.kind == Ostream)
		return extractstreamtext(doc, contents);

	return nil;
}

# Extract text from a content stream
extractstreamtext(doc: ref PdfDoc, stream: ref PdfObj): string
{
	(data, err) := decompressstream(stream);
	if(data == nil)
		return nil;

	return parsecontentstream(data);
}

# Parse PDF content stream operators to extract text
parsecontentstream(data: array of byte): string
{
	text := "";
	pos := 0;
	operands: list of string;  # stack of operand strings

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
			(s, newpos) := readtjarray(data, pos);
			operands = s :: operands;
			pos = newpos;
			continue;
		}

		# Skip dict << >> inline images, etc.
		if(c == '<' && pos+1 < len data && int data[pos+1] == '<'){
			pos = skipdict(data, pos);
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
					text += cleanpdftext(s);
				}
			"TJ" =>
				# Show strings from array: [...] TJ
				if(operands != nil)
					text += cleanpdftext(hd operands);
			"'" =>
				# Move to next line and show: (string) '
				if(operands != nil){
					text += "\n" + cleanpdftext(hd operands);
				}
			"\"" =>
				# Set spacing, move, show: aw ac (string) "
				if(operands != nil)
					text += "\n" + cleanpdftext(hd operands);
			"Td" or "TD" =>
				# Text positioning — large vertical moves suggest paragraph breaks
				text += resolvetextpos(operands);
			"Tm" =>
				# Text matrix — reset
				;
			"T*" =>
				# Move to start of next line
				text += "\n";
			"BT" =>
				# Begin text object
				;
			"ET" =>
				# End text object
				text += " ";
			"Tf" =>
				# Set font (ignore for now)
				;
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

# Resolve Td/TD operands: if large vertical offset, insert paragraph break
resolvetextpos(operands: list of string): string
{
	# operands are in reverse order: ty, tx
	if(operands == nil)
		return " ";
	ty := hd operands;
	tyval := real ty;
	if(tyval < -1.0 || tyval > 1.0)
		return "\n";  # Significant vertical move = new line
	return " ";
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
# Extract text from string elements, ignore numbers
readtjarray(data: array of byte, pos: int): (string, int)
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
			s += substr;
			pos = newpos;
			continue;
		}

		if(c == '<'){
			(substr, newpos) := readhexstr(data, pos);
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
