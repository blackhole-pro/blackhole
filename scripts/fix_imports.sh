#!/bin/bash

# Fix import paths in Blackhole project
# This script updates all Go imports from core/src/ to core/internal/

set -e

echo "=== Blackhole Import Path Fix Script ==="
echo "This script will update all import paths from 'core/src/' to 'core/internal/'"
echo ""

# Get the root directory of the project
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "Project root: $PROJECT_ROOT"

# Counter for changed files
CHANGED_FILES=0

# Function to fix imports in a single file
fix_imports_in_file() {
    local file="$1"
    
    # Check if file contains old import paths
    if grep -q "github.com/handcraftdev/blackhole/core/src/" "$file"; then
        echo "Fixing imports in: $file"
        
        # Create a backup just in case
        cp "$file" "${file}.import_backup"
        
        # Replace all occurrences of core/src/ with core/internal/
        # Use perl for better cross-platform compatibility
        perl -pi -e 's|github\.com/handcraftdev/blackhole/core/src/|github.com/handcraftdev/blackhole/core/internal/|g' "$file"
        
        # Remove the backup if the sed command succeeded
        rm "${file}.import_backup"
        
        ((CHANGED_FILES++))
    fi
}

# Find all Go files in the project
echo ""
echo "Searching for Go files..."
GO_FILES=$(find "$PROJECT_ROOT" -name "*.go" -type f | grep -v vendor | grep -v .git)
TOTAL_FILES=$(echo "$GO_FILES" | wc -l | tr -d ' ')
echo "Found $TOTAL_FILES Go files"

echo ""
echo "Fixing import paths..."

# Fix imports in each Go file
for file in $GO_FILES; do
    fix_imports_in_file "$file"
done

echo ""
echo "=== Summary ==="
echo "Total files scanned: $TOTAL_FILES"
echo "Files modified: $CHANGED_FILES"

# Check if go.mod needs updates
echo ""
echo "Checking go.mod files..."
if find "$PROJECT_ROOT" -name "go.mod" -type f -exec grep -l "core/src" {} \; | grep -q .; then
    echo "Found go.mod files with old paths, updating..."
    find "$PROJECT_ROOT" -name "go.mod" -type f -exec perl -pi -e 's|core/src/|core/internal/|g' {} \;
fi

# Check if go.work needs updates
if [ -f "$PROJECT_ROOT/go.work" ]; then
    echo "Checking go.work..."
    if grep -q "core/src" "$PROJECT_ROOT/go.work"; then
        echo "Updating go.work..."
        perl -pi -e 's|core/src/|core/internal/|g' "$PROJECT_ROOT/go.work"
    fi
fi

echo ""
echo "=== Running go mod tidy ==="
cd "$PROJECT_ROOT"
go mod tidy || echo "Note: go mod tidy failed - you may need to run it manually"

echo ""
echo "=== Import path fix complete! ==="
echo ""
echo "Next steps:"
echo "1. Run 'make test' to verify all tests pass"
echo "2. Run 'make build' to ensure everything compiles"
echo "3. Commit the changes"