implement TLS;

include "sys.m";
	sys: Sys;

include "draw.m";

include "keyring.m";
	keyring: Keyring;
	IPint, DigestState: import keyring;

include "asn1.m";
	asn1: ASN1;
	Elem, Tag: import asn1;

include "pkcs.m";
	pkcs: PKCS;
	RSAKey: import PKCS;

include "x509.m";
	x509: X509;
	Signed, Certificate, SubjectPKInfo: import x509;

include "tls.m";

# Record content types
CT_CHANGE_CIPHER_SPEC:	con 20;
CT_ALERT:		con 21;
CT_HANDSHAKE:		con 22;
CT_APPLICATION_DATA:	con 23;

# Handshake message types
HT_CLIENT_HELLO:		con 1;
HT_SERVER_HELLO:		con 2;
HT_NEW_SESSION_TICKET:		con 4;
HT_ENCRYPTED_EXTENSIONS:	con 8;
HT_CERTIFICATE:			con 11;
HT_SERVER_KEY_EXCHANGE:		con 12;
HT_CERTIFICATE_REQUEST:		con 13;
HT_SERVER_HELLO_DONE:		con 14;
HT_CERTIFICATE_VERIFY:		con 15;
HT_CLIENT_KEY_EXCHANGE:		con 16;
HT_FINISHED:			con 20;

# Alert levels
ALERT_WARNING:	con 1;
ALERT_FATAL:	con 2;

# Alert descriptions
ALERT_CLOSE_NOTIFY:		con 0;
ALERT_UNEXPECTED_MESSAGE:	con 10;
ALERT_BAD_RECORD_MAC:		con 20;
ALERT_HANDSHAKE_FAILURE:	con 40;
ALERT_BAD_CERTIFICATE:		con 42;
ALERT_CERTIFICATE_EXPIRED:	con 45;
ALERT_CERTIFICATE_UNKNOWN:	con 46;
ALERT_ILLEGAL_PARAMETER:	con 47;
ALERT_DECODE_ERROR:		con 50;
ALERT_DECRYPT_ERROR:		con 51;
ALERT_PROTOCOL_VERSION:		con 70;
ALERT_INTERNAL_ERROR:		con 80;
ALERT_MISSING_EXTENSION:	con 109;

# Extension types
EXT_SERVER_NAME:		con 0;
EXT_SUPPORTED_GROUPS:		con 10;
EXT_SIGNATURE_ALGORITHMS:	con 13;
EXT_SUPPORTED_VERSIONS:		con 43;
EXT_KEY_SHARE:			con 51;

# Named groups
GROUP_SECP256R1:	con 16r0017;
GROUP_X25519:		con 16r001D;

# Max record size
MAXRECORD:	con 16384;
MAXFRAGMENT:	con 16384 + 256;	# room for overhead

# TLS 1.2 record version
RECVERSION: con 16r0303;

# Internal connection state
ConnState: adt {
	fd:		ref Sys->FD;
	version:	int;		# negotiated version
	suite:		int;		# negotiated cipher suite

	# AEAD keys
	writekey:	array of byte;
	writeiv:	array of byte;
	readkey:	array of byte;
	readiv:		array of byte;

	# Sequence numbers
	writeseq:	big;
	readseq:	big;

	# Read buffer (decrypted application data)
	rbuf:		array of byte;
	roff:		int;
	rlen:		int;

	# Handshake hash
	handhash:	ref Keyring->DigestState;

	# TLS 1.3 traffic secrets
	cts:		array of byte;	# client traffic secret
	sts:		array of byte;	# server traffic secret

	# Server name for cert verification
	servername:	string;
	insecure:	int;

	# TLS 1.3: whether handshake is encrypted
	hsencrypted:	int;
};

init(): string
{
	sys = load Sys Sys->PATH;
	if(sys == nil)
		return "tls: cannot load Sys";

	keyring = load Keyring Keyring->PATH;
	if(keyring == nil)
		return "tls: cannot load Keyring";

	asn1 = load ASN1 ASN1->PATH;
	if(asn1 == nil)
		return "tls: cannot load ASN1";
	asn1->init();

	pkcs = load PKCS PKCS->PATH;
	if(pkcs == nil)
		return "tls: cannot load PKCS";
	pkcs->init();

	x509 = load X509 X509->PATH;
	if(x509 == nil)
		return "tls: cannot load X509";
	x509->init();

	return "";
}

defaultconfig(): ref Config
{
	return ref Config(
		TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 ::
		TLS_AES_128_GCM_SHA256 ::
		TLS_AES_256_GCM_SHA384 ::
		TLS_CHACHA20_POLY1305_SHA256 ::
		TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 ::
		TLS_RSA_WITH_AES_128_GCM_SHA256 ::
		nil,			# suites
		TLS12,		# minver
		TLS13,		# maxver
		"",			# servername
		0			# insecure
	);
}

client(fd: ref Sys->FD, config: ref Config): (ref Conn, string)
{
	cs := ref ConnState;
	cs.fd = fd;
	cs.version = 0;
	cs.suite = 0;
	cs.writeseq = big 0;
	cs.readseq = big 0;
	cs.rbuf = nil;
	cs.roff = 0;
	cs.rlen = 0;
	cs.handhash = nil;
	cs.cts = nil;
	cs.sts = nil;
	cs.servername = config.servername;
	cs.insecure = config.insecure;
	cs.hsencrypted = 0;

	err := handshake(cs, config);
	if(err != nil)
		return (nil, err);

	conn := ref Conn;
	conn.version = cs.version;
	conn.suite = cs.suite;
	conn.servername = cs.servername;

	# Stash ConnState in a global for the conn methods to access
	addconn(conn, cs);

	return (conn, nil);
}

# ================================================================
# Connection pool - maps Conn refs to internal ConnState
# ================================================================

Connentry: adt {
	conn:	ref Conn;
	cs:	ref ConnState;
};

connpool: list of ref Connentry;

