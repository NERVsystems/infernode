/*
 * Disassembler stub for arm64
 * No JIT compiler, so no disassembly needed
 */
#include <lib9.h>
#include <kernel.h>

void
das(uchar *x, int n)
{
	USED(x);
	USED(n);
}
