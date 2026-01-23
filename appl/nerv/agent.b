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

# Cleanup flag (set via -cleanup command-line option)
cleanup_windows := 0;

# Track windows created during execution for cleanup
created_windows: list of int;

# Safety limits
MAX_ITERATIONS := 5;       # Maximum iterations before forced stop
MAX_ERRORS := 3;           # Maximum consecutive errors before stop
MAX_HISTORY := 10;         # Maximum actions to remember

# Action record for history
Action: adt {
    cmd:     string;   # Command that was run
    path:    string;   # Primary path involved
    outcome: string;   # "OK" or "ERR"
    detail:  string;   # Result or error message
};

# System prompt that teaches the agent about its namespace
SYSTEM_PROMPT := "You are an agent running inside Inferno OS with a namespace-bounded sandbox. " +
    "Your capabilities are determined entirely by what files are mounted in your namespace.\n\n" +
    "== Namespace Model ==\n" +
    "Everything is a file. Tools, services, and devices appear as files you can read/write.\n" +
    "Your capabilities are bounded by mounts - you can only access what's been mounted for you.\n\n" +
    "== LLM Interaction ==\n" +
    "To query the LLM: echo 'prompt' > /n/llm/ask && cat /n/llm/ask\n" +
    "To set system context: echo 'context' > /n/llm/system\n" +
    "To start new conversation: echo '' > /n/llm/new\n\n" +
    "== Xenith UI (if /mnt/xenith is mounted) ==\n" +
    "You can create, write to, delete, and change colors of windows.\n" +
    "You CANNOT move, resize, or arrange windows - user does this with mouse.\n" +
    "If asked to arrange/position/resize windows, say DONE and explain this limitation.\n\n" +
    "Window commands:\n" +
    "  xenith new - create window, returns ID\n" +
    "  xenith write <id> <text> - write text to window body\n" +
    "  xenith delete <id> - delete window\n\n" +
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
    "== Available Commands ==\n" +
    "Built-in: echo, cat, ls, xenith\n" +
    "Only use commands you know exist. Do NOT invent commands.\n\n" +
    "== Instructions ==\n" +
    "Respond with ONLY shell commands. No explanations or commentary.\n" +
    "If a task cannot be done with available commands, say DONE and explain why.\n" +
    "When task is complete, respond with 'DONE' on its own line.";

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

# ============================================================
# Shell Command Execution
# ============================================================

# Execute a shell command and capture output
# Returns (exit_status, output)
execshell_capture(cmd: string): (int, string)
{
    if(sh == nil) {
        return (-1, "shell module not loaded");
    }

    # Create temp file for output
    pid := sys->pctl(0, nil);
    tmpfile := "/tmp/agent_out." + string pid;

    # Build command with output redirection
    fullcmd := cmd + " > " + tmpfile + " 2>&1";

    sys->print("shell: executing: %s\n", cmd);

    # Execute via shell
    # Note: sh->system() needs a context, which we don't have here
    # Use a simpler approach: write to a temp script and run it
    scriptfile := "/tmp/agent_script." + string pid;
    fd := sys->create(scriptfile, Sys->OWRITE, 8r755);
    if(fd == nil) {
        return (-1, "cannot create script file");
    }
    sys->fprint(fd, "#!/dis/sh\n%s\n", fullcmd);
    fd = nil;

    # Run the script via spawn
    (ok, nil) := sys->stat(scriptfile);
    if(ok < 0) {
        return (-1, "script file not created");
    }

    # We need to run the shell command. The simplest way in Limbo
    # is to spawn /dis/sh with the script.
    # For now, return a stub - the execcmd_v2 function will use
    # direct file operations for most commands.
    sys->remove(scriptfile);

    return (-1, "shell execution not yet implemented - use built-in commands");
}

