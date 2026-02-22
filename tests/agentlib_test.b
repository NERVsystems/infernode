implement AgentlibTest;

#
# agentlib_test.b - Tests for agentlib response parsing
#
# Covers parseaction() and stripaction() including the DONE+text case
# fixed in repl.b (text before DONE must be displayed, not discarded).
#
# To run: emu -r. /tests/agentlib_test.dis [-v]
#
# Note: tests that depend on tool discovery (/tool/tools) are skipped
# when tools9p is not mounted.
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "testing.m";
	testing: Testing;
	T: import testing;

include "../appl/veltro/agentlib.m";
	agentlib: AgentLib;

AgentlibTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

SRCFILE: con "/tests/agentlib_test.b";

passed := 0;
failed := 0;
skipped := 0;

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
	* =>
		t.failed = 1;
	}

	if(testing->done(t))
		passed++;
	else if(t.skipped)
		skipped++;
	else
		failed++;
}

# Check if tools9p is available (toollist non-empty)
hastoolfs(): int
{
	return agentlib->pathexists("/tool/tools");
}

# ---- parseaction tests (DONE handling — no toollist needed) ----

# Bare DONE returns ("DONE", "")
testParseDone(t: ref T)
{
	(tool, args) := agentlib->parseaction("DONE");
	t.assertseq(tool, "DONE", "bare DONE returns DONE tool");
	t.assertseq(args, "", "bare DONE has empty args");
}

# Lowercase done is recognized
testParseDoneLower(t: ref T)
{
	(tool, args) := agentlib->parseaction("done");
	t.assertseq(tool, "DONE", "lowercase done recognized");
	t.assertseq(args, "", "lowercase done has empty args");
}

# DONE with markdown noise is recognized
testParseDoneMarkdown(t: ref T)
{
	(tool, args) := agentlib->parseaction("**DONE**");
	t.assertseq(tool, "DONE", "markdown DONE recognized");
	t.assertseq(args, "", "markdown DONE has empty args");
}

# Text before DONE — the bug case fixed in repl.b.
# parseaction must return DONE (not lose it due to text on prior lines).
testParseDoneWithText(t: ref T)
{
	resp := "[Veltro]Hello! I'm Veltro, running on Inferno OS. What can I help you with?\n\nDONE";
	(tool, args) := agentlib->parseaction(resp);
	t.assertseq(tool, "DONE", "text+DONE: DONE recognized");
	t.assertseq(args, "", "text+DONE: empty args");
}

# Empty response returns ("", "")
testParseEmpty(t: ref T)
{
	(tool, args) := agentlib->parseaction("");
	t.assertseq(tool, "", "empty response: empty tool");
	t.assertseq(args, "", "empty response: empty args");
}

# ---- parseaction tests (tool parsing — requires tools9p) ----

# say tool is parsed correctly
testParseSay(t: ref T)
{
	if(!hastoolfs()) {
		t.skip("/tool not mounted — skipping say parse test");
		return;
	}
	(tool, args) := agentlib->parseaction("say Hello there!");
	t.assertseq(tool, "say", "say tool parsed");
	t.assertseq(args, "Hello there!", "say args captured");
}

# Tool invocation: read
testParseReadTool(t: ref T)
{
	if(!hastoolfs()) {
		t.skip("/tool not mounted — skipping read parse test");
		return;
	}
	(tool, args) := agentlib->parseaction("read /appl/veltro/repl.b");
	t.assertseq(tool, "read", "read tool parsed");
	t.assertseq(args, "/appl/veltro/repl.b", "read args captured");
}

# say before DONE: say is picked, not DONE (say comes first)
testParseSayBeforeDone(t: ref T)
{
	if(!hastoolfs()) {
		t.skip("/tool not mounted — skipping say-before-DONE test");
		return;
	}
	resp := "say Hello world!\nDONE";
	(tool, args) := agentlib->parseaction(resp);
	t.assertseq(tool, "say", "say before DONE: say picked");
	t.assert(agentlib->contains(args, "Hello world"), "say args captured");
}

