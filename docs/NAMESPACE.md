# InferNode Namespace Guide

A comprehensive guide to understanding and configuring the InferNode namespace system.

---

## Overview

InferNode inherits Inferno's powerful namespace abstraction, which provides a unified
view of all resources (files, devices, networks, services) as a hierarchical filesystem.
Unlike traditional operating systems where the filesystem is a fixed structure, InferNode's
namespace is **per-process** and **dynamically configurable**.

---

## Key Concepts

### What is a Namespace?

A namespace is a process's view of the filesystem hierarchy. Each process can have a
different namespace, meaning `/n/local` might point to different resources for different
processes. This enables:

- **Isolation**: Processes can be sandboxed with limited views
- **Flexibility**: Resources can be mounted anywhere in the hierarchy
- **Transparency**: Network resources appear as local files

### The Root Namespace

When InferNode starts, the emulator creates a minimal root namespace:
```
/dev          Device files (console, draw, pointer, etc.)
/fd           File descriptors
/prog         Process information
/net          Network stack
/env          Environment variables
/chan         Named channels for IPC
/dis          Dis bytecode modules (programs)
```

This is NOT the host filesystem - it's InferNode's internal virtual filesystem.

### Accessing the Host Filesystem

The host (macOS/Linux) filesystem is NOT automatically available. You must explicitly
mount it using the `trfs` (translate filesystem) command:

```sh
trfs '#U*' /n/local
```

This mounts the host's root filesystem (`#U*`) at `/n/local`, giving you access to:
- `/n/local/Users/pdfinn` (macOS home)
- `/n/local/etc` (host config)
- etc.

---

## Namespace Configuration

### The Profile System

InferNode uses a shell profile (`/lib/sh/profile`) to configure the namespace at login.
This profile runs when you start the shell with the `-l` (login) flag.

**Current Profile (`/lib/sh/profile`):**
```sh
#!/dis/sh.dis
# InferNode shell initialization for macOS
load std

# Set command search path
path=(/dis .)

# Get username from Inferno
user="{cat /dev/user}

# Mount namespace generator (runs in background - it's a 9P server)
mount -ac {mntgen} /n &

# Mount LLM filesystem if server is running (optional)
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

### Key Profile Components

#### 1. mntgen - Namespace Generator
```sh
mount -ac {mntgen} /n &
```
- Creates mount points on demand under `/n`
- The `{mntgen}` syntax runs mntgen as a 9P server
- `-a` means mount after (union mount)
- `-c` means create mount point if needed
- `&` runs in background (CRITICAL - see below)

#### 2. trfs - Host Filesystem Translator
```sh
trfs '#U*' /n/local &
```
- Translates between InferNode and host filesystem
- `#U*` is the host root device
- Must run in background with `&`

#### 3. Why Background Execution Matters

Both `mntgen` and `trfs` are **9P servers** - they run continuously to serve
filesystem requests. If you run them without `&`:

```sh
# WRONG - will block forever!
mount -ac {mntgen} /n
# Shell never reaches this line...
```

The shell waits for the server to exit, but 9P servers run indefinitely.
With `&`, the server runs in the background and the shell continues:

```sh
# CORRECT - runs in background
mount -ac {mntgen} /n &
# Shell continues immediately
```

---

## Running Applications

### With Namespace (Recommended)

To run applications with full namespace access, use the login shell:

```bash
# Xenith editor with namespace
./o.emu -r../.. sh -l -c 'xenith -t dark'

# Window manager with namespace
./o.emu -r../.. sh -l -c wm/wm

# Interactive shell with namespace
./o.emu -r../.. sh -l
```

The `-l` flag tells `sh` to source `/lib/sh/profile`, which configures:
- Host filesystem at `/n/local`
- Home directory (`$home`)
- Temp directory bound to `/tmp`

### Without Namespace

Running applications directly bypasses the profile:

```bash
# Window manager WITHOUT namespace
./o.emu -r../.. wm/wm

# This works but /n/local is NOT available
# You cannot access host files
```

### Why the Difference?

When you run `./o.emu -r../.. wm/wm`:
1. Emulator starts with root namespace only
2. `emuinit` runs, which executes `wm/wm` directly
3. Profile is NOT sourced
4. No namespace configuration happens

When you run `./o.emu -r../.. sh -l -c wm/wm`:
1. Emulator starts with root namespace
2. `emuinit` runs `sh -l -c wm/wm`
3. Shell sources `/lib/sh/profile`
4. Profile configures namespace (mntgen, trfs, home)
5. THEN `wm/wm` runs with full namespace

