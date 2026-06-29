#!/bin/sh

# Resolve the prebuilt binaries for THIS container's CPU architecture.
#
# The devkit ships native binaries for both linux/amd64 and linux/arm64: a Go
# -buildmode=plugin + CGO core .so can't be cross-compiled, so each arch was built
# natively (bin/<arch>/{flare,livereload}, core/plugin.<arch>.so). Docker runs the
# container at the host's arch on Apple Silicon, Windows (Docker + WSL2) and Linux
# alike, so `dpkg --print-architecture` tells us which set to activate:
#   Apple Silicon / Windows-on-ARM (WSL2) / Linux ARM -> arm64
#   Windows x86 (WSL2) / Linux x86                     -> amd64
#
# Copy (not symlink) into the canonical paths: symlinks are unreliable on
# bind-mounted Windows/NTFS paths, while `cp` works on ext4 (WSL2), NTFS and APFS.
# Idempotent — safe to re-run from both docker-cmd.sh and start-dev.sh.

set -e

ARCH=$(dpkg --print-architecture) # amd64 | arm64 (matches Go GOARCH)

if [ ! -f "bin/$ARCH/flare" ]; then
	echo "Unsupported devkit architecture: '$ARCH' (expected amd64 or arm64)" >&2
	echo "This devkit ships binaries for: $(ls bin 2>/dev/null | tr '\n' ' ')" >&2
	exit 1
fi

cp -f "bin/$ARCH/flare" bin/flare
cp -f "bin/$ARCH/livereload" bin/livereload
cp -f "core/plugin.$ARCH.so" core/plugin.so
chmod +x bin/flare bin/livereload
