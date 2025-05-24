# Plugin Compliance System

## Overview

The Blackhole plugin system enforces compliance with development guidelines at multiple stages:

1. **Build Time** - Via Makefile and shell script
2. **Package Time** - Before creating .plugin packages  
3. **Load Time** - When loading plugins into the framework

## Compliance Rules

All plugins MUST follow these rules (from `ecosystem/docs/06_guides/06_01-development_guidelines.md`):

### Architecture Rules
1. **Mesh Communication Only** - No direct plugin-to-plugin imports
2. **gRPC Service Definition** - All plugins must define gRPC interfaces
3. **Event-Driven Architecture** - Publish events for state changes
4. **Location Transparency** - Work regardless of execution location
5. **Self-Contained** - Own go.mod, independently buildable

### Code Quality Rules
6. **Structured Logging** - Use zap/logrus, not standard log
7. **Typed Errors** - Custom error types in types/errors.go
8. **Resource Limits** - Declare CPU/memory requirements
9. **Health Checks** - Implement health check endpoints
10. **Directory Structure** - Follow standard layout

## Compliance Checking

### 1. Build-Time Check

Run manually or automatically during build:

```bash
# Check single plugin
cd core/pkg/plugins/myplugin
make check-compliance

# Check all plugins
make plugin-check-compliance
```

The shell script (`core/scripts/check-plugin-compliance.sh`) validates:
- Required files (go.mod, plugin.yaml, README.md, Makefile)
- Directory structure (types/, proto/v1/)
- Typed errors implementation
- gRPC service definitions
- Mesh compliance (mesh client or grpc server)
- No direct plugin dependencies
- Structured logging usage
- Resource declarations
- Test coverage

### 2. Package-Time Check

Automatically runs before packaging:

```bash
make package  # Runs check-compliance first
```

Prevents non-compliant plugins from being packaged.

### 3. Load-Time Check

The plugin loader validates compliance when loading:

```go
// Strict mode - all rules enforced
loader := loader.NewWithOptions(true)

// Normal mode - critical rules only
loader := loader.New()
```

The Go validator (`core/internal/framework/plugins/validator/compliance.go`) checks:
- Plugin manifest completeness
- Resource requirements
- Mesh configuration
- Event patterns
- Dependencies

## Example: Compliant Plugin

The node plugin serves as a reference implementation:

```
core/pkg/plugins/node/
├── main.go                # Direct RPC entry
├── main_mesh.go          # Mesh-compliant entry
├── grpc_server.go        # gRPC service implementation
├── go.mod                # Self-contained module
├── plugin.yaml           # Complete manifest
├── Makefile              # With compliance target
├── README.md             # Documentation
├── proto/
│   └── v1/
│       └── node.proto    # gRPC service definition
├── types/
│   ├── types.go          # Core types
│   └── errors.go         # Typed errors
├── mesh/
│   └── client.go         # Mesh communication
└── [features]/           # Feature packages
```

## Enforcement

### CI/CD Integration

Add to your CI pipeline:

```yaml
- name: Check Plugin Compliance
  run: make plugin-check-compliance
```

### Pre-commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
if git diff --cached --name-only | grep -q "^core/pkg/plugins/"; then
    make plugin-check-compliance || exit 1
fi
```

## Violations and Fixes

### Common Violations

1. **Standard logging**
   ```go
   // Bad
   import "log"
   log.Println("message")
   
   // Good
   import "go.uber.org/zap"
   logger.Info("message")
   ```

2. **Direct plugin imports**
   ```go
   // Bad
   import "core/pkg/plugins/storage"
   
   // Good - use mesh
   client := mesh.ConnectToPlugin("storage")
   ```

3. **Generic errors**
   ```go
   // Bad
   return errors.New("failed")
   
   // Good
   return &types.ConnectionError{
       Operation: "connect",
       Err: err,
   }
   ```

4. **Missing resource limits**
   ```yaml
   # Bad - no resources section
   
   # Good
   resources:
     min_memory: 256M
     max_memory: 1G
     min_cpu: 0.2
     max_cpu: 2.0
   ```

## Benefits

1. **Quality Assurance** - Consistent plugin quality
2. **Security** - Enforced capabilities and limits
3. **Maintainability** - Standard structure across plugins
4. **Debuggability** - Structured logging and typed errors
5. **Scalability** - Mesh-ready from the start

## Future Enhancements

1. **Performance Checks** - Validate resource usage
2. **Security Scanning** - Check for vulnerabilities
3. **API Compatibility** - Verify gRPC backward compatibility
4. **Documentation Coverage** - Ensure adequate docs
5. **Metrics Compliance** - Verify proper instrumentation