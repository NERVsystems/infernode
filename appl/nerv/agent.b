# InferNode Agent - Namespace-Bounded LLM Agent
#
# This agent runs inside Inferno and accesses the LLM and tools
# entirely through the filesystem namespace. Its capabilities are
# bounded by what's mounted in its namespace.
#
# Usage:
#   mount -A tcp!127.0.0.1!5640 /n/llm    # Mount llm9p server
#   mount -A tcp!osm-server!5640 /n/osm   # Mount OSM server (optional)
#   agent [-cleanup] "task description"
#
# Options:
#   -cleanup    Delete created Xenith windows on agent exit
#
# Examples:
#   agent 'Say hello'
#   agent 'Create a Xenith window and write Hello World to it'
#   agent -cleanup 'Create a window, show the date, then clean up'

implement Agent;

include "sys.m";
    sys: Sys;

include "bufio.m";
    bufio: Bufio;
    Iobuf: import bufio;

include "draw.m";

include "string.m";
    str: String;

include "sh.m";
    sh: Sh;

# Generic module interface for loading dis files
DisModule: module {
    init: fn(ctxt: ref Draw->Context, argv: list of string);
};

Agent: module {
    init: fn(ctxt: ref Draw->Context, argv: list of string);
};

# Agent configuration - LLM paths (llm9p on port 5640)
LLM_ASK := "/n/llm/ask";
LLM_SYSTEM := "/n/llm/system";
LLM_NEW := "/n/llm/new";

# Xenith UI paths
XENITH_NEW := "/mnt/xenith/new/ctl";
XENITH_BASE := "/mnt/xenith";

# Todo9p path
TODO_BASE := "/n/todo";
TODO9P_DIS := "/dis/nerv/todo9p.dis";

# Cleanup flag (set via -cleanup command-line option)
cleanup_windows := 0;

# Track windows created during execution for cleanup
created_windows: list of int;

# Helper function to count list length
listlen(l: list of int): int
{
    count := 0;
    for(; l != nil; l = tl l)
        count++;
    return count;
}

# Safety limits
MAX_ITERATIONS := 50;      # Maximum iterations before forced stop (one command per iteration)
MAX_ERRORS := 5;           # Maximum consecutive errors before stop
MAX_HISTORY := 20;         # Maximum actions to remember
MAX_RETRIES := 3;          # Maximum retries for transient errors

# Retry backoff intervals (milliseconds)
BACKOFF_1 := 1000;         # 1 second
BACKOFF_2 := 2000;         # 2 seconds
BACKOFF_3 := 4000;         # 4 seconds

# Action record for history
Action: adt {
    cmd:     string;   # Command that was run
    path:    string;   # Primary path involved
    outcome: string;   # "OK" or "ERR"
    detail:  string;   # Result or error message
};

# System prompt - comprehensive documentation for agent capabilities
SYSTEM_PROMPT := "You are an agent running inside Inferno OS with a namespace-bounded sandbox. " +
    "Your capabilities are determined entirely by what files are mounted in your namespace.\n\n" +
    "== Namespace Model ==\n" +
    "Everything is a file. Tools, services, and devices appear as files you can read/write.\n" +
    "Your capabilities are bounded by mounts - you can only access what's been mounted for you.\n\n" +
    "== LLM Interaction ==\n" +
    "To query the LLM: echo 'prompt' > /n/llm/ask && cat /n/llm/ask\n" +
    "To set system context: echo 'context' > /n/llm/system\n" +
    "To start new conversation: echo '' > /n/llm/new\n\n" +
    "== Task Tracking (if /n/todo is mounted) ==\n" +
    "Track tasks via the todo9p filesystem:\n" +
    "  Create: echo 'task description' > /n/todo/new\n" +
    "  List all: cat /n/todo/list\n" +
    "  Read task: cat /n/todo/<id>/content\n" +
    "  Set status: echo 'pending' > /n/todo/<id>/status\n" +
    "              echo 'in_progress' > /n/todo/<id>/status\n" +
    "              echo 'completed' > /n/todo/<id>/status\n" +
    "  Delete: echo 'delete' > /n/todo/<id>/ctl\n\n" +
    "== File Editing ==\n" +
    "Simple text replacement with edit command:\n" +
    "  edit -f /path/to/file -old 'text to find' -new 'replacement'\n" +
    "  edit -f /path/to/file -old 'pattern' -new 'new text' -all\n" +
    "Errors if -old text not found or matches multiple times (use -all for multiple).\n\n" +
    "For complex structural edits, use Sam commands via Xenith:\n" +
    "  echo 'x/pattern/c/replacement/' > /mnt/xenith/<id>/edit\n" +
    "  Sam syntax: x/re/cmd, g/re/cmd, s/re/repl/, c/text/, d, i/text/, a/text/\n\n" +
    "== Xenith UI (if /mnt/xenith is mounted) ==\n" +
    "You can create, write to, delete, resize, and arrange windows.\n" +
    "NOTE: You can only delete windows YOU created. User windows are protected.\n\n" +
    "Window commands:\n" +
    "  xenith new - create window, returns ID\n" +
    "  xenith write <id> <text> - write text to window body\n" +
    "  xenith delete <id> - delete window (only if you created it)\n" +
    "  xenith ctl <id> <command> - send control command\n\n" +
    "Layout control (via ctl or echo to /mnt/xenith/<id>/ctl):\n" +
    "  grow - moderate size increase\n" +
    "  growmax - maximize within column\n" +
    "  growfull - take full column (hides other windows)\n" +
    "  moveto <y> - move to Y pixel position in current column\n" +
    "  tocol <colindex> [<y>] - move to column N at optional Y position\n" +
    "  newcol [<x>] - create new column at X position\n\n" +
    "Window colors (via /mnt/xenith/<id>/colors file):\n" +
    "  Read: cat /mnt/xenith/<id>/colors\n" +
    "  Set: echo 'tagbg #RRGGBB' > /mnt/xenith/<id>/colors\n" +
    "  Keys: tagbg, tagfg, bodybg, bodyfg, bord\n" +
    "  Reset: echo 'reset' > /mnt/xenith/<id>/colors\n" +
    "  IMPORTANT: Hex colors MUST use UPPERCASE (e.g. #FF0000 not #ff0000)\n\n" +
    "== Tool Patterns ==\n" +
    "Type A (query file): echo 'input' > /path/query && cat /path/query\n" +
    "Type B (param files): echo 'val' > /path/param1 && cat /path/result\n" +
    "Use 'ls' to discover tool structure.\n\n" +
    "== Web Access (if /n/web is mounted) ==\n" +
    "Fetch web content:\n" +
    "  echo 'url' > /n/web/url && cat /n/web/result\n" +
    "POST request:\n" +
    "  echo 'url' > /n/web/url\n" +
    "  echo 'POST' > /n/web/method\n" +
    "  echo 'body' > /n/web/body\n" +
    "  cat /n/web/result\n" +
    "Check status: cat /n/web/status\n\n" +
    "== Available Commands ==\n" +
    "Built-in: echo, cat, ls, xenith\n" +
    "Shell commands (Inferno sh, not bash): grep, sed, edit, date, wc, sort, uniq, head, tail\n" +
    "Only use commands you know exist. Do NOT invent commands.\n" +
    "Output raw commands only - no markdown code blocks.\n\n" +
    "== Instructions ==\n" +
    "You are an iterative agent. Output ONE command at a time, then STOP and wait.\n" +
    "After each command, you will see the result before deciding your next action.\n" +
    "Do NOT plan multiple steps ahead - execute, observe, then decide.\n" +
    "Do NOT guess IDs or assume outcomes - wait for actual results.\n" +
    "If a task cannot be done with available commands, say DONE and explain why.\n" +
    "When task is complete, respond with 'DONE' on its own line.";

