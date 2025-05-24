# 01-runtime: Runtime Domain

## Overview

The Runtime Domain serves as the foundational layer of the Blackhole Framework, providing process orchestration, lifecycle management, and system coordination capabilities that enable all other domains to function effectively.

## Core Components

This domain is organized into the following nested components:

### [04_01_01-Process Orchestration](./04_01_01-process_orchestration.md)
The foundation of runtime management, handling:
- Process spawning and supervision
- Resource allocation and limits
- Failure detection and restart logic
- Supervision tree management

### [04_01_02-Lifecycle Management](./04_01_02-lifecycle_management.md)
Coordinates service transitions and dependencies:
- Service state management (UNINITIALIZED → RUNNING → STOPPED)
- Dependency resolution and startup ordering
- Graceful startup and shutdown sequences
- Health-aware lifecycle coordination

### [04_01_03-Configuration System](./04_01_03-configuration_system.md)
Centralized configuration management:
- Hierarchical configuration loading (defaults → files → env → CLI)
- Environment-specific overrides and validation
- Runtime configuration updates and hot-reloading
- Secret management and security

### [04_01_04-Health Monitoring](./04_01_04-health_monitoring.md)
Comprehensive health tracking and recovery:
- Multi-level health checks (liveness, readiness, startup)
- Failure pattern detection and alerting
- Automated recovery mechanisms
- Health metrics and observability

## Runtime Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Runtime Domain                           │
├─────────────────┬─────────────────┬─────────────────┬───────┤
│ 04_01_01-Process│ 04_01_02-Lifecycle│ 04_01_03-Config │04_01_04│
│  Orchestration  │   Management    │    System       │Health │
│                 │                 │                 │Monitor│
├─────────────────┴─────────────────┴─────────────────┴───────┤
│              Foundation Infrastructure                      │
│           (Process Isolation & Resource Management)        │
└─────────────────────────────────────────────────────────────┘
```

## Component Interactions

### Process Orchestration ↔ Lifecycle Management
- **Process Spawning**: Lifecycle manager requests process starts from orchestrator
- **State Coordination**: Orchestrator reports process state to lifecycle manager
- **Resource Allocation**: Lifecycle coordinates resource needs with orchestrator

### Lifecycle Management ↔ Configuration System
- **Service Configuration**: Lifecycle manager retrieves service configs
- **Dependency Resolution**: Configuration provides service dependency information
- **Dynamic Updates**: Lifecycle responds to configuration changes

### Health Monitoring ↔ All Components
- **Process Health**: Monitors orchestrator-managed processes
- **Lifecycle Health**: Tracks service lifecycle state health
- **Configuration Health**: Validates configuration system health

## Implementation Status

### ✅ Implemented
- Process orchestrator with subprocess management
- Basic lifecycle management with dependency resolution
- Configuration loading and validation
- Service health monitoring with basic recovery

### 🔄 In Progress
- Advanced failure detection algorithms
- Dynamic configuration hot-reloading
- Comprehensive health metrics collection
- Automated recovery strategy optimization

### 🆕 Planned
- Distributed coordination capabilities
- Advanced resource management and optimization
- Performance monitoring and tuning
- Enterprise monitoring and alerting integration

## Architecture Principles

1. **Process Isolation**: Services run as independent OS processes for true isolation
2. **Failure Isolation**: Service failures never compromise the runtime core
3. **Resource Management**: OS-level resource allocation, limits, and monitoring
4. **Health-First Design**: Continuous health monitoring with automated recovery
5. **Configuration-Driven**: All behavior controlled through declarative configuration
6. **Dependency Awareness**: Intelligent service ordering and failure handling

## Configuration Example

```yaml
runtime:
  orchestrator:
    max_processes: 50
    restart_policy:
      max_attempts: 5
      backoff_multiplier: 2
      max_backoff: 60s
      
  lifecycle:
    startup_timeout: 60s
    shutdown_timeout: 30s
    dependency_timeout: 30s
    
  health:
    check_interval: 30s
    failure_threshold: 3
    recovery_threshold: 2
    
  configuration:
    hot_reload: true
    validation: strict
    secret_interpolation: true
```

## Getting Started

To understand the Runtime Domain, follow this recommended reading order:

1. **[01_01-Process Orchestration](./01_01-process_orchestration.md)** - Understand the foundation
2. **[01_02-Lifecycle Management](./01_02-lifecycle_management.md)** - Learn service coordination
3. **[01_03-Configuration System](./01_03-configuration_system.md)** - Master configuration management
4. **[01_04-Health Monitoring](./01_04-health_monitoring.md)** - Explore health and recovery

## Integration Points

The Runtime Domain provides foundational services to:

- **[02-Plugin Domain](../02-plugins/)**: Manages plugin process lifecycle and health
- **[03-Mesh Domain](../03-mesh/)**: Coordinates service discovery and mesh routing
- **[04-Resource Domain](../04-resources/)**: Provides resource allocation and monitoring
- **All Framework Services**: Foundational runtime capabilities for any service

## Troubleshooting

### Common Runtime Issues

- **Service Won't Start**: Check dependencies, configuration, and resource allocation
- **Frequent Restarts**: Review health check configuration and resource limits
- **Configuration Errors**: Validate YAML syntax and schema compliance
- **Health Check Failures**: Verify health endpoints and timeout settings

### Debugging Commands

```bash
# Runtime status
blackhole status --detailed

# Service lifecycle state
blackhole lifecycle status <service>

# Configuration validation
blackhole config validate

# Health monitoring
blackhole health monitor --all
```

For detailed troubleshooting guides, see each component's documentation.

## See Also

- [Blackhole Foundation Document](../../02-blackhole_foundation.md)
- [04-Domains Overview](../README.md)
- [Development Guidelines](../../06-guides/01-development_guidelines.md)