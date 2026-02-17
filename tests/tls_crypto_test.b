implement TLSCryptoTest;

#
# Tests for TLS crypto primitives against RFC/NIST test vectors:
#   - HMAC-SHA256/384/512 (RFC 4231)
#   - AES-GCM (NIST SP 800-38D)
#   - ChaCha20-Poly1305 (RFC 8439)
#   - X25519 (RFC 7748)
#   - P-256 ECDH + ECDSA (stub smoke test)
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "keyring.m";
	kr: Keyring;

include "testing.m";
	testing: Testing;
	T: import testing;

TLSCryptoTest: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

passed := 0;
failed := 0;
skipped := 0;

SRCFILE: con "/tests/tls_crypto_test.b";

run(name: string, testfn: ref fn(t: ref T))
{
	t := testing->newTsrc(name, SRCFILE);
	{
		testfn(t);
	} exception e {
	"fail:fatal" =>
		;
	"fail:skip" =>
		;
	"*" =>
		t.failed = 1;
		t.log("unexpected exception: " + e);
	}

	if(testing->done(t))
		passed++;
	else if(t.skipped)
		skipped++;
	else
		failed++;
}

# Convert hex string to byte array
hexdecode(s: string): array of byte
{
	if(len s % 2 != 0)
		return nil;
	buf := array[len s / 2] of byte;
	for(i := 0; i < len buf; i++) {
		hi := hexval(s[2*i]);
		lo := hexval(s[2*i+1]);
		if(hi < 0 || lo < 0)
			return nil;
		buf[i] = byte (hi * 16 + lo);
	}
	return buf;
}

hexval(c: int): int
{
	if(c >= '0' && c <= '9')
		return c - '0';
	if(c >= 'a' && c <= 'f')
		return c - 'a' + 10;
	if(c >= 'A' && c <= 'F')
		return c - 'A' + 10;
	return -1;
}

# Convert byte array to hex string
hexencode(buf: array of byte): string
{
	if(buf == nil)
		return "nil";
	s := "";
	for(i := 0; i < len buf; i++)
		s += sys->sprint("%02x", int buf[i]);
	return s;
}

# Compare two byte arrays
byteseq(a, b: array of byte): int
{
	if(a == nil && b == nil)
		return 1;
	if(a == nil || b == nil)
		return 0;
	if(len a != len b)
		return 0;
	for(i := 0; i < len a; i++)
		if(a[i] != b[i])
			return 0;
	return 1;
}

# Assert byte arrays equal, showing hex on failure
assertbytes(t: ref T, got, want: array of byte, msg: string)
{
	if(!byteseq(got, want)) {
		ghex := hexencode(got);
		whex := hexencode(want);
		if(len ghex > 80)
			ghex = ghex[0:80] + "...";
		if(len whex > 80)
			whex = whex[0:80] + "...";
		t.error(sys->sprint("%s:\n  got  %s\n  want %s", msg, ghex, whex));
	}
}

# Make array of n copies of byte b
makebytes(n: int, b: byte): array of byte
{
	buf := array[n] of byte;
	for(i := 0; i < n; i++)
		buf[i] = b;
	return buf;
}

# ============================================================
# HMAC-SHA256 tests (RFC 4231)
# ============================================================

testHMACSHA256(t: ref T)
{
	t.log("Testing HMAC-SHA256 against RFC 4231...");

	# Test Case 1: 20-byte key
	key1 := makebytes(20, byte 16r0b);
	data1 := array of byte "Hi There";
	want1 := hexdecode("b0344c61d8db38535ca8afceaf0bf12b"
		+ "881dc200c9833da726e9376c2e32cff7");
	digest1 := array[kr->SHA256dlen] of byte;
	kr->hmac_sha256(data1, len data1, key1, digest1, nil);
	assertbytes(t, digest1, want1, "HMAC-SHA256 TC1");

	# Test Case 2: short key "Jefe"
	key2 := array of byte "Jefe";
	data2 := array of byte "what do ya want for nothing?";
	want2 := hexdecode("5bdcc146bf60754e6a042426089575c7"
		+ "5a003f089d2739839dec58b964ec3843");
	digest2 := array[kr->SHA256dlen] of byte;
	kr->hmac_sha256(data2, len data2, key2, digest2, nil);
	assertbytes(t, digest2, want2, "HMAC-SHA256 TC2");

	# Test Case 6: key longer than block size (131 bytes > 64-byte block)
	# This tests the key-hashing path we fixed in hmac.c
	key6 := makebytes(131, byte 16raa);
	data6 := array of byte "Test Using Larger Than Block-Size Key - Hash Key First";
	want6 := hexdecode("60e431591ee0b67f0d8a26aacbf5b77f"
		+ "8e0bc6213728c5140546040f0ee37f54");
	digest6 := array[kr->SHA256dlen] of byte;
	kr->hmac_sha256(data6, len data6, key6, digest6, nil);
	assertbytes(t, digest6, want6, "HMAC-SHA256 TC6 (long key)");
}

