implement Rlayout;

include "sys.m";
	sys: Sys;

include "draw.m";
	drawm: Draw;
	Display, Image, Font, Rect, Point: import drawm;

include "rlayout.m";

display: ref Display;

# Layout state used during rendering
Lstate: adt {
	img: ref Image;       # Target image
	style: ref Style;
	x: int;               # Current x position
	y: int;               # Current y position (top of line)
	lineheight: int;      # Height of current line
	maxwidth: int;        # Usable width (style.width - 2*margin)
	indent: int;          # Current left indent (for lists, blockquotes)
};

init(d: ref Draw->Display)
{
	sys = load Sys Sys->PATH;
	drawm = load Draw Draw->PATH;
	display = d;
}

render(doc: list of ref DocNode, style: ref Style): (ref Draw->Image, int)
{
	if(style == nil || style.font == nil)
		return (nil, 0);

	width := style.width;
	if(width <= 0)
		width = 800;

	# First pass: calculate total height needed
	height := measureheight(doc, style);
	if(height < style.font.height * 2)
		height = style.font.height * 2;

	# Add bottom margin
	height += style.margin;

	# Create target image
	r := Rect(Point(0, 0), Point(width, height));
	img := display.newimage(r, drawm->RGB24, 0, drawm->White);
	if(img == nil)
		return (nil, 0);

	# Fill background
	img.draw(r, style.bgcolor, nil, Point(0, 0));

	# Set up layout state
	ls := ref Lstate(
		img, style,
		style.margin,           # x
		style.margin,           # y
		style.font.height,      # lineheight
		width - 2 * style.margin, # maxwidth
		0                       # indent
	);

	# Second pass: render
	renderblocks(ls, doc);

	return (img, ls.y + ls.lineheight);
}

# Measure total height needed for a document
measureheight(doc: list of ref DocNode, style: ref Style): int
{
	h := style.margin;
	fh := style.font.height;
	cfh := fh;
	if(style.codefont != nil)
		cfh = style.codefont.height;
	maxw := style.width - 2 * style.margin;

	for(; doc != nil; doc = tl doc){
		node := hd doc;
		case node.kind {
		Npara =>
			# Estimate paragraph height from wrapped text
			txt := flattentext(node.children);
			lines := wrapcount(txt, style.font, maxw);
			h += lines * fh + fh/2;  # paragraph spacing
		Nheading =>
			h += fh * 2;  # heading + spacing
		Ncodeblock =>
			txt := "";
			if(node.text != nil)
				txt = node.text;
			else
				txt = flattentext(node.children);
			nlines := 1;
			for(i := 0; i < len txt; i++)
				if(txt[i] == '\n')
					nlines++;
			h += nlines * cfh + fh;  # code lines + padding + spacing
		Nbullet or Nnumber =>
			txt := flattentext(node.children);
			lines := wrapcount(txt, style.font, maxw - 24);
			h += lines * fh + 2;
		Nhrule =>
			h += fh;
		Nblockquote =>
			txt := flattentext(node.children);
			lines := wrapcount(txt, style.font, maxw - 20);
			h += lines * fh + fh/2;
		* =>
			h += fh;
		}
	}
	return h;
}

# Count lines needed to wrap text at given width
wrapcount(text: string, font: ref Font, width: int): int
{
	if(text == nil || len text == 0)
		return 1;
	if(width <= 0)
		return 1;

	lines := 1;
	linestart := 0;
	lastspace := -1;

	for(i := 0; i < len text; i++){
		c := text[i];
		if(c == '\n'){
			lines++;
			linestart = i + 1;
			lastspace = -1;
			continue;
		}
		if(c == ' ' || c == '\t')
			lastspace = i;

		seg := text[linestart:i+1];
		w := font.width(seg);
		if(w > width && linestart < i){
			lines++;
			if(lastspace > linestart){
				linestart = lastspace + 1;
				lastspace = -1;
			} else
				linestart = i;
		}
	}
	return lines;
}

