# Process Orchestrator Simplified Implementation Plan

*Updated: May 20, 2025*

This document outlines a simplified implementation approach for the Process Orchestrator, the central component responsible for managing service processes in the Blackhole platform.

## Component Overview

The Process Orchestrator is responsible for:
- Discovering service binaries in a services directory
- Spawning and managing service processes using their individual binaries
- Monitoring service health
- Restarting failed services
- Managing process lifecycle
- Enforcing basic resource limits

## Simplified Design Principles

The simplified implementation follows these design principles:

1. **Clear Interfaces**: Separate interfaces for each responsibility to improve testing
2. **State Pattern**: Explicit state management for service processes
3. **Typed Errors**: Domain-specific error types for better error handling
4. **Dependency Injection**: Improve testability through constructor injection
5. **Single Responsibility**: Each component has a focused, well-defined role

## Implementation Approach

### 1. Core Interfaces and Types

```go
// Import Configuration System types
import (
    "github.com/handcraftdev/blackhole/pkg/config"
)

// ProcessState represents the state of a service process
type ProcessState string

const (
    ProcessStateStopped   ProcessState = "stopped"
    ProcessStateStarting  ProcessState = "starting"
    ProcessStateRunning   ProcessState = "running"
    ProcessStateFailed    ProcessState = "failed"
    ProcessStateRestarting ProcessState = "restarting"
)

// ProcessError provides contextual information about process errors
type ProcessError struct {
    Service string
    Err     error
    ExitCode int
}

func (e *ProcessError) Error() string {
    return fmt.Sprintf("service %s: %v (exit code: %d)", e.Service, e.Err, e.ExitCode)
}

func (e *ProcessError) Unwrap() error {
    return e.Err
}

// ProcessManager defines the interface for process lifecycle operations
type ProcessManager interface {
    Start(name string) error
    Stop(name string) error
    Restart(name string) error
    Status(name string) (ProcessState, error)
    IsRunning(name string) bool
}

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

// DefaultProcessExecutor uses os/exec to execute processes
type DefaultProcessExecutor struct{}

func (e *DefaultProcessExecutor) Command(path string, args ...string) ProcessCmd {
    return &DefaultProcessCmd{cmd: exec.Command(path, args...)}
}

// DefaultProcessCmd wraps os/exec.Cmd
type DefaultProcessCmd struct {
    cmd *exec.Cmd
}

func (c *DefaultProcessCmd) Start() error {
    return c.cmd.Start()
}

func (c *DefaultProcessCmd) Wait() error {
    return c.cmd.Wait()
}

func (c *DefaultProcessCmd) SetEnv(env []string) {
    c.cmd.Env = env
}

func (c *DefaultProcessCmd) SetDir(dir string) {
    c.cmd.Dir = dir
}

func (c *DefaultProcessCmd) SetOutput(stdout, stderr io.Writer) {
    c.cmd.Stdout = stdout
    c.cmd.Stderr = stderr
}

func (c *DefaultProcessCmd) Signal(sig os.Signal) error {
    if c.cmd.Process == nil {
        return fmt.Errorf("process not started")
    }
    return c.cmd.Process.Signal(sig)
}

func (c *DefaultProcessCmd) Process() Process {
    if c.cmd.Process == nil {
        return nil
    }
    return &DefaultProcess{process: c.cmd.Process}
}

// DefaultProcess wraps os.Process
type DefaultProcess struct {
    process *os.Process
}

func (p *DefaultProcess) Pid() int {
    return p.process.Pid
}

func (p *DefaultProcess) Kill() error {
    return p.process.Kill()
}
```

### 2. Orchestrator Implementation