# ============================================================
# HMAC-SHA384 tests (RFC 4231)
# ============================================================

testHMACSHA384(t: ref T)
{
	t.log("Testing HMAC-SHA384 against RFC 4231...");

	# Test Case 2
	key2 := array of byte "Jefe";
	data2 := array of byte "what do ya want for nothing?";
	want2 := hexdecode("af45d2e376484031617f78d2b58a6b1b"
		+ "9c7ef464f5a01b47e42ec3736322445e"
		+ "8e2240ca5e69e2c78b3239ecfab21649");
	digest2 := array[kr->SHA384dlen] of byte;
	kr->hmac_sha384(data2, len data2, key2, digest2, nil);
	assertbytes(t, digest2, want2, "HMAC-SHA384 TC2");

	# Test Case 6: key longer than block size (131 > 128-byte block for SHA-384)
	key6 := makebytes(131, byte 16raa);
	data6 := array of byte "Test Using Larger Than Block-Size Key - Hash Key First";
	want6 := hexdecode("4ece084485813e9088d2c63a041bc5b4"
		+ "4f9ef1012a2b588f3cd11f05033ac4c6"
		+ "0c2ef6ab4030fe8296248df163f44952");
	digest6 := array[kr->SHA384dlen] of byte;
	kr->hmac_sha384(data6, len data6, key6, digest6, nil);
	assertbytes(t, digest6, want6, "HMAC-SHA384 TC6 (long key)");
}

# ============================================================
# HMAC-SHA512 tests (RFC 4231)
# ============================================================

testHMACSHA512(t: ref T)
{
	t.log("Testing HMAC-SHA512 against RFC 4231...");

	# Test Case 2
	key2 := array of byte "Jefe";
	data2 := array of byte "what do ya want for nothing?";
	want2 := hexdecode("164b7a7bfcf819e2e395fbe73b56e0a3"
		+ "87bd64222e831fd610270cd7ea250554"
		+ "9758bf75c05a994a6d034f65f8f0e6fd"
		+ "caeab1a34d4a6b4b636e070a38bce737");
	digest2 := array[kr->SHA512dlen] of byte;
	kr->hmac_sha512(data2, len data2, key2, digest2, nil);
	assertbytes(t, digest2, want2, "HMAC-SHA512 TC2");

	# Test Case 6: key longer than block size (131 > 128-byte block for SHA-512)
	key6 := makebytes(131, byte 16raa);
	data6 := array of byte "Test Using Larger Than Block-Size Key - Hash Key First";
	want6 := hexdecode("80b24263c7c1a3ebb71493c1dd7be8b4"
		+ "9b46d1f41b4aeec1121b013783f8f352"
		+ "6b56d037e05f2598bd0fd2215d6a1e52"
		+ "95e64f73f63f0aec8b915a985d786598");
	digest6 := array[kr->SHA512dlen] of byte;
	kr->hmac_sha512(data6, len data6, key6, digest6, nil);
	assertbytes(t, digest6, want6, "HMAC-SHA512 TC6 (long key)");
}

# ============================================================
# AES-128-GCM tests (NIST SP 800-38D)
# ============================================================

testAESGCMEmpty(t: ref T)
{
	t.log("Testing AES-GCM with empty plaintext (NIST TC1)...");

	key := hexdecode("00000000000000000000000000000000");
	iv := hexdecode("000000000000000000000000");
	empty := array[0] of byte;
	wanttag := hexdecode("58e2fccefa7e3061367f1d57a4e7455a");

	state := kr->aesgcmsetup(key, iv);
	if(state == nil) {
		t.fatal("aesgcmsetup returned nil");
		return;
	}

	(ct, tag) := kr->aesgcmencrypt(state, empty, empty);
	t.asserteq(len ct, 0, "AES-GCM TC1 ciphertext should be empty");
	assertbytes(t, tag, wanttag, "AES-GCM TC1 tag");

	# Verify decryption
	state2 := kr->aesgcmsetup(key, iv);
	pt := kr->aesgcmdecrypt(state2, empty, empty, wanttag);
	if(pt == nil)
		t.error("AES-GCM TC1 decrypt returned nil (auth failed)");
	else
		t.asserteq(len pt, 0, "AES-GCM TC1 decrypted plaintext should be empty");
}

