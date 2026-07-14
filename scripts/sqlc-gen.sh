#!/bin/sh
set -eu

# sqlc-gen.sh — generate sqlc query code for core or a single plugin.
#
# Usage: ./scripts/sqlc-gen.sh <source_dir>
#
#   <source_dir>   ./core  or  ./data/plugins/local/<pkg>
#
# sqlc runs in a throwaway temp dir assembled from:
#   - core/sqlc.yml            the sqlc config (engine: sqlite)
#   - core migrations          always included, so plugin queries can JOIN core tables
#   - <source_dir> migrations  the source's own schema (nothing extra when it IS core)
#   - <source_dir> queries     the queries to generate Go for
# The generated Go is copied back to <source_dir>/db/queries.
#
# SQLite is the only engine (postgres was removed), so there is no driver argument.
# Any extra arguments are ignored for backward compatibility with older callers.

SRC_DIR="${1:-}"

if [ -z "$SRC_DIR" ]; then
    echo "Usage: $0 <source_dir>" >&2
    exit 1
fi
if [ ! -d "$SRC_DIR" ]; then
    echo "Error: '$SRC_DIR' is not a directory." >&2
    exit 1
fi

SRC_DIR="$(cd "$SRC_DIR" && pwd)"

# Check for queries before requiring core/sqlc.yml: a devkit build never ships
# core/sqlc.yml (core is closed-source there — see build-devkit.go), so a plugin
# with no queries must be able to skip out cleanly instead of erroring on a file
# it doesn't actually need.
if [ ! -d "$SRC_DIR/resources/queries" ]; then
    echo "No queries in $SRC_DIR/resources/queries — skipping sqlc generation."
    exit 0
fi

CORE_DIR="$(cd core && pwd)"
CORE_SQLC="$CORE_DIR/sqlc.yml"

if [ ! -f "$CORE_SQLC" ]; then
    echo "Error: core sqlc config not found: $CORE_SQLC" >&2
    exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
echo "Working in temp dir: $TMP_DIR"

# 1. sqlc config.
cp "$CORE_SQLC" "$TMP_DIR/sqlc.yml"

# 2. Schema: core migrations first, then the source's own (unioned in one dir).
mkdir -p "$TMP_DIR/resources/migrations"
cp -R "$CORE_DIR/resources/migrations/." "$TMP_DIR/resources/migrations/"
if [ "$SRC_DIR" != "$CORE_DIR" ] && [ -d "$SRC_DIR/resources/migrations" ]; then
    cp -R "$SRC_DIR/resources/migrations/." "$TMP_DIR/resources/migrations/"
fi

# 3. Queries: only the source's own.
cp -R "$SRC_DIR/resources/queries" "$TMP_DIR/resources/queries"

# 4. Generate, then copy the output back into the source tree.
echo "Running sqlc generate ..."
( cd "$TMP_DIR" && sqlc generate )

rm -rf "$SRC_DIR/db/queries"
mkdir -p "$SRC_DIR/db/queries"
cp -R "$TMP_DIR/db/queries/." "$SRC_DIR/db/queries/"
echo "Generated queries → $SRC_DIR/db/queries"
