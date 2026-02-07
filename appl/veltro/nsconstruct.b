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

# Thread-safe initialization
inited := 0;

# Maximum age of sandbox before it's considered stale (5 minutes in seconds)
STALE_SANDBOX_AGE: con 5 * 60;

init()
{
	# Quick check - already initialized
	if(inited)
		return;

	# Load modules - these are idempotent, so even if two threads
	# race here, the result is the same (just wasted work)
	sys = load Sys Sys->PATH;
	daytime = load Daytime Daytime->PATH;
	rand = load Rand Rand->PATH;
	if(rand != nil && !inited)
		rand->init(sys->millisec());

	inited = 1;

	# Clean up any stale sandboxes from crashed/interrupted runs
	cleanstalesandboxes();
}

# Remove sandboxes that are older than STALE_SANDBOX_AGE
# This handles cleanup for processes that were killed before cleanup ran
cleanstalesandboxes()
{
	fd := sys->open(SANDBOX_DIR, Sys->OREAD);
	if(fd == nil)
		return;  # No sandbox directory yet

	# Use wall clock time (seconds since epoch) to match gensandboxid()
	now := 0;
	if(daytime != nil)
		now = daytime->now();
	else
		now = sys->millisec() / 1000;  # Fallback

	for(;;) {
		(n, dirs) := sys->dirread(fd);
		if(n <= 0)
			break;
		for(i := 0; i < n; i++) {
			name := dirs[i].name;
			if(name == "." || name == "..")
				continue;

			# Parse sandbox ID to get timestamp
			# Format: timestamp-random (both in hex)
			timestamp := parsesandboxtime(name);
			if(timestamp < 0)
				continue;  # Invalid format, skip

			# Check if sandbox is stale
			age := now - timestamp;
			if(age > STALE_SANDBOX_AGE) {
				# Sandbox is stale, remove it
				cleanupsandbox(name);
			}
		}
	}
	fd = nil;
}

# Parse timestamp from sandbox ID (returns -1 if invalid)
parsesandboxtime(id: string): int
{
	# Find the dash separator
	for(i := 0; i < len id; i++) {
		if(id[i] == '-') {
			# Parse hex timestamp before dash
			timestamp := 0;
			for(j := 0; j < i; j++) {
				c := id[j];
				digit: int;
				if(c >= '0' && c <= '9')
					digit = c - '0';
				else if(c >= 'a' && c <= 'f')
					digit = c - 'a' + 10;
				else if(c >= 'A' && c <= 'F')
					digit = c - 'A' + 10;
				else
					return -1;  # Invalid hex
				timestamp = timestamp * 16 + digit;
			}
			return timestamp;
		}
	}
	return -1;  # No dash found
}

# Copy a file from src to dst
# Used instead of bind() because NEWNS removes binds where source is outside sandbox
copyfile(src, dst: string): string
{
	if(sys == nil)
		init();

	sfd := sys->open(src, Sys->OREAD);
	if(sfd == nil)
		return sys->sprint("cannot open %s: %r", src);

	dfd := sys->create(dst, Sys->OWRITE, 8r644);
	if(dfd == nil) {
		sfd = nil;
		return sys->sprint("cannot create %s: %r", dst);
	}

	buf := array[8192] of byte;
	for(;;) {
		n := sys->read(sfd, buf, len buf);
		if(n <= 0)
			break;
		if(sys->write(dfd, buf, n) != n) {
			sfd = nil;
			dfd = nil;
			return sys->sprint("write error copying to %s: %r", dst);
		}
	}

	sfd = nil;
	dfd = nil;
	return nil;
}

# Copy all .dis files from a directory
copydisfiles(srcdir, dstdir: string): string
{
	if(sys == nil)
		init();

	fd := sys->open(srcdir, Sys->OREAD);
	if(fd == nil)
		return sys->sprint("cannot open %s: %r", srcdir);

	for(;;) {
		(n, dirs) := sys->dirread(fd);
		if(n <= 0)
			break;
		for(i := 0; i < n; i++) {
			name := dirs[i].name;
			# Copy .dis files and recurse into subdirectories
			if(dirs[i].mode & Sys->DMDIR) {
				# Create subdirectory and recurse
				subdir := dstdir + "/" + name;
				err := mkdirp(subdir);
				if(err != nil) {
					fd = nil;
					return err;
				}
				err = copydisfiles(srcdir + "/" + name, subdir);
				if(err != nil) {
					fd = nil;
					return err;
				}
			} else if(len name > 4 && name[len name - 4:] == ".dis") {
				err := copyfile(srcdir + "/" + name, dstdir + "/" + name);
				if(err != nil) {
					fd = nil;
					return err;
				}
			}
		}
	}
	fd = nil;
	return nil;
}

