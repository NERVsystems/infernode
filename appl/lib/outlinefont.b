implement OutlineFont;

#
# Outline font renderer — CFF (Compact Font Format) backend.
#
# Parses CFF font programs, interprets Type 2 charstrings to extract
# glyph outlines, and rasterizes them via Draw->fillpoly.
#
# References:
#   Adobe Technical Note #5176 — CFF specification
#   Adobe Technical Note #5177 — Type 2 Charstring Format
#

include "sys.m";
	sys: Sys;

include "draw.m";
	drawm: Draw;
	Display, Image, Rect, Point: import drawm;

include "math.m";
	math: Math;

include "outlinefont.m";

# ---- Internal types ----

# Path segment for glyph outlines (font units)
PathSeg: adt {
	pick {
	Move =>
		x, y: real;
	Line =>
		x, y: real;
	Curve =>
		x1, y1, x2, y2, x3, y3: real;	# cubic Bezier
	Close =>
	}
};

# CFF INDEX data
CffIndex: adt {
	count:	int;
	data:	array of array of byte;
};

# CFF top-level DICT values
CffTopDict: adt {
	charstrings_off:	int;
	charset_off:		int;
	encoding_off:		int;
	private_size:		int;
	private_off:		int;
	fdarray_off:		int;
	fdselect_off:		int;
	ros:			int;	# 1 if CIDFont
	fontname:		string;
	# Font metrics (font units; defaults per spec)
	ascent:			int;
	descent:		int;
};

# CFF private DICT values
CffPrivateDict: adt {
	subrs_off:		int;	# relative to private dict start
	defaultw:		int;
	nominalw:		int;
};

# Per-glyph outline data
GlyphOutline: adt {
	path:	list of ref PathSeg;
	width:	int;	# advance width in font units
};

# Glyph cache entry
CacheEntry: adt {
	faceidx:	int;	# face index (different fonts have different GID assignments)
	gid:	int;
	qsize:	int;	# quantized size (size * 4, as int)
	img:	ref Image;
	width:	int;	# advance width in pixels
	ox, oy:	int;	# offset from draw point to image origin
};

# Internal face data (opaque to consumers)
FaceData: adt {
	cffdata:	array of byte;
	nglyphs:	int;
	upem:		int;
	ascent:		int;
	descent:	int;
	fontname:	string;
	charstrings:	ref CffIndex;
	gsubrs:		ref CffIndex;
	# For non-CID fonts: single private dict + local subrs
	privdict:	ref CffPrivateDict;
	lsubrs:		ref CffIndex;
	# For CID fonts: per-FD private dicts + local subrs
	iscid:		int;
	fdcount:	int;
	fdprivate:	array of ref CffPrivateDict;
	fdlsubrs:	array of ref CffIndex;
	fdselect:	array of byte;	# gid -> fd index
	# CID -> GID mapping (for CID-keyed fonts)
	cidmap:		array of int;	# indexed by CID, value is GID; -1 = unmapped
};

# Module state
display: ref Display;
facetab: array of ref FaceData;
nfaces: int;
cachetab: list of ref CacheEntry;
MAXCACHE: con 512;
ncached: int;

init(d: ref Display)
{
	sys = load Sys Sys->PATH;
	drawm = load Draw Draw->PATH;
	math = load Math Math->PATH;
	display = d;
	facetab = array[8] of ref FaceData;
	nfaces = 0;
	cachetab = nil;
	ncached = 0;
}

open(data: array of byte, format: string): (ref Face, string)
{
	if(format != "cff")
		return (nil, "unsupported format: " + format);
	if(len data < 4)
		return (nil, "data too small");

	(fd, err) := parsecff(data);
	if(fd == nil)
		return (nil, err);

	# Store face data
	idx := addface(fd);

	face := ref Face(
		fd.nglyphs,
		fd.upem,
		fd.ascent,
		fd.descent,
		fd.fontname,
		fd.iscid
	);
	# Stash index in ascent field? No — use a parallel lookup.
	# We'll use the face pointer identity via the name+nglyphs combo,
	# but simpler: store idx in a field we control.
	# Actually, Face is an adt the caller owns.  We need a way to map
	# Face -> FaceData.  Use the face index stored as a negative descent
	# sentinel?  No, too hacky.
	# Simplest: face identity by sequential index.  We'll add a hidden
	# field or use name encoding.  For now, encode index in name suffix.
	face.name = fd.fontname + "\t" + string idx;

	return (face, nil);
}

addface(fd: ref FaceData): int
{
	if(nfaces >= len facetab){
		newtab := array[len facetab * 2] of ref FaceData;
		newtab[0:] = facetab;
		facetab = newtab;
	}
	idx := nfaces;
	facetab[idx] = fd;
	nfaces++;
	return idx;
}

getfaceidx(f: ref Face): int
{
	nm := f.name;
	for(i := len nm - 1; i >= 0; i--){
		if(nm[i] == '\t')
			return int nm[i+1:];
	}
	return -1;
}

getfacedata(f: ref Face): ref FaceData
{
	idx := getfaceidx(f);
	if(idx >= 0 && idx < nfaces)
		return facetab[idx];
	return nil;
}

Face.cidtogid(f: self ref Face, cid: int): int
{
	fd := getfacedata(f);
	if(fd == nil || fd.cidmap == nil)
		return -1;
	if(cid < 0 || cid >= len fd.cidmap)
		return -1;
	return fd.cidmap[cid];
}

Face.drawglyph(f: self ref Face, gid: int, size: real,
	dst: ref Image, p: Point, src: ref Image): int
{
	if(display == nil || dst == nil || src == nil)
		return 0;

	fidx := getfaceidx(f);
	fd := getfacedata(f);
	if(fd == nil)
		return 0;

	if(gid < 0 || gid >= fd.nglyphs)
		return 0;

	# Check cache
	qsize := int (size * 4.0 + 0.5);
	ce := cachelookup(fidx, gid, qsize);
	if(ce != nil){
		if(ce.img != nil)
			dst.draw(Rect(
				(p.x + ce.ox, p.y + ce.oy),
				(p.x + ce.ox + ce.img.r.dx(), p.y + ce.oy + ce.img.r.dy())),
				src, ce.img, Point(0, 0));
		return ce.width;
	}

	# Extract outline
	outline := getoutline(fd, gid);
	if(outline == nil)
		return 0;

	# Compute advance width in pixels
	scale := size / real fd.upem;
	advpx := int (real outline.width * scale + 0.5);

	# Rasterize
	if(outline.path != nil){
		(gimg, ox, oy) := rasterize(outline.path, scale);
		cachestore(fidx, gid, qsize, gimg, advpx, ox, oy);
		if(gimg != nil)
			dst.draw(Rect(
				(p.x + ox, p.y + oy),
				(p.x + ox + gimg.r.dx(), p.y + oy + gimg.r.dy())),
				src, gimg, Point(0, 0));
	} else {
		cachestore(fidx, gid, qsize, nil, advpx, 0, 0);
	}

	return advpx;
}

