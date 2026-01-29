/*
 * Ed25519 signature algorithm implementation for Inferno keyring
 *
 * Ed25519 provides 128-bit security with fixed-size keys:
 * - Secret key: 64 bytes (32-byte seed + 32-byte public key)
 * - Public key: 32 bytes
 * - Signature: 64 bytes
 *
 * This implementation uses the reference Ed25519 from SUPERCOP/ref10
 * with modifications for Inferno's keyring interface.
 */

#include <lib9.h>
#include <kernel.h>
#include <isa.h>
#include "interp.h"
#include "../libinterp/keyringif.h"
#include "mp.h"
#include "libsec.h"
#include "keys.h"

/*
 * Ed25519 constants
 */
enum {
	Ed25519SecretKeyLen = 64,	/* seed (32) + public key (32) */
	Ed25519PublicKeyLen = 32,
	Ed25519SignatureLen = 64,
	Ed25519SeedLen = 32
};

/*
 * Ed25519 key structures
 */
typedef struct Ed25519priv Ed25519priv;
typedef struct Ed25519pub Ed25519pub;
typedef struct Ed25519sig Ed25519sig;

struct Ed25519priv {
	uchar	seed[Ed25519SeedLen];		/* 32-byte seed */
	uchar	pk[Ed25519PublicKeyLen];	/* derived public key */
};

struct Ed25519pub {
	uchar	key[Ed25519PublicKeyLen];
};

struct Ed25519sig {
	uchar	sig[Ed25519SignatureLen];
};

/*
 * Forward declarations for Ed25519 core operations
 * These are implemented at the bottom of this file using the ref10 algorithm
 */
static void ed25519_create_keypair(uchar *pk, uchar *sk, const uchar *seed);
static void ed25519_sign(uchar *sig, const uchar *m, ulong mlen, const uchar *sk);
static int ed25519_verify(const uchar *sig, const uchar *m, ulong mlen, const uchar *pk);

/*
 * Allocation functions
 */
static Ed25519priv*
ed25519privalloc(void)
{
	Ed25519priv *k;
	k = malloc(sizeof(Ed25519priv));
	if(k == nil)
		return nil;
	memset(k, 0, sizeof(Ed25519priv));
	return k;
}

static Ed25519pub*
ed25519puballoc(void)
{
	Ed25519pub *k;
	k = malloc(sizeof(Ed25519pub));
	if(k == nil)
		return nil;
	memset(k, 0, sizeof(Ed25519pub));
	return k;
}

static Ed25519sig*
ed25519sigalloc(void)
{
	Ed25519sig *s;
	s = malloc(sizeof(Ed25519sig));
	if(s == nil)
		return nil;
	memset(s, 0, sizeof(Ed25519sig));
	return s;
}

/*
 * Free functions
 */
static void
ed25519privfree(Ed25519priv *k)
{
	if(k == nil)
		return;
	memset(k, 0, sizeof(Ed25519priv));	/* clear secret key */
	free(k);
}

static void
ed25519pubfree(Ed25519pub *k)
{
	free(k);
}

static void
ed25519sigfree(Ed25519sig *s)
{
	free(s);
}

/*
 * Attribute lists for serialization
 */
static char* pkattr[] = { "pk", nil };
static char* skattr[] = { "seed", "pk", nil };
static char* sigattr[] = { "sig", nil };

/*
 * Base64 encoding/decoding for fixed-size byte arrays
 */
static int
bytes2base64(uchar *bytes, int len, char *buf, int buflen)
{
	int n;
	n = enc64(buf, buflen, bytes, len);
	if(n > 0 && n < buflen)
		buf[n++] = '\n';
	buf[n] = 0;
	return n;
}

static int
base64tobytes(char *str, uchar *bytes, int len, char **strp)
{
	char *p;
	int n;

	for(p = str; *p && *p != '\n'; p++)
		;
	n = dec64(bytes, len, str, p - str);
	if(strp){
		if(*p)
			p++;
		*strp = p;
	}
	return n;
}

/*
 * String to key conversions
 */
static void*
ed25519_str2sk(char *str, char **strp)
{
	Ed25519priv *k;
	char *p;
	int n;
	uchar sk[Ed25519SecretKeyLen];

	k = ed25519privalloc();
	if(k == nil)
		return nil;

	/* Read seed */
	n = base64tobytes(str, k->seed, Ed25519SeedLen, &p);
	if(n != Ed25519SeedLen){
		ed25519privfree(k);
		return nil;
	}

	/* Read public key */
	n = base64tobytes(p, k->pk, Ed25519PublicKeyLen, &p);
	if(n != Ed25519PublicKeyLen){
		ed25519privfree(k);
		return nil;
	}

	if(strp)
		*strp = p;

	return k;
}

static void*
ed25519_str2pk(char *str, char **strp)
{
	Ed25519pub *k;
	char *p;
	int n;

	k = ed25519puballoc();
	if(k == nil)
		return nil;

	n = base64tobytes(str, k->key, Ed25519PublicKeyLen, &p);
	if(n != Ed25519PublicKeyLen){
		ed25519pubfree(k);
		return nil;
	}

	if(strp)
		*strp = p;

	return k;
}

static void*
ed25519_str2sig(char *str, char **strp)
{
	Ed25519sig *s;
	char *p;
	int n;

	s = ed25519sigalloc();
	if(s == nil)
		return nil;

	n = base64tobytes(str, s->sig, Ed25519SignatureLen, &p);
	if(n != Ed25519SignatureLen){
		ed25519sigfree(s);
		return nil;
	}

	if(strp)
		*strp = p;

	return s;
}

/*
 * Key to string conversions
 */
static int
ed25519_sk2str(void *vk, char *buf, int len)
{
	Ed25519priv *k = (Ed25519priv*)vk;
	char *cp = buf;
	char *ep = buf + len - 1;

	cp += bytes2base64(k->seed, Ed25519SeedLen, cp, ep - cp);
	cp += bytes2base64(k->pk, Ed25519PublicKeyLen, cp, ep - cp);
	*cp = 0;

	return cp - buf;
}

static int
ed25519_pk2str(void *vk, char *buf, int len)
{
	Ed25519pub *k = (Ed25519pub*)vk;
	char *cp = buf;
	char *ep = buf + len - 1;

	cp += bytes2base64(k->key, Ed25519PublicKeyLen, cp, ep - cp);
	*cp = 0;

	return cp - buf;
}

static int
ed25519_sig2str(void *vs, char *buf, int len)
{
	Ed25519sig *s = (Ed25519sig*)vs;
	char *cp = buf;
	char *ep = buf + len - 1;

	cp += bytes2base64(s->sig, Ed25519SignatureLen, cp, ep - cp);
	*cp = 0;

	return cp - buf;
}

/*
 * Convert secret key to public key
 */
static void*
ed25519_sk2pk(void *vs)
{
	Ed25519priv *sk = (Ed25519priv*)vs;
	Ed25519pub *pk;
	int i;

	pk = ed25519puballoc();
	if(pk == nil)
		return nil;

	fprint(2, "ed25519_sk2pk: sk->pk[0:8] = ");
	for(i = 0; i < 8; i++) fprint(2, "%02x", sk->pk[i]);
	fprint(2, "\n");

	memmove(pk->key, sk->pk, Ed25519PublicKeyLen);

	fprint(2, "ed25519_sk2pk: pk->key[0:8] = ");
	for(i = 0; i < 8; i++) fprint(2, "%02x", pk->key[i]);
	fprint(2, "\n");

	return pk;
}

/*
 * Generate a new keypair
 * The 'len' parameter is ignored for Ed25519 (fixed key size)
 */
static void*
ed25519_gen(int len)
{
	Ed25519priv *k;
	uchar sk[Ed25519SecretKeyLen];

	USED(len);

	k = ed25519privalloc();
	if(k == nil)
		return nil;

	/* Generate random seed */
	genrandom(k->seed, Ed25519SeedLen);

	/* Derive keypair from seed */
	ed25519_create_keypair(k->pk, sk, k->seed);

	return k;
}

/*
 * Generate a new keypair from an existing public key
 * For Ed25519, this just generates a new independent keypair
 * (there's no parameter sharing like in ElGamal/DH)
 */
static void*
ed25519_genfrompk(void *vpub)
{
	USED(vpub);
	return ed25519_gen(0);
}

/*
 * Sign a message hash
 * Note: Ed25519 normally signs the full message, but Inferno's interface
 * passes a hash. We sign the hash bytes directly.
 */
static void*
ed25519_sign_hash(mpint *mp, void *key)
{
	Ed25519priv *sk = (Ed25519priv*)key;
	Ed25519sig *sig;
	uchar hash[SHA512dlen];
	uchar fullsk[Ed25519SecretKeyLen];
	int n, i;

	sig = ed25519sigalloc();
	if(sig == nil)
		return nil;

	/* Convert mpint to bytes (the hash to sign) */
	n = mptobe(mp, hash, sizeof(hash), nil);
	if(n < 0){
		ed25519sigfree(sig);
		return nil;
	}

	fprint(2, "ed25519_sign_hash: n=%d, hash[0:8] = ", n);
	for(i = 0; i < 8 && i < n; i++) fprint(2, "%02x", hash[i]);
	fprint(2, "\n");

	/* Create full secret key: seed || public key */
	memmove(fullsk, sk->seed, Ed25519SeedLen);
	memmove(fullsk + Ed25519SeedLen, sk->pk, Ed25519PublicKeyLen);

	fprint(2, "ed25519_sign_hash: seed[0:8] = ");
	for(i = 0; i < 8; i++) fprint(2, "%02x", sk->seed[i]);
	fprint(2, "\ned25519_sign_hash: sk->pk[0:8] = ");
	for(i = 0; i < 8; i++) fprint(2, "%02x", sk->pk[i]);
	fprint(2, "\n");

	/* Sign the hash */
	ed25519_sign(sig->sig, hash, n, fullsk);

	fprint(2, "ed25519_sign_hash: sig[0:8] = ");
	for(i = 0; i < 8; i++) fprint(2, "%02x", sig->sig[i]);
	fprint(2, "\ned25519_sign_hash: sig[32:40] = ");
	for(i = 32; i < 40; i++) fprint(2, "%02x", sig->sig[i]);
	fprint(2, "\n");

	/* Clear sensitive data */
	memset(fullsk, 0, sizeof(fullsk));
	memset(hash, 0, sizeof(hash));

	return sig;
}

