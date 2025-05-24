# Node Core Architecture

## Overview

The Blackhole node is designed as a single binary that orchestrates multiple service subprocesses, providing P2P networking, service management, and foundational capabilities. Services run as independent processes for true isolation while maintaining the simplicity of single binary deployment. All inter-process communication via gRPC happens through RPC (gRPC).

## Core Design Philosophy

The node architecture follows these principles:
- **Single Binary Distribution**: One executable that spawns service subprocesses
- **Process-Based Isolation**: Each service runs in its own process
- **RPC Communication**: All services communicate via gRPC (local Unix sockets or network)
- **Progressive Enhancement**: Start simple, scale when needed
- **Operator First**: Simple deployment with sophisticated architecture

## Architecture Layers

The node operates as an orchestrator managing separate service processes:

### 1. Foundation Layer (Orchestrator Process)

The orchestrator that manages all service subprocesses:

```go
type NodeOrchestrator struct {
    // Core runtime
    runtime    *Runtime
    config     *Configuration
    logger     *Logger
    metrics    *MetricsRegistry
    
    // Service process management
    services   *ServiceManager
    supervisor *ProcessSupervisor
    
    // RPC infrastructure
    rpcClients map[string]*grpc.ClientConn
    discovery  *ServiceDiscovery
    
    // Resource management
    resources  *ResourceMonitor  // Monitor subprocess resources
    health     *HealthChecker    // Service health monitoring
}
```

**Components**:
- Service subprocess lifecycle management
- Configuration distribution to services
- Centralized logging aggregation
- Metrics collection from all services
- Process monitoring and restart policies

### 2. Service Subprocess Layer

Each service runs as an independent subprocess:

```go
// Each service is a separate process
type ServiceProcess struct {
    Name       string
    Binary     string            // Same binary with different args
    Process    *exec.Cmd
    Port       int               // gRPC port
    UnixSocket string            // Unix socket for local RPC
    
    // RPC server (in subprocess)
    rpcServer  *grpc.Server
    
    // Health and monitoring
    health     *HealthEndpoint
    metrics    *MetricsExporter
}

// Main binary can run as service
func main() {
    if len(os.Args) > 1 && os.Args[1] == "service" {
        serviceName := os.Args[2]
        runService(serviceName)  // Run as subprocess
    } else {
        runOrchestrator()        // Run as main orchestrator
    }
}
```

**Features**:
- Process isolation for each service
- Independent resource limits per process
- Crash isolation between services
- Independent scaling and updates
- Local RPC via Unix sockets
- Network RPC for remote services

### 3. RPC Communication Layer

All services communicate through gRPC:

```go
type RPCInfrastructure struct {
    // Service discovery
    discovery   *ServiceDiscovery
    registry    *ServiceRegistry
    
    // Client management
    clients     map[string]*grpc.ClientConn
    balancer    *LoadBalancer
    
    // Connection options
    localOpts   []grpc.DialOption  // Unix socket options
    remoteOpts  []grpc.DialOption  // TCP options with TLS
    
    // Reliability
    circuitBreaker *CircuitBreaker
    retryPolicy    *RetryPolicy
    timeout        *TimeoutManager
}

// Service client interface
func (r *RPCInfrastructure) GetServiceClient(name string) (ServiceClient, error) {
    // Try local subprocess first
    if r.discovery.HasLocalService(name) {
        return r.connectLocal(name)
    }
    
    // Fallback to remote node
    return r.connectRemote(name)
}
```

**Capabilities**:
- Unified RPC for local and remote services
- Automatic service discovery
- Load balancing across instances
- Circuit breakers and retries
- Unix sockets for local performance
- TLS for remote security

### 4. Core Services Layer

Services run as separate subprocesses:

```go
// Service configuration
type ServiceConfig struct {
    Name       string
    Enabled    bool
    Port       int
    UnixSocket string
    Resources  ResourceLimits
}

// Available services
var CoreServices = map[string]ServiceConfig{
    "identity": {
        Name:       "identity",
        Port:       8081,
        UnixSocket: "/tmp/blackhole-identity.sock",
        Resources:  ResourceLimits{Memory: "512MB", CPU: 0.5},
    },
    "storage": {
        Name:       "storage", 
        Port:       8082,
        UnixSocket: "/tmp/blackhole-storage.sock",
        Resources:  ResourceLimits{Memory: "2GB", CPU: 1.0},
    },
    "ledger": {
        Name:       "ledger",
        Port:       8083,
        UnixSocket: "/tmp/blackhole-ledger.sock",
        Resources:  ResourceLimits{Memory: "1GB", CPU: 0.5},
    },
    // ... other services
}
```

All services communicate through RPC:
- Complete process isolation
- Independent deployment and updates
- Resource limits enforced by OS
- Crash isolation between services
- Language-agnostic service implementation

### 5. API Gateway Layer

Unified external interface:

```go
type APIGateway struct {
    // Protocol handlers
    http       *HTTPServer
    grpc       *GRPCServer
    websocket  *WSServer
    graphql    *GraphQLHandler
    
    // Gateway features
    auth       *Authenticator
    rateLimit  *RateLimiter
    cache      *APICache
    
    // Service proxy
    proxy      *ServiceProxy
}
```

**Features**:
- Single endpoint for all services
- Protocol translation
- Authentication and authorization
- Rate limiting and caching
- API versioning

## Core Components

### Service Manager

Manages subprocess lifecycle and communication:

```go
type ServiceManager struct {
    // Binary path (self)
    binary      string
    
    // Running services
    services    map[string]*ServiceProcess
    
    // Process supervision
    supervisor  *ProcessSupervisor
    
    // RPC clients
    rpcClients  map[string]*grpc.ClientConn
    
    // Configuration
    config      *NodeConfig
}

func (m *ServiceManager) StartService(name string) error {
    config := m.config.Services[name]
    if !config.Enabled {
        return nil
    }
    
    // Start subprocess
    cmd := exec.Command(m.binary, "service", name,
        "--port", strconv.Itoa(config.Port),
        "--socket", config.UnixSocket,
    )
    
    // Set resource limits
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Rlimits: m.getResourceLimits(config.Resources),
    }
    
    if err := cmd.Start(); err != nil {
        return err
    }
    
    // Wait for RPC server to be ready
    client, err := m.waitForService(name, config)
    if err != nil {
        cmd.Process.Kill()
        return err
    }
    
    m.services[name] = &ServiceProcess{
        Name:    name,
        Process: cmd,
        Config:  config,
    }
    
    m.rpcClients[name] = client
    return nil
}
```

### RPC Communication

Handles all inter-process communication via gRPC:

```go
type RPCManager struct {
    // Service connections
    clients    map[string]*grpc.ClientConn
    discovery  *ServiceDiscovery
    
    // Connection management
    connOpts   struct {
        local  []grpc.DialOption  // Unix socket options
        remote []grpc.DialOption  // TCP with TLS
    }
    
    // Reliability
    circuit    *CircuitBreaker
    retry      *RetryPolicy
    balancer   *LoadBalancer
    
    // Observability
    tracer     *DistributedTracer
    metrics    *MetricsCollector
}

// Service communication
func (m *RPCManager) Call(ctx context.Context, service string, req interface{}) (interface{}, error) {
    // Get client connection
    client, err := m.getClient(service)
    if err != nil {
        return nil, err
    }
    
    // Apply circuit breaker
    if !m.circuit.Allow(service) {
        return nil, ErrCircuitOpen
    }
    
    // Create typed client
    serviceClient := m.createServiceClient(service, client)
    
    // Execute with retry
    return m.retry.Execute(func() (interface{}, error) {
        return serviceClient.Call(ctx, req)
    })
}

// Connect to service (local or remote)
func (m *RPCManager) getClient(service string) (*grpc.ClientConn, error) {
    // Check cache
    if client, ok := m.clients[service]; ok {
        return client, nil
    }
    
    // Try local subprocess first
    if location := m.discovery.GetLocal(service); location != nil {
        conn, err := grpc.Dial("unix://"+location.UnixSocket, m.connOpts.local...)
        if err == nil {
            m.clients[service] = conn
            return conn, nil
        }
    }
    
    // Try remote node
    if locations := m.discovery.GetRemote(service); len(locations) > 0 {
        location := m.balancer.Select(locations)
        conn, err := grpc.Dial(location.Address, m.connOpts.remote...)
        if err == nil {
            m.clients[service] = conn
            return conn, nil
        }
    }
    
    return nil, fmt.Errorf("service %s not available", service)
}
```

