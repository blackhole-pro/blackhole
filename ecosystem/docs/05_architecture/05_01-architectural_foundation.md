# Blackhole Foundation: Architectural Structure

## Overview

Blackhole Foundation is a distributed computing framework built on a **5-domain architecture** with clear **vertical layers**. This document defines the foundational structure that all development follows.

## Core Architectural Principles

### 1. **Clean Domain Separation**
Each domain has clear responsibilities, boundaries, and interfaces. No circular dependencies between domains.

### 2. **Vertical Layer Organization**  
Higher layers can depend on lower layers, but never the reverse. Each layer provides abstractions for the layer above.

### 3. **Independent Evolution**
Each domain can evolve independently as long as interfaces remain stable. Enables parallel development and testing.

### 4. **Clear Developer Onboarding**
Project structure immediately communicates architecture. New developers understand the system within minutes.

## Vertical Layers

### Layer 1: 🖥️ **Infrastructure Layer** (Not Our Concern)
- Operating system
- Network infrastructure  
- Hardware resources
- External services (payment processors, etc.)

### Layer 2: ⚙️ **Runtime Layer** (Foundation)
**Purpose**: Process orchestration, lifecycle management, system health

**Location**: `core/internal/runtime/`

**Responsibilities**:
- Process spawning and supervision
- Service lifecycle management (start, stop, restart)
- Health monitoring and recovery
- Configuration management
- Logging and diagnostics
- System-level resource management

**Key Components**:
```
core/internal/runtime/
├── orchestrator/     # Process orchestration and supervision
├── lifecycle/        # Service lifecycle management
├── health/          # Health monitoring and recovery
├── config/          # Configuration loading and management
├── diagnostics/     # Logging, metrics, tracing
└── dashboard/       # Runtime monitoring dashboard
```

**Public Interface**:
```go
type Runtime interface {
    StartService(name string, config ServiceConfig) error
    StopService(name string) error
    RestartService(name string) error
    GetServiceStatus(name string) ServiceStatus
    RegisterHealthCheck(name string, check HealthCheck) error
}
```

### Layer 3: 🔌 **Framework Layer** (5 Core Domains)

The heart of Blackhole Foundation. Each domain is independent but collaborates through well-defined interfaces.

#### Domain 1: **Plugin Management**
**Purpose**: Plugin loading, execution, isolation, hot-swapping

**Location**: `core/internal/framework/plugins/`

**Responsibilities**:
- Plugin discovery and loading
- Plugin lifecycle management
- Plugin isolation and sandboxing
- Hot loading and unloading
- Plugin state management
- Plugin dependency resolution

**Key Components**:
```
core/internal/framework/plugins/
├── registry/        # Plugin discovery and registration
├── loader/          # Plugin loading and unloading
├── executor/        # Plugin execution and isolation
├── lifecycle/       # Plugin lifecycle management
└── state/          # Plugin state management and migration
```

**Public Interface**:
```go
type PluginManager interface {
    LoadPlugin(spec PluginSpec) error
    UnloadPlugin(name string) error
    ExecutePlugin(name string, request PluginRequest) (PluginResponse, error)
    ListPlugins() []PluginInfo
    HotSwapPlugin(name string, newVersion string) error
}
```

#### Domain 2: **Mesh Networking**
**Purpose**: Communication, discovery, coordination between nodes

**Location**: `core/internal/framework/mesh/`

**Responsibilities**:
- Node discovery and registration
- Inter-node communication
- Message routing and load balancing
- Network topology management
- Connection pooling and health
- Protocol negotiation

**Key Components**:
```
core/internal/framework/mesh/
├── discovery/       # Node discovery and registration
├── routing/         # Message routing and load balancing
├── transport/       # Transport protocols (gRPC, P2P, etc.)
├── topology/        # Network topology management
└── security/        # Mesh security and authentication
```

**Public Interface**:
```go
type MeshNetwork interface {
    RegisterNode(nodeInfo NodeInfo) error
    DiscoverNodes(criteria DiscoveryCriteria) ([]NodeInfo, error)
    SendMessage(target NodeID, message Message) error
    BroadcastMessage(message Message) error
    SetupRoute(from NodeID, to NodeID) (Route, error)
}
```