# Ensure todo9p is mounted, start it if not
ensuretodo(): int
{
    # Check if /n/todo is already accessible
    (ok, nil) := sys->stat(TODO_BASE + "/new");
    if(ok >= 0) {
        sys->print("todo9p: already mounted at %s\n", TODO_BASE);
        return 0;
    }

    # Not mounted, try to start todo9p
    sys->print("todo9p: starting %s %s\n", TODO9P_DIS, TODO_BASE);

    # Load and spawn todo9p
    todo9p := load DisModule TODO9P_DIS;
    if(todo9p == nil) {
        sys->print("todo9p: cannot load %s: %r\n", TODO9P_DIS);
        return -1;
    }

    # Spawn todo9p in background - it will mount itself
    spawn todo9p->init(nil, TODO9P_DIS :: TODO_BASE :: nil);

    # Give it a moment to mount
    sys->sleep(100);

    # Verify it mounted
    (ok, nil) = sys->stat(TODO_BASE + "/new");
    if(ok < 0) {
        sys->print("todo9p: failed to mount at %s\n", TODO_BASE);
        return -1;
    }

    sys->print("todo9p: mounted at %s\n", TODO_BASE);
    return 0;
}

init(ctxt: ref Draw->Context, argv: list of string)
{
    sys = load Sys Sys->PATH;
    bufio = load Bufio Bufio->PATH;
    str = load String String->PATH;
    sh = load Sh Sh->PATH;

    if(bufio == nil || str == nil) {
        sys->fprint(sys->fildes(2), "agent: failed to load modules\n");
        raise "fail:modules";
    }

    if(sh == nil) {
        sys->print("Warning: shell module not available, shell commands disabled\n");
    }

    sys->print("NervNode Agent starting\n");

    # Ensure todo9p is available
    if(ensuretodo() < 0) {
        sys->print("Warning: todo tracking unavailable\n");
    }

    # Initialize window tracking
    created_windows = nil;

    # Parse command line options
    argv = tl argv;  # skip program name
    while(argv != nil) {
        arg := hd argv;
        if(arg == "-cleanup") {
            cleanup_windows = 1;
            argv = tl argv;
        } else if(len arg > 0 && arg[0] == '-') {
            sys->fprint(sys->fildes(2), "usage: agent [-cleanup] 'task description'\n");
            raise "fail:usage";
        } else {
            break;
        }
    }

    # Get task from remaining arguments
    if(argv == nil) {
        sys->fprint(sys->fildes(2), "usage: agent [-cleanup] 'task description'\n");
        raise "fail:usage";
    }

    task := "";
    for(; argv != nil; argv = tl argv) {
        if(task != "")
            task += " ";
        task += hd argv;
    }

    sys->print("Task: %s\n", task);

    # Show current namespace
    sys->print("\nAvailable namespace:\n");
    showns("/n");

    # Set system prompt
    if(setsystem(SYSTEM_PROMPT) < 0) {
        sys->print("Warning: could not set system prompt\n");
    }

    # Run agent loop
    runagent(task);
}

# Show namespace contents (only recurse into known tool directories)
showns(path: string)
{
    fd := sys->open(path, Sys->OREAD);
    if(fd == nil)
        return;

    sys->print("%s:\n", path);

    for(;;) {
        (n, dir) := sys->dirread(fd);
        if(n <= 0)
            break;
        for(i := 0; i < n; i++) {
            name := dir[i].name;
            if(dir[i].mode & Sys->DMDIR)
                sys->print("  %s/\n", name);
            else
                sys->print("  %s\n", name);
        }
    }
    sys->print("\n");

    # Only recurse into known tool directories (not local, dev, etc.)
    fd = sys->open(path, Sys->OREAD);
    if(fd == nil)
        return;
    for(;;) {
        (n, dir) := sys->dirread(fd);
        if(n <= 0)
            break;
        for(i := 0; i < n; i++) {
            if(dir[i].mode & Sys->DMDIR) {
                name := dir[i].name;
                # Only recurse into tool directories
                if(name == "llm" || name == "osm" || name == "xenith") {
                    subpath := path + "/" + name;
                    showns(subpath);
                }
            }
        }
    }
}

# Set the LLM system prompt
setsystem(prompt: string): int
{
    fd := sys->open(LLM_SYSTEM, Sys->OWRITE);
    if(fd == nil)
        return -1;

    data := array of byte prompt;
    n := sys->write(fd, data, len data);
    return n;
}

# Query the LLM via /n/llm/ask
query(prompt: string): string
{
    # Show feedback that we're querying
    sys->print(">>> Querying LLM...\n");

    # Write prompt to /n/llm/ask
    fd := sys->open(LLM_ASK, Sys->OWRITE);
    if(fd == nil) {
        sys->fprint(sys->fildes(2), "agent: cannot open %s: %r\n", LLM_ASK);
        return "";
    }

    data := array of byte prompt;
    if(sys->write(fd, data, len data) < 0) {
        sys->fprint(sys->fildes(2), "agent: write error: %r\n");
        return "";
    }

    sys->print(">>> Waiting for response...\n");

    # Read response from same file
    fd = sys->open(LLM_ASK, Sys->OREAD);
    if(fd == nil) {
        sys->fprint(sys->fildes(2), "agent: cannot read response: %r\n");
        return "";
    }

    buf := array[65536] of byte;
    response := "";
    while((n := sys->read(fd, buf, len buf)) > 0) {
        response += string buf[0:n];
    }

    sys->print(">>> Response received (%d bytes)\n", len response);

    return response;
}

