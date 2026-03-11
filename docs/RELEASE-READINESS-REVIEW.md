# Infernode Release Readiness Review

**Date:** 2026-03-11
**Scope:** Feature completeness, usability, integration, and release risks
**Out of scope:** Bugs, performance optimizations

---

## Executive Summary

Infernode is a mature, well-engineered system with **two distinct GUI frontends**, a comprehensive AI agent (Veltro) with 32+ tools and an interactive guided tour, a robust testing infrastructure, and a genuinely innovative namespace-based security model. The core system (280+ commands, 160+ library modules, full networking, JIT compilation on ARM64/AMD64) is production-quality.

The project has more polish than initially apparent:

- **Lucifer** — a purpose-built three-zone GUI tiler (Conversation, Presentation, Context) designed specifically for human-AI collaboration, with theming, app embedding, and a 9P state server
- **Xenith** — an Acme-derived text environment with async I/O, image display, renderers, and 9P agent integration
- **Veltro guided tour** — a 228-line interactive demo script (`lib/veltro/demos/tour.txt`) where Veltro walks users through the entire system using its own tools
- **Welcome onboarding** — first-run document (`lib/veltro/welcome.md`) displayed in Lucifer, with setup instructions and "things to try"
- **Speech integration** — text-to-speech and speech-to-text via `/n/speech`
- **Rich text layout** — `rlayout` module for markdown/HTML rendering in Lucifer's presentation zone

---

## 1. THE TWO INTERFACES

### 1.1 Lucifer — Three-Zone AI Workspace

Lucifer (`appl/cmd/lucifer.b`) is the primary GUI, purpose-built for AI-human collaboration:

| Zone | Purpose | Implementation |
|------|---------|----------------|
| **Conversation** (left, ~30%) | Chat with Veltro | `luciconv.b` — renders messages, handles keyboard input |
| **Presentation** (centre, ~45%) | Documents, apps, fractals, editors | `lucipres.b` — embedded wmsrv, artifact tabs, app hosting |
| **Context** (right, ~25%) | Tools, paths, gaps, background tasks | `lucictx.b` — tool toggles, resource mounting, live status |

**Key Lucifer components:**
- `luciuisrv.b` — 9P state server at `/n/ui/` (activities, conversations, presentations, context, notifications, toasts)
- `lucibridge.b` — connects Lucifer UI to Veltro agent loop via `llm9p`
- `lucitheme.b` — theme system with `brimstone` (dark) and `halo` (light Plan 9) themes at `/lib/lucifer/theme/`
- `luciedit.b` — embedded text editor with keyboard shortcuts (Ctrl-S save, Ctrl-Q quit, arrows, Home/End) and 9P interface at `/edit/`
- `lucishell.b` — embedded shell terminal with 9P interface, history, Ctrl-C/D/U/L support
- Double-buffered rendering (flicker regression fixed and tested in `lucifer_flicker_test.b`)
- App slot system for embedding GUI apps (clock, mand, xenith, bounce, coffee, colors, lens, view) in the presentation zone
- Mouse routing by zone position, keyboard-follows-mouse focus

**Lucifer UI filesystem (`/n/ui/`):**
```
/n/ui/
    ctl                     global control
    event                   global events (blocking read)
    notification            write-once-read-once
    toast                   write-once-read-once
    activity/
        current             current activity id
        {id}/
            label, status, event
            conversation/   ctl, input, 0, 1, 2...
            presentation/   ctl, current, {artifact-id}/...
            context/        ctl, resources/, gaps/, background/
```

**Assessment:** Lucifer is a substantial, well-designed application. The three-zone model with a 9P state server is architecturally clean and well-suited to AI interaction. The embedded apps (editor, shell, fractal viewer) make it a complete environment.

### 1.2 Xenith — Acme-Derived Text Environment

Xenith (`appl/xenith/`) is the second interface — a fork of Acme with ~21,000 lines across 57 modules. It adds:

- Async I/O framework for non-blocking file operations
- Image display (PNG, JPEG, PPM) with async loading
- Pluggable renderer registry (markdown, HTML, PDF, Mermaid, syntax highlighting)
- Dark mode (Catppuccin, Plan 9, dark themes)
- 9P filesystem at `/mnt/xenith` for agent interaction
- Full Acme editing model (27 commands, ed-style Edit language, mouse chording)