# Copy essential modules needed by subagent
# These modules are required for the agent loop to function after NEWNS
copyessentialmodules(srcdir, dstdir: string): string
{
	if(sys == nil)
		init();

	# Essential modules for subagent operation
	# bufio.dis - needed for I/O buffering
	# string.dis - needed for string manipulation
	# arg.dis - may be needed by tools
	modules := array[] of {"bufio.dis", "string.dis", "arg.dis"};

	for(i := 0; i < len modules; i++) {
		modname := modules[i];
		srcpath := srcdir + "/" + modname;
		dstpath := dstdir + "/" + modname;

		# Check if module exists
		(ok, nil) := sys->stat(srcpath);
		if(ok >= 0) {
			err := copyfile(srcpath, dstpath);
			if(err != nil)
				return err;
		}
		# Not an error if module doesn't exist - some may be optional
	}

	return nil;
}

# Copy a directory tree recursively
# Copies all regular files and subdirectories from src to dst
# Skips device files and other special files (which can't/shouldn't be copied)
copytree(src, dst: string): string
{
	if(sys == nil)
		init();

	# Create destination directory
	err := mkdirp(dst);
	if(err != nil)
		return err;

	fd := sys->open(src, Sys->OREAD);
	if(fd == nil)
		return sys->sprint("cannot open %s: %r", src);

	for(;;) {
		(n, dirs) := sys->dirread(fd);
		if(n <= 0)
			break;
		for(i := 0; i < n; i++) {
			name := dirs[i].name;
			mode := dirs[i].mode;

			# Skip . and ..
			if(name == "." || name == "..")
				continue;

			# Skip device files and other special types
			# DMDIR = directory, all other DM* are special
			# Regular files have no DM* bits set (just permission bits)
			if(mode & (Sys->DMAPPEND | Sys->DMEXCL | Sys->DMAUTH))
				continue;

			srcpath := src + "/" + name;
			dstpath := dst + "/" + name;

			if(mode & Sys->DMDIR) {
				# Recursively copy subdirectory
				err = copytree(srcpath, dstpath);
				if(err != nil) {
					fd = nil;
					return err;
				}
			} else {
				# Check if it's a regular file by looking at type in path
				# Device files in /dev are synthesized and can't be copied
				if(isdevicepath(srcpath))
					continue;

				# Copy regular file
				err = copyfile(srcpath, dstpath);
				if(err != nil) {
					fd = nil;
					return err;
				}
			}
		}
	}
	fd = nil;
	return nil;
}