# Run the agent loop with safety limits
runagent(task: string)
{
    iterations := 0;
    consecutive_errors := 0;
    history: list of ref Action;  # Action history (most recent first)
    facts: list of string;        # Learned facts

    sys->print("\n=== Agent v3 Starting (max %d iterations, with memory) ===\n", MAX_ITERATIONS);

    while(iterations < MAX_ITERATIONS && consecutive_errors < MAX_ERRORS) {
        iterations++;
        sys->print("\n=== Iteration %d/%d ===\n", iterations, MAX_ITERATIONS);

        # Build prompt with namespace and task
        prompt := buildprompt(task);

        # Always include summary if we have history
        if(history != nil || facts != nil) {
            summary := buildsummary(history, facts);
            prompt += "\n\n" + summary;
        }

        if(iterations == 1)
            sys->print("Prompt:\n%s\n", prompt);

        response := query(prompt);
        if(response == "") {
            sys->fprint(sys->fildes(2), "agent: no response from LLM (error %d/%d)\n",
                consecutive_errors+1, MAX_ERRORS);
            consecutive_errors++;
            history = addaction(history, "query", LLM_ASK, "ERR", "empty response");
            continue;
        }

        sys->print("\n=== LLM Response ===\n%s\n", response);

        # Execute commands and collect actions
        sys->print("\n=== Executing Commands ===\n");
        (nerrs, actions, newfacts) := executecommands_v2(response);

        # Add actions to history
        for(; actions != nil; actions = tl actions)
            history = hd actions :: history;

        # Add any new facts
        for(; newfacts != nil; newfacts = tl newfacts)
            facts = hd newfacts :: facts;

        # Trim history to MAX_HISTORY
        history = trimhistory(history, MAX_HISTORY);

        if(nerrs > 0) {
            consecutive_errors += nerrs;
            sys->print("Errors this iteration: %d (consecutive: %d/%d)\n",
                nerrs, consecutive_errors, MAX_ERRORS);
        } else {
            consecutive_errors = 0;  # Reset consecutive counter, but keep history
            if(hascompletion(response)) {
                sys->print("\n=== Task Complete ===\n");
                break;
            }
        }
    }

    if(iterations >= MAX_ITERATIONS)
        sys->print("\n=== Safety Limit Reached (%d iterations) ===\n", MAX_ITERATIONS);
    if(consecutive_errors >= MAX_ERRORS)
        sys->print("\n=== Too Many Errors (%d consecutive) ===\n", consecutive_errors);

    # Cleanup windows if -cleanup flag was set
    xenith_cleanup();

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
    # Look for explicit completion indicators
    if(len response >= 4 && response[0:4] == "DONE")
        return 1;
    if(len response >= 8 && response[0:8] == "Complete")
        return 1;

    # Don't consider ls commands as completing a task
    lines := splitlines(response);
    for(; lines != nil; lines = tl lines) {
        line := trim(hd lines);
        if(line == "" || line[0] == '#')
            continue;
        if(len line >= 3 && line[0:3] == "```")
            continue;
        if(len line >= 2 && line[0:2] == "ls")
            return 0;  # ls is exploratory, not completion
    }

    # For cat commands that read results, consider potentially complete
    # (if no errors, the result was obtained)
    return 1;
}

# Execute shell commands from the LLM response
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
            return (-1, "xenith/write", "usage: xenith write <id> <text>", "");
        }
        winid := int (hd args);
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
            return (-1, "xenith/delete", "usage: xenith delete <id>", "");
        }
        winid := int (hd args);
        if(xenith_delete(winid) < 0)
            return (-1, sys->sprint("xenith/%d/delete", winid), "delete failed", "");
        return (0, sys->sprint("xenith/%d/delete", winid), "deleted", "");

    "image" =>
        if(len args < 2) {
            return (-1, "xenith/image", "usage: xenith image <id> <path>", "");
        }
        winid := int (hd args);
        imgpath := hd (tl args);
        if(xenith_image(winid, imgpath) < 0)
            return (-1, sys->sprint("xenith/%d/image", winid), "image display failed", "");
        return (0, sys->sprint("xenith/%d/image", winid), imgpath, "");

    "ctl" =>
        if(len args < 2) {
            return (-1, "xenith/ctl", "usage: xenith ctl <id> <command>", "");
        }
        winid := int (hd args);
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

    sys->print("Exec: cat %s\n", file);

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
