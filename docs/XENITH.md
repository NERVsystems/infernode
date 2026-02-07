# Xenith - AI-Native Text Environment

Xenith is InferNode's default graphical user interface, a fork of the Acme editor optimized for AI agents and AI-human collaboration.

## Overview

Xenith maintains Acme's elegant text-based philosophy while adding capabilities specifically designed for AI integration:

- **9P Filesystem Interface** - Agents interact via standard file operations
- **Namespace Security** - Capability-based access control for AI containment
- **Observable Operations** - All agent activity visible to humans
- **Multimodal Support** - Text and images in the same environment
- **Dark Mode** - Modern theming with Catppuccin and custom colors

## Why Xenith for AI?

### The Filesystem is the API

Unlike JSON-RPC protocols (MCP) or REST APIs, Xenith exposes everything as files:

```
/mnt/xenith/
├── new                  # Create window (write returns ID)
├── focus                # Current focus window
└── <id>/
    ├── body             # Window text content
    ├── tag              # Title/command line
    ├── addr             # Text address (selection range)
    ├── ctl              # Control commands
    ├── event            # Event stream
    ├── colors           # Per-window theming
    └── image            # Image display control
```

An AI agent reads and writes files. No SDK required. No parsing required. LLMs understand filesystem operations naturally.

### Namespace-Based Security

Inferno®'s namespace model provides capability-based security:

```limbo
# Agent sees only what you bind:
sys->bind("/services/llm", "/llm", Sys->MREPL);
sys->bind("/tools/safe", "/tools", Sys->MREPL);
sys->bind("/tmp/scratch", "/scratch", Sys->MCREATE);
# Nothing else exists from agent's perspective
```

Benefits:
- **Explicit grants** - Agent cannot access unbounded resources
- **Observable** - Human sees all namespace bindings
- **Dynamic** - Grant or revoke capabilities at runtime
- **No escape** - Namespace boundary is enforced by kernel

### Human-AI Collaboration

Xenith windows are shared workspaces:

```
┌─ Source Code ─────────────┐   ┌─ Agent Dialog ────────────┐
│ func main() {             │   │ Human: Add error handling │
│   // Code here            │   │ Agent: I'll wrap this in  │
│ }                         │   │ a try-catch block...      │
└───────────────────────────┘   └───────────────────────────┘
```

- Human edits appear as events to the agent
- Agent modifications are visible immediately
- Middle-click executes commands (Acme-style)
- Both parties work on the same text

## Features

### Dark Mode and Theming

Xenith includes a modern dark theme (Catppuccin Mocha) and full color customization:

```bash
# Use dark theme
xenith -t catppuccin

# Traditional Acme colors
xenith -t plan9

# Custom colors via environment
export xenith_bg_text_0=#1E1E2E
export xenith_fg_text=#CDD6F4
```

20+ color variables for complete UI customization.

### Image Display

Xenith supports inline image display (PNG, PPM formats):

```bash
# Load image in window
echo 'image /path/to/diagram.png' > /mnt/xenith/1/ctl

# Query image info
cat /mnt/xenith/1/image
# Returns: /path/to/diagram.png 800 600

# Clear image, return to text
echo 'clearimage' > /mnt/xenith/1/ctl
```

Useful for AI-generated visualizations, charts, and diagrams.

### Event Streams

Agents can monitor user activity:

```limbo
fd := sys->open("/mnt/xenith/1/event", Sys->OREAD);
for(;;) {
    n := sys->read(fd, buf, len buf);
    # Event format: "type origin q0 q1 flags length text"
    # React to user edits, selections, commands
}
```

Event types include insertions, deletions, selections, and command executions.

## Architecture

### Comparison with Acme

| Aspect | Acme | Xenith |
|--------|------|--------|
| Colors | Hardcoded pastels | 20+ customizable + dark theme |
| Images | Text only | PNG/PPM display |
| Per-window UI | Standard | Custom color schemes |
| AI focus | Generic editor | Agent-friendly design |
| Code size | ~16K lines | ~17K lines |

### Key Modules

| Module | Purpose |
|--------|---------|
| `xenith.b` | Main entry, theming |
| `fsys.b` | 9P filesystem interface |
| `exec.b` | Command execution |
| `imgload.b` | Image loading (PNG/PPM) |
| `wind.b` | Window management |
| `text.b` | Text editing |

## Usage

### Starting Xenith

```bash
# From InferNode
xenith

# With dark theme
xenith -t catppuccin

# With specific font
xenith -f /fonts/pelm/unicode.9.font
```

### Agent Interaction Example

```python
# Pseudocode for an AI agent

# 1. Create a window
write("/mnt/xenith/new/ctl", "scratch")
# Returns window ID, e.g., "3"

# 2. Write content
write("/mnt/xenith/3/body", "Analysis results:\n...")

# 3. Read user selection
selection = read("/mnt/xenith/3/rdsel")

# 4. Monitor events
for event in read_stream("/mnt/xenith/3/event"):
    if event.type == "insert":
        # User added text, respond...
```

### Mouse Chords (Acme Heritage)

- **B1 (Left)** - Select text
- **B2 (Middle)** - Execute selection as command
- **B3 (Right)** - Search/look up selection

## Design Philosophy

Xenith follows the principle: **"Minimal mechanism, maximal capability."**

From the Plan 9 tradition:
- Everything is a file
- Text is the universal interface
- Composition over configuration
- Small, sharp tools

Applied to AI:
- Filesystem operations are universal
- Observable beats opaque
- Human remains in control
- Capabilities are explicit

## Future Directions

Planned enhancements (see `IDEAS.md`):

- **Graphics languages** - `pic`, `grap` for diagrams
- **Audio support** - Voice I/O via `/dev/audio`
- **Structured data** - JSON/tree viewers
- **Token accounting** - LLM cost tracking
- **ARM64 JIT** - Performance optimization

## See Also

### Learning Acme

Xenith inherits Acme's interaction model. These resources explain the fundamentals:

- [A Tour of Acme](https://www.youtube.com/watch?v=dP1xVpMPn8M) - Russ Cox's video tutorial (recommended starting point)
- [Acme homepage](http://acme.cat-v.org) - Documentation, resources, and community
- [Acme: A User Interface for Programmers](http://doc.cat-v.org/plan_9/4th_edition/papers/acme/) - Rob Pike's original paper

### Xenith-Specific

- `appl/xenith/DESIGN.md` - Detailed design rationale
- `appl/xenith/IDEAS.md` - Feature roadmap
- `appl/xenith/IMAGE.md` - Image implementation details

## License

MIT License (as per InferNode)
