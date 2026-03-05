# GoDis Open Items Implementation Plan

## Overview

Four feasible improvements to the Go-to-Dis compiler, ordered by complexity:

1. Stack zeroing (small)
2. Float formatting with precision (moderate)
3. Native hash maps (large)
4. Separate compilation (large, architectural)

---

## Item 1: Stack Zeroing (Non-pointer frame slot initialization)

**Problem:** Non-pointer frame slots contain garbage from previous calls. The Dis VM only auto-initializes pointer slots (to H/-1) based on the type descriptor. Non-pointer word/float slots are uninitialized.

**Fix:** At function entry, emit explicit `MOVW $0, FP(slot)` for every non-pointer word slot and zero-init for float slots. This happens after the function prologue, before any user code.

**Files to modify:**
- `compiler/lower.go` — in the function lowering entry point, after frame setup, emit zero instructions for all non-pointer slots

**Scope:** ~30-50 lines. Low risk.

---

## Item 2: Float Formatting with Precision Control

**Problem:** `%f`, `%g`, `%e` all emit bare `CVTFC` which has no precision parameter. Format specs like `%.2f` are parsed but precision is ignored.

**Approach:** Synthesize precision-aware formatting in Dis instructions:
1. Parse precision from format string at compile time (already partially done)
2. For `%f` with precision N:
   - Separate integer and fractional parts using `CVTFW` (float→word truncation) and subtraction
   - Multiply fractional part by 10^N, truncate to integer
   - Convert both parts via `CVTWC` (word→string), concatenate with "."
   - Handle rounding: add 0.5 * 10^-N before truncation
   - Handle negative numbers and zero padding
3. For `%e`/`%g`: similar approach with exponent extraction
4. Fall back to bare `CVTFC` when no precision specified (preserves current behavior)

**Files to modify:**
- `compiler/lower.go` — float format verb handling in Sprintf/Printf lowering
- `compiler/lower_stdlib.go` — `strconv.FormatFloat` / `AppendFloat` interception

**Scope:** ~200-400 lines. Medium risk — needs careful numeric edge cases.

---

## Item 3: Native Hash Maps

**Problem:** Maps use O(n) linear scan with parallel arrays. Every lookup, insert, delete scans all keys.

**Approach:** Open-addressing hash table with linear probing, synthesized entirely in Dis instructions (no VM changes needed).

### Data Structure

New map wrapper struct (40 bytes):
```
Offset  0: PTR  keys array        (bucket storage for keys)
Offset  8: PTR  values array      (bucket storage for values)
Offset 16: PTR  state array       (byte array: 0=empty, 1=occupied, 2=deleted)
Offset 24: WORD count             (number of live entries)
Offset 32: WORD capacity          (number of buckets, always power of 2)
```

### Hash Function

FNV-1a for both string and integer keys, synthesized in Dis arithmetic:
- String keys: iterate bytes with `INDC`, XOR + multiply per byte
- Integer keys: direct bit mixing (multiply, shift, XOR)
- Bucket index: `hash & (capacity - 1)` (power-of-2 masking)

### Operations

**Lookup:** Hash key → probe from bucket index → linear probe until empty or found → O(1) average
**Insert:** Hash key → probe for empty/deleted/matching slot → insert → rehash if load > 75%
**Delete:** Mark slot as "deleted" (tombstone) → decrement count
**Rehash:** Allocate 2x capacity, re-insert all occupied entries
**Range:** Linear scan of state array, skip empty/deleted

### Migration Strategy

- Replace `lowerMakeMap`, `lowerMapUpdate`, `lowerMapLookup`, `lowerMapDelete`, `lowerNext` (map branch)
- Update `makeMapTD` for new 40-byte struct layout
- All existing map tests must continue passing
- Add new tests for hash collision behavior and large maps

**Files to modify:**
- `compiler/lower.go` — map operation lowering (~500-800 lines rewrite of map section)
- `compiler/compiler_test.go` — new test cases for hash maps

**Scope:** ~800-1200 lines changed. High complexity but well-contained.

---

## Item 4: Separate Compilation (Multi-module output)

**Problem:** All packages are inlined into a single .dis file. No incremental compilation.

**Approach:** Leverage existing Dis IMFRAME/IMCALL infrastructure (already used for Sys module) to link separately-compiled package modules.

### Design

1. **Per-package .dis files:** Each package compiles to its own .dis module
2. **Export table:** Non-main packages export public functions via `Links` entries
3. **Import via LDT:** Importing packages reference exported functions via LDT entries and IMFRAME/IMCALL
4. **Main module:** Still the entry point; its LDT lists all imported package modules
5. **Shared type registry:** Type tags for interfaces serialized to a sidecar file or embedded in module metadata

### Compilation Modes

- `godis -single file.go` — current behavior (monolithic, default)
- `godis -pkg ./mypkg/` — compile package to .dis with export table
- `godis -link main.go` — compile main, reference pre-compiled package .dis files

### Changes Required

- `compiler/compiler.go` — new compilation mode for package-only output
- `compiler/lower.go` — cross-package calls emit IMFRAME/IMCALL instead of CALL
- `compiler/compiler.go` — export table generation (Links entries for public functions)
- `cmd/godis/main.go` — new CLI flags

**Scope:** ~500-800 lines. Architecturally significant but builds on existing LDT infrastructure.

---

## Implementation Order

1. **Stack zeroing** — quick win, improves correctness
2. **Float formatting** — moderate effort, improves stdlib fidelity
3. **Native hash maps** — biggest performance improvement
4. **Separate compilation** — architectural improvement, can be done last

Each item will be committed independently so progress is incremental.
