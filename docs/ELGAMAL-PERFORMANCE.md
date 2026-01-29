# ElGamal Performance Analysis

## Problem Summary

ElGamal key generation is extremely slow in Inferno/infernode:

| Key Size | Time |
|----------|------|
| 512-bit | 77 seconds |
| 1024-bit | 104 seconds |
| 2048-bit | **486 seconds (8.1 minutes)** |

For comparison:
- Modern systems generate RSA 2048-bit keys in < 1 second
- OpenSSL generates 2048-bit DH parameters in ~10-30 seconds

## Root Cause

ElGamal requires **safe prime** parameters (p where (p-1)/2 is also prime).

### Algorithm (`libsec/gensafeprime.c`):

```c
void gensafeprime(mpint *p, mpint *alpha, int n, int accuracy)
{
    q = mpnew(n-1);
    while(1){
        genprime(q, n-1, accuracy);  // Find a prime q
        p = 2*q + 1;                  // Compute p
        if(probably_prime(p, accuracy))  // Check if p is also prime
            break;                    // If so, we have a safe prime
    }
    // Then find generator alpha...
}
```

### Why It's Slow:

1. **Safe prime probability**: For a random prime q, the probability that 2q+1 is also prime is roughly 1/(ln(n)). For 2048-bit, this means testing ~700+ primes on average.

2. **Primality testing cost**: Each `probably_prime()` call runs 18 Miller-Rabin iterations, each involving modular exponentiation of 2048-bit numbers.

3. **No assembly optimization**: Inferno's `libmp` is pure C without assembly-optimized big integer operations.

4. **Double primality testing**: Each candidate requires TWO primality tests (for q and for p).

### Comparison with RSA

RSA only needs regular primes:
```c
genprime(p, nlen/2, rounds);  // Just one prime
genprime(q, nlen/2, rounds);  // Just one prime
```

This is why RSA 2048-bit keygen completes in seconds while ElGamal takes minutes.

## Solutions

### 1. Use Pre-computed Parameters (Recommended)

Ship with pre-generated safe primes for common sizes. Store in `/lib/crypto/dhparams/`:

```
/lib/crypto/dhparams/dh2048.pem
/lib/crypto/dhparams/dh4096.pem
```

**Security**: Pre-computed parameters are safe as long as:
- They're generated properly (not backdoored)
- Each user generates their own secret exponent

This is what OpenSSL's `DH_get_2048_256()` does.

### 2. Parameter Reuse

Generate parameters once and reuse for multiple keys:

```limbo
# Generate parameters (slow, do once)
params := kr->dhparams(2048);

# Generate keys using those parameters (fast)
sk1 := kr->genSKfromPK(params, "user1");
sk2 := kr->genSKfromPK(params, "user2");
```

The `eg_genfrompk()` function in `egalg.c` already supports this:
```c
static void* eg_genfrompk(void *vpub)
{
    // Uses existing p and alpha, just generates new secret
    mprand(nlen-1, genrandom, priv->secret);
    mpexp(pub->alpha, priv->secret, pub->p, pub->key);
    return priv;
}
```

### 3. Use DSA-style Parameters

DSA uses a different parameter structure that's faster to generate:
- q is a small prime (256 bits)
- p = kq + 1 for some k, where p is a large prime

This is faster because q is small and you're just searching for a p of the form kq+1.

**Tradeoff**: Slightly different security properties.

### 4. Background Generation

Generate parameters in a background thread:

```limbo
spawn generate_dh_params(2048, result_chan);
# ... continue with other work ...
params := <-result_chan;  # Block when needed
```

### 5. Use Ed25519 Instead

For signatures, Ed25519 is:
- **Faster**: Key generation is instant (hash + scalar mult)
- **Smaller**: 32-byte keys vs 256+ bytes
- **Stronger**: 128-bit security with simpler code
- **Deterministic**: No random number needed for signing

**Recommendation**: Use Ed25519 for all new signature applications.

### 6. Optimize libmp (Major Effort)

Add assembly-optimized big integer operations:
- Montgomery multiplication
- Karatsuba multiplication
- Platform-specific assembly (ARM NEON, x86-64)

This could provide 10-50x speedup but requires significant effort.

## Recommendations

1. **Short term**: Use Ed25519 for signatures (already implemented)

2. **Medium term**: Add pre-computed DH parameters for common sizes

3. **Long term**: Consider optimizing libmp or using a faster library

## Test Data

Generated on MacOSX/arm64 (Apple Silicon):

```
Generating ElGamal 512-bit key...
  Done in 76806 ms

Generating ElGamal 1024-bit key...
  Done in 103838 ms

Generating ElGamal 2048-bit key...
  Done in 486071 ms
```

## References

- Menezes et al., "Handbook of Applied Cryptography", Algorithm 4.86
- RFC 3526 - MODP Diffie-Hellman groups (pre-computed parameters)
- RFC 7919 - Negotiated Finite Field DH Ephemeral Parameters
