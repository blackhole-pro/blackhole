# 02-plugins: Plugin Management Domain

## Overview

The Plugin Management Domain handles plugin loading, execution, isolation, and hot-swapping. This is one of Blackhole Foundation's core differentiators - the ability to load and unload plugins without system downtime while maintaining fault isolation.

## Core Components

This domain is organized with nested numbering to reflect the plugin development and deployment lifecycle:

### [04_02_01-Development Guide](./04_02_01-development.md)
Comprehensive plugin development resource covering:
- Plugin architecture and interfaces
- Development workflow and best practices
- Framework integration patterns
- Testing and validation strategies

### [04_02_02-Plugin Examples](./04_02_02_examples/)
Reference implementations demonstrating plugin capabilities:
- **[04_02_02_01-Analytics](./04_02_02_examples/04_02_02_01_analytics/)** - Data collection, analysis, and insights
- **[04_02_02_02-Telemetry](./04_02_02_examples/04_02_02_02_telemetry/)** - Real-time operational monitoring

### [04_02_03-Plugin Interface Architecture](./04_02_03-plugin_interface_architecture.md)
**IMPORTANT**: Explains why Blackhole plugins don't use common base interfaces - a fundamental architectural decision that enables maximum flexibility and type safety.

### [04_02_04-Plugin Development Guide](./04_02_04-plugin_development_guide.md)
Practical guide for building plugins with the unique Blackhole architecture, including key differences from traditional plugin systems.

## Plugin System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                Plugin Management Domain                     │
├─────────────────────────────────┬───────────────────────────┤
│      04_02_01-Development       │   04_02_02-Examples       │
│      (Guides & Interfaces)      │  (Reference Plugins)      │
├─────────────────────────────────┴───────────────────────────┤
│            Plugin Runtime Infrastructure                    │
│    (Registry, Loader, Executor, Lifecycle, State)          │
└─────────────────────────────────────────────────────────────┘
```

## Plugin Architecture

### Key Architectural Insight

**Blackhole plugins are different**: Unlike traditional plugin systems with common base interfaces, each Blackhole plugin defines its own complete gRPC interface. Plugins are:
- **Independent services** that run as separate processes
- **Connected via mesh network**, not direct API calls
- **Domain-specific** with interfaces optimized for their use case

Read [Plugin Interface Architecture](./04_02_03-plugin_interface_architecture.md) to understand why this matters.

### Framework Structure
```
internal/framework/plugins/
├── interfaces.go        # Core plugin interfaces and types
├── registry/           # Plugin discovery and registration
├── loader/             # Plugin loading and validation
├── executor/           # Plugin execution and isolation
├── lifecycle/          # Plugin lifecycle management
└── state/             # Plugin state management and migration
```

## Key Capabilities

### Plugin Isolation Levels
- **None**: Same process (fastest, no isolation)
- **Thread**: Separate thread (fast, basic isolation) 
- **Process**: Separate process (good isolation, our default)
- **Container**: Container isolation (strong isolation, higher overhead)
- **VM**: Virtual machine isolation (maximum isolation, highest overhead)

### Plugin Sources
- **Local**: Plugins from local filesystem
- **Remote**: Plugins from remote URLs  
- **Marketplace**: Plugins from the official marketplace

### Hot Swapping Process
1. **Prepare**: Signal current plugin to prepare for shutdown
2. **Export State**: Extract current plugin state
3. **Load New**: Load new version of plugin
4. **Import State**: Transfer state to new version
5. **Switch**: Atomically switch to new version
6. **Cleanup**: Remove old version

## Plugin Development Workflow

### Quick Start Example

```go
type EchoPlugin struct{}

func (p *EchoPlugin) Info() PluginInfo {
    return PluginInfo{
        Name:        "echo",
        Version:     "1.0.0", 
        Description: "Simple echo plugin",
    }
}

