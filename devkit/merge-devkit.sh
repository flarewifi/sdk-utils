#!/bin/sh

# Merge the per-architecture devkit trees that `docker buildx --platform
# linux/amd64,linux/arm64` wrote under output/devkit-stage/ into ONE fat devkit
# zip. buildx names the per-platform local outputs linux_amd64/ and linux_arm64/,
# each holding an identical flarewifi-devkit-<version>/ tree EXCEPT for the
# arch-specific binaries (bin/<arch>/{flare,livereload}, core/plugin.<arch>.so) and
# node_modules' native esbuild package. A UNION (rsync without --delete) therefore
# yields a single tree carrying both architectures; select-arch.sh picks the right
# set at container boot.

set -e

STAGE="output/devkit-stage"
OUT="output/devkit"

AMD64_DIR="$STAGE/linux_amd64"
ARM64_DIR="$STAGE/linux_arm64"

for d in "$AMD64_DIR" "$ARM64_DIR"; do
	[ -d "$d" ] || {
		echo "Missing buildx platform output: $d" >&2
		echo "Run: docker buildx build --platform linux/amd64,linux/arm64 ... --output type=local,dest=$STAGE" >&2
		exit 1
	}
done

# Both platform trees use the same flarewifi-devkit-<version> directory name.
RELNAME=$(cd "$AMD64_DIR" && ls -d flarewifi-devkit-* 2>/dev/null | head -n1)
[ -n "$RELNAME" ] || {
	echo "No flarewifi-devkit-* tree found in $AMD64_DIR" >&2
	exit 1
}

rm -rf "$OUT"
mkdir -p "$OUT/$RELNAME"

# Union both arch trees (no --delete): neutral files coincide; the arch binaries
# and both esbuild platform packages coexist.
rsync -a "$AMD64_DIR/$RELNAME/" "$OUT/$RELNAME/"
rsync -a "$ARM64_DIR/$RELNAME/" "$OUT/$RELNAME/"

# Sanity-check: both arch binary sets must be present in the merged tree.
for f in bin/amd64/flare bin/arm64/flare core/plugin.amd64.so core/plugin.arm64.so; do
	[ -e "$OUT/$RELNAME/$f" ] || {
		echo "Merged tree is missing $f — was it built for both platforms?" >&2
		exit 1
	}
done

(cd "$OUT" && zip -qr "$RELNAME.zip" "$RELNAME")
rm -rf "$STAGE"

echo "Multi-arch devkit created: $OUT/$RELNAME.zip"
