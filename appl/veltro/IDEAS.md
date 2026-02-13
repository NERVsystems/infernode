# Veltro Ideas and Future Work

## Completed (Phase 1)

### Phase 1a: Per-Agent LLM Isolation ✓
- LLMConfig type in nsconstruct.m
- Config files written to sandbox before bind
- Parent mounts llm9p into sandbox with MBEFORE
- Child inherits per-agent model/temperature/system settings
- 33 automated tests + 5 manual tests passing

### Phase 1b: mc9p (9P-based MCP) ✓
- Filesystem-as-schema design (no JSON)
- Domain/endpoint model via synthetic files
- Network isolation (requires -n flag for /net access)
- HTTP provider working

### Phase 1c: New Tools ✓
- **http** - HTTP client (HTTP working, HTTPS needs /net/ssl)
- **git** - Git operations (requires /cmd device)
- **json** - JSON parsing and path queries
- **ask** - User prompts via console
- **diff** - File comparison
- **memory** - Session persistence (basic implementation)

### Phase 1d: Agent Memory
- Basic memory tool implemented
- Full cross-session persistence not yet tested

---

## Usability Improvements

### Launcher Scripts

Create purpose-specific launcher scripts that preserve security while improving convenience.
The caller still explicitly chooses capabilities, but common configurations are pre-packaged.

**Example scripts:**

```sh
# /dis/veltro/launch/ui - For Xenith UI tasks
#!/dis/sh
tools9p read list xenith &
sleep 1
veltro $*

# /dis/veltro/launch/code - For code exploration
#!/dis/sh
tools9p read list find search &
sleep 1
veltro $*

# /dis/veltro/launch/edit - For code editing
#!/dis/sh
tools9p read list find search write edit &
sleep 1
veltro $*

# /dis/veltro/launch/full - All tools (trusted use only)
#!/dis/sh
tools9p read list find search write edit exec spawn xenith &
sleep 1
veltro $*
```

### Xenith Integration

Add Xenith menu items or tag commands for launching Veltro:

1. **Tag commands** - Right-click menu items like "Veltro:UI", "Veltro:Code"
2. **Window action** - Button in Xenith toolbar to launch agent with context
3. **Plumber integration** - Plumb selected text to Veltro as a task

### Profile Integration

Add optional tools9p startup to profile for users who want always-available tools:

```sh
# In /lib/sh/profile (user's choice)
# Start minimal tool server for interactive Veltro use
tools9p read list &
```

**Note:** This is the user's security decision - they choose what's always available.

---

## Tool Improvements

### Xenith Tool Enhancements

- **Batch operations** - Create multiple windows in one call
- **Templates** - Pre-configured window layouts (log viewer, code display, etc.)
- **Event subscription** - Tool to watch for window events (clicks, selections)
- **Clipboard integration** - Read/write system clipboard

### New Tools

- ~~**diff** - Compare files, show differences~~ ✓ Completed
- ~~**git** - Basic git operations (status, log, diff)~~ ✓ Completed
- ~~**http** - Fetch URLs, make API calls~~ ✓ Completed (HTTP only)
- ~~**json** - Parse and query JSON data~~ ✓ Completed
- ~~**ask** - Prompt user for input via dialog~~ ✓ Completed

---

## Architecture Ideas

### Tool Capability Levels

Define standard capability profiles:

| Level | Tools | Use Case |
|-------|-------|----------|
| readonly | read, list, find, search | Safe exploration |
| ui | readonly + xenith | Display results |
| write | ui + write, edit | Modify files |
| exec | write + exec | Run commands |
| full | exec + spawn | Create sub-agents |

### Persistent Tool Server

A system-wide tools9p that runs as a service:
- Started at boot
- Provides baseline tools to all agents
- Additional tools granted per-session via namespace overlays

**Security consideration:** Must not grant more than explicitly allowed.

### Agent Chaining - Partially Implemented

**Current State:**
The spawn tool creates isolated sandboxes with attenuated capabilities, but
only executes a **single tool call** - not a full agent loop.

```
spawn tools=find,search paths=/appl -- find *.b /appl
```
This runs `find` tool once. The "task" is parsed as: `toolname args`

**Limitation:**
Natural language tasks like `"Find how errors are handled"` don't work
because spawn tries to parse "Find" as a tool name.

**What's needed for true sub-agents:**
1. Run veltro.dis inside the sandbox
2. Mount /n/llm into sandbox (Phase 1a done - LLMConfig exists)
3. Child can then run full agent loop with LLM access

**Current workaround:**
Parent agent does the multi-step work itself rather than delegating.

**Completed:**
- ~~Parent grants subset of its own tools~~ ✓ spawn tool
- ~~Child cannot exceed parent's capabilities~~ ✓ FORKNS + bind-replace isolation (v3)
- ~~Audit trail tracks delegation chain~~ ✓ namespace audit
- ~~Namespace v3: FORKNS + bind-replace~~ ✓ replaces NEWNS + sandbox model

---

## Documentation

- Tutorial: "Your First Veltro Task"
- Guide: "Securing Veltro Deployments"
- Reference: Tool API documentation
- Examples: Common task patterns

---

## Performance

- Tool module caching (avoid reloading .dis files)
- Streaming results for large outputs
- Parallel tool execution where safe
