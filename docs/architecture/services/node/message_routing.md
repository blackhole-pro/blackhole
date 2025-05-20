# RPC Message Routing Architecture

## Overview

The RPC Message Routing system manages communication between service subprocesses within the Blackhole node orchestrator. It routes gRPC messages between services running as separate OS processes, handles service discovery, load balancing, and ensures reliable delivery through local Unix sockets and TCP connections.

## Architecture Components

### RPC Router

The core gRPC routing engine in the orchestrator:

#### Core Responsibilities
- **Service Discovery**: Locate subprocess RPC endpoints
- **Connection Management**: Maintain gRPC client connections
- **Load Balancing**: Distribute requests across service instances
- **Health Monitoring**: Track subprocess health status
- **Failure Recovery**: Handle process crashes and restarts

#### Routing Tables
- **Local Services**: Unix socket paths for subprocesses
- **Remote Services**: TCP endpoints for network services
- **Process Registry**: PID to service mapping
- **Health Status**: Service availability tracking
- **Connection Pool**: Reusable gRPC connections

### Message Types

#### Inter-Process Messages (RPC)
- **Service Requests**: gRPC method calls between services
- **Health Checks**: Process liveness verification
- **Resource Stats**: Process resource usage reports
- **Control Commands**: Process lifecycle management
- **Event Notifications**: Cross-process event delivery

#### Network Messages (P2P)
- **Node Communication**: Handled by Node service subprocess
- **DHT Messages**: Distributed hash table operations
- **Block Propagation**: Blockchain data distribution
- **Peer Discovery**: Network topology updates
- **Content Routing**: IPFS content location

#### Client Messages
- **API Gateway**: External client requests via orchestrator
- **Streaming RPC**: Bidirectional streaming support
- **Batch Operations**: Multiple requests in single RPC
- **Error Responses**: Process failure notifications
- **Status Updates**: Real-time service status

## Routing Mechanisms

### Service Discovery

#### Local Service Discovery
```go
type LocalServiceDiscovery struct {
    services  map[string]*ServiceEndpoint
    mu        sync.RWMutex
}

type ServiceEndpoint struct {
    Name       string
    PID        int
    UnixSocket string
    TCPPort    int
    Status     HealthStatus
    LastCheck  time.Time
}

func (d *LocalServiceDiscovery) DiscoverService(name string) (*ServiceEndpoint, error) {
    d.mu.RLock()
    defer d.mu.RUnlock()
    
    if endpoint, ok := d.services[name]; ok {
        if endpoint.Status == Healthy {
            return endpoint, nil
        }
    }
    
    return nil, ErrServiceUnavailable
}
```

#### Process Registration
```go
// Service subprocess registers itself on startup
func (s *ServiceProcess) Register() error {
    registration := &ServiceRegistration{
        Name:       s.Name,
        PID:        os.Getpid(),
        UnixSocket: s.UnixSocket,
        TCPPort:    s.Port,
        Timestamp:  time.Now(),
    }
    
    // Register with orchestrator via filesystem
    regPath := fmt.Sprintf("/var/run/blackhole/%s.reg", s.Name)
    data, _ := json.Marshal(registration)
    return ioutil.WriteFile(regPath, data, 0644)
}
```

### Connection Management

