# Process Orchestrator Implementation Plan

*Updated: May 21, 2025*

This document outlines the implemented approach for the Process Orchestrator, the central component responsible for managing service processes in the Blackhole platform.

## Component Overview

The Process Orchestrator is responsible for:
- Discovering service binaries in a services directory
- Spawning and managing service processes using their individual binaries
- Monitoring service health
- Restarting failed services
- Managing process lifecycle
- Enforcing resource limits
- Handling structured output logging

## Design Principles

The implementation follows these design principles:

1. **Modular Component Architecture**: The orchestrator is divided into specialized subpackages:
   - `types`: Core interfaces and error types
   - `service`: Service lifecycle management
   - `supervision`: Process monitoring and restart logic
   - `isolation`: Resource limits and environment setup
   - `output`: Structured output handling
   - `executor`: Process execution abstraction

2. **Clear Interfaces**: Separate interfaces for each responsibility to improve testing:
   - `ProcessManager`: Service lifecycle operations
   - `ProcessExecutor`: OS process abstraction
   - `ProcessSpawner`: Process creation

3. **Dependency Injection**: Improve testability through constructor injection
   - Functional options pattern for configuration
   - External dependencies like loggers can be injected

4. **Strong Error Handling**: Domain-specific error types
   - Detailed error contexts
   - Support for Go 1.13+ error wrapping
   - Error helper functions

5. **Flexible Configuration**: Dynamic configuration system integration
   - Configuration change subscription
   - Service-specific configurations
   - File and environment-based settings

## Core Interfaces and Types

The core interfaces and types define the contract between components:

### Process State Management

```go
// ProcessState represents the state of a service process
type ProcessState string

const (
    ProcessStateStopped    ProcessState = "stopped"
    ProcessStateStarting   ProcessState = "starting"
    ProcessStateRunning    ProcessState = "running"
    ProcessStateFailed     ProcessState = "failed"
    ProcessStateRestarting ProcessState = "restarting"
)

// ProcessManager defines the interface for process lifecycle operations
type ProcessManager interface {
    Start(name string) error
    Stop(name string) error
    Restart(name string) error
    Status(name string) (ProcessState, error)
    IsRunning(name string) bool
}
```

### OS Process Abstraction

```go
// ProcessExecutor abstracts the execution mechanism for better testability
type ProcessExecutor interface {
    Command(path string, args ...string) ProcessCmd
}

// ProcessCmd abstracts os/exec.Cmd for better testability
type ProcessCmd interface {
    Start() error
    Wait() error
    SetEnv(env []string)
    SetDir(dir string)
    SetOutput(stdout, stderr io.Writer)
    Signal(sig os.Signal) error
    Process() Process
}

// Process abstracts os.Process
type Process interface {
    Pid() int
    Kill() error
}
```

### Service Information

```go
// ServiceInfo contains diagnostic information about a service
type ServiceInfo struct {
    Name         string        `json:"name"`
    Configured   bool          `json:"configured"`
    Enabled      bool          `json:"enabled"`
    State        string        `json:"state"`
    PID          int           `json:"pid,omitempty"`
    Uptime       time.Duration `json:"uptime,omitempty"`
    Restarts     int           `json:"restarts,omitempty"`
    LastExitCode int           `json:"last_exit_code,omitempty"`
    LastError    string        `json:"last_error,omitempty"`
}
```

### Error Handling

```go
// Common process-related error types for classification and handling
var (
    ErrServiceNotFound = errors.New("service not found")
    ErrServiceDisabled = errors.New("service is disabled")
    ErrAlreadyRunning = errors.New("service is already running")
    ErrNotRunning = errors.New("service is not running")
    ErrShuttingDown = errors.New("orchestrator is shutting down")
    ErrConfigChanged = errors.New("configuration changed")
    ErrBinaryNotFound = errors.New("service binary not found")
    ErrTimeout = errors.New("operation timed out")
    ErrMaxRestartsExceeded = errors.New("maximum restart attempts exceeded")
)

// ProcessError provides contextual information about process errors
type ProcessError struct {
    Service  string
    Err      error
    ExitCode int
    PID      int
    Context  string
}
```