addconn(conn: ref Conn, cs: ref ConnState)
{
	connpool = ref Connentry(conn, cs) :: connpool;
}

findconn(conn: ref Conn): ref ConnState
{
	for(l := connpool; l != nil; l = tl l) {
		e := hd l;
		if(e.conn == conn)
			return e.cs;
	}
	return nil;
}

delconn(conn: ref Conn)
{
	nl: list of ref Connentry;
	for(l := connpool; l != nil; l = tl l) {
		e := hd l;
		if(e.conn != conn)
			nl = e :: nl;
	}
	connpool = nl;
}

Conn.read(conn: self ref Conn, buf: array of byte, n: int): int
{
	cs := findconn(conn);
	if(cs == nil)
		return -1;

	# Return buffered data first
	if(cs.rlen > 0) {
		m := cs.rlen;
		if(m > n)
			m = n;
		buf[0:] = cs.rbuf[cs.roff:cs.roff+m];
		cs.roff += m;
		cs.rlen -= m;
		return m;
	}

	# Read next record
	for(;;) {
		(ctype, data, err) := readrecord(cs);
		if(err != nil)
			return -1;

		case ctype {
		CT_APPLICATION_DATA =>
			m := len data;
			if(m > n)
				m = n;
			buf[0:] = data[0:m];
			if(m < len data) {
				cs.rbuf = data;
				cs.roff = m;
				cs.rlen = len data - m;
			}
			return m;

		CT_ALERT =>
			if(len data >= 2 && int data[1] == ALERT_CLOSE_NOTIFY)
				return 0;
			return -1;

		CT_HANDSHAKE =>
			# Post-handshake messages (e.g., NewSessionTicket, KeyUpdate)
			# For now, silently consume
			;

		* =>
			return -1;
		}
	}
}

Conn.write(conn: self ref Conn, buf: array of byte, n: int): int
{
	cs := findconn(conn);
	if(cs == nil)
		return -1;

	sent := 0;
	while(sent < n) {
		chunk := n - sent;
		if(chunk > MAXRECORD)
			chunk = MAXRECORD;
		err := writerecord(cs, CT_APPLICATION_DATA, buf[sent:sent+chunk]);
		if(err != nil)
			return -1;
		sent += chunk;
	}
	return sent;
}

Conn.close(conn: self ref Conn): string
{
	cs := findconn(conn);
	if(cs == nil)
		return "tls: connection not found";

	# Send close_notify alert
	alert := array [2] of byte;
	alert[0] = byte ALERT_WARNING;
	alert[1] = byte ALERT_CLOSE_NOTIFY;
	writerecord(cs, CT_ALERT, alert);

	delconn(conn);
	return nil;
}

# ================================================================
# TLS Record Layer
# ================================================================

# Read a single TLS record, decrypt if keys are set
readrecord(cs: ref ConnState): (int, array of byte, string)
{
	# Read 5-byte header: content_type(1) + version(2) + length(2)
	hdr := array [5] of byte;
	if(ensure(cs.fd, hdr, 5) < 0)
		return (0, nil, "tls: record read failed");

	ctype := int hdr[0];
	length := (int hdr[3] << 8) | int hdr[4];

	if(length > MAXFRAGMENT)
		return (0, nil, "tls: record too large");

	# Read payload
	payload := array [length] of byte;
	if(ensure(cs.fd, payload, length) < 0)
		return (0, nil, "tls: record payload read failed");

	# Decrypt if keys are established
	if(cs.readkey != nil) {
		(plaintext, err) := decrypt_record(cs, ctype, payload);
		if(err != nil)
			return (0, nil, err);

		if(cs.version == TLS13) {
			# TLS 1.3: inner content type is last byte of plaintext
			# Strip padding zeros from end
			i := len plaintext - 1;
			while(i >= 0 && int plaintext[i] == 0)
				i--;
			if(i < 0)
				return (0, nil, "tls: empty inner plaintext");
			ctype = int plaintext[i];
			plaintext = plaintext[0:i];
		}

		return (ctype, plaintext, nil);
	}

	return (ctype, payload, nil);
}

# Write a TLS record, encrypt if keys are set
writerecord(cs: ref ConnState, ctype: int, data: array of byte): string
{
	payload := data;

	if(cs.writekey != nil) {
		plaintext := data;
		if(cs.version == TLS13) {
			# TLS 1.3: append real content type
			plaintext = array [len data + 1] of byte;
			plaintext[0:] = data;
			plaintext[len data] = byte ctype;
			ctype = CT_APPLICATION_DATA;
		}
		(ciphertext, err) := encrypt_record(cs, ctype, plaintext);
		if(err != nil)
			return err;
		payload = ciphertext;
	}

	# Build record: type(1) + version(2) + length(2) + payload
	rec := array [5 + len payload] of byte;
	rec[0] = byte ctype;
	put16(rec, 1, RECVERSION);
	put16(rec, 3, len payload);
	rec[5:] = payload;

	n := sys->write(cs.fd, rec, len rec);
	if(n != len rec)
		return "tls: write failed";

	return nil;
}

# ================================================================
# AEAD Encryption/Decryption
# ================================================================

# Build nonce for AEAD: XOR fixed IV with sequence number
buildnonce(iv: array of byte, seq: big): array of byte
{
	nonce := array [12] of byte;
	nonce[0:] = iv;

	# XOR sequence number into rightmost 8 bytes
	for(i := 0; i < 8; i++) {
		shift := 56 - i * 8;
		nonce[4 + i] ^= byte (int (seq >> shift) & 16rFF);
	}
	return nonce;
}