# Render a list of block-level nodes
renderblocks(ls: ref Lstate, doc: list of ref DocNode)
{
	for(; doc != nil; doc = tl doc){
		node := hd doc;
		case node.kind {
		Npara =>
			renderpara(ls, node);
		Nheading =>
			renderheading(ls, node);
		Ncodeblock =>
			rendercodeblock(ls, node);
		Nbullet =>
			renderbullet(ls, node);
		Nnumber =>
			rendernumber(ls, node);
		Nhrule =>
			renderhrule(ls);
		Nblockquote =>
			renderblockquote(ls, node);
		* =>
			# Treat as paragraph
			renderpara(ls, node);
		}
	}
}

# Render a paragraph
renderpara(ls: ref Lstate, node: ref DocNode)
{
	ls.x = ls.style.margin + ls.indent;
	renderinlines(ls, node.children);
	newline(ls);
	ls.y += ls.style.font.height / 3;  # Paragraph spacing
}

# Render a heading
renderheading(ls: ref Lstate, node: ref DocNode)
{
	ls.y += ls.style.font.height / 3;  # Space before heading
	ls.x = ls.style.margin + ls.indent;

	# Render heading text in bold (uppercase prefix for emphasis)
	font := ls.style.font;
	color := ls.style.fgcolor;

	# Draw text
	txt := flattentext(node.children);
	if(node.aux <= 1){
		# H1: draw with extra vertical space
		ls.img.text(Point(ls.x, ls.y + font.ascent), color, Point(0, 0), font, txt);
		ls.y += font.height;
		# Draw underline for H1
		y := ls.y + 2;
		enx := ls.style.margin + ls.maxwidth;
		ls.img.line(Point(ls.style.margin, y), Point(enx, y), drawm->Endsquare, drawm->Endsquare, 1, color, Point(0, 0));
		ls.y += 4;
	} else if(node.aux == 2){
		# H2: normal size with underline
		ls.img.text(Point(ls.x, ls.y + font.ascent), color, Point(0, 0), font, txt);
		ls.y += font.height;
		# Dashed underline for H2
		y := ls.y + 1;
		enx := ls.x + font.width(txt);
		ls.img.line(Point(ls.x, y), Point(enx, y), drawm->Endsquare, drawm->Endsquare, 0, color, Point(0, 0));
		ls.y += 3;
	} else {
		# H3+: just render with spacing
		ls.img.text(Point(ls.x, ls.y + font.ascent), color, Point(0, 0), font, txt);
		ls.y += font.height;
	}
	ls.x = ls.style.margin + ls.indent;
	ls.y += ls.style.font.height / 4;  # Space after heading
}

# Render a code block
rendercodeblock(ls: ref Lstate, node: ref DocNode)
{
	font := ls.style.codefont;
	if(font == nil)
		font = ls.style.font;

	txt := "";
	if(node.text != nil)
		txt = node.text;
	else
		txt = flattentext(node.children);

	# Measure the code block
	nlines := 1;
	for(i := 0; i < len txt; i++)
		if(txt[i] == '\n')
			nlines++;

	pad := 6;
	blockh := nlines * font.height + 2 * pad;

	# Draw background rectangle
	x0 := ls.style.margin + ls.indent;
	x1 := ls.style.margin + ls.maxwidth;
	bgr := Rect(Point(x0, ls.y), Point(x1, ls.y + blockh));
	ls.img.draw(bgr, ls.style.codebgcolor, nil, Point(0, 0));

	# Render each line
	ty := ls.y + pad;
	linestart := 0;
	for(i = 0; i <= len txt; i++){
		if(i == len txt || txt[i] == '\n'){
			line := "";
			if(i > linestart)
				line = txt[linestart:i];
			ls.img.text(Point(x0 + pad, ty + font.ascent), ls.style.fgcolor, Point(0, 0), font, line);
			ty += font.height;
			linestart = i + 1;
		}
	}

	ls.y += blockh + ls.style.font.height / 3;
	ls.x = ls.style.margin + ls.indent;
}