```go
// ServiceProcess represents a running service process with state management
type ServiceProcess struct {
    Name        string
    Command     ProcessCmd
    PID         int
    State       ProcessState
    Started     time.Time
    Restarts    int
    LastError   error
    StopCh      chan struct{}
}

// Orchestrator manages service processes
type Orchestrator struct {
    // Configuration from Configuration System
    config        *config.OrchestratorConfig
    services      map[string]*config.ServiceConfig
    
    // Process tracking
    processes     map[string]*ServiceProcess
    processLock   sync.RWMutex
    
    // Communication channels
    sigCh         chan os.Signal
    doneCh        chan struct{}
    
    // Dependencies
    logger        *zap.Logger
    executor      ProcessExecutor
    
    // Control flags
    isShuttingDown atomic.Bool
}

// OrchestratorOption allows configuring the orchestrator with functional options
type OrchestratorOption func(*Orchestrator)

// WithLogger sets a custom logger
func WithLogger(logger *zap.Logger) OrchestratorOption {
    return func(o *Orchestrator) {
        o.logger = logger
    }
}

// WithExecutor sets a custom process executor
func WithExecutor(executor ProcessExecutor) OrchestratorOption {
    return func(o *Orchestrator) {
        o.executor = executor
    }
}

// NewOrchestrator creates a new process orchestrator
func NewOrchestrator(configManager *config.ConfigManager, options ...OrchestratorOption) (*Orchestrator, error) {
    // Get complete configuration 
    cfg := configManager.GetConfig()
    
    // Initialize orchestrator with configuration
    o := &Orchestrator{
        config:      &cfg.Orchestrator,
        services:    cfg.Services,
        processes:   make(map[string]*ServiceProcess),
        doneCh:      make(chan struct{}),
        executor:    &DefaultProcessExecutor{},
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
    
    // Setup signal handling
    o.setupSignals()
    
    // Verify services directory exists
    if !dirExists(o.config.ServicesDir) {
        return nil, fmt.Errorf("services directory not found: %s", o.config.ServicesDir)
    }
    
    // Subscribe to configuration changes
    configManager.SubscribeToChanges(func(newConfig *config.Config) {
        o.handleConfigChange(newConfig)
    })
    
    return o, nil
}

// handleConfigChange updates orchestrator with new configuration
func (o *Orchestrator) handleConfigChange(newConfig *config.Config) {
    o.processLock.Lock()
    defer o.processLock.Unlock()
    
    o.logger.Info("Configuration update received")
    
    // Update configuration
    o.config = &newConfig.Orchestrator
    
    // Check for removed services and stop them
    for name := range o.services {
        if _, exists := newConfig.Services[name]; !exists {
            o.logger.Info("Service removed from configuration", zap.String("service", name))
            process, exists := o.processes[name]
            if exists && process.State != ProcessStateStopped {
                // Schedule async stop to avoid deadlock (we already hold the lock)
                go func(serviceName string) {
                    if err := o.StopService(serviceName); err != nil {
                        o.logger.Error("Failed to stop removed service", 
                            zap.String("service", serviceName),
                            zap.Error(err))
                    }
                }(name)
            }
        }
    }
    
    // Update service configurations
    o.services = newConfig.Services
    
    o.logger.Info("Configuration updated", 
        zap.Int("num_services", len(o.services)))
}
```

### 3. Service Discovery and Process Management

