# 01_04: Health Monitoring

## Overview

The Health Monitoring component provides comprehensive health tracking, failure detection, and automated recovery for all services within the Blackhole Framework, ensuring system reliability and availability.

## Health Monitoring Architecture

### Component Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Health Checks  │───▶│ Health Monitor  │───▶│ Recovery Engine │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         ▼                        ▼                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Metrics       │    │   Dashboard     │    │   Alerting      │
│  Collection     │    │   & Logging     │    │    System       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Health Check Types

1. **Liveness Checks**: Is the service running?
2. **Readiness Checks**: Is the service ready to handle requests?
3. **Startup Checks**: Has the service completed initialization?
4. **Deep Health Checks**: Are all service dependencies healthy?

## Health Check Implementation

### Health Check Interface

```go
type HealthChecker interface {
    // Basic health check
    HealthCheck(ctx context.Context) HealthStatus
    
    // Detailed health information
    DetailedHealth(ctx context.Context) DetailedHealthStatus
    
    // Register custom health checks
    RegisterCheck(name string, check HealthCheckFunc)
}

type HealthStatus struct {
    Status    HealthState    `json:"status"`
    Timestamp time.Time      `json:"timestamp"`
    Message   string         `json:"message,omitempty"`
}

type DetailedHealthStatus struct {
    Overall   HealthStatus              `json:"overall"`
    Checks    map[string]HealthStatus   `json:"checks"`
    Metrics   map[string]interface{}    `json:"metrics"`
    Version   string                    `json:"version"`
    Uptime    time.Duration             `json:"uptime"`
}
```

### Health States

```go
type HealthState string

const (
    HealthStateHealthy   HealthState = "healthy"
    HealthStateUnhealthy HealthState = "unhealthy"
    HealthStateDegraded  HealthState = "degraded"
    HealthStateStarting  HealthState = "starting"
    HealthStateStopping  HealthState = "stopping"
    HealthStateUnknown   HealthState = "unknown"
)
```

### Service Health Endpoints

Each service exposes standardized health endpoints:

```
GET /health              # Basic health check
GET /health/live         # Liveness probe
GET /health/ready        # Readiness probe
GET /health/startup      # Startup probe
GET /health/detailed     # Comprehensive health information
```

## Health Check Configuration

### Global Health Configuration

```yaml
runtime:
  health:
    # Global health check settings
    check_interval: 30s
    timeout: 10s
    failure_threshold: 3
    recovery_threshold: 2
    
    # Health check endpoints
    endpoints:
      health: "/health"
      liveness: "/health/live"
      readiness: "/health/ready"
      startup: "/health/startup"
    
    # Alerting configuration
    alerting:
      enabled: true
      webhook_url: "https://alerts.blackhole.dev/webhook"
      
    # Metrics collection
    metrics:
      enabled: true
      collection_interval: 15s
```

### Service-Specific Health Configuration

```yaml
services:
  identity:
    health:
      startup_timeout: 60s
      liveness_interval: 30s
      readiness_interval: 10s
      custom_checks:
        - name: "database_connection"
          endpoint: "/health/db"
          timeout: 5s
        - name: "did_resolver"
          endpoint: "/health/resolver"
          timeout: 10s
          
  storage:
    health:
      startup_timeout: 120s
      liveness_interval: 45s
      deep_health_interval: 300s
      custom_checks:
        - name: "ipfs_connection"
          endpoint: "/health/ipfs"
          critical: true
        - name: "storage_capacity"
          endpoint: "/health/capacity"
          warning_threshold: 80
```

## Health Monitoring Process

### Continuous Monitoring

1. **Periodic Checks**: Execute health checks at configured intervals
2. **State Tracking**: Track health state transitions over time
3. **Failure Detection**: Identify when services become unhealthy
4. **Recovery Detection**: Monitor service recovery
5. **Metric Collection**: Gather health-related metrics

### Health Check Execution

```go
func (hm *HealthMonitor) executeHealthCheck(service *Service) {
    ctx, cancel := context.WithTimeout(context.Background(), service.HealthTimeout)
    defer cancel()
    
    start := time.Now()
    status := service.HealthChecker.HealthCheck(ctx)
    duration := time.Since(start)
    
    // Record metrics
    hm.recordHealthMetrics(service.ID, status, duration)
    
    // Update service state
    hm.updateServiceHealth(service.ID, status)
    
    // Check for state transitions
    if hm.hasStateChanged(service.ID, status) {
        hm.handleStateTransition(service.ID, status)
    }
}
```

### Failure Detection Algorithm

The health monitor uses a sliding window approach for failure detection:

```go
type FailureDetector struct {
    threshold      int           // Number of consecutive failures
    window         time.Duration // Time window for failure tracking
    recoveryCount  int           // Successful checks needed for recovery
}

func (fd *FailureDetector) evaluateHealth(history []HealthCheck) HealthState {
    recent := fd.getRecentChecks(history, fd.window)
    failures := fd.countFailures(recent)
    
    if failures >= fd.threshold {
        return HealthStateUnhealthy
    }
    
    if failures > 0 {
        return HealthStateDegraded
    }
    
    return HealthStateHealthy
}
```