Face.glyphwidth(f: self ref Face, gid: int, size: real): int
{
	fidx := getfaceidx(f);
	fd := getfacedata(f);
	if(fd == nil)
		return 0;

	if(gid < 0 || gid >= fd.nglyphs)
		return 0;

	# Check cache
	qsize := int (size * 4.0 + 0.5);
	ce := cachelookup(fidx, gid, qsize);
	if(ce != nil)
		return ce.width;

	# Extract outline for width
	outline := getoutline(fd, gid);
	if(outline == nil)
		return 0;

	scale := size / real fd.upem;
	return int (real outline.width * scale + 0.5);
}

Face.metrics(f: self ref Face, size: real): (int, int, int)
{
	fd := getfacedata(f);
	if(fd == nil)
		return (0, 0, 0);

	scale := size / real fd.upem;
	asc := int (real fd.ascent * scale + 0.5);
	desc := int (real fd.descent * scale - 0.5);	# descent is negative
	if(desc > 0) desc = -desc;
	height := asc - desc;
	return (height, asc, desc);
}

# ---- Glyph cache ----

cachelookup(faceidx, gid, qsize: int): ref CacheEntry
{
	for(cl := cachetab; cl != nil; cl = tl cl){
		ce := hd cl;
		if(ce.faceidx == faceidx && ce.gid == gid && ce.qsize == qsize)
			return ce;
	}
	return nil;
}

cachestore(faceidx, gid, qsize: int, img: ref Image, width, ox, oy: int)
{
	# Evict oldest if full
	if(ncached >= MAXCACHE){
		# Remove last quarter
		keep: list of ref CacheEntry;
		n := 0;
		for(cl := cachetab; cl != nil; cl = tl cl){
			if(n < MAXCACHE * 3 / 4)
				keep = hd cl :: keep;
			n++;
		}
		cachetab = keep;
		ncached = MAXCACHE * 3 / 4;
	}
	cachetab = ref CacheEntry(faceidx, gid, qsize, img, width, ox, oy) :: cachetab;
	ncached++;
}

# ---- Outline extraction ----

getoutline(fd: ref FaceData, gid: int): ref GlyphOutline
{
	if(gid < 0 || gid >= fd.charstrings.count)
		return nil;

	csdata := fd.charstrings.data[gid];
	if(csdata == nil || len csdata == 0)
		return ref GlyphOutline(nil, 0);

	# Select private dict and local subrs based on FD
	privd := fd.privdict;
	lsubrs := fd.lsubrs;
	if(fd.iscid && fd.fdselect != nil && gid < len fd.fdselect){
		fdi := int fd.fdselect[gid];
		if(fdi >= 0 && fdi < fd.fdcount){
			if(fd.fdprivate != nil && fdi < len fd.fdprivate)
				privd = fd.fdprivate[fdi];
			if(fd.fdlsubrs != nil && fdi < len fd.fdlsubrs)
				lsubrs = fd.fdlsubrs[fdi];
		}
	}

	nomw := 0;
	defw := 0;
	if(privd != nil){
		nomw = privd.nominalw;
		defw = privd.defaultw;
	}

	return interpcharstring(csdata, fd.gsubrs, lsubrs, nomw, defw);
}

# ---- Type 2 charstring interpreter ----

# Operand stack
T2MAXSTACK: con 48;

