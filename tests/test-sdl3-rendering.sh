#!/bin/sh
#
# Regression test: SDL3 Batched Rendering Performance
#
# This test verifies that the SDL3 rendering optimization is working correctly.
# The optimization batches dirty rectangle updates into a single GPU upload per
# frame, rather than doing a blocking dispatch_sync() on every flushmemscreen().
#
# Background:
#   flushmemscreen() is called 100-1000+ times per frame during text-heavy
#   operations (directory listings, text selection). The naive implementation
#   called dispatch_sync() for each update, causing multi-second delays.
#
#   The fix accumulates dirty rectangles in flushmemscreen() (no sync) and
#   does a single batched upload in sdl3_mainloop() at ~60Hz.
#
# See: emu/port/draw-sdl3.c header comment for full architecture details.
#
# Test approach:
#   1. Verify the code structure has the batched pattern
#   2. Check that flushmemscreen() does NOT call dispatch_sync
#   3. Check that sdl3_mainloop() does the batched update
#

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SDL3_FILE="${SCRIPT_DIR}/../emu/port/draw-sdl3.c"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

passed=0
failed=0

check() {
    desc="$1"
    if eval "$2"; then
        printf "${GREEN}PASS${NC}: %s\n" "$desc"
        passed=$((passed + 1))
    else
        printf "${RED}FAIL${NC}: %s\n" "$desc"
        failed=$((failed + 1))
    fi
}

echo "=== SDL3 Batched Rendering Regression Test ==="
echo "File: $SDL3_FILE"
echo ""

# Check file exists
if [ ! -f "$SDL3_FILE" ]; then
    printf "${RED}FAIL${NC}: draw-sdl3.c not found\n"
    exit 1
fi

# Test 1: flushmemscreen should NOT contain dispatch_sync
# Extract flushmemscreen function body (starts with 'flushmemscreen(' and ends with '^}')
check "flushmemscreen() does not call dispatch_sync" \
    "! sed -n '/^flushmemscreen(/,/^}/p' '$SDL3_FILE' | grep -q 'dispatch_sync'"

# Test 2: flushmemscreen should accumulate dirty_pending
check "flushmemscreen() sets dirty_pending flag" \
    "grep -q 'dirty_pending = 1' '$SDL3_FILE'"

# Test 3: flushmemscreen should accumulate dirty_min/max
check "flushmemscreen() accumulates dirty_min_x" \
    "grep -q 'dirty_min_x = r.min.x' '$SDL3_FILE'"

# Test 4: sdl3_mainloop should check dirty_pending
check "sdl3_mainloop() checks dirty_pending" \
    "grep -q 'if (dirty_pending' '$SDL3_FILE'"

# Test 5: sdl3_mainloop should call SDL_UpdateTexture
check "sdl3_mainloop() calls SDL_UpdateTexture" \
    "awk '/sdl3_mainloop/,/^}$/' '$SDL3_FILE' | grep -q 'SDL_UpdateTexture'"

# Test 6: Architecture documentation present
check "Batched rendering architecture documented" \
    "grep -q 'Batched Dirty Rectangle Accumulation' '$SDL3_FILE'"

# Test 7: dirty_pending variable exists
check "dirty_pending variable declared" \
    "grep -q 'static volatile int dirty_pending' '$SDL3_FILE'"

# Test 8: No dispatch_sync in main rendering path
# (dispatch_sync should only appear in initialization, not in flushmemscreen)
flushmemscreen_dispatch=$(awk '/^flushmemscreen/,/^}/' "$SDL3_FILE" | grep -c 'dispatch_sync' || true)
check "No dispatch_sync in flushmemscreen body" \
    "[ '$flushmemscreen_dispatch' = '0' ]"

echo ""
echo "=== Results ==="
echo "Passed: $passed"
echo "Failed: $failed"

if [ "$failed" -gt 0 ]; then
    printf "${RED}TEST FAILED${NC}\n"
    exit 1
else
    printf "${GREEN}ALL TESTS PASSED${NC}\n"
    exit 0
fi
