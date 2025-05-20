# Blackhole: Distributed Content Sharing Platform

## Project Overview

Blackhole is an open-source distributed content sharing platform built on decentralized technologies. It leverages IPFS and Filecoin for content storage, Root Network for content ledger functionality, decentralized identifiers (DIDs) for self-sovereign identity, and provides a comprehensive SDK for developers to build their own applications.

## Core Principles

1. **Decentralized Infrastructure**: Content storage and delivery without central points of failure
2. **Open Source Development**: Community-driven innovation and transparency
3. **Developer-First**: Comprehensive SDKs and tools for building on the platform
4. **User Ownership**: Users maintain control of their content and data
5. **Modular Architecture**: Composable components that can be used independently
6. **Self-Sovereign Identity**: Users control their own identity through DIDs
7. **Efficient Data Flow**: Optimized for single-transfer content handling

## Platform Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Client Applications                      │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │Web Platform │    │ Mobile App  │    │ Desktop App │     │
│  └─────────────┘    └─────────────┘    └─────────────┘     │
├─────────────────────────────────────────────────────────────┤
│               Client SDKs & Libraries                       │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │ JavaScript  │    │    React    │    │   Native    │     │
│  └─────────────┘    └─────────────┘    └─────────────┘     │
├─────────────────────────────────────────────────────────────┤
│                  Blackhole Orchestrator                     │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                Process Manager                       │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐ │   │
│  │ │Identity │  │ Storage │  │ Ledger  │  │ Social  │ │   │
│  │ │ Process │  │ Process │  │ Process │  │ Process │ │   │
│  │ └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘ │   │
│  │ ┌────┴────┐  ┌────┴────┐  ┌────┴────┐  ┌────┴────┐ │   │
│  │ │ Indexer │  │  Node   │  │Analytics│  │Telemetry│ │   │
│  │ │ Process │  │ Process │  │ Process │  │ Process │ │   │
│  │ └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘ │   │
│  │ ┌────┴────┐                                       │   │
│  │ │ Wallet  │                                       │   │
│  │ │ Process │                                       │   │
│  │ └────┬────┘                                       │   │
│  │      │             │             │             │     │   │
│  │      └─────────────┼─────────────┼─────────────┘     │   │
│  │                    │     gRPC    │                   │   │
│  │              ┌─────┴─────────────┴─────┐            │   │
│  │              │    Unix Sockets / TCP   │            │   │
│  │              └─────────────────────────┘            │   │
│  └─────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│                    External Services                        │
│  ┌─────────┐  ┌─────────┐  ┌─────────────────────┐         │
│  │  IPFS   │  │Filecoin │  │    Root Network     │         │
│  └─────────┘  └─────────┘  └─────────────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

### Key Components

1. **Content Storage**: IPFS for content addressing and Filecoin for persistent storage
2. **Content Ledger**: Root Network blockchain for ownership and transaction records
3. **Content Discovery**: Indexing and search functionality for content discovery
4. **Node Management**: Peer discovery and network topology management
5. **Social Interactions**: Comment, follow, like, and share functionality using ActivityPub
6. **Analytics**: Comprehensive content consumption tracking and engagement metrics
7. **Telemetry**: System health monitoring and performance tracking
8. **Client SDKs**: Cross-platform libraries for application development
9. **UI Components**: Reusable React components for building interfaces
10. **Identity System**: DID-based self-sovereign identity with verifiable credentials
11. **Wallet Service**: Dual-mode wallet system supporting decentralized and self-managed options

### Architectural Layers

The platform is designed with three distinct layers that separate responsibilities and enable a decentralized ecosystem:

1. **End Users (Clients)**
   - Lightweight web/mobile applications with minimal processing
   - Focused on content consumption and creation
   - Authentication via DIDs
   - Simple user interfaces for content interactions
   - No direct P2P responsibilities

2. **Service Providers (Intermediaries)**
   - Organizations building branded experiences on the Blackhole platform
   - Operate interfaces to the Blackhole network
   - Manage user experience and application logic
   - Orchestrate content uploads and retrievals
   - Verify DID-based authentication
   - Issue verifiable credentials
   - Add value-added services on top of core functionality

3. **Blackhole Nodes (Infrastructure)**
   - Decentralized P2P network
   - Handle content processing, storage, and distribution
   - IPFS/Filecoin integration
   - DID registry for decentralized identity
   - Node-to-node communication protocols
   - Content indexing and search
   - Analytics and telemetry collection

### Optimized Content Flow

The platform uses a single-transfer content flow to optimize bandwidth usage:

1. End user requests upload permission from service provider
2. Service provider obtains signed upload URL from Blackhole node
3. End user uploads content directly to Blackhole node (single transfer)
4. Blackhole node processes content (chunking, hashing, CID generation)
5. Blackhole node stores content on IPFS/Filecoin
6. Service provider receives notification with content CID
7. Content is available across the network

## Project Structure

The Blackhole platform uses a subprocess architecture where services run as independent OS processes orchestrated by a single binary, providing operational simplicity while maintaining true service isolation.

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

## Subprocess Architecture

The Blackhole platform uses a subprocess architecture where a single binary distributes as one file but spawns services as independent OS processes. This provides deployment simplicity while maintaining true service isolation.

### Architectural Overview

```
blackhole (orchestrator binary)
├── Process Manager            # Subprocess lifecycle management
├── Service Processes          # Independent OS processes
│   ├── Identity Process       # DIDs, registry, auth
│   ├── Storage Process        # IPFS, Filecoin
│   ├── Node Process           # Node operations & P2P networking
│   ├── Ledger Process         # Blockchain integration
│   ├── Indexer Process        # Content indexing
│   ├── Social Process         # ActivityPub
│   ├── Analytics Process      # Metrics collection
│   ├── Telemetry Process      # System monitoring
│   └── Wallet Process         # Wallet management
├── RPC Communication          # gRPC-based IPC
│   ├── Unix Sockets (local)   # High-performance local communication
│   └── TCP + TLS (remote)     # Secure remote communication
└── Resource Control           # OS-level resource management
    ├── CPU Quotas             # Process-specific CPU limits
    ├── Memory Limits          # Process-level memory control
    └── I/O Prioritization     # I/O resource allocation
```

### Key Benefits
- **Operational Simplicity**: Single binary to distribute and deploy
- **True Isolation**: Process crashes don't affect other services
- **Resource Control**: OS-level CPU, memory, and I/O limits
- **Independent Updates**: Restart individual services without downtime
- **Better Debugging**: Process-level profiling and monitoring
- **Future Flexibility**: Easy migration path to microservices
- **Security**: Process-level security boundaries

### Service Communication
All inter-service communication uses gRPC:
- **Local Communication**: Unix domain sockets for minimal overhead
- **Remote Communication**: TCP with mTLS for security
- **Service Discovery**: Automatic service registration and discovery
- **Connection Pooling**: Efficient connection reuse
- **Health Checking**: Built-in health monitoring

### Resource Management
Each service runs with specific resource allocations:
```yaml
identity:
  cpu: 200%      # 2 CPU cores
  memory: 1GB    # Memory limit
  io_weight: 500 # Medium I/O priority

storage:
  cpu: 100%      # 1 CPU core
  memory: 2GB    # Higher memory for caching
  io_weight: 900 # High I/O priority

ledger:
  cpu: 150%      # 1.5 CPU cores
  memory: 1GB    # Memory limit
  network: 100Mbps # High network usage
```

### Process Supervision
- **Automatic Restart**: Failed services are automatically restarted
- **Exponential Backoff**: Prevents rapid restart loops
- **Health Monitoring**: Regular health checks for each process
- **Graceful Shutdown**: Coordinated shutdown procedures
- **State Persistence**: Services maintain state across restarts

### Client Libraries

Separate client libraries are provided for different platforms:

```
client-libs/
├── javascript/               # JavaScript/TypeScript SDK
│   ├── src/
│   │   ├── identity.ts      # Identity operations
│   │   ├── storage.ts       # Storage operations
│   │   ├── social.ts        # Social interactions
│   │   └── client.ts        # Main client interface
│   └── dist/                # Compiled output
│
├── react/                   # React component library
│   ├── components/          # UI components
│   ├── hooks/               # React hooks
│   └── providers/           # Context providers
│
└── mobile/                  # Mobile SDKs
    ├── ios/                 # iOS SDK
    └── android/             # Android SDK
```

These client libraries communicate with the Blackhole binary through standard APIs (REST, gRPC, WebSocket).

### Key Domain Functionality

#### Identity System

The identity system is implemented as a service subprocess with complete DID functionality:

```
internal/services/identity/               # Identity service implementation
├── main.go                              # Service entry point
├── service.go                           # Service implementation
├── did/                                 # DID operations
│   ├── creation.go                      # DID creation logic
│   ├── resolution.go                    # DID resolution logic
│   └── verification.go                  # DID verification logic
├── registry/                            # DID registry
├── credentials/                         # Credential services
└── authentication/                      # Authentication service

pkg/api/identity/                        # Identity API clients
├── client.go                            # gRPC client implementation
├── types.go                            # Shared types
└── identity.proto                       # Protocol buffer definitions
```