```go
// DiscoverServices finds all service binaries in the services directory
func (o *Orchestrator) DiscoverServices() ([]string, error) {
    var services []string
    
    // Read services directory
    entries, err := os.ReadDir(o.config.ServicesDir)
    if err != nil {
        return nil, fmt.Errorf("failed to read services directory: %w", err)
    }
    
    // Check each entry for service binary
    for _, entry := range entries {
        if entry.IsDir() {
            serviceName := entry.Name()
            serviceBinaryPath := filepath.Join(o.config.ServicesDir, serviceName, serviceName)
            
            // Check if binary exists and is executable
            if fileExists(serviceBinaryPath) && isExecutable(serviceBinaryPath) {
                services = append(services, serviceName)
                o.logger.Debug("Discovered service binary", 
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

// RefreshServices re-discovers services in the services directory
func (o *Orchestrator) RefreshServices() ([]string, error) {
    services, err := o.DiscoverServices()
    if err != nil {
        return nil, err
    }
    
    o.processLock.Lock()
    defer o.processLock.Unlock()
    
    // Log any newly discovered services
    var newServices []string
    for _, serviceName := range services {
        if _, exists := o.services[serviceName]; !exists {
            newServices = append(newServices, serviceName)
        }
    }
    
    if len(newServices) > 0 {
        o.logger.Info("Discovered new services", 
            zap.Strings("services", newServices))
    }
    
    return services, nil
}

// Start launches the orchestrator
func (o *Orchestrator) Start() error {
    // Initial service discovery
    _, err := o.RefreshServices()
    if err != nil {
        o.logger.Warn("Initial service discovery failed", zap.Error(err))
        // Continue anyway, this is not fatal
    }
    
    o.logger.Info("Process orchestrator started")
    return nil
}

// StartService starts a specific service by name
func (o *Orchestrator) StartService(name string) error {
    o.processLock.RLock()
    // Get service configuration 
    serviceCfg, exists := o.services[name]
    if !exists {
        o.processLock.RUnlock()
        return fmt.Errorf("no configuration found for service %s", name)
    }
    
    // Skip disabled services
    if !serviceCfg.Enabled {
        o.logger.Info("Skipping disabled service", zap.String("service", name))
        o.processLock.RUnlock()
        return nil
    }
    
    // Check if service is already running
    process, exists := o.processes[name]
    o.processLock.RUnlock()
    
    if exists && process.State == ProcessStateRunning {
        o.logger.Info("Service already running", zap.String("service", name))
        return nil
    }
    
    // Spawn the service process
    return o.SpawnService(name)
}

// StartAll starts all enabled services
func (o *Orchestrator) StartAll() error {
    // Refresh available services
    _, err := o.RefreshServices()
    if err != nil {
        return fmt.Errorf("failed to refresh services: %w", err)
    }
    
    // Start each service that is configured and enabled
    var startErrors []string
    var wg sync.WaitGroup
    resultCh := make(chan error, len(o.services))
    
    // Lock to safely access the services map
    o.processLock.RLock()
    serviceConfigs := make(map[string]*config.ServiceConfig)
    for name, cfg := range o.services {
        if cfg.Enabled {
            serviceConfigs[name] = cfg
        }
    }
    o.processLock.RUnlock()
    
    // Start each enabled service in parallel
    for serviceName := range serviceConfigs {
        wg.Add(1)
        go func(name string) {
            defer wg.Done()
            if err := o.StartService(name); err != nil {
                o.logger.Error("Failed to start service", 
                    zap.String("service", name),
                    zap.Error(err))
                resultCh <- fmt.Errorf("%s: %w", name, err)
            }
        }(serviceName)
    }
    
    // Wait for all services to start
    wg.Wait()
    close(resultCh)
    
    // Collect any errors
    for err := range resultCh {
        startErrors = append(startErrors, err.Error())
    }
    
    // Return error if any service failed to start
    if len(startErrors) > 0 {
        return fmt.Errorf("failed to start %d services: %s", 
            len(startErrors), strings.Join(startErrors, "; "))
    }
    
    return nil
}

// Stop gracefully shuts down the orchestrator
func (o *Orchestrator) Stop() error {
    o.logger.Info("Stopping all services and shutting down orchestrator")
    
    // Mark as shutting down to prevent restarts
    o.isShuttingDown.Store(true)
    
    // Create a context with timeout for graceful shutdown
    ctx, cancel := context.WithTimeout(
        context.Background(),
        time.Duration(o.config.ShutdownTimeout) * time.Second,
    )
    defer cancel()
    
    // Get a list of all running services
    o.processLock.RLock()
    services := make([]string, 0, len(o.processes))
    for name, process := range o.processes {
        if process.State == ProcessStateRunning || process.State == ProcessStateStarting {
            services = append(services, name)
        }
    }
    o.processLock.RUnlock()
    
    // Stop all services in parallel with the same deadline
    var wg sync.WaitGroup
    for _, name := range services {
        wg.Add(1)
        go func(serviceName string) {
            defer wg.Done()
            if err := o.StopService(serviceName); err != nil {
                o.logger.Error("Failed to stop service",
                    zap.String("service", serviceName),
                    zap.Error(err))
            }
        }(name)
    }
    
    // Wait for services to stop or timeout
    stopped := make(chan struct{})
    go func() {
        wg.Wait()
        close(stopped)
    }()
    
    // Wait for either completion or timeout
    select {
    case <-stopped:
        o.logger.Info("All services stopped gracefully")
    case <-ctx.Done():
        o.logger.Warn("Shutdown grace period exceeded, some services may not have stopped gracefully")
    }
    
    // Close channels and resources
    close(o.doneCh)
    
    return nil
}
```

### 4. Process Spawning and Supervision

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
    
    // Determine binary path - use configured path or default
    binaryPath := serviceCfg.BinaryPath
    if binaryPath == "" {
        // Default to binary in service directory: servicesDir/name/name
        binaryPath = filepath.Join(o.config.ServicesDir, name, name)
    }
    
    // Ensure binary exists
    if !fileExists(binaryPath) {
        return fmt.Errorf("service binary not found at %s", binaryPath)
    }
    
    // Get current process if it exists
    var restartCount int
    existingProcess, exists := o.processes[name]
    if exists {
        restartCount = existingProcess.Restarts
        
        // If already running, return
        if existingProcess.State == ProcessStateRunning {
            return nil
        }
        
        // If restarting, increment counter
        if existingProcess.State == ProcessStateRestarting {
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
        State:    ProcessStateStarting,
        Started:  time.Now(),
        Restarts: restartCount,
        StopCh:   stopCh,
    }
    
    // Setup process output handling
    setupProcessOutput(cmd, name, o.logger)
    
    // Setup process attributes for isolation
    setupProcessIsolation(cmd, serviceCfg)
    
    // Start the process
    if err := cmd.Start(); err != nil {
        return &ProcessError{
            Service: name,
            Err:     fmt.Errorf("failed to start: %w", err),
            ExitCode: -1,
        }
    }
    
    // Get PID
    proc := cmd.Process()
    if proc != nil {
        process.PID = proc.Pid()
    }
    
    // Store in process map
    o.processes[name] = process
    
    // Begin supervision in a new goroutine
    go o.supervise(name, stopCh)
    
    o.logger.Info("Started service", 
        zap.String("service", name),
        zap.Int("pid", process.PID))
    
    return nil
}