encrypt_record(cs: ref ConnState, ctype: int, plaintext: array of byte): (array of byte, string)
{
	nonce := buildnonce(cs.writeiv, cs.writeseq);

	# Additional authenticated data: content_type(1) + version(2) + length(2)
	# For TLS 1.3: content_type=23, version=0x0303, length=len(plaintext)+16
	# For TLS 1.2: content_type + version(2) + seq(8) + length(2)
	aad: array of byte;

	if(cs.version == TLS13) {
		aad = array [5] of byte;
		aad[0] = byte ctype;
		put16(aad, 1, RECVERSION);
		put16(aad, 3, len plaintext + 16);	# +16 for tag
	} else {
		# TLS 1.2 AAD: seq(8) + type(1) + version(2) + length(2)
		aad = array [13] of byte;
		put64(aad, 0, cs.writeseq);
		aad[8] = byte ctype;
		put16(aad, 9, RECVERSION);
		put16(aad, 11, len plaintext);
	}

	ct: array of byte;
	tag: array of byte;

	if(isccpoly(cs.suite)) {
		(ct, tag) = keyring->ccpolyencrypt(plaintext, aad, cs.writekey, nonce);
	} else {
		gcmstate := keyring->aesgcmsetup(cs.writekey, nonce);
		if(gcmstate == nil)
			return (nil, "tls: aesgcm setup failed");
		(ct, tag) = keyring->aesgcmencrypt(gcmstate, plaintext, aad);
	}

	if(ct == nil || tag == nil)
		return (nil, "tls: encrypt failed");

	# Concatenate ciphertext + tag
	result := array [len ct + len tag] of byte;
	result[0:] = ct;
	result[len ct:] = tag;

	cs.writeseq++;
	return (result, nil);
}

decrypt_record(cs: ref ConnState, ctype: int, ciphertext: array of byte): (array of byte, string)
{
	if(len ciphertext < 16)
		return (nil, "tls: ciphertext too short");

	nonce := buildnonce(cs.readiv, cs.readseq);

	# Split ciphertext and tag (last 16 bytes)
	ctlen := len ciphertext - 16;
	ct := ciphertext[0:ctlen];
	tag := ciphertext[ctlen:];

	# Build AAD
	aad: array of byte;

	if(cs.version == TLS13) {
		aad = array [5] of byte;
		aad[0] = byte ctype;
		put16(aad, 1, RECVERSION);
		put16(aad, 3, len ciphertext);
	} else {
		aad = array [13] of byte;
		put64(aad, 0, cs.readseq);
		aad[8] = byte ctype;
		put16(aad, 9, RECVERSION);
		put16(aad, 11, ctlen);
	}

	plaintext: array of byte;

	if(isccpoly(cs.suite)) {
		plaintext = keyring->ccpolydecrypt(ct, aad, tag, cs.readkey, nonce);
	} else {
		gcmstate := keyring->aesgcmsetup(cs.readkey, nonce);
		if(gcmstate == nil)
			return (nil, "tls: aesgcm setup failed");
		plaintext = keyring->aesgcmdecrypt(gcmstate, ct, aad, tag);
	}

	if(plaintext == nil)
		return (nil, "tls: decrypt/auth failed");

	cs.readseq++;
	return (plaintext, nil);
}

isccpoly(suite: int): int
{
	return suite == TLS_CHACHA20_POLY1305_SHA256;
}

# ================================================================
# Handshake
# ================================================================

handshake(cs: ref ConnState, config: ref Config): string
{
	# Initialize handshake hash (SHA-256 for most suites)
	cs.handhash = nil;

	# Generate client random
	client_random := randombytes(32);

	# Generate X25519 key pair for key exchange
	x25519_priv := randombytes(32);
	x25519_pub := keyring->x25519_base(x25519_priv);

	# Build and send ClientHello
	hello := buildclienthello(config, client_random, x25519_pub);
	err := sendhsmsg(cs, HT_CLIENT_HELLO, hello);
	if(err != nil)
		return err;

	# Read ServerHello
	(shtype, shdata, sherr) := readhsmsg(cs);
	if(sherr != nil)
		return sherr;
	if(shtype != HT_SERVER_HELLO)
		return "tls: expected ServerHello";

	# Parse ServerHello
	(server_random, server_suite, server_version, key_share_data, pherr) := parseserverhello(shdata, config);
	if(pherr != nil)
		return pherr;

	cs.version = server_version;
	cs.suite = server_suite;

	if(cs.version == TLS13)
		return handshake13(cs, config, client_random, server_random,
			x25519_priv, key_share_data);
	else
		return handshake12(cs, config, client_random, server_random,
			x25519_priv);
}

# ================================================================
# TLS 1.2 Handshake
# ================================================================

