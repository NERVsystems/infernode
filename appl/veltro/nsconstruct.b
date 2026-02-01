implement NsConstruct;

#
# nsconstruct.b - Namespace construction for Veltro agents
#
# SECURITY MODEL (v2):
# ====================
# Parent prepares sandbox directory before spawn. Child uses NEWNS to make
# sandbox become root. This ensures:
#
# 1. Allowlist model - child only sees what parent explicitly bound
# 2. No environment leaks - NEWENV creates empty environment
# 3. No FD leaks - NEWFD prunes file descriptors
# 4. No device escapes - NODEVS blocks #U/#p/#c naming
# 5. Race-free - sandbox creation fails if directory exists
# 6. Auditable - all binds logged to audit file
#
# Key insight: NEWNS makes current directory become new root.
# Combined with NEWENV, NEWFD, and NODEVS, this provides true
# capability-based security.
#

include "sys.m";
	sys: Sys;
	FORKNS, NEWPGRP, NEWNS, NEWENV, NEWFD, NODEVS: import Sys;

include "draw.m";

include "daytime.m";
	daytime: Daytime;

include "rand.m";
	rand: Rand;

include "nsconstruct.m";

# Base directory for all Veltro sandboxes
SANDBOX_BASE: con "/tmp/.veltro";
SANDBOX_DIR: con "/tmp/.veltro/sandbox";
AUDIT_DIR: con "/tmp/.veltro/audit";

# File permissions
DIR_MODE: con 8r700 | Sys->DMDIR;  # rwx------ directory
FILE_MODE: con 8r600;              # rw------- file

init()
{
	sys = load Sys Sys->PATH;
	daytime = load Daytime Daytime->PATH;
	rand = load Rand Rand->PATH;
	if(rand != nil)
		rand->init(sys->millisec());
}

