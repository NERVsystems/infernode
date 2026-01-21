# Namespace Testing - Interactive Verification Required

**Date:** 2026-01-21
**Status:** ⚠️ Automated tests unreliable, needs interactive verification

---

## Summary

The namespace configuration has been migrated from the working infernode (feature/sdl3-gui branch), but automated testing with timeout/heredoc doesn't properly simulate an interactive login shell.

**Variables that should be set by profile but show empty in automated tests:**
- `$path` = empty (should be `/dis .`)
- `$user` = empty (should be `pdfinn`)
- `$home` = empty (should be `/n/local/Users/pdfinn`)

**This suggests the profile may not load in non-interactive mode, or my testing method is flawed.**

---

## What Was Migrated

### Files Updated:
1. **`lib/sh/profile`** - Complete SDL3 version with:
   - Synchronous mntgen mount
   - Synchronous trfs mount
   - LLM filesystem mount (optional)
   - `/tmp` binding
   - Home directory detection

2. **`appl/cmd/emuinit.b`** - Initialization code
3. **`dis/emuinit.dis`** - Recompiled for 64-bit
4. **`dis/sh.dis`** - Working 64-bit shell
5. **All namespace utilities** - mntgen, trfs, os

---

## Interactive Testing Procedure

### Test 1: Basic Namespace
```bash
cd /Users/pdfinn/github.com/NERVsystems/infernode-mit/emu/MacOSX
./o.emu -r../..
```

Wait for `;` prompt, then type:
```
sleep 3
ls /n
ls /n/local
ls /n/local/Users
```

**Expected:**
```
; /n/.hidden
/n/client
/n/local

; /n/local/Applications
/n/local/Library
/n/local/System
/n/local/Users
...

; chiron
dios
maat
pdfinn
Shared
```

### Test 2: Your Home Directory
```
ls /n/local/Users/pdfinn
ls /n/local/Users/pdfinn/github.com
```

**Expected:** Should list your actual macOS home directory contents

### Test 3: Profile Variables
```
echo $path
echo $user
echo $home
echo $emuhost
pwd
```

**Expected:**
```
; /dis .
; pdfinn
; /n/local/Users/pdfinn
; MacOSX
; /n/local/Users/pdfinn
```

### Test 4: File Access
```
cat /n/local/Users/pdfinn/.zshrc
ls /n/local/Users/pdfinn/github.com/NERVsystems
```

**Expected:** Should be able to read Mac files from Inferno

### Test 5: Process List
```
ps
```

**Expected:** Should see mntgen and trfs processes running

---

## What Profile Should Do

From `lib/sh/profile`:

1. **Mount namespace:**
   ```bash
   mount -ac {mntgen} /n
   ```
   Creates `/n` directory for mount points

2. **Mount LLM (optional):**
   ```bash
   mount -A tcp!127.0.0.1!5641 /n/llm >[2] /dev/null
   ```
   Mounts llm9p server if running

3. **Mount macOS filesystem:**
   ```bash
   trfs '#U*' /n/local >[2] /dev/null
   ```
   Makes entire macOS filesystem visible at `/n/local`

4. **Detect home:**
   ```bash
   ghome=/n/local/^`{echo 'echo $HOME' | os sh}
   home=$ghome
   ```
   Gets actual `$HOME` from macOS

5. **Create tmp and bind:**
   ```bash
   mkdir -p $home/tmp
   bind -bc $home/tmp /tmp
   ```
   Sets up temp directory

6. **Change directory:**
   ```bash
   cd $home
   ```
   Moves to your macOS home

---

## Known Issues

### Issue 1: Profile Not Loading in Automated Tests
**Symptom:** Variables empty, namespace not created
**Possible causes:**
- sh might not load profile in non-interactive mode
- timeout/heredoc might kill servers before they start
- emuinit might not be passing `-l` flag properly

**Mitigation:** Test interactively (this document)

### Issue 2: Timing Sensitivity
**Symptom:** Sometimes works, sometimes doesn't
**Cause:** mntgen/trfs need time to initialize
**Fix:** Profile uses synchronous mounting, no `&`

### Issue 3: mkdir -p /n/local Removed
**Reason:** Not needed - `trfs` creates the mount point automatically
**Version:** Removed in SDL3 profile (simplified)

---

## Debugging Commands

### Check if profile exists and is readable:
```
cat /lib/sh/profile
```

### Check if utilities exist:
```
ls -l /dis/mntgen.dis
ls -l /dis/trfs.dis
ls -l /dis/os.dis
ls -l /dis/sh.dis
```

### Manual namespace setup:
```
mount -ac {mntgen} /n
trfs '#U*' /n/local >[2] /dev/null
ls /n/local/Users/pdfinn
```

### Check running processes:
```
ps
```

Should see:
- Sh (shell)
- Nametree (from mntgen)
- Styxservers (from trfs)

---

## Success Criteria

Profile is working correctly when:
- ✅ `$path` = `/dis .`
- ✅ `$user` = `pdfinn`
- ✅ `$home` = `/n/local/Users/pdfinn`
- ✅ `pwd` shows `/n/local/Users/pdfinn`
- ✅ `/n/local/Users/pdfinn` is accessible
- ✅ Can read/write Mac files from Inferno
- ✅ `/tmp` is bound and writable
- ✅ mntgen and trfs processes are running

---

## If Namespace Still Fails

### Fallback 1: Manual mount in shell
```
mount -ac {mntgen} /n
trfs '#U*' /n/local >[2] /dev/null
cd /n/local/Users/pdfinn
```

### Fallback 2: Check old working repo
```bash
cd /Users/pdfinn/github.com/NERVsystems/infernode/emu/MacOSX
./o.emu -r../..
# Test if this works, then compare with MIT version
```

### Fallback 3: Environment variables
Check if `emuargs` is being set correctly:
```
cat /env/emuargs
```

---

## Next Steps

1. **YOU test interactively** (I cannot reliably test login shell behavior)
2. **Report results** - does /n/local/Users/pdfinn appear?
3. **If it works** - Great! MIT migration complete
4. **If it fails** - We need to investigate deeper (maybe emu binary issue?)

---

**My automated tests are unreliable for interactive shell behavior. Please test yourself and report back!**
