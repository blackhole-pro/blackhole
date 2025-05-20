# Single Binary Build System Design

This document outlines the build system design for the Blackhole platform's subprocess architecture.

## Overview

The build system creates a single executable that serves as both orchestrator and service subprocess:
- Single binary runs as orchestrator or service based on args
- Subprocess isolation with resource limits
- Conditional compilation for different profiles
- Cross-platform builds
- Version management

## Build Architecture

```
build/
├── cmd/
│   └── blackhole/          # Main binary entry point
│       └── main.go
├── internal/               # Internal packages
│   ├── core/              # Orchestrator core
│   ├── services/          # Service subprocess implementations
│   ├── rpc/              # gRPC communication
│   └── plugins/           # Plugin system
├── pkg/                   # Public packages
│   ├── api/              # Public APIs
│   ├── types/            # Shared types
│   └── utils/            # Utilities
├── scripts/              # Build scripts
│   ├── build.sh
│   ├── release.sh
│   └── version.sh
└── configs/              # Build configurations
    ├── dev.yaml
    ├── prod.yaml
    └── test.yaml
```

## Build System Components

### 1. Main Binary Structure

```go
// cmd/blackhole/main.go
package main

import (
    "github.com/blackhole/blackhole/internal/core"
    "github.com/blackhole/blackhole/internal/services"
    _ "github.com/blackhole/blackhole/internal/plugins/builtin"
)

func main() {
    app := core.NewApplication()
    
    // Register core services
    app.RegisterService(services.NewIdentityService())
    app.RegisterService(services.NewStorageService())
    app.RegisterService(services.NewP2PService())
    app.RegisterService(services.NewLedgerService())
    app.RegisterService(services.NewIndexerService())
    app.RegisterService(services.NewSocialService())
    app.RegisterService(services.NewAnalyticsService())
    app.RegisterService(services.NewTelemetryService())
    
    // Start application
    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### 2. Build Tags and Conditional Compilation

```go
// Production build includes all services
// +build production

package services

import (
    "github.com/blackhole/blackhole/internal/services/identity"
    "github.com/blackhole/blackhole/internal/services/storage"
    // ... all services
)

func RegisterAllServices(app *core.Application) {
    app.RegisterService(identity.NewService())
    app.RegisterService(storage.NewService())
    // ... register all services
}
```

```go
// Development build can exclude certain services
// +build development,!minimal

package services

func RegisterAllServices(app *core.Application) {
    // Register only essential services for development
}
```

### 3. Build Configuration

```yaml
# configs/prod.yaml
build:
  name: blackhole
  version: ${VERSION}
  
  tags:
    - production
    - with_telemetry
    - with_analytics
    
  features:
    identity: true
    storage: true
    p2p: true
    ledger: true
    indexer: true
    social: true
    analytics: true
    telemetry: true
    
  resources:
    embed_ui: true
    embed_migrations: true
    embed_configs: true
    
  optimizations:
    strip_debug: true
    compress: true
    static_link: true
```

### 4. Build Script

```bash
#!/bin/bash
# scripts/build.sh

set -e

# Load configuration
CONFIG=${1:-prod}
VERSION=$(./scripts/version.sh)

# Parse build tags from config
TAGS=$(yq e '.build.tags | join(",")' configs/${CONFIG}.yaml)

# Set build variables
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=$(date -u +%Y%m%d%H%M%S)"

if [ "$CONFIG" = "prod" ]; then
    LDFLAGS="$LDFLAGS -s -w"  # Strip debug info
fi

# Build binary
echo "Building blackhole ${VERSION} with config: ${CONFIG}"
go build \
    -tags "${TAGS}" \
    -ldflags "${LDFLAGS}" \
    -o dist/blackhole \
    ./cmd/blackhole

# Compress if production
if [ "$CONFIG" = "prod" ]; then
    upx dist/blackhole
fi

echo "Build complete: dist/blackhole"
```

### 5. Resource Embedding

```go
package resources

import (
    "embed"
)

//go:embed ui/dist/*
var UIAssets embed.FS

//go:embed migrations/*.sql
var Migrations embed.FS

//go:embed configs/default.yaml
var DefaultConfig []byte
```

## Build Profiles

### Development Build
- Fast compilation
- Debug symbols included
- Hot reload support
- Optional service loading
- Verbose logging

```bash
make build-dev
```

### Test Build
- Test helpers included
- Mock services available
- Coverage instrumentation
- Race detection enabled

```bash
make build-test
```

### Production Build
- All services included
- Optimized binary size
- Security hardening
- Release optimizations
- Signed binaries

```bash
make build-prod
```

## Plugin Build System

### Compiled-in Plugins

```go
// internal/plugins/builtin/register.go
package builtin

import (
    _ "github.com/blackhole/blackhole/plugins/auth/oauth"
    _ "github.com/blackhole/blackhole/plugins/storage/s3"
    _ "github.com/blackhole/blackhole/plugins/cache/redis"
)
```

### Dynamic Plugin Loading (Development)

```bash
# Build plugin
go build -buildmode=plugin \
    -o plugins/custom.so \
    ./plugins/custom

# Load at runtime
./blackhole --plugin-dir=./plugins
```

## Cross-Platform Builds

```makefile
# Makefile
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

build-all:
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		go build -o dist/blackhole-$${platform%/*}-$${platform#*/} \
		./cmd/blackhole; \
	done
```

## Version Management

```go
// internal/core/version.go
package core

var (
    Version   = "development"
    BuildTime = "unknown"
    GitCommit = "unknown"
)

func VersionInfo() string {
    return fmt.Sprintf("Blackhole %s (built %s, commit %s)",
        Version, BuildTime, GitCommit)
}
```

## CI/CD Integration

```yaml
# .github/workflows/build.yml
name: Build

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build
        run: make build-prod
      
      - name: Test
        run: make test
      
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: blackhole-binary
          path: dist/blackhole
```

## Dependency Management

```go
// go.mod
module github.com/blackhole/blackhole

go 1.21

require (
    github.com/spf13/cobra v1.7.0
    github.com/spf13/viper v1.16.0
    github.com/go-kit/kit v0.13.0
    github.com/prometheus/client_golang v1.16.0
    // ... other dependencies
)
```

## Build Optimization

### Size Optimization
```bash
# Remove debug symbols
go build -ldflags="-s -w"

# Use UPX compression
upx --best blackhole

# Remove unused code
go build -trimpath
```

### Performance Optimization
```bash
# Enable compiler optimizations
go build -gcflags="-l=4"

# Profile-guided optimization
go build -pgo=profile.pprof
```

## Security Considerations

### Code Signing
```bash
# Sign binary (macOS)
codesign --sign "Developer ID" dist/blackhole

# Sign binary (Windows)
signtool sign /a dist/blackhole.exe
```

### Build Reproducibility
```bash
# Ensure reproducible builds
go build -trimpath \
    -ldflags="-buildid= -X main.Version=${VERSION}" \
    ./cmd/blackhole
```

## Development Workflow

1. **Local Development**
   ```bash
   make dev
   # Starts with hot reload
   ```

2. **Testing**
   ```bash
   make test
   make test-integration
   ```

3. **Building**
   ```bash
   make build
   ```

4. **Release**
   ```bash
   make release VERSION=v1.0.0
   ```

This build system provides a flexible, maintainable approach to creating the single binary while supporting various deployment scenarios and development workflows.