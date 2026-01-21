# Current Status - MIT Migration

**Date:** 2026-01-22
**Honesty:** This is what ACTUALLY works vs what's broken

---

## What Actually Works ✅

### License
- ✅ 100% MIT licensed
- ✅ Forked from inferno-os/inferno-os
- ✅ Zero GPL contamination
- ✅ All code migrated

### Builds Compile
- ✅ Headless build: `./build-macos-headless.sh` → 1.0M binary
- ✅ SDL3 GUI build: `./build-macos-sdl3.sh` → 1.1M binary
- ✅ Both produce ARM64 executables
- ✅ No compilation errors

### Basic Headless Shell
- ✅ `./emu/MacOSX/o.emu -r.` starts
- ✅ Basic commands work: pwd, date, ls, cat, echo
- ✅ 64-bit architecture (WORD=8)

---

## What is BROKEN ❌

### CRITICAL: Profile Not Loading
**Symptom:**
```
; echo $path
(empty)
; echo $home
(empty)
; pwd
/
```

**Expected:**
- `$path` should be `/dis .`
- `$home` should be `/n/local/Users/pdfinn`
- `pwd` should show home directory

**Impact:**
- Namespace not auto-configured
- `/n/local/Users/pdfinn` not accessible
- Must manually run mount commands every time

**Workaround:**
Manual commands work perfectly:
```
mount -ac {mntgen} /n &
mkdir -p /n/local
trfs '#U*' /n/local &
sleep 2
ls /n/local/Users/pdfinn  ← Shows full macOS home
```

### SDL3 GUI Programs Hang
**Symptom:**
```
./o.emu -r.. sh -l -c 'xenith -t dark'
(hangs forever, no window appears)

./o.emu -r.. acme
(hangs forever)

./o.emu -r.. wm/wm
SYS: process faults: Bus error
```

**Expected:**
- Xenith should open GUI window
- Acme should open editor
- wm should start window manager

**Status:** SDL3 links correctly but programs don't initialize display

### App Bundle Path
**Issue:** User typed wrong path
```
open MacOSX/Inferode.app  ← typo
```

**Correct path:**
```
cd /Users/pdfinn/github.com/NERVsystems/infernode-mit
open MacOSX/Infernode.app
```

---

## Root Cause Analysis

### Profile Loading Issue

**What we know:**
1. Profile file exists: `/lib/sh/profile`
2. Profile has correct content (from old infernode master)
3. emuinit.b passes `-l` flag to sh (line 42)
4. sh.b has profile loading code (`runscript(ctxt, LIBSHELLRC, nil, 0)`)
5. sh.dis is compiled correctly (64-bit)

**What doesn't work:**
- Profile never executes
- No variables get set
- No namespace gets configured

**Theories:**
1. `-l` flag not reaching sh correctly
2. Profile path wrong or inaccessible
3. runscript() failing silently
4. Some initialization missing

**Evidence:**
- Same issue on old infernode when I test it
- Same issue on MIT version
- Suggests my testing method is flawed OR something fundamentally changed

### SDL3 GUI Hanging

**What we know:**
1. SDL3 library links correctly
2. Binary runs and starts VM
3. Basic shell commands work
4. GUI programs specifically hang/crash

**Theories:**
1. SDL3 needs main thread initialization (macOS requirement)
2. Display context not created
3. Event loop not starting
4. Profile not loading means GUI setup incomplete

---

## What Was Accomplished Today

### Positive
- ✅ Forked MIT-licensed inferno-os
- ✅ Migrated all custom code
- ✅ Both build scripts work
- ✅ Backups created (can restore old version)
- ✅ Security-separated builds (headless vs GUI)
- ✅ GPL-free codebase

### Negative
- ❌ Profile mechanism broken
- ❌ Namespace not working automatically
- ❌ SDL3 GUI not functional
- ❌ Overstated "complete" status multiple times

---

## What Needs Fixing

### Priority 1: Profile Loading
**This blocks everything else.**

Without profile:
- No namespace
- No /n/local
- No $home detection
- GUI programs can't find resources

**Action needed:**
- Debug why profile doesn't execute
- Compare binary behavior (old working vs new broken)
- Check if there's an initialization order issue
- Find the actual commit that made it work

### Priority 2: SDL3 GUI Initialization
**After profile works.**

GUI programs hang waiting for:
- Display creation?
- Event loop?
- Some Limbo module?

**Action needed:**
- Check SDL3 initialization code in draw-sdl3.c
- Compare with old working version
- Check if xenith needs specific startup sequence
- Review SDL3 documentation

---

## Testing Required

### Profile Loading
- Interactive testing (not automated)
- Binary comparison
- Commit bisection to find the fix

### SDL3 GUI
- Actual window should appear
- Mouse/keyboard should work
- Xenith editor should be functional

---

## Honest Assessment

**Migration Status:** Code migrated, license clean, but NOT functional

**Usable for:** Nothing productive yet

**Needs:** Actual debugging to make it work

---

**I will stop claiming things are "complete" until they actually work.**