## Main Orchestrator Implementation

The core Orchestrator type is the central component that manages services:

```go
// Orchestrator manages service processes
type Orchestrator struct {
    // Configuration
    config      *configtypes.OrchestratorConfig
    services    map[string]*configtypes.ServiceConfig
    
    // Process tracking
    processes   map[string]*ServiceProcess
    processLock sync.RWMutex
    
    // Communication channels
    sigCh       chan os.Signal
    doneCh      chan struct{}
    
    // Dependencies
    logger      *zap.Logger
    executor    types.ProcessExecutor
    
    // Managers
    serviceManager  *service.Manager
    infoProvider    *service.InfoProvider
    supervisor      *supervision.Supervisor
    
    // Control flags
    isShuttingDown atomic.Bool
}
```

### Initialization with Functional Options

```go
// OrchestratorOption is a functional option to configure the orchestrator
type OrchestratorOption func(*Orchestrator)

// WithLogger sets a custom logger
func WithLogger(logger *zap.Logger) OrchestratorOption {
    return func(o *Orchestrator) {
        o.logger = logger
    }
}

// WithExecutor sets a custom process executor
func WithExecutor(exec types.ProcessExecutor) OrchestratorOption {
    return func(o *Orchestrator) {
        o.executor = exec
    }
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(configManager *config.ConfigManager, options ...OrchestratorOption) (*Orchestrator, error) {
    // Get configuration
    cfg := configManager.GetConfig()
    
    // Initialize orchestrator with defaults
    o := &Orchestrator{
        config:      &cfg.Orchestrator,
        services:    make(map[string]*configtypes.ServiceConfig),
        processes:   make(map[string]*ServiceProcess),
        doneCh:      make(chan struct{}),
        executor:    executor.NewDefaultExecutor(),
    }
    
    // Copy service configurations
    for name, svcCfg := range cfg.Services {
        o.services[name] = svcCfg
    }
    
    // Apply options
    for _, option := range options {
        option(o)
    }
    
    // Initialize logger if not provided
    if o.logger == nil {
        logger, err := initLogger(o.config.LogLevel)
        if err != nil {
            return nil, fmt.Errorf("failed to initialize logger: %w", err)
        }
        o.logger = logger
    }
    
    // Initialize managers and components
    o.serviceManager = service.NewManager(o.services, nil, &o.processLock, o.logger)
    o.infoProvider = service.NewInfoProvider(o.services, nil, &o.processLock)
    o.supervisor = supervision.NewSupervisor(o, supervision.SupervisorConfig{
        AutoRestart:       o.config.AutoRestart,
        MaxRestartAttempts: 10,
        InitialBackoffMs:  1000,
        MaxBackoffMs:      30000,
    }, o.logger)
    
    // Setup signal handling
    o.setupSignals()
    
    // Create necessary directories
    // (Simplified for clarity)
    
    // Subscribe to configuration changes
    configManager.SubscribeToChanges(func(newConfig *configtypes.Config) {
        o.handleConfigChange(newConfig)
    })
    
    return o, nil
}
```

## Service Discovery and Process Management

### Discovery Implementation

