#!/bin/sh
set -eu

# Usage: ./generate-sqlc.sh <plugin_directory> [driver]
# Example: ./generate-sqlc.sh /home/user/myplugin postgres

PLUGIN_DIR="${1:-}"
TMP_DIR="${2:-}"
DRIVER="${3:-}"

# Ensure TMP_DIR exists
if [ ! -d "$TMP_DIR" ]; then
    echo "Creating temporary directory at: $TMP_DIR"
    mkdir -p "$TMP_DIR"
fi

# Absolute paths
PLUGIN_DIR="$(cd "$PLUGIN_DIR" && pwd)"
CORE_DIR="$(cd "core" && pwd)"
TMP_DIR="$(cd "$TMP_DIR" && pwd)"

if [ ! -d "$PLUGIN_DIR" ]; then
    echo "Error: '$PLUGIN_DIR' is not a valid directory."
    exit 1
fi

# Copy sqlc.yml
if [ -f "$CORE_DIR/sqlc.yml" ]; then
    cp "$CORE_DIR/sqlc.yml" "$TMP_DIR/"
    echo "Copied $CORE_DIR/sqlc.yml"
else
    echo "Error: $CORE_DIR/sqlc.yml not"
    rm -rf "$TMP_DIR"
    exit 1
fi

# Copy base resources
if [ -e "$PLUGIN_DIR/resources/migrations" ]; then
    echo "Copying base migrations: $PLUGIN_DIR/resources/migrations ..."
    mkdir -p "$TMP_DIR/resources"
    cp -r "$PLUGIN_DIR/resources/migrations" "$TMP_DIR/resources/"
    echo "Copied base migrations/"
fi

if [ -d "$PLUGIN_DIR/resources/queries" ]; then
    echo "Copying base queries: $PLUGIN_DIR/resources/queries ..."
    mkdir -p "$TMP_DIR/resources"
    cp -r "$PLUGIN_DIR/resources/queries" "$TMP_DIR/resources/"
    echo "Copied base queries/"
fi

# Copy driver-specific directories if provided
if [ -n "$DRIVER" ]; then
    echo "Checking driver-specific directories for '$DRIVER'..."

    if [ -d "$TMP_DIR/resources/migrations/$DRIVER" ]; then
        rm -rf "$TMP_DIR/resources/migrations/$DRIVER"
    fi

    if [ -d "$PLUGIN_DIR/resources/migrations/$DRIVER" ]; then
        mkdir -p "$TMP_DIR/resources/migrations"
        cp -r "$PLUGIN_DIR/resources/migrations/$DRIVER/." "$TMP_DIR/resources/migrations/"
        echo "Copied migrations/$DRIVER/"
    fi

    if [ -d "$TMP_DIR/resources/queries/$DRIVER" ]; then
        rm -rf "$TMP_DIR/resources/queries/$DRIVER"
    fi

    if [ -d "$PLUGIN_DIR/resources/queries/$DRIVER" ]; then
        mkdir -p "$TMP_DIR/resources/queries"
        cp -r "$PLUGIN_DIR/resources/queries/$DRIVER/." "$TMP_DIR/resources/queries/"
        echo "Copied queries/$DRIVER/"
    fi
fi

echo "All relevant SQL resources have been copied to: $TMP_DIR"
