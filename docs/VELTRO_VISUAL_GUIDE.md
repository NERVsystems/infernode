# Veltro Visual Guide & Cheat Sheet

*A quick-reference visual guide for understanding Veltro's architecture*

---

## The Big Idea

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                                                                              ║
║                     NAMESPACE = CAPABILITY                                   ║
║                                                                              ║
║   If you can see it  ───────▶  You can use it                               ║
║   If you can't see it ──────▶  It doesn't exist                             ║
║                                                                              ║
║   No permissions. No ACLs. No "access denied."                              ║
║   Just: present or absent.                                                  ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

---

## System At A Glance

```
                              ┌─────────────────┐
                              │  User Request   │
                              │  "Do this task" │
                              └────────┬────────┘
                                       │
                                       ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                                                                              │
│    ┌─────────────┐         ┌─────────────┐         ┌─────────────┐          │
│    │             │         │             │         │             │          │
│    │  VELTRO.B   │◀───────▶│  TOOLS9P.B  │◀───────▶│   TOOLS     │          │
│    │  (brain)    │         │  (hands)    │         │  (skills)   │          │
│    │             │         │             │         │             │          │
│    └──────┬──────┘         └─────────────┘         └─────────────┘          │
│           │                                                                  │
│           │                                                                  │
│           ▼                                                                  │
│    ┌─────────────┐                                                          │
│    │             │                                                          │
│    │   /n/llm    │ ◀──────── LLM thinks, agent acts                         │
│    │  (thought)  │                                                          │
│    │             │                                                          │
│    └─────────────┘                                                          │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## The Security Model (One Picture)

```
╔══════════════════════════════════════════════════════════════════════════════╗
║  PARENT AGENT                                     CHILD AGENT (sandboxed)   ║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                              ║
║  ┌────────────────────┐                          ┌──────────────────────┐   ║
║  │ /                  │                          │ /                    │   ║
║  │ ├── appl/          │      SPAWN               │ ├── tmp/             │   ║
║  │ │   └── veltro/    │  ═══════════════▶        │ │   └── scratch/     │   ║
║  │ ├── dis/           │                          │ ├── dis/             │   ║
║  │ │   └── everything │      Only what you       │ │   └── lib/         │   ║
║  │ ├── tmp/           │      explicitly grant    │ │       └── (subset) │   ║
║  │ ├── n/             │      gets copied in      │ ├── tool/            │   ║
║  │ │   └── llm/       │                          │ │   └── (few tools)  │   ║
║  │ ├── dev/           │                          │ └── dev/             │   ║
║  │ │   └── everything │                          │     ├── cons         │   ║
║  │ └── tool/          │                          │     └── null         │   ║
║  │     └── all tools  │                          │                      │   ║
║  └────────────────────┘                          └──────────────────────┘   ║
║                                                                              ║
║        FULL VIEW                                      RESTRICTED VIEW       ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝

                     Child CANNOT see parent's full namespace.
                     Child CANNOT escape sandbox.
                     Child's view = Child's capability.
