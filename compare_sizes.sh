#!/bin/bash
for size in 32 33; do
  echo "=== Building size $size ==="
  sed -i '' "s/static WORD aximm_storage\[[0-9]*\]/static WORD aximm_storage[$size]/" libinterp/comp-arm64.c
  rm -f emu/MacOSX/comp-arm64.o emu/MacOSX/o.emu
  ./build-macos-headless.sh > /tmp/build_$size.log 2>&1
  
  echo "test" | timeout 2 ./emu/MacOSX/o.emu -r. -c4 dis/echo.dis 2>&1 | head -50 > /tmp/output_$size.txt
  echo "Saved to /tmp/output_$size.txt"
done
echo ""
echo "=== Comparing preambles ==="
diff -u <(grep "Preamble\|\[ [0-9]\]" /tmp/output_32.txt) <(grep "Preamble\|\[ [0-9]\]" /tmp/output_33.txt) || true
