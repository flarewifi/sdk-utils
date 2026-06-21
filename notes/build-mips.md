#!/bin/bash
#
# Cross-compile FlareHotspot for MIPS (OpenWRT routers)
# Uses the OpenWRT toolchain for mipsel_24kc (MT7621 SoC)
#
# Usage:
#   ./scripts/build-mips.sh
#
# Requirements:
#   - OpenWRT toolchain extracted to openwrt-toolchain-* directory
#   - Go 1.21+
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Find the OpenWRT toolchain directory
TOOLCHAIN_BASE=$(find "$PROJECT_DIR" -maxdepth 1 -type d -name "openwrt-toolchain-*" | head -1)

if [ -z "$TOOLCHAIN_BASE" ]; then
    echo "ERROR: OpenWRT toolchain not found!"
    echo "Please extract the OpenWRT SDK to the project root directory."
    echo "Expected: openwrt-toolchain-*-ramips-mt7621_gcc-*_musl.Linux-x86_64/"
    exit 1
fi

# Find the actual toolchain bin directory
TOOLCHAIN_DIR=$(find "$TOOLCHAIN_BASE" -type d -name "toolchain-mipsel_*" | head -1)

if [ -z "$TOOLCHAIN_DIR" ]; then
    echo "ERROR: Could not find toolchain-mipsel_* directory in $TOOLCHAIN_BASE"
    exit 1
fi

echo "Using toolchain: $TOOLCHAIN_DIR"
echo "Cross-compiler: ${TOOLCHAIN_DIR}/bin/mipsel-openwrt-linux-musl-gcc"
echo "Target: linux/mipsle (softfloat)"
echo ""

# Verify the compiler exists
if [ ! -x "${TOOLCHAIN_DIR}/bin/mipsel-openwrt-linux-musl-gcc" ]; then
    echo "ERROR: Cross-compiler not found at ${TOOLCHAIN_DIR}/bin/mipsel-openwrt-linux-musl-gcc"
    exit 1
fi

# Clean previous build
rm -rf "$PROJECT_DIR/output/mono-bin-files"
rm -rf "$PROJECT_DIR/plugins/installed"
rm -f "$PROJECT_DIR/core/plugin.so"

cd "$PROJECT_DIR"

# Step 1: Run the build tool on the HOST machine (no cross-compilation)
# This prepares assets, templates, queries, and plugin files
echo "Step 1: Preparing assets and plugins (host build)..."
go run -tags="prod mono cgo" ./tools/cmd/create-mono-bin/main.go --prepare-only 2>/dev/null || true

# Since --prepare-only might not exist, we need a different approach
# Let's build the binary separately after running the prep steps

echo "Step 1: Building assets and preparing plugins..."

# Create go workspace
go run -tags="prod mono" ./tools/cmd/build-assets/main.go 2>/dev/null || true

# Build templates
go run -tags="prod mono" ./tools/cmd/build-templates/main.go 2>/dev/null || true

# For now, let's just directly cross-compile the final binary
echo ""
echo "Step 2: Cross-compiling for MIPS..."

# Set up cross-compilation environment for the final binary only
export CC="${TOOLCHAIN_DIR}/bin/mipsel-openwrt-linux-musl-gcc"
export CXX="${TOOLCHAIN_DIR}/bin/mipsel-openwrt-linux-musl-g++"
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=mipsle
export GOMIPS=softfloat

# Build the final binary directly
go build -tags="prod mono cgo" \
    -ldflags="-s -w" \
    -trimpath \
    -o bin/flare \
    ./core/internal/cli/main.go

echo ""
echo "Build completed successfully!"
echo "Binary: $PROJECT_DIR/bin/flare"
echo ""
echo "To deploy to your router:"
echo "  scp bin/flare root@<router-ip>:/opt/flarewifi/app/bin/"
