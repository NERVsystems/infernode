implement Mermaid;

#
# mermaid.b — Native Mermaid diagram renderer for Inferno/Limbo
#
# Renders Mermaid syntax to Draw->Image using only Inferno drawing
# primitives.  No floating point in layout.  No external dependencies.
#
# Supported types: flowchart/graph, pie, sequenceDiagram, gantt, xychart-beta
#

include "sys.m";
	sys: Sys;

include "draw.m";
	draw: Draw;
	Display, Font, Image, Point, Rect: import draw;

include "mermaid.m";

# ═══════════════════════════════════════════════════════════════════════════════
# Layout constants
# ═══════════════════════════════════════════════════════════════════════════════

HPAD:		con 14;		# node: horizontal text padding
VPAD:		con 7;		# node: vertical text padding
MINNODEW:	con 64;		# minimum node width
MINNODEH:	con 28;		# minimum node height
VGAP:		con 40;		# gap between layers (TD) / columns (LR)
HGAP:		con 18;		# gap between nodes in the same layer
MARGIN:		con 22;		# outer margin around diagram
AHEADLEN:	con 10;		# arrowhead length (pixels)
AHEADW:		con 6;		# arrowhead half-width

# Sequence diagram
SEQ_COLW:	con 140;	# centre-to-centre column spacing
SEQ_BOXW:	con 110;	# participant box width
SEQ_BOXH:	con 26;		# participant box height
SEQ_ROWH:	con 38;		# row height per message

# Gantt
GNT_ROWH:	con 26;		# task row height
GNT_LBLW:	con 150;	# left label column width
GNT_HDRY:	con 32;		# date header height
GNT_SECTH:	con 20;		# section title height

# XY chart
XY_AXISW:	con 50;		# left axis width
XY_AXISH:	con 28;		# bottom axis height
XY_PLOTH:	con 180;	# plot area height

# Default width when caller passes 0
DEFWIDTH:	con 800;

# ═══════════════════════════════════════════════════════════════════════════════
# Diagram type IDs
# ═══════════════════════════════════════════════════════════════════════════════

DT_FLOW:	con 0;
DT_PIE:		con 1;
DT_SEQ:		con 2;
DT_GANTT:	con 3;
DT_XY:		con 4;
DT_UNKNOWN:	con 99;

# Flowchart direction
DIRN_TD:	con 0;
DIRN_LR:	con 1;
DIRN_BT:	con 2;
DIRN_RL:	con 3;

# Node shapes
SH_RECT:	con 0;		# [label]
SH_ROUND:	con 1;		# (label)
SH_DIAMOND:	con 2;		# {label}
SH_CIRCLE:	con 3;		# ((label))
SH_STADIUM:	con 4;		# ([label])
SH_HEX:		con 5;		# {{label}}
SH_SUBR:	con 6;		# [[label]]
SH_FLAG:	con 7;		# >label]

# Edge styles
ES_SOLID:	con 0;		# -->
ES_DASH:	con 1;		# -.->
ES_THICK:	con 2;		# ==>
ES_LINE:	con 3;		# --- (no arrowhead)

# Sequence message types
SM_SOLID:	con 0;		# ->>
SM_DASH:	con 1;		# -->>

# ═══════════════════════════════════════════════════════════════════════════════
# Data structures
# ═══════════════════════════════════════════════════════════════════════════════

FCNode: adt {
	id:	string;
	label:	string;
	shape:	int;
	# layout (filled by layout pass)
	layer:	int;
	col:	int;
	x:	int;		# centre pixel x
	y:	int;		# centre pixel y
	w:	int;		# pixel width
	h:	int;		# pixel height
};

FCEdge: adt {
	src:	string;
	dst:	string;
	label:	string;
	style:	int;
	arrow:	int;		# 1 = has arrowhead at dst
};

FCGraph: adt {
	dir:	int;
	title:	string;
	nodes:	list of ref FCNode;
	nnodes:	int;
	edges:	list of ref FCEdge;
	nedges:	int;
};

PieSlice: adt {
	label:	string;
	value:	int;		# ×1024 fixed-point
};

PieChart: adt {
	title:	string;
	showdata: int;
	slices:	list of ref PieSlice;
	nslices: int;
};

SeqPart: adt {
	id:	string;
	alias:	string;
	idx:	int;
};

SeqMsg: adt {
	from:	string;
	dst:	string;
	text:	string;
	mtype:	int;		# SM_SOLID or SM_DASH
	isnote:	int;		# 1 = Note annotation
	notetext: string;
};

SeqDiag: adt {
	parts:	list of ref SeqPart;
	nparts:	int;
	msgs:	list of ref SeqMsg;
	nmsgs:	int;
};

GTask: adt {
	section: string;
	label:	string;
	id:	string;
	crit:	int;
	active:	int;
	done:	int;
	after:	string;
	startday: int;		# days since 2000-01-01
	durdays:  int;
};

GanttChart: adt {
	title:	string;
	tasks:	list of ref GTask;
	ntasks:	int;
	minday:	int;
	maxday:	int;
};

XYSeries: adt {
	isbar:	int;		# 1=bar, 0=line
	vals:	array of int;	# ×1024 fixed-point
	nvals:	int;
};

XYChart: adt {
	title:	string;
	xlabels: array of string;
	nxlbl:	int;
	ylower:	int;		# ×1024
	yupper:	int;		# ×1024
	series:	list of ref XYSeries;
};

# ═══════════════════════════════════════════════════════════════════════════════
# Module state
# ═══════════════════════════════════════════════════════════════════════════════

mdisp:	ref Display;
mfont:	ref Font;
mofont:	ref Font;

# Color images (allocated in init)
cbg:	ref Image;	# background
cnode:	ref Image;	# node fill
cbord:	ref Image;	# node border / edge
ctext:	ref Image;	# primary text
ctext2:	ref Image;	# secondary text
cacc:	ref Image;	# accent (arrows, active)
cgreen:	ref Image;	# done / ok
cred:	ref Image;	# critical / error
cyel:	ref Image;	# warning
cgrid:	ref Image;	# axis grid lines
csect:	ref Image;	# section title bar
cwhite:	ref Image;	# white

# Pie / XY series palette (8 entries)
cpie:	array of ref Image;


# ═══════════════════════════════════════════════════════════════════════════════
# init
# ═══════════════════════════════════════════════════════════════════════════════

init(d: ref Display, mainfont: ref Font, monofont: ref Font)
{
	sys = load Sys Sys->PATH;
	draw = load Draw Draw->PATH;
	mdisp = d;
	mfont = mainfont;
	mofont = monofont;
	if(mfont == nil)
		mfont = Font.open(d, "*default*");
	if(mofont == nil)
		mofont = mfont;

	cbg    = d.color(int 16r1E1E2EFF);
	cnode  = d.color(int 16r313244FF);
	cbord  = d.color(int 16r89B4FAFF);
	ctext  = d.color(int 16rCDD6F4FF);
	ctext2 = d.color(int 16r8B949EFF);
	cacc   = d.color(int 16r89B4FAFF);
	cgreen = d.color(int 16rA6E3A1FF);
	cred   = d.color(int 16rF38BA8FF);
	cyel   = d.color(int 16rF9E2AFFF);
	cgrid  = d.color(int 16r45475AFF);
	csect  = d.color(int 16r45475AFF);
	cwhite = d.color(int 16rCDD6F4FF);

	cpie = array[8] of ref Image;
	cpie[0] = d.color(int 16r89B4FAFF);	# blue
	cpie[1] = d.color(int 16rA6E3A1FF);	# green
	cpie[2] = d.color(int 16rF9E2AFFF);	# yellow
	cpie[3] = d.color(int 16rF38BA8FF);	# red
	cpie[4] = d.color(int 16rCBA6F7FF);	# mauve
	cpie[5] = d.color(int 16r94E2D5FF);	# teal
	cpie[6] = d.color(int 16rFAB387FF);	# peach
	cpie[7] = d.color(int 16r89DCEBFF);	# sky
}

# ═══════════════════════════════════════════════════════════════════════════════
# render — main dispatcher
# ═══════════════════════════════════════════════════════════════════════════════

render(syntax: string, width: int): (ref Draw->Image, string)
{
	if(mdisp == nil)
		return (nil, "mermaid: not initialized — call init() first");
	if(width <= 0)
		width = DEFWIDTH;

	lines := splitlines(syntax);
	dtype := detecttype(lines);

	img: ref Image;
	err: string;
	{
		case dtype {
		DT_FLOW =>
			(img, err) = renderflow(lines, width);
		DT_PIE =>
			(img, err) = renderpie(lines, width);
		DT_SEQ =>
			(img, err) = renderseq(lines, width);
		DT_GANTT =>
			(img, err) = rendergantt(lines, width);
		DT_XY =>
			(img, err) = renderxy(lines, width);
		* =>
			return rendererror("Unsupported diagram type", width);
		}
	} exception e {
	"*" =>
		return (nil, "mermaid: " + e);
	}

	if(err != nil)
		return rendererror(err, width);
	return (img, nil);
}

# ═══════════════════════════════════════════════════════════════════════════════
# Diagram type detection
# ═══════════════════════════════════════════════════════════════════════════════

