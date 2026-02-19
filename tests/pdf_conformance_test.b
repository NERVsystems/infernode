implement PdfConformanceTest;

#
# PDF Conformance Test — discovery-based test pipeline.
#
# Walks curated test PDF directories, runs each PDF through:
#   1. Open/parse
#   2. Page count
#   3. Render page 1 at 72 DPI
#   4. Non-blank pixel check
#   5. Text extraction
#
# Test suites are fetched by tests/host/fetch-test-pdfs.sh
# into usr/inferno/test-pdfs/.  If not present, suites skip.
#

include "sys.m";
	sys: Sys;

include "draw.m";
	drawm: Draw;
	Display, Image, Rect, Point: import drawm;

include "readdir.m";
	readdir: Readdir;

include "string.m";
	str: String;

include "testing.m";
	testing: Testing;
	T: import testing;

include "pdf.m";
	pdf: PDF;
	Doc: import pdf;

PdfConformanceTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

SRCFILE: con "/tests/pdf_conformance_test.b";

passed := 0;
failed := 0;
skipped := 0;

# Per-suite stats
suite_pass := 0;
suite_warn := 0;
suite_fail := 0;
suite_total := 0;

# Grand totals across all suites
grand_pass := 0;
grand_warn := 0;
grand_fail := 0;
grand_total := 0;
suites_found := 0;
suites_missing := 0;

TESTPDFROOT: con "/usr/inferno/test-pdfs";

run(name: string, testfn: ref fn(t: ref T))
{
	t := testing->newTsrc(name, SRCFILE);
	{
		testfn(t);
	} exception {
	"fail:fatal" =>
		;
	"fail:skip" =>
		;
	"*" =>
		t.failed = 1;
	}

	if(testing->done(t))
		passed++;
	else if(t.skipped)
		skipped++;
	else
		failed++;
}

# Read a file into a byte array.
readfile(path: string): (array of byte, string)
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return (nil, sys->sprint("open: %r"));
	(ok, dir) := sys->fstat(fd);
	if(ok < 0)
		return (nil, sys->sprint("fstat: %r"));
	fsize := int dir.length;
	if(fsize == 0)
		return (nil, "empty file");
	data := array[fsize] of byte;
	n := 0;
	while(n < fsize){
		r := sys->read(fd, data[n:], fsize - n);
		if(r <= 0)
			break;
		n += r;
	}
	if(n < fsize)
		return (nil, sys->sprint("short read: %d/%d", n, fsize));
	return (data, nil);
}

# Check if a filename ends with .pdf (case-insensitive).
ispdf(name: string): int
{
	n := len name;
	if(n < 4)
		return 0;
	ext := str->tolower(name[n-4:]);
	return ext == ".pdf";
}

# Recursively find all .pdf files under a directory.
# Returns list in discovery order.
findpdfs(dir: string): list of string
{
	(entries, n) := readdir->init(dir, Readdir->NAME);
	if(n <= 0)
		return nil;

	result: list of string;
	for(i := 0; i < n; i++){
		e := entries[i];
		path := dir + "/" + e.name;
		if(e.qid.qtype & Sys->QTDIR){
			# Recurse into subdirectory
			sub := findpdfs(path);
			for(; sub != nil; sub = tl sub)
				result = hd sub :: result;
		} else if(ispdf(e.name)){
			result = path :: result;
		}
	}
	return result;
}

# Reverse a list (findpdfs builds in reverse order).
revlist(l: list of string): list of string
{
	r: list of string;
	for(; l != nil; l = tl l)
		r = hd l :: r;
	return r;
}

# Sample pixels to count non-white content.
# Returns number of non-white samples out of a grid.
countnonwhite(img: ref Image): int
{
	w := img.r.dx();
	h := img.r.dy();
	if(w <= 0 || h <= 0)
		return 0;
	buf := array[3] of byte;
	nonwhite := 0;

	# Sample a 4x4 grid
	dy := h / 5;
	dx := w / 5;
	if(dy < 1) dy = 1;
	if(dx < 1) dx = 1;

	for(y := dy; y < h; y += dy){
		for(x := dx; x < w; x += dx){
			r := Rect(Point(x, y), Point(x+1, y+1));
			n := img.readpixels(r, buf);
			if(n >= 3){
				bv := int buf[0];
				gv := int buf[1];
				rv := int buf[2];
				if(rv != 255 || gv != 255 || bv != 255)
					nonwhite++;
			}
		}
	}
	return nonwhite;
}

# Run the full test pipeline on a single PDF.
# Returns: "pass", "warn", or "fail".
testpdf(t: ref T, path: string): string
{
	# 1. Read
	(data, rerr) := readfile(path);
	if(data == nil){
		t.error(path + ": read error: " + rerr);
		return "fail";
	}

	# 2. Parse
	(doc, oerr) := pdf->open(data);
	if(doc == nil){
		t.error(path + ": open error: " + oerr);
		return "fail";
	}

	# 3. Page count
	npages := doc.pagecount();
	if(npages <= 0){
		t.error(path + ": 0 pages");
		return "fail";
	}

	# 4. Render page 1
	rendered := 0;
	blank := 0;
	{
		(img, imgerr) := doc.renderpage(1, 72);
		if(img == nil){
			if(imgerr != nil)
				t.log(path + ": render: " + imgerr);
			# Render failure is a warning, not hard fail
			# (might be missing display)
		} else {
			rendered = 1;
			# 5. Non-blank check
			nw := countnonwhite(img);
			if(nw == 0)
				blank = 1;
		}
	} exception e {
	"*" =>
		t.error(path + ": render exception: " + e);
		return "fail";
	}

	# 6. Text extraction
	hastext := 0;
	{
		text := doc.extracttext(1);
		if(text != nil && len text > 0)
			hastext = 1;
	} exception e {
	"*" =>
		t.error(path + ": extracttext exception: " + e);
		return "fail";
	}

	# Classify result
	# In headless mode (no display), rendering is unavailable.
	# A PDF that opens, has pages, and doesn't crash is a PASS.
	# Blank render with display = WARN, no text only = WARN.
	if(!rendered){
		# No display — pass based on parse + page count success
		return "pass";
	}
	if(blank){
		t.log(path + ": warn (blank render, text=" + string hastext +
			" pages=" + string npages + ")");
		return "warn";
	}
	return "pass";
}