```go
// DiscoverServices finds all service binaries in the services directory
func (o *Orchestrator) DiscoverServices() ([]string, error) {
    var services []string
    
    // Read services directory
    o.logger.Info("Searching for services in directory", zap.String("directory", o.config.ServicesDir))
    entries, err := os.ReadDir(o.config.ServicesDir)
    if err != nil {
        o.logger.Error("Failed to read services directory", 
            zap.String("directory", o.config.ServicesDir),
            zap.Error(err))
        return nil, fmt.Errorf("failed to read services directory: %w", err)
    }
    
    // Check each entry for service binary
    for _, entry := range entries {
        if entry.IsDir() {
            serviceName := entry.Name()
            serviceBinaryPath := filepath.Join(o.config.ServicesDir, serviceName, serviceName)
            
            // Check if binary exists and is executable
            fileExistsResult := fileExists(serviceBinaryPath)
            isExecutableResult := isExecutable(serviceBinaryPath)
            
            if fileExistsResult && isExecutableResult {
                services = append(services, serviceName)
                o.logger.Info("Discovered service binary", 
                    zap.String("service", serviceName),
                    zap.String("path", serviceBinaryPath))
            }
        }
    }
    
    if len(services) == 0 {
        o.logger.Warn("No service binaries found in services directory", 
            zap.String("directory", o.config.ServicesDir))
    } else {
        o.logger.Info("Discovered service binaries", 
            zap.Int("count", len(services)),
            zap.Strings("services", services))
    }
    
    return services, nil
}
```

### Lifecycle Management

The lifecycle management functions implement the ProcessManager interface:

```go
// Start starts a specific service by name
func (o *Orchestrator) Start(name string) error {
    o.logger.Info("Starting service", zap.String("service", name))
    return o.StartService(name)
}

// Stop stops a running service
func (o *Orchestrator) Stop(name string) error {
    o.logger.Info("Stopping service", zap.String("service", name))
    return o.StopService(name)
}

// Restart restarts a service
func (o *Orchestrator) Restart(name string) error {
    o.logger.Info("Restarting service", zap.String("service", name))
    return o.RestartService(name)
}

// Status gets the current state of a service
func (o *Orchestrator) Status(name string) (types.ProcessState, error) {
    o.processLock.RLock()
    defer o.processLock.RUnlock()
    
    process, exists := o.processes[name]
    if !exists {
        // Check if it's configured but not running
        if _, configExists := o.services[name]; configExists {
            return types.ProcessStateStopped, nil
        }
        return "", fmt.Errorf("service %s not found", name)
    }
    
    return process.State, nil
}

// IsRunning checks if a service is running
func (o *Orchestrator) IsRunning(name string) bool {
    state, err := o.Status(name)
    if err != nil {
        return false
    }
    return state == types.ProcessStateRunning
}
```

## Process Spawning and Supervision

### Process Spawning

```go
// SpawnService starts a new service process
func (o *Orchestrator) SpawnService(name string) error {
    o.processLock.Lock()
    defer o.processLock.Unlock()
    
    // Check if already shutting down
    if o.isShuttingDown.Load() {
        return fmt.Errorf("orchestrator is shutting down, cannot start new services")
    }
    
    // Lookup service configuration
    serviceCfg, exists := o.services[name]
    if !exists {
        return fmt.Errorf("no configuration found for service %s", name)
    }
    
    // Find binary path
    binaryPath, err := isolation.FindServiceBinary(o.config.ServicesDir, name, serviceCfg.BinaryPath)
    if err != nil {
        return err
    }
    
    // Get current process if it exists
    var restartCount int
    existingProcess, exists := o.processes[name]
    if exists {
        restartCount = existingProcess.Restarts
        
        // If already running, return
        if existingProcess.State == types.ProcessStateRunning {
            return nil
        }
        
        // If restarting, increment counter
        if existingProcess.State == types.ProcessStateRestarting {
            restartCount++
        }
        
        // Close stop channel if it exists
        if existingProcess.StopCh != nil {
            close(existingProcess.StopCh)
        }
    }
    
    // Build command-line arguments
    args := []string{"--service", name}
    
    if o.config.LogLevel != "" {
        args = append(args, "--log-level", o.config.LogLevel)
    }
    
    // Add any additional service-specific arguments
    if len(serviceCfg.Args) > 0 {
        args = append(args, serviceCfg.Args...)
    }
    
    // Create command using our executor
    cmd := o.executor.Command(binaryPath, args...)
    
    // Create stop channel for this process
    stopCh := make(chan struct{})
    
    // Create the process record
    process := &ServiceProcess{
        Name:     name,
        Command:  cmd,
        State:    types.ProcessStateStarting,
        Started:  time.Now(),
        Restarts: restartCount,
        StopCh:   stopCh,
    }
    
    // Save wait function for later use by StopService
    process.CommandWait = cmd.Wait
    
    // Setup process output handling
    output.Setup(cmd, name, o.logger)
    
    // Setup process attributes for isolation
    isolation.Setup(cmd, serviceCfg)
    
    // Start the process
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start service %s: %w", name, err)
    }
    
    // Get PID
    proc := cmd.Process()
    if proc != nil {
        process.PID = proc.Pid()
    }
    
    // Store in process map
    o.processes[name] = process
    
    // Begin supervision in a new goroutine
    go o.supervisor.Supervise(&supervision.ProcessInfo{
        Name:     name,
        Command:  cmd,
        State:    types.ProcessStateStarting, 
        PID:      process.PID,
        Restarts: restartCount,
        StopCh:   stopCh,
        Started:  time.Now(),
    }, o.isShuttingDown.Load)
    
    o.logger.Info("Started service", 
        zap.String("service", name),
        zap.Int("pid", process.PID))
    
    return nil
}
```