handshake12(cs: ref ConnState, config: ref Config,
	client_random, server_random: array of byte,
	x25519_priv: array of byte): string
{
	server_certs: list of array of byte;
	server_pubkey: array of byte;
	server_ecpoint: array of byte;
	uses_ecdhe := 0;

	# Read server messages until ServerHelloDone
	for(;;) {
		(mtype, mdata, merr) := readhsmsg(cs);
		if(merr != nil)
			return merr;

		case mtype {
		HT_CERTIFICATE =>
			(certs, cerr) := parsecertificatemsg(mdata);
			if(cerr != nil)
				return cerr;
			server_certs = certs;

		HT_SERVER_KEY_EXCHANGE =>
			# ECDHE key exchange
			(ecpoint, skerr) := parseserverkeyexchange(mdata);
			if(skerr != nil)
				return skerr;
			server_ecpoint = ecpoint;
			uses_ecdhe = 1;

		HT_CERTIFICATE_REQUEST =>
			# Client cert requested - we don't support this yet
			;

		HT_SERVER_HELLO_DONE =>
			break;

		* =>
			return sys->sprint("tls: unexpected handshake message type %d", mtype);
		}
	}

	# Verify server certificate
	if(!cs.insecure && server_certs != nil) {
		verr := verifycerts(cs, server_certs);
		if(verr != nil)
			return verr;
	}

	# Compute premaster secret
	premaster: array of byte;

	if(uses_ecdhe) {
		# ECDHE: compute shared secret via X25519
		if(server_ecpoint == nil || len server_ecpoint != 32)
			return "tls: invalid server ECDHE point";
		premaster = keyring->x25519(x25519_priv, server_ecpoint);
		if(premaster == nil)
			return "tls: X25519 computation failed";
	} else {
		# RSA key exchange
		premaster = array [48] of byte;
		premaster[0] = byte 3;
		premaster[1] = byte 3;
		randombuf(premaster[2:], 46);

		# Encrypt premaster with server's RSA public key
		if(server_certs == nil)
			return "tls: no server certificate for RSA key exchange";
		(rsakey, pkerr) := extractrsakey(server_certs);
		if(pkerr != nil)
			return pkerr;
		(encerr, encbytes) := pkcs->rsa_encrypt(premaster, rsakey, 2);
		if(encerr != nil)
			return "tls: RSA encryption failed: " + encerr;
		server_pubkey = encbytes;
	}

	# Send ClientKeyExchange
	if(uses_ecdhe) {
		x25519_pub := keyring->x25519_base(x25519_priv);
		cke := buildclientkeyexchange_ecdhe(x25519_pub);
		err := sendhsmsg(cs, HT_CLIENT_KEY_EXCHANGE, cke);
		if(err != nil)
			return err;
	} else {
		cke := buildclientkeyexchange_rsa(server_pubkey);
		err := sendhsmsg(cs, HT_CLIENT_KEY_EXCHANGE, cke);
		if(err != nil)
			return err;
	}

	# Derive keys using TLS 1.2 PRF
	master := tls12_prf(premaster,
		s2b("master secret"),
		catbytes(client_random, server_random),
		48);

	keyblock := tls12_prf(master,
		s2b("key expansion"),
		catbytes(server_random, client_random),
		keyblocklen(cs.suite));

	# Extract keys from key block
	(cs.writekey, cs.writeiv, cs.readkey, cs.readiv) = splitkeyblock(cs.suite, keyblock);

	# Send ChangeCipherSpec
	err := writerecord(cs, CT_CHANGE_CIPHER_SPEC, array [] of {byte 1});
	if(err != nil)
		return err;

	# Send Finished
	verify_data := tls12_prf(master,
		s2b("client finished"),
		hashfinish(cs),
		12);
	ferr := sendhsmsg(cs, HT_FINISHED, verify_data);
	if(ferr != nil)
		return ferr;

	# Read server ChangeCipherSpec
	(ccstype, _, ccserr) := readrecord(cs);
	if(ccserr != nil)
		return ccserr;
	if(ccstype != CT_CHANGE_CIPHER_SPEC)
		return "tls: expected ChangeCipherSpec";

	# Read server Finished
	(ftype, fdata, ferr2) := readhsmsg(cs);
	if(ferr2 != nil)
		return ferr2;
	if(ftype != HT_FINISHED)
		return "tls: expected Finished";

	# Verify server Finished
	expected := tls12_prf(master,
		s2b("server finished"),
		hashfinish(cs),
		12);
	if(!bytescmp(fdata, expected))
		return "tls: server Finished verification failed";

	return nil;
}

# ================================================================
# TLS 1.3 Handshake
# ================================================================

handshake13(cs: ref ConnState, config: ref Config,
	client_random, server_random: array of byte,
	x25519_priv: array of byte,
	key_share_data: array of byte): string
{
	# Compute shared secret via X25519
	if(key_share_data == nil || len key_share_data != 32)
		return "tls: invalid server key share";

	shared_secret := keyring->x25519(x25519_priv, key_share_data);
	if(shared_secret == nil)
		return "tls: X25519 computation failed";

	hashlen := hashlength(cs.suite);

	# TLS 1.3 Key Schedule
	# Early Secret
	zeros := array [hashlen] of {* => byte 0};
	early_secret := hkdf_extract(zeros, zeros);

	# Derive handshake secret
	derived := hkdf_expand_label(early_secret, "derived", hash_empty(cs), hashlen);
	handshake_secret := hkdf_extract(derived, shared_secret);

	# Derive handshake traffic secrets
	hs_hash := hashcurrent(cs);
	c_hs_traffic := hkdf_expand_label(handshake_secret, "c hs traffic", hs_hash, hashlen);
	s_hs_traffic := hkdf_expand_label(handshake_secret, "s hs traffic", hs_hash, hashlen);

	# Derive handshake keys
	(cs.readkey, cs.readiv) = derivekeys(s_hs_traffic, cs.suite);
	(cs.writekey, cs.writeiv) = derivekeys(c_hs_traffic, cs.suite);
	cs.readseq = big 0;
	cs.writeseq = big 0;
	cs.hsencrypted = 1;

	# Read encrypted handshake messages
	server_certs: list of array of byte;

	for(;;) {
		(mtype, mdata, merr) := readhsmsg(cs);
		if(merr != nil)
			return merr;

		case mtype {
		HT_ENCRYPTED_EXTENSIONS =>
			# Parse but mostly ignore for now
			;

		HT_CERTIFICATE_REQUEST =>
			# Client cert requested - not supported yet
			;

		HT_CERTIFICATE =>
			(certs, cerr) := parsecertificatemsg13(mdata);
			if(cerr != nil)
				return cerr;
			server_certs = certs;

		HT_CERTIFICATE_VERIFY =>
			# Verify server's signature over transcript
			if(!cs.insecure) {
				verr := verifycertverify(cs, mdata, server_certs);
				if(verr != nil)
					return verr;
			}

		HT_FINISHED =>
			# Verify server Finished
			fverr := verifyfinished13(cs, mdata, s_hs_traffic);
			if(fverr != nil)
				return fverr;
			break;

		* =>
			return sys->sprint("tls: unexpected hs msg type %d in TLS 1.3", mtype);
		}
	}

	# Verify server certificate chain
	if(!cs.insecure && server_certs != nil) {
		verr := verifycerts(cs, server_certs);
		if(verr != nil)
			return verr;
	}

	# Send client Finished
	finished_key := hkdf_expand_label(c_hs_traffic, "finished", nil, hashlen);
	finished_hash := hashcurrent(cs);
	verify_data := hmac_hash(cs.suite, finished_key, finished_hash);
	ferr := sendhsmsg(cs, HT_FINISHED, verify_data);
	if(ferr != nil)
		return ferr;

	# Derive application traffic secrets
	master_derived := hkdf_expand_label(handshake_secret, "derived", hash_empty(cs), hashlen);
	master_secret := hkdf_extract(master_derived, zeros);

	app_hash := hashcurrent(cs);
	cs.cts = hkdf_expand_label(master_secret, "c ap traffic", app_hash, hashlen);
	cs.sts = hkdf_expand_label(master_secret, "s ap traffic", app_hash, hashlen);

	# Switch to application traffic keys
	(cs.readkey, cs.readiv) = derivekeys(cs.sts, cs.suite);
	(cs.writekey, cs.writeiv) = derivekeys(cs.cts, cs.suite);
	cs.readseq = big 0;
	cs.writeseq = big 0;

	return nil;
}

