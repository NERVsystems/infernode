# AXIMM Storage Debugging Session
**Date:** 2026-01-19
**Branch:** feature/jit-64bit
**Status:** Investigating SEGV with larger AXIMM storage

## Problem Statement

The AXIMM storage array size of 8 is too small:
- Emuinit module alone needs 33 AXIMM slots
- With size 8, only first module can partially compile
- calc.dis fails because Math module can't load

## Attempted Solutions & Results

### Test 1: Larger Static Array
**Approach:** Increase `static WORD aximm_storage[N]`
**Tested sizes:** 8, 16, 24, 32, 34, 36, 40, 48, 64, 128, 256

**Results:**
- Size 8: ✓ Works (baseline from commit 159e360)
- Size 16-32: Reports "WORKS" in binary search but actually still has issues
- Size 34+: Consistent SEGV at addr=0x24821

### Test 2: Heap Allocation
**Approach:** `static WORD *aximm_storage = nil;` + `mallocz()` in preamble()
**Result:** SEGV at addr=0x24821 (same as large static arrays)

### Test 3: Unique Index Management
**Approach:** Don't reset `aximm_next=0` per module; each module gets unique slots
**Result:** SEGV at addr=0x24821

### Test 4: Literal Pool for Addresses
**Approach:** Use `con(..., RA0, 0)` to force consistent code size
**Result:** SEGV at addr=0x24821

## The SEGV Pattern

**Crash location:** libinterp/xec.c:390
```c
nsp = R.SP + t->size;  // Crash here
```

**Values at crash:**
- `t = 0x24809` (invalid Type pointer - should be a valid heap address)
- SEGV addr = 0x24821 = 0x24809 + 24 (offset of Type.size field)
- `t` is read from `ml->links[o].frame`

**Key observation:** The value 0x24809 (149,513 decimal) is CONSISTENT across all failure modes

## Root Cause Hypothesis

When `aximm_storage` is at certain addresses (larger static arrays or heap):
1. Something causes `ml->links[o].frame` to contain 0x24809 instead of valid Type pointer
2. This suggests either:
   - `ml` pointer is wrong
   - `o` index is wrong
   - `ml->links[o].frame` was never properly initialized
   - Memory corruption overwrites the frame pointer

## Why Size 8 Works

With static array size 8 (~64 bytes in BSS), the array is located at one memory address. With larger sizes or heap allocation, it's at different addresses. The specific address seems to affect whether the code works.

**Theories:**
1. **Address encoding**: Different addresses generate different MOVZ/MOVK sequences, causing phase errors
2. **Memory layout**: Larger BSS pushes other static variables to different locations
3. **Alignment**: Different array sizes have different alignment, affecting subsequent data
4. **Compiler/linker bug**: macOS clang or linker issue with large BSS sections

## What We Know For Sure

- ✓ JIT execution works correctly (echo, cat produce correct output)
- ✓ AXIMM storage concept is correct (fixes R.t corruption)
- ✗ Cannot scale beyond ~8 slots without SEGV
- ✗ calc.dis cannot work (needs Math module which won't load)

## Next Steps to Try

1. **Force consistent code generation**
   - Verify opt=0 is actually being used
   - Check if literal pool has size limits
   - Ensure flushcon() is called appropriately

2. **Debug the garbage value 0x24809**
   - Add instrumentation to see where this value comes from
   - Check if it's related to array base address
   - See if it's an offset being misinterpreted as pointer

3. **Alternative storage approaches**
   - Per-module allocation in JIT code buffer (like literal pool)
   - Use memory-mapped region separate from BSS
   - Pack values differently (avoid WORD array)

4. **Investigate compiler/linker**
   - Check if optimization level affects it
   - Try different compiler flags
   - Examine generated assembly for differences

## Current Blocked State

Every approach to increase AXIMM storage beyond 8 slots results in identical SEGV at 0x24821. Need to understand the root cause of why this specific value appears before proceeding.

---

*Debugging by: Claude Sonnet 4.5 (1M context)*
*Time spent: 4+ hours on AXIMM scaling issue*
*Status: Blocked pending root cause analysis*