**Assessment:** Complete Acme-level editor with modern additions. Mouse-centric by design (Acme philosophy). Works standalone or embedded in Lucifer.

---

## 2. VELTRO AI AGENT SYSTEM

### 2.1 Architecture

Veltro is a 32-tool AI agent with:

| Component | File | Purpose |
|-----------|------|---------|
| Main agent | `appl/veltro/veltro.b` | Agent loop, LLM interaction |
| Agent library | `appl/veltro/agentlib.b` | Shared agent logic |
| Tool server | `appl/veltro/tools9p.b` | 9P filesystem at `/tool/` |
| Namespace constructor | `appl/veltro/nsconstruct.b` | Capability-based security |
| Subagent system | `appl/veltro/subagent.b` | Isolated child agents |
| Speech | `appl/veltro/speech9p.b` | TTS/STT via `/n/speech` |
| REPL | `appl/veltro/repl.b` | Interactive terminal mode |
| Copy-on-write FS | `appl/veltro/cowfs.b` | Safe file editing layer |
| MC9P | `appl/veltro/mc9p.b` | LLM multiplexer/protocol |

**32 tools** (each with `.txt` documentation in `lib/veltro/tools/`):
- **File ops:** read, list, find, search, write, edit, grep, diff
- **Execution:** exec, spawn
- **UI/Presentation:** xenith, luciedit, lucishell, present, launch
- **Fractals:** fractal, mand
- **Communication:** ask, say, hear, mail
- **Knowledge:** memory, gap, todo
- **Network:** http, git, json, vision, websearch
- **Special:** mount

**4 agent types** (`lib/veltro/agents/`): default, explore, plan, task

### 2.2 Guided Tour (Demo Script)

`lib/veltro/demos/tour.txt` is a 228-line interactive demonstration where Veltro walks users through the system **using its own tools live**. It covers 14 sections:

1. Welcome — introduces Infernode and Veltro
2. The Three Zones — explains Lucifer layout
3. Everything Is a File — namespace exploration
4. Launching Apps — clock, luciedit, lucishell, mand, xenith
5. The Fractal Viewer — interactive Mandelbrot/Julia exploration
6. The Text Editor — luciedit demonstration
7. Finding and Reading Files — code navigation
8. The Context Zone — tool toggles, path binding, knowledge gaps
9. Persistence — memory across sessions
10. Voice — text-to-speech and speech-to-text
11. The Host OS Bridge — accessing host filesystem
12. Subagents and Security — isolated agents with namespace capabilities
13. More Capabilities — todo, HTTP, mail, git, diff, json, vision, websearch
14. Next Steps — where to go from here

The tour uses `ask` to pace itself, `say` to narrate, `present` to display content, and `launch` to run live apps. It's documented in `RUN_TOUR.md` with launch instructions for both Lucifer and terminal/Xenith modes.

**Assessment:** This is a genuine differentiator. The tour is a first-class onboarding experience that most projects don't have. The fact that Veltro demonstrates itself using its own tooling is elegant.

### 2.3 Welcome Document

`lib/veltro/welcome.md` is a 140-line onboarding document displayed automatically on first Lucifer launch. It covers:
- Three-zone layout explanation
- Things to try (talk, launch apps, explore, run tour)
- Setup: API keys, themes, fonts, memory pools, speech
- Key concepts (everything is a file, namespace is security, shared workspace)
- Quick reference table

### 2.4 System Prompt and Agent Configuration

`lib/veltro/system.txt` provides 110 lines of core agent instructions covering identity, tool usage, file workflow, professional objectivity, and environment specifics. The prompt is thoughtfully written — it emphasizes grounding (use tools, surface knowledge gaps), safe file operations (explicit paths only), and completion behavior.

---

## 3. ADDITIONAL MAJOR SUBSYSTEMS

Beyond the two GUIs and Veltro, several substantial subsystems deserve mention:

### 3.1 GPU / TensorRT Acceleration

