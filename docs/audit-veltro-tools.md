# Veltro Tools Audit — Pre-Production Review

**Date:** 2026-03-22
**Scope:** All 39 tool implementations in `appl/veltro/tools/`, 30 definition
files in `lib/veltro/tools/`, and the `tools9p.b` registry.

---

## Executive Summary

The Veltro tool system is architecturally sound — a clean `tool.m` interface
(init/name/doc/exec), 9P-based dispatch via `tools9p`, namespace isolation per
invocation, and eager pre-loading before restriction. However, the audit
identifies **9 tools missing definition files**, **3 tools missing from the
registry**, **inconsistent documentation patterns** across the 30 existing
definitions, and **significant test coverage gaps** (only 12 of 39 tools have
any test coverage).

---

## 1. Registration and Definition Gaps

### 1.1 Tools Missing Definition Files (9)

These tools have `.b` implementations but no corresponding `.txt` in
`lib/veltro/tools/`. The LLM agent relies on definition files for tool
discovery and usage guidance — without them, the agent gets only the
`doc()` output from the module, which is typically terser.

| Tool | Status | Priority |
|------|--------|----------|
| **browse** | Registered in TOOL_PATHS. Has inline `doc()`. | Medium |
| **charon** | Registered. Full `doc()` with 8 commands. | **High** — feature-rich, needs user-facing docs |
| **gpu** | Registered. `doc()` covers 5 subcommands. | Medium — hardware-dependent |
| **http** | **NOT registered** in TOOL_PATHS. Listed in usage string. | **High** — either register or remove from usage |
| **keyring** | Registered. `doc()` covers 3 subcommands. | Medium |
| **payfetch** | Registered. `doc()` covers usage and flags. | Medium |
| **present** | Registered. `doc()` covers 9 commands + 7 types. | **High** — heavily used for artifacts |
| **safeexec** | **NOT registered** in TOOL_PATHS. | Low — internal/security tool |
| **wallet** | Registered. `doc()` covers 7 commands. | Medium |

### 1.2 Tools Missing from TOOL_PATHS (3)

| Tool | Has .txt | Has .b | Notes |
|------|----------|--------|-------|
| **http** | No | Yes | Referenced in usage() help text but not loadable. Bug or intentional? |
| **safeexec** | No | Yes | Likely intentional — internal security shim, not agent-facing. |
| **mount** | Yes | Yes | Implementation is a stub returning error. Registered nowhere. The `.txt` describes mounting but the tool always fails. Confusing for the LLM. |

**Recommendation:** Either add `http` and `mount` to TOOL_PATHS or remove
their misleading references (usage string for http, definition file for mount).
Document `safeexec` as intentionally internal.

---

## 2. Documentation Consistency

### 2.1 Title Format Inconsistencies

Three different title formats are used across the 30 definition files:

| Format | Count | Files |
|--------|-------|-------|
| `name - description` | 27 | Most tools |
| `name — description` (em-dash) | 1 | launch |
| `== name — description ==` (wiki-style) | 2 | webfetch, websearch |

**Recommendation:** Standardize on `name - description` (hyphen) across all files.

### 2.2 Section Presence Matrix

Not all tools document the same sections. Key gaps:

| Section | Present In | Missing From |
|---------|-----------|--------------|
| **Usage/Syntax** | All 30 | — |
| **Examples** | 17 of 30 | diff, edit, find, gap, json (has inline), list, memory, plan, present/xenith (implicit), read, search, webfetch, websearch, write |
| **Return format** | 12 of 30 | edit, exec, editor, fractal, gap, hear, launch, man, memory, mount, plan, say, shell (partial), spawn (partial), task, todo, xenith |
| **Error handling** | 5 of 30 | Most tools don't describe error format |
| **Constraints/limits** | 11 of 30 | Most tools don't state max results, timeouts, or size limits |
| **Prerequisites** | 6 of 30 | Only hear, say, fractal, man, shell, editor note requirements |