detecttype(lines: list of string): int
{
	for(l := lines; l != nil; l = tl l) {
		s := trimstr(hd l);
		if(s == "" || hasprefix(s, "%%"))
			continue;
		sl := tolower(s);
		if(hasprefix(sl, "graph ") || hasprefix(sl, "graph\t") ||
				hasprefix(sl, "flowchart ") || hasprefix(sl, "flowchart\t"))
			return DT_FLOW;
		if(hasprefix(sl, "pie"))
			return DT_PIE;
		if(hasprefix(sl, "sequencediagram"))
			return DT_SEQ;
		if(hasprefix(sl, "gantt"))
			return DT_GANTT;
		if(hasprefix(sl, "xychart"))
			return DT_XY;
		break;
	}
	return DT_UNKNOWN;
}

# ═══════════════════════════════════════════════════════════════════════════════
# ─── FLOWCHART ────────────────────────────────────────────────────────────────
# ═══════════════════════════════════════════════════════════════════════════════

parseflow(lines: list of string): ref FCGraph
{
	g := ref FCGraph(DIRN_TD, "", nil, 0, nil, 0);
	# Parse direction from first line
	for(l := lines; l != nil; l = tl l) {
		s := trimstr(hd l);
		if(s == "" || hasprefix(s, "%%")) continue;
		sl := tolower(s);
		i := 0;
		if(hasprefix(sl, "flowchart ") || hasprefix(sl, "flowchart\t"))
			i = 10;
		else if(hasprefix(sl, "graph ") || hasprefix(sl, "graph\t"))
			i = 6;
		if(i > 0) {
			while(i < len sl && (sl[i] == ' ' || sl[i] == '\t'))
				i++;
			dir := sl[i:];
			if(hasprefix(dir, "lr"))  g.dir = DIRN_LR;
			else if(hasprefix(dir, "bt")) g.dir = DIRN_BT;
			else if(hasprefix(dir, "rl")) g.dir = DIRN_RL;
			else g.dir = DIRN_TD;
		}
		break;
	}

	for(l = lines; l != nil; l = tl l) {
		s := trimstr(hd l);
		if(s == "" || hasprefix(s, "%%")) continue;
		sl := tolower(s);
		# Skip keywords
		if(hasprefix(sl, "graph ") || hasprefix(sl, "graph\t") ||
				hasprefix(sl, "flowchart ") || hasprefix(sl, "flowchart\t") ||
				hasprefix(sl, "subgraph") || s == "end" ||
				hasprefix(sl, "classdef") || hasprefix(sl, "class ") ||
				hasprefix(sl, "style ") || hasprefix(sl, "linkstyle") ||
				hasprefix(sl, "click "))
			continue;
		parseflowline(s, g);
	}
	return g;
}

parseflowline(line: string, g: ref FCGraph)
{
	id1, lbl1, id2, lbl2, elabel: string;
	sh1, sh2, estyle, ehasarrow, ni: int;
	i := 0;
	n := len line;
	while(i < n && (line[i] == ' ' || line[i] == '\t'))
		i++;
	if(i >= n)
		return;

	# Parse first node
	(id1, lbl1, sh1, ni) = parsefcnode(line, i);
	i = ni;
	if(id1 == "")
		return;
	addnode(g, id1, lbl1, sh1);

	# Parse edge(s) and subsequent nodes
	for(;;) {
		while(i < n && (line[i] == ' ' || line[i] == '\t'))
			i++;
		if(i >= n)
			break;

		(estyle, ehasarrow, elabel, ni) = parseedgeop(line, i);
		i = ni;
		if(estyle < 0)
			break;

		while(i < n && (line[i] == ' ' || line[i] == '\t'))
			i++;
		if(i >= n)
			break;

		(id2, lbl2, sh2, ni) = parsefcnode(line, i);
		i = ni;
		if(id2 == "")
			break;
		addnode(g, id2, lbl2, sh2);
		addedge(g, id1, id2, elabel, estyle, ehasarrow);
		id1 = id2;
	}
}

# Parse a node expression at position i; return (id, label, shape, new_i)
parsefcnode(s: string, i: int): (string, string, int, int)
{
	n := len s;
	while(i < n && (s[i] == ' ' || s[i] == '\t'))
		i++;
	if(i >= n)
		return ("", "", SH_RECT, i);

	# Read node ID
	idstart := i;
	while(i < n && s[i] != '[' && s[i] != '(' && s[i] != '{' &&
			s[i] != '>' && s[i] != ' ' && s[i] != '\t' &&
			s[i] != '-' && s[i] != '=' && s[i] != '|' &&
			s[i] != '&' && s[i] != ';')
		i++;
	id := s[idstart:i];
	if(id == "")
		return ("", "", SH_RECT, i);

	while(i < n && (s[i] == ' ' || s[i] == '\t'))
		i++;

	if(i >= n || s[i] == '-' || s[i] == '=' || s[i] == '&' || s[i] == ';')
		return (id, id, SH_RECT, i);

	shape := SH_RECT;
	label := id;

	case s[i] {
	'[' =>
		# Check [[subroutine]] or [regular]
		if(i+1 < n && s[i+1] == '[') {
			i += 2;
			(label, i) = readuntil(s, i, ']');
			if(i+1 < n && s[i+1] == ']') i += 2;
			else if(i < n && s[i] == ']') i++;
			shape = SH_SUBR;
		} else {
			i++;
			(label, i) = readuntil(s, i, ']');
			if(i < n && s[i] == ']') i++;
			shape = SH_RECT;
		}
	'(' =>
		# ((circle)) or ([stadium]) or (round)
		if(i+1 < n && s[i+1] == '(') {
			i += 2;
			(label, i) = readuntil(s, i, ')');
			if(i+1 < n && s[i+1] == ')') i += 2;
			else if(i < n && s[i] == ')') i++;
			shape = SH_CIRCLE;
		} else if(i+1 < n && s[i+1] == '[') {
			i += 2;
			(label, i) = readuntil(s, i, ']');
			# expect ])
			if(i < n && s[i] == ']') i++;
			if(i < n && s[i] == ')') i++;
			shape = SH_STADIUM;
		} else {
			i++;
			(label, i) = readuntil(s, i, ')');
			if(i < n && s[i] == ')') i++;
			shape = SH_ROUND;
		}
	'{' =>
		# {{hex}} or {diamond}
		if(i+1 < n && s[i+1] == '{') {
			i += 2;
			(label, i) = readuntil(s, i, '}');
			if(i+1 < n && s[i+1] == '}') i += 2;
			else if(i < n && s[i] == '}') i++;
			shape = SH_HEX;
		} else {
			i++;
			(label, i) = readuntil(s, i, '}');
			if(i < n && s[i] == '}') i++;
			shape = SH_DIAMOND;
		}
	'>' =>
		i++;
		(label, i) = readuntil(s, i, ']');
		if(i < n && s[i] == ']') i++;
		shape = SH_FLAG;
	}
	return (id, label, shape, i);
}

# Parse edge operator at position i; return (style, hasarrow, label, new_i)
parseedgeop(s: string, i: int): (int, int, string, int)
{
	n := len s;
	while(i < n && (s[i] == ' ' || s[i] == '\t'))
		i++;
	if(i >= n)
		return (-1, 0, "", i);

	style := -1;
	hasarrow := 1;
	label := "";

	# Match edge patterns
	if(i+3 <= n && s[i:i+3] == "==>") {
		style = ES_THICK; i += 3;
	} else if(i+4 <= n && s[i:i+4] == "-.->") {
		style = ES_DASH; i += 4;
	} else if(i+3 <= n && s[i:i+3] == "-.-") {
		style = ES_DASH; hasarrow = 0; i += 3;
	} else if(i+3 <= n && s[i:i+3] == "-->") {
		style = ES_SOLID; i += 3;
	} else if(i+3 <= n && s[i:i+3] == "---") {
		style = ES_LINE; hasarrow = 0; i += 3;
	} else if(i+2 <= n && s[i:i+2] == "--") {
		# --text-->: scan for -->
		j := i + 2;
		for(; j < n && s[j] != '-'; j++)
			;
		if(j+2 < n && s[j:j+3] == "-->") {
			label = trimstr(s[i+2:j]);
			style = ES_SOLID; i = j + 3;
		} else {
			style = ES_SOLID; i = j;
			if(i < n && s[i] == '>') { i++; }
		}
	}

	if(style < 0)
		return (-1, 0, "", i);

	# Check for inline label: -->|text|
	while(i < n && (s[i] == ' ' || s[i] == '\t'))
		i++;
	if(i < n && s[i] == '|' && label == "") {
		i++;
		(label, i) = readuntil(s, i, '|');
		if(i < n && s[i] == '|') i++;
	}
	return (style, hasarrow, label, i);
}

addnode(g: ref FCGraph, id, label: string, shape: int)
{
	for(nl := g.nodes; nl != nil; nl = tl nl)
		if((hd nl).id == id) {
			# Update label/shape only if we have more info
			n := hd nl;
			if(n.label == n.id && label != id)
				n.label = label;
			if(n.shape == SH_RECT && shape != SH_RECT)
				n.shape = shape;
			return;
		}
	node := ref FCNode(id, label, shape, 0, 0, 0, 0, 0, 0);
	g.nodes = node :: g.nodes;
	g.nnodes++;
}

