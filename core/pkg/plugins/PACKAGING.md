# Blackhole Plugin Packaging Standard

## Overview

This document defines the standard for packaging, distributing, and installing Blackhole plugins. All plugins must follow these standards to ensure compatibility with the Blackhole ecosystem.

## Plugin Package Structure

A packaged plugin is distributed as a `.plugin` file (tar.gz archive) with the following structure:

```
myplugin-v1.0.0.plugin
├── plugin.yaml          # Plugin manifest (required)
├── bin/                 # Plugin binaries (required)
│   ├── darwin-amd64/    # macOS Intel binary
│   ├── darwin-arm64/    # macOS Apple Silicon binary
│   ├── linux-amd64/     # Linux x64 binary
│   └── linux-arm64/     # Linux ARM64 binary
├── proto/               # Plugin protobuf definitions (required)
│   └── v1/
│       └── myplugin.proto
├── docs/                # Plugin documentation (required)
│   ├── README.md        # User documentation
│   └── API.md           # API documentation
├── examples/            # Usage examples (optional)
├── LICENSE              # License file (required)
└── CHANGELOG.md         # Version history (optional)
```

## Plugin Manifest (plugin.yaml)

Every plugin must include a `plugin.yaml` manifest file:

```yaml
# Plugin metadata
name: node                      # Unique plugin identifier
version: 1.0.0                  # Semantic version
api_version: v1                 # Plugin API version
description: "P2P networking and node management plugin"
author: "Blackhole Foundation"
license: MIT
homepage: https://github.com/blackhole/plugins/node
repository: https://github.com/blackhole/plugins/node

# Plugin type and architecture
type: service                   # service, library, or hybrid
architecture:                   # Supported architectures
  - darwin-amd64
  - darwin-arm64
  - linux-amd64
  - linux-arm64

# Binary configuration
binary:
  name: node-plugin             # Binary name (without extension)
  main: bin/${GOOS}-${GOARCH}/ # Binary path template

# Network configuration
network:
  protocol: grpc                # Communication protocol
  transport: unix               # unix or tcp
  service_name: blackhole.plugin.node.v1.NodeService

# Resource requirements
resources:
  min_memory: 128M              # Minimum memory
  max_memory: 512M              # Maximum memory
  min_cpu: 0.1                  # Minimum CPU cores
  max_cpu: 1.0                  # Maximum CPU cores
  disk_space: 100M              # Required disk space

# Plugin dependencies
dependencies:
  mesh: ">=1.0.0"               # Mesh framework version
  plugins:                      # Other plugin dependencies
    - name: identity
      version: ">=1.0.0"

# Security capabilities
capabilities:
  - network                     # Network access
  - filesystem:read             # File system read
  - mesh:publish                # Mesh publishing
  - mesh:subscribe              # Mesh subscription

# Health check configuration
health:
  interval: 30s                 # Health check interval
  timeout: 5s                   # Health check timeout
  retries: 3                    # Number of retries

# Configuration schema
config_schema:
  type: object
  properties:
    bootstrap_peers:
      type: array
      items:
        type: string
      description: "Initial peer addresses"
    listen_address:
      type: string
      default: "/ip4/0.0.0.0/tcp/0"
      description: "P2P listen address"
```

## Binary Naming Convention

Plugin binaries must follow this naming convention:
- `<plugin-name>-<os>-<arch>`
- Example: `node-plugin-darwin-arm64`

Supported platforms:
- `darwin-amd64` - macOS Intel
- `darwin-arm64` - macOS Apple Silicon
- `linux-amd64` - Linux x64
- `linux-arm64` - Linux ARM64
- `windows-amd64` - Windows x64 (future)

## Plugin Versioning

Plugins must use semantic versioning (semver):
- Format: `MAJOR.MINOR.PATCH`
- Example: `1.2.3`

Version compatibility:
- API version changes require major version bump
- New features require minor version bump
- Bug fixes require patch version bump

## Plugin Categories

Plugins are categorized by type:

### Service Plugins
- Run as long-lived processes
- Provide gRPC services
- Example: node, identity, ledger

### Library Plugins
- Provide shared functionality
- Loaded on-demand
- Example: crypto, compression

### Hybrid Plugins
- Combine service and library features
- Example: storage (service + client library)

## Build Process

### 1. Plugin Development Structure

```
my-plugin/
├── go.mod
├── go.sum
├── Makefile
├── main.go
├── plugin.yaml
├── proto/
│   └── v1/
│       └── myplugin.proto
├── internal/
│   ├── service.go
│   └── handlers.go
├── docs/
│   ├── README.md
│   └── API.md
└── examples/
    └── basic/
        └── main.go
```

### 2. Makefile for Plugin Building

```makefile
PLUGIN_NAME := myplugin
VERSION := $(shell git describe --tags --always --dirty)
PLATFORMS := darwin-amd64 darwin-arm64 linux-amd64 linux-arm64

.PHONY: build
build: $(PLATFORMS)

$(PLATFORMS):
	GOOS=$(word 1,$(subst -, ,$@)) GOARCH=$(word 2,$(subst -, ,$@)) \
		go build -o bin/$@/$(PLUGIN_NAME) .

.PHONY: proto
proto:
	protoc --go_out=. --go-grpc_out=. proto/v1/*.proto

.PHONY: package
package: build
	@mkdir -p dist
	@tar -czf dist/$(PLUGIN_NAME)-$(VERSION).plugin \
		plugin.yaml \
		bin/ \
		proto/ \
		docs/ \
		examples/ \
		LICENSE \
		CHANGELOG.md

.PHONY: clean
clean:
	rm -rf bin/ dist/
```