### Process Supervisor

Monitors and manages service subprocesses:

```go
type ProcessSupervisor struct {
    services     map[string]*MonitoredService
    policies     map[string]RestartPolicy
    
    // Health monitoring
    healthChecks map[string]*HealthChecker
    
    // Resource monitoring
    resources    *ResourceMonitor
}

type MonitoredService struct {
    *ServiceProcess
    Restarts    int
    LastRestart time.Time
    CrashCount  int
    Status      ServiceStatus
}

func (s *ProcessSupervisor) Monitor(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            s.checkServices()
        case <-ctx.Done():
            return
        }
    }
}

func (s *ProcessSupervisor) checkServices() {
    for name, service := range s.services {
        // Check if process is still running
        if service.Process.ProcessState != nil {
            // Process exited
            s.handleProcessExit(name, service)
        }
        
        // Health check via RPC
        if err := s.healthChecks[name].Check(); err != nil {
            service.Status = Unhealthy
            s.handleUnhealthyService(name, service)
        }
    }
}

func (s *ProcessSupervisor) handleProcessExit(name string, service *MonitoredService) {
    service.CrashCount++
    
    policy := s.policies[name]
    if policy.ShouldRestart(service.Restarts, service.CrashCount) {
        backoff := policy.GetBackoffDuration(service.Restarts)
        
        log.Printf("Service %s crashed, restarting in %v", name, backoff)
        
        time.AfterFunc(backoff, func() {
            if err := s.restartService(name); err != nil {
                log.Printf("Failed to restart service %s: %v", name, err)
            }
        })
    }
}
```

### Adaptive Resource Manager

Provides self-tuning resource allocation based on historical usage:

```go
type AdaptiveResourceManager struct {
    // Rolling metrics per service
    usageMetrics    map[string]*RollingMetrics
    
    // Dynamic guarantees based on historical usage
    adaptiveQuotas  map[string]*AdaptiveQuota
    
    // Shared burst pool for peaks
    burstPool       *BurstResourcePool
    
    // Service tiers for prioritization
    serviceTiers    map[string]ServiceTier
    
    // Monitoring and adjustment
    monitor         *ResourceMonitor
    adjuster        *QuotaAdjuster
}

// Allocate resources with adaptive guarantees and burst capability
func (m *AdaptiveResourceManager) Allocate(service string, request ResourceRequest) (*Allocation, error) {
    quota := m.adaptiveQuotas[service]
    
    // Phase 1: Try guaranteed allocation
    if request.Size <= quota.currentGuarantee {
        return quota.AllocateGuaranteed(request)
    }
    
    // Phase 2: Try burst allocation
    burstNeeded := request.Size - quota.currentGuarantee
    if m.burstPool.Available() >= burstNeeded {
        return m.allocateWithBurst(service, request)
    }
    
    // Phase 3: Preemption for higher tiers
    if m.canPreempt(service) {
        return m.preemptAndAllocate(service, request)
    }
    
    return nil, ErrInsufficientResources
}

// Continuous monitoring and adjustment
func (m *AdaptiveResourceManager) Monitor(ctx context.Context) {
    ticker := time.NewTicker(m.config.adjustInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            m.adjustQuotas()
        case <-ctx.Done():
            return
        }
    }
}
```

