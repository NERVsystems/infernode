/*
 * Disassembler stub for arm64
 * ARM64 instructions are 32-bit, so the code pointer is u32int*
 */
#include <lib9.h>
#include <kernel.h>

void
das(u32int *x, int n)
{
	USED(x);
	USED(n);
}
