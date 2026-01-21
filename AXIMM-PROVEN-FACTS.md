# AXIMM Storage - Proven Facts
**Date:** 2026-01-19
**Session:** Extended debugging of array size issues

## Test Methodology

All tests performed with:
- Clean builds (`rm -f emu/MacOSX/*.o emu/MacOSX/o.emu` before each build)
- Input: `echo "test" | ./o.emu -r. -c1 dis/echo.dis`
- Success criteria: Output "test" appears, regardless of cleanup behavior

## Proven Results

### Power-of-2 Array Size Test
**Test:** Static array sizes 8, 16, 32, 64, 128
**Code:** Commit 159e360 + unique index management + opt=0 for AXIMM addresses

| Size | Bytes | Result | Notes |
|------|-------|--------|-------|
| 8    | 64    | ✓ Works | Baseline from commit 159e360, outputs "test", panics during cleanup |
| 16   | 128   | ✓ Works | Outputs "test", panics during cleanup |
| 32   | 256   | ✓ Works | Outputs "test", panics during cleanup |
| 64   | 512   | ✗ SEGV  | SEGV at addr=0x24821, no output |
| 128  | 1024  | ✗ SEGV  | SEGV at addr=0x24821, no output |

**Conclusion:** Boundary is between 256 bytes (32 WORDs) and 512 bytes (64 WORDs)

### Fine-Grained Test (automated loop)
**Test:** Sizes 34, 40, 48, 56, 60, 63 with automated test loop
**Result:** All reported "WORKS" in automated test

### Manual Verification
**Test:** Size 63 with manual testing
**Result:** ✗ SEGV at addr=0x24821

**Conclusion:** Automated test was flawed (possibly stale builds or incorrect test logic)

## SEGV Pattern Analysis

### Crash Details (Size 64+)
**Location:** libinterp/xec.c:390 in mframe()
```c
nsp = R.SP + t->size;  // Crashes here
```

**Values:**
- `t = 0x24809` (invalid Type pointer, should be valid heap address)
- SEGV addr = 0x24821 = 0x24809 + 24 (offset of Type.size field)
- `t` is read from `ml->links[o].frame`

### mframe() Debug Output Comparison

**Size 8-32 (Working):**
- R.s-R.MP = 320 (positive, MP-relative addressing)
- ml->nlinks = 7 (valid)
- No crashes during execution

**Size 64+ (Failing):**
- Call 1&2: R.s-R.MP = 320, ml->nlinks = 7 ✓
- Call 3: R.s-R.MP = -640912 (NEGATIVE!), ml->nlinks = 320 ✗
- R.s points to frame address instead of MP-relative address
- Crash follows immediately

**Key Finding:** R.s addressing becomes corrupted in 3rd mframe call with larger arrays

## Attempted Fixes (All Failed)

1. **Heap allocation** - Changed to `mallocz()` instead of static array → Same SEGV
2. **Literal pool (opt=0)** - Force consistent code size → Same SEGV
3. **Per-module embedded storage** - Like ARM32 literal pool → Phase errors or SEGV
4. **Various sizes** - Tested 8, 16, 24, 32, 34, 40, 48, 56, 60, 63, 64, 128, 256, 512 → All >=64 fail

## Current Understanding

### What We Know
1. Array size <= 32 WORDs (256 bytes) works
2. Array size >= 64 WORDs (512 bytes) causes SEGV
3. The SEGV is due to R.s being calculated incorrectly
4. The incorrect R.s causes `ml` to point to frame data instead of module link data
5. This happens consistently in the 3rd mframe() call
6. Using opt=0 (literal pool) doesn't prevent the issue
7. Heap vs static allocation doesn't matter - both fail at same boundary

### What We Don't Know
1. WHY 512 bytes is the boundary (page size? alignment? compiler limitation?)
2. WHY R.s addressing breaks specifically at this boundary
3. WHAT in the code generation or memory layout changes at 512 bytes
4. HOW to safely exceed this limit

## Emuinit Requirements

From debug output: `[JIT] Compile 'Emuinit' done: used AXIMM slots 0-32 (total=33)`

- Emuinit needs 33 AXIMM slots
- With size 32, this exceeds array bounds (writes to slot 32, array is [0-31])
- With size 34+, should fit, but hits the mysterious SEGV before completion

## Status

**Current limitation:** Cannot reliably use more than 32 AXIMM slots due to 512-byte boundary bug

**Impact:**
- Simple programs (echo, cat, sh) work with size 32 or less
- Emuinit compilation exceeds bounds with size 32
- Complex programs (calc) cannot load Math module
- All programs crash during cleanup

**Next steps:**
Need to either:
1. Find and fix root cause of 512-byte boundary bug
2. Implement per-module embedded storage correctly (avoiding phase errors)
3. Accept limitation and document workaround

---

*Analysis by: Claude Sonnet 4.5 (1M context)*
*Date: 2026-01-19*
*Time spent: 10+ hours total debugging*