addedge(g: ref FCGraph, src, dst, label: string, style, arrow: int)
{
	e := ref FCEdge(src, dst, label, style, arrow);
	g.edges = e :: g.edges;
	g.nedges++;
}

# ─── Flowchart layout ─────────────────────────────────────────────────────────

layoutflow(g: ref FCGraph, imgw: int)
{
	j, k: int;
	nl: list of ref FCNode;
	el: list of ref FCEdge;
	if(g.nnodes == 0)
		return;

	nodes := revnodes(g.nodes);
	edges := revedges(g.edges);

	# Compute node pixel dimensions
	for(nl = nodes; nl != nil; nl = tl nl) {
		nd := hd nl;
		tw := mfont.width(nd.label);
		nd.w = tw + 2*HPAD;
		if(nd.w < MINNODEW) nd.w = MINNODEW;
		nd.h = mfont.height + 2*VPAD;
		if(nd.h < MINNODEH) nd.h = MINNODEH;
		# Diamond/hex need more room
		if(nd.shape == SH_DIAMOND || nd.shape == SH_HEX) {
			nd.w = nd.w * 3 / 2;
			nd.h = nd.h * 3 / 2;
		}
		if(nd.shape == SH_CIRCLE) {
			d := nd.w;
			if(nd.h > d) d = nd.h;
			nd.w = d; nd.h = d;
		}
	}

	# Build in-degree table (index by node list position)
	na := nodestoarray(nodes, g.nnodes);
	indeg := array[g.nnodes] of {* => 0};

	for(el = edges; el != nil; el = tl el) {
		e := hd el;
		for(j = 0; j < g.nnodes; j++)
			if(na[j].id == e.dst) {
				indeg[j]++;
				break;
			}
	}

	# BFS longest-path layering
	layer := array[g.nnodes] of {* => -1};
	queue := array[g.nnodes] of int;
	qhead := 0; qtail := 0;
	indeg2 := array[g.nnodes] of {* => 0};
	for(j = 0; j < g.nnodes; j++)
		indeg2[j] = indeg[j];

	for(j = 0; j < g.nnodes; j++)
		if(indeg2[j] == 0) {
			layer[j] = 0;
			queue[qtail++] = j;
		}
	# Handle fully cyclic graphs: assign any unassigned nodes to layer 0
	for(j = 0; j < g.nnodes; j++)
		if(layer[j] < 0) {
			layer[j] = 0;
			queue[qtail++] = j;
		}

	while(qhead < qtail) {
		u := queue[qhead++];
		# Propagate to successors
		for(el = edges; el != nil; el = tl el) {
			e := hd el;
			if(na[u].id != e.src) continue;
			for(j = 0; j < g.nnodes; j++) {
				if(na[j].id != e.dst) continue;
				if(layer[u] + 1 > layer[j])
					layer[j] = layer[u] + 1;
				indeg2[j]--;
				if(indeg2[j] == 0)
					queue[qtail++] = j;
				break;
			}
		}
	}

	# Count layers and max column per layer
	nlayers := 0;
	for(j = 0; j < g.nnodes; j++)
		if(layer[j] + 1 > nlayers) nlayers = layer[j] + 1;

	# Assign column positions within each layer (order of BFS discovery)
	col := array[g.nnodes] of {* => 0};
	colcount := array[nlayers] of {* => 0};
	for(j = 0; j < g.nnodes; j++) {
		l := layer[j];
		if(l < 0) l = 0;
		col[j] = colcount[l]++;
	}

	# Compute pixel dimensions per layer
	maxnodeh := 0;
	maxnodew := 0;
	for(nl = nodes; nl != nil; nl = tl nl) {
		nd := hd nl;
		if(nd.h > maxnodeh) maxnodeh = nd.h;
		if(nd.w > maxnodew) maxnodew = nd.w;
	}

	# Image width: accommodate the widest layer
	maxcols := 0;
	for(k = 0; k < nlayers; k++)
		if(colcount[k] > maxcols) maxcols = colcount[k];
	layerw := maxcols * (maxnodew + HGAP) - HGAP;
	iw := layerw + 2 * MARGIN;
	if(iw < imgw) iw = imgw;

	# Assign pixel coordinates
	for(j = 0; j < g.nnodes; j++) {
		l := layer[j];
		if(l < 0) l = 0;
		c := col[j];
		cnt := colcount[l];
		# Centre this layer within image
		lw := cnt * (maxnodew + HGAP) - HGAP;
		startx := (iw - lw) / 2;

		nd := na[j];
		if(g.dir == DIRN_LR || g.dir == DIRN_RL) {
			# LR: layers go left→right, nodes stack top→bottom
			if(g.dir == DIRN_RL)
				nd.x = iw - MARGIN - l * (maxnodew + VGAP) - maxnodew/2;
			else
				nd.x = MARGIN + l * (maxnodew + VGAP) + maxnodew/2;
			lh := cnt * (maxnodeh + HGAP) - HGAP;
			ih := lh + 2 * MARGIN;
			starty := (ih - lh) / 2;
			nd.y = starty + c * (maxnodeh + HGAP) + maxnodeh/2;
		} else {
			# TD or BT
			if(g.dir == DIRN_BT)
				nd.y = 2*MARGIN + (nlayers - 1 - l) * (maxnodeh + VGAP) + maxnodeh/2;
			else
				nd.y = MARGIN + l * (maxnodeh + VGAP) + maxnodeh/2;
			nd.x = startx + c * (maxnodew + HGAP) + maxnodew/2;
		}
		nd.layer = layer[j];
		nd.col = col[j];
	}
}

# Compute flow diagram image dimensions
flowimgdims(g: ref FCGraph, imgw: int): (int, int)
{
	if(g.nnodes == 0)
		return (imgw, 100);

	na := nodestoarray(g.nodes, g.nnodes);
	maxx := 0; maxy := 0;
	for(j := 0; j < g.nnodes; j++) {
		nd := na[j];
		rx := nd.x + nd.w/2 + MARGIN;
		ry := nd.y + nd.h/2 + MARGIN;
		if(rx > maxx) maxx = rx;
		if(ry > maxy) maxy = ry;
	}
	if(maxx < imgw) maxx = imgw;
	return (maxx, maxy);
}

# ─── Flowchart renderer ───────────────────────────────────────────────────────

renderflow(lines: list of string, width: int): (ref Image, string)
{
	k: int;
	g := parseflow(lines);
	if(g.nnodes == 0)
		return rendererror("empty flowchart", width);

	# Reverse lists to preserve declaration order
	g.nodes = revnodes(g.nodes);
	g.edges = revedges(g.edges);
	layoutflow(g, width);
	(iw, ih) := flowimgdims(g, width);

	img := mdisp.newimage(Rect((0,0),(iw,ih)), mdisp.image.chans, 0, Draw->Nofill);
	if(img == nil)
		return (nil, "cannot allocate image");
	img.draw(img.r, cbg, nil, (0,0));

	na := nodestoarray(g.nodes, g.nnodes);
	ea := edgestoarray(g.edges, g.nedges);

	# Draw edges first (behind nodes)
	for(k = 0; k < g.nedges; k++) {
		e := ea[k];
		src := findnode(na, g.nnodes, e.src);
		dst := findnode(na, g.nnodes, e.dst);
		if(src == nil || dst == nil) continue;
		drawfcedge(img, src, dst, e, g.dir);
	}

	# Draw nodes on top
	for(k = 0; k < g.nnodes; k++)
		drawfcnode(img, na[k]);

	return (img, nil);
}

drawfcnode(img: ref Image, nd: ref FCNode)
{
	cx := nd.x; cy := nd.y;
	hw := nd.w / 2; hh := nd.h / 2;
	r := Rect((cx-hw, cy-hh), (cx+hw, cy+hh));

	case nd.shape {
	SH_RECT or SH_SUBR or SH_FLAG =>
		img.draw(r, cnode, nil, (0,0));
		drawrectrect(img, r, cbord);
		if(nd.shape == SH_SUBR) {
			# double vertical bars on left/right
			img.draw(Rect((r.min.x+4, r.min.y), (r.min.x+6, r.max.y)), cbord, nil, (0,0));
			img.draw(Rect((r.max.x-6, r.min.y), (r.max.x-4, r.max.y)), cbord, nil, (0,0));
		}
	SH_ROUND or SH_STADIUM =>
		img.draw(r, cnode, nil, (0,0));
		drawroundrect(img, r, cbord);
	SH_DIAMOND =>
		drawdiamond(img, cx, cy, nd.w, nd.h, cnode, cbord);
	SH_CIRCLE =>
		rad := hw;
		if(hh < rad) rad = hh;
		img.fillellipse(Point(cx,cy), rad, rad, cnode, Point(0,0));
		img.ellipse(Point(cx,cy), rad, rad, 0, cbord, Point(0,0));
	SH_HEX =>
		drawhex(img, cx, cy, nd.w, nd.h, cnode, cbord);
	* =>
		img.draw(r, cnode, nil, (0,0));
		drawrectrect(img, r, cbord);
	}

	# Label
	lw := mfont.width(nd.label);
	lx := cx - lw/2;
	ly := cy - mfont.height/2;
	img.text(Point(lx, ly), ctext, Point(0,0), mfont, nd.label);
}

