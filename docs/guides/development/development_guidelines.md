# Blackhole Project: Development Guidelines

*Last Updated: May 23, 2025*

This document outlines the development guidelines and best practices for the Blackhole project. These guidelines are designed to ensure maintainability, testability, scalability, and consistent collaboration as the project grows.

## Table of Contents

1. [Code Organization](#code-organization)
   - [Directory Structure](#directory-structure)
   - [Package Organization](#package-organization)
   - [Import Management](#import-management)
   - [Cross-Package Communication](#cross-package-communication)
   - [Component Implementation](#component-implementation)
   - [Coding Conventions](#coding-conventions)
   - [Error Handling](#error-handling)
   - [Logging Standards](#logging-standards)
   - [Testing Strategy](#testing-strategy)
   - [Code Review Checklist](#code-review-checklist)

2. [Git Workflow](#git-workflow)
   - [Repository Structure](#repository-structure)
   - [Branching Strategy](#branching-strategy)
   - [Commit Message Conventions](#commit-message-conventions)
   - [Pull Request Workflow](#pull-request-workflow)
   - [Code Review Practices](#code-review-practices)
   - [Versioning Strategy](#versioning-strategy)
   - [Release Process](#release-process)
   - [Handling Hotfixes](#handling-hotfixes)
   - [Git Hooks](#git-hooks)
   - [CI/CD Integration](#cicd-integration)

---

# Code Organization

## Directory Structure

Major components should follow a subdirectory-based organization for clear separation of concerns:

```
# Project Structure

blackhole/
├── cmd/                   # Command line applications
│   └── blackhole/         # Main binary entry point
│
├── internal/              # Private application code
│   ├── [component]/       # A logical component
│   │   ├── types/         # All interface and type definitions
│   │   │   ├── types.go   # Core type definitions
│   │   │   └── errors.go  # Error type definitions
│   │   │
│   │   ├── [component].go # Main struct definition and initialization
│   │   │
│   │   ├── [feature1]/    # Feature-specific functionality
│   │   │   └── feature1.go # Main feature implementation
│   │   │
│   │   └── utils/         # Shared utilities
│   │       └── helpers.go # Helper functions
│   │
│   └── services/          # Service implementations
│       ├── identity/      # Identity service
│       ├── node/          # Node service
│       └── [service]/     # Other services
│
├── pkg/                   # Public packages
│   ├── api/               # Public API clients
│   └── sdk/               # SDK for developers
│
├── bin/                   # Build artifacts (generated)
│   ├── blackhole          # Main binary
│   └── services/          # Service binaries
│       ├── identity/      # Identity service directory
│       │   └── identity   # Identity service executable
│       ├── node/          # Node service directory
│       │   └── node       # Node service executable
│       └── [service]/     # Other service directories
│           └── [service]  # Service executable
│
└── test/                  # All tests
    ├── unit/              # Unit tests
    │   └── [component]/   # Mirrors internal structure
    │       └── [component]_test.go
    │
    ├── integration/       # Integration tests
    │   └── [feature]_test.go
    │
    └── fixtures/          # Shared test fixtures
        └── [fixture].go
```

## Build System Structure

The Blackhole project uses a structured approach to building and organizing executables, ensuring clear separation between the main application and services.

### Build Directory Structure

```
bin/                       # All build artifacts
├── blackhole              # Main application binary
└── services/              # Service binaries (organized by service)
    ├── identity/          # Identity service directory
    │   └── identity       # Identity service executable
    ├── node/              # Node service directory
    │   └── node           # Node service executable
    └── [service]/         # Other service directories
        └── [service]      # Service executable
```

### Build Guidelines

1. **Service Isolation**: Each service executable is placed in its own directory under `bin/services/`
2. **Consistent Naming**: Service directory name matches the service name and executable name
3. **Clean Structure**: Service executables are self-contained within their directories
4. **Main Binary**: The main orchestrator binary remains in the root `bin/` directory

### Makefile Configuration

The project Makefile is configured to automatically create the proper directory structure:

```makefile
# Build individual services
.PHONY: $(SERVICES)
$(SERVICES):
	@echo "Building $@ service..."
	@mkdir -p $(BINARY_DIR)/services/$@
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/services/$@/$@ ./internal/services/$@
```

This ensures that:
- Each service gets its own directory: `bin/services/servicename/`
- The executable is named consistently: `bin/services/servicename/servicename`
- Directory structure is created automatically during build

### Build Commands

- `make build` - Build the main application binary
- `make build-services` - Build all service binaries in their organized structure
- `make identity` - Build specific identity service binary
- `make node` - Build specific node service binary
- `make clean` - Remove all build artifacts

### Service Directory Benefits

1. **Deployment Flexibility**: Each service can be deployed independently with its directory
2. **Configuration Isolation**: Services can have dedicated configuration files in their directories
3. **Asset Management**: Service-specific assets (configs, docs) can be co-located with executables
4. **Clear Organization**: Prevents binary confusion in multi-service environments
5. **Docker Compatibility**: Service directories can be easily packaged into individual containers

## Package Organization

### Rules

1. **Subdirectory = Package**: Each subdirectory is its own package with a focused responsibility
2. **Package Naming**: Use simple, descriptive names that indicate the package's purpose
3. **Package Documentation**: Each package must have a package comment in at least one file
4. **Dependency Direction**: Higher-level packages can import lower-level ones, not vice versa
5. **Minimal Public API**: Export only what's necessary from each package
6. **Unified Workspace**: All services MUST use the main workspace - no separate go.mod files in services
7. **Dependency Management**: All dependencies managed through the root go.mod file

### Guidelines

- Keep packages focused on a single responsibility
- Aim for package cohesion - functions in a package should be related
- Limit package size to maintain clarity (typically 3-7 files per package)
- Use consistent naming conventions across packages
- Document the purpose and usage of each package

### Example Package Comment

```go
// Package types defines the core type definitions and interfaces for the
// storage component. It provides the contract that implementations must follow
// and ensures consistency across the storage subsystem.
package types
```

## Import Management

### Rules

1. **Reduced Import Overhead**: Each package handles its own imports internally
2. **Import Order**: Standard library first, then external packages, then internal packages
3. **No Dot Imports**: Never use dot imports (import . "package")
4. **Package Aliases**: Use meaningful aliases when package names conflict
5. **Minimize External Dependencies**: Limit use of external libraries to essentials

### Example

```go
import (
    // Standard library
    "context"
    "fmt"
    "os"
    "sync"
    
    // External packages
    "go.uber.org/zap"
    
    // Internal packages
    "github.com/handcraftdev/blackhole/internal/storage/types"
    "github.com/handcraftdev/blackhole/internal/storage/utils"
)
```

## Cross-Package Communication

### Rules

1. **Interface-Based**: Communication between packages should be via interfaces defined in types/
2. **Dependency Injection**: Pass dependencies explicitly rather than using global state
3. **Minimal Coupling**: Each package should know as little as possible about others
4. **Avoid Circular Dependencies**: Ensure package dependency graph remains acyclic

### Example

```go
// In types/storage_types.go
type ContentStore interface {
    Store(ctx context.Context, content []byte) (string, error)
    Retrieve(ctx context.Context, id string) ([]byte, error)
}

// In ipfs/store.go
type IPFSStore struct {
    // implementation details
}

func (s *IPFSStore) Store(ctx context.Context, content []byte) (string, error) {
    // implementation
}

// In storage.go
func NewStorage(store types.ContentStore, logger *zap.Logger) *Storage {
    return &Storage{
        store: store,
        logger: logger,
    }
}
```

## Component Implementation

### Rules

1. **Central Controller**: [component].go acts as the central entry point
2. **Package Integration**: Main component integrates functionality from subpackages
3. **Delegate Pattern**: Use delegation to appropriate packages for specific functionality
4. **State Management**: Component maintains shared state used by all packages
5. **Public API**: Component provides the public API for the subsystem

### Example

```go
// In storage.go
type Storage struct {
    // Core state
    config      *types.StorageConfig
    
    // Dependencies
    store       types.ContentStore
    indexer     types.ContentIndexer
    validator   types.ContentValidator
    logger      *zap.Logger
}

func (s *Storage) StoreContent(ctx context.Context, content []byte) (string, error) {
    // Validate content
    if err := s.validator.Validate(content); err != nil {
        return "", fmt.Errorf("content validation failed: %w", err)
    }
    
    // Store content
    id, err := s.store.Store(ctx, content)
    if err != nil {
        return "", fmt.Errorf("content storage failed: %w", err)
    }
    
    // Index content
    if err := s.indexer.Index(ctx, id, content); err != nil {
        s.logger.Warn("Content indexing failed", 
            zap.String("id", id),
            zap.Error(err))
        // Continue despite indexing error
    }
    
    return id, nil
}
```

## Coding Conventions

### Rules

1. **Method Grouping**: Keep methods that operate on the same data together in the same file
2. **Function Size**: Aim for functions under 50 lines for readability
3. **Naming Conventions**: Use descriptive, consistent naming throughout
4. **Comments**: Add comments for non-obvious code and all exported functions
5. **Error Handling**: Handle all errors explicitly, never use _ to ignore errors

### Go-Specific Conventions

1. Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
2. Use Go [Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) as a style guide
3. Run `gofmt -s` on all code before committing
4. Use `golint` and `go vet` to check for issues

### Example

```go
// StoreContent validates, stores, and indexes the provided content.
// It returns a unique identifier that can be used to retrieve the content later.
// If validation fails, no storage operation is performed.
// If indexing fails, the content is still stored but a warning is logged.
func (s *Storage) StoreContent(ctx context.Context, content []byte) (string, error) {
    if len(content) == 0 {
        return "", errors.New("cannot store empty content")
    }
    
    // Validate content
    if err := s.validator.Validate(content); err != nil {
        return "", fmt.Errorf("content validation failed: %w", err)
    }
    
    // Store content with context timeout
    storeCtx, cancel := context.WithTimeout(ctx, s.config.StoreTimeout)
    defer cancel()
    
    id, err := s.store.Store(storeCtx, content)
    if err != nil {
        return "", fmt.Errorf("content storage failed: %w", err)
    }
    
    s.logger.Debug("Content stored successfully", 
        zap.String("id", id), 
        zap.Int("size", len(content)))
    
    return id, nil
}
```

## Error Handling

### Rules

1. **Typed Errors**: Use typed errors for specific error cases
2. **Error Wrapping**: Use fmt.Errorf("context: %w", err) to add context to errors
3. **Error Propagation**: Return errors to the caller when they can't be handled
4. **Graceful Degradation**: When possible, continue operation despite errors
5. **Actionable Messages**: Error messages should guide the user toward resolution

### Example

```go
// In types/errors.go
type StorageError struct {
    ID        string
    Operation string
    Err       error
}

func (e *StorageError) Error() string {
    return fmt.Sprintf("storage %s failed for %s: %v", e.Operation, e.ID, e.Err)
}

func (e *StorageError) Unwrap() error {
    return e.Err
}

// In ipfs/store.go
func (s *IPFSStore) Retrieve(ctx context.Context, id string) ([]byte, error) {
    content, err := s.ipfs.Cat(ctx, id)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            return nil, &types.StorageError{
                ID:        id,
                Operation: "retrieve",
                Err:       fmt.Errorf("timeout exceeded: %w", err),
            }
        }
        return nil, &types.StorageError{
            ID:        id,
            Operation: "retrieve",
            Err:       err,
        }
    }
    return content, nil
}
```

## Logging Standards

### Rules

1. **Structured Logging**: Use structured logging with field names
2. **Log Levels**: Use appropriate log levels (debug, info, warn, error)
3. **Context**: Include relevant context in log messages
4. **No Sensitive Data**: Never log sensitive information
5. **Performance**: Be mindful of logging performance in hot paths

### Log Level Guidelines

- **Debug**: Detailed information useful during development and debugging
- **Info**: General operational information about system behavior
- **Warn**: Potentially harmful situations that might indicate problems
- **Error**: Error events that might still allow the application to continue
- **Fatal**: Very severe events that will lead to application termination

### Example

```go
func (s *Storage) DeleteContent(ctx context.Context, id string) error {
    s.logger.Info("Deleting content", 
        zap.String("id", id),
        zap.String("requestor", ctx.Value(auth.UserKey).(string)))
    
    // Attempt deletion
    if err := s.store.Delete(ctx, id); err != nil {
        s.logger.Error("Content deletion failed",
            zap.String("id", id),
            zap.Error(err))
        return fmt.Errorf("failed to delete content %s: %w", id, err)
    }
    
    s.logger.Info("Content deleted successfully",
        zap.String("id", id))
    
    return nil
}
```

## Testing Strategy

### Rules

1. **Strict Test Separation**: All tests MUST be placed in the dedicated `/test` directory, never alongside source code
2. **Dedicated Test Directories**: Use a clear directory structure for different test types (unit, integration, etc.)
3. **Mirror Directory Structure**: Tests in `/test/unit/` must mirror the internal source structure
4. **Interface Mocking**: Use interfaces from types/ package to create mocks for testing
5. **Table-Driven Tests**: Use table-driven tests for testing multiple scenarios
6. **Test Coverage**: Aim for at least 80% test coverage for each package

### Test Directory Structure

```
blackhole/
├── cmd/                                # Command line applications
│   └── blackhole/                      # Main binary entry point
│
├── internal/                           # Private application code
│   ├── core/                           # Core runtime components
│   │   ├── app/                        # Application component
│   │   │   └── app.go                  # Application implementation
│   │   ├── process/                    # Process management
│   │   │   └── orchestrator.go         # Orchestrator implementation
│   │   └── config/                     # Configuration system
│   │       └── config.go               # Config implementation
│   │
│   └── services/                       # Service implementations
│       ├── identity/                   # Identity service
│       ├── storage/                    # Storage service
│       └── node/                       # Node service
│
├── pkg/                                # Public packages
│   ├── api/                            # Public API clients
│   └── sdk/                            # SDK for developers
│
└── test/                               # All tests strictly here
    ├── unit/                           # Unit tests
    │   ├── core/                       # Mirrors internal structure
    │   │   ├── app/                    
    │   │   │   └── app_test.go         # App unit tests
    │   │   ├── process/                
    │   │   │   └── orchestrator_test.go # Orchestrator unit tests
    │   │   └── config/                 
    │   │       └── config_test.go      # Config unit tests
    │   │
    │   └── services/                   # Service tests
    │       ├── identity/               
    │       │   └── identity_test.go    # Identity unit tests
    │       └── storage/
    │           └── storage_test.go     # Storage unit tests
    │
    ├── integration/                    # Integration tests
    │   ├── app_adapter_test.go         # Cross-component tests
    │   ├── README.md                   # Test documentation
    │   └── test-service/               # Test service fixtures
    │       ├── main.go                 # Test service implementation
    │       └── go.mod                  # Service module definition
    │
    ├── functional/                     # Functional tests
    │   └── api/
    │       └── api_test.go             # API level tests
    │
    ├── performance/                    # Performance tests
    │   └── benchmarks/
    │       └── orchestrator_bench_test.go
    │
    └── fixtures/                       # Shared test fixtures
        ├── configs/                    # Test configurations
        └── data/                       # Test data files
```

### Test Types

1. **Unit Tests**: 
   - Located in the `/test/unit/` directory, mirroring the internal structure
   - Test individual functions and methods in isolation
   - Focus on function/method behavior with mocked dependencies
   - All unit tests must be in the dedicated test directory, not alongside code

2. **Integration Tests**:
   - Located in the `/test/integration/` directory
   - Test interaction between multiple components
   - Often require test service implementations and fixtures
   - May create temporary directories and configurations
   - Should be skippable with `-short` flag for quick runs

3. **Functional Tests**:
   - Located in the `/test/functional/` directory
   - Test complete features from a user perspective
   - Often involve running the full system in a test environment

4. **Performance Tests**:
   - Located in the `/test/performance/` directory
   - Benchmark system performance under load
   - Compare performance metrics across changes

### Example Unit Test

```go
// In test/unit/services/storage/storage_test.go
func TestStorage_StoreContent(t *testing.T) {
    tests := []struct {
        name        string
        content     []byte
        mockStore   func(ctx context.Context, content []byte) (string, error)
        mockValidate func(content []byte) error
        wantID      string
        wantErr     bool
    }{
        {
            name:      "Empty content",
            content:   []byte{},
            wantErr:   true,
        },
        // Additional test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mocks
            mockStore := &testing.MockContentStore{
                StoreFunc: tt.mockStore,
            }
            mockValidator := &testing.MockContentValidator{
                ValidateFunc: tt.mockValidate,
            }
            
            // Create storage with mocks
            s := NewStorage(mockStore, mockValidator, zap.NewNop())
            
            // Execute test
            gotID, err := s.StoreContent(context.Background(), tt.content)
            
            // Check results
            if (err != nil) != tt.wantErr {
                t.Errorf("StoreContent() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if gotID != tt.wantID {
                t.Errorf("StoreContent() gotID = %v, want %v", gotID, tt.wantID)
            }
        })
    }
}
```

### Example Integration Test

```go
// In test/integration/app_adapter_test.go
func TestAppAdapter_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration tests in short mode")
    }

    // Setup test environment
    tempDir := setupTestDir(t)
    testServiceBin := buildTestService(t, tempDir)
    testConfig := createTestConfig(t, tempDir, testServiceBin)
    
    // Create the application to test
    app := createTestApp(t, configManager)
    
    // Test service lifecycle
    t.Run("ServiceLifecycle", func(t *testing.T) {
        // Initialize app
        err := app.Initialize()
        require.NoError(t, err)
        
        // Start service
        err = app.StartService("test-service")
        require.NoError(t, err)
        
        // Verify running state
        info, err := app.GetServiceInfo("test-service")
        require.NoError(t, err)
        assert.True(t, info.Running)
        
        // Stop service
        err = app.StopService("test-service")
        require.NoError(t, err)
        
        // Verify stopped state
        info, err = app.GetServiceInfo("test-service")
        require.NoError(t, err)
        assert.False(t, info.Running)
    })
}
```

### Test Execution Commands

The Makefile provides several commands for running tests with different scopes:

```makefile
# Run tests (excluding integration tests)
.PHONY: test
test:
	@echo "Running tests (excluding integration)..."
	$(GO) test -v -short ./...

# Run only integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	$(GO) test -v ./test/integration/...

# Run all tests including integration
.PHONY: test-all
test-all:
	@echo "Running all tests..."
	$(GO) test -v ./...

# Run tests with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	$(GO) test -race -v -short ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out -short ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
```

Usage:
- `make test` - Run just unit tests (fast, for frequent usage)
- `make test-integration` - Run only integration tests
- `make test-all` - Run all tests including integration tests (thorough, but slower)
- `make test-race` - Run tests with race detection to find concurrency issues
- `make test-coverage` - Generate test coverage report

## Code Review Checklist

Before submitting code for review, ensure:

1. **Package Placement**: Is the new code in the appropriate package?
2. **Package API**: Is the package exposing a clean, minimal API?
3. **Cross-Package Dependencies**: Are dependencies between packages clean and explicit?
4. **Error Handling**: Are errors properly created, wrapped, and handled?
5. **Testing**: Are there comprehensive tests for the new functionality?
6. **Documentation**: Is the code properly documented with comments?
7. **Interface Conformance**: Does the implementation properly implement interfaces?
8. **Logging**: Is appropriate logging in place with the correct levels?
9. **Performance**: Are there any performance concerns with the implementation?
10. **Security**: Are there any security implications that need to be addressed?

## Dashboard and Web Development Guidelines

### Rules

1. **Unified Service Monitoring**: Dashboard must show all services in a single table with consistent columns
2. **Mesh Status Tracking**: Include mesh registration status for all services to monitor connectivity
3. **Real-time Updates**: Services should update in real-time, core components may have delayed updates
4. **Responsive Design**: Dashboard must work on various screen sizes and devices
5. **Clean HTML Structure**: Use semantic HTML with proper accessibility considerations

### Dashboard Requirements

- **Service Table Columns**: Service, Status, Mesh, Port, PID, Uptime, CPU, Memory, Actions
- **Status Indicators**: Use color-coded badges for service status (running, stopped, error)
- **Mesh Connectivity**: Track and display service registration with mesh router
- **Action Buttons**: Provide start, stop, restart actions for each service
- **Core Components**: Include daemon, orchestrator, and mesh router in the services table
- **Storage Removal**: Storage service has been removed - do not include in dashboards

### Implementation Standards

- Use vanilla JavaScript for functionality - avoid unnecessary framework dependencies
- Implement WebSocket connections for real-time updates where appropriate
- Follow consistent naming conventions for DOM elements and CSS classes
- Ensure proper error handling for failed service operations
- Maintain clean separation between HTML structure, CSS styling, and JavaScript functionality

---

# Git Workflow

## Repository Structure

The Blackhole repository follows a structured branching model based on the GitFlow workflow, adapted for our subprocess architecture.

### Main Branches

- **`main`**: The production-ready branch containing stable releases
- **`develop`**: The integration branch for ongoing development
- **`release/*`**: Feature-complete releases undergoing final testing
- **`hotfix/*`**: Emergency fixes for production issues

### Service-Specific Feature Branches

Given Blackhole's subprocess architecture, we organize feature branches by service:

```
feature/identity/did-resolution
feature/storage/ipfs-integration
feature/node/peer-discovery
feature/ledger/transaction-validation
feature/analytics/privacy-preserving-metrics
```

## Branching Strategy

We use a modified GitFlow workflow optimized for our subprocess architecture:

```
        hotfix/1.0.1  ------>|
                             v
main     ---------------o----o-------------------o---->
           ^                                     ^
           |                                     |
release    |    release/1.0  o---->o----------->|
           ^                 ^                   
           |                 |                   
develop    o---o---o---o---o-o---o---o---o---o---o---->
                 ^       ^     ^         ^
                 |       |     |         |
feature    -->o--o-->    -->o--o-->  -->o--o-->
```

### Workflow Rules

1. **Feature Development**:
   - Create feature branches from `develop`
   - Prefix with service name: `feature/service-name/feature-description`
   - Merge back to `develop` via pull request

2. **Release Preparation**:
   - Branch `release/x.y.z` from `develop` when ready
   - Only bug fixes and documentation on release branches
   - Merge to both `main` and `develop` when complete

3. **Hotfix Process**:
   - Branch `hotfix/x.y.z` from `main` for urgent production fixes
   - Merge to both `main` and `develop` when complete

4. **Main Branch Protection**:
   - Direct commits to `main` and `develop` are prohibited
   - All changes must go through pull requests with reviews

## Commit Message Conventions

We use conventional commits to maintain a clean and informative git history.

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### AI-Generated Code and Signature Policy

**CRITICAL RULE: No AI signatures or co-authoring credits in commits**

- **NEVER** include AI assistant signatures like "Generated with Claude Code" or "Co-Authored-By: Claude"
- **NEVER** add any references to AI assistance in commit messages, PR descriptions, or code comments
- All commits must appear as if written entirely by human developers
- Code quality and standards must be maintained regardless of generation method
- This policy applies to all AI tools including Claude, GitHub Copilot, ChatGPT, etc.

**Rationale**: Maintains professional development practices and clean git history without implementation details.

### Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation changes
- **style**: Formatting changes (whitespace, formatting, etc.)
- **refactor**: Code changes that neither fix bugs nor add features
- **perf**: Performance improvements
- **test**: Test-related changes
- **chore**: Maintenance tasks, build changes, etc.

### Scopes

For Blackhole, scopes align with our service architecture:

- **core**: Core orchestration layer
- **mesh**: Service mesh components
- **identity**: Identity service
- **storage**: Storage service
- **node**: Node service
- **ledger**: Ledger service
- **indexer**: Indexer service
- **social**: Social service
- **analytics**: Analytics service
- **telemetry**: Telemetry service
- **wallet**: Wallet service
- **build**: Build system
- **docs**: Documentation
- **config**: Configuration

### Examples

```
feat(identity): implement DID resolution endpoint

Added HTTP and gRPC handlers for DID resolution according to the W3C spec.
This enables clients to resolve DIDs through our identity service.

Resolves: #123
```

```
fix(storage): correct IPFS connection retry logic

Fixed an issue where IPFS connection would fail permanently after timeout
instead of implementing the exponential backoff strategy.

Fixes: #456
```

## Pull Request Workflow

Pull requests are the primary method for integrating changes into the codebase.

### PR Creation Guidelines

1. **Naming**: Use prefix + brief description:
   - `[FEATURE] Add DID resolution endpoint`
   - `[FIX] Correct IPFS connection retry logic`
   - `[DOCS] Update storage service architecture`
   - `[REFACTOR] Improve process manager performance`

2. **Description Template**:
   ```markdown
   ## Description
   Brief description of changes and motivation

   ## Type of change
   - [ ] Bug fix
   - [ ] New feature
   - [ ] Breaking change
   - [ ] Documentation update

   ## Service Impact
   - Primary service: identity
   - Related services: storage, node

   ## Testing
   - [ ] Unit tests added/updated
   - [ ] Integration tests added/updated
   - [ ] Manual testing completed

   ## Checklist
   - [ ] My code follows the style guidelines
   - [ ] I have performed a self-review
   - [ ] I have commented my code appropriately
   - [ ] My changes generate no new warnings
   - [ ] I have updated the documentation

   Closes #789
   ```

3. **Size Guidelines**:
   - Keep PRs focused on a single logical change
   - Target under 500 lines of changes where possible
   - Split large features into smaller, sequential PRs

### Review Process

1. **Required Approvals**: Minimum 1 approval, 2 for core components
2. **Code Owners**: Automatic assignment based on service
3. **CI Checks**: All checks must pass before merging
4. **Review SLA**: Reviews completed within 24 hours

### Merging Strategy

1. **Squash and Merge** for feature branches to maintain clean history
2. **Merge Commit** for release and hotfix branches to preserve history

## Code Review Practices

Effective code reviews ensure quality and knowledge sharing.

### Reviewer Guidelines

1. **Focus Areas**:
   - Code correctness
   - Service isolation boundaries
   - RPC communication patterns
   - Security implications
   - Resource management
   - Testing coverage
   - Documentation completeness

2. **Review Checklist**:
   - Does the code follow our design principles?
   - Are subprocess boundaries properly maintained?
   - Is error handling comprehensive?
   - Are there sufficient tests?
   - Is the code optimized for our use case?
   - Are there any security implications?
   - Is the documentation updated?

3. **Constructive Feedback**:
   - Be specific and actionable
   - Reference patterns and documentation
   - Suggest alternatives
   - Distinguish between required changes and suggestions

### Author Responsibilities

1. **PR Preparation**:
   - Self-review before submission
   - Run full test suite locally
   - Document design decisions
   - Explain complex or non-obvious code

2. **Responding to Feedback**:
   - Address all comments
   - Explain changes made
   - Request re-review when ready

## Versioning Strategy

We use Semantic Versioning (SemVer) for the Blackhole project.

### Version Format

```
MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
```

- **MAJOR**: Incompatible API changes
- **MINOR**: Backward-compatible new functionality
- **PATCH**: Backward-compatible bug fixes
- **PRERELEASE**: Alpha, beta, rc designations (e.g., 1.0.0-beta.1)
- **BUILD**: Build metadata (e.g., 1.0.0+20230501)

### Version Bumping Guidelines

1. **MAJOR Version**: Incremented for:
   - Changes to service RPC interfaces
   - Breaking changes to client APIs
   - Significant architectural changes

2. **MINOR Version**: Incremented for:
   - New services or features
   - Non-breaking API additions
   - Substantial improvements to existing functionality

3. **PATCH Version**: Incremented for:
   - Bug fixes
   - Small optimizations
   - Documentation improvements

### Example Version Progression

```
1.0.0-alpha.1 -> 1.0.0-alpha.2 -> 1.0.0-beta.1 -> 1.0.0-rc.1 -> 1.0.0 -> 1.0.1 -> 1.1.0 -> 2.0.0
```

## Release Process

The release process ensures stable and predictable software delivery.

### Release Preparation

1. **Create Release Branch**:
   ```bash
   git checkout develop
   git pull
   git checkout -b release/1.0.0
   ```

2. **Version Bump**:
   - Update version in appropriate files:
     - `cmd/blackhole/version.go`
     - `configs/version.yaml`
     - Documentation references

3. **Release Testing**:
   - Run comprehensive test suite
   - Deploy to staging environment
   - Perform integration testing
   - Validate service interoperability

4. **Release Notes**:
   - Compile changelog from commits
   - Document upgrade steps
   - Note breaking changes
   - Highlight new features

### Release Finalization

1. **Final Pull Request**:
   - Create PR from `release/1.0.0` to `main`
   - Include finalized release notes
   - Require approvals from senior team members

2. **Merge to Main**:
   ```bash
   # After PR approval
   git checkout main
   git pull
   git merge --no-ff release/1.0.0
   git tag -a v1.0.0 -m "Release 1.0.0"
   git push origin main --tags
   ```

3. **Merge Back to Develop**:
   ```bash
   git checkout develop
   git pull
   git merge --no-ff release/1.0.0
   git push origin develop
   ```

4. **Cleanup**:
   ```bash
   git branch -d release/1.0.0
   ```

### Release Artifacts

1. **Binary Releases**:
   - Cross-platform binaries (Linux, macOS, Windows)
   - Docker images
   - Checksums and signatures

2. **Documentation**:
   - Updated user guides
   - API documentation
   - Release notes

3. **Announcements**:
   - GitHub releases
   - Project website
   - Community channels

## Handling Hotfixes

Hotfixes address critical issues in production releases.

### Hotfix Workflow

1. **Create Hotfix Branch**:
   ```bash
   git checkout main
   git pull
   git checkout -b hotfix/1.0.1
   ```

2. **Implement Fix**:
   - Keep changes minimal and focused
   - Add specific tests for the issue
   - Update version numbers

3. **Review and Testing**:
   - Thorough code review
   - Comprehensive testing
   - Verification that the fix addresses the issue

4. **Merge to Main**:
   ```bash
   git checkout main
   git pull
   git merge --no-ff hotfix/1.0.1
   git tag -a v1.0.1 -m "Hotfix 1.0.1"
   git push origin main --tags
   ```

5. **Merge to Develop**:
   ```bash
   git checkout develop
   git pull
   git merge --no-ff hotfix/1.0.1
   git push origin develop
   ```

6. **Cleanup**:
   ```bash
   git branch -d hotfix/1.0.1
   ```

### Emergency Release Process

For critical issues requiring immediate attention:

1. **Expedited Review**: Shorter but mandatory review cycle
2. **Emergency Testing**: Focused test suite execution
3. **Staged Rollout**: Phased deployment to detect issues
4. **Post-mortem**: Document root cause and prevention

## Git Hooks

Git hooks automate quality checks and ensure consistency.

### Pre-commit Hooks

1. **Linting**:
   - Go: `golangci-lint`
   - JavaScript/TypeScript: `eslint`
   - YAML: `yamllint`

2. **Formatting**:
   - Go: `gofmt` or `goimports`
   - JavaScript/TypeScript: `prettier`
   - YAML/JSON: `prettier`

3. **License Headers**:
   - Verify copyright notices
   - Check license headers

4. **Commit Message Validation**:
   - Ensure conventional commit format
   - Check for scope validity
   - Verify references to issues

### Setup Instructions

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install
```

### Sample `.pre-commit-config.yaml`

```yaml
repos:
-   repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.4.0
    hooks:
    -   id: trailing-whitespace
    -   id: end-of-file-fixer
    -   id: check-yaml
    -   id: check-added-large-files

-   repo: https://github.com/golangci/golangci-lint
    rev: v1.51.2
    hooks:
    -   id: golangci-lint

-   repo: https://github.com/commitizen-tools/commitizen
    rev: v2.42.1
    hooks:
    -   id: commitizen
        stages: [commit-msg]
```

## CI/CD Integration

Our CI/CD pipeline is tightly integrated with our Git workflow.

### CI Pipeline Components

1. **Build Verification**:
   - Compile all services
   - Verify module dependencies
   - Check for build warnings

2. **Test Execution**:
   - Unit tests
   - Integration tests
   - End-to-end tests

3. **Code Quality**:
   - Static analysis
   - Test coverage reporting
   - Duplicate code detection

4. **Security Scanning**:
   - Dependency vulnerability checking
   - SAST (Static Application Security Testing)
   - License compliance

### CD Pipeline Components

1. **Automated Deployments**:
   - Development environment: Every merge to `develop`
   - Staging environment: Every merge to `release/*`
   - Production environment: Every merge to `main` (with approval)

2. **Deployment Verification**:
   - Smoke tests
   - Health checks
   - Performance validation

3. **Rollback Capability**:
   - Automated rollback for failed deployments
   - Version pinning
   - Deployment history

### GitHub Actions Example

```yaml
name: Blackhole CI

on:
  push:
    branches: [main, develop, 'release/*', 'hotfix/*']
  pull_request:
    branches: [main, develop, 'release/*']

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.22'
      - name: Build
        run: make build
      - name: Unit Tests
        run: make test
      - name: Integration Tests
        run: make test-integration
      - name: Lint
        run: make lint

  service-tests:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: [identity, storage, node, ledger, indexer, social, analytics, telemetry, wallet]
    steps:
      - uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.22'
      - name: Test ${{ matrix.service }} service
        run: cd internal/services/${{ matrix.service }} && go test -v ./...

  deploy-dev:
    if: github.ref == 'refs/heads/develop'
    needs: [build, service-tests]
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to development
        run: |
          echo "Deploying to development environment"
          # Deployment steps

  deploy-staging:
    if: startsWith(github.ref, 'refs/heads/release/')
    needs: [build, service-tests]
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to staging
        run: |
          echo "Deploying to staging environment"
          # Deployment steps

  deploy-production:
    if: github.ref == 'refs/heads/main'
    needs: [build, service-tests]
    runs-on: ubuntu-latest
    environment: production
    steps:
      - name: Deploy to production
        run: |
          echo "Deploying to production environment"
          # Deployment steps with approval
```

---

## Conclusion

Following these development guidelines ensures consistent, high-quality code and collaboration across the Blackhole project. These practices are designed to support our subprocess architecture, with special consideration for service boundaries, RPC interfaces, and the unique requirements of distributed systems.

The combination of proper code organization and a well-defined git workflow creates a foundation for efficient, reliable, and maintainable software development as our project evolves.

---

**Document Status**: APPROVED  
**Last Updated**: May 21, 2025  
**Author**: Blackhole Development Team