## Service Isolation

Services achieve complete isolation through process boundaries:

### Process-Based Resource Isolation

```go
type ProcessResourceManager struct {
    services map[string]*ServiceProcess
    monitor  *ResourceMonitor
}

// Set resource limits at process level
func (m *ProcessResourceManager) ApplyLimits(cmd *exec.Cmd, limits ResourceLimits) {
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Rlimits: []syscall.Rlimit{
            {
                Type: syscall.RLIMIT_AS,     // Memory limit
                Cur:  limits.Memory,
                Max:  limits.Memory,
            },
            {
                Type: syscall.RLIMIT_CPU,    // CPU time limit
                Cur:  limits.CPUSeconds,
                Max:  limits.CPUSeconds,
            },
            {
                Type: syscall.RLIMIT_NOFILE, // File descriptor limit
                Cur:  limits.MaxFiles,
                Max:  limits.MaxFiles,
            },
        },
    }
    
    // CPU affinity
    if limits.CPUCores > 0 {
        cmd.Env = append(cmd.Env, 
            fmt.Sprintf("GOMAXPROCS=%d", limits.CPUCores))
    }
}

// Monitor resource usage
func (m *ProcessResourceManager) MonitorUsage(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            for name, service := range m.services {
                stats := m.getProcessStats(service.Process.Pid)
                m.monitor.RecordUsage(name, stats)
            }
        case <-ctx.Done():
            return
        }
    }
}
```

### Crash Isolation

```go
type CrashIsolation struct {
    supervisor *ProcessSupervisor
}

// Process crashes don't affect other services
func (c *CrashIsolation) HandleCrash(service string) {
    log.Printf("Service %s crashed, other services unaffected", service)
    
    // Other services continue running
    // Orchestrator decides restart policy
    c.supervisor.HandleServiceCrash(service)
}
```

### Network Isolation

```go
type NetworkIsolation struct {
    services map[string]NetworkConfig
}

type NetworkConfig struct {
    UnixSocket string  // Local communication
    TCPPort    int     // Network communication
    TLSConfig  *tls.Config
}

// Each service has isolated network configuration
func (n *NetworkIsolation) ConfigureService(name string) NetworkConfig {
    return NetworkConfig{
        UnixSocket: fmt.Sprintf("/tmp/blackhole-%s.sock", name),
        TCPPort:    n.allocatePort(name),
        TLSConfig:  n.generateTLSConfig(name),
    }
}
```

## Node Configuration

Single configuration file controls all subprocesses:

```yaml
# blackhole.yaml
node:
  name: "my-blackhole-node"
  dataDir: "/var/lib/blackhole"
  
# Process management
process_management:
  supervisor:
    check_interval: 5s
    restart_policy: exponential_backoff
    max_restarts: 5
    restart_window: 5m
  
  health_checks:
    interval: 10s
    timeout: 5s
    unhealthy_threshold: 3
  
network:
  listenAddrs:
    - "/ip4/0.0.0.0/tcp/4001"
    - "/ip4/0.0.0.0/quic/4001"
  bootstrapPeers:
    - "/dnsaddr/bootstrap.blackhole.network"

services:
  identity:
    enabled: true
    port: 8081
    unix_socket: "/tmp/blackhole-identity.sock"
    restart_policy: always
    resources:
      memory: "512MB"
      cpu_seconds: 3600
      cpu_cores: 1
      max_files: 1024
  
  storage:
    enabled: true
    port: 8082
    unix_socket: "/tmp/blackhole-storage.sock"
    restart_policy: always
    resources:
      memory: "2GB"
      cpu_seconds: 7200
      cpu_cores: 2
      max_files: 4096
      disk: "100GB"
  
  ledger:
    enabled: true
    port: 8083
    unix_socket: "/tmp/blackhole-ledger.sock"
    restart_policy: on_failure
    resources:
      memory: "1GB"
      cpu_seconds: 3600
      cpu_cores: 1
      max_files: 2048
  
  analytics:
    enabled: false  # Optional service
    port: 8084
    unix_socket: "/tmp/blackhole-analytics.sock"
    restart_policy: on_failure
    resources:
      memory: "1GB"
      cpu_seconds: 3600
      cpu_cores: 1

api:
  httpPort: 8080
  grpcPort: 9090
  wsPort: 8081
  
  rateLimit:
    enabled: true
    requestsPerMinute: 1000

monitoring:
  metricsPort: 9091
  tracingEndpoint: "http://jaeger:14268"
  logLevel: "info"
  log_aggregation:
    enabled: true
    buffer_size: 10000
```