interpcharstring(csdata: array of byte, gsubrs, lsubrs: ref CffIndex,
	nominalw, defaultw: int): ref GlyphOutline
{
	stack := array[T2MAXSTACK] of { * => 0.0 };
	sp := 0;
	path: list of ref PathSeg;
	cx := 0.0;
	cy := 0.0;
	width := defaultw;
	widthset := 0;
	nhints := 0;
	firstmove := 1;

	# Call stack for subrs
	MAXCALLSTACK: con 10;
	callstack := array[MAXCALLSTACK] of {* => (array[0] of byte, 0)};
	calldepth := 0;

	data := csdata;
	pos := 0;

	for(;;){
		if(pos >= len data){
			# Return from subr?
			if(calldepth > 0){
				calldepth--;
				(data, pos) = callstack[calldepth];
				continue;
			}
			break;
		}

		b0 := int data[pos];
		pos++;

		# ---- Number encoding ----
		if(b0 >= 32 && b0 <= 246){
			if(sp < T2MAXSTACK)
				stack[sp++] = real (b0 - 139);
			continue;
		}
		if(b0 >= 247 && b0 <= 250){
			if(pos >= len data) break;
			b1 := int data[pos]; pos++;
			if(sp < T2MAXSTACK)
				stack[sp++] = real ((b0 - 247) * 256 + b1 + 108);
			continue;
		}
		if(b0 >= 251 && b0 <= 254){
			if(pos >= len data) break;
			b1 := int data[pos]; pos++;
			if(sp < T2MAXSTACK)
				stack[sp++] = real (-(b0 - 251) * 256 - b1 - 108);
			continue;
		}
		if(b0 == 255){
			# 5-byte fixed point: 16.16
			if(pos + 4 > len data) break;
			v := (int data[pos] << 24) | (int data[pos+1] << 16) |
			     (int data[pos+2] << 8) | int data[pos+3];
			pos += 4;
			# int is 32-bit signed — sign extension is automatic
			if(sp < T2MAXSTACK)
				stack[sp++] = real v / 65536.0;
			continue;
		}

		# ---- Two-byte operators (escape) ----
		if(b0 == 12){
			if(pos >= len data) break;
			b1 := int data[pos]; pos++;
			case b1 {
			34 =>	# hflex
				if(sp >= 7){
					dx1 := stack[0]; dy1 := 0.0;
					dx2 := stack[1]; dy2 := stack[2];
					dx3 := stack[3]; dy3 := 0.0;
					dx4 := stack[4]; dy4 := 0.0;
					dx5 := stack[5]; dy5 := -dy2;
					dx6 := stack[6]; dy6 := 0.0;
					x1 := cx + dx1; y1 := cy + dy1;
					x2 := x1 + dx2; y2 := y1 + dy2;
					x3 := x2 + dx3; y3 := y2 + dy3;
					path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
					cx = x3; cy = y3;
					x4 := cx + dx4; y4 := cy + dy4;
					x5 := x4 + dx5; y5 := y4 + dy5;
					x6 := x5 + dx6; y6 := y5 + dy6;
					path = ref PathSeg.Curve(x4, y4, x5, y5, x6, y6) :: path;
					cx = x6; cy = y6;
				}
				sp = 0;
			35 =>	# flex
				if(sp >= 13){
					dx1 := stack[0]; dy1 := stack[1];
					dx2 := stack[2]; dy2 := stack[3];
					dx3 := stack[4]; dy3 := stack[5];
					dx4 := stack[6]; dy4 := stack[7];
					dx5 := stack[8]; dy5 := stack[9];
					dx6 := stack[10]; dy6 := stack[11];
					# stack[12] is fd (flex depth), ignored
					x1 := cx + dx1; y1 := cy + dy1;
					x2 := x1 + dx2; y2 := y1 + dy2;
					x3 := x2 + dx3; y3 := y2 + dy3;
					path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
					cx = x3; cy = y3;
					x4 := cx + dx4; y4 := cy + dy4;
					x5 := x4 + dx5; y5 := y4 + dy5;
					x6 := x5 + dx6; y6 := y5 + dy6;
					path = ref PathSeg.Curve(x4, y4, x5, y5, x6, y6) :: path;
					cx = x6; cy = y6;
				}
				sp = 0;
			36 =>	# hflex1
				if(sp >= 9){
					dx1 := stack[0]; dy1 := stack[1];
					dx2 := stack[2]; dy2 := stack[3];
					dx3 := stack[4]; dy3 := 0.0;
					dx4 := stack[5]; dy4 := 0.0;
					dx5 := stack[6]; dy5 := stack[7];
					dx6 := stack[8];
					dy6 := -(dy1 + dy2 + dy3 + dy4 + dy5);
					x1 := cx + dx1; y1 := cy + dy1;
					x2 := x1 + dx2; y2 := y1 + dy2;
					x3 := x2 + dx3; y3 := y2 + dy3;
					path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
					cx = x3; cy = y3;
					x4 := cx + dx4; y4 := cy + dy4;
					x5 := x4 + dx5; y5 := y4 + dy5;
					x6 := x5 + dx6; y6 := y5 + dy6;
					path = ref PathSeg.Curve(x4, y4, x5, y5, x6, y6) :: path;
					cx = x6; cy = y6;
				}
				sp = 0;
			37 =>	# flex1
				if(sp >= 11){
					dx1 := stack[0]; dy1 := stack[1];
					dx2 := stack[2]; dy2 := stack[3];
					dx3 := stack[4]; dy3 := stack[5];
					dx4 := stack[6]; dy4 := stack[7];
					dx5 := stack[8]; dy5 := stack[9];
					# last arg is either dx6 or dy6
					sdx := dx1+dx2+dx3+dx4+dx5;
					sdy := dy1+dy2+dy3+dy4+dy5;
					dx6 := 0.0;
					dy6 := 0.0;
					if(fabs(sdx) > fabs(sdy)){
						dx6 = stack[10];
						dy6 = -sdy;
					} else {
						dx6 = -sdx;
						dy6 = stack[10];
					}
					x1 := cx + dx1; y1 := cy + dy1;
					x2 := x1 + dx2; y2 := y1 + dy2;
					x3 := x2 + dx3; y3 := y2 + dy3;
					path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
					cx = x3; cy = y3;
					x4 := cx + dx4; y4 := cy + dy4;
					x5 := x4 + dx5; y5 := y4 + dy5;
					x6 := x5 + dx6; y6 := y5 + dy6;
					path = ref PathSeg.Curve(x4, y4, x5, y5, x6, y6) :: path;
					cx = x6; cy = y6;
				}
				sp = 0;
			* =>
				# Other 2-byte ops: arithmetic, etc. — ignore
				sp = 0;
			}
			continue;
		}

		# ---- Single-byte operators ----
		case b0 {
		1 or 3 or 18 or 23 =>
			# hstem, vstem, hstemhm, vstemhm
			# Consume hint pairs; check for width
			if(!widthset && (sp & 1) != 0){
				width = int stack[0] + nominalw;
				widthset = 1;
				# Shift stack down by 1
				for(si := 0; si < sp - 1; si++)
					stack[si] = stack[si+1];
				sp--;
			}
			nhints += sp / 2;
			sp = 0;
		19 or 20 =>
			# hintmask, cntrmask
			if(!widthset && (sp & 1) != 0){
				width = int stack[0] + nominalw;
				widthset = 1;
				for(si := 0; si < sp - 1; si++)
					stack[si] = stack[si+1];
				sp--;
			}
			nhints += sp / 2;
			sp = 0;
			# Skip mask bytes
			nbytes := (nhints + 7) / 8;
			pos += nbytes;
			if(pos > len data) pos = len data;
		21 =>
			# rmoveto
			if(!widthset && sp > 2){
				width = int stack[0] + nominalw;
				widthset = 1;
				stack[0] = stack[sp-2];
				stack[1] = stack[sp-1];
				sp = 2;
			}
			widthset = 1;
			if(sp >= 2){
				if(!firstmove)
					path = ref PathSeg.Close :: path;
				firstmove = 0;
				cx += stack[0]; cy += stack[1];
				path = ref PathSeg.Move(cx, cy) :: path;
			}
			sp = 0;
		22 =>
			# hmoveto
			if(!widthset && sp > 1){
				width = int stack[0] + nominalw;
				widthset = 1;
				stack[0] = stack[sp-1];
				sp = 1;
			}
			widthset = 1;
			if(sp >= 1){
				if(!firstmove)
					path = ref PathSeg.Close :: path;
				firstmove = 0;
				cx += stack[0];
				path = ref PathSeg.Move(cx, cy) :: path;
			}
			sp = 0;
		4 =>
			# vmoveto
			if(!widthset && sp > 1){
				width = int stack[0] + nominalw;
				widthset = 1;
				stack[0] = stack[sp-1];
				sp = 1;
			}
			widthset = 1;
			if(sp >= 1){
				if(!firstmove)
					path = ref PathSeg.Close :: path;
				firstmove = 0;
				cy += stack[0];
				path = ref PathSeg.Move(cx, cy) :: path;
			}
			sp = 0;
		5 =>
			# rlineto
			i := 0;
			while(i + 1 < sp){
				cx += stack[i]; cy += stack[i+1];
				path = ref PathSeg.Line(cx, cy) :: path;
				i += 2;
			}
			sp = 0;
		6 =>
			# hlineto — alternating horizontal/vertical lines
			i := 0;
			while(i < sp){
				cx += stack[i];
				path = ref PathSeg.Line(cx, cy) :: path;
				i++;
				if(i >= sp) break;
				cy += stack[i];
				path = ref PathSeg.Line(cx, cy) :: path;
				i++;
			}
			sp = 0;
		7 =>
			# vlineto — alternating vertical/horizontal lines
			i := 0;
			while(i < sp){
				cy += stack[i];
				path = ref PathSeg.Line(cx, cy) :: path;
				i++;
				if(i >= sp) break;
				cx += stack[i];
				path = ref PathSeg.Line(cx, cy) :: path;
				i++;
			}
			sp = 0;
		8 =>
			# rrcurveto
			i := 0;
			while(i + 5 < sp){
				x1 := cx + stack[i];   y1 := cy + stack[i+1];
				x2 := x1 + stack[i+2]; y2 := y1 + stack[i+3];
				x3 := x2 + stack[i+4]; y3 := y2 + stack[i+5];
				path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
				cx = x3; cy = y3;
				i += 6;
			}
			sp = 0;
		27 =>
			# hhcurveto
			i := 0;
			dy1 := 0.0;
			if((sp & 1) != 0){
				dy1 = stack[0];
				i = 1;
			}
			while(i + 3 < sp){
				x1 := cx + stack[i];
				y1 := cy + dy1;
				x2 := x1 + stack[i+1]; y2 := y1 + stack[i+2];
				x3 := x2 + stack[i+3]; y3 := y2;
				path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
				cx = x3; cy = y3;
				dy1 = 0.0;
				i += 4;
			}
			sp = 0;
		26 =>
			# vvcurveto
			i := 0;
			dx1 := 0.0;
			if((sp & 1) != 0){
				dx1 = stack[0];
				i = 1;
			}
			while(i + 3 < sp){
				x1 := cx + dx1;
				y1 := cy + stack[i];
				x2 := x1 + stack[i+1]; y2 := y1 + stack[i+2];
				x3 := x2;              y3 := y2 + stack[i+3];
				path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
				cx = x3; cy = y3;
				dx1 = 0.0;
				i += 4;
			}
			sp = 0;
		31 =>
			# hvcurveto — alternating h-start/v-start curves
			i := 0;
			phase := 0;
			while(i + 3 < sp){
				if(phase == 0){
					# h-start
					x1 := cx + stack[i]; y1 := cy;
					x2 := x1 + stack[i+1]; y2 := y1 + stack[i+2];
					x3 := x2; y3 := y2 + stack[i+3];
					# last curve may have extra dx
					if(i + 4 == sp - 1){
						x3 += stack[i+4];
						i++;
					}
					path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
					cx = x3; cy = y3;
				} else {
					# v-start
					x1 := cx; y1 := cy + stack[i];
					x2 := x1 + stack[i+1]; y2 := y1 + stack[i+2];
					x3 := x2 + stack[i+3]; y3 := y2;
					# last curve may have extra dy
					if(i + 4 == sp - 1){
						y3 += stack[i+4];
						i++;
					}
					path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
					cx = x3; cy = y3;
				}
				i += 4;
				phase = 1 - phase;
			}
			sp = 0;
		30 =>
			# vhcurveto — alternating v-start/h-start curves
			i := 0;
			phase := 0;
			while(i + 3 < sp){
				if(phase == 0){
					# v-start
					x1 := cx; y1 := cy + stack[i];
					x2 := x1 + stack[i+1]; y2 := y1 + stack[i+2];
					x3 := x2 + stack[i+3]; y3 := y2;
					if(i + 4 == sp - 1){
						y3 += stack[i+4];
						i++;
					}
					path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
					cx = x3; cy = y3;
				} else {
					# h-start
					x1 := cx + stack[i]; y1 := cy;
					x2 := x1 + stack[i+1]; y2 := y1 + stack[i+2];
					x3 := x2; y3 := y2 + stack[i+3];
					if(i + 4 == sp - 1){
						x3 += stack[i+4];
						i++;
					}
					path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
					cx = x3; cy = y3;
				}
				i += 4;
				phase = 1 - phase;
			}
			sp = 0;
		24 =>
			# rcurveline — curves then a line
			i := 0;
			while(i + 7 < sp){
				x1 := cx + stack[i];   y1 := cy + stack[i+1];
				x2 := x1 + stack[i+2]; y2 := y1 + stack[i+3];
				x3 := x2 + stack[i+4]; y3 := y2 + stack[i+5];
				path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
				cx = x3; cy = y3;
				i += 6;
			}
			if(i + 1 < sp){
				cx += stack[i]; cy += stack[i+1];
				path = ref PathSeg.Line(cx, cy) :: path;
			}
			sp = 0;
		25 =>
			# rlinecurve — lines then a curve
			i := 0;
			nlines := (sp - 6) / 2;
			nl := 0;
			while(nl < nlines && i + 1 < sp){
				cx += stack[i]; cy += stack[i+1];
				path = ref PathSeg.Line(cx, cy) :: path;
				i += 2;
				nl++;
			}
			if(i + 5 < sp){
				x1 := cx + stack[i];   y1 := cy + stack[i+1];
				x2 := x1 + stack[i+2]; y2 := y1 + stack[i+3];
				x3 := x2 + stack[i+4]; y3 := y2 + stack[i+5];
				path = ref PathSeg.Curve(x1, y1, x2, y2, x3, y3) :: path;
				cx = x3; cy = y3;
			}
			sp = 0;
		14 =>
			# endchar
			if(!widthset && sp > 0){
				width = int stack[0] + nominalw;
				widthset = 1;
			}
			if(!firstmove)
				path = ref PathSeg.Close :: path;
			sp = 0;
			# End of glyph
			if(calldepth > 0){
				calldepth = 0;
			}
			break;
		10 =>
			# callsubr (local)
			if(sp > 0 && lsubrs != nil){
				sp--;
				subridx := int stack[sp];
				subridx += subrbiasn(lsubrs.count);
				if(subridx >= 0 && subridx < lsubrs.count){
					if(calldepth < MAXCALLSTACK){
						callstack[calldepth] = (data, pos);
						calldepth++;
						data = lsubrs.data[subridx];
						pos = 0;
					}
				}
			} else
				sp = 0;
		29 =>
			# callgsubr (global)
			if(sp > 0 && gsubrs != nil){
				sp--;
				subridx := int stack[sp];
				subridx += subrbiasn(gsubrs.count);
				if(subridx >= 0 && subridx < gsubrs.count){
					if(calldepth < MAXCALLSTACK){
						callstack[calldepth] = (data, pos);
						calldepth++;
						data = gsubrs.data[subridx];
						pos = 0;
					}
				}
			} else
				sp = 0;
		11 =>
			# return
			if(calldepth > 0){
				calldepth--;
				(data, pos) = callstack[calldepth];
			}
		* =>
			# Unknown operator — clear stack
			sp = 0;
		}
	}

	return ref GlyphOutline(path, width);
}

