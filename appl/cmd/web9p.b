implement Web9p;

#
# web9p - HTTP as Filesystem for AI Agents
#
# A 9P file server that exposes HTTP operations as files.
# Modeled after ftpfs but using the query-file pattern.
#
# NOTE: This API is PROVISIONAL and may change based on agent testing.
# See appl/xenith/IDEAS.md for design considerations including:
# - Single-file I/O vs separate input/output files
# - POST body format (JSON vs form-encoded vs raw)
# - State model and concurrent request handling
#
# Filesystem structure:
#   /n/web/
#   ├── url           # (w) write URL to fetch
#   ├── method        # (rw) "GET" or "POST" (default: GET)
#   ├── body          # (rw) POST body
#   ├── result        # (r) response content
#   ├── status        # (r) "ok" or "error: message"
#   └── help          # (r) usage documentation
#
# Usage:
#   web9p /n/web
#   echo 'https://example.com' > /n/web/url
#   cat /n/web/result
#
# POST example:
#   echo 'https://api.example.com/data' > /n/web/url
#   echo 'POST' > /n/web/method
#   echo '{"key": "value"}' > /n/web/body
#   cat /n/web/result
#

include "sys.m";
	sys: Sys;
	Qid: import Sys;

include "draw.m";

include "arg.m";

include "styx.m";
	styx: Styx;
	Tmsg, Rmsg: import styx;

include "styxservers.m";
	styxservers: Styxservers;
	Fid, Styxserver, Navigator, Navop: import styxservers;
	Enotfound, Eperm, Ebadarg: import styxservers;

include "web.m";
	web: Web;

Web9p: module {
	init: fn(nil: ref Draw->Context, nil: list of string);
};

# Qid types for synthetic files
Qroot, Qurl, Qmethod, Qbody, Qresult, Qstatus, Qhelp: con iota;

# Connection state
State: adt {
	url:     string;          # URL to fetch
	method:  string;          # "GET" or "POST"
	body:    string;          # POST body
	result:  array of byte;   # Response content
	status:  string;          # "ok" or "error: ..."
	fetched: int;             # Has a fetch been performed?
	vers:    int;             # Version for qid
};

stderr: ref Sys->FD;
state: ref State;
user: string;

HELP_TEXT := "web9p - HTTP as Filesystem\n\n" +
	"Usage:\n" +
	"  Fetch URL:    echo 'url' > /n/web/url && cat /n/web/result\n" +
	"  POST request: echo 'url' > /n/web/url\n" +
	"                echo 'POST' > /n/web/method\n" +
	"                echo 'body' > /n/web/body\n" +
	"                cat /n/web/result\n\n" +
	"Files:\n" +
	"  url     (w)  Write URL to fetch (triggers GET if method=GET)\n" +
	"  method  (rw) GET or POST (default: GET)\n" +
	"  body    (rw) POST body content\n" +
	"  result  (r)  Response content\n" +
	"  status  (r)  \"ok\" or \"error: message\"\n" +
	"  help    (r)  This help text\n\n" +
	"Supports HTTP/HTTPS via webget service.\n";

usage()
{
	sys->fprint(stderr, "Usage: web9p [-D] [mountpoint]\n");
	raise "fail:usage";
}

nomod(s: string)
{
	sys->fprint(stderr, "web9p: can't load %s: %r\n", s);
	raise "fail:load";
}

init(nil: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	sys->pctl(Sys->FORKFD|Sys->NEWPGRP, nil);
	stderr = sys->fildes(2);

	styx = load Styx Styx->PATH;
	if(styx == nil)
		nomod(Styx->PATH);
	styx->init();

	styxservers = load Styxservers Styxservers->PATH;
	if(styxservers == nil)
		nomod(Styxservers->PATH);
	styxservers->init(styx);

	web = load Web Web->PATH;
	if(web == nil)
		nomod(Web->PATH);
	web->init0();

	arg := load Arg Arg->PATH;
	if(arg == nil)
		nomod(Arg->PATH);
	arg->init(args);

	while((o := arg->opt()) != 0)
		case o {
		'D' =>	styxservers->traceset(1);
		* =>	usage();
		}
	args = arg->argv();
	arg = nil;

	mountpt := "/n/web";
	if(args != nil)
		mountpt = hd args;

	# Initialize state
	state = ref State("", "GET", "", nil, "", 0, 0);

	sys->pctl(Sys->FORKFD, nil);

	user = rf("/dev/user");
	if(user == nil)
		user = "inferno";

	fds := array[2] of ref Sys->FD;
	if(sys->pipe(fds) < 0) {
		sys->fprint(stderr, "web9p: can't create pipe: %r\n");
		raise "fail:pipe";
	}

	navops := chan of ref Navop;
	spawn navigator(navops);

	(tchan, srv) := Styxserver.new(fds[0], Navigator.new(navops), big Qroot);
	fds[0] = nil;

	pidc := chan of int;
	spawn serveloop(tchan, srv, pidc, navops);
	<-pidc;

	# Ensure mount point exists
	ensuredir(mountpt);

	if(sys->mount(fds[1], nil, mountpt, Sys->MREPL|Sys->MCREATE, nil) < 0) {
		sys->fprint(stderr, "web9p: mount failed: %r\n");
		raise "fail:mount";
	}
}

