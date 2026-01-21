# NERV InferNode

[![Quick Verification](https://github.com/NERVsystems/infernode/actions/workflows/simple-verify.yml/badge.svg)](https://github.com/NERVsystems/infernode/actions/workflows/simple-verify.yml)
[![Security Scanning](https://github.com/NERVsystems/infernode/actions/workflows/security-scan.yml/badge.svg)](https://github.com/NERVsystems/infernode/actions/workflows/security-scan.yml)

**64-bit Inferno OS for embedded systems, servers, and AI agents**

InferNode is a lightweight, headless Inferno OS designed for modern 64-bit systems. Built for efficiency and minimal resource usage, it provides a complete Plan 9-inspired operating environment without graphics overhead.

## Features

- **Lightweight:** 15-30 MB RAM, 2-second startup
- **Headless:** Console-only operation, no X11 dependency
- **Complete:** 280+ utilities, full shell environment
- **Networked:** TCP/IP stack, 9P filesystem protocol
- **Portable:** Host filesystem access via Plan 9 namespace

## Quick Start

```bash
# Linux x86_64 (Intel/AMD)
./build-linux-amd64.sh
./emu/Linux/o.emu -r.

# Linux ARM64 (Jetson, Raspberry Pi, etc.)
./build-linux-arm64.sh
./emu/Linux/o.emu -r.

# macOS ARM64 (Apple Silicon)
./emu/MacOSX/o.emu -r.
```

The `-r.` option tells the emulator to use the current directory as the Inferno root filesystem. The `.` is standard Unix for "current directory" - so `-r.` means "root is here". This lets you run directly from the source tree without installing.

You'll see the `;` prompt:

```
; ls /dis
; pwd
; date
```

See [QUICKSTART.md](QUICKSTART.md) for details.

## GUI Support (Optional)

InferNode supports an **optional SDL3 GUI backend** for graphical applications like Acme and the window manager.

```bash
# Install SDL3 (macOS)
brew install sdl3 sdl3_ttf

# Build with GUI support
cd emu/MacOSX
mk GUIBACK=sdl3 o.emu

# Run Acme editor (graphical)
./o.emu -r../.. acme

# Run window manager
./o.emu -r../.. wm/wm
```

**Features:**
- Cross-platform (macOS Metal, Linux Vulkan, Windows D3D)
- GPU-accelerated rendering
- High-DPI support (Retina displays)
- Zero overhead when GUI not used

**Default is headless** (no SDL dependency). See [docs/SDL3-GUI-PLAN.md](docs/SDL3-GUI-PLAN.md) for details.

## Use Cases

- **Embedded Systems** - Minimal footprint (10-20 MB)
- **Server Applications** - Lightweight, efficient
- **AI Agents** - Scriptable environment with data processing
- **Development** - Fast Limbo compilation and testing
- **9P Services** - Filesystem export/import over network

## What's Inside

- **Shell** - Interactive command environment
- **280+ Utilities** - Standard Unix-like tools
- **Limbo Compiler** - Fast compilation of Limbo programs
- **9P Protocol** - Distributed filesystem support
- **Namespace Management** - Plan 9 style bind/mount
- **TCP/IP Stack** - Full networking capabilities

## Performance

- **Memory:** 15-30 MB typical usage
- **Startup:** 2 seconds cold start
- **CPU:** 0-1% idle, efficient under load
- **Footprint:** 1 MB emulator + 10 MB runtime

See [docs/PERFORMANCE-SPECS.md](docs/PERFORMANCE-SPECS.md) for benchmarks.

## Platforms

- **ARM64 Linux** - Jetson AGX, Raspberry Pi 4/5
- **AMD64 Linux** - Containers, minimal Linux
- **ARM64 macOS** - Apple Silicon (M1/M2/M3)

## Documentation

- [QUICKSTART.md](QUICKSTART.md) - Getting started
- [docs/PERFORMANCE-SPECS.md](docs/PERFORMANCE-SPECS.md) - Performance benchmarks
- [docs/DIFFERENCES-FROM-STANDARD-INFERNO.md](docs/DIFFERENCES-FROM-STANDARD-INFERNO.md) - How InferNode differs
- [docs/](docs/) - Complete technical documentation

## Building

```bash
# Linux x86_64 (Intel/AMD)
./build-linux-amd64.sh

# Linux ARM64
./build-linux-arm64.sh

# macOS ARM64
export PATH="$PWD/MacOSX/arm64/bin:$PATH"
mk install
```

## About

InferNode is a specialized fork of [acme-sac](https://github.com/caerwynj/acme-sac), focused on headless operation and modern 64-bit platforms. It provides a complete Inferno OS environment optimized for embedded systems, servers, and AI applications.

## License

As per original Inferno OS and acme-sac licenses.

---

**NERV InferNode** - Lightweight Inferno OS for ARM64 and AMD64
