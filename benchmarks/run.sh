#!/bin/bash
#
# Benchmark runner for Go-on-Dis vs Limbo vs Native Go
#
# Runs each benchmark in 5 modes:
#   1. Native Go (compiled natively with go build)
#   2. Go-on-Dis JIT (compiled with godis, run with emu -c1)
#   3. Go-on-Dis Interpreter (compiled with godis, run with emu -c0)
#   4. Limbo JIT (compiled with limbo, run with emu -c1)
#   5. Limbo Interpreter (compiled with limbo, run with emu -c0)
#
# Output: parseable benchmark results
#

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

EMU="$ROOT/emu/Linux/o.emu"
GODIS="$ROOT/tools/godis/godis"
LIMBO="$ROOT/Linux/amd64/bin/limbo"

# Timeout for emu runs (seconds)
EMU_TIMEOUT="${EMU_TIMEOUT:-120}"

# Benchmark list (names match across go/, limbo/, native/ dirs)
BENCHMARKS="fib sieve qsort strcat matrix channel nbody spawn bsearch closure interface map_ops"

# Allow filtering
if [ -n "$1" ]; then
    BENCHMARKS="$*"
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Result storage
declare -A RESULTS

echo "============================================================"
echo "  Go-on-Dis Benchmark Suite"
echo "  $(date)"
echo "  Platform: $(uname -m) $(uname -s)"
echo "============================================================"
echo ""

# Check prerequisites
ERRORS=0
if [ ! -x "$EMU" ]; then
    echo -e "${RED}ERROR: emu not found at $EMU${NC}"
    ERRORS=1
fi
if [ ! -x "$GODIS" ]; then
    echo -e "${YELLOW}Building godis compiler...${NC}"
    cd "$ROOT/tools/godis"
    go build ./cmd/godis/ || { echo -e "${RED}ERROR: failed to build godis${NC}"; ERRORS=1; }
    cd "$ROOT"
fi
if [ ! -x "$LIMBO" ]; then
    echo -e "${RED}ERROR: limbo not found at $LIMBO${NC}"
    ERRORS=1
fi
if ! command -v go &> /dev/null; then
    echo -e "${RED}ERROR: go not found${NC}"
    ERRORS=1
fi
if [ $ERRORS -ne 0 ]; then
    echo "Fix errors above before running benchmarks."
    exit 1
fi

# Create output directory for compiled files
OUTDIR="$SCRIPT_DIR/_build"
mkdir -p "$OUTDIR"

# Extract ms from BENCH output line
extract_ms() {
    echo "$1" | grep '^BENCH ' | head -1 | awk '{print $3}'
}

# Run a dis file on emu and capture output
run_emu() {
    local dis_path="$1"  # absolute path to .dis file
    local mode="$2"      # -c0 or -c1
    local inferno_path

    # Convert to Inferno-relative path
    inferno_path="${dis_path#$ROOT}"

    cd "$ROOT"
    local output
    output=$(timeout "$EMU_TIMEOUT" "$EMU" "-r." "$mode" "$inferno_path" 2>&1) || true
    echo "$output"
}

# Print a result line
print_result() {
    local bench="$1"
    local mode="$2"
    local ms="$3"
    local status="$4"

    if [ "$status" = "OK" ]; then
        printf "  %-20s %-25s %8s ms\n" "$bench" "$mode" "$ms"
    else
        printf "  %-20s %-25s %8s\n" "$bench" "$mode" "$status"
    fi
}

# Track failures
FAILURES=""