// StopService stops a running service
func (o *Orchestrator) StopService(name string) error {
    // Find process in process map
    o.processLock.Lock()
    process, exists := o.processes[name]
    if !exists {
        o.processLock.Unlock()
        return fmt.Errorf("service %s not found", name)
    }
    
    // Check if already stopped
    if process.State == ProcessStateStopped {
        o.processLock.Unlock()
        return nil
    }
    
    // Update status and get stop channel
    process.State = ProcessStateStopped
    stopCh := process.StopCh
    pid := process.PID
    o.processLock.Unlock()
    
    // Send stop signal to supervision goroutine
    if stopCh != nil {
        close(stopCh)
    }
    
    o.logger.Info("Sending SIGTERM to service", 
        zap.String("service", name),
        zap.Int("pid", pid))
    
    // Send SIGTERM for graceful shutdown
    if err := o.sendSignal(name, syscall.SIGTERM); err != nil {
        o.logger.Warn("Failed to send SIGTERM", 
            zap.String("service", name),
            zap.Error(err))
    }
    
    // Wait for process to exit with timeout
    exitChan := make(chan error, 1)
    go func() {
        o.processLock.RLock()
        process, exists := o.processes[name]
        if !exists || process.Command == nil {
            o.processLock.RUnlock()
            exitChan <- nil
            return
        }
        o.processLock.RUnlock()
        
        exitErr := process.Command.Wait()
        exitChan <- exitErr
    }()
    
    // Wait for exit or timeout
    select {
    case err := <-exitChan:
        if err != nil {
            o.logger.Warn("Error waiting for service to exit",
                zap.String("service", name),
                zap.Error(err))
        }
    case <-time.After(time.Duration(o.config.ShutdownTimeout) * time.Second):
        // Timeout occurred, force kill
        o.logger.Warn("Service did not exit gracefully, sending SIGKILL",
            zap.String("service", name),
            zap.Int("pid", pid))
        
        if err := o.sendSignal(name, syscall.SIGKILL); err != nil {
            o.logger.Error("Failed to send SIGKILL",
                zap.String("service", name),
                zap.Error(err))
            return &ProcessError{
                Service:  name,
                Err:      fmt.Errorf("failed to forcefully terminate: %w", err),
                ExitCode: -1,
            }
        }
    }
    
    o.logger.Info("Service stopped", zap.String("service", name))
    return nil
}

// RestartService restarts a running service
func (o *Orchestrator) RestartService(name string) error {
    o.logger.Info("Restarting service", zap.String("service", name))
    
    o.processLock.Lock()
    process, exists := o.processes[name]
    if exists {
        process.State = ProcessStateRestarting
    }
    o.processLock.Unlock()
    
    // Stop the service
    if err := o.StopService(name); err != nil {
        // Log but continue, as we'll try to start anyway
        o.logger.Warn("Error stopping service during restart",
            zap.String("service", name),
            zap.Error(err))
    }
    
    // Start the service
    return o.StartService(name)
}

// supervise monitors a service and restarts it if needed
func (o *Orchestrator) supervise(name string, stopCh chan struct{}) {
    o.logger.Debug("Starting supervision for service", zap.String("service", name))
    
    o.processLock.RLock()
    process, exists := o.processes[name]
    if !exists || process.Command == nil {
        o.processLock.RUnlock()
        o.logger.Error("Invalid process state for supervision", 
            zap.String("service", name))
        return
    }
    o.processLock.RUnlock()
    
    // Mark as running
    o.processLock.Lock()
    if process, exists := o.processes[name]; exists {
        process.State = ProcessStateRunning
    }
    o.processLock.Unlock()
    
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
    case <-stopCh:
        // Stop requested, return immediately
        return
    }
    
    // Check if shutting down
    if o.isShuttingDown.Load() {
        o.logger.Info("Service exited during shutdown",
            zap.String("service", name))
        return
    }
    
    // Process exited unexpectedly
    exitCode := 0
    if exitErr != nil {
        if exitError, ok := exitErr.(*exec.ExitError); ok {
            exitCode = exitError.ExitCode()
        }
    }
    
    o.logger.Warn("Service exited unexpectedly",
        zap.String("service", name),
        zap.Int("exit_code", exitCode),
        zap.Error(exitErr))
    
    // Update status to failed
    o.processLock.Lock()
    process, exists = o.processes[name]
    if exists {
        process.State = ProcessStateFailed
        process.LastError = &ProcessError{
            Service:  name,
            Err:      exitErr,
            ExitCode: exitCode,
        }
    }
    o.processLock.Unlock()
    
    // Check if restart is enabled
    if !o.config.AutoRestart {
        o.logger.Info("Auto-restart disabled, not restarting service",
            zap.String("service", name))
        return
    }
    
    // Get current restart count
    o.processLock.RLock()
    restartCount := 0
    if proc, exists := o.processes[name]; exists {
        restartCount = proc.Restarts
    }
    o.processLock.RUnlock()
    
    // Check if maximum restart limit is reached
    const maxRestartAttempts = 10
    if restartCount >= maxRestartAttempts {
        o.logger.Error("Service reached maximum restart attempts, not restarting",
            zap.String("service", name),
            zap.Int("restarts", restartCount))
        return
    }
    
    // Calculate exponential backoff
    backoffDelay := calculateBackoffDelay(restartCount)
    
    o.logger.Info("Restarting service after backoff",
        zap.String("service", name),
        zap.Duration("backoff", backoffDelay),
        zap.Int("restart_count", restartCount))
        
    // Wait for backoff period or stop signal
    select {
    case <-time.After(backoffDelay):
        // Backoff completed, restart service
    case <-stopCh:
        // Stop requested during backoff, exit
        return
    }
    
    // Restart the service
    if err := o.SpawnService(name); err != nil {
        o.logger.Error("Failed to restart service",
            zap.String("service", name),
            zap.Error(err))
    }
}

