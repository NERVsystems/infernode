implement Todo9p;

#
# todo9p - Task tracking 9P file server
#
# A native Inferno 9P file server for persistent task tracking.
#
# Filesystem structure:
#   /n/todo/
#   ├── new          # (w) write title to create todo, returns ID
#   ├── list         # (r) read all todos (tab-separated: id, status, content)
#   └── <id>/        # per-todo directory
#       ├── content  # (rw) todo description
#       ├── status   # (rw) "pending" | "in_progress" | "completed"
#       └── ctl      # (w) "delete" command
#
# Persistence: /n/local/$HOME/.agent/todos
#
# Usage:
#   mount {todo9p} /n/todo
#   echo "Fix the bug" > /n/todo/new
#   cat /n/todo/list
#   echo "in_progress" > /n/todo/1/status
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

include "bufio.m";
	bufio: Bufio;
	Iobuf: import bufio;

include "string.m";
	str: String;

include "env.m";
	env: Env;

# Todo record
Todo: adt {
	id:       int;       # unique ID
	status:   string;    # "pending", "in_progress", "completed"
	content:  string;    # description
	deleted:  int;       # 1 if deleted
	vers:     int;       # version for qid
};

# Todo database
TodoDB: adt {
	todos:   array of ref Todo;
	nextid:  int;
	dirty:   int;
	vers:    int;

	findbyid: fn(db: self ref TodoDB, id: int): ref Todo;
	add:      fn(db: self ref TodoDB, content: string): ref Todo;
	sync:     fn(db: self ref TodoDB): int;
};

Todo9p: module {
	init: fn(nil: ref Draw->Context, nil: list of string);
};

# Qid types
Qroot, Qnew, Qlist, Qtododir, Qcontent, Qstatus, Qctl: con iota;

stderr: ref Sys->FD;
db: ref TodoDB;
user: string;
persistfile: string;
Eremoved: con "todo removed";

usage()
{
	sys->fprint(stderr, "Usage: todo9p [-D] [mountpoint]\n");
	raise "fail:usage";
}

