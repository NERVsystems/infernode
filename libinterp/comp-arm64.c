/*
 * ARM64 JIT compiler for Dis Virtual Machine
 *
 * Based on comp-arm.c (ARM32) but adapted for 64-bit ARM64 (AArch64).
 * Key differences from ARM32:
 *   - 64-bit registers (X0-X30) instead of 32-bit (R0-R15)
 *   - Different instruction encoding
 *   - sizeof(WORD) = 8, sizeof(Modl) = 16
 *   - Different addressing mode scaling
 *   - macOS requires MAP_JIT for executable memory
 */

#include "lib9.h"
#include "isa.h"
#include "interp.h"
#include "raise.h"

#ifdef __APPLE__
#include <pthread.h>
#include <libkern/OSCacheControl.h>
#include <sys/mman.h>
#endif

#define RESCHED 1	/* check for interpreter reschedule */

enum
{
	/* 64-bit general purpose registers */
	X0  = 0,  X1  = 1,  X2  = 2,  X3  = 3,
	X4  = 4,  X5  = 5,  X6  = 6,  X7  = 7,
	X8  = 8,  X9  = 9,  X10 = 10, X11 = 11,
	X12 = 12, X13 = 13, X14 = 14, X15 = 15,
	X16 = 16, X17 = 17, X18 = 18, X19 = 19,
	X20 = 20, X21 = 21, X22 = 22, X23 = 23,
	X24 = 24, X25 = 25, X26 = 26, X27 = 27,
	X28 = 28, X29 = 29, X30 = 30, XZR = 31,
	SP  = 31,

	/* 32-bit register names (same encoding) */
	W0  = 0,  W1  = 1,  W2  = 2,  W3  = 3,
	WZR = 31,

	/* Condition codes (same as ARM32) */
	EQ = 0,   /* Equal */
	NE = 1,   /* Not equal */
	CS = 2,   /* Carry set / unsigned higher or same */
	CC = 3,   /* Carry clear / unsigned lower */
	MI = 4,   /* Minus / negative */
	PL = 5,   /* Plus / positive or zero */
	VS = 6,   /* Overflow */
	VC = 7,   /* No overflow */
	HI = 8,   /* Unsigned higher */
	LS = 9,   /* Unsigned lower or same */
	GE = 10,  /* Signed greater or equal */
	LT = 11,  /* Signed less than */
	GT = 12,  /* Signed greater than */
	LE = 13,  /* Signed less or equal */
	AL = 14,  /* Always */
	NV = 15,  /* Never (reserved) */

	HS = CS,  /* Unsigned higher or same */
	LO = CC,  /* Unsigned lower */

	/* Shift types */
	LSL = 0,  /* Logical shift left */
	LSR = 1,  /* Logical shift right */
	ASR = 2,  /* Arithmetic shift right */
	ROR = 3,  /* Rotate right */

	/* Memory operation types */
	Lea = 100,  /* Load effective address */
	Ldw,        /* Load word (64-bit on ARM64) */
	Ldw32,      /* Load 32-bit word */
	Ldb,        /* Load byte */
	Stw,        /* Store word (64-bit) */
	Stw32,      /* Store 32-bit word */
	Stb,        /* Store byte */

	/* Offsets for 64-bit big integers */
	Blo = 0,    /* Low word offset (little endian) */
	Bhi = 4,    /* High word offset - NOTE: still 4 for 32-bit halves */

	/* Literal pool size - ARM64 has +/-1MB range for PC-relative LDR */
	NCON = 512,

	/* Operation flags for punt() */
	SRCOP  = (1<<0),
	DSTOP  = (1<<1),
	WRTPC  = (1<<2),
	TCHECK = (1<<3),
	NEWPC  = (1<<4),
	DBRAN  = (1<<5),
	THREOP = (1<<6),

	/* Branch combination modes */
	ANDAND = 1,
	OROR   = 2,
	EQAND  = 3,

	/* Macro indices */
	MacFRP = 0,
	MacRET,
	MacCASE,
	MacCOLR,
	MacMCAL,
	MacFRAM,
	MacMFRA,
	MacRELQ,
	NMACRO
};

/*
 * VM Register allocation for ARM64
 *
 * Using CALLER-SAVED registers to avoid Apple clang -ffixed limitation
 * (Apple clang doesn't support -ffixed-xNN for arm64-apple-darwin)
 *
 * Caller-saved (must save before C calls):
 *   X9  = RFP   - Dis Frame Pointer
 *   X10 = RMP   - Module Pointer (R.MP)
 *   X11 = RREG  - Pointer to REG struct (&R)
 *   X12 = RM    - Cached R.M (Modlink*)
 *
 * Caller-saved (scratch - not preserved):
 *   X0-X7  = Arguments and temps
 *   X8, X13-X15 = Additional temps
 */
#define RFP     X9      /* Dis Frame Pointer */
#define RMP     X10     /* Module Pointer (R.MP) */
#define RREG    X11     /* Pointer to REG struct (&R) */
#define RM      X12     /* Cached R.M */

#define RA0     X0      /* General purpose 0, return value */
#define RA1     X1      /* General purpose 1 */
#define RA2     X2      /* General purpose 2 */
#define RA3     X3      /* General purpose 3 */
#define RTA     X4      /* Temporary address */
#define RCON    X5      /* Constant builder */
#define RLINK   X30     /* Link register */

/* ARM64 instruction encoding macros */

/* Generate a 32-bit instruction word */
#define emit(w)  (*code++ = (w))

/*
 * Data Processing - Immediate (Add/Sub with 12-bit immediate)
 * 31 30 29 28-24  23 22 21-10    9-5  4-0
 * sf op  S  10001  sh imm12      Rn   Rd
 */
#define ADD_IMM(Rd, Rn, imm12) \
	(0x91000000 | (((imm12)&0xFFF)<<10) | ((Rn)<<5) | (Rd))
#define ADDS_IMM(Rd, Rn, imm12) \
	(0xB1000000 | (((imm12)&0xFFF)<<10) | ((Rn)<<5) | (Rd))
#define SUB_IMM(Rd, Rn, imm12) \
	(0xD1000000 | (((imm12)&0xFFF)<<10) | ((Rn)<<5) | (Rd))
#define SUBS_IMM(Rd, Rn, imm12) \
	(0xF1000000 | (((imm12)&0xFFF)<<10) | ((Rn)<<5) | (Rd))
#define CMP_IMM(Rn, imm12) \
	SUBS_IMM(XZR, Rn, imm12)
#define CMN_IMM(Rn, imm12) \
	ADDS_IMM(XZR, Rn, imm12)

/* 32-bit variants */
#define ADD_IMM32(Rd, Rn, imm12) \
	(0x11000000 | (((imm12)&0xFFF)<<10) | ((Rn)<<5) | (Rd))
#define SUB_IMM32(Rd, Rn, imm12) \
	(0x51000000 | (((imm12)&0xFFF)<<10) | ((Rn)<<5) | (Rd))
#define SUBS_IMM32(Rd, Rn, imm12) \
	(0x71000000 | (((imm12)&0xFFF)<<10) | ((Rn)<<5) | (Rd))
#define CMP_IMM32(Rn, imm12) \
	SUBS_IMM32(WZR, Rn, imm12)

/*
 * Data Processing - Register (shifted register)
 * 31 30 29 28-24 23-22 21 20-16 15-10  9-5  4-0
 * sf op  S  01011 shift  0  Rm   imm6   Rn   Rd
 */