/*
 * Verify a signature
 */
static int
ed25519_verify_hash(mpint *mp, void *vsig, void *vkey)
{
	Ed25519pub *pk = (Ed25519pub*)vkey;
	Ed25519sig *sig = (Ed25519sig*)vsig;
	uchar hash[SHA512dlen];
	int n, ok, i;

	fprint(2, "ed25519_verify_hash: called\n");

	/* Convert mpint to bytes */
	n = mptobe(mp, hash, sizeof(hash), nil);
	if(n < 0) {
		fprint(2, "ed25519_verify_hash: mptobe failed\n");
		return 0;
	}

	fprint(2, "ed25519_verify_hash: hash len=%d, hash[0:8] = ", n);
	for(i = 0; i < 8 && i < n; i++) fprint(2, "%02x", hash[i]);
	fprint(2, "\n");

	/* Verify */
	ok = ed25519_verify(sig->sig, hash, n, pk->key);

	fprint(2, "ed25519_verify_hash: result=%d\n", ok);

	memset(hash, 0, sizeof(hash));
	return ok;
}

/*
 * Free functions for SigAlgVec
 */
static void
ed25519_freepub(void *a)
{
	ed25519pubfree((Ed25519pub*)a);
}

static void
ed25519_freepriv(void *a)
{
	ed25519privfree((Ed25519priv*)a);
}

static void
ed25519_freesig(void *a)
{
	ed25519sigfree((Ed25519sig*)a);
}

/*
 * Initialize and return the Ed25519 signature algorithm vector
 */
SigAlgVec*
ed25519init(void)
{
	SigAlgVec *vec;

	vec = malloc(sizeof(SigAlgVec));
	if(vec == nil)
		return nil;

	vec->name = "ed25519";

	vec->pkattr = pkattr;
	vec->skattr = skattr;
	vec->sigattr = sigattr;

	vec->str2sk = ed25519_str2sk;
	vec->str2pk = ed25519_str2pk;
	vec->str2sig = ed25519_str2sig;

	vec->sk2str = ed25519_sk2str;
	vec->pk2str = ed25519_pk2str;
	vec->sig2str = ed25519_sig2str;

	vec->sk2pk = ed25519_sk2pk;

	vec->gensk = ed25519_gen;
	vec->genskfrompk = ed25519_genfrompk;
	vec->sign = ed25519_sign_hash;
	vec->verify = ed25519_verify_hash;

	vec->skfree = ed25519_freepriv;
	vec->pkfree = ed25519_freepub;
	vec->sigfree = ed25519_freesig;

	return vec;
}

/*
 * ============================================================================
 * Ed25519 Core Implementation (based on ref10 from SUPERCOP)
 * ============================================================================
 *
 * This is a self-contained Ed25519 implementation.
 * The code below implements the Ed25519 signature scheme as specified in:
 *   https://ed25519.cr.yp.to/
 *
 * Based on the reference implementation from SUPERCOP with modifications
 * for portability and integration with Inferno's build system.
 */

typedef vlong i64;
typedef uvlong u64;
typedef int i32;
typedef uint u32;

/* Field element representation: 10 limbs of ~25.5 bits each */
typedef i32 fe[10];

/* Group element representations */
typedef struct {
	fe X;
	fe Y;
	fe Z;
} ge_p2;

typedef struct {
	fe X;
	fe Y;
	fe Z;
	fe T;
} ge_p3;

typedef struct {
	fe X;
	fe Y;
	fe Z;
	fe T;
} ge_p1p1;

typedef struct {
	fe yplusx;
	fe yminusx;
	fe xy2d;
} ge_precomp;

typedef struct {
	fe YplusX;
	fe YminusX;
	fe Z;
	fe T2d;
} ge_cached;

/*
 * Load/store functions
 */
static u64
load_3(const uchar *in)
{
	u64 result;
	result = (u64)in[0];
	result |= ((u64)in[1]) << 8;
	result |= ((u64)in[2]) << 16;
	return result;
}

static u64
load_4(const uchar *in)
{
	u64 result;
	result = (u64)in[0];
	result |= ((u64)in[1]) << 8;
	result |= ((u64)in[2]) << 16;
	result |= ((u64)in[3]) << 24;
	return result;
}

/*
 * Field arithmetic
 */
static void
fe_0(fe h)
{
	int i;
	for(i = 0; i < 10; i++)
		h[i] = 0;
}

static void
fe_1(fe h)
{
	h[0] = 1;
	h[1] = 0;
	h[2] = 0;
	h[3] = 0;
	h[4] = 0;
	h[5] = 0;
	h[6] = 0;
	h[7] = 0;
	h[8] = 0;
	h[9] = 0;
}

static void
fe_copy(fe h, const fe f)
{
	int i;
	for(i = 0; i < 10; i++)
		h[i] = f[i];
}

static void
fe_neg(fe h, const fe f)
{
	int i;
	for(i = 0; i < 10; i++)
		h[i] = -f[i];
}

static void
fe_add(fe h, const fe f, const fe g)
{
	int i;
	for(i = 0; i < 10; i++)
		h[i] = f[i] + g[i];
}

static void
fe_sub(fe h, const fe f, const fe g)
{
	int i;
	for(i = 0; i < 10; i++)
		h[i] = f[i] - g[i];
}

static void
fe_frombytes(fe h, const uchar *s)
{
	i64 h0 = load_4(s);
	i64 h1 = load_3(s + 4) << 6;
	i64 h2 = load_3(s + 7) << 5;
	i64 h3 = load_3(s + 10) << 3;
	i64 h4 = load_3(s + 13) << 2;
	i64 h5 = load_4(s + 16);
	i64 h6 = load_3(s + 20) << 7;
	i64 h7 = load_3(s + 23) << 5;
	i64 h8 = load_3(s + 26) << 4;
	i64 h9 = (load_3(s + 29) & 8388607) << 2;
	i64 carry0, carry1, carry2, carry3, carry4;
	i64 carry5, carry6, carry7, carry8, carry9;

	carry9 = (h9 + (i64)(1 << 24)) >> 25; h0 += carry9 * 19; h9 -= carry9 << 25;
	carry1 = (h1 + (i64)(1 << 24)) >> 25; h2 += carry1; h1 -= carry1 << 25;
	carry3 = (h3 + (i64)(1 << 24)) >> 25; h4 += carry3; h3 -= carry3 << 25;
	carry5 = (h5 + (i64)(1 << 24)) >> 25; h6 += carry5; h5 -= carry5 << 25;
	carry7 = (h7 + (i64)(1 << 24)) >> 25; h8 += carry7; h7 -= carry7 << 25;

	carry0 = (h0 + (i64)(1 << 25)) >> 26; h1 += carry0; h0 -= carry0 << 26;
	carry2 = (h2 + (i64)(1 << 25)) >> 26; h3 += carry2; h2 -= carry2 << 26;
	carry4 = (h4 + (i64)(1 << 25)) >> 26; h5 += carry4; h4 -= carry4 << 26;
	carry6 = (h6 + (i64)(1 << 25)) >> 26; h7 += carry6; h6 -= carry6 << 26;
	carry8 = (h8 + (i64)(1 << 25)) >> 26; h9 += carry8; h8 -= carry8 << 26;

	h[0] = (i32)h0;
	h[1] = (i32)h1;
	h[2] = (i32)h2;
	h[3] = (i32)h3;
	h[4] = (i32)h4;
	h[5] = (i32)h5;
	h[6] = (i32)h6;
	h[7] = (i32)h7;
	h[8] = (i32)h8;
	h[9] = (i32)h9;
}

static void
fe_reduce(fe h, const fe f)
{
	i32 h0 = f[0];
	i32 h1 = f[1];
	i32 h2 = f[2];
	i32 h3 = f[3];
	i32 h4 = f[4];
	i32 h5 = f[5];
	i32 h6 = f[6];
	i32 h7 = f[7];
	i32 h8 = f[8];
	i32 h9 = f[9];
	i32 q;
	i32 carry0, carry1, carry2, carry3, carry4;
	i32 carry5, carry6, carry7, carry8, carry9;

	q = (19 * h9 + (((i32)1) << 24)) >> 25;
	q = (h0 + q) >> 26;
	q = (h1 + q) >> 25;
	q = (h2 + q) >> 26;
	q = (h3 + q) >> 25;
	q = (h4 + q) >> 26;
	q = (h5 + q) >> 25;
	q = (h6 + q) >> 26;
	q = (h7 + q) >> 25;
	q = (h8 + q) >> 26;
	q = (h9 + q) >> 25;

	h0 += 19 * q;

	carry0 = h0 >> 26; h1 += carry0; h0 -= carry0 << 26;
	carry1 = h1 >> 25; h2 += carry1; h1 -= carry1 << 25;
	carry2 = h2 >> 26; h3 += carry2; h2 -= carry2 << 26;
	carry3 = h3 >> 25; h4 += carry3; h3 -= carry3 << 25;
	carry4 = h4 >> 26; h5 += carry4; h4 -= carry4 << 26;
	carry5 = h5 >> 25; h6 += carry5; h5 -= carry5 << 25;
	carry6 = h6 >> 26; h7 += carry6; h6 -= carry6 << 26;
	carry7 = h7 >> 25; h8 += carry7; h7 -= carry7 << 25;
	carry8 = h8 >> 26; h9 += carry8; h8 -= carry8 << 26;
	carry9 = h9 >> 25; h9 -= carry9 << 25;

	h[0] = h0;
	h[1] = h1;
	h[2] = h2;
	h[3] = h3;
	h[4] = h4;
	h[5] = h5;
	h[6] = h6;
	h[7] = h7;
	h[8] = h8;
	h[9] = h9;
}