#### Domain 3: **Resource Management**
**Purpose**: Resource allocation, monitoring, optimization

**Location**: `core/internal/framework/resources/`

**Responsibilities**:
- Resource discovery and inventory
- Resource allocation and scheduling
- Usage monitoring and optimization
- Cost tracking and optimization
- Performance monitoring
- Capacity planning

**Key Components**:
```
core/internal/framework/resources/
├── inventory/       # Resource discovery and inventory
├── scheduler/       # Resource allocation and scheduling
├── monitor/         # Usage monitoring and metrics
├── optimizer/       # Performance and cost optimization
└── planner/        # Capacity planning and forecasting
```

**Public Interface**:
```go
type ResourceManager interface {
    AllocateResources(requirements ResourceRequirements) (Allocation, error)
    ReleaseResources(allocation Allocation) error
    GetResourceUsage(nodeID NodeID) (ResourceUsage, error)
    OptimizeAllocation(constraints OptimizationConstraints) error
    GetCapacityPlan(timeHorizon time.Duration) (CapacityPlan, error)
}
```

#### Domain 4: **Economic System**
**Purpose**: Payments, revenue distribution, usage tracking

**Location**: `core/internal/framework/economics/`

**Responsibilities**:
- Usage tracking and metering
- Payment processing
- Revenue distribution
- Cost calculation and optimization
- Billing and invoicing
- Economic analytics

**Key Components**:
```
core/internal/framework/economics/
├── metering/        # Usage tracking and measurement
├── payments/        # Payment processing and billing
├── distribution/    # Revenue distribution to participants
├── pricing/         # Dynamic pricing and cost calculation
└── analytics/       # Economic analytics and reporting
```

**Public Interface**:
```go
type EconomicSystem interface {
    TrackUsage(userID UserID, resource ResourceType, amount uint64) error
    ProcessPayment(payment Payment) error
    DistributeRevenue(revenue Revenue, participants []Participant) error
    CalculateCost(usage Usage) (Cost, error)
    GetBill(userID UserID, period TimePeriod) (Bill, error)
}
```

#### Domain 5: **Ecosystem Domain**
**Purpose**: Community, marketplace, documentation, governance

**Location**: `ecosystem/` (non-code) and `core/pkg/` (code)

**Responsibilities**:
- Community management and events
- Plugin marketplace operations
- Documentation and training
- Governance and certification
- Enterprise support
- Developer tools (SDK, templates)

**Key Components**:
```
ecosystem/                    # Non-code components
├── marketplace/             # Plugin marketplace operations
├── docs/                    # Documentation
├── community/               # Community programs
├── governance/              # Board and policies
├── events/                  # Conferences and meetups
├── certification/           # Certification programs
├── training/                # Education programs
├── partners/                # Partner network
├── jobs/                    # Career opportunities
└── enterprise/              # Enterprise solutions

core/pkg/                    # Code components
├── sdk/                     # Plugin development SDK
├── tools/                   # Development tools
└── templates/               # Plugin templates
```

**Public Interface**:
```go
type EcosystemManager interface {
    // SDK and development tools (from core/pkg/)
    CreatePluginProject(template PluginTemplate) (Project, error)
    BuildPlugin(project Project) (PluginPackage, error)
    
    // Marketplace operations (from ecosystem/)
    PublishPlugin(package PluginPackage) error
    SearchPlugins(criteria SearchCriteria) ([]PluginInfo, error)
    
    // Documentation (from ecosystem/docs/)
    GetDocumentation(topic string) (Documentation, error)
}
```

### Layer 4: 🛠️ **Platform Layer** (Developer Interface)
**Purpose**: Developer-facing tools and interfaces

**Location**: `core/pkg/`, `core/cmd/`

**Responsibilities**:
- Public APIs and SDKs
- CLI tools and utilities
- Plugin development kit
- Documentation and examples
- Testing frameworks
- Development environment setup

### Layer 5: 🎯 **Application Layer** (User Interface)
**Purpose**: End-user applications built on the framework

**Location**: `applications/`, `core/examples/`

**Responsibilities**:
- User-facing applications
- Web interfaces and dashboards
- Mobile applications
- Desktop applications
- Integration examples
- Reference implementations

## Domain Interaction Patterns

### Core Interaction Rules