testAESGCMZeros(t: ref T)
{
	t.log("Testing AES-GCM with zero plaintext (NIST TC2)...");

	key := hexdecode("00000000000000000000000000000000");
	iv := hexdecode("000000000000000000000000");
	pt := hexdecode("00000000000000000000000000000000");
	wantct := hexdecode("0388dace60b6a392f328c2b971b2fe78");
	wanttag := hexdecode("ab6e47d42cec13bdf53a67b21257bddf");

	state := kr->aesgcmsetup(key, iv);
	if(state == nil) {
		t.fatal("aesgcmsetup returned nil");
		return;
	}

	(ct, tag) := kr->aesgcmencrypt(state, pt, array[0] of byte);
	assertbytes(t, ct, wantct, "AES-GCM TC2 ciphertext");
	assertbytes(t, tag, wanttag, "AES-GCM TC2 tag");

	# Verify round-trip decryption
	state2 := kr->aesgcmsetup(key, iv);
	dec := kr->aesgcmdecrypt(state2, wantct, array[0] of byte, wanttag);
	assertbytes(t, dec, pt, "AES-GCM TC2 round-trip");
}

testAESGCMData(t: ref T)
{
	t.log("Testing AES-GCM with data (NIST TC3)...");

	key := hexdecode("feffe9928665731c6d6a8f9467308308");
	iv := hexdecode("cafebabefacedbaddecaf888");
	pt := hexdecode(
		"d9313225f88406e5a55909c5aff5269a"
		+ "86a7a9531534f7da2e4c303d8a318a72"
		+ "1c3c0c95956809532fcf0e2449a6b525"
		+ "b16aedf5aa0de657ba637b391aafd255");
	wantct := hexdecode(
		"42831ec2217774244b7221b784d0d49c"
		+ "e3aa212f2c02a4e035c17e2329aca12e"
		+ "21d514b25466931c7d8f6a5aac84aa05"
		+ "1ba30b396a0aac973d58e091473f5985");
	wanttag := hexdecode("4d5c2af327cd64a62cf35abd2ba6fab4");

	state := kr->aesgcmsetup(key, iv);
	if(state == nil) {
		t.fatal("aesgcmsetup returned nil");
		return;
	}

	(ct, tag) := kr->aesgcmencrypt(state, pt, array[0] of byte);
	assertbytes(t, ct, wantct, "AES-GCM TC3 ciphertext");
	assertbytes(t, tag, wanttag, "AES-GCM TC3 tag");
}

testAESGCMWithAAD(t: ref T)
{
	t.log("Testing AES-GCM with AAD (NIST TC4)...");

	key := hexdecode("feffe9928665731c6d6a8f9467308308");
	iv := hexdecode("cafebabefacedbaddecaf888");
	pt := hexdecode(
		"d9313225f88406e5a55909c5aff5269a"
		+ "86a7a9531534f7da2e4c303d8a318a72"
		+ "1c3c0c95956809532fcf0e2449a6b525"
		+ "b16aedf5aa0de657ba637b39");
	aad := hexdecode("feedfacedeadbeeffeedfacedeadbeefabaddad2");
	wantct := hexdecode(
		"42831ec2217774244b7221b784d0d49c"
		+ "e3aa212f2c02a4e035c17e2329aca12e"
		+ "21d514b25466931c7d8f6a5aac84aa05"
		+ "1ba30b396a0aac973d58e091");
	wanttag := hexdecode("5bc94fbc3221a5db94fae95ae7121a47");

	state := kr->aesgcmsetup(key, iv);
	if(state == nil) {
		t.fatal("aesgcmsetup returned nil");
		return;
	}

	(ct, tag) := kr->aesgcmencrypt(state, pt, aad);
	assertbytes(t, ct, wantct, "AES-GCM TC4 ciphertext");
	assertbytes(t, tag, wanttag, "AES-GCM TC4 tag");

	# Verify decryption with correct tag
	state2 := kr->aesgcmsetup(key, iv);
	dec := kr->aesgcmdecrypt(state2, wantct, aad, wanttag);
	assertbytes(t, dec, pt, "AES-GCM TC4 round-trip");

	# Verify decryption fails with wrong tag
	badtag := hexdecode("00000000000000000000000000000000");
	state3 := kr->aesgcmsetup(key, iv);
	bad := kr->aesgcmdecrypt(state3, wantct, aad, badtag);
	if(bad != nil)
		t.error("AES-GCM TC4 decrypt should fail with wrong tag");
}

