# Lucifer Context Zone — Design Document

Status: **Draft / Brainstorm**

## Purpose

The context zone is the right panel (~25%) of Lucifer's three-zone layout. Its
primary purpose is **situational awareness** — giving the user a glanceable,
spatial understanding of what the Veltro agent is working with and where its
knowledge has limits.

This is deliberately *not* a log or chain-of-thought viewer. Logs are
sequential and require sustained attention. The context zone is a persistent
map of the agent's world that works in peripheral vision.

## Design Principles

1. **Namespace is context.** Every resource the agent can access is in its
   namespace. The context zone is a human-readable view of that namespace.

2. **Glanceable, not demanding.** The 25% strip is peripheral. Items have
   visual state (active, idle) but don't compete with conversation for
   attention.

3. **Non-modal expansion.** When the user needs to explore available resources
   or a large agent namespace, the panel expands without fully obscuring
   conversation or presentation. The agent keeps working.

4. **Abstract away namespaces.** Users think in "stuff," not mount points.
   Resources have plain-language names and descriptions. The underlying
   namespace mechanics are invisible.

## Summary View (25% Strip)

The default view the user sees 90% of the time. Two sections, top to bottom:

### Agent Resources (top)

A compact list of everything in the agent's namespace: files, directories,
tools, mounted services. Each item shows:

- **Name** — human-readable label ("API Docs", "search tool", "GPU Cluster")
- **Type indicator** — icon or glyph for file, directory, tool, service, etc.
- **Activity state** — visual indicator that responds to agent interaction:
  - **Flash/pulse** — the agent just accessed this resource (fades over ~2-3s)
  - **Steady/lit** — in namespace, available, not recently touched
  - The user's eye catches the flash without conscious monitoring

Ordered by most-recently-used, so active resources float to the top.

Example of what the user sees:

```
  AGENT CONTEXT
  ─────────────────────
  ● API Reference        docs    ← just accessed (bright)
  ● search               tool    ← just accessed (bright)
  ○ Project Specs        docs    ← idle
  ○ edit                 tool    ← idle
  ○ present              tool    ← idle
  ○ Production DB        service ← idle
```

The filled/bright dot (●) indicates recent activity. The open/dim dot (○)
indicates a mounted but idle resource. This distinction is enough — the user
doesn't need to know *how* the agent used it, just *that* it did.

**"Did the agent read a critical piece of documentation before starting this
task?"** Yes — it's at the top of the list and just flashed.

### Gaps (below resources)

Short phrases the agent has surfaced about its own blind spots:

```
  GAPS
  ─────────────────────
  ▲ No test coverage data
  ● API rate limits unknown
  ○ Auth flow undocumented
```

Gaps have relevance indicators (high ▲, medium ●, low ○) and are ordered by
relevance.

Gaps can be:

- **Resolved automatically** — the agent finds the information and the gap
  disappears
- **Resolved by the user** — the user provides a resource that addresses the
  gap
- **Persistent** — the gap remains as an acknowledged limitation

Gaps sit *above* the available-resources section, acting as a natural bridge:
the user reads what the agent is missing, then looks below to see what they
could provide.

### Available Resources (below gaps, collapsed by default)

A heading the user can click to expand:

```
  AVAILABLE ▸
```

When collapsed, it's a single line. This keeps the summary view compact. When
the user clicks, it either expands in-place (for a small catalog) or triggers
the expanded panel view (for a large one).

## Expanded View

Triggered by clicking the "Available" heading, or by a general expand gesture
on the context zone. The panel grows to ~50-60% width, partially overlapping
conversation and/or presentation without obscuring them entirely.

### Agent Context (expanded)

When the agent's resource list grows large (40+ files, 15 tools), the expanded
view shows the full namespace organized by category:

```
  AGENT CONTEXT (expanded)
  ────────────────────────────────
  Tools
    ● search         — web search
    ○ edit           — file editing
    ○ present        — artifact display
    ○ shell          — command execution

  Documents
    ● API Reference  — /docs/api/
    ○ Project Specs  — /docs/specs/
    ○ README         — /README.md

  Services
    ○ Production DB  — postgres, read-only
    ○ GPU Cluster    — 4x A100

  [collapse ▾]
```

