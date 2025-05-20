# Process Orchestrator Implementation Plan (Simplified)

*Updated: May 20, 2025*

This document outlines a simplified implementation approach for the Process Orchestrator, which is the central component responsible for spawning, monitoring, and managing service processes.

## Component Overview

The Process Orchestrator is responsible for:
- Discovering service binaries in a services directory
- Spawning and managing service processes using their individual binaries
- Monitoring service health
- Restarting failed services
- Managing process lifecycle
- Enforcing basic resource limits

## Implementation Approach

The implementation is designed around clear interfaces and modular components that work together to provide process management capabilities. This approach improves maintainability, testability, and extensibility.

### 1. Core Interfaces and Types

```go
// Package orchestrator provides process management for service binaries
package orchestrator

import (
    "context"
    "io"
    "os"
    "os/exec"
    "syscall"
    "time"
    
    "github.com/blackhole/pkg/config"
    "go.uber.org/zap"
)

// ProcessManager defines the interface for managing processes
type ProcessManager interface {
    // CreateCommand creates a command for execution
    CreateCommand(path string, args ...string) Command
    // GetProcessGroup gets the process group ID for a PID
    GetProcessGroup(pid int) (int, error)
    // SendSignal sends a signal to a process group
    SendSignal(pgid int, signal syscall.Signal) error
}

// Command defines the interface for command execution
type Command interface {
    // Start starts the command
    Start() error
    // Wait waits for the command to complete
    Wait() error
    // Process returns the underlying process
    Process() Process
    // SetOutput sets the stdout and stderr writers
    SetOutput(stdout, stderr io.Writer)
    // SetEnv sets the environment variables
    SetEnv(env []string)
    // SetDir sets the working directory
    SetDir(dir string)
    // SetSysProcAttr sets the syscall.SysProcAttr
    SetSysProcAttr(attr *syscall.SysProcAttr)
}

// Process defines the interface for OS processes
type Process interface {
    // Pid returns the process ID
    Pid() int
    // Signal sends a signal to the process
    Signal(sig os.Signal) error
}

// ServiceDiscoverer defines interface for service discovery
type ServiceDiscoverer interface {
    // DiscoverServices discovers available service binaries
    DiscoverServices(ctx context.Context) (map[string]string, error)
}

// ServiceState represents the state of a service process
type ServiceState interface {
    // Name returns the state name
    Name() string
    // Enter is called when the state is entered
    Enter(process *ServiceProcess)
    // Exit is called when the state is exited
    Exit(process *ServiceProcess)
    // CanTransitionTo checks if transition to the given state is valid
    CanTransitionTo(state ServiceState) bool
}

// ProcessError represents a service process error
type ProcessError struct {
    // Service is the name of the service
    Service string
    // Operation is the operation that failed
    Operation string
    // Err is the underlying error
    Err error
}

// Error implements the error interface
func (e *ProcessError) Error() string {
    if e.Err != nil {
        return "service '" + e.Service + "' " + e.Operation + " failed: " + e.Err.Error()
    }
    return "service '" + e.Service + "' " + e.Operation + " failed"
}

// Unwrap returns the underlying error
func (e *ProcessError) Unwrap() error {
    return e.Err
}

// Orchestrator manages service processes
type Orchestrator struct {
    // Configuration
    config *config.OrchestratorConfig
    
    // Service configurations
    services map[string]*config.ServiceConfig
    
    // Component dependencies
    procManager      ProcessManager
    serviceDiscoverer ServiceDiscoverer
    logger            *zap.Logger
    
    // Service processes
    processes      map[string]*ServiceProcess
    processLock    sync.RWMutex
    
    // Lifecycle management
    ctx        context.Context
    cancelFunc context.CancelFunc
    wg         sync.WaitGroup
    
    // Concrete states (for convenience)
    stateManager *ServiceStateManager
}

// ServiceProcess represents a managed service process
type ServiceProcess struct {
    // Service identification
    Name      string
    BinaryPath string
    
    // Process state
    command   Command
    pid       int
    state     ServiceState
    
    // Execution statistics
    startTime time.Time
    restarts  int
    lastError error
    
    // Context for cancellation
    ctx        context.Context
    cancelFunc context.CancelFunc
    
    // Internal fields
    lock      sync.RWMutex
    logger    *zap.Logger
    orchestrator *Orchestrator
}
```

### 2. Service States Implementation