## Deployment Model

### Single Binary with Subprocesses

```bash
# Deploy single binary that spawns services
./blackhole-node --config blackhole.yaml

# Creates multiple processes:
# PID   COMMAND
# 1001  blackhole-node                           # Orchestrator
# 1002  blackhole-node service identity         # Identity subprocess  
# 1003  blackhole-node service storage          # Storage subprocess
# 1004  blackhole-node service ledger           # Ledger subprocess
# 1005  blackhole-node service social           # Social subprocess

# With systemd
[Unit]
Description=Blackhole Node Orchestrator
After=network.target

[Service]
Type=forking
User=blackhole
ExecStart=/usr/bin/blackhole-node --config /etc/blackhole/config.yaml
ExecStop=/usr/bin/blackhole-node stop
Restart=always
KillMode=control-group  # Kill all subprocesses on stop

[Install]
WantedBy=multi-user.target
```

### Docker Container

```dockerfile
FROM alpine:latest
COPY blackhole-node /usr/bin/
EXPOSE 4001 8080 9090 9091
CMD ["blackhole-node"]
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: blackhole-node
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: blackhole
        image: blackhole/node:latest
        ports:
        - containerPort: 4001
        - containerPort: 8080
        - containerPort: 9091
```

## Scaling Strategy

### Service-Level Scaling

Each service can be independently managed:

```yaml
# Scale individual services within node
services:
  storage:
    enabled: true
    instances: 3  # Run multiple storage processes
    resources:
      memory: "2GB"
      cpu_cores: 2
      
  analytics:
    enabled: false  # Disable on this node
```

### Horizontal Node Scaling

Deploy multiple nodes with different service configurations:

```yaml
# Node A - Storage focused
services:
  storage: { enabled: true, instances: 3 }
  identity: { enabled: true }
  analytics: { enabled: false }

# Node B - Compute focused  
services:
  storage: { enabled: false }
  identity: { enabled: true }
  analytics: { enabled: true, instances: 2 }
```

### Service Updates

Individual service updates without full node restart:

```go
type ServiceUpdater struct {
    manager *ServiceManager
}

// Update single service
func (u *ServiceUpdater) UpdateService(name string, newVersion string) error {
    // Download new binary version
    if err := u.downloadVersion(name, newVersion); err != nil {
        return err
    }
    
    // Graceful restart
    if err := u.manager.GracefulRestart(name); err != nil {
        // Rollback on failure
        return u.manager.Rollback(name)
    }
    
    return nil
}

// Rolling update across nodes
func (u *ServiceUpdater) RollingUpdate(service string, version string) error {
    nodes := u.getNodesRunningService(service)
    
    for _, node := range nodes {
        if err := node.UpdateService(service, version); err != nil {
            return fmt.Errorf("update failed on node %s: %w", node.ID, err)
        }
        
        // Wait for health checks
        u.waitForHealthy(node, service)
    }
    
    return nil
}
```

## Monitoring and Operations

### Health Monitoring