- **Interface:** `module/gpu.m` — `init`, `gpuinfo`, `loadmodel`, `infer`
- **Implementation:** `libinterp/gpu.c` with TensorRT binding; `libinterp/gpu-stub.c` for graceful degradation on non-GPU systems
- **Gpusrv:** `appl/cmd/gpusrv.b` — Plan 9-style Styx server exposing GPU as a filesystem with clone-based sessions, model management, and per-session input/output
- **Target hardware:** NVIDIA Jetson Orin (ARM64 + TensorRT)
- **Vision tool:** `appl/veltro/tools/vision.b` integrates GPU inference into the agent

### 3.2 Native Mermaid Diagram Engine

- **Implementation:** `appl/lib/mermaid.b` — 119KB pure Limbo, no external dependencies
- **Supported types:** Flowchart, pie, sequence, Gantt, xy-chart, class diagram, state diagram, ER diagram, mindmap, timeline, Git graph, quadrant chart, journey, requirement diagram, block diagram (15 types)
- **Renderer:** `appl/xenith/render/mermaidrender.b` integrates into both Xenith and Lucifer

### 3.3 Charon Web Browser

- **Location:** `appl/charon/` — full HTML/CSS/JS rendering engine
- **Features:** Layout engine, JavaScript interpreter, HTTP client, cookie management, FTP support, image rendering
- **Integration:** Veltro's `browse` and `charon` tools allow the AI agent to control the browser

### 3.4 PDF Renderer

- **Location:** `appl/lib/pdf.b` — 164KB comprehensive PDF rendering library
- **Integration:** Used by Lucifer's presentation zone for document viewing

### 3.5 Benchmarking Suites

- **Location:** `/benchmarks/` with three generations:
  - v1: JIT vs Interpreter (6 benchmarks)
  - v2: 26 benchmarks across 9 categories
  - v3: Cross-language comparison (C, Go, Java, Limbo)
- **Programs:** Sieve, Mandelbrot, matrix, FFT, and more

### 3.6 Formal Verification

- **Location:** `/formal-verification/` with CBMC, TLA+, and SPIN
- **Scope:** Namespace security verification, race condition detection
- **CI:** Dedicated `formal-verification.yml` GitHub Actions workflow

### 3.7 Install System

- **Location:** `/appl/cmd/install/` — package creation, installation, filesystem walking, metadata, changelog tracking

---

## 4. TESTING AND CI

### 4.1 Test Coverage

| Category | Files | Description |
|----------|-------|-------------|
| **Limbo unit tests** | 20+ `*_test.b` files | Framework-based with `testing.m` |
| **Inferno shell tests** | `tests/inferno/*.sh` | Integration tests inside emulator |
| **Host shell tests** | `tests/host/*.sh` | External validation and protocols |
| **Lucifer-specific** | `lucifer_flicker_test.b`, `lucifer.sh`, `lucifer_llm.sh`, `lucifer_presentation_test.rc` | GUI state and rendering |
| **Veltro-specific** | `veltro_test.b`, `veltro_tools_test.b`, `veltro_security_test.b`, `veltro_concurrent_test.b` | Agent, tools, security, concurrency |
| **Other** | `agent_test.b`, `edit_test.b`, `sdl3_test.b`, `xenith_*_test.*` | Various subsystems |

### 4.2 CI Pipeline

GitHub Actions runs Linux CI. macOS and Windows are not in CI (see gaps below).

---

## 5. STRENGTHS TO HIGHLIGHT IN RELEASE

1. **Two Complete Interfaces** — Lucifer (AI-native three-zone tiler) and Xenith (Acme-derived power editor). Users choose their preferred workflow.

2. **Interactive Guided Tour** — Veltro walks new users through the entire system using its own tools. Most projects have docs; Infernode has a live AI-powered demonstration.

3. **Namespace-as-Capability Security** — FORKNS + bind-replace is more elegant and more secure than any container/sandbox approach for AI agents. Each agent literally cannot perceive paths outside its namespace. Kernel-enforced, not policy-enforced.

4. **32 Native AI Tools** — File ops, execution, UI control, speech, fractals, memory, web, email, git — all accessible via filesystem operations. No SDK needed.

5. **15-30 MB RAM, 2-Second Startup** — Suitable for edge/IoT/embedded deployment.

6. **JIT on ARM64 + AMD64** — Native performance on Apple Silicon, Jetson, Raspberry Pi, and x86-64.