subrbiasn(n: int): int
{
	if(n < 1240) return 107;
	if(n < 33900) return 1131;
	return 32768;
}

fabs(x: real): real
{
	if(x < 0.0) return -x;
	return x;
}

# ---- Rasterizer ----

# Convert outline to image via fillpoly.
# Uses a single fillpoly call with all contours — the non-zero winding
# rule handles holes correctly (outer CW, inner CCW per CFF convention).
SS: con 4;	# supersample factor for antialiasing

rasterize(path: list of ref PathSeg, scale: real): (ref Image, int, int)
{
	sscale := scale * real SS;

	# Reverse path (built in reverse order by charstring interpreter)
	rpath := revsegs(path);

	# Flatten to subpaths (each Move starts a new subpath)
	subpaths: list of array of Point;
	curpts: list of Point;
	curlen := 0;
	fcx := 0.0;
	fcy := 0.0;
	startx := 0.0;
	starty := 0.0;

	# Reset bounding box
	gminx = 16r7FFFFFFF;
	gminy = 16r7FFFFFFF;
	gmaxx = -16r7FFFFFFF;
	gmaxy = -16r7FFFFFFF;

	for(p := rpath; p != nil; p = tl p){
		seg := hd p;
		pick s := seg {
		Move =>
			# Close previous subpath if any
			if(curlen >= 3)
				subpaths = listtoarray(curpts, curlen) :: subpaths;
			curpts = nil;
			curlen = 0;
			fcx = s.x * sscale;
			fcy = -s.y * sscale;
			startx = fcx; starty = fcy;
			pt := Point(int (fcx + 0.5), int (fcy + 0.5));
			curpts = pt :: curpts;
			curlen++;
			updatebb(pt);
		Line =>
			fcx = s.x * sscale;
			fcy = -s.y * sscale;
			pt := Point(int (fcx + 0.5), int (fcy + 0.5));
			curpts = pt :: curpts;
			curlen++;
			updatebb(pt);
		Curve =>
			tx1 := s.x1 * sscale;
			ty1 := -s.y1 * sscale;
			tx2 := s.x2 * sscale;
			ty2 := -s.y2 * sscale;
			tx3 := s.x3 * sscale;
			ty3 := -s.y3 * sscale;
			bpts := subdivbezier(fcx, fcy, tx1, ty1, tx2, ty2, tx3, ty3, 0);
			for(; bpts != nil; bpts = tl bpts){
				cpt := hd bpts;
				curpts = cpt :: curpts;
				curlen++;
				updatebb(cpt);
			}
			fcx = tx3; fcy = ty3;
		Close =>
			pt := Point(int (startx + 0.5), int (starty + 0.5));
			curpts = pt :: curpts;
			curlen++;
			if(curlen >= 3)
				subpaths = listtoarray(curpts, curlen) :: subpaths;
			curpts = nil;
			curlen = 0;
		}
	}
	# Close final subpath if not explicitly closed
	if(curlen >= 3)
		subpaths = listtoarray(curpts, curlen) :: subpaths;

	if(subpaths == nil)
		return (nil, 0, 0);

	# Add padding at SS resolution
	ix0 := gminx - SS;
	iy0 := gminy - SS;
	sw := gmaxx - ix0 + SS + 1;
	sh := gmaxy - iy0 + SS + 1;
	sw = ((sw + SS - 1) / SS) * SS;
	sh = ((sh + SS - 1) / SS) * SS;
	if(sw <= 0 || sh <= 0 || sw > 4096 || sh > 4096)
		return (nil, 0, 0);

	# Offset all subpath points to image coordinates
	for(sp := subpaths; sp != nil; sp = tl sp){
		pts := hd sp;
		for(i := 0; i < len pts; i++){
			pts[i].x -= ix0;
			pts[i].y -= iy0;
		}
	}

	# Scanline fill at SS resolution into byte array
	buf := array[sw * sh] of { * => byte 0 };
	scanlinefillbuf(buf, subpaths, sw, sh);

	# Downsample SSxSS blocks → 1 pixel with 8-bit alpha
	dw := sw / SS;
	dh := sh / SS;
	pixels := array[dw * dh] of byte;
	for(dy := 0; dy < dh; dy++){
		for(dx := 0; dx < dw; dx++){
			sum := 0;
			sy := dy * SS;
			sx := dx * SS;
			for(yy := 0; yy < SS; yy++)
				for(xx := 0; xx < SS; xx++)
					sum += int buf[(sy + yy) * sw + sx + xx];
			pixels[dy * dw + dx] = byte ((sum * 255 + SS*SS/2) / (SS*SS));
		}
	}

	mask := display.newimage(Rect((0, 0), (dw, dh)), Draw->GREY8, 0, Draw->Transparent);
	if(mask == nil)
		return (nil, 0, 0);
	mask.writepixels(mask.r, pixels);

	return (mask, ix0 / SS, iy0 / SS);
}