// calculateBackoffDelay implements exponential backoff with jitter
func calculateBackoffDelay(restartCount int) time.Duration {
    // Base delay and max delay in milliseconds
    const (
        initialDelay = 1000  // 1 second
        maxDelay     = 30000 // 30 seconds
    )
    
    // Calculate exponential backoff
    delayMs := math.Min(
        float64(initialDelay) * math.Pow(2, float64(restartCount)),
        float64(maxDelay),
    )
    
    // Add jitter (± 10%)
    jitterFactor := 0.9 + (0.2 * rand.Float64())
    delayMs = delayMs * jitterFactor
    
    return time.Duration(delayMs) * time.Millisecond
}
```

### 5. Process Utilities 

```go
// sendSignal sends a signal to a process
func (o *Orchestrator) sendSignal(name string, sig syscall.Signal) error {
    o.processLock.RLock()
    process, exists := o.processes[name]
    if !exists || process.PID <= 0 {
        o.processLock.RUnlock()
        return fmt.Errorf("process %s not found or not running", name)
    }
    o.processLock.RUnlock()
    
    // Send signal to process group to ensure all child processes receive it
    pgid, err := syscall.Getpgid(process.PID)
    if err != nil {
        return fmt.Errorf("failed to get process group: %w", err)
    }
    
    if err := syscall.Kill(-pgid, sig); err != nil {
        return fmt.Errorf("failed to send signal: %w", err)
    }
    
    return nil
}

// GetServiceInfo returns diagnostic information about a service
func (o *Orchestrator) GetServiceInfo(name string) (*ServiceInfo, error) {
    o.processLock.RLock()
    defer o.processLock.RUnlock()
    
    // Check if service is configured
    serviceCfg, exists := o.services[name]
    if !exists {
        return nil, fmt.Errorf("service %s not configured", name)
    }
    
    // Get process info if running
    process, exists := o.processes[name]
    if !exists {
        return &ServiceInfo{
            Name:      name,
            Configured: true,
            Enabled:   serviceCfg.Enabled,
            State:     string(ProcessStateStopped),
        }, nil
    }
    
    // Build service info
    info := &ServiceInfo{
        Name:       name,
        Configured: true,
        Enabled:    serviceCfg.Enabled,
        State:      string(process.State),
        PID:        process.PID,
        Uptime:     time.Since(process.Started),
        Restarts:   process.Restarts,
    }
    
    // Add error info if available
    if process.LastError != nil {
        if procErr, ok := process.LastError.(*ProcessError); ok {
            info.LastExitCode = procErr.ExitCode
            info.LastError = procErr.Error()
        } else {
            info.LastError = process.LastError.Error()
        }
    }
    
    return info, nil
}

// GetAllServices returns information about all services
func (o *Orchestrator) GetAllServices() (map[string]*ServiceInfo, error) {
    o.processLock.RLock()
    defer o.processLock.RUnlock()
    
    services := make(map[string]*ServiceInfo)
    
    // Add all configured services
    for name, cfg := range o.services {
        info := &ServiceInfo{
            Name:       name,
            Configured: true,
            Enabled:    cfg.Enabled,
            State:      string(ProcessStateStopped),
        }
        
        // Add process info if running
        if process, exists := o.processes[name]; exists {
            info.State = string(process.State)
            info.PID = process.PID
            info.Uptime = time.Since(process.Started)
            info.Restarts = process.Restarts
            
            // Add error info if available
            if process.LastError != nil {
                if procErr, ok := process.LastError.(*ProcessError); ok {
                    info.LastExitCode = procErr.ExitCode
                    info.LastError = procErr.Error()
                } else {
                    info.LastError = process.LastError.Error()
                }
            }
        }
        
        services[name] = info
    }
    
    return services, nil
}

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

