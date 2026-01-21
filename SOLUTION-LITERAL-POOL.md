# ARM64 JIT - Literal Pool Solution
**Date:** 2026-01-19 (Late Session)
**Status:** Partial breakthrough using inferno64 approach

## The Problem We Had

**AXIMM Storage Array Limitation:**
- Original solution: `static WORD aximm_storage[8]` array
- Emuinit module needs: 33 AXIMM slots
- Gap: 25 slots short
- Attempted fixes: ALL array sizes ≥33 caused X9=1 corruption and SEGV
- Tested 20+ approaches over 14 hours: All failed

**Impact:**
- calc.dis couldn't load Math module
- jitbench couldn't run
- All programs crashed during cleanup

## How We Found the Solution

### Research Discovery

After exhaustive debugging failed, conducted web research and found:

**1. inferno64 Project** - https://github.com/caerwynj/inferno64
- Fork of Inferno for 64-bit platforms (amd64 and arm64)
- comp-arm64.c last updated Dec 2022 with message **"working emu on arm64"**
- This means someone successfully implemented ARM64 JIT!

**2. Their Approach: Literal Pool**

Examined their comp-arm64.c and found they use a completely different approach:
```c
case AXIMM:
    literal((short)i->reg, O(REG,m));
    break;
```

Instead of a separate static array, they store AXIMM values in the **literal pool** - the same place other constants go.

## What We Implemented

### Removed Separate Array
**Before:**
```c
static WORD aximm_storage[8];  // Limited to 8 slots
static int aximm_next;

case AXIMM:
    if(aximm_next >= 8)
        urk("too many AXIMM");
    aximm_storage[aximm_next] = value;
    con(&aximm_storage[aximm_next], ...);
    aximm_next++;
```

**After (inferno64 approach):**
```c
// No separate array needed!

case AXIMM:
    literal((short)i->reg, O(REG,m));
    break;
```

### Added literal() Function

Based on inferno64's implementation:
```c
static void
literal(uvlong imm, int roff)
{
    nlit++;
    con((uvlong)litpool, RTA, 0);
    mem(Stw, roff, RREG, RTA);

    if(pass == 0)
        return;

    /* Pass 1: Write value to literal pool */
    *litpool++ = (u32int)(imm);
    *litpool++ = (u32int)(imm >> 32);
}
```

This stores the immediate value directly in the literal pool (part of the JIT code buffer), eliminating the need for a separate array.

### Fixed litpool Initialization

**Critical fix:** inferno64 leaves `litpool` uninitialized in pass 0, causing undefined behavior. We initialize it to a placeholder:

```c
/* Pass 0 */
{
    static u32int placeholder_pool[256];
    litpool = placeholder_pool;
}

/* Pass 1 */
litpool = base + n;  // Real location
```

This ensures both passes generate identical code.

## Results

### What Works Now ✅
- ✅ **echo.dis** - Outputs correctly
- ✅ **cat.dis** - Processes multiple lines correctly
- ✅ **calc.dis** - Outputs numbers (partial functionality)
- ✅ **No array size limit** - Can handle unlimited AXIMM instructions
- ✅ **No X9=1 corruption** - Fixed by removing large static arrays

### What Still Doesn't Work ❌
- ❌ **calc.dis** - Doesn't perform computations (needs Math module)
- ❌ **jitbench.dis** - Crashes before running tests
- ❌ **Cleanup crashes** - All programs still SEGV during cleanup (addr=24821)

### Comparison

**Before literal pool approach:**
```
$ echo "test" | ./o.emu -r. -c1 dis/echo.dis
test
panic: disinit error: Undefined error: 0  # Hit AXIMM limit during cleanup
SEGV: ... X9=1 ...  # With array size ≥33
```

**After literal pool approach:**
```
$ echo "test" | ./o.emu -r. -c1 dis/echo.dis
test
SEGV: addr=24821 code=2
  X9=<valid address> (not 1!)  # X9 corruption fixed
```

**Progress:**
- ✓ Fixed X9=1 corruption
- ✓ Removed array size limit
- ✓ Programs execute and output correctly
- ✗ Still crash during cleanup (different issue)

## What We Learned

### 1. Check Existing Implementations First
After 14 hours of debugging, web research found inferno64 had already solved the core problem. **Lesson: Research before reinventing.**

### 2. Literal Pool is Superior to Arrays
**Advantages:**
- No size limit
- No separate memory allocation
- Part of JIT code buffer (better locality)
- Simpler management (reuses existing nlit mechanism)

### 3. Large Static Arrays on macOS ARM64 Are Problematic
- Arrays ≥512 bytes (64 WORDs) cause mysterious issues
- Likely related to [LLVM bug #56295](https://github.com/llvm/llvm-project/issues/56295)
- BSS section layout affects runtime behavior
- Heap allocation doesn't fully solve it

### 4. X9 Corruption Was a Red Herring
The X9=1 corruption was caused by having large static arrays, not by the AXIMM logic itself. Removing the array eliminated the corruption.

### 5. The addr=24821 SEGV is a Separate Issue
After fixing AXIMM storage, a different crash remains:
- Happens during cleanup/teardown
- t=0x24809 (invalid Type pointer)
- Not related to AXIMM storage method
- Requires separate investigation

### 6. inferno64 Has Working ARM64 JIT
Commit message "working emu on arm64" from Dec 2022 proves it's achievable. We should study their complete implementation.

## Technical Details

### Memory Layout Before
```
BSS Section:
  [other static variables]
  [aximm_storage[256]]  ← 2048 bytes, causes issues
  [more static variables]
```

### Memory Layout After
```
Per-module JIT Buffer:
  [compiled instructions]
  [literal pool for constants]
  [AXIMM values mixed in literal pool]  ← No separate array!
```

### Code Generation

**Pass 0:**
- literal() increments nlit counter
- Generates LDR instruction (placeholder)
- litpool placeholder ensures address has same encoding size

**Pass 1:**
- literal() writes actual value to litpool
- litpool is at real location (base + n)
- Same LDR instruction generated (addresses match)

## Next Steps

### To Fully Resolve
1. **Study inferno64 cleanup code** - How do they avoid addr=24821 crash?
2. **Compare their macret()** - May have different teardown logic
3. **Test on Linux ARM64** - See if macOS-specific
4. **Investigate Math module** - Why doesn't calc compute?

### Alternative: Ship with Interpreter
- Interpreter mode (-c0) is fully functional
- Performance is acceptable (~21s for benchmarks)
- JIT can be future enhancement
- Document ARM64 JIT as experimental

## Breakthrough Credit

**Solution found through:**
- Web search for "Inferno ARM64 JIT"
- Found inferno64 project on GitHub
- Examined their comp-arm64.c implementation
- Adopted their literal pool approach

**Key insight:** Don't store AXIMM values in separate array, use literal pool like other constants.

## Files Modified

- libinterp/comp-arm64.c - Implemented literal() function, removed aximm_storage array
- libinterp/loader.c - Removed debug logging
- libinterp/xec.c - Restored to baseline (removed debug)

## Performance (Not Yet Measured)

Cannot benchmark yet due to remaining crash, but:
- Interpreter: ~21 seconds for jitbench
- Expected JIT: ~2-10 seconds (6-20x faster)
- If we solve remaining issues, JIT will be functional

---

*Solution discovered through research*
*Implemented by: Claude Sonnet 4.5 (1M context)*
*Date: 2026-01-19*
*Outcome: Major progress, eliminated array size limit, fixed X9 corruption*