# Render a bullet list item
renderbullet(ls: ref Lstate, node: ref DocNode)
{
	font := ls.style.font;
	bulletindent := 20;

	# Draw bullet character
	bx := ls.style.margin + ls.indent + 6;
	by := ls.y + font.ascent;
	ls.img.text(Point(bx, by), ls.style.fgcolor, Point(0, 0), font, "•");

	# Render content with indent
	oldi := ls.indent;
	ls.indent += bulletindent;
	ls.x = ls.style.margin + ls.indent;
	renderinlines(ls, node.children);
	newline(ls);
	ls.indent = oldi;
}

# Render a numbered list item
rendernumber(ls: ref Lstate, node: ref DocNode)
{
	font := ls.style.font;
	numindent := 24;

	# Draw number
	numstr := sys->sprint("%d.", node.aux);
	nx := ls.style.margin + ls.indent + 2;
	ny := ls.y + font.ascent;
	ls.img.text(Point(nx, ny), ls.style.fgcolor, Point(0, 0), font, numstr);

	# Render content with indent
	oldi := ls.indent;
	ls.indent += numindent;
	ls.x = ls.style.margin + ls.indent;
	renderinlines(ls, node.children);
	newline(ls);
	ls.indent = oldi;
}

# Render a horizontal rule
renderhrule(ls: ref Lstate)
{
	ls.y += ls.style.font.height / 3;
	y := ls.y + ls.style.font.height / 2;
	x0 := ls.style.margin;
	x1 := ls.style.margin + ls.maxwidth;
	ls.img.line(Point(x0, y), Point(x1, y), drawm->Endsquare, drawm->Endsquare, 0, ls.style.fgcolor, Point(0, 0));
	ls.y += ls.style.font.height;
}

# Render a blockquote
renderblockquote(ls: ref Lstate, node: ref DocNode)
{
	# Draw left border line
	bx := ls.style.margin + ls.indent + 4;
	y0 := ls.y;

	# Render content indented
	oldi := ls.indent;
	ls.indent += 16;
	ls.x = ls.style.margin + ls.indent;
	renderinlines(ls, node.children);
	newline(ls);
	y1 := ls.y;
	ls.indent = oldi;

	# Draw the quote bar
	ls.img.line(Point(bx, y0), Point(bx, y1), drawm->Endsquare, drawm->Endsquare, 1, ls.style.linkcolor, Point(0, 0));

	ls.y += ls.style.font.height / 4;
}

# Render inline nodes (text, bold, italic, code, links) with word wrapping
renderinlines(ls: ref Lstate, nodes: list of ref DocNode)
{
	for(; nodes != nil; nodes = tl nodes){
		node := hd nodes;
		case node.kind {
		Ntext =>
			rendertext(ls, node.text, ls.style.font, ls.style.fgcolor, 0);
		Nbold =>
			# Bold: render twice with 1px offset for faux bold
			txt := flattentext(node.children);
			renderbold(ls, txt);
		Nitalic =>
			# Italic: render with underline (we lack italic fonts)
			txt := flattentext(node.children);
			rendertext(ls, txt, ls.style.font, ls.style.fgcolor, 1);
		Ncode =>
			# Inline code: monospace with background
			font := ls.style.codefont;
			if(font == nil)
				font = ls.style.font;
			renderinlinecode(ls, node.text, font);
		Nlink =>
			txt := flattentext(node.children);
			rendertext(ls, txt, ls.style.font, ls.style.linkcolor, 1);
		Nnewline =>
			newline(ls);
		* =>
			# Recurse for nested structures
			if(node.children != nil)
				renderinlines(ls, node.children);
			else if(node.text != nil)
				rendertext(ls, node.text, ls.style.font, ls.style.fgcolor, 0);
		}
	}
}