#### Connection Pool
```go
type RPCConnectionPool struct {
    connections map[string]*grpc.ClientConn
    options     ConnectionOptions
    mu          sync.RWMutex
}

func (p *RPCConnectionPool) GetConnection(service string) (*grpc.ClientConn, error) {
    p.mu.RLock()
    if conn, ok := p.connections[service]; ok {
        p.mu.RUnlock()
        if conn.GetState() == connectivity.Ready {
            return conn, nil
        }
    }
    p.mu.RUnlock()
    
    // Create new connection
    return p.createConnection(service)
}

func (p *RPCConnectionPool) createConnection(service string) (*grpc.ClientConn, error) {
    endpoint, err := p.discovery.DiscoverService(service)
    if err != nil {
        return nil, err
    }
    
    var opts []grpc.DialOption
    
    // Use Unix socket for local services
    if endpoint.UnixSocket != "" {
        opts = append(opts, 
            grpc.WithInsecure(),
            grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
                return net.Dial("unix", endpoint.UnixSocket)
            }),
        )
        target = "unix://" + endpoint.UnixSocket
    } else {
        // Use TCP for remote services
        opts = append(opts, grpc.WithTransportCredentials(p.credentials))
        target = fmt.Sprintf("%s:%d", endpoint.Host, endpoint.TCPPort)
    }
    
    conn, err := grpc.DialContext(ctx, target, opts...)
    if err != nil {
        return nil, err
    }
    
    p.mu.Lock()
    p.connections[service] = conn
    p.mu.Unlock()
    
    return conn, nil
}
```

### Load Balancing

#### Round-Robin Balancer
```go
type RoundRobinBalancer struct {
    services map[string][]*ServiceEndpoint
    current  map[string]int
    mu       sync.Mutex
}

func (b *RoundRobinBalancer) NextEndpoint(service string) (*ServiceEndpoint, error) {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    endpoints := b.services[service]
    if len(endpoints) == 0 {
        return nil, ErrNoAvailableEndpoints
    }
    
    // Get current index and increment
    idx := b.current[service]
    b.current[service] = (idx + 1) % len(endpoints)
    
    return endpoints[idx], nil
}
```

#### Health-Aware Balancer
```go
type HealthAwareBalancer struct {
    balancer LoadBalancer
    health   HealthChecker
}

func (b *HealthAwareBalancer) NextEndpoint(service string) (*ServiceEndpoint, error) {
    for attempts := 0; attempts < 3; attempts++ {
        endpoint, err := b.balancer.NextEndpoint(service)
        if err != nil {
            return nil, err
        }
        
        if b.health.IsHealthy(endpoint) {
            return endpoint, nil
        }
    }
    
    return nil, ErrNoHealthyEndpoints
}
```

#### Table Types
- **Service Table**: Internal services
- **Peer Table**: Network nodes
- **Topic Table**: Pub/sub topics
- **Geographic Table**: Location routes
- **Policy Table**: Routing rules

## RPC Request Management

### Request Handling

#### Request Types
```go
type RPCRequest struct {
    ID           string
    Service      string
    Method       string
    Payload      []byte
    Timeout      time.Duration
    Priority     RequestPriority
    RetryPolicy  *RetryPolicy
}

type RequestPriority int
const (
    LowPriority RequestPriority = iota
    NormalPriority
    HighPriority
    CriticalPriority
)
```

#### Request Queue Per Service
```go
type ServiceRequestQueue struct {
    service    string
    queue      *PriorityQueue
    processing int32  // Atomic counter
    maxWorkers int
}

func (q *ServiceRequestQueue) Enqueue(req *RPCRequest) error {
    if q.queue.Size() >= q.queue.Capacity() {
        return ErrQueueFull
    }
    
    q.queue.Push(req, int(req.Priority))
    return nil
}

func (q *ServiceRequestQueue) Process(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            if atomic.LoadInt32(&q.processing) >= int32(q.maxWorkers) {
                time.Sleep(10 * time.Millisecond)
                continue
            }
            
            req := q.queue.Pop()
            if req == nil {
                time.Sleep(10 * time.Millisecond)
                continue
            }
            
            atomic.AddInt32(&q.processing, 1)
            go q.handleRequest(req.(*RPCRequest))
        }
    }
}
```

### Circuit Breaker

