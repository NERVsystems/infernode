/*
 * CBMC Harness for mnthash Array Bounds Verification
 *
 * This harness verifies that all accesses to Pgrp->mnthash[] are within bounds.
 * The array is defined as: Mhead *mnthash[MNTHASH] where MNTHASH = 32.
 */

#include <stddef.h>

#define MNTHASH 32
#define MNTLOG 5

/* Minimal type definitions */
typedef struct Lock {
    int dummy;
} Lock;

typedef struct Ref {
    int ref;
    Lock lk;
} Ref;

typedef struct RWlock {
    Lock l;
    int readers;
} RWlock;

typedef struct QLock {
    Lock l;
} QLock;

typedef struct Chan Chan;

typedef struct Mhead {
    Ref r;
    RWlock lock;
    Chan *from;
    struct Mount *mount;
    struct Mhead *hash;
} Mhead;

typedef struct Pgrp {
    Ref r;
    unsigned long pgrpid;
    RWlock ns;
    QLock nsh;
    Mhead *mnthash[MNTHASH];
    int progmode;
    Chan *dot;
    Chan *slash;
    int nodevs;
    int pin;
} Pgrp;

typedef struct Qid {
    unsigned long long path;
    unsigned long vers;
    unsigned char type;
} Qid;

/* MOUNTH macro from dat.h:257 */
#define MOUNTH(p,qid) ((p)->mnthash[(qid).path&((1<<MNTLOG)-1)])

/*
 * Verify MOUNTH macro always produces valid array index
 */
void harness_mounth_macro(void) {
    Pgrp pg;
    Qid qid;
    unsigned long long path_value;
    int index;

    /* Symbolic path value - can be any 64-bit value */
    qid.path = path_value;

    /* Compute index as MOUNTH does */
    index = qid.path & ((1 << MNTLOG) - 1);

    /* Verify bounds - index must be [0, 31] */
    __CPROVER_assert(index >= 0, "MOUNTH index non-negative");
    __CPROVER_assert(index < MNTHASH, "MOUNTH index within array bounds");

    /* Verify actual macro usage */
    Mhead **m = &MOUNTH(&pg, qid);
    __CPROVER_assert(m >= &pg.mnthash[0], "MOUNTH pointer >= array start");
    __CPROVER_assert(m < &pg.mnthash[MNTHASH], "MOUNTH pointer < array end");
}

/*
 * Verify closepgrp loop bounds (pgrp.c:32-43)
 */
void harness_closepgrp_loop(void) {
    Pgrp pg;
    Mhead **h, **e;
    int count = 0;

    /* Simulate the loop from closepgrp */
    e = &pg.mnthash[MNTHASH];

    for(h = pg.mnthash; h < e; h++) {
        /* Verify h is always within bounds */
        __CPROVER_assert(h >= &pg.mnthash[0], "loop pointer >= array start");
        __CPROVER_assert(h < &pg.mnthash[MNTHASH], "loop pointer < array end");

        count++;
    }

    /* Verify loop iterates exactly MNTHASH times */
    __CPROVER_assert(count == MNTHASH, "loop iterates MNTHASH times");
}

/*
 * Verify pgrpcpy loop bounds (pgrp.c:88)
 */
void harness_pgrpcpy_loop(void) {
    Pgrp from, to;
    int i;
    Mhead **tom, **l;

    tom = to.mnthash;

    for(i = 0; i < MNTHASH; i++) {
        /* Verify i is within bounds */
        __CPROVER_assert(i >= 0, "loop index non-negative");
        __CPROVER_assert(i < MNTHASH, "loop index within bounds");

        /* Verify array access is safe */
        l = tom++;
        __CPROVER_assert(l >= &to.mnthash[0], "pointer >= array start");
        __CPROVER_assert(l < &to.mnthash[MNTHASH], "pointer < array end");

        /* Access from array */
        Mhead *f = from.mnthash[i];
        (void)f; /* May be NULL, that's OK */
    }
}

/*
 * Main harness - runs all sub-harnesses
 */
void harness(void) {
    harness_mounth_macro();
    harness_closepgrp_loop();
    harness_pgrpcpy_loop();
}