# Render text with word wrapping
rendertext(ls: ref Lstate, text: string, font: ref Font, color: ref Image, underline: int)
{
	if(text == nil || len text == 0)
		return;

	maxright := ls.style.margin + ls.indent + ls.maxwidth - ls.indent;

	# Process word by word
	i := 0;
	for(;;){
		# Skip to next non-space or end
		wordstart := i;
		while(i < len text && text[i] != ' ' && text[i] != '\t' && text[i] != '\n')
			i++;

		word := "";
		if(i > wordstart)
			word = text[wordstart:i];

		if(len word > 0){
			ww := font.width(word);
			# Check if word fits on current line
			if(ls.x + ww > maxright && ls.x > ls.style.margin + ls.indent){
				newline(ls);
			}
			# Draw word
			pt := Point(ls.x, ls.y + font.ascent);
			ls.img.text(pt, color, Point(0, 0), font, word);
			if(underline){
				uy := ls.y + font.ascent + 2;
				ls.img.line(Point(ls.x, uy), Point(ls.x + ww, uy),
					drawm->Endsquare, drawm->Endsquare, 0, color, Point(0, 0));
			}
			ls.x += ww;
		}

		# Handle space / newline after word
		if(i >= len text)
			break;
		if(text[i] == '\n'){
			newline(ls);
			i++;
		} else {
			# Space - add space width
			if(ls.x > ls.style.margin + ls.indent)
				ls.x += font.width(" ");
			i++;
		}
	}
}

# Render bold text (faux bold: draw twice with 1px x offset)
renderbold(ls: ref Lstate, text: string)
{
	if(text == nil || len text == 0)
		return;

	font := ls.style.font;
	color := ls.style.fgcolor;
	maxright := ls.style.margin + ls.indent + ls.maxwidth - ls.indent;

	i := 0;
	for(;;){
		wordstart := i;
		while(i < len text && text[i] != ' ' && text[i] != '\t' && text[i] != '\n')
			i++;

		word := "";
		if(i > wordstart)
			word = text[wordstart:i];

		if(len word > 0){
			ww := font.width(word);
			if(ls.x + ww > maxright && ls.x > ls.style.margin + ls.indent)
				newline(ls);

			pt := Point(ls.x, ls.y + font.ascent);
			# Draw twice for faux bold
			ls.img.text(pt, color, Point(0, 0), font, word);
			ls.img.text(Point(pt.x + 1, pt.y), color, Point(0, 0), font, word);
			ls.x += ww + 1;
		}

		if(i >= len text)
			break;
		if(text[i] == '\n'){
			newline(ls);
			i++;
		} else {
			if(ls.x > ls.style.margin + ls.indent)
				ls.x += font.width(" ");
			i++;
		}
	}
}

# Render inline code with background
renderinlinecode(ls: ref Lstate, text: string, font: ref Font)
{
	if(text == nil)
		return;

	pad := 3;
	tw := font.width(text);
	maxright := ls.style.margin + ls.indent + ls.maxwidth - ls.indent;

	if(ls.x + tw + 2*pad > maxright && ls.x > ls.style.margin + ls.indent)
		newline(ls);

	# Draw background
	bgr := Rect(Point(ls.x, ls.y), Point(ls.x + tw + 2*pad, ls.y + font.height));
	ls.img.draw(bgr, ls.style.codebgcolor, nil, Point(0, 0));

	# Draw text
	pt := Point(ls.x + pad, ls.y + font.ascent);
	ls.img.text(pt, ls.style.fgcolor, Point(0, 0), font, text);
	ls.x += tw + 2*pad + 2;
}

# Move to next line
newline(ls: ref Lstate)
{
	ls.y += ls.style.font.height;
	ls.x = ls.style.margin + ls.indent;
}

# Flatten all inline children to plain text
flattentext(nodes: list of ref DocNode): string
{
	s := "";
	for(; nodes != nil; nodes = tl nodes){
		node := hd nodes;
		if(node.text != nil)
			s += node.text;
		if(node.children != nil)
			s += flattentext(node.children);
	}
	return s;
}