```go
// Predefined service states
var (
    StoppedState     = &stoppedState{}
    StartingState    = &startingState{}
    RunningState     = &runningState{}
    FailedState      = &failedState{}
    RestartingState  = &restartingState{}
)

// ServiceStateManager manages service state transitions
type ServiceStateManager struct {
    // State transition map for validation
    transitions map[string]map[string]bool
}

// NewServiceStateManager creates a new service state manager
func NewServiceStateManager() *ServiceStateManager {
    manager := &ServiceStateManager{
        transitions: make(map[string]map[string]bool),
    }
    
    // Define valid state transitions
    manager.addTransition(StoppedState, StartingState)
    manager.addTransition(StartingState, RunningState)
    manager.addTransition(StartingState, FailedState)
    manager.addTransition(RunningState, FailedState)
    manager.addTransition(RunningState, StoppedState)
    manager.addTransition(RunningState, RestartingState)
    manager.addTransition(FailedState, RestartingState)
    manager.addTransition(FailedState, StoppedState)
    manager.addTransition(RestartingState, StartingState)
    manager.addTransition(RestartingState, StoppedState)
    
    return manager
}

// addTransition adds a valid state transition
func (m *ServiceStateManager) addTransition(from, to ServiceState) {
    if _, exists := m.transitions[from.Name()]; !exists {
        m.transitions[from.Name()] = make(map[string]bool)
    }
    m.transitions[from.Name()][to.Name()] = true
}

// IsValidTransition checks if a transition is valid
func (m *ServiceStateManager) IsValidTransition(from, to ServiceState) bool {
    if from == nil || to == nil {
        return false
    }
    
    if validTransitions, exists := m.transitions[from.Name()]; exists {
        return validTransitions[to.Name()]
    }
    return false
}

// State implementations

// stoppedState represents a stopped service
type stoppedState struct{}

func (s *stoppedState) Name() string { return "stopped" }

func (s *stoppedState) Enter(process *ServiceProcess) {
    process.logger.Info("Service entered stopped state",
        zap.String("service", process.Name))
}

func (s *stoppedState) Exit(process *ServiceProcess) {}

func (s *stoppedState) CanTransitionTo(state ServiceState) bool {
    return process.orchestrator.stateManager.IsValidTransition(s, state)
}

// startingState represents a starting service
type startingState struct{}

func (s *startingState) Name() string { return "starting" }

func (s *startingState) Enter(process *ServiceProcess) {
    process.logger.Info("Service entering starting state",
        zap.String("service", process.Name))
    process.startTime = time.Now()
}

func (s *startingState) Exit(process *ServiceProcess) {}

func (s *startingState) CanTransitionTo(state ServiceState) bool {
    return process.orchestrator.stateManager.IsValidTransition(s, state)
}

// runningState represents a running service
type runningState struct{}

func (s *runningState) Name() string { return "running" }

func (s *runningState) Enter(process *ServiceProcess) {
    process.logger.Info("Service entered running state",
        zap.String("service", process.Name),
        zap.Int("pid", process.pid))
}

func (s *runningState) Exit(process *ServiceProcess) {}

func (s *runningState) CanTransitionTo(state ServiceState) bool {
    return process.orchestrator.stateManager.IsValidTransition(s, state)
}

// failedState represents a failed service
type failedState struct{}

func (s *failedState) Name() string { return "failed" }

func (s *failedState) Enter(process *ServiceProcess) {
    process.logger.Error("Service entered failed state",
        zap.String("service", process.Name),
        zap.Error(process.lastError))
}

func (s *failedState) Exit(process *ServiceProcess) {}

func (s *failedState) CanTransitionTo(state ServiceState) bool {
    return process.orchestrator.stateManager.IsValidTransition(s, state)
}

// restartingState represents a service being restarted
type restartingState struct{}

func (s *restartingState) Name() string { return "restarting" }

func (s *restartingState) Enter(process *ServiceProcess) {
    process.logger.Info("Service entering restarting state",
        zap.String("service", process.Name),
        zap.Int("restart_count", process.restarts))
    process.restarts++
}

func (s *restartingState) Exit(process *ServiceProcess) {}

func (s *restartingState) CanTransitionTo(state ServiceState) bool {
    return process.orchestrator.stateManager.IsValidTransition(s, state)
}
```

### 3. Core Orchestrator Implementation

