# Adaptive Resource Management Architecture

*Date: January 19, 2025*

## Overview

The Blackhole platform implements an adaptive resource management system that combines rolling average-based guarantees with burst capability to ensure efficient resource utilization while preventing service starvation. This system operates at the subprocess level, with the orchestrator managing resource limits for each subprocess. The approach provides self-tuning resource allocation that adapts to actual usage patterns over time.

## Core Components

### 1. Subprocess Resource Controller

The orchestrator manages resource limits for each service subprocess:

```go
type SubprocessResourceController struct {
    // Process-level resource tracking
    processMetrics   map[int]*ProcessMetrics // PID -> metrics
    serviceLimits    map[string]*ProcessLimits // service -> limits
    
    // OS-level resource enforcement
    cgroupManager    *CgroupManager
    rlimitManager    *RlimitManager
    
    // Adaptive resource allocation
    adaptiveManager  *AdaptiveResourceManager
}

type ProcessMetrics struct {
    PID         int
    ServiceName string
    CPU         float64  // CPU usage percentage
    Memory      int64    // Memory in bytes
    FDs         int      // File descriptors
    Threads     int      // Thread count
    LastUpdate  time.Time
}
```

### 2. Adaptive Resource Manager

```go
type AdaptiveResourceManager struct {
    // Rolling metrics per service subprocess
    usageMetrics    map[string]*RollingMetrics
    
    // Dynamic guarantees based on historical usage
    adaptiveQuotas  map[string]*AdaptiveQuota
    
    // Shared burst pool for peaks
    burstPool       *BurstResourcePool
    
    // Safety margins and policies
    config          *ResourceConfig
    
    // Process monitoring
    monitor         *ProcessMonitor
}

type ResourceConfig struct {
    windowDuration   time.Duration  // Rolling window (default: 24h)
    sampleInterval   time.Duration  // Metric collection (default: 1m)
    adjustInterval   time.Duration  // Guarantee adjustment (default: 1h)
    safetyMargins    map[ServiceTier]float64
    minimumGuarantees map[string]ResourceAmount
}
```

### 2. Rolling Metrics System

```go
type RollingMetrics struct {
    window       *CircularBuffer
    samples      []ResourceSample
    statistics   *ResourceStatistics
    mu           sync.RWMutex
}

type ResourceStatistics struct {
    average      float64
    median       float64
    percentile95 float64
    peak         float64
    stdDev       float64
    trend        float64  // Positive = increasing usage
}

type ResourceSample struct {
    timestamp time.Time
    cpu       float64
    memory    uint64
    disk      uint64
    network   uint64
    custom    map[string]float64
}
```

### 3. Adaptive Quota System

```go
type AdaptiveQuota struct {
    serviceName      string
    tier            ServiceTier
    baseMinimum     ResourceAmount  // Never go below this
    currentGuarantee ResourceAmount  // Based on rolling average
    maxBurst        ResourceAmount  // Peak allowance
    lastAdjusted    time.Time
    adjustmentLog   []AdjustmentRecord
}

type AdjustmentRecord struct {
    timestamp    time.Time
    oldGuarantee ResourceAmount
    newGuarantee ResourceAmount
    reason       string
    metrics      ResourceStatistics
}
```

## Service Tiers

### Tier Definitions

```go
type ServiceTier int

const (
    Critical ServiceTier = iota  // Identity, Network, Consensus
    Core                        // Storage, Ledger
    Standard                    // Social, Indexer  
    BestEffort                  // Analytics, Telemetry
)

type TierPolicy struct {
    metricType      MetricType  // average, median, p95
    safetyMargin    float64     // Multiplier above baseline
    burstRatio      float64     // Max burst as ratio of guarantee
    preemptible     bool        // Can be preempted for higher tiers
    minGuaranteeRatio float64   // Minimum as ratio of total resources
}
```

### Tier Configuration

