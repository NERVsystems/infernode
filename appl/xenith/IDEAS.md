# Xenith: Ideas for AI Agent Desktop Environment

## Design Philosophy

**Minimal mechanism, maximal capability.**

Follow Plan 9 principles:
- Everything is a file
- Namespaces as schema (no JSON, no XML, no protocol buffers)
- Text streams for communication
- Small, composable tools
- Network transparency via 9P

The filesystem structure IS the API. An AI agent that understands files understands Xenith.

---

## Namespaces as Schema

From the NERV 9P paper: directory structure defines data model.

```
/mnt/xenith/
├── 1/                    # Window 1
│   ├── body              # Text content (read/write)
│   ├── ctl               # Commands (write)
│   ├── tag               # Tag line (read/write)
│   ├── sel               # Selection: "start end" or empty
│   ├── dot               # Cursor position: "line char"
│   ├── dirty             # "1" if unsaved, "0" if clean
│   ├── event             # Stream: "type data\n" per event
│   └── image             # Image path and dimensions
├── 2/
│   └── ...
├── new                   # Write to create window, returns id
├── focus                 # Current focus window id
└── event                 # Global event stream
```

No parsing required. `cat /mnt/xenith/1/sel` returns "42 67" - two numbers.
The schema is the path. The type is implied by convention.

---

## Plan 9 Languages for Graphics

Plan 9 had minimal, powerful text-to-graphics languages. Consider porting or adapting:

### pic - Diagrams
```pic
box "Start"
arrow
box "Process"
arrow
box "End"
```
Compiles to graphics. Ideal for AI-generated diagrams.

### grap - Graphs
```grap
frame ht 2 wid 3
label left "Sales"
label bot "Quarter"
draw solid
1 10
2 15
3 12
4 20
```
Simple data visualization from text.

### ideal - Geometric Constraints
Constraint-based drawing. Describe relationships, system solves layout.

### Implementation Approach
These could be Limbo modules that output to Draw primitives:
```
echo 'pic' > /mnt/xenith/1/ctl
# Now body interprets pic language and renders
```

Or a filter:
```
pic2draw < diagram.pic > /mnt/xenith/1/image
```

---

## Audio as Files

Plan 9 model:
```
/dev/audio      # Write PCM, read from mic
/dev/volume     # "master 80" etc
```

For Xenith/Inferno:
```
/dev/audio              # Raw PCM stream (like Plan 9)
/dev/audioctl           # Sample rate, channels, format
/mnt/speech/
├── say                 # Write text, speaks it
├── listen              # Read returns transcription
└── voice               # Voice settings
```

Implementation: Bridge to host audio via emu.
- macOS: Core Audio
- Linux: ALSA/PulseAudio
- Portable C in emu/port/

AI agent can:
```sh
echo "Hello, I found three errors in your code" > /mnt/speech/say
response=$(cat /mnt/speech/listen)
```

---

## Live Window Redraw

Current issue: Window contents don't redraw during resize/move operations.

Plan 9/Rio approach:
- Draw
- Notify on
Approach: Window reshape events trigger content redraw.

Implementation considerations:
- Redraw on every mouse move during drag = expensive
- Redraw on
- Compromise: Redraw at
- Or: Show outline during drag, redraw on

For image windows specifically:
- Scale-to-fit already implemented
- Just need redraw trigger on reshape completion

---

## Event Streams

Plan 9 style: Events as lines of text on a file.

```
/mnt/xenith/1/event
```

Reading blocks until event occurs. Events are lines:
```
key a
key Return
mouse 1 150 200
sel 10 25
focus
resize 800 600
```

Simple format: `type [args...]`

AI agent can:
```limbo
fd := sys->open("/mnt/xenith/1/event", Sys->OREAD);
while((n := sys->read(fd, buf, len buf)) > 0){
    (nil, event) := sys->tokenize(string buf[:n], " \n");
    case hd event {
    "key" => handlekey(hd tl event);
    "sel" => handlesel(hd tl event, hd tl tl event);
    ...
    }
}
```

No JSON parsing. No event loop frameworks. Just read lines.

---

## Selection and Dot

Expose cursor/selection state as files:

```
/mnt/xenith/1/sel       # "start end" byte offsets, or empty
/mnt/xenith/1/dot       # "line char" - cursor position
```

Writing sets selection:
```sh
echo "100 150" > /mnt/xenith/1/sel    # Select bytes 100-150
echo "5 0" > /mnt/xenith/1/dot        # Move cursor to line 5, char 0
```

AI knows what user is looking at by reading `sel`.
AI can direct attention by writing to `dot`.