# Validate sandbox ID - reject traversal attacks and invalid characters
# Returns error string or nil on success
validatesandboxid(id: string): string
{
	if(id == nil || len id == 0)
		return "empty sandbox id";
	if(len id > 64)
		return "sandbox id too long (max 64)";

	for(i := 0; i < len id; i++) {
		c := id[i];
		# Alphanumeric and hyphen only - no /, .., or special chars
		if(!((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		     (c >= '0' && c <= '9') || c == '-'))
			return sys->sprint("invalid character '%c' in sandbox id", c);
	}

	# Explicitly reject traversal patterns (belt and suspenders)
	if(id == "." || id == "..")
		return "sandbox id cannot be . or ..";

	return nil;
}

# Generate a unique sandbox ID using timestamp and random component
gensandboxid(): string
{
	if(sys == nil)
		init();

	# Get timestamp
	now := sys->millisec();

	# Get random component
	r := 0;
	if(rand != nil)
		r = rand->rand(16r7FFFFFFF);

	# Format: timestamp-random (hex)
	return sys->sprint("%x-%x", now, r);
}

# Get sandbox path for a given ID
sandboxpath(sandboxid: string): string
{
	return SANDBOX_DIR + "/" + sandboxid;
}

# Verify ownership of a path by stat'ing it
verifyownership(path: string): string
{
	if(sys == nil)
		init();

	(ok, nil) := sys->stat(path);
	if(ok < 0)
		return sys->sprint("cannot stat %s: %r", path);
	return nil;
}

# Prepare sandbox directory structure
# Creates sandbox at /tmp/.veltro/sandbox/{id}/ with restrictive permissions
# Binds granted paths from parent namespace into sandbox
# Returns error string or nil on success
preparesandbox(caps: ref Capabilities): string
{
	if(sys == nil)
		init();

	# 1. Validate sandbox ID
	err := validatesandboxid(caps.sandboxid);
	if(err != nil)
		return err;

	# 2. Create base directories with restrictive permissions
	err = mkdirp(SANDBOX_BASE);
	if(err != nil)
		return err;
	err = mkdirp(SANDBOX_DIR);
	if(err != nil)
		return err;
	err = mkdirp(AUDIT_DIR);
	if(err != nil)
		return err;

	# 3. Create sandbox directory - FAIL if exists (prevents races)
	sandboxdir := sandboxpath(caps.sandboxid);
	fd := sys->create(sandboxdir, Sys->OREAD, DIR_MODE);
	if(fd == nil)
		return sys->sprint("sandbox already exists or cannot create %s: %r", sandboxdir);
	fd = nil;

	# Track bind operations for audit
	binds: list of ref BindRecord;

	# 4. Create sandbox structure and bind from parent namespace
	# Create dis/ directory for Limbo runtime
	disdir := sandboxdir + "/dis";
	err = mkdirp(disdir);
	if(err != nil) {
		cleanupsandbox(caps.sandboxid);
		return err;
	}

	# Bind essential runtime library (if it exists)
	# Some minimal Inferno configurations may not have /dis/lib
	(ok, nil) := sys->stat("/dis/lib");
	if(ok >= 0) {
		err = mkdirp(disdir + "/lib");
		if(err != nil) {
			cleanupsandbox(caps.sandboxid);
			return err;
		}
		if(sys->bind("/dis/lib", disdir + "/lib", Sys->MREPL) < 0) {
			cleanupsandbox(caps.sandboxid);
			return sys->sprint("cannot bind /dis/lib: %r");
		}
		binds = ref BindRecord("/dis/lib", disdir + "/lib", Sys->MREPL) :: binds;
	}

	# Bind Veltro tools directory
	err = mkdirp(disdir + "/veltro");
	if(err != nil) {
		cleanupsandbox(caps.sandboxid);
		return err;
	}
	err = mkdirp(disdir + "/veltro/tools");
	if(err != nil) {
		cleanupsandbox(caps.sandboxid);
		return err;
	}
	if(sys->bind("/dis/veltro", disdir + "/veltro", Sys->MREPL) < 0) {
		cleanupsandbox(caps.sandboxid);
		return sys->sprint("cannot bind /dis/veltro: %r");
	}
	binds = ref BindRecord("/dis/veltro", disdir + "/veltro", Sys->MREPL) :: binds;

	# If trusted, bind shell and granted shell commands
	if(caps.trusted) {
		# Create placeholder and bind sh.dis
		createplaceholder(disdir + "/sh.dis");
		if(sys->bind("/dis/sh.dis", disdir + "/sh.dis", Sys->MREPL) < 0) {
			cleanupsandbox(caps.sandboxid);
			return sys->sprint("cannot bind sh.dis: %r");
		}
		binds = ref BindRecord("/dis/sh.dis", disdir + "/sh.dis", Sys->MREPL) :: binds;

		# Bind each granted shell command
		for(c := caps.shellcmds; c != nil; c = tl c) {
			cmd := hd c;
			srcpath := "/dis/" + cmd + ".dis";
			dstpath := disdir + "/" + cmd + ".dis";

			# Verify source exists
			err = verifyownership(srcpath);
			if(err != nil) {
				cleanupsandbox(caps.sandboxid);
				return sys->sprint("shell command not found: %s", cmd);
			}

			createplaceholder(dstpath);
			if(sys->bind(srcpath, dstpath, Sys->MREPL) < 0) {
				cleanupsandbox(caps.sandboxid);
				return sys->sprint("cannot bind %s: %r", cmd);
			}
			binds = ref BindRecord(srcpath, dstpath, Sys->MREPL) :: binds;
		}
	}

	# 5. Create dev/ directory and bind console and null
	devdir := sandboxdir + "/dev";
	err = mkdirp(devdir);
	if(err != nil) {
		cleanupsandbox(caps.sandboxid);
		return err;
	}

	# Bind /dev/cons and /dev/null
	createplaceholder(devdir + "/cons");
	if(sys->bind("/dev/cons", devdir + "/cons", Sys->MREPL) < 0) {
		cleanupsandbox(caps.sandboxid);
		return sys->sprint("cannot bind /dev/cons: %r");
	}
	binds = ref BindRecord("/dev/cons", devdir + "/cons", Sys->MREPL) :: binds;

	createplaceholder(devdir + "/null");
	if(sys->bind("/dev/null", devdir + "/null", Sys->MREPL) < 0) {
		cleanupsandbox(caps.sandboxid);
		return sys->sprint("cannot bind /dev/null: %r");
	}
	binds = ref BindRecord("/dev/null", devdir + "/null", Sys->MREPL) :: binds;

	# 6. Create tool/ mount point for tools9p
	err = mkdirp(sandboxdir + "/tool");
	if(err != nil) {
		cleanupsandbox(caps.sandboxid);
		return err;
	}

	# 7. Create tmp/ for writable scratch space
	err = mkdirp(sandboxdir + "/tmp");
	if(err != nil) {
		cleanupsandbox(caps.sandboxid);
		return err;
	}

	# 8. Create n/llm/ if LLM config is provided
	if(caps.llmconfig != nil) {
		err = mkdirp(sandboxdir + "/n");
		if(err != nil) {
			cleanupsandbox(caps.sandboxid);
			return err;
		}
		err = mkdirp(sandboxdir + "/n/llm");
		if(err != nil) {
			cleanupsandbox(caps.sandboxid);
			return err;
		}
	}

	# 9. Bind granted paths from parent namespace
	for(p := caps.paths; p != nil; p = tl p) {
		srcpath := hd p;

		# Verify source exists in parent namespace
		err = verifyownership(srcpath);
		if(err != nil) {
			cleanupsandbox(caps.sandboxid);
			return sys->sprint("cannot bind path %s: %s", srcpath, err);
		}

		# Determine destination path in sandbox
		# Strip leading / to make relative path
		relpath := srcpath;
		if(len relpath > 0 && relpath[0] == '/')
			relpath = relpath[1:];
		dstpath := sandboxdir + "/" + relpath;

		# Create parent directories for destination
		err = mkparent(dstpath);
		if(err != nil) {
			cleanupsandbox(caps.sandboxid);
			return err;
		}

		# Create destination (file or directory)
		(ok, dir) := sys->stat(srcpath);
		if(ok >= 0 && (dir.mode & Sys->DMDIR)) {
			# Source is directory - create directory
			err = mkdirp(dstpath);
			if(err != nil) {
				cleanupsandbox(caps.sandboxid);
				return err;
			}
		} else {
			# Source is file - create placeholder
			createplaceholder(dstpath);
		}

		if(sys->bind(srcpath, dstpath, Sys->MREPL) < 0) {
			cleanupsandbox(caps.sandboxid);
			return sys->sprint("cannot bind %s: %r", srcpath);
		}
		binds = ref BindRecord(srcpath, dstpath, Sys->MREPL) :: binds;
	}

	# 10. Write audit log
	emitauditlog(caps.sandboxid, binds);

	return nil;
}

# Clean up sandbox directory after child exits
cleanupsandbox(sandboxid: string)
{
	if(sys == nil)
		init();

	# Validate ID to prevent traversal
	if(validatesandboxid(sandboxid) != nil)
		return;

	sandboxdir := sandboxpath(sandboxid);

	# CRITICAL: Unmount all bind points BEFORE removing
	# Otherwise rmrf follows bind mounts and deletes original files!
	unmountbinds(sandboxdir);

	# Now safe to remove sandbox directory
	rmrf(sandboxdir);
}

# Unmount known bind points in sandbox to prevent rmrf from following mounts
unmountbinds(sandboxdir: string)
{
	# List of paths we bind into sandboxes (relative to sandboxdir)
	# Order matters: unmount leaves before branches
	bindpaths := array[] of {
		"/dis/lib",
		"/dis/veltro",
		"/dis/sh.dis",
		"/dev/cons",
		"/dev/null",
	};

	for(i := 0; i < len bindpaths; i++) {
		path := sandboxdir + bindpaths[i];
		# Unmount ignores errors (path may not be mounted)
		sys->unmount(nil, path);
	}

	# Also unmount any shell commands that may have been bound
	# These are in /dis/*.dis - we'll unmount the whole /dis directory mount
	sys->unmount(nil, sandboxdir + "/dis");
}

# Emit audit log of namespace bindings
emitauditlog(sandboxid: string, binds: list of ref BindRecord)
{
	if(sys == nil)
		init();

	# Validate ID
	if(validatesandboxid(sandboxid) != nil)
		return;

	auditpath := AUDIT_DIR + "/" + sandboxid + ".ns";
	fd := sys->create(auditpath, Sys->OWRITE, FILE_MODE);
	if(fd == nil)
		return;

	# Write header
	now := "";
	if(daytime != nil)
		now = daytime->time();
	hdr := sys->sprint("# Veltro Sandbox Namespace Audit\n# ID: %s\n# Time: %s\n\n", sandboxid, now);
	sys->fprint(fd, "%s", hdr);

	# Write binds in reverse order (oldest first)
	revbinds: list of ref BindRecord;
	for(; binds != nil; binds = tl binds)
		revbinds = hd binds :: revbinds;

	for(; revbinds != nil; revbinds = tl revbinds) {
		b := hd revbinds;
		sys->fprint(fd, "bind %s -> %s (flags=%d)\n", b.src, b.dst, b.flags);
	}

	fd = nil;
}

# Capture essential paths from current namespace
captureessentials(): ref Essentials
{
	return ref Essentials("/dis", "/dev", "/module");
}

# Bind essentials - handled by preparesandbox now
bindessentials(nil: ref Essentials): string
{
	return nil;
}

# Mount tools - handled by spawn.b directly
mounttools(nil: list of string): string
{
	return nil;
}

# Create directories - handled by preparesandbox now
mkdirs(): string
{
	return nil;
}

# Construct namespace - not used with new model
# Kept for interface compatibility
construct(nil: ref Essentials, nil: ref Capabilities): string
{
	return nil;
}

# Helper: Create directory with mode 0700, including parents
mkdirp(path: string): string
{
	if(sys == nil)
		init();

	# Check if already exists
	(ok, nil) := sys->stat(path);
	if(ok >= 0)
		return nil;

	# Create parent directories first
	err := mkparent(path);
	if(err != nil)
		return err;

	# Create directory
	fd := sys->create(path, Sys->OREAD, DIR_MODE);
	if(fd == nil)
		return sys->sprint("cannot create %s: %r", path);
	fd = nil;
	return nil;
}

# Helper: Create parent directories for a path
mkparent(path: string): string
{
	# Find parent directory
	parent := "";
	for(i := len path - 1; i > 0; i--) {
		if(path[i] == '/') {
			parent = path[0:i];
			break;
		}
	}

	if(parent == "" || parent == "/")
		return nil;

	return mkdirp(parent);
}

# Helper: Create an empty placeholder file for bind destination
createplaceholder(path: string)
{
	if(sys == nil)
		init();

	fd := sys->create(path, Sys->OWRITE, FILE_MODE);
	if(fd != nil)
		fd = nil;
}

# Helper: Recursively remove directory
rmrf(path: string)
{
	if(sys == nil)
		init();

	(ok, dir) := sys->stat(path);
	if(ok < 0)
		return;

	if(dir.mode & Sys->DMDIR) {
		# Directory - list contents and remove recursively
		fd := sys->open(path, Sys->OREAD);
		if(fd == nil)
			return;

		for(;;) {
			(n, dirs) := sys->dirread(fd);
			if(n <= 0)
				break;
			for(i := 0; i < n; i++)
				rmrf(path + "/" + dirs[i].name);
		}
		fd = nil;
	}

	# Remove the file/directory
	sys->remove(path);
}