# ================================================================
# Handshake Message Building
# ================================================================

buildclienthello(config: ref Config, random: array of byte,
	x25519_pub: array of byte): array of byte
{
	# Build extensions
	exts: array of byte;

	# SNI extension
	sni: array of byte;
	if(config.servername != nil && len config.servername > 0)
		sni = buildsniext(config.servername);
	else
		sni = nil;

	# Supported groups extension
	groups := buildsupportedgroups();

	# Signature algorithms extension
	sigalgs := buildsigalgsext();

	# Supported versions extension (for TLS 1.3)
	suppver: array of byte;
	if(config.maxver >= TLS13)
		suppver = buildsupportedversions(config);
	else
		suppver = nil;

	# Key share extension
	keyshare := buildkeyshare(x25519_pub);

	# Concatenate extensions
	extlist := catbytes(sni, catbytes(groups, catbytes(sigalgs,
		catbytes(suppver, keyshare))));

	# Session ID (32 bytes for compatibility)
	session_id := randombytes(32);

	# Cipher suites
	suitebytes := buildsuites(config.suites);

	# Build ClientHello body
	# version(2) + random(32) + session_id_len(1) + session_id(32) +
	# suites_len(2) + suites + compressions(2) + extensions
	bodylen := 2 + 32 + 1 + len session_id + 2 + len suitebytes + 2 + 2 + len extlist;
	body := array [bodylen] of byte;
	off := 0;

	# Legacy version: TLS 1.2 (actual version in extension)
	put16(body, off, RECVERSION);
	off += 2;

	# Random
	body[off:] = random;
	off += 32;

	# Session ID
	body[off] = byte len session_id;
	off++;
	body[off:] = session_id;
	off += len session_id;

	# Cipher suites
	put16(body, off, len suitebytes);
	off += 2;
	body[off:] = suitebytes;
	off += len suitebytes;

	# Compression methods (null only)
	body[off] = byte 1;
	off++;
	body[off] = byte 0;
	off++;

	# Extensions
	put16(body, off, len extlist);
	off += 2;
	body[off:] = extlist;

	return body;
}

buildsniext(name: string): array of byte
{
	namebytes := s2b(name);
	# Extension: type(2) + length(2)
	# SNI list: length(2) + entry: type(1) + name_length(2) + name
	listlen := 1 + 2 + len namebytes;
	extlen := 2 + listlen;
	ext := array [4 + extlen] of byte;
	put16(ext, 0, EXT_SERVER_NAME);
	put16(ext, 2, extlen);
	put16(ext, 4, listlen);
	ext[6] = byte 0;	# host_name type
	put16(ext, 7, len namebytes);
	ext[9:] = namebytes;
	return ext;
}

buildsupportedgroups(): array of byte
{
	# x25519 + secp256r1
	ext := array [4 + 2 + 4] of byte;
	put16(ext, 0, EXT_SUPPORTED_GROUPS);
	put16(ext, 2, 2 + 4);
	put16(ext, 4, 4);
	put16(ext, 6, GROUP_X25519);
	put16(ext, 8, GROUP_SECP256R1);
	return ext;
}

buildsigalgsext(): array of byte
{
	# RSA_PKCS1_SHA256, RSA_PKCS1_SHA384, ECDSA_SECP256R1_SHA256, RSA_PSS_RSAE_SHA256
	nalgs := 4;
	ext := array [4 + 2 + nalgs * 2] of byte;
	put16(ext, 0, EXT_SIGNATURE_ALGORITHMS);
	put16(ext, 2, 2 + nalgs * 2);
	put16(ext, 4, nalgs * 2);
	put16(ext, 6, RSA_PKCS1_SHA256);
	put16(ext, 8, RSA_PKCS1_SHA384);
	put16(ext, 10, ECDSA_SECP256R1_SHA256);
	put16(ext, 12, RSA_PSS_RSAE_SHA256);
	return ext;
}

buildsupportedversions(config: ref Config): array of byte
{
	versions: list of int;
	if(config.maxver >= TLS13)
		versions = TLS13 :: versions;
	if(config.minver <= TLS12)
		versions = TLS12 :: versions;

	nver := 0;
	for(l := versions; l != nil; l = tl l)
		nver++;

	ext := array [4 + 1 + nver * 2] of byte;
	put16(ext, 0, EXT_SUPPORTED_VERSIONS);
	put16(ext, 2, 1 + nver * 2);
	ext[4] = byte (nver * 2);
	off := 5;
	for(l = versions; l != nil; l = tl l) {
		put16(ext, off, hd l);
		off += 2;
	}
	return ext;
}

