# Core Architecture Documentation

This directory contains the detailed architecture documentation for the Blackhole core functionality. The core is a subprocess architecture where services run as independent OS processes orchestrated by a single binary.

## Documentation Structure

- [Subprocess Architecture](./subprocess_architecture.md) - Core subprocess design pattern
- [Process Management](./process_management.md) - Service process lifecycle and supervision
- [RPC Communication](./rpc_communication.md) - gRPC communication patterns
- [RPC Service Architecture](./rpc_service_architecture.md) - Comprehensive RPC-based service communication
- [Resource Isolation](./resource_isolation.md) - OS-level process isolation
- [Service Lifecycle](./service_lifecycle.md) - Service startup/shutdown patterns
- [Deployment Patterns](./deployment_patterns.md) - Single binary deployment options
- [Resource Management](./resource_management.md) - OS-level resource control
- [Service Boundaries](./service_boundaries.md) - Process boundaries and interactions
- [Service Interfaces](./service_interfaces.md) - Service interface contracts

## Core Principles

The core architecture is designed around subprocess isolation:

1. **Process Isolation** - Each service runs in its own OS process
2. **RPC Communication** - All services communicate via gRPC
3. **Resource Control** - OS-level resource management per process
4. **Fault Tolerance** - Process crashes don't affect other services
5. **Security Boundaries** - Process-level security isolation
6. **Operational Simplicity** - Single binary manages all processes
7. **Scalability** - Services can be distributed across machines

## Component Overview

The core consists of several major components:

### Process Orchestrator
- Spawns and manages service processes
- Monitors process health
- Handles process restarts
- Distributes configuration
- Manages service discovery

### RPC Infrastructure
- gRPC servers for each service
- Unix sockets for local communication
- TCP/TLS for remote communication
- Connection pooling and management
- Service discovery integration

### Resource Manager
- OS-level resource limits (cgroups, rlimits)
- CPU quota management
- Memory limit enforcement
- I/O bandwidth control
- File descriptor limits

### Process Supervisor
- Health check monitoring
- Automatic restart policies
- Exponential backoff strategies
- Crash detection and recovery
- Graceful shutdown coordination

### Security Layer
- [Process Security](./process_security.md) - Process sandboxing and isolation
- [Security Architecture](./security_architecture.md) - Overall security model
- [Security Model](./security_model.md) - Comprehensive security framework
- [Security Zones](./security_zones.md) - Access control zones
- Process-level isolation
- mTLS for service authentication
- File system permissions
- Network namespace separation
- Resource access control

### Monitoring System
- Process metrics collection
- Performance monitoring
- Health status aggregation
- Log aggregation from processes
- Distributed tracing support

For detailed information about each component, please refer to the specific documentation files.

## Related Documentation

### Service-Specific Architecture
- [Node Core Architecture](../services/node/node_core_architecture.md) - Node service orchestration
- [Bootstrap Sequence](../services/node/bootstrap_sequence.md) - Node service startup sequence
- [Message Routing](../services/node/message_routing.md) - Node service RPC routing
- [Event System](../services/node/event_system.md) - Node service event management
- [State Management](../services/node/state_management.md) - Node service state synchronization
- [Data Persistence](../services/storage/data_persistence.md) - Storage service data layer

### Operations and Configuration  
- [Configuration System](../../reference/configuration.md) - Unified configuration management
- [Performance Optimization](../../guides/operations/performance_optimization.md) - Process and RPC optimization
- [Monitoring and Diagnostics](../../guides/operations/monitoring_diagnostics.md) - Process monitoring capabilities