### 6. Process Output Handling

```go
// setupProcessOutput configures stdout/stderr handling for the process
func setupProcessOutput(cmd ProcessCmd, serviceName string, logger *zap.Logger) {
    // Create service logger
    serviceLogger := logger.With(zap.String("service", serviceName))
    
    // Create prefixed writers for stdout and stderr
    stdout := newPrefixedLogWriter(serviceLogger, serviceName, false)
    stderr := newPrefixedLogWriter(serviceLogger, serviceName, true)
    
    // Attach to command
    cmd.SetOutput(stdout, stderr)
}

// prefixedLogWriter writes process output to a logger
type prefixedLogWriter struct {
    logger     *zap.Logger
    service    string
    isError    bool
    buffer     bytes.Buffer
    bufferLock sync.Mutex
}

// newPrefixedLogWriter creates a new prefixed log writer
func newPrefixedLogWriter(logger *zap.Logger, service string, isError bool) *prefixedLogWriter {
    return &prefixedLogWriter{
        logger:  logger,
        service: service,
        isError: isError,
    }
}

// Write implements io.Writer
func (w *prefixedLogWriter) Write(p []byte) (n int, err error) {
    w.bufferLock.Lock()
    defer w.bufferLock.Unlock()
    
    // Write to buffer
    n, err = w.buffer.Write(p)
    if err != nil {
        return n, err
    }
    
    // Process complete lines
    for {
        line, err := w.buffer.ReadString('\n')
        if err == io.EOF {
            // Put back incomplete line
            w.buffer.WriteString(line)
            break
        }
        
        // Trim trailing newline
        line = strings.TrimSuffix(line, "\n")
        if line == "" {
            continue
        }
        
        // Log the line
        if w.isError {
            w.logger.Error(line, zap.String("source", "stderr"))
        } else {
            w.logger.Info(line, zap.String("source", "stdout"))
        }
    }
    
    return n, nil
}
```

### 7. Signal Handling

```go
// setupSignals initializes signal handling
func (o *Orchestrator) setupSignals() {
    o.sigCh = make(chan os.Signal, 1)
    signal.Notify(o.sigCh, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        sig := <-o.sigCh
        o.logger.Info("Received signal", zap.String("signal", sig.String()))
        o.Stop()
    }()
}

// IsShuttingDown returns true if the orchestrator is in the process of shutting down
func (o *Orchestrator) IsShuttingDown() bool {
    return o.isShuttingDown.Load()
}
```

### 8. Process Isolation

```go
// setupProcessIsolation configures process isolation and resource limits
func setupProcessIsolation(cmd ProcessCmd, serviceCfg *config.ServiceConfig) {
    // Set process group ID and other system attributes
    if cmd, ok := cmd.(*DefaultProcessCmd); ok && cmd.cmd != nil {
        cmd.cmd.SysProcAttr = &syscall.SysProcAttr{
            Setpgid: true, // Create new process group for better signal handling
        }
    }
    
    // Create a clean environment
    cleanEnv := []string{
        "PATH=" + os.Getenv("PATH"),
        "HOME=" + serviceCfg.DataDir,
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
    
    // Set working directory if specified
    if serviceCfg.DataDir != "" {
        cmd.SetDir(serviceCfg.DataDir)
    }
}
```

## Testing Approach

The simplified implementation enables much more comprehensive testing through the use of interfaces and dependency injection:

### 1. Unit Tests with Mocking