1. **Runtime Layer** provides foundation services to all framework domains
2. **Framework Domains** collaborate as peers through well-defined interfaces
3. **No circular dependencies** between domains at the same layer
4. **Higher layers** can use lower layers, never the reverse

### Key Collaboration Patterns

#### Plugin Execution Flow
```
User Request → Platform Layer → Plugin Manager → Mesh Network → Resource Manager → Economic System
                    ↓              ↓              ↓              ↓               ↓
               Developer API → Load Plugin → Route Request → Allocate Resources → Track Usage
```

#### Cross-Domain Communication
```go
// Domains communicate through well-defined interfaces
type FrameworkBus interface {
    GetPluginManager() PluginManager
    GetMeshNetwork() MeshNetwork  
    GetResourceManager() ResourceManager
    GetEconomicSystem() EconomicSystem
    GetEcosystemManager() EcosystemManager
}
```

## Implementation Strategy

### Phase 1: Foundation (Current → 4 weeks)
1. **Restructure codebase** to match this architecture
2. **Implement Runtime Layer** (enhance current orchestrator)
3. **Define domain interfaces** for Framework Layer
4. **Create basic project structure** that developers can understand

### Phase 2: Core Domains (Weeks 5-12)  
1. **Implement Plugin Management** domain
2. **Enhance Mesh Networking** domain (build on current P2P)
3. **Create Resource Management** domain
4. **Build Economic System** domain
5. **Establish Ecosystem** domain

### Phase 3: Integration (Weeks 13-16)
1. **Connect all domains** through FrameworkBus
2. **Implement cross-domain workflows**
3. **Create developer onboarding experience**
4. **Build first working applications**

### Phase 4: Polish (Weeks 17-20)
1. **Comprehensive testing** across all domains
2. **Performance optimization** and monitoring
3. **Documentation completion**
4. **Production readiness**

## Directory Structure