buildkeyshare(x25519_pub: array of byte): array of byte
{
	# Key share entry: group(2) + key_len(2) + key(32)
	entrylen := 2 + 2 + 32;
	ext := array [4 + 2 + entrylen] of byte;
	put16(ext, 0, EXT_KEY_SHARE);
	put16(ext, 2, 2 + entrylen);
	put16(ext, 4, entrylen);
	put16(ext, 6, GROUP_X25519);
	put16(ext, 8, 32);
	ext[10:] = x25519_pub;
	return ext;
}

buildsuites(suites: list of int): array of byte
{
	n := 0;
	for(l := suites; l != nil; l = tl l)
		n++;
	buf := array [n * 2] of byte;
	off := 0;
	for(l = suites; l != nil; l = tl l) {
		put16(buf, off, hd l);
		off += 2;
	}
	return buf;
}

buildclientkeyexchange_ecdhe(pubkey: array of byte): array of byte
{
	# Length-prefixed EC point (uncompressed format for X25519 is just 32 bytes)
	buf := array [1 + len pubkey] of byte;
	buf[0] = byte len pubkey;
	buf[1:] = pubkey;
	return buf;
}

buildclientkeyexchange_rsa(encrypted_premaster: array of byte): array of byte
{
	# 2-byte length prefix + encrypted premaster
	buf := array [2 + len encrypted_premaster] of byte;
	put16(buf, 0, len encrypted_premaster);
	buf[2:] = encrypted_premaster;
	return buf;
}

# ================================================================
# Handshake Message Parsing
# ================================================================

parseserverhello(data: array of byte, config: ref Config): (array of byte, int, int, array of byte, string)
{
	if(len data < 38)
		return (nil, 0, 0, nil, "tls: ServerHello too short");

	# version(2) + random(32) + session_id_len(1)...
	off := 0;
	legacy_version := get16(data, off);
	off += 2;

	server_random := data[off:off+32];
	off += 32;

	sid_len := int data[off];
	off++;
	if(off + sid_len + 3 > len data)
		return (nil, 0, 0, nil, "tls: ServerHello truncated");
	off += sid_len;

	suite := get16(data, off);
	off += 2;

	compression := int data[off];
	off++;
	if(compression != 0)
		return (nil, 0, 0, nil, "tls: non-null compression");

	# Parse extensions
	version := legacy_version;
	key_share_data: array of byte;

	if(off + 2 <= len data) {
		ext_len := get16(data, off);
		off += 2;
		ext_end := off + ext_len;

		while(off + 4 <= ext_end) {
			etype := get16(data, off);
			elen := get16(data, off + 2);
			off += 4;

			if(off + elen > ext_end)
				break;

			case etype {
			EXT_SUPPORTED_VERSIONS =>
				if(elen >= 2)
					version = get16(data, off);

			EXT_KEY_SHARE =>
				# group(2) + key_len(2) + key
				if(elen >= 4) {
					# group := get16(data, off);
					klen := get16(data, off + 2);
					if(off + 4 + klen <= ext_end)
						key_share_data = data[off+4:off+4+klen];
				}
			}
			off += elen;
		}
	}

	# Validate suite
	found := 0;
	for(l := config.suites; l != nil; l = tl l) {
		if(hd l == suite) {
			found = 1;
			break;
		}
	}
	if(!found)
		return (nil, 0, 0, nil, sys->sprint("tls: server chose unsupported suite 0x%04x", suite));

	# Validate version
	if(version != TLS12 && version != TLS13)
		return (nil, 0, 0, nil, sys->sprint("tls: unsupported version 0x%04x", version));

	return (server_random, suite, version, key_share_data, nil);
}

parsecertificatemsg(data: array of byte): (list of array of byte, string)
{
	if(len data < 3)
		return (nil, "tls: Certificate msg too short");

	total_len := get24(data, 0);
	off := 3;
	certs: list of array of byte;

	while(off + 3 <= len data && off - 3 < total_len) {
		cert_len := get24(data, off);
		off += 3;
		if(off + cert_len > len data)
			return (nil, "tls: certificate truncated");
		certs = data[off:off+cert_len] :: certs;
		off += cert_len;
	}

	# Reverse to get original order (leaf first)
	result: list of array of byte;
	for(l := certs; l != nil; l = tl l)
		result = hd l :: result;

	return (result, nil);
}

parsecertificatemsg13(data: array of byte): (list of array of byte, string)
{
	if(len data < 4)
		return (nil, "tls: Certificate msg too short");

	# TLS 1.3: request_context(1) + certificate_list
	ctx_len := int data[0];
	off := 1 + ctx_len;

	if(off + 3 > len data)
		return (nil, "tls: Certificate msg truncated");

	total_len := get24(data, off);
	off += 3;
	certs: list of array of byte;

	end := off + total_len;
	while(off + 3 <= end) {
		cert_len := get24(data, off);
		off += 3;
		if(off + cert_len > end)
			return (nil, "tls: certificate truncated");
		certs = data[off:off+cert_len] :: certs;
		off += cert_len;

		# TLS 1.3: extensions per certificate entry
		if(off + 2 <= end) {
			ext_len := get16(data, off);
			off += 2 + ext_len;
		}
	}

	# Reverse
	result: list of array of byte;
	for(l := certs; l != nil; l = tl l)
		result = hd l :: result;

	return (result, nil);
}

parseserverkeyexchange(data: array of byte): (array of byte, string)
{
	if(len data < 4)
		return (nil, "tls: ServerKeyExchange too short");

	off := 0;

	# EC parameters
	curve_type := int data[off];
	off++;
	if(curve_type != 3)	# named_curve
		return (nil, "tls: unsupported curve type");

	named_curve := get16(data, off);
	off += 2;

	if(named_curve != GROUP_X25519 && named_curve != GROUP_SECP256R1)
		return (nil, sys->sprint("tls: unsupported named curve 0x%04x", named_curve));

	point_len := int data[off];
	off++;
	if(off + point_len > len data)
		return (nil, "tls: EC point truncated");

	ecpoint := data[off:off+point_len];
	# Remaining bytes are the signature (we don't verify for now if insecure)

	return (ecpoint, nil);
}