drawfcedge(img: ref Image, src, dst: ref FCNode, e: ref FCEdge, dir: int)
{
	sp := Point(0,0);

	# Connection points depend on layout direction
	p0, p1: Point;
	if(dir == DIRN_LR) {
		p0 = Point(src.x + src.w/2, src.y);
		p1 = Point(dst.x - dst.w/2, dst.y);
	} else if(dir == DIRN_RL) {
		p0 = Point(src.x - src.w/2, src.y);
		p1 = Point(dst.x + dst.w/2, dst.y);
	} else if(dir == DIRN_BT) {
		p0 = Point(src.x, src.y - src.h/2);
		p1 = Point(dst.x, dst.y + dst.h/2);
	} else {
		# TD (default)
		p0 = Point(src.x, src.y + src.h/2);
		p1 = Point(dst.x, dst.y - dst.h/2);
	}

	col := cbord;
	thick := 0;
	if(e.style == ES_THICK) thick = 1;

	# Choose route: straight if aligned, orthogonal bend otherwise
	if(dir == DIRN_LR || dir == DIRN_RL) {
		if(p0.y == p1.y) {
			drawedgeseg(img, p0, p1, e.style, col, thick);
		} else {
			mid := (p0.x + p1.x) / 2;
			drawedgeseg(img, p0, Point(mid, p0.y), e.style, col, thick);
			drawedgeseg(img, Point(mid, p0.y), Point(mid, p1.y), e.style, col, thick);
			drawedgeseg(img, Point(mid, p1.y), p1, e.style, col, thick);
		}
	} else {
		if(p0.x == p1.x) {
			drawedgeseg(img, p0, p1, e.style, col, thick);
		} else {
			mid := (p0.y + p1.y) / 2;
			drawedgeseg(img, p0, Point(p0.x, mid), e.style, col, thick);
			drawedgeseg(img, Point(p0.x, mid), Point(p1.x, mid), e.style, col, thick);
			drawedgeseg(img, Point(p1.x, mid), p1, e.style, col, thick);
		}
	}

	# Arrowhead at p1
	if(e.arrow) {
		adir := 0;
		if(dir == DIRN_LR) adir = 1;
		else if(dir == DIRN_RL) adir = 3;
		else if(dir == DIRN_BT) adir = 2;
		else adir = 0;
		drawarrowhead(img, p1, adir, col);
	}

	# Edge label at midpoint of middle segment
	if(e.label != "") {
		mx := (p0.x + p1.x) / 2;
		my := (p0.y + p1.y) / 2 - mfont.height - 2;
		lw := mfont.width(e.label);
		img.draw(Rect((mx-lw/2-2, my-1), (mx+lw/2+2, my+mfont.height+1)),
			cnode, nil, (0,0));
		img.text(Point(mx-lw/2, my), ctext2, Point(0,0), mfont, e.label);
	}
	sp = sp;	# suppress unused warning
}

drawedgeseg(img: ref Image, p0, p1: Point, style: int, col: ref Image, thick: int)
{
	if(style == ES_DASH) {
		dashedline(img, p0, p1, col);
		return;
	}
	img.line(p0, p1, Draw->Endsquare, Draw->Endsquare, thick, col, Point(0,0));
}

# ═══════════════════════════════════════════════════════════════════════════════
# ─── PIE CHART ────────────────────────────────────────────────────────────────
# ═══════════════════════════════════════════════════════════════════════════════

parsepiechart(lines: list of string): ref PieChart
{
	p := ref PieChart("", 0, nil, 0);
	for(l := lines; l != nil; l = tl l) {
		s := trimstr(hd l);
		if(s == "" || hasprefix(s, "%%")) continue;
		sl := tolower(s);
		if(hasprefix(sl, "pie")) {
			rest := trimstr(s[3:]);
			rsl := tolower(rest);
			if(hasprefix(rsl, "showdata")) {
				p.showdata = 1;
				rest = trimstr(rest[8:]);
				rsl = tolower(rest);
			}
			if(hasprefix(rsl, "title "))
				p.title = trimstr(rest[6:]);
			continue;
		}
		if(hasprefix(sl, "title ")) {
			p.title = trimstr(s[6:]);
			continue;
		}
		# "label" : value
		if(len s < 3) continue;
		if(s[0] != '"') continue;
		j := 1;
		for(; j < len s && s[j] != '"'; j++)
			;
		if(j >= len s) continue;
		lbl := s[1:j];
		j++;
		while(j < len s && (s[j] == ' ' || s[j] == '\t' || s[j] == ':'))
			j++;
		if(j >= len s) continue;
		val := parsenum(trimstr(s[j:]));
		slice := ref PieSlice(lbl, val);
		p.slices = slice :: p.slices;
		p.nslices++;
	}
	p.slices = revslices(p.slices);
	return p;
}

renderpie(lines: list of string, width: int): (ref Image, string)
{
	sl: list of ref PieSlice;
	p := parsepiechart(lines);
	if(p.nslices == 0)
		return rendererror("pie chart has no slices", width);

	# Layout
	pad := MARGIN;
	titleh := 0;
	if(p.title != "")
		titleh = mfont.height + 8;
	radius := (width - 2*pad) / 3;
	if(radius < 40) radius = 40;
	h := titleh + 2*radius + 2*pad + 4;
	leglineh := mfont.height + 4;
	legh := p.nslices * leglineh + 4;
	if(legh > h - 2*pad) h = legh + 2*pad;
	if(h < radius*2 + 2*pad + titleh + 8)
		h = radius*2 + 2*pad + titleh + 8;

	img := mdisp.newimage(Rect((0,0),(width,h)), mdisp.image.chans, 0, Draw->Nofill);
	if(img == nil) return (nil, "cannot allocate image");
	img.draw(img.r, cbg, nil, (0,0));

	# Title
	ty := pad;
	if(p.title != "") {
		tw := mfont.width(p.title);
		img.text(Point(width/2 - tw/2, ty), ctext, Point(0,0), mfont, p.title);
		ty += titleh;
	}

	cx := pad + radius;
	cy := ty + radius;

	# Compute total (×1024)
	total := 0;
	for(sl = p.slices; sl != nil; sl = tl sl)
		total += (hd sl).value;
	if(total == 0)
		return rendererror("pie chart: all zero values", width);

	# Draw slices
	# alpha=90 in Limbo → 12 o'clock (memarc negates → internal 270°)
	# phi negative → clockwise sweep
	startangle := 90;
	i := 0;
	for(sl = p.slices; sl != nil; sl = tl sl) {
		sv := (hd sl).value;
		phi := -(sv * 360 / total);
		if(phi == 0) phi = -1;
		img.fillarc(Point(cx,cy), radius, radius, cpie[i%8], Point(0,0), startangle, phi);
		# Thin border arc
		img.arc(Point(cx,cy), radius, radius, 0, cbg, Point(0,0), startangle, phi);
		startangle += phi;
		i++;
	}

	# Legend
	lx := cx + radius + 16;
	ly := ty + 4;
	i = 0;
	for(sl = p.slices; sl != nil; sl = tl sl) {
		sv := (hd sl).value;
		# Colour swatch
		img.draw(Rect((lx, ly+2), (lx+14, ly+14)), cpie[i%8], nil, (0,0));
		# Label
		lstr := (hd sl).label;
		if(p.showdata) {
			# Show percentage
			pct := sv * 100 / total;
			lstr += sys->sprint(" (%d%%)", pct);
		}
		img.text(Point(lx+18, ly), ctext, Point(0,0), mfont, lstr);
		ly += leglineh;
		i++;
	}

	return (img, nil);
}

# ═══════════════════════════════════════════════════════════════════════════════
# ─── SEQUENCE DIAGRAM ─────────────────────────────────────────────────────────
# ═══════════════════════════════════════════════════════════════════════════════

parseseq(lines: list of string): ref SeqDiag
{
	d := ref SeqDiag(nil, 0, nil, 0);
	for(l := lines; l != nil; l = tl l) {
		s := trimstr(hd l);
		if(s == "" || hasprefix(s, "%%")) continue;
		sl := tolower(s);
		if(hasprefix(sl, "sequencediagram") || hasprefix(sl, "title ") ||
				hasprefix(sl, "autonumber") || hasprefix(sl, "activate") ||
				hasprefix(sl, "deactivate"))
			continue;
		# participant
		if(hasprefix(sl, "participant ") || hasprefix(sl, "actor ")) {
			skip := 12;
			if(hasprefix(sl, "actor ")) skip = 6;
			rest := trimstr(s[skip:]);
			id := rest; alias := rest;
			# "Name as Alias"
			ai := findkw(rest, " as ");
			if(ai >= 0) {
				id = trimstr(rest[0:ai]);
				alias = trimstr(rest[ai+4:]);
			}
			addseqpart(d, id, alias);
			continue;
		}
		# Note over / Note left of / Note right of
		if(hasprefix(sl, "note ")) {
			rest := trimstr(s[5:]);
			# Find ":"
			ci := 0;
			for(; ci < len rest && rest[ci] != ':'; ci++)
				;
			if(ci < len rest) {
				text := trimstr(rest[ci+1:]);
				m := ref SeqMsg("", "", "", SM_SOLID, 1, text);
				d.msgs = m :: d.msgs;
				d.nmsgs++;
			}
			continue;
		}
		# Message: A ->> B : text  or  A -->> B : text
		ai := findkw(s, "->>");
		dai := findkw(s, "-->>");
		mtype := SM_SOLID;
		mlen := 3;
		mi := ai;
		if(dai >= 0 && (ai < 0 || dai < ai)) {
			mi = dai; mtype = SM_DASH; mlen = 4;
		}
		if(mi >= 0) {
			from := trimstr(s[0:mi]);
			rest := trimstr(s[mi+mlen:]);
			# Find ":"
			ci := 0;
			for(; ci < len rest && rest[ci] != ':'; ci++)
				;
			dst := trimstr(rest[0:ci]);
			text := "";
			if(ci < len rest)
				text = trimstr(rest[ci+1:]);
			# Auto-register participants
			addseqpart(d, from, from);
			addseqpart(d, dst, dst);
			m := ref SeqMsg(from, dst, text, mtype, 0, "");
			d.msgs = m :: d.msgs;
			d.nmsgs++;
		}
	}
	d.parts = revseqparts(d.parts);
	d.msgs = revseqmsgs(d.msgs);
	return d;
}

