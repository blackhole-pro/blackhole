# Process Orchestrator Implementation Update

*Updated: May 21, 2025*

## Overview

This document provides an updated implementation plan for the Process Orchestrator component, addressing the gap between the current implementation and the design documentation. The Process Orchestrator is a critical component that manages service processes in the Blackhole platform.

## Implementation Status Analysis

After reviewing the current codebase and documentation, we found a significant gap between the implementation document and the actual code. The orchestrator.go file is missing several critical methods that are expected by the tests and required by the types/types.go interface.

### What's Currently Implemented

1. Core interfaces and types in `/internal/core/process/types/types.go`:
   - ProcessState enum for representing process states
   - ProcessManager interface defining lifecycle operations
   - ProcessExecutor interface for abstracting process execution
   - ProcessCmd interface for abstracting command operations
   - Process interface for abstracting OS process operations
   - ServiceInfo struct for diagnostic information

2. Implementation in `/internal/core/process/orchestrator.go`:
   - Constructor (NewOrchestrator)
   - handleConfigChange method
   - DiscoverServices method
   - Helper functions for file operations
   - Logger initialization

### What's Missing

1. Core Methods Required by ProcessManager Interface:
   - Start(name string) - Start a specific service
   - Stop(name string) - Stop a specific service
   - Restart(name string) - Restart a specific service
   - Status(name string) - Get the status of a specific service
   - IsRunning(name string) - Check if a specific service is running

2. Additional Methods Referenced in Tests:
   - StartService(name string) - Alternative version of Start
   - StopService(name string) - Alternative version of Stop
   - RestartService(name string) - Alternative version of Restart
   - GetServiceInfo(name string) - Get detailed service information
   - GetAllServices() - Get information about all services
   - SpawnService(name string) - Spawn a new service process
   - supervise(name string, stopCh chan struct{}) - Supervise a service process
   - Shutdown(ctx context.Context) - Shutdown all services
   - setupSignals() - Setup signal handling
   - calculateBackoffDelay(count int) - Calculate backoff delay

## Implementation Plan

This document provides the implementation for all missing methods according to the design document.

### 1. Core Start/Stop/Restart Methods

```go
// Start starts a specific service by name (implements ProcessManager interface)
func (o *Orchestrator) Start(name string) error {
    o.logger.Info("Starting service", zap.String("service", name))
    return o.StartService(name)
}

// Stop stops a running service (implements ProcessManager interface)
func (o *Orchestrator) Stop(name string) error {
    o.logger.Info("Stopping service", zap.String("service", name))
    return o.StopService(name)
}

// Restart restarts a service (implements ProcessManager interface)
func (o *Orchestrator) Restart(name string) error {
    o.logger.Info("Restarting service", zap.String("service", name))
    return o.RestartService(name)
}
```

### 2. Status and Info Methods

```go
// Status gets the current state of a service (implements ProcessManager interface)
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

// IsRunning checks if a service is running (implements ProcessManager interface)
func (o *Orchestrator) IsRunning(name string) bool {
    state, err := o.Status(name)
    if err != nil {
        return false
    }
    return state == types.ProcessStateRunning
}
```

### 3. Service Management Methods

```go
// StartService starts a specific service
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
    
    if exists && process.State == types.ProcessStateRunning {
        o.logger.Info("Service already running", zap.String("service", name))
        return nil
    }
    
    // Spawn the service process
    return o.SpawnService(name)
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
    if process.State == types.ProcessStateStopped {
        o.processLock.Unlock()
        return nil
    }
    
    // Update status and get stop channel
    process.State = types.ProcessStateStopped
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
            return &types.ProcessError{
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
        process.State = types.ProcessStateRestarting
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
    
    // Setup process output handling
    setupProcessOutput(cmd, name, o.logger)
    
    // Setup process attributes for isolation
    setupProcessIsolation(cmd, serviceCfg)
    
    // Start the process
    if err := cmd.Start(); err != nil {
        return &types.ProcessError{
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
        process.State = types.ProcessStateRunning
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
        process.State = types.ProcessStateFailed
        process.LastError = &types.ProcessError{
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
```

### 5. Shutdown and Signal Handling

