# Veltro Namespace Security Model (v2)

## Overview

Veltro uses Inferno's namespace isolation primitives to create secure sandboxes for AI agent execution. The key insight is that **NEWNS makes the current directory become the new root**, providing true capability-based security where the namespace itself IS the capability.

## Architecture

### Security Model

```
Parent (before spawn):
  1. validatesandboxid(id)     - Reject traversal attacks (../, /, special chars)
  2. preparesandbox(caps)      - Create sandbox with restrictive permissions
  3. Pre-load tool modules     - Load .dis files while paths exist

Child (after spawn):
  1. pctl(NEWPGRP, nil)        - Fresh process group (empty srv registry)
  2. pctl(FORKNS, nil)         - Fork namespace for mutation
  3. pctl(NEWENV, nil)         - Empty environment (no inherited secrets)
  4. verifysafefds()           - Check FDs 0-2 are safe
  5. pctl(NEWFD, keepfds)      - Prune all other FDs
  6. pctl(NODEVS, nil)         - Block #U/#p/#c device naming
  7. chdir(sandboxdir)         - Enter prepared sandbox
  8. pctl(NEWNS, nil)          - Sandbox becomes /
  9. safeexec(task)            - Execute using pre-loaded modules
```

### Sandbox Structure

```
/tmp/.veltro/sandbox/{id}/
├── dis/
│   ├── lib/              ← Bound from /dis/lib (runtime)
│   ├── veltro/tools/     ← Bound from /dis/veltro/tools
│   └── sh.dis            ← Only if trusted=1
├── dev/
│   ├── cons              ← Bound from /dev/cons
│   └── null              ← Bound from /dev/null
├── tool/                 ← Mount point for tools9p
├── tmp/                  ← Writable scratch space
├── n/llm/                ← LLM access (if configured)
└── [granted paths]       ← Copied from parent namespace
```

### Security Properties

| Property | Mechanism |
|----------|-----------|
| No #U escape | NODEVS blocks device naming before sandbox entry |
| No env secrets | NEWENV creates empty environment (not FORKENV) |
| No FD leaks | NEWFD with explicit keep-list [0,1,2,pipefd] |
| Empty srv registry | NEWPGRP creates fresh process group |
| Truthful namespace | Only granted paths exist after NEWNS |
| No shell for untrusted | safeexec() loads .dis directly, no shell interpretation |
| Race-free creation | sys->create() fails if sandbox exists |
| Auditable | All binds logged to /tmp/.veltro/audit/{id}.ns |

## Key Files

| File | Purpose |
|------|---------|
| `appl/veltro/nsconstruct.m` | Interface definitions (Capabilities, Mountpoints) |
| `appl/veltro/nsconstruct.b` | Sandbox preparation, validation, cleanup |
| `appl/veltro/tools/spawn.b` | Child process isolation and execution |
| `appl/veltro/tool.m` | Tool interface with init() for pre-loading |
| `appl/veltro/tools9p.b` | Tool filesystem server |

## Learnings and Gotchas

### 1. NEWNS Discards Binds

**Problem**: After `pctl(NEWNS, nil)`, bind mounts from the parent namespace are lost. The child only sees the physical file structure.

**Solution**: Copy files instead of binding for granted paths. Binds only work for paths that are physically inside the sandbox directory.

```limbo
# WRONG: This bind is lost after NEWNS
sys->bind("/appl/veltro", sandboxdir + "/appl/veltro", Sys->MREPL);

# RIGHT: Copy files so they survive NEWNS
copytree("/appl/veltro", sandboxdir + "/appl/veltro");
```

### 2. Module Pre-loading is Essential

**Problem**: After NEWNS, paths like `/dis/veltro/tools/list.dis` no longer exist. Loading modules fails.

**Solution**: Pre-load all required modules BEFORE the spawn. Limbo's `spawn` creates threads that share memory, so pre-loaded modules remain accessible after NEWNS.

```limbo
# BEFORE spawn: load modules while paths exist
preloadmodules(tools);

# AFTER NEWNS: modules are in memory, use directly
tool := findpreloaded("list");
result := tool->exec(args);
```

### 3. Tool init() Must Happen Before NEWNS

**Problem**: Tool modules may load dependencies in their `exec()` function. After NEWNS, these dependencies can't be found.

**Solution**: Added `init(): string` to the Tool interface. Called during pre-loading while paths still exist.

```limbo
Tool: module {
    init: fn(): string;  # Initialize while paths exist
    name: fn(): string;
    doc:  fn(): string;
    exec: fn(args: string): string;
};
```

### 4. Device Files Block Copy Operations

**Problem**: `copytree()` trying to copy `/dev` would block forever when opening `/dev/cons` (waiting for console input).

**Solution**: Skip device files during copy. Device files are bound, not copied.

```limbo
isdevicepath(path: string): int
{
    if(len path >= 5 && path[0:5] == "/dev/")
        return 1;
    if(len path > 0 && path[0] == '#')
        return 1;
    return 0;
}
```

### 5. Stale Sandbox Detection Requires Wall Clock Time

**Problem**: Sandbox IDs using `sys->millisec()` (boot-relative time) broke stale detection across Inferno restarts. Old sandboxes appeared newer than current time.

**Solution**: Use `daytime->now()` (seconds since epoch) for sandbox timestamps.

```limbo
# WRONG: Resets to 0 on each boot
now := sys->millisec();

# RIGHT: Survives reboots
now := daytime->now();
```

### 6. NODEVS Doesn't Block Everything

**Critical**: NODEVS blocks `#U` (host filesystem), `#p` (prog), `#c` (console driver), but still permits:
- `#e` (environment) - mitigated by NEWENV
- `#s` (srv registry) - mitigated by NEWPGRP
- `#|` (pipes) - needed for IPC
- `#D` (SSL) - may be needed for secure connections

The full security model requires NEWENV + NEWPGRP + NODEVS together.

### 7. tools9p Has Heavy Dependencies

**Problem**: tools9p.b loads styx, styxservers, and other modules. After NEWNS, these fail to load.

**Solution**: Don't start tools9p in the child. Instead, use pre-loaded tool modules directly via `safeexec()`.

## Testing

```sh
# Build
export ROOT=$PWD && export PATH=$PWD/MacOSX/arm64/bin:$PATH
cd appl/veltro && mk install
cd tests && mk install

# Run security tests (9 tests)
./emu/MacOSX/Infernode -r . /tests/veltro_security_test.dis -v

# Run spawn execution tests (4 tests)
./emu/MacOSX/Infernode -r . /tests/spawn_exec_test.dis -v
```

## Usage Example

```limbo
# Spawn a child with limited capabilities
result := spawn->exec("tools=list,read paths=/appl/veltro -- list /appl/veltro");

# Spawn a trusted child with shell access
result := spawn->exec("tools=read,exec paths=/tmp shellcmds=cat trusted=1 -- cat /tmp/file.txt");
```

## Future Work

1. **LLM Integration**: Mount actual LLM service at /n/llm/
2. **Network Isolation**: Control /net access for trusted agents
3. **Resource Limits**: Add CPU/memory constraints
4. **Deeper Integration Tests**: Verify isolation from inside spawned children