# Build the prompt for the agent
buildprompt(task: string): string
{
    # Get full namespace listing (includes /mnt/xenith if available)
    nslist := getfullnslist();

    prompt := "Your namespace:\n" + nslist + "\n\n";
    prompt += "Task: " + task;

    return prompt;
}

# Get namespace listing as a string (recursive for known tool directories)
getnslist(path: string): string
{
    result := "";

    fd := sys->open(path, Sys->OREAD);
    if(fd == nil)
        return result;

    result += path + ":\n";

    for(;;) {
        (n, dir) := sys->dirread(fd);
        if(n <= 0)
            break;
        for(i := 0; i < n; i++) {
            name := dir[i].name;
            if(dir[i].mode & Sys->DMDIR) {
                result += "  " + name + "/\n";
                # Recurse into known tool directories
                if(name == "llm" || name == "osm" || name == "xenith")
                    result += getnslist(path + "/" + name);
            } else {
                result += "  " + name + "\n";
            }
        }
    }

    return result;
}

# Also check /mnt/xenith if available
getfullnslist(): string
{
    result := getnslist("/n");

    # Check if xenith is mounted
    fd := sys->open(XENITH_BASE, Sys->OREAD);
    if(fd != nil) {
        result += "\n" + getnslist(XENITH_BASE);
    }

    return result;
}

# ============================================================
# Xenith Window Functions
# ============================================================

# Create a new Xenith window, returns window ID or -1 on error
xenith_newwindow(): int
{
    fd := sys->open(XENITH_NEW, Sys->ORDWR);
    if(fd == nil) {
        sys->fprint(sys->fildes(2), "xenith: cannot open %s: %r\n", XENITH_NEW);
        return -1;
    }

    # Reading from new/ctl returns the window ID
    buf := array[64] of byte;
    n := sys->read(fd, buf, len buf);
    if(n <= 0) {
        sys->fprint(sys->fildes(2), "xenith: cannot read window id: %r\n");
        return -1;
    }

    idstr := trim(string buf[0:n]);
    winid := int idstr;

    # Track for cleanup
    created_windows = winid :: created_windows;

    sys->print("xenith: created window %d\n", winid);
    return winid;
}

# Write text to a Xenith window body
xenith_write(winid: int, text: string): int
{
    path := sys->sprint("%s/%d/body", XENITH_BASE, winid);
    fd := sys->open(path, Sys->OWRITE);
    if(fd == nil) {
        sys->fprint(sys->fildes(2), "xenith: cannot open %s: %r\n", path);
        return -1;
    }

    data := array of byte text;
    n := sys->write(fd, data, len data);
    if(n < 0) {
        sys->fprint(sys->fildes(2), "xenith: write error: %r\n");
        return -1;
    }

    sys->print("xenith: wrote %d bytes to window %d\n", n, winid);
    return n;
}

# Send control command to a Xenith window
xenith_ctl(winid: int, cmd: string): int
{
    path := sys->sprint("%s/%d/ctl", XENITH_BASE, winid);
    fd := sys->open(path, Sys->OWRITE);
    if(fd == nil) {
        sys->fprint(sys->fildes(2), "xenith: cannot open %s: %r\n", path);
        return -1;
    }

    data := array of byte cmd;
    n := sys->write(fd, data, len data);
    if(n < 0) {
        sys->fprint(sys->fildes(2), "xenith: ctl write error: %r\n");
        return -1;
    }

    sys->print("xenith: sent ctl '%s' to window %d\n", cmd, winid);
    return 0;
}

# Delete a Xenith window
xenith_delete(winid: int): int
{
    result := xenith_ctl(winid, "delete");

    # Remove from tracking list
    newlist: list of int;
    for(w := created_windows; w != nil; w = tl w) {
        if(hd w != winid)
            newlist = hd w :: newlist;
    }
    created_windows = newlist;

    return result;
}

# Display an image in a Xenith window
xenith_image(winid: int, imgpath: string): int
{
    # First set the window to display an image
    if(xenith_ctl(winid, "image " + imgpath) < 0)
        return -1;

    sys->print("xenith: displayed image %s in window %d\n", imgpath, winid);
    return 0;
}

# Cleanup all tracked windows (called on exit if -cleanup flag set)
xenith_cleanup()
{
    if(cleanup_windows == 0)
        return;

    sys->print("xenith: cleaning up %d windows\n", len created_windows);
    for(; created_windows != nil; created_windows = tl created_windows) {
        winid := hd created_windows;
        xenith_ctl(winid, "delete");
    }
}

# Check if path is a xenith window body
isxenithbody(path: string): int
{
    # Pattern: /mnt/xenith/<id>/body
    if(len path < 17)  # minimum: /mnt/xenith/1/body
        return 0;
    if(len path < len XENITH_BASE || path[0:len XENITH_BASE] != XENITH_BASE)
        return 0;
    if(len path > 5 && path[len path - 5:] == "/body")
        return 1;
    return 0;
}

# Extract window ID from xenith path
getwinid(path: string): int
{
    # Pattern: /mnt/xenith/<id>/...
    if(len path <= len XENITH_BASE + 1)
        return -1;
    rest := path[len XENITH_BASE + 1:];  # skip "/mnt/xenith/"
    # Find end of ID (next /)
    idend := 0;
    for(; idend < len rest; idend++) {
        if(rest[idend] == '/')
            break;
    }
    if(idend == 0)
        return -1;
    return int rest[0:idend];
}

# Check if window is a system window (should not be read by agent)
issystemwindow(winid: int): int
{
    # Read the window tag
    tagpath := sys->sprint("%s/%d/tag", XENITH_BASE, winid);
    fd := sys->open(tagpath, Sys->OREAD);
    if(fd == nil)
        return 0;  # Can't read tag, allow access

    buf := array[512] of byte;
    n := sys->read(fd, buf, len buf);
    if(n <= 0)
        return 0;

    tag := string buf[0:n];

    # System windows start with + (like +Errors, +Scratch)
    if(len tag > 0 && tag[0] == '+')
        return 1;

    return 0;
}

# Resolve window ID reference: $ = last, $1 = first created, $2 = second, etc.
# Returns -1 if reference is invalid
resolvewinref(winref: string): int
{
    if(winref == "$" || winref == "$0") {
        # Last created window
        if(created_windows == nil)
            return -1;
        return hd created_windows;  # List is most-recent-first
    }

    if(len winref > 1 && winref[0] == '$') {
        # $N = Nth created window (1-indexed)
        n := int winref[1:];
        if(n < 1)
            return -1;
        # created_windows is most-recent-first, so we need to reverse index
        count := listlen(created_windows);
        if(n > count)
            return -1;
        # Walk to the (count - n)th element (0-indexed from head)
        idx := count - n;
        w := created_windows;
        for(i := 0; i < idx && w != nil; i++)
            w = tl w;
        if(w == nil)
            return -1;
        return hd w;
    }

    # Not a reference, try to parse as integer
    return int winref;
}