listtoarray(pts: list of Point, n: int): array of Point
{
	a := array[n] of Point;
	i := n - 1;
	for(; pts != nil; pts = tl pts)
		a[i--] = hd pts;
	return a;
}

# Bounding box globals (updated during rasterize)
gminx, gminy, gmaxx, gmaxy: int;

updatebb(p: Point)
{
	if(p.x < gminx) gminx = p.x;
	if(p.x > gmaxx) gmaxx = p.x;
	if(p.y < gminy) gminy = p.y;
	if(p.y > gmaxy) gmaxy = p.y;
}

# Scanline fill into byte array using non-zero winding rule.
# Handles multiple subpaths correctly — each subpath's edges wrap independently.
scanlinefillbuf(buf: array of byte, subpaths: list of array of Point, w, h: int)
{
	for(y := 0; y < h; y++){
		yf := real y + 0.5;
		xlist: list of (real, int);

		# Collect edge crossings from ALL subpaths
		for(sp := subpaths; sp != nil; sp = tl sp){
			pts := hd sp;
			npts := len pts;
			for(i := 0; i < npts; i++){
				j := (i + 1) % npts;	# wraps within this subpath only
				y0 := real pts[i].y;
				y1 := real pts[j].y;
				if(y0 == y1)
					continue;
				if((yf < y0 && yf < y1) || (yf >= y0 && yf >= y1))
					continue;
				t := (yf - y0) / (y1 - y0);
				xc := real pts[i].x + t * real (pts[j].x - pts[i].x);
				dir := 1;
				if(y1 < y0)
					dir = -1;
				xlist = (xc, dir) :: xlist;
			}
		}

		ncross := 0;
		for(xl := xlist; xl != nil; xl = tl xl)
			ncross++;
		if(ncross < 2)
			continue;
		xarr := array[ncross] of (real, int);
		k := 0;
		for(xl = xlist; xl != nil; xl = tl xl)
			xarr[k++] = hd xl;
		for(a := 1; a < ncross; a++){
			tmp := xarr[a];
			b := a - 1;
			while(b >= 0 && xarr[b].t0 > tmp.t0){
				xarr[b+1] = xarr[b];
				b--;
			}
			xarr[b+1] = tmp;
		}

		winding := 0;
		row := y * w;
		for(c := 0; c < ncross - 1; c++){
			winding += xarr[c].t1;
			if(winding != 0){
				xl := int xarr[c].t0;
				xr := int (xarr[c+1].t0 + 0.5);
				if(xl < 0) xl = 0;
				if(xr > w) xr = w;
				for(x := xl; x < xr; x++)
					buf[row + x] = byte 1;
			}
		}
	}
}

