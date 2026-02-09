# NERV InferNode - Performance Benchmarks

**Platform:** Apple M-series (ARM64)
**Build:** Headless, optimized
**Date:** January 2026

## Executive Summary

InferNode is **lightweight and fast**, suitable for embedded systems, edge computing, and AI agents.

**Key Metrics:**
- **Startup:** 2 seconds
- **Memory:** 15-30 MB
- **Footprint:** 10 MB on disk
- **CPU idle:** <1%

## Startup Performance

### Cold Start
```
Time to ; prompt: 2.0 seconds
```

**Breakdown:**
- Emulator init: ~0.5s
- emuinit.dis load: ~0.3s
- Shell load: ~0.5s
- Profile execution: ~0.7s

**Consistent** - No variance between runs.

### Warm Start
Not applicable - no daemon mode, fresh start each time.

## Memory Footprint

### Binary Sizes
| Component | Size |
|-----------|------|
| Emulator | 1.0 MB |
| Limbo compiler | 376 KB |
| Core libraries | 2.5 MB |
| .dis programs | 2.2 MB |
| **Total runtime** | **~6 MB** |

### RAM Usage (Resident Set Size)

**Idle at prompt:**
```
RSS: 15-20 MB
```

**Light usage** (few commands):
```
RSS: 20-30 MB
```

**Moderate usage** (multiple programs):
```
RSS: 30-50 MB
```

**Heavy usage** (many concurrent operations):
```
RSS: 50-100 MB
```

**Average:** 25 MB for typical interactive use.

### Virtual Memory
```
VSZ: ~4.1 GB
```

Most is virtual/unmapped. Actual RAM usage is RSS value.

## CPU Usage

### At Idle
```
CPU: 0.0-0.5%
```

Minimal - efficient event loop.

### During Operations

| Operation | CPU % | Duration |
|-----------|-------|----------|
| Shell command | 2-5% | <10ms |
| File listing (ls) | 3-8% | 20ms |
| Text search (grep) | 10-20% | 50-200ms |
| Limbo compilation | 20-40% | 50-500ms |
| Network I/O | 5-15% | Variable |

**Single-threaded** - Uses one core efficiently.

## Operation Benchmarks

### File Operations (Average)

| Operation | Time | Notes |
|-----------|------|-------|
| ls /dis (157 files) | 20ms | Directory listing |
| cat 100KB file | 15ms | Sequential read |
| grep pattern *.b | 100ms | Search 100 files |
| cp 1MB file | 30ms | File copy |
| rm file | 5ms | File deletion |

**Fast** - Native filesystem performance.

### Compilation (Limbo)

| Program Size | Compile Time |
|--------------|--------------|
| Hello world (10 lines) | 30-50ms |
| Small utility (100 lines) | 50-100ms |
| Medium program (500 lines) | 200-400ms |
| Large program (2000 lines) | 800ms-1.5s |
| Very large (5000 lines) | 2-3s |

**Much faster** than C compilation.

### Network Operations

| Operation | Time | Notes |
|-----------|------|-------|
| TCP connect (localhost) | 5ms | Local connection |
| TCP connect (LAN) | 10-20ms | Network latency |
| TCP connect (internet) | 50-200ms | Depends on host |
| 9P mount (local) | 15ms | Start server |
| 9P export | 10ms | Start export |

**Efficient** - Low protocol overhead.

### Process Operations

| Operation | Time |
|-----------|------|
| Spawn new Dis program | 5-10ms |
| Process switch | <1ms |
| IPC (channel send/recv) | <1ms |

**Lightweight** - Fast process creation.

## Throughput

### File I/O

**Sequential read:**
- **Speed:** ~500 MB/s (native filesystem speed)
- **Overhead:** Minimal (direct system calls)

**Sequential write:**
- **Speed:** ~400 MB/s (native filesystem speed)
- **Overhead:** Minimal

### Network Throughput

**TCP (tested with iperf-equivalent):**
- **Bandwidth:** 100+ Mbps easily sustained
- **Latency:** <10ms local, normal for network
- **Connections:** Tested with 50+ concurrent

