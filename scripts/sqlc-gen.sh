#!/bin/sh
set -eu

# Usage: ./generate-sqlc.sh <plugin_directory> [driver]
# Example: ./generate-sqlc.sh /home/user/myplugin postgres

PLUGIN_DIR="${1:-}"
DRIVER="${2:-}"

if [ -z "$PLUGIN_DIR" ]; then
    echo "Usage: $0 <plugin_directory> [driver]"
    exit 1
fi

if [ ! -d "$PLUGIN_DIR" ]; then
    echo "Error: '$PLUGIN_DIR' is not a valid directory."
    exit 1
fi

# Absolute paths
PLUGIN_DIR="$(cd "$PLUGIN_DIR" && pwd)"
CORE_DIR="$(cd core && pwd)"
TMP_DIR="$(mktemp -d)"
echo "Temporary working directory created at: $TMP_DIR"

if [ ! -d "$PLUGIN_DIR/resources/queries" ]; then
  echo "queries directory not found in plugin: $PLUGIN_DIR/resources/queries"
  echo "skipping sqlc generation."
  exit 0
fi

# Copy core migrations and queries files to temp directory
(
  if [ $CORE_DIR = $PLUGIN_DIR ]; then
      echo "Core and plugin directories are the same. Skipping copy of core files."
  else
    ./scripts/copy-sql.sh "core" "$TMP_DIR" "$DRIVER"
    # Remove core queries
    rm -rf "$TMP_DIR/resources/queries"
  fi

  ./scripts/copy-sql.sh "$PLUGIN_DIR" "$TMP_DIR" "$DRIVER"
)

# Run sqlc generate
echo "Running sqlc generate in $TMP_DIR ..."
(
    cd "$TMP_DIR"
    sqlc generate
)
echo "sqlc generate completed successfully."

# Copy generated output back
if [ -d "$TMP_DIR/db/queries" ]; then
    mkdir -p "$PLUGIN_DIR/db/queries"
    cp -r "$TMP_DIR/db/queries/." "$PLUGIN_DIR/db/queries/"
    echo "Copied generated queries to $PLUGIN_DIR/db/queries/"
else
    echo "Warning: No generated output found in $TMP_DIR/db/queries"
fi

# Cleanup
rm -rf "$TMP_DIR"
echo "Temporary directory deleted: $TMP_DIR"

echo "Done."
