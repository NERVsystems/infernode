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

## Testing

Security tests are in `tests/veltro_security_test.b`:

```sh
./emu/MacOSX/Infernode -r . /dis/tests/veltro_security_test.dis -v
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