```go
// OrchestratorOption allows customizing the orchestrator
type OrchestratorOption func(*Orchestrator)

// WithProcessManager sets a custom process manager
func WithProcessManager(manager ProcessManager) OrchestratorOption {
    return func(o *Orchestrator) {
        o.procManager = manager
    }
}

// WithServiceDiscoverer sets a custom service discoverer
func WithServiceDiscoverer(discoverer ServiceDiscoverer) OrchestratorOption {
    return func(o *Orchestrator) {
        o.serviceDiscoverer = discoverer
    }
}

// WithLogger sets a custom logger
func WithLogger(logger *zap.Logger) OrchestratorOption {
    return func(o *Orchestrator) {
        o.logger = logger
    }
}

// NewOrchestrator creates a new process orchestrator
func NewOrchestrator(configManager *config.ConfigManager, options ...OrchestratorOption) (*Orchestrator, error) {
    // Get configuration
    cfg := configManager.GetConfig()
    
    // Create context for lifecycle management
    ctx, cancel := context.WithCancel(context.Background())
    
    // Create state manager
    stateManager := NewServiceStateManager()
    
    // Create base orchestrator
    o := &Orchestrator{
        config:       &cfg.Orchestrator,
        services:     cfg.Services,
        processes:    make(map[string]*ServiceProcess),
        ctx:          ctx,
        cancelFunc:   cancel,
        stateManager: stateManager,
    }
    
    // Apply options
    for _, option := range options {
        option(o)
    }
    
    // Set defaults for required dependencies
    if o.logger == nil {
        logger, err := zap.NewProduction()
        if err != nil {
            return nil, fmt.Errorf("failed to create default logger: %w", err)
        }
        o.logger = logger
    }
    
    if o.procManager == nil {
        o.procManager = NewDefaultProcessManager()
    }
    
    if o.serviceDiscoverer == nil {
        o.serviceDiscoverer = NewDefaultServiceDiscoverer(o.config.ServicesDir, o.logger)
    }
    
    // Validate services directory
    if !dirExists(o.config.ServicesDir) {
        return nil, fmt.Errorf("services directory not found: %s", o.config.ServicesDir)
    }
    
    // Subscribe to configuration changes
    configManager.SubscribeToChanges(func(newConfig *config.Config) {
        o.handleConfigChange(newConfig)
    })
    
    return o, nil
}

// Start starts the orchestrator
func (o *Orchestrator) Start() error {
    o.logger.Info("Starting process orchestrator")
    
    // Discover services
    services, err := o.serviceDiscoverer.DiscoverServices(o.ctx)
    if err != nil {
        o.logger.Warn("Service discovery failed", zap.Error(err))
        // Continue anyway as this is not fatal
    }
    
    // Create any missing service configurations
    for name, path := range services {
        if _, exists := o.services[name]; !exists {
            o.logger.Info("Creating default configuration for discovered service",
                zap.String("service", name),
                zap.String("path", path))
            
            o.services[name] = &config.ServiceConfig{
                Enabled:    true,
                BinaryPath: path,
                DataDir:    filepath.Join(o.config.ServicesDir, name, "data"),
                Args:       []string{},
            }
        }
    }
    
    // Start automatic services
    for name, cfg := range o.services {
        if cfg.Enabled {
            if err := o.StartService(name); err != nil {
                o.logger.Error("Failed to start service",
                    zap.String("service", name),
                    zap.Error(err))
                // Continue with other services
            }
        }
    }
    
    return nil
}

// Stop stops the orchestrator and all managed services
func (o *Orchestrator) Stop() error {
    o.logger.Info("Stopping process orchestrator")
    
    // Cancel context to stop all operations
    o.cancelFunc()
    
    // Create shutdown context with timeout
    shutdownCtx, cancel := context.WithTimeout(
        context.Background(),
        time.Duration(o.config.ShutdownTimeout) * time.Second,
    )
    defer cancel()
    
    // Get running services
    o.processLock.RLock()
    runningServices := []string{}
    for name, process := range o.processes {
        if process.state == RunningState {
            runningServices = append(runningServices, name)
        }
    }
    o.processLock.RUnlock()
    
    // Stop all services in parallel
    var wg sync.WaitGroup
    for _, name := range runningServices {
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
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        o.logger.Info("All services stopped gracefully")
    case <-shutdownCtx.Done():
        o.logger.Warn("Shutdown timeout exceeded, some services may not have stopped gracefully")
    }
    
    // Wait for all goroutines to finish
    o.wg.Wait()
    
    return nil
}

// StartService starts a specific service
func (o *Orchestrator) StartService(name string) error {
    o.logger.Debug("Starting service", zap.String("service", name))
    
    // Get service configuration
    serviceCfg, exists := o.services[name]
    if !exists {
        return &ProcessError{
            Service:   name,
            Operation: "start",
            Err:       fmt.Errorf("service configuration not found"),
        }
    }
    
    // Skip disabled services
    if !serviceCfg.Enabled {
        o.logger.Info("Skipping disabled service", zap.String("service", name))
        return nil
    }
    
    // Check if already running
    o.processLock.RLock()
    process, exists := o.processes[name]
    o.processLock.RUnlock()
    
    if exists && process.state == RunningState {
        o.logger.Info("Service already running", zap.String("service", name))
        return nil
    }
    
    // Create or reset process
    if !exists {
        // Create new service process
        process = &ServiceProcess{
            Name:         name,
            BinaryPath:   serviceCfg.BinaryPath,
            state:        StoppedState,
            logger:       o.logger.With(zap.String("service", name)),
            orchestrator: o,
        }
        
        // Create context for this process
        procCtx, procCancel := context.WithCancel(o.ctx)
        process.ctx = procCtx
        process.cancelFunc = procCancel
        
        // Store the process
        o.processLock.Lock()
        o.processes[name] = process
        o.processLock.Unlock()
    }
    
    // Launch the process
    return o.launchProcess(process, serviceCfg)
}

// StopService stops a specific service
func (o *Orchestrator) StopService(name string) error {
    o.logger.Debug("Stopping service", zap.String("service", name))
    
    // Find the process
    o.processLock.RLock()
    process, exists := o.processes[name]
    o.processLock.RUnlock()
    
    if !exists {
        return &ProcessError{
            Service:   name,
            Operation: "stop",
            Err:       fmt.Errorf("service not found"),
        }
    }
    
    // Cancel the process context
    process.cancelFunc()
    
    // Check if already stopped
    process.lock.RLock()
    isRunning := process.state == RunningState
    process.lock.RUnlock()
    
    if !isRunning {
        return nil
    }
    
    // Transition to stopped state
    process.lock.Lock()
    process.transitionTo(StoppedState)
    process.lock.Unlock()
    
    // Send SIGTERM
    o.logger.Info("Sending SIGTERM to service", 
        zap.String("service", name),
        zap.Int("pid", process.pid))
    
    if err := o.sendSignal(process, syscall.SIGTERM); err != nil {
        o.logger.Warn("Failed to send SIGTERM",
            zap.String("service", name),
            zap.Error(err))
    }
    
    // Wait for process to exit with timeout
    shutdownTimeout := time.Duration(o.config.ShutdownTimeout) * time.Second
    exitChan := make(chan error, 1)
    
    go func() {
        if cmd := process.command; cmd != nil {
            exitChan <- cmd.Wait()
        } else {
            exitChan <- nil
        }
    }()
    
    // Wait for exit or timeout
    select {
    case err := <-exitChan:
        if err != nil {
            o.logger.Warn("Error waiting for service to exit",
                zap.String("service", name),
                zap.Error(err))
        } else {
            o.logger.Info("Service exited gracefully",
                zap.String("service", name))
        }
    case <-time.After(shutdownTimeout):
        // Timeout occurred, force kill
        o.logger.Warn("Service did not exit gracefully, sending SIGKILL",
            zap.String("service", name),
            zap.Int("pid", process.pid))
        
        if err := o.sendSignal(process, syscall.SIGKILL); err != nil {
            o.logger.Error("Failed to send SIGKILL",
                zap.String("service", name),
                zap.Error(err))
            return &ProcessError{
                Service:   name,
                Operation: "stop",
                Err:       fmt.Errorf("failed to forcefully terminate: %w", err),
            }
        }
    }
    
    return nil
}

// RestartService restarts a specific service
func (o *Orchestrator) RestartService(name string) error {
    o.logger.Debug("Restarting service", zap.String("service", name))
    
    // Find the process
    o.processLock.RLock()
    process, exists := o.processes[name]
    o.processLock.RUnlock()
    
    if !exists {
        return &ProcessError{
            Service:   name,
            Operation: "restart",
            Err:       fmt.Errorf("service not found"),
        }
    }
    
    // Transition to restarting state
    process.lock.Lock()
    process.transitionTo(RestartingState)
    process.lock.Unlock()
    
    // Stop the service
    if err := o.StopService(name); err != nil {
        o.logger.Warn("Error stopping service during restart",
            zap.String("service", name),
            zap.Error(err))
        // Continue with restart anyway
    }
    
    // Start the service
    return o.StartService(name)
}

// launchProcess launches a service process
func (o *Orchestrator) launchProcess(process *ServiceProcess, cfg *config.ServiceConfig) error {
    // Get binary path
    binaryPath := cfg.BinaryPath
    if binaryPath == "" {
        // Default to binary in service directory: servicesDir/name/name
        binaryPath = filepath.Join(o.config.ServicesDir, process.Name, process.Name)
    }
    
    // Ensure binary exists
    if !fileExists(binaryPath) {
        return &ProcessError{
            Service:   process.Name,
            Operation: "launch",
            Err:       fmt.Errorf("binary not found at %s", binaryPath),
        }
    }
    
    // Update binary path
    process.BinaryPath = binaryPath
    
    // Transition to starting state
    process.lock.Lock()
    process.transitionTo(StartingState)
    process.lock.Unlock()
    
    // Build command-line arguments
    args := []string{"--service", process.Name}
    
    if o.config.LogLevel != "" {
        args = append(args, "--log-level", o.config.LogLevel)
    }
    
    // Add any additional service-specific arguments
    if len(cfg.Args) > 0 {
        args = append(args, cfg.Args...)
    }
    
    // Create command
    cmd := o.procManager.CreateCommand(binaryPath, args...)
    
    // Setup process output handling
    logWriters := newServiceLogWriters(process.Name, o.logger)
    cmd.SetOutput(logWriters.stdout, logWriters.stderr)
    
    // Setup process isolation and environment
    setupProcessIsolation(cmd, cfg)
    
    // Store the command
    process.lock.Lock()
    process.command = cmd
    process.lock.Unlock()
    
    // Start the process
    if err := cmd.Start(); err != nil {
        process.lock.Lock()
        process.lastError = err
        process.transitionTo(FailedState)
        process.lock.Unlock()
        
        return &ProcessError{
            Service:   process.Name,
            Operation: "launch",
            Err:       fmt.Errorf("failed to start: %w", err),
        }
    }
    
    // Update process details
    process.lock.Lock()
    process.pid = cmd.Process().Pid()
    process.lock.Unlock()
    
    // Begin supervision
    o.wg.Add(1)
    go o.superviseProcess(process)
    
    o.logger.Info("Service process launched",
        zap.String("service", process.Name),
        zap.Int("pid", process.pid),
        zap.String("binary", binaryPath))
    
    return nil
}

// superviseProcess monitors a service process and handles its lifecycle
func (o *Orchestrator) superviseProcess(process *ServiceProcess) {
    defer o.wg.Done()
    
    o.logger.Debug("Starting supervision for service",
        zap.String("service", process.Name))
    
    // Transition to running state
    process.lock.Lock()
    process.transitionTo(RunningState)
    process.lock.Unlock()
    
    // Create wait channel
    waitChan := make(chan error, 1)
    
    // Wait for the process in a goroutine
    go func() {
        if cmd := process.command; cmd != nil {
            waitChan <- cmd.Wait()
        } else {
            waitChan <- fmt.Errorf("nil command")
        }
    }()
    
    // Wait for either process exit or context cancellation
    select {
    case err := <-waitChan:
        // Process exited
        
        // Check if we're shutting down
        select {
        case <-process.ctx.Done():
            // This was an expected shutdown
            o.logger.Info("Service exited during shutdown",
                zap.String("service", process.Name))
            return
        default:
            // Unexpected exit
            exitCode := 0
            if err != nil {
                if exitErr, ok := err.(*exec.ExitError); ok {
                    exitCode = exitErr.ExitCode()
                }
            }
            
            o.logger.Warn("Service exited unexpectedly",
                zap.String("service", process.Name),
                zap.Int("exit_code", exitCode),
                zap.Error(err))
            
            // Update process state
            process.lock.Lock()
            process.lastError = err
            process.transitionTo(FailedState)
            process.lock.Unlock()
            
            // Check if restart is enabled
            if !o.config.AutoRestart {
                o.logger.Info("Auto-restart disabled, not restarting service",
                    zap.String("service", process.Name))
                return
            }
            
            // Handle restart with backoff
            o.handleServiceRestart(process)
        }
        
    case <-process.ctx.Done():
        // Process context was cancelled (service is being stopped)
        o.logger.Debug("Service supervision cancelled",
            zap.String("service", process.Name))
        return
    }
}

// handleServiceRestart implements restart with exponential backoff
func (o *Orchestrator) handleServiceRestart(process *ServiceProcess) {
    const (
        maxRestartAttempts = 10
        initialDelay      = 1 * time.Second
        maxDelay          = 30 * time.Second
    )
    
    // Check if we've reached the maximum restart attempts
    process.lock.RLock()
    restarts := process.restarts
    name := process.Name
    process.lock.RUnlock()
    
    if restarts >= maxRestartAttempts {
        o.logger.Error("Service reached maximum restart attempts, not restarting",
            zap.String("service", name),
            zap.Int("restarts", restarts))
        return
    }
    
    // Transition to restarting state
    process.lock.Lock()
    process.transitionTo(RestartingState)
    process.lock.Unlock()
    
    // Calculate backoff delay with exponential increase
    delayMs := int(math.Min(
        float64(initialDelay.Milliseconds()) * math.Pow(2, float64(restarts)),
        float64(maxDelay.Milliseconds()),
    ))
    
    // Add jitter (±10%)
    jitterRange := delayMs / 10
    if jitterRange > 0 {
        delayMs += rand.Intn(jitterRange*2) - jitterRange
    }
    
    delay := time.Duration(delayMs) * time.Millisecond
    
    o.logger.Info("Restarting service after backoff",
        zap.String("service", name),
        zap.Duration("backoff", delay),
        zap.Int("restart_count", restarts))
    
    // Wait with cancellation support
    select {
    case <-time.After(delay):
        // Get service config
        serviceCfg, exists := o.services[name]
        if !exists {
            o.logger.Error("Service configuration no longer exists",
                zap.String("service", name))
            return
        }
        
        // Launch the process again
        if err := o.launchProcess(process, serviceCfg); err != nil {
            o.logger.Error("Failed to restart service",
                zap.String("service", name),
                zap.Error(err))
        }
        
    case <-process.ctx.Done():
        // Process context was cancelled during backoff
        o.logger.Debug("Service restart cancelled during backoff",
            zap.String("service", name))
    }
}

// sendSignal sends a signal to a process
func (o *Orchestrator) sendSignal(process *ServiceProcess, sig syscall.Signal) error {
    process.lock.RLock()
    pid := process.pid
    process.lock.RUnlock()
    
    if pid <= 0 {
        return fmt.Errorf("invalid process ID")
    }
    
    // Get the process group ID
    pgid, err := o.procManager.GetProcessGroup(pid)
    if err != nil {
        return fmt.Errorf("failed to get process group: %w", err)
    }
    
    // Send signal to the process group
    if err := o.procManager.SendSignal(-pgid, sig); err != nil {
        return fmt.Errorf("failed to send signal: %w", err)
    }
    
    return nil
}

// handleConfigChange handles configuration changes
func (o *Orchestrator) handleConfigChange(newConfig *config.Config) {
    o.logger.Info("Handling configuration change")
    
    o.processLock.Lock()
    defer o.processLock.Unlock()
    
    // Update core configuration
    o.config = &newConfig.Orchestrator
    
    // Update service configurations
    o.services = newConfig.Services
    
    o.logger.Info("Configuration updated", 
        zap.Int("num_services", len(o.services)))
    
    // NOTE: In Phase 1, we don't automatically restart services on config change
    // This behavior can be added in a future phase if needed
}

// transitionTo transitions a service process to a new state
func (p *ServiceProcess) transitionTo(newState ServiceState) {
    if !p.state.CanTransitionTo(newState) {
        p.logger.Warn("Invalid state transition",
            zap.String("from", p.state.Name()),
            zap.String("to", newState.Name()))
        return
    }
    
    p.logger.Debug("State transition",
        zap.String("from", p.state.Name()),
        zap.String("to", newState.Name()))
    
    p.state.Exit(p)
    p.state = newState
    p.state.Enter(p)
}
```

