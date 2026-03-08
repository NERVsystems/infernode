# Veltro Agent Tools — Production Readiness Review

Date: 2026-03-08

## Executive Summary

33 tool modules exist in `appl/veltro/tools/`. Of these, **7 are production-ready** (core file/code tools), **8 are mature but need hardening**, **10 are functional but undertested**, and **8 are stubs, specialty, or hardware-dependent**. The most critical gap is that **22 of 33 tools (67%) have zero test coverage**. The security infrastructure (NsConstruct) is solid but no integration test actually runs a tool inside a restricted namespace.

---

## Tier 1: Production-Ready (7 tools)

These are well-implemented, well-structured, and have at least basic test coverage.

| Tool | Lines | Tests | Assessment |
|------|-------|-------|------------|
| **read** | 145 | exec, error, security | Clean. Bufio-based line reading, offset/limit, line numbers. No issues found. |
| **list** | 108 | exec, error, security | Clean. Simple directory listing. |
| **spawn** | 763 | exec, parse, validation | Excellent security model. FORKNS, NEWPGRP, NODEVS, per-child SubAgent instances. Path traversal blocked in `loadagentprompt()`. |
| **diff** | 363 | exec (basic) | Solid file comparison. |
| **json** | 603 | exec, nested keys, arrays | Feature-complete JSON parser with query syntax. |
| **memory** | 420 | exec (save/load/list/delete/append) | Clean key validation (alphanumeric + underscore, max 64 chars), 8r600 permissions. |
| **todo** | 487 | exec (add/list/done/delete/clear/status) | Well tested including edge cases. |

### Action items for Tier 1:
- **read**: Add test for offset/limit edge cases (offset beyond EOF).
- **spawn**: Add integration test: spawn child, verify it cannot access paths not in `caps.paths`.
- **memory**: `getagentid()` always returns `"default"` — all agents share one memory namespace. Either implement proper agent ID detection or document this as intentional.

---

## Tier 2: Mature, Needs Hardening (8 tools)

Good implementations but missing test coverage or have specific issues.

### exec (383 lines) — SECURITY PRIORITY

**Bugs/Issues:**
- `sanitizecmd()` strips backticks and `${}/$()` but semicolons pass through. This is documented as intentional, but an LLM-injected `; rm -rf /` bypasses sanitization. The namespace restriction is the real boundary, but defense-in-depth should consider command chaining.
- `convertquotes()` at line 320 uses `result[len result] = '\'';` to start a single-quoted string — relies on Limbo string auto-extension. Correct but unusual.
- No output size tracking before the timeout loop; large outputs could consume memory before `MAX_OUTPUT` check on line 228.

**Security:** Relies on namespace restriction as primary boundary (correct design). Shell module load is deferred, so agents without `exec` in `caps.tools` correctly can't use it.

**Missing tests:** No exec test within restricted namespace. No test of `sanitizecmd()` with injection patterns.

### edit (345 lines)

**Issues:**
- `readfile()` at line 319: if `d.length == 0` (synthetic/9P file), reads incrementally. But if file is legitimately empty (length 0), returns `nil` and `exec()` reports "cannot read" instead of "file is empty". Minor UX bug.
- No backup/undo mechanism. A bad edit is destructive.
- Ambiguity check (`count > 1 && !all`) is good safety.

**Missing tests:** Zero exec tests. The `edit_test.b` tests the shell `edit` command, not this Veltro tool.

### write (225 lines)

**Issues:**
- `ensuredir()` recursively creates parent directories. No depth limit — a malicious path like `/tmp/a/b/c/.../z` (thousands deep) could recurse excessively.
- Creates files with `8r644` permissions — appropriate for temp files but may be too open for sensitive content.
- No file size limit. Agent could write arbitrarily large files.

**Missing tests:** Zero exec tests. Namespace restriction test verifies `/tmp` is writable but never calls `write.exec()`.

### grep (455 lines)

**Solid implementation.** Regex via Plan 9 ERE, case-insensitive flag, recursive with depth/match limits, timeout-protected directory opens to skip blocked 9P paths.

