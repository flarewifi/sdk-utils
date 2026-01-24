#!/bin/bash
# ⚠️  DEPRECATED: Please use update-translations.py instead
#
# This shell script is deprecated. Use the Python script for better features:
# - UTF-8 validation
# - Backup creation
# - Dry-run mode
# - Better error handling
#
# Migration:
#   ./scripts/update-translations.py <type> <key> <translations_json>
#
# Original usage (still works, but deprecated):
# ./scripts/update-translations.sh <type> <key> <translations_json>

echo "⚠️  WARNING: This script is deprecated. Please use update-translations.py instead."
echo "   Example: ./scripts/update-translations.py $@"
echo ""
echo "Continuing with deprecated script in 3 seconds..."
sleep 3

set -e

TYPE="$1"
KEY="$2"
TRANSLATIONS_JSON="$3"

if [ -z "$TYPE" ] || [ -z "$KEY" ] || [ -z "$TRANSLATIONS_JSON" ]; then
    echo "Usage: $0 <type> <key> <translations_json>"
    echo "Example: $0 warning \"The purchase has been cancelled\" '{\"en\":\"text\",\"es\":\"texto\"}'"
    exit 1
fi

# Base directory for translations
BASE_DIR="core/resources/translations"

# Parse JSON and create files
echo "$TRANSLATIONS_JSON" | jq -r 'to_entries | .[] | "\(.key)=\(.value)"' | while IFS='=' read -r lang translation; do
    # Create directory if it doesn't exist
    dir="${BASE_DIR}/${lang}/${TYPE}"
    mkdir -p "$dir"

    # File path
    file="${dir}/${KEY}.txt"

    # Write the translation
    echo "$translation" > "$file"
    echo "Created/Updated: $file"
done

echo "Translation files updated successfully!"
