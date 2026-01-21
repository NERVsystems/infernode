#!/bin/sh
#
# Regression test: Temp file slot exhaustion detection
#
# This test checks that the Inferno tmp/ directory doesn't have exhausted
# temp file slots that would cause acme/xenith to fail with:
#   "can't create temp file file does not exist: file does not exist"
#
# Background:
#   tempfile() in appl/acme/disk.b and appl/xenith/disk.b creates files
#   named /tmp/{A-Z}{pid}.{user}{app} where:
#     - A-Z is iterated to find an available slot
#     - pid is the Inferno process ID (often 1 for standalone apps)
#     - user is first 4 chars of username
#     - app is "acme" or "xenith"
#
#   If all 26 slots (A-Z) for a given PID are exhausted, tempfile() fails.
#   This happens because ORCLOSE doesn't clean up files after crashes.
#
# See: docs/TEMPFILE-EXHAUSTION.md for full details
#

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TMP_DIR="${SCRIPT_DIR}/../tmp"
THRESHOLD=20  # Warn if any PID has this many slots used
MAX_SLOTS=26  # A-Z

RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

failed=0
warned=0

check_app() {
    app=$1
    pattern="*.pdfi${app}"

    echo "Checking ${app} temp files..."

    # Extract PIDs and count occurrences
    if [ -d "$TMP_DIR" ]; then
        # Get list of PIDs with their counts
        pid_counts=$(ls "$TMP_DIR"/$pattern 2>/dev/null | \
            sed "s|.*/[A-Z]\([0-9]*\)\..*|\\1|" | \
            sort | uniq -c | sort -rn)

        if [ -z "$pid_counts" ]; then
            echo "  No temp files found for ${app}"
            return 0
        fi

        echo "$pid_counts" | while read count pid; do
            if [ "$count" -ge "$MAX_SLOTS" ]; then
                printf "  ${RED}FAIL: PID %s has %d/%d slots EXHAUSTED${NC}\n" "$pid" "$count" "$MAX_SLOTS"
                echo "        Fix: rm ${TMP_DIR}/*${pid}.pdfi${app}"
                failed=1
            elif [ "$count" -ge "$THRESHOLD" ]; then
                printf "  ${YELLOW}WARN: PID %s has %d/%d slots used${NC}\n" "$pid" "$count" "$MAX_SLOTS"
                warned=1
            else
                printf "  ${GREEN}OK: PID %s has %d/%d slots${NC}\n" "$pid" "$count" "$MAX_SLOTS"
            fi
        done
    else
        echo "  Warning: tmp directory not found at $TMP_DIR"
    fi
}

echo "=== Temp File Slot Exhaustion Test ==="
echo "Directory: $TMP_DIR"
echo ""

check_app "acme"
echo ""
check_app "xenith"
echo ""

# Check total temp file count
total=$(ls "$TMP_DIR"/*.pdfi* 2>/dev/null | wc -l | tr -d ' ')
echo "Total temp files: $total"

if [ "$failed" = "1" ]; then
    echo ""
    printf "${RED}TEST FAILED: Exhausted temp file slots detected${NC}\n"
    echo "Run the suggested rm commands above to fix."
    exit 1
elif [ "$warned" = "1" ]; then
    echo ""
    printf "${YELLOW}TEST PASSED WITH WARNINGS: Slots getting full${NC}\n"
    exit 0
else
    echo ""
    printf "${GREEN}TEST PASSED: No slot exhaustion detected${NC}\n"
    exit 0
fi
