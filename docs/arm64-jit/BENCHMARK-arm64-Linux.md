# ARM64 JIT Benchmark Results — Linux

## Platform
- **Hardware:** NVIDIA Jetson AGX Orin
- **CPU:** ARMv8 Processor rev 1 (v8l) — ARM Cortex-A78AE
- **OS:** Linux 5.15.148-tegra (aarch64)
- **Date:** 2026-02-06

## Results (3 runs)

### Per-benchmark breakdown (Run 3)

| Benchmark              | Interp (ms) | JIT (ms) | Speedup |
|------------------------|-------------|----------|---------|
| Integer Arithmetic     |         863 |      442 |   1.95x |
| Loop with Array Access |      33,738 |    7,603 |   4.43x |
| Function Calls         |          38 |       35 |   1.08x |
| Fibonacci              |       1,680 |      606 |   2.77x |
| Sieve of Eratosthenes  |         136 |       61 |   2.22x |
| Nested Loops           |       1,924 |    1,751 |   1.09x |

### Totals

| Run | Interp (ms) | JIT (ms) | Speedup |
|-----|-------------|----------|---------|
| 1   |      38,347 |   10,509 |   3.64x |
| 2   |      38,374 |   10,493 |   3.65x |
| 3   |      38,380 |   10,498 |   3.65x |
| **Avg** | **38,367** | **10,500** | **3.65x** |

## Notes
- Variation across runs: < 0.3%
- System was idle during benchmark (no competing workloads)
- Benchmark: `dis/jitbench.dis` (source: `appl/cmd/jitbench.b`)
- Interpreter: `emu -c0`, JIT: `emu -c1`
- Produced by: `bash scratchpad/bench.sh 3`
