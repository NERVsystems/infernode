# Veltro Namespace Security Model (v2)

## Overview

Veltro uses Inferno OS namespace isolation to create secure sandboxes for AI agents. The key insight is that `NEWNS` makes the current directory become the new root, providing true capability-based security when combined with `NEWENV`, `NEWFD`, and `NODEVS`.

## Security Architecture

### Parent Process (before spawn)

1. **validatesandboxid(id)** - Reject path traversal attacks
2. **preparesandbox(caps)** - Create sandbox directory with restrictive permissions (0700)
3. **verifyownership(path)** - stat() every path before binding

### Child Process (after spawn)

The child executes this pctl sequence for isolation:

```
1. pctl(NEWPGRP, nil)    - Fresh process group (empty srv registry)
2. pctl(FORKNS, nil)     - Fork namespace for mutation
3. pctl(NEWENV, nil)     - Empty environment (NOT FORKENV!)
4. verifysafefds()       - Verify FDs 0-2 point at safe endpoints
5. pctl(NEWFD, keepfds)  - Prune all other FDs
6. pctl(NODEVS, nil)     - Block #U/#p/#c device naming
7. chdir(sandboxdir)     - Enter prepared sandbox
8. pctl(NEWNS, nil)      - Sandbox becomes /
9. mounttools9p(tools)   - Mount tools without /srv or /net
10. executetask(task)    - No policy checks; namespace IS capability
```

## Sandbox Structure

```
/tmp/.veltro/sandbox/{id}/
├── dis/
│   ├── lib/            ← bound from /dis/lib
│   ├── sh.dis          ← ONLY if trusted=1
│   ├── echo.dis        ← each granted shellcmd (trusted only)
│   └── veltro/tools/   ← bound from /dis/veltro/tools
├── dev/
│   ├── cons            ← bound from /dev/cons
│   └── null            ← bound from /dev/null
├── tool/               ← mount point for tools9p
├── tmp/                ← writable scratch
├── n/llm/              ← LLM if llmconfig != nil
└── [granted paths]     ← bound from parent namespace

Audit log: /tmp/.veltro/audit/{id}.ns
```

## Security Properties

| Property | Mechanism |
|----------|-----------|
| No #U escape | NODEVS before sandbox entry |
| No env secrets | NEWENV creates empty environment |
| No FD leaks | NEWFD with explicit keep-list |
| Safe FD 0-2 | verifysafefds() before NEWFD |
| Empty srv registry | NEWPGRP first - fresh process group |
| Truthful namespace | Only granted paths exist after NEWNS |
| Capability attenuation | Parent binds from own namespace |
| No /prog discovery | Not bound into sandbox |
| Sandbox ID validated | No traversal (/, ..), length limits |
| Race-free creation | Create fails if directory exists |
| Auditable | Bind transcript written to log |
| No shell for untrusted | safeexec runs .dis directly |

## Trusted vs Untrusted Agents

### Untrusted (default)
- No shell access
- `safeexec` tool loads .dis modules directly
- No shell metacharacter interpretation
- Cannot execute arbitrary commands

### Trusted
- Shell access via bound sh.dis
- Granted shell commands bound individually
- Can use exec tool with shell

## Sandbox ID Validation

Sandbox IDs must:
- Be 1-64 characters long
- Contain only alphanumeric characters and hyphens
- Not be "." or ".."
- Not contain path separators (/, \)

This prevents path traversal attacks like `../../../etc/passwd`.

## Cleanup

The `cleanupsandbox()` function properly unmounts all bind points before removing the sandbox directory. This prevents the recursive removal from following bind mounts and accidentally deleting original files.

## Files

| File | Purpose |
|------|---------|
| `nsconstruct.m` | Module interface with types and function signatures |
| `nsconstruct.b` | Sandbox preparation, validation, audit logging |
| `tools/spawn.b` | Secure spawn with pctl sequence |
| `tools/safeexec.b` | Direct .dis execution without shell |

## Tool Server Separation (Security by Design)

Veltro requires `tools9p` to be started separately by the caller, with explicit tool grants:

```sh
tools9p read list xenith &    # Caller chooses tools
veltro "do something"         # Agent operates within constraints
```

**This separation is intentional security architecture, not a usability oversight.**

### Why the Agent Cannot Self-Grant Tools

If Veltro could auto-start `tools9p` with its own tool selection:

```
INSECURE: Agent → decides its own tools → privilege escalation
SECURE:   Caller → decides tools → agent operates within constraints
```

The principle is **capability granting flows from caller to callee**, never the reverse.

### Security Implications

| Design | Who Grants | Risk |
|--------|-----------|------|
| Caller starts tools9p | Trusted caller | None - correct model |
| Agent auto-starts tools9p | Untrusted agent | Agent chooses own capabilities |
| Default tool set | Implicit/config | Config becomes attack surface |

### The Inconvenience is a Feature

Running commands together (`tools9p ... ; veltro ...`) ensures:
1. Explicit capability grants visible in command
2. No hidden default permissions
3. Audit trail shows what was granted
4. Cannot escalate beyond what caller provided

### Safe Usability Alternatives

These preserve security while improving convenience:

1. **Wrapper scripts** - User creates scripts with their chosen tools
2. **Profile integration** - User adds to profile (their choice, their risk)
3. **Xenith actions** - UI buttons that run pre-configured commands

See `IDEAS.md` for implementation suggestions.

## Testing

Security tests are in `tests/veltro_security_test.b`:

```sh
./emu/MacOSX/o.emu -r. /dis/tests/veltro_security_test.dis -v
```

Tests cover:
- Sandbox ID validation (path traversal rejection)
- Sandbox ID generation (uniqueness)
- Sandbox path format
- Path ownership verification
- Sandbox preparation (directory structure)
- Path binding
- Trusted vs untrusted (shell access)
- Race condition protection
- Audit logging
