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

## AMD64 JIT Compiler Benchmarks

**Platform:** Linux x86_64, Intel (16 cores, 2.1 GHz, 8 MB cache), 21 GB RAM
**Build:** Headless, JIT-compiled (comp-amd64.c, 2418 lines)
**Date:** February 2026
**Benchmark suite:** jitbench2.b (30 benchmarks, 9 categories)
**Runs:** 3 (median values reported)

### Results

The AMD64 JIT compiler translates Dis VM bytecode directly to x86-64 machine code at module load time. All 30 benchmarks use 1,000,000 iterations (ITER) or 100,000 iterations (SMALL) unless otherwise noted.

#### Category 1: Integer ALU (1M iterations)

| Benchmark | Time (ms) | Description |
|-----------|-----------|-------------|
| ADD/SUB chain | 25 | Chained add/subtract with shifts |
| MUL/DIV/MOD | 3 | Multiply, divide, modulo (100K iter) |
| Bitwise ops | 32 | XOR, OR, AND, shift combinations |
| Shift ops | 25 | Rotate-left emulation via shifts |
| Mixed ALU | 35 | Combined arithmetic and bitwise |

#### Category 2: Branch & Control Flow

| Benchmark | Time (ms) | Description |
|-----------|-----------|-------------|
| Simple branch | 22 | Alternating if/else (1M iter) |
| Compare chain | 29 | 4-way if/else if chain (1M iter) |
| Nested branches | 3 | Two-level nested conditionals (100K iter) |
| Loop countdown | 13 | Nested while loops (1M total iterations) |

#### Category 3: Memory Access (array operations)

| Benchmark | Time (ms) | Description |
|-----------|-----------|-------------|
| Sequential read | 17 | 1000-element array, 1000 passes |
| Sequential write | 18 | 1000-element array, 1000 passes |
| Stride access | 32 | 1024-element array, stride 1/2/4/8, 1000 passes |
| Small array hot | 30 | 16-element array, 100K passes |

#### Category 4: Function Calls

| Benchmark | Time (ms) | Description |
|-----------|-----------|-------------|
| Simple call | 15 | 1M calls to trivial function |
| Recursive fib | 414 | fib(25) x 50 (~2.5M recursive calls) |
| Mutual recursion | 21 | is_even/is_odd mutual recursion (100K iter) |
| Deep call chain | 20 | 10-deep A/B call chain (100K iter) |

#### Category 5: Big (64-bit) Operations (1M iterations)

| Benchmark | Time (ms) | Description |
|-----------|-----------|-------------|
| Big add/sub | 19 | 64-bit integer add/subtract |
| Big bitwise | 29 | 64-bit XOR, OR, AND |
| Big shifts | 32 | 64-bit rotate-left emulation |
| Big comparisons | 32 | 64-bit less-than, equal, not-equal |

#### Category 6: Byte Operations

| Benchmark | Time (ms) | Description |
|-----------|-----------|-------------|
| Byte arithmetic | 26 | Byte add, XOR, AND (1M iter) |
| Byte array | 52 | 256-element byte array, 10K passes |

#### Category 7: List Operations

| Benchmark | Time (ms) | Description |
|-----------|-----------|-------------|
| List build | 4 | Build 100-element list, 1000 times |
| List traverse | 19 | Traverse 1000-element list, 1000 times |

#### Category 8: Mixed Workloads

| Benchmark | Time (ms) | Description |
|-----------|-----------|-------------|
| Sieve of Eratosthenes | 65 | Primes up to 10,000 x 100 |
| Matrix multiply | 130 | 32x32 matrix multiply x 100 |
| Bubble sort | 73 | Sort 500 elements (reverse) x 10 |
| Binary search | 48 | Search 10,000-element array (100K searches) |

#### Category 9: Type Conversions (1M iterations)

| Benchmark | Time (ms) | Description |
|-----------|-----------|-------------|
| int <-> big | 22 | int to big and back |
| int <-> byte | 22 | int to byte and back |

### Summary

| Metric | Value |
|--------|-------|
| **Total time (30 benchmarks)** | **~1330 ms** |
| **Fastest benchmark** | MUL/DIV/MOD (3 ms) |
| **Slowest benchmark** | Recursive fib (414 ms) |
| **ALU throughput** | ~40M ops/sec (mixed) |
| **Memory access** | 17-32 ms for 1M element accesses |
| **Function call overhead** | ~15 ns/call (simple) |

### Variance

Three consecutive runs showed consistent timing (total: 1305, 1344, 1338 ms), with individual benchmark variance typically under 10%.

### Notes

- The AMD64 JIT (comp-amd64.c) is the only fully implemented JIT backend (2,418 lines)
- ARM64 currently falls back to interpreter mode (stub JIT)
- The JIT compiles Dis bytecode to native x86-64 at module load time using System V AMD64 ABI
- 64-bit (big) operations are natively JIT-compiled on AMD64, not emulated
- All results verified correct (result values match across runs)