```yaml
tiers:
  critical:
    services: [identity, network, consensus]
    policy:
      metric_type: p95
      safety_margin: 1.3    # 30% above P95
      burst_ratio: 2.0      # Can burst to 2x guarantee
      preemptible: false
      min_guarantee_ratio: 0.3  # At least 30% of resources

  core:
    services: [storage, ledger]
    policy:
      metric_type: p90
      safety_margin: 1.25
      burst_ratio: 1.5
      preemptible: false
      min_guarantee_ratio: 0.2

  standard:
    services: [social, indexer]
    policy:
      metric_type: average
      safety_margin: 1.2
      burst_ratio: 1.3
      preemptible: true
      min_guarantee_ratio: 0.1

  best_effort:
    services: [analytics, telemetry]
    policy:
      metric_type: average
      safety_margin: 1.1
      burst_ratio: 1.2
      preemptible: true
      min_guarantee_ratio: 0.05
```

## Resource Allocation Algorithm

### 1. Guarantee Calculation

```go
func (m *AdaptiveResourceManager) calculateGuarantee(service string) ResourceAmount {
    metrics := m.usageMetrics[service]
    tier := m.getServiceTier(service)
    policy := m.getTierPolicy(tier)
    
    // Select baseline metric based on tier
    var baseline float64
    switch policy.metricType {
    case P95:
        baseline = metrics.statistics.percentile95
    case P90:
        baseline = metrics.statistics.percentile90
    case Average:
        baseline = metrics.statistics.average
    case Median:
        baseline = metrics.statistics.median
    }
    
    // Apply trend adjustment
    if metrics.statistics.trend > 0 {
        // Increasing usage - add trend factor
        baseline *= (1 + metrics.statistics.trend * 0.1)
    }
    
    // Apply safety margin
    guarantee := baseline * policy.safetyMargin
    
    // Enforce minimum thresholds
    minimum := m.config.minimumGuarantees[service]
    return max(guarantee, minimum)
}
```

### 2. Resource Allocation

```go
func (m *AdaptiveResourceManager) Allocate(service string, request ResourceRequest) (*Allocation, error) {
    quota := m.adaptiveQuotas[service]
    
    // Phase 1: Try guaranteed allocation
    if request.Size <= quota.currentGuarantee {
        return quota.AllocateGuaranteed(request)
    }
    
    // Phase 2: Try burst allocation
    burstNeeded := request.Size - quota.currentGuarantee
    policy := m.getTierPolicy(quota.tier)
    maxBurst := quota.currentGuarantee * policy.burstRatio
    
    if burstNeeded <= maxBurst && m.burstPool.Available() >= burstNeeded {
        guaranteed := quota.AllocateGuaranteed(quota.currentGuarantee)
        burst := m.burstPool.Allocate(burstNeeded)
        return CombineAllocations(guaranteed, burst)
    }
    
    // Phase 3: Preemption (if allowed)
    if !policy.preemptible {
        preempted := m.tryPreemption(service, request)
        if preempted != nil {
            return preempted
        }
    }
    
    // Phase 4: Partial allocation or queue
    return m.handleOverflow(service, request)
}
```

### 3. Preemption Logic

```go
func (m *AdaptiveResourceManager) tryPreemption(service string, request ResourceRequest) *Allocation {
    requesterTier := m.getServiceTier(service)
    
    // Find lower-tier services using burst resources
    for candidateService, allocation := range m.activeAllocations {
        candidateTier := m.getServiceTier(candidateService)
        
        // Higher tier can preempt lower tier's burst
        if candidateTier > requesterTier && allocation.BurstAmount > 0 {
            candidatePolicy := m.getTierPolicy(candidateTier)
            if candidatePolicy.preemptible {
                if allocation.BurstAmount >= request.Size {
                    preempted := allocation.PreemptBurst(request.Size)
                    return m.reallocate(preempted, service)
                }
            }
        }
    }
    
    return nil
}
```