for bench in $BENCHMARKS; do
    echo -e "${CYAN}--- $bench ---${NC}"

    # 1. Native Go
    native_src="$SCRIPT_DIR/native/$bench.go"
    if [ -f "$native_src" ]; then
        native_bin="$OUTDIR/${bench}_native"
        if go build -o "$native_bin" "$native_src" 2>/dev/null; then
            output=$("$native_bin" 2>&1)
            ms=$(extract_ms "$output")
            if [ -n "$ms" ]; then
                print_result "$bench" "Native Go" "$ms" "OK"
                RESULTS["${bench}_native"]="$ms"
            else
                print_result "$bench" "Native Go" "" "PARSE ERROR"
                echo "    output: $output"
                FAILURES="$FAILURES ${bench}:native"
            fi
        else
            print_result "$bench" "Native Go" "" "BUILD FAIL"
            FAILURES="$FAILURES ${bench}:native-build"
        fi
    else
        print_result "$bench" "Native Go" "" "NO SOURCE"
    fi

    # 2. Go-on-Dis JIT
    go_src="$SCRIPT_DIR/go/$bench.go"
    go_dis="$OUTDIR/${bench}_go.dis"
    if [ -f "$go_src" ]; then
        if "$GODIS" -o "$go_dis" "$go_src" 2>/dev/null; then
            output=$(run_emu "$go_dis" "-c1")
            ms=$(extract_ms "$output")
            if [ -n "$ms" ]; then
                print_result "$bench" "Go-on-Dis JIT" "$ms" "OK"
                RESULTS["${bench}_go_jit"]="$ms"
            else
                print_result "$bench" "Go-on-Dis JIT" "" "RUNTIME ERROR"
                echo "    output: $(echo "$output" | head -5)"
                FAILURES="$FAILURES ${bench}:go-jit"
            fi
        else
            print_result "$bench" "Go-on-Dis JIT" "" "COMPILE FAIL"
            FAILURES="$FAILURES ${bench}:go-compile"
        fi
    fi

    # 3. Go-on-Dis Interpreter
    if [ -f "$go_dis" ]; then
        output=$(run_emu "$go_dis" "-c0")
        ms=$(extract_ms "$output")
        if [ -n "$ms" ]; then
            print_result "$bench" "Go-on-Dis Interp" "$ms" "OK"
            RESULTS["${bench}_go_interp"]="$ms"
        else
            print_result "$bench" "Go-on-Dis Interp" "" "RUNTIME ERROR"
            echo "    output: $(echo "$output" | head -5)"
            FAILURES="$FAILURES ${bench}:go-interp"
        fi
    fi

    # 4. Limbo JIT
    limbo_src="$SCRIPT_DIR/limbo/$bench.b"
    limbo_dis="$OUTDIR/${bench}_limbo.dis"
    if [ -f "$limbo_src" ]; then
        if "$LIMBO" -I "$ROOT/module" -o "$limbo_dis" "$limbo_src" 2>/dev/null; then
            output=$(run_emu "$limbo_dis" "-c1")
            ms=$(extract_ms "$output")
            if [ -n "$ms" ]; then
                print_result "$bench" "Limbo JIT" "$ms" "OK"
                RESULTS["${bench}_limbo_jit"]="$ms"
            else
                print_result "$bench" "Limbo JIT" "" "RUNTIME ERROR"
                echo "    output: $(echo "$output" | head -5)"
                FAILURES="$FAILURES ${bench}:limbo-jit"
            fi
        else
            print_result "$bench" "Limbo JIT" "" "COMPILE FAIL"
            FAILURES="$FAILURES ${bench}:limbo-compile"
        fi
    else
        print_result "$bench" "Limbo JIT" "" "NO SOURCE"
    fi

    # 5. Limbo Interpreter
    if [ -f "$limbo_dis" ]; then
        output=$(run_emu "$limbo_dis" "-c0")
        ms=$(extract_ms "$output")
        if [ -n "$ms" ]; then
            print_result "$bench" "Limbo Interp" "$ms" "OK"
            RESULTS["${bench}_limbo_interp"]="$ms"
        else
            print_result "$bench" "Limbo Interp" "" "RUNTIME ERROR"
            echo "    output: $(echo "$output" | head -5)"
            FAILURES="$FAILURES ${bench}:limbo-interp"
        fi
    fi

    echo ""
done

# Summary table
echo "============================================================"
echo "  SUMMARY TABLE (all times in milliseconds)"
echo "============================================================"
printf "%-14s %8s %10s %10s %10s %10s\n" "Benchmark" "Native" "GoDis JIT" "GoDis Int" "Limbo JIT" "Limbo Int"
printf "%-14s %8s %10s %10s %10s %10s\n" "---------" "------" "---------" "---------" "---------" "---------"

for bench in $BENCHMARKS; do
    native="${RESULTS[${bench}_native]:-"-"}"
    go_jit="${RESULTS[${bench}_go_jit]:-"-"}"
    go_interp="${RESULTS[${bench}_go_interp]:-"-"}"
    limbo_jit="${RESULTS[${bench}_limbo_jit]:-"-"}"
    limbo_interp="${RESULTS[${bench}_limbo_interp]:-"-"}"
    printf "%-14s %8s %10s %10s %10s %10s\n" "$bench" "$native" "$go_jit" "$go_interp" "$limbo_jit" "$limbo_interp"
done

echo ""

# Speedup ratios
echo "============================================================"
echo "  SPEEDUP RATIOS (relative to Go-on-Dis Interpreter)"
echo "============================================================"
printf "%-14s %10s %10s %10s %10s\n" "Benchmark" "Native" "GoDis JIT" "Limbo JIT" "Limbo Int"
printf "%-14s %10s %10s %10s %10s\n" "---------" "------" "---------" "---------" "---------"

for bench in $BENCHMARKS; do
    go_interp="${RESULTS[${bench}_go_interp]:-0}"
    if [ "$go_interp" = "0" ] || [ "$go_interp" = "-" ]; then
        printf "%-14s %10s %10s %10s %10s\n" "$bench" "-" "-" "-" "-"
        continue
    fi

    native="${RESULTS[${bench}_native]:-0}"
    go_jit="${RESULTS[${bench}_go_jit]:-0}"
    limbo_jit="${RESULTS[${bench}_limbo_jit]:-0}"
    limbo_interp="${RESULTS[${bench}_limbo_interp]:-0}"

    fmt_ratio() {
        local val="$1"
        if [ "$val" = "0" ] || [ "$val" = "-" ]; then
            echo "-"
        else
            echo "$go_interp $val" | awk '{if($2>0) printf "%.1fx", $1/$2; else print "-"}'
        fi
    }

    r_native=$(fmt_ratio "$native")
    r_go_jit=$(fmt_ratio "$go_jit")
    r_limbo_jit=$(fmt_ratio "$limbo_jit")
    r_limbo_interp=$(fmt_ratio "$limbo_interp")

    printf "%-14s %10s %10s %10s %10s\n" "$bench" "$r_native" "$r_go_jit" "$r_limbo_jit" "$r_limbo_interp"
done

echo ""

if [ -n "$FAILURES" ]; then
    echo -e "${RED}FAILURES:${NC}$FAILURES"
    echo ""
fi

echo "Done."