---

## Structural Regular Expressions

Sam/Acme's power feature. Expose via ctl:

```sh
echo 'x/pattern/ c/replacement/' > /mnt/xenith/1/ctl
```

Commands:
- `x/re/` - for each match
- `y/re/` - for each non-match
- `g/re/` - if contains match
- `v/re/` - if doesn't contain match
- `c/text/` - change selection to text
- `a/text/` - append after selection
- `i/text/` - insert before selection

AI can perform complex edits with single commands:
```sh
# Change all "foo" to "bar" in function bodies
echo 'x/func.*{[^}]*}/ x/foo/ c/bar/' > /mnt/xenith/1/ctl
```

---

## Tool Discovery

No registry needed. Tools are directories:

```
/mnt/tools/
├── search/
│   ├── ctl         # Write query, read results
│   └── help        # Usage text
├── compile/
│   ├── ctl
│   └── help
└── format/
    ├── ctl
    └── help
```

AI discovers tools by listing `/mnt/tools/`.
AI learns usage by reading `help`.
AI invokes by writing to `ctl`.

---

## Minimal Enhancements Summary

| Feature | Files Added | Lines of Code | AI Benefit |
|---------|-------------|---------------|------------|
| Selection exposure | `sel`, `dot` | ~50 | Context awareness |
| Event stream | `event` | ~100 | React to user |
| Live redraw | - | ~20 | Better UX |
| Audio bridge | `/dev/audio` | ~300 (C) | Voice I/O |
| pic/grap | modules | ~500 each | Visualization |
| Native zlib | C module | ~200 | Performance |

Each addition follows the pattern:
- Expose as file
- Text in, text out
- Composable with existing tools

---

## Anti-Patterns to Avoid

- **JSON/XML** - Parse complexity, schema drift
- **Binary protocols** - Not inspectable, not composable
- **Large frameworks** - Dependency bloat
- **Special APIs** - Learn once, forget once
- **Configuration files** - Prefer runtime adjustment via ctl files

---

## The NERV 9P Vision

Xenith as universal AI interface:

1. **Any AI** can interact - just read/write files over 9P
2. **Any language** can be client - 9P libraries exist for all major languages
3. **Network transparent** - Remote AI, local Xenith (or vice versa)
4. **Inspectable** - `ls`, `cat`, `echo` for debugging
5. **Composable** - Shell pipelines work naturally
6. **Minimal** - Small code, small attack surface, small learning curve

The filesystem is the API.
The namespace is the schema.
Everything is a file.

---

## TODO: ARM64 JIT Compiler

**Priority: High** - Would dramatically improve all Limbo performance.

### Current State
- `libinterp/comp-arm64.c` is a stub (343 bytes) - returns 0, falls back to interpreter
- `libinterp/comp-amd64.c` is also a stub (865 bytes)
- No existing 64-bit JIT in Inferno - MIPS, PowerPC, SPARC JITs are all 32-bit
- ARM 32-bit JIT exists (`comp-arm.c`, 43KB) - closest reference

### Why It Matters
- Interpreter overhead is the root cause of slow PNG loading
- JIT compiles Dis bytecode to native machine code at module load time
- Estimated 10-100x speedup for CPU-bound Limbo code
- Benefits ALL Limbo code, not just image loading

### Implementation Scope
- ~35-40KB of C code (based on other JIT sizes)
- Map Dis VM operations to ARM64 instructions
- Handle 64-bit registers (X0-X30)
- ARM64 instruction encoding (different from ARM32)
- ARM64 ABI calling conventions
- 64-bit addressing modes

### Reference Files
- `libinterp/comp-arm.c` - ARM 32-bit JIT (closest architecturally)
- `libinterp/comp-386.c` - x86 JIT (most complete/tested)
- `libinterp/interp.h` - Dis VM structures
- `libinterp/isa.h` - Dis instruction set

### Benefits Beyond Xenith
- All Inferno applications run faster
- ARM64 Linux (Raspberry Pi 4/5) benefits too
- Makes Inferno competitive on modern hardware

---

## References

- Plan 9 Programmer's Manual: http://man.cat-v.org/plan_9/
- Inferno Programmer's Manual: http://www.intgat.tigress.co.uk/rmy/inferno/
- pic language: http://man.cat-v.org/plan_9/1/pic
- grap language: http://man.cat-v.org/plan_9/1/grap
- Structural Regular Expressions: http://doc.cat-v.org/bell_labs/structural_regexps/
- NERV 9P Paper: ../nerva-9p-paper/
- ARM64 Instruction Set: https://developer.arm.com/documentation/ddi0596/