```go
// Orchestrator health endpoint shows all services
GET /health

{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "24h35m",
  "orchestrator": {
    "pid": 1001,
    "memory": "125MB",
    "cpu": "2%"
  },
  "services": {
    "identity": {
      "status": "healthy",
      "pid": 1002,
      "restarts": 0,
      "memory": "245MB/512MB",
      "cpu": "5%",
      "last_health_check": "2024-01-15T10:30:00Z"
    },
    "storage": {
      "status": "healthy", 
      "pid": 1003,
      "restarts": 1,
      "memory": "1.2GB/2GB",
      "cpu": "15%",
      "last_health_check": "2024-01-15T10:30:00Z"
    },
    "analytics": {
      "status": "stopped",
      "reason": "disabled"
    }
  }
}
```

### Process Monitoring

```go
type ProcessMonitor struct {
    services map[string]*ProcessStats
}

type ProcessStats struct {
    PID        int
    Memory     int64
    CPU        float64
    Threads    int
    OpenFiles  int
    Uptime     time.Duration
    RestartCount int
}

// Per-service health endpoints
func (s *ServiceProcess) HealthEndpoint() {
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(HealthStatus{
            Service: s.Name,
            Status:  s.Health,
            Version: s.Version,
            Uptime:  time.Since(s.StartTime),
        })
    })
}
```

### Log Aggregation

```go
type LogAggregator struct {
    services map[string]io.Reader
    output   io.Writer
    parser   *LogParser
}

// Aggregate logs from all services
func (a *LogAggregator) Start(ctx context.Context) {
    for name, reader := range a.services {
        go a.streamLogs(ctx, name, reader)
    }
}

func (a *LogAggregator) streamLogs(ctx context.Context, service string, reader io.Reader) {
    scanner := bufio.NewScanner(reader)
    for scanner.Scan() {
        entry := a.parser.Parse(scanner.Text())
        entry.Service = service
        entry.NodeID = a.nodeID
        
        a.output.Write(entry.JSON())
    }
}

// Structured log format
{
  "timestamp": "2024-01-15T10:30:00Z",
  "node_id": "node-1",
  "service": "storage",
  "pid": 1003,
  "level": "info",
  "message": "Block stored successfully",
  "fields": {
    "block_id": "Qm...",
    "size": 1024,
    "duration_ms": 45
  }
}
```

## Best Practices

### Service Development

1. **gRPC Interfaces**: Define clear protobuf contracts for all services
2. **Health Endpoints**: Implement health checks for each service
3. **Resource Awareness**: Respect OS-level resource limits
4. **Graceful Shutdown**: Handle SIGTERM for clean exit
5. **Structured Logging**: Use consistent log formats
6. **Metrics Export**: Expose Prometheus metrics endpoint

### Operations

1. **Single Binary**: Deploy one file, spawn all services
2. **Process Monitoring**: Watch subprocess health and resources
3. **Rolling Updates**: Update services individually
4. **Log Aggregation**: Centralize logs from all services
5. **Resource Limits**: Set appropriate limits per service

### Service Communication

1. **Local First**: Use Unix sockets for local services
2. **Network Ready**: TCP/TLS for remote services
3. **Circuit Breakers**: Handle service failures gracefully
4. **Retry Logic**: Implement exponential backoff
5. **Load Balancing**: Distribute load across service instances

## Architecture Benefits

### Process Isolation
- Service crashes don't affect others
- Independent resource limits
- Clear security boundaries
- Language-agnostic services

### Deployment Simplicity
- Single binary distribution
- Subprocess management built-in
- Unified configuration
- Simple updates and rollbacks

### Scalability
- Service-level scaling
- Independent updates
- Resource efficiency
- Horizontal node scaling

## Conclusion

The Blackhole node architecture combines the simplicity of single binary deployment with the robustness of process isolation:

- **One Binary**: Simple distribution and deployment
- **Process Isolation**: True service separation and fault tolerance
- **RPC Communication**: Unified protocol for all services
- **Flexible Scaling**: Per-service resource management
- **Production Ready**: Comprehensive monitoring and health checks

This subprocess architecture provides the perfect balance between operational simplicity and architectural sophistication, enabling both small deployments and large-scale distributed systems.