# Ensure a directory exists, creating it if needed
ensuredir(path: string)
{
	fd := sys->open(path, Sys->OREAD);
	if(fd != nil)
		return;

	# Try to create parent first
	for(i := len path - 1; i > 0; i--) {
		if(path[i] == '/') {
			ensuredir(path[0:i]);
			break;
		}
	}

	# Create this directory
	fd = sys->create(path, Sys->OREAD, Sys->DMDIR | 8r755);
	if(fd == nil)
		sys->fprint(stderr, "web9p: cannot create directory %s: %r\n", path);
}

rf(f: string): string
{
	fd := sys->open(f, Sys->OREAD);
	if(fd == nil)
		return nil;
	b := array[Sys->NAMEMAX] of byte;
	n := sys->read(fd, b, len b);
	if(n < 0)
		return nil;
	return string b[0:n];
}

# Perform HTTP request
dofetch()
{
	if(state.url == "") {
		state.status = "error: no URL set";
		state.result = nil;
		state.fetched = 1;
		state.vers++;
		return;
	}

	# Perform the request
	data: array of byte;
	if(state.method == "POST") {
		data = web->posturl(state.url, state.body);
	} else {
		data = web->readurl(state.url);
	}

	if(data == nil) {
		state.status = "error: request failed";
		state.result = nil;
	} else {
		state.status = "ok";
		state.result = data;
	}
	state.fetched = 1;
	state.vers++;
}

serveloop(tchan: chan of ref Tmsg, srv: ref Styxserver, pidc: chan of int, navops: chan of ref Navop)
{
	pidc <-= sys->pctl(Sys->FORKNS|Sys->NEWFD, 1::2::srv.fd.fd::nil);

Serve:
	while((gm := <-tchan) != nil) {
		pick m := gm {
		Readerror =>
			sys->fprint(stderr, "web9p: fatal read error: %s\n", m.error);
			break Serve;

		Open =>
			c := srv.getfid(m.fid);
			if(c == nil) {
				srv.open(m);
				break;
			}

			mode := styxservers->openmode(m.mode);
			if(mode < 0) {
				srv.reply(ref Rmsg.Error(m.tag, Ebadarg));
				break;
			}
			qid := Qid(c.path, 0, Sys->QTFILE);
			c.open(mode, qid);
			srv.reply(ref Rmsg.Open(m.tag, qid, srv.iounit()));

		Read =>
			(c, err) := srv.canread(m);
			if(c == nil) {
				srv.reply(ref Rmsg.Error(m.tag, err));
				break;
			}

			if(c.qtype & Sys->QTDIR) {
				srv.read(m);  # navigator handles directory reads
				break;
			}

			qtype := int c.path & 16rFF;

			case qtype {
			Qurl =>
				data := array of byte state.url;
				srv.reply(styxservers->readbytes(m, data));

			Qmethod =>
				data := array of byte state.method;
				srv.reply(styxservers->readbytes(m, data));

			Qbody =>
				data := array of byte state.body;
				srv.reply(styxservers->readbytes(m, data));

			Qresult =>
				# Fetch on first read if not done
				if(!state.fetched)
					dofetch();
				if(state.result == nil)
					srv.reply(styxservers->readbytes(m, array[0] of byte));
				else
					srv.reply(styxservers->readbytes(m, state.result));

			Qstatus =>
				# Fetch if needed
				if(!state.fetched)
					dofetch();
				data := array of byte state.status;
				srv.reply(styxservers->readbytes(m, data));

			Qhelp =>
				data := array of byte HELP_TEXT;
				srv.reply(styxservers->readbytes(m, data));

			* =>
				srv.reply(ref Rmsg.Error(m.tag, Eperm));
			}

		Write =>
			(c, merr) := srv.canwrite(m);
			if(c == nil) {
				srv.reply(ref Rmsg.Error(m.tag, merr));
				break;
			}

			qtype := int c.path & 16rFF;
			data := string m.data;

			# Strip trailing newline
			if(len data > 0 && data[len data - 1] == '\n')
				data = data[0:len data - 1];

			case qtype {
			Qurl =>
				state.url = data;
				state.fetched = 0;  # Reset for new URL
				state.result = nil;
				state.status = "";
				state.vers++;
				srv.reply(ref Rmsg.Write(m.tag, len m.data));

			Qmethod =>
				# Validate method
				udata := toupper(data);
				if(udata != "GET" && udata != "POST") {
					srv.reply(ref Rmsg.Error(m.tag, "invalid method: must be GET or POST"));
					break;
				}
				state.method = udata;
				state.fetched = 0;  # Reset on method change
				state.vers++;
				srv.reply(ref Rmsg.Write(m.tag, len m.data));

			Qbody =>
				state.body = string m.data;  # Keep original data with newlines
				state.fetched = 0;  # Reset on body change
				state.vers++;
				srv.reply(ref Rmsg.Write(m.tag, len m.data));

			Qresult | Qstatus | Qhelp =>
				srv.reply(ref Rmsg.Error(m.tag, Eperm));

			* =>
				srv.reply(ref Rmsg.Error(m.tag, Eperm));
			}

		Clunk =>
			srv.clunk(m);

		Remove =>
			srv.remove(m);

		* =>
			srv.default(gm);
		}
	}
	navops <-= nil;  # shut down navigator
}

