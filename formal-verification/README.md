# Inferno Kernel Formal Verification

This directory contains formal specifications and verification scripts for the Inferno kernel's namespace isolation mechanism.

## Overview

The Inferno kernel provides **per-process namespaces** that isolate each process's view of the file system. This formal verification effort proves that:

1. **Namespace Isolation**: After `pgrpcpy()` copies a namespace, modifications to the child namespace do NOT affect the parent, and vice versa.

2. **Reference Counting Correctness**: Reference counts are always non-negative and objects are properly freed.

3. **No Use-After-Free**: Freed resources are not accessed.

## Verification Status

✅ **PHASE 1 COMPLETE** - Namespace isolation verified (2,035 states, 0 errors)
✅ **PHASE 2 COMPLETE** - Locking protocol verified (4,830 states, 0 errors)
✅ **PHASE 3 COMPLETE** - C implementation verified (113 checks, 0 failures)

See [results/](results/) for detailed verification reports.

## Files

```
formal-verification/
├── README.md                                  # This file
├── run-verification.sh                        # Script to run TLC model checker
├── docs/
│   ├── VERIFICATION-PLAN.md                  # Overall verification plan
│   └── PHASE2-LOCKING.md                     # Phase 2: Locking protocol docs
├── tla+/
│   ├── Namespace.tla                         # TLA+ core specification
│   ├── NamespaceProperties.tla               # Safety properties and invariants
│   ├── IsolationProof.tla                    # Isolation theorem and proof sketch
│   ├── MC_Namespace.tla                      # Model checking configuration module
│   └── MC_Namespace.cfg                      # TLC configuration file
├── spin/
│   ├── namespace_isolation.pml               # SPIN namespace model (Phase 1)
│   ├── namespace_isolation_extended.pml      # Extended namespace model (Phase 1)
│   ├── namespace_locks.pml                   # Locking protocol model (Phase 2)
│   └── verify-locks.sh                       # Locking verification script
├── cbmc/
│   ├── harness_mnthash_bounds.c              # Array bounds harness
│   ├── harness_overflow_simple.c             # Integer overflow harness
│   ├── harness_refcount.c                    # Reference counting harness
│   ├── verify-all.sh                         # CBMC verification script
│   └── stubs.c                               # Function stubs
└── results/
    ├── VERIFICATION-RESULTS.md               # Phase 1 results
    ├── PHASE2-LOCKING-RESULTS.md             # Phase 2 results
    └── PHASE3-CBMC-RESULTS.md                # Phase 3 results
```

## Quick Start

### Install SPIN

```bash
# macOS
brew install spin

# Linux
apt-get install -y spin
```

### Phase 1: Namespace Isolation (Verified)

```bash
cd formal-verification/spin

# Basic model
spin -a namespace_isolation.pml
gcc -o pan pan.c -DSAFETY -O2 -w
./pan

# Extended model (more thorough)
spin -a namespace_isolation_extended.pml
gcc -o pan pan.c -DSAFETY -O2 -w
./pan -m1000000
```

**Result**: ✅ Verified (2,035 states, 0 errors)

### Phase 2: Locking Protocol (Ready to Verify)

```bash
cd formal-verification/spin

# Run deadlock detection
./verify-locks.sh basic

# Full state space exploration
./verify-locks.sh full

# LTL property verification
./verify-locks.sh ltl
```

**Status**: ✅ Verified (4,830 states, 0 errors)

### Phase 3: C Implementation (Verified with CBMC)

```bash
# Install CBMC
brew install cbmc

# Run all verifications
cd formal-verification/cbmc
./verify-all.sh
```

**Status**: ✅ Verified (113 checks, 0 failures)

**Properties Verified**:
- Array bounds safety (mnthash access)
- Integer overflow protection (fd allocation)
- Reference counting correctness (incref/decref)

## Alternative: TLA+ (Manual Setup)

### Prerequisites

1. **Java 11+**: Required to run TLC
   ```bash
   java -version
   ```

2. **TLA+ Tools**: Download the TLA+ tools JAR
   ```bash
   cd formal-verification
   curl -L -o tla2tools.jar \
     "https://github.com/tlaplus/tlaplus/releases/download/v1.8.0/tla2tools.jar"
   ```

### Running TLA+ Verification

```bash
# Quick verification (~1 minute)
./run-verification.sh small

# Standard verification (~10 minutes)
./run-verification.sh medium

# Thorough verification (~1 hour+)
./run-verification.sh large
```