```

---

## The 10-Step Isolation Dance

```
                    PARENT                              CHILD
                    ══════                              ═════

              1. Validate ID
              2. Create sandbox dir
              3. Copy granted files
              4. Preload modules
                      │
                      │ spawn ──────────────────────────────┐
                      │                                      │
                      │                                      ▼
                      │                              5.  NEWPGRP
                      │                                  (fresh proc group)
                      │                                      │
                      │                              6.  FORKNS
                      │                                  (fork namespace)
                      │                                      │
                      │                              7.  NEWENV
                      │                                  (empty env)
                      │                                      │
                      │                              8.  NEWFD
                      │                                  (prune FDs)
                      │                                      │
                      │                              9.  NODEVS
                      │                                  (block #U)
                      │                                      │
                      │                              10. NEWNS
                      │                                  (sandbox = /)
                      │                                      │
                      │                                      ▼
                      │                              ┌──────────────┐
                      │                              │ ISOLATED     │
                      │                              │ Cannot escape│
                      │◀─────── result ─────────────│ Run task     │
                      │                              └──────────────┘
                      │
              Cleanup sandbox
```

---

## Agent Loop

```
    ┌─────────────────────────────────────────────────────────────────┐
    │                                                                 │
    │   START ──▶ Discover ──▶ Assemble ──▶ Query ──▶ Parse ──┬──▶ Execute ──┐
    │             Namespace    Prompt       LLM      Response │       Tool   │
    │                                                         │              │
    │                                                         │              │
    │                                              ┌──────────┘              │
    │                                              │                         │
    │                                              ▼                         │
    │                                           DONE? ──yes──▶ END           │
    │                                              │                         │
    │                                              no                        │
    │                                              │                         │
    │                                              └─────────────────────────┘
    │                                                                 │
    │                         Loop until DONE or max steps            │
    │                                                                 │
    └─────────────────────────────────────────────────────────────────┘
```

---

## Tool Invocation Protocol

```
    LLM Output                    Veltro Parser                     Tool
    ══════════                    ═════════════                     ════

    "read /tmp/x.b"       ──▶     tool = "read"           ──▶     read.exec("/tmp/x.b")
                                  args = "/tmp/x.b"

    "list /appl"          ──▶     tool = "list"           ──▶     list.exec("/appl")
                                  args = "/appl"

    "DONE                         tool = "DONE"           ──▶     Exit loop
     Task complete"               (completion signal)


    HEREDOC FOR MULTI-LINE:
    ═══════════════════════

    "write /tmp/x <<EOF   ──▶     tool = "write"          ──▶     write.exec("/tmp/x line1
     line1                        args = "/tmp/x line1             line2
     line2                               line2                     line3")
     line3                               line3"
     EOF"
```

---

## File System Layout

```
    /tool/                          /n/llm/                    Sandbox
    ══════                          ═══════                    ═══════

    /tool/                          /n/llm/                    /tmp/.veltro/
    ├── tools    (r)                └── ask   (rw)             └── sandbox/
    │   → list of names                 write: prompt              └── {id}/
    │                                   read:  response                ├── dis/
    ├── help     (rw)                                                  │   ├── lib/
    │   write: tool name                                               │   └── veltro/
    │   read:  documentation                                           ├── dev/
    │                                                                  │   ├── cons
    └── <tool>   (rw)                                                  │   └── null
        write: arguments                                               ├── tool/
        read:  result                                                  └── tmp/


    HOW TOOL CALLS WORK:
    ════════════════════

    1. open("/tool/read")     →  Get file handle
    2. write("/tmp/x.b")      →  Send arguments to tool
    3. seek(0)                →  Reset to beginning
    4. read()                 →  Get result
```

---

## Security Properties Quick Reference

| Threat | Blocked By | How |
|--------|-----------|-----|
| Host filesystem escape | `NODEVS` | Blocks `#U`, `#p`, `#c` device naming |
| Environment secrets | `NEWENV` | Creates empty environment (not inherited) |
| File descriptor leak | `NEWFD` | Explicit keep-list, prune all others |
| Service discovery | `NEWPGRP` | Fresh process group, empty srv registry |
| Namespace expansion | `NEWNS` | Sandbox becomes `/`, no way to escape |
| Path traversal | `validatesandboxid()` | Only alphanumeric + hyphen allowed |
| Race conditions | `create()` | Fails if directory exists |
| Shell injection | `safeexec` | Loads `.dis` directly, no shell |
| Audit evasion | Audit log | All binds logged before execution |

---

## Trusted vs Untrusted

```
    UNTRUSTED (default)                     TRUSTED (trusted=1)
    ═══════════════════                     ═══════════════════

    ┌─────────────────────┐                 ┌─────────────────────┐
    │ NO shell access     │                 │ HAS shell access    │
    │                     │                 │                     │
    │ Tools:              │                 │ Tools:              │
    │ • read              │                 │ • read              │
    │ • write             │                 │ • write             │
    │ • list              │                 │ • list              │
    │ • safeexec          │                 │ • exec (shell!)     │
    │                     │                 │                     │
    │ exec tool?          │                 │ exec tool?          │
    │ ✗ NOT available     │                 │ ✓ Available         │
    │                     │                 │                     │
    │ sh.dis bound?       │                 │ sh.dis bound?       │
    │ ✗ NO                │                 │ ✓ YES               │
    └─────────────────────┘                 └─────────────────────┘

    Use for:                                Use for:
    • Code analysis                         • Build tasks
    • File reading                          • System operations
    • Safe operations                       • Trusted automation
```

---

## Key Files

| File | Purpose | Lines |
|------|---------|-------|
| `veltro.b` | Main agent loop | ~600 |
| `tools9p.b` | 9P tool server | ~660 |
| `nsconstruct.b` | Sandbox prep & security | ~860 |
| `spawn.b` | Sub-agent spawning | ~660 |
| `subagent.b` | Sandboxed agent loop | ~450 |
| `tool.m` | Tool interface | ~20 |

---

## Common Tool Patterns

```
    READ FILE:
    read /path/to/file
    read /path/to/file 10        (from line 10)
    read /path/to/file 10 50     (50 lines from line 10)

    LIST DIRECTORY:
    list /path/to/dir
    list /path/to/dir -a         (show hidden)
    list /path/to/dir -l         (long format)

    SEARCH:
    search "pattern" /path
    find *.b /path

    WRITE FILE:
    write /path/to/file <<EOF
    content here
    more content
    EOF

    SPAWN SUB-AGENT:
    Spawn tools=read,list paths=/appl -- "List all .b files"
    Spawn tools=read agenttype=explore paths=/appl -- "Find handlers"
```

---

## Module Preloading (Why It Matters)

```
    THE PROBLEM:
    ════════════

    After NEWNS, the sandbox becomes /
    Original paths like /dis/veltro/tools/read.dis don't exist!

    If you try to load a module AFTER NEWNS:
    ┌─────────────────────────────────────────────────────────────────┐
    │  load Tool "/dis/veltro/tools/read.dis"  →  FAILS!             │
    │                                              "file not found"   │
    └─────────────────────────────────────────────────────────────────┘


    THE SOLUTION:
    ═════════════

    Load modules BEFORE NEWNS while paths still exist:

    BEFORE SPAWN:                          AFTER NEWNS:
    ┌──────────────────────┐               ┌──────────────────────┐
    │ mod = load Tool path │               │ // mod already in    │
    │ mod->init()          │    ──────▶    │ // memory!           │
    │ preloadedtools = mod │               │ mod->exec(args)      │
    └──────────────────────┘               └──────────────────────┘

    Modules live in memory. Filesystem not needed after load.
```

---

## LLM FD Trick

```
    PROBLEM: Binds don't survive NEWNS
    SOLUTION: Open FDs DO survive!

    BEFORE SPAWN:
    ┌─────────────────────────────────────────────┐
    │ llmfd = open("/n/llm/ask")                  │
    │ llmfdnum = llmfd.fd        // save number   │
    └─────────────────────────────────────────────┘
                        │
                        │ spawn + NEWNS
                        ▼
    AFTER NEWNS:
    ┌─────────────────────────────────────────────┐
    │ // llmfd ref is invalid                     │
    │ // but FD NUMBER is still valid!            │
    │ llmfd = sys->fildes(llmfdnum)  // recreate  │
    │ write(llmfd, prompt)           // works!    │
    └─────────────────────────────────────────────┘

    Pass FD number through NEWFD keep-list.
    Recreate ref after NEWNS using fildes().
```

---

## Quick Command Reference

```bash
# Run Veltro agent
veltro "your task here"
veltro -v "task"                    # verbose
veltro -n 100 "task"                # max 100 steps

# Start tools9p
tools9p read list find              # serve only these tools
tools9p -D read list                # debug mode
tools9p -m /mytool read             # custom mount point

# Run tests
./emu/MacOSX/Infernode -r . /tests/veltro_test.dis
./emu/MacOSX/Infernode -r . /tests/veltro_security_test.dis -v
```

---

## Design Philosophy

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                                                                              ║
║  "Namespace = Capability" means:                                             ║
║                                                                              ║
║  • No policy engine needed                                                   ║
║  • No ACL checking code                                                      ║
║  • No permission escalation bugs                                             ║
║  • Audit by `ls` — what you see is what's allowed                           ║
║                                                                              ║
║  The security model is structural, not behavioral.                           ║
║  It's not "you can't do X" — it's "X doesn't exist for you."                ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

---

*Quick reference for Veltro Agent Framework*