# Convert string to uppercase
toupper(s: string): string
{
	b := array of byte s;
	for(i := 0; i < len b; i++) {
		c := int b[i];
		if(c >= 'a' && c <= 'z')
			b[i] = byte (c - 'a' + 'A');
	}
	return string b;
}

dir(qid: Sys->Qid, name: string, length: big, perm: int): ref Sys->Dir
{
	d := ref sys->zerodir;
	d.qid = qid;
	if(qid.qtype & Sys->QTDIR)
		perm |= Sys->DMDIR;
	d.mode = perm;
	d.name = name;
	d.uid = user;
	d.gid = user;
	d.length = length;
	return d;
}

dirgen(p: big): (ref Sys->Dir, string)
{
	qtype := int p & 16rFF;

	case qtype {
	Qroot =>
		return (dir(Qid(p, state.vers, Sys->QTDIR), "/", big 0, 8r755), nil);

	Qurl =>
		return (dir(Qid(p, state.vers, Sys->QTFILE), "url", big len state.url, 8r644), nil);

	Qmethod =>
		return (dir(Qid(p, state.vers, Sys->QTFILE), "method", big len state.method, 8r644), nil);

	Qbody =>
		return (dir(Qid(p, state.vers, Sys->QTFILE), "body", big len state.body, 8r644), nil);

	Qresult =>
		reslen := 0;
		if(state.result != nil)
			reslen = len state.result;
		return (dir(Qid(p, state.vers, Sys->QTFILE), "result", big reslen, 8r444), nil);

	Qstatus =>
		return (dir(Qid(p, state.vers, Sys->QTFILE), "status", big len state.status, 8r444), nil);

	Qhelp =>
		return (dir(Qid(p, 0, Sys->QTFILE), "help", big len HELP_TEXT, 8r444), nil);
	}

	return (nil, Enotfound);
}

navigator(navops: chan of ref Navop)
{
	while((m := <-navops) != nil) {
		pick n := m {
		Stat =>
			n.reply <-= dirgen(n.path);

		Walk =>
			qtype := int n.path & 16rFF;

			case qtype {
			Qroot =>
				case n.name {
				".." =>
					;  # stay at root
				"url" =>
					n.path = big Qurl;
				"method" =>
					n.path = big Qmethod;
				"body" =>
					n.path = big Qbody;
				"result" =>
					n.path = big Qresult;
				"status" =>
					n.path = big Qstatus;
				"help" =>
					n.path = big Qhelp;
				* =>
					n.reply <-= (nil, Enotfound);
					continue;
				}
				n.reply <-= dirgen(n.path);

			* =>
				n.reply <-= (nil, "not a directory");
			}

		Readdir =>
			qtype := int m.path & 16rFF;

			case qtype {
			Qroot =>
				# Root directory contains: url, method, body, result, status, help
				files := array[] of {Qurl, Qmethod, Qbody, Qresult, Qstatus, Qhelp};
				i := n.offset;
				for(; i < len files && n.count > 0; i++) {
					n.reply <-= dirgen(big files[i]);
					n.count--;
				}
				n.reply <-= (nil, nil);

			* =>
				n.reply <-= (nil, "not a directory");
			}
		}
	}
}