```go
// Define mock executor for testing
type MockProcessExecutor struct {
    CommandFunc func(path string, args ...string) ProcessCmd
}

func (m *MockProcessExecutor) Command(path string, args ...string) ProcessCmd {
    if m.CommandFunc != nil {
        return m.CommandFunc(path, args...)
    }
    return &MockProcessCmd{}
}

// Mock command for testing
type MockProcessCmd struct {
    StartFunc  func() error
    WaitFunc   func() error
    ProcessFunc func() Process
    
    // Track method calls for verification
    StartCalled  bool
    WaitCalled   bool
    Env          []string
    Dir          string
    Stdout       io.Writer
    Stderr       io.Writer
}

func (m *MockProcessCmd) Start() error {
    m.StartCalled = true
    if m.StartFunc != nil {
        return m.StartFunc()
    }
    return nil
}

func (m *MockProcessCmd) Wait() error {
    m.WaitCalled = true
    if m.WaitFunc != nil {
        return m.WaitFunc()
    }
    return nil
}

func (m *MockProcessCmd) SetEnv(env []string) {
    m.Env = env
}

func (m *MockProcessCmd) SetDir(dir string) {
    m.Dir = dir
}

func (m *MockProcessCmd) SetOutput(stdout, stderr io.Writer) {
    m.Stdout = stdout
    m.Stderr = stderr
}

func (m *MockProcessCmd) Signal(sig os.Signal) error {
    return nil
}

func (m *MockProcessCmd) Process() Process {
    if m.ProcessFunc != nil {
        return m.ProcessFunc()
    }
    return &MockProcess{pid: 1000}
}

// Mock process for testing
type MockProcess struct {
    pid int
}

func (m *MockProcess) Pid() int {
    return m.pid
}

func (m *MockProcess) Kill() error {
    return nil
}

// Test the orchestrator with mocks
func TestOrchestrator_UnitTests(t *testing.T) {
    // Create mock config manager
    configManager := &MockConfigManager{
        GetConfigFunc: func() *config.Config {
            return &config.Config{
                Orchestrator: config.OrchestratorConfig{
                    ServicesDir:    "/tmp/services",
                    AutoRestart:    true,
                    ShutdownTimeout: 30,
                },
                Services: map[string]*config.ServiceConfig{
                    "test-service": {
                        Enabled:   true,
                        DataDir:   "/tmp/services/test-service/data",
                        Args:      []string{"--debug"},
                    },
                },
            }
        },
    }
    
    // Create mock process executor
    mockExec := &MockProcessExecutor{
        CommandFunc: func(path string, args ...string) ProcessCmd {
            return &MockProcessCmd{
                StartFunc: func() error { return nil },
                WaitFunc: func() error { return nil },
                ProcessFunc: func() Process {
                    return &MockProcess{pid: 1000}
                },
            }
        },
    }
    
    // Setup file system tests by simulating servicesDir
    setupTestDir(t, "/tmp/services/test-service")
    
    // Create orchestrator with mocks
    orch, err := NewOrchestrator(configManager, WithExecutor(mockExec))
    require.NoError(t, err)
    
    // Test starting a service
    t.Run("StartService", func(t *testing.T) {
        err := orch.StartService("test-service")
        require.NoError(t, err)
        
        // Verify service is marked as running
        info, err := orch.GetServiceInfo("test-service")
        require.NoError(t, err)
        assert.Equal(t, "running", info.State)
        assert.Equal(t, 1000, info.PID)
    })
    
    // Test stopping a service
    t.Run("StopService", func(t *testing.T) {
        err := orch.StopService("test-service")
        require.NoError(t, err)
        
        // Verify service is marked as stopped
        info, err := orch.GetServiceInfo("test-service")
        require.NoError(t, err)
        assert.Equal(t, "stopped", info.State)
    })
    
    // Test restarting a service
    t.Run("RestartService", func(t *testing.T) {
        err := orch.StartService("test-service")
        require.NoError(t, err)
        
        err = orch.RestartService("test-service")
        require.NoError(t, err)
        
        // Verify service is marked as running with restart count incremented
        info, err := orch.GetServiceInfo("test-service")
        require.NoError(t, err)
        assert.Equal(t, "running", info.State)
        assert.Equal(t, 1, info.Restarts)
    })
}
```

### 2. Integration Tests with Real Processes

