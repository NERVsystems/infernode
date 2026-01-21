#include <stdio.h>
#include <stddef.h>

/* Copy of REG struct from interp.h */
typedef long WORD;
typedef struct Inst Inst;
typedef struct Modlink Modlink;

struct REG
{
	Inst*		PC;		/* Program counter */
	unsigned char*		MP;		/* Module data */
	unsigned char*		FP;		/* Frame pointer */
	unsigned char*		SP;		/* Stack pointer */
	unsigned char*		TS;		/* Top of allocated stack */
	unsigned char*		EX;		/* Extent register */
	Modlink*	M;		/* Module */
	int		IC;		/* Instruction count for this quanta */
	Inst*		xpc;		/* Saved program counter */
	void*		s;		/* Source */
	void*		d;		/* Destination */
	void*		m;		/* Middle */
	WORD		t;		/* Middle temporary */
	WORD		st;		/* Source temporary */
	WORD		dt;		/* Destination temporary */
};

int main() {
	printf("sizeof(WORD) = %zu\n", sizeof(WORD));
	printf("sizeof(void*) = %zu\n", sizeof(void*));
	printf("sizeof(int) = %zu\n", sizeof(int));
	printf("sizeof(struct REG) = %zu\n", sizeof(struct REG));
	printf("\nREG field offsets:\n");
	printf("PC:  %zu\n", offsetof(struct REG, PC));
	printf("MP:  %zu\n", offsetof(struct REG, MP));
	printf("FP:  %zu\n", offsetof(struct REG, FP));
	printf("SP:  %zu\n", offsetof(struct REG, SP));
	printf("TS:  %zu\n", offsetof(struct REG, TS));
	printf("EX:  %zu\n", offsetof(struct REG, EX));
	printf("M:   %zu\n", offsetof(struct REG, M));
	printf("IC:  %zu\n", offsetof(struct REG, IC));
	printf("xpc: %zu\n", offsetof(struct REG, xpc));
	printf("s:   %zu\n", offsetof(struct REG, s));
	printf("d:   %zu\n", offsetof(struct REG, d));
	printf("m:   %zu\n", offsetof(struct REG, m));
	printf("t:   %zu\n", offsetof(struct REG, t));
	printf("st:  %zu\n", offsetof(struct REG, st));
	printf("dt:  %zu\n", offsetof(struct REG, dt));
	return 0;
}