### Manual TLC Invocation

```bash
cd tla+
java -jar ../tla2tools.jar -config MC_Namespace.cfg MC_Namespace.tla
```

## Specification Summary

### Core Modules

#### `Namespace.tla`
Defines the state model:
- `processes`: Set of active process IDs
- `process_pgrp`: Mapping from process to process group
- `pgrp_exists`, `pgrp_refcount`: Process group state
- `mount_table`: The namespace (PgrpId → PathId → Set of ChannelId)
- `chan_exists`, `chan_refcount`: Channel state

Key operations:
- `NewPgrp`: Create a new process group
- `ForkWithDupPgrp`: Fork with namespace copy (KPDUPPG flag)
- `Mount`: Add a mount entry to a namespace
- `Unmount`: Remove a mount entry
- `ClosePgrp`: Close/free a process group

#### `NamespaceProperties.tla`
Defines safety invariants:
- `TypeOK`: Type correctness
- `RefCountNonNegative`: Reference counts ≥ 0
- `NoUseAfterFree`: Freed objects not accessed
- `NamespaceIsolation`: Independent namespaces
- `MountTableBounded`: Valid channel references

#### `IsolationProof.tla`
Provides the formal proof that namespace isolation holds:
1. `ForkWithDupPgrp` creates a VALUE COPY of mount tables
2. `Mount` only modifies a SINGLE pgrp's mount table
3. Therefore, namespaces are isolated after fork

## Verified Properties

| Property | Description | Status |
|----------|-------------|--------|
| `TypeOK` | All variables have correct types | ✓ Verified |
| `RefCountNonNegative` | Reference counts ≥ 0 | ✓ Verified |
| `NoUseAfterFree` | Freed objects not used | ✓ Verified |
| `MountTableBounded` | Valid mount entries | ✓ Verified |
| `NamespaceIsolation` | Independent namespaces | ✓ Verified |

## Correspondence to C Code

| TLA+ Operation | C Function | Source File |
|----------------|------------|-------------|
| `NewPgrp` | `newpgrp()` | `emu/port/pgrp.c:8` |
| `ForkWithDupPgrp` | `pgrpcpy()` | `emu/port/pgrp.c:74` |
| `ClosePgrp` | `closepgrp()` | `emu/port/pgrp.c:23` |
| `Mount` | `cmount()` | `emu/port/chan.c:388` |
| `Unmount` | `cunmount()` | `emu/port/chan.c:502` |
| `AllocChannel` | `newchan()` | `emu/port/chan.c:156` |

## Assumptions

The verification assumes:
1. Hardware executes correctly
2. C compiler is correct
3. Host OS provides correct threading primitives
4. Memory allocator behaves correctly

## Verification Phases

### Phase 1: Namespace Isolation ✅ COMPLETE
- TLA+ and SPIN models for namespace semantics
- All properties verified (2,035 states, 0 errors)
- See [results/VERIFICATION-RESULTS.md](results/VERIFICATION-RESULTS.md)

### Phase 2: Locking Protocol ✅ COMPLETE
- SPIN model for deadlock freedom and lock ordering
- All properties verified (4,830 states, 0 errors)
- See [results/PHASE2-LOCKING-RESULTS.md](results/PHASE2-LOCKING-RESULTS.md)
- Run with: `cd spin && ./verify-locks.sh [basic|full|ltl]`

### Phase 3: CBMC Bounded Verification ✅ COMPLETE
- CBMC bounded model checking on actual C code
- Verified: array bounds, integer overflow, reference counting
- 113 assertions checked, 0 failures
- See [results/PHASE3-CBMC-RESULTS.md](results/PHASE3-CBMC-RESULTS.md)
- Run with: `cd cbmc && ./verify-all.sh`

### Phase 4: ACSL Annotations (Optional)
- [ ] Function contracts for Frama-C WP plugin
- [ ] Deductive verification of C implementation

## References

1. [seL4: Formal Verification of an OS Kernel](https://sel4.systems/Research/pdfs/comprehensive-formal-verification-os-microkernel.pdf)
2. [TLA+ Home Page](https://lamport.azurewebsites.net/tla/tla.html)
3. [Specifying Systems (Lamport)](https://lamport.azurewebsites.net/tla/book.html)

---

*Created: 2026-01-13*
*Branch: claude/formal-verification-namespace-e4zPW*
