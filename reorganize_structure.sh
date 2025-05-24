#!/bin/bash

# Script to reorganize the Blackhole project directory structure
# This script will move directories to their new locations and clean up empty directories

set -e  # Exit on error

echo "Starting directory reorganization..."

# Function to safely move a directory
move_dir() {
    local src="$1"
    local dest="$2"
    
    if [ -d "$src" ]; then
        # Create parent directory if needed
        mkdir -p "$(dirname "$dest")"
        
        echo "Moving $src to $dest"
        mv "$src" "$dest"
    else
        echo "Warning: Source directory $src does not exist, skipping..."
    fi
}

# Function to remove directory if empty
remove_if_empty() {
    local dir="$1"
    
    if [ -d "$dir" ]; then
        if [ -z "$(ls -A "$dir")" ]; then
            echo "Removing empty directory: $dir"
            rmdir "$dir"
        else
            echo "Warning: Directory $dir is not empty, not removing"
        fi
    fi
}

# 1. Move platform/docs to ecosystem/docs
move_dir "core/src/framework/platform/docs" "ecosystem/docs"

# 2. Move platform subdirectories to ecosystem/
move_dir "core/src/framework/platform/sdk" "ecosystem/sdk"
move_dir "core/src/framework/platform/tools" "ecosystem/tools"
move_dir "core/src/framework/platform/templates" "ecosystem/templates"
move_dir "core/src/framework/platform/marketplace" "ecosystem/marketplace"

# 3. Remove empty platform directory
remove_if_empty "core/src/framework/platform"

# 4. Move foundation directories to ecosystem/
move_dir "foundation/governance" "ecosystem/governance"
move_dir "foundation/community" "ecosystem/community"
move_dir "foundation/events" "ecosystem/events"
move_dir "foundation/certification" "ecosystem/certification"

# 5. Remove empty foundation directory
remove_if_empty "foundation"

# 6. Remove products directory entirely
if [ -d "products" ]; then
    echo "Removing products directory..."
    rm -rf "products"
fi

# 7. Move core/web/dashboard to core/internal/runtime/dashboard
if [ -d "core/web/dashboard" ]; then
    # Create the internal/runtime directory if it doesn't exist
    mkdir -p "core/src/runtime"
    move_dir "core/web/dashboard" "core/src/runtime/dashboard"
    
    # Remove empty web directory if it exists
    remove_if_empty "core/web"
fi

# Also check for the web/dashboard at root level (based on directory structure)
if [ -d "web/dashboard" ]; then
    mkdir -p "core/src/runtime"
    move_dir "web/dashboard" "core/src/runtime/dashboard"
    remove_if_empty "web"
fi

echo "Directory reorganization complete!"
echo ""
echo "Summary of changes:"
echo "- Platform docs, sdk, tools, templates, and marketplace moved to ecosystem/"
echo "- Foundation directories moved to ecosystem/"
echo "- Products directory removed"
echo "- Dashboard moved to core/src/runtime/dashboard"
echo ""
echo "Please verify the new structure and update any references in your code and documentation."