This structure provides a complete self-sovereign identity solution with:
- DID creation and management
- Verifiable credential verification
- DID-based authentication
- Service provider integration with DIDs

#### Wallet System

The wallet system implements both decentralized and self-managed wallet options:

```
internal/services/wallet/                 # Wallet service implementation
├── main.go                              # Service entry point
├── service.go                           # Service implementation  
├── storage.go                           # Encrypted storage service
├── api.go                               # Wallet API endpoints
└── sync.go                              # Synchronization service

pkg/api/wallet/                          # Wallet API clients
├── client.go                            # gRPC client implementation
├── types.go                             # Shared types
└── wallet.proto                         # Protocol buffer definitions

applications/wallet-app/                 # User-facing wallet application
├── web/                                 # Web implementation
├── mobile/                              # Mobile implementation
└── desktop/                             # Desktop implementation
```

#### Content Storage

Decentralized content storage through IPFS and Filecoin:

```
internal/services/storage/                # Storage service implementation
├── main.go                              # Service entry point
├── service.go                           # Service implementation
├── ipfs/                                # IPFS integration
├── filecoin/                            # Filecoin integration
└── content/                             # Content management

pkg/api/storage/                         # Storage API clients
├── client.go                            # gRPC client implementation
├── types.go                             # Shared types
└── storage.proto                        # Protocol buffer definitions
```

#### Social Interactions

ActivityPub-based social networking capabilities:

```
internal/services/social/                 # Social service implementation
├── main.go                              # Service entry point
├── service.go                           # Service implementation
├── activitypub/                         # ActivityPub implementation
└── federation/                          # Federation protocols

pkg/api/social/                          # Social API clients
├── client.go                            # gRPC client implementation
├── types.go                             # Shared types
└── social.proto                         # Protocol buffer definitions
```

#### Analytics and Telemetry

Privacy-preserving metrics collection:

```
internal/services/analytics/              # Analytics service implementation
├── main.go                              # Service entry point
├── service.go                           # Service implementation
└── collection/                          # Data collection systems

internal/services/telemetry/              # Telemetry service implementation
├── main.go                              # Service entry point
├── service.go                           # Service implementation
└── monitoring/                          # System monitoring

pkg/api/analytics/                       # Analytics API clients
├── client.go                            # gRPC client implementation
├── types.go                             # Shared types
└── analytics.proto                      # Protocol buffer definitions
```

## Applications

### Web Platform
The main web application showcasing the platform's capabilities.
- Content discovery and browsing
- Creator dashboard
- User profiles and social interactions
- Media consumption
- Administrative tools

### Mobile App
React Native mobile application for iOS and Android.
- Mobile-optimized content browsing
- Native media playback
- Push notifications
- Camera integration
- Offline capabilities

### Desktop App
Electron desktop application for enhanced functionality.
- Enhanced local caching
- System tray integration
- File system integration
- Background processing
- Advanced content tools

## Implementation Strategy

The implementation follows a phased approach based on our single binary architecture design.

### Phase 1: Foundation (Weeks 1-2)
- Set up Go project structure
- Implement process orchestrator
- Develop subprocess lifecycle manager
- Create internal service mesh with gRPC
- Build configuration system (Viper)
- Establish logging and metrics foundation

### Phase 2: Service Implementation (Weeks 3-5)
- Implement Identity Service subprocess (foundational)
- Add Storage Service subprocess with IPFS/Filecoin
- Create Node Service subprocess for P2P networking
- Build Ledger Service subprocess for Root Network
- Add Wallet Service subprocess with dual-mode support
- Add Indexer Service subprocess
- Implement Social Service subprocess with ActivityPub
- Create Analytics Service subprocess
- Add Telemetry Service subprocess

### Phase 3: Resource Management (Weeks 6-7)
- Implement memory pools per service
- Add CPU quota enforcement
- Create resource monitoring
- Implement circuit breakers
- Add health check system
- Build recovery mechanisms

### Phase 4: Plugin System (Weeks 8-9)
- Create plugin interfaces
- Implement plugin registry
- Build hook system for extensibility
- Add compiled-in plugins
- Create plugin lifecycle management
- Implement security sandboxing

### Phase 5: Deployment & Operations (Weeks 10-11)
- Create deployment packages
- Build CLI management interface
- Add configuration validation
- Implement performance profiling
- Create diagnostic utilities
- Write operational documentation

