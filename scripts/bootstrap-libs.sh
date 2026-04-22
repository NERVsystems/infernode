#!/bin/sh
#
# Shared macOS ARM64 library bootstrap for host-side build scripts.
# Mirrors the CI build order and intentionally skips freetype, which the
# macOS emulator config does not link.
#

bootstrap_build_required_lib() {
    bootstrap_dir=$1

    echo "  Building $bootstrap_dir..."
    (cd "$ROOT/$bootstrap_dir" && "$BOOTSTRAP_MK" install) || {
        echo "ERROR: $bootstrap_dir build failed" >&2
        return 1
    }
}

bootstrap_build_optional_lib() {
    bootstrap_dir=$1

    echo "  Building $bootstrap_dir..."
    if ! (cd "$ROOT/$bootstrap_dir" && "$BOOTSTRAP_MK" install); then
        echo "WARNING: $bootstrap_dir build failed (non-fatal bootstrap step)" >&2
    fi
}

bootstrap_macos_arm64_libs() {
    if [ -z "$ROOT" ]; then
        echo "ERROR: ROOT must be set before bootstrapping libraries" >&2
        return 1
    fi

    BOOTSTRAP_MK="$ROOT/MacOSX/arm64/bin/mk"
    BOOTSTRAP_LIMBO="$ROOT/MacOSX/arm64/bin/limbo"

    mkdir -p "$ROOT/MacOSX/arm64/lib"

    if [ ! -x "$BOOTSTRAP_MK" ]; then
        echo "ERROR: mk not found at $BOOTSTRAP_MK" >&2
        return 1
    fi

    if [ ! -x "$BOOTSTRAP_LIMBO" ]; then
        echo "ERROR: native limbo compiler not found at $BOOTSTRAP_LIMBO" >&2
        return 1
    fi

    echo "=== Building Libraries ==="
    for bootstrap_dir in lib9 libbio libmp libsec libmath; do
        bootstrap_build_required_lib "$bootstrap_dir" || return 1
    done

    bootstrap_build_required_lib libinterp || return 1

    for bootstrap_dir in libkeyring libdraw libmemdraw libmemlayer; do
        bootstrap_build_optional_lib "$bootstrap_dir"
    done

    echo ""
}