### 4. Subprocess Resource Enforcement

The orchestrator enforces resource limits at the OS level:

```go
func (c *SubprocessResourceController) EnforceLimit(pid int, limits ProcessLimits) error {
    // Apply CPU limits via cgroups
    if err := c.cgroupManager.SetCPUQuota(pid, limits.CPUQuota); err != nil {
        return fmt.Errorf("failed to set CPU quota: %w", err)
    }
    
    // Apply memory limits
    if err := c.cgroupManager.SetMemoryLimit(pid, limits.MemoryLimit); err != nil {
        return fmt.Errorf("failed to set memory limit: %w", err)
    }
    
    // Apply file descriptor limits via rlimit
    if err := c.rlimitManager.SetFDLimit(pid, limits.MaxFDs); err != nil {
        return fmt.Errorf("failed to set FD limit: %w", err)
    }
    
    // Monitor for violations
    c.monitor.WatchProcess(pid, limits)
    
    return nil
}

type ProcessLimits struct {
    ServiceName string
    PID         int
    CPUQuota    float64  // CPU cores (e.g., 1.5 = 150%)
    MemoryLimit int64    // Bytes
    MaxFDs      int      // File descriptors
    IOPriority  int      // I/O scheduling priority
    Nice        int      // Process priority
}

// Per-subprocess monitoring
func (m *ProcessMonitor) WatchProcess(pid int, limits ProcessLimits) {
    go func() {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        
        for range ticker.C {
            metrics := m.collectProcessMetrics(pid)
            if metrics == nil {
                // Process terminated
                return
            }
            
            // Check for violations
            if metrics.Memory > limits.MemoryLimit {
                m.alerts.TriggerMemoryViolation(limits.ServiceName, metrics)
            }
            
            // Update rolling averages
            m.updateRollingMetrics(limits.ServiceName, metrics)
        }
    }()
}
```

## Monitoring and Metrics

### 1. Resource Monitor

```go
type ResourceMonitor struct {
    manager    *AdaptiveResourceManager
    collectors map[string]MetricCollector
    alerts     *AlertManager
    dashboard  *ResourceDashboard
}

func (m *ResourceMonitor) CollectMetrics(ctx context.Context) {
    ticker := time.NewTicker(m.manager.config.sampleInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            sample := m.collectSample()
            m.processSample(sample)
            m.checkAlerts(sample)
        case <-ctx.Done():
            return
        }
    }
}

func (m *ResourceMonitor) collectSample() *ResourceSample {
    sample := &ResourceSample{
        timestamp: time.Now(),
        services:  make(map[string]ServiceMetrics),
    }
    
    for service, collector := range m.collectors {
        sample.services[service] = collector.Collect()
    }
    
    return sample
}
```

### 2. Resource Dashboard

```go
type ResourceDashboard struct {
    // Real-time metrics
    currentUsage      map[string]ResourceUsage
    currentGuarantees map[string]ResourceAmount
    burstUtilization  float64
    
    // Historical data
    usageTrends       map[string]TimeSeries
    adjustmentHistory map[string][]AdjustmentRecord
    
    // Predictions
    projections       map[string]ResourceProjection
    capacityForecast  CapacityForecast
}

type ResourceProjection struct {
    service          string
    projectedUsage   TimeSeries
    recommendedQuota ResourceAmount
    confidence       float64
}
```

## Safety Mechanisms

### 1. Anomaly Detection

```go
type AnomalyDetector struct {
    zscore    float64  // Z-score threshold
    iqrFactor float64  // IQR multiplier
}

func (d *AnomalyDetector) IsAnomaly(sample ResourceSample, history []ResourceSample) bool {
    // Use z-score for normally distributed data
    if d.isNormallyDistributed(history) {
        mean, stddev := calculateStats(history)
        zscore := math.Abs(sample.Value() - mean) / stddev
        return zscore > d.zscore
    }
    
    // Use IQR for non-normal distributions
    q1, q3 := calculateQuartiles(history)
    iqr := q3 - q1
    lowerBound := q1 - d.iqrFactor*iqr
    upperBound := q3 + d.iqrFactor*iqr
    
    return sample.Value() < lowerBound || sample.Value() > upperBound
}
```