```go
// Integration tests with real processes
func TestOrchestrator_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration tests in short mode")
    }
    
    // Create test service binaries
    tempDir := t.TempDir()
    servicesDir := filepath.Join(tempDir, "services")
    require.NoError(t, os.MkdirAll(servicesDir, 0755))
    
    // Build test binaries
    successBin := buildTestBinary(t, servicesDir, "success", 0)
    failureBin := buildTestBinary(t, servicesDir, "failure", 1)
    
    // Create config manager
    configManager := &MockConfigManager{
        GetConfigFunc: func() *config.Config {
            return &config.Config{
                Orchestrator: config.OrchestratorConfig{
                    ServicesDir:    servicesDir,
                    AutoRestart:    true,
                    ShutdownTimeout: 5,
                },
                Services: map[string]*config.ServiceConfig{
                    "success": {
                        Enabled:   true,
                        DataDir:   filepath.Join(servicesDir, "success", "data"),
                    },
                    "failure": {
                        Enabled:   true,
                        DataDir:   filepath.Join(servicesDir, "failure", "data"),
                    },
                },
            }
        },
    }
    
    // Create orchestrator with real process executor
    orch, err := NewOrchestrator(configManager)
    require.NoError(t, err)
    
    // Test starting a successful service
    t.Run("StartSuccessService", func(t *testing.T) {
        err := orch.StartService("success")
        require.NoError(t, err)
        
        // Verify service is running
        info, err := orch.GetServiceInfo("success")
        require.NoError(t, err)
        assert.Equal(t, "running", info.State)
        assert.Greater(t, info.PID, 0)
        
        // Let it run for a moment
        time.Sleep(500 * time.Millisecond)
        
        // Stop the service
        err = orch.StopService("success")
        require.NoError(t, err)
    })
    
    // Test service that fails and gets restarted
    t.Run("RestartFailingService", func(t *testing.T) {
        // Set a smaller restart interval for faster testing
        setTestRestartDelay(t, orch)
        
        err := orch.StartService("failure")
        require.NoError(t, err)
        
        // Wait for it to fail and attempt restart (up to 3 seconds)
        deadline := time.Now().Add(3 * time.Second)
        var info *ServiceInfo
        for time.Now().Before(deadline) {
            info, err = orch.GetServiceInfo("failure")
            require.NoError(t, err)
            
            // If we see a restart count > 0, test passed
            if info.Restarts > 0 {
                break
            }
            time.Sleep(100 * time.Millisecond)
        }
        
        assert.Greater(t, info.Restarts, 0, "Service should have restarted at least once")
        assert.Equal(t, 1, info.LastExitCode)
        
        // Stop the service
        err = orch.StopService("failure")
        require.NoError(t, err)
    })
    
    // Test orchestrator shutdown
    t.Run("OrchestratorShutdown", func(t *testing.T) {
        // Start both services
        err := orch.StartAll()
        require.NoError(t, err)
        
        // Stop all services
        err = orch.Stop()
        require.NoError(t, err)
        
        // Verify all services are stopped
        services, err := orch.GetAllServices()
        require.NoError(t, err)
        
        for name, info := range services {
            assert.Equal(t, "stopped", info.State, "Service %s should be stopped", name)
        }
    })
}

// Helper to build test binaries
func buildTestBinary(t *testing.T, dir, name string, exitCode int) string {
    // Create service directory
    serviceDir := filepath.Join(dir, name)
    require.NoError(t, os.MkdirAll(serviceDir, 0755))
    
    // Create data directory
    dataDir := filepath.Join(serviceDir, "data")
    require.NoError(t, os.MkdirAll(dataDir, 0755))
    
    // Create source file
    srcPath := filepath.Join(t.TempDir(), name+".go")
    src := fmt.Sprintf(`
package main

import (
    "fmt"
    "os"
    "time"
)

func main() {
    fmt.Println("Starting test service: %s")
    fmt.Println("Arguments:", os.Args)
    fmt.Println("Environment:")
    for _, env := range os.Environ() {
        fmt.Println(" ", env)
    }
    
    // Sleep briefly if not failure service
    if %d == 0 {
        time.Sleep(1 * time.Hour) // Long sleep for success service
    } else {
        time.Sleep(100 * time.Millisecond) // Brief sleep then exit
    }
    
    os.Exit(%d)
}
`, name, exitCode, exitCode)
    
    require.NoError(t, os.WriteFile(srcPath, []byte(src), 0644))
    
    // Build binary
    binPath := filepath.Join(serviceDir, name)
    cmd := exec.Command("go", "build", "-o", binPath, srcPath)
    output, err := cmd.CombinedOutput()
    require.NoError(t, err, "Failed to build test binary: %s", output)
    
    return binPath
}
```

## Implementation Steps

1. Implement the core interfaces and types:
   - ProcessState, ProcessError
   - ProcessManager, ProcessExecutor, ProcessCmd interfaces
   - Default implementations for interfaces

2. Implement the Orchestrator struct and constructor:
   - Add functional options for dependency injection
   - Implement configuration integration

3. Develop the service discovery functions:
   - DiscoverServices, RefreshServices
   - Add service tracking

4. Implement process management functions:
   - StartService, StartAll, StopService
   - RestartService and service info functions

5. Create process spawning and supervision:
   - SpawnService with proper isolation
   - supervise with exponential backoff
   - Restart error handling

6. Add process output handling:
   - Implement prefixedLogWriter
   - Line buffering and logger integration

7. Implement signal handling:
   - Setup signal capture and forwarding
   - Process group signaling

8. Create unit and integration tests:
   - Develop mock implementations
   - Create integration tests with real binaries

## Future Enhancements

1. **Advanced Resource Management**:
   - CPU allocation and quota enforcement
   - Memory limit enforcement with cgroups
   - Disk I/O prioritization

2. **Enhanced Security**:
   - Binary verification with checksums
   - Privilege dropping for services
   - Namespaces for stronger isolation

3. **Dependency-Aware Process Management**:
   - Service dependency resolution
   - Start ordering based on dependencies
   - Health-based dependency validation

4. **Advanced Health Checking**:
   - Protocol-based health checking (HTTP, gRPC)
   - Customizable liveness and readiness probes
   - Automatic recovery based on health status

5. **Observability Enhancements**:
   - Prometheus metrics for process health
   - Process profiling integration
   - Detailed resource usage tracking