# ---- Markdown Parser (shared with mdrender and external callers) ----

# Parse markdown text into a list of DocNode blocks.
parsemd(text: string): list of ref DocNode
{
	doc: list of ref DocNode;
	lines := pmd_splitlines(text);

	i := 0;
	nlines := len lines;
	for(;;){
		if(i >= nlines)
			break;
		line := lines[i];

		# Blank line - skip
		if(pmd_isblank(line)){
			i++;
			continue;
		}

		# Code block (```)
		if(len line >= 3 && line[0:3] == "```"){
			(block, ni) := pmd_parsecodeblock(lines, i);
			doc = block :: doc;
			i = ni;
			continue;
		}

		# Heading (#)
		if(len line > 0 && line[0] == '#'){
			(heading, ni) := pmd_parseheading(line);
			if(heading != nil){
				doc = heading :: doc;
				if(ni > i)
					i = ni;
				else
					i++;
				continue;
			}
		}

		# Horizontal rule (---, ***, ___)
		if(pmd_ishrule(line)){
			doc = ref DocNode(Nhrule, nil, nil, 0) :: doc;
			i++;
			continue;
		}

		# Blockquote (>)
		if(len line > 0 && line[0] == '>'){
			(bq, ni) := pmd_parseblockquote(lines, i, nlines);
			doc = bq :: doc;
			i = ni;
			continue;
		}

		# Bullet list (- or *)
		if(len line >= 2 && (line[0] == '-' || line[0] == '*') && line[1] == ' '){
			(item, ni) := pmd_parsebullet(lines, i, nlines);
			doc = item :: doc;
			i = ni;
			continue;
		}

		# Numbered list (1. 2. etc)
		if(len line >= 3 && line[0] >= '0' && line[0] <= '9'){
			(item, ni) := pmd_parsenumber(lines, i, nlines);
			if(item != nil){
				doc = item :: doc;
				i = ni;
				continue;
			}
		}

		# Default: paragraph
		(para, ni) := pmd_parsepara(lines, i, nlines);
		doc = para :: doc;
		i = ni;
	}

	return pmd_reverselist(doc);
}

pmd_parseheading(line: string): (ref DocNode, int)
{
	level := 0;
	i := 0;
	while(i < len line && line[i] == '#'){
		level++;
		i++;
	}
	if(level == 0 || level > 6)
		return (nil, 0);
	while(i < len line && line[i] == ' ')
		i++;

	text := "";
	if(i < len line)
		text = line[i:];
	while(len text > 0 && text[len text - 1] == '#')
		text = text[:len text - 1];
	while(len text > 0 && text[len text - 1] == ' ')
		text = text[:len text - 1];

	children := pmd_parseinline(text);
	return (ref DocNode(Nheading, nil, children, level), 0);
}

pmd_parsecodeblock(lines: array of string, start: int): (ref DocNode, int)
{
	i := start + 1;
	code := "";

	while(i < len lines){
		if(len lines[i] >= 3 && lines[i][0:3] == "```"){
			i++;
			break;
		}
		if(len code > 0)
			code += "\n";
		code += lines[i];
		i++;
	}

	return (ref DocNode(Ncodeblock, code, nil, 0), i);
}

pmd_parseblockquote(lines: array of string, start, nlines: int): (ref DocNode, int)
{
	text := "";
	i := start;
	while(i < nlines){
		line := lines[i];
		if(len line == 0 || line[0] != '>')
			break;
		content := "";
		j := 1;
		if(j < len line && line[j] == ' ')
			j++;
		if(j < len line)
			content = line[j:];
		if(len text > 0)
			text += " ";
		text += content;
		i++;
	}

	children := pmd_parseinline(text);
	return (ref DocNode(Nblockquote, nil, children, 0), i);
}