addseqpart(d: ref SeqDiag, id, alias: string)
{
	for(pl := d.parts; pl != nil; pl = tl pl)
		if((hd pl).id == id) return;
	p := ref SeqPart(id, alias, d.nparts);
	d.parts = p :: d.parts;
	d.nparts++;
}

seqpartidx(d: ref SeqDiag, id: string): int
{
	for(pl := d.parts; pl != nil; pl = tl pl)
		if((hd pl).id == id) return (hd pl).idx;
	return -1;
}

renderseq(lines: list of string, width: int): (ref Image, string)
{
	pl: list of ref SeqPart;
	d := parseseq(lines);
	if(d.nparts == 0)
		return rendererror("sequence diagram has no participants", width);

	pad := MARGIN;
	titleh := mfont.height + 6;

	# Image dimensions
	ncols := d.nparts;
	iw := ncols * SEQ_COLW + 2 * pad;
	if(iw < width) iw = width;
	msgsh := d.nmsgs * SEQ_ROWH + SEQ_ROWH;
	ih := 2*pad + 2*titleh + 2*SEQ_BOXH + msgsh;

	img := mdisp.newimage(Rect((0,0),(iw,ih)), mdisp.image.chans, 0, Draw->Nofill);
	if(img == nil) return (nil, "cannot allocate image");
	img.draw(img.r, cbg, nil, (0,0));

	# Participant centres
	xcol := array[ncols] of int;
	i := 0;
	for(pl = d.parts; pl != nil; pl = tl pl) {
		xcol[i] = pad + i*SEQ_COLW + SEQ_COLW/2;
		i++;
	}

	topy := pad;
	boty := ih - pad - SEQ_BOXH;
	firstmegy := topy + SEQ_BOXH + 8;

	# Draw participant boxes (top and bottom)
	i = 0;
	for(pl = d.parts; pl != nil; pl = tl pl) {
		pt := hd pl;
		cx := xcol[i];
		alias := pt.alias;
		if(alias == "") alias = pt.id;
		# Top box
		bx := cx - SEQ_BOXW/2;
		drawparticipantbox(img, bx, topy, alias);
		# Bottom box
		drawparticipantbox(img, bx, boty, alias);
		# Lifeline (dashed vertical)
		dashedline(img, Point(cx, topy+SEQ_BOXH), Point(cx, boty), cgrid);
		i++;
	}

	# Draw messages
	my := firstmegy;
	for(ml := d.msgs; ml != nil; ml = tl ml) {
		m := hd ml;
		if(m.isnote) {
			# Draw note as a text strip
			nw := mfont.width(m.notetext) + 8;
			nx := iw/2 - nw/2;
			img.draw(Rect((nx-2, my-2), (nx+nw+2, my+mfont.height+2)), csect, nil, (0,0));
			img.text(Point(nx+4, my), ctext, Point(0,0), mfont, m.notetext);
		} else {
			fi := seqpartidx(d, m.from);
			ti := seqpartidx(d, m.dst);
			if(fi < 0 || ti < 0 || fi >= ncols || ti >= ncols) {
				my += SEQ_ROWH; continue;
			}
			x0 := xcol[fi];
			x1 := xcol[ti];
			drawseqmsg(img, x0, x1, my, m);
		}
		my += SEQ_ROWH;
	}

	return (img, nil);
}

drawparticipantbox(img: ref Image, bx, by: int, label: string)
{
	r := Rect((bx, by), (bx+SEQ_BOXW, by+SEQ_BOXH));
	img.draw(r, cnode, nil, (0,0));
	drawrectrect(img, r, cbord);
	lw := mfont.width(label);
	cx := bx + SEQ_BOXW/2;
	img.text(Point(cx-lw/2, by+(SEQ_BOXH-mfont.height)/2), ctext, Point(0,0), mfont, label);
}

drawseqmsg(img: ref Image, x0, x1, y: int, m: ref SeqMsg)
{
	col := cbord;
	if(m.mtype == SM_DASH) col = ctext2;

	if(x0 == x1) {
		# Self-message: right-angle loop
		loop := 24;
		drawedgeseg(img, Point(x0, y), Point(x0+loop, y), m.mtype, col, 0);
		drawedgeseg(img, Point(x0+loop, y), Point(x0+loop, y+SEQ_ROWH/2), m.mtype, col, 0);
		drawedgeseg(img, Point(x0+loop, y+SEQ_ROWH/2), Point(x0, y+SEQ_ROWH/2), m.mtype, col, 0);
		drawarrowhead(img, Point(x0, y+SEQ_ROWH/2), 3, col);
		if(m.text != "")
			img.text(Point(x0+loop+4, y-mfont.height), ctext, Point(0,0), mfont, m.text);
		return;
	}

	# Horizontal arrow
	img.line(Point(x0, y), Point(x1, y), Draw->Endsquare, Draw->Endsquare, 0, col, Point(0,0));
	if(x1 > x0)
		drawarrowhead(img, Point(x1, y), 1, col);
	else
		drawarrowhead(img, Point(x1, y), 3, col);
	# Label
	if(m.text != "") {
		mx := (x0 + x1) / 2;
		lw := mfont.width(m.text);
		img.text(Point(mx-lw/2, y-mfont.height-1), ctext, Point(0,0), mfont, m.text);
	}
}

# ═══════════════════════════════════════════════════════════════════════════════
# ─── GANTT CHART ──────────────────────────────────────────────────────────────
# ═══════════════════════════════════════════════════════════════════════════════

parsegantt(lines: list of string): ref GanttChart
{
	g := ref GanttChart("", nil, 0, 999999, 0);
	cursect := "Default";

	for(l := lines; l != nil; l = tl l) {
		s := trimstr(hd l);
		if(s == "" || hasprefix(s, "%%")) continue;
		sl := tolower(s);
		if(hasprefix(sl, "gantt"))  continue;
		if(hasprefix(sl, "title ")) { g.title = trimstr(s[6:]); continue; }
		if(hasprefix(sl, "dateformat ") || hasprefix(sl, "dateformat\t")) continue;
		if(hasprefix(sl, "axisformat ") || hasprefix(sl, "axisformat\t")) continue;
		if(hasprefix(sl, "excludes ")) continue;
		if(hasprefix(sl, "todaymarker")) continue;
		if(hasprefix(sl, "section ")) {
			cursect = trimstr(s[8:]);
			continue;
		}
		# Task line: label : [modifiers,] startdate, duration
		#   or: label :id, after id2, duration
		ci := 0;
		for(; ci < len s && s[ci] != ':'; ci++)
			;
		if(ci >= len s) continue;
		label := trimstr(s[0:ci]);
		rest := trimstr(s[ci+1:]);

		t := ref GTask(cursect, label, "", 0, 0, 0, "", 0, 1);
		parseganttrest(rest, t);
		g.tasks = t :: g.tasks;
		g.ntasks++;
	}
	g.tasks = revtasks(g.tasks);

	# Resolve "after" dependencies and compute date range
	ta := taskstoarray(g.tasks, g.ntasks);
	resolvetaskdeps(ta, g.ntasks);
	for(k := 0; k < g.ntasks; k++) {
		t := ta[k];
		if(t.startday < g.minday) g.minday = t.startday;
		endday := t.startday + t.durdays;
		if(endday > g.maxday) g.maxday = endday;
	}
	if(g.minday >= g.maxday) {
		g.minday = 0;
		g.maxday = 30;
	}
	return g;
}

parseganttrest(s: string, t: ref GTask)
{
	# Parse: [crit,] [active,] [done,] [id,] [after id,] startdate, duration
	# or: [id,] after othertask, duration
	parts := splittokens(s, ',');
	for(pl := parts; pl != nil; pl = tl pl) {
		tok := trimstr(hd pl);
		tl2 := tolower(tok);
		if(tl2 == "crit")   { t.crit = 1; continue; }
		if(tl2 == "active") { t.active = 1; continue; }
		if(tl2 == "done")   { t.done = 1; continue; }
		if(hasprefix(tl2, "after ")) { t.after = trimstr(tok[6:]); continue; }
		# If it looks like a date (YYYY-MM-DD)
		if(isdate(tok)) { t.startday = parsedate(tok); continue; }
		# If it looks like a duration (e.g. "7d", "2w")
		if(isduration(tok)) { t.durdays = parsedur(tok); continue; }
		# Else treat as ID
		if(t.id == "") t.id = tok;
	}
	if(t.durdays <= 0) t.durdays = 1;
}

