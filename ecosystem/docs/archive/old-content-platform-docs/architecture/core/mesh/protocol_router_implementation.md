# Protocol-Level Router Implementation

## Overview

The Protocol-Level Router provides intelligent gRPC request routing with advanced resource management and connection pooling. It implements the design discussed in our architecture sessions, focusing on 80% system utilization by default with runtime adjustability.

## Key Features

### 1. Protocol-Level Routing
- **gRPC Method Parsing**: Parses HTTP/2 paths like `/storage.v1.StorageService/Store`
- **Service Discovery**: Maintains registry of available service endpoints
- **Health Monitoring**: Tracks service health and adjusts routing accordingly
- **Request Distribution**: Intelligent load balancing based on connection load

### 2. Dynamic Resource Management
- **System Detection**: Automatically detects CPU cores, memory, and file descriptor limits
- **80% Default Utilization**: Configurable utilization percentage (default 80%)
- **Runtime Adjustment**: Update resource limits without restart
- **Cross-Platform**: Works on macOS, Linux, and other Unix systems

### 3. Connection Pooling
- **Per-Service Pools**: Dedicated connection pools for each service
- **Intelligent Selection**: Chooses connections based on current load
- **Health Checks**: Periodic health monitoring and connection lifecycle management
- **Resource Limits**: Respects global and per-service connection limits

### 4. Rate Limiting
- **Token Bucket Algorithm**: Implements efficient rate limiting
- **Configurable Limits**: Adjustable requests per second limits
- **Concurrency Control**: Limits concurrent requests to prevent overload

## Architecture

```
ProtocolRouter
├── ResourceDetector (detects system capacity)
├── ResourceManager (enforces limits, tracks usage)
├── ConnectionPools (per-service connection management)
│   ├── ProtocolLevelConnectionPool
│   │   ├── PooledConnection[]
│   │   ├── HealthCheck routine
│   │   └── LoadBalancing
└── ServiceRegistry (endpoint discovery)
```

## Configuration

### Resource Limits Calculation

Based on system resources with configurable utilization percentage:

```go
// Total connections: min(FD_limit/4, network_capacity) * utilization%
MaxTotalConnections = min(fd_limit/4, network_capacity) * 0.80

// Per-service connections: CPU_cores * 2 * utilization%
MaxConnectionsPerService = cpu_cores * 2 * 0.80

// Concurrent requests: CPU_cores * 10 * utilization%
MaxConcurrentRequests = cpu_cores * 10 * 0.80

// Request rate: CPU_cores * 50 * utilization%
MaxRequestsPerSecond = cpu_cores * 50 * 0.80
```

### Connection Lifecycle
- **Idle Timeout**: 5 minutes (configurable)
- **Max Age**: 30 minutes (configurable)
- **Health Check Interval**: 30 seconds

## Usage Examples

### Basic Service Registration

```go
// Create router with 80% resource utilization
router := mesh.NewProtocolRouter(logger)

// Register local service
endpoint := mesh.Endpoint{
    Socket:      "sockets/storage.sock",
    IsLocal:     true,
    LastUpdated: time.Now(),
}
router.RegisterService("storage", endpoint)

// Register remote service
endpoint = mesh.Endpoint{
    Address:     "api.external.com:443",
    IsLocal:     false,
    LastUpdated: time.Now(),
}
router.RegisterService("external-api", endpoint)
```

### Protocol-Level Request Routing

```go
// Route gRPC request
ctx := context.Background()
requestData := []byte(`{"key": "value"}`)

responseData, err := router.RouteRequest(
    ctx, 
    "storage", 
    "/storage.v1.StorageService/Store", 
    requestData,
)
```

### Runtime Resource Management

```go
// Check current resource usage
usage := router.GetResourceUsage()
fmt.Printf("Connection utilization: %.1f%%\n", usage.ConnectionUtilization)

// Adjust resource limits to 60% utilization
router.UpdateResourceLimits(60)

// Get connection pool statistics
poolStats := router.GetPoolStats()
for service, stats := range poolStats {
    fmt.Printf("%s: %d/%d connections, %.1f%% success rate\n",
        service, stats.TotalConnections, stats.MaxConnections, stats.SuccessRate)
}
```

## Implementation Details

### Resource Detection

The `ResourceDetector` automatically discovers:
- **CPU Cores**: Using `runtime.NumCPU()`
- **Memory**: Cross-platform estimation with reasonable bounds
- **File Descriptors**: Using `syscall.Getrlimit(RLIMIT_NOFILE)`
- **Network Capacity**: Calculated based on CPU, memory, and FD limits

### Connection Pool Management

Each service gets a dedicated `ProtocolLevelConnectionPool` that:
- Creates connections on-demand up to service limits
- Selects connections based on lowest active request count
- Performs periodic health checks
- Removes unhealthy or aged connections
- Tracks detailed metrics (latency, success rate, etc.)

### Health Monitoring

Connection health is determined by:
- gRPC connection state (Ready, Idle vs TransientFailure, Shutdown)
- Connection age (max 30 minutes)
- Idle time (max 5 minutes)
- Request success/failure patterns

## Benefits

1. **Automatic Scaling**: Adapts to system capacity without manual tuning
2. **Resource Protection**: Prevents system overload with intelligent limits
3. **High Performance**: Connection pooling reduces connection overhead
4. **Fault Tolerance**: Health monitoring and automatic failover
5. **Observability**: Detailed metrics and resource usage tracking
6. **P2P Optimized**: Designed for P2P networks without centralized load balancers

## Integration

The Protocol Router integrates with:
- **Service Discovery**: Automatic registration of discovered services
- **Health Checks**: Service mesh health monitoring
- **Metrics**: Prometheus/telemetry integration
- **Configuration**: Runtime configuration updates
- **Middleware**: Request/response processing pipeline

## Future Enhancements

- **Advanced Load Balancing**: Weighted round-robin, least connections
- **Circuit Breakers**: Automatic failure detection and isolation
- **Request Tracing**: Distributed tracing integration
- **Compression**: gRPC compression for bandwidth optimization
- **TLS Support**: Secure communication for remote services