### 4. Process Output Handling

```go
// serviceLogWriters contains writers for service process output
type serviceLogWriters struct {
    stdout io.Writer
    stderr io.Writer
}

// newServiceLogWriters creates new log writers for a service
func newServiceLogWriters(serviceName string, logger *zap.Logger) *serviceLogWriters {
    return &serviceLogWriters{
        stdout: newServiceLogWriter(serviceName, "stdout", false, logger),
        stderr: newServiceLogWriter(serviceName, "stderr", true, logger),
    }
}

// serviceLogWriter is an io.Writer for service logs
type serviceLogWriter struct {
    service  string
    stream   string
    isError  bool
    logger   *zap.Logger
    buffer   bytes.Buffer
    mutex    sync.Mutex
}

// newServiceLogWriter creates a new service log writer
func newServiceLogWriter(service, stream string, isError bool, logger *zap.Logger) *serviceLogWriter {
    return &serviceLogWriter{
        service: service,
        stream:  stream,
        isError: isError,
        logger:  logger,
    }
}

// Write implements io.Writer
func (w *serviceLogWriter) Write(p []byte) (n int, err error) {
    w.mutex.Lock()
    defer w.mutex.Unlock()
    
    // Keep track of bytes written
    n = len(p)
    
    // First, write to buffer
    _, err = w.buffer.Write(p)
    if err != nil {
        return 0, err
    }
    
    // Process complete lines
    for {
        line, err := w.buffer.ReadString('\n')
        if err == io.EOF {
            // Put back incomplete line
            w.buffer.WriteString(line)
            break
        }
        
        // Process the complete line
        line = strings.TrimSuffix(line, "\n")
        if line == "" {
            continue
        }
        
        // Log based on stream type
        if w.isError {
            w.logger.Error(line,
                zap.String("service", w.service),
                zap.String("stream", w.stream))
        } else {
            w.logger.Info(line,
                zap.String("service", w.service),
                zap.String("stream", w.stream))
        }
    }
    
    return n, nil
}
```