## Health Metrics and Observability

### Health Metrics

The health monitor collects comprehensive metrics:

```go
type HealthMetrics struct {
    // Service availability
    ServiceUptime     map[string]time.Duration
    AvailabilityRate  map[string]float64
    
    // Health check performance
    CheckDuration     map[string]time.Duration
    CheckSuccessRate  map[string]float64
    
    // Failure tracking
    FailureCount      map[string]int64
    RecoveryTime      map[string]time.Duration
    
    // System health
    OverallHealth     HealthState
    HealthyServices   int
    UnhealthyServices int
}
```

### Health Events

The system emits health events for monitoring and alerting:

```go
type HealthEvent struct {
    ServiceID     string                 `json:"service_id"`
    EventType     HealthEventType        `json:"event_type"`
    OldState      HealthState            `json:"old_state"`
    NewState      HealthState            `json:"new_state"`
    Timestamp     time.Time              `json:"timestamp"`
    Details       map[string]interface{} `json:"details"`
}
```

### Health Dashboard

Real-time health dashboard showing:

- Service health status overview
- Health check success rates
- Response time trends
- Failure and recovery patterns
- System-wide health metrics

## Automated Recovery

### Recovery Strategies

1. **Service Restart**: Restart unhealthy services
2. **Dependency Restart**: Restart services that depend on failed service
3. **Circuit Breaking**: Temporarily disable failed services
4. **Failover**: Route traffic to healthy instances
5. **Scaling**: Start additional instances to handle load

### Recovery Configuration

```yaml
runtime:
  health:
    recovery:
      enabled: true
      strategies:
        - type: "restart"
          max_attempts: 3
          backoff: "exponential"
          base_delay: 1s
          max_delay: 60s
          
        - type: "circuit_breaker"
          failure_threshold: 5
          timeout: 30s
          
        - type: "failover"
          enabled: false  # Requires multiple instances
```

### Recovery Process

```go
func (rm *RecoveryManager) handleUnhealthyService(serviceID string) {
    service := rm.getService(serviceID)
    strategy := rm.getRecoveryStrategy(service)
    
    switch strategy.Type {
    case RecoveryRestart:
        rm.restartService(serviceID, strategy)
        
    case RecoveryCircuitBreaker:
        rm.enableCircuitBreaker(serviceID, strategy)
        
    case RecoveryFailover:
        rm.enableFailover(serviceID, strategy)
    }
    
    // Schedule recovery verification
    rm.scheduleRecoveryCheck(serviceID, strategy.Timeout)
}
```

## Integration with Other Components

### Process Orchestrator Integration

- Provides health status for restart decisions
- Coordinates service lifecycle with health state
- Manages resource allocation based on health

### Lifecycle Manager Integration

- Influences service startup and shutdown decisions
- Coordinates dependency management with health status
- Manages graceful degradation scenarios

### Mesh Router Integration

- Updates service registry with health status
- Influences traffic routing decisions
- Coordinates load balancing with health state

## Best Practices

1. **Meaningful Health Checks**: Check actual service functionality, not just process existence
2. **Appropriate Timeouts**: Set realistic timeouts for health checks
3. **Dependency Checking**: Include health of critical dependencies
4. **Resource Monitoring**: Include resource utilization in health assessment
5. **Graceful Degradation**: Design services to handle partial failures

## Common Health Check Patterns

### Database Health Check

```go
func (s *Service) checkDatabaseHealth(ctx context.Context) HealthStatus {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    if err := s.db.PingContext(ctx); err != nil {
        return HealthStatus{
            Status:  HealthStateUnhealthy,
            Message: fmt.Sprintf("Database connection failed: %v", err),
        }
    }
    
    return HealthStatus{Status: HealthStateHealthy}
}
```

### External Service Health Check

```go
func (s *Service) checkExternalService(ctx context.Context) HealthStatus {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    resp, err := s.httpClient.Get(s.externalServiceURL + "/health")
    if err != nil {
        return HealthStatus{
            Status:  HealthStateDegraded,
            Message: fmt.Sprintf("External service unreachable: %v", err),
        }
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return HealthStatus{
            Status:  HealthStateDegraded,
            Message: fmt.Sprintf("External service unhealthy: %d", resp.StatusCode),
        }
    }
    
    return HealthStatus{Status: HealthStateHealthy}
}
```

## Troubleshooting

### Health Check Failures

- **Timeout Issues**: Increase timeout or optimize health check logic
- **False Positives**: Review health check logic for accuracy
- **Frequent State Changes**: Adjust thresholds to reduce flapping
- **Resource Contention**: Monitor resource usage during health checks

### Debugging Health Issues

```bash
# Check service health status
blackhole health status <service>

# View health check history
blackhole health history <service>

# Monitor real-time health
blackhole health monitor

# Run manual health check
blackhole health check <service>
```

## See Also

- [01_01-Process Orchestration](./01_01-process_orchestration.md)
- [01_02-Lifecycle Management](./01_02-lifecycle_management.md)
- [01_03-Configuration System](./01_03-configuration_system.md)