### Process Supervision

```go
// Supervise monitors a process and restarts it if it fails
func (s *Supervisor) Supervise(process *ProcessInfo, isShuttingDown func() bool) {
    s.logger.Debug("Starting supervision for service", zap.String("service", process.Name))
    
    if process == nil || process.Command == nil {
        s.logger.Error("Invalid process state for supervision", 
            zap.String("service", process.Name))
        return
    }
    
    // Mark as running
    process.State = types.ProcessStateRunning
    
    // Wait for either process exit or stop signal
    exitChan := make(chan error, 1)
    go func() {
        exitErr := process.Command.Wait()
        exitChan <- exitErr
    }()
    
    // Wait for exit or stop signal
    var exitErr error
    select {
    case exitErr = <-exitChan:
        // Process exited on its own
    case <-process.StopCh:
        // Stop requested, return immediately
        return
    }
    
    // Check if shutting down
    if isShuttingDown() {
        s.logger.Info("Service exited during shutdown",
            zap.String("service", process.Name))
        return
    }
    
    // Process exited
    exitCode := 0
    if exitErr != nil {
        if exitError, ok := exitErr.(*exec.ExitError); ok {
            exitCode = exitError.ExitCode()
        }
    }
    
    // Check if exit was successful or failed
    if exitCode != 0 || exitErr != nil {
        // Process exited with error
        s.logger.Warn("Service exited unexpectedly",
            zap.String("service", process.Name),
            zap.Int("exit_code", exitCode),
            zap.Error(exitErr))
        
        // Update status to failed and store error
        process.State = types.ProcessStateFailed
        process.LastError = fmt.Errorf("service exited with code %d: %w", exitCode, exitErr)
    }
    
    // Check if restart is enabled
    if !s.config.AutoRestart {
        s.logger.Info("Auto-restart disabled, not restarting service",
            zap.String("service", process.Name))
        return
    }
    
    // Check if maximum restart limit is reached
    if process.Restarts >= s.config.MaxRestartAttempts {
        s.logger.Error("Service reached maximum restart attempts, not restarting",
            zap.String("service", process.Name),
            zap.Int("restarts", process.Restarts))
        return
    }
    
    // Calculate exponential backoff
    backoffDelay := CalculateBackoffDelay(process.Restarts, s.config.InitialBackoffMs, s.config.MaxBackoffMs)
    
    s.logger.Info("Restarting service after backoff",
        zap.String("service", process.Name),
        zap.Duration("backoff", backoffDelay),
        zap.Int("restart_count", process.Restarts))
        
    // Wait for backoff period or stop signal
    select {
    case <-time.After(backoffDelay):
        // Backoff completed, restart service
    case <-process.StopCh:
        // Stop requested during backoff, exit
        return
    }
    
    // Restart the service
    if err := s.spawner.SpawnProcess(process.Name); err != nil {
        s.logger.Error("Failed to restart service",
            zap.String("service", process.Name),
            zap.Error(err))
    }
}
```