### Phase 6: Client Libraries (Weeks 12-13)
- Develop JavaScript/TypeScript SDK
- Create React component library
- Build mobile SDKs (iOS/Android)
- Implement example applications
- Write developer documentation
- Create integration tests

## Analytics & Privacy Considerations

Our analytics and telemetry systems are designed with privacy as a core principle, following these guidelines:

### Privacy-First Design

1. **Consent Management**
   - Clear opt-in/opt-out mechanisms for different levels of analytics
   - Granular controls for users to manage what data is collected
   - Transparent explanations of how data is used
   - Local storage options for analytics data

2. **Data Minimization**
   - Collection of only necessary data points
   - Automatic data aggregation where possible
   - Limited retention periods
   - Regular data purging schedules

3. **Anonymization & Pseudonymization**
   - Client-side anonymization before data transmission
   - Differential privacy techniques for aggregate reports
   - Data segmentation to prevent correlation
   - Pseudonymous identifiers instead of personal information

4. **Federated Analytics Architecture**
   - Local-first processing of sensitive metrics
   - Node-level storage and aggregation
   - Optional federation of anonymized aggregate data
   - Distributed storage matching our decentralized architecture

5. **Transparency & Control**
   - Open analytics dashboards for users to see their own data
   - Data export capabilities
   - One-click data deletion
   - Clear documentation on collection practices

### Implementation Approach

- Event-based analytics collection with privacy filters
- Blockchain verification for critical metrics where appropriate
- Secure storage with encryption at rest
- Strong access controls for analytics dashboards
- Regular privacy audits and impact assessments

## Technology Choices

Our technology stack is optimized for the single binary architecture with Go as the primary language.

### Core Platform (Single Binary)
- **Go**: Primary language for the single binary implementation
- **Go Modules**: Dependency management
- **gRPC**: Internal service communication
- **Protocol Buffers**: Service interface definitions
- **IPFS Go SDK**: Content addressing and distribution
- **Filecoin Go SDK**: Persistent storage integration
- **libp2p**: P2P networking stack
- **BoltDB/BadgerDB**: Embedded database for local state
- **Prometheus**: Metrics collection
- **OpenTelemetry**: Distributed tracing

### External Integrations
- **Root Network SDK**: Blockchain integration for content ledger
- **ActivityPub**: Social federation protocol
- **DIDs**: Decentralized identity standard
- **Verifiable Credentials**: Identity attestations
- **Ed25519/secp256k1**: Cryptographic algorithms

### Client Libraries
- **TypeScript**: Primary language for JavaScript SDK
- **React**: Component library framework
- **React Native**: Mobile SDK framework
- **Swift**: iOS SDK implementation
- **Kotlin**: Android SDK implementation
- **gRPC-Web**: Browser-based API communication
- **WebSockets**: Real-time client connections

### Applications
- **React**: Web application framework
- **React Native**: Mobile application framework
- **Electron**: Desktop application framework
- **Tailwind CSS**: Styling system
- **Next.js**: Web application optimization

### Development Infrastructure

For managing a Go single binary with multiple subprocess services, we use a combination of Go-native tools and established practices:

#### Monorepo Management
- **Go Modules**: Native dependency management with workspace support
  ```go
  // go.work (Go 1.21+)
  go 1.22

  use (
      .
      ./cmd/blackhole
      ./internal/services/identity
      ./internal/services/storage
      ./internal/services/node
      ./internal/services/ledger
      ./internal/services/indexer
      ./internal/services/social
      ./internal/services/analytics
      ./internal/services/telemetry
      ./internal/services/wallet
      ./pkg/api
      ./pkg/sdk
  )
  ```
- **Module Structure**: Each service can be its own module for clear dependency boundaries
- **Shared Dependencies**: Common code in `pkg/` directory
- **Local Development**: `go.work` files for local multi-module development

#### Build Automation
- **Make**: Coordinated builds across all services
  ```makefile
  # Build main binary
  build:
      go build -o bin/blackhole ./cmd/blackhole

  # Build all services
  all: identity storage node ledger indexer social analytics telemetry wallet

  # Individual service builds
  identity:
      go build -o bin/identity ./internal/services/identity

  # Run all tests
  test:
      go test ./...

  # Cross-compilation
  build-all:
      GOOS=linux GOARCH=amd64 make build
      GOOS=darwin GOARCH=amd64 make build
      GOOS=windows GOARCH=amd64 make build
  ```
