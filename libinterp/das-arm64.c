/*
 * Disassembler for arm64 JIT — hex dump of 32-bit instruction words
 */
#include <lib9.h>
#include <kernel.h>

void
das(u32int *x, int n)
{
	int i;

	for(i = 0; i < n; i++)
		print("  %.8p  %.8ux\n", &x[i], x[i]);
}