### 5. Process Isolation

```go
// setupProcessIsolation configures process isolation and resource limits
func setupProcessIsolation(cmd Command, cfg *config.ServiceConfig) {
    // Set process group to enable signaling the entire process tree
    cmd.SetSysProcAttr(&syscall.SysProcAttr{
        Setpgid: true,
    })
    
    // Create a clean environment
    cleanEnv := []string{
        "PATH=" + os.Getenv("PATH"),
        "HOME=" + cfg.DataDir,
        "TEMP=" + os.TempDir(),
        "TMP=" + os.TempDir(),
    }
    
    // Add service-specific environment variables
    if len(cfg.Environment) > 0 {
        for key, value := range cfg.Environment {
            cleanEnv = append(cleanEnv, fmt.Sprintf("%s=%s", key, value))
        }
    }
    
    // Add Go memory limit (for Go services)
    if cfg.MemoryLimit > 0 {
        cleanEnv = append(cleanEnv, fmt.Sprintf("GOMEMLIMIT=%dMiB", cfg.MemoryLimit))
    }
    
    // Set the clean environment
    cmd.SetEnv(cleanEnv)
    
    // Set working directory to service-specific directory
    if cfg.DataDir != "" {
        cmd.SetDir(cfg.DataDir)
    }
}
```