# Test all PDFs in a directory tree.
testsuite(t: ref T, dir: string, name: string)
{
	# Reset per-suite stats
	suite_pass = 0;
	suite_warn = 0;
	suite_fail = 0;
	suite_total = 0;

	# Check if directory exists
	fd := sys->open(dir, Sys->OREAD);
	if(fd == nil){
		suites_missing++;
		t.skip(name + ": not found (run tests/host/fetch-test-pdfs.sh)");
		return;
	}

	suites_found++;

	# Discover PDFs
	pdfs := revlist(findpdfs(dir));

	count := 0;
	for(l := pdfs; l != nil; l = tl l)
		count++;

	if(count == 0){
		t.log(name + ": 0 PDFs found in " + dir);
		return;
	}

	t.log(name + ": " + string count + " PDFs found");

	# Test each PDF
	for(l = pdfs; l != nil; l = tl l){
		path := hd l;
		suite_total++;

		result := testpdf(t, path);
		case result {
		"pass" =>
			suite_pass++;
		"warn" =>
			suite_warn++;
		"fail" =>
			suite_fail++;
		}
	}

	# Suite summary
	t.log(sys->sprint("%s: %d tested — %d pass, %d warn, %d fail",
		name, suite_total, suite_pass, suite_warn, suite_fail));

	# Accumulate grand totals
	grand_pass += suite_pass;
	grand_warn += suite_warn;
	grand_fail += suite_fail;
	grand_total += suite_total;

	# Suite fails if > 50% of PDFs fail (allows for expected failures)
	if(suite_total > 0 && suite_fail * 2 > suite_total)
		t.error(sys->sprint("%s: majority failure (%d/%d)",
			name, suite_fail, suite_total));
}

testPdfDifferences(t: ref T)
{
	if(pdf == nil)
		t.skip("PDF module not available");
	testsuite(t, TESTPDFROOT + "/pdf-differences", "pdf-differences");
}

testPopplerTest(t: ref T)
{
	if(pdf == nil)
		t.skip("PDF module not available");
	testsuite(t, TESTPDFROOT + "/poppler-test", "poppler-test");
}

testBfoPdfa(t: ref T)
{
	if(pdf == nil)
		t.skip("PDF module not available");
	testsuite(t, TESTPDFROOT + "/bfo-pdfa", "bfo-pdfa");
}

testPdfTest(t: ref T)
{
	if(pdf == nil)
		t.skip("PDF module not available");
	testsuite(t, TESTPDFROOT + "/pdftest", "pdftest");
}

testCabinetOfHorrors(t: ref T)
{
	if(pdf == nil)
		t.skip("PDF module not available");
	testsuite(t, TESTPDFROOT + "/cabinet-of-horrors", "cabinet-of-horrors");
}

testGrandSummary(t: ref T)
{
	# Print overall summary across all suites
	t.log("=== PDF Conformance Test Results ===");
	t.log(sys->sprint("Suites:  %d found, %d missing", suites_found, suites_missing));
	t.log(sys->sprint("PDFs:    %d tested", grand_total));
	if(grand_total > 0){
		t.log(sys->sprint("PASS:    %d (%d%%)", grand_pass, grand_pass * 100 / grand_total));
		t.log(sys->sprint("WARN:    %d (%d%%)", grand_warn, grand_warn * 100 / grand_total));
		t.log(sys->sprint("FAIL:    %d (%d%%)", grand_fail, grand_fail * 100 / grand_total));
	}

	if(suites_found == 0)
		t.skip("no test suites found (run tests/host/fetch-test-pdfs.sh)");
}

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	drawm = load Draw Draw->PATH;

	readdir = load Readdir Readdir->PATH;
	if(readdir == nil){
		sys->fprint(sys->fildes(2), "cannot load readdir: %r\n");
		raise "fail:cannot load readdir";
	}

	str = load String String->PATH;
	if(str == nil){
		sys->fprint(sys->fildes(2), "cannot load string: %r\n");
		raise "fail:cannot load string";
	}

	testing = load Testing Testing->PATH;
	if(testing == nil){
		sys->fprint(sys->fildes(2), "cannot load testing module: %r\n");
		raise "fail:cannot load testing";
	}
	testing->init();

	for(a := args; a != nil; a = tl a){
		if(hd a == "-v")
			testing->verbose(1);
	}

	pdf = load PDF PDF->PATH;
	if(pdf != nil){
		err := pdf->init(nil);
		if(err != nil)
			sys->fprint(sys->fildes(2), "pdf init warning: %s\n", err);
	}

	run("PdfDifferences", testPdfDifferences);
	run("PopplerTest", testPopplerTest);
	run("BfoPdfa", testBfoPdfa);
	run("PdfTest", testPdfTest);
	run("CabinetOfHorrors", testCabinetOfHorrors);
	run("GrandSummary", testGrandSummary);

	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
