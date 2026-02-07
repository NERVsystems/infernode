implement Limbtest;

#
# limbtest - Limbo test runner
#
# Usage: limbtest [-v] [-c] [packages...]
#
# Discovers *_test.dis files, loads and runs them.
# With -c flag, compiles *_test.b files first.
# Test files should use the testing module.
#

include "sys.m";
	sys: Sys;

include "draw.m";

include "arg.m";
	arg: Arg;

include "readdir.m";
	readdir: Readdir;

include "string.m";
	str: String;

Limbtest: module
{
	init: fn(ctxt: ref Draw->Context, args: list of string);
};

Command: module
{
	init: fn(ctxt: ref Draw->Context, args: list of string);
};

verbosemode := 0;
compilemode := 0;
ctxt: ref Draw->Context;

init(drawctxt: ref Draw->Context, args: list of string)
{
	sys = load Sys Sys->PATH;
	arg = load Arg Arg->PATH;
	readdir = load Readdir Readdir->PATH;
	str = load String String->PATH;
	ctxt = drawctxt;

	arg->init(args);
	arg->setusage("limbtest [-v] [-c] [paths...]");

	while((opt := arg->opt()) != 0) {
		case opt {
		'v' =>
			verbosemode = 1;
		'c' =>
			compilemode = 1;
		* =>
			arg->usage();
		}
	}

	paths := arg->argv();
	if(paths == nil)
		paths = "." :: nil;

	totalfailed := 0;
	totalpassed := 0;

	for(; paths != nil; paths = tl paths) {
		path := hd paths;
		(passed, failed) := runtests(path);
		totalpassed += passed;
		totalfailed += failed;
	}

	sys->fprint(sys->fildes(2), "\n=== SUMMARY ===\n");
	if(totalfailed > 0) {
		sys->fprint(sys->fildes(2), "FAIL: %d test files passed, %d failed\n", totalpassed, totalfailed);
		raise "fail:tests failed";
	}

	if(totalpassed > 0)
		sys->fprint(sys->fildes(2), "PASS: %d test files\n", totalpassed);
	else
		sys->fprint(sys->fildes(2), "No tests found\n");
}

# runtests: run all tests in a path
runtests(path: string): (int, int)
{
	passed := 0;
	failed := 0;

	# Handle recursive paths ending in /...
	recursive := 0;
	if(len path >= 4 && path[len path - 4:] == "/...") {
		recursive = 1;
		path = path[:len path - 4];
		if(path == "")
			path = ".";
	}

	(ok, dir) := sys->stat(path);
	if(ok < 0) {
		sys->fprint(sys->fildes(2), "limbtest: cannot stat %s: %r\n", path);
		return (0, 1);
	}

	if(dir.mode & Sys->DMDIR) {
		# It's a directory, find test files
		(p, f) := rundir(path, recursive);
		passed += p;
		failed += f;
	} else if(issuffix(path, "_test.dis")) {
		# It's a .dis file
		if(runtest(path) < 0)
			failed++;
		else
			passed++;
	} else if(issuffix(path, "_test.b")) {
		# It's a .b file
		if(compilemode) {
			if(compile(path) < 0) {
				failed++;
				return (passed, failed);
			}
		}
		dispath := path[:len path - 2] + ".dis";
		if(runtest(dispath) < 0)
			failed++;
		else
			passed++;
	} else {
		sys->fprint(sys->fildes(2), "limbtest: %s: not a test file or directory\n", path);
		return (0, 1);
	}

	return (passed, failed);
}

# rundir: run all tests in a directory
rundir(path: string, recursive: int): (int, int)
{
	passed := 0;
	failed := 0;

	(dirs, n) := readdir->init(path, Readdir->NAME);
	if(n < 0) {
		sys->fprint(sys->fildes(2), "limbtest: cannot read %s: %r\n", path);
		return (0, 1);
	}

	for(i := 0; i < n; i++) {
		d := dirs[i];
		fullpath := path + "/" + d.name;

		if(d.mode & Sys->DMDIR) {
			if(recursive) {
				(p, f) := rundir(fullpath, recursive);
				passed += p;
				failed += f;
			}
		} else if(issuffix(d.name, "_test.b") && compilemode) {
			dispath := fullpath[:len fullpath - 2] + ".dis";

			# Compile the test
			if(compile(fullpath) < 0) {
				failed++;
				continue;
			}

			# Run the test
			if(runtest(dispath) < 0)
				failed++;
			else
				passed++;
		} else if(issuffix(d.name, "_test.dis")) {
			# Pre-compiled test, just run it
			if(runtest(fullpath) < 0)
				failed++;
			else
				passed++;
		}
	}

	return (passed, failed);
}

# compile: compile a .b file to .dis using limbo
compile(srcpath: string): int
{
	sys->fprint(sys->fildes(2), "=== COMPILE %s\n", srcpath);

	# Load and run the limbo compiler
	limbo := load Command "/dis/limbo.dis";
	if(limbo == nil) {
		sys->fprint(sys->fildes(2), "    cannot load limbo compiler: %r\n");
		return -1;
	}

	{
		limbo->init(ctxt, "limbo" :: "-I" :: "/module" :: srcpath :: nil);
	} exception e {
	"*" =>
		sys->fprint(sys->fildes(2), "    compile failed: %s\n", e);
		return -1;
	}

	limbo = nil;

	# Check if .dis file was created
	dispath := srcpath[:len srcpath - 2] + ".dis";
	(ok, nil) := sys->stat(dispath);
	if(ok < 0) {
		sys->fprint(sys->fildes(2), "    compile failed: %s not created\n", dispath);
		return -1;
	}

	if(verbosemode)
		sys->fprint(sys->fildes(2), "    compiled: %s\n", dispath);
	return 0;
}

# runtest: run a compiled test
runtest(dispath: string): int
{
	sys->fprint(sys->fildes(2), "\n=== TEST  %s\n", dispath);

	# Load the test module
	testmod := load Command dispath;
	if(testmod == nil) {
		sys->fprint(sys->fildes(2), "    cannot load %s: %r\n", dispath);
		return -1;
	}

	# Build args
	args: list of string;
	args = dispath :: nil;
	if(verbosemode)
		args = dispath :: "-v" :: nil;

	# Run the test
	failed := 0;
	{
		testmod->init(ctxt, args);
	} exception e {
	"fail:*" =>
		failed = 1;
	"*" =>
		failed = 1;
		sys->fprint(sys->fildes(2), "    exception: %s\n", e);
	}

	testmod = nil;
	if(failed)
		return -1;
	return 0;
}

# issuffix: check if s ends with suffix
issuffix(s, suffix: string): int
{
	if(len s < len suffix)
		return 0;
	return s[len s - len suffix:] == suffix;
}