```go
// Shutdown gracefully shuts down the orchestrator with context
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
        return fmt.Errorf("shutdown context canceled: %w", ctx.Done())
    }
    
    // Check for errors
    close(errCh)
    var errs []string
    for err := range errCh {
        errs = append(errs, err.Error())
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("errors during shutdown: %s", strings.Join(errs, "; "))
    }
    
    // Close channels
    close(o.doneCh)
    
    return nil
}

// Stop implements stopping all services (compatibility with older interface)
func (o *Orchestrator) Stop() error {
    ctx, cancel := context.WithTimeout(
        context.Background(),
        time.Duration(o.config.ShutdownTimeout) * time.Second,
    )
    defer cancel()
    
    return o.Shutdown(ctx)
}

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
```

### 6. Service Information

```go
// GetServiceInfo returns diagnostic information about a service
func (o *Orchestrator) GetServiceInfo(name string) (*types.ServiceInfo, error) {
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
        return &types.ServiceInfo{
            Name:      name,
            Configured: true,
            Enabled:   serviceCfg.Enabled,
            State:     string(types.ProcessStateStopped),
        }, nil
    }
    
    // Build service info
    info := &types.ServiceInfo{
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
        if procErr, ok := process.LastError.(*types.ProcessError); ok {
            info.LastExitCode = procErr.ExitCode
            info.LastError = procErr.Error()
        } else {
            info.LastError = process.LastError.Error()
        }
    }
    
    return info, nil
}

// GetAllServices returns information about all services
func (o *Orchestrator) GetAllServices() (map[string]*types.ServiceInfo, error) {
    o.processLock.RLock()
    defer o.processLock.RUnlock()
    
    services := make(map[string]*types.ServiceInfo)
    
    // Add all configured services
    for name, cfg := range o.services {
        info := &types.ServiceInfo{
            Name:       name,
            Configured: true,
            Enabled:    cfg.Enabled,
            State:      string(types.ProcessStateStopped),
        }
        
        // Add process info if running
        if process, exists := o.processes[name]; exists {
            info.State = string(process.State)
            info.PID = process.PID
            info.Uptime = time.Since(process.Started)
            info.Restarts = process.Restarts
            
            // Add error info if available
            if process.LastError != nil {
                if procErr, ok := process.LastError.(*types.ProcessError); ok {
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
```

### 7. Helper Functions

```go
// sendSignal sends a signal to a process
func (o *Orchestrator) sendSignal(name string, sig syscall.Signal) error {
    o.processLock.RLock()
    process, exists := o.processes[name]
    if !exists || process.Command == nil || process.PID <= 0 {
        o.processLock.RUnlock()
        return fmt.Errorf("process %s not found or not running", name)
    }
    cmd := process.Command
    o.processLock.RUnlock()
    
    return cmd.Signal(sig)
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

// setupProcessOutput configures stdout/stderr handling for the process
func setupProcessOutput(cmd types.ProcessCmd, serviceName string, logger *zap.Logger) {
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

// setupProcessIsolation configures process isolation and resource limits
func setupProcessIsolation(cmd types.ProcessCmd, serviceCfg *config.ServiceConfig) {
    // Set working directory if specified
    if serviceCfg.DataDir != "" {
        cmd.SetDir(serviceCfg.DataDir)
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
}
```

## Implementation Recommendations

1. **Complete the Implementation**: Implement all missing methods in orchestrator.go following the code provided in this document.

2. **Ensure Interface Compliance**: Verify that the Orchestrator struct properly implements the ProcessManager interface from types.go.

3. **Add Missing Tests**: Create additional tests for any newly implemented methods that aren't covered by existing tests.

4. **Update Documentation**: Update the process orchestrator implementation document to reflect these changes and the actual implementation.

5. **Address Process Recovery Mechanism Gap**: Implement the process recovery mechanism identified in the core architecture audit to ensure proper recovery after orchestrator restarts.

## Next Steps

After implementing these missing methods, the next steps should be:

1. Complete unit and integration tests for the Process Orchestrator
2. Implement the Configuration System component
3. Begin implementing Service Mesh components (Router, EventBus, Middleware)
4. Enhance the Build System
5. Add Configuration UI

By addressing these gaps, the Process Orchestrator will be fully functional and aligned with the design documentation, which is critical for the success of the Blackhole platform's subprocess architecture.