pmd_parsebullet(lines: array of string, start, nlines: int): (ref DocNode, int)
{
	text := lines[start][2:];
	i := start + 1;
	while(i < nlines && len lines[i] > 0 && (lines[i][0] == ' ' || lines[i][0] == '\t')){
		text += " " + pmd_stripws(lines[i]);
		i++;
	}

	children := pmd_parseinline(text);
	return (ref DocNode(Nbullet, nil, children, 0), i);
}

pmd_parsenumber(lines: array of string, start, nlines: int): (ref DocNode, int)
{
	line := lines[start];
	i := 0;
	while(i < len line && line[i] >= '0' && line[i] <= '9')
		i++;
	if(i == 0 || i >= len line || line[i] != '.')
		return (nil, start);
	num := int line[0:i];
	i++;
	if(i < len line && line[i] == ' ')
		i++;

	text := "";
	if(i < len line)
		text = line[i:];
	j := start + 1;
	while(j < nlines && len lines[j] > 0 && (lines[j][0] == ' ' || lines[j][0] == '\t')){
		text += " " + pmd_stripws(lines[j]);
		j++;
	}

	children := pmd_parseinline(text);
	return (ref DocNode(Nnumber, nil, children, num), j);
}

pmd_parsepara(lines: array of string, start, nlines: int): (ref DocNode, int)
{
	text := "";
	i := start;
	while(i < nlines){
		line := lines[i];
		if(pmd_isblank(line))
			break;
		if(i > start){
			if(len line > 0 && line[0] == '#')
				break;
			if(len line >= 3 && line[0:3] == "```")
				break;
			if(pmd_ishrule(line))
				break;
			if(len line > 0 && line[0] == '>')
				break;
			if(len line >= 2 && (line[0] == '-' || line[0] == '*') && line[1] == ' ')
				break;
			if(len line >= 3 && line[0] >= '0' && line[0] <= '9' && pmd_hasdotspace(line))
				break;
		}
		if(len text > 0)
			text += " ";
		text += line;
		i++;
	}

	children := pmd_parseinline(text);
	return (ref DocNode(Npara, nil, children, 0), i);
}

pmd_parseinline(text: string): list of ref DocNode
{
	nodes: list of ref DocNode;
	i := 0;
	plain := "";

	while(i < len text){
		c := text[i];

		# Bold: **text**
		if(c == '*' && i+1 < len text && text[i+1] == '*'){
			if(len plain > 0){
				nodes = ref DocNode(Ntext, plain, nil, 0) :: nodes;
				plain = "";
			}
			end := pmd_findclose(text, i+2, "**");
			if(end > 0){
				inner := text[i+2:end];
				nodes = ref DocNode(Nbold, nil, ref DocNode(Ntext, inner, nil, 0) :: nil, 0) :: nodes;
				i = end + 2;
				continue;
			}
		}

		# Italic: *text*  (but not **)
		if(c == '*' && !(i+1 < len text && text[i+1] == '*')){
			if(len plain > 0){
				nodes = ref DocNode(Ntext, plain, nil, 0) :: nodes;
				plain = "";
			}
			end := pmd_findclose(text, i+1, "*");
			if(end > 0){
				inner := text[i+1:end];
				nodes = ref DocNode(Nitalic, nil, ref DocNode(Ntext, inner, nil, 0) :: nil, 0) :: nodes;
				i = end + 1;
				continue;
			}
		}

		# Inline code: `text`
		if(c == '`'){
			if(len plain > 0){
				nodes = ref DocNode(Ntext, plain, nil, 0) :: nodes;
				plain = "";
			}
			end := pmd_findclose(text, i+1, "`");
			if(end > 0){
				inner := text[i+1:end];
				nodes = ref DocNode(Ncode, inner, nil, 0) :: nodes;
				i = end + 1;
				continue;
			}
		}

		# Link: [text](url)
		if(c == '['){
			if(len plain > 0){
				nodes = ref DocNode(Ntext, plain, nil, 0) :: nodes;
				plain = "";
			}
			(linknode, ni) := pmd_parselink(text, i);
			if(linknode != nil){
				nodes = linknode :: nodes;
				i = ni;
				continue;
			}
		}

		plain[len plain] = c;
		i++;
	}

	if(len plain > 0)
		nodes = ref DocNode(Ntext, plain, nil, 0) :: nodes;

	return pmd_reverselist(nodes);
}