## Process Isolation and Resource Management

```go
// Setup configures process isolation settings for a command
func Setup(cmd processtypes.ProcessCmd, serviceCfg *types.ServiceConfig) {
    // Set working directory if specified
    if serviceCfg.DataDir != "" {
        cmd.SetDir(serviceCfg.DataDir)
    }
    
    // Create a clean environment
    cleanEnv := []string{
        "PATH=" + os.Getenv("PATH"),
        "HOME=" + os.Getenv("HOME"),
        "TEMP=" + os.TempDir(),
        "TMP=" + os.TempDir(),
    }
    
    // Add service-specific environment variables
    if len(serviceCfg.Environment) > 0 {
        for key, value := range serviceCfg.Environment {
            cleanEnv = append(cleanEnv, fmt.Sprintf("%s=%s", key, value))
        }
    }
    
    // Add Go memory limit (for Go services)
    if serviceCfg.MemoryLimit > 0 {
        cleanEnv = append(cleanEnv, fmt.Sprintf("GOMEMLIMIT=%dMiB", serviceCfg.MemoryLimit))
    }
    
    // Set the environment variables
    cmd.SetEnv(cleanEnv)
}
```

## Process Output Handling

```go
// Setup configures process output handling
func Setup(cmd types.ProcessCmd, serviceName string, logger *zap.Logger) {
    // Create service logger
    serviceLogger := logger.With(zap.String("service", serviceName))
    
    // Create output handler
    handler := NewHandler(serviceLogger, serviceName)
    
    // Connect to process outputs
    cmd.SetOutput(handler.Writer(false), handler.Writer(true))
}

// Handler manages service process output
type Handler struct {
    logger      *zap.Logger
    serviceName string
    stdoutBuf   *lineBuffer
    stderrBuf   *lineBuffer
}

// Writer returns a writer for stdout or stderr
func (h *Handler) Writer(isStderr bool) io.Writer {
    if isStderr {
        return h.stderrBuf
    }
    return h.stdoutBuf
}

// Write implements io.Writer for line-buffered logging
func (b *lineBuffer) Write(p []byte) (n int, err error) {
    // Line buffering implementation
    // ...
}
```

## Shutdown and Signal Handling

```go
// Shutdown gracefully shuts down the orchestrator and all managed services
func (o *Orchestrator) Shutdown(ctx context.Context) error {
    o.logger.Info("Orchestrator shutdown requested")
    
    // Mark as shutting down to prevent restarts
    o.isShuttingDown.Store(true)
    
    // Get a list of all running services
    o.processLock.RLock()
    services := make([]string, 0, len(o.processes))
    for name, process := range o.processes {
        if process.State == types.ProcessStateRunning || process.State == types.ProcessStateStarting {
            services = append(services, name)
        }
    }
    o.processLock.RUnlock()
    
    // Create wait group for stopping services
    var wg sync.WaitGroup
    errCh := make(chan error, len(services))
    
    // Stop all services in parallel
    for _, name := range services {
        wg.Add(1)
        go func(serviceName string) {
            defer wg.Done()
            if err := o.StopService(serviceName); err != nil {
                o.logger.Error("Failed to stop service during shutdown",
                    zap.String("service", serviceName),
                    zap.Error(err))
                errCh <- fmt.Errorf("failed to stop %s: %w", serviceName, err)
            }
        }(name)
    }
    
    // Wait for all services to stop or context to be done
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()
    
    // Wait for either completion or context done
    select {
    case <-done:
        o.logger.Info("All services stopped gracefully")
    case <-ctx.Done():
        return fmt.Errorf("shutdown context canceled: %w", ctx.Err())
    }
    
    // Close channels
    close(errCh)
    close(o.doneCh)
    
    return nil
}
```