static void
fe_tobytes(uchar *s, const fe h)
{
	fe t;
	fe_reduce(t, h);

	s[0] = (uchar)(t[0]);
	s[1] = (uchar)(t[0] >> 8);
	s[2] = (uchar)(t[0] >> 16);
	s[3] = (uchar)((t[0] >> 24) | (t[1] << 2));
	s[4] = (uchar)(t[1] >> 6);
	s[5] = (uchar)(t[1] >> 14);
	s[6] = (uchar)((t[1] >> 22) | (t[2] << 3));
	s[7] = (uchar)(t[2] >> 5);
	s[8] = (uchar)(t[2] >> 13);
	s[9] = (uchar)((t[2] >> 21) | (t[3] << 5));
	s[10] = (uchar)(t[3] >> 3);
	s[11] = (uchar)(t[3] >> 11);
	s[12] = (uchar)((t[3] >> 19) | (t[4] << 6));
	s[13] = (uchar)(t[4] >> 2);
	s[14] = (uchar)(t[4] >> 10);
	s[15] = (uchar)(t[4] >> 18);
	s[16] = (uchar)(t[5]);
	s[17] = (uchar)(t[5] >> 8);
	s[18] = (uchar)(t[5] >> 16);
	s[19] = (uchar)((t[5] >> 24) | (t[6] << 1));
	s[20] = (uchar)(t[6] >> 7);
	s[21] = (uchar)(t[6] >> 15);
	s[22] = (uchar)((t[6] >> 23) | (t[7] << 3));
	s[23] = (uchar)(t[7] >> 5);
	s[24] = (uchar)(t[7] >> 13);
	s[25] = (uchar)((t[7] >> 21) | (t[8] << 4));
	s[26] = (uchar)(t[8] >> 4);
	s[27] = (uchar)(t[8] >> 12);
	s[28] = (uchar)((t[8] >> 20) | (t[9] << 6));
	s[29] = (uchar)(t[9] >> 2);
	s[30] = (uchar)(t[9] >> 10);
	s[31] = (uchar)(t[9] >> 18);
}

static void
fe_mul(fe h, const fe f, const fe g)
{
	i32 f0 = f[0], f1 = f[1], f2 = f[2], f3 = f[3], f4 = f[4];
	i32 f5 = f[5], f6 = f[6], f7 = f[7], f8 = f[8], f9 = f[9];
	i32 g0 = g[0], g1 = g[1], g2 = g[2], g3 = g[3], g4 = g[4];
	i32 g5 = g[5], g6 = g[6], g7 = g[7], g8 = g[8], g9 = g[9];
	i32 g1_19 = 19 * g1, g2_19 = 19 * g2, g3_19 = 19 * g3, g4_19 = 19 * g4, g5_19 = 19 * g5;
	i32 g6_19 = 19 * g6, g7_19 = 19 * g7, g8_19 = 19 * g8, g9_19 = 19 * g9;
	i32 f1_2 = 2 * f1, f3_2 = 2 * f3, f5_2 = 2 * f5, f7_2 = 2 * f7, f9_2 = 2 * f9;
	i64 h0, h1, h2, h3, h4, h5, h6, h7, h8, h9;
	i64 carry0, carry1, carry2, carry3, carry4;
	i64 carry5, carry6, carry7, carry8, carry9;

	h0 = (i64)f0*g0 + (i64)f1_2*g9_19 + (i64)f2*g8_19 + (i64)f3_2*g7_19 + (i64)f4*g6_19 + (i64)f5_2*g5_19 + (i64)f6*g4_19 + (i64)f7_2*g3_19 + (i64)f8*g2_19 + (i64)f9_2*g1_19;
	h1 = (i64)f0*g1 + (i64)f1*g0 + (i64)f2*g9_19 + (i64)f3*g8_19 + (i64)f4*g7_19 + (i64)f5*g6_19 + (i64)f6*g5_19 + (i64)f7*g4_19 + (i64)f8*g3_19 + (i64)f9*g2_19;
	h2 = (i64)f0*g2 + (i64)f1_2*g1 + (i64)f2*g0 + (i64)f3_2*g9_19 + (i64)f4*g8_19 + (i64)f5_2*g7_19 + (i64)f6*g6_19 + (i64)f7_2*g5_19 + (i64)f8*g4_19 + (i64)f9_2*g3_19;
	h3 = (i64)f0*g3 + (i64)f1*g2 + (i64)f2*g1 + (i64)f3*g0 + (i64)f4*g9_19 + (i64)f5*g8_19 + (i64)f6*g7_19 + (i64)f7*g6_19 + (i64)f8*g5_19 + (i64)f9*g4_19;
	h4 = (i64)f0*g4 + (i64)f1_2*g3 + (i64)f2*g2 + (i64)f3_2*g1 + (i64)f4*g0 + (i64)f5_2*g9_19 + (i64)f6*g8_19 + (i64)f7_2*g7_19 + (i64)f8*g6_19 + (i64)f9_2*g5_19;
	h5 = (i64)f0*g5 + (i64)f1*g4 + (i64)f2*g3 + (i64)f3*g2 + (i64)f4*g1 + (i64)f5*g0 + (i64)f6*g9_19 + (i64)f7*g8_19 + (i64)f8*g7_19 + (i64)f9*g6_19;
	h6 = (i64)f0*g6 + (i64)f1_2*g5 + (i64)f2*g4 + (i64)f3_2*g3 + (i64)f4*g2 + (i64)f5_2*g1 + (i64)f6*g0 + (i64)f7_2*g9_19 + (i64)f8*g8_19 + (i64)f9_2*g7_19;
	h7 = (i64)f0*g7 + (i64)f1*g6 + (i64)f2*g5 + (i64)f3*g4 + (i64)f4*g3 + (i64)f5*g2 + (i64)f6*g1 + (i64)f7*g0 + (i64)f8*g9_19 + (i64)f9*g8_19;
	h8 = (i64)f0*g8 + (i64)f1_2*g7 + (i64)f2*g6 + (i64)f3_2*g5 + (i64)f4*g4 + (i64)f5_2*g3 + (i64)f6*g2 + (i64)f7_2*g1 + (i64)f8*g0 + (i64)f9_2*g9_19;
	h9 = (i64)f0*g9 + (i64)f1*g8 + (i64)f2*g7 + (i64)f3*g6 + (i64)f4*g5 + (i64)f5*g4 + (i64)f6*g3 + (i64)f7*g2 + (i64)f8*g1 + (i64)f9*g0;

	carry0 = (h0 + (i64)(1 << 25)) >> 26; h1 += carry0; h0 -= carry0 << 26;
	carry4 = (h4 + (i64)(1 << 25)) >> 26; h5 += carry4; h4 -= carry4 << 26;
	carry1 = (h1 + (i64)(1 << 24)) >> 25; h2 += carry1; h1 -= carry1 << 25;
	carry5 = (h5 + (i64)(1 << 24)) >> 25; h6 += carry5; h5 -= carry5 << 25;
	carry2 = (h2 + (i64)(1 << 25)) >> 26; h3 += carry2; h2 -= carry2 << 26;
	carry6 = (h6 + (i64)(1 << 25)) >> 26; h7 += carry6; h6 -= carry6 << 26;
	carry3 = (h3 + (i64)(1 << 24)) >> 25; h4 += carry3; h3 -= carry3 << 25;
	carry7 = (h7 + (i64)(1 << 24)) >> 25; h8 += carry7; h7 -= carry7 << 25;
	carry4 = (h4 + (i64)(1 << 25)) >> 26; h5 += carry4; h4 -= carry4 << 26;
	carry8 = (h8 + (i64)(1 << 25)) >> 26; h9 += carry8; h8 -= carry8 << 26;
	carry9 = (h9 + (i64)(1 << 24)) >> 25; h0 += carry9 * 19; h9 -= carry9 << 25;
	carry0 = (h0 + (i64)(1 << 25)) >> 26; h1 += carry0; h0 -= carry0 << 26;

	h[0] = (i32)h0; h[1] = (i32)h1; h[2] = (i32)h2; h[3] = (i32)h3; h[4] = (i32)h4;
	h[5] = (i32)h5; h[6] = (i32)h6; h[7] = (i32)h7; h[8] = (i32)h8; h[9] = (i32)h9;
}

static void
fe_sq(fe h, const fe f)
{
	fe_mul(h, f, f);
}

static void
fe_sq2(fe h, const fe f)
{
	fe_sq(h, f);
	fe_add(h, h, h);
}

static void
fe_invert(fe out, const fe z)
{
	fe t0, t1, t2, t3;
	int i;

	fe_sq(t0, z);
	fe_sq(t1, t0);
	fe_sq(t1, t1);
	fe_mul(t1, z, t1);
	fe_mul(t0, t0, t1);
	fe_sq(t2, t0);
	fe_mul(t1, t1, t2);
	fe_sq(t2, t1);
	for(i = 0; i < 4; i++)
		fe_sq(t2, t2);
	fe_mul(t1, t2, t1);
	fe_sq(t2, t1);
	for(i = 0; i < 9; i++)
		fe_sq(t2, t2);
	fe_mul(t2, t2, t1);
	fe_sq(t3, t2);
	for(i = 0; i < 19; i++)
		fe_sq(t3, t3);
	fe_mul(t2, t3, t2);
	fe_sq(t2, t2);
	for(i = 0; i < 9; i++)
		fe_sq(t2, t2);
	fe_mul(t1, t2, t1);
	fe_sq(t2, t1);
	for(i = 0; i < 49; i++)
		fe_sq(t2, t2);
	fe_mul(t2, t2, t1);
	fe_sq(t3, t2);
	for(i = 0; i < 99; i++)
		fe_sq(t3, t3);
	fe_mul(t2, t3, t2);
	fe_sq(t2, t2);
	for(i = 0; i < 49; i++)
		fe_sq(t2, t2);
	fe_mul(t1, t2, t1);
	fe_sq(t1, t1);
	for(i = 0; i < 4; i++)
		fe_sq(t1, t1);
	fe_mul(out, t1, t0);
}

