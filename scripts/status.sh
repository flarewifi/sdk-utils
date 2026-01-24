#!/bin/bash

# Show git status for core and all plugins
# Displays counts of modified, added, deleted, and untracked files

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGINS_DIR="$PROJECT_ROOT/data/plugins/local"

# Color codes
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BRIGHT_GREEN='\033[1;32m'
BRIGHT_YELLOW='\033[1;33m'
BRIGHT_RED='\033[1;31m'
BRIGHT_CYAN='\033[1;36m'
NC='\033[0m' # No Color

# Function to get git status counts for a repository
get_status_counts() {
    local repo_path=$1
    
    # Get status in short format
    local status=$(cd "$repo_path" && git status --short 2>/dev/null)
    
    local modified=0
    local added=0
    local deleted=0
    local untracked=0
    
    while IFS= read -r line; do
        if [ -z "$line" ]; then
            continue
        fi
        
        # Get the status code (first 2 characters)
        local code="${line:0:2}"
        
        case "$code" in
            " M")  ((modified++)) ;;     # Modified in working tree
            "M ")  ((modified++)) ;;     # Modified in index
            "MM")  ((modified++)) ;;     # Modified in both
            "A ")  ((added++)) ;;        # Added to index
            " D")  ((deleted++)) ;;      # Deleted from working tree
            "D ")  ((deleted++)) ;;      # Deleted from index
            "DD")  ((deleted++)) ;;      # Deleted in both
            "??")  ((untracked++)) ;;    # Untracked
        esac
    done <<< "$status"
    
    echo "$modified:$added:$deleted:$untracked"
}

# Function to format stat with color based on value
format_stat() {
    local label=$1
    local value=$2
    local color=$3
    
    printf "  %-12s ${color}%d${NC}\n" "$label" "$value"
}

# Function to display status for a repository
display_status() {
    local name=$1
    local repo_path=$2
    
    local counts=$(get_status_counts "$repo_path")
    IFS=':' read -r modified added deleted untracked <<< "$counts"
    
    printf "${CYAN}%s${NC}\n" "$name"
    
    # Color stats based on their values
    if [ "$modified" -gt 0 ]; then
        format_stat "Modified:" "$modified" "$BRIGHT_YELLOW"
    else
        format_stat "Modified:" "$modified" "$NC"
    fi
    
    if [ "$added" -gt 0 ]; then
        format_stat "Added:" "$added" "$BRIGHT_YELLOW"
    else
        format_stat "Added:" "$added" "$NC"
    fi
    
    if [ "$deleted" -gt 0 ]; then
        format_stat "Deleted:" "$deleted" "$BRIGHT_RED"
    else
        format_stat "Deleted:" "$deleted" "$NC"
    fi
    
    if [ "$untracked" -gt 0 ]; then
        format_stat "Untracked:" "$untracked" "$BRIGHT_RED"
    else
        format_stat "Untracked:" "$untracked" "$NC"
    fi
    
    echo ""
}

# Main script
echo ""
display_status "Core Repository:" "$PROJECT_ROOT"

# Check if plugins directory exists
if [ -d "$PLUGINS_DIR" ]; then
    # Iterate through each plugin directory in sorted order
    for plugin_path in $(find "$PLUGINS_DIR" -maxdepth 1 -type d -not -name "local" | sort); do
        if [ -d "$plugin_path" ]; then
            plugin_name=$(basename "$plugin_path")
            
            # Only process if it's a git repository
            if [ -d "$plugin_path/.git" ]; then
                display_status "Plugin: $plugin_name" "$plugin_path"
            fi
        fi
    done
fi
