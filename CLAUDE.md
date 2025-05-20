# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project: Blackhole

Blackhole is a distributed content sharing platform implemented as a subprocess architecture where services run as independent OS processes, providing operational simplicity while maintaining true service isolation through RPC communication.

## Project Structure

```
blackhole/
├── cmd/                    # Command-line applications
│   └── blackhole/          # Main binary
│       ├── main.go         # Application entry point
│       └── commands/       # CLI commands
│
├── internal/               # Private application code
│   ├── core/               # Core runtime
│   │   ├── orchestrator.go # Service orchestrator
│   │   ├── process.go      # Process management
│   │   └── config.go       # Configuration
│   │
│   ├── mesh/               # Internal service mesh
│   │   ├── router.go       # Request routing
│   │   ├── eventbus.go     # Event system
│   │   └── middleware.go   # Middleware chain
│   │
│   ├── services/           # Service implementations
│   │   ├── identity/       # Identity service (DIDs, registry, auth)
│   │   ├── storage/        # Storage service (IPFS, Filecoin)
│   │   ├── node/           # Node operations & P2P networking
│   │   ├── ledger/         # Ledger service (Root Network)
│   │   ├── indexer/        # Indexer service (SubQuery)
│   │   ├── social/         # Social service (ActivityPub)
│   │   ├── analytics/      # Analytics service
│   │   ├── telemetry/      # Telemetry service
│   │   └── wallet/         # Wallet service
│   │
│   └── plugins/            # Plugin system
│       ├── manager.go      # Plugin manager
│       └── builtin/        # Built-in plugins
│
├── pkg/                    # Public packages
│   ├── api/                # Public API clients
│   ├── types/              # Shared type definitions
│   └── sdk/                # SDK for developers
│
├── client-libs/            # Client libraries
│   ├── javascript/         # JavaScript/TypeScript SDK
│   ├── react/              # React components
│   └── mobile/             # Mobile SDKs
│
├── applications/           # Production-ready applications
│   ├── web-platform/       # Main web application
│   ├── mobile-app/         # React Native mobile app
│   ├── desktop-app/        # Electron desktop app
│   └── wallet-app/         # Self-managed wallet app
│
├── scripts/                # Build and utility scripts
├── configs/                # Configuration files
├── deployments/            # Deployment configurations
├── examples/               # Example applications
├── test/                   # Integration tests
├── docs/                   # Documentation
│   ├── architecture/       # Architecture documentation
│   ├── api/                # API documentation
│   ├── guides/             # Developer guides
│   └── tutorials/          # Tutorials
│
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
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
- Start specific services: `./bin/blackhole start --services=identity,storage`
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
- **IPFS/Filecoin**: Content storage
- **Root Network**: Blockchain integration
- **ActivityPub**: Social federation
- **gRPC**: Internal service communication
- **REST/GraphQL**: External APIs

## Development Workflow

1. **Service Development**: Implement services in `internal/services/`
2. **Core Development**: Work on orchestrator in `internal/core/`
3. **RPC Development**: Define gRPC services in `internal/rpc/`
4. **API Development**: Define public APIs in `pkg/api/`
5. **SDK Development**: Build SDK in `pkg/sdk/`
6. **Testing**: Write tests alongside code (using `_test.go` files)
7. **Documentation**: Update docs as you go

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