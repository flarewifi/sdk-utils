#!/bin/sh
set -eu

# Usage: ./copy-sql.sh <plugin_directory> <tmp_dir>
# Example: ./copy-sql.sh /home/user/myplugin /tmp/sqlc-build

PLUGIN_DIR="${1:-}"
TMP_DIR="${2:-}"

if [ ! -d "$PLUGIN_DIR" ]; then
    echo "Usage: $0 <plugin_directory> <temporary_directory>"
    exit 1
fi

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

# Use sqlc.yml (SQLite only)
CORE_SQLC_FILE="$CORE_DIR/sqlc.yml"
PLUGIN_SQLC_FILE="$PLUGIN_DIR/sqlc.yml"

# Check if core sqlc file exists
if [ ! -f "$CORE_SQLC_FILE" ]; then
    echo "Error: $CORE_SQLC_FILE not found"
    rm -rf "$TMP_DIR"
    exit 1
fi

# Copy core sqlc file
cp "$CORE_SQLC_FILE" "$TMP_DIR/sqlc.yml"
echo "Copied $CORE_SQLC_FILE to $TMP_DIR/sqlc.yml"

# Merge plugin sqlc overrides if plugin has custom sqlc config
# Skip merge if plugin sqlc file is the same as core sqlc file (i.e., plugin is core)
if [ -f "$PLUGIN_SQLC_FILE" ]; then
    # Check if plugin sqlc file is the same as core sqlc file
    if [ "$CORE_SQLC_FILE" = "$PLUGIN_SQLC_FILE" ]; then
        echo "Plugin and core sqlc files are the same. Skipping merge."
    else
        echo "Found plugin sqlc config: $PLUGIN_SQLC_FILE"
        echo "Merging plugin overrides with core overrides..."
        
        # Extract overrides from plugin sqlc file (lines after "overrides:" and properly indented)
        PLUGIN_OVERRIDES=$(awk '
            /^[[:space:]]*overrides:/ { in_overrides=1; next }
            in_overrides && /^[[:space:]]*- / { print; next }
            in_overrides && /^[[:space:]]*[a-z_]+:/ && !/^[[:space:]]{12,}/ { in_overrides=0 }
            in_overrides && /^[[:space:]]{12,}/ { print }
        ' "$PLUGIN_SQLC_FILE")
        
        if [ -n "$PLUGIN_OVERRIDES" ]; then
            # Insert plugin overrides before the last line of the core config
            # This appends them to the existing overrides array
            awk -v overrides="$PLUGIN_OVERRIDES" '
                # Track if we are in the overrides section
                /^[[:space:]]*overrides:/ { in_overrides=1; print; next }
                
                # If in overrides and we hit a non-override key at the same or lower indentation, insert plugin overrides
                in_overrides && /^[[:space:]]*[a-z_]+:/ && !/^[[:space:]]{12,}/ {
                    if (overrides) {
                        print overrides
                        overrides=""
                    }
                    in_overrides=0
                }
                
                # Print all lines
                { print }
                
                # If we reach EOF and still have overrides, append them
                END {
                    if (overrides) {
                        print overrides
                    }
                }
            ' "$TMP_DIR/sqlc.yml" > "$TMP_DIR/sqlc.yml.tmp"
            
            mv "$TMP_DIR/sqlc.yml.tmp" "$TMP_DIR/sqlc.yml"
            echo "Plugin overrides merged successfully"
        else
            echo "No overrides found in plugin sqlc config"
        fi
    fi
else
    echo "No plugin sqlc config found at: $PLUGIN_SQLC_FILE"
    echo "Using core sqlc config only"
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

# No driver-specific directories needed (SQLite only)

echo "All relevant SQL resources have been copied to: $TMP_DIR"
