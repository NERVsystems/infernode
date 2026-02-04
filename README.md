# NERV InferNode

[![Quick Verification](https://github.com/NERVsystems/infernode/actions/workflows/simple-verify.yml/badge.svg)](https://github.com/NERVsystems/infernode/actions/workflows/simple-verify.yml)
[![Security Scanning](https://github.com/NERVsystems/infernode/actions/workflows/security-scan.yml/badge.svg)](https://github.com/NERVsystems/infernode/actions/workflows/security-scan.yml)

**64-bit Inferno® OS for embedded systems, servers, and AI agents**

InferNode is a lightweight, headless Inferno® OS designed for modern 64-bit systems. Built for efficiency and minimal resource usage, it provides a complete Plan 9-inspired operating environment without graphics overhead.

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

InferNode supports an **optional SDL3 GUI backend** with **Xenith** as the default graphical interface.

### Xenith - AI-Native Text Environment

Xenith is an Acme fork optimized for AI agents and AI-human collaboration:

- **9P Filesystem Interface** - Agents interact via file operations, no SDK needed
- **Namespace Security** - Capability-based containment for AI agents
- **Observable** - All agent activity visible to humans
- **Multimodal** - Text and images in the same environment
- **Dark Mode** - Modern theming (Catppuccin) with full customization

See [docs/XENITH.md](docs/XENITH.md) for details.

### UI Improvements

Xenith addresses several usability issues in traditional Acme:

- **Async File I/O** - Text files, images, directories, and saves run in background threads
- **Non-Blocking UI** - UI remains responsive during file operations
- **Progressive Display** - Text appears incrementally; images show "Loading..." indicator
- **Buffered Channels** - Non-blocking sends prevent deadlocks during nested event loops
- **Unicode Input** - UTF-8 text entry with Plan 9 latin1 composition (e.g., `a'` → `á`)
- **Keyboard Handling** - Ctrl+letter support, macOS integration, compose sequences

Classic Acme freezes during file operations. On high-latency connections (remote 9P mounts, slow storage) or with large files, this blocks all interaction. The async architecture allows users to open windows, switch focus, or cancel operations while background tasks run.

### Building with GUI

```bash
# Install SDL3 (macOS)
brew install sdl3 sdl3_ttf

# Build with GUI support
cd emu/MacOSX
mk GUIBACK=sdl3 o.emu

# Run Xenith (AI-native interface)
./o.emu -r../.. xenith

# Run Acme (traditional)
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

| Platform | VM (Interpreter) | JIT Compiler | Status |
|----------|------------------|--------------|--------|
| AMD64 Linux | ✅ Working | ✅ Working | Stable |
| ARM64 Linux | ✅ Working | ⚠️ In Development | Interpreter mode stable |
| ARM64 macOS | ✅ Working | ⚠️ In Development | Interpreter mode stable |

### Platform Details

- **AMD64 Linux** - Full JIT support. Containers, servers, workstations.
- **ARM64 Linux** - Jetson AGX, Raspberry Pi 4/5. JIT in development on `feature/arm64-jit` branch.
- **ARM64 macOS** - Apple Silicon (M1/M2/M3/M4). SDL3 GUI with Metal acceleration.

## Documentation

- [docs/USER-MANUAL.md](docs/USER-MANUAL.md) - **Comprehensive user guide** (namespaces, devices, host integration)
- [QUICKSTART.md](QUICKSTART.md) - Getting started in 3 commands
- [docs/XENITH.md](docs/XENITH.md) - Xenith text environment for AI agents
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

## Development Status

### Working

- **Dis Virtual Machine** - Fully functional on all platforms (interpreter mode)
- **AMD64 JIT Compiler** - Complete and tested
- **SDL3 GUI Backend** - Cross-platform graphics with Metal/Vulkan/D3D
- **Xenith** - AI-native text environment with async I/O
- **Modern Cryptography** - Ed25519 signatures, updated certificate generation and authentication
- **Limbo Test Framework** - Unit testing with clickable error addresses
- **All 280+ utilities** - Shell, networking, filesystems, development tools

### In Development

- **ARM64 JIT Compiler** - Basic programs work (echo, cat, sh). Blocked on AXIMM storage scaling issue (32-slot limit). See `feature/arm64-jit` branch and `docs/arm64-jit/` for details.

### Roadmap

- Complete ARM64 JIT compiler
- Linux ARM64 SDL3 GUI support
- Windows port

## About

InferNode is a GPL-free Inferno® OS distribution developed by NERV Systems, focused on headless operation and modern 64-bit platforms. It provides a complete Inferno® OS environment optimized for embedded systems, servers, and AI agent applications. InferNode's namespace model provides a capability-based security architecture well-suited for AI agent isolation.

Inspired by the concept of standalone Inferno® environments, InferNode builds on the MIT-licensed Inferno® OS codebase to deliver a lightweight, headless-capable system.

## License

MIT License (as per original Inferno® OS).

---

**NERV InferNode** - Lightweight Inferno® OS for ARM64 and AMD64

<sub>Inferno® is a distributed operating system, originally developed at Bell Labs, but now maintained by trademark owner Vita Nuova®.</sub>
