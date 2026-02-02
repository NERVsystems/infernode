# Veltro Agent Architecture Analysis

## Executive Summary

**Veltro** is an AI agent framework built on Inferno OS that uses namespace-based capability security. Its fundamental innovation is that **the namespace itself IS the capability system** - if something isn't visible in an agent's namespace, it doesn't exist from that agent's perspective.

This document provides comprehensive diagrams and analysis to help understand Veltro's architecture, security model, and design philosophy.

---

## Table of Contents

1. [High-Level Architecture](#1-high-level-architecture)
2. [Component Diagrams](#2-component-diagrams)
3. [Security Model](#3-security-model)
4. [Agent Lifecycle](#4-agent-lifecycle)
5. [Data Flow](#5-data-flow)
6. [Sub-Agent Spawning](#6-sub-agent-spawning)
7. [Tool System](#7-tool-system)
8. [Critical Review](#8-critical-review)

---

## 1. High-Level Architecture

### 1.1 System Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              INFERNO OS HOST                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         VELTRO AGENT FRAMEWORK                       │   │
│  │                                                                      │   │
│  │  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐   │   │
│  │  │              │    │              │    │                      │   │   │
│  │  │  veltro.b    │───▶│  tools9p.b   │───▶│   Tool Modules       │   │   │
│  │  │  (Agent      │    │  (9P Server) │    │   (/dis/veltro/      │   │   │
│  │  │   Loop)      │    │              │    │    tools/*.dis)      │   │   │
│  │  │              │    │              │    │                      │   │   │
│  │  └──────────────┘    └──────────────┘    └──────────────────────┘   │   │
│  │         │                   │                      │                │   │
│  │         │                   │                      │                │   │
│  │         ▼                   ▼                      ▼                │   │
│  │  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐   │   │
│  │  │   /n/llm     │    │   /tool      │    │  nsconstruct.b       │   │   │
│  │  │  (LLM 9P     │    │  (Mounted    │    │  (Namespace          │   │   │
│  │  │   Interface) │    │   Tools)     │    │   Construction)      │   │   │
│  │  └──────────────┘    └──────────────┘    └──────────────────────┘   │   │
│  │                                                    │                │   │
│  │                                                    ▼                │   │
│  │                                          ┌──────────────────────┐   │   │
│  │                                          │    subagent.b        │   │   │
│  │                                          │    (Sandboxed        │   │   │
│  │                                          │     Agent Loop)      │   │   │
│  │                                          └──────────────────────┘   │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Layered Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                      APPLICATION LAYER                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ veltro   │  │ spawn    │  │ subagent │  │ tools    │            │
│  │ (main)   │  │ (sub-    │  │ (sand-   │  │ (read,   │            │
│  │          │  │  agents) │  │  boxed)  │  │  write..│            │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘            │
├─────────────────────────────────────────────────────────────────────┤
│                      SERVICE LAYER                                   │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐        │
│  │    tools9p     │  │   nsconstruct  │  │     mc9p       │        │
│  │   (9P Tool     │  │   (Namespace   │  │   (MCP over    │        │
│  │    Server)     │  │    Security)   │  │    9P)         │        │
│  └────────────────┘  └────────────────┘  └────────────────┘        │
├─────────────────────────────────────────────────────────────────────┤
│                      INTERFACE LAYER                                 │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐        │
│  │    tool.m      │  │  nsconstruct.m │  │   subagent.m   │        │
│  │   (Tool API)   │  │  (Security API)│  │  (Agent API)   │        │
│  └────────────────┘  └────────────────┘  └────────────────┘        │
├─────────────────────────────────────────────────────────────────────┤
│                      INFERNO OS LAYER                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ Styx/9P  │  │ Namespace│  │  pctl    │  │ Channels │            │
│  │ Protocol │  │ (NEWNS)  │  │ (process │  │ (IPC)    │            │
│  │          │  │          │  │  control)│  │          │            │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘            │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Component Diagrams

### 2.1 Core Module Relationships

```
                                ┌─────────────────┐
                                │    veltro.b     │
                                │   (Entry Point) │
                                └────────┬────────┘
                                         │
              ┌──────────────────────────┼──────────────────────────┐
              │                          │                          │
              ▼                          ▼                          ▼
     ┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
     │    tools9p.b    │       │   /n/llm/ask    │       │   /lib/veltro/  │
     │ (Tool Provider) │       │  (LLM Access)   │       │   system.txt    │
     └────────┬────────┘       └─────────────────┘       └─────────────────┘
              │
              │ implements
              ▼
     ┌─────────────────┐
     │     tool.m      │◀──────────────────────────────────────────┐
     │   (Interface)   │                                           │
     └─────────────────┘                                           │
              △                                                    │
              │ implements                                         │
              │                                                    │
     ┌────────┴────────┬──────────────┬──────────────┬────────────┤
     │                 │              │              │            │
┌────┴────┐      ┌────┴────┐   ┌─────┴────┐  ┌─────┴────┐ ┌─────┴────┐
│ read.b  │      │ write.b │   │ spawn.b  │  │  exec.b  │ │  ...     │
│         │      │         │   │          │  │          │ │(16 total)│
└─────────┘      └─────────┘   └────┬─────┘  └──────────┘ └──────────┘
                                    │
                                    │ uses
                                    ▼
                          ┌─────────────────┐
                          │  nsconstruct.b  │
                          │   (Security)    │
                          └────────┬────────┘
                                   │
                                   │ spawns
                                   ▼
                          ┌─────────────────┐
                          │   subagent.b    │
                          │  (Sandboxed     │
                          │   Agent Loop)   │
                          └─────────────────┘
```

### 2.2 Tool Module Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         TOOL.M INTERFACE                            │
├─────────────────────────────────────────────────────────────────────┤
│  Tool: module {                                                     │
│      init:  fn(): string;     // Initialize module                  │
│      name:  fn(): string;     // Return tool name                   │
│      doc:   fn(): string;     // Return documentation               │
│      exec:  fn(args: string): string;  // Execute with args         │
│  }                                                                  │
└─────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────────┐     ┌───────────────────┐     ┌───────────────────┐
│  FILE OPERATIONS  │     │     EXECUTION     │     │   INTERACTION     │
├───────────────────┤     ├───────────────────┤     ├───────────────────┤
│ • read.b          │     │ • exec.b          │     │ • xenith.b        │
│ • write.b         │     │ • spawn.b         │     │ • ask.b           │
│ • list.b          │     │ • safeexec.b      │     │ • http.b          │
│ • find.b          │     │                   │     │                   │
│ • edit.b          │     │                   │     │                   │
│ • diff.b          │     │                   │     │                   │
│ • search.b        │     │                   │     │                   │
└───────────────────┘     └───────────────────┘     └───────────────────┘
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────────┐     ┌───────────────────┐     ┌───────────────────┐
│      DATA         │     │    DEVELOPMENT    │     │   PERSISTENCE     │
├───────────────────┤     ├───────────────────┤     ├───────────────────┤
│ • json.b          │     │ • git.b           │     │ • memory.b        │
│                   │     │                   │     │                   │
└───────────────────┘     └───────────────────┘     └───────────────────┘
```

### 2.3 9P Filesystem Structure

```
                            /tool (mounted by tools9p)
                                     │
         ┌───────────────────────────┼───────────────────────────┐
         │                           │                           │
         ▼                           ▼                           ▼
    ┌─────────┐                ┌─────────┐               ┌─────────────┐
    │  tools  │                │  help   │               │ <toolname>  │
    │  (r)    │                │  (rw)   │               │    (rw)     │
    └─────────┘                └─────────┘               └─────────────┘
         │                           │                           │
         ▼                           ▼                           ▼
    ┌─────────────┐           ┌──────────────┐          ┌──────────────┐
    │ List of     │           │ Write: name  │          │ Write: args  │
    │ available   │           │ Read:  docs  │          │ Read: result │
    │ tool names  │           │              │          │              │
    │ (newline    │           │              │          │              │
    │  separated) │           │              │          │              │
    └─────────────┘           └──────────────┘          └──────────────┘


Example Session:
────────────────

   ┌─────────────┐     write "read"      ┌─────────────┐
   │   Client    │ ──────────────────▶   │  /tool/help │
   │             │ ◀──────────────────   │             │
   └─────────────┘     read docs         └─────────────┘

   ┌─────────────┐    write "/tmp/x"     ┌─────────────┐
   │   Client    │ ──────────────────▶   │ /tool/read  │
   │             │ ◀──────────────────   │             │
   └─────────────┘   read file contents  └─────────────┘
```

---

## 3. Security Model

### 3.1 Capability Attenuation Principle

```
┌───────────────────────────────────────────────────────────────────────────┐
│                    CAPABILITY ATTENUATION MODEL                           │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│   Parent Agent                    Child Agent (Sub-agent)                 │
│   ────────────                    ──────────────────────                  │
│                                                                           │
│   ┌─────────────────┐            ┌─────────────────┐                     │
│   │ Full Namespace  │            │ Reduced         │                     │
│   │                 │   spawn    │ Namespace       │                     │
│   │ • /appl         │ ────────▶  │                 │                     │
│   │ • /tmp          │            │ • /tmp          │                     │
│   │ • /dis          │            │ • /dis (subset) │                     │
│   │ • /tool (all)   │            │ • /tool (some)  │                     │
│   │ • /n/llm        │            │ • /n/llm        │                     │
│   │ • /dev          │            │ • /dev (limited)│                     │
│   └─────────────────┘            └─────────────────┘                     │
│                                                                           │
│   KEY INVARIANT:                                                          │
│   ══════════════                                                          │
│   child.namespace ⊆ parent.namespace                                      │
│                                                                           │
│   A child can NEVER have more capabilities than its parent.               │
│   This is enforced by the sandbox preparation + NEWNS mechanism.          │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Security Isolation Sequence

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       SPAWN SECURITY SEQUENCE                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  PARENT PROCESS (before spawn)                                              │
│  ─────────────────────────────                                              │
│                                                                             │
│     ┌────────────────────┐                                                  │
│     │ 1. Validate ID     │ ─▶ Reject: "../../../", ".", ".."               │
│     │    (sandboxid)     │    Accept: alphanumeric + hyphen only           │
│     └─────────┬──────────┘                                                  │
│               │                                                             │
│               ▼                                                             │
│     ┌────────────────────┐                                                  │
│     │ 2. Prepare Sandbox │ ─▶ Create /tmp/.veltro/sandbox/{id}/            │
│     │    (0700 perms)    │    with restrictive permissions                 │
│     └─────────┬──────────┘                                                  │
│               │                                                             │
│               ▼                                                             │
│     ┌────────────────────┐                                                  │
│     │ 3. Bind Paths      │ ─▶ Copy/bind only granted resources             │
│     │    (stat first)    │    Verify ownership before each bind            │
│     └─────────┬──────────┘                                                  │
│               │                                                             │
│               ▼                                                             │
│     ┌────────────────────┐                                                  │
│     │ 4. Preload Modules │ ─▶ Load tool.dis, subagent.dis                  │
│     │    (before NEWNS)  │    Initialize while paths exist                 │
│     └─────────┬──────────┘                                                  │
│               │                                                             │
│               │ spawn                                                       │
│               ▼                                                             │
│  ════════════════════════════════════════════════════════════════════      │
│                                                                             │
│  CHILD PROCESS (isolated)                                                   │
│  ────────────────────────                                                   │
│               │                                                             │
│               ▼                                                             │
│     ┌────────────────────┐                                                  │
│     │ 5. NEWPGRP         │ ─▶ Fresh process group                          │
│     │                    │    Empty service registry                       │
│     └─────────┬──────────┘                                                  │
│               │                                                             │
│               ▼                                                             │
│     ┌────────────────────┐                                                  │
│     │ 6. FORKNS          │ ─▶ Fork namespace for mutation                  │
│     │                    │                                                  │
│     └─────────┬──────────┘                                                  │
│               │                                                             │
│               ▼                                                             │
│     ┌────────────────────┐                                                  │
│     │ 7. NEWENV          │ ─▶ Empty environment                            │
│     │    (NOT FORKENV)   │    No inherited secrets                         │
│     └─────────┬──────────┘                                                  │
│               │                                                             │
│               ▼                                                             │
│     ┌────────────────────┐                                                  │
│     │ 8. Verify Safe FDs │ ─▶ Check FDs 0-2 point to safe endpoints        │
│     │                    │    Redirect to /dev/null if suspicious          │
│     └─────────┬──────────┘                                                  │
│               │                                                             │
│               ▼                                                             │
│     ┌────────────────────┐                                                  │
│     │ 9. NEWFD           │ ─▶ Keep only: 0, 1, 2, pipe, llm_fd             │
│     │    (keeplist)      │    All other FDs closed                         │
│     └─────────┬──────────┘                                                  │
│               │                                                             │
│               ▼                                                             │
│     ┌────────────────────┐                                                  │
│     │ 10. NODEVS         │ ─▶ Block #U, #p, #c device naming               │
│     │                    │    Prevents host filesystem escape              │
│     └─────────┬──────────┘                                                  │
│               │                                                             │
│               ▼                                                             │
│     ┌────────────────────┐                                                  │
│     │ 11. chdir(sandbox) │ ─▶ Enter prepared sandbox directory             │
│     │                    │                                                  │
│     └─────────┬──────────┘                                                  │
│               │                                                             │
│               ▼                                                             │
│     ┌────────────────────┐                                                  │
│     │ 12. NEWNS          │ ─▶ CRITICAL: Sandbox becomes /                  │
│     │                    │    Nothing outside sandbox exists               │
│     └─────────┬──────────┘                                                  │
│               │                                                             │
│               ▼                                                             │
│     ┌────────────────────┐                                                  │
│     │ 13. Execute Task   │ ─▶ Run subagent loop with pre-loaded tools      │
│     │                    │    No capability checks needed                  │
│     └────────────────────┘    Namespace IS the capability                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.3 Sandbox Directory Structure

```
/tmp/.veltro/
├── sandbox/
│   └── {sandboxid}/          ◀── After NEWNS, this becomes /
│       │
│       ├── dis/
│       │   ├── lib/
│       │   │   ├── bufio.dis     ◀── Copied (not bound)
│       │   │   ├── string.dis    ◀── Essential for subagent
│       │   │   └── arg.dis
│       │   │
│       │   ├── veltro/
│       │   │   ├── tools/
│       │   │   │   ├── read.dis  ◀── Only granted tools
│       │   │   │   ├── list.dis
│       │   │   │   └── ...
│       │   │   ├── subagent.dis
│       │   │   └── nsconstruct.dis
│       │   │
│       │   └── sh.dis            ◀── Only if trusted=1
│       │
│       ├── dev/
│       │   ├── cons              ◀── Bound from /dev/cons
│       │   └── null              ◀── Bound from /dev/null
│       │
│       ├── tool/                 ◀── Mount point for tools9p
│       │
│       ├── tmp/                  ◀── Writable scratch space
│       │
│       ├── n/
│       │   └── llm/              ◀── LLM access (if granted)
│       │       ├── config_model
│       │       ├── config_temperature
│       │       ├── config_system
│       │       └── ask           ◀── Query endpoint
│       │
│       └── [granted paths]       ◀── Copied from parent
│           └── appl/
│               └── veltro/...
│
└── audit/
    └── {sandboxid}.ns            ◀── Bind operation log
```

### 3.4 Security Properties Matrix

```
┌────────────────────────┬─────────────────────────────────────────────────────┐
│     THREAT             │              MITIGATION                              │
├────────────────────────┼─────────────────────────────────────────────────────┤
│                        │                                                     │
│ Host FS Escape (#U)    │ NODEVS before sandbox entry blocks device naming    │
│                        │                                                     │
├────────────────────────┼─────────────────────────────────────────────────────┤
│                        │                                                     │
│ Environment Secrets    │ NEWENV (not FORKENV) creates empty environment      │
│                        │                                                     │
├────────────────────────┼─────────────────────────────────────────────────────┤
│                        │                                                     │
│ FD Inheritance Leak    │ NEWFD with explicit keep-list prunes all others     │
│                        │ verifysafefds() checks FDs 0-2 before NEWFD         │
│                        │                                                     │
├────────────────────────┼─────────────────────────────────────────────────────┤
│                        │                                                     │
│ Service Registry Leak  │ NEWPGRP first creates fresh process group with      │
│                        │ empty srv registry (no inherited services)          │
│                        │                                                     │
├────────────────────────┼─────────────────────────────────────────────────────┤
│                        │                                                     │
│ Namespace Expansion    │ Impossible: NEWNS makes sandbox become /, and       │
│                        │ there's no "unbind and get more" operation          │
│                        │                                                     │
├────────────────────────┼─────────────────────────────────────────────────────┤
│                        │                                                     │
│ Path Traversal         │ validatesandboxid() rejects /, .., special chars    │
│                        │ Only alphanumeric + hyphen allowed                  │
│                        │                                                     │
├────────────────────────┼─────────────────────────────────────────────────────┤
│                        │                                                     │
│ Race Conditions        │ sys->create() fails if directory exists             │
│                        │ No TOCTOU between validate and create               │
│                        │                                                     │
├────────────────────────┼─────────────────────────────────────────────────────┤
│                        │                                                     │
│ Shell Injection        │ Untrusted agents use safeexec (loads .dis directly) │
│                        │ No shell metacharacter interpretation               │
│                        │                                                     │
├────────────────────────┼─────────────────────────────────────────────────────┤
│                        │                                                     │
│ Audit Evasion          │ All bind operations logged to audit file before     │
│                        │ task execution begins                               │
│                        │                                                     │
├────────────────────────┼─────────────────────────────────────────────────────┤
│                        │                                                     │
│ Stale Sandbox Attack   │ cleanstalesandboxes() removes sandboxes older       │
│                        │ than 5 minutes on init                              │
│                        │                                                     │
└────────────────────────┴─────────────────────────────────────────────────────┘
```

### 3.5 Trusted vs Untrusted Agent Model

```
                    AGENT TRUST LEVELS
    ════════════════════════════════════════════

    ┌─────────────────────────────────────────────────────────────┐
    │                    UNTRUSTED (default)                       │
    │                                                             │
    │  ┌─────────────────┐        ┌─────────────────┐            │
    │  │  Agent Process  │        │   Available     │            │
    │  │                 │        │   Resources     │            │
    │  │  • No sh.dis    │        │                 │            │
    │  │  • No exec      │        │  • Granted      │            │
    │  │  • safeexec     │───────▶│    tools only   │            │
    │  │    only         │        │  • Granted      │            │
    │  │                 │        │    paths only   │            │
    │  └─────────────────┘        │  • No /net      │            │
    │                             │  • No /srv      │            │
    │                             │  • No /prog     │            │
    │                             └─────────────────┘            │
    │                                                             │
    │  CANNOT:                                                    │
    │  ✗ Execute shell commands                                   │
    │  ✗ Use shell metacharacters                                 │
    │  ✗ Access network (unless explicitly granted)               │
    │  ✗ Discover processes                                       │
    │                                                             │
    └─────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────┐
    │                    TRUSTED (trusted=1)                       │
    │                                                             │
    │  ┌─────────────────┐        ┌─────────────────┐            │
    │  │  Agent Process  │        │   Available     │            │
    │  │                 │        │   Resources     │            │
    │  │  • sh.dis       │        │                 │            │
    │  │    bound        │        │  • All granted  │            │
    │  │  • exec tool    │───────▶│    tools        │            │
    │  │    enabled      │        │  • Shell        │            │
    │  │  • shellcmds    │        │    commands     │            │
    │  │    available    │        │  • echo, cat,   │            │
    │  └─────────────────┘        │    etc. (if     │            │
    │                             │    granted)     │            │
    │                             └─────────────────┘            │
    │                                                             │
    │  CAN (if granted):                                          │
    │  ✓ Execute shell commands via exec tool                     │
    │  ✓ Use granted shell utilities (echo, cat, ls...)           │
    │  ✓ Chain commands with shell                                │
    │                                                             │
    │  STILL CANNOT:                                              │
    │  ✗ Access paths not in namespace                            │
    │  ✗ Escape sandbox via #U                                    │
    │  ✗ Inherit parent's environment                             │
    │                                                             │
    └─────────────────────────────────────────────────────────────┘
```

---

## 4. Agent Lifecycle

### 4.1 Main Agent Loop

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         VELTRO AGENT LIFECYCLE                               │
└─────────────────────────────────────────────────────────────────────────────┘

      User: veltro "task description"
                    │
                    ▼
         ┌──────────────────────┐
         │   1. INITIALIZATION  │
         │   ─────────────────  │
         │   • Load modules     │
         │   • Parse args       │
         │   • Check /tool,     │
         │     /n/llm mounted   │
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │ 2. DISCOVER NS       │
         │ ─────────────────    │
         │ • Read /tool/tools   │
         │ • List accessible    │◀─────── Namespace = Capabilities
         │   paths              │
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │ 3. ASSEMBLE PROMPT   │
         │ ─────────────────    │
         │ • Load system.txt    │
         │ • Add namespace      │
         │ • Add tool docs      │
         │ • Add task           │
         └──────────┬───────────┘
                    │
                    ▼
    ┌───────────────────────────────────────┐
    │           AGENT LOOP                  │
    │ ┌───────────────────────────────────┐ │
    │ │                                   │ │
    │ │  ┌─────────────────┐              │ │
    │ │  │ 4. QUERY LLM    │              │ │
    │ │  │ • Write /n/llm/ │              │ │
    │ │  │   ask           │              │ │
    │ │  │ • Read response │              │ │
    │ │  └────────┬────────┘              │ │
    │ │           │                       │ │
    │ │           ▼                       │ │
    │ │  ┌─────────────────┐              │ │
    │ │  │ 5. PARSE ACTION │              │ │
    │ │  │ • Extract tool  │              │ │
    │ │  │   name          │              │ │
    │ │  │ • Parse heredoc │              │ │
    │ │  │   if present    │              │ │
    │ │  └────────┬────────┘              │ │
    │ │           │                       │ │
    │ │     ┌─────┴─────┐                 │ │
    │ │     │           │                 │ │
    │ │     ▼           ▼                 │ │
    │ │  ┌──────┐    ┌──────────────┐     │ │
    │ │  │ DONE │    │ TOOL CALL    │     │ │
    │ │  │      │    │              │     │ │
    │ │  └──┬───┘    └──────┬───────┘     │ │
    │ │     │               │             │ │
    │ │     │         ┌─────┴─────┐       │ │
    │ │     │         ▼           │       │ │
    │ │     │  ┌─────────────┐    │       │ │
    │ │     │  │ 6. EXECUTE  │    │       │ │
    │ │     │  │ • Open      │    │       │ │
    │ │     │  │   /tool/X   │    │       │ │
    │ │     │  │ • Write args│    │       │ │
    │ │     │  │ • Read      │    │       │ │
    │ │     │  │   result    │    │       │ │
    │ │     │  └──────┬──────┘    │       │ │
    │ │     │         │           │       │ │
    │ │     │         ▼           │       │ │
    │ │     │  ┌─────────────┐    │       │ │
    │ │     │  │ 7. FEEDBACK │    │       │ │
    │ │     │  │ "Tool X     │────┘       │ │
    │ │     │  │  returned:" │            │ │
    │ │     │  └─────────────┘            │ │
    │ │     │                             │ │
    │ └─────┼─────────────────────────────┘ │
    │       │        Loop until DONE        │
    │       │        or max steps           │
    └───────┼───────────────────────────────┘
            │
            ▼
         ┌──────────────────────┐
         │ 8. COMPLETION        │
         │ • Print final result │
         │ • Exit               │
         └──────────────────────┘
```

### 4.2 Tool Execution Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       TOOL EXECUTION DETAIL                                  │
└─────────────────────────────────────────────────────────────────────────────┘

   Agent                  tools9p                 Tool Module
   (veltro.b)             (9P Server)             (e.g. read.b)
       │                      │                        │
       │  open /tool/read     │                        │
       │ ────────────────────▶│                        │
       │                      │                        │
       │  write("/tmp/x.b")   │                        │
       │ ────────────────────▶│                        │
       │                      │  loadtool() if needed  │
       │                      │───────────────────────▶│
       │                      │                        │
       │                      │  mod->init()           │
       │                      │───────────────────────▶│
       │                      │                        │
       │                      │  mod->exec(args)       │
       │                      │───────────────────────▶│
       │                      │                        │
       │                      │◀───────────────────────│
       │                      │  return result         │
       │                      │                        │
       │  seek(0)             │                        │
       │ ────────────────────▶│                        │
       │                      │                        │
       │  read()              │                        │
       │ ────────────────────▶│                        │
       │                      │                        │
       │◀─────────────────────│                        │
       │  file contents       │                        │
       │  (with line nums)    │                        │
       │                      │                        │

   Note: tools9p processes 9P messages sequentially (single-threaded).
         Tool modules are loaded lazily on first use.
```

---

## 5. Data Flow

### 5.1 LLM Communication

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       LLM COMMUNICATION FLOW                                 │
└─────────────────────────────────────────────────────────────────────────────┘

        Veltro Agent                    /n/llm (9P Interface)           LLM API
             │                                   │                          │
             │     1. Open /n/llm/ask            │                          │
             │──────────────────────────────────▶│                          │
             │                                   │                          │
             │     2. Write prompt               │                          │
             │──────────────────────────────────▶│                          │
             │                                   │   3. HTTP POST to        │
             │                                   │      API endpoint        │
             │                                   │─────────────────────────▶│
             │                                   │                          │
             │                                   │◀─────────────────────────│
             │                                   │   4. API response        │
             │                                   │                          │
             │     5. Read response              │                          │
             │◀──────────────────────────────────│                          │
             │                                   │                          │


   Prompt Structure:
   ─────────────────

   ┌────────────────────────────────────────────────────────────┐
   │  [System Prompt - from /lib/veltro/system.txt]            │
   │                                                            │
   │  == Your Namespace ==                                      │
   │  TOOLS: read, list, find, ...                             │
   │  PATHS: /, /tool, /tmp                                    │
   │                                                            │
   │  == Tool Documentation ==                                  │
   │  ### read                                                  │
   │  Read - Read file contents...                             │
   │                                                            │
   │  == Reminders ==                                          │
   │  [Context-specific reminders based on available tools]    │
   │                                                            │
   │  == Task ==                                               │
   │  [User's task description]                                │
   │                                                            │
   │  Respond with a tool invocation or DONE if complete.      │
   └────────────────────────────────────────────────────────────┘
```

### 5.2 Result Streaming

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       LARGE RESULT HANDLING                                  │
└─────────────────────────────────────────────────────────────────────────────┘

                              STREAM_THRESHOLD = 4096 bytes

   Tool Result                     Decision                     Output
   ───────────                     ────────                     ──────

   ┌─────────────┐           len(result) < 4096           ┌─────────────┐
   │  Small      │ ─────────────────────────────────────▶ │ Direct      │
   │  Result     │         Return directly                │ to LLM      │
   │  (< 4KB)    │                                        │             │
   └─────────────┘                                        └─────────────┘


   ┌─────────────┐           len(result) >= 4096          ┌─────────────┐
   │  Large      │ ─────────────────────────────────────▶ │ Write to    │
   │  Result     │     1. Write to scratch file           │ scratch     │
   │  (>= 4KB)   │     2. Return pointer                  │             │
   └─────────────┘                                        └──────┬──────┘
                                                                 │
                                                                 ▼
                                                          ┌─────────────┐
                                                          │ "(output    │
                                                          │  written to │
                                                          │  /tmp/      │
                                                          │  veltro/    │
                                                          │  scratch/   │
                                                          │  step3.txt, │
                                                          │  45678      │
                                                          │  bytes)"    │
                                                          └─────────────┘
```

---

## 6. Sub-Agent Spawning

### 6.1 Spawn Tool Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        SUB-AGENT SPAWN FLOW                                  │
└─────────────────────────────────────────────────────────────────────────────┘

   Parent Agent                         spawn.b                    Child Agent
        │                                  │                            │
        │  Spawn tools=read,list           │                            │
        │  paths=/appl -- "List files"     │                            │
        │ ────────────────────────────────▶│                            │
        │                                  │                            │
        │                    1. Parse args (tools, paths, llmconfig)    │
        │                                  │                            │
        │                    2. Generate sandbox ID                     │
        │                                  │                            │
        │                    3. Preload modules (BEFORE NEWNS!)         │
        │                       • Load tool modules                     │
        │                       • Load subagent module                  │
        │                       • Initialize all                        │
        │                                  │                            │
        │                    4. Prepare sandbox                         │
        │                       • Create directory structure            │
        │                       • Copy/bind granted resources           │
        │                       • Write LLM config files                │
        │                       • Emit audit log                        │
        │                                  │                            │
        │                    5. Open LLM FD (survives NEWNS)            │
        │                                  │                            │
        │                    6. Create pipe for result                  │
        │                                  │                            │
        │                    7. spawn runchild()                        │
        │                                  │                            │
        │                                  │ ─────────────────────────▶ │
        │                                  │                            │
        │                                  │      [Security sequence    │
        │                                  │       NEWPGRP, FORKNS,     │
        │                                  │       NEWENV, NEWFD,       │
        │                                  │       NODEVS, chdir,       │
        │                                  │       NEWNS]               │
        │                                  │                            │
        │                                  │                            │
        │                                  │      subagent->runloop()   │
        │                                  │                            │
        │                                  │          ┌────────────┐    │
        │                                  │          │ Query LLM  │    │
        │                                  │          │ via FD     │    │
        │                                  │          └─────┬──────┘    │
        │                                  │                │           │
        │                                  │          ┌─────┴──────┐    │
        │                                  │          │ Call tool  │    │
        │                                  │          │ (preloaded)│    │
        │                                  │          └─────┬──────┘    │
        │                                  │                │           │
        │                                  │          ┌─────┴──────┐    │
        │                                  │          │ Loop until │    │
        │                                  │          │ DONE       │    │
        │                                  │          └─────┬──────┘    │
        │                                  │                │           │
        │                                  │ ◀─────────────────────────│
        │                                  │  Write result to pipe     │
        │                                  │                            │
        │ ◀────────────────────────────────│                            │
        │        Return result             │                            │
        │                                  │                            │
        │                    8. Cleanup sandbox                         │
        │                                  │                            │
```

### 6.2 Module Preloading Strategy

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     WHY MODULE PRELOADING IS CRITICAL                        │
└─────────────────────────────────────────────────────────────────────────────┘

   Problem: NEWNS makes sandbox become / - paths outside don't exist!

   ════════════════════════════════════════════════════════════════════════

   WITHOUT PRELOADING (BROKEN):
   ─────────────────────────────

   Parent NS                             Child NS (after NEWNS)
   ────────                              ──────────────────────

   /dis/veltro/tools/read.dis  ─────▶    NOT VISIBLE
                                         (doesn't exist in sandbox)

   Child tries: load Tool "/dis/veltro/tools/read.dis"
   Result: FAILS - path doesn't exist!

   ════════════════════════════════════════════════════════════════════════

   WITH PRELOADING (CORRECT):
   ──────────────────────────

   1. BEFORE spawn:
      ┌─────────────────────────────────┐
      │  mod = load Tool path           │  ◀── Loads while paths exist
      │  mod->init()                    │  ◀── Initializes modules
      │  preloadedtools = mod :: list   │  ◀── Stores in memory
      └─────────────────────────────────┘

   2. AFTER NEWNS:
      ┌─────────────────────────────────┐
      │  for pt := preloadedtools; ... │  ◀── Uses stored references
      │      (hd pt).mod->exec(args)   │  ◀── Module already in memory
      └─────────────────────────────────┘

   The module is already loaded in memory - no filesystem access needed!

   ════════════════════════════════════════════════════════════════════════

   ALSO: LLM FD Survives NEWNS
   ────────────────────────────

   ┌─────────────────┐                    ┌─────────────────┐
   │ Open FD before  │                    │ FD still valid  │
   │ spawn:          │    ───NEWNS───▶    │ after NEWNS:    │
   │                 │                    │                 │
   │ llmfd = open(   │                    │ llmfd.fd is     │
   │  "/n/llm/ask")  │                    │ still usable    │
   │                 │                    │                 │
   │ llmfdnum =      │                    │ llmfd = fildes( │
   │   llmfd.fd      │                    │   llmfdnum)     │
   └─────────────────┘                    └─────────────────┘

   Binds don't survive NEWNS, but open FDs do!
```

---

## 7. Tool System

### 7.1 Tool Registry Model

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      TOOL REGISTRY (ALLOWLIST MODEL)                         │
└─────────────────────────────────────────────────────────────────────────────┘

   Traditional Model (Filter/Denylist):
   ─────────────────────────────────────

   ┌─────────────────────────────────────┐
   │  ALL POSSIBLE TOOLS                  │
   │  ┌──────┐ ┌──────┐ ┌──────┐        │
   │  │read  │ │write │ │exec  │  ...   │
   │  └──────┘ └──────┘ └──────┘        │
   │                                     │
   │  Apply filter: "block dangerous"    │
   │           │                         │
   │           ▼                         │
   │  ┌─────────────────────────────┐   │
   │  │ Available (filtered subset) │   │
   │  │ read, write, list, find ... │   │
   │  └─────────────────────────────┘   │
   │                                     │
   │  Problem: Must enumerate all        │
   │  dangers. What if you miss one?     │
   └─────────────────────────────────────┘


   Veltro Model (Build-up/Allowlist):
   ───────────────────────────────────

   ┌─────────────────────────────────────┐
   │  START WITH NOTHING                  │
   │                                     │
   │  ┌───────────────────────────────┐ │
   │  │         (empty)               │ │
   │  └───────────────────────────────┘ │
   │                                     │
   │  Explicitly add: tools9p read list  │
   │           │                         │
   │           ▼                         │
   │  ┌─────────────────────────────┐   │
   │  │ Available (explicit list)   │   │
   │  │ read, list                  │   │
   │  └─────────────────────────────┘   │
   │                                     │
   │  Benefit: Only what you asked for   │
   │  exists. No hidden capabilities.    │
   └─────────────────────────────────────┘
```

### 7.2 Heredoc Syntax for Multi-line Content

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         HEREDOC PARSING                                      │
└─────────────────────────────────────────────────────────────────────────────┘

   Input (LLM response):
   ──────────────────────

   xenith write 4 body <<EOF
   Line one of content
   Line two of content
   Line three
   EOF

   ────────────────────────────────────────────────────────────────────────

   Parse Steps:
   ─────────────

   1. Find "<<EOF" marker in args
      ┌────────────────────────────────────────┐
      │ args = "4 body <<EOF"                  │
      │ markerpos = 8  (position of "<<")      │
      │ delim = "EOF"                          │
      │ argsbefore = "4 body"                  │
      └────────────────────────────────────────┘

   2. Collect lines until delimiter
      ┌────────────────────────────────────────┐
      │ lines[0] = "Line one of content"       │
      │ lines[1] = "Line two of content"       │
      │ lines[2] = "Line three"                │
      │ lines[3] = "EOF"  ◀── stop here        │
      └────────────────────────────────────────┘

   3. Combine: argsbefore + content
      ┌────────────────────────────────────────┐
      │ result = "4 body Line one of content   │
      │          Line two of content           │
      │          Line three"                   │
      └────────────────────────────────────────┘

   ────────────────────────────────────────────────────────────────────────

   Why This Matters:
   ─────────────────

   WITHOUT heredoc (BROKEN):
   ┌────────────────────────────────────────┐
   │ xenith write 4 body Line one           │
   │ Line two                               │  ◀── NOT part of args!
   │ Line three                             │
   └────────────────────────────────────────┘

   Tool receives: "4 body Line one" (only first line)

   WITH heredoc (CORRECT):
   ┌────────────────────────────────────────┐
   │ xenith write 4 body <<EOF              │
   │ Line one                               │
   │ Line two                               │
   │ Line three                             │
   │ EOF                                    │
   └────────────────────────────────────────┘

   Tool receives: "4 body Line one\nLine two\nLine three"
```

---

## 8. Critical Review

### 8.1 Strengths

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ARCHITECTURAL STRENGTHS                              │
└─────────────────────────────────────────────────────────────────────────────┘

   ✓ ELEGANT SECURITY MODEL
   ─────────────────────────
   The "namespace IS capability" principle is genuinely elegant. Instead of
   complex ACLs, policy engines, or security layers, the sandbox simply
   doesn't contain what the agent can't access. This is:
   - Impossible to misconfigure (no permissions to set wrong)
   - Easy to audit (ls shows capabilities)
   - Matches mental model (what you see is what you can use)

   ✓ LEVERAGES OS PRIMITIVES
   ──────────────────────────
   Rather than reinventing security in userspace, Veltro uses Inferno's
   native primitives (NEWNS, NEWENV, NEWFD, NODEVS). These are:
   - Battle-tested (part of Plan 9/Inferno heritage)
   - Kernel-enforced (not bypassable by user code)
   - Composable (work well together)

   ✓ CAPABILITY ATTENUATION
   ─────────────────────────
   The invariant that child.capabilities ⊆ parent.capabilities is enforced
   structurally. A parent literally cannot grant what it doesn't have:
   - It can only bind from its own namespace
   - It can only copy files it can read
   - There's no privilege escalation path

   ✓ CLEAN SEPARATION OF CONCERNS
   ───────────────────────────────
   - nsconstruct.b: Sandbox preparation (parent's responsibility)
   - spawn.b: Security sequence execution (child's responsibility)
   - subagent.b: Agent loop (task execution)
   - tools9p.b: Tool provision (9P interface)

   ✓ STATELESS TOOL DESIGN
   ────────────────────────
   Tools are stateless modules with simple interface (init, name, doc, exec).
   State lives in the 9P server's per-fid data. Benefits:
   - Tools can be safely reloaded
   - No hidden state between invocations
   - Clear ownership of state

   ✓ PRELOADING SOLUTION
   ──────────────────────
   The insight that modules must be loaded BEFORE NEWNS is critical and
   handled correctly. The preloading strategy:
   - Loads all granted tools while paths exist
   - Stores module references in memory
   - Uses those references after NEWNS

   ✓ FD INHERITANCE FOR LLM
   ─────────────────────────
   Passing the LLM file descriptor number (not ref) through NEWFD is clever.
   The ref becomes invalid, but the FD number persists and can be wrapped
   again with sys->fildes(). This avoids needing /n/llm visible in sandbox.

   ✓ AUDIT LOGGING
   ────────────────
   All bind operations are logged to /tmp/.veltro/audit/{id}.ns before
   task execution. This provides:
   - Non-repudiation (what was actually granted)
   - Debugging aid (why something failed)
   - Security audit trail
```

### 8.2 Potential Weaknesses and Concerns

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      POTENTIAL WEAKNESSES                                    │
└─────────────────────────────────────────────────────────────────────────────┘

   ⚠ SINGLE-THREADED 9P SERVER
   ────────────────────────────
   tools9p processes 9P messages sequentially. While noted as intentional
   (avoiding race conditions), this could be a bottleneck if:
   - Tools take long to execute
   - Multiple agents share a tools9p instance
   - High-frequency tool calls are needed

   Mitigation: Each agent could spawn its own tools9p, but that adds overhead.

   ⚠ COPY VS BIND TRADEOFF
   ────────────────────────
   Because NEWNS discards binds where source is outside sandbox, nsconstruct
   COPIES files instead of binding them. This:
   - Uses more disk space for large files
   - Takes time for big directories
   - May have stale data if original changes

   Alternative: Could use union mounts or a different sandbox structure,
   but that would add complexity.

   ⚠ LIMITED TIMEOUT HANDLING
   ──────────────────────────
   spawn.b uses a 30-second hard timeout. This:
   - May be too short for complex tasks
   - May be too long for quick operations
   - Is not configurable per-spawn

   Should be: Configurable timeout in spawn arguments.

   ⚠ NO RESOURCE LIMITS
   ─────────────────────
   The current model doesn't limit:
   - Memory usage
   - CPU time
   - Disk writes
   - Number of sub-agents spawned

   A malicious or buggy agent could exhaust host resources.

   ⚠ NODEVS LIMITATIONS
   ─────────────────────
   NODEVS blocks #U, #p, #c but still allows:
   - #e (environment device) - but NEWENV makes this empty
   - #s (srv device) - but NEWPGRP makes registry empty
   - #| (pipe device) - could be used for IPC

   The #| allowance is probably fine, but should be explicitly documented.

   ⚠ STALE SANDBOX CLEANUP RACE
   ────────────────────────────
   cleanstalesandboxes() runs on init and removes sandboxes older than 5
   minutes. If a long-running spawn exists:
   - Parent might clean up while child is still running
   - Edge case, but possible

   Could be improved with: Lock file or process liveness check.

   ⚠ ERROR PROPAGATION
   ────────────────────
   Error messages are string-prefixed ("error:", "ERROR:"). This:
   - Is informal and prone to parsing issues
   - Doesn't distinguish error types
   - Could be confused with tool output that starts with "error"

   Better approach: Structured error types or out-of-band error channel.

   ⚠ LLM DEPENDENCY
   ─────────────────
   The agent assumes /n/llm is available but continues with warnings if not.
   Without LLM:
   - queryllm() returns empty string
   - Agent immediately terminates

   Could improve: Clearer failure mode or local fallback.
```

### 8.3 Design Decisions Worth Noting

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    NOTABLE DESIGN DECISIONS                                  │
└─────────────────────────────────────────────────────────────────────────────┘

   DECISION: Trust tools9p runs in same trust domain as agent
   ───────────────────────────────────────────────────────────
   From tools9p.b:
   "We intentionally DON'T use FORKNS here... tools run in the same trust
    domain as the agent."

   Implication: Tools can affect parent namespace. This is acceptable for
   single-agent use but would need rethinking for multi-tenant scenarios.

   DECISION: Copy instead of bind for granted paths
   ─────────────────────────────────────────────────
   From nsconstruct.b:
   "We COPY instead of BIND because NEWNS doesn't preserve binds."

   Trade-off: Correctness over efficiency. The alternative (keeping binds
   by not using NEWNS) would weaken the security model.

   DECISION: Preload ALL granted tools
   ────────────────────────────────────
   spawn.b loads every granted tool module before NEWNS, even if the
   sub-agent might not use all of them.

   Trade-off: Startup time vs runtime reliability. Loading unused tools
   wastes time, but ensures no tool fails to load when needed.

   DECISION: Single agent loop, not reactive
   ──────────────────────────────────────────
   The agent runs a synchronous loop: query → parse → execute → repeat.
   Not event-driven or reactive.

   Implication: Clean and simple, but can't handle concurrent events or
   interrupts. Suitable for batch tasks, less so for interactive agents.

   DECISION: Max 50/100 steps limit
   ─────────────────────────────────
   DEFAULT_MAX_STEPS = 50, MAX_MAX_STEPS = 100

   Rationale: Prevents runaway agents, but might limit legitimate long
   tasks. Should probably be adjustable or removed for trusted scenarios.
```

### 8.4 Recommendations for Future Development

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      RECOMMENDATIONS                                         │
└─────────────────────────────────────────────────────────────────────────────┘

   1. RESOURCE LIMITS
      ────────────────
      Add configurable resource limits to spawn:
      - maxmemory: Memory limit in bytes
      - maxcpu: CPU time limit in milliseconds
      - maxdisk: Disk write limit in bytes
      - maxchildren: Max nested sub-agents

   2. STRUCTURED ERRORS
      ──────────────────
      Replace string-prefix errors with structured types:

      Error: adt {
          code: int;
          tool: string;
          message: string;
      };

   3. CONFIGURABLE TIMEOUTS
      ──────────────────────
      Add timeout parameter to spawn tool:

      Spawn tools=read timeout=60000 -- "Long task"

   4. LAZY TOOL LOADING
      ──────────────────
      Instead of preloading all granted tools, preload only on demand:
      - Faster spawn startup
      - Lower memory usage
      - Requires handling load failures gracefully

   5. METRIC COLLECTION
      ──────────────────
      Add observability:
      - Tool execution times
      - LLM query latency
      - Step counts
      - Error rates

   6. FORMAL VERIFICATION
      ────────────────────
      The security model is elegant but could benefit from formal analysis:
      - Prove capability attenuation invariant
      - Model check for escape paths
      - Verify pctl sequence ordering

   7. RECOVERY MECHANISMS
      ────────────────────
      Add graceful handling for:
      - LLM unavailability (retry with backoff)
      - Tool failures (skip or retry)
      - Timeout (checkpoint and resume)
```

---

## 9. Summary

Veltro represents a thoughtful approach to AI agent security that leverages Inferno OS's unique namespace capabilities. The core insight—that the namespace itself can serve as the capability system—eliminates entire classes of security vulnerabilities by making unauthorized access structurally impossible rather than policy-blocked.

**Key Takeaways:**

1. **Security through structure**: Rather than filtering dangerous capabilities, Veltro builds up capabilities from nothing.

2. **OS primitives matter**: By using NEWNS, NEWENV, NEWFD, and NODEVS, Veltro gets kernel-enforced isolation without userspace security code.

3. **The preloading trick**: Loading modules before NEWNS is essential and non-obvious. This pattern could apply to other sandbox designs.

4. **FD inheritance**: File descriptors survive NEWNS even when binds don't—a useful property for passing communication channels to sandboxed processes.

5. **Auditability**: The combination of explicit tool lists, audit logs, and "namespace = capability" makes security state transparent.

The architecture is production-quality for single-agent use cases. Multi-tenant or high-security deployments would benefit from the additional hardening suggested in the recommendations section.

---

*Document generated: Analysis of Veltro Agent Framework*
*Codebase version: a33d327 (true sub-agent spawning)*