### 2.3 Recommended Standard Template

Every definition file should include:

```
name - One-line description

Usage:
  name <args>              # Primary usage
  name <args> [options]    # Variant

Arguments:
  arg1    Description
  arg2    Description (default: X, max: Y)

Returns:
  Description of output format.
  Errors prefixed with "error: ".

Examples:
  name foo              # What this does
  name bar baz          # What this does

Notes:
  - Constraints (max results, timeouts, size limits)
  - Prerequisites (requires X running/mounted)
```

### 2.4 Specific Documentation Issues

1. **mount.txt** describes functional mounting but `mount.b` always returns an
   error. The definition file should say "stub — user operation only" upfront,
   or be removed entirely.

2. **grep.txt** doesn't mention max result limits (grep.b likely has one).

3. **git.txt** doesn't mention output size limits for `log` or `show`.

4. **exec.txt** mentions 30s max timeout but doesn't show timeout syntax.

5. **editor.txt** documents a 500ms file-polling mechanism that's an
   implementation detail the LLM doesn't need — better to focus on the
   command interface.

6. **webfetch.txt** and **websearch.txt** use a "workflow guidance" style
   (coaching the LLM on research methodology) rather than documenting the tool
   interface. This is useful but should be separated from the tool reference.

---

## 3. Test Coverage

### 3.1 Current State

| Coverage | Tools | Count |
|----------|-------|-------|
| **Tested** | diff, json, memory, todo, read, list, find, search, write, edit, exec, spawn | 12 |
| **Untested** | browse, charon, editor, fractal, gap, git, gpu, grep, hear, http, keyring, launch, mail, man, mount, payfetch, plan, present, safeexec, say, shell, task, vision, wallet, webfetch, websearch, xenith | 27 |

**Only 31% of tools have any test coverage.**

### 3.2 Priority Test Gaps

**High priority** (core tools, testable without hardware/network):

| Tool | Why |
|------|-----|
| **grep** | Core search tool, no tests at all |
| **git** | Complex dual-backend (read via git/fs, write via worker), no tests |
| **plan** | Complex state machine (create→goal→approach→step→approve→progress→complete), no tests |
| **editor** | 11 commands with file-based state machine, no tests |
| **xenith** | 8+ commands managing windows, no tests |
| **present** | 9 commands, 7 artifact types, no tests |
| **gap** | User-visible knowledge tracking, no tests |
| **task** | Most complex parameter interface, spawns subagents, no tests |

**Medium priority** (require mocking or controlled environment):

| Tool | Reason |
|------|--------|
| **mail** | 8 subcommands, requires IMAP mock |
| **webfetch/websearch** | Network-dependent but could test URL parsing, SSRF protection |
| **http** | SSRF protection logic is critical and testable |
| **launch** | App registry validation testable |
| **shell** | Read-only access verification testable |

**Lower priority** (hardware/service dependent):

| Tool | Reason |
|------|--------|
| gpu, vision, hear, say | Require GPU/speech hardware |
| charon, browse | Require Charon browser running |
| wallet, keyring, payfetch | Require factotum/wallet services |
| fractal, man | Require GUI apps running |

### 3.3 Existing Test Quality

- `veltro_tools_test.b`: Tests diff, json, memory, todo — good functional
  coverage but lacks edge cases (empty input, large data, special characters).
- `veltro_test.b`: Tests read, list, find, search, write, edit, exec, spawn
  but many tests only verify `load` + `doc()` — not actual `exec()` behavior.
- `veltro_security_test.b`: Strong coverage of namespace restriction but
  doesn't test tool-specific security (SSRF in http, path traversal in
  safeexec, injection in exec).
- `veltro_concurrent_test.b`: Tests concurrent namespace ops but not concurrent
  tool execution.

---

## 4. Architectural Concerns

### 4.1 http Tool Status

