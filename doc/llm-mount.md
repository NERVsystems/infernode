# LLM Filesystem Mount for Inferno/Xenith

This document describes how to use the LLM 9P filesystem with Inferno and Xenith.

## Overview

The `llm9p` server provides a 9P filesystem interface to Claude LLM. When mounted,
it exposes files for sending prompts and reading responses.

## Prerequisites

1. The `llm9p` server must be running on port 5641:
   ```bash
   # From the host system
   cd /path/to/llm9p
   ./llm9p -addr :5641
   ```

2. Inferno/emu must be started through the shell to ensure the profile runs:
   ```bash
   cd /path/to/infernode/emu/MacOSX
   ./o.emu -r../.. sh -l -c 'xenith -t dark'
   ```

   **Important**: Running `./o.emu xenith` directly bypasses the shell profile,
   so the LLM mount won't be set up. Always use `sh -l -c 'command'` to ensure
   the profile executes.

## Filesystem Structure

Once mounted at `/n/llm`, the filesystem provides:

```
/n/llm/
├── prompt/
│   └── query      # Write prompts, then read responses from same file
└── ctl            # Control file (optional)
```

## Usage from Inferno Shell

### Simple Query
```sh
echo 'What is the capital of France' > /n/llm/prompt/query
cat /n/llm/prompt/query
```

The `query` file serves as both input and output - write your prompt, then read
the response from the same file.

### Quoting Notes

The Inferno shell uses single quotes for strings. Apostrophes in text need
special handling:

```sh
# This will FAIL (apostrophe interpreted as quote):
echo "What's the weather" > /n/llm/prompt/query

# This works (no apostrophe):
echo 'What is the weather' > /n/llm/prompt/query

# Or escape the apostrophe:
echo 'What'\''s the weather' > /n/llm/prompt/query
```

## Usage from Xenith

In Xenith, you can interact with the LLM by:

1. Opening `/n/llm/prompt/query` and writing your prompt
2. Reading back from `/n/llm/prompt/query` to see the response

Or use the shell window (Win) in Xenith to run the echo/cat commands.

## Troubleshooting

### Mount not appearing

If `/n/llm` doesn't exist:

1. Ensure llm9p server is running: `lsof -i :5641` (on host)
2. Ensure you started emu through the shell: `sh -l -c 'xenith'`
3. Check manually: `mount -A tcp!127.0.0.1!5641 /n/llm`

### Connection refused

The llm9p server is not running or not listening on the expected port.

### Parse errors

Shell quoting issue - avoid apostrophes or escape them properly.

## Configuration

The mount is configured in `/lib/sh/profile`:

```sh
# Mount LLM filesystem if server is running
mount -A tcp!127.0.0.1!5641 /n/llm >[2] /dev/null
```

To change the server address, edit this line.

## Architecture Notes

Xenith forks its namespace at startup (`sys->pctl(Sys->FORKNS, ...)`), which
means mounts done from within Xenith are isolated. The profile mount happens
before Xenith starts, ensuring the LLM mount is inherited by Xenith's
namespace.