# Expand window references in a path like /mnt/xenith/$1/body
expandwinrefs(path: string): string
{
    # Look for $ followed by digits or just $
    result := "";
    i := 0;
    while(i < len path) {
        if(path[i] == '$') {
            # Find end of reference
            j := i + 1;
            while(j < len path && path[j] >= '0' && path[j] <= '9')
                j++;
            winref := path[i:j];
            winid := resolvewinref(winref);
            if(winid >= 0)
                result += string winid;
            else
                result += winref;  # Keep original if invalid
            i = j;
        } else {
            result += path[i:i+1];
            i++;
        }
    }
    return result;
}

# ============================================================
# Shell Command Execution
# ============================================================

# Channel for collecting command output from spawned process
cmdresult: chan of (int, string);

# Execute a shell command and capture output
# Returns (exit_status, output)
execshell_capture(cmd: string): (int, string)
{
    sys->print("shell: executing: %s\n", cmd);

    # Create a pipe for output capture
    fds := array[2] of ref Sys->FD;
    if(sys->pipe(fds) < 0) {
        return (-1, "cannot create pipe");
    }

    # Parse the command to extract the program and arguments
    (argc, argv) := sys->tokenize(cmd, " \t");
    if(argc == 0 || argv == nil) {
        return (-1, "empty command");
    }

    progname := hd argv;

    # Try to load the command as a dis module
    dispath := progname;
    if(len dispath < 4 || dispath[len dispath - 4:] != ".dis")
        dispath += ".dis";

    # Try loading from current path first, then /dis
    prog := load DisModule dispath;
    if(prog == nil)
        prog = load DisModule "/dis/" + dispath;
    if(prog == nil) {
        # Try without adding .dis if it was already there
        prog = load DisModule progname;
        if(prog == nil)
            prog = load DisModule "/dis/" + progname;
    }

    if(prog == nil) {
        return (-1, "cannot load command: " + progname);
    }

    # Spawn the command with stdout redirected to our pipe
    cmdresult = chan of (int, string);
    spawn runcmd(prog, argv, fds[1], cmdresult);
    fds[1] = nil;  # Close write end in parent

    # Read output from pipe
    output := "";
    buf := array[4096] of byte;
    while((n := sys->read(fds[0], buf, len buf)) > 0) {
        output += string buf[0:n];
    }
    fds[0] = nil;

    # Wait for command completion
    (status, errmsg) := <-cmdresult;

    if(status < 0)
        return (-1, errmsg);

    return (0, output);
}

# Run a command with redirected stdout
runcmd(prog: DisModule, argv: list of string, outfd: ref Sys->FD, result: chan of (int, string))
{
    # Redirect stdout to our pipe
    sys->dup(outfd.fd, 1);
    sys->dup(outfd.fd, 2);  # Also capture stderr
    outfd = nil;

    # Run the command
    {
        prog->init(nil, argv);
        result <-= (0, "");
    } exception e {
        "fail:*" =>
            result <-= (-1, e[5:]);
        "*" =>
            result <-= (-1, e);
    }
}

# Run the agent loop - Claude Code pattern
# llm9p maintains conversation context, we just query and execute
runagent(task: string)
{
    iterations := 0;
    consecutive_errors := 0;

    sys->print("\n=== Agent v5 Starting (max %d steps) ===\n", MAX_ITERATIONS);
    sys->print("Using llm9p context for conversation memory\n");

    # First prompt includes namespace and task
    prompt := buildprompt(task);
    sys->print("Initial prompt:\n%s\n", prompt);

    while(iterations < MAX_ITERATIONS && consecutive_errors < MAX_ERRORS) {
        iterations++;
        sys->print("\n=== Step %d/%d ===\n", iterations, MAX_ITERATIONS);

        # Query LLM with retry logic for transient errors
        (response, err) := retryquery(prompt);

        # Check for fatal error
        if(err != "" && len err > 7 && err[0:7] == "fatal: ") {
            sys->fprint(sys->fildes(2), "agent: %s\n", err);
            break;
        }

        # Check for error
        if(response == "" || err != "") {
            sys->fprint(sys->fildes(2), "agent: LLM error (error %d/%d): %s\n",
                consecutive_errors+1, MAX_ERRORS, err);
            consecutive_errors++;
            prompt = "error: " + err;
            continue;
        }

        sys->print("\n=== LLM Response ===\n%s\n", response);

        # Check for completion BEFORE executing
        if(hascompletion(response)) {
            sys->print("\n=== Task Complete ===\n");
            break;
        }

        # Execute ONLY the first command
        sys->print("\n=== Executing Command ===\n");
        (nerr, action, nil) := executefirstcommand(response);

        # Build next prompt from result - llm9p remembers the conversation
        if(action != nil) {
            if(action.outcome == "OK") {
                prompt = "Result: " + action.detail;
            } else {
                prompt = "ERROR: " + action.detail;
            }
            sys->print("Next prompt: %s\n", prompt);
        } else {
            prompt = "No command found in your response. Output ONE command.";
        }

        if(nerr > 0) {
            consecutive_errors++;
            sys->print("Error (consecutive: %d/%d)\n", consecutive_errors, MAX_ERRORS);
        } else {
            consecutive_errors = 0;
        }
    }

    if(iterations >= MAX_ITERATIONS)
        sys->print("\n=== Safety Limit Reached (%d steps) ===\n", MAX_ITERATIONS);
    if(consecutive_errors >= MAX_ERRORS)
        sys->print("\n=== Too Many Errors (%d consecutive) ===\n", consecutive_errors);

    # Cleanup windows if -cleanup flag was set
    xenith_cleanup();

    # Write result if running as subagent
    success := consecutive_errors < MAX_ERRORS && iterations < MAX_ITERATIONS;
    writeresult("Agent completed task", success);

    sys->print("\n=== Agent Complete ===\n");
}

