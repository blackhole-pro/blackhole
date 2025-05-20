# Configuration System Implementation

*Document updated: May 20, 2025*

This document details the implementation plan for the Blackhole Configuration System, which is a key component in the core implementation plan.

## 1. Overview

The Configuration System provides a streamlined approach to managing settings for the core orchestrator and all services. It enables configuration via YAML files and environment variables, with a focus on simplicity and maintainability.

## 2. Core Data Structures

```go
// Config represents the complete system configuration
type Config struct {
    // Core orchestrator configuration
    Orchestrator OrchestratorConfig `yaml:"orchestrator" json:"orchestrator"`
    
    // Services configuration map
    Services map[string]*ServiceConfig `yaml:"services" json:"services"`
    
    // Node-specific configuration
    Node NodeConfig `yaml:"node" json:"node"`
    
    // System-wide settings
    System SystemConfig `yaml:"system" json:"system"`
}

// OrchestratorConfig contains essential settings for the process orchestrator
type OrchestratorConfig struct {
    // Directory containing service binaries
    ServicesDir string `yaml:"services_dir" json:"services_dir" default:"/usr/lib/blackhole"`
    
    // Log level for orchestrator
    LogLevel string `yaml:"log_level" json:"log_level" default:"info"`
    
    // Auto-restart crashed services
    AutoRestart bool `yaml:"auto_restart" json:"auto_restart" default:"true"`
    
    // Process shutdown grace period (seconds)
    ShutdownTimeout int `yaml:"shutdown_timeout" json:"shutdown_timeout" default:"10"`
}

// ServiceConfig contains essential settings for an individual service
type ServiceConfig struct {
    // Whether this service is enabled
    Enabled bool `yaml:"enabled" json:"enabled" default:"true"`
    
    // Path to the service binary (optional, defaults to servicesDir/name/name)
    BinaryPath string `yaml:"binary_path" json:"binary_path"`
    
    // Working directory for the service
    DataDir string `yaml:"data_dir" json:"data_dir"`
    
    // Resource limits (memory in MB)
    MemoryLimit int `yaml:"memory_limit" json:"memory_limit" default:"512"`
    
    // Additional command-line arguments
    Args []string `yaml:"args" json:"args"`
    
    // Environment variables
    Environment map[string]string `yaml:"environment" json:"environment"`
}

// NodeConfig contains node-specific configuration
type NodeConfig struct {
    // Unique node identifier
    ID string `yaml:"id" json:"id"`
    
    // Node role (full, light, storage, etc.)
    Role string `yaml:"role" json:"role" default:"full"`
    
    // Node data directory
    DataDir string `yaml:"data_dir" json:"data_dir" default:"/var/lib/blackhole"`
}

// SystemConfig contains system-wide settings
type SystemConfig struct {
    // Global log level
    LogLevel string `yaml:"log_level" json:"log_level" default:"info"`
}

// ConfigManager handles loading and accessing configuration
type ConfigManager struct {
    // Current active configuration
    current *Config
    
    // Configuration file paths
    filePaths []string
    
    // For thread-safe access
    mu sync.RWMutex
    
    // Event bus for change notifications
    eventBus EventBus
}
```

## 3. Core Interfaces and Methods

```go
// ConfigManager methods
type ConfigManager interface {
    // Initialize the configuration system
    Init() error
    
    // Load configuration from file
    LoadFromFile(path string) error
    
    // Get the current configuration
    GetConfig() *Config
    
    // Get service-specific configuration
    GetServiceConfig(serviceName string) (*ServiceConfig, error)
    
    // Apply environment variable overrides
    ApplyEnvironmentOverrides() error
    
    // Subscribe to configuration changes
    SubscribeToChanges(callback func(*Config)) Subscription
}
```

## 4. Implementation Details

### 4.1 Configuration Loading with Split Files Approach

The configuration system uses a split files approach with search paths, where core configuration and service configurations are stored in separate files:

```
/etc/blackhole/
  blackhole.yaml           # Core/orchestrator config
  identity/                # Identity service directory
    config.yaml            # Identity service config
  storage/                 # Storage service directory
    config.yaml            # Storage service config
  ledger/                  # Ledger service directory
    config.yaml            # Ledger service config
```

The system searches for the core configuration in these locations, in order:

1. `/etc/blackhole/blackhole.yaml` (system-wide)
2. `$HOME/.blackhole/blackhole.yaml` (user-specific)
3. `./blackhole.yaml` (local directory)
4. `./configs/blackhole.yaml` (project directory)

For each service, it looks for service-specific configurations in the corresponding service directory.

```go
// Default configuration search paths for core config
var DefaultCoreConfigPaths = []string{
    "/etc/blackhole/blackhole.yaml",
    os.ExpandEnv("$HOME/.blackhole/blackhole.yaml"),
    "./blackhole.yaml",
    "./configs/blackhole.yaml",
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(coreSearchPaths []string) *ConfigManager {
    // Use default paths if none provided
    if len(coreSearchPaths) == 0 {
        coreSearchPaths = DefaultCoreConfigPaths
    }
    
    return &ConfigManager{
        coreFilePaths:  coreSearchPaths,
        current:        getDefaultConfig(),
        eventBus:       NewEventBus(),
        loadedFrom:     "",
        servicePaths:   make(map[string]string),
    }
}

// Init initializes the configuration system
func (cm *ConfigManager) Init() error {
    // Step 1: Load core configuration
    coreConfigFound := false
    
    // Try each path in order for core config
    for _, path := range cm.coreFilePaths {
        if fileExists(path) {
            err := cm.LoadCoreConfig(path)
            if err != nil {
                return fmt.Errorf("error loading core config from %s: %w", path, err)
            }
            
            // Store loaded path and log it
            cm.loadedFrom = path
            log.Printf("Core configuration loaded from: %s", path)
            
            // Also set the base directory for service configs
            cm.baseConfigDir = filepath.Dir(path)
            
            coreConfigFound = true
            break
        }
    }
    
    // Log if using defaults for core
    if !coreConfigFound {
        log.Printf("No core configuration file found, using default values")
    }
    
    // Step 2: Load service configurations
    err := cm.LoadServiceConfigs()
    if err != nil {
        return fmt.Errorf("error loading service configurations: %w", err)
    }
    
    // Step 3: Apply environment variable overrides
    err = cm.ApplyEnvironmentOverrides()
    if err != nil {
        return fmt.Errorf("error applying environment overrides: %w", err)
    }
    
    return nil
}

// LoadCoreConfig loads the core orchestrator configuration from a file
func (cm *ConfigManager) LoadCoreConfig(path string) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    
    // Create a new config starting with defaults
    config := getDefaultConfig()
    
    // Unmarshal the YAML into the config (core sections only)
    err = yaml.Unmarshal(data, config)
    if err != nil {
        return err
    }
    
    // Set as current configuration
    cm.current = config
    cm.loadedFrom = path
    
    return nil
}

// LoadServiceConfigs discovers and loads all service configurations
func (cm *ConfigManager) LoadServiceConfigs() error {
    // Skip if no base config directory
    if cm.baseConfigDir == "" || !dirExists(cm.baseConfigDir) {
        log.Printf("No base configuration directory found")
        return nil
    }
    
    // Get all service names from the orchestrator configuration or default list
    serviceNames := cm.getKnownServiceNames()
    
    // Try to load configuration for each service
    for _, serviceName := range serviceNames {
        // Check if the service directory exists
        serviceDir := filepath.Join(cm.baseConfigDir, serviceName)
        if !dirExists(serviceDir) {
            log.Printf("No configuration directory found for service: %s", serviceName)
            continue
        }
        
        // Look for config.yaml in the service directory
        serviceConfigPath := filepath.Join(serviceDir, "config.yaml")
        if !fileExists(serviceConfigPath) {
            log.Printf("No config.yaml found for service: %s", serviceName)
            continue
        }
        
        // Load this service config
        err := cm.LoadServiceConfig(serviceName, serviceConfigPath)
        if err != nil {
            return fmt.Errorf("error loading service config for %s: %w", serviceName, err)
        }
    }
    
    return nil
}

// getKnownServiceNames returns the list of known services from configuration or defaults
func (cm *ConfigManager) getKnownServiceNames() []string {
    // Default list of services
    return []string{
        "identity",
        "storage",
        "node",
        "ledger",
        "indexer",
        "social",
        "analytics",
        "telemetry",
        "wallet",
    }
}

// LoadServiceConfig loads configuration for a specific service
func (cm *ConfigManager) LoadServiceConfig(serviceName, path string) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    
    // Parse the service config
    var serviceConfig ServiceConfig
    err = yaml.Unmarshal(data, &serviceConfig)
    if err != nil {
        return err
    }
    
    // Add to the current configuration
    if cm.current.Services == nil {
        cm.current.Services = make(map[string]*ServiceConfig)
    }
    cm.current.Services[serviceName] = &serviceConfig
    
    // Record where this service config came from
    cm.servicePaths[serviceName] = path
    
    log.Printf("Loaded configuration for service: %s from %s", serviceName, path)
    
    return nil
}
```