func (p *EchoPlugin) Handle(ctx context.Context, req PluginRequest) (PluginResponse, error) {
    return PluginResponse{
        Success: true,
        Data:    append([]byte("Echo: "), req.Data...),
    }, nil
}
```

### Plugin Manifest

```yaml
name: echo
version: 1.0.0
description: Simple echo plugin for testing
author: Blackhole Team
license: MIT

capabilities:
  - computation

permissions:
  - network

resources:
  cpu: 50      # 50% of one core
  memory: 100  # 100MB
  
isolation: process

dependencies: []
```

## Implementation Status

### ✅ Implemented
- Plugin interfaces and type definitions
- Basic plugin discovery mechanisms
- Plugin example implementations (analytics, telemetry)

### 🔄 In Progress  
- Plugin loading and validation system
- Process-based plugin execution
- Hot swapping infrastructure
- State management and migration

### 🆕 Planned
- Remote plugin loading capabilities
- Plugin marketplace integration
- Advanced dependency management
- Performance optimization and monitoring

## Key Interfaces

### PluginManager
Main interface for all plugin operations including loading, execution, and hot swapping.

### Plugin  
Interface that all plugins must implement for lifecycle management and execution.

### PluginRegistry
Handles plugin discovery, registration, and marketplace integration.

### StateManager
Manages plugin state during hot swapping and version migrations.

## Getting Started

Follow this recommended learning path:

1. **[04_02_03-Plugin Interface Architecture](./04_02_03-plugin_interface_architecture.md)** - **START HERE** to understand the unique architecture
2. **[04_02_04-Plugin Development Guide](./04_02_04-plugin_development_guide.md)** - Practical guide with examples
3. **[04_02_01-Development Guide](./04_02_01-development.md)** - Detailed plugin development fundamentals
4. **[04_02_02_01-Analytics Example](./04_02_02_examples/04_02_02_01_analytics/)** - Study a complete plugin implementation
5. **[04_02_02_02-Telemetry Example](./04_02_02_examples/04_02_02_02_telemetry/)** - Explore operational monitoring patterns

## Development Resources

### Framework Integration
- Read `interfaces.go` to understand the plugin system architecture
- Study `registry/` for plugin discovery patterns
- Review `loader/` for plugin loading implementation
- Examine `executor/` for plugin execution and isolation
- Explore `state/` for hot swapping and state management

### Best Practices
- **Fault Isolation**: Design plugins to fail gracefully without affecting the core
- **Resource Management**: Respect CPU, memory, and I/O limits
- **State Management**: Implement clean state export/import for hot swapping
- **Security**: Follow principle of least privilege for permissions
- **Testing**: Provide comprehensive test coverage

## Integration Points

The Plugin Domain integrates with:

- **[04_01-Runtime Domain](../04_01_runtime/)**: Process orchestration and lifecycle management
- **[04_03-Mesh Domain](../04_03_mesh/)**: Network communication and service discovery
- **[04_04-Resource Domain](../04_04_resources/)**: Resource allocation and monitoring
- **[04_06-Platform Domain](../04_06_platform/)**: SDK and marketplace integration

## Contributing

Priority areas for contribution:

1. **Plugin Loading**: Implement robust plugin discovery and loading mechanisms
2. **Hot Swapping**: Build production-ready hot swapping infrastructure  
3. **State Management**: Create reliable state migration and persistence systems
4. **Isolation**: Enhance plugin isolation and security models
5. **Performance**: Optimize plugin execution and communication
6. **Marketplace**: Develop plugin marketplace integration
7. **Testing**: Create comprehensive plugin testing frameworks

## See Also

- [Blackhole Foundation Document](../../02-blackhole_foundation.md)
- [04-Domains Overview](../README.md)
- [Development Guidelines](../../06-guides/01-development_guidelines.md)

This domain is critical to Blackhole Foundation's value proposition - the ability to extend and modify the system without downtime while maintaining stability and security.