# Build summary from history and facts
buildsummary(history: list of ref Action, facts: list of string): string
{
    s := "=== Context from previous actions ===\n";

    # Add facts first
    if(facts != nil) {
        s += "Facts:\n";
        for(f := facts; f != nil; f = tl f)
            s += "  " + hd f + "\n";
    }

    # Collect errors with counts
    errcounts: list of (string, int);
    for(h := history; h != nil; h = tl h) {
        a := hd h;
        if(a.outcome == "ERR") {
            key := a.path + ": " + a.detail;
            errcounts = inccount(errcounts, key);
        }
    }

    if(errcounts != nil) {
        s += "Errors:\n";
        for(; errcounts != nil; errcounts = tl errcounts) {
            (key, count) := hd errcounts;
            s += sys->sprint("  %s (x%d)\n", key, count);
        }
    }

    # Show last few actions
    s += "Recent actions:\n";
    actioncount := 0;
    for(ha := history; ha != nil && actioncount < 5; ha = tl ha) {
        a := hd ha;
        s += sys->sprint("  %s %s → %s", a.cmd, a.path, a.outcome);
        if(a.outcome == "OK" && a.cmd == "ls" && len a.detail < 80)
            s += " (" + a.detail + ")";
        s += "\n";
        actioncount++;
    }

    return s;
}

# Increment count for key in list
inccount(counts: list of (string, int), key: string): list of (string, int)
{
    result: list of (string, int);
    found := 0;
    for(; counts != nil; counts = tl counts) {
        (k, c) := hd counts;
        if(k == key) {
            result = (k, c+1) :: result;
            found = 1;
        } else {
            result = (k, c) :: result;
        }
    }
    if(!found)
        result = (key, 1) :: result;
    return result;
}

# Trim history to max length
trimhistory(history: list of ref Action, max: int): list of ref Action
{
    if(max <= 0)
        return nil;
    count := 0;
    result: list of ref Action;
    for(; history != nil && count < max; history = tl history) {
        result = hd history :: result;
        count++;
    }
    # Reverse to maintain order
    rev: list of ref Action;
    for(; result != nil; result = tl result)
        rev = hd result :: rev;
    return rev;
}

# Add action to history
addaction(history: list of ref Action, cmd, path, outcome, detail: string): list of ref Action
{
    a := ref Action(cmd, path, outcome, detail);
    return a :: history;
}

# Check if response indicates task completion
hascompletion(response: string): int
{
    # Only complete when the LLM explicitly says DONE
    lines := splitlines(response);
    for(; lines != nil; lines = tl lines) {
        line := trim(hd lines);
        if(line == "DONE" || line == "done")
            return 1;
    }

    # Also check if the raw response starts with DONE (no other lines)
    trimmed := trim(response);
    if(trimmed == "DONE" || trimmed == "done")
        return 1;

    return 0;
}

# Execute ONLY THE FIRST command from the LLM response
# This is the Claude Code pattern: execute one, observe, decide next
# Returns (error_count, action, fact)
executefirstcommand(response: string): (int, ref Action, string)
{
    lines := splitlines(response);
    for(; lines != nil; lines = tl lines) {
        line := hd lines;
        line = trim(line);

        # Skip empty lines and comments
        if(line == "" || line[0] == '#')
            continue;

        # Skip markdown code block markers
        if(len line >= 3 && line[0:3] == "```")
            continue;

        # Skip "DONE" completion marker
        if(line == "DONE" || line == "done")
            continue;

        # Skip lines that look like explanatory text (not commands)
        if(isexplanation(line))
            continue;

        # Found a command line - execute just the first command
        # (if there are && chains, only do the first part)
        cmds := splitand(line);
        if(cmds == nil)
            continue;

        cmd := trim(hd cmds);
        if(cmd == "")
            continue;

        # Execute this one command and return immediately
        (ok, path, detail, fact) := execcmd_v2(cmd);
        if(ok < 0) {
            return (1, ref Action(getcmdtype(cmd), path, "ERR", detail), fact);
        }
        return (0, ref Action(getcmdtype(cmd), path, "OK", detail), fact);
    }

    # No command found
    return (0, nil, "");
}

# Execute shell commands from the LLM response (batch mode - deprecated)
# Returns (error_count, actions, facts)
executecommands_v2(response: string): (int, list of ref Action, list of string)
{
    errors := 0;
    actions: list of ref Action;
    facts: list of string;

    lines := splitlines(response);
    for(; lines != nil; lines = tl lines) {
        line := hd lines;
        line = trim(line);

        # Skip empty lines and comments
        if(line == "" || line[0] == '#')
            continue;

        # Skip markdown code block markers
        if(len line >= 3 && line[0:3] == "```")
            continue;

        # Skip "DONE" completion marker
        if(line == "DONE" || line == "done")
            continue;

        # Skip lines that look like explanatory text (not commands)
        if(isexplanation(line))
            continue;

        # Split on && and execute each part
        cmds := splitand(line);
        for(; cmds != nil; cmds = tl cmds) {
            cmd := trim(hd cmds);
            if(cmd == "")
                continue;

            (ok, path, detail, fact) := execcmd_v2(cmd);
            if(ok < 0) {
                errors++;
                actions = ref Action(getcmdtype(cmd), path, "ERR", detail) :: actions;
            } else {
                actions = ref Action(getcmdtype(cmd), path, "OK", detail) :: actions;
            }

            # Add any discovered facts
            if(fact != "")
                facts = fact :: facts;
        }
    }

    # Reverse actions to maintain order
    rev: list of ref Action;
    for(; actions != nil; actions = tl actions)
        rev = hd actions :: rev;

    return (errors, rev, facts);
}

# Check if a line looks like explanatory text rather than a command
isexplanation(line: string): int
{
    if(len line == 0)
        return 0;

    # Commands start with: echo, cat, ls, xenith, or a path
    # Explanations start with capital letters and contain spaces

    # Check for known command prefixes
    if(len line >= 5 && line[0:5] == "echo ")
        return 0;
    if(len line >= 4 && line[0:4] == "cat ")
        return 0;
    if(len line >= 3 && line[0:3] == "ls ")
        return 0;
    if(len line >= 2 && line[0:2] == "ls")
        return 0;
    if(len line >= 7 && line[0:7] == "xenith ")
        return 0;
    if(len line >= 6 && line[0:6] == "xenith")
        return 0;
    if(line[0] == '/')
        return 0;  # Path
    if(line[0] == '.')
        return 0;  # Relative path

    # If starts with capital letter and contains spaces, likely explanation
    if(line[0] >= 'A' && line[0] <= 'Z') {
        for(i := 0; i < len line; i++) {
            if(line[i] == ' ')
                return 1;  # Capital + space = explanation
        }
    }

    # If contains common explanation words, skip
    if(contains(line, "need to") || contains(line, "Let me") ||
       contains(line, "Looking at") || contains(line, "Based on") ||
       contains(line, "Wait,") || contains(line, "This will"))
        return 1;

    return 0;
}

# Check if string contains substring
contains(s, sub: string): int
{
    if(len sub > len s)
        return 0;
    for(i := 0; i <= len s - len sub; i++) {
        match := 1;
        for(j := 0; j < len sub; j++) {
            if(s[i+j] != sub[j]) {
                match = 0;
                break;
            }
        }
        if(match)
            return 1;
    }
    return 0;
}

