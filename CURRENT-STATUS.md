# Current Status - MIT Migration

**Date:** 2026-01-22
**Status:** GUI Working, Some Issues Remain

---

## What Works ✅

### License
- ✅ 100% MIT licensed (forked from inferno-os/inferno-os)
- ✅ Zero GPL contamination
- ✅ All custom code migrated from original Infernode

### SDL3 GUI
- ✅ `./o.emu -r../.. wm/wm` - Window manager displays correctly
- ✅ `./o.emu -r../.. sh -l -c 'xenith -t dark'` - Xenith editor works
- ✅ Mouse and keyboard input functional
- ✅ Window resizing works
- ✅ Child windows spawn correctly (clock, colors, etc.)

### Namespace (Xenith Mode)
- ✅ Profile loads when using `sh -l`
- ✅ `/n/local/Users/<username>` accessible in xenith mode
- ✅ Home directory properly set

### Builds
- ✅ `./build-macos-sdl3.sh` produces working GUI binary
- ✅ `./build-macos-headless.sh` produces headless binary
- ✅ ARM64 executables, no compilation errors

---

## Recently Fixed

### 1. Debug Output Removed
**Fixed:** Removed RECT_DX debug messages from `libinterp/geom.c`.

### 2. App Bundle Working
**Fixed:** `open MacOSX/Infernode.app` now launches correctly.
- Icon displays in Dock
- App name shows as "Infernode"
- Launches xenith in dark mode with full namespace

---

## Known Issues ❌

### 1. Namespace Not Available in wm/wm Mode
**Symptom:** When running `./o.emu -r../.. wm/wm`, the namespace is not configured.
`/n/local/Users/pdfinn` is not accessible.

**Works:** `./o.emu -r../.. sh -l -c 'xenith -t dark'` (profile runs first)

**Root Cause:** `wm/wm` runs directly without going through shell login profile.
The profile sets up mntgen, trfs, and home directory.

**Solution:** Run wm/wm via shell: `./o.emu -r../.. sh -l -c wm/wm`

See `docs/NAMESPACE.md` for comprehensive namespace documentation

---

## Namespace Architecture

### How Namespace Setup Works

The Inferno namespace is configured by `/lib/sh/profile` when shell runs with `-l` flag:

```sh
#!/dis/sh.dis
# Infernode shell initialization for macOS
load std

# Set command search path
path=(/dis .)

# Get username from Inferno
user="{cat /dev/user}

# Mount namespace generator (runs in background - it's a 9P server)
mount -ac {mntgen} /n &

# Mount LLM filesystem if server is running (optional, non-blocking)
mount -A tcp!127.0.0.1!5641 /n/llm >[2] /dev/null

# Mount host filesystem (runs in background - it's a 9P server)
mkdir -p /n/local
trfs '#U*' /n/local &

# Give servers time to initialize
sleep 1

# Set home directory (macOS: /Users/username)
home=/n/local/Users/^$user

# Create tmp directory and bind to /tmp
mkdir -p $home/tmp
bind -bc $home/tmp /tmp

# Change to home directory
cd $home
```

### Key Components

1. **mntgen** - Namespace generator server that creates mount points on demand
2. **trfs** - Translates host filesystem (`#U*`) to Inferno namespace
3. **`&` suffix** - Critical: servers must run in background or shell blocks

### Why `-l` Flag Matters

- `sh -l` runs the login shell which sources `/lib/sh/profile`
- Without `-l`, namespace is not configured
- `wm/wm` runs directly, bypassing shell profile
- `sh -l -c 'xenith -t dark'` runs profile first, then xenith

---

## Debugging Session Summary

### Problem
GUI programs showed blank white screen after MIT migration.

### Investigation Process

1. **Initial hypothesis:** Missing flush calls in Limbo code
   - Added explicit `flush(Draw->Flushnow)` calls
   - Did not fix the issue

2. **Compared with working Infernode:**
   - Source files were identical
   - Copied known-good .dis files - still blank screen
   - Issue was NOT in Limbo code

3. **Checked commit history in working branch:**
   - Found critical fixes for 64-bit pixel handling
   - `memset32` function for 32-bit pixels on 64-bit systems
   - `0xFFFFFFFF` vs `~0` comparisons

4. **Discovered stale build artifacts:**
   - `libmemdraw.a` built Jan 19, source modified Jan 21
   - `libinterp.a` built Jan 19, source modified Jan 21
   - Libraries contained OLD code without 64-bit fixes!

### Root Cause
**Stale library build artifacts.** The C libraries were not rebuilt after critical
source changes were made. The emulator was linking against old libraries that
didn't have the 64-bit pixel format fixes.

### Solution
```bash
# Rebuild out-of-date libraries
cd libmemdraw && mk clean && mk install
cd libinterp && mk clean && mk install

# Rebuild emulator
./build-macos-sdl3.sh
```

### Lesson Learned
**Always check library modification dates vs source dates!**
```bash
# Check if libraries are current
ls -la MacOSX/arm64/lib/*.a
ls -la libmemdraw/*.c | head -1
```

---

## Build Commands

### Full Rebuild (Recommended)
```bash
cd /Users/pdfinn/github.com/NERVsystems/infernode-mit

# Rebuild libraries
for lib in lib9 libdraw libmemdraw libmemlayer libinterp; do
  (cd $lib && mk clean && mk install)
done

# Rebuild emulator
./build-macos-sdl3.sh
```

### Quick Emulator Rebuild
```bash
./build-macos-sdl3.sh
```

### Run Commands
```bash
cd emu/MacOSX

# Xenith with namespace (RECOMMENDED)
./o.emu -r../.. sh -l -c 'xenith -t dark'

# Window manager (no namespace)
./o.emu -r../.. wm/wm

# Interactive shell with namespace
./o.emu -r../.. sh -l
```

---

## TODO List

1. **Test on fresh clone** - Ensure build works from scratch
2. ~~Fix namespace in wm/wm mode~~ - DONE: Run via shell: `sh -l -c wm/wm`
3. ~~Remove RECT_DX debug output~~ - DONE: Removed from `libinterp/geom.c`
4. ~~Fix app bundle launch~~ - DONE: Working correctly

---

## Files Modified in This Session

- `lib/sh/profile` - Fixed mntgen/trfs to run in background with `&`
- `libmemdraw/draw.c` - Rebuilt (was stale)
- `libinterp/geom.c` - Removed RECT_DX debug output
- `emu/MacOSX/o.emu` - Rebuilt with fresh libraries
- `docs/NAMESPACE.md` - New comprehensive namespace documentation