### 6. Default OS Implementations

```go
// DefaultProcessManager implements ProcessManager with OS operations
type DefaultProcessManager struct{}

// NewDefaultProcessManager creates a new DefaultProcessManager
func NewDefaultProcessManager() *DefaultProcessManager {
    return &DefaultProcessManager{}
}

// CreateCommand creates a new command
func (m *DefaultProcessManager) CreateCommand(path string, args ...string) Command {
    return &OSCommand{
        cmd: exec.Command(path, args...),
    }
}

// GetProcessGroup gets the process group ID for a PID
func (m *DefaultProcessManager) GetProcessGroup(pid int) (int, error) {
    return syscall.Getpgid(pid)
}

// SendSignal sends a signal to a process group
func (m *DefaultProcessManager) SendSignal(pgid int, signal syscall.Signal) error {
    return syscall.Kill(pgid, signal)
}

// OSCommand wraps exec.Cmd to implement Command
type OSCommand struct {
    cmd *exec.Cmd
}

// Start starts the command
func (c *OSCommand) Start() error {
    return c.cmd.Start()
}

// Wait waits for the command to complete
func (c *OSCommand) Wait() error {
    return c.cmd.Wait()
}

// Process returns the underlying process
func (c *OSCommand) Process() Process {
    if c.cmd.Process == nil {
        return nil
    }
    return &OSProcess{process: c.cmd.Process}
}

// SetOutput sets the stdout and stderr writers
func (c *OSCommand) SetOutput(stdout, stderr io.Writer) {
    c.cmd.Stdout = stdout
    c.cmd.Stderr = stderr
}

// SetEnv sets the environment variables
func (c *OSCommand) SetEnv(env []string) {
    c.cmd.Env = env
}

// SetDir sets the working directory
func (c *OSCommand) SetDir(dir string) {
    c.cmd.Dir = dir
}

// SetSysProcAttr sets the syscall.SysProcAttr
func (c *OSCommand) SetSysProcAttr(attr *syscall.SysProcAttr) {
    c.cmd.SysProcAttr = attr
}

// OSProcess wraps os.Process to implement Process
type OSProcess struct {
    process *os.Process
}

// Pid returns the process ID
func (p *OSProcess) Pid() int {
    return p.process.Pid
}

// Signal sends a signal to the process
func (p *OSProcess) Signal(sig os.Signal) error {
    return p.process.Signal(sig)
}

// DefaultServiceDiscoverer implements ServiceDiscoverer
type DefaultServiceDiscoverer struct {
    servicesDir string
    logger      *zap.Logger
}

// NewDefaultServiceDiscoverer creates a new DefaultServiceDiscoverer
func NewDefaultServiceDiscoverer(servicesDir string, logger *zap.Logger) *DefaultServiceDiscoverer {
    return &DefaultServiceDiscoverer{
        servicesDir: servicesDir,
        logger:      logger,
    }
}

// DiscoverServices discovers available service binaries
func (d *DefaultServiceDiscoverer) DiscoverServices(ctx context.Context) (map[string]string, error) {
    services := make(map[string]string)
    
    entries, err := os.ReadDir(d.servicesDir)
    if err != nil {
        return nil, fmt.Errorf("failed to read services directory: %w", err)
    }
    
    for _, entry := range entries {
        // Check for cancellation
        select {
        case <-ctx.Done():
            return services, ctx.Err()
        default:
            // Continue
        }
        
        if entry.IsDir() {
            serviceName := entry.Name()
            binaryPath := filepath.Join(d.servicesDir, serviceName, serviceName)
            
            // Check if binary exists and is executable
            if fileExists(binaryPath) && isExecutable(binaryPath) {
                services[serviceName] = binaryPath
                d.logger.Info("Discovered service binary",
                    zap.String("service", serviceName),
                    zap.String("path", binaryPath))
            }
        }
    }
    
    d.logger.Info("Service discovery completed",
        zap.Int("count", len(services)),
        zap.Strings("services", maps.Keys(services)))
    
    return services, nil
}

// Helper functions
func fileExists(path string) bool {
    info, err := os.Stat(path)
    return err == nil && !info.IsDir()
}

func isExecutable(path string) bool {
    info, err := os.Stat(path)
    if err != nil {
        return false
    }
    return info.Mode()&0111 != 0
}

func dirExists(path string) bool {
    info, err := os.Stat(path)
    return err == nil && info.IsDir()
}
```