static void
fe_pow22523(fe out, const fe z)
{
	fe t0, t1, t2;
	int i;

	fe_sq(t0, z);
	fe_sq(t1, t0);
	fe_sq(t1, t1);
	fe_mul(t1, z, t1);
	fe_mul(t0, t0, t1);
	fe_sq(t0, t0);
	fe_mul(t0, t1, t0);
	fe_sq(t1, t0);
	for(i = 0; i < 4; i++)
		fe_sq(t1, t1);
	fe_mul(t0, t1, t0);
	fe_sq(t1, t0);
	for(i = 0; i < 9; i++)
		fe_sq(t1, t1);
	fe_mul(t1, t1, t0);
	fe_sq(t2, t1);
	for(i = 0; i < 19; i++)
		fe_sq(t2, t2);
	fe_mul(t1, t2, t1);
	fe_sq(t1, t1);
	for(i = 0; i < 9; i++)
		fe_sq(t1, t1);
	fe_mul(t0, t1, t0);
	fe_sq(t1, t0);
	for(i = 0; i < 49; i++)
		fe_sq(t1, t1);
	fe_mul(t1, t1, t0);
	fe_sq(t2, t1);
	for(i = 0; i < 99; i++)
		fe_sq(t2, t2);
	fe_mul(t1, t2, t1);
	fe_sq(t1, t1);
	for(i = 0; i < 49; i++)
		fe_sq(t1, t1);
	fe_mul(t0, t1, t0);
	fe_sq(t0, t0);
	fe_sq(t0, t0);
	fe_mul(out, t0, z);
}

static int
fe_isnegative(const fe f)
{
	uchar s[32];
	fe_tobytes(s, f);
	return s[0] & 1;
}

static int
fe_isnonzero(const fe f)
{
	uchar s[32];
	int i;
	uchar r = 0;

	fe_tobytes(s, f);
	for(i = 0; i < 32; i++)
		r |= s[i];
	return r != 0;
}

/*
 * Constant d = -121665/121666 mod p
 * Using non-negative limbs for compatibility with fe_reduce
 */
static const fe d = {
	56195235, 13857412, 51736253, 6949390, 114729,
	24766616, 60832955, 30306712, 48412415, 21499315
};

/* 2*d - also using non-negative limbs */
static const fe d2 = {
	45281625, 27714825, 36363642, 13898781, 229458,
	15978800, 54557047, 27058993, 29715967, 9444199
};

/* sqrt(-1) - using non-negative limbs */
static const fe sqrtm1 = {
	34513072, 25610706, 9377949, 3500415, 12389472,
	33281959, 41962654, 31548777, 326685, 11406482
};

/*
 * Group element operations
 */
static void
ge_p3_0(ge_p3 *h)
{
	fe_0(h->X);
	fe_1(h->Y);
	fe_1(h->Z);
	fe_0(h->T);
}

static void
ge_p3_tobytes(uchar *s, const ge_p3 *h)
{
	fe recip, x, y;

	fe_invert(recip, h->Z);
	fe_mul(x, h->X, recip);
	fe_mul(y, h->Y, recip);
	fe_tobytes(s, y);
	s[31] ^= fe_isnegative(x) << 7;
}

static int
ge_frombytes_negate_vartime(ge_p3 *h, const uchar *s)
{
	fe u, v, v3, vxx, check;

	fe_frombytes(h->Y, s);
	fe_1(h->Z);
	fe_sq(u, h->Y);
	fe_mul(v, u, d);
	fe_sub(u, u, h->Z);
	fe_add(v, v, h->Z);

	fe_sq(v3, v);
	fe_mul(v3, v3, v);
	fe_sq(h->X, v3);
	fe_mul(h->X, h->X, v);
	fe_mul(h->X, h->X, u);

	fe_pow22523(h->X, h->X);
	fe_mul(h->X, h->X, v3);
	fe_mul(h->X, h->X, u);

	fe_sq(vxx, h->X);
	fe_mul(vxx, vxx, v);
	fe_sub(check, vxx, u);
	if(fe_isnonzero(check)){
		fe_add(check, vxx, u);
		if(fe_isnonzero(check))
			return -1;
		fe_mul(h->X, h->X, sqrtm1);
	}

	if(fe_isnegative(h->X) == (s[31] >> 7))
		fe_neg(h->X, h->X);

	fe_mul(h->T, h->X, h->Y);
	return 0;
}

static void
ge_p1p1_to_p2(ge_p2 *r, const ge_p1p1 *p)
{
	fe_mul(r->X, p->X, p->T);
	fe_mul(r->Y, p->Y, p->Z);
	fe_mul(r->Z, p->Z, p->T);
}

static void
ge_p1p1_to_p3(ge_p3 *r, const ge_p1p1 *p)
{
	fe_mul(r->X, p->X, p->T);
	fe_mul(r->Y, p->Y, p->Z);
	fe_mul(r->Z, p->Z, p->T);
	fe_mul(r->T, p->X, p->Y);
}

static void
ge_p2_dbl(ge_p1p1 *r, const ge_p2 *p)
{
	fe t0;

	fe_sq(r->X, p->X);
	fe_sq(r->Z, p->Y);
	fe_sq2(r->T, p->Z);
	fe_add(r->Y, p->X, p->Y);
	fe_sq(t0, r->Y);
	fe_add(r->Y, r->Z, r->X);
	fe_sub(r->Z, r->Z, r->X);
	fe_sub(r->X, t0, r->Y);
	fe_sub(r->T, r->T, r->Z);
}

static void
ge_p3_dbl(ge_p1p1 *r, const ge_p3 *p)
{
	ge_p2 q;
	fe_copy(q.X, p->X);
	fe_copy(q.Y, p->Y);
	fe_copy(q.Z, p->Z);
	ge_p2_dbl(r, &q);
}

static void
ge_madd(ge_p1p1 *r, const ge_p3 *p, const ge_precomp *q)
{
	fe t0;

	fe_add(r->X, p->Y, p->X);
	fe_sub(r->Y, p->Y, p->X);
	fe_mul(r->Z, r->X, q->yplusx);
	fe_mul(r->Y, r->Y, q->yminusx);
	fe_mul(r->T, q->xy2d, p->T);
	fe_add(t0, p->Z, p->Z);
	fe_sub(r->X, r->Z, r->Y);
	fe_add(r->Y, r->Z, r->Y);
	fe_add(r->Z, t0, r->T);
	fe_sub(r->T, t0, r->T);
}

static void
ge_msub(ge_p1p1 *r, const ge_p3 *p, const ge_precomp *q)
{
	fe t0;

	fe_add(r->X, p->Y, p->X);
	fe_sub(r->Y, p->Y, p->X);
	fe_mul(r->Z, r->X, q->yminusx);
	fe_mul(r->Y, r->Y, q->yplusx);
	fe_mul(r->T, q->xy2d, p->T);
	fe_add(t0, p->Z, p->Z);
	fe_sub(r->X, r->Z, r->Y);
	fe_add(r->Y, r->Z, r->Y);
	fe_sub(r->Z, t0, r->T);
	fe_add(r->T, t0, r->T);
}

static void
ge_add(ge_p1p1 *r, const ge_p3 *p, const ge_cached *q)
{
	fe t0;

	fe_add(r->X, p->Y, p->X);
	fe_sub(r->Y, p->Y, p->X);
	fe_mul(r->Z, r->X, q->YplusX);
	fe_mul(r->Y, r->Y, q->YminusX);
	fe_mul(r->T, q->T2d, p->T);
	fe_mul(t0, p->Z, q->Z);
	fe_add(t0, t0, t0);
	fe_sub(r->X, r->Z, r->Y);
	fe_add(r->Y, r->Z, r->Y);
	fe_add(r->Z, t0, r->T);
	fe_sub(r->T, t0, r->T);
}

static void
ge_sub(ge_p1p1 *r, const ge_p3 *p, const ge_cached *q)
{
	fe t0;

	fe_add(r->X, p->Y, p->X);
	fe_sub(r->Y, p->Y, p->X);
	fe_mul(r->Z, r->X, q->YminusX);
	fe_mul(r->Y, r->Y, q->YplusX);
	fe_mul(r->T, q->T2d, p->T);
	fe_mul(t0, p->Z, q->Z);
	fe_add(t0, t0, t0);
	fe_sub(r->X, r->Z, r->Y);
	fe_add(r->Y, r->Z, r->Y);
	fe_sub(r->Z, t0, r->T);
	fe_add(r->T, t0, r->T);
}

static void
ge_p3_to_cached(ge_cached *r, const ge_p3 *p)
{
	fe_add(r->YplusX, p->Y, p->X);
	fe_sub(r->YminusX, p->Y, p->X);
	fe_copy(r->Z, p->Z);
	fe_mul(r->T2d, p->T, d2);
}

/*
 * Base point B = (x, 4/5)
 * x = 15112221349535400772501151409588531511454012693041857206046113283949847762202
 * y = 46316835694926478169428394003475163141307993866256225615783033603165251855960
 * Note: Using non-negative limbs for compatibility with fe_reduce
 */
static const fe Bx = {
	52811034, 25909283, 16144682, 17082669, 27570973,
	30858332, 40966398, 8378388, 20764389, 8758491
};

static const fe By = {
	40265304, 26843545, 13421772, 20132659, 26843545,
	6710886, 53687091, 13421772, 40265318, 26843545
};

