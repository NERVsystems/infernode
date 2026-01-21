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

## Formal Verification

**InferNode includes the first-ever formal verification of Inferno OS namespace isolation.**

✅ **Phase 1**: Namespace semantics (2,035 states, 0 errors)
✅ **Phase 2**: Locking protocol (4,830 states, 0 errors)
✅ **Phase 3**: C implementation (196 checks, 0 failures)
✅ **Phase 4**: Mathematical proofs (60/60 proofs, 100%)
✅ **Confinement**: Security property (2,079 states + 83 checks, 0 errors)

**Tools**: TLA+, SPIN, CBMC, Frama-C
**Result**: **100% success - zero errors across all phases**

See **[formal-verification/](formal-verification/)** for complete details, verification scripts, and reproducible results.

## Documentation

- [QUICKSTART.md](QUICKSTART.md) - Getting started
- [formal-verification/](formal-verification/) - **Formal verification (NEW)**
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
