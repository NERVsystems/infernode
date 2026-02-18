implement PDF;

#
# Native PDF parsing and rendering module.
#
# Parses PDF files in-memory, extracts text via ToUnicode CMaps,
# and renders pages to Draw images using Inferno graphics primitives.
#
# Phase 1: Vector graphics (paths, fills, strokes, colors)
# Phase 2: Text rendering (BT/ET, Tj/TJ, font mapping)
#

include "sys.m";
	sys: Sys;

include "draw.m";
	drawm: Draw;
	Display, Image, Font, Rect, Point: import drawm;

include "math.m";
	math: Math;

include "filter.m";
	filtermod: Filter;

include "pdf.m";

# ---- PDF internal types ----

Onull, Obool, Oint, Oreal, Ostring, Oname,
Oarray, Odict, Ostream, Oref: con iota;

PdfObj: adt {
	kind: int;
	ival: int;
	rval: real;
	sval: string;
	aval: list of ref PdfObj;
	dval: list of ref DictEntry;
	stream: array of byte;
};

DictEntry: adt {
	key: string;
	val: ref PdfObj;
};

XrefEntry: adt {
	offset: int;
	gen: int;
	inuse: int;
};

CMapEntry: adt {
	lo: int;
	hi: int;
	unicode: int;
};

FontMapEntry: adt {
	name: string;
	twobyte: int;
	entries: list of ref CMapEntry;
};

PdfDoc: adt {
	data: array of byte;
	xref: array of ref XrefEntry;
	trailer: ref PdfObj;
	nobjs: int;
};

# ---- Graphics state types ----

GState: adt {
	ctm: array of real;        # 6-element affine [a b c d e f]
	fillcolor: (int, int, int);
	strokecolor: (int, int, int);
	linewidth: real;
	linecap: int;
	linejoin: int;
	miterlimit: real;
	fontname: string;
	fontsize: real;
	tm: array of real;         # text matrix [a b c d e f]
	tlm: array of real;        # text line matrix
	charspace: real;
	wordspace: real;
	hscale: real;
	leading: real;
	rise: real;
	rendermode: int;
};

PathSeg: adt {
	pick {
	Move =>
		x, y: real;
	Line =>
		x, y: real;
	Curve =>
		x1, y1, x2, y2, x3, y3: real;
	Close =>
	}
};

# Color cache entry
ColorCacheEntry: adt {
	r, g, b: int;
	img: ref Image;
};

# ---- Module state ----
display: ref Display;
colorcache: list of ref ColorCacheEntry;

# Font paths
SANSFONT: con "/fonts/vera/Vera/unicode.14.font";
MONOFONT: con "/fonts/vera/VeraMono/VeraMono.14.font";

sansfont: ref Font;
monofont: ref Font;

init(d: ref Display): string
{
	sys = load Sys Sys->PATH;
	drawm = load Draw Draw->PATH;
	math = load Math Math->PATH;
	if(sys == nil || drawm == nil || math == nil)
		return "cannot load system modules";
	display = d;
	colorcache = nil;

	if(d != nil){
		sansfont = Font.open(d, SANSFONT);
		monofont = Font.open(d, MONOFONT);
		if(sansfont == nil)
			sansfont = Font.open(d, "*default*");
		if(monofont == nil)
			monofont = sansfont;
	}
	return nil;
}

open(data: array of byte): (ref Doc, string)
{
	if(len data < 20)
		return (nil, "file too small");

	(pdoc, err) := parsepdf(data);
	if(pdoc == nil)
		return (nil, err);

	# Store in docs table, return handle with index
	idx := adddoc(pdoc);
	return (ref Doc(idx), nil);
}

# Document table (supports multiple open documents)
doctab: array of ref PdfDoc;
ndocs := 0;

adddoc(pdoc: ref PdfDoc): int
{
	if(doctab == nil)
		doctab = array[4] of ref PdfDoc;
	if(ndocs >= len doctab){
		newtab := array[len doctab * 2] of ref PdfDoc;
		newtab[0:] = doctab;
		doctab = newtab;
	}
	idx := ndocs;
	doctab[idx] = pdoc;
	ndocs++;
	return idx;
}

getdoc(idx: int): ref PdfDoc
{
	if(doctab == nil || idx < 0 || idx >= ndocs)
		return nil;
	return doctab[idx];
}

Doc.pagecount(d: self ref Doc): int
{
	pdoc := getdoc(d.idx);
	if(pdoc == nil)
		return 0;
	return countpages(pdoc);
}

Doc.pagesize(d: self ref Doc, page: int): (real, real)
{
	pdoc := getdoc(d.idx);
	if(pdoc == nil)
		return (0.0, 0.0);
	pobj := getpageobj(pdoc, page);
	if(pobj == nil)
		return (612.0, 792.0);  # default US Letter
	return getmediabox(pdoc, pobj);
}

Doc.renderpage(d: self ref Doc, page, dpi: int): (ref Image, string)
{
	pdoc := getdoc(d.idx);
	if(pdoc == nil)
		return (nil, "no document");
	if(display == nil)
		return (nil, "no display");
	pobj := getpageobj(pdoc, page);
	if(pobj == nil)
		return (nil, sys->sprint("page %d not found", page));
	return renderpage(pdoc, pobj, dpi);
}

Doc.extracttext(d: self ref Doc, page: int): string
{
	pdoc := getdoc(d.idx);
	if(pdoc == nil)
		return nil;
	pobj := getpageobj(pdoc, page);
	if(pobj == nil)
		return nil;
	return extractpagetext_full(pdoc, pobj);
}

Doc.extractall(d: self ref Doc): string
{
	pdoc := getdoc(d.idx);
	if(pdoc == nil)
		return nil;
	(text, nil) := extracttext(pdoc);
	return text;
}

# ---- Page tree navigation ----

countpages(doc: ref PdfDoc): int
{
	root := dictget(doc.trailer.dval, "Root");
	if(root == nil) return 0;
	root = resolve(doc, root);
	if(root == nil) return 0;
	pages := dictget(root.dval, "Pages");
	if(pages == nil) return 0;
	pages = resolve(doc, pages);
	if(pages == nil) return 0;
	return countpagenode(doc, pages);
}