# say with heredoc multi-line content
testParseSayHeredoc(t: ref T)
{
	if(!hastoolfs()) {
		t.skip("/tool not mounted — skipping say heredoc test");
		return;
	}
	resp := "say <<EOF\nLine one.\nLine two.\nEOF\n";
	(tool, args) := agentlib->parseaction(resp);
	t.assertseq(tool, "say", "say with heredoc parsed");
	t.assert(agentlib->contains(args, "Line one"), "heredoc line one in args");
	t.assert(agentlib->contains(args, "Line two"), "heredoc line two in args");
}

# ---- parseactions tests ----

listlen(l: list of (string, string)): int
{
	n := 0;
	for(; l != nil; l = tl l)
		n++;
	return n;
}

# Empty response → nil
testParseActionsEmpty(t: ref T)
{
	actions := agentlib->parseactions("");
	t.assert(actions == nil, "empty response → nil");
}

# Bare DONE → ("DONE", "") :: nil
testParseActionsDone(t: ref T)
{
	actions := agentlib->parseactions("DONE");
	t.assert(actions != nil, "DONE → non-nil list");
	t.asserteq(listlen(actions), 1, "DONE list has 1 element");
	(tool, args) := hd actions;
	t.assertseq(tool, "DONE", "DONE tool recognized");
	t.assertseq(args, "", "DONE has empty args");
}

# Preamble text then DONE → ("DONE", "") :: nil
testParseActionsTextThenDone(t: ref T)
{
	resp := "Here is my answer.\nDONE";
	actions := agentlib->parseactions(resp);
	t.assert(actions != nil, "text+DONE → non-nil list");
	(tool, nil) := hd actions;
	t.assertseq(tool, "DONE", "text+DONE: DONE recognized");
}

# Two consecutive tool lines (requires /tool)
testParseActionsMultiple(t: ref T)
{
	if(!hastoolfs()) {
		t.skip("/tool not mounted — skipping multi-tool test");
		return;
	}
	resp := "read /file1\nread /file2\n";
	actions := agentlib->parseactions(resp);
	t.assert(actions != nil, "two tools: non-nil list");
	t.asserteq(listlen(actions), 2, "two tools: list has 2 elements");
	(t1, a1) := hd actions;
	(t2, a2) := hd tl actions;
	t.assertseq(t1, "read", "first tool name");
	t.assertseq(a1, "/file1", "first tool args");
	t.assertseq(t2, "read", "second tool name");
	t.assertseq(a2, "/file2", "second tool args");
}

# Tool then DONE → only the tool (DONE stops, not added when prior tools found)
testParseActionsToolThenDone(t: ref T)
{
	if(!hastoolfs()) {
		t.skip("/tool not mounted — skipping tool-then-DONE test");
		return;
	}
	resp := "read /file1\nDONE\n";
	actions := agentlib->parseactions(resp);
	t.assert(actions != nil, "tool+DONE: non-nil list");
	t.asserteq(listlen(actions), 1, "tool+DONE: exactly 1 tool (no DONE entry)");
	(t1, a1) := hd actions;
	t.assertseq(t1, "read", "tool name correct");
	t.assertseq(a1, "/file1", "tool args correct");
}

# ---- stripaction tests (pure string — no toollist needed) ----

# Strips [Veltro] prefix and DONE line, returns text
testStripBasic(t: ref T)
{
	resp := "[Veltro]Hello! Good to have you here.\n\nDONE";
	result := agentlib->stripaction(resp);
	t.assertseq(result, "Hello! Good to have you here.", "strips prefix and DONE");
}

# Pure DONE returns empty string
testStripDoneOnly(t: ref T)
{
	result := agentlib->stripaction("DONE");
	t.assertseq(result, "", "pure DONE → empty");
}

