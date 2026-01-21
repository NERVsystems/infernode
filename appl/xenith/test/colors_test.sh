#!/bin/sh
# Test per-window color control
# Run this script from within Xenith

XENITH=/mnt/xenith

# Find first window
WIN=$(ls $XENITH | grep -E '^[0-9]+$' | head -1)
if [ -z "$WIN" ]; then
    echo "No windows found"
    exit 1
fi

echo "Testing window $WIN"

# Test 1: Read default colors
echo "=== Test 1: Read defaults ==="
cat $XENITH/$WIN/colors
echo ""

# Test 2: Set tag background to red (warning)
echo "=== Test 2: Set red tag (warning) ==="
echo 'tagbg #F38BA8
tagfg #1E1E2E' > $XENITH/$WIN/colors
cat $XENITH/$WIN/colors
sleep 1

# Test 3: Set tag background to green (success)
echo "=== Test 3: Set green tag (success) ==="
echo 'tagbg #A6E3A1
tagfg #1E1E2E' > $XENITH/$WIN/colors
cat $XENITH/$WIN/colors
sleep 1

# Test 4: Set tag background to yellow (caution)
echo "=== Test 4: Set yellow tag (caution) ==="
echo 'tagbg #F9E2AF
tagfg #1E1E2E' > $XENITH/$WIN/colors
cat $XENITH/$WIN/colors
sleep 1

# Test 5: Set multiple colors (Catppuccin Mocha theme)
echo "=== Test 5: Multiple colors (dark theme) ==="
echo 'tagbg #1E1E2E
tagfg #CDD6F4
bodybg #1E1E2E
bodyfg #CDD6F4
bord #89B4FA' > $XENITH/$WIN/colors
cat $XENITH/$WIN/colors
sleep 1

# Test 6: Set body colors only
echo "=== Test 6: Body colors only ==="
echo 'bodybg #313244
bodyfg #BAC2DE' > $XENITH/$WIN/colors
cat $XENITH/$WIN/colors
sleep 1

# Test 7: Reset to defaults
echo "=== Test 7: Reset ==="
echo 'reset' > $XENITH/$WIN/colors
cat $XENITH/$WIN/colors

echo "=== Tests complete ==="