**9P Protocol:**
- **Small files:** Efficient (low overhead)
- **Large files:** Good (streaming optimized)
- **Many files:** Excellent (protocol designed for this)

## Scalability

### Concurrent Programs
**Tested:** 20 simultaneous Dis programs
**Result:** All responsive, total RAM: ~80 MB
**Limit:** Memory-bound (each program ~2-5 MB)

### File Handles
**Limit:** OS limit (typically 1024-4096)
**InferNode overhead:** Minimal

### Network Connections
**Tested:** 50 concurrent TCP connections
**Result:** All stable, no performance degradation
**Limit:** OS limit (10,000+)

## Comparison

### vs Full Desktop Linux

| Metric | InferNode | Linux Desktop |
|--------|-----------|---------------|
| Startup | 2s | 30-60s |
| RAM idle | 20 MB | 1-2 GB |
| Footprint | 10 MB | 5-10 GB |
| CPU idle | <1% | 2-5% |

**100x lighter** than desktop OS.

### vs Docker Container

| Metric | InferNode | Docker + Alpine |
|--------|-----------|-----------------|
| Startup | 2s | 1-3s |
| RAM | 20 MB | 20-40 MB |
| Footprint | 10 MB | 15-30 MB |
| Overhead | None (native) | Container runtime |

**Comparable** but simpler (no container needed).

### vs Node.js Process

| Metric | InferNode | Node.js |
|--------|-----------|---------|
| Startup | 2s | 0.5-1s |
| RAM idle | 20 MB | 30-50 MB |
| RAM active | 30-50 MB | 100-200 MB |
| Footprint | 10 MB | 50-100 MB |

**Lighter** for equivalent functionality.

## Real-World Performance

### Use Case: File Server (9P export)
```
Memory: 25 MB
CPU: 1-5% (serving files)
Handles: 20+ concurrent clients tested
```

**Efficient for embedded file server.**

### Use Case: Automation Script
```
Startup: 2s
Memory: 20 MB (including loaded modules)
CPU: Spikes to 20-40% during execution, 0% waiting
```

**Fast for cron jobs and automation.**

### Use Case: Development Environment
```
Memory: 30-40 MB (editor, compiler, tools)
Compile: 100ms average
Test cycle: <5s total
```

**Responsive for interactive development.**

## Tuning

### Memory Pools (emu/port/alloc.c)

**Current settings (optimal for 64-bit):**
```c
{ "main",  0, 32MB max, 127 quanta, 512KB initial }
{ "heap",  1, 32MB max, 127 quanta, 512KB initial }
{ "image", 2, 64MB max, 127 quanta, 4MB initial }
```

**To increase available memory:**
- Increase maxsize (first parameter)
- Keep quanta at 127 (critical for 64-bit!)

### Thread Stack Sizes

**Default:** Adequate for most use
**If needed:** Adjust in emu/port/main.c

## Bottlenecks

**None identified in typical use.**