#### Process-Level Circuit Breaking
```go
type ProcessCircuitBreaker struct {
    service     string
    state       CircuitState
    failures    int64
    lastFailure time.Time
    timeout     time.Duration
    threshold   int
}

func (cb *ProcessCircuitBreaker) Call(req *RPCRequest) (*RPCResponse, error) {
    if !cb.canMakeRequest() {
        return nil, ErrCircuitOpen
    }
    
    resp, err := cb.executeRequest(req)
    
    if err != nil {
        cb.recordFailure()
        return nil, err
    }
    
    cb.recordSuccess()
    return resp, nil
}

func (cb *ProcessCircuitBreaker) canMakeRequest() bool {
    switch cb.state {
    case Open:
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = HalfOpen
            return true
        }
        return false
    case HalfOpen:
        return true
    case Closed:
        return true
    }
    return false
}
```

### Retry Logic

#### Service-Aware Retry
```go
type RetryManager struct {
    policies map[string]*RetryPolicy
}

type RetryPolicy struct {
    MaxAttempts int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
    Jitter       bool
}

func (r *RetryManager) ExecuteWithRetry(req *RPCRequest) (*RPCResponse, error) {
    policy := r.policies[req.Service]
    if policy == nil {
        policy = r.policies["default"]
    }
    
    var lastErr error
    delay := policy.InitialDelay
    
    for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
        resp, err := r.executeRequest(req)
        if err == nil {
            return resp, nil
        }
        
        lastErr = err
        
        // Check if error is retryable
        if !isRetryable(err) {
            return nil, err
        }
        
        // Apply backoff
        time.Sleep(delay)
        delay = time.Duration(float64(delay) * policy.Multiplier)
        if delay > policy.MaxDelay {
            delay = policy.MaxDelay
        }
        
        if policy.Jitter {
            jitter := time.Duration(rand.Int63n(int64(delay) / 4))
            delay = delay + jitter
        }
    }
    
    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

## Delivery Guarantees

### Reliability Levels

#### At-Most-Once
- **Characteristics**: No duplicates, possible loss
- **Use Cases**: Metrics, logs, non-critical data
- **Implementation**: Fire-and-forget
- **Performance**: Highest throughput
- **Overhead**: Minimal

#### At-Least-Once
- **Characteristics**: No loss, possible duplicates
- **Use Cases**: Important notifications
- **Implementation**: Acknowledgment-based
- **Performance**: Good throughput
- **Overhead**: Moderate

#### Exactly-Once
- **Characteristics**: No loss, no duplicates
- **Use Cases**: Financial transactions
- **Implementation**: Two-phase commit
- **Performance**: Lower throughput
- **Overhead**: Highest

### Delivery Mechanisms

#### Direct Delivery
- **Point-to-Point**: Direct routing
- **Synchronous**: Immediate delivery
- **Connection-Oriented**: Persistent connection
- **Reliability**: TCP-like guarantees
- **Use Cases**: Critical messages

#### Store-and-Forward
- **Hop-by-Hop**: Intermediate storage
- **Asynchronous**: Delayed delivery
- **Resilient**: Network partition tolerant
- **Reliability**: High durability
- **Use Cases**: Distributed systems

#### Publish-Subscribe
- **Topic-Based**: Content channels
- **Fan-Out**: Multiple recipients
- **Decoupled**: Producer-consumer separation
- **Scalable**: Horizontal scaling
- **Use Cases**: Event systems

## Routing Optimization

### Performance Optimization

#### Route Caching
- **LRU Cache**: Recent routes
- **TTL-Based**: Time expiration
- **Invalidation**: Change detection
- **Preloading**: Anticipatory caching
- **Hierarchical**: Multi-level cache

#### Path Compression
- **Route Aggregation**: Combined paths
- **Hop Reduction**: Shorter paths
- **Direct Links**: Bypass intermediaries
- **Express Routes**: Fast lanes
- **Tunnel Creation**: Virtual paths

#### Parallel Processing
- **Multi-Path**: Concurrent routing
- **Load Spreading**: Traffic distribution
- **Redundant Paths**: Reliability
- **Aggregation**: Result combination
- **Race Conditions**: First-to-finish

### Adaptive Routing

#### Dynamic Adaptation
- **Performance Monitoring**: Route metrics
- **Congestion Detection**: Overload identification
- **Failure Detection**: Dead route discovery
- **Load Balancing**: Traffic redistribution
- **Quality Metrics**: Service level tracking

#### Machine Learning
- **Pattern Recognition**: Traffic patterns
- **Predictive Routing**: Anticipatory paths
- **Anomaly Detection**: Unusual behavior
- **Optimization**: Route improvement
- **Self-Learning**: Continuous improvement

## Security and Privacy

### Message Security

#### Encryption
- **Transport Security**: TLS/DTLS
- **End-to-End**: Message encryption
- **Key Management**: Secure key exchange
- **Perfect Forward Secrecy**: Session keys
- **Algorithm Selection**: Cipher suites

#### Authentication
- **Message Signing**: Digital signatures
- **Origin Verification**: Sender authentication
- **Integrity Checking**: Tampering detection
- **Non-Repudiation**: Proof of origin
- **Certificate Validation**: Trust verification

### Privacy Protection

#### Anonymous Routing
- **Onion Routing**: Layered encryption
- **Mix Networks**: Traffic mixing
- **Cover Traffic**: Pattern obfuscation
- **Timing Attacks**: Mitigation strategies
- **Metadata Protection**: Header encryption

#### Content Protection
- **Payload Encryption**: Data confidentiality
- **Selective Disclosure**: Partial decryption
- **Zero-Knowledge**: Privacy proofs
- **Homomorphic Processing**: Encrypted computation
- **Data Minimization**: Reduced exposure

## Error Handling

### Error Types

#### Routing Errors
- **No Route**: Destination unreachable
- **Route Loop**: Circular routing
- **Expired Route**: Stale information
- **Overload**: Capacity exceeded
- **Timeout**: Delivery deadline missed

#### Message Errors
- **Corruption**: Data integrity failure
- **Format Error**: Invalid message structure
- **Size Limit**: Message too large
- **Authentication**: Security failure
- **Authorization**: Access denied

### Error Recovery

#### Retry Strategies
- **Exponential Backoff**: Increasing delays
- **Linear Backoff**: Fixed increments
- **Jittered Retry**: Randomized timing
- **Circuit Breaker**: Failure protection
- **Alternate Routes**: Fallback paths

#### Error Propagation
- **Error Codes**: Standardized errors
- **Error Context**: Diagnostic information
- **Stack Traces**: Debug information
- **Error Aggregation**: Related errors
- **Upstream Notification**: Source alerting

## Monitoring and Diagnostics

### Performance Metrics

#### Routing Metrics
- **Message Latency**: End-to-end time
- **Hop Count**: Path length
- **Queue Depth**: Backlog size
- **Throughput**: Messages per second
- **Route Changes**: Stability metric

#### Resource Metrics
- **CPU Usage**: Processing load
- **Memory Usage**: Buffer consumption
- **Network Usage**: Bandwidth utilization
- **Disk I/O**: Persistence operations
- **Connection Count**: Active routes

### Diagnostic Tools

#### Route Tracing
- **Path Discovery**: Route visualization
- **Hop Analysis**: Per-hop metrics
- **Bottleneck Detection**: Slow segments
- **Loop Detection**: Circular routes
- **MTU Discovery**: Size limits

#### Message Tracing
- **Message Tracking**: End-to-end flow
- **Timestamp Analysis**: Timing breakdown
- **Queue Inspection**: Buffer status
- **Drop Analysis**: Lost messages
- **Duplicate Detection**: Redundant messages

## Integration Patterns

### Service Integration

#### Service Mesh
- **Sidecar Proxy**: Service routing
- **Load Balancing**: Request distribution
- **Circuit Breaking**: Failure handling
- **Observability**: Metrics collection
- **Security**: mTLS enforcement

#### Event Bus
- **Topic Management**: Channel organization
- **Event Routing**: Message distribution
- **Subscription Handling**: Interest management
- **Event Ordering**: Sequence guarantee
- **Event Replay**: Historical events

### Network Integration

#### P2P Integration
- **DHT Routing**: Distributed routing
- **Gossip Protocol**: Information spread
- **Peer Selection**: Node choosing
- **Relay Support**: NAT traversal
- **Content Routing**: Data discovery

#### Gateway Integration
- **Protocol Translation**: Format conversion
- **Border Routing**: Network edges
- **Security Enforcement**: Policy application
- **Rate Limiting**: Traffic control
- **Monitoring**: Edge visibility

## Advanced Features

### Multicast Routing

#### Tree Construction
- **Spanning Tree**: Optimal paths
- **Steiner Tree**: Minimal cost
- **Source-Specific**: Directed trees
- **Shared Tree**: Common paths
- **Dynamic Trees**: Adaptive topology

#### Group Management
- **Membership**: Join/leave operations
- **Group Discovery**: Finding groups
- **Access Control**: Permission management
- **State Synchronization**: Group consistency
- **Failure Handling**: Member recovery

### Geographic Routing

#### Location-Based
- **Coordinate Systems**: Position mapping
- **Distance Metrics**: Proximity calculation
- **Region Mapping**: Area definitions
- **Geocasting**: Area broadcast
- **Location Privacy**: Position protection

#### Latency Optimization
- **RTT Measurements**: Round-trip times
- **Path Selection**: Lowest latency
- **Regional Caching**: Local content
- **Edge Computing: Distributed processing
- **CDN Integration**: Content delivery

### Quality of Service

#### Traffic Classes
- **Real-Time**: Voice/video traffic
- **Interactive**: User interactions
- **Bulk Transfer**: Large files
- **Background**: Low priority
- **Control**: System messages

#### Resource Reservation
- **Bandwidth Allocation**: Capacity reservation
- **Buffer Management**: Queue allocation
- **Priority Scheduling**: Processing order
- **Admission Control**: Load management
- **SLA Enforcement**: Service guarantees

## Performance Tuning

### Optimization Strategies

#### Algorithm Tuning
- **Route Calculation**: Algorithm selection
- **Cache Sizing**: Memory allocation
- **Timeout Values**: Deadline tuning
- **Batch Sizes**: Processing groups
- **Thread Pools**: Concurrency levels

#### Resource Management
- **Memory Pooling**: Allocation efficiency
- **Connection Pooling**: Socket reuse
- **Buffer Recycling**: Memory reuse
- **CPU Affinity**: Processor binding
- **I/O Scheduling**: Disk optimization

### Scalability Considerations

#### Horizontal Scaling
- **Sharding**: Route distribution
- **Replication**: Redundancy
- **Load Distribution**: Even spreading
- **Stateless Design**: Easy scaling
- **Consistent Hashing**: Stable distribution

#### Vertical Scaling
- **Resource Limits**: Capacity planning
- **Performance Profiling**: Bottleneck identification
- **Algorithm Complexity**: O(n) analysis
- **Memory Optimization**: Efficient structures
- **Concurrency Tuning**: Thread optimization

## Best Practices

### Design Principles
- **Loose Coupling**: Service independence
- **High Cohesion**: Related functionality
- **Idempotency**: Repeated operations
- **Graceful Degradation**: Failure handling
- **Progressive Enhancement**: Feature layers

### Implementation Guidelines
- **Error Handling**: Comprehensive coverage
- **Logging**: Diagnostic information
- **Monitoring**: Performance tracking
- **Documentation**: Clear interfaces
- **Testing**: Comprehensive validation

### Operational Practices
- **Capacity Planning**: Growth preparation
- **Performance Monitoring**: Continuous tracking
- **Security Audits**: Regular reviews
- **Disaster Recovery**: Backup procedures
- **Update Procedures**: Safe deployment

### Maintenance
- **Route Optimization**: Path improvement
- **Cache Tuning**: Hit rate optimization
- **Queue Management**: Backlog control
- **Metric Analysis**: Performance insights
- **Documentation Updates**: Current information