## Testing Approach

The Process Orchestrator should be tested using a hybrid approach that combines unit tests with focused integration tests.

### 1. Unit Testing with Mocks

```go
// TestOrchestrator_StartService tests starting a service
func TestOrchestrator_StartService(t *testing.T) {
    // Create mock components
    mockProcManager := &MockProcessManager{
        CreateCommandFunc: func(path string, args ...string) Command {
            return &MockCommand{
                StartFunc: func() error { return nil },
                ProcessFunc: func() Process {
                    return &MockProcess{
                        PidFunc: func() int { return 1000 },
                    }
                },
            }
        },
    }
    
    mockDiscoverer := &MockServiceDiscoverer{
        DiscoverServicesFunc: func(ctx context.Context) (map[string]string, error) {
            return map[string]string{
                "test-service": "/path/to/test-service",
            }, nil
        },
    }
    
    logger := zap.NewNop()
    
    // Create test configuration
    cfg := &config.Config{
        Orchestrator: config.OrchestratorConfig{
            ServicesDir:      "/tmp/services",
            LogLevel:         "info",
            AutoRestart:      true,
            ShutdownTimeout:  10,
        },
        Services: map[string]*config.ServiceConfig{
            "test-service": {
                Enabled:    true,
                BinaryPath: "/path/to/test-service",
                DataDir:    "/tmp/services/test-service/data",
                Args:       []string{},
            },
        },
    }
    
    configManager := &MockConfigManager{
        GetConfigFunc: func() *config.Config {
            return cfg
        },
        SubscribeToChangesFunc: func(callback func(*config.Config)) {
            // No-op
        },
    }
    
    // Create orchestrator with mocks
    orchestrator, err := NewOrchestrator(
        configManager,
        WithProcessManager(mockProcManager),
        WithServiceDiscoverer(mockDiscoverer),
        WithLogger(logger),
    )
    require.NoError(t, err)
    
    // Test starting a service
    err = orchestrator.StartService("test-service")
    assert.NoError(t, err)
    
    // Verify process was properly tracked
    process, exists := orchestrator.getProcess("test-service")
    assert.True(t, exists)
    assert.Equal(t, RunningState, process.state)
    assert.Equal(t, 1000, process.pid)
}
```

### 2. Integration Tests with Real Processes

```go
// buildTestService builds a test service binary
func buildTestService(t *testing.T, name string, behavior string) string {
    // Create a temporary directory for the test service
    tempDir, err := os.MkdirTemp("", "blackhole-test-")
    require.NoError(t, err)
    
    // Define the test service source code
    source := fmt.Sprintf(`