resolvetaskdeps(ta: array of ref GTask, n: int)
{
	# Two passes to resolve chains
	for(pass := 0; pass < 2; pass++)
	for(i := 0; i < n; i++) {
		t := ta[i];
		if(t.after == "") continue;
		for(j := 0; j < n; j++) {
			if(ta[j].id == t.after || ta[j].label == t.after) {
				t.startday = ta[j].startday + ta[j].durdays;
				break;
			}
		}
	}
}

rendergantt(lines: list of string, width: int): (ref Image, string)
{
	g := parsegantt(lines);
	if(g.ntasks == 0)
		return rendererror("gantt chart has no tasks", width);

	pad := MARGIN;
	titleh := 0;
	if(g.title != "")
		titleh = mfont.height + 8;

	plotw := width - 2*pad - GNT_LBLW;
	if(plotw < 80) plotw = 80;

	ih := pad + titleh + GNT_HDRY + g.ntasks*GNT_ROWH + pad;

	# Count sections for extra section title rows
	cursect := "";
	nsects := 0;
	for(tl2 := g.tasks; tl2 != nil; tl2 = tl tl2) {
		t := hd tl2;
		if(t.section != cursect) { nsects++; cursect = t.section; }
	}
	ih += nsects * GNT_SECTH;

	img := mdisp.newimage(Rect((0,0),(width,ih)), mdisp.image.chans, 0, Draw->Nofill);
	if(img == nil) return (nil, "cannot allocate image");
	img.draw(img.r, cbg, nil, (0,0));

	ty := pad;
	if(g.title != "") {
		tw := mfont.width(g.title);
		img.text(Point(width/2 - tw/2, ty), ctext, Point(0,0), mfont, g.title);
		ty += titleh;
	}

	# Date header
	dayrange := g.maxday - g.minday;
	if(dayrange <= 0) dayrange = 1;
	# Draw every N days as marker (choose N so markers are ≥40px apart)
	markevery := 1;
	while(markevery * plotw / dayrange < 40)
		markevery++;

	hdrr := Rect((pad + GNT_LBLW, ty), (pad + GNT_LBLW + plotw, ty + GNT_HDRY));
	img.draw(hdrr, csect, nil, (0,0));
	k := 0;
	while(k * markevery < dayrange) {
		dx := k * markevery * plotw / dayrange;
		mx := pad + GNT_LBLW + dx;
		# Tick
		img.draw(Rect((mx, ty+GNT_HDRY-4), (mx+1, ty+GNT_HDRY)), ctext, nil, (0,0));
		# Day number label
		lbl := sys->sprint("+%d", k*markevery);
		img.text(Point(mx+2, ty+4), ctext2, Point(0,0), mfont, lbl);
		k++;
	}
	ty += GNT_HDRY;

	# Tasks
	cursect = "";
	for(tl3 := g.tasks; tl3 != nil; tl3 = tl tl3) {
		t := hd tl3;
		# Section header
		if(t.section != cursect) {
			cursect = t.section;
			img.draw(Rect((pad, ty), (width-pad, ty+GNT_SECTH)), csect, nil, (0,0));
			img.text(Point(pad+4, ty+2), ctext, Point(0,0), mfont, cursect);
			ty += GNT_SECTH;
		}
		# Label column
		lbl := t.label;
		lw := mfont.width(lbl);
		if(lw > GNT_LBLW - 8) {
			# Truncate
			for(; len lbl > 0 && mfont.width(lbl+"…") > GNT_LBLW-8; )
				lbl = lbl[0:len lbl-1];
			lbl += "…";
		}
		img.text(Point(pad, ty+(GNT_ROWH-mfont.height)/2), ctext, Point(0,0), mfont, lbl);

		# Bar
		sd := t.startday - g.minday;
		ed := sd + t.durdays;
		bx := pad + GNT_LBLW + sd * plotw / dayrange;
		bw := (ed - sd) * plotw / dayrange;
		if(bw < 3) bw = 3;
		br := Rect((bx, ty+2), (bx+bw, ty+GNT_ROWH-2));
		barcol := cacc;
		if(t.crit) barcol = cred;
		else if(t.done) barcol = cgreen;
		else if(t.active) barcol = cyel;
		img.draw(br, barcol, nil, (0,0));

		# Grid line
		img.draw(Rect((pad+GNT_LBLW, ty+GNT_ROWH-1), (width-pad, ty+GNT_ROWH)), cgrid, nil, (0,0));
		ty += GNT_ROWH;
	}

	return (img, nil);
}

# ═══════════════════════════════════════════════════════════════════════════════
# ─── XY CHART ─────────────────────────────════════════════════════════════════
# ═══════════════════════════════════════════════════════════════════════════════

parsexychart(lines: list of string): ref XYChart
{
	pl: list of string;
	c := ref XYChart("", nil, 0, 0, 0, nil);
	c.ylower = 0; c.yupper = 1024;	# default range 0..1

	for(l := lines; l != nil; l = tl l) {
		s := trimstr(hd l);
		if(s == "" || hasprefix(s, "%%")) continue;
		sl := tolower(s);
		if(hasprefix(sl, "xychart")) continue;
		if(hasprefix(sl, "title ")) {
			# Strip surrounding quotes if present
			t := trimstr(s[6:]);
			if(len t >= 2 && t[0] == '"')
				t = t[1:len t-1];
			c.title = t;
			continue;
		}
		if(hasprefix(sl, "x-axis ") || hasprefix(sl, "x-axis[")) {
			# x-axis [Lab1, Lab2, ...]  or  x-axis --> N
			bi := 0;
			for(; bi < len s && s[bi] != '['; bi++)
				;
			if(bi < len s) {
				ei := bi+1;
				for(; ei < len s && s[ei] != ']'; ei++)
					;
				inner := s[bi+1:ei];
				parts := splittokens(inner, ',');
				n := 0;
				for(pl = parts; pl != nil; pl = tl pl)
					n++;
				c.xlabels = array[n] of string;
				c.nxlbl = n;
				k := 0;
				for(pl = parts; pl != nil; pl = tl pl) {
					c.xlabels[k] = trimstr(hd pl);
					if(len c.xlabels[k] >= 2 && c.xlabels[k][0] == '"')
						c.xlabels[k] = c.xlabels[k][1:len c.xlabels[k]-1];
					k++;
				}
			}
			continue;
		}
		if(hasprefix(sl, "y-axis ")) {
			# y-axis "label" lower --> upper
			rest := trimstr(s[7:]);
			# Strip quoted label
			if(len rest > 0 && rest[0] == '"') {
				j := 1;
				for(; j < len rest && rest[j] != '"'; j++)
					;
				rest = trimstr(rest[j+1:]);
			}
			# parse "lower --> upper"
			ai := findkw(rest, "-->");
			if(ai >= 0) {
				c.ylower = parsenum(trimstr(rest[0:ai]));
				c.yupper = parsenum(trimstr(rest[ai+3:]));
			}
			continue;
		}
		if(hasprefix(sl, "bar ") || hasprefix(sl, "bar[")) {
			ser := parsexyseries(s, 1);
			if(ser != nil) c.series = ser :: c.series;
			continue;
		}
		if(hasprefix(sl, "line ") || hasprefix(sl, "line[")) {
			ser := parsexyseries(s, 0);
			if(ser != nil) c.series = ser :: c.series;
			continue;
		}
	}
	c.series = revxyseries(c.series);
	return c;
}

parsexyseries(s: string, isbar: int): ref XYSeries
{
	pl: list of string;
	bi := 0;
	for(; bi < len s && s[bi] != '['; bi++)
		;
	if(bi >= len s) return nil;
	ei := bi+1;
	for(; ei < len s && s[ei] != ']'; ei++)
		;
	inner := s[bi+1:ei];
	parts := splittokens(inner, ',');
	n := 0;
	for(pl = parts; pl != nil; pl = tl pl)
		n++;
	if(n == 0) return nil;
	ser := ref XYSeries(isbar, array[n] of int, n);
	k := 0;
	for(pl = parts; pl != nil; pl = tl pl) {
		ser.vals[k] = parsenum(trimstr(hd pl));
		k++;
	}
	return ser;
}

