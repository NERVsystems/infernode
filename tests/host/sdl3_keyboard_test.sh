#!/bin/sh
#
# Regression test: SDL3 Keyboard Handling
#
# This test verifies that the SDL3 backend properly handles keyboard input:
#
# 1. Ctrl+letter control characters (^A=1, ^H=8 backspace, etc.)
#    - Must use virtual keycode (event.key.key), not scancode
#    - Scancodes are physical positions and vary by keyboard layout
#
# 2. macOS Option+key composition (hold Option + press key)
#    - TEXT_INPUT receives composed character from OS (e.g., Option+t -> dagger)
#    - Should NOT be blocked when Alt is held
#
# 3. Plan 9 latin1 composition (press Option, release, type two chars)
#    - Alt/Option release sends Latin key (0xE06F) to enter compose mode
#    - Subsequent keypresses compose via latin1.h table
#
# 4. UTF-8 support in TEXT_INPUT
#    - Full UTF-8 decoding (1-4 byte sequences)
#
# See: emu/port/draw-sdl3.c keyboard event handlers
#

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SDL3_FILE="${SCRIPT_DIR}/../../emu/port/draw-sdl3.c"

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

echo "=== SDL3 Keyboard Handling Regression Test ==="
echo "File: $SDL3_FILE"
echo ""

# Check file exists
if [ ! -f "$SDL3_FILE" ]; then
    printf "${RED}FAIL${NC}: draw-sdl3.c not found\n"
    exit 1
fi

echo "--- Ctrl+Letter Control Characters ---"

# Test 1: Uses virtual keycode (event.key.key), not scancode for Ctrl+letter
check "Ctrl+letter uses event.key.key (virtual keycode)" \
    "grep -A5 'Ctrl+letter' '$SDL3_FILE' | grep -q 'event.key.key\|kc >= '"

# Test 2: Does NOT use scancode for Ctrl+letter calculation
check "Ctrl+letter does NOT use scancode for calculation" \
    "! grep -E 'SDL_KMOD_CTRL.*scancode.*SDL_SCANCODE_A' '$SDL3_FILE'"

# Test 3: Control character calculation uses lowercase 'a'-'z' range
check "Control char calculation: kc >= 'a' && kc <= 'z'" \
    "grep -q \"kc >= 'a' && kc <= 'z'\" '$SDL3_FILE'"

# Test 4: Correct formula: key = kc - 'a' + 1
check "Control char formula: kc - 'a' + 1" \
    "grep -q \"kc - 'a' + 1\" '$SDL3_FILE'"

echo ""
echo "--- macOS Option+Key Composition ---"

# Test 5: TEXT_INPUT handler exists
check "TEXT_INPUT event handler exists" \
    "grep -q 'SDL_EVENT_TEXT_INPUT' '$SDL3_FILE'"

# Test 6: TEXT_INPUT is NOT blocked when Alt is held (for macOS composition)
# The old code had: if (mods & SDL_KMOD_ALT) break;
# New code should NOT have this unconditional Alt block
check "TEXT_INPUT not unconditionally blocked by Alt" \
    "! grep -A10 'SDL_EVENT_TEXT_INPUT' '$SDL3_FILE' | grep -q 'SDL_KMOD_ALT.*break'"

# Test 7: Control characters (< 0x20) are filtered in TEXT_INPUT
check "TEXT_INPUT filters control characters (< 0x20)" \
    "grep -q 'text\[0\] < 0x20' '$SDL3_FILE'"

echo ""
echo "--- Plan 9 Latin1 Composition ---"

# Test 8: Latin key sent on Alt KEY_UP (release), not KEY_DOWN
check "Latin key sent on KEY_UP for Alt" \
    "grep -A5 'SDL_EVENT_KEY_UP' '$SDL3_FILE' | grep -q 'Latin'"

# Test 9: Alt scancodes (LALT/RALT) handled in KEY_UP
check "LALT/RALT handled in KEY_UP handler" \
    "grep -A10 'SDL_EVENT_KEY_UP' '$SDL3_FILE' | grep -q 'SDL_SCANCODE_LALT\|SDL_SCANCODE_RALT'"

# Test 10: Latin key NOT sent on Alt KEY_DOWN (would interfere with macOS composition)
# Look for Alt->Latin in the KEY_DOWN switch, which should NOT be there
check "Latin NOT sent on KEY_DOWN for Alt" \
    "! awk '/case SDL_EVENT_KEY_DOWN/,/case SDL_EVENT_KEY_UP/' '$SDL3_FILE' | grep -q 'case SDL_SCANCODE_LALT:.*Latin\|case SDL_SCANCODE_RALT:.*key = Latin'"

echo ""
echo "--- UTF-8 Support ---"

# Test 11: UTF-8 1-byte ASCII handling (0x80 check)
check "UTF-8 1-byte ASCII detection (0x80)" \
    "grep -q '0x80.*== 0' '$SDL3_FILE' || grep -q '\*text & 0x80' '$SDL3_FILE'"

# Test 12: UTF-8 2-byte sequence handling (0xE0/0xC0)
check "UTF-8 2-byte sequence handling" \
    "grep -q '0xE0.*0xC0\|0xC0.*0x1F' '$SDL3_FILE'"

# Test 13: UTF-8 3-byte sequence handling (0xF0/0xE0)
check "UTF-8 3-byte sequence handling" \
    "grep -q '0xF0.*0xE0\|0x0F.*12' '$SDL3_FILE'"

# Test 14: UTF-8 4-byte sequence handling (0xF8/0xF0)
check "UTF-8 4-byte sequence handling" \
    "grep -q '0xF8.*0xF0\|0x07.*18' '$SDL3_FILE'"

echo ""
echo "--- Event Modifier Handling ---"

# Test 15: Uses event.key.mod instead of SDL_GetModState() for KEY_DOWN
check "KEY_DOWN uses event.key.mod for modifiers" \
    "grep -A10 'SDL_EVENT_KEY_DOWN' '$SDL3_FILE' | grep -q 'event.key.mod'"

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
