# RISC vs CISC: A Comparison Grounded in Infernode's JIT Compilers

This document compares RISC and CISC architectures using evidence from Infernode's
ARM64 (RISC) and AMD64 (CISC) JIT compilers in `libinterp/comp-arm64.c` and
`libinterp/comp-amd64.c`.

## Core Architectural Differences

| Aspect | RISC (ARM64) | CISC (AMD64/x86-64) |
|--------|-------------|---------------------|
| Instruction size | Fixed 4 bytes | Variable 1-15 bytes |
| Operand model | Load/store (3-register) | Register-memory (2-operand) |
| General-purpose registers | 32, orthogonal | 16, some with implicit roles |
| Instruction encoding | Uniform bit fields | REX + opcode + ModRM + SIB + disp |
| Addressing modes | ~16 patterns | 256+ theoretical (ModRM × SIB) |

## Evidence From Infernode's JIT Compilers

### Integer Division

ARM64 — one dedicated instruction, any register combination:
```c
SDIV_REG(RA0, RA0, RA1);   // RA0 = RA0 / RA1
```

AMD64 — three instructions, implicit register constraints:
```c
genb(REXW);                  // REX prefix for 64-bit
genb(Ocqo);                  // Sign-extend RAX → RDX:RAX
modrr(0xf7, RRTMP, 7);      // IDIV: RDX:RAX / RRTMP → RAX (quot), RDX (rem)
```

The dividend must be in RAX. RDX is clobbered. The quotient must be read from RAX.
None of this is configurable.

### Integer Modulo

ARM64 — two instructions, explicit operands:
```c
SDIV_REG(RA2, RA0, RA1);        // RA2 = quotient
MSUB_REG(RA0, RA1, RA2, RA0);   // RA0 = RA0 - (RA2 * RA1) = remainder
```

AMD64 — division plus a register swap:
```c
// After IDIV, remainder is in RDX but the JIT expects results in RAX
modrr(Oxchg, RAX, RDX);   // XCHG RAX, RDX
```

### Instruction Encoding

ARM64 — consistent bit manipulation:
```c
#define ADD_REG(Rd, Rn, Rm)   *code++ = (0x8B000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define SDIV_REG(Rd, Rn, Rm)  *code++ = (0x9AC00C00 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
```

Register fields are always at the same bit positions. Every instruction is one 32-bit word.

AMD64 — variable encoding with special cases:
```c
modrm(int inst, vlong disp, int rm, int r)
{
    // REX prefix needed for registers R8-R15 or 64-bit operand size
    // ModRM byte: [2-bit mod | 3-bit reg | 3-bit r/m]
    // SIB byte required when base register is RSP
    // Special case: RBP with mod=0 means [RIP+disp32], not [RBP]
    // Displacement: 0, 8-bit, or 32-bit depending on mod field
}
```

### Floating Point

ARM64 JIT compiles floating-point inline:
```c
case IMULF:
    opflld(i, Ldw, FA0);
    opflld(i, Ldw, FA1);
    FMUL_D(FA1, FA1, FA0);    // Single dedicated instruction
    opflst(i, Stw, FA1);
```

AMD64 JIT punts to the interpreter:
```c
case IMULF:
case IDIVF:
    punt(i, SRCOP|DSTOP|THREOP, optab[i->op]);   // Too complex to JIT
```

### Memory Allocation

The AMD64 JIT requires a custom allocator (`jitmalloc`) that places generated code
within 2GB of the text segment, because x86-64 rel32 branches have a ±2GB range limit.
The ARM64 JIT has no such constraint.

### Scaling Workarounds

When computing `index * 16` for module link table access:

AMD64 — SIB byte max scale is 8, so two LEAs are needed:
```c
// sizeof(Modl) = 16, but max SIB scale = 8
gen3(Olea, ...RCX*8...RAX);   // RAX = RAX + RCX*8
gen3(Olea, ...RCX*8...RAX);   // RAX = RAX + RCX*8 (total: RCX*16)
```

ARM64 — load constant and multiply:
```c
con(16, RCON);
MUL_REG(RA1, RA1, RCON);
ADD_REG(RA1, RA3, RA1);
```

## Where CISC Has Real Advantages

### 1. Code Density for Memory Operations

A single x86-64 instruction like `ADD [RBP+8], RAX` performs load + add + store.
ARM64 requires three separate instructions (LDR, ADD, STR). For tight loops with
good cache locality, fewer instructions mean less I-cache pressure.

### 2. Backward Compatibility

x86-64 can run code from 1978 (8086) through today. The variable-length encoding
allows extending the ISA without breaking existing binaries. ARM has made clean
breaks (ARMv7 → AArch64 is a different ISA).

### 3. Complex Addressing Modes

AMD64's `base + index*scale + displacement` encodes common array access patterns
in a single instruction. This genuinely reduces instruction count for
structure-of-arrays access patterns.

### 4. Atomic Read-Modify-Write

x86's `LOCK ADD [mem], reg` atomically reads, modifies, and writes in one
instruction. ARM64 needs a load-linked/store-conditional (LL/SC) loop that can
spuriously fail and must retry.

## Where RISC Has Real Advantages

### 1. Decode Simplicity and Power Efficiency

Fixed-size instructions make decoding trivial. Modern x86-64 chips spend
significant die area and power on their decode stage, internally translating CISC
instructions into RISC-like micro-ops (since Pentium Pro, 1995). ARM's simpler
decode enables better power efficiency, which is why ARM dominates mobile and is
competitive in servers (AWS Graviton, Apple Silicon).

### 2. Compiler and JIT Friendliness

This codebase demonstrates it directly: the ARM64 JIT compiles floating-point
operations inline while the AMD64 JIT punts them to the interpreter. The ARM64
JIT has fewer special cases, no implicit register constraints, and no encoding
workarounds. The uniform instruction format makes code generation straightforward
bit manipulation.

### 3. Register Availability

32 orthogonal GPRs vs 16 constrained GPRs means less register spilling on ARM64.
The AMD64 JIT must reserve RAX and RDX for multiply/divide operations, further
reducing the usable register set.

### 4. Scalability

Simpler decode logic means smaller cores, which means more cores per die at the
same power budget. This matters for throughput-oriented workloads.

## The Modern Reality

The RISC/CISC distinction has blurred. Modern x86-64 processors are effectively
RISC internally — they decode CISC instructions into micro-ops and execute those
on a wide, out-of-order backend. The ISA is a compatibility layer.

x86-64's real advantages today are ecosystem (software compatibility, existing
binaries) and microarchitectural investment (Intel and AMD spend billions on
execution engines). ARM64's real advantages are power efficiency, ISA cleanliness,
and scalability.

Neither architecture is categorically superior. But for anyone writing a JIT
compiler — as Infernode does — RISC is unambiguously easier to target correctly.

## Quantitative Summary From Infernode

| Metric | ARM64 JIT | AMD64 JIT |
|--------|-----------|-----------|
| Total lines | 2,729 | 2,513 |
| Inline FP math | Yes | No (punted) |
| Scratch registers | 6 (no constraints) | ~4 (RAX/RDX implicit) |
| Instruction encoding | 1 word per instruction | 1-7+ bytes per instruction |
| Prefix bytes needed | 0 | 1-2 (REX, SIB) |
| Custom memory allocator | No | Yes (rel32 constraint) |
| Division instructions | 1 (SDIV) | 3 (REX + CQO + IDIV) |
| Modulo extra work | 1 (MSUB) | 1 (XCHG RAX, RDX) |