package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    fmt.Printf("Test service %s starting\\n")
    
    // Handle signals
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    
    // Different behaviors based on service type
    switch %q {
    case "normal":
        fmt.Println("Normal service running")
        
        // Wait for signal
        sig := <-sigCh
        fmt.Printf("Received signal %%s, shutting down\\n", sig)
        
    case "crash":
        fmt.Println("Crash service running, will exit in 1 second")
        time.Sleep(1 * time.Second)
        os.Exit(1)
        
    case "hang":
        fmt.Println("Hang service running, will ignore SIGTERM")
        
        // Wait for signal
        sig := <-sigCh
        fmt.Printf("Received signal %%s, ignoring\\n", sig)
        
        // Ignore SIGTERM and wait for SIGKILL
        if sig == syscall.SIGTERM {
            time.Sleep(10 * time.Second)
        }
    }
}
`, name, behavior)
    
    // Write source to a temporary file
    sourceFile := filepath.Join(tempDir, "main.go")
    err = os.WriteFile(sourceFile, []byte(source), 0644)
    require.NoError(t, err)
    
    // Build the service
    binaryPath := filepath.Join(tempDir, name)
    cmd := exec.Command("go", "build", "-o", binaryPath, sourceFile)
    err = cmd.Run()
    require.NoError(t, err)
    
    // Make binary executable
    err = os.Chmod(binaryPath, 0755)
    require.NoError(t, err)
    
    return binaryPath
}

// IntegrationTest_ProcessOrchestrator tests the complete orchestrator with real processes
func IntegrationTest_ProcessOrchestrator(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Create test services
    normalService := buildTestService(t, "normal-service", "normal")
    crashService := buildTestService(t, "crash-service", "crash")
    hangService := buildTestService(t, "hang-service", "hang")
    
    // Create test services directory
    servicesDir, err := os.MkdirTemp("", "blackhole-services-")
    require.NoError(t, err)
    defer os.RemoveAll(servicesDir)
    
    // Create individual service directories
    for _, name := range []string{"normal-service", "crash-service", "hang-service"} {
        serviceDir := filepath.Join(servicesDir, name)
        err := os.Mkdir(serviceDir, 0755)
        require.NoError(t, err)
        
        // Create data directory
        dataDir := filepath.Join(serviceDir, "data")
        err = os.Mkdir(dataDir, 0755)
        require.NoError(t, err)
    }
    
    // Copy binaries to service directories
    err = os.Rename(normalService, filepath.Join(servicesDir, "normal-service", "normal-service"))
    require.NoError(t, err)
    
    err = os.Rename(crashService, filepath.Join(servicesDir, "crash-service", "crash-service"))
    require.NoError(t, err)
    
    err = os.Rename(hangService, filepath.Join(servicesDir, "hang-service", "hang-service"))
    require.NoError(t, err)
    
    // Create test configuration
    cfg := &config.Config{
        Orchestrator: config.OrchestratorConfig{
            ServicesDir:      servicesDir,
            LogLevel:         "info",
            AutoRestart:      true,
            ShutdownTimeout:  3, // Short timeout for tests
        },
        Services: map[string]*config.ServiceConfig{
            "normal-service": {
                Enabled:    true,
                DataDir:    filepath.Join(servicesDir, "normal-service", "data"),
                Args:       []string{},
            },
            "crash-service": {
                Enabled:    true,
                DataDir:    filepath.Join(servicesDir, "crash-service", "data"),
                Args:       []string{},
            },
            "hang-service": {
                Enabled:    false, // Start disabled
                DataDir:    filepath.Join(servicesDir, "hang-service", "data"),
                Args:       []string{},
            },
        },
    }
    
    configManager := &MockConfigManager{
        GetConfigFunc: func() *config.Config {
            return cfg
        },
        SubscribeToChangesFunc: func(callback func(*config.Config)) {
            // No-op for testing
        },
    }
    
    // Create test logger
    logger, err := zap.NewDevelopment()
    require.NoError(t, err)
    
    // Create orchestrator
    orchestrator, err := NewOrchestrator(
        configManager,
        WithLogger(logger),
    )
    require.NoError(t, err)
    
    // Start orchestrator
    err = orchestrator.Start()
    require.NoError(t, err)
    
    // Verify normal service is running
    time.Sleep(100 * time.Millisecond) // Allow time for processes to start
    process, exists := orchestrator.getProcess("normal-service")
    assert.True(t, exists)
    assert.Equal(t, RunningState, process.state)
    
    // Verify crash service is restarted automatically
    time.Sleep(2 * time.Second) // Give time for crash and restart
    crashProcess, exists := orchestrator.getProcess("crash-service")
    assert.True(t, exists)
    assert.Equal(t, RunningState, crashProcess.state)
    assert.Greater(t, crashProcess.restarts, 0)
    
    // Verify hang service is not running (disabled)
    _, exists = orchestrator.getProcess("hang-service")
    assert.False(t, exists)
    
    // Start hang service manually
    err = orchestrator.StartService("hang-service")
    assert.NoError(t, err)
    
    hangProcess, exists := orchestrator.getProcess("hang-service")
    assert.True(t, exists)
    assert.Equal(t, RunningState, hangProcess.state)
    
    // Stop orchestrator
    err = orchestrator.Stop()
    assert.NoError(t, err)
    
    // Verify all processes have been stopped
    for _, name := range []string{"normal-service", "crash-service", "hang-service"} {
        process, exists := orchestrator.getProcess(name)
        if exists {
            assert.Equal(t, StoppedState, process.state)
        }
    }
}
```

## Implementation Steps

1. Define core interfaces and types (ProcessManager, Command, ServiceState, etc.)
2. Implement state pattern for process state management
3. Create the Orchestrator struct and constructor with options pattern
4. Implement service process output handling with buffered line processing
5. Develop process isolation and environment setup
6. Implement supervision with exponential backoff restart
7. Add proper signal handling and graceful termination
8. Create service discovery with context cancellation support
9. Develop unit tests with mocks for fast feedback
10. Create integration tests with real processes for validation
11. Implement default OS-specific implementations

## Future Enhancements

Future enhancements (for later phases) include:

1. **Advanced Resource Management**:
   - CPU quota enforcement
   - More precise memory limits

2. **Enhanced Security**:
   - Binary verification
   - Privilege dropping

3. **Dependency-Aware Process Management**:
   - Service dependencies
   - Start ordering based on dependency graph

4. **Advanced Health Checking**:
   - Protocol-based health checks
   - Liveness and readiness probes