### 2. Adjustment Dampening

```go
type AdjustmentDampener struct {
    maxChangeRate   float64  // Max change per adjustment
    cooldownPeriod  time.Duration
    smoothingFactor float64  // Exponential smoothing
}

func (d *AdjustmentDampener) DampenAdjustment(current, proposed ResourceAmount) ResourceAmount {
    maxChange := current * d.maxChangeRate
    
    if math.Abs(proposed-current) > maxChange {
        if proposed > current {
            return current + maxChange
        }
        return current - maxChange
    }
    
    // Apply exponential smoothing
    return current*d.smoothingFactor + proposed*(1-d.smoothingFactor)
}
```

## Configuration

### 1. Resource Configuration File

```yaml
resource_management:
  # Monitoring configuration
  monitoring:
    window_duration: 24h
    sample_interval: 1m
    adjustment_interval: 1h
    retention_period: 30d
  
  # Safety margins by tier
  safety_margins:
    critical: 1.3
    core: 1.25
    standard: 1.2
    best_effort: 1.1
  
  # Minimum guarantees (never go below)
  minimum_guarantees:
    identity: 
      cpu: 0.5
      memory: 256MB
    network:
      cpu: 0.5
      memory: 512MB
    storage:
      cpu: 1.0
      memory: 1GB
  
  # Burst pool configuration
  burst_pool:
    size: 4GB
    cpu_cores: 4
    max_burst_duration: 5m
    cooldown_period: 1m
  
  # Anomaly detection
  anomaly_detection:
    enabled: true
    zscore_threshold: 3.0
    iqr_factor: 1.5
    min_samples: 100
  
  # Adjustment dampening
  dampening:
    max_change_rate: 0.2  # 20% max change
    cooldown_period: 5m
    smoothing_factor: 0.7
```

### 2. Service Subprocess Configuration

```yaml
services:
  identity:
    tier: critical
    subprocess:
      executable: "./bin/identity-service"
      working_dir: "/var/blackhole/identity"
      environment:
        - SERVICE_NAME=identity
        - RPC_SOCKET=/var/run/blackhole/identity.sock
    resource_requirements:
      min_cpu: 0.5
      min_memory: 256MB
      max_burst_cpu: 2.0
      max_burst_memory: 1GB
      max_fds: 1024
      io_priority: high
    
  storage:
    tier: core
    subprocess:
      executable: "./bin/storage-service"
      working_dir: "/var/blackhole/storage"
      environment:
        - SERVICE_NAME=storage
        - RPC_SOCKET=/var/run/blackhole/storage.sock
    resource_requirements:
      min_cpu: 1.0
      min_memory: 1GB
      max_burst_cpu: 4.0
      max_burst_memory: 4GB
      max_fds: 4096
      io_priority: normal
    
  analytics:
    tier: best_effort
    subprocess:
      executable: "./bin/analytics-service"
      working_dir: "/var/blackhole/analytics"
      environment:
        - SERVICE_NAME=analytics
        - RPC_SOCKET=/var/run/blackhole/analytics.sock
    resource_requirements:
      min_cpu: 0.1
      min_memory: 128MB
      max_burst_cpu: 1.0
      max_burst_memory: 512MB
      max_fds: 512
      io_priority: idle
```

## Implementation Guide

### 1. Integration with RPC Communication