- **go generate**: Code generation for protobuf, mocks, etc.
- **GoReleaser**: Automated releases with cross-compilation

#### Service Development
- **protoc**: Protocol buffer compilation for gRPC services
- **mockgen**: Generate mocks for testing
- **wire**: Dependency injection code generation

#### Quality & Testing
- **golangci-lint**: Comprehensive linting across all modules
- **go test**: Unit and integration testing
  ```bash
  # Test all modules
  go test ./...
  
  # Test with race detection
  go test -race ./...
  
  # Coverage across modules
  go test -coverprofile=coverage.out ./...
  ```
- **go bench**: Performance benchmarking
- **go vet**: Static analysis

#### CI/CD Infrastructure
- **GitHub Actions**: Automated testing and deployment
  ```yaml
  # Test matrix for all services
  strategy:
    matrix:
      service: [identity, storage, node, ledger, indexer, social, analytics, telemetry, wallet]
  ```
- **Docker**: Multi-stage builds for each service
- **Container Registry**: Service-specific image management

#### Development Tools
- **air**: Hot reload for Go services during development
- **delve**: Go debugger for service debugging
- **pprof**: Performance profiling for each service
- **go mod tidy**: Dependency management per service
- **go work sync**: Synchronize workspace dependencies

## Licensing & Governance

### Licensing
- **Core & Protocol**: Apache 2.0
- **Client Libraries**: MIT
- **Examples**: MIT

### Governance
- **Decision Making**: RFC process
- **Contribution Process**: Pull request workflow
- **Community Building**: Discord, forums, hackathons
- **Sustainability**: Optional hosted services, support contracts

## Next Steps

Following our single binary architecture design:

1. Set up Go project structure with modules
2. Implement subprocess orchestrator in `internal/core`
3. Create RPC communication layer in `internal/rpc`
4. Build process manager and service registry
5. Implement first service (Identity) as subprocess
6. Add remaining services as independent processes
7. Create gRPC API definitions for all services
8. Develop JavaScript/TypeScript client SDK
9. Build example applications
10. Write comprehensive documentation

## Deployment and Operations

### Starting Services

The Blackhole binary provides commands to manage service processes:

```bash
# Start all services
blackhole start --all

# Start specific services
blackhole start --services=identity,storage,ledger

# Start with custom resource limits
blackhole start --services=storage --cpu=200 --memory=4096

# Start with development configuration
blackhole start --config=dev.yaml
```

### Service Management

```bash
# View service status
blackhole status

# Restart a crashed service
blackhole restart identity

# Stop a specific service
blackhole stop storage

# View service logs
blackhole logs identity --follow

# Check service health
blackhole health ledger
```

### Monitoring and Debugging

Since services run as separate processes, standard tools can be used:

```bash
# Monitor process resources
htop -p $(blackhole pid identity)

# Profile a service
blackhole profile storage --cpu --duration=30s

# Trace RPC calls
blackhole trace --service=ledger --method=Transfer

# Export metrics
blackhole metrics --format=prometheus
```

### Configuration

Services are configured through YAML files:

```yaml
# blackhole.yaml
orchestrator:
  socket_dir: /var/run/blackhole
  log_level: info
  
services:
  identity:
    enabled: true
    resources:
      cpu: 200
      memory: 1024
      io_weight: 500
    config:
      database: postgres://identity:pass@localhost/identity
      cache_size: 100MB
      
  storage:
    enabled: true
    resources:
      cpu: 100
      memory: 2048
      io_weight: 900
    config:
      ipfs_api: http://localhost:5001
      filecoin_api: http://localhost:1234
      chunk_size: 1MB
```

### Production Deployment

For production environments, the subprocess architecture provides several deployment options:

1. **Single Host**: All services on one machine
   ```bash
   blackhole start --all --config=production.yaml
   ```

2. **Distributed**: Services across multiple hosts
   ```bash
   # Host 1: Core services
   blackhole start --services=identity,ledger --bind=0.0.0.0:9000
   
   # Host 2: Storage services
   blackhole start --services=storage,indexer --connect=host1:9000
   ```

3. **Kubernetes**: Container orchestration
   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: blackhole
   spec:
     containers:
     - name: blackhole
       image: blackhole:latest
       command: ["blackhole", "start", "--all"]
       securityContext:
         capabilities:
           add: ["SYS_RESOURCE"]
   ```

The subprocess architecture ensures that regardless of deployment method, services remain isolated and manageable while benefiting from the simplicity of a single binary distribution.

---

*This document serves as the primary reference for the Blackhole platform architecture and will be updated as the project evolves.*