# Verification of inferno64 ARM64 Status

## Their README Says:
"The JIT compiler for amd64 works, and JIT for arm64 is in development."

## Their comp-arm64.c File Contains:
- Register definitions: R9, R8, R5 (32-bit ARM registers)
- NOT X9, X10, X11 (64-bit ARM64 registers)
- ARM32 instruction encoding

## Conclusion:
comp-arm64.c appears to be ARM32 code, not ARM64.
"JIT for arm64 is in development" = NOT FINISHED (same as us!)

## What "working emu on arm64" Likely Means:
- The emulator binary runs on ARM64 host platforms
- Uses interpreter mode (not JIT)
- Or has ARM32 JIT targeting 32-bit ARM guests
- NOT a working ARM64-native JIT