# Get command type from command string
getcmdtype(cmd: string): string
{
    if(len cmd > 5 && cmd[0:5] == "echo ")
        return "write";
    if(len cmd > 4 && cmd[0:4] == "cat ")
        return "read";
    if(len cmd >= 2 && cmd[0:2] == "ls")
        return "ls";
    if(len cmd >= 6 && cmd[0:6] == "xenith")
        return "xenith";
    return "shell";
}

# Execute a single command
# Returns (0, "") on success, (-1, "error message") on error
execcmd(cmd: string): (int, string)
{
    cmd = trim(cmd);
    if(cmd == "")
        return (0, "");

    # Execute echo > file commands
    if(len cmd > 5 && cmd[0:5] == "echo ")
        return exececho(cmd);

    # Execute cat commands
    if(len cmd > 4 && cmd[0:4] == "cat ")
        return execcat(cmd[4:]);

    # Execute ls commands
    if(len cmd >= 2 && cmd[0:2] == "ls")
        return execls(cmd);

    sys->print("Skipping unknown command: %s\n", cmd);
    return (0, "");
}

# Execute a single command (v2)
# Returns (ok, path, detail, fact)
# fact is empty unless we learned something (e.g., from ls)
execcmd_v2(cmd: string): (int, string, string, string)
{
    cmd = trim(cmd);
    if(cmd == "")
        return (0, "", "", "");

    # Execute echo > file commands
    if(len cmd > 5 && cmd[0:5] == "echo ")
        return exececho_v2(cmd);

    # Execute cat commands
    if(len cmd > 4 && cmd[0:4] == "cat ")
        return execcat_v2(cmd[4:]);

    # Execute ls commands
    if(len cmd >= 2 && cmd[0:2] == "ls")
        return execls_v2(cmd);

    # Execute xenith commands
    if(len cmd >= 6 && cmd[0:6] == "xenith")
        return execxenith_v2(cmd);

    # Fallback: try to execute as shell command
    return execshell_v2(cmd);
}

# Execute xenith commands
# Syntax: xenith <subcmd> [args...]
# Returns (ok, path, detail, fact)
execxenith_v2(cmd: string): (int, string, string, string)
{
    # Parse: xenith <subcmd> [args...]
    args := splitwords(cmd);
    if(len args < 2) {
        return (-1, "xenith", "missing subcommand", "");
    }

    # Skip "xenith" prefix
    args = tl args;
    subcmd := hd args;
    args = tl args;

    sys->print("Exec: xenith %s\n", subcmd);

    case subcmd {
    "new" =>
        winid := xenith_newwindow();
        if(winid < 0)
            return (-1, "xenith/new", "failed to create window", "");
        return (0, "xenith/new", sys->sprint("window %d", winid), "");

    "write" =>
        if(len args < 2) {
            return (-1, "xenith/write", "usage: xenith write <id|$|$N> <text>", "");
        }
        winid := resolvewinref(hd args);
        if(winid < 0)
            return (-1, "xenith/write", "invalid window reference: " + hd args, "");
        args = tl args;
        # Join remaining args as text
        text := "";
        for(; args != nil; args = tl args) {
            if(text != "")
                text += " ";
            text += hd args;
        }
        # Strip surrounding quotes if present
        text = stripquotes(text);
        if(xenith_write(winid, text) < 0)
            return (-1, sys->sprint("xenith/%d/write", winid), "write failed", "");
        return (0, sys->sprint("xenith/%d/write", winid), sys->sprint("%d bytes", len text), "");

    "delete" =>
        if(args == nil) {
            return (-1, "xenith/delete", "usage: xenith delete <id|$|$N>", "");
        }
        winid := resolvewinref(hd args);
        if(winid < 0)
            return (-1, "xenith/delete", "invalid window reference: " + hd args, "");
        if(xenith_delete(winid) < 0)
            return (-1, sys->sprint("xenith/%d/delete", winid), "delete failed", "");
        return (0, sys->sprint("xenith/%d/delete", winid), "deleted", "");

    "image" =>
        if(len args < 2) {
            return (-1, "xenith/image", "usage: xenith image <id|$|$N> <path>", "");
        }
        winid := resolvewinref(hd args);
        if(winid < 0)
            return (-1, "xenith/image", "invalid window reference: " + hd args, "");
        imgpath := hd (tl args);
        if(xenith_image(winid, imgpath) < 0)
            return (-1, sys->sprint("xenith/%d/image", winid), "image display failed", "");
        return (0, sys->sprint("xenith/%d/image", winid), imgpath, "");

    "ctl" =>
        if(len args < 2) {
            return (-1, "xenith/ctl", "usage: xenith ctl <id|$|$N> <command>", "");
        }
        winid := resolvewinref(hd args);
        if(winid < 0)
            return (-1, "xenith/ctl", "invalid window reference: " + hd args, "");
        args = tl args;
        ctlcmd := "";
        for(; args != nil; args = tl args) {
            if(ctlcmd != "")
                ctlcmd += " ";
            ctlcmd += hd args;
        }
        if(xenith_ctl(winid, ctlcmd) < 0)
            return (-1, sys->sprint("xenith/%d/ctl", winid), "ctl failed", "");
        return (0, sys->sprint("xenith/%d/ctl", winid), ctlcmd, "");

    * =>
        return (-1, "xenith", "unknown subcommand: " + subcmd, "");
    }
}

# Execute arbitrary shell command via fallback
# Returns (ok, path, detail, fact)
execshell_v2(cmd: string): (int, string, string, string)
{
    sys->print("Exec (shell): %s\n", cmd);

    (ok, output) := execshell_capture(cmd);
    if(ok < 0) {
        sys->fprint(sys->fildes(2), "  Error: %s\n", output);
        return (-1, cmd, output, "");
    }

    if(len output > 100)
        output = output[0:100] + "...";

    sys->print("  Output: %s\n", output);
    return (0, cmd, output, "");
}

# Execute ls command (v2)
# Returns (ok, path, listing, fact)
execls_v2(cmd: string): (int, string, string, string)
{
    path := "/n";
    if(len cmd > 3)
        path = trim(cmd[3:]);

    sys->print("Exec: ls %s\n", path);

    fd := sys->open(path, Sys->OREAD);
    if(fd == nil) {
        errmsg := "file does not exist";
        sys->fprint(sys->fildes(2), "  Error: %s\n", errmsg);
        return (-1, path, errmsg, "");
    }

    listing := "";
    for(;;) {
        (n, dir) := sys->dirread(fd);
        if(n <= 0)
            break;
        for(i := 0; i < n; i++) {
            name := dir[i].name;
            if(listing != "")
                listing += " ";
            listing += name;
            if(dir[i].mode & Sys->DMDIR)
                sys->print("  %s/\n", name);
            else
                sys->print("  %s\n", name);
        }
    }

    # Generate fact if this looks like a tool directory
    fact := "";
    if(hasfile(listing, "result") && !hasfile(listing, "query")) {
        fact = path + " is Type B (use param files + result)";
    } else if(hasfile(listing, "query")) {
        fact = path + " is Type A (use query file)";
    }

    return (0, path, listing, fact);
}