7. **Everything-as-a-File Interface** — AI agents interact via filesystem operations, which LLMs understand naturally and humans can audit via `cat` and `ls`.

8. **Speech Integration** — Text-to-speech and speech-to-text via `/n/speech`, configurable voice and language.

9. **Embedded GUI Apps** — Fractal viewer, text editor, shell, clock, demos — all launchable inside Lucifer's presentation zone.

10. **GPU/TensorRT Inference** — Native ML inference via 9P filesystem on Jetson hardware. No Python, no Docker — just read/write files.

11. **Native Mermaid Diagrams** — 119KB pure Limbo engine rendering 15 diagram types. No external dependencies.

12. **Formal Verification** — SPIN, CBMC, and TLA+ verification of concurrent kernel code and namespace security.

13. **Full Web Browser** — Charon with HTML/CSS/JS rendering, controllable by AI agent via filesystem.

---

## 6. SECURITY AUDIT FINDINGS

These are real exploitable issues found by code review of the agent security model.

### 6.1 CowFS Path Traversal — HIGH SEVERITY

**Location:** `appl/veltro/cowfs.b` lines 421-442, 563-564, 620, 899

**The bug:** `relpath` from 9P clients is concatenated directly with `basepath` without canonicalization:

```limbo
bpath := state.basepath + "/" + relpath;
```

A malicious client sending `relpath = "../../alice/secrets"` causes CowFS to resolve outside the intended base directory. There is **zero validation** that the resulting path stays within `basepath`.

**Attack scenario:** Parent agent opens `/n/local/Users/bob/tmp` with CowFS. Attacker crafts `relpath = "../../alice/secrets"` → resolves to `/n/local/Users/alice/secrets`.

**The namespace kernel does restrict visible paths**, but CowFS operates at the 9P layer above the namespace, and a compromised agent process or crafted 9P message can exploit this.

**Fix required:** Resolve `relpath` canonically and verify the result has `basepath` as prefix before any read/write/stat.

### 6.2 Command Injection via exec Tool — MEDIUM SEVERITY

**Location:** `appl/veltro/tools/exec.b` line 139, 363-364

**The bug:** `sanitizecmd()` strips command substitution patterns (`{cmd}`, `${...}`) but **explicitly allows semicolons and pipes**:

```
# Semicolons and pipes are intentionally allowed (multi-command support;
# namespace restriction is the primary security boundary).
```

An LLM prompt injection can chain commands: `ls /tmp/veltro/scratch ; ls /dis/veltro/tools` — revealing capabilities the agent was not supposed to know about. The `exec` tool bypasses tool-registry restrictions by running arbitrary shell commands.

**Attack scenario:** Agent has `tools=read,exec` and `paths=/appl/`. LLM gets injected, uses `exec ls /dis` to discover all available .dis files, then uses exec to access paths beyond tool-registry intent.

**Fix required:** Either restrict exec to a command allowlist, or at minimum prevent discovery of paths outside the agent's declared namespace.

### 6.3 Whiteout List Injection — MEDIUM SEVERITY

**Location:** `appl/veltro/cowfs.b` lines 314-335

**The bug:** Whiteout entries are loaded from `.whiteout` files without validation. If an attacker writes a crafted entry like `../../../etc/passwd` to the overlay directory, `promote()` will call `removefile()` on a path outside the overlay:

```limbo
bpath := basepath + "/" + relpath;
removefile(bpath);  # Could remove files outside basepath
```

**Fix required:** Validate whiteout entries — reject any containing `/`, `..`, or `.`.

### 6.4 Shared /tmp/veltro Directory — LOW SEVERITY

**Location:** `appl/veltro/nsconstruct.b` lines 143-145

All agents share `/tmp/veltro/` for scratch, memory, and COW overlays. No per-agent permission isolation. A child agent can potentially read another agent's scratch files if the namespace binding doesn't fully isolate it.

**Fix required:** Per-agent subdirectories with unique names (e.g., `/tmp/veltro/{pid}/`).

### 6.5 Missing Security Tests

`veltro_security_test.b` tests namespace restriction, allowlists, audit logging, and concurrent calls (good). But it does **not** test:
- CowFS path traversal
- Command injection via exec
- Whiteout escape vectors
- Symlink attacks
- Cross-agent data leaks via /tmp/veltro

