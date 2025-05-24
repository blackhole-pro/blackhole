#!/bin/bash
# Plugin Compliance Checker
# Validates that plugins follow all development guidelines

set -e

PLUGIN_DIR="${1:-$(pwd)}"
ERRORS=0
WARNINGS=0

# Colors for output
RED='\033[0;31m'
YELLOW='\033[0;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Helper functions
error() {
    echo -e "${RED}[ERROR]${NC} $1"
    ((ERRORS++))
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
    ((WARNINGS++))
}

success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

# Check if directory exists
if [ ! -d "$PLUGIN_DIR" ]; then
    error "Plugin directory not found: $PLUGIN_DIR"
    exit 1
fi

cd "$PLUGIN_DIR"

echo "=== Plugin Compliance Check ==="
echo "Checking plugin at: $PLUGIN_DIR"
echo

# 1. Check for required files
echo "Checking required files..."
REQUIRED_FILES=(
    "go.mod"
    "plugin.yaml"
    "README.md"
    "Makefile"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        success "$file exists"
    else
        error "$file is missing"
    fi
done

# 2. Check for proper directory structure
echo -e "\nChecking directory structure..."
REQUIRED_DIRS=(
    "types"
    "proto/v1"
)

for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        success "$dir/ directory exists"
    else
        error "$dir/ directory is missing"
    fi
done

# 3. Check for typed errors
echo -e "\nChecking for typed errors..."
if [ -f "types/errors.go" ]; then
    success "types/errors.go exists"
    
    # Check if it contains custom error types
    if grep -q "type.*Error struct" "types/errors.go"; then
        success "Custom error types found"
    else
        warning "No custom error types found in types/errors.go"
    fi
else
    error "types/errors.go is missing"
fi

# 4. Check for gRPC proto definition
echo -e "\nChecking for gRPC service definition..."
if [ -f "proto/v1/"*.proto ]; then
    PROTO_FILE=$(ls proto/v1/*.proto | head -1)
    success "Proto file found: $PROTO_FILE"
    
    # Check for service definition
    if grep -q "service" "$PROTO_FILE"; then
        success "gRPC service definition found"
    else
        error "No gRPC service definition found in proto file"
    fi
else
    error "No proto files found in proto/v1/"
fi

# 5. Check for mesh compliance
echo -e "\nChecking for mesh compliance..."
MESH_REQUIRED=false

# Check for mesh client
if [ -f "mesh/client.go" ]; then
    success "mesh/client.go exists"
    MESH_REQUIRED=true
elif [ -f "main_mesh.go" ]; then
    success "main_mesh.go exists"
    MESH_REQUIRED=true
fi

# Check for grpc server implementation
if [ -f "grpc_server.go" ]; then
    success "grpc_server.go exists"
    
    # Check for event publishing
    if grep -q "PublishEvent\|publishEvent" "grpc_server.go"; then
        success "Event publishing found"
    else
        warning "No event publishing found in grpc_server.go"
    fi
fi

# 6. Check go.mod for independence
echo -e "\nChecking plugin independence..."
if [ -f "go.mod" ]; then
    # Check for direct plugin imports
    if grep -q "core/pkg/plugins/" "go.mod"; then
        error "Direct plugin dependencies found in go.mod"
    else
        success "No direct plugin dependencies"
    fi
    
    # Check module name
    MODULE_NAME=$(grep "^module" go.mod | awk '{print $2}')
    if [ "$MODULE_NAME" != "" ]; then
        success "Module name: $MODULE_NAME"
    else
        error "Invalid module name in go.mod"
    fi
fi

# 7. Check for structured logging
echo -e "\nChecking for structured logging..."
LOG_IMPORTS_FOUND=false

# Check all go files for log imports
for gofile in $(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*"); do
    if grep -q 'import.*"log"' "$gofile" || grep -q 'import.*log "log"' "$gofile"; then
        error "Standard log package imported in $gofile"
        LOG_IMPORTS_FOUND=true
    fi
    
    if grep -q 'zap\|logrus\|zerolog' "$gofile"; then
        success "Structured logging found in $gofile"
    fi
done

if [ "$LOG_IMPORTS_FOUND" = false ]; then
    success "No standard log imports found"
fi

# 8. Check plugin.yaml
echo -e "\nChecking plugin manifest..."
if [ -f "plugin.yaml" ]; then
    # Check for required fields
    REQUIRED_FIELDS=(
        "name:"
        "version:"
        "description:"
        "resources:"
        "capabilities:"
    )
    
    for field in "${REQUIRED_FIELDS[@]}"; do
        if grep -q "^$field" "plugin.yaml"; then
            success "plugin.yaml contains $field"
        else
            error "plugin.yaml missing $field"
        fi
    done
    
    # Check for mesh configuration
    if grep -q "mesh:" "plugin.yaml"; then
        success "Mesh configuration found in plugin.yaml"
    else
        warning "No mesh configuration in plugin.yaml"
    fi
fi

# 9. Check Makefile targets
echo -e "\nChecking Makefile targets..."
if [ -f "Makefile" ]; then
    REQUIRED_TARGETS=(
        "build"
        "test"
        "clean"
    )
    
    for target in "${REQUIRED_TARGETS[@]}"; do
        if grep -q "^$target:" "Makefile"; then
            success "Makefile has $target target"
        else
            warning "Makefile missing $target target"
        fi
    done
fi

# 10. Check for tests
echo -e "\nChecking for tests..."
TEST_COUNT=$(find . -name "*_test.go" -not -path "./vendor/*" | wc -l)
if [ "$TEST_COUNT" -gt 0 ]; then
    success "Found $TEST_COUNT test files"
else
    warning "No test files found"
fi

# 11. Check for health check implementation
echo -e "\nChecking for health check..."
if grep -r "HealthCheck\|healthCheck\|health_check" --include="*.go" . | grep -v "_test.go" | grep -q .; then
    success "Health check implementation found"
else
    warning "No health check implementation found"
fi

# 12. Check for resource limits in manifest
echo -e "\nChecking resource declarations..."
if [ -f "plugin.yaml" ]; then
    if grep -q "min_memory:\|max_memory:\|min_cpu:\|max_cpu:" "plugin.yaml"; then
        success "Resource limits declared in plugin.yaml"
    else
        error "No resource limits declared in plugin.yaml"
    fi
fi

# Summary
echo -e "\n=== Compliance Summary ==="
echo "Errors: $ERRORS"
echo "Warnings: $WARNINGS"

if [ "$ERRORS" -gt 0 ]; then
    echo -e "${RED}Plugin is NOT compliant with development guidelines${NC}"
    exit 1
elif [ "$WARNINGS" -gt 0 ]; then
    echo -e "${YELLOW}Plugin has warnings but meets minimum requirements${NC}"
    exit 0
else
    echo -e "${GREEN}Plugin is fully compliant!${NC}"
    exit 0
fi