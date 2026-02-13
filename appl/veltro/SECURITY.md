# Veltro Namespace Security Model (v3)

## Overview

Veltro uses Inferno OS namespace isolation to create secure environments for AI agents. The key insight is that `bind(shadow, target, MREPL)` replaces a directory's contents entirely with only what's in the shadow directory — an allowlist operation where anything not explicitly placed in the shadow is invisible.

## Security Architecture

### Core Primitive: `restrictdir(target, allowed)`

1. Create a shadow directory
2. For each allowed item: bind `target/item` into the shadow
3. `bind(shadow, target, MREPL)` — replace entire target
4. Result: only allowed items are visible; everything else is gone

### Two-Level Restriction

```
Veltro startup:  FORKNS + restrictns() — restrict parent namespace
Subagent spawn:  FORKNS + restrictns() — inherit + further restrict
```

Both levels use the same `restrictdir()` primitive. Capability attenuation is natural: children fork an already-restricted namespace and can only narrow further.

### Parent Process (veltro.b)

After loading modules and mounting tools9p/llm:

```
1. pctl(FORKNS)         - Fork namespace (caller unaffected)
2. restrictns(caps)     - Restrict /dis, /dev, /n, /lib, /tmp
3. verifyns(expected)   - Verify restrictions applied
4. runagent(task)       - Agent operates in restricted namespace
```

### Child Process (spawn.b)

```
1. pctl(NEWPGRP)        - Fresh process group (empty srv registry)
2. pctl(FORKNS)         - Fork already-restricted parent namespace
3. pctl(NEWENV)         - Empty environment (NOT FORKENV!)
4. Open LLM FDs         - While /n/llm still accessible
5. restrictns(caps)     - Further bind-replace restrictions
6. verifysafefds()      - Verify FDs 0-2 point at safe endpoints
7. pctl(NEWFD, keepfds) - Prune all other FDs
8. pctl(NODEVS)         - Block #U/#p/#c device naming
9. subagent->runloop()  - Execute task
```

## Namespace After Restriction

### Parent Veltro

```
/
├── dis/
│   ├── lib/            ← runtime libraries
│   └── veltro/         ← agent modules + tools
├── dev/
│   ├── cons            ← console
│   └── null            ← null device
├── lib/
│   └── veltro/         ← agents, reminders, system.txt
├── n/
│   └── llm/            ← LLM access (if mounted)
├── tmp/
│   └── veltro/
│       ├── scratch/    ← agent workspace
│       └── .ns/        ← shadow dirs + audit logs
└── tool/               ← tools9p mount (unchanged)

NOT VISIBLE after restriction:
/n/local (host filesystem — explicitly unmounted)
/dis/*.dis (top-level commands)
/dev/* (other devices)
/lib/* (fonts, etc.)
```

### Child Subagent

Inherits parent's restricted namespace, further restricted:
- `/dis/veltro/tools/` — only granted tool .dis files
- Everything else inherited from already-restricted parent

## Security Properties

| Property | Mechanism |
|----------|-----------|
| No #U escape | NODEVS after all bind operations (child only) |
| No env secrets | NEWENV creates empty environment |
| No FD leaks | NEWFD with explicit keep-list |
| Safe FD 0-2 | verifysafefds() before NEWFD |
| Empty srv registry | NEWPGRP first |
| Truthful namespace | bind-replace shows only allowed items |
| Capability attenuation | Child forks restricted parent, can only narrow |
| No /n/local access | Explicitly unmounted before restriction |
| No cleanup needed | bind-replace is namespace-only, no physical dirs |
| Auditable | Restriction ops logged to audit file |
| Shell access controlled | sh.dis only bound if shellcmds is non-nil |

## Shell Access

Shell access is controlled by the `shellcmds` field in `Capabilities`. If `shellcmds` is nil, no shell. If non-nil, `sh.dis` plus every named command `.dis` are added to the `/dis` allowlist.

```
# No shell — shellcmds is nil
caps := ref Capabilities(..., nil, ...);

# Shell with cat and ls — sh.dis + cat.dis + ls.dis visible
caps := ref Capabilities(..., "cat" :: "ls" :: nil, ...);
```

## Key Design Decisions

### Why bind-replace instead of NEWNS + sandbox?

| Criterion | v2 (NEWNS + sandbox) | v3 (FORKNS + bind-replace) |
|-----------|---------------------|---------------------------|
| File copying | Required (NEWNS loses binds) | None |
| Cleanup | Required (rmrf sandbox dir) | None (namespace-only) |
| Bootstrap | Chicken-and-egg problem | No problem (fork existing) |
| Code size | ~864 lines | ~200 lines |
| Security model | Allowlist (by construction) | Allowlist (by replacement) |
| Race conditions | Create-fails-if-exists | PID-scoped shadow dirs |

### Shadow Directory Management

Shadow directories are created under `/tmp/veltro/.ns/shadow/` with PID-prefixed names to avoid collisions between parent and child processes. After `/tmp` is restricted to only `veltro/`, the shadow dirs remain accessible through `/tmp/veltro/`.

The `/tmp` restriction is always applied LAST so that earlier `restrictdir()` calls can create their shadow dirs.

## Files

| File | Purpose |
|------|---------|
| `nsconstruct.m` | Module interface: restrictdir, restrictns, verifyns |
| `nsconstruct.b` | Core implementation (~200 lines) |
| `tools/spawn.b` | Secure spawn with FORKNS + restrictns |
| `veltro.b` | Parent namespace restriction at startup |
| `subagent.b` | Agent loop (runs in restricted namespace) |

## Tool Server Separation (Security by Design)

Veltro requires `tools9p` to be started separately by the caller, with explicit tool grants:

```sh
tools9p read list xenith &    # Caller chooses tools
veltro "do something"         # Agent operates within constraints
```

**This separation is intentional security architecture, not a usability oversight.**

The principle is **capability granting flows from caller to callee**, never the reverse.

## Testing

Security tests are in `tests/veltro_security_test.b`:

```sh
./emu/MacOSX/o.emu -r. /tests/veltro_security_test.dis -v
```

Tests cover:
- restrictdir() allowlist (only allowed items visible)
- restrictdir() exclusion (non-allowed items invisible)
- restrictdir() idempotent (multiple calls safe)
- restrictns() full policy (/dis, /dev, /n, /lib, /tmp)
- restrictns() shell access via shellcmds
- restrictns() concurrent (race safety)
- verifyns() violation detection
- Audit logging
- Missing items handled gracefully

Concurrency tests in `tests/veltro_concurrent_test.b`:
- Concurrent init
- Concurrent restrictdir
- Concurrent restrictns
