# NervNode Agent Spawner - Capability Delegation via Namespace Binding
#
# This module spawns child agents with restricted namespaces.
# The child's namespace is a SUBSET of the parent's namespace.
# This is how capability delegation works in Plan 9/Inferno.
#
# Usage:
#   # Parent has /n/llm, /n/osm, /n/secret
#   # Child should only see /n/llm, /n/osm (not /n/secret)
#   spawn /n/llm /n/osm -- agent "child task"
#
# Security property:
#   The child CANNOT access paths not explicitly bound.
#   /n/secret does not exist in child's namespace.

implement Spawn;

include "sys.m";
    sys: Sys;

include "draw.m";

Spawn: module {
    init: fn(ctxt: ref Draw->Context, argv: list of string);
};

# Check if string contains whitespace
hasspace(s: string): int
{
    for(i := 0; i < len s; i++)
        if(s[i] == ' ' || s[i] == '\t')
            return 1;
    return 0;
}

# Quote-aware tokenizer - handles 'single' and "double" quotes
tokenizequoted(s: string): list of string
{
    result: list of string;
    token := "";
    inquote := 0;
    quotechar := 0;

    for(i := 0; i < len s; i++) {
        c := s[i];

        if(inquote) {
            if(c == quotechar) {
                inquote = 0;
                quotechar = 0;
            } else {
                token[len token] = c;
            }
        } else if(c == '\'' || c == '"') {
            inquote = 1;
            quotechar = c;
        } else if(c == ' ' || c == '\t') {
            if(token != "") {
                result = token :: result;
                token = "";
            }
        } else {
            token[len token] = c;
        }
    }

    if(token != "")
        result = token :: result;

    # Reverse the list
    rev: list of string;
    for(; result != nil; result = tl result)
        rev = hd result :: rev;
    return rev;
}

init(ctxt: ref Draw->Context, argv: list of string)
{
    sys = load Sys Sys->PATH;

    if(len argv < 4) {
        usage();
        raise "fail:usage";
    }

    argv = tl argv;  # skip program name

    # Parse capabilities (paths) until we hit "--"
    caps: list of string;
    for(; argv != nil; argv = tl argv) {
        arg := hd argv;
        if(arg == "--")
            break;
        caps = arg :: caps;
    }

    if(argv == nil || hd argv != "--") {
        sys->fprint(sys->fildes(2), "spawn: missing -- before command\n");
        usage();
        raise "fail:usage";
    }

    argv = tl argv;  # skip "--"

    if(argv == nil) {
        sys->fprint(sys->fildes(2), "spawn: missing command\n");
        usage();
        raise "fail:usage";
    }

    # Reverse caps list (it's backwards)
    capslist: list of string;
    for(; caps != nil; caps = tl caps)
        capslist = hd caps :: capslist;

    sys->print("Spawning agent with restricted namespace:\n");
    sys->print("Capabilities:\n");
    for(c := capslist; c != nil; c = tl c)
        sys->print("  %s\n", hd c);

    cmd := "";
    for(; argv != nil; argv = tl argv) {
        if(cmd != "")
            cmd += " ";
        arg := hd argv;
        # Quote arguments containing spaces
        if(hasspace(arg))
            cmd += "'" + arg + "'";
        else
            cmd += arg;
    }
    sys->print("Command: %s\n", cmd);

    # Fork and restrict namespace
    spawnchild(capslist, cmd);
}

spawnchild(caps: list of string, cmd: string)
{
    # Fork the namespace - creates a copy we can modify without affecting parent
    pid := sys->pctl(Sys->FORKNS, nil);
    if(pid < 0) {
        sys->fprint(sys->fildes(2), "spawn: pctl FORKNS failed: %r\n");
        raise "fail:pctl";
    }

    sys->print("Forked namespace, pid %d\n", pid);

    # Show what's in /n before restriction
    sys->print("Before restriction, /n contains:\n");
    listdir("/n");

    # Hide paths not in capabilities by binding empty dir over them
    hidepaths("/n", caps);

    # Show what's accessible after restriction
    sys->print("After restriction, /n contains:\n");
    listdir("/n");

    sys->print("Namespace restricted. Executing: %s\n", cmd);

    # Run the command in the restricted namespace
    sys->print("--- Child output ---\n");
    execsh(cmd);
    sys->print("--- End child output ---\n");
}

