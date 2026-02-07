# Cryptographic Debugging Guide for Inferno

This guide captures lessons learned from debugging cryptographic implementations in Inferno/infernode. It provides a systematic approach to diagnosing and fixing crypto bugs.

## General Principles

### 1. Crypto Bugs Are Silent

Cryptographic code rarely crashes - it just produces wrong output. You won't get segfaults; you'll get verification failures, decryption garbage, or invalid signatures.

### 2. Trust Nothing, Verify Everything

Every intermediate value should be verified against known-good reference implementations. Python with the `cryptography` or built-in `hashlib` libraries is invaluable.

### 3. Test Vectors Are Sacred

RFC test vectors are your ground truth. If your implementation doesn't match them exactly, you have a bug.

## Debugging Methodology

### Phase 1: Identify the Failure Point

```
1. Does keygen work? (Can you generate valid keys?)
2. Does signing work? (Can you produce a signature?)
3. Does verification work? (Can you verify a known-good signature?)
4. Does round-trip work? (Sign then verify your own signature?)
```

### Phase 2: Add Self-Tests

Add verification at every step:

```c
// After computing something
DEBUG_PRINT("computed_value[0:8] = %s\n", hex_encode(val, 8));
DEBUG_PRINT("expected[0:8] = %s\n", KNOWN_GOOD_HEX);
```

### Phase 3: Binary Search the Problem

If sign works but verify fails:
1. Add self-verify inside sign function
2. If self-verify fails, bug is in sign
3. If self-verify passes, bug is in verify

### Phase 4: Compare with Reference

Extract intermediate values and verify with Python:

```python
# Example: Verify Ed25519 hash computation
import hashlib

R = bytes.fromhex("...")  # First 32 bytes of signature
pk = bytes.fromhex("...")  # Public key
msg = b"test message"

h = hashlib.sha512(R + pk + msg).digest()
print(f"h[0:8] = {h[:8].hex()}")  # Compare with C output
```

## Common Bug Patterns

### 1. Endianness Mismatches

Ed25519 uses little-endian. Many libraries use big-endian. Converting between them:

```c
// Little-endian to big-endian
for(i = 0; i < 32; i++)
    be[i] = le[31-i];
```

**Symptom**: Values are "reversed" or produce wrong results.

### 2. Incorrect Constants

Cryptographic constants must be exactly right. Common mistakes:
- Wrong base point coordinates
- Wrong curve parameters
- Wrong modulus values

**Solution**: Compute constants programmatically and verify:

```python
# Ed25519 base point y-coordinate
p = 2**255 - 19
y = 4 * pow(5, p-2, p) % p
print(y.to_bytes(32, 'little').hex())  # Should be 5866...
```

### 3. Field Reduction Bugs

Field arithmetic must properly reduce results modulo p.

**Symptom**: Large values, occasional wrong results.

**Solution**: Use non-negative intermediate values when reduction is simple.

### 4. Scalar Reduction Bugs

Scalars must be reduced modulo the group order L, not the field order p.

**Symptom**: Signatures that don't verify, even for the same key.

**Solution**: Use arbitrary-precision arithmetic (mpint) for correctness.

### 5. Memory/Pointer Bugs

Stack corruption, buffer overflows, use-after-free.

**Symptom**: Intermittent failures, works in debug but not release.

**Solution**: Add bounds checking, use valgrind/ASAN.

## Inferno-Specific Notes

### Using mpint

Inferno's `mpint` library provides arbitrary-precision integers:

```c
#include <mp.h>

mpint *a = mpnew(0);
mpint *b = betomp(bytes, len, nil);  // Big-endian bytes to mpint
mpmul(a, b, result);                  // result = a * b
mpmod(result, modulus, result);       // result = result mod modulus
mptobe(result, bytes, len, nil);      // mpint to big-endian bytes
mpfree(a);
```

### SHA-512 in Inferno

```c
#include <libsec.h>

DigestState *ds = sha512(data, len, nil, nil);  // Init + update
sha512(data2, len2, digest, ds);                 // Final with output
```

### Debug Output

Use `fprint(2, ...)` for debug output (stderr):

```c
fprint(2, "DEBUG: value[0:8] = ");
for(i = 0; i < 8; i++)
    fprint(2, "%02x", buf[i]);
fprint(2, "\n");
```

## Testing Checklist

Before declaring a crypto implementation complete:

- [ ] All RFC test vectors pass
- [ ] Key generation produces valid keys
- [ ] Sign/verify round-trip works
- [ ] Self-verify in sign function passes
- [ ] Verification rejects modified messages
- [ ] Verification rejects modified signatures
- [ ] Verification rejects wrong public keys
- [ ] No memory leaks (if applicable)
- [ ] Consistent results across multiple runs

## Python Verification Scripts

Keep Python scripts for verifying each component:

```
scripts/
├── verify_sha512.py
├── verify_scalar_mult.py
├── verify_point_add.py
├── verify_muladd.py
└── verify_full_signature.py
```

These are invaluable for debugging. When C produces wrong output, Python tells you what it should be.

## Reference Implementations

For each algorithm, identify reference implementations:

| Algorithm | Reference |
|-----------|-----------|
| Ed25519 | TweetNaCl, ref10 |
| SHA-256 | OpenSSL, Python hashlib |
| SHA-512 | OpenSSL, Python hashlib |
| RSA | OpenSSL, Go crypto/rsa |
| ElGamal | libgcrypt |

## When All Else Fails

1. **Simplify**: Test the smallest possible case
2. **Compare byte-by-byte**: Print every intermediate value
3. **Use a debugger**: Step through both implementations
4. **Ask for help**: Fresh eyes catch what you miss
5. **Start over**: Sometimes a rewrite is faster than debugging

## Conclusion

Cryptographic debugging is methodical work. The bug is always logical - either a wrong value, wrong operation, or wrong order. Systematic verification of each step will always find it.

Remember: In crypto, "almost correct" is completely broken.
