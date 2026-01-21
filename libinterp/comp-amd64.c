/*
 * JIT compilation stub for x86_64 (amd64)
 *
 * This stub disables JIT compilation, causing the emulator to fall back
 * to the portable interpreter. A full JIT compiler for x86_64 would need
 * to generate proper 64-bit machine code with REX prefixes, 64-bit addressing,
 * and the System V AMD64 ABI calling convention.
 *
 * The interpreter fallback is fully functional but slower than JIT.
 */

#include "lib9.h"
#include "isa.h"
#include "interp.h"

void	(*comvec)(void);

/*
 * compile - JIT compile a module
 * Returns 0 to indicate JIT is not available, causing fallback to interpreter
 */
int
compile(Module *m, int size, Modlink *ml)
{
	USED(m);
	USED(size);
	USED(ml);
	return 0;	/* No JIT - use interpreter */
}

/*
 * typecom - compile type initialization/destruction
 * No-op stub since we're not doing JIT
 */
void
typecom(Type *t)
{
	USED(t);
}