# ============================================================
# ChaCha20-Poly1305 tests (RFC 8439 section 2.8.2)
# ============================================================

testCCPolyEncrypt(t: ref T)
{
	t.log("Testing ChaCha20-Poly1305 against RFC 8439 §2.8.2...");

	key := hexdecode(
		"808182838485868788898a8b8c8d8e8f"
		+ "909192939495969798999a9b9c9d9e9f");
	nonce := hexdecode("070000004041424344454647");
	aad := hexdecode("50515253c0c1c2c3c4c5c6c7");
	pt := array of byte "Ladies and Gentlemen of the class of '99: If I could offer you only one tip for the future, sunscreen would be it.";
	wantct := hexdecode(
		"d31a8d34648e60db7b86afbc53ef7ec2"
		+ "a4aded51296e08fea9e2b5a736ee62d6"
		+ "3dbea45e8ca9671282fafb69da92728b"
		+ "1a71de0a9e060b2905d6a5b67ecd3b36"
		+ "92ddbd7f2d778b8c9803aee328091b58"
		+ "fab324e4fad675945585808b4831d7bc"
		+ "3ff4def08e4b7a9de576d26586cec64b"
		+ "6116");
	wanttag := hexdecode("1ae10b594f09e26a7e902ecbd0600691");

	(ct, tag) := kr->ccpolyencrypt(pt, aad, key, nonce);
	assertbytes(t, ct, wantct, "CC20P1305 ciphertext");
	assertbytes(t, tag, wanttag, "CC20P1305 tag");
}

testCCPolyDecrypt(t: ref T)
{
	t.log("Testing ChaCha20-Poly1305 decryption...");

	key := hexdecode(
		"808182838485868788898a8b8c8d8e8f"
		+ "909192939495969798999a9b9c9d9e9f");
	nonce := hexdecode("070000004041424344454647");
	aad := hexdecode("50515253c0c1c2c3c4c5c6c7");
	ct := hexdecode(
		"d31a8d34648e60db7b86afbc53ef7ec2"
		+ "a4aded51296e08fea9e2b5a736ee62d6"
		+ "3dbea45e8ca9671282fafb69da92728b"
		+ "1a71de0a9e060b2905d6a5b67ecd3b36"
		+ "92ddbd7f2d778b8c9803aee328091b58"
		+ "fab324e4fad675945585808b4831d7bc"
		+ "3ff4def08e4b7a9de576d26586cec64b"
		+ "6116");
	tag := hexdecode("1ae10b594f09e26a7e902ecbd0600691");
	wantpt := array of byte "Ladies and Gentlemen of the class of '99: If I could offer you only one tip for the future, sunscreen would be it.";

	pt := kr->ccpolydecrypt(ct, aad, tag, key, nonce);
	assertbytes(t, pt, wantpt, "CC20P1305 decrypt");

	# Verify decryption fails with wrong tag
	badtag := hexdecode("00000000000000000000000000000000");
	bad := kr->ccpolydecrypt(ct, aad, badtag, key, nonce);
	if(bad != nil)
		t.error("CC20P1305 decrypt should fail with wrong tag");
}

# ============================================================
# X25519 tests (RFC 7748 section 6.1)
# ============================================================