**Issues:**
- `istext()` at line 355: hardcoded extension list misses `.json`, `.yaml`, `.toml`, `.xml`, `.html`, `.css`. Falls back to null-byte peek, but the extra stat+read adds overhead.

**Missing tests:** Zero tests despite being a core code navigation tool.

### find (292 lines)

**Clean implementation.** Glob matching, depth/result/directory limits, timeout-guarded opens, accepts both native and Unix-style `-name` syntax.

**Issues:**
- `MAX_DIRS: con 2000` with `OPEN_TIMEOUT: con 500` means worst case ~16 minutes of wall time if every directory takes 500ms. In practice unlikely but worth noting.

**Missing tests:** Load-only test (name/doc), no exec tests.

### search (410 lines)

**Overlap with grep.** Both do recursive regex search. `search` has 20-result limit (vs grep's 200), and lacks `-l` and `-i` flags.

**Issues:**
- `istext()` uses a different hardcoded extension list than grep's `istext()` — inconsistency. Both should share a common implementation or at least identical lists.
- Should consider deprecating `search` in favor of `grep` which is strictly more capable.

**Missing tests:** Load-only test (name/doc), no exec tests.

### safeexec (144 lines)

**Good security design.** Validates tool name has no path components (rejects `/`, `\`, `.`), loads only from `/dis/veltro/tools/`, prevents shell injection entirely.

**Issues:**
- Tool module interface declaration at line 31 is missing `init: fn(): string;` — the module header omits `init` but the body defines it. This may cause a compile warning or be silently ignored by the Limbo compiler depending on version.
- No test coverage at all.

### http (276 lines)

**Strong SSRF protection.** Blocks localhost, RFC 1918 ranges, link-local, cloud metadata endpoints, decimal/hex IP addresses, IPv6 ULA/link-local.

**Issues:**
- `isblocked()` at line 224: blocks anything starting with `::` — this catches `::1` but also catches legitimate IPv6 addresses that are shortened. May be overly aggressive but errs on the safe side.
- `extracthost()` strips userinfo AFTER stripping port. If URL is `http://user:pass@evil.com:8080/`, the order is: strip path → `user:pass@evil.com:8080`, strip port → `user:pass@evil.com` (wrong — `user` becomes the host after `@` strip is too late). **Bug:** userinfo should be stripped before port.
- `Content-Type` is hardcoded to `application/json` for all POST/PUT requests. Should be configurable.

**Missing tests:** Skipped in test header ("requires network"). Should have unit tests for `extracthost()`, `isblocked()`.

---

## Tier 3: Functional, Undertested (10 tools)

### git (1224 lines) — largest tool

**Comprehensive implementation.** Full read (status, log, show, branch, tag, cat) and write (add, commit, push, fetch, checkout, merge, rm) operations via git/fs mount + pre-restriction worker thread.

**Issues:**
- Worker thread pattern is correct but has no shutdown mechanism. If the parent agent exits, the worker goroutine blocks forever on `<-workcmd`.
- `workcheckout()` validates branch names well (rejects `..`, absolute paths, shell metacharacters).
- `readcfg()` reads only first 8192 bytes of `.git/config` — fine for typical configs.
- `addfile()` silently skips unreadable files (`return (entries, added)` with no error) — could lead to confusing commit results where files seem staged but aren't.
- `workfetch()` creates a temporary packfile and renames it. If the process crashes mid-fetch, the temp file persists. Not a security issue but a cleanup concern.

**Missing tests:** Skipped ("requires git"). Should have unit tests for branch name validation, `relativepath()`, `getidentity()`.

### websearch (373 lines)

**Clean implementation.** Brave Search API, proper URL encoding, HTML tag stripping from descriptions.

**Issues:**
- API key stored in plaintext at `/lib/veltro/keys/brave`. Should verify this path isn't exposed via namespace restriction.
- `urlencode()` treats characters as single bytes but Limbo strings are Unicode — characters > 127 need UTF-8 multi-byte encoding, not single `hexbyte(c)`. **Bug** for non-ASCII queries.

**Missing tests:** Zero.

### mail (565 lines)

**Full IMAP client.** Config, check, list, read, search, flag, send, folders.

**Issues:**
- Module-level mutable state (`imapserver`, `imapconnected`, `currentmbox`) — breaks the "stateless tool" design from `tool.m`. If two agents share this tool module, they share IMAP state. **Architecture concern** for concurrent use.
- `dosend()` at line 454: derives SMTP server from IMAP server name (e.g., `imap.gmail.com` → tries `imap.gmail.com` as SMTP). This is wrong for most providers. Should be configurable or derive `smtp.gmail.com`.
- `dosend()` builds email with no MIME headers (missing `MIME-Version`, `Content-Type`). May cause encoding issues.

**Missing tests:** Zero.

### vision (621 lines)

**Dual-backend (local GPU + Anthropic API).** Good architecture.

**Issues:**
- `MODEL: con "claude-sonnet-4-20250514"` — hardcoded model version. Should be configurable.
- `readfile()` at line 462: reads only first 8192 bytes. For `sessdir + "/output"`, GPU inference results could exceed this. Should use chunked reads (like `readbytes()` does).
- `errmsg()` reads `/dev/sysctl` for error messages — this is not the standard Inferno error mechanism (`%r` format in `sprint`). May return wrong data.

**Missing tests:** Zero.

### browse (347 lines)

**Web page viewer in Xenith.** Fetches URL, HTML-to-text formatting, creates Xenith window.

**Issues:**
- No SSRF protection like `http.b` has. **Security gap** — should share/call the same `isblocked()` function.
- Depends on `htmlfmt` module at `/dis/xenith/render/htmlfmt.dis` — hard dependency on Xenith being installed.
- Module-level mutable state (`owned` list of windows) — concurrency concern.

**Missing tests:** Zero.

### launch (309 lines)

**Clean implementation.** App name normalization, path traversal rejection, Tk-app blacklist, whitelist for apps outside `/dis/wm/`.

**Issues:**
- `extraapp()` and `listapps()` both maintain parallel whitelist arrays — must stay in sync manually. Comment notes this but it's fragile.

**Missing tests:** Zero.

### xenith (535 lines), luciedit (295 lines), lucishell (218 lines)

GUI integration tools. These interact with Xenith windows and Lucifer shell via 9P filesystem. Functional but entirely untested.

### charon (299 lines)

Web browser tool wrapping the Charon browser. Untested.

---

## Tier 4: Stubs & Specialty (8 tools)

| Tool | Lines | Status |
|------|-------|--------|
| **mount** | 48 | Intentional stub — correctly rejects all calls. Ready. |
| **ask** | 235 | User interaction tool. Requires console — can't be unit tested easily. |
| **say** | 138 | Speech output via speech9p. Hardware-dependent. |
| **hear** | 134 | Speech input via speech9p. Hardware-dependent. |
| **gpu** | 259 | GPU inference. Hardware-dependent. Code duplicates vision.b's `runinfer()`. |
| **fractal** | 187 | Mandelbrot viewer control. Requires mand app running. |
| **gap** | 270 | Context zone gap management. Requires Lucifer UI. |
| **present** | 537 | Presentation zone management. Requires Lucifer UI. |

### Action items for Tier 4:
- **gpu** and **vision**: Share identical `runinfer()` code. Extract to a shared module.
- **ask**: Document that it's untestable in automated environments.

---

## Cross-Cutting Issues

### 1. Code Duplication

Many tools reimplement the same helper functions:

| Function | Duplicated in |
|----------|--------------|
| `strip()` | 18+ tools |
| `splitfirst()` | 12+ tools |
| `hasprefix()` | 10+ tools |
| `readfile()` | 10+ tools |
| `readbytes()` | gpu, vision |
| `writefile()` | gpu, vision, fractal, gap |
| `istext()` | search, grep (different implementations!) |
| `runinfer()` | gpu, vision (identical code) |

**Recommendation:** Extract common helpers into a shared `toolutil.m` module. This reduces maintenance burden, ensures consistency (e.g., `istext()` divergence), and reduces compiled `.dis` sizes.

### 2. Security: No Integration Tests

The security test suite (`veltro_security_test.b`) verifies that `restrictns()` correctly hides paths via stat checks. But **no test actually runs a tool's `exec()` inside a restricted namespace** to confirm the tool works correctly under restriction. For example:
- Does `read.exec("/etc/passwd")` return "error: cannot open" after restriction?
- Does `exec.exec("ls /")` only show restricted paths?
- Can `write.exec()` create files outside `/tmp`?

These integration tests are critical for a production security boundary.

### 3. Cowfs (Copy-on-Write Filesystem) — Completely Untested

`module/cowfs.m` defines `start()`, `diff()`, `modcount()`, `promote()`, `revert()` — critical safety mechanisms for agent writes. Zero test coverage. If cowfs has bugs, agents could corrupt the filesystem without rollback capability.

### 4. Tool Module Statefulness

The `tool.m` interface describes tools as stateless (`"each execution receives fresh arguments"`), but several tools have module-level mutable state:
- **mail**: `imapserver`, `imapconnected`, `currentmbox`
- **browse**: `owned` window list
- **git**: `gitavail`, `workeravail`, `workcmd` channel, `preloadedtools`
- **spawn**: `preloadedtools` (global list)

When multiple agents share a loaded tool module, they share this state. The spawn tool addresses this by loading separate SubAgent instances, but tool modules themselves are shared. This could cause data races if two agents concurrently use `git` or `mail`.

### 5. SSRF Inconsistency

`http.b` has comprehensive SSRF protection (`isblocked()`). `browse.b` and `websearch.b` do not. All tools that make outbound HTTP requests should share the same SSRF protection.

### 6. extracthost() Bug in http.b

The URL parsing in `extracthost()` strips userinfo after stripping the port, which can produce incorrect hostnames for URLs with `user:pass@` in them. Fix: strip userinfo first, then port.

---

## Priority Action Plan

### P0 — Must Fix Before Release

1. **Fix `http.b` `extracthost()` bug** — userinfo must be stripped before port.
2. **Add SSRF protection to `browse.b`** — either import `http.b`'s `isblocked()` or extract to shared module.
3. **Fix `websearch.b` `urlencode()` for Unicode** — non-ASCII characters produce wrong percent-encoding.
4. **Integration security tests** — write tests that call tool `exec()` inside a restricted namespace.

### P1 — Should Fix

5. **Write exec tests for edit, write, grep, find, search** — these are core tools with zero exec coverage.
6. **Add cowfs tests** — test promote/revert/diff workflow.
7. **Fix `mail.b` mutable state** — document or fix concurrent usage.
8. **Extract common helpers** into `toolutil.m`.
9. **Address `memory.b` agent ID** — `getagentid()` hardcodes "default".

### P2 — Nice to Have

10. **Consolidate search/grep** — deprecate `search` or clearly differentiate.
11. **Add unit tests for `git.b`** branch validation, identity, path helpers.
12. **Extract shared GPU inference** from `gpu.b` and `vision.b`.
13. **Add websearch/http unit tests** for URL parsing and blocklist functions.
14. **Add size limits to `write.b`** — prevent agent from writing arbitrarily large files.

---

## Test Coverage Summary

| Category | Tools | Count |
|----------|-------|-------|
| Has exec tests | read, list, spawn, diff, json, memory, todo | 7 (21%) |
| Load-only tests | find, search, write, edit, exec | 5 (15%) |
| Zero tests | ask, grep, hear, say, http, git, gpu, browse, fractal, gap, vision, present, mount, mail, websearch, xenith, charon, luciedit, lucishell, launch, safeexec | 22 (67%) |

**Target for release:** Get all Tier 1 and Tier 2 tools to exec-test coverage (15 tools). Fix P0 bugs.