revsegs(path: list of ref PathSeg): list of ref PathSeg
{
	rev: list of ref PathSeg;
	for(; path != nil; path = tl path)
		rev = hd path :: rev;
	return rev;
}

# Flatten cubic Bezier to polyline via de Casteljau subdivision
FLAT_THRESH: con 0.5;

subdivbezier(x0, y0, x1, y1, x2, y2, x3, y3: real, depth: int): list of Point
{
	# Check flatness
	dx := x3 - x0;
	dy := y3 - y0;
	d1 := fabs((x1 - x3) * dy - (y1 - y3) * dx);
	d2 := fabs((x2 - x3) * dy - (y2 - y3) * dx);

	if((d1 + d2) * (d1 + d2) <= FLAT_THRESH * (dx*dx + dy*dy) || depth > 10)
		return Point(int (x3 + 0.5), int (y3 + 0.5)) :: nil;

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

	left := subdivbezier(x0, y0, mx01, my01, mx012, my012, mx0123, my0123, depth+1);
	right := subdivbezier(mx0123, my0123, mx123, my123, mx23, my23, x3, y3, depth+1);

	# Concatenate: append right to left
	result := right;
	for(l := revpts(left); l != nil; l = tl l)
		result = hd l :: result;
	return result;
}

revpts(pts: list of Point): list of Point
{
	rev: list of Point;
	for(; pts != nil; pts = tl pts)
		rev = hd pts :: rev;
	return rev;
}

# ---- CFF parser ----