# Check if path is a device file (in /dev or #-prefixed)
isdevicepath(path: string): int
{
	if(len path >= 5 && path[0:5] == "/dev/")
		return 1;
	if(len path > 0 && path[0] == '#')
		return 1;
	return 0;
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

	# Get ACTUAL wall clock time (seconds since epoch) - survives reboots
	# This is critical for stale sandbox detection to work correctly
	now := 0;
	if(daytime != nil)
		now = daytime->now();
	else
		now = sys->millisec() / 1000;  # Fallback, but less reliable

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

	# Copy essential runtime library modules (instead of bind)
	# NOTE: We COPY instead of BIND because NEWNS doesn't preserve binds.
	# After NEWNS, binds from parent are lost. Copying ensures modules survive.
	(ok, nil) := sys->stat("/dis/lib");
	if(ok >= 0) {
		err = mkdirp(disdir + "/lib");
		if(err != nil) {
			cleanupsandbox(caps.sandboxid);
			return err;
		}
		# Copy essential modules needed by subagent
		err = copyessentialmodules("/dis/lib", disdir + "/lib");
		if(err != nil) {
			cleanupsandbox(caps.sandboxid);
			return err;
		}
		binds = ref BindRecord("/dis/lib", disdir + "/lib", 0) :: binds;  # 0 = copy
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

	# 8. Set up LLM access if LLM config is provided
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

		# Write per-agent LLM configuration files FIRST
		# These are read-only to child (parent sets policy)
		llmdir := sandboxdir + "/n/llm";

		# Write model configuration
		err = writeconfigfile(llmdir + "/config_model", caps.llmconfig.model);
		if(err != nil) {
			cleanupsandbox(caps.sandboxid);
			return err;
		}

		# Write temperature configuration
		err = writeconfigfile(llmdir + "/config_temperature", sys->sprint("%g", caps.llmconfig.temperature));
		if(err != nil) {
			cleanupsandbox(caps.sandboxid);
			return err;
		}

		# Write system prompt if provided
		if(caps.llmconfig.system != "") {
			err = writeconfigfile(llmdir + "/config_system", caps.llmconfig.system);
			if(err != nil) {
				cleanupsandbox(caps.sandboxid);
				return err;
			}
		}

		# Bind parent's /n/llm if it exists (gives child LLM access)
		# Use MBEFORE so config files remain visible alongside llm9p
		(llmok, nil) := sys->stat("/n/llm");
		if(llmok >= 0) {
			if(sys->bind("/n/llm", sandboxdir + "/n/llm", Sys->MBEFORE) < 0) {
				# Not fatal - child just won't have LLM access
				# Log for debugging but continue
				sys->fprint(sys->fildes(2), "warning: cannot bind /n/llm: %r\n");
			} else {
				binds = ref BindRecord("/n/llm", sandboxdir + "/n/llm", Sys->MBEFORE) :: binds;
			}
		}
	}

	# 9. Copy granted paths from parent namespace
	# NOTE: We COPY instead of BIND because NEWNS doesn't preserve binds.
	# After NEWNS, the sandbox becomes root and binds from parent are lost.
	# Copying files ensures they survive NEWNS.
	for(p := caps.paths; p != nil; p = tl p) {
		srcpath := hd p;

		# Verify source exists in parent namespace
		err = verifyownership(srcpath);
		if(err != nil) {
			cleanupsandbox(caps.sandboxid);
			return sys->sprint("cannot copy path %s: %s", srcpath, err);
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

		# Copy source to destination (file or directory)
		(ok, dir) := sys->stat(srcpath);
		if(ok >= 0 && (dir.mode & Sys->DMDIR)) {
			# Source is directory - copy recursively
			err = copytree(srcpath, dstpath);
			if(err != nil) {
				cleanupsandbox(caps.sandboxid);
				return err;
			}
		} else {
			# Source is file - copy it
			err = copyfile(srcpath, dstpath);
			if(err != nil) {
				cleanupsandbox(caps.sandboxid);
				return err;
			}
		}
		binds = ref BindRecord(srcpath, dstpath, 0) :: binds;  # 0 = copy, not bind
	}

	# 10. Write audit log
	emitauditlog(caps.sandboxid, binds);

	return nil;
}

# Clean up sandbox directory after child exits
# Safe to call multiple times - returns early if already cleaned
cleanupsandbox(sandboxid: string)
{
	if(sys == nil)
		init();

	# Validate ID to prevent traversal
	if(validatesandboxid(sandboxid) != nil)
		return;

	sandboxdir := sandboxpath(sandboxid);

	# Check if sandbox exists - if not, already cleaned up
	# This makes concurrent cleanup calls safe
	(ok, nil) := sys->stat(sandboxdir);
	if(ok < 0)
		return;

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
		"/n/llm",		# LLM mount (if bound)
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

# Helper: Write a configuration file with content
# Used for per-agent LLM config files
writeconfigfile(path, content: string): string
{
	if(sys == nil)
		init();

	fd := sys->create(path, Sys->OWRITE, FILE_MODE);
	if(fd == nil)
		return sys->sprint("cannot create config %s: %r", path);

	data := array of byte content;
	if(sys->write(fd, data, len data) != len data) {
		fd = nil;
		return sys->sprint("cannot write config %s: %r", path);
	}
	fd = nil;
	return nil;
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