#define ADD_REG(Rd, Rn, Rm) \
	(0x8B000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define ADDS_REG(Rd, Rn, Rm) \
	(0xAB000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define SUB_REG(Rd, Rn, Rm) \
	(0xCB000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define SUBS_REG(Rd, Rn, Rm) \
	(0xEB000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define CMP_REG(Rn, Rm) \
	SUBS_REG(XZR, Rn, Rm)

/* Shifted register variants */
#define ADD_REG_SHIFT(Rd, Rn, Rm, sh, amt) \
	(0x8B000000 | ((sh)<<22) | ((Rm)<<16) | (((amt)&0x3F)<<10) | ((Rn)<<5) | (Rd))
#define SUB_REG_SHIFT(Rd, Rn, Rm, sh, amt) \
	(0xCB000000 | ((sh)<<22) | ((Rm)<<16) | (((amt)&0x3F)<<10) | ((Rn)<<5) | (Rd))

/* 32-bit register variants */
#define ADD_REG32(Rd, Rn, Rm) \
	(0x0B000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define SUB_REG32(Rd, Rn, Rm) \
	(0x4B000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define SUBS_REG32(Rd, Rn, Rm) \
	(0x6B000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define CMP_REG32(Rn, Rm) \
	SUBS_REG32(WZR, Rn, Rm)

/*
 * Logical - Register
 */
#define AND_REG(Rd, Rn, Rm) \
	(0x8A000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define ORR_REG(Rd, Rn, Rm) \
	(0xAA000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define EOR_REG(Rd, Rn, Rm) \
	(0xCA000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define ANDS_REG(Rd, Rn, Rm) \
	(0xEA000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define TST_REG(Rn, Rm) \
	ANDS_REG(XZR, Rn, Rm)

/* 32-bit logical */
#define AND_REG32(Rd, Rn, Rm) \
	(0x0A000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define ORR_REG32(Rd, Rn, Rm) \
	(0x2A000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define EOR_REG32(Rd, Rn, Rm) \
	(0x4A000000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))

/*
 * Shift instructions (variable shift)
 */
#define LSLV(Rd, Rn, Rm) \
	(0x9AC02000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define LSRV(Rd, Rn, Rm) \
	(0x9AC02400 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define ASRV(Rd, Rn, Rm) \
	(0x9AC02800 | ((Rm)<<16) | ((Rn)<<5) | (Rd))

/* 32-bit shifts */
#define LSLV32(Rd, Rn, Rm) \
	(0x1AC02000 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define LSRV32(Rd, Rn, Rm) \
	(0x1AC02400 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define ASRV32(Rd, Rn, Rm) \
	(0x1AC02800 | ((Rm)<<16) | ((Rn)<<5) | (Rd))

/* Immediate shifts */
#define LSL_IMM(Rd, Rn, shift) \
	(0xD3400000 | ((63-(shift))<<16) | (((63-(shift))&0x3F)<<10) | ((Rn)<<5) | (Rd))
#define LSR_IMM(Rd, Rn, shift) \
	(0xD340FC00 | (((shift)&0x3F)<<16) | ((Rn)<<5) | (Rd))
#define ASR_IMM(Rd, Rn, shift) \
	(0x9340FC00 | (((shift)&0x3F)<<16) | ((Rn)<<5) | (Rd))

/*
 * Multiply
 */
#define MUL(Rd, Rn, Rm) \
	(0x9B007C00 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define MUL32(Rd, Rn, Rm) \
	(0x1B007C00 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define SMULL(Rd, Rn, Rm) \
	(0x9B207C00 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define UMULH(Rd, Rn, Rm) \
	(0x9BC07C00 | ((Rm)<<16) | ((Rn)<<5) | (Rd))

/*
 * Division
 */
#define SDIV(Rd, Rn, Rm) \
	(0x9AC00C00 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define UDIV(Rd, Rn, Rm) \
	(0x9AC00800 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define SDIV32(Rd, Rn, Rm) \
	(0x1AC00C00 | ((Rm)<<16) | ((Rn)<<5) | (Rd))
#define UDIV32(Rd, Rn, Rm) \
	(0x1AC00800 | ((Rm)<<16) | ((Rn)<<5) | (Rd))

/*
 * Move instructions
 */
#define MOV_REG(Rd, Rm) \
	ORR_REG(Rd, XZR, Rm)
#define MOV_REG32(Rd, Rm) \
	ORR_REG32(Rd, WZR, Rm)
#define MVN_REG(Rd, Rm) \
	(0xAA200000 | ((Rm)<<16) | (XZR<<5) | (Rd))

/* Move wide immediate */
#define MOVZ(Rd, imm16, hw) \
	(0xD2800000 | ((hw)<<21) | (((imm16)&0xFFFF)<<5) | (Rd))
#define MOVN(Rd, imm16, hw) \
	(0x92800000 | ((hw)<<21) | (((imm16)&0xFFFF)<<5) | (Rd))
#define MOVK(Rd, imm16, hw) \
	(0xF2800000 | ((hw)<<21) | (((imm16)&0xFFFF)<<5) | (Rd))

/* 32-bit move wide */
#define MOVZ32(Rd, imm16, hw) \
	(0x52800000 | ((hw)<<21) | (((imm16)&0xFFFF)<<5) | (Rd))
#define MOVN32(Rd, imm16, hw) \
	(0x12800000 | ((hw)<<21) | (((imm16)&0xFFFF)<<5) | (Rd))
#define MOVK32(Rd, imm16, hw) \
	(0x72800000 | ((hw)<<21) | (((imm16)&0xFFFF)<<5) | (Rd))

/*
 * Load/Store - Unsigned offset (scaled)
 * For 64-bit loads, offset is scaled by 8
 * For 32-bit loads, offset is scaled by 4
 * For byte loads, offset is not scaled
 */
#define LDR_UOFF(Rt, Rn, imm12) \
	(0xF9400000 | ((((imm12)>>3)&0xFFF)<<10) | ((Rn)<<5) | (Rt))
#define STR_UOFF(Rt, Rn, imm12) \
	(0xF9000000 | ((((imm12)>>3)&0xFFF)<<10) | ((Rn)<<5) | (Rt))
#define LDR32_UOFF(Rt, Rn, imm12) \
	(0xB9400000 | ((((imm12)>>2)&0xFFF)<<10) | ((Rn)<<5) | (Rt))
#define STR32_UOFF(Rt, Rn, imm12) \
	(0xB9000000 | ((((imm12)>>2)&0xFFF)<<10) | ((Rn)<<5) | (Rt))
#define LDRB_UOFF(Rt, Rn, imm12) \
	(0x39400000 | (((imm12)&0xFFF)<<10) | ((Rn)<<5) | (Rt))
#define STRB_UOFF(Rt, Rn, imm12) \
	(0x39000000 | (((imm12)&0xFFF)<<10) | ((Rn)<<5) | (Rt))
#define LDRH_UOFF(Rt, Rn, imm12) \
	(0x79400000 | ((((imm12)>>1)&0xFFF)<<10) | ((Rn)<<5) | (Rt))
#define STRH_UOFF(Rt, Rn, imm12) \
	(0x79000000 | ((((imm12)>>1)&0xFFF)<<10) | ((Rn)<<5) | (Rt))

/* Load/Store - Register offset */
#define LDR_REG(Rt, Rn, Rm) \
	(0xF8606800 | ((Rm)<<16) | ((Rn)<<5) | (Rt))
#define STR_REG(Rt, Rn, Rm) \
	(0xF8206800 | ((Rm)<<16) | ((Rn)<<5) | (Rt))
#define LDR32_REG(Rt, Rn, Rm) \
	(0xB8606800 | ((Rm)<<16) | ((Rn)<<5) | (Rt))
#define STR32_REG(Rt, Rn, Rm) \
	(0xB8206800 | ((Rm)<<16) | ((Rn)<<5) | (Rt))
#define LDRB_REG(Rt, Rn, Rm) \
	(0x38606800 | ((Rm)<<16) | ((Rn)<<5) | (Rt))
#define STRB_REG(Rt, Rn, Rm) \
	(0x38206800 | ((Rm)<<16) | ((Rn)<<5) | (Rt))

/* Load/Store - Unscaled immediate (for arbitrary offsets) */
#define LDUR(Rt, Rn, imm9) \
	(0xF8400000 | (((imm9)&0x1FF)<<12) | ((Rn)<<5) | (Rt))
#define STUR(Rt, Rn, imm9) \
	(0xF8000000 | (((imm9)&0x1FF)<<12) | ((Rn)<<5) | (Rt))
#define LDUR32(Rt, Rn, imm9) \
	(0xB8400000 | (((imm9)&0x1FF)<<12) | ((Rn)<<5) | (Rt))
#define STUR32(Rt, Rn, imm9) \
	(0xB8000000 | (((imm9)&0x1FF)<<12) | ((Rn)<<5) | (Rt))
#define LDURB(Rt, Rn, imm9) \
	(0x38400000 | (((imm9)&0x1FF)<<12) | ((Rn)<<5) | (Rt))
#define STURB(Rt, Rn, imm9) \
	(0x38000000 | (((imm9)&0x1FF)<<12) | ((Rn)<<5) | (Rt))

/* Load literal (PC-relative) - 19-bit signed offset, scaled by 4 */
#define LDR_LIT(Rt, imm19) \
	(0x58000000 | ((((imm19)>>2)&0x7FFFF)<<5) | (Rt))

/* Load/Store pair */
#define LDP(Rt1, Rt2, Rn, imm7) \
	(0xA9400000 | ((((imm7)>>3)&0x7F)<<15) | ((Rt2)<<10) | ((Rn)<<5) | (Rt1))
#define STP(Rt1, Rt2, Rn, imm7) \
	(0xA9000000 | ((((imm7)>>3)&0x7F)<<15) | ((Rt2)<<10) | ((Rn)<<5) | (Rt1))
#define LDP_PRE(Rt1, Rt2, Rn, imm7) \
	(0xA9C00000 | ((((imm7)>>3)&0x7F)<<15) | ((Rt2)<<10) | ((Rn)<<5) | (Rt1))
#define STP_PRE(Rt1, Rt2, Rn, imm7) \
	(0xA9800000 | ((((imm7)>>3)&0x7F)<<15) | ((Rt2)<<10) | ((Rn)<<5) | (Rt1))
#define LDP_POST(Rt1, Rt2, Rn, imm7) \
	(0xA8C00000 | ((((imm7)>>3)&0x7F)<<15) | ((Rt2)<<10) | ((Rn)<<5) | (Rt1))
#define STP_POST(Rt1, Rt2, Rn, imm7) \
	(0xA8800000 | ((((imm7)>>3)&0x7F)<<15) | ((Rt2)<<10) | ((Rn)<<5) | (Rt1))

/* Pre-index and post-index */
#define LDR_PRE(Rt, Rn, imm9) \
	(0xF8400C00 | (((imm9)&0x1FF)<<12) | ((Rn)<<5) | (Rt))
#define STR_PRE(Rt, Rn, imm9) \
	(0xF8000C00 | (((imm9)&0x1FF)<<12) | ((Rn)<<5) | (Rt))
#define LDR_POST(Rt, Rn, imm9) \
	(0xF8400400 | (((imm9)&0x1FF)<<12) | ((Rn)<<5) | (Rt))
#define STR_POST(Rt, Rn, imm9) \
	(0xF8000400 | (((imm9)&0x1FF)<<12) | ((Rn)<<5) | (Rt))

/* Sign/Zero extend */
#define SXTB(Rd, Rn) \
	(0x93401C00 | ((Rn)<<5) | (Rd))
#define SXTH(Rd, Rn) \
	(0x93403C00 | ((Rn)<<5) | (Rd))
#define SXTW(Rd, Rn) \
	(0x93407C00 | ((Rn)<<5) | (Rd))
#define UXTB(Rd, Rn) \
	(0x53001C00 | ((Rn)<<5) | (Rd))
#define UXTH(Rd, Rn) \
	(0x53003C00 | ((Rn)<<5) | (Rd))

/*
 * Branch instructions
 */
/* Unconditional branch - 26-bit signed offset */
#define B(imm26) \
	(0x14000000 | ((imm26)&0x3FFFFFF))
#define BL(imm26) \
	(0x94000000 | ((imm26)&0x3FFFFFF))

/* Conditional branch - 19-bit signed offset */
#define BCOND(cond, imm19) \
	(0x54000000 | ((((imm19)>>2)&0x7FFFF)<<5) | (cond))

/* Compare and branch */
#define CBZ(Rt, imm19) \
	(0xB4000000 | ((((imm19)>>2)&0x7FFFF)<<5) | (Rt))
#define CBNZ(Rt, imm19) \
	(0xB5000000 | ((((imm19)>>2)&0x7FFFF)<<5) | (Rt))
#define CBZ32(Rt, imm19) \
	(0x34000000 | ((((imm19)>>2)&0x7FFFF)<<5) | (Rt))
#define CBNZ32(Rt, imm19) \
	(0x35000000 | ((((imm19)>>2)&0x7FFFF)<<5) | (Rt))

/* Test bit and branch */
#define TBZ(Rt, bit, imm14) \
	(0x36000000 | (((bit)&0x20)<<26) | (((bit)&0x1F)<<19) | ((((imm14)>>2)&0x3FFF)<<5) | (Rt))
#define TBNZ(Rt, bit, imm14) \
	(0x37000000 | (((bit)&0x20)<<26) | (((bit)&0x1F)<<19) | ((((imm14)>>2)&0x3FFF)<<5) | (Rt))

/* Branch to register */
#define BR(Rn) \
	(0xD61F0000 | ((Rn)<<5))
#define BLR(Rn) \
	(0xD63F0000 | ((Rn)<<5))
#define RET_Rn(Rn) \
	(0xD65F0000 | ((Rn)<<5))
#define RET_LR \
	RET_Rn(X30)

/* Nop */
#define NOP \
	(0xD503201F)

/*
 * Address generation
 */
#define ADR(Rd, imm21) \
	(0x10000000 | (((imm21)&3)<<29) | ((((imm21)>>2)&0x7FFFF)<<5) | (Rd))
#define ADRP(Rd, imm21) \
	(0x90000000 | (((imm21)&3)<<29) | ((((imm21)>>2)&0x7FFFF)<<5) | (Rd))

/* Helper macros */
#define FITS12(v)       ((uvlong)(v) < (1ULL<<12))
#define FITS12S(v)      ((vlong)(v) >= 0 && (vlong)(v) < (1LL<<12))
#define FITS9S(v)       ((vlong)(v) >= -256 && (vlong)(v) < 256)
#define FITS16(v)       ((uvlong)(v) < (1ULL<<16))

/* Check if value is H (nil = -1) */
#define CMPH(Rn)        emit(CMN_IMM(Rn, 1))

/* Branch helpers */
#define IA(s, o)        (uvlong)(base + s[o])
#define RELPC(pc)       (uvlong)(base + (pc))
#define PATCH_BCOND(ptr, off) \
	do { *(ptr) = (*(ptr) & ~(0x7FFFF<<5)) | (((((off)>>2))&0x7FFFF)<<5); } while(0)
#define PATCH_B(ptr, off) \
	do { *(ptr) = (*(ptr) & ~0x3FFFFFF) | (((off)>>2)&0x3FFFFFF); } while(0)
/* CBZ/CBNZ use same offset encoding as B.cond */
#define PATCH_CBZ(ptr, off)  PATCH_BCOND(ptr, off)
#define PATCH_CBNZ(ptr, off) PATCH_BCOND(ptr, off)

/* Static variables */
static	u32int*	code;		/* ARM64 instructions are 32-bit */
static	u32int*	base;
static	u32int*	patch;
static	ulong	codeoff;
static	int	pass;
static	Module*	mod;
static	uchar*	tinit;
static	u32int*	litpool;
static	int	nlit;
static	ulong	macro[NMACRO];
	void	(*comvec)(void);

/* Forward declarations */
static	void	macfrp(void);
static	void	macret(void);
static	void	maccase(void);
static	void	maccolr(void);
static	void	macmcal(void);
static	void	macfram(void);
static	void	macmfra(void);
static	void	macrelq(void);
static	void	opwld(Inst*, int, int);
static	void	opwst(Inst*, int, int);
static	void	mid(Inst*, int, int);
static	void	mem(int, vlong, int, int);
static	void	con(uvlong, int, int);
static	void	punt(Inst*, int, void(*)(void));

extern	void	das(u32int*, int);

#define T(r)	*((void**)(R.r))

/* Macro table */
struct
{
	int	idx;
	void	(*gen)(void);
	char*	name;
} mactab[] =
{
	MacFRP,		macfrp,		"FRP",
	MacRET,		macret,		"RET",
	MacCASE,	maccase,	"CASE",
	MacCOLR,	maccolr,	"COLR",
	MacMCAL,	macmcal,	"MCAL",
	MacFRAM,	macfram,	"FRAM",
	MacMFRA,	macmfra,	"MFRA",
	MacRELQ,	macrelq,	"RELQ",
};

/* Constant pool management */
typedef struct Const Const;
struct Const
{
	uvlong	o;
	u32int*	code;		/* ARM64 instructions are 32-bit */
	u32int*	pc;
};

typedef struct Con Con;
struct Con
{
	int	ptr;
	Const	table[NCON];
};
static Con rcon;

/* No separate AXIMM storage needed - using literal pool (inferno64 approach) */

static void
jitdebug(int n)
{
	print("JIT: after punt, n=%d, R.FP=%p, R.MP=%p, R.PC=%p\n", n, R.FP, R.MP, R.PC);
}

static void
puntdebug(void)
{
	print("JIT punt: R.s=%p *R.s=%p R.m=%p R.t=%ld &R.t=%p\n",
		R.s, R.s ? *(void**)R.s : nil, R.m, (long)R.t, &R.t);
	print("  &R=%p R.FP=%p R.MP=%p\n", &R, R.FP, R.MP);
}

static void
trace_store_regs(void *addr, void *value)
{
	/* Called from JIT with actual addr/value being stored */
	vlong diff = (uchar*)value - (uchar*)addr;
	if(cflag > 3)
		print("trace_store: addr=%p value=%p diff=%ld\n", addr, value, diff);
	/* Detect the suspicious pattern: value = addr + 8 */
	if(diff == 8 || diff == -8 || (diff > 0 && diff < 32)) {
		print("SUSPICIOUS_STORE: addr=%p value=%p diff=%ld RFP=%p RMP=%p\n",
			addr, value, diff, R.FP, R.MP);
		print("  R.PC=%p\n", R.PC);
	}
}

static void
rdestroy(void)
{
	destroy(R.s);
}

static void
rmcall(void)
{
	Frame *f;
	Prog *p;

	if((void*)R.dt == H)
		error(exModule);

	f = (Frame*)R.FP;
	if(f == H)
		error(exModule);

	f->mr = nil;
	((void(*)(Frame*))R.dt)(f);
	R.SP = (uchar*)f;
	R.FP = f->fp;
	if(f->t == nil)
		unextend(f);
	else
		freeptrs(f, f->t);
	p = currun();
	if(p->kill != nil)
		error(p->kill);
}

static void
rmfram(void)
{
	Type *t;
	Frame *f;
	uchar *nsp;

	if(R.d == H)
		error(exModule);
	t = (Type*)R.s;
	if(t == H)
		error(exModule);
	nsp = R.SP + t->size;
	if(nsp >= R.TS) {
		R.s = t;
		extend();
		T(d) = R.s;
		return;
	}
	f = (Frame*)R.SP;
	R.SP = nsp;
	f->t = t;
	f->mr = nil;
	initmem(t, f);
	T(d) = f;
}

static void
urk(char *s)
{
	print("[JIT] urk() error: %s\n", s);
	USED(s);
	error(exCompile);
}

static void
bounds(void)
{
	error(exBounds);
}

/*
 * Flush constant pool to code stream
 */
static void
flushcon(int genbr)
{
	int i;
	Const *c;
	vlong disp;

	if(rcon.ptr == 0)
		return;

	if(cflag > 3 && pass)
		print("flushcon: genbr=%d ptr=%d code=%p pass=%d\n",
			genbr, rcon.ptr, code, pass);

	if(genbr) {
		/* Branch over literal pool - each constant is 2 words (64-bit) */
		emit(B(rcon.ptr * 2 + 1));
	}

	c = &rcon.table[0];
	for(i = 0; i < rcon.ptr; i++) {
		if(pass) {
			disp = (vlong)(code - c->code) * 4;
			if(cflag > 3)
				print("flushcon: i=%d code=%p c->code=%p disp=%lld words=%lld val=%llx\n",
					i, code, c->code, disp, disp/4, (uvlong)c->o);
			if(disp < 0 || disp >= (1<<20)) {
				print("constant range error: %lld\n", disp);
				urk("constant range");
			}
			/* Patch LDR literal instruction */
			*c->code = (*c->code & ~(0x7FFFF<<5)) |
			           (((disp>>2)&0x7FFFF)<<5);
		}
		/* Emit 64-bit constant (little endian, 2 x 32-bit words) */
		*code++ = (u32int)(c->o);
		*code++ = (u32int)(c->o >> 32);
		c++;
	}
	rcon.ptr = 0;
}

/*
 * Check if constant pool needs flushing
 */
static void
flushchk(void)
{
	if(rcon.ptr >= NCON ||
	   (rcon.ptr > 0 && (code + codeoff + 2 - rcon.table[0].pc) * 4 >= (1<<19) - 256)) {
		if(cflag > 3 && pass)
			print("flushchk: FLUSHING ptr=%d code=%p codeoff=%lu\n",
				rcon.ptr, code, codeoff);
		flushcon(1);
	}
}

/*
 * Store immediate value in literal pool and set R field to point to it
 * Based on inferno64 approach - avoids needing separate aximm_storage array
 */
static void
literal(uvlong imm, int roff)
{
	nlit++;
	con((uvlong)litpool, RTA, 0);
	mem(Stw, roff, RREG, RTA);

	if(pass == 0)
		return;

	/* Pass 1: Write value to literal pool (little endian, 64-bit WORD) */
	*litpool++ = (u32int)(imm);
	*litpool++ = (u32int)(imm >> 32);
}

/*
 * Load a 64-bit constant into a register
 */
static void
con(uvlong o, int r, int opt)
{
	Const *c;

	/* Try immediate forms first */
	if(opt) {
		/* Zero */
		if(o == 0) {
			emit(MOV_REG(r, XZR));
			return;
		}
		/* Small positive - fits in 16 bits */
		if(o < (1ULL<<16)) {
			emit(MOVZ(r, o, 0));
			return;
		}
		/* Small negative - inverted fits in 16 bits */
		if(~o < (1ULL<<16)) {
			emit(MOVN(r, ~o, 0));
			return;
		}
		/* Two-instruction sequence for 32-bit values */
		if(o < (1ULL<<32)) {
			emit(MOVZ(r, o & 0xFFFF, 0));
			if(o >> 16)
				emit(MOVK(r, (o >> 16) & 0xFFFF, 1));
			return;
		}
		/* Check for values that need only a few MOVK instructions */
		{
			int hw, first = 1;
			int count = 0;
			for(hw = 0; hw < 4; hw++) {
				if((o >> (hw * 16)) & 0xFFFF)
					count++;
			}
			if(count <= 2) {
				for(hw = 0; hw < 4; hw++) {
					ushort imm16 = (o >> (hw * 16)) & 0xFFFF;
					if(imm16 != 0) {
						if(first) {
							emit(MOVZ(r, imm16, hw));
							first = 0;
						} else {
							emit(MOVK(r, imm16, hw));
						}
					}
				}
				return;
			}
		}
	}

	/* Use literal pool for large constants */
	flushchk();
	c = &rcon.table[rcon.ptr++];
	c->o = o;
	c->code = code;
	c->pc = code + codeoff;
	if(cflag > 3 && pass)
		print("con: pool idx=%d val=%llx code=%p\n",
			rcon.ptr-1, (uvlong)o, code);
	emit(LDR_LIT(r, 0));  /* Placeholder, patched in flushcon */
}

/*
 * Memory operation with displacement
 */
static void
mem(int inst, vlong disp, int rm, int r)
{
	if(inst == Lea) {
		/* Load effective address - add displacement to base */
		if(disp == 0) {
			if(rm != r)
				emit(MOV_REG(r, rm));
			return;
		}
		if(FITS12S(disp)) {
			emit(ADD_IMM(r, rm, disp));
			return;
		}
		if(FITS12S(-disp)) {
			emit(SUB_IMM(r, rm, -disp));
			return;
		}
		/* Large displacement - load constant and add */
		con(disp, RCON, 1);
		emit(ADD_REG(r, rm, RCON));
		return;
	}

	/* Load/Store operations */
	/* Try scaled unsigned offset first (most common case) */
	if(disp >= 0) {
		switch(inst) {
		case Ldw:
			if((disp & 7) == 0 && disp < (1<<15)) {
				emit(LDR_UOFF(r, rm, disp));
				return;
			}
			break;
		case Stw:
			if((disp & 7) == 0 && disp < (1<<15)) {
				emit(STR_UOFF(r, rm, disp));
				return;
			}
			break;
		case Ldw32:
			if((disp & 3) == 0 && disp < (1<<14)) {
				emit(LDR32_UOFF(r, rm, disp));
				return;
			}
			break;
		case Stw32:
			if((disp & 3) == 0 && disp < (1<<14)) {
				emit(STR32_UOFF(r, rm, disp));
				return;
			}
			break;
		case Ldb:
			if(disp < (1<<12)) {
				emit(LDRB_UOFF(r, rm, disp));
				return;
			}
			break;
		case Stb:
			if(disp < (1<<12)) {
				emit(STRB_UOFF(r, rm, disp));
				return;
			}
			break;
		}
	}

	/* Try unscaled signed offset */
	if(FITS9S(disp)) {
		switch(inst) {
		case Ldw:
			emit(LDUR(r, rm, disp));
			return;
		case Stw:
			emit(STUR(r, rm, disp));
			return;
		case Ldw32:
			emit(LDUR32(r, rm, disp));
			return;
		case Stw32:
			emit(STUR32(r, rm, disp));
			return;
		case Ldb:
			emit(LDURB(r, rm, disp));
			return;
		case Stb:
			emit(STURB(r, rm, disp));
			return;
		}
	}

	/* Large displacement - use register offset */
	con(disp, RCON, 1);
	switch(inst) {
	case Ldw:
		emit(LDR_REG(r, rm, RCON));
		break;
	case Stw:
		emit(STR_REG(r, rm, RCON));
		break;
	case Ldw32:
		emit(LDR32_REG(r, rm, RCON));
		break;
	case Stw32:
		emit(STR32_REG(r, rm, RCON));
		break;
	case Ldb:
		emit(LDRB_REG(r, rm, RCON));
		break;
	case Stb:
		emit(STRB_REG(r, rm, RCON));
		break;
	}
}

/*
 * Load source operand
 */
static void
opwld(Inst *i, int mi, int r)
{
	int ir, rta;

	switch(UXSRC(i->add)) {
	default:
		print("%D\n", i);
		urk("opwld");
	case SRC(AFP):
		if(cflag > 4 && pass && mi == Lea)
			print("  opwld: AFP Lea offset=%d -> r%d = RFP + %d\n", i->s.ind, r, i->s.ind);
		mem(mi, i->s.ind, RFP, r);
		return;
	case SRC(AMP):
		if(cflag > 4 && pass && mi == Lea)
			print("  opwld: AMP Lea offset=%d -> r%d = RMP + %d\n", i->s.ind, r, i->s.ind);
		mem(mi, i->s.ind, RMP, r);
		return;
	case SRC(AIMM):
		con(i->s.imm, r, 1);
		/* Special case: Lea of immediate requires storing to temp and taking address */
		if(mi == Lea) {
			mem(Stw, O(REG, st), RREG, r);   /* Store immediate to R.st */
			mem(Lea, O(REG, st), RREG, r);   /* r = &R.st (address) */
		}
		return;
	case SRC(AIND|AFP):
		ir = RFP;
		break;
	case SRC(AIND|AMP):
		ir = RMP;
		break;
	}
	rta = RTA;
	if(mi == Lea)
		rta = r;
	if(cflag > 4 && pass && mi == Lea)
		print("  opwld: AIND Lea f=%d s=%d -> load ptr, then add offset\n", i->s.i.f, i->s.i.s);
	mem(Ldw, i->s.i.f, ir, rta);
	mem(mi, i->s.i.s, rta, r);
}

/*
 * Store to destination operand
 */
static void
opwst(Inst *i, int mi, int r)
{
	int ir, rta;

	if(cflag > 4 && pass && mi == Lea)
		print("  opwst(Lea): UXDST=0x%x d.i.f=%d d.i.s=%d\n",
			UXDST(i->add), i->d.i.f, i->d.i.s);

	switch(UXDST(i->add)) {
	default:
		print("%D\n", i);
		urk("opwst");
	case DST(AIMM):
		con(i->d.imm, r, 1);
		/* Special case: Lea of immediate requires storing to temp and taking address */
		if(mi == Lea) {
			mem(Stw, O(REG, dt), RREG, r);   /* Store immediate to R.dt */
			mem(Lea, O(REG, dt), RREG, r);   /* r = &R.dt (address) */
		}
		return;
	case DST(AFP):
		if(cflag > 4 && pass && mi == Lea)
			print("    -> AFP: r%d = RFP + %d\n", r, i->d.ind);
		mem(mi, i->d.ind, RFP, r);
		return;
	case DST(AMP):
		if(cflag > 4 && pass && mi == Lea)
			print("    -> AMP: r%d = RMP + %d\n", r, i->d.ind);
		mem(mi, i->d.ind, RMP, r);
		return;
	case DST(AIND|AFP):
		ir = RFP;
		break;
	case DST(AIND|AMP):
		ir = RMP;
		break;
	}
	rta = RTA;
	if(mi == Lea)
		rta = r;
	if(cflag > 4 && pass && mi == Lea)
		print("    -> AIND: load ptr from ir+%d, then r%d = ptr + %d\n",
			i->d.i.f, r, i->d.i.s);
	mem(Ldw, i->d.i.f, ir, rta);
	mem(mi, i->d.i.s, rta, r);
}

/*
 * Load middle operand (for three-operand instructions)
 */
static void
mid(Inst *i, int mi, int r)
{
	int ir;

	switch(i->add & ARM) {
	default:
		opwst(i, mi, r);
		return;
	case AXIMM:
		con((short)i->reg, r, 1);
		return;
	case AXINF:
		ir = RFP;
		break;
	case AXINM:
		ir = RMP;
		break;
	}
	mem(mi, i->reg, ir, r);
}

/*
 * Punt to interpreter for complex operations
 */
static void
punt(Inst *i, int m, void (*fn)(void))
{
	uvlong pc;

	/* Save R.FP */
	mem(Stw, O(REG, FP), RREG, RFP);

	if(m & SRCOP) {
		if(cflag > 3 && pass)
			print("punt: SRCOP add=0x%x UXSRC=0x%x s.i.f=%d s.i.s=%d\n",
				i->add, UXSRC(i->add), i->s.i.f, i->s.i.s);
		if(UXSRC(i->add) == SRC(AIMM)) {
			con(i->s.imm, RA0, 1);
			mem(Stw, O(REG, s), RREG, RA0);
		} else {
			opwld(i, Lea, RA0);
			mem(Stw, O(REG, s), RREG, RA0);
		}
	}

	if(m & DSTOP) {
		opwst(i, Lea, RA0);
		mem(Stw, O(REG, d), RREG, RA0);
	}

	if(m & WRTPC) {
		pc = patch[i - mod->prog + 1];
		con((uvlong)(base + pc), RA0, 0);  /* Must use opt=0: base differs between passes */
		mem(Stw, O(REG, PC), RREG, RA0);
	}

	if(m & DBRAN) {
		pc = patch[(Inst*)i->d.imm - mod->prog];
		con((uvlong)(base + pc), RA0, 0);  /* Must use opt=0: base differs between passes */
		mem(Stw, O(REG, d), RREG, RA0);
	}

	if(cflag > 3 && pass && (m & THREOP))
		print("punt THREOP: add&ARM=0x%x reg=%d\n", i->add & ARM, i->reg);

	switch(i->add & ARM) {
	case AXNON:
		if(m & THREOP) {
			mem(Ldw, O(REG, d), RREG, RA0);
			mem(Stw, O(REG, m), RREG, RA0);
		}
		break;
	case AXIMM:
		/*
		 * Store immediate in literal pool to avoid R.t corruption by interpreter.
		 * R.t can be modified by C interpreter functions (see dec.c), so using it
		 * as temporary storage for AXIMM values is unsafe.
		 * Based on inferno64 approach: use literal pool instead of separate array.
		 */
		literal((short)i->reg, O(REG, m));
		break;
	case AXINF:
		mem(Lea, i->reg, RFP, RA0);
		mem(Stw, O(REG, m), RREG, RA0);
		break;
	case AXINM:
		mem(Lea, i->reg, RMP, RA0);
		mem(Stw, O(REG, m), RREG, RA0);
		break;
	}

	/* Save VM registers before calling C (X9-X12 are caller-saved) */
	emit(STP_PRE(RFP, RMP, SP, -32));   /* Save X9, X10 */
	emit(STP(RREG, RM, SP, 16));        /* Save X11, X12 */

	/* Debug: print R.s and R.m before calling interpreter */
	if(cflag > 2) {
		con((uvlong)puntdebug, RA0, 1);
		emit(BLR(RA0));
	}

	/* Call the function */
	con((uvlong)fn, RA0, 1);
	emit(BLR(RA0));

	/* Restore VM registers after C call */
	emit(LDP(RREG, RM, SP, 16));        /* Restore X11, X12 */
	emit(LDP_POST(RFP, RMP, SP, 32));   /* Restore X9, X10 */

	/* Check for thread termination */
	if(m & TCHECK) {
		mem(Ldw, O(REG, t), RREG, RA0);
		emit(CBNZ(RA0, 3*4));  /* Skip restore and return if t != 0 */
	}

	/* Reload potentially changed values from R struct */
	mem(Ldw, O(REG, FP), RREG, RFP);
	mem(Ldw, O(REG, MP), RREG, RMP);
	mem(Ldw, O(REG, M), RREG, RM);

	if(m & TCHECK) {
		/* Return to interpreter if t != 0 */
		emit(RET_LR);
	}

	if(m & NEWPC) {
		mem(Ldw, O(REG, PC), RREG, RA0);
		emit(BR(RA0));
	}
}

/*
 * Arithmetic operations
 */
static void
arith(Inst *i, int op)
{
	/* op: 0=add, 1=sub */
	if(UXSRC(i->add) == SRC(AIMM)) {
		mid(i, Ldw, RA0);
		if(FITS12S(i->s.imm)) {
			if(op == 0)
				emit(ADD_IMM(RA0, RA0, i->s.imm));
			else
				emit(SUB_IMM(RA0, RA0, i->s.imm));
		} else if(FITS12S(-i->s.imm)) {
			if(op == 0)
				emit(SUB_IMM(RA0, RA0, -i->s.imm));
			else
				emit(ADD_IMM(RA0, RA0, -i->s.imm));
		} else {
			con(i->s.imm, RA1, 1);
			if(op == 0)
				emit(ADD_REG(RA0, RA0, RA1));
			else
				emit(SUB_REG(RA0, RA0, RA1));
		}
		opwst(i, Stw, RA0);
		return;
	}

	opwld(i, Ldw, RA1);
	mid(i, Ldw, RA0);
	if(op == 0)
		emit(ADD_REG(RA0, RA0, RA1));
	else
		emit(SUB_REG(RA0, RA0, RA1));
	opwst(i, Stw, RA0);
}

static void
arithb(Inst *i, int op)
{
	opwld(i, Ldb, RA1);
	mid(i, Ldb, RA0);
	if(op == 0)
		emit(ADD_REG32(RA0, RA0, RA1));
	else
		emit(SUB_REG32(RA0, RA0, RA1));
	opwst(i, Stb, RA0);
}

static void
logic(Inst *i, int op)
{
	/* op: 0=and, 1=or, 2=xor */
	opwld(i, Ldw, RA1);
	mid(i, Ldw, RA0);
	switch(op) {
	case 0:
		emit(AND_REG(RA0, RA0, RA1));
		break;
	case 1:
		emit(ORR_REG(RA0, RA0, RA1));
		break;
	case 2:
		emit(EOR_REG(RA0, RA0, RA1));
		break;
	}
	opwst(i, Stw, RA0);
}

static void
logicb(Inst *i, int op)
{
	opwld(i, Ldb, RA1);
	mid(i, Ldb, RA0);
	switch(op) {
	case 0:
		emit(AND_REG32(RA0, RA0, RA1));
		break;
	case 1:
		emit(ORR_REG32(RA0, RA0, RA1));
		break;
	case 2:
		emit(EOR_REG32(RA0, RA0, RA1));
		break;
	}
	opwst(i, Stb, RA0);
}

static void
shift(Inst *i, int op)
{
	/* op: 0=shl, 1=shr(logical), 2=shr(arithmetic) */
	opwld(i, Ldw, RA1);  /* shift amount */
	mid(i, Ldw, RA0);    /* value */
	switch(op) {
	case 0:
		emit(LSLV(RA0, RA0, RA1));
		break;
	case 1:
		emit(LSRV(RA0, RA0, RA1));
		break;
	case 2:
		emit(ASRV(RA0, RA0, RA1));
		break;
	}
	opwst(i, Stw, RA0);
}

static void
shiftb(Inst *i, int op)
{
	opwld(i, Ldb, RA1);
	mid(i, Ldb, RA0);
	switch(op) {
	case 0:
		emit(LSLV32(RA0, RA0, RA1));
		break;
	case 1:
		emit(LSRV32(RA0, RA0, RA1));
		break;
	case 2:
		emit(ASRV32(RA0, RA0, RA1));
		break;
	}
	opwst(i, Stb, RA0);
}

/*
 * Reschedule check for backward branches
 */
static void
schedcheck(Inst *i)
{
	if(RESCHED && i->d.ins <= i) {
		/* Decrement R.IC and reschedule if <= 0 */
		mem(Ldw32, O(REG, IC), RREG, RA0);
		emit(SUB_IMM32(RA0, RA0, 1));
		mem(Stw32, O(REG, IC), RREG, RA0);
		/* If IC > 0, skip reschedule */
		emit(BCOND(GT, 2*4));  /* Skip next 2 instructions */
		/* Call reschedule macro */
		con((uvlong)(base + macro[MacRELQ]), RA0, 0);
		emit(BLR(RA0));
	}
}

/*
 * Conditional branch (word comparison)
 */
static void
cbra(Inst *i, int cond)
{
	vlong dst;

	if(RESCHED)
		schedcheck(i);

	mid(i, Ldw, RA0);
	if(UXSRC(i->add) == SRC(AIMM)) {
		if(FITS12S(i->s.imm)) {
			emit(CMP_IMM(RA0, i->s.imm));
		} else {
			con(i->s.imm, RA1, 1);
			emit(CMP_REG(RA0, RA1));
		}
		/* Swap condition for reversed operands */
		switch(cond) {
		case GT: cond = LT; break;
		case LT: cond = GT; break;
		case GE: cond = LE; break;
		case LE: cond = GE; break;
		case HI: cond = LO; break;
		case LO: cond = HI; break;
		case HS: cond = LS; break;
		case LS: cond = HS; break;
		}
	} else {
		opwld(i, Ldw, RA1);
		emit(CMP_REG(RA0, RA1));
	}

	dst = patch[i->d.ins - mod->prog];
	if(pass) {
		vlong off = ((vlong)base + dst) - (vlong)code;
		emit(BCOND(cond, off));
	} else {
		emit(BCOND(cond, 0));  /* Placeholder */
	}
}

/*
 * Conditional branch (byte comparison)
 */
static void
cbrab(Inst *i, int cond)
{
	vlong dst;

	if(RESCHED)
		schedcheck(i);

	mid(i, Ldb, RA0);
	opwld(i, Ldb, RA1);
	/* Zero-extend and compare */
	emit(CMP_REG32(RA0, RA1));

	dst = patch[i->d.ins - mod->prog];
	if(pass) {
		vlong off = ((vlong)base + dst) - (vlong)code;
		emit(BCOND(cond, off));
	} else {
		emit(BCOND(cond, 0));
	}
}

/*
 * 64-bit (long) operations
 */
static void
larithl(Inst *i, int op)
{
	/* 64-bit add/sub/logic - same as word on ARM64 */
	opwld(i, Ldw, RA1);
	mid(i, Ldw, RA0);
	switch(op) {
	case 0:  /* add */
		emit(ADD_REG(RA0, RA0, RA1));
		break;
	case 1:  /* sub */
		emit(SUB_REG(RA0, RA0, RA1));
		break;
	case 2:  /* and */
		emit(AND_REG(RA0, RA0, RA1));
		break;
	case 3:  /* or */
		emit(ORR_REG(RA0, RA0, RA1));
		break;
	case 4:  /* xor */
		emit(EOR_REG(RA0, RA0, RA1));
		break;
	}
	opwst(i, Stw, RA0);
}

static void
shiftl(Inst *i, int op)
{
	opwld(i, Ldw, RA1);
	mid(i, Ldw, RA0);
	switch(op) {
	case 0:
		emit(LSLV(RA0, RA0, RA1));
		break;
	case 1:
		emit(LSRV(RA0, RA0, RA1));
		break;
	case 2:
		emit(ASRV(RA0, RA0, RA1));
		break;
	}
	opwst(i, Stw, RA0);
}

/*
 * Conditional branch (64-bit long comparison)
 */
static void
cbral(Inst *i, int jmsw, int jlsw, int mode)
{
	vlong dst;
	u32int *label = nil;

	if(RESCHED)
		schedcheck(i);

	/* On ARM64, 64-bit comparison is the same as word comparison */
	opwld(i, Ldw, RA1);
	mid(i, Ldw, RA0);
	emit(CMP_REG(RA0, RA1));

	dst = patch[i->d.ins - mod->prog];

	USED(jmsw);
	USED(mode);
	USED(label);

	if(pass) {
		vlong off = ((vlong)base + dst) - (vlong)code;
		emit(BCOND(jlsw, off));
	} else {
		emit(BCOND(jlsw, 0));
	}
}

/*
 * Compile a single instruction
 */
static void
comp(Inst *i)
{
	char buf[64];

	flushchk();

	switch(i->op) {
	default:
		snprint(buf, sizeof buf, "%s compile, no '%D'", mod->name, i);
		error(buf);
		break;

	/* Operations that punt to interpreter */
	case IMCALL:
		punt(i, SRCOP|DSTOP|THREOP|WRTPC|NEWPC, optab[i->op]);
		break;
	case ISEND:
	case IRECV:
	case IALT:
		punt(i, SRCOP|DSTOP|TCHECK|WRTPC, optab[i->op]);
		break;
	case ISPAWN:
		punt(i, SRCOP|DBRAN, optab[i->op]);
		break;
	case IBNEC:
	case IBEQC:
	case IBLTC:
	case IBLEC:
	case IBGTC:
	case IBGEC:
		punt(i, SRCOP|DBRAN|NEWPC|WRTPC, optab[i->op]);
		break;
	case ICASEC:
	case ICASEL:
		punt(i, SRCOP|DSTOP|NEWPC, optab[i->op]);
		break;
	case IADDC:
	case IMULL:
	case IDIVL:
	case IMODL:
	case IMNEWZ:
	case ILSRW:
	case ILSRL:
	case IMODW:
	case IMODB:
	case IDIVW:
	case IDIVB:
		punt(i, SRCOP|DSTOP|THREOP, optab[i->op]);
		break;
	case ILOAD:
	case INEWA:
	case INEWAZ:
	case INEW:
	case INEWZ:
	case ISLICEA:
	case ISLICELA:
	case ICONSB:
	case ICONSW:
	case ICONSL:
	case ICONSF:
	case ICONSM:
	case ICONSMP:
	case ICONSP:
	case IMOVMP:
	case IHEADMP:
	case IHEADB:
	case IHEADW:
	case IHEADL:
	case IINSC:
	case ICVTAC:
	case ICVTCW:
	case ICVTWC:
	case ICVTLC:
	case ICVTCL:
	case ICVTFC:
	case ICVTCF:
	case ICVTRF:
	case ICVTFR:
	case ICVTWS:
	case ICVTSW:
	case IMSPAWN:
	case ICVTCA:
	case ISLICEC:
	case INBALT:
		punt(i, SRCOP|DSTOP, optab[i->op]);
		break;
	case INEWCM:
	case INEWCMP:
		punt(i, SRCOP|DSTOP|THREOP, optab[i->op]);
		break;
	case IMFRAME:
		if(cflag > 2 && pass)
			print("IMFRAME: src add=0x%x UXSRC=0x%x s.ind=%lld add&ARM=0x%x reg=%d d.ind=%lld\n",
				i->add, UXSRC(i->add), (vlong)i->s.ind, i->add & ARM, i->reg, (vlong)i->d.ind);
		punt(i, SRCOP|DSTOP|THREOP, optab[i->op]);
		break;
	case INEWCB:
	case INEWCW:
	case INEWCF:
	case INEWCP:
	case INEWCL:
		punt(i, DSTOP|THREOP, optab[i->op]);
		break;
	case IEXIT:
		punt(i, 0, optab[i->op]);
		break;

	/* Floating point - punt for now */
	case IMOVF:
	case IADDF:
	case ISUBF:
	case IMULF:
	case IDIVF:
	case INEGF:
	case IBEQF:
	case IBNEF:
	case IBLTF:
	case IBLEF:
	case IBGTF:
	case IBGEF:
	case ICVTFW:
	case ICVTWF:
	case ICVTFL:
	case ICVTLF:
		punt(i, SRCOP|DSTOP, optab[i->op]);
		break;

	/* Type conversions */
	case ICVTBW:
		opwld(i, Ldb, RA0);
		emit(UXTB(RA0, RA0));
		opwst(i, Stw, RA0);
		break;
	case ICVTWB:
		opwld(i, Ldw, RA0);
		opwst(i, Stb, RA0);
		break;
	case ICVTWL:
		/* Sign-extend 32-bit to 64-bit */
		opwld(i, Ldw32, RA0);
		emit(SXTW(RA0, RA0));
		opwst(i, Stw, RA0);
		break;
	case ICVTLW:
		/* Truncate 64-bit to 32-bit (just load low word) */
		opwld(i, Ldw, RA0);
		opwst(i, Stw32, RA0);
		break;

	/* Data movement */
	case IMOVW:
		opwld(i, Ldw, RA0);
		opwst(i, Stw, RA0);
		break;
	case IMOVB:
		opwld(i, Ldb, RA0);
		opwst(i, Stb, RA0);
		break;
	case IMOVL:
		/* 64-bit move - same as MOVW on ARM64 */
		opwld(i, Ldw, RA0);
		opwst(i, Stw, RA0);
		break;
	case ILEA:
		opwld(i, Lea, RA0);
		opwst(i, Stw, RA0);
		break;

	/* Pointer operations */
	case IMOVP:
		opwld(i, Ldw, RA1);
		goto movp;
	case ITAIL:
		opwld(i, Ldw, RA0);
		CMPH(RA0);
		emit(BCOND(EQ, 5*4));  /* Skip to end if nil */
		mem(Ldw, O(List, tail), RA0, RA1);
		goto movp;
	case IHEADP:
		opwld(i, Ldw, RA0);
		CMPH(RA0);
		emit(BCOND(EQ, 5*4));
		mem(Ldw, OA(List, data), RA0, RA1);
	movp:
		/* Color pointer if not H */
		CMPH(RA1);
		{
			u32int *skipcolor = code;
			emit(BCOND(EQ, 0));  /* Skip color if nil */
			con((uvlong)(base + macro[MacCOLR]), RA0, 0);
			emit(BLR(RA0));
			if(pass)
				PATCH_BCOND(skipcolor, (vlong)code - (vlong)skipcolor);
		}
		/* Store new pointer */
		opwst(i, Lea, RA2);
		mem(Ldw, 0, RA2, RA0);  /* Load old value */
		mem(Stw, 0, RA2, RA1);  /* Store new value */
		/* Free old pointer */
		con((uvlong)(base + macro[MacFRP]), RA1, 0);
		emit(BLR(RA1));
		break;

	/* Arithmetic - Word */
	case IADDW:
		arith(i, 0);
		break;
	case ISUBW:
		arith(i, 1);
		break;
	case IMULW:
		opwld(i, Ldw, RA1);
		mid(i, Ldw, RA0);
		emit(MUL(RA0, RA0, RA1));
		opwst(i, Stw, RA0);
		break;

	/* Arithmetic - Byte */
	case IADDB:
		arithb(i, 0);
		break;
	case ISUBB:
		arithb(i, 1);
		break;
	case IMULB:
		opwld(i, Ldb, RA1);
		mid(i, Ldb, RA0);
		emit(MUL32(RA0, RA0, RA1));
		opwst(i, Stb, RA0);
		break;

	/* Logic - Word */
	case IANDW:
		logic(i, 0);
		break;
	case IORW:
		logic(i, 1);
		break;
	case IXORW:
		logic(i, 2);
		break;

	/* Logic - Byte */
	case IANDB:
		logicb(i, 0);
		break;
	case IORB:
		logicb(i, 1);
		break;
	case IXORB:
		logicb(i, 2);
		break;

	/* Shifts - Word */
	case ISHLW:
		shift(i, 0);
		break;
	case ISHRW:
		shift(i, 2);  /* Arithmetic shift */
		break;

	/* Shifts - Byte */
	case ISHLB:
		shiftb(i, 0);
		break;
	case ISHRB:
		shiftb(i, 1);  /* Logical shift for bytes */
		break;

	/* 64-bit operations */
	case IADDL:
		larithl(i, 0);
		break;
	case ISUBL:
		larithl(i, 1);
		break;
	case IANDL:
		larithl(i, 2);
		break;
	case IORL:
		larithl(i, 3);
		break;
	case IXORL:
		larithl(i, 4);
		break;
	case ISHLL:
		shiftl(i, 0);
		break;
	case ISHRL:
		shiftl(i, 2);
		break;

	/* Conditional branches - Word */
	case IBEQW:
		cbra(i, EQ);
		break;
	case IBNEW:
		cbra(i, NE);
		break;
	case IBLTW:
		cbra(i, LT);
		break;
	case IBLEW:
		cbra(i, LE);
		break;
	case IBGTW:
		cbra(i, GT);
		break;
	case IBGEW:
		cbra(i, GE);
		break;

	/* Conditional branches - Byte (unsigned) */
	case IBEQB:
		cbrab(i, EQ);
		break;
	case IBNEB:
		cbrab(i, NE);
		break;
	case IBLTB:
		cbrab(i, LO);  /* Unsigned */
		break;
	case IBLEB:
		cbrab(i, LS);
		break;
	case IBGTB:
		cbrab(i, HI);
		break;
	case IBGEB:
		cbrab(i, HS);
		break;

	/* Conditional branches - Long */
	case IBEQL:
		cbral(i, NE, EQ, ANDAND);
		break;
	case IBNEL:
		cbral(i, NE, NE, OROR);
		break;
	case IBLTL:
		cbral(i, LT, LO, EQAND);
		break;
	case IBLEL:
		cbral(i, LT, LS, EQAND);
		break;
	case IBGTL:
		cbral(i, GT, HI, EQAND);
		break;
	case IBGEL:
		cbral(i, GT, HS, EQAND);
		break;

	/* Control flow */
	case IJMP:
		if(RESCHED)
			schedcheck(i);
		{
			vlong dst = patch[i->d.ins - mod->prog];
			if(pass) {
				vlong off = ((vlong)base + dst) - (vlong)code;
				emit(B(off >> 2));
			} else {
				emit(B(0));
			}
		}
		break;

	case ICALL:
		if(UXDST(i->add) != DST(AIMM))
			opwst(i, Ldw, RTA);  /* Get call target */
		opwld(i, Ldw, RA0);         /* Get frame pointer */
		/* Store return address */
		{
			uvlong retpc = (uvlong)(base + patch[i - mod->prog + 1]);
			con(retpc, RA1, 0);  /* Must use opt=0: base differs between passes */
			mem(Stw, O(Frame, lr), RA0, RA1);
		}
		/* Store old FP */
		mem(Stw, O(Frame, fp), RA0, RFP);
		/* Update FP */
		emit(MOV_REG(RFP, RA0));
		/* Jump to target */
		if(UXDST(i->add) != DST(AIMM)) {
			emit(BR(RTA));
		} else {
			vlong dst = patch[i->d.ins - mod->prog];
			if(pass) {
				vlong off = (vlong)(base + dst) - (vlong)code;
				emit(B(off >> 2));
			} else {
				emit(B(0));
			}
		}
		break;

	case IRET:
		con((uvlong)(base + macro[MacRET]), RA0, 0);
		emit(BR(RA0));
		break;

	case IFRAME:
		if(UXSRC(i->add) != SRC(AIMM)) {
			punt(i, SRCOP|DSTOP, optab[i->op]);
			break;
		}
		if(cflag > 2 && pass)
			print("IFRAME: dst add=0x%x UXDST=0x%x d.ind=%lld d.i.f=%d d.i.s=%d\n",
				i->add, UXDST(i->add), (vlong)i->d.ind, i->d.i.f, i->d.i.s);
		tinit[i->s.imm] = 1;
		con((uvlong)mod->type[i->s.imm], RA3, 1);
		con((uvlong)(base + macro[MacFRAM]), RA0, 0);
		emit(BLR(RA0));
		opwst(i, Stw, RA2);
		break;

	/* Length operations */
	case ILENA:
		opwld(i, Ldw, RA0);
		emit(MOV_REG(RA1, XZR));
		CMPH(RA0);
		emit(BCOND(EQ, 2*4));
		mem(Ldw, O(Array, len), RA0, RA1);
		opwst(i, Stw, RA1);
		break;

	case ILENL:
		emit(MOV_REG(RA1, XZR));
		opwld(i, Ldw, RA0);
		{
			u32int *loop = code;
			vlong broff;
			CMPH(RA0);
			emit(BCOND(EQ, 3*4));
			mem(Ldw, O(List, tail), RA0, RA0);
			emit(ADD_IMM(RA1, RA1, 1));
			broff = ((vlong)loop - (vlong)code) >> 2;
			emit(B(broff));
		}
		opwst(i, Stw, RA1);
		break;

	case ICASE:
	case IGOTO:
		punt(i, SRCOP|DSTOP|NEWPC, optab[i->op]);
		break;

	/* Array operations - punt for now */
	case IINDX:
	case IINDW:
	case IINDB:
	case IINDF:
	case IINDL:
	case IINDC:
		punt(i, SRCOP|DSTOP|THREOP, optab[i->op]);
		break;

	/* Movm/Headm */
	case IMOVM:
	case IHEADM:
	case IHEADF:
		punt(i, SRCOP|DSTOP|THREOP, optab[i->op]);
		break;

	/* Other punt operations */
	case IRAISE:
		punt(i, SRCOP|WRTPC|NEWPC, optab[i->op]);
		break;
	case IMULX:
	case IDIVX:
	case ICVTXX:
	case IMULX0:
	case IDIVX0:
	case ICVTXX0:
	case IMULX1:
	case IDIVX1:
	case ICVTXX1:
	case ICVTFX:
	case ICVTXF:
	case IEXPW:
	case IEXPL:
	case IEXPF:
		punt(i, SRCOP|DSTOP|THREOP, optab[i->op]);
		break;
	case ISELF:
		punt(i, DSTOP, optab[i->op]);
		break;
	case ILENC:
		punt(i, SRCOP|DSTOP, optab[i->op]);
		break;
	case IMOVPC:
		{
			uvlong pc = (uvlong)(base + patch[i->s.imm]);
			con(pc, RA0, 0);  /* Must use opt=0: base differs between passes */
			opwst(i, Stw, RA0);
		}
		break;
	}
}

/*
 * Generate preamble - entry point for JIT'd code
 */
static void
preamble(void)
{
	u32int *codestart;
	int codesize;

	if(comvec)
		return;

	/* Allocate space for preamble code - must be executable */
#ifdef __APPLE__
	comvec = mmap(0, 4096, PROT_READ | PROT_WRITE | PROT_EXEC,
	              MAP_PRIVATE | MAP_ANON | MAP_JIT, -1, 0);
	if(comvec == MAP_FAILED) {
		comvec = nil;
		error(exNomem);
	}
	pthread_jit_write_protect_np(0);  /* Enable writing */
#else
	comvec = malloc(64 * sizeof(u32int));
	if(comvec == nil)
		error(exNomem);
#endif
	code = (u32int*)comvec;
	codestart = code;

	/* X9-X12 are caller-saved - no need to save them in preamble */
	/* Caller (xec.c) is responsible for saving if needed */

	/* Load VM state pointer (&R) into RREG (X11) using MOVZ/MOVK */
	{
		uvlong raddr = (uvlong)&R;
		if(cflag > 0)
			print("ARM64 JIT Preamble: Using caller-saved regs X9-X12, &R=%p\n", &R);
		emit(MOVZ(RREG, raddr & 0xFFFF, 0));
		emit(MOVK(RREG, (raddr >> 16) & 0xFFFF, 1));
		emit(MOVK(RREG, (raddr >> 32) & 0xFFFF, 2));
		emit(MOVK(RREG, (raddr >> 48) & 0xFFFF, 3));
	}

	/* Save return address (X30/LR) to R.xpc for later return */
	emit(STR_UOFF(X30, RREG, O(REG, xpc)));

	/* Load VM state from R struct into X9-X12 */
	emit(LDR_UOFF(RFP, RREG, O(REG, FP)));   /* X9 = R.FP */
	emit(LDR_UOFF(RMP, RREG, O(REG, MP)));   /* X10 = R.MP */
	emit(LDR_UOFF(RM, RREG, O(REG, M)));     /* X12 = R.M */

	/* Jump to compiled code via R.PC */
	emit(LDR_UOFF(RA0, RREG, O(REG, PC)));
	emit(BR(RA0));

	codesize = (code - codestart) * sizeof(u32int);
#ifdef __APPLE__
	pthread_jit_write_protect_np(1);  /* Enable execution */
	sys_icache_invalidate(comvec, codesize);
#else
	segflush(comvec, codesize);
#endif

	/* Debug: print preamble code and struct sizes */
	if(cflag > 0) {
		int j;
		print("Preamble at %p (%d words):\n", comvec, (int)(code - codestart));
		print("  sizeof(WORD)=%d sizeof(Modl)=%d sizeof(Modlink)=%d\n",
			(int)sizeof(WORD), (int)sizeof(Modl), (int)sizeof(Modlink));
		print("  O(REG,FP)=%ld O(REG,MP)=%ld O(REG,M)=%ld O(REG,PC)=%ld\n",
			O(REG, FP), O(REG, MP), O(REG, M), O(REG, PC));
		for(j = 0; j < (code - codestart) && j < 20; j++) {
			print("  [%2d] %p: %08lx\n", j, &codestart[j], (ulong)codestart[j]);
		}
	}
}

/*
 * Macro: Free pointer (decrement reference count)
 */
static void
macfrp(void)
{
	u32int *lnil, *lnz;

	/* Input: RA0 = pointer to free */
	/* Check for nil */
	CMPH(RA0);
	emit(BCOND(EQ, 0));  /* Placeholder: return if nil */
	lnil = code - 1;

	/* Decrement reference count (ref is ulong = 64-bit on ARM64) */
	mem(Ldw, O(Heap, ref) - sizeof(Heap), RA0, RA1);
	emit(SUB_IMM(RA1, RA1, 1));
	mem(Stw, O(Heap, ref) - sizeof(Heap), RA0, RA1);

	/* If still > 0, return */
	emit(CBNZ(RA1, 0));  /* Placeholder */
	lnz = code - 1;

	/* ref == 0, need to destroy */
	mem(Stw, O(REG, FP), RREG, RFP);
	mem(Stw, O(REG, st), RREG, X30);   /* Save link register (BLR will clobber) */
	mem(Stw, O(REG, s), RREG, RA0);

	/* Save VM registers before C call (caller-saved) */
	emit(STP_PRE(RFP, RMP, SP, -32));
	emit(STP(RREG, RM, SP, 16));

	con((uvlong)rdestroy, RA1, 0);
	emit(BLR(RA1));

	/* Restore VM registers */
	emit(LDP(RREG, RM, SP, 16));
	emit(LDP_POST(RFP, RMP, SP, 32));

	mem(Ldw, O(REG, st), RREG, X30);
	mem(Ldw, O(REG, FP), RREG, RFP);
	mem(Ldw, O(REG, MP), RREG, RMP);
	mem(Ldw, O(REG, M), RREG, RM);

	/* Patch branches to point to RET */
	PATCH_BCOND(lnil, (vlong)code - (vlong)lnil);
	PATCH_CBNZ(lnz, (vlong)code - (vlong)lnz);

	emit(RET_LR);
	flushcon(0);
}

/*
 * Macro: Return from function
 */
static void
macret(void)
{
	u32int *lpunt, *lnomr, *lfrmr, *linterp;

	/* Check if frame has type */
	mem(Ldw, O(Frame, t), RFP, RA0);
	emit(CBZ(RA0, 0));
	lpunt = code - 1;

	/* Check if type has destroy function */
	mem(Ldw, O(Type, destroy), RA0, RA1);
	emit(CBZ(RA1, 0));
	PATCH_BCOND(lpunt, (vlong)code - (vlong)lpunt);
	lpunt = code - 1;

	/* Check if we have a saved frame pointer */
	mem(Ldw, O(Frame, fp), RFP, RA2);
	emit(CBZ(RA2, 0));
	PATCH_BCOND(lpunt, (vlong)code - (vlong)lpunt);
	lpunt = code - 1;

	/* Check if we have a saved module reference */
	mem(Ldw, O(Frame, mr), RFP, RA3);
	emit(CBZ(RA3, 0));
	lnomr = code - 1;

	/* Decrement old module refcount (ref is ulong = 64-bit on ARM64) */
	mem(Ldw, O(REG, M), RREG, RTA);
	mem(Ldw, O(Heap, ref) - sizeof(Heap), RTA, RA0);
	emit(SUB_IMM(RA0, RA0, 1));
	mem(Stw, O(Heap, ref) - sizeof(Heap), RTA, RA0);
	emit(CBNZ(RA0, 0));
	lfrmr = code - 1;

	/* Need to keep module alive */
	emit(ADD_IMM(RA0, RA0, 1));
	mem(Stw, O(Heap, ref) - sizeof(Heap), RTA, RA0);
	emit(B(0));
	PATCH_BCOND(lpunt, (vlong)code - (vlong)lpunt);
	lpunt = code - 1;

	/* Restore module context */
	PATCH_BCOND(lfrmr, (vlong)code - (vlong)lfrmr);
	mem(Ldw, O(Frame, mr), RFP, RTA);
	mem(Stw, O(REG, M), RREG, RTA);
	mem(Ldw, O(Modlink, MP), RTA, RMP);
	mem(Stw, O(REG, MP), RREG, RMP);

	/* Check if compiled */
	mem(Ldw32, O(Modlink, compiled), RTA, RA0);
	emit(CBZ32(RA0, 0));
	linterp = code - 1;

	/* Compiled - call destroy and return */
	PATCH_BCOND(lnomr, (vlong)code - (vlong)lnomr);
	mem(Ldw, O(Frame, t), RFP, RA0);
	mem(Ldw, O(Type, destroy), RA0, RA0);

	/* Save VM registers before destroy call */
	emit(STP_PRE(RFP, RMP, SP, -32));
	emit(STP(RREG, RM, SP, 16));
	emit(BLR(RA0));
	emit(LDP(RREG, RM, SP, 16));
	emit(LDP_POST(RFP, RMP, SP, 32));

	mem(Stw, O(REG, SP), RREG, RFP);
	mem(Ldw, O(Frame, lr), RFP, RA0);
	mem(Ldw, O(Frame, fp), RFP, RFP);
	mem(Stw, O(REG, FP), RREG, RFP);
	emit(BR(RA0));

	/* Return to interpreter */
	PATCH_BCOND(linterp, (vlong)code - (vlong)linterp);
	mem(Ldw, O(Frame, t), RFP, RA0);
	mem(Ldw, O(Type, destroy), RA0, RA0);

	/* Save VM registers before destroy call */
	emit(STP_PRE(RFP, RMP, SP, -32));
	emit(STP(RREG, RM, SP, 16));
	emit(BLR(RA0));
	emit(LDP(RREG, RM, SP, 16));
	emit(LDP_POST(RFP, RMP, SP, 32));

	mem(Stw, O(REG, SP), RREG, RFP);
	mem(Ldw, O(Frame, lr), RFP, RA0);
	mem(Stw, O(REG, PC), RREG, RA0);
	mem(Ldw, O(Frame, fp), RFP, RFP);
	mem(Stw, O(REG, FP), RREG, RFP);
	/* Restore LR from R.xpc and return to caller */
	mem(Ldw, O(REG, xpc), RREG, X30);
	emit(RET_LR);

	/* Punt to interpreter */
	PATCH_B(lpunt, (vlong)code - (vlong)lpunt);
	punt(&(Inst){.add = AXNON}, TCHECK|NEWPC, optab[IRET]);
}

/*
 * Macro: Case dispatch (binary search)
 */
static void
maccase(void)
{
	/* Simplified - punt to interpreter */
	punt(&(Inst){.add = AXNON}, SRCOP|DSTOP|NEWPC, optab[ICASE]);
}

/*
 * Macro: Color pointer for GC
 */
static void
maccolr(void)
{
	u32int *lskip;

	/* Input: RA1 = pointer to color */
	/* Increment reference count (ref is ulong = 64-bit on ARM64) */
	mem(Ldw, O(Heap, ref) - sizeof(Heap), RA1, RA0);
	emit(ADD_IMM(RA0, RA0, 1));
	mem(Stw, O(Heap, ref) - sizeof(Heap), RA1, RA0);

	/* Check color */
	con((uvlong)&mutator, RA0, 1);
	mem(Ldw32, 0, RA0, RA0);
	mem(Ldw32, O(Heap, color) - sizeof(Heap), RA1, RA2);
	emit(CMP_REG32(RA2, RA0));
	emit(BCOND(EQ, 0));  /* Placeholder: skip to RET if h->color == mutator */
	lskip = code - 1;

	/* Set propagator color */
	con(propagator, RA0, 1);
	mem(Stw32, O(Heap, color) - sizeof(Heap), RA1, RA0);
	con((uvlong)&nprop, RA2, 1);
	con(1, RA0, 1);
	mem(Stw32, 0, RA2, RA0);

	/* Patch branch to skip here */
	PATCH_BCOND(lskip, (vlong)code - (vlong)lskip);
	emit(RET_LR);
	flushcon(0);
}

/*
 * Macro: Module call
 */
static void
macmcal(void)
{
	mem(Stw, O(REG, FP), RREG, RFP);
	mem(Stw, O(REG, st), RREG, X30);

	/* Save VM registers before C call */
	emit(STP_PRE(RFP, RMP, SP, -32));
	emit(STP(RREG, RM, SP, 16));

	con((uvlong)rmcall, RA0, 1);
	emit(BLR(RA0));

	/* Restore VM registers */
	emit(LDP(RREG, RM, SP, 16));
	emit(LDP_POST(RFP, RMP, SP, 32));

	mem(Ldw, O(REG, st), RREG, X30);
	mem(Ldw, O(REG, FP), RREG, RFP);
	mem(Ldw, O(REG, MP), RREG, RMP);
	mem(Ldw, O(REG, M), RREG, RM);
	emit(RET_LR);
	flushcon(0);
}

/*
 * Macro: Frame allocation
 */
static void
macfram(void)
{
	u32int *overflow, *noinitskip;

	/* Input: RA3 = type pointer */
	/* Save link register upfront (may be clobbered by initializer or extend) */
	mem(Stw, O(REG, st), RREG, X30);

	/* Calculate new SP */
	mem(Ldw, O(REG, SP), RREG, RA0);
	mem(Ldw32, O(Type, size), RA3, RA1);
	emit(ADD_REG(RA0, RA0, RA1));

	/* Check for overflow */
	mem(Ldw, O(REG, TS), RREG, RA1);
	emit(CMP_REG(RA0, RA1));
	emit(BCOND(HS, 0));
	overflow = code - 1;

	/* No overflow - allocate frame */
	mem(Ldw, O(REG, SP), RREG, RA2);  /* RA2 = new frame */
	mem(Stw, O(REG, SP), RREG, RA0);  /* Update SP */
	mem(Stw, O(Frame, t), RA2, RA3);  /* f->t = type */
	con(0, RA0, 1);
	mem(Stw, O(Frame, mr), RA2, RA0); /* f->mr = nil */

	/* Call initializer if present */
	mem(Ldw, O(Type, initialize), RA3, RA0);
	emit(CBZ(RA0, 0));
	noinitskip = code - 1;

	/* Save VM registers before initializer call */
	emit(STP_PRE(RFP, RMP, SP, -32));
	emit(STP(RREG, RM, SP, 16));
	emit(BLR(RA0));
	emit(LDP(RREG, RM, SP, 16));
	emit(LDP_POST(RFP, RMP, SP, 32));

	PATCH_CBZ(noinitskip, (vlong)code - (vlong)noinitskip);

	/* Restore link register and return */
	mem(Ldw, O(REG, st), RREG, X30);
	emit(RET_LR);
	flushcon(0);

	/* Overflow - call extend */
	PATCH_BCOND(overflow, (vlong)code - (vlong)overflow);
	mem(Stw, O(REG, s), RREG, RA3);
	mem(Stw, O(REG, FP), RREG, RFP);

	/* Save VM registers before extend call */
	emit(STP_PRE(RFP, RMP, SP, -32));
	emit(STP(RREG, RM, SP, 16));

	con((uvlong)extend, RA0, 1);
	emit(BLR(RA0));

	/* Restore VM registers */
	emit(LDP(RREG, RM, SP, 16));
	emit(LDP_POST(RFP, RMP, SP, 32));

	mem(Ldw, O(REG, st), RREG, X30);
	mem(Ldw, O(REG, FP), RREG, RFP);
	mem(Ldw, O(REG, MP), RREG, RMP);
	mem(Ldw, O(REG, M), RREG, RM);
	mem(Ldw, O(REG, s), RREG, RA2);
	emit(RET_LR);
	flushcon(0);
}

/*
 * Macro: Module frame allocation
 */
static void
macmfra(void)
{
	mem(Stw, O(REG, FP), RREG, RFP);
	mem(Stw, O(REG, st), RREG, X30);

	/* Save VM registers before C call */
	emit(STP_PRE(RFP, RMP, SP, -32));
	emit(STP(RREG, RM, SP, 16));

	con((uvlong)rmfram, RA0, 1);
	emit(BLR(RA0));

	/* Restore VM registers */
	emit(LDP(RREG, RM, SP, 16));
	emit(LDP_POST(RFP, RMP, SP, 32));

	mem(Ldw, O(REG, st), RREG, X30);
	mem(Ldw, O(REG, FP), RREG, RFP);
	mem(Ldw, O(REG, MP), RREG, RMP);
	mem(Ldw, O(REG, M), RREG, RM);
	emit(RET_LR);
	flushcon(0);
}

/*
 * Macro: Reschedule
 * Called when IC <= 0 and we need to return to interpreter.
 * Save current JIT position to R.PC, then return to R.xpc (interpreter).
 */
static void
macrelq(void)
{
	mem(Stw, O(REG, FP), RREG, RFP);   /* R.FP = RFP */
	/* Save return address (current JIT position) as PC */
	emit(MOV_REG(RA0, X30));
	mem(Stw, O(REG, PC), RREG, RA0);   /* R.PC = return address */
	/* Load R.xpc (interpreter exit point) into X30 and return */
	mem(Ldw, O(REG, xpc), RREG, X30);
	emit(RET_LR);
}

/*
 * Compile type initializer/destroyer
 */
static void
comd(Type *t)
{
	int i, j, m, c;

	for(i = 0; i < t->np; i++) {
		c = t->map[i];
		j = i * 8 * sizeof(WORD);  /* Each map byte covers 8 WORD-sized slots */
		for(m = 0x80; m != 0; m >>= 1) {
			if(c & m) {
				mem(Ldw, j, RFP, RA0);
				con((uvlong)(base + macro[MacFRP]), RA1, 0);
				emit(BLR(RA1));
			}
			j += sizeof(WORD);
		}
	}
	emit(RET_LR);
	flushcon(0);
}

static void
comi(Type *t)
{
	int i, j, m, c;

	con((uvlong)H, RA0, 1);
	for(i = 0; i < t->np; i++) {
		c = t->map[i];
		j = i * 8 * sizeof(WORD);  /* Each map byte covers 8 WORD-sized slots */
		for(m = 0x80; m != 0; m >>= 1) {
			if(c & m)
				mem(Stw, j, RA2, RA0);
			j += sizeof(WORD);
		}
	}
	emit(RET_LR);
	flushcon(0);
}

void
typecom(Type *t)
{
	int n;
	u32int *tmp, *start;
	size_t codesize;

	if(t == nil || t->initialize != 0)
		return;

	tmp = mallocz(4096 * sizeof(u32int), 0);
	if(tmp == nil)
		error(exNomem);

	/* Measure initialize */
	code = tmp;
	comi(t);
	n = code - tmp;

	/* Measure destroy */
	code = tmp;
	comd(t);
	n += code - tmp;

	free(tmp);

	/* Allocate executable memory for type functions */
	codesize = n * sizeof(u32int);
#ifdef __APPLE__
	code = mmap(0, codesize, PROT_READ | PROT_WRITE | PROT_EXEC,
	            MAP_PRIVATE | MAP_ANON | MAP_JIT, -1, 0);
	if(code == MAP_FAILED)
		return;
	pthread_jit_write_protect_np(0);  /* Enable writing */
#else
	code = mallocz(codesize, 0);
	if(code == nil)
		return;
#endif

	start = code;
	t->initialize = code;
	comi(t);
	t->destroy = code;
	comd(t);

#ifdef __APPLE__
	pthread_jit_write_protect_np(1);  /* Enable execution */
	sys_icache_invalidate(start, codesize);
#else
	segflush(start, codesize);
#endif

	if(cflag > 3)
		print("typ= %.16p %4d i %.16p d %.16p asm=%d\n",
			t, t->size, t->initialize, t->destroy, n);
}

static void
patchex(Module *m, u32int *p)
{
	Handler *h;
	Except *e;

	if((h = m->htab) == nil)
		return;
	for(; h->etab != nil; h++) {
		h->pc1 = p[h->pc1];
		h->pc2 = p[h->pc2];
		for(e = h->etab; e->s != nil; e++)
			e->pc = p[e->pc];
		if(e->pc != (ulong)-1)
			e->pc = p[e->pc];
	}
}

/*
 * Main compile function
 */
int
compile(Module *m, int size, Modlink *ml)
{
	Link *l;
	Modl *e;
	int i, n;
	u32int *s, *tmp;

	base = nil;
	patch = mallocz(size * sizeof(*patch), 0);
	tinit = malloc(m->ntype * sizeof(*tinit));
	tmp = malloc(4096 * sizeof(u32int));
	if(tinit == nil || patch == nil || tmp == nil)
		goto bad;

	preamble();

	mod = m;
	n = 0;
	pass = 0;
	nlit = 0;
	rcon.ptr = 0;
	/* Initialize litpool to placeholder for pass 0 (inferno64 leaves it uninitialized!) */
	/* Use address similar to runtime to ensure consistent code generation */
	{
		static u32int placeholder_pool[256];
		litpool = placeholder_pool;
	}

	/* Pass 0: measure code size */
	for(i = 0; i < size; i++) {
		codeoff = n;
		code = tmp;
		comp(&m->prog[i]);
		patch[i] = n;
		n += code - tmp;
	}

	/* Generate macros */
	for(i = 0; i < nelem(mactab); i++) {
		codeoff = n;
		code = tmp;
		if(cflag > 3)
			print("pass0 BEFORE %s: n=%d code=%p tmp=%p rcon.ptr=%d\n",
				mactab[i].name, n, code, tmp, rcon.ptr);
		mactab[i].gen();
		if(cflag > 3)
			print("pass0 AFTER %s: n=%d code=%p tmp=%p size=%ld rcon.ptr=%d\n",
				mactab[i].name, n, code, tmp, (long)(code - tmp), rcon.ptr);
		macro[mactab[i].idx] = n;
		n += code - tmp;
	}

	/* Flush remaining constants */
	code = tmp;
	flushcon(0);
	n += code - tmp;

	/* Allocate final code buffer */
#ifdef __APPLE__
	base = mmap(0, (n + nlit) * sizeof(*code),
	            PROT_READ | PROT_WRITE | PROT_EXEC,
	            MAP_PRIVATE | MAP_ANON | MAP_JIT, -1, 0);
	if(base == MAP_FAILED) {
		base = nil;
		goto bad;
	}
	pthread_jit_write_protect_np(0);  /* Enable writing */
#else
	base = mallocz((n + nlit) * sizeof(*code), 0);
	if(base == nil)
		goto bad;
#endif

	if(cflag > 3)
		print("dis=%5d %5d arm64=%5d asm=%.16p: %s\n",
			size, (int)(size * sizeof(Inst)), n, base, m->name);

	/* Pass 1: generate code */
	pass++;
	nlit = 0;
	rcon.ptr = 0;
	litpool = base + n;
	code = base;
	n = 0;
	codeoff = 0;

	for(i = 0; i < size; i++) {
		s = code;
		comp(&m->prog[i]);
		if(patch[i] != n) {
			print("%3d %D\n", i, &m->prog[i]);
			print("%lu != %d\n", patch[i], n);
			urk("phase error");
		}
		n += code - s;
		if(cflag > 4) {
			int j;
			print("%3d %D (offset %d, %d words)\n", i, &m->prog[i], n, (int)(code - s));
			for(j = 0; j < code - s; j++) {
				print("  %08lux\n", (ulong)s[j]);
			}
		}
	}

	/* Generate macros */
	for(i = 0; i < nelem(mactab); i++) {
		s = code;
		if(cflag > 3)
			print("pass1 BEFORE %s: n=%d code=%p s=%p rcon.ptr=%d\n",
				mactab[i].name, n, code, s, rcon.ptr);
		mactab[i].gen();
		if(cflag > 3)
			print("pass1 AFTER %s: n=%d code=%p s=%p size=%ld expected_offset=%lu rcon.ptr=%d\n",
				mactab[i].name, n, code, s, (long)(code - s), macro[mactab[i].idx], rcon.ptr);
		if(macro[mactab[i].idx] != n) {
			print("mac phase err: %s expected=%lu got=%d diff=%ld\n",
				mactab[i].name, macro[mactab[i].idx], n,
				(long)(n - macro[mactab[i].idx]));
			urk("phase error");
		}
		n += code - s;
		if(cflag > 4) {
			print("%s:\n", mactab[i].name);
			das(s, code - s);
		}
	}

	/* Flush remaining constants */
	s = code;
	flushcon(0);
	n += code - s;

	/* Patch external links */
	for(l = m->ext; l->name; l++) {
		l->u.pc = (Inst*)RELPC(patch[l->u.pc - m->prog]);
		typecom(l->frame);
	}

	if(ml != nil) {
		e = &ml->links[0];
		for(i = 0; i < ml->nlinks; i++) {
			e->u.pc = (Inst*)RELPC(patch[e->u.pc - m->prog]);
			typecom(e->frame);
			e++;
		}
	}

	for(i = 0; i < m->ntype; i++) {
		if(tinit[i] != 0)
			typecom(m->type[i]);
	}

	patchex(m, patch);
	m->entry = (Inst*)RELPC(patch[mod->entry - mod->prog]);

	free(patch);
	free(tinit);
	free(tmp);
	free(m->prog);
	m->prog = (Inst*)base;
	m->compiled = 1;

#ifdef __APPLE__
	pthread_jit_write_protect_np(1);  /* Enable execution */
	sys_icache_invalidate(base, n * sizeof(*base));
#else
	segflush(base, n * sizeof(*base));
#endif

	if(cflag > 3) {
		int j;
		print("JIT code at %p (first 30 words):\n", base);
		for(j = 0; j < 30 && j < n; j++) {
			print("  [%3d] %p: %08lux\n", j, &base[j], (ulong)base[j]);
		}
	}

	return 1;

bad:
	free(patch);
	free(tinit);
	free(tmp);
#ifdef __APPLE__
	if(base != nil && base != MAP_FAILED)
		munmap(base, (n + nlit) * sizeof(*code));
#else
	free(base);
#endif
	return 0;
}
