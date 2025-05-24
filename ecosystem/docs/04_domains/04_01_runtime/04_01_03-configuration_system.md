# 01_03: Configuration System

## Overview

The Configuration System provides centralized, hierarchical configuration management for the Blackhole Framework, supporting runtime configuration updates, environment-specific overrides, and service-specific settings.

## Configuration Architecture

### Configuration Hierarchy

```
Global Framework Config (blackhole.yaml)
├── Runtime Configuration
│   ├── Process Orchestrator Settings
│   ├── Lifecycle Management Settings
│   └── Health Monitoring Settings
├── Domain Configuration
│   ├── Plugin Management Settings
│   ├── Mesh Networking Settings
│   ├── Resource Management Settings
│   ├── Economics Settings
│   └── Platform Settings
└── Service Configuration
    ├── Identity Service Settings
    ├── Storage Service Settings
    ├── Node Service Settings
    └── [Additional Services...]
```

### Configuration Sources

1. **Default Configuration**: Built-in framework defaults
2. **Configuration Files**: YAML, JSON, TOML files
3. **Environment Variables**: Runtime environment overrides
4. **Command Line Arguments**: CLI parameter overrides
5. **Configuration Server**: External configuration service (optional)

### Priority Order

Configuration values are resolved in priority order (highest to lowest):

1. Command line arguments
2. Environment variables
3. Configuration files
4. Default values

## Configuration Structure

### Main Configuration File

```yaml
# blackhole.yaml - Main framework configuration
framework:
  name: "blackhole"
  version: "1.0.0"
  environment: "development"  # development, staging, production
  log_level: "info"           # debug, info, warn, error

runtime:
  orchestrator:
    max_processes: 50
    restart_policy:
      max_attempts: 5
      backoff_multiplier: 2
      max_backoff: 60s
    resource_limits:
      default_cpu_percent: 10
      default_memory_mb: 512

  lifecycle:
    startup_timeout: 60s
    shutdown_timeout: 30s
    health_check_interval: 10s

  health:
    check_interval: 30s
    timeout: 10s
    failure_threshold: 3

domains:
  plugins:
    registry_path: "./plugins"
    hot_reload: true
    max_load_time: 30s

  mesh:
    router_port: 8080
    discovery_port: 8081
    enable_tls: true

  resources:
    scheduler_interval: 5s
    optimization_enabled: true

  economics:
    billing_enabled: false
    currency: "USD"

  platform:
    sdk_enabled: true
    marketplace_enabled: false

services:
  identity:
    enabled: true
    port: 9001
    config_file: "./configs/identity.yaml"
    
  storage:
    enabled: true
    port: 9002
    config_file: "./configs/storage.yaml"
    dependencies: ["identity"]
    
  node:
    enabled: true
    port: 9003
    config_file: "./configs/node.yaml"
    dependencies: ["identity", "storage"]
```

### Service-Specific Configuration

Services can have their own configuration files:

```yaml
# configs/identity.yaml - Identity service configuration
identity:
  did:
    method: "blackhole"
    network: "mainnet"
    resolver_url: "https://resolver.blackhole.dev"
    
  registry:
    storage_type: "ipfs"
    ipfs_gateway: "https://ipfs.blackhole.dev"
    
  authentication:
    jwt_secret: "${JWT_SECRET}"
    token_expiry: "24h"
    
  database:
    type: "badger"
    path: "./data/identity"
```

## Configuration Management

### Loading Configuration

The configuration system loads configuration in the following order:

1. **Load Defaults**: Initialize with built-in defaults
2. **Load Files**: Merge configuration files
3. **Apply Environment**: Override with environment variables
4. **Apply CLI**: Override with command line arguments
5. **Validate**: Validate final configuration
6. **Distribute**: Send to relevant services

### Environment Variable Mapping

Environment variables follow a hierarchical naming convention:

```bash
# Framework level
export BLACKHOLE_FRAMEWORK_ENVIRONMENT=production
export BLACKHOLE_FRAMEWORK_LOG_LEVEL=warn

# Runtime level
export BLACKHOLE_RUNTIME_ORCHESTRATOR_MAX_PROCESSES=100
export BLACKHOLE_RUNTIME_LIFECYCLE_STARTUP_TIMEOUT=120s

# Service level
export BLACKHOLE_SERVICES_IDENTITY_PORT=9001
export BLACKHOLE_SERVICES_STORAGE_ENABLED=false
```

### Dynamic Configuration Updates

The configuration system supports runtime updates for certain settings:

```go
// Hot-reloadable configuration types
type HotReloadableConfig struct {
    LogLevel        string        `yaml:"log_level" hot_reload:"true"`
    HealthInterval  time.Duration `yaml:"health_interval" hot_reload:"true"`
    ResourceLimits  ResourceLimit `yaml:"resource_limits" hot_reload:"true"`
}

// Configuration requiring restart
type StaticConfig struct {
    ServicePort    int      `yaml:"port" hot_reload:"false"`
    Dependencies   []string `yaml:"dependencies" hot_reload:"false"`
    TLSConfig      TLS      `yaml:"tls" hot_reload:"false"`
}
```

## Configuration Validation

### Schema Validation

Configuration is validated against JSON Schema:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "runtime": {
      "type": "object",
      "properties": {
        "orchestrator": {
          "type": "object",
          "properties": {
            "max_processes": {
              "type": "integer",
              "minimum": 1,
              "maximum": 1000
            }
          }
        }
      }
    }
  },
  "required": ["runtime"]
}
```

### Custom Validation

Services can provide custom validation logic:

```go
func (c *IdentityConfig) Validate() error {
    if c.DID.Method == "" {
        return errors.New("DID method is required")
    }
    
    if c.Authentication.JWTSecret == "" {
        return errors.New("JWT secret is required")
    }
    
    return nil
}
```

## Configuration Security

### Sensitive Data Handling

1. **Environment Variables**: Store secrets in environment variables
2. **External Secret Stores**: Integrate with HashiCorp Vault, AWS Secrets Manager
3. **Encryption**: Encrypt configuration files at rest
4. **Access Control**: Restrict configuration file permissions

### Secret Interpolation

Support for secret interpolation in configuration:

```yaml
identity:
  authentication:
    jwt_secret: "${JWT_SECRET}"                    # Environment variable
    db_password: "${vault:secret/db/password}"     # Vault secret
    api_key: "${file:/etc/secrets/api_key}"        # File-based secret
```

## Configuration Distribution

### Service Configuration Injection

Services receive configuration through multiple channels:

1. **Configuration Files**: Service-specific config files
2. **Environment Variables**: Runtime environment settings
3. **gRPC Configuration Service**: Dynamic configuration updates
4. **Command Line Arguments**: CLI parameter overrides

### Configuration Updates

The configuration system broadcasts updates to services:

```go
type ConfigurationService interface {
    // Get current configuration for a service
    GetConfiguration(ctx context.Context, serviceID string) (*Config, error)
    
    // Subscribe to configuration updates
    SubscribeUpdates(ctx context.Context, serviceID string) (<-chan *Config, error)
    
    // Update configuration (triggers validation and distribution)
    UpdateConfiguration(ctx context.Context, config *Config) error
}
```

## Monitoring and Observability

### Configuration Metrics

- Configuration load times
- Validation failure rates
- Hot reload success rates
- Configuration drift detection

### Configuration Events

- Configuration file changes
- Validation errors
- Hot reload operations
- Service configuration updates

## Best Practices

1. **Environment Separation**: Use environment-specific configuration files
2. **Secret Management**: Never store secrets in configuration files
3. **Validation**: Always validate configuration before deployment
4. **Documentation**: Document all configuration options
5. **Versioning**: Version configuration schemas for compatibility

## Common Configuration Patterns

### Development Environment

```yaml
framework:
  environment: "development"
  log_level: "debug"

runtime:
  orchestrator:
    max_processes: 10
    
services:
  identity:
    enabled: true
    port: 9001
  storage:
    enabled: true
    port: 9002
```

### Production Environment

```yaml
framework:
  environment: "production"
  log_level: "warn"

runtime:
  orchestrator:
    max_processes: 100
    resource_limits:
      default_cpu_percent: 20
      default_memory_mb: 1024

services:
  identity:
    enabled: true
    port: 9001
    config_file: "/etc/blackhole/identity.yaml"
```

## Troubleshooting

### Configuration Issues

- **Invalid YAML**: Use YAML validator to check syntax
- **Missing Secrets**: Verify environment variables are set
- **Permission Errors**: Check file permissions
- **Schema Validation**: Review validation error messages

### Debugging Configuration

```bash
# Validate configuration
blackhole config validate

# Show effective configuration
blackhole config show

# Test configuration with dry run
blackhole start --dry-run --config production.yaml
```

## See Also

- [01_01-Process Orchestration](./01_01-process_orchestration.md)
- [01_02-Lifecycle Management](./01_02-lifecycle_management.md)
- [01_04-Health Monitoring](./01_04-health_monitoring.md)