renderxy(lines: list of string, width: int): (ref Image, string)
{
	sl: list of ref XYSeries;
	xi: int;
	c := parsexychart(lines);

	pad := MARGIN;
	titleh := 0;
	if(c.title != "")
		titleh = mfont.height + 8;

	plotw := width - 2*pad - XY_AXISW;
	if(plotw < 60) plotw = 60;
	ih := pad + titleh + XY_PLOTH + XY_AXISH + pad;

	img := mdisp.newimage(Rect((0,0),(width,ih)), mdisp.image.chans, 0, Draw->Nofill);
	if(img == nil) return (nil, "cannot allocate image");
	img.draw(img.r, cbg, nil, (0,0));

	ty := pad;
	if(c.title != "") {
		tw := mfont.width(c.title);
		img.text(Point(width/2 - tw/2, ty), ctext, Point(0,0), mfont, c.title);
		ty += titleh;
	}

	plotx := pad + XY_AXISW;
	ploty := ty;
	plotboty := ploty + XY_PLOTH;

	# Y range (×1024)
	yrange := c.yupper - c.ylower;
	if(yrange <= 0) yrange = 1024;

	# Grid lines (5 horizontal)
	for(gi := 0; gi <= 4; gi++) {
		gy := ploty + gi * XY_PLOTH / 4;
		img.draw(Rect((plotx, gy), (plotx+plotw, gy+1)), cgrid, nil, (0,0));
		# Y axis label
		yval := c.yupper - gi * yrange / 4;
		lbl := sys->sprint("%d", yval / 1024);
		lw := mfont.width(lbl);
		img.text(Point(plotx - lw - 4, gy), ctext2, Point(0,0), mfont, lbl);
	}

	# Axes
	img.line(Point(plotx, ploty), Point(plotx, plotboty), Draw->Endsquare, Draw->Endsquare, 0, cbord, Point(0,0));
	img.line(Point(plotx, plotboty), Point(plotx+plotw, plotboty), Draw->Endsquare, Draw->Endsquare, 0, cbord, Point(0,0));

	ncols := c.nxlbl;
	if(ncols <= 0) {
		# Infer from first series
		for(sl = c.series; sl != nil; sl = tl sl)
			if((hd sl).nvals > ncols) ncols = (hd sl).nvals;
	}
	if(ncols <= 0) ncols = 1;

	colw := plotw / ncols;
	nser := 0;
	for(sl = c.series; sl != nil; sl = tl sl)
		nser++;

	# Draw series
	si := 0;
	for(sl = c.series; sl != nil; sl = tl sl) {
		ser := hd sl;
		col := cpie[si % 8];

		if(ser.isbar) {
			# Bars
			barw := colw * 7 / (8 * (nser + 1));
			if(barw < 2) barw = 2;
			for(xi = 0; xi < ser.nvals && xi < ncols; xi++) {
				v := ser.vals[xi];
				# ypix: top of bar relative to ploty
				ypix := XY_PLOTH - int(big(v - c.ylower) * big XY_PLOTH / big yrange);
				if(ypix < 0) ypix = 0;
				if(ypix > XY_PLOTH) ypix = XY_PLOTH;
				bx := plotx + xi*colw + si*barw + (colw - nser*barw)/2;
				br := Rect((bx, ploty+ypix), (bx+barw, plotboty));
				img.draw(br, col, nil, (0,0));
			}
		} else {
			# Line
			if(ser.nvals > 0) {
				pts := array[ser.nvals] of Point;
				for(xi = 0; xi < ser.nvals && xi < ncols; xi++) {
					v := ser.vals[xi];
					ypix := XY_PLOTH - int(big(v - c.ylower) * big XY_PLOTH / big yrange);
					if(ypix < 0) ypix = 0;
					if(ypix > XY_PLOTH) ypix = XY_PLOTH;
					pts[xi] = Point(plotx + xi*colw + colw/2, ploty+ypix);
				}
				img.poly(pts[0:ser.nvals], Draw->Enddisc, Draw->Enddisc, 1, col, Point(0,0));
				for(xi = 0; xi < ser.nvals; xi++) {
					img.fillellipse(pts[xi], 3, 3, col, Point(0,0));
				}
			}
		}
		si++;
	}

	# X axis labels
	for(xi = 0; xi < ncols; xi++) {
		lbl := "";
		if(xi < c.nxlbl) lbl = c.xlabels[xi];
		else lbl = sys->sprint("%d", xi+1);
		lw := mfont.width(lbl);
		cx := plotx + xi*colw + colw/2;
		img.text(Point(cx-lw/2, plotboty+4), ctext2, Point(0,0), mfont, lbl);
	}

	return (img, nil);
}

# ═══════════════════════════════════════════════════════════════════════════════
# ─── Drawing utilities ────────────────────────────────────────────────────────
# ═══════════════════════════════════════════════════════════════════════════════

# Arrowhead pointing in direction dir: 0=down, 1=right, 2=up, 3=left
drawarrowhead(img: ref Image, tip: Point, dir: int, col: ref Image)
{
	AW: con AHEADW;
	AH: con AHEADLEN;
	pts := array[3] of Point;
	case dir {
	0 =>	# down
		pts[0] = Point(tip.x,    tip.y);
		pts[1] = Point(tip.x-AW, tip.y-AH);
		pts[2] = Point(tip.x+AW, tip.y-AH);
	1 =>	# right
		pts[0] = Point(tip.x,    tip.y);
		pts[1] = Point(tip.x-AH, tip.y-AW);
		pts[2] = Point(tip.x-AH, tip.y+AW);
	2 =>	# up
		pts[0] = Point(tip.x,    tip.y);
		pts[1] = Point(tip.x-AW, tip.y+AH);
		pts[2] = Point(tip.x+AW, tip.y+AH);
	3 =>	# left
		pts[0] = Point(tip.x,    tip.y);
		pts[1] = Point(tip.x+AH, tip.y-AW);
		pts[2] = Point(tip.x+AH, tip.y+AW);
	}
	img.fillpoly(pts, ~0, col, Point(0,0));
}

# Simulated dashed line (8px on, 4px skip)
dashedline(img: ref Image, p0, p1: Point, col: ref Image)
{
	DASH: con 8;
	GAP:  con 4;
	dx := p1.x - p0.x;
	dy := p1.y - p0.y;
	dist := dx;
	if(dist < 0) dist = -dist;
	if(dy < 0 && -dy > dist) dist = -dy;
	else if(dy > dist) dist = dy;
	if(dist == 0) return;

	step := DASH + GAP;
	nstep := dist / step + 1;
	for(i := 0; i < nstep; i++) {
		t0 := i * step;
		t1 := t0 + DASH;
		if(t0 > dist) break;
		if(t1 > dist) t1 = dist;
		x0 := p0.x + dx * t0 / dist;
		y0 := p0.y + dy * t0 / dist;
		x1 := p0.x + dx * t1 / dist;
		y1 := p0.y + dy * t1 / dist;
		img.line(Point(x0,y0), Point(x1,y1), Draw->Endsquare, Draw->Endsquare, 0, col, Point(0,0));
	}
}

# Draw a 1px rectangle border
drawrectrect(img: ref Image, r: Rect, col: ref Image)
{
	img.draw(Rect(r.min, (r.max.x, r.min.y+1)), col, nil, (0,0));
	img.draw(Rect((r.min.x, r.max.y-1), r.max), col, nil, (0,0));
	img.draw(Rect(r.min, (r.min.x+1, r.max.y)), col, nil, (0,0));
	img.draw(Rect((r.max.x-1, r.min.y), r.max), col, nil, (0,0));
}

# Draw a rounded rectangle (fill + border)
drawroundrect(img: ref Image, r: Rect, col: ref Image)
{
	rad := 4;
	if(r.dy() < 2*rad+2) rad = r.dy()/2 - 1;
	if(rad < 1) rad = 1;
	# 4 corner ellipses to overdraw background on corners
	corners := array[4] of Point;
	corners[0] = Point(r.min.x+rad, r.min.y+rad);
	corners[1] = Point(r.max.x-rad-1, r.min.y+rad);
	corners[2] = Point(r.max.x-rad-1, r.max.y-rad-1);
	corners[3] = Point(r.min.x+rad, r.max.y-rad-1);
	for(i := 0; i < 4; i++)
		img.ellipse(corners[i], rad, rad, 0, col, Point(0,0));
	# Top and bottom horizontal bars
	img.draw(Rect((r.min.x+rad, r.min.y), (r.max.x-rad, r.min.y+1)), col, nil, (0,0));
	img.draw(Rect((r.min.x+rad, r.max.y-1), (r.max.x-rad, r.max.y)), col, nil, (0,0));
	# Left and right vertical bars
	img.draw(Rect((r.min.x, r.min.y+rad), (r.min.x+1, r.max.y-rad)), col, nil, (0,0));
	img.draw(Rect((r.max.x-1, r.min.y+rad), (r.max.x, r.max.y-rad)), col, nil, (0,0));
}

# Draw a diamond / rhombus shape
drawdiamond(img: ref Image, cx, cy, w, h: int, fill, border: ref Image)
{
	pts := array[4] of Point;
	pts[0] = Point(cx,    cy-h/2);	# top
	pts[1] = Point(cx+w/2, cy);	# right
	pts[2] = Point(cx,    cy+h/2);	# bottom
	pts[3] = Point(cx-w/2, cy);	# left
	img.fillpoly(pts, ~0, fill, Point(0,0));
	img.poly(pts, Draw->Endsquare, Draw->Endsquare, 0, border, Point(0,0));
	# close the outline
	img.line(pts[3], pts[0], Draw->Endsquare, Draw->Endsquare, 0, border, Point(0,0));
}

# Draw a hexagon
drawhex(img: ref Image, cx, cy, w, h: int, fill, border: ref Image)
{
	pts := array[6] of Point;
	qw := w / 4;
	hw := w / 2;
	hh := h / 2;
	pts[0] = Point(cx+qw, cy-hh);
	pts[1] = Point(cx+hw, cy);
	pts[2] = Point(cx+qw, cy+hh);
	pts[3] = Point(cx-qw, cy+hh);
	pts[4] = Point(cx-hw, cy);
	pts[5] = Point(cx-qw, cy-hh);
	img.fillpoly(pts, ~0, fill, Point(0,0));
	img.poly(pts, Draw->Endsquare, Draw->Endsquare, 0, border, Point(0,0));
	img.line(pts[5], pts[0], Draw->Endsquare, Draw->Endsquare, 0, border, Point(0,0));
}