## Integration Tests

The Process Orchestrator is tested with integration tests that verify:

1. Basic service lifecycle (start, stop)
2. Service auto-restart capability
3. Multi-service orchestration
4. Signal handling
5. Shutdown behavior

```go
// TestStartAndStopService tests starting and stopping a service
func TestStartAndStopService(t *testing.T) {
    // ...
}

// TestServiceAutoRestart tests service auto-restart capability
func TestServiceAutoRestart(t *testing.T) {
    // ...
}

// TestMultiServiceOrchestration tests orchestrating multiple services
func TestMultiServiceOrchestration(t *testing.T) {
    // ...
}

// TestSignalHandling tests proper signal handling by services
func TestSignalHandling(t *testing.T) {
    // ...
}
```

## Completed Features

The Process Orchestrator implementation successfully includes:

1. **Service Discovery**:
   - Dynamic service binary discovery
   - Support for custom binary paths
   - Executable verification

2. **Process Management**:
   - Start, stop, restart operations
   - Service status information
   - Clean environment setup
   - Working directory management

3. **Process Supervision**:
   - Automatic restart of failed services
   - Exponential backoff with jitter
   - Maximum restart attempt limiting
   - Process state tracking

4. **Signal Handling**:
   - Graceful shutdown with SIGTERM
   - Force kill with SIGKILL after timeout
   - Propagation of system signals

5. **Resource Management**:
   - Environment variable isolation
   - Basic memory limits via GOMEMLIMIT
   - Working directory isolation

6. **Output Handling**:
   - Structured logging of process output
   - Line buffering for complete log lines
   - Tagging with service metadata

7. **Configuration Integration**:
   - Dynamic configuration updates
   - Service-specific configurations
   - Default values with overrides

8. **Extensibility**:
   - Modular component architecture
   - Interface-based design
   - Dependency injection

## New Features Beyond Original Plan

The implemented orchestrator includes several enhancements beyond the original simplified plan:

1. **Component Separation**: The orchestrator is divided into specialized subpackages for better separation of concerns:
   - `service`: Service lifecycle management
   - `supervision`: Process monitoring and restart logic
   - `isolation`: Resource limits and environment setup
   - `output`: Structured output handling
   - `executor`: Process execution abstraction

2. **Context-Based Shutdown**: Timeout-aware shutdown using context.Context for better cancellation support.

3. **Enhanced Error Types**: More comprehensive error handling with detailed context and helper functions.

4. **Directory Management**: Automatic creation of necessary directories if they don't exist.

5. **Robust Service Discovery**: More sophisticated service binary discovery with better error reporting.

6. **Parallel Service Operations**: Concurrent starting and stopping of services with proper synchronization.

7. **Restartable Config Manager**: Integration with a configuration manager that supports live updates.

8. **Detailed Service Information**: More comprehensive service diagnostic information.

9. **Memory Limits**: Go services can utilize GOMEMLIMIT environment variable for memory constraints.

10. **Improved Signal Handling**: More sophisticated signal propagation and handling.

## Future Enhancements

Several areas could be enhanced in future iterations:

1. **Advanced Resource Management**:
   - CPU allocation and quota enforcement with cgroups
   - Network bandwidth limiting
   - Disk I/O prioritization

2. **Enhanced Security**:
   - User namespace isolation
   - Capability-based security
   - Network namespaces
   - Seccomp profiles

3. **Dependency-Aware Process Management**:
   - Service dependency resolution
   - Start ordering based on dependencies
   - Health-based dependency validation

4. **Health Checking System**:
   - Protocol-based health checking (HTTP, gRPC)
   - Customizable liveness and readiness probes
   - Automatic recovery based on health status

5. **Observability Enhancements**:
   - Prometheus metrics integration
   - Process profiling support
   - Resource usage tracking

6. **Upgrade Management**:
   - Zero-downtime binary upgrades
   - Rollback support
   - Version tracking