countpagenode(doc: ref PdfDoc, node: ref PdfObj): int
{
	if(node == nil) return 0;
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

# Get the Nth page object (1-indexed)
getpageobj(doc: ref PdfDoc, page: int): ref PdfObj
{
	root := dictget(doc.trailer.dval, "Root");
	if(root == nil) return nil;
	root = resolve(doc, root);
	if(root == nil) return nil;
	pages := dictget(root.dval, "Pages");
	if(pages == nil) return nil;
	pages = resolve(doc, pages);
	if(pages == nil) return nil;

	(pobj, nil) := findpage(doc, pages, page, 0);
	return pobj;
}

# Find page by number, returns (page obj, count so far)
findpage(doc: ref PdfDoc, node: ref PdfObj, target, sofar: int): (ref PdfObj, int)
{
	if(node == nil)
		return (nil, sofar);
	typobj := dictget(node.dval, "Type");
	typ := "";
	if(typobj != nil && typobj.kind == Oname)
		typ = typobj.sval;

	if(typ == "Page"){
		sofar++;
		if(sofar == target)
			return (node, sofar);
		return (nil, sofar);
	}

	if(typ == "Pages"){
		kids := dictget(node.dval, "Kids");
		if(kids != nil && kids.kind == Oarray){
			for(k := kids.aval; k != nil; k = tl k){
				child := resolve(doc, hd k);
				if(child == nil)
					continue;
				(pobj, ns) := findpage(doc, child, target, sofar);
				if(pobj != nil)
					return (pobj, ns);
				sofar = ns;
			}
		}
	}
	return (nil, sofar);
}

# Get MediaBox (or CropBox) dimensions in points
getmediabox(doc: ref PdfDoc, page: ref PdfObj): (real, real)
{
	box := dictget(page.dval, "CropBox");
	if(box == nil)
		box = dictget(page.dval, "MediaBox");
	if(box != nil)
		box = resolve(doc, box);
	if(box == nil || box.kind != Oarray)
		return (612.0, 792.0);

	vals := array[4] of { * => 0.0 };
	i := 0;
	for(l := box.aval; l != nil && i < 4; l = tl l){
		o := hd l;
		if(o.kind == Oint)
			vals[i] = real o.ival;
		else if(o.kind == Oreal)
			vals[i] = o.rval;
		i++;
	}
	w := vals[2] - vals[0];
	h := vals[3] - vals[1];
	if(w <= 0.0) w = 612.0;
	if(h <= 0.0) h = 792.0;
	return (w, h);
}

# ---- Rendering engine ----

renderpage(doc: ref PdfDoc, page: ref PdfObj, dpi: int): (ref Image, string)
{
	(pw, ph) := getmediabox(doc, page);
	scale := real dpi / 72.0;
	pixw := int (pw * scale + 0.5);
	pixh := int (ph * scale + 0.5);

	if(pixw <= 0) pixw = 1;
	if(pixh <= 0) pixh = 1;

	# Create page image with white background
	img := display.newimage(Rect(Point(0,0), Point(pixw, pixh)),
		drawm->RGB24, 0, drawm->White);
	if(img == nil)
		return (nil, "cannot allocate page image");

	# Initialize graphics state
	gs := newgstate();
	# PDF coordinate system: origin bottom-left, y-up
	# Screen: origin top-left, y-down
	# CTM transforms PDF coords -> pixel coords:
	# x_pixel = x_pdf * scale
	# y_pixel = pixh - y_pdf * scale
	gs.ctm[0] = scale;    # a
	gs.ctm[1] = 0.0;      # b
	gs.ctm[2] = 0.0;      # c
	gs.ctm[3] = -scale;   # d (flip y)
	gs.ctm[4] = 0.0;      # e
	gs.ctm[5] = real pixh; # f

	# Get page resources
	resources := dictget(page.dval, "Resources");
	if(resources != nil)
		resources = resolve(doc, resources);

	# Build font map for text
	fontmap := buildfontmap(doc, page);

	# Get content streams
	contents := dictget(page.dval, "Contents");
	if(contents == nil)
		return (img, nil);  # blank page
	contents = resolve(doc, contents);
	if(contents == nil)
		return (img, nil);

	# Collect content stream data
	csdata: array of byte;
	if(contents.kind == Oarray){
		chunks: list of array of byte;
		total := 0;
		for(a := contents.aval; a != nil; a = tl a){
			stream := resolve(doc, hd a);
			if(stream != nil && stream.kind == Ostream){
				(sd, nil) := decompressstream(stream);
				if(sd != nil){
					chunks = sd :: chunks;
					total += len sd;
				}
			}
		}
		csdata = array[total] of byte;
		pos := total;
		for(; chunks != nil; chunks = tl chunks){
			chunk := hd chunks;
			pos -= len chunk;
			csdata[pos:] = chunk;
		}
	} else if(contents.kind == Ostream){
		(sd, nil) := decompressstream(contents);
		csdata = sd;
	}

	if(csdata == nil || len csdata == 0)
		return (img, nil);

	# Execute content stream
	execcontentstream(doc, img, csdata, gs, resources, fontmap);
	return (img, nil);
}

newgstate(): ref GState
{
	ctm := array[6] of { * => 0.0 };
	ctm[0] = 1.0; ctm[3] = 1.0;  # identity
	tm := array[6] of { * => 0.0 };
	tm[0] = 1.0; tm[3] = 1.0;
	tlm := array[6] of { * => 0.0 };
	tlm[0] = 1.0; tlm[3] = 1.0;
	return ref GState(
		ctm,
		(0, 0, 0),       # fillcolor (black)
		(0, 0, 0),       # strokecolor (black)
		1.0,              # linewidth
		0,                # linecap
		0,                # linejoin
		10.0,             # miterlimit
		nil,              # fontname
		12.0,             # fontsize
		tm,               # text matrix
		tlm,              # text line matrix
		0.0,              # charspace
		0.0,              # wordspace
		100.0,            # hscale
		0.0,              # leading
		0.0,              # rise
		0                 # rendermode
	);
}

copygstate(gs: ref GState): ref GState
{
	ctm := array[6] of real;
	ctm[0:] = gs.ctm;
	tm := array[6] of real;
	tm[0:] = gs.tm;
	tlm := array[6] of real;
	tlm[0:] = gs.tlm;
	return ref GState(
		ctm,
		gs.fillcolor,
		gs.strokecolor,
		gs.linewidth,
		gs.linecap,
		gs.linejoin,
		gs.miterlimit,
		gs.fontname,
		gs.fontsize,
		tm, tlm,
		gs.charspace,
		gs.wordspace,
		gs.hscale,
		gs.leading,
		gs.rise,
		gs.rendermode
	);
}

# ---- Content stream interpreter ----

execcontentstream(doc: ref PdfDoc, img: ref Image, data: array of byte,
	gs: ref GState, resources: ref PdfObj, fontmap: list of ref FontMapEntry)
{
	pos := 0;
	operands: list of real;
	stroperands: list of string;
	path: list of ref PathSeg;
	gsstack: list of ref GState;
	curfont: ref FontMapEntry;

	while(pos < len data){
		pos = skipws(data, pos);
		if(pos >= len data)
			break;

		c := int data[pos];

		# Number
		if((c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.'){
			(val, newpos) := readreal(data, pos);
			operands = val :: operands;
			pos = newpos;
			continue;
		}

		# String operand (...)
		if(c == '('){
			(s, newpos) := readlitstr(data, pos);
			stroperands = s :: stroperands;
			pos = newpos;
			continue;
		}

		# Hex string <...>
		if(c == '<' && (pos+1 >= len data || int data[pos+1] != '<')){
			(s, newpos) := readhexstr(data, pos);
			stroperands = s :: stroperands;
			pos = newpos;
			continue;
		}

		# Array [...] for TJ
		if(c == '['){
			(s, newpos) := readtjarray(data, pos, curfont);
			stroperands = s :: stroperands;
			pos = newpos;
			continue;
		}

		# Dict << >> (inline image dict etc)
		if(c == '<' && pos+1 < len data && int data[pos+1] == '<'){
			pos = skipdict(data, pos);
			continue;
		}

		# Name /Foo
		if(c == '/'){
			(name, newpos) := readcsname(data, pos);
			stroperands = name :: stroperands;
			pos = newpos;
			continue;
		}

		# Comment
		if(c == '%'){
			while(pos < len data && int data[pos] != '\n')
				pos++;
			continue;
		}

		# Operator
		if((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		   c == '\'' || c == '"' || c == '*'){
			(op, newpos) := readtoken(data, pos);
			pos = newpos;

			case op {
			# ---- Graphics state ----
			"q" =>
				gsstack = copygstate(gs) :: gsstack;
			"Q" =>
				if(gsstack != nil){
					ngs := hd gsstack;
					gsstack = tl gsstack;
					gs.fillcolor = ngs.fillcolor;
					gs.strokecolor = ngs.strokecolor;
					gs.linewidth = ngs.linewidth;
					gs.linecap = ngs.linecap;
					gs.linejoin = ngs.linejoin;
					gs.miterlimit = ngs.miterlimit;
					gs.fontname = ngs.fontname;
					gs.fontsize = ngs.fontsize;
					gs.ctm[0:] = ngs.ctm;
				}
			"cm" =>
				if(lenlist(operands) >= 6){
					(f, e, dd, cc, b, a, nil) := pop6(operands);
					operands = nil;
					newctm := matmul(array[] of {a, b, cc, dd, e, f}, gs.ctm);
					gs.ctm[0:] = newctm;
				}
			"w" =>
				if(operands != nil){
					gs.linewidth = hd operands;
					operands = nil;
				}
			"J" =>
				if(operands != nil){
					gs.linecap = int (hd operands);
					operands = nil;
				}
			"j" =>
				if(operands != nil){
					gs.linejoin = int (hd operands);
					operands = nil;
				}
			"M" =>
				if(operands != nil){
					gs.miterlimit = hd operands;
					operands = nil;
				}
			"d" =>
				# dash pattern - ignore for now
				operands = nil;
			"gs" =>
				# ExtGState - ignore for now
				stroperands = nil;
			"ri" or "i" =>
				operands = nil;

			# ---- Path construction ----
			"m" =>
				if(lenlist(operands) >= 2){
					(y, x, nil) := pop2(operands);
					operands = nil;
					path = ref PathSeg.Move(x, y) :: path;
				}
			"l" =>
				if(lenlist(operands) >= 2){
					(y, x, nil) := pop2(operands);
					operands = nil;
					path = ref PathSeg.Line(x, y) :: path;
				}
			"c" =>
				if(lenlist(operands) >= 6){
					(y3, x3, y2, x2, y1, x1, nil) := pop6(operands);
					operands = nil;
					path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
				}
			"v" =>
				# current point is first control point
				if(lenlist(operands) >= 4){
					(y3, x3, y2, x2, nil) := pop4(operands);
					operands = nil;
					(cx, cy) := currentpoint(path);
					path = ref PathSeg.Curve(cx, cy, x2, y2, x3, y3) :: path;
				}
			"y" =>
				# endpoint is second control point
				if(lenlist(operands) >= 4){
					(y3, x3, y1, x1, nil) := pop4(operands);
					operands = nil;
					path = ref PathSeg.Curve(x1, y1, x3, y3, x3, y3) :: path;
				}
			"h" =>
				path = ref PathSeg.Close :: path;
			"re" =>
				if(lenlist(operands) >= 4){
					(h, w, y, x, nil) := pop4(operands);
					operands = nil;
					path = ref PathSeg.Move(x, y) :: path;
					path = ref PathSeg.Line(x+w, y) :: path;
					path = ref PathSeg.Line(x+w, y+h) :: path;
					path = ref PathSeg.Line(x, y+h) :: path;
					path = ref PathSeg.Close :: path;
				}

			# ---- Paint operators ----
			"S" =>
				strokepath(img, gs, path);
				path = nil;
			"s" =>
				path = ref PathSeg.Close :: path;
				strokepath(img, gs, path);
				path = nil;
			"f" or "F" =>
				fillpath(img, gs, path, 0);
				path = nil;
			"f*" =>
				fillpath(img, gs, path, 1);
				path = nil;
			"B" =>
				fillpath(img, gs, path, 0);
				strokepath(img, gs, path);
				path = nil;
			"B*" =>
				fillpath(img, gs, path, 1);
				strokepath(img, gs, path);
				path = nil;
			"b" =>
				path = ref PathSeg.Close :: path;
				fillpath(img, gs, path, 0);
				strokepath(img, gs, path);
				path = nil;
			"b*" =>
				path = ref PathSeg.Close :: path;
				fillpath(img, gs, path, 1);
				strokepath(img, gs, path);
				path = nil;
			"n" =>
				path = nil;

			# ---- Clipping (stub) ----
			"W" or "W*" =>
				;  # clipping not implemented

			# ---- Color operators ----
			"g" =>
				if(operands != nil){
					gray := hd operands;
					operands = nil;
					v := clampcolor(gray);
					gs.fillcolor = (v, v, v);
				}
			"G" =>
				if(operands != nil){
					gray := hd operands;
					operands = nil;
					v := clampcolor(gray);
					gs.strokecolor = (v, v, v);
				}
			"rg" =>
				if(lenlist(operands) >= 3){
					(bv, gv, rv, nil) := pop3(operands);
					operands = nil;
					gs.fillcolor = (clampcolor(rv), clampcolor(gv), clampcolor(bv));
				}
			"RG" =>
				if(lenlist(operands) >= 3){
					(bv, gv, rv, nil) := pop3(operands);
					operands = nil;
					gs.strokecolor = (clampcolor(rv), clampcolor(gv), clampcolor(bv));
				}
			"k" =>
				if(lenlist(operands) >= 4){
					(kk, yy, mm, cc, nil) := pop4(operands);
					operands = nil;
					(r, g, b) := cmyk2rgb(cc, mm, yy, kk);
					gs.fillcolor = (r, g, b);
				}
			"K" =>
				if(lenlist(operands) >= 4){
					(kk, yy, mm, cc, nil) := pop4(operands);
					operands = nil;
					(r, g, b) := cmyk2rgb(cc, mm, yy, kk);
					gs.strokecolor = (r, g, b);
				}
			"cs" or "CS" =>
				# color space - consume name, use defaults
				stroperands = nil;
			"sc" or "scn" =>
				# set fill color in current space
				n := lenlist(operands);
				if(n >= 3){
					(bv, gv, rv, nil) := pop3(operands);
					gs.fillcolor = (clampcolor(rv), clampcolor(gv), clampcolor(bv));
				} else if(n >= 1){
					v := clampcolor(hd operands);
					gs.fillcolor = (v, v, v);
				}
				operands = nil;
				stroperands = nil;
			"SC" or "SCN" =>
				n := lenlist(operands);
				if(n >= 3){
					(bv, gv, rv, nil) := pop3(operands);
					gs.strokecolor = (clampcolor(rv), clampcolor(gv), clampcolor(bv));
				} else if(n >= 1){
					v := clampcolor(hd operands);
					gs.strokecolor = (v, v, v);
				}
				operands = nil;
				stroperands = nil;

			# ---- Text operators (Phase 2) ----
			"BT" =>
				gs.tm[0] = 1.0; gs.tm[1] = 0.0;
				gs.tm[2] = 0.0; gs.tm[3] = 1.0;
				gs.tm[4] = 0.0; gs.tm[5] = 0.0;
				gs.tlm[0:] = gs.tm;
			"ET" =>
				;
			"Td" =>
				if(lenlist(operands) >= 2){
					(ty, tx, nil) := pop2(operands);
					operands = nil;
					gs.tlm[4] += tx * gs.tlm[0] + ty * gs.tlm[2];
					gs.tlm[5] += tx * gs.tlm[1] + ty * gs.tlm[3];
					gs.tm[0:] = gs.tlm;
				}
			"TD" =>
				if(lenlist(operands) >= 2){
					(ty, tx, nil) := pop2(operands);
					operands = nil;
					gs.leading = -ty;
					gs.tlm[4] += tx * gs.tlm[0] + ty * gs.tlm[2];
					gs.tlm[5] += tx * gs.tlm[1] + ty * gs.tlm[3];
					gs.tm[0:] = gs.tlm;
				}
			"Tm" =>
				if(lenlist(operands) >= 6){
					(f, e, dd, cc, b, a, nil) := pop6(operands);
					operands = nil;
					gs.tm[0] = a; gs.tm[1] = b;
					gs.tm[2] = cc; gs.tm[3] = dd;
					gs.tm[4] = e; gs.tm[5] = f;
					gs.tlm[0:] = gs.tm;
				}
			"T*" =>
				gs.tlm[4] += (-gs.leading) * gs.tlm[2];
				gs.tlm[5] += (-gs.leading) * gs.tlm[3];
				gs.tm[0:] = gs.tlm;
			"Tf" =>
				if(stroperands != nil){
					gs.fontname = hd stroperands;
					stroperands = tl stroperands;
					curfont = fontmaplookup(fontmap, gs.fontname);
				}
				if(operands != nil){
					gs.fontsize = hd operands;
					operands = nil;
				}
			"Tc" =>
				if(operands != nil){
					gs.charspace = hd operands;
					operands = nil;
				}
			"Tw" =>
				if(operands != nil){
					gs.wordspace = hd operands;
					operands = nil;
				}
			"Tz" =>
				if(operands != nil){
					gs.hscale = hd operands;
					operands = nil;
				}
			"TL" =>
				if(operands != nil){
					gs.leading = hd operands;
					operands = nil;
				}
			"Ts" =>
				if(operands != nil){
					gs.rise = hd operands;
					operands = nil;
				}
			"Tr" =>
				if(operands != nil){
					gs.rendermode = int (hd operands);
					operands = nil;
				}
			"Tj" =>
				if(stroperands != nil){
					s := hd stroperands;
					stroperands = nil;
					if(curfont != nil)
						s = decodecidstr(s, curfont);
					rendertext(img, gs, s);
				}
			"TJ" =>
				if(stroperands != nil){
					s := hd stroperands;
					stroperands = nil;
					# TJ array already decoded in readtjarray
					rendertext(img, gs, s);
				}
			"'" =>
				# newline + show
				gs.tlm[4] += (-gs.leading) * gs.tlm[2];
				gs.tlm[5] += (-gs.leading) * gs.tlm[3];
				gs.tm[0:] = gs.tlm;
				if(stroperands != nil){
					s := hd stroperands;
					stroperands = nil;
					if(curfont != nil)
						s = decodecidstr(s, curfont);
					rendertext(img, gs, s);
				}
			"\"" =>
				# set word/char space, newline, show
				if(lenlist(operands) >= 2){
					(ac, aw, nil) := pop2(operands);
					operands = nil;
					gs.wordspace = aw;
					gs.charspace = ac;
				}
				gs.tlm[4] += (-gs.leading) * gs.tlm[2];
				gs.tlm[5] += (-gs.leading) * gs.tlm[3];
				gs.tm[0:] = gs.tlm;
				if(stroperands != nil){
					s := hd stroperands;
					stroperands = nil;
					if(curfont != nil)
						s = decodecidstr(s, curfont);
					rendertext(img, gs, s);
				}

			# ---- XObject (Phase 3 stub) ----
			"Do" =>
				stroperands = nil;

			# ---- Inline images ----
			"BI" =>
				pos = skipinlineimage(data, pos);

			# ---- Marked content ----
			"BDC" or "BMC" or "EMC" or "MP" or "DP" =>
				stroperands = nil;
				operands = nil;

			* =>
				# Unknown operator
				operands = nil;
				stroperands = nil;
			}
			continue;
		}

		# Skip unrecognized byte
		pos++;
	}
}

# ---- Text rendering ----

rendertext(img: ref Image, gs: ref GState, text: string)
{
	if(text == nil || len text == 0)
		return;
	if(gs.rendermode == 3)  # invisible
		return;

	font := pickfont(gs.fontname);
	if(font == nil)
		return;

	(fr, fg, fb) := gs.fillcolor;
	colimg := getcolor(fr, fg, fb);
	if(colimg == nil)
		return;

	# Compute text position through CTM
	# Text rendering matrix = Tm * CTM
	trm := matmul(gs.tm, gs.ctm);

	# The font size from PDF doesn't map well to bitmap fonts,
	# but we place text at the correct position
	px := int (trm[4] + 0.5);
	py := int (trm[5] + 0.5);

	p := Point(px, py);

	# Adjust for font height (PDF y is baseline, Draw y is top)
	p.y -= font.height * 3 / 4;

	# Draw the text
	img.text(p, colimg, Point(0,0), font, text);

	# Advance text matrix by approximate string width
	# Use font metrics for width estimate
	w := font.width(text);
	adv := real w;
	# Transform advance back to text space
	if(trm[0] != 0.0)
		adv = adv / trm[0];
	gs.tm[4] += adv;
}

pickfont(name: string): ref Font
{
	if(name == nil)
		return sansfont;
	# Check for monospace indicators
	for(i := 0; i < len name; i++){
		if(i + 4 <= len name){
			sub := "";
			for(j := i; j < i + 7 && j < len name; j++)
				sub += sys->sprint("%c", tolower(name[j]));
			if(len sub >= 4 && sub[0:4] == "mono")
				return monofont;
			if(len sub >= 7 && sub[0:7] == "courier")
				return monofont;
		}
	}
	return sansfont;
}

tolower(c: int): int
{
	if(c >= 'A' && c <= 'Z')
		return c - 'A' + 'a';
	return c;
}

# ---- Path rendering ----

fillpath(img: ref Image, gs: ref GState, path: list of ref PathSeg, evenodd: int)
{
	if(path == nil)
		return;

	# Reverse path (it was built in reverse order)
	rpath := reversepath(path);

	# Flatten to points
	pts := flattenpath(rpath, gs.ctm);
	if(pts == nil || len pts < 3)
		return;

	(r, g, b) := gs.fillcolor;
	colimg := getcolor(r, g, b);
	if(colimg == nil)
		return;

	wind := ~0;
	if(evenodd)
		wind = 1;

	img.fillpoly(pts, wind, colimg, Point(0,0));
}

strokepath(img: ref Image, gs: ref GState, path: list of ref PathSeg)
{
	if(path == nil)
		return;

	rpath := reversepath(path);
	pts := flattenpath(rpath, gs.ctm);
	if(pts == nil || len pts < 2)
		return;

	(r, g, b) := gs.strokecolor;
	colimg := getcolor(r, g, b);
	if(colimg == nil)
		return;

	# Compute line width in pixels
	# Use average of x and y scale factors
	sx := math->sqrt(gs.ctm[0]*gs.ctm[0] + gs.ctm[1]*gs.ctm[1]);
	radius := int (gs.linewidth * sx / 2.0 + 0.5);
	if(radius < 0) radius = 0;

	# Map line cap
	end0 := drawm->Enddisc;
	case gs.linecap {
	0 => end0 = drawm->Endsquare;
	1 => end0 = drawm->Enddisc;
	2 => end0 = drawm->Endarrow;  # projecting square ~ arrow
	}

	img.poly(pts, end0, end0, radius, colimg, Point(0,0));
}

# Reverse a path segment list
reversepath(path: list of ref PathSeg): list of ref PathSeg
{
	rev: list of ref PathSeg;
	for(; path != nil; path = tl path)
		rev = hd path :: rev;
	return rev;
}

# Flatten path to array of Points, transforming through CTM
flattenpath(path: list of ref PathSeg, ctm: array of real): array of Point
{
	pts: list of Point;
	npts := 0;
	cx := 0.0;
	cy := 0.0;
	startx := 0.0;
	starty := 0.0;

	for(; path != nil; path = tl path){
		seg := hd path;
		pick s := seg {
		Move =>
			cx = s.x; cy = s.y;
			startx = cx; starty = cy;
			(px, py) := xformpt(cx, cy, ctm);
			pts = Point(px, py) :: pts;
			npts++;
		Line =>
			cx = s.x; cy = s.y;
			(px, py) := xformpt(cx, cy, ctm);
			pts = Point(px, py) :: pts;
			npts++;
		Curve =>
			# De Casteljau subdivision to polyline
			bpts := flattenbezier(cx, cy, s.x1, s.y1, s.x2, s.y2, s.x3, s.y3, ctm);
			for(bp := bpts; bp != nil; bp = tl bp){
				pts = hd bp :: pts;
				npts++;
			}
			cx = s.x3; cy = s.y3;
		Close =>
			cx = startx; cy = starty;
			(px, py) := xformpt(cx, cy, ctm);
			pts = Point(px, py) :: pts;
			npts++;
		}
	}

	if(npts == 0)
		return nil;

	# Reverse to correct order
	result := array[npts] of Point;
	i := npts - 1;
	for(; pts != nil; pts = tl pts)
		result[i--] = hd pts;
	return result;
}

# Transform a point through CTM
xformpt(x, y: real, ctm: array of real): (int, int)
{
	px := x * ctm[0] + y * ctm[2] + ctm[4];
	py := x * ctm[1] + y * ctm[3] + ctm[5];
	return (int (px + 0.5), int (py + 0.5));
}

# Flatten a cubic bezier to polyline points via subdivision
FLAT_THRESH: con 1.0;  # pixel tolerance

flattenbezier(x0, y0, x1, y1, x2, y2, x3, y3: real,
	ctm: array of real): list of Point
{
	# Transform all control points
	(px0, py0) := xformpt(x0, y0, ctm);
	(px1, py1) := xformpt(x1, y1, ctm);
	(px2, py2) := xformpt(x2, y2, ctm);
	(px3, py3) := xformpt(x3, y3, ctm);

	return subdividebezier(
		real px0, real py0,
		real px1, real py1,
		real px2, real py2,
		real px3, real py3,
		0);
}

subdividebezier(x0, y0, x1, y1, x2, y2, x3, y3: real,
	depth: int): list of Point
{
	# Check flatness: if control points are close to the line x0,y0 -> x3,y3
	dx := x3 - x0;
	dy := y3 - y0;
	d2 := math->fabs((x1 - x3) * dy - (y1 - y3) * dx);
	d3 := math->fabs((x2 - x3) * dy - (y2 - y3) * dx);

	if((d2 + d3) * (d2 + d3) <= FLAT_THRESH * (dx*dx + dy*dy) || depth > 8){
		return Point(int (x3 + 0.5), int (y3 + 0.5)) :: nil;
	}

	# Subdivide at t=0.5
	mx01 := (x0 + x1) / 2.0;
	my01 := (y0 + y1) / 2.0;
	mx12 := (x1 + x2) / 2.0;
	my12 := (y1 + y2) / 2.0;
	mx23 := (x2 + x3) / 2.0;
	my23 := (y2 + y3) / 2.0;
	mx012 := (mx01 + mx12) / 2.0;
	my012 := (my01 + my12) / 2.0;
	mx123 := (mx12 + mx23) / 2.0;
	my123 := (my12 + my23) / 2.0;
	mx0123 := (mx012 + mx123) / 2.0;
	my0123 := (my012 + my123) / 2.0;

	left := subdividebezier(x0, y0, mx01, my01, mx012, my012, mx0123, my0123, depth+1);
	right := subdividebezier(mx0123, my0123, mx123, my123, mx23, my23, x3, y3, depth+1);

	# Concatenate: append right to left
	result := right;
	for(l := revpoints(left); l != nil; l = tl l)
		result = hd l :: result;
	return result;
}

revpoints(pts: list of Point): list of Point
{
	rev: list of Point;
	for(; pts != nil; pts = tl pts)
		rev = hd pts :: rev;
	return rev;
}

# Get current point from path
currentpoint(path: list of ref PathSeg): (real, real)
{
	# Path is reversed; head is most recent
	for(; path != nil; path = tl path){
		seg := hd path;
		pick s := seg {
		Move => return (s.x, s.y);
		Line => return (s.x, s.y);
		Curve => return (s.x3, s.y3);
		Close => ;  # keep looking
		}
	}
	return (0.0, 0.0);
}

# ---- Color helpers ----

clampcolor(v: real): int
{
	i := int (v * 255.0 + 0.5);
	if(i < 0) i = 0;
	if(i > 255) i = 255;
	return i;
}

cmyk2rgb(c, m, y, k: real): (int, int, int)
{
	r := 1.0 - (c + k);
	g := 1.0 - (m + k);
	b := 1.0 - (y + k);
	if(r < 0.0) r = 0.0;
	if(g < 0.0) g = 0.0;
	if(b < 0.0) b = 0.0;
	return (clampcolor(r), clampcolor(g), clampcolor(b));
}

getcolor(r, g, b: int): ref Image
{
	# Check cache
	for(cl := colorcache; cl != nil; cl = tl cl){
		e := hd cl;
		if(e.r == r && e.g == g && e.b == b)
			return e.img;
	}
	# Create new color image
	rgb := (r << 24) | (g << 16) | (b << 8) | 16rFF;
	img := display.newimage(Rect(Point(0,0), Point(1,1)), drawm->RGB24, 1, rgb);
	if(img != nil)
		colorcache = ref ColorCacheEntry(r, g, b, img) :: colorcache;
	return img;
}

# ---- Matrix operations ----

# Multiply two 3x3 affine matrices stored as [a b c d e f]
# Result = A * B
matmul(a, b: array of real): array of real
{
	r := array[6] of real;
	r[0] = a[0]*b[0] + a[1]*b[2];
	r[1] = a[0]*b[1] + a[1]*b[3];
	r[2] = a[2]*b[0] + a[3]*b[2];
	r[3] = a[2]*b[1] + a[3]*b[3];
	r[4] = a[4]*b[0] + a[5]*b[2] + b[4];
	r[5] = a[4]*b[1] + a[5]*b[3] + b[5];
	return r;
}

# ---- Operand stack helpers ----

lenlist(l: list of real): int
{
	n := 0;
	for(; l != nil; l = tl l)
		n++;
	return n;
}

pop2(l: list of real): (real, real, list of real)
{
	a := hd l; l = tl l;
	b := hd l; l = tl l;
	return (a, b, l);
}

pop3(l: list of real): (real, real, real, list of real)
{
	a := hd l; l = tl l;
	b := hd l; l = tl l;
	c := hd l; l = tl l;
	return (a, b, c, l);
}

pop4(l: list of real): (real, real, real, real, list of real)
{
	a := hd l; l = tl l;
	b := hd l; l = tl l;
	c := hd l; l = tl l;
	d := hd l; l = tl l;
	return (a, b, c, d, l);
}

pop6(l: list of real): (real, real, real, real, real, real, list of real)
{
	a := hd l; l = tl l;
	b := hd l; l = tl l;
	c := hd l; l = tl l;
	d := hd l; l = tl l;
	e := hd l; l = tl l;
	f := hd l; l = tl l;
	return (a, b, c, d, e, f, l);
}

# ---- Read number from content stream ----

readreal(data: array of byte, pos: int): (real, int)
{
	start := pos;
	if(pos < len data && (int data[pos] == '-' || int data[pos] == '+'))
		pos++;
	isreal := 0;
	while(pos < len data){
		c := int data[pos];
		if(c >= '0' && c <= '9')
			pos++;
		else if(c == '.' && !isreal){
			isreal = 1;
			pos++;
		} else
			break;
	}
	if(pos == start)
		return (0.0, pos);
	s := slicestr(data, start, pos - start);
	return (real s, pos);
}

# ---- PDF Parser (extracted from pdfrender.b) ----

parsepdf(data: array of byte): (ref PdfDoc, string)
{
	if(len data < 20)
		return (nil, "file too small");
	if(data[0] != byte '%' || data[1] != byte 'P' ||
	   data[2] != byte 'D' || data[3] != byte 'F')
		return (nil, "not a PDF file");

	(xrefoff, err) := findstartxref(data);
	if(xrefoff < 0)
		return (nil, "cannot find startxref: " + err);

	(xref, nobjs, traileroff, xerr) := parsexref(data, xrefoff);
	if(xref != nil){
		(trailer, nil, terr) := parseobj(data, traileroff);
		if(trailer == nil)
			return (nil, "cannot parse trailer: " + terr);
		doc := ref PdfDoc(data, xref, trailer, nobjs);
		return (doc, nil);
	}

	trailer: ref PdfObj;
	xserr: string;
	(xref, nobjs, trailer, xserr) = parsexrefstream(data, xrefoff);
	if(xref == nil)
		return (nil, "cannot parse xref: " + xerr + "; xref stream: " + xserr);

	doc := ref PdfDoc(data, xref, trailer, nobjs);
	return (doc, nil);
}

findstartxref(data: array of byte): (int, string)
{
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
		if(found)
			pos = i;
	}
	if(pos < 0)
		return (-1, "startxref not found");

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

parsexref(data: array of byte, offset: int): (array of ref XrefEntry, int, int, string)
{
	pos := offset;
	if(pos + 4 > len data)
		return (nil, 0, 0, "truncated xref");
	tag := slicestr(data, pos, 4);
	if(tag != "xref")
		return (nil, 0, 0, "expected 'xref' at offset " + string offset);

	pos += 4;
	pos = skipws(data, pos);

	maxobj := 0;
	entries: list of (int, int, array of ref XrefEntry);

	for(;;){
		if(pos >= len data)
			break;
		if(pos + 7 <= len data && slicestr(data, pos, 7) == "trailer")
			break;

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

		sect := array[count] of ref XrefEntry;
		for(i := 0; i < count; i++){
			(eoff, p3) := readint(data, pos);
			pos = skipws(data, p3);
			(egen, p4) := readint(data, pos);
			pos = skipws(data, p4);
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

	xref := array[maxobj] of ref XrefEntry;
	for(; entries != nil; entries = tl entries){
		(sobj, cnt, sect) := hd entries;
		for(i := 0; i < cnt; i++)
			xref[sobj + i] = sect[i];
	}

	trailerpos := pos;
	if(trailerpos + 7 <= len data && slicestr(data, trailerpos, 7) == "trailer")
		trailerpos += 7;
	trailerpos = skipws(data, trailerpos);

	return (xref, maxobj, trailerpos, nil);
}

parsexrefstream(data: array of byte, offset: int): (array of ref XrefEntry, int, ref PdfObj, string)
{
	pos := offset;
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

	(obj, nil, perr) := parseobj(data, pos);
	if(obj == nil)
		return (nil, 0, nil, "cannot parse xref stream object: " + perr);
	if(obj.kind != Ostream)
		return (nil, 0, nil, "xref stream object is not a stream");

	typeobj := dictget(obj.dval, "Type");
	if(typeobj == nil || typeobj.kind != Oname || typeobj.sval != "XRef")
		return (nil, 0, nil, "/Type is not /XRef");

	size := dictgetint(obj.dval, "Size");
	if(size <= 0)
		return (nil, 0, nil, "missing or invalid /Size");

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
	ww := array[3] of {* => 0};
	i := 0;
	for(wr := wvals; wr != nil; wr = tl wr)
		i++;
	if(i != 3)
		return (nil, 0, nil, sys->sprint("/W has %d entries, expected 3", i));
	i = 0;
	for(wr = wvals; wr != nil; wr = tl wr){
		ww[2 - i] = hd wr;
		i++;
	}

	entrysize := ww[0] + ww[1] + ww[2];
	if(entrysize <= 0)
		return (nil, 0, nil, "invalid /W field widths");

	idxobj := dictget(obj.dval, "Index");
	subsections: list of (int, int);
	if(idxobj != nil && idxobj.kind == Oarray){
		il := idxobj.aval;
		for(;;){
			if(il == nil) break;
			sobj := hd il; il = tl il;
			if(il == nil) break;
			cobj := hd il; il = tl il;
			sv := 0; cv := 0;
			if(sobj.kind == Oint) sv = sobj.ival;
			if(cobj.kind == Oint) cv = cobj.ival;
			subsections = (sv, cv) :: subsections;
		}
		rev: list of (int, int);
		for(; subsections != nil; subsections = tl subsections)
			rev = (hd subsections) :: rev;
		subsections = rev;
	} else
		subsections = (0, size) :: nil;

	(sdata, derr) := decompressstream(obj);
	if(sdata == nil)
		return (nil, 0, nil, "cannot decompress xref stream: " + derr);

	xref := array[size] of ref XrefEntry;
	dpos := 0;
	for(sl := subsections; sl != nil; sl = tl sl){
		(startobj, count) := hd sl;
		for(j := 0; j < count; j++){
			if(dpos + entrysize > len sdata)
				break;
			f0 := readfield(sdata, dpos, ww[0]);
			dpos += ww[0];
			f1 := readfield(sdata, dpos, ww[1]);
			dpos += ww[1];
			f2 := readfield(sdata, dpos, ww[2]);
			dpos += ww[2];

			ftype := f0;
			if(ww[0] == 0) ftype = 1;

			objnum := startobj + j;
			if(objnum >= size) break;

			case ftype {
			0 => xref[objnum] = ref XrefEntry(0, f2, 0);
			1 => xref[objnum] = ref XrefEntry(f1, f2, 1);
			2 => xref[objnum] = ref XrefEntry(f1, f2, 2);
			* => xref[objnum] = ref XrefEntry(0, 0, 0);
			}
		}
	}

	trailer := ref PdfObj(Odict, 0, 0.0, nil, nil, obj.dval, nil);
	return (xref, size, trailer, nil);
}

readfield(data: array of byte, pos, width: int): int
{
	v := 0;
	for(i := 0; i < width && pos + i < len data; i++)
		v = (v << 8) | int data[pos + i];
	return v;
}

# ---- Object parser ----

parseobj(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	if(pos >= len data)
		return (nil, pos, "unexpected end of data");
	pos = skipws(data, pos);
	if(pos >= len data)
		return (nil, pos, "unexpected end of data");

	c := int data[pos];

	if(c == '<' && pos+1 < len data && int data[pos+1] == '<')
		return parsedict(data, pos);
	if(c == '<')
		return parsehexstring(data, pos);
	if(c == '(')
		return parselitstring(data, pos);
	if(c == '/')
		return parsename(data, pos);
	if(c == '[')
		return parsearray(data, pos);
	if(c == 't' && pos+4 <= len data && slicestr(data, pos, 4) == "true")
		return (ref PdfObj(Obool, 1, 0.0, nil, nil, nil, nil), pos+4, nil);
	if(c == 'f' && pos+5 <= len data && slicestr(data, pos, 5) == "false")
		return (ref PdfObj(Obool, 0, 0.0, nil, nil, nil, nil), pos+5, nil);
	if(c == 'n' && pos+4 <= len data && slicestr(data, pos, 4) == "null")
		return (ref PdfObj(Onull, 0, 0.0, nil, nil, nil, nil), pos+4, nil);
	if((c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.')
		return parsenumber(data, pos);

	return (nil, pos, "unexpected character: " + string c);
}

parsedict(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	pos += 2;
	pos = skipws(data, pos);
	entries: list of ref DictEntry;

	while(pos < len data){
		pos = skipws(data, pos);
		if(pos >= len data) break;
		if(int data[pos] == '>' && pos+1 < len data && int data[pos+1] == '>'){
			pos += 2;
			break;
		}
		if(int data[pos] != '/')
			return (nil, pos, "expected name key in dict");

		(keyobj, p1, kerr) := parsename(data, pos);
		if(keyobj == nil) return (nil, p1, kerr);
		pos = p1;

		(valobj, p2, verr) := parseobj(data, pos);
		if(valobj == nil) return (nil, p2, verr);
		pos = p2;

		entries = ref DictEntry(keyobj.sval, valobj) :: entries;
	}

	spos := skipws(data, pos);
	if(spos + 6 <= len data && slicestr(data, spos, 6) == "stream")
		return parsestreamdata(data, spos + 6, entries);

	return (ref PdfObj(Odict, 0, 0.0, nil, nil, entries, nil), pos, nil);
}

parsestreamdata(data: array of byte, pos: int,
	entries: list of ref DictEntry): (ref PdfObj, int, string)
{
	if(pos < len data && int data[pos] == '\r') pos++;
	if(pos < len data && int data[pos] == '\n') pos++;

	slen := dictgetint(entries, "Length");
	if(slen <= 0){
		(slen, pos) = findendstream(data, pos);
		if(slen < 0)
			return (nil, pos, "cannot determine stream length");
	}
	if(pos + slen > len data) slen = len data - pos;

	streamdata := array[slen] of byte;
	streamdata[0:] = data[pos:pos+slen];
	pos += slen;

	pos = skipws(data, pos);
	if(pos + 9 <= len data && slicestr(data, pos, 9) == "endstream")
		pos += 9;

	obj := ref PdfObj(Ostream, 0, 0.0, nil, nil, entries, streamdata);
	return (obj, pos, nil);
}

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

parsehexstring(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	pos++;
	s := "";
	nibble := -1;
	while(pos < len data){
		c := int data[pos]; pos++;
		if(c == '>') break;
		if(isws(c)) continue;
		v := hexval(c);
		if(v < 0) continue;
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

parselitstring(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	pos++;
	depth := 1;
	s := "";
	while(pos < len data && depth > 0){
		c := int data[pos]; pos++;
		case c {
		'(' =>
			depth++;
			s[len s] = c;
		')' =>
			depth--;
			if(depth > 0) s[len s] = c;
		'\\' =>
			if(pos < len data){
				ec := int data[pos]; pos++;
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
				* => s[len s] = ec;
				}
			}
		* => s[len s] = c;
		}
	}
	return (ref PdfObj(Ostring, 0, 0.0, s, nil, nil, nil), pos, nil);
}

parsename(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	pos++;
	name := "";
	while(pos < len data){
		c := int data[pos];
		if(isws(c) || c == '/' || c == '<' || c == '>' ||
		   c == '[' || c == ']' || c == '(' || c == ')' ||
		   c == '{' || c == '}' || c == '%')
			break;
		if(c == '#' && pos+2 < len data){
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

parsearray(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	pos++;
	pos = skipws(data, pos);
	items: list of ref PdfObj;
	while(pos < len data){
		pos = skipws(data, pos);
		if(pos >= len data) break;
		if(int data[pos] == ']'){
			pos++;
			break;
		}
		(obj, p, err) := parseobj(data, pos);
		if(obj == nil) return (nil, p, err);
		items = obj :: items;
		pos = p;
	}
	rev: list of ref PdfObj;
	for(; items != nil; items = tl items)
		rev = hd items :: rev;
	return (ref PdfObj(Oarray, 0, 0.0, nil, rev, nil, nil), pos, nil);
}

parsenumber(data: array of byte, pos: int): (ref PdfObj, int, string)
{
	numstr := "";
	isreal := 0;
	start := pos;
	if(pos < len data && (int data[pos] == '-' || int data[pos] == '+')){
		numstr[len numstr] = int data[pos]; pos++;
	}
	while(pos < len data){
		c := int data[pos];
		if(c >= '0' && c <= '9'){
			numstr[len numstr] = c; pos++;
		} else if(c == '.' && !isreal){
			isreal = 1;
			numstr[len numstr] = c; pos++;
		} else
			break;
	}
	if(len numstr == 0)
		return (nil, start, "expected number");
	if(isreal)
		return (ref PdfObj(Oreal, 0, real numstr, nil, nil, nil, nil), pos, nil);

	num := int numstr;
	svpos := pos;
	pos = skipws(data, pos);
	if(pos < len data && int data[pos] >= '0' && int data[pos] <= '9'){
		genstr := "";
		while(pos < len data && int data[pos] >= '0' && int data[pos] <= '9'){
			genstr[len genstr] = int data[pos]; pos++;
		}
		pos = skipws(data, pos);
		if(pos < len data && int data[pos] == 'R'){
			pos++;
			gen := int genstr;
			return (ref PdfObj(Oref, num, real gen, nil, nil, nil, nil), pos, nil);
		}
	}
	return (ref PdfObj(Oint, num, 0.0, nil, nil, nil, nil), svpos, nil);
}

# ---- Object resolution ----

resolve(doc: ref PdfDoc, obj: ref PdfObj): ref PdfObj
{
	if(obj == nil) return nil;
	if(obj.kind != Oref) return obj;

	objnum := obj.ival;
	if(objnum < 0 || objnum >= doc.nobjs) return nil;

	entry := doc.xref[objnum];
	if(entry == nil || entry.inuse == 0) return nil;

	if(entry.inuse == 2)
		return resolveobjstm(doc, entry.offset, entry.gen);

	offset := entry.offset;
	if(offset >= len doc.data) return nil;

	pos := offset;
	(nil, p1) := readint(doc.data, pos);
	pos = skipws(doc.data, p1);
	(nil, p2) := readint(doc.data, pos);
	pos = skipws(doc.data, p2);
	if(pos + 3 <= len doc.data && slicestr(doc.data, pos, 3) == "obj")
		pos += 3;
	pos = skipws(doc.data, pos);

	(parsed, nil, nil) := parseobj(doc.data, pos);
	return parsed;
}

resolveobjstm(doc: ref PdfDoc, stmnum, idx: int): ref PdfObj
{
	if(stmnum < 0 || stmnum >= doc.nobjs) return nil;
	stmentry := doc.xref[stmnum];
	if(stmentry == nil || stmentry.inuse != 1) return nil;

	offset := stmentry.offset;
	if(offset >= len doc.data) return nil;

	pos := offset;
	(nil, p1) := readint(doc.data, pos);
	pos = skipws(doc.data, p1);
	(nil, p2) := readint(doc.data, pos);
	pos = skipws(doc.data, p2);
	if(pos + 3 <= len doc.data && slicestr(doc.data, pos, 3) == "obj")
		pos += 3;
	pos = skipws(doc.data, pos);

	(stmobj, nil, nil) := parseobj(doc.data, pos);
	if(stmobj == nil || stmobj.kind != Ostream) return nil;

	n := dictgetint(stmobj.dval, "N");
	first := dictgetint(stmobj.dval, "First");
	if(n <= 0 || first <= 0 || idx >= n) return nil;

	(sdata, derr) := decompressstream(stmobj);
	if(sdata == nil || derr != nil) return nil;

	spos := 0;
	offsets := array[n] of int;
	for(i := 0; i < n; i++){
		spos = skipwsbytes(sdata, spos);
		(nil, sp1) := readint(sdata, spos);
		spos = skipwsbytes(sdata, sp1);
		(ooff, sp2) := readint(sdata, spos);
		spos = sp2;
		offsets[i] = first + ooff;
	}

	if(idx >= n) return nil;
	opos := offsets[idx];
	if(opos >= len sdata) return nil;

	(parsed, nil, nil) := parseobj(sdata, opos);
	return parsed;
}

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

# ---- Stream decompression ----

decompressstream(obj: ref PdfObj): (array of byte, string)
{
	if(obj == nil || obj.kind != Ostream)
		return (nil, "not a stream");
	raw := obj.stream;
	if(raw == nil)
		return (nil, "empty stream");

	filterobj := dictget(obj.dval, "Filter");
	if(filterobj == nil)
		return (raw, nil);

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

	return (raw, nil);
}

inflate(data: array of byte): (array of byte, string)
{
	filtermod = load Filter Filter->INFLATEPATH;
	if(filtermod == nil)
		return (nil, sys->sprint("cannot load inflate: %r"));

	filtermod->init();
	rqchan := filtermod->start("z");

	rq := <-rqchan;
	pick r := rq {
	Start => ;
	* => return (nil, "inflate: unexpected initial message");
	}

	result: list of array of byte;
	resultlen := 0;
	inpos := 0;
	done := 0;

	while(!done){
		rq = <-rqchan;
		pick r := rq {
		Fill =>
			n := len data - inpos;
			if(n > len r.buf) n = len r.buf;
			if(n > 0) r.buf[0:] = data[inpos:inpos+n];
			inpos += n;
			r.reply <-= n;
		Result =>
			chunk := array[len r.buf] of byte;
			chunk[0:] = r.buf;
			result = chunk :: result;
			resultlen += len chunk;
			r.reply <-= 0;
		Info => ;
		Finished => done = 1;
		Error => return (nil, "inflate error: " + r.e);
		* => done = 1;
		}
	}

	out := array[resultlen] of byte;
	pos := resultlen;
	for(; result != nil; result = tl result){
		chunk := hd result;
		pos -= len chunk;
		out[pos:] = chunk;
	}
	return (out, nil);
}

asciihexdecode(data: array of byte): (array of byte, string)
{
	out := array[len data / 2 + 1] of byte;
	n := 0;
	nibble := -1;
	for(i := 0; i < len data; i++){
		c := int data[i];
		if(c == '>') break;
		if(isws(c)) continue;
		v := hexval(c);
		if(v < 0) continue;
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

# ---- Text extraction ----

extracttext(doc: ref PdfDoc): (string, string)
{
	root := dictget(doc.trailer.dval, "Root");
	if(root == nil) return (nil, "no Root in trailer");
	root = resolve(doc, root);
	if(root == nil) return (nil, "cannot resolve Root");

	pages := dictget(root.dval, "Pages");
	if(pages == nil) return (nil, "no Pages in catalog");
	pages = resolve(doc, pages);
	if(pages == nil) return (nil, "cannot resolve Pages");

	text := "";
	pagenum := 0;
	(text, pagenum) = extractpages(doc, pages, text, pagenum);
	if(pagenum == 0)
		return (nil, "no pages found");
	return (text, nil);
}

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
		kids := dictget(node.dval, "Kids");
		if(kids != nil && kids.kind == Oarray){
			for(k := kids.aval; k != nil; k = tl k){
				child := resolve(doc, hd k);
				if(child != nil)
					(text, pagenum) = extractpages(doc, child, text, pagenum);
			}
		}
	} else if(typ == "Page"){
		pagenum++;
		if(len text > 0) text += "\n\n";
		if(pagenum > 1)
			text += "--- Page " + string pagenum + " ---\n\n";

		fontmap := buildfontmap(doc, node);
		contents := dictget(node.dval, "Contents");
		if(contents != nil){
			pagetext := extractpagetext_cs(doc, contents, fontmap);
			if(pagetext != nil)
				text += pagetext;
		}
	}
	return (text, pagenum);
}

# Extract text for a single page (public API)
extractpagetext_full(doc: ref PdfDoc, page: ref PdfObj): string
{
	fontmap := buildfontmap(doc, page);
	contents := dictget(page.dval, "Contents");
	if(contents == nil)
		return nil;
	return extractpagetext_cs(doc, contents, fontmap);
}

extractpagetext_cs(doc: ref PdfDoc, contents: ref PdfObj,
	fontmap: list of ref FontMapEntry): string
{
	if(contents == nil) return nil;
	contents = resolve(doc, contents);
	if(contents == nil) return nil;

	if(contents.kind == Oarray){
		text := "";
		for(a := contents.aval; a != nil; a = tl a){
			stream := resolve(doc, hd a);
			if(stream != nil){
				t := extractstreamtext(doc, stream, fontmap);
				if(t != nil) text += t;
			}
		}
		return text;
	}
	if(contents.kind == Ostream)
		return extractstreamtext(doc, contents, fontmap);
	return nil;
}

extractstreamtext(doc: ref PdfDoc, stream: ref PdfObj,
	fontmap: list of ref FontMapEntry): string
{
	(data, nil) := decompressstream(stream);
	if(data == nil) return nil;
	return parsecontentstream_text(data, fontmap);
}

# Parse content stream for text extraction only
parsecontentstream_text(data: array of byte, fontmap: list of ref FontMapEntry): string
{
	text := "";
	pos := 0;
	operands: list of string;
	curfont: ref FontMapEntry;

	while(pos < len data){
		pos = skipws(data, pos);
		if(pos >= len data) break;
		c := int data[pos];

		if(c == '('){
			(s, newpos) := readlitstr(data, pos);
			operands = s :: operands;
			pos = newpos;
			continue;
		}
		if(c == '<' && (pos+1 >= len data || int data[pos+1] != '<')){
			(s, newpos) := readhexstr(data, pos);
			operands = s :: operands;
			pos = newpos;
			continue;
		}
		if(c == '['){
			(s, newpos) := readtjarray(data, pos, curfont);
			operands = s :: operands;
			pos = newpos;
			continue;
		}
		if(c == '<' && pos+1 < len data && int data[pos+1] == '<'){
			pos = skipdict(data, pos);
			continue;
		}
		if(c == '/'){
			(tok, newpos) := readcsname(data, pos);
			operands = tok :: operands;
			pos = newpos;
			continue;
		}
		if((c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.'){
			(tok, newpos) := readtoken(data, pos);
			operands = tok :: operands;
			pos = newpos;
			continue;
		}
		if((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		   c == '\'' || c == '"' || c == '*'){
			(op, newpos) := readtoken(data, pos);
			pos = newpos;

			case op {
			"Tj" =>
				if(operands != nil){
					s := hd operands;
					if(curfont != nil)
						s = decodecidstr(s, curfont);
					text += cleanpdftext(s);
				}
			"TJ" =>
				if(operands != nil)
					text += cleanpdftext(hd operands);
			"'" =>
				if(operands != nil){
					s := hd operands;
					if(curfont != nil)
						s = decodecidstr(s, curfont);
					text += "\n" + cleanpdftext(s);
				}
			"\"" =>
				if(operands != nil){
					s := hd operands;
					if(curfont != nil)
						s = decodecidstr(s, curfont);
					text += "\n" + cleanpdftext(s);
				}
			"Td" or "TD" =>
				if(operands != nil && tl operands != nil){
					ty := real (hd operands);
					tx := real (hd tl operands);
					if(ty < -1.5 || ty > 1.5)
						text += "\n";
					else if(tx > 5.0)
						text += " ";
				}
			"T*" =>
				text += "\n";
			"Tf" =>
				if(operands != nil && tl operands != nil)
					curfont = fontmaplookup(fontmap, hd tl operands);
			"BI" =>
				pos = skipinlineimage(data, pos);
			}
			operands = nil;
			continue;
		}
		if(c == '%'){
			while(pos < len data && int data[pos] != '\n')
				pos++;
			continue;
		}
		pos++;
	}
	return text;
}

# ---- Content stream reading helpers ----

readlitstr(data: array of byte, pos: int): (string, int)
{
	pos++;
	depth := 1;
	s := "";
	while(pos < len data && depth > 0){
		c := int data[pos]; pos++;
		case c {
		'(' =>
			depth++;
			s[len s] = c;
		')' =>
			depth--;
			if(depth > 0) s[len s] = c;
		'\\' =>
			if(pos < len data){
				ec := int data[pos]; pos++;
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
				* => s[len s] = ec;
				}
			}
		* => s[len s] = c;
		}
	}
	return (s, pos);
}

readhexstr(data: array of byte, pos: int): (string, int)
{
	pos++;
	s := "";
	nibble := -1;
	while(pos < len data){
		c := int data[pos]; pos++;
		if(c == '>') break;
		if(isws(c)) continue;
		v := hexval(c);
		if(v < 0) continue;
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

readtjarray(data: array of byte, pos: int, curfont: ref FontMapEntry): (string, int)
{
	pos++;
	s := "";
	while(pos < len data){
		pos = skipws(data, pos);
		if(pos >= len data) break;
		c := int data[pos];
		if(c == ']'){
			pos++;
			break;
		}
		if(c == '('){
			(substr, newpos) := readlitstr(data, pos);
			if(curfont != nil) substr = decodecidstr(substr, curfont);
			s += substr;
			pos = newpos;
			continue;
		}
		if(c == '<'){
			(substr, newpos) := readhexstr(data, pos);
			if(curfont != nil) substr = decodecidstr(substr, curfont);
			s += substr;
			pos = newpos;
			continue;
		}
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

readcsname(data: array of byte, pos: int): (string, int)
{
	pos++;
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

skipinlineimage(data: array of byte, pos: int): int
{
	while(pos < len data - 1){
		if(int data[pos] == 'I' && int data[pos+1] == 'D'){
			pos += 2;
			break;
		}
		pos++;
	}
	while(pos < len data - 1){
		if(int data[pos] == 'E' && int data[pos+1] == 'I'){
			if(pos > 0 && isws(int data[pos-1])){
				pos += 2;
				return pos;
			}
		}
		pos++;
	}
	return pos;
}

skipdict(data: array of byte, pos: int): int
{
	pos += 2;
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

cleanpdftext(s: string): string
{
	if(s == nil) return nil;
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

# ---- ToUnicode CMap support ----

parsecmap(text: string): (int, list of ref CMapEntry)
{
	entries: list of ref CMapEntry;
	twobyte := 0;
	pos := 0;
	tlen := len text;

	while(pos < tlen){
		if(pos + 19 <= tlen && text[pos:pos+19] == "begincodespacerange"){
			pos += 19;
			while(pos < tlen && text[pos] != '<') pos++;
			if(pos < tlen){
				(nil, np) := parsecmaphex(text, pos);
				hstart := pos + 1;
				ndigits := 0;
				for(h := hstart; h < tlen && text[h] != '>'; h++)
					ndigits++;
				if(ndigits >= 4) twobyte = 1;
				pos = np;
			}
			continue;
		}
		if(pos + 11 <= tlen && text[pos:pos+11] == "beginbfchar"){
			pos += 11;
			for(;;){
				while(pos < tlen && (text[pos] == ' ' || text[pos] == '\n' || text[pos] == '\r' || text[pos] == '\t'))
					pos++;
				if(pos + 9 <= tlen && text[pos:pos+9] == "endbfchar")
					break;
				if(pos >= tlen) break;
				if(text[pos] != '<'){
					pos++;
					continue;
				}
				(cid, np1) := parsecmaphex(text, pos);
				pos = np1;
				while(pos < tlen && text[pos] != '<') pos++;
				if(pos >= tlen) break;
				(uni, np2) := parsecmaphex(text, pos);
				pos = np2;
				entries = ref CMapEntry(cid, cid, uni) :: entries;
			}
			continue;
		}
		if(pos + 12 <= tlen && text[pos:pos+12] == "beginbfrange"){
			pos += 12;
			for(;;){
				while(pos < tlen && (text[pos] == ' ' || text[pos] == '\n' || text[pos] == '\r' || text[pos] == '\t'))
					pos++;
				if(pos + 10 <= tlen && text[pos:pos+10] == "endbfrange")
					break;
				if(pos >= tlen) break;
				if(text[pos] != '<'){
					pos++;
					continue;
				}
				(lo, np1) := parsecmaphex(text, pos);
				pos = np1;
				while(pos < tlen && text[pos] != '<') pos++;
				if(pos >= tlen) break;
				(hi, np2) := parsecmaphex(text, pos);
				pos = np2;
				while(pos < tlen && text[pos] != '<') pos++;
				if(pos >= tlen) break;
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

parsecmaphex(s: string, pos: int): (int, int)
{
	slen := len s;
	if(pos >= slen || s[pos] != '<')
		return (0, pos);
	pos++;
	val := 0;
	while(pos < slen && s[pos] != '>'){
		c := s[pos]; pos++;
		v := hexval(c);
		if(v >= 0) val = (val << 4) | v;
	}
	if(pos < slen && s[pos] == '>') pos++;
	return (val, pos);
}

buildfontmap(doc: ref PdfDoc, page: ref PdfObj): list of ref FontMapEntry
{
	if(page == nil) return nil;

	resources := dictget(page.dval, "Resources");
	if(resources == nil) return nil;
	resources = resolve(doc, resources);
	if(resources == nil) return nil;

	fonts := dictget(resources.dval, "Font");
	if(fonts == nil) return nil;
	fonts = resolve(doc, fonts);
	if(fonts == nil || (fonts.kind != Odict && fonts.kind != Ostream))
		return nil;

	fontmap: list of ref FontMapEntry;
	for(fl := fonts.dval; fl != nil; fl = tl fl){
		de := hd fl;
		fontname := de.key;
		fontobj := resolve(doc, de.val);
		if(fontobj == nil) continue;

		twobyte := 0;
		fentries: list of ref CMapEntry;

		enc := dictget(fontobj.dval, "Encoding");
		if(enc != nil){
			enc = resolve(doc, enc);
			if(enc != nil && enc.kind == Oname && enc.sval == "Identity-H")
				twobyte = 1;
		}

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
					if(tb) twobyte = 1;
					fentries = ent;
				}
			}
		}

		if(fentries != nil || twobyte)
			fontmap = ref FontMapEntry(fontname, twobyte, fentries) :: fontmap;
	}
	return fontmap;
}

cmaplookup(entries: list of ref CMapEntry, cid: int): int
{
	for(; entries != nil; entries = tl entries){
		e := hd entries;
		if(cid >= e.lo && cid <= e.hi)
			return e.unicode + (cid - e.lo);
	}
	return cid;
}

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
		if(cid == 0) continue;
		uni := cmaplookup(fm.entries, cid);
		if(uni > 0) out[len out] = uni;
	}
	return out;
}

fontmaplookup(fontmap: list of ref FontMapEntry, name: string): ref FontMapEntry
{
	for(; fontmap != nil; fontmap = tl fontmap){
		fm := hd fontmap;
		if(fm.name == name)
			return fm;
	}
	return nil;
}

# ---- Utility functions ----

dictget(entries: list of ref DictEntry, key: string): ref PdfObj
{
	for(; entries != nil; entries = tl entries){
		e := hd entries;
		if(e.key == key) return e.val;
	}
	return nil;
}

dictgetint(entries: list of ref DictEntry, key: string): int
{
	obj := dictget(entries, key);
	if(obj == nil) return 0;
	if(obj.kind == Oint) return obj.ival;
	return 0;
}

slicestr(data: array of byte, pos, length: int): string
{
	if(pos + length > len data) length = len data - pos;
	s := "";
	for(i := 0; i < length; i++)
		s[len s] = int data[pos + i];
	return s;
}

readint(data: array of byte, pos: int): (int, int)
{
	start := pos;
	while(pos < len data && int data[pos] >= '0' && int data[pos] <= '9')
		pos++;
	if(pos == start) return (0, start);
	return (int slicestr(data, start, pos - start), pos);
}

skipws(data: array of byte, pos: int): int
{
	while(pos < len data){
		c := int data[pos];
		if(c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == 0)
			pos++;
		else if(c == '%'){
			while(pos < len data && int data[pos] != '\n')
				pos++;
		} else
			break;
	}
	return pos;
}

isws(c: int): int
{
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == 0;
}

hexval(c: int): int
{
	if(c >= '0' && c <= '9') return c - '0';
	if(c >= 'a' && c <= 'f') return c - 'a' + 10;
	if(c >= 'A' && c <= 'F') return c - 'A' + 10;
	return -1;
}