nomod(s: string)
{
	sys->fprint(stderr, "todo9p: can't load %s: %r\n", s);
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

	bufio = load Bufio Bufio->PATH;
	if(bufio == nil)
		nomod(Bufio->PATH);

	str = load String String->PATH;
	if(str == nil)
		nomod(String->PATH);

	env = load Env Env->PATH;
	if(env == nil)
		nomod(Env->PATH);

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

	mountpt := "/n/todo";
	if(args != nil)
		mountpt = hd args;

	# Determine persistence file location
	# Try /n/local first, fall back to /tmp if not mounted
	home := env->getenv("HOME");
	if(home == nil)
		home = "";

	# Try to find a writable base directory
	# Check /n/local first, then fall back to /usr/inferno
	basedir := "/usr/inferno";  # default fallback
	testfile := "/n/local/.todo9p_test";
	fd := sys->create(testfile, Sys->OWRITE, 8r644);
	if(fd != nil) {
		basedir = "/n/local";
		fd = nil;
		sys->remove(testfile);
	}

	agentdir := basedir + home + "/.agent";
	persistfile = agentdir + "/todos";

	# Ensure directory exists
	ensuredir(agentdir);

	# Load database from persistence file
	db = loaddb();

	sys->pctl(Sys->FORKFD, nil);

	user = rf("/dev/user");
	if(user == nil)
		user = "inferno";

	fds := array[2] of ref Sys->FD;
	if(sys->pipe(fds) < 0) {
		sys->fprint(stderr, "todo9p: can't create pipe: %r\n");
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
		sys->fprint(stderr, "todo9p: mount failed: %r\n");
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
		sys->fprint(stderr, "todo9p: cannot create directory %s: %r\n", path);
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

# Load database from persistence file
loaddb(): ref TodoDB
{
	newdb := ref TodoDB(array[0] of ref Todo, 1, 0, 0);

	f := bufio->open(persistfile, Sys->OREAD);
	if(f == nil)
		return newdb;

	maxid := 0;
	todos: list of ref Todo;

	while((line := f.gets('\n')) != nil) {
		# Skip comments and empty lines
		if(len line == 0 || line[0] == '#' || line[0] == '\n')
			continue;

		# Remove trailing newline
		if(line[len line - 1] == '\n')
			line = line[0:len line - 1];

		# Parse: id\tstatus\tcontent
		(n, fields) := sys->tokenize(line, "\t");
		if(n < 3)
			continue;

		id := int hd fields;
		fields = tl fields;
		status := hd fields;
		fields = tl fields;
		# Join remaining fields as content (in case content has tabs)
		content := "";
		for(; fields != nil; fields = tl fields) {
			if(content != "")
				content += "\t";
			content += hd fields;
		}

		todo := ref Todo(id, status, content, 0, 0);
		todos = todo :: todos;

		if(id > maxid)
			maxid = id;
	}

	# Reverse and store
	count := 0;
	for(lst := todos; lst != nil; lst = tl lst)
		count++;

	newdb.todos = array[count] of ref Todo;
	i := count - 1;
	for(lst = todos; lst != nil; lst = tl lst) {
		newdb.todos[i] = hd lst;
		i--;
	}

	newdb.nextid = maxid + 1;
	return newdb;
}

# Find todo by ID
TodoDB.findbyid(d: self ref TodoDB, id: int): ref Todo
{
	for(i := 0; i < len d.todos; i++) {
		t := d.todos[i];
		if(t != nil && t.id == id && !t.deleted)
			return t;
	}
	return nil;
}

# Add new todo
TodoDB.add(d: self ref TodoDB, content: string): ref Todo
{
	todo := ref Todo(d.nextid, "pending", content, 0, 0);
	d.nextid++;

	# Extend array
	n := len d.todos;
	na := array[n + 1] of ref Todo;
	na[0:] = d.todos;
	na[n] = todo;
	d.todos = na;

	d.dirty++;
	d.vers++;
	return todo;
}

# Sync database to disk
TodoDB.sync(d: self ref TodoDB): int
{
	if(!d.dirty)
		return 0;

	f := bufio->create(persistfile, Sys->OWRITE, 8r644);
	if(f == nil) {
		sys->fprint(stderr, "todo9p: cannot write %s: %r\n", persistfile);
		return -1;
	}

	f.puts("# id\tstatus\tcontent\n");

	for(i := 0; i < len d.todos; i++) {
		t := d.todos[i];
		if(t != nil && !t.deleted) {
			f.puts(sys->sprint("%d\t%s\t%s\n", t.id, t.status, t.content));
		}
	}

	if(f.flush() < 0)
		return -1;

	d.dirty = 0;
	return 0;
}

# Generate list of all todos
genlist(): string
{
	result := "";
	for(i := 0; i < len db.todos; i++) {
		t := db.todos[i];
		if(t != nil && !t.deleted) {
			result += sys->sprint("%d\t%s\t%s\n", t.id, t.status, t.content);
		}
	}
	return result;
}

# Count active todos (for directory readdir)
countactive(): int
{
	n := 0;
	for(i := 0; i < len db.todos; i++) {
		t := db.todos[i];
		if(t != nil && !t.deleted)
			n++;
	}
	return n;
}

# Get nth active todo
getnth(n: int): ref Todo
{
	for(i := 0; i < len db.todos; i++) {
		t := db.todos[i];
		if(t != nil && !t.deleted) {
			if(n == 0)
				return t;
			n--;
		}
	}
	return nil;
}

serveloop(tchan: chan of ref Tmsg, srv: ref Styxserver, pidc: chan of int, navops: chan of ref Navop)
{
	pidc <-= sys->pctl(Sys->FORKNS|Sys->NEWFD, 1::2::srv.fd.fd::nil);

Serve:
	while((gm := <-tchan) != nil) {
		pick m := gm {
		Readerror =>
			sys->fprint(stderr, "todo9p: fatal read error: %s\n", m.error);
			break Serve;

		Open =>
			c := srv.getfid(m.fid);
			if(c == nil) {
				srv.open(m);
				break;
			}

			# Handle opening 'new' file - this allocates a new todo
			qtype := TYPE(c.path);
			if(qtype == Qnew) {
				mode := styxservers->openmode(m.mode);
				if(mode < 0) {
					srv.reply(ref Rmsg.Error(m.tag, Ebadarg));
					break;
				}
				qid := Qid(c.path, 0, Sys->QTFILE);
				c.open(mode, qid);
				srv.reply(ref Rmsg.Open(m.tag, qid, srv.iounit()));
				break;
			}
			srv.open(m);

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

			qtype := TYPE(c.path);
			todoid := TODOID(c.path);

			case qtype {
			Qlist =>
				data := array of byte genlist();
				srv.reply(styxservers->readbytes(m, data));

			Qcontent =>
				todo := db.findbyid(todoid);
				if(todo == nil) {
					srv.reply(ref Rmsg.Error(m.tag, Eremoved));
					break;
				}
				data := array of byte todo.content;
				srv.reply(styxservers->readbytes(m, data));

			Qstatus =>
				todo := db.findbyid(todoid);
				if(todo == nil) {
					srv.reply(ref Rmsg.Error(m.tag, Eremoved));
					break;
				}
				data := array of byte todo.status;
				srv.reply(styxservers->readbytes(m, data));

			Qctl =>
				srv.reply(styxservers->readbytes(m, array[0] of byte));

			* =>
				srv.reply(ref Rmsg.Error(m.tag, Eperm));
			}

		Write =>
			(c, merr) := srv.canwrite(m);
			if(c == nil) {
				srv.reply(ref Rmsg.Error(m.tag, merr));
				break;
			}

			qtype := TYPE(c.path);
			todoid := TODOID(c.path);
			data := string m.data;

			# Strip trailing newline
			if(len data > 0 && data[len data - 1] == '\n')
				data = data[0:len data - 1];

			case qtype {
			Qnew =>
				# Create new todo with content
				todo := db.add(data);
				db.sync();
				# Return the new ID
				reply := sys->sprint("%d\n", todo.id);
				srv.reply(ref Rmsg.Write(m.tag, len m.data));

			Qcontent =>
				todo := db.findbyid(todoid);
				if(todo == nil) {
					srv.reply(ref Rmsg.Error(m.tag, Eremoved));
					break;
				}
				todo.content = data;
				todo.vers++;
				db.dirty++;
				db.sync();
				srv.reply(ref Rmsg.Write(m.tag, len m.data));

			Qstatus =>
				todo := db.findbyid(todoid);
				if(todo == nil) {
					srv.reply(ref Rmsg.Error(m.tag, Eremoved));
					break;
				}
				# Validate status
				if(data != "pending" && data != "in_progress" && data != "completed") {
					srv.reply(ref Rmsg.Error(m.tag, "invalid status: must be pending, in_progress, or completed"));
					break;
				}
				todo.status = data;
				todo.vers++;
				db.dirty++;
				db.sync();
				srv.reply(ref Rmsg.Write(m.tag, len m.data));

			Qctl =>
				todo := db.findbyid(todoid);
				if(todo == nil) {
					srv.reply(ref Rmsg.Error(m.tag, Eremoved));
					break;
				}
				if(data == "delete") {
					todo.deleted = 1;
					db.dirty++;
					db.vers++;
					db.sync();
					srv.reply(ref Rmsg.Write(m.tag, len m.data));
				} else {
					srv.reply(ref Rmsg.Error(m.tag, "unknown ctl command"));
				}

			* =>
				srv.reply(ref Rmsg.Error(m.tag, Eperm));
			}

		Clunk =>
			srv.clunk(m);

		Remove =>
			# Don't allow remove via filesystem (use ctl delete instead)
			srv.remove(m);

		* =>
			srv.default(gm);
		}
	}
	navops <-= nil;  # shut down navigator
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
	qtype := TYPE(p);
	todoid := TODOID(p);

	case qtype {
	Qroot =>
		return (dir(Qid(p, db.vers, Sys->QTDIR), "/", big 0, 8r755), nil);

	Qnew =>
		return (dir(Qid(p, 0, Sys->QTFILE), "new", big 0, 8r222), nil);

	Qlist =>
		listdata := genlist();
		return (dir(Qid(p, db.vers, Sys->QTFILE), "list", big len listdata, 8r444), nil);

	Qtododir =>
		todo := db.findbyid(todoid);
		if(todo == nil)
			return (nil, Enotfound);
		return (dir(Qid(p, todo.vers, Sys->QTDIR), sys->sprint("%d", todoid), big 0, 8r755), nil);

	Qcontent =>
		todo := db.findbyid(todoid);
		if(todo == nil)
			return (nil, Enotfound);
		return (dir(Qid(p, todo.vers, Sys->QTFILE), "content", big len todo.content, 8r644), nil);

	Qstatus =>
		todo := db.findbyid(todoid);
		if(todo == nil)
			return (nil, Enotfound);
		return (dir(Qid(p, todo.vers, Sys->QTFILE), "status", big len todo.status, 8r644), nil);

	Qctl =>
		todo := db.findbyid(todoid);
		if(todo == nil)
			return (nil, Enotfound);
		return (dir(Qid(p, 0, Sys->QTFILE), "ctl", big 0, 8r222), nil);
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
			qtype := TYPE(n.path);
			todoid := TODOID(n.path);

			case qtype {
			Qroot =>
				case n.name {
				".." =>
					;  # stay at root
				"new" =>
					n.path = QPATH(0, Qnew);
				"list" =>
					n.path = QPATH(0, Qlist);
				* =>
					# Try to parse as todo ID
					id := int n.name;
					if(id <= 0 || db.findbyid(id) == nil) {
						n.reply <-= (nil, Enotfound);
						continue;
					}
					n.path = QPATH(id, Qtododir);
				}
				n.reply <-= dirgen(n.path);

			Qtododir =>
				case n.name {
				".." =>
					n.path = QPATH(0, Qroot);
				"content" =>
					n.path = QPATH(todoid, Qcontent);
				"status" =>
					n.path = QPATH(todoid, Qstatus);
				"ctl" =>
					n.path = QPATH(todoid, Qctl);
				* =>
					n.reply <-= (nil, Enotfound);
					continue;
				}
				n.reply <-= dirgen(n.path);

			* =>
				n.reply <-= (nil, "not a directory");
			}

		Readdir =>
			qtype := TYPE(m.path);
			todoid := TODOID(m.path);

			case qtype {
			Qroot =>
				# Root directory contains: new, list, and todo directories
				i := n.offset;
				if(i == 0)
					n.reply <-= dirgen(QPATH(0, Qnew));
				if(i <= 1 && n.count > 0) {
					n.reply <-= dirgen(QPATH(0, Qlist));
					n.count--;
				}
				# Skip first 2 (new, list) for todo dirs
				todoidx := 0;
				if(i > 1)
					todoidx = i - 2;
				else if(i <= 1)
					todoidx = 0;
				for(; n.count > 0; n.count--) {
					todo := getnth(todoidx);
					if(todo == nil)
						break;
					if(i <= 2 + todoidx)
						n.reply <-= dirgen(QPATH(todo.id, Qtododir));
					todoidx++;
				}
				n.reply <-= (nil, nil);

			Qtododir =>
				# Todo directory contains: content, status, ctl
				i := n.offset;
				if(i == 0 && n.count > 0) {
					n.reply <-= dirgen(QPATH(todoid, Qcontent));
					n.count--;
				}
				if(i <= 1 && n.count > 0) {
					n.reply <-= dirgen(QPATH(todoid, Qstatus));
					n.count--;
				}
				if(i <= 2 && n.count > 0) {
					n.reply <-= dirgen(QPATH(todoid, Qctl));
					n.count--;
				}
				n.reply <-= (nil, nil);

			* =>
				n.reply <-= (nil, "not a directory");
			}
		}
	}
}

# Encode path: upper 24 bits = todo ID, lower 8 bits = qtype
QPATH(todoid, qtype: int): big
{
	return big ((todoid << 8) | qtype);
}

TYPE(path: big): int
{
	return int path & 16rFF;
}

TODOID(path: big): int
{
	return (int path >> 8) & 16rFFFFFF;
}