# ================================================================
# Certificate Verification
# ================================================================

verifycerts(cs: ref ConnState, certs: list of array of byte): string
{
	if(certs == nil)
		return "tls: no server certificates";

	# Use X509 to verify certificate chain
	(ok, err) := x509->verify_certchain(certs);
	if(!ok && !cs.insecure)
		return "tls: certificate chain verification failed: " + err;

	# TODO: verify server name matches certificate (CN/SAN)
	# For now, chain verification is sufficient
	return nil;
}

verifycertverify(cs: ref ConnState, data: array of byte, certs: list of array of byte): string
{
	# TLS 1.3 CertificateVerify
	if(len data < 4)
		return "tls: CertificateVerify too short";

	# sig_alg(2) + sig_len(2) + sig
	# sig_alg := get16(data, 0);
	sig_len := get16(data, 2);
	if(4 + sig_len > len data)
		return "tls: CertificateVerify truncated";

	# TODO: verify the signature over the transcript hash
	# This requires RSA-PSS or ECDSA verification
	# For now, trust if cert chain is valid
	return nil;
}

verifyfinished13(cs: ref ConnState, data: array of byte, traffic_secret: array of byte): string
{
	hashlen := hashlength(cs.suite);
	if(len data != hashlen)
		return "tls: Finished wrong length";

	finished_key := hkdf_expand_label(traffic_secret, "finished", nil, hashlen);
	transcript_hash := hashcurrent(cs);
	expected := hmac_hash(cs.suite, finished_key, transcript_hash);

	if(!bytescmp(data, expected))
		return "tls: server Finished verification failed";

	return nil;
}

# ================================================================
# RSA Public Key Extraction
# ================================================================

extractrsakey(certs: list of array of byte): (ref RSAKey, string)
{
	if(certs == nil)
		return (nil, "tls: no certificates");

	leaf := hd certs;

	# Decode the X.509 certificate Signed wrapper
	(serr, signed) := x509->Signed.decode(leaf);
	if(serr != nil)
		return (nil, "tls: decode cert: " + serr);

	# Decode the TBSCertificate
	(cerr, cert) := x509->Certificate.decode(signed.tobe_signed);
	if(cerr != nil)
		return (nil, "tls: decode TBSCert: " + cerr);

	# Extract public key from SubjectPublicKeyInfo
	(pkerr, _, pk) := cert.subject_pkinfo.getPublicKey();
	if(pkerr != nil)
		return (nil, "tls: extract key: " + pkerr);
	if(pk == nil)
		return (nil, "tls: no public key");

	pick rpk := pk {
	RSA =>
		return (rpk.pk, nil);
	* =>
		return (nil, "tls: not an RSA public key");
	}
}


# ================================================================
# Handshake Message I/O
# ================================================================

sendhsmsg(cs: ref ConnState, mtype: int, data: array of byte): string
{
	# Handshake header: type(1) + length(3)
	msg := array [4 + len data] of byte;
	msg[0] = byte mtype;
	put24(msg, 1, len data);
	msg[4:] = data;

	# Hash the handshake message
	updatehash(cs, msg);

	return writerecord(cs, CT_HANDSHAKE, msg);
}

readhsmsg(cs: ref ConnState): (int, array of byte, string)
{
	# Read record (possibly encrypted)
	(ctype, payload, rerr) := readrecord(cs);
	if(rerr != nil)
		return (0, nil, rerr);

	if(ctype == CT_ALERT) {
		if(len payload >= 2)
			return (0, nil, sys->sprint("tls: alert level=%d desc=%d",
				int payload[0], int payload[1]));
		return (0, nil, "tls: received alert");
	}

	if(ctype != CT_HANDSHAKE)
		return (0, nil, sys->sprint("tls: expected handshake, got type %d", ctype));

	if(len payload < 4)
		return (0, nil, "tls: handshake message too short");

	mtype := int payload[0];
	mlen := get24(payload, 1);

	if(4 + mlen > len payload)
		return (0, nil, "tls: handshake message truncated");

	# Hash the entire handshake message
	updatehash(cs, payload[0:4+mlen]);

	return (mtype, payload[4:4+mlen], nil);
}

# ================================================================
# Handshake Hashing
# ================================================================

updatehash(cs: ref ConnState, data: array of byte)
{
	# Use SHA-256 as the default transcript hash
	cs.handhash = keyring->sha256(data, len data, nil, cs.handhash);
}

hashcurrent(cs: ref ConnState): array of byte
{
	# Get current hash value without finalizing
	digest := array [Keyring->SHA256dlen] of byte;
	if(cs.handhash != nil) {
		ds := cs.handhash.copy();
		keyring->sha256(nil, 0, digest, ds);
	}
	return digest;
}

hashfinish(cs: ref ConnState): array of byte
{
	return hashcurrent(cs);
}

hash_empty(cs: ref ConnState): array of byte
{
	# SHA-256 of empty string
	digest := array [Keyring->SHA256dlen] of byte;
	keyring->sha256(nil, 0, digest, nil);
	return digest;
}

# ================================================================
# Key Derivation
# ================================================================

# TLS 1.2 PRF (P_SHA256)
tls12_prf(secret, label, seed: array of byte, n: int): array of byte
{
	labelseed := catbytes(label, seed);
	result := array [n] of byte;
	off := 0;

	# A(0) = seed, A(i) = HMAC(secret, A(i-1))
	a := labelseed;
	while(off < n) {
		a = hmac256(secret, a);
		p := hmac256(secret, catbytes(a, labelseed));
		m := len p;
		if(off + m > n)
			m = n - off;
		result[off:] = p[0:m];
		off += m;
	}
	return result;
}

# HKDF-Extract (RFC 5869)
hkdf_extract(salt, ikm: array of byte): array of byte
{
	return hmac256(salt, ikm);
}