# Multi-line text before DONE: all lines preserved, DONE removed
testStripMultiLine(t: ref T)
{
	resp := "Line one.\nLine two.\nDONE";
	result := agentlib->stripaction(resp);
	t.assert(agentlib->contains(result, "Line one"), "first line preserved");
	t.assert(agentlib->contains(result, "Line two"), "second line preserved");
	t.assert(!agentlib->contains(result, "DONE"), "DONE removed");
}

# [Veltro] prefix stripped, text preserved
testStripVeltroPrefix(t: ref T)
{
	resp := "[Veltro]Hello! I'm Veltro.\n\nDONE";
	result := agentlib->stripaction(resp);
	t.assert(!agentlib->hasprefix(result, "[Veltro]"), "Veltro prefix stripped");
	t.assert(agentlib->contains(result, "Hello"), "text content preserved");
}

# Empty input returns empty output
testStripEmpty(t: ref T)
{
	result := agentlib->stripaction("");
	t.assertseq(result, "", "empty input → empty output");
}

# Only blank lines + DONE returns empty
testStripBlankPlusDone(t: ref T)
{
	result := agentlib->stripaction("\n\n\nDONE\n");
	t.assertseq(result, "", "blank lines + DONE → empty");
}

# Regression: the exact pattern from the hello-Veltro bug
testStripHelloVeltroBugPattern(t: ref T)
{
	# This is the exact response pattern that triggered the bug where
	# repl.b discarded text before DONE
	resp := "[Veltro]Hello! I'm Veltro, running on Inferno OS. What can I help you with?\n\nDONE";
	result := agentlib->stripaction(resp);
	t.assert(len result > 0, "regression: greeting text not empty");
	t.assert(agentlib->contains(result, "Hello"), "regression: greeting preserved");
	t.assert(!agentlib->contains(result, "DONE"), "regression: DONE stripped");
	t.assert(!agentlib->contains(result, "[Veltro]"), "regression: prefix stripped");
}

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	testing = load Testing Testing->PATH;

	if(testing == nil) {
		sys->fprint(sys->fildes(2), "cannot load testing module: %r\n");
		raise "fail:cannot load testing";
	}

	agentlib = load AgentLib AgentLib->PATH;
	if(agentlib == nil) {
		sys->fprint(sys->fildes(2), "cannot load agentlib: %r\n");
		raise "fail:cannot load agentlib";
	}
	agentlib->init();

	testing->init();

	for(a := args; a != nil; a = tl a) {
		if(hd a == "-v")
			testing->verbose(1);
	}

	# parseaction — DONE handling (no /tool needed)
	run("ParseDone", testParseDone);
	run("ParseDoneLower", testParseDoneLower);
	run("ParseDoneMarkdown", testParseDoneMarkdown);
	run("ParseDoneWithText", testParseDoneWithText);
	run("ParseEmpty", testParseEmpty);

	# parseaction — tool parsing (skipped without /tool)
	run("ParseSay", testParseSay);
	run("ParseReadTool", testParseReadTool);
	run("ParseSayBeforeDone", testParseSayBeforeDone);
	run("ParseSayHeredoc", testParseSayHeredoc);

	# parseactions — DONE handling (no /tool needed)
	run("ParseActionsEmpty", testParseActionsEmpty);
	run("ParseActionsDone", testParseActionsDone);
	run("ParseActionsTextThenDone", testParseActionsTextThenDone);

	# parseactions — multi-tool (skipped without /tool)
	run("ParseActionsMultiple", testParseActionsMultiple);
	run("ParseActionsToolThenDone", testParseActionsToolThenDone);

	# stripaction — pure string (no /tool needed)
	run("StripBasic", testStripBasic);
	run("StripDoneOnly", testStripDoneOnly);
	run("StripMultiLine", testStripMultiLine);
	run("StripVeltroPrefix", testStripVeltroPrefix);
	run("StripEmpty", testStripEmpty);
	run("StripBlankPlusDone", testStripBlankPlusDone);
	run("StripHelloVeltroBugPattern", testStripHelloVeltroBugPattern);

	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