pmd_findclose(text: string, start: int, delim: string): int
{
	dlen := len delim;
	for(i := start; i <= len text - dlen; i++){
		if(text[i:i+dlen] == delim)
			return i;
	}
	return -1;
}

pmd_parselink(text: string, start: int): (ref DocNode, int)
{
	i := start + 1;
	while(i < len text && text[i] != ']')
		i++;
	if(i >= len text)
		return (nil, start + 1);

	linktext := text[start+1:i];
	i++;

	if(i >= len text || text[i] != '(')
		return (nil, start + 1);
	i++;

	j := i;
	while(j < len text && text[j] != ')')
		j++;
	if(j >= len text)
		return (nil, start + 1);
	j++;

	children := ref DocNode(Ntext, linktext, nil, 0) :: nil;
	return (ref DocNode(Nlink, nil, children, 0), j);
}

pmd_splitlines(text: string): array of string
{
	nlines := 1;
	for(i := 0; i < len text; i++)
		if(text[i] == '\n')
			nlines++;

	lines := array[nlines] of string;
	li := 0;
	start := 0;
	j := 0;
	for(j = 0; j < len text; j++){
		if(text[j] == '\n'){
			lines[li++] = text[start:j];
			start = j + 1;
		}
	}
	if(start <= len text)
		lines[li] = text[start:];
	return lines;
}

pmd_isblank(line: string): int
{
	for(i := 0; i < len line; i++)
		if(line[i] != ' ' && line[i] != '\t' && line[i] != '\r')
			return 0;
	return 1;
}

pmd_ishrule(line: string): int
{
	if(len line < 3)
		return 0;
	c := line[0];
	if(c != '-' && c != '*' && c != '_')
		return 0;
	count := 0;
	for(i := 0; i < len line; i++){
		if(line[i] == c)
			count++;
		else if(line[i] != ' ')
			return 0;
	}
	return count >= 3;
}

pmd_hasdotspace(line: string): int
{
	for(i := 0; i < len line; i++){
		if(line[i] == '.' && i+1 < len line && line[i+1] == ' ')
			return 1;
		if(line[i] < '0' || line[i] > '9')
			return 0;
	}
	return 0;
}

pmd_stripws(s: string): string
{
	i := 0;
	while(i < len s && (s[i] == ' ' || s[i] == '\t'))
		i++;
	if(i >= len s)
		return "";
	return s[i:];
}

pmd_reverselist(l: list of ref DocNode): list of ref DocNode
{
	r: list of ref DocNode;
	for(; l != nil; l = tl l)
		r = hd l :: r;
	return r;
}

# Extract plain text from an entire document tree
totext(doc: list of ref DocNode): string
{
	s := "";
	for(; doc != nil; doc = tl doc){
		node := hd doc;
		case node.kind {
		Npara or Nblockquote =>
			s += flattentext(node.children) + "\n\n";
		Nheading =>
			s += flattentext(node.children) + "\n\n";
		Ncodeblock =>
			if(node.text != nil)
				s += node.text + "\n\n";
			else
				s += flattentext(node.children) + "\n\n";
		Nbullet =>
			s += "• " + flattentext(node.children) + "\n";
		Nnumber =>
			s += sys->sprint("%d. ", node.aux) + flattentext(node.children) + "\n";
		Nhrule =>
			s += "---\n\n";
		* =>
			if(node.text != nil)
				s += node.text;
			if(node.children != nil)
				s += flattentext(node.children);
			s += "\n";
		}
	}
	return s;
}
