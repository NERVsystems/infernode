#!/bin/bash
#
# bench.sh - Dis VM Benchmark: JIT vs Interpreter
#
# Runs jitbench.dis in both interpreter (-c0) and JIT (-c1) modes
# and compares performance. Works on all supported platforms.
#
# Usage: bash scratchpad/bench.sh [runs]
#   runs: number of iterations (default: 3)
#
# Platforms: ARM64 Linux, ARM64 macOS, AMD64 Linux, AMD64 macOS
#
# Run from the infernode root directory.
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RUNS="${1:-3}"
TIMEOUT_SEC=300

# --- Platform detection ---

detect_platform() {
    local os arch
    os=$(uname -s)
    arch=$(uname -m)

    case "$os" in
        Darwin) EMUHOST=MacOSX ;;
        Linux)  EMUHOST=Linux ;;
        *)      echo "Unsupported OS: $os"; exit 1 ;;
    esac

    case "$arch" in
        aarch64|arm64) OBJTYPE=arm64 ;;
        x86_64|amd64)  OBJTYPE=amd64 ;;
        *)             echo "Unsupported arch: $arch"; exit 1 ;;
    esac

    PLATFORM="${OBJTYPE}-${EMUHOST}"

    # Find emulator binary
    EMU=""
    for name in o.emu Infernode; do
        if [ -x "$ROOT/emu/$EMUHOST/$name" ]; then
            EMU="$ROOT/emu/$EMUHOST/$name"
            break
        fi
    done

    if [ -z "$EMU" ]; then
        echo "ERROR: No emulator binary found in emu/$EMUHOST/"
        echo "Build with: cd emu/$EMUHOST && ROOT=../.. OBJTYPE=$OBJTYPE ../../${EMUHOST}/${OBJTYPE}/bin/mk"
        exit 1
    fi
}

detect_cpu() {
    case "$(uname -s)" in
        Darwin)
            sysctl -n machdep.cpu.brand_string 2>/dev/null || echo "unknown"
            ;;
        Linux)
            grep -m1 'model name\|Hardware' /proc/cpuinfo 2>/dev/null | sed 's/.*: //' || echo "unknown"
            ;;
    esac
}

detect_platform

if [ ! -f "$ROOT/dis/jitbench.dis" ]; then
    echo "ERROR: dis/jitbench.dis not found"
    echo "Compile: limbo -I module -o dis/jitbench.dis appl/cmd/jitbench.b"
    exit 1
fi

EMUROOT="-r$ROOT"

echo "=============================================="
echo "  Dis VM Benchmark: Interpreter vs JIT"
echo "=============================================="
echo "Platform: $PLATFORM ($(uname -m), $(uname -sr))"
echo "CPU:      $(detect_cpu)"
echo "Date:     $(date -Iseconds 2>/dev/null || date)"
echo "Emulator: $EMU"
echo "Runs:     $RUNS"
echo ""

# --- Run benchmarks ---

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

declare -a interp_totals
declare -a jit_totals

for run in $(seq 1 $RUNS); do
    echo "========== Run $run of $RUNS =========="
    echo ""

    INTERP_OUT="$TMPDIR/interp_${run}.txt"
    timeout $TIMEOUT_SEC "$EMU" $EMUROOT -c0 dis/jitbench.dis > "$INTERP_OUT" 2>&1 || true
    sleep 1

    JIT_OUT="$TMPDIR/jit_${run}.txt"
    timeout $TIMEOUT_SEC "$EMU" $EMUROOT -c1 dis/jitbench.dis > "$JIT_OUT" 2>&1 || true
    sleep 1

    INTERP_TOTAL=$(grep "Total Time:" "$INTERP_OUT" | grep -o '[0-9]*' || echo "0")
    JIT_TOTAL=$(grep "Total Time:" "$JIT_OUT" | grep -o '[0-9]*' || echo "0")
    interp_totals+=($INTERP_TOTAL)
    jit_totals+=($JIT_TOTAL)

    echo "  Interpreter: ${INTERP_TOTAL} ms"
    echo "  JIT:         ${JIT_TOTAL} ms"
    if [ "$JIT_TOTAL" -gt 0 ] && [ "$INTERP_TOTAL" -gt 0 ]; then
        SPEEDUP_X100=$(( (INTERP_TOTAL * 100) / JIT_TOTAL ))
        printf "  Speedup:     %d.%02dx\n" $((SPEEDUP_X100 / 100)) $((SPEEDUP_X100 % 100))
    fi
    echo ""
done

# --- Summary ---

echo "=============================================="
echo "  Summary ($PLATFORM)"
echo "=============================================="
echo ""

echo "Per-benchmark (last run):"
echo ""
printf "  %-30s %12s %12s %10s\n" "Benchmark" "Interp (ms)" "JIT (ms)" "Speedup"
printf "  %-30s %12s %12s %10s\n" "------------------------------" "------------" "------------" "----------"

BENCHMARKS=("Integer Arithmetic" "Loop with Array Access" "Function Calls" "Fibonacci" "Sieve of Eratosthenes" "Nested Loops")
for bench in "${BENCHMARKS[@]}"; do
    IT=$(grep -A1 "$bench" "$INTERP_OUT" | grep "Time:" | grep -o '[0-9]* ms' | grep -o '[0-9]*' || echo "?")
    JT=$(grep -A1 "$bench" "$JIT_OUT" | grep "Time:" | grep -o '[0-9]* ms' | grep -o '[0-9]*' || echo "?")
    if [ "$IT" != "?" ] && [ "$JT" != "?" ] && [ "$JT" -gt 0 ]; then
        SP_X100=$(( (IT * 100) / JT ))
        printf "  %-30s %12s %12s %7d.%02dx\n" "$bench" "$IT" "$JT" $((SP_X100 / 100)) $((SP_X100 % 100))
    else
        printf "  %-30s %12s %12s %10s\n" "$bench" "$IT" "$JT" "N/A"
    fi
done

echo ""
echo "Totals across $RUNS runs:"
echo ""
printf "  %-6s %12s %12s %10s\n" "Run" "Interp (ms)" "JIT (ms)" "Speedup"
printf "  %-6s %12s %12s %10s\n" "------" "------------" "------------" "----------"

SUM_I=0
SUM_J=0
for i in $(seq 0 $(( RUNS - 1 ))); do
    IT=${interp_totals[$i]}
    JT=${jit_totals[$i]}
    SUM_I=$(( SUM_I + IT ))
    SUM_J=$(( SUM_J + JT ))
    if [ "$JT" -gt 0 ]; then
        SP_X100=$(( (IT * 100) / JT ))
        printf "  %-6d %12d %12d %7d.%02dx\n" $((i+1)) $IT $JT $((SP_X100 / 100)) $((SP_X100 % 100))
    fi
done

echo ""
if [ "$SUM_J" -gt 0 ]; then
    AVG_I=$(( SUM_I / RUNS ))
    AVG_J=$(( SUM_J / RUNS ))
    SP_X100=$(( (AVG_I * 100) / AVG_J ))
    printf "  %-6s %12d %12d %7d.%02dx\n" "Avg" $AVG_I $AVG_J $((SP_X100 / 100)) $((SP_X100 % 100))
fi
echo ""
echo "=============================================="
