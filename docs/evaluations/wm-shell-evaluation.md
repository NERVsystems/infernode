# wm/shell Production Readiness Evaluation

**Date:** 2026-03-22
**Component:** `appl/wm/shell.b` (GUI terminal emulator)
**Binary:** `dis/wm/shell.dis` (19.7 KB)
**Lines of code:** ~1,803

## Verdict: Production-Ready

The wm/shell application is a complete, well-maintained GUI terminal emulator
for the Inferno shell. It is approved for production release with no blocking
issues.

## Feature Completeness

### Terminal Emulation
- Synthetic `/dev/cons` via file2chan
- Raw/cooked mode switching
- ANSI escape sequence filtering
- Text wrapping at window edge (added 2026-03-22)

### Input Handling
- **Keyboard:** Enter, Backspace, Ctrl-C (interrupt), Ctrl-D (EOF),
  Ctrl-U (clear line), Ctrl-W (delete word), Ctrl-L (clear screen),
  Ctrl-Q (quit), Up/Down (history), Page Up/Down (scroll)
- **Mouse:** B1 select, B2 paste, B3 context menu, scroll wheel
- **Hold mode:** ESC toggles output freeze

### Visual Features
- Live theme updates via lucitheme (dark mode support)
- HiDPI font support (k8 combined fonts)
- Cursor blinking (500ms timer)
- Scrollbar with drag support
- Status bar (mode, line count)
- Dynamic button bar (up to 20 buttons)
- Text selection with visual highlighting

### AI Agent Integration
- Read-only file-based IPC at `/tmp/veltro/shell/`
  - `body` — full transcript (read-only)
  - `input` — current input line (read-only)
- Veltro agents can observe but never inject commands
- Companion tool: `appl/veltro/tools/shell.b` (218 lines)

## Dependencies

All dependencies are stable, available system modules:

| Module | Purpose |
|--------|---------|
| sys.m | System calls, file I/O |
| draw.m | Graphics, fonts, images |
| wmclient.m | Window management |
| menu.m | Context menus |
| string.m | String utilities |
| sh.m | Shell interface |
| lucitheme.m | Theme colors |
| widget.m | Scrollbar, status bar |
| arg.m | Argument parsing |
| workdir.m | Working directory tracking |
| plumbmsg.m | Plumbing for word selection |

All modules have proper nil-load guards.

## Security

- **Namespace isolation:** `FORKNS` and `FORKFD` isolate synthetic console
- **Read-only IPC:** Agents cannot send commands through file interface
- **No injection vectors:** Button/command parsing handles quoted strings safely
- **File permissions:** IPC files created 644

## Error Handling

- Nil checks on all module loads with meaningful error messages
- Exception handling with `alt{}` for non-blocking channel dispatch
- Resource cleanup (fd = nil after use)
- Bounds checking on arrays and strings
- Graceful font/theme fallback chains

## Testing

- `tests/shell_tool_test.b` — 20+ test cases covering:
  - String parsing and line splitting
  - Line counting logic
  - Tail output logic
  - Command dispatch
  - Read target validation
  - File-based IPC simulation

## Known Minor Issues (Non-Blocking)

1. **BADOP cosmetic errors** — When commands fail through shell, BADOP messages
   may appear. Documented in `docs/SHELL-BADOP-ISSUE.md`. Harmless.

2. **Startup delay** — 50ms hardcoded sleep before shell starts to allow
   file2chan setup. Adequate on current hardware.

3. **Transcript limit** — Maximum 4,000 lines, trims to 3,000 (MAXLINES,
   TRIMLINES). Appropriate for interactive use.

4. **Button limit** — Maximum 20 dynamic buttons (MAXBUTTONS). Unlikely to
   hit in practice.

## Recent Maintenance History

- `785414d` (2026-03-22): Text wrapping, Send fix, selection lag reduction
- `0b4d535`: Live theme updates
- `847c0c9`: HiDPI font updates
- `b794d34`: Nil channel crash fix

## Checklist

- [x] Source code complete
- [x] Compiled binary present and current
- [x] All dependencies available
- [x] Error handling comprehensive
- [x] Security review passed
- [x] Tests present
- [x] No critical bugs
- [x] Active maintenance
- [x] Theme/accessibility support