# Check if listing contains a file name
hasfile(listing, name: string): int
{
    words := splitwords(listing);
    for(; words != nil; words = tl words) {
        if(hd words == name)
            return 1;
    }
    return 0;
}

# Split string on whitespace
splitwords(s: string): list of string
{
    words: list of string;
    start := -1;
    for(i := 0; i <= len s; i++) {
        if(i == len s || s[i] == ' ' || s[i] == '\t' || s[i] == '\n') {
            if(start >= 0 && i > start) {
                words = s[start:i] :: words;
            }
            start = -1;
        } else if(start < 0) {
            start = i;
        }
    }
    # Reverse
    rev: list of string;
    for(; words != nil; words = tl words)
        rev = hd words :: rev;
    return rev;
}

# Execute echo command (v2)
# Returns (ok, path, detail, fact)
exececho_v2(cmd: string): (int, string, string, string)
{
    sys->print("Exec: %s\n", cmd);

    # Find the > or >>
    redir := -1;
    for(i := 0; i < len cmd; i++) {
        if(cmd[i] == '>') {
            redir = i;
            break;
        }
    }

    if(redir < 0) {
        sys->print("  (no redirection found)\n");
        return (-1, cmd, "missing redirection", "");
    }

    # Extract data and file
    data := cmd[5:redir];  # after "echo "
    file := cmd[redir+1:];

    # Clean up data (remove quotes)
    data = trim(data);
    if(len data >= 2) {
        if((data[0] == '"' && data[len data - 1] == '"') ||
           (data[0] == '\'' && data[len data - 1] == '\'')) {
            data = data[1:len data - 1];
        }
    }

    # Clean up file
    file = trim(file);
    if(len file > 0 && file[0] == '>')  # handle >>
        file = file[1:];
    file = trim(file);

    # Expand window references like $, $1, $2 in path
    file = expandwinrefs(file);

    sys->print("  Writing '%s' to %s\n", data, file);

    # Write to file
    fd := sys->open(file, Sys->OWRITE);
    if(fd == nil) {
        errmsg := "file does not exist";
        sys->fprint(sys->fildes(2), "  Error: %s\n", errmsg);
        return (-1, file, errmsg, "");
    }

    bytes := array of byte data;
    n := sys->write(fd, bytes, len bytes);
    if(n < 0) {
        errmsg := "write failed";
        sys->fprint(sys->fildes(2), "  Error: %s\n", errmsg);
        return (-1, file, errmsg, "");
    }
    sys->print("  Wrote %d bytes\n", n);
    return (0, file, sys->sprint("%d bytes", n), "");
}

# Execute cat command (v2)
# Returns (ok, path, content, fact)
execcat_v2(file: string): (int, string, string, string)
{
    file = trim(file);

    # Expand window references like $, $1, $2 in path
    file = expandwinrefs(file);

    sys->print("Exec: cat %s\n", file);

    # Check if this is a xenith window body - if so, check for system windows
    if(isxenithbody(file)) {
        winid := getwinid(file);
        if(winid >= 0 && issystemwindow(winid)) {
            errmsg := "cannot read system window";
            sys->fprint(sys->fildes(2), "  Skipped: %s\n", errmsg);
            return (-1, file, errmsg, "");
        }
    }

    fd := sys->open(file, Sys->OREAD);
    if(fd == nil) {
        errmsg := "file does not exist";
        sys->fprint(sys->fildes(2), "  Error: %s\n", errmsg);
        return (-1, file, errmsg, "");
    }

    buf := array[8192] of byte;
    content := "";
    sys->print("  Result:\n");
    while((n := sys->read(fd, buf, len buf)) > 0) {
        content += string buf[0:n];
        sys->print("%s", string buf[0:n]);
    }
    sys->print("\n");

    # Truncate for history
    if(len content > 100)
        content = content[0:100] + "...";

    return (0, file, content, "");
}

# Execute ls command
execls(cmd: string): (int, string)
{
    path := "/n";
    if(len cmd > 3)
        path = trim(cmd[3:]);

    sys->print("Exec: ls %s\n", path);

    fd := sys->open(path, Sys->OREAD);
    if(fd == nil) {
        errmsg := sys->sprint("ls %s: file does not exist", path);
        sys->fprint(sys->fildes(2), "  Error: %s\n", errmsg);
        return (-1, errmsg);
    }

    for(;;) {
        (n, dir) := sys->dirread(fd);
        if(n <= 0)
            break;
        for(i := 0; i < n; i++) {
            name := dir[i].name;
            if(dir[i].mode & Sys->DMDIR)
                sys->print("  %s/\n", name);
            else
                sys->print("  %s\n", name);
        }
    }
    return (0, "");
}

# Split a command line on &&
splitand(s: string): list of string
{
    result: list of string;
    start := 0;
    i := 0;
    while(i < len s) {
        if(i+1 < len s && s[i] == '&' && s[i+1] == '&') {
            if(i > start)
                result = s[start:i] :: result;
            start = i + 2;
            i = start;
        } else {
            i++;
        }
    }
    if(start < len s)
        result = s[start:] :: result;

    # Reverse the list
    rev: list of string;
    for(; result != nil; result = tl result)
        rev = hd result :: rev;
    return rev;
}

