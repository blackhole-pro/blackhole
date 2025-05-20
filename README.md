# Blackhole

Blackhole is a truly decentralized content sharing platform that puts users in control of their data while enabling engaging social experiences. Built on a foundation of IPFS/Filecoin for storage, DIDs for self-sovereign identity, and ActivityPub for social federation, Blackhole creates a new paradigm for content sharing that balances user sovereignty with service provider innovation.

## Three-Layer Architecture

Blackhole's architecture is designed with decentralization at its core:

1. **End Users (Clients)** - Lightweight web and mobile applications that interact with content and social features
2. **Service Providers** - Organizations building branded experiences on the Blackhole platform while preserving user sovereignty
3. **Blackhole Nodes** - Decentralized P2P network handling storage, processing, identity, and federation

## Key Features

- **Self-Sovereign Identity** - DID-based identity with verifiable credentials
- **Single-Transfer Content Flow** - Content uploaded directly to Blackhole nodes, optimizing bandwidth
- **Decentralized Storage** - IPFS for content addressing and Filecoin for persistent storage
- **Content Ledger** - Root Network blockchain for ownership and transaction records
- **Social Federation** - ActivityPub-compatible social interactions that connect with the fediverse
- **Service Provider SDK** - Comprehensive tools for building on the platform with minimal complexity
- **Privacy-Preserving Analytics** - Content consumption tracking with user privacy controls
- **P2P Infrastructure** - Distributed node network using libp2p for robust infrastructure

## Subprocess Architecture

Blackhole uses a subprocess architecture that distributes as a single binary but runs services as independent OS processes. This approach provides:

- **Deployment Simplicity** - One binary to distribute and deploy
- **Service Isolation** - Process crashes don't affect other services  
- **Individual Control** - Restart services independently
- **Resource Management** - OS-level CPU, memory, and I/O limits
- **Security Boundaries** - Process-level security isolation

Services communicate via gRPC over Unix sockets locally and TCP for remote communication.

## Project Structure

```
blackhole/
├── cmd/                    # Command-line applications
│   └── blackhole/          # Main binary (orchestrator)
├── internal/               # Private application code
│   ├── core/               # Core orchestrator
│   │   ├── orchestrator.go # Service orchestrator
│   │   ├── process.go      # Process management
│   │   └── config.go       # Configuration
│   ├── rpc/                # RPC communication
│   │   ├── client.go       # gRPC clients
│   │   ├── server.go       # gRPC servers
│   │   └── registry.go     # Service registry
│   ├── services/           # Service implementations
│   │   ├── identity/       # Identity service
│   │   ├── storage/        # Storage service
│   │   ├── ledger/         # Ledger service
│   │   ├── social/         # Social service
│   │   └── ...             # Other services
│   └── plugins/            # Plugin system
├── pkg/                    # Public packages
│   ├── api/                # API clients
│   ├── types/              # Type definitions
│   └── sdk/                # Developer SDK
├── client-libs/            # Client libraries
│   ├── javascript/         # JS/TS SDK
│   ├── react/              # React components
│   └── mobile/             # Mobile SDKs
└── docs/                   # Documentation
```

For detailed architecture, see [PROJECT.md](./PROJECT.md) and [subprocess_architecture.md](./docs/architecture/subprocess_architecture.md).

## Getting Started

### Prerequisites

- Go 1.21+
- Node.js 18+ (for client SDKs)
- Git

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/blackhole.git
   cd blackhole
   ```

2. Build the binary:
   ```bash
   make build
   ```

3. Start all services:
   ```bash
   ./blackhole start --all
   ```

### Service Management

```bash
# Start specific services
./blackhole start --services=identity,storage,ledger

# Check service status
./blackhole status

# Restart a service
./blackhole restart identity

# View service logs
./blackhole logs storage --follow

# Stop services
./blackhole stop --all
```

## Development

Blackhole is primarily written in Go, with client SDKs in various languages.

### Common Commands

```bash
# Build the main binary
make build

# Run tests
make test

# Run with hot reload
make dev

# Build for all platforms
make build-all

# Run linter
make lint

# Generate protobuf files
make proto

# Build client SDKs
cd client-libs/javascript && npm run build
```

### Configuration

Services are configured via YAML:

```yaml
# blackhole.yaml
orchestrator:
  socket_dir: /var/run/blackhole
  log_level: info

services:
  identity:
    resources:
      cpu: 200      # 2 CPU cores
      memory: 1024  # 1GB RAM
    config:
      database: postgres://localhost/identity
```

## For Service Providers

Blackhole empowers service providers to build branded experiences on a decentralized foundation. By handling the complex infrastructure, Blackhole allows providers to focus on creating value for their users. The client SDK provides:

- **User Authentication** - DID-based authentication with verifiable credentials
- **Content Orchestration** - Single-transfer uploads with progress tracking
- **Discovery Engine** - Content search, recommendations, and trending
- **Social Integration** - ActivityPub-compatible social interactions
- **UI Components** - Customizable, themeable component library
- **Analytics Dashboard** - Privacy-preserving user engagement metrics
- **Developer Resources** - Comprehensive documentation and examples

## Documentation

- **Architecture**: [PROJECT.md](./PROJECT.md) - Detailed platform architecture and design
- **Subprocess Architecture**: [docs/architecture/subprocess_architecture.md](./docs/architecture/subprocess_architecture.md) - Process management patterns
- **RPC Communication**: [docs/architecture/rpc_communication.md](./docs/architecture/rpc_communication.md) - gRPC patterns and practices
- **Service Lifecycle**: [docs/architecture/service_lifecycle.md](./docs/architecture/service_lifecycle.md) - Startup, shutdown, and restart procedures
- **Current Status**: [CURRENT_STATUS.md](./CURRENT_STATUS.md) - Project progress and next steps
- **User Flows**: [docs/flowcharts/](./docs/flowcharts/) - Visual diagrams of key platform processes
- **Data Models**: [docs/ast/](./docs/ast/) - JSON Schema AST models for core data structures
- **Service Documentation**: [docs/architecture/services/](./docs/architecture/services/) - Individual service architectures
- **Developer Guides**: Coming soon - Tutorials and implementation guides

## Decentralized Ecosystem

Blackhole is more than a platform—it's a decentralized ecosystem that:

- Preserves user sovereignty through DIDs and decentralized storage
- Enables service provider innovation without compromising privacy
- Connects to the broader fediverse through ActivityPub
- Leverages IPFS, Filecoin, and Root Network for robust infrastructure
- Optimizes content flows for bandwidth efficiency
- Provides comprehensive analytics while respecting privacy

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! The Blackhole platform is designed to be community-driven with a focus on creating a truly decentralized ecosystem. Please see our [contributing guidelines](./CONTRIBUTING.md) for more information.