The `http` tool is implemented, referenced in the `tools9p` usage string, but
**not registered** in TOOL_PATHS. This means:
- It cannot be loaded by the agent
- The usage help text is misleading
- Its SSRF protection code is unreachable

This is either a bug (forgot to register) or the tool was intentionally
replaced by `webfetch` but not cleaned up. Either way, it needs resolution.

### 4.2 mount Tool Contradiction

`mount.txt` describes mounting as if it works. `mount.b` always returns an
error. The LLM will read the definition, attempt to use mount, and get a
confusing error. The definition file should either be removed or rewritten to
explain that mounting is user-initiated.

### 4.3 safeexec Visibility

`safeexec` is not in TOOL_PATHS, which is correct for an internal tool. But
it's unclear how/when it's invoked. If it's used by `tools9p` internally, this
should be documented in a comment. If it's dead code, it should be removed.

### 4.4 Tool Limit Inconsistencies

Different tools impose different limits without consistent documentation:

| Tool | Limit | Documented? |
|------|-------|-------------|
| find | 100 results | Yes |
| search | 20 results | Yes |
| grep | 200 matches | No (in .txt) |
| read | 100 default, 1000 max lines | Yes |
| diff | 20 context lines | Yes |
| webfetch | 512KB | Yes |
| websearch | 15 results | Yes |
| exec | 5s default, 30s max | Partial |
| git log | ? | No |
| mail list | ? | No |

---

## 5. Pre-Production Checklist

### Must Fix (P0)

- [ ] **Resolve http tool status**: Register in TOOL_PATHS or remove from
  usage string and consider removing the implementation
- [ ] **Fix mount.txt**: Rewrite to state it's user-only, or remove the
  definition file to avoid LLM confusion
- [ ] **Add definition files for high-use tools**: At minimum `present.txt`
  and `charon.txt` — these are feature-rich tools the agent needs guidance on
- [ ] **Standardize title format**: Convert webfetch.txt, websearch.txt, and
  launch.txt to use `name - description` format

### Should Fix (P1)

- [ ] **Add definition files** for browse, gpu, http (if kept), keyring,
  payfetch, wallet
- [ ] **Add examples** to the 13 definition files that lack them (diff, edit,
  find, gap, list, memory, plan, read, search, webfetch, websearch, write,
  xenith)
- [ ] **Document return formats** consistently — every tool should state what
  success and error responses look like
- [ ] **Document constraints/limits** in every tool that has them — grep max
  matches, git output limits, mail list limits, exec timeout syntax
- [ ] **Add tests for core tools**: grep, git, plan, editor, xenith, present,
  gap, task (all testable without hardware)
- [ ] **Add SSRF/injection tests**: http tool's SSRF filter, exec tool's
  command sanitization, safeexec's path traversal prevention

### Nice to Have (P2)

- [ ] **Separate workflow guidance** from tool reference in webfetch.txt and
  websearch.txt (move to reminders or system prompt)
- [ ] **Add edge-case tests** for existing tested tools (empty input, large
  data, Unicode, concurrent access)
- [ ] **Document safeexec** as internal-only with a comment in tools9p.b
- [ ] **Create integration tests** for tool chaining (websearch → webfetch,
  plan → todo, spawn with multiple tools)
- [ ] **Add prerequisite checking**: Tools that require running services
  (fractal, man, editor, shell) should return actionable errors when the
  service isn't running, not just fail silently

---

## 6. Summary Metrics

| Metric | Value | Target |
|--------|-------|--------|
| Tools implemented | 39 | — |
| Tools registered (TOOL_PATHS) | 36 | 37+ (add http or remove) |
| Tools with definition files | 30 | 37+ (all registered tools) |
| Tools with examples in docs | 17 | 30+ |
| Tools with documented limits | 11 | All tools with limits |
| Tools with test coverage | 12 (31%) | 25+ (65%) |
| Title format consistency | 3 formats | 1 format |
| Registry/usage string alignment | Mismatched (http) | Aligned |
