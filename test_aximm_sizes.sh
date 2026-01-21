#!/bin/bash
# Systematic test of AXIMM array sizes

for size in 8 16 24 32 34 36 38 40 48 64; do
    echo "========================================="
    echo "Testing array size: $size"
    echo "========================================="

    # Update array size
    sed -i.bak "s/static WORD aximm_storage\[[0-9]*\];/static WORD aximm_storage[$size];/" libinterp/comp-arm64.c

    # Clean build
    cd emu/MacOSX && rm -f *.o o.emu && cd ../..
    ./build-macos-headless.sh > /tmp/build_$size.log 2>&1

    if [ ! -f emu/MacOSX/o.emu ]; then
        echo "FAILED TO BUILD with size $size"
        continue
    fi

    # Test echo
    echo "Testing echo..."
    result=$(echo "test" | timeout 1 ./emu/MacOSX/o.emu -r. -c1 dis/echo.dis 2>&1)
    if echo "$result" | grep -q "^test$"; then
        echo "  echo: OUTPUT OK"
    else
        echo "  echo: FAILED"
        echo "$result" | grep -E "SEGV|panic|urk|error" | head -3
    fi

    # Test calc
    echo "Testing calc..."
    result=$(echo "2+2" | timeout 1 ./emu/MacOSX/o.emu -r. -c1 dis/calc.dis 2>&1)
    if echo "$result" | grep -q "^4$"; then
        echo "  calc: OUTPUT OK"
    elif echo "$result" | grep -q "^2+2$"; then
        echo "  calc: Input echoed, no output"
        echo "$result" | grep -E "SEGV|panic|urk|error" | head -3
    else
        echo "  calc: FAILED"
        echo "$result" | grep -E "SEGV|panic|urk|error" | head -3
    fi

    echo ""
done

# Restore original
mv libinterp/comp-arm64.c.bak libinterp/comp-arm64.c 2>/dev/null || true
