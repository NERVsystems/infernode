# Cryptographic Modernization Summary

**Status:** Complete (Phase 1-3 + ElGamal fix)

This document summarizes the cryptographic modernization work done on Inferno/infernode to support autonomous agent systems requiring identity verification and non-repudiation.

## Design Decision

**Clean break approach** - No backward compatibility with weak crypto. All deployments must regenerate keys after updating.

## Changes Implemented

### 1. Ed25519 Signatures (Phase 3)

Ed25519 is now the default signature algorithm for all new keys.

| Aspect | Before | After |
|--------|--------|-------|
| Default algorithm | ElGamal | **Ed25519** |
| Key size | 256+ bytes | 32 bytes |
| Signature size | Variable | 64 bytes |
| Performance | Slow keygen | Instant keygen |

**Files modified:**
- `libkeyring/ed25519alg.c` (new) - Ed25519 implementation
- `libkeyring/ed25519.c` (new) - Core Ed25519 operations
- `appl/cmd/auth/signer.b` - Uses Ed25519 by default
- `appl/cmd/auth/createsignerkey.b` - Ed25519 first in algorithm list

### 2. SHA-256 for Certificates (Phase 2)

All certificate hashing now uses SHA-256 instead of SHA-1.

| Aspect | Before | After |
|--------|--------|-------|
| Certificate hash | SHA-1 (broken) | **SHA-256** |
| Password hash | SHA-1 | **SHA-256** |
| Protocol digest | Mixed | **SHA-256** |

**Files modified:**
- `appl/cmd/auth/signer.b` - SHA-256 for signing
- `appl/cmd/auth/createsignerkey.b` - SHA-256 for certificates
- `appl/cmd/auth/logind.b` - SHA-256 for protocol
- `appl/cmd/auth/mkauthinfo.b` - SHA-256 for certificates
- `appl/cmd/auth/changelogin.b` - SHA-256 for passwords
- `appl/cmd/auth/keysrv.b` - SHA-256 for secret hashing
- `appl/lib/login.b` - SHA-256 for protocol

### 3. Key Size Defaults (Phase 1)

Minimum key sizes increased to 2048 bits.

| Component | Before | After |
|-----------|--------|-------|
| Signer PKmodlen | 512 bits | **2048 bits** |
| Signer DHmodlen | 512 bits | **2048 bits** |
| User PKmodlen | 1024 bits | **2048 bits** |
| User DHmodlen | 1024 bits | **2048 bits** |

**Files modified:**
- `appl/cmd/auth/signer.b` - 2048-bit defaults
- `appl/cmd/auth/createsignerkey.b` - 2048-bit defaults

### 4. ElGamal Performance Fix

ElGamal 2048-bit key generation improved from 8 minutes to 2 seconds (215x speedup).

| Metric | Before | After |
|--------|--------|-------|
| 2048-bit keygen | 486,000 ms | **2,258 ms** |
| Speedup | - | **215x** |

**Solution:** Pre-computed RFC 3526 MODP Group 14 parameters.

**Files modified:**
- `libsec/dhparams.c` (new) - RFC 3526 parameters
- `libsec/eggen.c` - Uses pre-computed params when available
- `include/libsec.h` - `getdhparams()` declaration
- `libsec/mkfile` - Build dhparams.c

## Security Properties

After these changes:

- **Signatures:** Ed25519 provides 128-bit security with deterministic signatures
- **Certificates:** SHA-256 hash prevents collision attacks
- **Key exchange:** 2048-bit DH provides ~112-bit security
- **Transport:** WireGuard + Rosenpass handles (external to Inferno)

## What's Not Changed

### RC4 in Login Protocol

The initial login key exchange still uses RC4 (`alg rc4`) for encrypting the DH component with the password-derived key. This is a protocol-level change requiring coordinated client/server updates.

**Risk assessment:** Low - used only for brief DH exchange, not bulk data.

### SSL3 Cipher Suites

The SSL3 implementation retains legacy cipher support for protocol compatibility. SHA-1 usage in HMAC contexts remains (HMAC-SHA1 is still secure).

## Migration Guide

### For Existing Deployments

1. **Regenerate all keys:**
   ```sh
   # Generate new signer key with Ed25519
   auth/createsignerkey -a ed25519 signer_name

   # Recreate user accounts
   auth/changelogin username
   ```

2. **Update client certificates:**
   - Old certificates will fail verification (SHA-1 vs SHA-256)
   - Clients must re-authenticate to get new certificates

### For New Deployments

No action needed - Ed25519 and SHA-256 are the defaults.

## Testing

### Ed25519

```sh
# Run Ed25519 test in Inferno
/tests/ed25519_test.dis
```

### ElGamal Performance

```sh
# Run keygen benchmark
/tests/keygen_benchmark.dis
```

## Related Documentation

- [CRYPTO-DEBUGGING-GUIDE.md](CRYPTO-DEBUGGING-GUIDE.md) - Debugging methodology
- [ELGAMAL-PERFORMANCE.md](ELGAMAL-PERFORMANCE.md) - Detailed performance analysis

## Commits

- `850e906a` - feat(crypto): Add Ed25519 signatures and modernize cryptography
- `172d7f7f` - Add RFC 3526 pre-computed DH params for fast ElGamal 2048-bit keygen

## Future Work

Optional improvements not implemented:

1. **RC4 → AES in login protocol** - Requires protocol version negotiation
2. **Disable weak SSL ciphers** - May break legacy clients
3. **Certificate revocation** - Not currently supported
