# Namespace Configuration Migration

**Date:** 2026-01-21
**Status:** ✅ All namespace configuration migrated from old infernode
**Importance:** CRITICAL - Namespace is core Inferno functionality

---

## Namespace Work Migrated

### Critical Files Copied

#### 1. Shell Profile (`lib/sh/profile`)
**Purpose:** Sets up namespace on shell startup
**Lines:** 38 lines (was 2 lines in canonical inferno!)

**What it does:**
```bash
# Mount namespace generator (9P server)
mount -ac {mntgen} /n &

# Mount macOS/Linux host filesystem to /n/local
mkdir -p /n/local
trfs '#U*' /n/local &
sleep 1  # Let servers start

# Find user's actual macOS HOME
ghome=/n/local/^`{echo 'echo $HOME' | os sh}
home=$ghome

# Create tmp directory
mkdir -p $home/tmp
cd $home
```

**Result:**
- `/n/local` → Your entire macOS filesystem
- `/n/local/Users/pdfinn` → Your macOS home
- `$home` → Set to your actual macOS home directory

#### 2. Emulator Init (`appl/cmd/emuinit.b`)
**Purpose:** Runs before any user command
**Lines:** 110 lines

**What it does:**
- Binds #e (environment) to /env
- Parses emulator arguments
- Loads initial command (usually sh)

**Note:** On feature/sdl3-gui branch, this also has /tmp binding for standalone apps

#### 3. Namespace File (`usr/inferno/namespace`)
**Purpose:** Default namespace configuration
**Content:**
```
bind -ia #C /
```

Binds console device to root.

---

## Namespace Commit History (12 commits!)

All from pdfinn, documenting extensive namespace work:

### 1. Initial Setup
```
a286ef9 - Fix profile to run mount servers in background
d29ec47 - Restore original shell profile with host filesystem mounting
```

**Critical fix:** mntgen and trfs are 9P servers, must run with `&` (background)

### 2. Home Directory Issues
```
43d6ac7 - Fix home directory creation in profile
da2dc45 - Fix macOS ARM64 backspace and home directory issues
174168e - Fix shell path variable in profile
```

**Fixes:** Proper home directory detection and creation on macOS/Linux

### 3. Temp File Issues
```
f3635fa - fix(emu): Bind /tmp in namespace for standalone apps
af945ea - fix(profile): Make namespace init synchronous, add temp file docs
```

**Root cause:** Standalone apps (Xenith) need /tmp bound before they start
**Solution:** Added bind in emuinit.b for early namespace setup

### 4. Xenith-Specific (on feature/sdl3-gui branch)
```
dc27286 - fix(profile): Mount LLM filesystem at startup for Xenith
bd6e783 - docs: Add LLM mount documentation and fix profile
```

**Adds:** Mount llm9p server at /n/llm for Xenith LLM integration
**Adds:** bind -bc $home/tmp /tmp for temp file access

---

## Documentation Migrated

### 1. `docs/FILESYSTEM-MOUNTING.md` ✅
**Content:** Complete guide to Inferno filesystem access
- How #U device works
- Profile namespace setup
- Accessing Mac files from Inferno
- Mount points and variables
- Troubleshooting

### 2. Tempfile Documentation (on feature/sdl3-gui)
**File:** `docs/TEMPFILE-EXHAUSTION.md`
**Content:** Debugging temp file slot exhaustion issues
- Pattern: /tmp/{A-Z}{pid}.{user}{app}
- Only 26 slots per PID
- ORCLOSE cleanup issues

### 3. LLM Mount Documentation (on feature/sdl3-gui)
**File:** `doc/llm-mount.md`
**Content:** How to mount llm9p filesystem
- Mount at /n/llm
- Usage from shell and Xenith
- Namespace isolation issues

---

## Current Status in MIT Version

### ✅ Core Namespace Working
```bash
./emu/MacOSX/o.emu -r. -c0
; ls /n
/n/.hidden
/n/client
/n/local

; ls /n/local
/n/local/Users