parsecff(data: array of byte): (ref FaceData, string)
{
	if(len data < 4)
		return (nil, "too short for CFF");

	# CFF header
	major := int data[0];
	# minor := int data[1];
	hdrsize := int data[2];
	if(major != 1)
		return (nil, sys->sprint("unsupported CFF version %d", major));

	pos := hdrsize;

	# Name INDEX
	(nameidx, np1, nerr) := parseindex(data, pos);
	if(nerr != nil)
		return (nil, "Name INDEX: " + nerr);
	pos = np1;

	fontname := "";
	if(nameidx.count > 0 && nameidx.data[0] != nil)
		fontname = string nameidx.data[0];

	# Top DICT INDEX
	(tdidx, np2, terr) := parseindex(data, pos);
	if(terr != nil)
		return (nil, "Top DICT INDEX: " + terr);
	pos = np2;

	if(tdidx.count < 1)
		return (nil, "no Top DICT");

	# Parse Top DICT
	td := parsetopdict(tdidx.data[0]);
	td.fontname = fontname;

	# String INDEX (skip — we don't need string lookups for rendering)
	(nil, np3, serr) := parseindex(data, pos);
	if(serr != nil)
		return (nil, "String INDEX: " + serr);
	pos = np3;

	# Global Subr INDEX
	(gsubrs, np4, gerr) := parseindex(data, pos);
	if(gerr != nil)
		return (nil, "Global Subr INDEX: " + gerr);

	# suppress unused warning
	if(np4 < 0) np4 = np4;

	# Parse CharStrings INDEX
	if(td.charstrings_off <= 0 || td.charstrings_off >= len data)
		return (nil, "bad CharStrings offset");
	(csidx, nil, cerr) := parseindex(data, td.charstrings_off);
	if(cerr != nil)
		return (nil, "CharStrings INDEX: " + cerr);

	nglyphs := csidx.count;

	# Parse charset (GID -> SID/CID mapping)
	cidmap: array of int;
	if(td.ros && td.charset_off > 0 && td.charset_off < len data)
		cidmap = parsecharset(data, td.charset_off, nglyphs);

	# Parse Private DICT
	privd: ref CffPrivateDict;
	lsubrs: ref CffIndex;
	if(td.private_size > 0 && td.private_off > 0 && td.private_off + td.private_size <= len data){
		pdata := data[td.private_off:td.private_off + td.private_size];
		privd = parseprivatedict(pdata);
		# Local subrs
		if(privd.subrs_off > 0){
			lsoff := td.private_off + privd.subrs_off;
			if(lsoff < len data){
				(ls, nil, lerr) := parseindex(data, lsoff);
				if(lerr == nil)
					lsubrs = ls;
			}
		}
	}

	# CID font handling
	iscid := td.ros;
	fdcount := 0;
	fdprivate: array of ref CffPrivateDict;
	fdlsubrs: array of ref CffIndex;
	fdsel: array of byte;

	if(iscid){
		# FDArray
		if(td.fdarray_off > 0 && td.fdarray_off < len data){
			(fdaidx, nil, faerr) := parseindex(data, td.fdarray_off);
			if(faerr == nil && fdaidx.count > 0){
				fdcount = fdaidx.count;
				fdprivate = array[fdcount] of ref CffPrivateDict;
				fdlsubrs = array[fdcount] of ref CffIndex;
				for(i := 0; i < fdcount; i++){
					fdict := parsetopdict(fdaidx.data[i]);
					if(fdict.private_size > 0 && fdict.private_off > 0 &&
					   fdict.private_off + fdict.private_size <= len data){
						fpdata := data[fdict.private_off:fdict.private_off + fdict.private_size];
						fdprivate[i] = parseprivatedict(fpdata);
						if(fdprivate[i].subrs_off > 0){
							flsoff := fdict.private_off + fdprivate[i].subrs_off;
							if(flsoff < len data){
								(fls, nil, flerr) := parseindex(data, flsoff);
								if(flerr == nil)
									fdlsubrs[i] = fls;
							}
						}
					}
				}
			}
		}

		# FDSelect
		if(td.fdselect_off > 0 && td.fdselect_off < len data){
			fdsel = parsefdselect(data, td.fdselect_off, nglyphs);
		}
	}

	# Default metrics
	upem := 1000;
	ascent := td.ascent;
	descent := td.descent;
	if(ascent == 0) ascent = 800;
	if(descent == 0) descent = -200;

	fd := ref FaceData(
		data,
		nglyphs,
		upem,
		ascent,
		descent,
		fontname,
		csidx,
		gsubrs,
		privd,
		lsubrs,
		iscid,
		fdcount,
		fdprivate,
		fdlsubrs,
		fdsel,
		cidmap
	);

	return (fd, nil);
}

# Parse a CFF INDEX structure
parseindex(data: array of byte, offset: int): (ref CffIndex, int, string)
{
	pos := offset;
	if(pos + 2 > len data)
		return (nil, 0, "truncated INDEX count");

	count := (int data[pos] << 8) | int data[pos+1];
	pos += 2;

	if(count == 0)
		return (ref CffIndex(0, nil), pos, nil);

	if(pos >= len data)
		return (nil, 0, "truncated INDEX offSize");
	offsize := int data[pos];
	pos++;

	if(offsize < 1 || offsize > 4)
		return (nil, 0, sys->sprint("bad INDEX offSize %d", offsize));

	# Read offset array (count+1 entries)
	offsets := array[count + 1] of int;
	for(i := 0; i <= count; i++){
		if(pos + offsize > len data)
			return (nil, 0, "truncated INDEX offsets");
		v := 0;
		for(j := 0; j < offsize; j++)
			v = (v << 8) | int data[pos + j];
		offsets[i] = v;
		pos += offsize;
	}

	# Data starts at current pos, offsets are 1-based
	datastart := pos - 1;	# offsets are 1-based in CFF
	endpos := datastart + offsets[count];

	items := array[count] of array of byte;
	for(i = 0; i < count; i++){
		start := datastart + offsets[i];
		end := datastart + offsets[i + 1];
		if(start < 0 || end > len data || start > end){
			items[i] = nil;
			continue;
		}
		item := array[end - start] of byte;
		item[0:] = data[start:end];
		items[i] = item;
	}

	return (ref CffIndex(count, items), endpos, nil);
}

# Parse CFF Top DICT
parsetopdict(data: array of byte): ref CffTopDict
{
	td := ref CffTopDict(0, 0, 0, 0, 0, 0, 0, 0, "", 0, 0);
	if(data == nil || len data == 0)
		return td;

	operands: list of int;
	pos := 0;

	while(pos < len data){
		b0 := int data[pos];

		# Number: bytes 28-30 and 32-254 are number encodings in CFF DICT
		if(b0 >= 28 && b0 != 31){
			(val, np) := dictreadnum(data, pos);
			operands = val :: operands;
			pos = np;
			continue;
		}

		# Operator: bytes 0-27 and 31
		pos++;
		if(b0 == 12){
			if(pos >= len data) break;
			b1 := int data[pos]; pos++;
			op := 3000 + b1;
			case op {
			3030 =>	# ROS (12 30) — CIDFont
				td.ros = 1;
			3036 =>	# FDArray (12 36)
				td.fdarray_off = popint(operands);
			3037 =>	# FDSelect (12 37)
				td.fdselect_off = popint(operands);
			}
			operands = nil;
			continue;
		}

		case b0 {
		15 =>	# charset
			td.charset_off = popint(operands);
		16 =>	# Encoding
			td.encoding_off = popint(operands);
		17 =>	# CharStrings
			td.charstrings_off = popint(operands);
		18 =>	# Private (size, offset)
			if(operands != nil){
				td.private_off = hd operands;
				operands = tl operands;
			}
			if(operands != nil){
				td.private_size = hd operands;
				operands = tl operands;
			}
		}
		operands = nil;
	}

	return td;
}