# Error placeholder image
rendererror(msg: string, width: int): (ref Image, string)
{
	h := mfont.height + 2*MARGIN;
	img := mdisp.newimage(Rect((0,0),(width,h)), mdisp.image.chans, 0, Draw->Nofill);
	if(img == nil)
		return (nil, msg);
	img.draw(img.r, cbg, nil, (0,0));
	tw := mfont.width(msg);
	img.text(Point(width/2-tw/2, MARGIN), cred, Point(0,0), mfont, msg);
	return (img, nil);
}

# ═══════════════════════════════════════════════════════════════════════════════
# ─── String / parsing utilities ───────────────────────────────────────────────
# ═══════════════════════════════════════════════════════════════════════════════

splitlines(s: string): list of string
{
	lines: list of string;
	i := 0; n := len s;
	start := 0;
	while(i < n) {
		if(s[i] == '\n') {
			lines = s[start:i] :: lines;
			start = i + 1;
		}
		i++;
	}
	if(start < n)
		lines = s[start:n] :: lines;
	rev: list of string;
	for(; lines != nil; lines = tl lines)
		rev = hd lines :: rev;
	return rev;
}

trimstr(s: string): string
{
	i := 0; n := len s;
	while(i < n && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r'))
		i++;
	while(n > i && (s[n-1] == ' ' || s[n-1] == '\t' || s[n-1] == '\r' || s[n-1] == '\n'))
		n--;
	if(i >= n) return "";
	return s[i:n];
}

tolower(s: string): string
{
	r := s;
	for(i := 0; i < len r; i++) {
		c := r[i];
		if(c >= 'A' && c <= 'Z')
			r[i] = c + ('a' - 'A');
	}
	return r;
}

hasprefix(s, pfx: string): int
{
	return len s >= len pfx && s[0:len pfx] == pfx;
}

# Read until stop character; return (content, new_i).
# i should point to first char of content (after opening delimiter).
readuntil(s: string, i: int, stop: int): (string, int)
{
	n := len s;
	start := i;
	while(i < n && s[i] != stop)
		i++;
	return (s[start:i], i);
}

# Find first occurrence of keyword kw in s; return index or -1
findkw(s, kw: string): int
{
	nl := len s - len kw;
	for(i := 0; i <= nl; i++)
		if(s[i:i+len kw] == kw) return i;
	return -1;
}

# Split s by delimiter ch; returns list of tokens
splittokens(s: string, ch: int): list of string
{
	toks: list of string;
	i := 0; n := len s;
	start := 0;
	while(i < n) {
		if(s[i] == ch) {
			toks = s[start:i] :: toks;
			start = i + 1;
		}
		i++;
	}
	toks = s[start:n] :: toks;
	rev: list of string;
	for(; toks != nil; toks = tl toks)
		rev = hd toks :: rev;
	return rev;
}

# Parse "3.14" or "42" as ×1024 fixed-point integer; integers only
parsenum(s: string): int
{
	i: int;
	s = trimstr(s);
	if(s == "") return 0;
	neg := 0;
	if(len s > 0 && s[0] == '-') { neg = 1; s = s[1:]; }
	dot := -1;
	for(i = 0; i < len s; i++)
		if(s[i] == '.') { dot = i; break; }
	result := 0;
	if(dot < 0) {
		for(i = 0; i < len s; i++) {
			c := s[i];
			if(c < '0' || c > '9') break;
			result = result * 10 + (c - '0');
		}
		result *= 1024;
	} else {
		for(i = 0; i < dot; i++) {
			c := s[i];
			if(c < '0' || c > '9') break;
			result = result * 10 + (c - '0');
		}
		result *= 1024;
		# Fractional part (up to 4 digits)
		scale := 1024;
		for(i = dot+1; i < len s && i <= dot+4; i++) {
			c := s[i];
			if(c < '0' || c > '9') break;
			scale = scale * 10;
			result = result * 10 + (c - '0') * 1024 / scale * scale / 1024;
		}
		# simpler fractional: just integer truncation
		frac := 0;
		fscale := 1;
		for(i = dot+1; i < len s && i-dot <= 4; i++) {
			c := s[i];
			if(c < '0' || c > '9') break;
			frac = frac * 10 + (c - '0');
			fscale *= 10;
		}
		result = (result / 1024) * 1024 + frac * 1024 / fscale;
	}
	if(neg) result = -result;
	return result;
}

# Parse YYYY-MM-DD → days since 2000-01-01
parsedate(s: string): int
{
	s = trimstr(s);
	if(len s < 10) return 0;
	y := intfrom(s, 0, 4);
	m := intfrom(s, 5, 7);
	d := intfrom(s, 8, 10);
	moff := array[13] of int;
	moff[0]=0; moff[1]=0; moff[2]=31; moff[3]=59; moff[4]=90;
	moff[5]=120; moff[6]=151; moff[7]=181; moff[8]=212;
	moff[9]=243; moff[10]=273; moff[11]=304; moff[12]=334;
	if(m < 1) m = 1;
	if(m > 12) m = 12;
	y -= 2000;
	leaps := y / 4;
	return y*365 + leaps + moff[m] + d - 1;
}

isdate(s: string): int
{
	s = trimstr(s);
	if(len s < 10) return 0;
	if(s[4] != '-' || s[7] != '-') return 0;
	for(i := 0; i < 4; i++)
		if(s[i] < '0' || s[i] > '9') return 0;
	return 1;
}

# Parse duration "7d", "2w", "1M" → days
parsedur(s: string): int
{
	s = trimstr(s);
	if(s == "") return 1;
	n := 0;
	i := 0;
	while(i < len s && s[i] >= '0' && s[i] <= '9') {
		n = n * 10 + (s[i] - '0');
		i++;
	}
	if(n == 0) n = 1;
	if(i < len s) {
		case s[i] {
		'w' or 'W' => n *= 7;
		'M'        => n *= 30;
		'y' or 'Y' => n *= 365;
		}
	}
	return n;
}

isduration(s: string): int
{
	s = trimstr(s);
	if(s == "") return 0;
	i := 0;
	while(i < len s && s[i] >= '0' && s[i] <= '9')
		i++;
	return i > 0 && i < len s && (s[i] == 'd' || s[i] == 'w' || s[i] == 'M' || s[i] == 'y' || s[i] == 'Y' || s[i] == 'W');
}

# Parse integer from s[a:b]
intfrom(s: string, a, b: int): int
{
	n := 0;
	for(i := a; i < b && i < len s; i++) {
		c := s[i];
		if(c < '0' || c > '9') break;
		n = n * 10 + (c - '0');
	}
	return n;
}

# ═══════════════════════════════════════════════════════════════════════════════
# ─── List/array conversion helpers ────────────────────────────────────────────
# ═══════════════════════════════════════════════════════════════════════════════

nodestoarray(l: list of ref FCNode, n: int): array of ref FCNode
{
	a := array[n] of ref FCNode;
	i := 0;
	for(; l != nil && i < n; l = tl l)
		a[i++] = hd l;
	return a;
}

edgestoarray(l: list of ref FCEdge, n: int): array of ref FCEdge
{
	a := array[n] of ref FCEdge;
	i := 0;
	for(; l != nil && i < n; l = tl l)
		a[i++] = hd l;
	return a;
}

taskstoarray(l: list of ref GTask, n: int): array of ref GTask
{
	a := array[n] of ref GTask;
	i := 0;
	for(; l != nil && i < n; l = tl l)
		a[i++] = hd l;
	return a;
}

findnode(na: array of ref FCNode, n: int, id: string): ref FCNode
{
	for(i := 0; i < n; i++)
		if(na[i].id == id) return na[i];
	return nil;
}

revnodes(l: list of ref FCNode): list of ref FCNode
{
	r: list of ref FCNode;
	for(; l != nil; l = tl l) r = hd l :: r;
	return r;
}

revedges(l: list of ref FCEdge): list of ref FCEdge
{
	r: list of ref FCEdge;
	for(; l != nil; l = tl l) r = hd l :: r;
	return r;
}

revslices(l: list of ref PieSlice): list of ref PieSlice
{
	r: list of ref PieSlice;
	for(; l != nil; l = tl l) r = hd l :: r;
	return r;
}

revseqparts(l: list of ref SeqPart): list of ref SeqPart
{
	r: list of ref SeqPart;
	for(; l != nil; l = tl l) r = hd l :: r;
	return r;
}

revseqmsgs(l: list of ref SeqMsg): list of ref SeqMsg
{
	r: list of ref SeqMsg;
	for(; l != nil; l = tl l) r = hd l :: r;
	return r;
}

revtasks(l: list of ref GTask): list of ref GTask
{
	r: list of ref GTask;
	for(; l != nil; l = tl l) r = hd l :: r;
	return r;
}

revxyseries(l: list of ref XYSeries): list of ref XYSeries
{
	r: list of ref XYSeries;
	for(; l != nil; l = tl l) r = hd l :: r;
	return r;
}
