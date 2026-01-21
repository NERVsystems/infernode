/*
 * Stub JIT compiler for arm64
 * Falls back to interpreter mode
 */
#include "lib9.h"
#include "isa.h"
#include "interp.h"
#include "raise.h"

void	(*comvec)(void);

void
typecom(Type *t)
{
	USED(t);
}

int
compile(Module *m, int size, Modlink *ml)
{
	USED(m);
	USED(size);
	USED(ml);
	return 0;	/* JIT not available, use interpreter */
}
