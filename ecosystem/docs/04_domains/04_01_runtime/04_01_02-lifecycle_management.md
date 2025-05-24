# 01_02: Lifecycle Management

## Overview

The Lifecycle Management component coordinates the startup, shutdown, and state transitions of all services within the Blackhole Framework, ensuring proper dependency handling and graceful state management.

## Service Lifecycle States

### State Diagram

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ UNINITIALIZED│───▶│ INITIALIZING│───▶│    READY    │───▶│   RUNNING   │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                          │                                       │
                          ▼                                       ▼
                   ┌─────────────┐                        ┌─────────────┐
                   │    ERROR    │                        │  STOPPING   │
                   └─────────────┘                        └─────────────┘
                          ▲                                       │
                          └───────────────────────────────────────▼
                                                          ┌─────────────┐
                                                          │   STOPPED   │
                                                          └─────────────┘
```

### State Descriptions

1. **UNINITIALIZED**: Service not yet loaded or configured
2. **INITIALIZING**: Service loading configuration and dependencies
3. **READY**: Service initialized and ready to start
4. **RUNNING**: Service actively processing requests
5. **STOPPING**: Service gracefully shutting down
6. **STOPPED**: Service cleanly terminated
7. **ERROR**: Service encountered unrecoverable error

## Dependency Management

### Dependency Types

1. **Hard Dependencies**: Must be running before service can start
2. **Soft Dependencies**: Preferred to be running but not required
3. **Circular Dependencies**: Detected and prevented
4. **Optional Dependencies**: Can start without but will integrate when available

### Dependency Resolution

The lifecycle manager uses topological sorting to determine startup order:

```yaml
# Example dependency configuration
services:
  identity:
    dependencies: []
  
  storage:
    dependencies: [identity]
  
  mesh:
    dependencies: [identity]
  
  node:
    dependencies: [identity, storage, mesh]
```

## Startup Sequence

### Framework Startup

1. **Configuration Loading**: Load and validate main configuration
2. **Core Services**: Start essential runtime services (orchestrator, mesh router)
3. **Foundation Services**: Start identity and base services
4. **Application Services**: Start business logic services
5. **Health Verification**: Verify all services are healthy

### Service Startup Process

1. **Environment Preparation**: Set up service environment
2. **Configuration Injection**: Provide service-specific configuration
3. **Dependency Verification**: Ensure dependencies are available
4. **Process Spawning**: Launch service process
5. **Health Check**: Verify successful startup
6. **Registration**: Register with mesh router and discovery

## Shutdown Sequence

### Graceful Shutdown

1. **Stop New Requests**: Prevent new work from starting
2. **Drain Connections**: Complete in-flight requests
3. **Dependency Order**: Shutdown in reverse dependency order
4. **Resource Cleanup**: Release allocated resources
5. **Process Termination**: Cleanly terminate processes

### Shutdown Timeout

Services have configurable shutdown timeouts:

```yaml
runtime:
  lifecycle:
    shutdown_timeout: 30s
    force_kill_timeout: 60s
    dependency_wait_timeout: 10s
```

## Health Management Integration

### Health Check Coordination

The lifecycle manager coordinates with the health monitor to:

- Track service health during transitions
- Trigger restarts for unhealthy services
- Prevent startup of services with unhealthy dependencies
- Coordinate rolling updates and deployments

### Recovery Strategies

1. **Restart**: Simple service restart for transient issues
2. **Cascade Restart**: Restart dependent services after core service restart
3. **Partial Shutdown**: Stop subset of services to prevent cascade failures
4. **Emergency Stop**: Immediate termination for security or stability

## Configuration

### Lifecycle Configuration

```yaml
runtime:
  lifecycle:
    startup_timeout: 60s
    shutdown_timeout: 30s
    health_check_interval: 10s
    dependency_timeout: 30s
    max_concurrent_starts: 5
    
  services:
    identity:
      startup_timeout: 30s
      shutdown_timeout: 15s
      health_check_path: "/health"
      
    storage:
      startup_timeout: 45s
      shutdown_timeout: 30s
      dependencies: ["identity"]
```

## Monitoring and Observability

### Lifecycle Metrics

- Service transition durations
- Dependency resolution times
- Startup and shutdown success rates
- Health check response times
- Error rates by lifecycle phase

### Lifecycle Events

The lifecycle manager emits events for:

- Service state transitions
- Dependency resolution results
- Health check status changes
- Timeout and error conditions

## Error Handling

### Common Failure Scenarios

1. **Startup Timeout**: Service fails to start within timeout
2. **Dependency Failure**: Required dependency unavailable
3. **Health Check Failure**: Service becomes unhealthy after startup
4. **Shutdown Timeout**: Service fails to shutdown gracefully
5. **Resource Exhaustion**: Insufficient resources for service startup

### Recovery Mechanisms

- Exponential backoff for restart attempts
- Circuit breaker pattern for repeated failures
- Cascading failure prevention
- Emergency shutdown procedures

## Integration Points

### With Process Orchestrator
- Coordinates process spawning and termination
- Manages resource allocation during transitions

### With Health Monitor
- Receives health status updates
- Triggers lifecycle transitions based on health

### With Configuration System
- Loads service-specific configurations
- Handles configuration updates and reloads

## Best Practices

1. **Dependency Design**: Minimize hard dependencies between services
2. **Graceful Handling**: Implement proper shutdown handlers
3. **Health Checks**: Provide meaningful health endpoints
4. **Timeout Configuration**: Set appropriate timeouts for your services
5. **Error Logging**: Log lifecycle events for debugging

## Troubleshooting

### Service Won't Start
- Check dependency availability
- Verify configuration validity
- Review resource allocation
- Examine startup logs

### Service Won't Stop
- Check for hanging connections
- Review shutdown handler implementation
- Verify timeout configuration
- Use force kill as last resort

## See Also

- [01_01-Process Orchestration](./01_01-process_orchestration.md)
- [01_03-Configuration System](./01_03-configuration_system.md)
- [01_04-Health Monitoring](./01_04-health_monitoring.md)