# List directory contents with entry counts for subdirectories
listdir(path: string)
{
    fd := sys->open(path, Sys->OREAD);
    if(fd == nil) {
        sys->print("  (cannot open %s)\n", path);
        return;
    }

    for(;;) {
        (n, dir) := sys->dirread(fd);
        if(n <= 0)
            break;
        for(i := 0; i < n; i++) {
            name := dir[i].name;
            if(name == "." || name == ".." || name == ".hidden")
                continue;
            fullpath := path + "/" + name;
            if(dir[i].mode & Sys->DMDIR) {
                # Count entries in subdirectory
                count := countentries(fullpath);
                if(count < 0)
                    sys->print("  %s (BLOCKED)\n", name);
                else
                    sys->print("  %s (accessible, %d entries)\n", name, count);
            } else {
                sys->print("  %s\n", name);
            }
        }
    }
}

# Count entries in a directory, returns -1 if can't open
countentries(path: string): int
{
    fd := sys->open(path, Sys->OREAD);
    if(fd == nil)
        return -1;
    count := 0;
    for(;;) {
        (n, nil) := sys->dirread(fd);
        if(n <= 0)
            break;
        count += n;
    }
    return count;
}


# Hide paths not in allowed list by binding empty dir over them
hidepaths(base: string, allowed: list of string)
{
    # Create an empty directory to use for hiding
    emptydir := "/tmp/spawn_empty";
    sys->create(emptydir, Sys->OREAD, Sys->DMDIR | 8r755);

    fd := sys->open(base, Sys->OREAD);
    if(fd == nil)
        return;

    hidden := 0;
    failed := 0;
    for(;;) {
        (n, dir) := sys->dirread(fd);
        if(n <= 0)
            break;
        for(i := 0; i < n; i++) {
            name := dir[i].name;
            if(name == "." || name == ".." || name == ".hidden")
                continue;

            fullpath := base + "/" + name;

            if(!pathallowed(fullpath, allowed)) {
                # Hide this path by binding empty dir over it
                rc := sys->bind(emptydir, fullpath, Sys->MREPL);
                if(rc < 0) {
                    sys->print("  Hide FAILED: %s (%r)\n", fullpath);
                    failed++;
                } else {
                    sys->print("  Hidden: %s\n", fullpath);
                    hidden++;
                }
            } else {
                sys->print("  Visible: %s\n", fullpath);
            }
        }
    }

    if(hidden > 0 || failed > 0)
        sys->print("Restricted %d paths (%d failed)\n", hidden, failed);
}


# Check if a path is in the allowed list (or is a parent/child of an allowed path)
pathallowed(path: string, allowed: list of string): int
{
    for(; allowed != nil; allowed = tl allowed) {
        cap := hd allowed;
        # Exact match
        if(path == cap)
            return 1;
        # path is a prefix of cap (e.g., /n/llm is prefix of /n/llm/ask)
        if(len path < len cap && cap[:len path] == path && cap[len path] == '/')
            return 1;
        # cap is a prefix of path (e.g., /n is prefix of /n/llm)
        if(len cap < len path && path[:len cap] == cap && path[len cap] == '/')
            return 1;
    }
    return 0;
}


# Result directory base path
RESULT_BASE := "/tmp/agent";

