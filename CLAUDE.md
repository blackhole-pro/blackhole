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
│   │   └── services/       # Service implementations
│   │       ├── identity/   # Identity service
│   │       ├── node/       # Node service & P2P
│   │       ├── ledger/     # Ledger service
│   │       ├── indexer/    # Indexer service
│   │       ├── social/     # Social service
│   │       ├── analytics/  # Analytics service
│   │       ├── telemetry/  # Telemetry service
│   │       └── wallet/     # Wallet service
│   │
│   ├── pkg/                # Public packages and developer tools
│   │   ├── api/            # Public API clients
│   │   ├── sdk/            # SDK for developers
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
- Build all services: `make build-services`
- Build specific service: `make identity` (or any service name)
- Build for all platforms: `make build-all`

### Development

- Run the binary: `./bin/blackhole`
- Start all services: `./bin/blackhole start --all`
- Start specific services: `./bin/blackhole start --services=identity,node`
- Check service status: `./bin/blackhole status`
- View service logs: `./bin/blackhole logs identity`
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

1. **Service Development**: Implement services in `core/internal/services/`
2. **Core Development**: Work on components in `core/internal/core/`
3. **Runtime Development**: Work on orchestrator in `core/internal/runtime/`
4. **Framework Development**: Work on domains in `core/internal/framework/`
5. **RPC Development**: Define gRPC services in `core/internal/rpc/`
6. **API Development**: Define public APIs in `core/pkg/api/`
7. **SDK Development**: Build SDK in `core/pkg/sdk/`
8. **Testing**: Write tests in `core/test/` directory (NOT alongside code)
9. **Documentation**: Update docs as you go

## Service Architecture

Each service follows this structure:
```
internal/services/<service>/
├── main.go           # Service entry point
├── go.mod            # Service module definition
├── service.go        # Service implementation
├── handlers.go       # gRPC handlers
├── config.go         # Configuration structures
└── <service>_test.go # Service tests
```

## Adding a New Service

1. Create service directory: `mkdir -p internal/services/myservice`
2. Create service module: `cd internal/services/myservice && go mod init`
3. Add to workspace: Update `go.work` to include the new service
4. Create service main.go with gRPC server setup
5. Add to Makefile: Add build target for the service
6. Update configuration: Add service config to `configs/blackhole.yaml`

## Project Identifiers

- Atlas Project ID: proj_4e8d592d8b764c6eb5f7d8ce361d4bf1