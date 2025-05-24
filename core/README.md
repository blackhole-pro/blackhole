# Blackhole Foundation Core

The core technical implementation of the Blackhole distributed computing framework, organized following Drupal's architectural principles of modularity, extensibility, and community-driven development.

## Core Architecture

### [framework/](./framework/) - Framework Layer
**The foundational distributed computing framework**
- Plugin management and lifecycle
- Mesh networking and service discovery
- Resource management and scheduling
- Economics and monetization engine
- Security and access control

### [runtime/](./runtime/) - Runtime Layer
**Process orchestration and system runtime**
- Process lifecycle management
- Configuration and health monitoring
- Service supervision and recovery
- Resource isolation and limits
- Distributed event system

### [plugins/](./plugins/) - Plugin System
**Plugin architecture and core plugins**
- Plugin discovery and loading
- Hot-reload capabilities
- State migration and persistence
- Language-agnostic execution
- Security sandboxing

### [platform/](./platform/) - Platform Services
**Platform-level services and tools**
- Developer SDK and tools
- Marketplace infrastructure
- Documentation generation
- Community features
- Analytics and monitoring

## Design Principles

Following Drupal's architectural philosophy:

### 1. **Modularity** (like Drupal modules)
- Everything is a plugin by default
- Clear separation of concerns
- Composable architecture
- Independent versioning

### 2. **Extensibility** (like Drupal hooks)
- Plugin extension points
- Event-driven architecture
- Configurable workflows
- Custom implementations

### 3. **Community-First** (like Drupal contrib)
- Open plugin marketplace
- Community-driven development
- Shared ownership model
- Collaborative governance

### 4. **Backwards Compatibility** (like Drupal updates)
- Stable plugin APIs
- Migration pathways
- Version compatibility
- Graceful degradation

## Core vs Contrib

Similar to Drupal's core/contrib distinction:

### **Core** (this repository)
- Essential framework components
- Stable, battle-tested features
- Long-term support guarantee
- Rigorous testing and review

### **Contrib** (ecosystem plugins)
- Community-contributed plugins
- Experimental features
- Rapid innovation space
- Market-driven development

## Technical Stack

- **Language**: Go (for performance and concurrency)
- **Communication**: gRPC (for type safety and performance)
- **Discovery**: mDNS/Consul (for service discovery)
- **Packaging**: OCI containers (for plugin distribution)
- **Orchestration**: Custom (for plugin-native design)

## Development Workflow

1. **Core Development**: Framework improvements and stable features
2. **Plugin Development**: Community contributions and extensions
3. **Platform Development**: Tools and ecosystem services
4. **Integration Testing**: End-to-end validation
5. **Release Management**: Coordinated releases with compatibility

## Getting Started

### Framework Developers
```bash
# Clone the repository
git clone https://github.com/blackhole-foundation/core
cd core

# Setup development environment
make setup-dev

# Run tests
make test

# Build framework
make build
```

### Plugin Developers
- Start with [Plugin Development Guide](../docs/06_guides/06_02-plugin_development.md)
- Use [Plugin SDK](../products/foundation-core/sdk/)
- Join [Developer Community](../foundation/community/README.md)

## Contribution Guidelines

- [Development Guidelines](../docs/06_guides/06_01-development_guidelines.md)
- [Code of Conduct](../foundation/governance/code-of-conduct.md)
- [Security Policy](../foundation/governance/security-policy.md)
- [Release Process](../foundation/governance/release-process.md)