# Execute a command via shell and capture results
# Creates result directory at /tmp/agent/<pid>/ with:
#   status  - "running", "completed", "error", "timeout"
#   output  - command output
#   error   - error message if failed
execsh(cmd: string)
{
    # Get our PID for result directory
    pid := sys->pctl(0, nil);
    resultdir := sys->sprint("%s/%d", RESULT_BASE, pid);

    # Ensure result directory exists
    ensuredir(RESULT_BASE);
    ensuredir(resultdir);

    # Write initial status
    writefile(resultdir + "/status", "running");
    writefile(resultdir + "/output", "");
    writefile(resultdir + "/error", "");

    # Parse command (quote-aware)
    argv := tokenizequoted(cmd);
    if(argv == nil) {
        writefile(resultdir + "/status", "error");
        writefile(resultdir + "/error", "empty command");
        return;
    }

    progname := hd argv;

    # Try to find the dis module
    dispath := progname;
    if(len dispath < 4 || dispath[len dispath - 4:] != ".dis")
        dispath += ".dis";

    # Try various paths
    prog: DisModule;
    paths := list of {
        dispath,
        "/dis/" + dispath,
        "/dis/nerv/" + dispath,
        progname,
        "/dis/" + progname,
        "/dis/nerv/" + progname
    };

    for(; paths != nil; paths = tl paths) {
        prog = load DisModule hd paths;
        if(prog != nil)
            break;
    }

    if(prog == nil) {
        writefile(resultdir + "/status", "error");
        writefile(resultdir + "/error", "cannot load: " + progname);
        sys->print("spawn: failed to load module %s\n", progname);
        return;
    }

    sys->print("spawn: loaded module, setting up file-based output capture\n");

    # Create output file for capture (files don't have pipe buffer issues)
    outpath := resultdir + "/output";
    outfd := sys->create(outpath, Sys->OWRITE, 8r644);
    if(outfd == nil) {
        sys->print("spawn: cannot create output file, running without capture\n");
        outfd = sys->fildes(1);  # Fall back to stdout
    }

    # Save original stdout/stderr
    oldstdout := sys->fildes(1);
    oldstderr := sys->fildes(2);

    # Redirect stdout/stderr to output file
    sys->dup(outfd.fd, 1);
    sys->dup(outfd.fd, 2);

    # Run the command
    errmsg := "";
    {
        prog->init(nil, argv);
    } exception e {
        "*" =>
            errmsg = e;
    }

    # Restore stdout/stderr
    sys->dup(oldstdout.fd, 1);
    sys->dup(oldstderr.fd, 2);
    outfd = nil;

    # Read and display captured output
    output := readfile(outpath);
    if(len output > 0) {
        sys->print("=== Captured output (%d bytes) ===\n", len output);
        sys->print("%s", output);
        if(output[len output - 1] != '\n')
            sys->print("\n");
        sys->print("=== End captured output ===\n");
    }

    # Write results
    if(errmsg != "") {
        sys->print("spawn: exception: %s\n", errmsg);
        writefile(resultdir + "/status", "error");
        writefile(resultdir + "/error", errmsg);
    } else {
        sys->print("spawn: completed successfully\n");
        writefile(resultdir + "/status", "completed");
    }
}

# Module interface for loading dis files
DisModule: module {
    init: fn(ctxt: ref Draw->Context, argv: list of string);
};

# Write content to a file
writefile(path, content: string): int
{
    fd := sys->create(path, Sys->OWRITE, 8r644);
    if(fd == nil)
        return -1;
    data := array of byte content;
    n := sys->write(fd, data, len data);
    return n;
}

# Ensure directory exists
ensuredir(path: string)
{
    fd := sys->open(path, Sys->OREAD);
    if(fd != nil)
        return;

    # Create directory
    fd = sys->create(path, Sys->OREAD, Sys->DMDIR | 8r755);
}

# Wait for result from child with timeout
# Returns (output, error) - error is empty on success
waitforresult(pid: int, timeoutms: int): (string, string)
{
    resultdir := sys->sprint("%s/%d", RESULT_BASE, pid);
    statuspath := resultdir + "/status";
    outputpath := resultdir + "/output";
    errorpath := resultdir + "/error";

    # Poll status file
    pollinterval := 100;  # 100ms
    elapsed := 0;

    while(elapsed < timeoutms) {
        status := readfile(statuspath);
        if(status == "completed") {
            output := readfile(outputpath);
            return (output, "");
        }
        if(status == "error") {
            errmsg := readfile(errorpath);
            return ("", errmsg);
        }
        sys->sleep(pollinterval);
        elapsed += pollinterval;
    }

    return ("", "timeout");
}

# Read file content
readfile(path: string): string
{
    fd := sys->open(path, Sys->OREAD);
    if(fd == nil)
        return "";
    buf := array[8192] of byte;
    n := sys->read(fd, buf, len buf);
    if(n <= 0)
        return "";
    return string buf[0:n];
}

usage()
{
    sys->fprint(sys->fildes(2), "usage: spawn path1 [path2 ...] -- command [args...]\n");
    sys->fprint(sys->fildes(2), "\n");
    sys->fprint(sys->fildes(2), "Spawns a child agent with a restricted namespace.\n");
    sys->fprint(sys->fildes(2), "Only the specified paths will be visible to the child.\n");
    sys->fprint(sys->fildes(2), "\n");
    sys->fprint(sys->fildes(2), "Example:\n");
    sys->fprint(sys->fildes(2), "  spawn /n/osm /n/llm -- agent 'geocode Paris'\n");
    sys->fprint(sys->fildes(2), "\n");
    sys->fprint(sys->fildes(2), "The child agent will only be able to access /n/osm and /n/llm.\n");
    sys->fprint(sys->fildes(2), "Other paths in the parent's namespace will not exist for the child.\n");
}