static void
ge_scalarmult_base(ge_p3 *h, const uchar *a)
{
	ge_p3 B;
	ge_p1p1 t;
	ge_cached Bcached;
	int i;
	static int tested = 0;

	/* Load base point B and cache it */
	fe_copy(B.X, Bx);
	fe_copy(B.Y, By);
	fe_1(B.Z);
	fe_mul(B.T, Bx, By);
	ge_p3_to_cached(&Bcached, &B);

	/* DEBUG: one-time test of [1]B = B */
	if(!tested){
		uchar one[32] = {1};  /* scalar 1 in little-endian */
		uchar Benc[32], result[32];
		ge_p3 test;

		tested = 1;

		/* Encode B directly */
		ge_p3_tobytes(Benc, &B);

		/* Compute [1]B the hard way */
		ge_p3_0(&test);
		for(i = 255; i >= 0; i--){
			ge_p3_dbl(&t, &test);
			ge_p1p1_to_p3(&test, &t);
			if((one[i/8] >> (i&7)) & 1){
				ge_add(&t, &test, &Bcached);
				ge_p1p1_to_p3(&test, &t);
			}
		}
		ge_p3_tobytes(result, &test);

		fprint(2, "SELFTEST [1]B: B_enc[0:8] = ");
		for(i = 0; i < 8; i++) fprint(2, "%02x", Benc[i]);
		fprint(2, "\nSELFTEST [1]B: result[0:8] = ");
		for(i = 0; i < 8; i++) fprint(2, "%02x", result[i]);
		fprint(2, "\nSELFTEST [1]B: match=%d\n", memcmp(Benc, result, 32) == 0);

		/* Also test [0]B = identity (Y=1, X=0) */
		{
			uchar zero[32] = {0};
			ge_p3 ident;
			uchar ident_enc[32];

			ge_p3_0(&ident);
			for(i = 255; i >= 0; i--){
				ge_p3_dbl(&t, &ident);
				ge_p1p1_to_p3(&ident, &t);
				/* no bit set in zero scalar */
			}
			ge_p3_tobytes(ident_enc, &ident);
			fprint(2, "SELFTEST [0]B: identity_enc[0:8] = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", ident_enc[i]);
			/* Should be 0100000...00 (y=1, x=0) */
			fprint(2, " (expected 01000000...)\n");
		}

		/* Test [2]B via scalar mult vs 2*B via ge_p3_dbl */
		{
			uchar two[32] = {2};  /* scalar 2 */
			ge_p3 twoB_scalarmult;
			ge_p3 twoB_dbl;
			uchar enc1[32], enc2[32];

			/* Method 1: [2]B via scalar mult */
			ge_p3_0(&twoB_scalarmult);
			for(i = 255; i >= 0; i--){
				ge_p3_dbl(&t, &twoB_scalarmult);
				ge_p1p1_to_p3(&twoB_scalarmult, &t);
				if((two[i/8] >> (i&7)) & 1){
					ge_add(&t, &twoB_scalarmult, &Bcached);
					ge_p1p1_to_p3(&twoB_scalarmult, &t);
				}
			}

			/* Method 2: 2*B via ge_p3_dbl */
			ge_p3_dbl(&t, &B);
			ge_p1p1_to_p3(&twoB_dbl, &t);

			ge_p3_tobytes(enc1, &twoB_scalarmult);
			ge_p3_tobytes(enc2, &twoB_dbl);

			fprint(2, "SELFTEST [2]B: scalar_mult[0:8] = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", enc1[i]);
			fprint(2, "\nSELFTEST [2]B: 2*B_dbl[0:8]    = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", enc2[i]);
			fprint(2, "\nSELFTEST [2]B: match=%d\n", memcmp(enc1, enc2, 32) == 0);

			/* Also test B + B via ge_add (should use unified formula) */
			{
				ge_p3 twoB_add;
				uchar enc3[32];
				ge_add(&t, &B, &Bcached);
				ge_p1p1_to_p3(&twoB_add, &t);
				ge_p3_tobytes(enc3, &twoB_add);
				fprint(2, "SELFTEST [2]B: B+B_add[0:8]    = ");
				for(i = 0; i < 8; i++) fprint(2, "%02x", enc3[i]);
				fprint(2, "\nSELFTEST [2]B: add_vs_dbl=%d\n", memcmp(enc2, enc3, 32) == 0);
			}
		}

		/* Test SHA512 correctness first */
		{
			/* SHA512("") = cf83e1357eefb8bdf1542850d66d8007... */
			/* SHA512("abc") = ddaf35a193617aba... */
			/* SHA512(32 zeros) = 1c4acf14 30f8b4c6... */
			uchar empty_hash[64];
			uchar abc_hash[64];
			uchar zeros[32] = {0};
			uchar zeros_hash[64];
			sha512((uchar*)"", 0, empty_hash, nil);
			sha512((uchar*)"abc", 3, abc_hash, nil);
			sha512(zeros, 32, zeros_hash, nil);
			fprint(2, "SELFTEST SHA512: sha512('')[0:8] = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", empty_hash[i]);
			fprint(2, " (expect cf83e135...)\n");
			fprint(2, "SELFTEST SHA512: sha512('abc')[0:8] = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", abc_hash[i]);
			fprint(2, " (expect ddaf35a1...)\n");
			fprint(2, "SELFTEST SHA512: sha512(32 zeros)[0:8] = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", zeros_hash[i]);
			fprint(2, " (expect 5046adc1...)\n");
		}

		/* RFC 8032 Test Vector 1: empty message */
		{
			/* Seed (secret key in RFC): 9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60 */
			static const uchar rfc_seed[32] = {
				0x9d, 0x61, 0xb1, 0x9d, 0xef, 0xfd, 0x5a, 0x60,
				0xba, 0x84, 0x4a, 0xf4, 0x92, 0xec, 0x2c, 0xc4,
				0x44, 0x49, 0xc5, 0x69, 0x7b, 0x32, 0x69, 0x19,
				0x70, 0x3b, 0xac, 0x03, 0x1c, 0xae, 0x7f, 0x60
			};
			/* Expected public key: d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a */
			static const uchar rfc_pk[32] = {
				0xd7, 0x5a, 0x98, 0x01, 0x82, 0xb1, 0x0a, 0xb7,
				0xd5, 0x4b, 0xfe, 0xd3, 0xc9, 0x64, 0x07, 0x3a,
				0x0e, 0xe1, 0x72, 0xf3, 0xda, 0xa6, 0x23, 0x25,
				0xaf, 0x02, 0x1a, 0x68, 0xf7, 0x07, 0x51, 0x1a
			};
			/* Expected clamped scalar (first 32 bytes of SHA512(seed), clamped):
			   4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb
			   Wait, that's test 2. For test 1 (seed 9d61...), the SHA512 is:
			   f0760e1696cf1a4e55754bd67b1ec05a5b9edc0e0dc0095f85b84b0f5f4d7e68
			   + more bytes, then clamped
			*/
			uchar test_pk[32];
			uchar test_hash[64];
			ge_p3 test_A;

			/* Compute public key from seed */
			fprint(2, "SELFTEST RFC8032: seed[0:32] = ");
			for(i = 0; i < 32; i++) fprint(2, "%02x", rfc_seed[i]);
			fprint(2, "\n(expect 9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60)\n");

			/* Copy to local array to rule out static array issues */
			{
				uchar seed_copy[32];
				memmove(seed_copy, rfc_seed, 32);
				fprint(2, "SELFTEST RFC8032: seed_copy[0:8] = ");
				for(i = 0; i < 8; i++) fprint(2, "%02x", seed_copy[i]);
				fprint(2, "\n");
				sha512(seed_copy, 32, test_hash, nil);
			}

			fprint(2, "SELFTEST RFC8032: sha512(seed)[0:8] = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", test_hash[i]);
			fprint(2, "\nSELFTEST RFC8032: sha512(seed)[24:32] = ");
			for(i = 24; i < 32; i++) fprint(2, "%02x", test_hash[i]);
			fprint(2, "\n");

			test_hash[0] &= 248;
			test_hash[31] &= 127;
			test_hash[31] |= 64;

			fprint(2, "SELFTEST RFC8032: clamped[0:8] = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", test_hash[i]);
			fprint(2, "\nSELFTEST RFC8032: clamped[24:32] = ");
			for(i = 24; i < 32; i++) fprint(2, "%02x", test_hash[i]);
			fprint(2, "\n");

			ge_scalarmult_base(&test_A, test_hash);
			ge_p3_tobytes(test_pk, &test_A);

			fprint(2, "SELFTEST RFC8032: expected_pk[0:8] = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", rfc_pk[i]);
			fprint(2, "\nSELFTEST RFC8032: computed_pk[0:8] = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", test_pk[i]);
			fprint(2, "\nSELFTEST RFC8032: pk_match=%d\n", memcmp(rfc_pk, test_pk, 32) == 0);

			/* Test RFC 8032 signature verification */
			{
				/* RFC 8032 Test 1 signature (message is empty):
				   e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b */
				static const uchar rfc_sig[64] = {
					0xe5, 0x56, 0x43, 0x00, 0xc3, 0x60, 0xac, 0x72,
					0x90, 0x86, 0xe2, 0xcc, 0x80, 0x6e, 0x82, 0x8a,
					0x84, 0x87, 0x7f, 0x1e, 0xb8, 0xe5, 0xd9, 0x74,
					0xd8, 0x73, 0xe0, 0x65, 0x22, 0x49, 0x01, 0x55,
					0x5f, 0xb8, 0x82, 0x15, 0x90, 0xa3, 0x3b, 0xac,
					0xc6, 0x1e, 0x39, 0x70, 0x1c, 0xf9, 0xb4, 0x6b,
					0xd2, 0x5b, 0xf5, 0xf0, 0x59, 0x5b, 0xbe, 0x24,
					0x65, 0x51, 0x41, 0x43, 0x8e, 0x7a, 0x10, 0x0b
				};
				int verify_result;
				fprint(2, "SELFTEST RFC8032 VERIFY: sig[0:8] = ");
				for(i = 0; i < 8; i++) fprint(2, "%02x", rfc_sig[i]);
				fprint(2, "\nSELFTEST RFC8032 VERIFY: pk[0:8] = ");
				for(i = 0; i < 8; i++) fprint(2, "%02x", rfc_pk[i]);
				fprint(2, "\nSELFTEST RFC8032 VERIFY: calling ed25519_verify (empty message)\n");
				verify_result = ed25519_verify(rfc_sig, nil, 0, rfc_pk);
				fprint(2, "SELFTEST RFC8032 VERIFY: result=%d (expected 1)\n", verify_result);
			}
		}
	}

	/* Simple binary double-and-add: h = [a]B */
	ge_p3_0(h);
	for(i = 255; i >= 0; i--){
		ge_p3_dbl(&t, h);
		ge_p1p1_to_p3(h, &t);
		if((a[i/8] >> (i&7)) & 1){
			ge_add(&t, h, &Bcached);
			ge_p1p1_to_p3(h, &t);
		}
	}
}

