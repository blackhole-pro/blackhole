#!/bin/bash

# Script to fix remaining import path issues

echo "Fixing remaining import path issues..."

# Find all Go files and fix specific problematic imports
find core/ -name "*.go" -type f | while read file; do
    echo "Processing: $file"
    
    # Fix import paths that are still pointing to old core/ structure
    sed -i '' \
        -e 's|github.com/handcraftdev/blackhole/core/src/core/config|github.com/handcraftdev/blackhole/core/src/runtime/config|g' \
        -e 's|github.com/handcraftdev/blackhole/core/src/core/process|github.com/handcraftdev/blackhole/core/src/runtime/orchestrator|g' \
        -e 's|github.com/handcraftdev/blackhole/core/src/core/mesh|github.com/handcraftdev/blackhole/core/src/framework/mesh|g' \
        "$file"
done

echo "Import path fixes completed!"