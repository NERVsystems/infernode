implement Test9P;

include "sys.m";
	sys: Sys;
include "draw.m";

Test9P: module
{
	init: fn(nil: ref Draw->Context, args: list of string);
};

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;

	if(len args < 2){
		sys->print("usage: test9p tcp!host!port\n");
		return;
	}

	addr := hd tl args;
	sys->print("Dialing %s...\n", addr);

	(ok, c) := sys->dial(addr, nil);
	if(ok < 0){
		sys->print("dial failed: %r\n");
		return;
	}
	sys->print("Connected.\n");

	# Build Tversion: size[4] Tversion[1] tag[2] msize[4] version[s]
	# msize = 8192, version = "9P2000"
	msize := 8192;
	version := array of byte "9P2000";
	vlen := len version;
	msglen := 4 + 1 + 2 + 4 + 2 + vlen;  # total message size

	msg := array[msglen] of byte;
	# size[4] - little endian
	msg[0] = byte msglen;
	msg[1] = byte (msglen >> 8);
	msg[2] = byte (msglen >> 16);
	msg[3] = byte (msglen >> 24);
	# type[1] - Tversion = 100
	msg[4] = byte 100;
	# tag[2] - NOTAG = 0xFFFF
	msg[5] = byte 16rFF;
	msg[6] = byte 16rFF;
	# msize[4]
	msg[7] = byte msize;
	msg[8] = byte (msize >> 8);
	msg[9] = byte (msize >> 16);
	msg[10] = byte (msize >> 24);
	# version length[2]
	msg[11] = byte vlen;
	msg[12] = byte (vlen >> 8);
	# version string
	for(i := 0; i < vlen; i++)
		msg[13 + i] = version[i];

	sys->print("Sending Tversion (%d bytes)...\n", msglen);
	printhex(msg);

	n := sys->write(c.dfd, msg, len msg);
	if(n != len msg){
		sys->print("write failed: wrote %d of %d: %r\n", n, len msg);
		return;
	}
	sys->print("Sent Tversion.\n");

	# Read Rversion
	sys->print("Reading Rversion...\n");
	resp := array[256] of byte;
	n = sys->read(c.dfd, resp, len resp);
	if(n <= 0){
		sys->print("read failed: %r\n");
		return;
	}

	sys->print("Got response (%d bytes):\n", n);
	printhex(resp[0:n]);

	# Parse basic header
	if(n >= 7){
		size := int resp[0] | (int resp[1] << 8) | (int resp[2] << 16) | (int resp[3] << 24);
		mtype := int resp[4];
		tag := int resp[5] | (int resp[6] << 8);
		sys->print("  size=%d type=%d (Rversion=101) tag=%d\n", size, mtype, tag);

		if(mtype == 101 && n >= 13){
			rmsize := int resp[7] | (int resp[8] << 8) | (int resp[9] << 16) | (int resp[10] << 24);
			rvlen := int resp[11] | (int resp[12] << 8);
			rversion := string resp[13:13+rvlen];
			sys->print("  msize=%d version=%s\n", rmsize, rversion);
		}
	}

	# Now try Tattach
	sys->print("\nSending Tattach...\n");

	# Get username
	uname := array of byte "inferno";
	aname := array of byte "";
	fid := 1;
	afid := ~0;  # NOFID

	attlen := 4 + 1 + 2 + 4 + 4 + 2 + len uname + 2 + len aname;
	att := array[attlen] of byte;

	# size[4]
	att[0] = byte attlen;
	att[1] = byte (attlen >> 8);
	att[2] = byte (attlen >> 16);
	att[3] = byte (attlen >> 24);
	# type[1] - Tattach = 104
	att[4] = byte 104;
	# tag[2]
	att[5] = byte 1;
	att[6] = byte 0;
	# fid[4]
	att[7] = byte fid;
	att[8] = byte (fid >> 8);
	att[9] = byte (fid >> 16);
	att[10] = byte (fid >> 24);
	# afid[4]
	att[11] = byte afid;
	att[12] = byte (afid >> 8);
	att[13] = byte (afid >> 16);
	att[14] = byte (afid >> 24);
	# uname[s]
	att[15] = byte len uname;
	att[16] = byte (len uname >> 8);
	for(i = 0; i < len uname; i++)
		att[17 + i] = uname[i];
	# aname[s]
	off := 17 + len uname;
	att[off] = byte len aname;
	att[off+1] = byte (len aname >> 8);

	printhex(att);

	n = sys->write(c.dfd, att, len att);
	if(n != len att){
		sys->print("write Tattach failed: wrote %d of %d: %r\n", n, len att);
		return;
	}
	sys->print("Sent Tattach.\n");

	# Read Rattach
	sys->print("Reading Rattach...\n");
	n = sys->read(c.dfd, resp, len resp);
	if(n <= 0){
		sys->print("read Rattach failed: %r\n");
		return;
	}

	sys->print("Got Rattach (%d bytes):\n", n);
	printhex(resp[0:n]);

	if(n >= 7){
		size := int resp[0] | (int resp[1] << 8) | (int resp[2] << 16) | (int resp[3] << 24);
		mtype := int resp[4];
		tag := int resp[5] | (int resp[6] << 8);
		sys->print("  size=%d type=%d (Rattach=105) tag=%d\n", size, mtype, tag);

		if(mtype == 105 && n >= 20){
			qtype := int resp[7];
			qvers := int resp[8] | (int resp[9] << 8) | (int resp[10] << 16) | (int resp[11] << 24);
			qpath := big resp[12] | (big resp[13] << 8) | (big resp[14] << 16) | (big resp[15] << 24) |
			         (big resp[16] << 32) | (big resp[17] << 40) | (big resp[18] << 48) | (big resp[19] << 56);
			sys->print("  qid: type=0x%02x vers=%d path=%bd\n", qtype, qvers, qpath);
		} else if(mtype == 107 && n >= 9){
			elen := int resp[7] | (int resp[8] << 8);
			ename := string resp[9:9+elen];
			sys->print("  ERROR: %s\n", ename);
		}
	}

	sys->print("\n9P test complete.\n");
}

printhex(data: array of byte)
{
	sys->print("  ");
	for(i := 0; i < len data; i++){
		sys->print("%02x", int data[i]);
		if((i + 1) % 16 == 0 && i + 1 < len data)
			sys->print("\n  ");
		else if((i + 1) % 4 == 0)
			sys->print(" ");
	}
	sys->print("\n");
}