/*
 * Scalar reduction modulo L (group order)
 * L = 2^252 + 27742317777372353535851937790883648493
 */
static void
sc_reduce(uchar *s)
{
	i64 s0 = 2097151 & load_3(s);
	i64 s1 = 2097151 & (load_4(s + 2) >> 5);
	i64 s2 = 2097151 & (load_3(s + 5) >> 2);
	i64 s3 = 2097151 & (load_4(s + 7) >> 7);
	i64 s4 = 2097151 & (load_4(s + 10) >> 4);
	i64 s5 = 2097151 & (load_3(s + 13) >> 1);
	i64 s6 = 2097151 & (load_4(s + 15) >> 6);
	i64 s7 = 2097151 & (load_3(s + 18) >> 3);
	i64 s8 = 2097151 & load_3(s + 21);
	i64 s9 = 2097151 & (load_4(s + 23) >> 5);
	i64 s10 = 2097151 & (load_3(s + 26) >> 2);
	i64 s11 = 2097151 & (load_4(s + 28) >> 7);
	i64 s12 = 2097151 & (load_4(s + 31) >> 4);
	i64 s13 = 2097151 & (load_3(s + 34) >> 1);
	i64 s14 = 2097151 & (load_4(s + 36) >> 6);
	i64 s15 = 2097151 & (load_3(s + 39) >> 3);
	i64 s16 = 2097151 & load_3(s + 42);
	i64 s17 = 2097151 & (load_4(s + 44) >> 5);
	i64 s18 = 2097151 & (load_3(s + 47) >> 2);
	i64 s19 = 2097151 & (load_4(s + 49) >> 7);
	i64 s20 = 2097151 & (load_4(s + 52) >> 4);
	i64 s21 = 2097151 & (load_3(s + 55) >> 1);
	i64 s22 = 2097151 & (load_4(s + 57) >> 6);
	i64 s23 = (load_4(s + 60) >> 3);
	i64 carry0, carry1, carry2, carry3, carry4, carry5, carry6, carry7;
	i64 carry8, carry9, carry10, carry11, carry12, carry13, carry14, carry15;
	i64 carry16;

	s11 += s23 * 666643;
	s12 += s23 * 470296;
	s13 += s23 * 654183;
	s14 -= s23 * 997805;
	s15 += s23 * 136657;
	s16 -= s23 * 683901;
	s23 = 0;

	s10 += s22 * 666643;
	s11 += s22 * 470296;
	s12 += s22 * 654183;
	s13 -= s22 * 997805;
	s14 += s22 * 136657;
	s15 -= s22 * 683901;
	s22 = 0;

	s9 += s21 * 666643;
	s10 += s21 * 470296;
	s11 += s21 * 654183;
	s12 -= s21 * 997805;
	s13 += s21 * 136657;
	s14 -= s21 * 683901;
	s21 = 0;

	s8 += s20 * 666643;
	s9 += s20 * 470296;
	s10 += s20 * 654183;
	s11 -= s20 * 997805;
	s12 += s20 * 136657;
	s13 -= s20 * 683901;
	s20 = 0;

	s7 += s19 * 666643;
	s8 += s19 * 470296;
	s9 += s19 * 654183;
	s10 -= s19 * 997805;
	s11 += s19 * 136657;
	s12 -= s19 * 683901;
	s19 = 0;

	s6 += s18 * 666643;
	s7 += s18 * 470296;
	s8 += s18 * 654183;
	s9 -= s18 * 997805;
	s10 += s18 * 136657;
	s11 -= s18 * 683901;
	s18 = 0;

	carry6 = (s6 + (1 << 20)) >> 21; s7 += carry6; s6 -= carry6 << 21;
	carry8 = (s8 + (1 << 20)) >> 21; s9 += carry8; s8 -= carry8 << 21;
	carry10 = (s10 + (1 << 20)) >> 21; s11 += carry10; s10 -= carry10 << 21;
	carry12 = (s12 + (1 << 20)) >> 21; s13 += carry12; s12 -= carry12 << 21;
	carry14 = (s14 + (1 << 20)) >> 21; s15 += carry14; s14 -= carry14 << 21;
	carry16 = (s16 + (1 << 20)) >> 21; s17 += carry16; s16 -= carry16 << 21;

	carry7 = (s7 + (1 << 20)) >> 21; s8 += carry7; s7 -= carry7 << 21;
	carry9 = (s9 + (1 << 20)) >> 21; s10 += carry9; s9 -= carry9 << 21;
	carry11 = (s11 + (1 << 20)) >> 21; s12 += carry11; s11 -= carry11 << 21;
	carry13 = (s13 + (1 << 20)) >> 21; s14 += carry13; s13 -= carry13 << 21;
	carry15 = (s15 + (1 << 20)) >> 21; s16 += carry15; s15 -= carry15 << 21;

	s5 += s17 * 666643;
	s6 += s17 * 470296;
	s7 += s17 * 654183;
	s8 -= s17 * 997805;
	s9 += s17 * 136657;
	s10 -= s17 * 683901;
	s17 = 0;

	s4 += s16 * 666643;
	s5 += s16 * 470296;
	s6 += s16 * 654183;
	s7 -= s16 * 997805;
	s8 += s16 * 136657;
	s9 -= s16 * 683901;
	s16 = 0;

	s3 += s15 * 666643;
	s4 += s15 * 470296;
	s5 += s15 * 654183;
	s6 -= s15 * 997805;
	s7 += s15 * 136657;
	s8 -= s15 * 683901;
	s15 = 0;

	s2 += s14 * 666643;
	s3 += s14 * 470296;
	s4 += s14 * 654183;
	s5 -= s14 * 997805;
	s6 += s14 * 136657;
	s7 -= s14 * 683901;
	s14 = 0;

	s1 += s13 * 666643;
	s2 += s13 * 470296;
	s3 += s13 * 654183;
	s4 -= s13 * 997805;
	s5 += s13 * 136657;
	s6 -= s13 * 683901;
	s13 = 0;

	s0 += s12 * 666643;
	s1 += s12 * 470296;
	s2 += s12 * 654183;
	s3 -= s12 * 997805;
	s4 += s12 * 136657;
	s5 -= s12 * 683901;
	s12 = 0;

	carry0 = (s0 + (1 << 20)) >> 21; s1 += carry0; s0 -= carry0 << 21;
	carry2 = (s2 + (1 << 20)) >> 21; s3 += carry2; s2 -= carry2 << 21;
	carry4 = (s4 + (1 << 20)) >> 21; s5 += carry4; s4 -= carry4 << 21;
	carry6 = (s6 + (1 << 20)) >> 21; s7 += carry6; s6 -= carry6 << 21;
	carry8 = (s8 + (1 << 20)) >> 21; s9 += carry8; s8 -= carry8 << 21;
	carry10 = (s10 + (1 << 20)) >> 21; s11 += carry10; s10 -= carry10 << 21;

	carry1 = (s1 + (1 << 20)) >> 21; s2 += carry1; s1 -= carry1 << 21;
	carry3 = (s3 + (1 << 20)) >> 21; s4 += carry3; s3 -= carry3 << 21;
	carry5 = (s5 + (1 << 20)) >> 21; s6 += carry5; s5 -= carry5 << 21;
	carry7 = (s7 + (1 << 20)) >> 21; s8 += carry7; s7 -= carry7 << 21;
	carry9 = (s9 + (1 << 20)) >> 21; s10 += carry9; s9 -= carry9 << 21;
	carry11 = (s11 + (1 << 20)) >> 21; s12 += carry11; s11 -= carry11 << 21;

	s0 += s12 * 666643;
	s1 += s12 * 470296;
	s2 += s12 * 654183;
	s3 -= s12 * 997805;
	s4 += s12 * 136657;
	s5 -= s12 * 683901;
	s12 = 0;

	carry0 = s0 >> 21; s1 += carry0; s0 -= carry0 << 21;
	carry1 = s1 >> 21; s2 += carry1; s1 -= carry1 << 21;
	carry2 = s2 >> 21; s3 += carry2; s2 -= carry2 << 21;
	carry3 = s3 >> 21; s4 += carry3; s3 -= carry3 << 21;
	carry4 = s4 >> 21; s5 += carry4; s4 -= carry4 << 21;
	carry5 = s5 >> 21; s6 += carry5; s5 -= carry5 << 21;
	carry6 = s6 >> 21; s7 += carry6; s6 -= carry6 << 21;
	carry7 = s7 >> 21; s8 += carry7; s7 -= carry7 << 21;
	carry8 = s8 >> 21; s9 += carry8; s8 -= carry8 << 21;
	carry9 = s9 >> 21; s10 += carry9; s9 -= carry9 << 21;
	carry10 = s10 >> 21; s11 += carry10; s10 -= carry10 << 21;
	carry11 = s11 >> 21; s12 += carry11; s11 -= carry11 << 21;

	s0 += s12 * 666643;
	s1 += s12 * 470296;
	s2 += s12 * 654183;
	s3 -= s12 * 997805;
	s4 += s12 * 136657;
	s5 -= s12 * 683901;
	s12 = 0;

	carry0 = s0 >> 21; s1 += carry0; s0 -= carry0 << 21;
	carry1 = s1 >> 21; s2 += carry1; s1 -= carry1 << 21;
	carry2 = s2 >> 21; s3 += carry2; s2 -= carry2 << 21;
	carry3 = s3 >> 21; s4 += carry3; s3 -= carry3 << 21;
	carry4 = s4 >> 21; s5 += carry4; s4 -= carry4 << 21;
	carry5 = s5 >> 21; s6 += carry5; s5 -= carry5 << 21;
	carry6 = s6 >> 21; s7 += carry6; s6 -= carry6 << 21;
	carry7 = s7 >> 21; s8 += carry7; s7 -= carry7 << 21;
	carry8 = s8 >> 21; s9 += carry8; s8 -= carry8 << 21;
	carry9 = s9 >> 21; s10 += carry9; s9 -= carry9 << 21;
	carry10 = s10 >> 21; s11 += carry10; s10 -= carry10 << 21;

	s[0] = (uchar)(s0);
	s[1] = (uchar)(s0 >> 8);
	s[2] = (uchar)((s0 >> 16) | (s1 << 5));
	s[3] = (uchar)(s1 >> 3);
	s[4] = (uchar)(s1 >> 11);
	s[5] = (uchar)((s1 >> 19) | (s2 << 2));
	s[6] = (uchar)(s2 >> 6);
	s[7] = (uchar)((s2 >> 14) | (s3 << 7));
	s[8] = (uchar)(s3 >> 1);
	s[9] = (uchar)(s3 >> 9);
	s[10] = (uchar)((s3 >> 17) | (s4 << 4));
	s[11] = (uchar)(s4 >> 4);
	s[12] = (uchar)(s4 >> 12);
	s[13] = (uchar)((s4 >> 20) | (s5 << 1));
	s[14] = (uchar)(s5 >> 7);
	s[15] = (uchar)((s5 >> 15) | (s6 << 6));
	s[16] = (uchar)(s6 >> 2);
	s[17] = (uchar)(s6 >> 10);
	s[18] = (uchar)((s6 >> 18) | (s7 << 3));
	s[19] = (uchar)(s7 >> 5);
	s[20] = (uchar)(s7 >> 13);
	s[21] = (uchar)(s8);
	s[22] = (uchar)(s8 >> 8);
	s[23] = (uchar)((s8 >> 16) | (s9 << 5));
	s[24] = (uchar)(s9 >> 3);
	s[25] = (uchar)(s9 >> 11);
	s[26] = (uchar)((s9 >> 19) | (s10 << 2));
	s[27] = (uchar)(s10 >> 6);
	s[28] = (uchar)((s10 >> 14) | (s11 << 7));
	s[29] = (uchar)(s11 >> 1);
	s[30] = (uchar)(s11 >> 9);
	s[31] = (uchar)(s11 >> 17);
}

