implement NsConstruct;

#
# nsconstruct.b - Namespace construction for Veltro agents (v3)
#
# SECURITY MODEL (v3): FORKNS + bind-replace
# ============================================
# Replace NEWNS + sandbox with FORKNS + bind-replace (MREPL).
# restrictdir() is the core primitive:
#   1. Create shadow directory
#   2. Bind allowed items from target into shadow
#   3. Bind shadow over target (MREPL)
# Result: target only shows allowed items. Everything else is invisible.
#
# This is an allowlist operation. No file copying, no sandbox directories,
# no cleanup needed. Capability attenuation is natural: children fork an
# already-restricted namespace and can only narrow further.
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "nsconstruct.m";

# Shadow directories live under /tmp/veltro/.ns/ so they survive
# the /tmp restriction (which allows only "veltro/")
SHADOW_BASE: con "/tmp/veltro/.ns/shadow";
AUDIT_DIR: con "/tmp/veltro/.ns/audit";

# Directory/file permissions
DIR_MODE: con 8r700 | Sys->DMDIR;  # rwx------ directory
FILE_MODE: con 8r600;              # rw------- file

# Per-process shadow sequence counter
# Uses PID prefix to avoid collisions between parent and child processes
shadowseq := 0;

# Thread-safe initialization
inited := 0;

init()
{
	if(inited)
		return;

	sys = load Sys Sys->PATH;
	inited = 1;
}

# Core primitive: restrict a directory to only allowed entries
# Creates a shadow dir with only the allowed items, then replaces target
restrictdir(target: string, allowed: list of string): string
{
	if(sys == nil)
		init();

	# Create unique shadow dir using PID + sequence
	pid := sys->pctl(0, nil);
	shadowdir := sys->sprint("%s/%d-%d", SHADOW_BASE, pid, shadowseq++);
	err := mkdirp(shadowdir);
	if(err != nil)
		return err;

	for(a := allowed; a != nil; a = tl a) {
		item := hd a;
		srcpath := target + "/" + item;
		dstpath := shadowdir + "/" + item;

		# Check if source exists and get type
		(ok, dir) := sys->stat(srcpath);
		if(ok < 0)
			continue;  # Skip items that don't exist in target

		# Create mount point matching source type
		if(dir.mode & Sys->DMDIR) {
			dfd := sys->create(dstpath, Sys->OREAD, DIR_MODE);
			if(dfd != nil)
				dfd = nil;
		} else {
			dfd := sys->create(dstpath, Sys->OWRITE, FILE_MODE);
			if(dfd != nil)
				dfd = nil;
		}

		# Bind original into shadow
		if(sys->bind(srcpath, dstpath, Sys->MREPL) < 0)
			return sys->sprint("cannot bind %s: %r", srcpath);
	}

	# Replace target with shadow — only allowed items visible
	if(sys->bind(shadowdir, target, Sys->MREPL) < 0)
		return sys->sprint("cannot replace %s: %r", target);

	return nil;
}