# Execute an echo command (echo "data" > file)
# Returns (0, "") on success, (-1, "error message") on error
exececho(cmd: string): (int, string)
{
    # Very simple parser for: echo "data" > file
    # or: echo 'data' > file
    sys->print("Exec: %s\n", cmd);

    # Find the > or >>
    redir := -1;
    for(i := 0; i < len cmd; i++) {
        if(cmd[i] == '>') {
            redir = i;
            break;
        }
    }

    if(redir < 0) {
        sys->print("  (no redirection found)\n");
        return (-1, "echo command missing redirection (>)");
    }

    # Extract data and file
    data := cmd[5:redir];  # after "echo "
    file := cmd[redir+1:];

    # Clean up data (remove quotes)
    data = trim(data);
    if(len data >= 2) {
        if((data[0] == '"' && data[len data - 1] == '"') ||
           (data[0] == '\'' && data[len data - 1] == '\'')) {
            data = data[1:len data - 1];
        }
    }

    # Clean up file
    file = trim(file);
    if(len file > 0 && file[0] == '>')  # handle >>
        file = file[1:];
    file = trim(file);

    sys->print("  Writing '%s' to %s\n", data, file);

    # Write to file
    fd := sys->open(file, Sys->OWRITE);
    if(fd == nil) {
        errmsg := sys->sprint("cannot write to %s: file does not exist", file);
        sys->fprint(sys->fildes(2), "  Error: %s\n", errmsg);
        return (-1, errmsg);
    }

    bytes := array of byte data;
    n := sys->write(fd, bytes, len bytes);
    if(n < 0) {
        errmsg := sys->sprint("write to %s failed", file);
        sys->fprint(sys->fildes(2), "  Error: %s\n", errmsg);
        return (-1, errmsg);
    }
    sys->print("  Wrote %d bytes\n", n);
    return (0, "");
}

# Execute a cat command
# Returns (0, "") on success, (-1, "error message") on error
execcat(file: string): (int, string)
{
    file = trim(file);

    sys->print("Exec: cat %s\n", file);

    fd := sys->open(file, Sys->OREAD);
    if(fd == nil) {
        errmsg := sys->sprint("cat %s: file does not exist", file);
        sys->fprint(sys->fildes(2), "  Error: %s\n", errmsg);
        return (-1, errmsg);
    }

    buf := array[8192] of byte;
    sys->print("  Result:\n");
    while((n := sys->read(fd, buf, len buf)) > 0) {
        sys->print("%s", string buf[0:n]);
    }
    sys->print("\n");
    return (0, "");
}

# Helper: split string into lines
splitlines(s: string): list of string
{
    lines: list of string;
    start := 0;
    for(i := 0; i < len s; i++) {
        if(s[i] == '\n') {
            if(i > start)
                lines = s[start:i] :: lines;
            start = i + 1;
        }
    }
    if(start < len s)
        lines = s[start:] :: lines;

    # Reverse the list
    result: list of string;
    for(; lines != nil; lines = tl lines)
        result = hd lines :: result;
    return result;
}

# Helper: trim whitespace
trim(s: string): string
{
    # Trim leading whitespace
    start := 0;
    for(; start < len s; start++) {
        c := s[start];
        if(c != ' ' && c != '\t' && c != '\n' && c != '\r')
            break;
    }

    # Trim trailing whitespace
    end := len s;
    for(; end > start; end--) {
        c := s[end-1];
        if(c != ' ' && c != '\t' && c != '\n' && c != '\r')
            break;
    }

    if(start >= end)
        return "";
    return s[start:end];
}

# Helper: strip surrounding quotes from string
stripquotes(s: string): string
{
    if(len s < 2)
        return s;
    if((s[0] == '"' && s[len s - 1] == '"') ||
       (s[0] == '\'' && s[len s - 1] == '\'')) {
        return s[1:len s - 1];
    }
    return s;
}

# ============================================================
# Error Classification and Retry Logic
# ============================================================

# Check if error is transient (should retry)
# Transient: rate limit, timeout, connection refused, temporary failures
istransient(err: string): int
{
    if(contains(err, "rate limit") || contains(err, "rate_limit") ||
       contains(err, "timeout") || contains(err, "timed out") ||
       contains(err, "connection refused") || contains(err, "connection reset") ||
       contains(err, "temporarily unavailable") || contains(err, "try again") ||
       contains(err, "overloaded") || contains(err, "503") || contains(err, "529") ||
       contains(err, "ECONNREFUSED") || contains(err, "ETIMEDOUT"))
        return 1;
    return 0;
}

# Check if error is fatal (should stop agent)
# Fatal: namespace errors, llm9p not mounted, critical system failures
isfatal(err: string): int
{
    if(contains(err, "cannot open /n/llm") || contains(err, "not mounted") ||
       contains(err, "namespace") || contains(err, "permission denied on /n/llm") ||
       contains(err, "authentication failed") || contains(err, "invalid API key"))
        return 1;
    return 0;
}

# Query LLM with retry logic for transient errors
# Returns (response, error) - error is empty on success
retryquery(prompt: string): (string, string)
{
    lasterr := "";

    for(attempt := 0; attempt < MAX_RETRIES; attempt++) {
        response := query(prompt);

        # Check for error in response
        if(response == "") {
            lasterr = "empty response";
        } else if(len response > 7 && response[0:7] == "Error: ") {
            lasterr = response[7:];
        } else {
            # Success
            return (response, "");
        }

        # Check error type
        if(isfatal(lasterr)) {
            sys->fprint(sys->fildes(2), "agent: fatal error: %s\n", lasterr);
            return ("", "fatal: " + lasterr);
        }

        if(istransient(lasterr) == 0) {
            # Permanent error - don't retry
            return ("", lasterr);
        }

        # Transient error - retry with backoff
        if(attempt < MAX_RETRIES - 1) {
            delay := getbackoff(attempt);
            sys->fprint(sys->fildes(2), "agent: transient error, retrying in %dms: %s\n",
                delay, lasterr);
            sys->sleep(delay);
        }
    }

    return ("", "max retries exceeded: " + lasterr);
}

# Get backoff delay for retry attempt
getbackoff(attempt: int): int
{
    if(attempt == 0)
        return BACKOFF_1;
    if(attempt == 1)
        return BACKOFF_2;
    return BACKOFF_3;
}

# ============================================================
# Subagent Result Reporting
# ============================================================

# Result directory for subagent mode
RESULT_BASE := "/tmp/agent";

# Write result when running as a subagent
# Checks for result directory and writes status/output
writeresult(output: string, success: int)
{
    # Get our PID
    pid := sys->pctl(0, nil);
    resultdir := sys->sprint("%s/%d", RESULT_BASE, pid);

    # Check if result directory exists (indicates we're a subagent)
    fd := sys->open(resultdir + "/status", Sys->OWRITE);
    if(fd == nil)
        return;  # Not running as subagent

    # Write status
    status := "completed";
    if(!success)
        status = "error";

    statusdata := array of byte status;
    sys->write(fd, statusdata, len statusdata);
    fd = nil;

    # Write output
    outputpath := resultdir + "/output";
    fd = sys->open(outputpath, Sys->OWRITE);
    if(fd != nil) {
        data := array of byte output;
        sys->write(fd, data, len data);
    }
}

# Check if running as subagent
issubagent(): int
{
    pid := sys->pctl(0, nil);
    resultdir := sys->sprint("%s/%d", RESULT_BASE, pid);
    fd := sys->open(resultdir + "/status", Sys->OREAD);
    if(fd != nil)
        return 1;
    return 0;
}