```
blackhole/
├── README.md                          # Clear project overview
├── PROJECT.md                         # Project details and roadmap
├── CLAUDE.md                          # AI assistant context
│
├── core/                              # All technical implementation
│   ├── cmd/                           # CLI tools and utilities
│   │   ├── blackhole/                 # Main CLI tool
│   │   └── tools/                     # Other command-line tools
│   │
│   ├── internal/                      # Private implementation
│   │   ├── runtime/                   # Layer 2: Runtime Layer
│   │   │   ├── orchestrator/
│   │   │   ├── lifecycle/
│   │   │   ├── health/
│   │   │   ├── config/
│   │   │   ├── diagnostics/
│   │   │   └── dashboard/             # Runtime monitoring dashboard
│   │   │
│   │   ├── framework/                 # Layer 3: Framework Layer (4 core domains)
│   │   │   ├── plugins/               # Domain 1: Plugin Management
│   │   │   │   ├── registry/
│   │   │   │   ├── loader/
│   │   │   │   ├── executor/
│   │   │   │   ├── lifecycle/
│   │   │   │   └── state/
│   │   │   │
│   │   │   ├── mesh/                  # Domain 2: Mesh Networking
│   │   │   │   ├── discovery/
│   │   │   │   ├── routing/
│   │   │   │   ├── transport/
│   │   │   │   ├── topology/
│   │   │   │   └── security/
│   │   │   │
│   │   │   ├── resources/             # Domain 3: Resource Management
│   │   │   │   ├── inventory/
│   │   │   │   ├── scheduler/
│   │   │   │   ├── monitor/
│   │   │   │   ├── optimizer/
│   │   │   │   └── planner/
│   │   │   │
│   │   │   └── economics/             # Domain 4: Economic System
│   │   │       ├── metering/
│   │   │       ├── payments/
│   │   │       ├── distribution/
│   │   │       ├── pricing/
│   │   │       └── analytics/
│   │   │
│   │   ├── services/                  # Service implementations
│   │   │   ├── identity/              # Identity service
│   │   │   ├── node/                  # Node service & P2P
│   │   │   ├── ledger/                # Ledger service
│   │   │   ├── indexer/               # Indexer service
│   │   │   ├── social/                # Social service
│   │   │   ├── analytics/             # Analytics service
│   │   │   ├── telemetry/             # Telemetry service
│   │   │   └── wallet/                # Wallet service
│   │   │
│   │   └── rpc/                       # RPC definitions
│   │       ├── proto/                 # Protocol definitions
│   │       └── gen/                   # Generated code
│   │
│   ├── pkg/                           # Layer 4: Platform Layer (Public APIs)
│   │   ├── api/                       # Public API clients
│   │   ├── sdk/                       # Plugin development SDK
│   │   ├── tools/                     # Development tools
│   │   ├── templates/                 # Plugin templates
│   │   └── types/                     # Shared type definitions
│   │
│   ├── examples/                      # Reference implementations
│   │   ├── plugins/                   # Example plugins
│   │   ├── applications/              # Example applications
│   │   └── integrations/              # Integration examples
│   │
│   ├── test/                          # Testing
│   │   ├── unit/                      # Unit tests (mirrors internal/)
│   │   ├── integration/               # Integration tests
│   │   └── e2e/                       # End-to-end tests
│   │
│   ├── configs/                       # Configuration files
│   ├── scripts/                       # Build and utility scripts
│   └── bin/                           # Build artifacts
│
├── ecosystem/                         # Domain 5: Ecosystem (non-code)
│   ├── docs/                          # All documentation
│   │   ├── 01-ARCHITECTURE_QUICK_START.md
│   │   ├── 02-blackhole_foundation.md
│   │   ├── 03-ORGANIZATION.md
│   │   ├── 04_domains/                # Domain documentation
│   │   ├── 05_architecture/           # Architecture specs
│   │   ├── 06_guides/                 # Developer guides
│   │   ├── 07_reference/              # API reference
│   │   └── 08_strategy/               # Strategy docs
│   │
│   ├── marketplace/                   # Plugin marketplace
│   ├── community/                     # Community programs
│   ├── governance/                    # Board and policies
│   ├── events/                        # Conferences and meetups
│   ├── certification/                 # Certification programs
│   ├── training/                      # Education programs
│   ├── partners/                      # Partner network
│   ├── jobs/                          # Career opportunities
│   └── enterprise/                    # Enterprise solutions
│
├── applications/                      # Layer 5: Application Layer
│   ├── file-storage/                  # Personal file storage app
│   ├── media-streaming/               # Media streaming app
│   ├── office-suite/                  # Office productivity app
│   └── social-network/                # Social networking app
│
├── go.mod                             # Go module definition
├── go.sum                             # Go module checksums
├── go.work                            # Go workspace
└── Makefile                           # Build automation
```

## Developer Onboarding Path

### New Developer Journey

1. **Read README.md** → Understand what Blackhole Foundation is
2. **Read this document** → Understand the 5-domain structure  
3. **Choose a domain** → Pick Runtime, Plugins, Mesh, Resources, Economics, or Ecosystem
4. **Read domain docs** → `ecosystem/docs/04_domains/<domain>/README.md`
5. **Look at examples** → `core/examples/<domain>/`
6. **Run tutorials** → `ecosystem/docs/06_guides/`
7. **Start contributing** → Clear separation makes it easy

### Domain-Specific Contribution

```
Want to work on plugins? → core/internal/framework/plugins/
Want to work on networking? → core/internal/framework/mesh/
Want to work on economics? → core/internal/framework/economics/
Want to build apps? → applications/
Want to improve developer experience? → core/pkg/sdk/ or ecosystem/
Want to work on runtime? → core/internal/runtime/
Want to create services? → core/internal/services/
```

## Success Criteria

### Architectural Clarity
- [ ] New developers understand the system within 15 minutes
- [ ] Each domain has clear boundaries and responsibilities  
- [ ] No circular dependencies between domains
- [ ] Clean interfaces enable independent development

### Development Experience
- [ ] Domains can be developed and tested independently
- [ ] Clear contribution guidelines for each domain
- [ ] Simple setup and development workflow
- [ ] Comprehensive documentation and examples

### Technical Foundation
- [ ] Runtime layer provides stable foundation
- [ ] Framework domains collaborate effectively
- [ ] Platform layer offers excellent developer experience
- [ ] Applications demonstrate real value

This architectural foundation will make Blackhole Foundation immediately understandable to any developer, while providing the structure needed to build the economic revolution we've designed.

---

*Next: Implement this structure in the actual codebase to replace the current messy organization.*