---

## Debugging Namespace Issues

### Check if Namespace is Configured

```sh
# In InferNode shell, check for /n/local
ls /n/local

# If you see "file does not exist", namespace is not configured
# If you see host directories (Users, etc), it's working
```

### Check Running Servers

```sh
# List processes
ps

# Look for mntgen and trfs processes
# If missing, profile didn't run or servers died
```

### Check Mount Points

```sh
# Show current namespace
ns

# Look for:
#   mount ... /n
#   bind ... /n/local
```

### Common Problems

#### Problem: `/n/local` doesn't exist
**Cause:** Profile didn't run (missing `-l` flag)
**Solution:** Run with `sh -l -c 'your-command'`

#### Problem: Profile hangs
**Cause:** Missing `&` on server commands
**Solution:** Ensure mntgen and trfs run in background:
```sh
mount -ac {mntgen} /n &    # Note the &
trfs '#U*' /n/local &      # Note the &
```

#### Problem: `/n/local` exists but is empty
**Cause:** trfs didn't start or failed
**Solution:**
1. Check if trfs process is running: `ps | grep trfs`
2. Try mounting manually: `trfs '#U*' /n/local &`
3. Wait a moment: `sleep 1`
4. Check again: `ls /n/local`

#### Problem: Can't find home directory
**Cause:** Username mismatch or path error
**Solution:** Check your username:
```sh
cat /dev/user
# Should show your macOS username

# Home should be at:
ls /n/local/Users/$user
```

---

## Advanced Configuration

### Custom Mount Points

You can mount additional resources:

```sh
# Mount a network filesystem
mount tcp!fileserver!564 /n/remote

# Mount a specific host directory
bind /n/local/Users/pdfinn/projects /projects
```

### Union Mounts

InferNode supports union mounts (overlay filesystems):

```sh
# Mount with -b (before) to overlay
bind -b /custom/bin /dis

# Now /dis contains both original and /custom/bin files
# /custom/bin takes precedence for conflicts
```

### Per-Application Namespaces

Applications can create their own namespace modifications:

```limbo
# In Limbo code
sys->bind("/n/local/app-data", "/data", Sys->MREPL);
```

---

## Architecture Reference

### Namespace Hierarchy

```
/                       Root of namespace
├── dev/                Devices
│   ├── cons            Console (keyboard/screen)
│   ├── draw            Graphics
│   ├── pointer         Mouse
│   └── user            Current username
├── dis/                Dis modules (programs)
├── n/                  Mount point (via mntgen)
│   ├── local/          Host filesystem (via trfs)
│   │   └── Users/
│   │       └── pdfinn/ Your macOS home
│   └── llm/            LLM filesystem (optional)
├── chan/               Named channels
├── env/                Environment variables
├── prog/               Process information
└── tmp/                Temp files (bound from $home/tmp)
```

### 9P Protocol

All namespace operations use the 9P protocol:
- `mount` - Attach a 9P server to namespace
- `bind` - Create aliases in namespace
- `unmount` - Remove mount points

9P servers (like mntgen, trfs) speak this protocol to provide filesystem access.

---

## Quick Reference

### Essential Commands

| Command | Purpose |
|---------|---------|
| `ns` | Show current namespace |
| `ps` | Show running processes |
| `mount` | Attach 9P server |
| `bind` | Create namespace alias |
| `unmount` | Remove mount point |

### Launch Commands

| Command | Namespace | Use Case |
|---------|-----------|----------|
| `./o.emu -r../.. sh -l` | Yes | Interactive shell |
| `./o.emu -r../.. sh -l -c 'app'` | Yes | Run app with namespace |
| `./o.emu -r../.. app` | No | Quick test (no host files) |

### Profile Location

- Profile: `/lib/sh/profile`
- Minimal profile: `/lib/sh/profile.minimal`
- Profile loaded by: `sh -l` (login shell)

---

## Troubleshooting Checklist

1. [ ] Are you using `sh -l` or `sh -l -c`?
2. [ ] Does `/lib/sh/profile` exist?
3. [ ] Are mntgen and trfs running? (`ps`)
4. [ ] Do they have `&` (background)?
5. [ ] Is there a `sleep` after starting servers?
6. [ ] Can you access `/n/local`?
7. [ ] Is your username correct? (`cat /dev/user`)

---

*Document created: 2026-01-22*
*InferNode MIT Edition*
