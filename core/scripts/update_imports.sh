#!/bin/bash

# Script to update import paths from old structure to new core/ structure

echo "Updating import paths to new core/ structure..."

# Find all Go files in core directory
find core/ -name "*.go" -type f | while read file; do
    echo "Processing: $file"
    
    # Update import paths
    sed -i '' \
        -e 's|github.com/handcraftdev/blackhole/cmd/|github.com/handcraftdev/blackhole/core/cmd/|g' \
        -e 's|github.com/handcraftdev/blackhole/internal/|github.com/handcraftdev/blackhole/core/src/|g' \
        -e 's|github.com/handcraftdev/blackhole/pkg/|github.com/handcraftdev/blackhole/core/pkg/|g' \
        "$file"
done

echo "Import path updates completed!"