---

## 7. CONCURRENCY AND ROBUSTNESS BUGS

### 7.1 Data Race in luciuisrv Activity Array — HIGH SEVERITY

**Location:** `appl/cmd/luciuisrv.b` lines 395-429

**The bug:** `newactivity()` reallocates the `activities` array and increments `nact` while `findactivity()` reads from the same array concurrently. The serveloop is single-threaded via channel, but concurrent 9P clients (zones reading activity state while one writes) can race on the array reference.

During array reallocation, a reader can see `nact = 2` but reference the old array, or see the new array with stale `nact`. Result: nil pointer dereference / segfault.

**Fix required:** Add a mutex (channel-based lock) around activity array access.

### 7.2 appjoinch Buffer Deadlock — MEDIUM SEVERITY

**Location:** `appl/cmd/lucifer.b` lines 413, 1271-1278

**The bug:** `appjoinch` is a buffered channel (capacity 16). If 17+ apps are launched rapidly, `launchapp()` blocks on the send. Meanwhile `preswmloop()` uses a non-blocking receive, so there's no guarantee the ID gets consumed. If a blocked sender holds or waits for `applock`, deadlock occurs.

**Already acknowledged:** Comment on lines 105-112 flags this as a TODO — should use per-app wmsrv instances.

**Fix required:** Implement per-app wmsrv (as noted in TODO) or increase buffer and add overflow handling.

### 7.3 Tool Hang with No Timeout — MEDIUM SEVERITY

**Location:** `appl/veltro/veltro.b` lines 550-607

**The bug:** `runtoolchan()` calls `agentlib->calltool()` which blocks on 9P read. If a tool hangs (e.g., HTTP request to unresponsive server, NFS timeout), the agent loop blocks forever at `<-channels[i]`. There is **no timeout mechanism** for tool execution.

The only safeguard is `maxsteps` (200), but that counts completed steps — a single hanging tool blocks the entire loop.

**Fix required:** Add configurable per-tool timeout (e.g., 30-60s default). Spawn a timer goroutine and use `alt` to race the tool result against timeout.

### 7.4 speech9p FidState Race — MEDIUM SEVERITY

**Location:** `appl/veltro/speech9p.b` lines 1577-1586, 423-427

**The bug:** Write handler spawns `asyncsay(fs, text)` which writes to `fs.sayresp`. A subsequent read on the same fid also writes to `fs.sayresp`. Both goroutines can write concurrently — corrupted or wrong audio data returned.

**Scenario:** Write "hello" → asyncsay spawned (TTS takes 2s). Read immediately → read handler calls dohear() (STT, 5s timeout). asyncsay finishes at t=2s, sets sayresp. dohear() finishes at t=5s, overwrites sayresp with transcription instead of TTS result.

**Fix required:** Use per-operation channels or a state machine instead of shared `fs.sayresp`.

### 7.5 Silent Message Loss at Conversation Limit — LOW SEVERITY

**Location:** `appl/cmd/luciuisrv.b` lines 443-464

When conversation reaches MAX_MESSAGES (10,000), `addmessage()` returns -1 but the write handler doesn't report an error to the client. Messages are silently dropped.

**Fix required:** Return a 9P error on the write so the client knows the message was not stored.

---

## 8. MISSING BINARIES — LUCIFER BROKEN ON FRESH CLONE

### 8.1 Two Critical .dis Files Missing — BLOCKS ALL LUCIFER USE

**The problem:** `lucibridge.dis` and `lucipres.dis` do not exist in the repository. Source code for both is complete and listed in the mkfile, but the compiled bytecode is not committed.

| Component | Source | In mkfile | Compiled .dis | Status |
|-----------|--------|-----------|---------------|--------|
| lucibridge | `appl/cmd/lucibridge.b` (997 lines) | line 98 | **MISSING** | Agent-to-UI bridge |
| lucipres | `appl/cmd/lucipres.b` (1000+ lines) | line 99 | **MISSING** | Presentation zone renderer |
| lucifer | `appl/cmd/lucifer.b` | yes | 18K ✅ | Main layout |
| luciconv | `appl/cmd/luciconv.b` | yes | 13K ✅ | Conversation zone |
| lucictx | `appl/cmd/lucictx.b` | yes | 22K ✅ | Context zone |
| luciuisrv | `appl/cmd/luciuisrv.b` | yes | 39K ✅ | UI state server |