/*
 * Scalar multiply-add: s = (a * b + c) mod L
 * Uses mpint for correctness (slower but verified)
 * a, b, c are 32-byte scalars in little-endian
 * Result s is also 32 bytes little-endian
 */
static void
sc_muladd_simple(uchar *s, const uchar *a, const uchar *b, const uchar *c)
{
	/* L = 2^252 + 27742317777372353535851937790883648493 */
	static const uchar L_bytes[32] = {
		0xed, 0xd3, 0xf5, 0x5c, 0x1a, 0x63, 0x12, 0x58,
		0xd6, 0x9c, 0xf7, 0xa2, 0xde, 0xf9, 0xde, 0x14,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10
	};
	mpint *ma, *mb, *mc, *ml, *mab, *mabc, *ms;
	int i;

	/* Convert from little-endian bytes to mpint */
	ma = betomp((uchar*)a, 32, nil);
	mb = betomp((uchar*)b, 32, nil);
	mc = betomp((uchar*)c, 32, nil);
	ml = betomp((uchar*)L_bytes, 32, nil);

	/* Need to reverse bytes since betomp expects big-endian */
	{
		uchar rev_a[32], rev_b[32], rev_c[32], rev_L[32];
		for(i = 0; i < 32; i++){
			rev_a[i] = a[31-i];
			rev_b[i] = b[31-i];
			rev_c[i] = c[31-i];
			rev_L[i] = L_bytes[31-i];
		}
		mpfree(ma); mpfree(mb); mpfree(mc); mpfree(ml);
		ma = betomp(rev_a, 32, nil);
		mb = betomp(rev_b, 32, nil);
		mc = betomp(rev_c, 32, nil);
		ml = betomp(rev_L, 32, nil);
	}

	/* Compute (a * b + c) mod L */
	mab = mpnew(0);
	mabc = mpnew(0);
	ms = mpnew(0);

	mpmul(ma, mb, mab);       /* mab = a * b */
	mpadd(mab, mc, mabc);     /* mabc = a * b + c */
	mpmod(mabc, ml, ms);      /* ms = (a * b + c) mod L */

	/* Convert back to little-endian bytes */
	{
		uchar be_s[32];
		memset(be_s, 0, 32);
		mptobe(ms, be_s, 32, nil);
		for(i = 0; i < 32; i++)
			s[i] = be_s[31-i];
	}

	mpfree(ma);
	mpfree(mb);
	mpfree(mc);
	mpfree(ml);
	mpfree(mab);
	mpfree(mabc);
	mpfree(ms);
}

/*
 * Ed25519 core operations
 */
static void
ed25519_create_keypair(uchar *pk, uchar *sk, const uchar *seed)
{
	ge_p3 A;
	uchar hash[SHA512dlen];

	/* Hash seed to create secret scalar */
	sha512(seed, Ed25519SeedLen, hash, nil);
	hash[0] &= 248;
	hash[31] &= 127;
	hash[31] |= 64;

	/* Compute public key A = [s]B */
	ge_scalarmult_base(&A, hash);
	ge_p3_tobytes(pk, &A);

	/* DEBUG: verify pk can be decompressed */
	{
		ge_p3 Acheck;
		int i;
		int rc = ge_frombytes_negate_vartime(&Acheck, pk);
		fprint(2, "ed25519_create_keypair: pk[0:8] = ");
		for(i = 0; i < 8; i++) fprint(2, "%02x", pk[i]);
		fprint(2, "\ned25519_create_keypair: ge_frombytes rc=%d\n", rc);
	}

	/* Secret key is seed || public key */
	memmove(sk, seed, Ed25519SeedLen);
	memmove(sk + Ed25519SeedLen, pk, Ed25519PublicKeyLen);
}

static void
ed25519_sign(uchar *sig, const uchar *m, ulong mlen, const uchar *sk)
{
	uchar hash[SHA512dlen];
	uchar hram[SHA512dlen];
	uchar r[SHA512dlen];
	ge_p3 R;
	DigestState *ds;
	int dbgi;

	/* DEBUG SELF-TEST: test fe_tobytes directly */
	{
		uchar testbytes[32];
		static const uchar Bcompressed[32] = {
			0x58, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
			0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
			0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
			0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66
		};

		/* Test: fe_tobytes on By directly */
		fprint(2, "TEST By limbs: ");
		for(dbgi = 0; dbgi < 10; dbgi++) fprint(2, "%d ", By[dbgi]);
		fprint(2, "\n");

		fe_tobytes(testbytes, By);
		fprint(2, "TEST fe_tobytes(By) = ");
		for(dbgi = 0; dbgi < 32; dbgi++) fprint(2, "%02x", testbytes[dbgi]);
		fprint(2, "\nTEST expected       = ");
		for(dbgi = 0; dbgi < 32; dbgi++) fprint(2, "%02x", Bcompressed[dbgi]);
		fprint(2, "\nTEST match=%d\n", memcmp(testbytes, Bcompressed, 32) == 0);
	}

	/* Hash secret key half to get prefix for nonce derivation */
	sha512(sk, Ed25519SeedLen, hash, nil);
	hash[0] &= 248;
	hash[31] &= 127;
	hash[31] |= 64;

	/* DEBUG: verify that sk[32:] == [s]B where s is the clamped hash[0:32] */
	{
		ge_p3 checkA;
		uchar checkpk[32];
		ge_scalarmult_base(&checkA, hash);
		ge_p3_tobytes(checkpk, &checkA);
		fprint(2, "ed25519_sign: DEBUG pk consistency check\n");
		fprint(2, "  sk[32:40] (stored pk) = ");
		for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", sk[32+dbgi]);
		fprint(2, "\n  [s]B (computed pk)    = ");
		for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", checkpk[dbgi]);
		fprint(2, "\n  pk_match = %d\n", memcmp(sk+32, checkpk, 32) == 0);
	}

	/* Compute nonce r = H(prefix || m) */
	ds = sha512(hash + 32, 32, nil, nil);
	sha512(m, mlen, r, ds);
	sc_reduce(r);

	fprint(2, "ed25519_sign: r (nonce)[0:8] = ");
	for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", r[dbgi]);
	fprint(2, "\ned25519_sign: s (scalar)[0:8] = ");
	for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", hash[dbgi]);
	fprint(2, "\n");

	/* Compute R = [r]B */
	ge_scalarmult_base(&R, r);
	ge_p3_tobytes(sig, &R);

	/* DEBUG: Verify that the computed R can be recovered by [r]B */
	{
		ge_p3 Rcheck;
		uchar Rcheck_enc[32];
		ge_scalarmult_base(&Rcheck, r);
		ge_p3_tobytes(Rcheck_enc, &Rcheck);
		fprint(2, "ed25519_sign: R[0:8] = ");
		for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", sig[dbgi]);
		fprint(2, "\ned25519_sign: Rcheck[0:8] (recomputed [r]B) = ");
		for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", Rcheck_enc[dbgi]);
		fprint(2, "\ned25519_sign: R matches recomputed = %d\n", memcmp(sig, Rcheck_enc, 32) == 0);
	}

	/* DEBUG: print R and pk */
	fprint(2, "ed25519_sign: R[0:8] = ");
	for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", sig[dbgi]);
	fprint(2, "\ned25519_sign: pk[0:8] = ");
	for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", sk[32+dbgi]);
	fprint(2, "\ned25519_sign: m[0:8] = ");
	for(dbgi = 0; dbgi < 8 && dbgi < mlen; dbgi++) fprint(2, "%02x", m[dbgi]);
	fprint(2, " (mlen=%ld)\n", mlen);

	/* Compute S = r + H(R || A || m) * s */
	ds = sha512(sig, 32, nil, nil);
	ds = sha512(sk + 32, 32, nil, ds);
	sha512(m, mlen, hram, ds);
	sc_reduce(hram);

	fprint(2, "ed25519_sign: hram[0:8] = ");
	for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", hram[dbgi]);
	fprint(2, "\n");

	/* DEBUG: Print full 32-byte values for Python verification */
	fprint(2, "MULADD_DEBUG: r_full = ");
	for(dbgi = 0; dbgi < 32; dbgi++) fprint(2, "%02x", r[dbgi]);
	fprint(2, "\nMULADD_DEBUG: s_full = ");
	for(dbgi = 0; dbgi < 32; dbgi++) fprint(2, "%02x", hash[dbgi]);
	fprint(2, "\nMULADD_DEBUG: h_full = ");
	for(dbgi = 0; dbgi < 32; dbgi++) fprint(2, "%02x", hram[dbgi]);
	fprint(2, "\n");

	/* s * hram + r mod L -> sig[32..63] */
	/* Use simple mpint-based muladd for correctness */
	sc_muladd_simple(sig + 32, hram, hash, r);

	fprint(2, "ed25519_sign: S (sig[32:40]) = ");
	for(dbgi = 32; dbgi < 40; dbgi++) fprint(2, "%02x", sig[dbgi]);
	fprint(2, "\nMULADD_DEBUG: S_full = ");
	for(dbgi = 32; dbgi < 64; dbgi++) fprint(2, "%02x", sig[dbgi]);
	fprint(2, "\n");

	/* DEBUG: Verify S computation by checking [S]B = [r]B + [h*s]B */
	{
		ge_p3 SB, rB, hsB;
		uchar SB_enc[32], rB_enc[32];
		ge_p1p1 tt;
		ge_cached rBcached;

		/* Compute [S]B directly */
		ge_scalarmult_base(&SB, sig + 32);
		ge_p3_tobytes(SB_enc, &SB);

		/* Compute [r]B (should match R) */
		ge_scalarmult_base(&rB, r);
		ge_p3_tobytes(rB_enc, &rB);

		fprint(2, "ed25519_sign: DEBUG S verification:\n");
		fprint(2, "  [S]B[0:8] = ");
		for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", SB_enc[dbgi]);
		fprint(2, "\n  [r]B[0:8] = ");
		for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", rB_enc[dbgi]);
		fprint(2, "\n  R[0:8]    = ");
		for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", sig[dbgi]);
		fprint(2, "\n  [r]B = R: %d\n", memcmp(rB_enc, sig, 32) == 0);

		/* Now compute [r]B + [h*s]B and check if it equals [S]B */
		/* First compute [h*s]B = [h]([s]B) = [h]A */
		{
			ge_p3 A, hA;
			ge_cached Acached;
			uchar hA_enc[32], SB_minus_rB_enc[32];
			ge_p2 SB_minus_rB_p2;
			int i;

			/* Get A = [s]B from public key */
			ge_frombytes_negate_vartime(&A, sk + 32);  /* This gives -A */
			/* We want +A, so negate again */
			fe_neg(A.X, A.X);
			fe_neg(A.T, A.T);

			/* Compute [h]A */
			ge_p3_to_cached(&Acached, &A);
			ge_p3_0(&hA);
			for(i = 255; i >= 0; i--){
				ge_p3_dbl(&tt, &hA);
				ge_p1p1_to_p3(&hA, &tt);
				if((hram[i/8] >> (i&7)) & 1){
					ge_add(&tt, &hA, &Acached);
					ge_p1p1_to_p3(&hA, &tt);
				}
			}
			ge_p3_tobytes(hA_enc, &hA);

			fprint(2, "  [h]*A[0:8] (= [h*s]B) = ");
			for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", hA_enc[dbgi]);
			fprint(2, "\n");

			/* Compute [r]B + [h*s]B */
			ge_p3_to_cached(&rBcached, &rB);
			{
				ge_cached hAcached;
				ge_p3 sum;
				uchar sum_enc[32];

				ge_p3_to_cached(&hAcached, &hA);
				ge_add(&tt, &rB, &hAcached);
				ge_p1p1_to_p3(&sum, &tt);
				ge_p3_tobytes(sum_enc, &sum);

				fprint(2, "  [r]B + [h*s]B[0:8] = ");
				for(dbgi = 0; dbgi < 8; dbgi++) fprint(2, "%02x", sum_enc[dbgi]);
				fprint(2, "\n  Should equal [S]B: %d\n", memcmp(sum_enc, SB_enc, 32) == 0);
			}
		}
	}

	/* DEBUG: Self-verify the signature we just created */
	{
		int selfverify = ed25519_verify(sig, m, mlen, sk + 32);
		fprint(2, "ed25519_sign: SELF-VERIFY = %d (should be 1)\n", selfverify);
	}

	memset(hash, 0, sizeof(hash));
	memset(r, 0, sizeof(r));
}