### 3. Build Commands

```bash
# Build for all platforms
make build

# Generate protobuf code
make proto

# Create plugin package
make package

# Result: dist/myplugin-v1.0.0.plugin
```

## Installation Process

### 1. Local Installation

```bash
# Install plugin
blackhole plugin install ./myplugin-v1.0.0.plugin

# Install from URL
blackhole plugin install https://plugins.blackhole.io/node-v1.0.0.plugin

# Install from marketplace
blackhole plugin install node@1.0.0
```

### 2. Installation Locations

```
~/.blackhole/
├── plugins/                    # Installed plugins
│   ├── node/
│   │   ├── v1.0.0/            # Version directory
│   │   │   ├── plugin.yaml
│   │   │   ├── bin/
│   │   │   └── proto/
│   │   └── current -> v1.0.0  # Symlink to active version
│   └── identity/
│       └── v2.1.0/
├── plugin-cache/               # Downloaded packages
└── plugin-config/              # Plugin configurations
```

### 3. Plugin Discovery

The framework discovers plugins through:
1. Manifest registration in `~/.blackhole/plugins/`
2. Service announcement on mesh network
3. Health check verification

## Plugin Lifecycle

### 1. Installation
1. Verify package integrity
2. Extract to version directory
3. Verify binary signatures
4. Register in plugin registry
5. Create default configuration

### 2. Startup
1. Load plugin manifest
2. Check dependencies
3. Allocate resources
4. Start plugin process
5. Connect to mesh network
6. Verify health check

### 3. Runtime
1. Monitor health status
2. Enforce resource limits
3. Handle configuration updates
4. Manage service discovery

### 4. Shutdown
1. Graceful shutdown signal
2. Cleanup resources
3. Update plugin state
4. Remove from service discovery

### 5. Uninstall
1. Stop plugin if running
2. Remove from registry
3. Clean up data (optional)
4. Remove plugin directory

## Security Considerations

### 1. Package Signing
- All plugins should be signed
- Signature verification during install
- Publisher key management

### 2. Capability Model
- Plugins declare required capabilities
- Framework enforces capability restrictions
- Principle of least privilege

### 3. Resource Isolation
- Process-level isolation
- Resource limits enforcement
- Network namespace isolation (optional)

## Distribution Methods

### 1. Direct Distribution
- Host `.plugin` files on web servers
- Provide download URLs
- Include checksums

### 2. Plugin Marketplace
- Central registry for plugins
- Automated security scanning
- Version management
- Dependency resolution

### 3. Private Registries
- Enterprise plugin distribution
- Access control
- Audit logging

## Plugin Development Kit (PDK)

The PDK provides tools for plugin development:

```go
import "github.com/blackhole/core/pkg/sdk/plugin"

// Use the plugin client library
client := plugin.NewClient(&plugin.Config{
    ServiceName: "blackhole.plugin.myplugin.v1.MyService",
    Version:     "1.0.0",
})
```

## Best Practices

### 1. Manifest Guidelines
- Use descriptive names and descriptions
- Specify accurate resource requirements
- Declare all capabilities needed
- Keep dependencies minimal

### 2. Binary Guidelines
- Build with static linking when possible
- Strip debug symbols for release
- Include all target architectures
- Test on all platforms

### 3. Documentation Guidelines
- Provide clear README
- Document all API methods
- Include usage examples
- Maintain CHANGELOG

### 4. Security Guidelines
- Never include secrets in packages
- Use secure communication only
- Validate all inputs
- Handle errors gracefully

## Example: Node Plugin Package

Here's a complete example of packaging the node plugin:

### 1. Plugin Structure
```
node-plugin/
├── plugin.yaml
├── bin/
│   ├── darwin-amd64/node-plugin
│   ├── darwin-arm64/node-plugin
│   ├── linux-amd64/node-plugin
│   └── linux-arm64/node-plugin
├── proto/
│   └── v1/
│       └── node.proto
├── docs/
│   ├── README.md
│   └── API.md
├── examples/
│   └── basic-p2p/
│       └── main.go
├── LICENSE
└── CHANGELOG.md
```

### 2. Build and Package
```bash
# Build the plugin
cd node-plugin
make build

# Create package
make package

# Output: dist/node-v1.0.0.plugin
```

### 3. Install and Run
```bash
# Install
blackhole plugin install dist/node-v1.0.0.plugin

# Verify installation
blackhole plugin list

# Start the plugin
blackhole plugin start node
```

## Troubleshooting

### Common Issues

1. **Binary not found**
   - Ensure correct platform binary exists
   - Check binary permissions

2. **Dependency conflicts**
   - Use `blackhole plugin deps` to check
   - Update to compatible versions

3. **Resource limits**
   - Check system resources
   - Adjust plugin resource requirements

### Debug Commands

```bash
# Check plugin status
blackhole plugin status <plugin-name>

# View plugin logs
blackhole plugin logs <plugin-name>

# Inspect plugin manifest
blackhole plugin info <plugin-name>

# Verify plugin package
blackhole plugin verify <package-file>
```

## Migration Guide

For migrating existing services to plugins:

1. Create plugin.yaml manifest
2. Refactor to use plugin SDK
3. Build for all platforms
4. Package according to standard
5. Test installation process
6. Update documentation

## Future Enhancements

Planned improvements to packaging:
- WebAssembly plugin support
- Plugin composition/bundling
- Hot reload capabilities
- A/B testing support
- Automatic updates
- Plugin sandboxing