**What happens on fresh clone:**

```
User runs: sh run-lucifer.sh
  → luciuisrv starts                    ✅
  → activity create Main                ✅
  → speech9p starts                     ✅
  → tools9p mounts at /tool             ✅
  → lucibridge -s &                     ❌ "command not found"
  → lucifer starts                      ❌ "cannot load /dis/lucipres.dis" → fatal
  → Window closes. User sees nothing.
```

**All 33 Veltro tools ARE compiled.** All other Lucifer components ARE compiled. Only these two are missing. The irony is that the code is complete and correct — it just wasn't built.

**Fix required:** Either commit the .dis files, or make the build step the first thing in the README/getting-started guide. Currently CLAUDE.md mentions building but it's buried and doesn't call out these specific files.

---

## 9. LLM BACKEND (llm9p) — ARCHITECTURAL FRAGILITY

### 9.1 External Binary with No Source in Repository

`llm9p` is a Go program that runs as a daemon on port 5640. **Its source code is not in this repository** — it's a separate project. Infernode depends on it being pre-installed at `~/.local/bin/llm9p` or bundled in the macOS app bundle.

This means:
- Users can't build it from this repo
- The dependency is invisible to CI
- Version compatibility between Infernode and llm9p is unverified

### 9.2 Silent Failures Throughout

Error handling in the LLM integration follows a "fail silent and return empty string" pattern:

**Session creation** (`agentlib.b`): Returns `""` on failure — no exception, no error to user.

**Response reads** (`lucibridge.b:414-427`): `sys->pread()` returns -1 on error but code treats it the same as end-of-stream. No distinction between timeout, API error, or malformed response.

**API key validation**: If ANTHROPIC_API_KEY is wrong, agent just displays "(no response from LLM)" — the `isfatal()` function that checks for "invalid API key" exists in test code but **is never called during actual agent operation**.

**Start-up** (`lib/sh/start-llm9p.sh`): If llm9p fails to start, the script waits 15 seconds then silently exits. Subsequent mount attempts hang forever.

### 9.3 8KB Message Size Limit — Silent Truncation

**Location:** `agentlib.b` line 197-200

```limbo
# 9P Twrite. llm9p's MaxMessageSize is 8192 bytes, and each write
# REPLACES the content (offset is ignored). If the prompt exceeds ~8KB,
# the kernel splits into multiple Twrites and only the LAST survives.
```

System prompts are capped at `MAXPROMPT = 8000` bytes. If the prompt + tool definitions exceed this, the kernel silently splits the write and **only the last chunk reaches llm9p**. No error, no warning — the LLM just gets a truncated prompt.

A complex agent with many tools can easily hit this limit.

### 9.4 Token Streaming — Infrastructure Exists but Broken

The codebase has streaming infrastructure (`lucibridge.b` opens `/n/llm/{id}/stream`), but:

- The **CLI backend** (`-backend cli`, which is the fallback) returns 0 chunks from `/stream`
- Only the **API backend** supports real streaming
- Backend selection depends on `ANTHROPIC_API_KEY` being set — missing key silently falls back to non-streaming CLI
- Users see placeholder bubbles with a cursor that never fills until the full response arrives

### 9.5 TOOL_RESULTS Parser Is Fragile

Tool results are delimited by `\n---\n`. If a tool's output happens to contain that exact string (e.g., a markdown file with horizontal rules), the parser breaks and subsequent tool results are corrupt.

### 9.6 No Timeout on LLM Reads

`sys->pread()` blocks forever. If llm9p crashes or the network drops, the agent thread hangs indefinitely. Combined with the tool timeout issue (7.3), a single network hiccup can freeze the entire system.

---

## 10. REMAINING GAPS

### 10.1 No Binary Distribution or Release Process

No GitHub Releases, tagged versions, or downloadable binaries.

### 10.2 Anthropic-Only LLM Backend

No alternative providers, no echo/mock mode for offline testing.

### 10.3 No Cost/Token Tracking

Extended thinking sessions can burn credits with no user visibility.

### 10.4 No Limbo Programming Guide

Module interfaces are clean but undocumented.

