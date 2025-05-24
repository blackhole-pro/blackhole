# Blackhole Development and Deployment Design

*Created: May 20, 2025*

This document outlines the comprehensive development and deployment design for the Blackhole platform, providing guidance on development workflows, build processes, deployment patterns, and implementation strategies.

## 1. Project Overview

Blackhole is a distributed content sharing platform built on a subprocess architecture where services run as independent OS processes managed by a single binary orchestrator. The platform leverages decentralized technologies like IPFS/Filecoin for storage, DIDs for identity, and ActivityPub for social federation.

## 2. Development Workflow

### 2.1 Development Environment Setup

1. **Prerequisites**:
   - Go 1.21+ (for core development)
   - Node.js 18+ (for client libraries)
   - Git (for version control)
   - Make (for build automation)
   - Docker (optional, for containerized testing)

2. **Directory Structure**:
   - The project follows a standard Go project layout
   - Core components are in `internal/core`
   - Service implementations are in `internal/services/*`
   - Public APIs and types are in `pkg/*`
   - Client libraries are in `client-libs/*`

3. **Development Tools**:
   - Use `make` for building and testing
   - Use `make dev` for hot-reload development
   - Use `go mod` for dependency management
   - Use Git for version control

### 2.2 Build System

The project uses a Makefile with the following primary targets:

1. **Build Main Binary**:
   ```bash
   make build
   ```
   This builds the main Blackhole binary in the `bin` directory.

2. **Build Services**:
   ```bash
   make build-services
   ```
   This builds all service binaries.

3. **Build Specific Service**:
   ```bash
   make identity  # or any other service name
   ```
   This builds a specific service binary.

4. **Run Tests**:
   ```bash
   make test          # Run all tests
   make test-race     # Run tests with race detection
   make test-coverage # Run tests with coverage report
   ```

5. **Development Mode**:
   ```bash
   make dev
   ```
   This runs the application with hot reload for faster development.

6. **Cross-Compilation**:
   ```bash
   make build-all
   ```
   This builds binaries for multiple platforms (Linux, macOS, Windows).

### 2.3 Dependency Management

The project uses Go modules for dependency management:

1. **Workspace Structure**:
   - Main module at the root
   - Service modules in `internal/services/*`
   - Go workspace (go.work) for multi-module development

2. **Managing Dependencies**:
   ```bash
   make deps        # Install dependencies
   make update-deps # Update dependencies
   ```

3. **Key Dependencies**:
   - gRPC/protobuf: For RPC communication
   - libp2p: For P2P networking
   - IPFS/Filecoin SDKs: For content storage
   - zap/logrus: For logging
   - cobra/cli: For CLI interfaces
   - viper: For configuration

## 3. Development Strategy

Based on the implementation phases defined in the documentation, development should proceed as follows:

### 3.1 Phase 1: Core Foundation (Current Focus)

1. **Process Orchestrator Implementation**:
   - Implement the interface-driven design as defined in `docs/implementation/core/03_process_orchestrator_implementation_simplified.md`
   - Create core interfaces: ProcessManager, Command, ServiceState
   - Implement process spawning and management
   - Add state pattern for process lifecycle
   - Implement process output handling with buffered line processing
   - Add proper error handling with typed errors

2. **Configuration System Implementation**:
   - Implement hierarchical configuration
   - Add environment variable overrides
   - Implement file watching for dynamic updates
   - Create configuration validation
   - Add change notifications for configuration updates

3. **CLI Interface**:
   - Create commands for managing services (start, stop, restart)
   - Add logging and status commands
   - Implement configuration commands

### 3.2 Next Phases (After Phase 1)

1. **Service Mesh Implementation** (Phase 2):
   - Implement Router for service discovery
   - Create EventBus for pub/sub communication
   - Add Middleware for cross-cutting concerns

2. **Security Implementation** (Phase 3):
   - Implement process isolation
   - Add mTLS for service communication
   - Set up security policies

3. **Resource Management** (Phase 4):
   - Implement CPU, memory, and I/O limits
   - Add monitoring for resource usage

4. **Service Integration** (Phase 5):
   - Implement service dependencies
   - Create coordinated startup/shutdown

## 4. Testing Strategy

The project uses multiple levels of testing to ensure quality:

1. **Unit Tests**:
   - Test individual components with mocks
   - Run with `go test ./...`
   - Use interface-based design for better testability

2. **Integration Tests**:
   - Test interactions between components
   - Run with `go test -tags=integration ./...`

3. **System Tests**:
   - End-to-end testing of entire system
   - Test process management, failures, and recovery

4. **Benchmark Tests**:
   - Measure performance under load
   - Identify bottlenecks

## 5. Deployment Patterns

The subprocess architecture supports multiple deployment patterns:

### 5.1 Single Host Deployment

Run all services on a single machine:

```bash
# Install binary
cp bin/blackhole /usr/local/bin/

# Create data directories
mkdir -p /var/lib/blackhole
mkdir -p /var/run/blackhole

# Create configuration
cp configs/blackhole.yaml /etc/blackhole/

# Start all services
blackhole start --all
```

### 5.2 Multi-Host Deployment

Distribute services across multiple hosts:

```bash
# Host 1: Core services
blackhole start --services=identity,ledger --bind=0.0.0.0:9000

# Host 2: Storage services
blackhole start --services=storage,indexer --connect=host1:9000
```

### 5.3 Container Deployment

Run in Docker or Kubernetes:

```bash
# Build Docker image
make docker-build

# Run container
docker run -p 8080:8080 -p 9090:9090 -v /path/to/data:/var/lib/blackhole blackhole:latest
```

For Kubernetes:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: blackhole
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: blackhole
        image: blackhole:latest
        command: ["blackhole", "start", "--all"]
        securityContext:
          capabilities:
            add: ["SYS_RESOURCE"]
```

## 6. Service Configuration

Services are configured via YAML:

```yaml
# /etc/blackhole/blackhole.yaml
orchestrator:
  socket_dir: /var/run/blackhole
  log_level: info
  auto_restart: true
  shutdown_timeout: 30

services:
  identity:
    enabled: true
    binary_path: /usr/local/bin/identity
    data_dir: /var/lib/blackhole/identity
    resources:
      cpu: 200      # 2 CPU cores
      memory: 1024  # 1GB
      io_weight: 500
    environment:
      DATABASE_URL: postgres://user:pass@localhost/identity

  storage:
    enabled: true
    binary_path: /usr/local/bin/storage
    data_dir: /var/lib/blackhole/storage
    resources:
      cpu: 100      # 1 CPU core
      memory: 2048  # 2GB
      io_weight: 900
    environment:
      IPFS_API: http://localhost:5001
```

## 7. Implementation Plan

### 7.1 Phase 1 Implementation (Core Components)

#### Process Orchestrator Implementation

1. **Core Interfaces and Types**:
   - Define `ProcessState`, `ProcessError`, `ProcessManager` interfaces
   - Create `ProcessCmd` and `Process` abstractions
   - Implement default implementations

2. **Orchestrator Implementation**:
   - Implement `Orchestrator` struct with state management
   - Create functional options for configuration
   - Add service discovery
   - Implement process spawn/stop logic

3. **Process Management**:
   - Add process supervision
   - Implement exponential backoff restart
   - Create health checking
   - Add process output handling with buffered line processing

4. **Error Handling and Lifecycle**:
   - Implement typed errors with proper context
   - Add context-based cancellation
   - Implement graceful shutdown
   - Create state transition validation

#### Configuration System Implementation

1. **Core Configuration Types**:
   - Define `Config`, `OrchestratorConfig`, `ServiceConfig` structs
   - Create validation and defaults
   - Implement helpers and utilities

2. **Configuration Loading**:
   - Add file-based loading with viper
   - Implement environment variable overrides
   - Add watch functionality for dynamic updates
   - Create change notification system

3. **Configuration Integration**:
   - Connect with Orchestrator
   - Add distributed configuration for services
   - Implement configuration updates

#### CLI Interface Implementation

1. **Command Structure**:
   - Create main command with cobra
   - Add service management commands (start, stop, restart)
   - Implement status and logging commands

2. **Service Operations**:
   - Add start command with service selection
   - Implement stop and restart logic
   - Create service status reporting

3. **Utility Commands**:
   - Add configuration commands
   - Implement log viewing
   - Create debugging utilities

### 7.2 Testing Implementation

1. **Unit Tests**:
   - Create mocks for interfaces
   - Implement test utilities
   - Add comprehensive test cases

2. **Integration Tests**:
   - Create test services
   - Implement process isolation tests
   - Add configuration tests

3. **Testing Infrastructure**:
   - Implement test fixtures
   - Create helper utilities
   - Add CI/CD configuration

## 8. Development Guidelines

1. **Code Structure**:
   - Follow interface-driven design
   - Use proper error handling
   - Implement graceful shutdown
   - Write comprehensive tests

2. **Coding Standards**:
   - Follow Go best practices
   - Add comments for exported types and functions
   - Implement proper logging
   - Use context for cancellation

3. **Testing Requirements**:
   - Write tests for all new code
   - Include unit and integration tests
   - Test error conditions and edge cases
   - Test with race detection

4. **Documentation**:
   - Update implementation documentation as code evolves
   - Document interfaces and types
   - Create examples for common usage
   - Keep CURRENT_STATUS.md updated

## 9. Next Steps

1. **Process Orchestrator Implementation**:
   - Create the interface-driven design
   - Implement process spawning and state management
   - Add proper error handling and context-based cancellation
   - Implement process output handling and restart logic

2. **Configuration System Enhancement**:
   - Create hierarchical configuration structures
   - Implement loading and validation
   - Add environment variable overrides
   - Implement file watching and change notifications

3. **Create Basic Service Structure**:
   - Implement the service interface
   - Create basic lifecycle management (start, stop)
   - Add health checking and status reporting

4. **Develop CLI Interface**:
   - Create the main command structure
   - Implement service management commands
   - Add status and logging commands

## 10. Conclusion

This development and deployment design provides a comprehensive guide for building and operating the Blackhole platform. By following this design, developers can create a robust, maintainable system that leverages the subprocess architecture to provide true service isolation while maintaining operational simplicity.

The focus on interface-driven design, proper state management, and comprehensive testing will ensure that the platform is resilient, extensible, and maintainable over time.