### Available Resources (expanded)

Pre-configured resource catalog. Items have plain-language names and
descriptions, not paths:

```
  AVAILABLE RESOURCES
  ────────────────────────────────
  ┌─────────────────────────────┐
  │ + GPU Compute               │
  │   NVIDIA cluster, 4x A100   │
  ├─────────────────────────────┤
  │ + Project Docs              │
  │   Design specs, API refs    │
  ├─────────────────────────────┤
  │ + Production Logs           │
  │   Last 7 days, read-only    │
  ├─────────────────────────────┤
  │ + CI Pipeline               │
  │   Build status, test results│
  └─────────────────────────────┘
```

Click `+` to add a resource to the agent's namespace. The item animates from
the available list into the agent's resource list. The user never sees a mount
command or namespace path.

Similarly, in the agent context section, each resource could have a `−` or
removal action to unmount it from the agent's namespace.

## Resource Registry

Available resources are defined in a registry, curated by the team or
environment administrator:

```
/lib/veltro/resources/
  gpu-compute.resource
  project-docs.resource
  production-logs.resource
  ci-pipeline.resource
```

Each descriptor file:

```
name=GPU Compute
desc=NVIDIA cluster, 4x A100
mount=net!gpu-cluster!styx
type=compute
icon=server
```

Fields:

| Field   | Description                                      |
|---------|--------------------------------------------------|
| name    | Human-readable label shown in the UI             |
| desc    | Short description                                |
| mount   | Internal mount recipe (hidden from user)         |
| type    | Category: docs, tool, compute, service, data     |
| icon    | Visual indicator type                            |

This registry approach means:
- Teams curate what's available for their agents
- Users see a managed catalog, not the raw filesystem
- Adding new resources is a configuration task, not a code change
- The catalog can be scoped per-environment or per-team

## Activity Indicators

lucibridge already intercepts every tool call and file access. Activity
signaling works as a side effect of the existing bridge:

1. Agent calls a tool or reads a file → lucibridge handles the call
2. After the call, lucibridge writes an activity event to `/n/ui/activity/{id}/context/activity`
3. luciuisrv records the event and notifies lucifer
4. lucifer renders a flash/pulse on the corresponding resource item
5. The flash fades over ~2-3 seconds back to idle state

No additional token cost to the agent. No explicit "context update" tool
calls needed. The agent works normally; the UI observes.

## Per-Task Agents

Each activity in Lucifer can have its own Veltro agent with its own namespace.
When the user switches activities, the context zone updates to show the
current agent's world:

- Activity 1: coding agent with editor tools and source files
- Activity 2: research agent with search tools and document collections
- Activity 3: ops agent with deployment tools and production access

The context zone is always scoped to *this* agent. Switching activities
switches the entire context view.

## Interaction Model Summary

| Action                          | Gesture                              |
|---------------------------------|--------------------------------------|
| See agent's resources           | Glance at context zone (always shown)|
| Notice agent activity           | Peripheral flash on resource item    |
| Read agent's knowledge gaps     | Glance at gaps section               |
| Browse available resources      | Click "Available" to expand          |
| Add resource to agent           | Click `+` on an available resource   |
| Remove resource from agent      | Click `−` on an agent resource       |
| Explore large agent namespace   | Expand context zone                  |
| Return to summary               | Click collapse / click outside       |

## Open Questions

- **Ordering heuristics:** MRU works for activity, but should idle resources
  be alphabetical, by type, or by some relevance score?
- **Resource granularity:** When is something "a directory" versus individual
  files? If the user adds "Project Docs," does the agent see the directory
  or every file enumerated?
- **Gap lifecycle:** How long do resolved gaps stay visible before
  disappearing? Immediate removal vs. brief "resolved" state?
- **Expanded view trigger:** Click on heading vs. drag to resize vs. button?
- **Search/filter in expanded view:** Needed for large resource catalogs?
- **Resource permissions:** Read-only vs. read-write indicators?
- **Activity fade timing:** 2-3 seconds feels right but needs user testing.