# Apply full namespace restriction policy
restrictns(caps: ref Capabilities): string
{
	if(sys == nil)
		init();

	# Set up infrastructure directories first (before any restrictdir calls)
	# These must exist because restrictdir creates shadow dirs under SHADOW_BASE
	mkdirp("/tmp/veltro");
	mkdirp("/tmp/veltro/scratch");
	mkdirp(SHADOW_BASE);
	mkdirp(AUDIT_DIR);

	# 1. Restrict /dis to: lib/, veltro/ (plus shell if shellcmds granted)
	disallow := "lib" :: "veltro" :: nil;
	if(caps.shellcmds != nil) {
		disallow = "sh.dis" :: disallow;
		for(c := caps.shellcmds; c != nil; c = tl c)
			disallow = (hd c) + ".dis" :: disallow;
	}
	err := restrictdir("/dis", disallow);
	if(err != nil)
		return sys->sprint("restrict /dis: %s", err);

	# 2. If tools specified, restrict /dis/veltro/tools/ to granted tools only
	if(caps.tools != nil) {
		toolallow: list of string;
		for(t := caps.tools; t != nil; t = tl t)
			toolallow = (hd t) + ".dis" :: toolallow;
		err = restrictdir("/dis/veltro/tools", toolallow);
		if(err != nil)
			return sys->sprint("restrict /dis/veltro/tools: %s", err);
	}

	# 3. Restrict /dev to: cons, null
	err = restrictdir("/dev", "cons" :: "null" :: nil);
	if(err != nil)
		return sys->sprint("restrict /dev: %s", err);

	# 4. Unmount /n/local (host filesystem — primary security concern)
	sys->unmount(nil, "/n/local");

	# 5. Restrict /n to allowed network paths
	(nok, nil) := sys->stat("/n");
	if(nok >= 0) {
		nallow: list of string;
		# Keep /n/llm if LLM is available
		(llmok, nil) := sys->stat("/n/llm");
		if(llmok >= 0)
			nallow = "llm" :: nallow;
		# Keep /n/mcp if mc9p providers exist
		if(caps.mcproviders != nil) {
			(mcpok, nil) := sys->stat("/n/mcp");
			if(mcpok >= 0)
				nallow = "mcp" :: nallow;
		}
		if(nallow != nil) {
			err = restrictdir("/n", nallow);
			if(err != nil)
				return sys->sprint("restrict /n: %s", err);
		}
	}

	# 6. Restrict /lib to: veltro/ (read-only data for agents)
	(libok, nil) := sys->stat("/lib");
	if(libok >= 0) {
		err = restrictdir("/lib", "veltro" :: nil);
		if(err != nil)
			return sys->sprint("restrict /lib: %s", err);
	}

	# 7. Restrict /tmp to: veltro/ (LAST — shadow dirs are under here)
	# After this, /tmp only shows veltro/ which contains scratch/ and .ns/
	err = restrictdir("/tmp", "veltro" :: nil);
	if(err != nil)
		return sys->sprint("restrict /tmp: %s", err);

	return nil;
}

# Verify namespace matches expected security policy
verifyns(expected: list of string): string
{
	if(sys == nil)
		init();

	# Read current namespace
	pid := sys->pctl(0, nil);
	nspath := sys->sprint("/prog/%d/ns", pid);

	content := readfile(nspath);
	if(content == "")
		return nil;  # Cannot read /prog — not a security failure

	# Check for known dangerous paths that should not appear
	(nil, lines) := sys->tokenize(content, "\n");
	for(; lines != nil; lines = tl lines) {
		line := hd lines;
		if(contains(line, "/n/local"))
			return "violation: /n/local still accessible";
		if(contains(line, "'#U'") && !contains(line, "/tmp"))
			return sys->sprint("violation: #U binding found: %s", line);
	}

	# Check that all expected paths are present
	for(e := expected; e != nil; e = tl e) {
		path := hd e;
		(ok, nil) := sys->stat(path);
		if(ok < 0)
			return sys->sprint("expected path missing: %s", path);
	}

	return nil;
}

# Emit audit log of namespace restriction operations
emitauditlog(id: string, ops: list of string)
{
	if(sys == nil)
		init();

	mkdirp(AUDIT_DIR);

	auditpath := AUDIT_DIR + "/" + id + ".ns";
	fd := sys->create(auditpath, Sys->OWRITE, FILE_MODE);
	if(fd == nil)
		return;

	sys->fprint(fd, "# Veltro Namespace Audit (v3)\n# ID: %s\n\n", id);

	# Write operations in reverse order (oldest first)
	revops: list of string;
	for(; ops != nil; ops = tl ops)
		revops = hd ops :: revops;
	for(; revops != nil; revops = tl revops)
		sys->fprint(fd, "%s\n", hd revops);

	# Dump current namespace state
	pid := sys->pctl(0, nil);
	nscontent := readfile(sys->sprint("/prog/%d/ns", pid));
	if(nscontent != "")
		sys->fprint(fd, "\n# Current namespace:\n%s", nscontent);

	fd = nil;
}

# Helper: create directory with parents
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

	fd := sys->create(path, Sys->OREAD, DIR_MODE);
	if(fd == nil)
		return sys->sprint("cannot create %s: %r", path);
	fd = nil;
	return nil;
}

# Helper: create parent directory
mkparent(path: string): string
{
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

# Helper: read file contents
readfile(path: string): string
{
	fd := sys->open(path, Sys->OREAD);
	if(fd == nil)
		return "";

	result := "";
	buf := array[8192] of byte;
	for(;;) {
		n := sys->read(fd, buf, len buf);
		if(n <= 0)
			break;
		result += string buf[0:n];
	}
	return result;
}

# Helper: check if string contains substring
contains(s, sub: string): int
{
	if(len sub > len s)
		return 0;
	for(i := 0; i <= len s - len sub; i++) {
		if(s[i:i+len sub] == sub)
			return 1;
	}
	return 0;
}