# Parse CFF Private DICT
parseprivatedict(data: array of byte): ref CffPrivateDict
{
	pd := ref CffPrivateDict(0, 0, 0);
	if(data == nil || len data == 0)
		return pd;

	operands: list of int;
	pos := 0;

	while(pos < len data){
		b0 := int data[pos];

		# Number: bytes 28-30 and 32-254 are number encodings in CFF DICT
		if(b0 >= 28 && b0 != 31){
			(val, np) := dictreadnum(data, pos);
			operands = val :: operands;
			pos = np;
			continue;
		}

		pos++;
		if(b0 == 12){
			if(pos >= len data) break;
			pos++;	# skip 2nd byte
			operands = nil;
			continue;
		}

		case b0 {
		19 =>	# Subrs
			pd.subrs_off = popint(operands);
		20 =>	# defaultWidthX
			pd.defaultw = popint(operands);
		21 =>	# nominalWidthX
			pd.nominalw = popint(operands);
		}
		operands = nil;
	}

	return pd;
}

# Read a DICT number (integer or real encoded as integer)
dictreadnum(data: array of byte, pos: int): (int, int)
{
	if(pos >= len data)
		return (0, pos);

	b0 := int data[pos];
	pos++;

	if(b0 == 28){
		if(pos + 1 >= len data)
			return (0, pos);
		v := (int data[pos] << 8) | int data[pos+1];
		if(v & 16r8000) v -= 16r10000;
		return (v, pos + 2);
	}
	if(b0 == 29){
		if(pos + 3 >= len data)
			return (0, pos);
		v := (int data[pos] << 24) | (int data[pos+1] << 16) |
		     (int data[pos+2] << 8) | int data[pos+3];
		return (v, pos + 4);
	}
	if(b0 == 30){
		# Real number — skip nibbles until end sentinel
		while(pos < len data){
			b := int data[pos];
			pos++;
			n1 := (b >> 4) & 16rF;
			n2 := b & 16rF;
			if(n1 == 16rF || n2 == 16rF)
				break;
		}
		return (0, pos);	# return 0 for reals (we only need ints)
	}
	if(b0 >= 32 && b0 <= 246)
		return (b0 - 139, pos);
	if(b0 >= 247 && b0 <= 250){
		if(pos >= len data)
			return (0, pos);
		b1 := int data[pos]; pos++;
		return ((b0 - 247) * 256 + b1 + 108, pos);
	}
	if(b0 >= 251 && b0 <= 254){
		if(pos >= len data)
			return (0, pos);
		b1 := int data[pos]; pos++;
		return (-(b0 - 251) * 256 - b1 - 108, pos);
	}
	return (0, pos);
}

popint(operands: list of int): int
{
	if(operands == nil)
		return 0;
	return hd operands;
}

# Parse FDSelect (format 0 and 3)
parsefdselect(data: array of byte, offset, nglyphs: int): array of byte
{
	fdsel := array[nglyphs] of { * => byte 0 };
	if(offset >= len data)
		return fdsel;

	fmt := int data[offset];
	pos := offset + 1;

	case fmt {
	0 =>
		# Format 0: one byte per glyph
		for(i := 0; i < nglyphs && pos < len data; i++){
			fdsel[i] = data[pos];
			pos++;
		}
	3 =>
		# Format 3: ranges
		if(pos + 1 >= len data)
			return fdsel;
		nranges := (int data[pos] << 8) | int data[pos+1];
		pos += 2;
		for(i := 0; i < nranges; i++){
			if(pos + 2 >= len data) break;
			first := (int data[pos] << 8) | int data[pos+1];
			fd := int data[pos + 2];
			pos += 3;
			# Next range start (or sentinel)
			nextfirst := nglyphs;
			if(i + 1 < nranges && pos + 1 < len data)
				nextfirst = (int data[pos] << 8) | int data[pos+1];
			for(g := first; g < nextfirst && g < nglyphs; g++)
				fdsel[g] = byte fd;
		}
	}

	return fdsel;
}

# Parse CFF charset table and build a CID->GID reverse map.
# For CID-keyed fonts, the charset maps GID -> CID.
# We invert it to CID -> GID for efficient lookup during rendering.
parsecharset(data: array of byte, offset, nglyphs: int): array of int
{
	if(offset >= len data)
		return nil;

	# Build GID -> CID array first
	gidtocid := array[nglyphs] of { * => 0 };
	# GID 0 is always .notdef (CID 0)
	maxcid := 0;

	fmt := int data[offset];
	pos := offset + 1;

	case fmt {
	0 =>
		# Format 0: one 2-byte SID/CID per glyph (starting at GID 1)
		for(gid := 1; gid < nglyphs; gid++){
			if(pos + 1 >= len data) break;
			cid := (int data[pos] << 8) | int data[pos+1];
			pos += 2;
			gidtocid[gid] = cid;
			if(cid > maxcid) maxcid = cid;
		}
	1 =>
		# Format 1: ranges with 1-byte count
		gid := 1;
		while(gid < nglyphs && pos + 2 < len data){
			first := (int data[pos] << 8) | int data[pos+1];
			nleft := int data[pos+2];
			pos += 3;
			for(j := 0; j <= nleft && gid < nglyphs; j++){
				cid := first + j;
				gidtocid[gid] = cid;
				if(cid > maxcid) maxcid = cid;
				gid++;
			}
		}
	2 =>
		# Format 2: ranges with 2-byte count
		gid := 1;
		while(gid < nglyphs && pos + 3 < len data){
			first := (int data[pos] << 8) | int data[pos+1];
			nleft := (int data[pos+2] << 8) | int data[pos+3];
			pos += 4;
			for(j := 0; j <= nleft && gid < nglyphs; j++){
				cid := first + j;
				gidtocid[gid] = cid;
				if(cid > maxcid) maxcid = cid;
				gid++;
			}
		}
	* =>
		return nil;	# unknown format
	}

	# Build reverse map: CID -> GID
	cidmap := array[maxcid + 1] of { * => -1 };
	cidmap[0] = 0;	# .notdef
	for(gid := 1; gid < nglyphs; gid++){
		cid := gidtocid[gid];
		if(cid >= 0 && cid <= maxcid)
			cidmap[cid] = gid;
	}

	return cidmap;
}