```go
// Resource-aware RPC router
type ResourceAwareRPCRouter struct {
    router   *RPCRouter
    resource *AdaptiveResourceManager
}

func (r *ResourceAwareRPCRouter) Call(service, method string, req interface{}) (interface{}, error) {
    // Check resource availability before routing
    if !r.resource.CanAllocate(service, r.estimateResources(method)) {
        return nil, ErrInsufficientResources
    }
    
    // Allocate resources for the call
    allocation, err := r.resource.Allocate(service, ResourceRequest{
        Service: service,
        Method:  method,
        Estimate: r.estimateResources(method),
    })
    if err != nil {
        return nil, err
    }
    defer allocation.Release()
    
    // Execute the RPC call with allocated resources
    return r.router.CallWithResources(service, method, req, allocation)
}
```

### 2. Service Integration

```go
// Example: Identity service with resource management
type IdentityService struct {
    service.BaseService
    resourceManager *AdaptiveResourceManager
    allocation      *ResourceAllocation
}

func (s *IdentityService) Start(ctx context.Context) error {
    // Request initial resources
    allocation, err := s.resourceManager.Allocate("identity", ResourceRequest{
        Service: "identity",
        Type:    "startup",
        CPU:     0.5,
        Memory:  256 * MB,
    })
    if err != nil {
        return fmt.Errorf("failed to allocate startup resources: %w", err)
    }
    s.allocation = allocation
    
    // Start service with allocated resources
    return s.BaseService.Start(ctx)
}

func (s *IdentityService) Authenticate(ctx context.Context, req AuthRequest) (*AuthResponse, error) {
    // Request additional resources for operation
    opAllocation, err := s.resourceManager.Allocate("identity", ResourceRequest{
        Service: "identity",
        Type:    "authenticate",
        CPU:     0.1,
        Memory:  10 * MB,
    })
    if err != nil {
        return nil, fmt.Errorf("insufficient resources for authentication: %w", err)
    }
    defer opAllocation.Release()
    
    // Perform authentication with allocated resources
    return s.performAuthentication(ctx, req)
}
```

## Performance Considerations

### 1. Low-Overhead Monitoring

- Use lock-free data structures for metric collection
- Batch metric updates to reduce contention
- Implement efficient circular buffers for rolling windows
- Use memory-mapped files for persistent metrics

### 2. Fast Allocation Path

- Cache recent allocation decisions
- Pre-compute common allocation scenarios
- Use atomic operations for simple allocations
- Defer complex calculations to background threads

### 3. Efficient Preemption

- Maintain sorted lists of preemptible allocations
- Use heap data structures for quick selection
- Implement graceful preemption with notifications
- Allow services to voluntarily release resources

## Future Enhancements

### 1. Machine Learning Integration

- Predict resource needs using time-series analysis
- Detect patterns in usage across services
- Optimize burst pool sizing
- Improve anomaly detection accuracy

### 2. Multi-Node Coordination

- Share resource metrics across nodes
- Coordinate burst pools globally
- Implement cross-node preemption
- Balance resources across the network

### 3. Advanced Policies

- Time-of-day aware allocations
- Workload-specific resource profiles
- Dynamic tier adjustments
- Cost-based optimization

## Benefits

### 1. Self-Tuning System

- Automatically adjusts to actual usage patterns
- No manual tuning required
- Learns from historical data
- Adapts to changing workloads

### 2. Efficient Resource Utilization

- No waste from static over-provisioning
- Resources flow to where they're needed
- Burst capability for handling spikes
- Optimal packing and allocation

### 3. Service Reliability

- Guaranteed minimums prevent starvation
- Critical services always have resources
- Graceful degradation under load
- Predictable performance characteristics

### 4. Operational Simplicity

- Single configuration file
- Automatic resource management
- Clear monitoring and alerting
- Self-documenting through metrics

## Conclusion

The adaptive resource management system provides a sophisticated yet efficient approach to resource allocation in the Blackhole platform. By combining rolling average-based guarantees with burst capability and intelligent preemption, it ensures optimal resource utilization while preventing service starvation. The self-tuning nature of the system reduces operational overhead while maintaining high performance and reliability.