**Potential bottlenecks:**
- Single-threaded (won't use multiple cores)
- Memory pools (32 MB limit by default)
- Host filesystem via trfs (slight overhead)

**All addressable if needed.**

## Tested Workloads

**Sustained operations (no degradation):**
- Continuous shell use: 4+ hours
- File server: 2+ hours, 100+ operations
- Network connections: 1+ hour, 50+ clients
- Compilation: 1000+ programs

**Stable under load.**

## Resource Limits

**Practical limits observed:**
- Programs: 50+ concurrent (memory-bound)
- Files: 1000+ open (OS limit)
- Connections: 50+ tested (OS limit applies)
- Threads: 100+ (OS scheduler)

**Scales well for embedded/server use.**

## Summary

**NERV InferNode is:**

| Aspect | Rating | Notes |
|--------|--------|-------|
| Startup | ⚡ Excellent | 2 seconds |
| Memory | ⚡ Excellent | 15-30 MB |
| CPU efficiency | ⚡ Excellent | <1% idle |
| Disk usage | ⚡ Excellent | 10 MB |
| Compilation | ⚡ Excellent | 50-500ms |
| File I/O | ⚡ Excellent | Native speed |
| Networking | ✓ Good | Standard TCP/IP |
| Scalability | ✓ Good | Memory-bound |

**Ideal for resource-constrained environments.**

**Not suitable for:**
- Heavy computation (use native code)
- Multi-core parallelism (single-threaded)
- Graphics (headless build)

---

**Benchmarked on Apple M1 Pro, 16GB RAM, macOS 13-15**

**Performance data represents typical usage patterns.**

---

## JIT Compiler Benchmarks

**Date:** February 2026
**Benchmark suites:** jitbench.b (6 benchmarks), jitbench2.b (26 benchmarks, 9 categories)
**Runs:** 3 per configuration (best-of-3 reported)

Both AMD64 and ARM64 JIT compilers translate Dis VM bytecode directly to native machine code at module load time. All benchmarks verified correct (result values match across runs and between JIT/interpreter).

Full per-benchmark breakdowns: [`docs/arm64-jit/BENCHMARK-amd64-Linux.md`](arm64-jit/BENCHMARK-amd64-Linux.md), [`docs/arm64-jit/BENCHMARK-arm64-Linux.md`](arm64-jit/BENCHMARK-arm64-Linux.md), [`docs/arm64-jit/BENCHMARK-arm64-macOS.md`](arm64-jit/BENCHMARK-arm64-macOS.md)

### Cross-Platform JIT Performance

| Platform | CPU | v1 Interp | v1 JIT | v1 Speedup | v2 Interp | v2 JIT | v2 Speedup |
|----------|-----|-----------|--------|------------|-----------|--------|------------|
| **AMD64 Linux** | Intel x86-64 (2.1 GHz) | 21,255 ms | 1,500 ms | **14.2x** | 1,504 ms | 263 ms | **5.7x** |
| **ARM64 Linux** | Cortex-A78AE (Jetson) | 38,320 ms | 4,615 ms | **8.3x** | 2,743 ms | 938 ms | **2.9x** |
| **ARM64 macOS** | Apple M2 Max | 16,697 ms | 1,735 ms | **9.6x** | 1,086 ms | 413 ms | **2.6x** |

### v1 Highlights (AMD64 JIT, best-of-3)

| Benchmark              | Interp (ms) | JIT (ms) | Speedup |
|------------------------|-------------|----------|---------|
| Integer Arithmetic     |         466 |       23 |  20.3x |
| Loop with Array Access |      18,284 |    1,131 |  16.2x |
| Function Calls         |          20 |        1 |  20.0x |
| Fibonacci (recursive)  |         777 |      243 |   3.2x |
| Sieve of Eratosthenes  |          74 |        5 |  14.8x |
| Nested Loops           |       1,152 |       65 |  17.7x |

### v2 Category Aggregates (AMD64 JIT, best-of-3)

| Category           | JIT (ms) | Interp (ms) | Speedup |
|--------------------|----------|-------------|---------|
| Integer ALU        |       10 |         139 |  13.9x |
| Branch & Control   |        2 |          72 |  36.0x |
| Memory Access      |        5 |         111 |  22.2x |
| Function Calls     |      164 |         455 |   2.8x |
| Big (64-bit)       |       18 |         135 |   7.5x |
| Byte Ops           |        7 |          88 |  12.6x |
| List Ops           |        4 |          23 |   5.8x |
| Mixed Workloads    |       25 |         382 |  15.3x |
| Type Conversions   |        6 |          51 |   8.5x |

### Notes

- AMD64 JIT: `comp-amd64.c`, System V AMD64 ABI, x86-64 native code
- ARM64 JIT: `comp-arm64.c`, AAPCS64 ABI, ARMv8-A native code
- 64-bit (big) operations are natively JIT-compiled on both architectures
- AMD64 achieves highest speedup ratios due to efficient x86-64 instruction encoding
- JIT correctness: 181/181 tests pass on all three platforms
