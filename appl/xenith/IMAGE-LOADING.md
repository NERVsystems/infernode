# Xenith Image Loading Implementation

## Status: Work In Progress

**Date:** 2025-01-17
**Branch:** feature/sdl3-gui

## What Was Implemented

### Core Functionality
- **Image display in Xenith windows** via `echo 'image /path' > /mnt/xenith/<id>/ctl`
- **PPM format support** (P3 ASCII, P6 binary) with subsampling for large images
- **PNG format support** including:
  - Standard PNG via Inferno's `readpng` module (for small images)
  - Custom streaming decoder with subsampling (for large images)
  - Adam7 interlaced PNG support

### Memory Management
- Automatic subsampling for images exceeding 16 megapixels
- Stricter 8 megapixel limit for interlaced PNGs (require full image buffer)
- Streaming row-by-row processing to minimize memory footprint

### Files Modified
- `appl/xenith/imgload.b` - Core image loading module
- `appl/xenith/imgload.m` - Module interface
- `appl/xenith/wind.b` - Image display and scaling
- `appl/xenith/xfid.b` - 9P file system integration
- `appl/xenith/dat.m` - Data structures
- `appl/xenith/fsys.b` - File system setup

## Current Limitations

### 1. Performance (Critical)
**Small interlaced PNG (200x200) takes many minutes to load.**

Root cause identified: Both `inflate.b` (zlib decompression) and PNG filter
application are implemented in **interpreted Limbo/Dis bytecode**, not native C.

For comparison:
- macOS native libpng: Opens 534MP image "almost instantly" (SIMD, native code)
- Xenith/Limbo: Even small images take minutes (interpreted bytecode)

The bottlenecks are:
1. `appl/lib/inflate.b` - 820 lines of Limbo implementing zlib decompression
2. PNG filter loops in `imgload.b` - Process every byte of every row

### 2. UI Blocking
Image loading blocks the entire Xenith UI. Users cannot multitask while
an image loads. This compounds the performance issue - a slow load that
allowed other work would be tolerable; a slow load that freezes the UI is not.

### 3. Large Image Handling
For the test image (20800x25675 interlaced PNG, 534 megapixels):
- Subsampled to ~2311x2852 (factor 9) to fit memory
- Still requires decompressing ALL 534M pixels through interpreted code
- Estimated processing: billions of bytecode operations

## Optimizations Made

1. **Output loop optimization** - Iterate destination pixels (~2300) instead of
   source pixels (~20800) for interlaced images. Reduces loop iterations ~9x.

2. **Early dimension check** - Read PNG header before attempting full decode
   to fail fast on oversized images.

3. **Streaming decoder** - Process rows as they decompress rather than
   loading entire image into memory.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Xenith                                   │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │   xfid.b    │───▶│  imgload.b  │───▶│   wind.b    │         │
│  │ (9P ctl)    │    │ (decode)    │    │ (display)   │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│                            │                   │                │
│                            ▼                   ▼                │
│                     ┌─────────────┐    ┌─────────────┐         │
│                     │ inflate.b   │    │  Draw->     │         │
│                     │ (SLOW -     │    │  Image      │         │
│                     │  Limbo!)    │    │ (native)    │         │
│                     └─────────────┘    └─────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   Dis VM        │
                    │ (interpreted)   │
                    └─────────────────┘
```

The Draw system is native C. The bottleneck is everything before it.

## Next Steps

### Priority 1: Native zlib Implementation
Replace `appl/lib/inflate.b` with native C code in the emu. This would:
- Dramatically speed up PNG (and all compression operations)
- Benefit the entire system, not just image loading
- Require changes to `emu/port/` or `libinterp/`

### Priority 2: Async/Background Loading
Make image loading non-blocking so users can continue working while
images load. Options:
- Spawn image loading in a separate Limbo thread
- Show progress indicator
- Allow cancellation

### Priority 3: Native PNG Decoder (Alternative)
If native zlib is too complex, add a dedicated native PNG module:
- C implementation using libpng or custom code
- Takes path, returns Draw->Image
- Bypasses Limbo entirely for image decode

### Priority 4: Format Optimization
- Consider adding JPEG support (more common, often smaller)
- Native Inferno image format is already fast (no compression)
- Pre-convert large PNGs to uncompressed format for faster loading

## Testing

### Verified Working
- Small non-interlaced PNG: ✓
- Small interlaced PNG (200x200 RGBA): ✓ (but slow)
- PPM format: ✓

### Known Issues
- Large interlaced PNG: Functional but impractically slow
- UI blocks during any image load

### Test Commands
```sh
# Start Xenith
cd /Users/pdfinn/github.com/NERVsystems/infernode/emu/MacOSX
./o.emu -r../.. sh -l -c 'xenith -t dark'

# Load test image (in Xenith)
echo 'image /n/local/tmp/test-rgba-interlaced.png' > /mnt/xenith/1/ctl

# Clear image
echo 'clearimage' > /mnt/xenith/1/ctl
```

## Commits

1. `c85c03a` - feat(xenith): Add portable streaming PNG/PPM image loading with subsampling
2. `b4ae02a` - feat(xenith): Add Adam7 interlaced PNG support for large images
3. `8136dd4` - fix(xenith): Increase subsample factor for interlaced PNGs to fit heap
4. `11e117d` - perf(xenith): Optimize interlaced PNG output loop

## References

- PNG Specification: http://www.libpng.org/pub/png/spec/
- Adam7 Interlacing: https://en.wikipedia.org/wiki/Adam7_algorithm
- Inferno Filter module: `/module/filter.m`
- Inferno Draw module: `/module/draw.m`