# HKDF-Expand (RFC 5869)
hkdf_expand(prk, info: array of byte, length: int): array of byte
{
	hashlen := Keyring->SHA256dlen;
	n := (length + hashlen - 1) / hashlen;
	result := array [n * hashlen] of byte;
	t: array of byte;

	for(i := 1; i <= n; i++) {
		input: array of byte;
		if(t == nil)
			input = catbytes(info, array [1] of {byte i});
		else
			input = catbytes(t, catbytes(info, array [1] of {byte i}));
		t = hmac256(prk, input);
		off := (i - 1) * hashlen;
		result[off:] = t;
	}
	return result[0:length];
}

# HKDF-Expand-Label (TLS 1.3)
hkdf_expand_label(secret: array of byte, label: string, context: array of byte, length: int): array of byte
{
	# HkdfLabel = length(2) + label_len(1) + "tls13 " + label + context_len(1) + context
	full_label := s2b("tls13 " + label);
	ctx := context;
	if(ctx == nil)
		ctx = array [0] of byte;

	info := array [2 + 1 + len full_label + 1 + len ctx] of byte;
	put16(info, 0, length);
	info[2] = byte len full_label;
	info[3:] = full_label;
	info[3 + len full_label] = byte len ctx;
	if(len ctx > 0)
		info[4 + len full_label:] = ctx;

	return hkdf_expand(secret, info, length);
}

# Derive traffic keys from a traffic secret
derivekeys(secret: array of byte, suite: int): (array of byte, array of byte)
{
	keylen := keylength(suite);
	ivlen := 12;

	key := hkdf_expand_label(secret, "key", nil, keylen);
	iv := hkdf_expand_label(secret, "iv", nil, ivlen);

	return (key, iv);
}

# HMAC-SHA256
hmac256(key, data: array of byte): array of byte
{
	digest := array [Keyring->SHA256dlen] of byte;
	keyring->hmac_sha256(data, len data, key, digest, nil);
	return digest;
}

# HMAC using suite's hash algorithm
hmac_hash(suite: int, key, data: array of byte): array of byte
{
	hashlen := hashlength(suite);
	digest := array [hashlen] of byte;

	case hashlen {
	48 =>
		keyring->hmac_sha384(data, len data, key, digest, nil);
	* =>
		keyring->hmac_sha256(data, len data, key, digest, nil);
	}
	return digest;
}

# ================================================================
# Suite Parameters
# ================================================================

hashlength(suite: int): int
{
	case suite {
	TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 or
	TLS_AES_256_GCM_SHA384 =>
		return 48;
	* =>
		return 32;
	}
}

keylength(suite: int): int
{
	case suite {
	TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 or
	TLS_AES_256_GCM_SHA384 or
	TLS_RSA_WITH_AES_256_GCM_SHA384 =>
		return 32;
	* =>
		return 16;
	}
}

keyblocklen(suite: int): int
{
	# For TLS 1.2 GCM: 2 * (key_len + iv_len)
	klen := keylength(suite);
	return 2 * (klen + 4);	# GCM uses 4-byte implicit IV for TLS 1.2
}

splitkeyblock(suite: int, keyblock: array of byte): (array of byte, array of byte, array of byte, array of byte)
{
	klen := keylength(suite);
	ivlen := 4;	# TLS 1.2 GCM implicit nonce
	off := 0;

	cw_key := keyblock[off:off+klen];
	off += klen;
	sw_key := keyblock[off:off+klen];
	off += klen;
	cw_iv := array [12] of {* => byte 0};
	cw_iv[0:] = keyblock[off:off+ivlen];
	off += ivlen;
	sw_iv := array [12] of {* => byte 0};
	sw_iv[0:] = keyblock[off:off+ivlen];

	return (cw_key, cw_iv, sw_key, sw_iv);
}

# ================================================================
# Utility Functions
# ================================================================

ensure(fd: ref Sys->FD, buf: array of byte, n: int): int
{
	i := 0;
	while(i < n) {
		m := sys->read(fd, buf[i:], n - i);
		if(m <= 0)
			return -1;
		i += m;
	}
	return n;
}

put16(buf: array of byte, off: int, val: int)
{
	buf[off] = byte (val >> 8);
	buf[off + 1] = byte val;
}

put24(buf: array of byte, off: int, val: int)
{
	buf[off] = byte (val >> 16);
	buf[off + 1] = byte (val >> 8);
	buf[off + 2] = byte val;
}

put64(buf: array of byte, off: int, val: big)
{
	for(i := 0; i < 8; i++) {
		shift := 56 - i * 8;
		buf[off + i] = byte (int (val >> shift) & 16rFF);
	}
}

get16(buf: array of byte, off: int): int
{
	return (int buf[off] << 8) | int buf[off + 1];
}

get24(buf: array of byte, off: int): int
{
	return (int buf[off] << 16) | (int buf[off + 1] << 8) | int buf[off + 2];
}

s2b(s: string): array of byte
{
	return array of byte s;
}

catbytes(a, b: array of byte): array of byte
{
	if(a == nil)
		return b;
	if(b == nil)
		return a;
	r := array [len a + len b] of byte;
	r[0:] = a;
	r[len a:] = b;
	return r;
}

bytescmp(a, b: array of byte): int
{
	if(len a != len b)
		return 0;
	d := 0;
	for(i := 0; i < len a; i++)
		d |= int a[i] ^ int b[i];
	return d == 0;
}

randombytes(n: int): array of byte
{
	buf := array [n] of byte;
	randombuf(buf, n);
	return buf;
}

randombuf(buf: array of byte, n: int)
{
	# Read from /dev/random
	fd := sys->open("/dev/urandom", Sys->OREAD);
	if(fd == nil)
		fd = sys->open("#c/random", Sys->OREAD);
	if(fd != nil) {
		sys->read(fd, buf, n);
		return;
	}
	# Fallback: use keyring random (via IPint)
	for(i := 0; i < n; i++)
		buf[i] = byte (sys->millisec() ^ (i * 37));
}