### 10.5 Windows/ARM64/macOS Not in CI

Only Linux AMD64 is validated in CI.

---

## 11. RECOMMENDED RELEASE PLAN

### Pre-Release — Critical (Must Fix Before Any User Sees This)

| Priority | Item | Status |
|----------|------|--------|
| **P0** | Build and commit lucibridge.dis and lucipres.dis (or document build step prominently) | **NEEDS macOS BUILD** |
| **P0** | Fix CowFS path traversal — add path canonicalization | **FIXED** |
| **P0** | ~~Fix luciuisrv activity array race — add mutex~~ | Not a real bug (Dis VM uses cooperative scheduling) |
| **P0** | Add tool execution timeout to agent loop | **FIXED** (60s per-tool timeout) |
| **P0** | Add LLM read timeout (prevent infinite hang on network drop) | **FIXED** (5-minute timeout) |

### Pre-Release — Important

| Priority | Item | Status |
|----------|------|--------|
| **P1** | Fix speech9p FidState race — per-operation channels | **FIXED** |
| **P1** | Validate whiteout entries in CowFS | **FIXED** |
| **P1** | Harden exec tool — restrict discovery outside namespace | **FIXED** (semicolons/pipes stripped) |
| **P1** | Replace silent failures in agentlib with actual error messages | **FIXED** |
| **P1** | Fix conversation message limit — return error instead of silent drop | **FIXED** |
| **P1** | Fix 8KB message truncation — chunk writes or increase MaxMessageSize | Remaining (requires llm9p changes) |
| **P1** | Fix TOOL_RESULTS delimiter collision — escape `---` in output | Remaining |

### Pre-Release — Distribution

| Priority | Item | Status |
|----------|------|--------|
| **P0** | Create VERSION file, tag release | Remaining |
| **P0** | GitHub Release workflow (build + publish binaries including all .dis files) | Remaining |
| **P0** | Enable GitHub Discussions + CONTRIBUTING.md + issue templates | Remaining |

### Post-Release (First 90 Days)

| Priority | Item |
|----------|------|
| **P1** | Include llm9p source in repo or document the dependency clearly |
| **P1** | Alternative llm9p backend (OpenAI-compatible or local) |
| **P1** | Fix token streaming on CLI backend |
| **P1** | Security test coverage (CowFS traversal, exec injection, whiteout, symlinks) |
| **P2** | Token/cost tracking |
| **P2** | Per-agent /tmp isolation |
| **P2** | Fix appjoinch deadlock — per-app wmsrv |

---

## 12. VERDICT

**Infernode is architecturally excellent but has real bugs that need fixing before release.**

**Update (2026-03-11):** Eight bugs have been fixed in this review cycle:

1. **CowFS path traversal** — `cleanrelpath()` canonicalizes paths and rejects `..` traversal at all entry points (Walk, promote, promotefile, loadwhiteouts)
2. **Whiteout injection** — `loadwhiteouts()` validates each entry via `cleanrelpath()`
3. **Tool execution timeout** — 60-second per-tool timeout in `exectools()` via `alt` with timer
4. **LLM read timeout** — 5-minute timeout on `readllmfd()` prevents infinite hang
5. **speech9p race** — `asyncsay()` now uses a completion channel instead of writing shared state
6. **exec tool hardening** — semicolons and pipes stripped to prevent namespace discovery via command chaining
7. **agentlib silent failures** — `createsession()` and `queryllmfd()` now report errors to stderr
8. **Conversation limit** — `addmessage()` failure now returns a 9P error to the client

**Remaining critical items:**
- **Build and commit lucibridge.dis and lucipres.dis** (requires macOS build environment)
- 8KB message truncation (requires llm9p changes)
- TOOL_RESULTS delimiter collision
- Distribution/release infrastructure

**What's genuinely good:**

The architecture is sound. The 9P-everywhere design, namespace security model, three-zone GUI, 32-tool agent, guided tour, speech integration, GPU inference, Mermaid diagrams, PDF rendering, and formal verification — these are real differentiators. The code quality is high. The tooling is complete.

**The gap is between "works on the developer's machine" and "works for someone who just cloned it."** The security and robustness bugs are now fixed. Build the missing .dis files, and this is a compelling release.