### 4.2 Environment Variable Handling

The configuration system allows environment variables to override configuration values:

```go
// ApplyEnvironmentOverrides applies environment variable values to the config
func (cm *ConfigManager) ApplyEnvironmentOverrides() error {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    // Get all environment variables
    envVars := os.Environ()
    
    // Process direct overrides
    for _, envVar := range envVars {
        // Split into key and value
        parts := strings.SplitN(envVar, "=", 2)
        if len(parts) != 2 {
            continue
        }
        
        key, value := parts[0], parts[1]
        
        // Check if this is one of our variables
        if !strings.HasPrefix(key, "BLACKHOLE_") {
            continue
        }
        
        // Convert environment variable name to config path
        configPath := envVarToConfigPath(key, "BLACKHOLE_")
        
        // Apply the override
        err := setConfigValueByPath(cm.current, configPath, value)
        if err != nil {
            // Just log errors but continue
            log.Printf("Warning: failed to apply environment override %s: %v", key, err)
        }
    }
    
    return nil
}

// envVarToConfigPath converts an environment variable name to a config path
// Example: BLACKHOLE_ORCHESTRATOR_LOG_LEVEL -> orchestrator.log_level
func envVarToConfigPath(envVar, prefix string) string {
    // Remove prefix
    path := strings.TrimPrefix(envVar, prefix)
    
    // Convert to lowercase
    path = strings.ToLower(path)
    
    // Replace underscores with dots
    path = strings.ReplaceAll(path, "_", ".")
    
    return path
}

// setConfigValueByPath sets a value in the config by its dot-notation path
func setConfigValueByPath(config *Config, path, value string) error {
    // Split the path into parts
    parts := strings.Split(path, ".")
    
    // Use reflection to navigate and set the value
    return setValueByPathParts(reflect.ValueOf(config), parts, value)
}
```

### 4.3 Configuration Validation

Given our focus on simplicity, validation is kept minimal and straightforward:

```go
// ValidateConfig validates a configuration 
func (cm *ConfigManager) ValidateConfig(config *Config) error {
    // Basic validation for essential fields
    if config.Orchestrator.ServicesDir == "" {
        return errors.New("orchestrator.services_dir cannot be empty")
    }
    
    // Validate specific fields with business logic
    if config.Orchestrator.ShutdownTimeout < 1 {
        return errors.New("orchestrator.shutdown_timeout must be at least 1 second")
    }
    
    return nil
}
```

### 4.4 Change Notifications

To support dynamic configuration updates, the Configuration System includes a subscription mechanism:

```go
// SubscribeToChanges subscribes to configuration change events
func (cm *ConfigManager) SubscribeToChanges(callback func(*Config)) Subscription {
    return cm.eventBus.Subscribe("config.changed", func(args ...interface{}) {
        if len(args) > 0 {
            if config, ok := args[0].(*Config); ok {
                callback(config)
            }
        }
    })
}

// notifySubscribers notifies all subscribers of configuration changes
func (cm *ConfigManager) notifySubscribers() {
    // Get a copy of the current config
    config := cm.GetConfig()
    
    // Publish the event
    cm.eventBus.Publish("config.changed", config)
}
```

### 4.5 Default Configuration

A default configuration is provided when no configuration file is found:

```go
// getDefaultConfig returns a configuration with default values
func getDefaultConfig() *Config {
    return &Config{
        Orchestrator: OrchestratorConfig{
            ServicesDir:      "/usr/lib/blackhole",
            LogLevel:         "info",
            AutoRestart:      true,
            ShutdownTimeout:  10,
        },
        Services: make(map[string]*ServiceConfig),
        Node: NodeConfig{
            Role:    "full",
            DataDir: "/var/lib/blackhole",
        },
        System: SystemConfig{
            LogLevel: "info",
        },
    }
}
```

## 5. Cross-Cutting Concerns

### 5.1 Thread Safety

All configuration access and modification methods are thread-safe using mutexes.

### 5.2 Error Handling

The configuration system provides clear error messages with context, especially for common issues:

1. Configuration file not found
2. Parse errors in YAML
3. Validation failures

### 5.3 Security Considerations

1. Sensitive values in environment variables are not logged
2. File permissions are checked during load
3. Configuration directories should be access-restricted

## 6. Usage Example

```go
// Example of creating and using the ConfigManager
func main() {
    // Create a new ConfigManager with default search paths
    configManager := config.NewConfigManager(nil)
    
    // Initialize - this loads from files and applies environment overrides
    err := configManager.Init()
    if err != nil {
        log.Fatalf("Failed to initialize configuration: %v", err)
    }
    
    // Get the full configuration
    cfg := configManager.GetConfig()
    
    // Use configuration values
    log.Printf("Services directory: %s", cfg.Orchestrator.ServicesDir)
    log.Printf("Log level: %s", cfg.Orchestrator.LogLevel)
    
    // Subscribe to configuration changes
    configManager.SubscribeToChanges(func(newConfig *config.Config) {
        log.Printf("Configuration updated")
    })
    
    // Get a specific service configuration
    identityConfig, err := configManager.GetServiceConfig("identity")
    if err != nil {
        log.Printf("No configuration for identity service: %v", err)
    } else {
        log.Printf("Identity service enabled: %v", identityConfig.Enabled)
    }
}
```

## 7. Implementation Priorities

1. **Core Loading**: Basic file loading and struct definitions *(High)*
2. **Environment Overrides**: Application of environment variables *(High)*
3. **Simple Validation**: Basic configuration validation logic *(Medium)*
4. **Change Notifications**: Subscriber notification system *(Medium)*

## 8. Dependencies

- **External**: `gopkg.in/yaml.v3` for YAML parsing
- **Internal**: `EventBus` for change notifications

## 9. Configuration File Example

```yaml
# Core Orchestrator Configuration
orchestrator:
  services_dir: "/usr/local/lib/blackhole"
  log_level: "info"
  auto_restart: true
  shutdown_timeout: 10

# Node Configuration
node:
  id: "node1"
  role: "full"
  data_dir: "/var/lib/blackhole"

# System Configuration
system:
  log_level: "info"
```

Service-specific configuration example (in `/etc/blackhole/identity/config.yaml`):
```yaml
# Identity Service Configuration
enabled: true
binary_path: "/usr/local/lib/blackhole/identity/identity"
data_dir: "/var/lib/blackhole/identity"
memory_limit: 512
args:
  - "--no-cache"
environment:
  RUST_LOG: "info"
```

This simplified approach focuses on what's truly necessary, making the configuration system easy to understand, maintain, and extend as needed.