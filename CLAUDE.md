# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project: Blackhole

Blackhole is a distributed content sharing platform implemented as a subprocess architecture where services run as independent OS processes, providing operational simplicity while maintaining true service isolation through RPC communication.

## Project Structure

```
blackhole/
├── core/                   # Technical implementation
│   ├── cmd/                # Command-line applications
│   │   └── blackhole/      # Main binary
│   │       ├── main.go     # Application entry point
│   │       └── commands/   # CLI commands
│   │
│   ├── internal/           # Private application code
│   │   ├── core/           # Core components
│   │   │   ├── app/        # Application layer
│   │   │   └── version.go  # Version information
│   │   │
│   │   ├── framework/      # Framework domains
│   │   │   ├── economics/  # Economics domain
│   │   │   ├── mesh/       # Mesh networking domain
│   │   │   ├── plugins/    # Plugin management domain
│   │   │   └── resources/  # Resource management domain
│   │   │
│   │   ├── runtime/        # Runtime domain
│   │   │   ├── config/     # Configuration system
│   │   │   ├── orchestrator/ # Process orchestration
│   │   │   ├── lifecycle/  # Lifecycle management
│   │   │   └── dashboard/   # Runtime monitoring dashboard
│   │   │
│   │   ├── rpc/            # RPC definitions
│   │   │   ├── gen/        # Generated protobuf code
│   │   │   └── proto/      # Protocol definitions
│   │   │
│   │   └── services/       # (DEPRECATED - Moving to plugins)
│   │
│   ├── pkg/                # Public packages and developer tools
│   │   ├── api/            # Public API clients
│   │   ├── sdk/            # SDK for developers
│   │   ├── plugins/        # Plugin implementations
│   │   │   ├── node/       # P2P networking plugin
│   │   │   ├── identity/   # Identity management plugin
│   │   │   ├── storage/    # Distributed storage plugin
│   │   │   └── ...         # Other plugins
│   │   ├── tools/          # Developer tools
│   │   ├── templates/      # Project templates
│   │   └── types/          # Shared type definitions
│   │
│   ├── test/               # All tests
│   │   ├── unit/           # Unit tests
│   │   └── integration/    # Integration tests
│   │
│   ├── configs/            # Configuration files
│   ├── examples/           # Example applications
│   ├── scripts/            # Build and utility scripts
│   └── bin/                # Build artifacts
│
├── ecosystem/              # Community, governance, docs, and business
│   ├── docs/               # Documentation
│   │   ├── 04_domains/     # Domain documentation
│   │   ├── 05_architecture/# Architecture specs
│   │   ├── 06_guides/      # Developer guides
│   │   ├── 07_reference/   # API reference
│   │   └── 08_strategy/    # Strategy docs
│   ├── marketplace/        # Plugin marketplace
│   ├── partners/           # Partner network
│   ├── training/           # Education programs
│   ├── jobs/               # Career opportunities
│   ├── governance/         # Board and policies
│   ├── community/          # Community programs
│   ├── events/             # Conferences and meetups
│   ├── certification/      # Certification programs
│   └── enterprise/         # Enterprise solutions and support
│
├── applications/           # Reference applications
│
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── go.work                 # Go workspace
├── Makefile                # Build automation
├── README.md               # Project overview
└── CLAUDE.md               # AI assistant context
```

## Common Commands

### Building

- Build the main binary: `make build`
- Build all plugins: `make build-plugins`
- Build specific plugin: `make plugin-node` (or any plugin name)
- Package plugins: `make plugin-package`
- Build for all platforms: `make build-all`

### Development

- Run the binary: `./bin/blackhole`
- Start with plugins: `./bin/blackhole start --plugins=node`
- List loaded plugins: `./bin/blackhole plugins list`
- Check plugin status: `./bin/blackhole plugins status`
- Hot reload plugin: `./bin/blackhole plugins reload node`
- Run with hot reload: `make dev`

### Testing

- Run all tests: `make test`
- Run tests with race detection: `make test-race`
- Run tests with coverage: `make test-coverage`
- Run linter: `make lint`

### Docker

- Build docker image: `make docker-build`
- Run in docker: `docker run blackhole:latest`

### Dependency Management

- Install dependencies: `make deps`
- Update dependencies: `make update-deps`
- Clean build artifacts: `make clean`

## Architecture

The platform uses a subprocess architecture where services run as independent OS processes:

- **Subprocess Architecture**: Services run as separate OS processes
- **RPC Communication**: All service communication via gRPC
- **Process Isolation**: True isolation between services
- **Resource Control**: OS-level CPU, memory, and I/O limits
- **Service Management**: Individual service restart capability
- **Hot Updates**: Services can be restarted independently
- **Debugging**: Process-level profiling and monitoring

## Technology Stack

- **Go**: Primary language for the single binary
- **libp2p**: P2P networking
- **Root Network**: Blockchain integration
- **ActivityPub**: Social federation
- **gRPC**: Internal service communication
- **REST/GraphQL**: External APIs

## Development Workflow

1. **Plugin Development**: Implement plugins in `core/pkg/plugins/`
2. **Core Development**: Work on components in `core/internal/core/`
3. **Runtime Development**: Work on orchestrator in `core/internal/runtime/`
4. **Framework Development**: Work on domains in `core/internal/framework/`
5. **RPC Development**: Define gRPC services in plugin proto files
6. **API Development**: Define public APIs in `core/pkg/api/`
7. **SDK Development**: Build SDK in `core/pkg/sdk/`
8. **Testing**: Write tests in `core/test/` directory (NOT alongside code)
9. **Documentation**: Update docs as you go

## Plugin Architecture

Each plugin follows this structure:
```
pkg/plugins/<plugin>/
├── main.go           # Plugin entry point
├── go.mod            # Plugin module definition
├── plugin.yaml       # Plugin manifest
├── Makefile          # Plugin build system
├── README.md         # Plugin documentation
├── proto/
│   └── v1/
│       └── plugin.proto  # gRPC service definition
├── docs/             # Plugin documentation
└── examples/         # Usage examples
```

## Adding a New Plugin

1. Create plugin directory: `mkdir -p core/pkg/plugins/myplugin`
2. Create plugin module: `cd core/pkg/plugins/myplugin && go mod init`
3. Add to workspace: Update `go.work` to include the new plugin
4. Create plugin.yaml manifest with metadata and capabilities
5. Create main.go using the plugin SDK client
6. Add to Makefile: Add plugin to PLUGINS variable
7. Define gRPC interface in proto/v1/plugin.proto

## Project Identifiers

- Atlas Project ID: proj_4e8d592d8b764c6eb5f7d8ce361d4bf1