static int
ed25519_verify(const uchar *sig, const uchar *m, ulong mlen, const uchar *pk)
{
	ge_p3 A;
	uchar h[SHA512dlen];
	uchar Rcheck[32];
	ge_p3 R;
	ge_p1p1 t;
	ge_p2 Rp2;
	ge_cached Acached;
	DigestState *ds;
	int i;

	fprint(2, "ed25519_verify: enter, mlen=%ld\n", mlen);
	fprint(2, "ed25519_verify: sig[0:8] = ");
	for(i=0;i<8;i++) fprint(2, "%02x", sig[i]);
	fprint(2, "\ned25519_verify: sig[32:40] = ");
	for(i=32;i<40;i++) fprint(2, "%02x", sig[i]);
	fprint(2, "\ned25519_verify: pk[0:8] = ");
	for(i=0;i<8;i++) fprint(2, "%02x", pk[i]);
	fprint(2, "\n");

	/* Decode public key */
	{
		int decoderc = ge_frombytes_negate_vartime(&A, pk);
		if(decoderc != 0) {
			fprint(2, "ed25519_verify: ge_frombytes_negate_vartime failed, rc=%d\n", decoderc);
			return 0;
		}
	}

	/* h = H(R || A || m) */
	ds = sha512(sig, 32, nil, nil);
	ds = sha512(pk, 32, nil, ds);
	sha512(m, mlen, h, ds);
	sc_reduce(h);

	fprint(2, "ed25519_verify: h[0:8] = ");
	for(i = 0; i < 8; i++) fprint(2, "%02x", h[i]);
	fprint(2, "\n");

	/* Compute R' = [S]B - [h]A */
	/* First compute [h]A */
	ge_p3_to_cached(&Acached, &A);

	/* Simple scalar mult for [h]A - note A is already negated */
	ge_p3_0(&R);
	for(i = 255; i >= 0; i--){
		ge_p3_dbl(&t, &R);
		ge_p1p1_to_p3(&R, &t);
		if((h[i/8] >> (i&7)) & 1){
			ge_add(&t, &R, &Acached);
			ge_p1p1_to_p3(&R, &t);
		}
	}

	/* DEBUG: print [h](-A) encoded */
	{
		uchar hA_enc[32];
		ge_p3_tobytes(hA_enc, &R);
		fprint(2, "ed25519_verify: [h](-A)[0:8] = ");
		for(i = 0; i < 8; i++) fprint(2, "%02x", hA_enc[i]);
		fprint(2, "\n");

		/* DEBUG: Verify scalar mult loop by computing [2](-A) and comparing with doubling */
		{
			ge_p3 testR;
			ge_p1p1 testT;
			uchar two[32] = {2, 0};  /* scalar = 2 */
			uchar dblA_enc[32], scalmultA_enc[32];
			int ti;

			/* Method 1: Direct doubling of A */
			ge_p3_dbl(&testT, &A);
			ge_p1p1_to_p3(&testR, &testT);
			ge_p3_tobytes(dblA_enc, &testR);

			/* Method 2: Scalar mult with h=2 */
			ge_p3_0(&testR);
			for(ti = 255; ti >= 0; ti--){
				ge_p3_dbl(&testT, &testR);
				ge_p1p1_to_p3(&testR, &testT);
				if((two[ti/8] >> (ti&7)) & 1){
					ge_add(&testT, &testR, &Acached);
					ge_p1p1_to_p3(&testR, &testT);
				}
			}
			ge_p3_tobytes(scalmultA_enc, &testR);

			fprint(2, "ed25519_verify: DEBUG [2](-A) doubling[0:8] = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", dblA_enc[i]);
			fprint(2, "\ned25519_verify: DEBUG [2](-A) scalar_mult[0:8] = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", scalmultA_enc[i]);
			fprint(2, "\ned25519_verify: DEBUG [2](-A) match = %d\n", memcmp(dblA_enc, scalmultA_enc, 32) == 0);
		}
	}

	/* Now compute [S]B and add to get R' */
	{
		ge_p3 SB;
		ge_cached SBcached;

		ge_scalarmult_base(&SB, sig + 32);

		/* DEBUG: print [S]B encoded */
		{
			uchar SB_enc[32];
			ge_p3_tobytes(SB_enc, &SB);
			fprint(2, "ed25519_verify: [S]B[0:8] = ");
			for(i = 0; i < 8; i++) fprint(2, "%02x", SB_enc[i]);
			fprint(2, "\n");
		}

		ge_p3_to_cached(&SBcached, &SB);
		ge_add(&t, &R, &SBcached);
		ge_p1p1_to_p2(&Rp2, &t);
	}

	/* Convert to bytes and compare with sig[0..31] */
	{
		fe recip, x, y;
		fe_invert(recip, Rp2.Z);
		fe_mul(x, Rp2.X, recip);
		fe_mul(y, Rp2.Y, recip);
		fe_tobytes(Rcheck, y);
		Rcheck[31] ^= fe_isnegative(x) << 7;
	}

	/* DEBUG: print comparison values */
	fprint(2, "ed25519_verify: R (sig)[0:8] = ");
	for(i = 0; i < 8; i++) fprint(2, "%02x", sig[i]);
	fprint(2, "\ned25519_verify: Rcheck[0:8]  = ");
	for(i = 0; i < 8; i++) fprint(2, "%02x", Rcheck[i]);
	fprint(2, "\ned25519_verify: pk[0:8] = ");
	for(i = 0; i < 8; i++) fprint(2, "%02x", pk[i]);
	fprint(2, "\ned25519_verify: m[0:8] = ");
	for(i = 0; i < 8 && i < mlen; i++) fprint(2, "%02x", m[i]);
	fprint(2, " (mlen=%ld)\n", mlen);

	/* Constant time comparison */
	{
		uchar diff = 0;
		for(i = 0; i < 32; i++)
			diff |= sig[i] ^ Rcheck[i];
		fprint(2, "ed25519_verify: result=%d\n", diff == 0);
		return diff == 0;
	}
}