testX25519(t: ref T)
{
	t.log("Testing X25519 against RFC 7748 §6.1...");

	# Alice's keys
	alice_priv := hexdecode(
		"77076d0a7318a57d3c16c17251b26645"
		+ "df4c2f87ebc0992ab177fba51db92c2a");
	alice_pub_want := hexdecode(
		"8520f0098930a754748b7ddcb43ef75a"
		+ "0dbf3a0d26381af4eba4a98eaa9b4e6a");

	# Bob's keys
	bob_priv := hexdecode(
		"5dab087e624a8a4b79e17f8b83800ee6"
		+ "6f3bb1292618b6fd1c2f8b27ff88e0eb");
	bob_pub_want := hexdecode(
		"de9edb7d7b7dc1b4d35b61c2ece43537"
		+ "3f8343c85b78674dadfc7e146f882b4f");

	# Shared secret
	shared_want := hexdecode(
		"4a5d9d5ba4ce2de1728e3bf480350f25"
		+ "e07e21c947d19e3376f09b3c1e161742");

	# Test x25519_base: scalar * basepoint
	alice_pub := kr->x25519_base(alice_priv);
	if(alice_pub == nil) {
		t.fatal("x25519_base returned nil for Alice");
		return;
	}
	assertbytes(t, alice_pub, alice_pub_want, "Alice public key");

	bob_pub := kr->x25519_base(bob_priv);
	if(bob_pub == nil) {
		t.fatal("x25519_base returned nil for Bob");
		return;
	}
	assertbytes(t, bob_pub, bob_pub_want, "Bob public key");

	# Test x25519: ECDH shared secret
	shared_ab := kr->x25519(alice_priv, bob_pub_want);
	if(shared_ab == nil) {
		t.fatal("x25519(alice, bob) returned nil");
		return;
	}
	assertbytes(t, shared_ab, shared_want, "shared secret (Alice*Bob)");

	shared_ba := kr->x25519(bob_priv, alice_pub_want);
	if(shared_ba == nil) {
		t.fatal("x25519(bob, alice) returned nil");
		return;
	}
	assertbytes(t, shared_ba, shared_want, "shared secret (Bob*Alice)");

	# Both sides should agree
	assertbytes(t, shared_ab, shared_ba, "ECDH commutativity");
}

# ============================================================
# P-256 smoke test (ecc.c is a STUB — just verify no crash)
# ============================================================

testP256Smoke(t: ref T)
{
	t.log("Testing P-256 (stub smoke test)...");
	t.log("NOTE: ecc.c is a stub returning zeros. Testing crash safety only.");

	# keygen should not crash
	(priv, pub) := kr->p256_keygen();
	if(priv == nil) {
		t.error("p256_keygen returned nil priv");
		return;
	}
	if(pub == nil) {
		t.error("p256_keygen returned nil pub");
		return;
	}
	t.log(sys->sprint("p256_keygen: priv=%d bytes, pub=non-nil", len priv));

	# ecdh should not crash
	shared := kr->p256_ecdh(priv, pub);
	if(shared == nil)
		t.log("p256_ecdh returned nil (expected from stub)");
	else
		t.log(sys->sprint("p256_ecdh: shared=%d bytes", len shared));

	# ecdsa_sign should not crash
	hash := hexdecode("e3b0c44298fc1c149afbf4c8996fb924"
		+ "27ae41e4649b934ca495991b7852b855");  # SHA-256 of ""
	sig := kr->p256_ecdsa_sign(priv, hash);
	if(sig == nil)
		t.log("p256_ecdsa_sign returned nil (expected from stub)");
	else
		t.log(sys->sprint("p256_ecdsa_sign: sig=%d bytes", len sig));

	# ecdsa_verify should not crash
	if(sig != nil) {
		result := kr->p256_ecdsa_verify(pub, hash, sig);
		t.log(sys->sprint("p256_ecdsa_verify: result=%d (0 expected from stub)", result));
	}

	t.log("P-256 stub: no crashes, awaiting real implementation");
}

# ============================================================
# Main
# ============================================================

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	kr = load Keyring Keyring->PATH;
	testing = load Testing Testing->PATH;

	if(kr == nil) {
		sys->fprint(sys->fildes(2), "cannot load Keyring: %r\n");
		raise "fail:cannot load Keyring";
	}

	if(testing == nil) {
		sys->fprint(sys->fildes(2), "cannot load Testing: %r\n");
		raise "fail:cannot load Testing";
	}

	testing->init();

	for(a := args; a != nil; a = tl a) {
		if(hd a == "-v")
			testing->verbose(1);
	}

	sys->fprint(sys->fildes(2), "\n=== TLS Crypto Primitive Tests ===\n\n");

	# HMAC tests
	run("HMAC/SHA256", testHMACSHA256);
	run("HMAC/SHA384", testHMACSHA384);
	run("HMAC/SHA512", testHMACSHA512);

	# AES-GCM tests
	run("AES-GCM/Empty", testAESGCMEmpty);
	run("AES-GCM/Zeros", testAESGCMZeros);
	run("AES-GCM/Data", testAESGCMData);
	run("AES-GCM/WithAAD", testAESGCMWithAAD);

	# ChaCha20-Poly1305 tests
	run("CCPoly/Encrypt", testCCPolyEncrypt);
	run("CCPoly/Decrypt", testCCPolyDecrypt);

	# X25519 tests
	run("X25519/RFC7748", testX25519);

	# P-256 smoke test
	run("P256/Smoke", testP256Smoke);

	if(testing->summary(passed, failed, skipped) > 0)
		raise "fail:tests failed";
}
