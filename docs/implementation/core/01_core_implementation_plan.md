# Blackhole Core Implementation Plan

*Created: May 20, 2025*

This document outlines the implementation strategy for the Blackhole Core system, based on the architecture described in the core architecture documentation.

## Table of Contents

1. [Implementation Overview](#implementation-overview)
2. [Core Component Implementation](#core-component-implementation)
3. [Service Implementation Strategy](#service-implementation-strategy)
4. [Implementation Phases](#implementation-phases)
5. [Testing Strategy](#testing-strategy)
6. [Development Guidelines](#development-guidelines)
7. [Integration Points](#integration-points)
8. [Dependency Management](#dependency-management)

## Implementation Overview

The Blackhole platform will be implemented as a single binary that manages multiple independent service processes. This implementation plan focuses on the core functionality needed to orchestrate, manage, and facilitate communication between these services.

### Key Implementation Principles

1. **Process Isolation**: True service isolation through separate OS processes
2. **RPC Communication**: All inter-service communication via gRPC
3. **Resource Control**: OS-level resource management for each service
4. **Unified Binary**: Single executable that manages all services
5. **Security Boundaries**: Process-level security isolation

## Core Component Implementation

### 1. Process Orchestrator (`internal/core/orchestrator.go`)

The Process Orchestrator is the central component responsible for spawning, monitoring, and managing all service processes.

```go
// Core functionality
type Orchestrator struct {
    config      *Config
    processes   map[string]*ServiceProcess
    binaryPath  string
    serviceCfgs map[string]*ServiceConfig
    
    // Synchronization primitives
    mu          sync.RWMutex
    wg          sync.WaitGroup
    
    // Communication channels
    sigCh       chan os.Signal
    doneCh      chan struct{}
}

// ServiceProcess represents a running service process
type ServiceProcess struct {
    Name      string
    Command   *exec.Cmd
    PID       int
    Socket    string
    Port      int
    Status    ProcessStatus
    Started   time.Time
    Restarts  int
    Health    HealthStatus
}

// Process management methods
func (o *Orchestrator) Start() error
func (o *Orchestrator) Stop() error
func (o *Orchestrator) SpawnService(name string, config *ServiceConfig) error
func (o *Orchestrator) StopService(name string) error
func (o *Orchestrator) RestartService(name string) error
func (o *Orchestrator) supervise(name string)
func (o *Orchestrator) checkHealth(name string) HealthStatus
```

### 2. Configuration System (`internal/core/config.go`)

Manages configuration for the core system and all services.

```go
// Configuration
type Config struct {
    Global     GlobalConfig
    Services   map[string]*ServiceConfig
    Logging    LoggingConfig
    Security   SecurityConfig
    Monitoring MonitoringConfig
}

// Service-specific configuration
type ServiceConfig struct {
    Enabled     bool
    DataDir     string
    LogLevel    string
    UnixSocket  string
    Port        int
    Resources   ResourceConfig
    HealthCheck HealthCheckConfig
    Restart     RestartConfig
    Security    ServiceSecurityConfig
}

// Configuration methods
func LoadConfig(path string) (*Config, error)
func (c *Config) Save(path string) error
func (c *Config) GetServiceConfig(name string) (*ServiceConfig, error)
func (c *Config) ApplyEnvironmentOverrides() error
```

### 3. Service Mesh (`internal/mesh/`)

The Service Mesh provides the communication fabric between services, implementing routing, event distribution, and cross-cutting concerns.

#### 3.1 Router (`internal/mesh/router.go`)

Handles request routing between services, service discovery, and connection management.

```go
// Router manages service-to-service request routing
type Router struct {
    discovery    *ServiceDiscovery
    connections  map[string]*ConnectionPool
    loadBalancer LoadBalancer
    healthChecker HealthChecker
    mu           sync.RWMutex
}

// ServiceDiscovery handles endpoint registration and discovery
type ServiceDiscovery struct {
    services map[string][]Endpoint
    health   map[string]HealthStatus
    mu       sync.RWMutex
}

// Routing methods
func (r *Router) Route(ctx context.Context, service, method string, request interface{}) (interface{}, error)
func (r *Router) RegisterService(name string, endpoint Endpoint) error
func (r *Router) DiscoverService(name string) (Endpoint, error)
func (r *Router) UpdateHealth(name string, status HealthStatus) error
```

#### 3.2 EventBus (`internal/mesh/eventbus.go`)

Implements publish-subscribe pattern for asynchronous, event-driven communication.

```go
// EventBus provides publish-subscribe messaging
type EventBus struct {
    subscribers map[string][]chan Event
    persistence EventPersistence
    delivery    DeliveryOptions
    mu          sync.RWMutex
}

// Event represents a message sent through the eventbus
type Event struct {
    Type    string
    Source  string
    Data    interface{}
    Time    time.Time
    TraceID string
}

// EventBus methods
func (e *EventBus) Publish(ctx context.Context, event Event) error
func (e *EventBus) Subscribe(ctx context.Context, eventType string) (<-chan Event, error)
func (e *EventBus) Unsubscribe(eventType string, channel <-chan Event) error
func (e *EventBus) ListTopics() []string
```

#### 3.3 Middleware (`internal/mesh/middleware.go`)

Provides processing chains for cross-cutting concerns across service communications.

```go
// Middleware processes requests and responses
type Middleware interface {
    Process(ctx context.Context, req interface{}, next HandlerFunc) (interface{}, error)
}

// HandlerFunc defines a function that processes requests
type HandlerFunc func(ctx context.Context, req interface{}) (interface{}, error)

// MiddlewareChain manages a sequence of middleware
type MiddlewareChain struct {
    middleware []Middleware
}

// Middleware methods
func (c *MiddlewareChain) Add(m Middleware) *MiddlewareChain
func (c *MiddlewareChain) Execute(ctx context.Context, req interface{}, handler HandlerFunc) (interface{}, error)
```

### 4. RPC Service Manager (`internal/rpc/manager.go`)

Handles the RPC communication infrastructure, integrating with the mesh components.

```go
// RPC Manager
type RPCManager struct {
    orchestrator     *Orchestrator
    router           *mesh.Router
    clientCache      map[string]*grpc.ClientConn
    serviceEndpoints map[string]Endpoint
    mu               sync.RWMutex
}

// Endpoint represents a service endpoint
type Endpoint struct {
    Socket  string
    Address string
    Port    int
    Secure  bool
}

// RPC management methods
func (r *RPCManager) GetServiceClient(name string) (*grpc.ClientConn, error)
func (r *RPCManager) RegisterService(name string, endpoint Endpoint) error
func (r *RPCManager) HealthCheck(name string) error
func (r *RPCManager) CloseConnections() error
```

### 5. Resource Manager (`internal/core/resources.go`)

Implements OS-level resource controls for CPU, memory, and I/O limits for each service process.

```go
// Resource Manager
type ResourceManager struct {
    config   *Config
    services map[string]*ServiceResources
}

// ServiceResources defines resource limits for a service
type ServiceResources struct {
    CPUPercent   int
    MemoryMB     int
    IOWeight     int
    OpenFiles    int
    MaxProcesses int
}

// Resource management methods
func (r *ResourceManager) ApplyLimits(name string, cmd *exec.Cmd) error
func (r *ResourceManager) SetCgroupLimits(name string, pid int) error
func (r *ResourceManager) SetRlimits(name string, cmd *exec.Cmd) error
func (r *ResourceManager) MonitorResources() error
```

### 6. Security Manager (`internal/core/security.go`)

Implements security features including mTLS setup, credential management, and process sandboxing.

```go
// Security Manager
type SecurityManager struct {
    config        *Config
    certManager   *CertificateManager
    credentials   map[string]*ServiceCredentials
}

// Security-related types
type ServiceCredentials struct {
    CertPath    string
    KeyPath     string
    CACertPath  string
    Permissions []string
}

// Security methods
func (s *SecurityManager) SetupServiceSecurity(name string, cmd *exec.Cmd) error
func (s *SecurityManager) GenerateServiceCredentials(name string) (*ServiceCredentials, error)
func (s *SecurityManager) ValidateServiceAuth(name string, peerCert *x509.Certificate) error
func (s *SecurityManager) ApplySecurityPolicy(name string, cmd *exec.Cmd) error
```

### 7. Process Controller (`internal/core/process.go`)

Handles process control operations including signals, monitoring, and supervision.

```go
// Process Controller
type ProcessController struct {
    orchestrator *Orchestrator
    signals      map[string]chan os.Signal
}

// Process control methods
func (p *ProcessController) SendSignal(name string, sig os.Signal) error
func (p *ProcessController) WaitForExit(name string, timeout time.Duration) error
func (p *ProcessController) IsRunning(name string) bool
func (p *ProcessController) GetProcessStats(name string) (*ProcessStats, error)
```

### 8. Lifecycle Manager (`internal/core/lifecycle.go`)

Manages the lifecycle of all services, including startup sequence, coordination, and graceful shutdown.

```go
// Lifecycle Manager
type LifecycleManager struct {
    orchestrator  *Orchestrator
    dependencies  map[string][]string
    startOrder    []string
    stopOrder     []string
}

// Lifecycle methods
func (l *LifecycleManager) DetermineStartOrder() []string
func (l *LifecycleManager) StartAllServices() error
func (l *LifecycleManager) GracefulShutdown() error
func (l *LifecycleManager) WaitForServiceHealth(name string, timeout time.Duration) error
```

## Service Implementation Strategy

Each service will be implemented as a separate package in the `internal/services/` directory, following a standardized structure:

```
internal/services/<service>/
├── main.go           # Service entry point
├── go.mod            # Service module definition
├── service.go        # Service implementation
├── handlers.go       # gRPC handlers
├── config.go         # Configuration handling
└── <service>_test.go # Service tests
```

### Service Template

Each service will implement a common interface to standardize interactions:

```go
// Service interface
type Service interface {
    Start() error
    Stop() error
    Status() ServiceStatus
    HealthCheck() error
    GetConfig() *ServiceConfig
}

// Base service implementation
type BaseService struct {
    Name      string
    Config    *ServiceConfig
    GrpcServer *grpc.Server
    Logger    *zap.Logger
    Context   context.Context
    Cancel    context.CancelFunc
}

// Common service methods
func (s *BaseService) Start() error
func (s *BaseService) Stop() error
func (s *BaseService) Status() ServiceStatus
func (s *BaseService) HealthCheck() error
func (s *BaseService) GetConfig() *ServiceConfig
```

### Standard Service Entry Point

Each service will have a standardized main function:

```go
func main() {
    // Parse command line flags
    configPath := flag.String("config", "", "Path to service configuration")
    socketPath := flag.String("socket", "", "Unix socket path")
    port := flag.Int("port", 0, "TCP port")
    flag.Parse()
    
    // Initialize service
    service := NewService(&ServiceConfig{
        ConfigPath: *configPath,
        Socket: *socketPath,
        Port: *port,
    })
    
    // Setup signal handling
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    
    // Start service
    if err := service.Start(); err != nil {
        log.Fatalf("Failed to start service: %v", err)
    }
    
    // Wait for shutdown signal
    <-sigCh
    
    // Graceful shutdown
    if err := service.Stop(); err != nil {
        log.Printf("Error during shutdown: %v", err)
    }
}
```

## Implementation Phases

The core system will be implemented in phases to allow for incremental development and testing:

### Phase 1: Core Orchestrator

1. Implement basic process spawning and management
2. Setup configuration system
3. Implement signal handling and process supervision
4. Create basic CLI for starting/stopping services

### Phase 2: Communication Infrastructure

1. Implement gRPC server/client setup
2. Implement Service Mesh components:
   - Router with service discovery and request routing
   - EventBus with publish-subscribe capabilities
   - Middleware framework with basic middleware components
3. Implement connection management and pooling
4. Create comprehensive health check system
5. Implement service registration and discovery
6. Add initial cross-cutting middleware (logging, basic tracing)

### Phase 3: Security Implementation

1. Add process isolation
2. Implement mTLS for service communication
3. Setup credential management
4. Add security policies and sandboxing
5. Enhance mesh security:
   - Add authentication middleware
   - Implement authorization middleware
   - Add secure communication channels for EventBus
   - Implement middleware for security policy enforcement

### Phase 4: Resource Management

1. Implement CPU limits
2. Add memory controls
3. Setup I/O bandwidth management
4. Implement file descriptor limits
5. Add resource monitoring

### Phase 5: Service Integration

1. Implement service lifecycle manager
2. Add dependency resolution
3. Create coordinated startup/shutdown
4. Implement service health monitoring
5. Enhance mesh event system:
   - Add lifecycle events (service start/stop)
   - Implement system-wide event handling
   - Create dependency-aware event processing
   - Add event-driven coordination mechanisms

### Phase 6: Core Observability Infrastructure

1. Add internal logging infrastructure for core processes
2. Implement core process metrics collection (separate from Analytics/Telemetry services)
3. Create CLI tools for core system management
4. Add debugging and profiling capabilities for core components
5. Enhance mesh observability:
   - Add logging middleware for request/response tracking
   - Implement metrics middleware for performance monitoring
   - Create distributed tracing middleware
   - Add request/response debugging capabilities

> Note: This phase focuses solely on the operational health and observability of the core infrastructure itself, providing the foundation that the dedicated Analytics and Telemetry services will build upon. These core monitoring components are distinct from the business-level analytics and system-wide telemetry that will be implemented as separate services.

### Phase 7: Recovery and Resilience

1. **Failure Detection**
   - Process crash detection
   - Hang detection
   - Resource exhaustion detection

2. **Automatic Recovery**
   - Process restart with backoff
   - State recovery
   - Dependency chain recovery

3. **Circuit Breakers**
   - RPC circuit breakers
   - Fallback mechanisms
   - Degraded mode operation

4. **Mesh Resilience**
   - Add circuit breaker middleware
   - Implement retry middleware with exponential backoff
   - Create fault tolerance for event delivery
   - Add graceful degradation capabilities
   - Implement message persistence for critical events

### Phase 8: Advanced Features

1. **Hot Upgrades**
   - Rolling service updates
   - Versioned RPC support
   - Zero-downtime upgrades

2. **Dynamic Configuration**
   - Runtime configuration changes
   - Configuration propagation
   - Hot reloading

3. **Advanced Deployment**
   - Multi-host support
   - Kubernetes integration
   - Cloud provider support

## Testing Strategy

Testing will be a critical part of implementation to ensure reliability:

### Unit Tests

- Each component should have comprehensive unit tests
- Focus on testing individual functions and methods
- Use mocks for external dependencies
- Run as part of CI/CD pipeline

### Integration Tests

- Test service interactions
- Verify RPC communication
- Test resource management
- Validate security controls

### System Tests

- End-to-end testing of the entire system
- Test process management under load
- Verify correct behavior during crashes/failures
- Test resource limitations

### Benchmark Tests

- Measure performance of RPC calls
- Test resource usage under load
- Benchmark startup/shutdown times
- Measure memory overhead

## Development Guidelines

### Coding Standards

- Follow Go best practices
- Use standard Go project layout
- Implement error handling consistently
- Add proper documentation

### Dependency Management

- Use Go modules for dependency management
- Clearly document external dependencies
- Keep third-party dependencies to a minimum
- Version dependencies appropriately

### Documentation

- Add godoc comments to all exported types and functions
- Create architecture documentation
- Update implementation docs as code evolves
- Create examples for common tasks

## Integration Points

### CLI Integration

The core system will be controlled via a CLI interface:

```go
// CLI entrypoint
func main() {
    app := cli.NewApp()
    app.Name = "blackhole"
    app.Usage = "Distributed content sharing platform"
    
    app.Commands = []cli.Command{
        {
            Name: "start",
            Usage: "Start services",
            Flags: []cli.Flag{
                cli.BoolFlag{
                    Name: "all",
                    Usage: "Start all services",
                },
                cli.StringSliceFlag{
                    Name: "services",
                    Usage: "List of services to start",
                },
            },
            Action: startServices,
        },
        {
            Name: "stop",
            Usage: "Stop services",
            Action: stopServices,
        },
        {
            Name: "status",
            Usage: "Check service status",
            Action: statusServices,
        },
        {
            Name: "logs",
            Usage: "View service logs",
            Action: viewLogs,
        },
    }
    
    app.Run(os.Args)
}
```

### Configuration Integration

The core system will use a single configuration file that defines all service configurations:

```yaml
# Example configuration
global:
  data_dir: /var/lib/blackhole
  log_level: info
  socket_dir: /var/run/blackhole

services:
  identity:
    enabled: true
    data_dir: ${global.data_dir}/identity
    log_level: ${global.log_level}
    unix_socket: ${global.socket_dir}/identity.sock
    tcp_port: 50001
    resources:
      cpu_percent: 200
      memory_mb: 1024
      io_weight: 500
    
  storage:
    enabled: true
    data_dir: ${global.data_dir}/storage
    log_level: ${global.log_level}
    unix_socket: ${global.socket_dir}/storage.sock
    tcp_port: 50002
    resources:
      cpu_percent: 100
      memory_mb: 2048
      io_weight: 900
```

## Dependency Management

The project will use Go modules for dependency management:

### Workspace Structure

For multi-module development, we'll use Go workspaces:

```
go.work      # Workspace file
go.mod       # Main module
internal/services/identity/go.mod  # Identity service module
internal/services/storage/go.mod   # Storage service module
```

### Example go.work file:

```
go 1.19

use (
    .
    ./internal/services/identity
    ./internal/services/storage
    ./internal/services/node
    ./internal/services/ledger
    ./internal/services/indexer
    ./internal/services/social
    ./internal/services/analytics
    ./internal/services/telemetry
    ./internal/services/wallet
)
```

### Main Dependencies

- **gRPC/protobuf**: For RPC communication
- **libp2p**: For P2P networking
- **IPFS/Filecoin**: For content storage
- **zap/logrus**: For logging
- **cobra/cli**: For CLI interface
- **yaml/viper**: For configuration
- **prometheus**: For core process metrics (distinct from Analytics/Telemetry services)
- **testify**: For testing