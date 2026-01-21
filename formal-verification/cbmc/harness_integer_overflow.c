/*
 * CBMC Harness for Integer Overflow Verification
 *
 * Verifies that integer arithmetic operations in fd allocation
 * do not overflow.
 */

#include <limits.h>

#define DELTAFD 20
#define MAXNFD 4000

/*
 * Verify newfgrp fd allocation (pgrp.c:146)
 *
 * Code: n = (old->maxfd+1 + DELTAFD-1)/DELTAFD * DELTAFD;
 *
 * Potential overflow: old->maxfd + 1
 */
void harness_newfgrp_alloc(void) {
    int maxfd;
    int n;

    /* Symbolic maxfd - can be any int value */
    __CPROVER_assume(maxfd >= 0);          /* Assume reasonable starting state */
    __CPROVER_assume(maxfd <= MAXNFD);     /* Assume within system limit */

    /* Check that maxfd+1 doesn't overflow */
    __CPROVER_assert(maxfd < INT_MAX, "maxfd+1 will not overflow");

    /* If maxfd >= DELTAFD, perform the calculation */
    if(maxfd >= DELTAFD) {
        /* The actual calculation from the code */
        int temp1 = maxfd + 1;
        __CPROVER_assert(temp1 > 0, "maxfd+1 positive");

        int temp2 = temp1 + DELTAFD - 1;
        __CPROVER_assert(temp2 > 0, "intermediate calculation positive");

        int temp3 = temp2 / DELTAFD;
        __CPROVER_assert(temp3 > 0, "division result positive");

        n = temp3 * DELTAFD;
        __CPROVER_assert(n > 0, "final allocation size positive");
        __CPROVER_assert(n >= maxfd, "allocated size >= maxfd");
    } else {
        n = DELTAFD;
    }

    /* Verify allocation size is reasonable */
    __CPROVER_assert(n > 0, "allocation size positive");
    __CPROVER_assert(n <= MAXNFD + DELTAFD, "allocation size bounded");
}

/*
 * Verify dupfgrp fd allocation (pgrp.c:173)
 *
 * Same calculation as newfgrp
 */
void harness_dupfgrp_alloc(void) {
    int maxfd;
    int n;

    /* Symbolic maxfd */
    __CPROVER_assume(maxfd >= 0);
    __CPROVER_assume(maxfd <= MAXNFD);

    /* Perform calculation */
    if(maxfd >= DELTAFD)
        n = (maxfd + 1 + DELTAFD - 1) / DELTAFD * DELTAFD;
    else
        n = DELTAFD;

    /* Post-conditions */
    __CPROVER_assert(n > 0, "allocation size positive");
    __CPROVER_assert(n >= maxfd, "allocation size >= maxfd");
    __CPROVER_assert(n % DELTAFD == 0, "allocation size is multiple of DELTAFD");
}

/*
 * Verify array access bounds in dupfgrp (pgrp.c:184)
 *
 * Code: for(i = 0; i <= f->maxfd; i++) { new->fd[i] = ...; }
 */
void harness_dupfgrp_array(void) {
    int maxfd;
    int nfd;
    int i;

    /* Setup */
    __CPROVER_assume(maxfd >= 0);
    __CPROVER_assume(maxfd < MAXNFD);
    __CPROVER_assume(nfd > maxfd); /* Invariant: nfd > maxfd */

    /* Simulate the loop */
    for(i = 0; i <= maxfd; i++) {
        /* Verify i is within bounds */
        __CPROVER_assert(i >= 0, "index non-negative");
        __CPROVER_assert(i < nfd, "index within new->nfd bounds");

        /* Since nfd > maxfd, i <= maxfd implies i < nfd */
    }
}

/*
 * Verify closefgrp array access (pgrp.c:202)
 */
void harness_closefgrp_array(void) {
    int maxfd;
    int nfd;
    int i;

    /* Setup */
    __CPROVER_assume(maxfd >= 0);
    __CPROVER_assume(maxfd < nfd); /* Invariant */
    __CPROVER_assume(nfd <= MAXNFD);

    /* Simulate the loop */
    for(i = 0; i <= maxfd; i++) {
        /* Verify bounds */
        __CPROVER_assert(i >= 0, "index non-negative");
        __CPROVER_assert(i < nfd, "index within fd array bounds");
    }
}

/*
 * Main harness
 */
void harness(void) {
    harness_newfgrp_alloc();
    harness_dupfgrp_alloc();
    harness_dupfgrp_array();
    harness_closefgrp_array();
}