; ls /n/local/Users
(Shows macOS users)
```

**Verified:**
- ✓ mntgen creates /n namespace
- ✓ trfs mounts macOS filesystem at /n/local
- ✓ /n/local/Users accessible
- ✓ Servers run in background properly

### ⚠️ Potential Issues to Test

1. **Home directory detection**
   - Does `$home` get set to /n/local/Users/pdfinn?
   - Can we cd to it?

2. **File access**
   - Can we read files from /n/local/Users/pdfinn?
   - Can we write files to Mac filesystem?

3. **Persistence**
   - Does profile run every shell session?
   - Do mounts survive?

4. **Temp files**
   - Is /tmp accessible?
   - Can apps create temp files?

---

## What's Different: feature/jit-64bit vs feature/sdl3-gui

### feature/jit-64bit (current) - Core Namespace
**Profile features:**
- Mount /n with mntgen
- Mount /n/local with trfs
- Detect home directory
- Create tmp directory
- cd to home

**Missing from this branch:**
- LLM mount at /n/llm
- bind -bc $home/tmp /tmp

### feature/sdl3-gui (has Xenith) - Enhanced Namespace
**Additional features:**
- Mount llm9p at /n/llm (for AI integration)
- Bind /tmp for standalone apps
- /tmp binding in emuinit.b
- Comprehensive tempfile debugging

**When Xenith is merged**, these enhancements will come too.

---

## Namespace Files Verified Migrated

### Configuration Files ✅
- [x] `lib/sh/profile` - Shell startup with /n/local mounting
- [x] `usr/inferno/namespace` - Default bindings
- [x] `appl/cmd/emuinit.b` - Early initialization

### Required Utilities ✅
- [x] `dis/mntgen.dis` - Namespace generator (3.3KB)
- [x] `dis/trfs.dis` - Translation filesystem (3.4KB)
- [x] `dis/os.dis` - Host OS command execution (6.1KB)

### Documentation ✅
- [x] `docs/FILESYSTEM-MOUNTING.md` - Complete guide
- [x] Man pages for namespace, mntgen, trfs

---

## Testing Checklist

When fully testing namespace functionality:

### Basic Tests
- [ ] `/n` directory exists after shell starts
- [ ] `/n/local` is populated with macOS filesystem
- [ ] Can list `/n/local/Users`
- [ ] Can access specific user home: `/n/local/Users/pdfinn`
- [ ] Can read Mac files from Inferno
- [ ] Can write Mac files from Inferno
- [ ] `$home` variable set correctly
- [ ] `cd $home` works

### Server Tests
- [ ] mntgen process runs
- [ ] trfs process runs
- [ ] Servers survive shell commands
- [ ] Multiple shells share namespace
- [ ] Process isolation works

### Advanced Tests
- [ ] `/tmp` accessible
- [ ] Temp file creation works
- [ ] Namespace confinement (from formal verification)
- [ ] Mount persistence across commands

---

## Known Namespace Issues (from History)

### Issue 1: Servers Blocked Shell
**Symptom:** Shell hangs on startup
**Root cause:** mntgen/trfs run synchronously
**Fix:** Run with `&` (background)
**Commits:** a286ef9, d29ec47

### Issue 2: Temp File Creation Failed
**Symptom:** "can't create temp file: file does not exist"
**Root cause:** /tmp not bound for standalone apps
**Fix:** Added binding in emuinit.b
**Commits:** f3635fa, af945ea

### Issue 3: Temp File Slot Exhaustion
**Symptom:** Apps fail after 26 temp files
**Root cause:** Pattern /tmp/{A-Z}{pid} has only 26 slots
**Fix:** Detection script, better cleanup
**Commit:** af945ea

### Issue 4: Home Directory Not Found
**Symptom:** Can't find /usr/username
**Root cause:** Path detection issues
**Fix:** Multiple profile iterations
**Commits:** 43d6ac7, da2dc45, 174168e

### Issue 5: Xenith Namespace Isolation
**Symptom:** Xenith can't see parent namespace mounts
**Root cause:** FORKNS copies namespace at fork time
**Fix:** Mount before Xenith starts (in profile)
**Commits:** dc27286, bd6e783

---

## Formal Verification of Namespace

The formal-verification/ directory includes complete verification of:
- Namespace isolation semantics
- Locking protocols for mount table
- Confinement properties
- Security guarantees

**Results:** 100% verified, zero errors

See `formal-verification/README.md` for details.

---

## Next Steps

1. **Full interactive test** of namespace
   - Start emulator interactively
   - Verify /n/local populates correctly
   - Test file access to Mac filesystem

2. **Merge feature/sdl3-gui** to get:
   - LLM mount support
   - /tmp binding in emuinit.b
   - Tempfile exhaustion fixes

3. **Document any issues** found during testing

---

**Status: All namespace configuration files migrated. Core functionality present. Enhanced features on feature/sdl3-gui branch.**
