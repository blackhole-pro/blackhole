package core

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"
)

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
	
	// Create data directory if it doesn't exist
	if serviceCfg.DataDir != "" {
		if err := os.MkdirAll(serviceCfg.DataDir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
	}
	
	// Build command-line arguments
	args := []string{"--service", name}
	
	if o.config.LogLevel != "" {
		args = append(args, "--log-level", o.config.LogLevel)
	}
	
	// Add socket directory for IPC
	if o.config.SocketDir != "" {
		args = append(args, "--socket-dir", o.config.SocketDir)
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
			Service:  name,
			Err:      fmt.Errorf("failed to start: %w", err),
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
		exitCode = getExitCode(exitErr)
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

// setupProcessIsolation configures process isolation and resource limits
func setupProcessIsolation(cmd ProcessCmd, serviceCfg *ServiceConfig) {
	// Set process group ID and other system attributes
	if cmd, ok := cmd.(*DefaultProcessCmd); ok && cmd.cmd != nil {
		cmd.cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true, // Create new process group for better signal handling
		}
	}
	
	// Create a clean environment
	cleanEnv := []string{
		"PATH=" + os.Getenv("PATH"),
	}
	
	// Add data directory to environment
	if serviceCfg.DataDir != "" {
		cleanEnv = append(cleanEnv, "HOME="+serviceCfg.DataDir)
	}
	
	// Add temp directories
	cleanEnv = append(cleanEnv, 
		"TEMP="+os.TempDir(),
		"TMP="+os.TempDir())
	
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

// getExitCode extracts the exit code from an error
func getExitCode(err error) int {
	if err == nil {
		return 0
	}
	
	// Try to extract exit code from error
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	
	// Default to -1 for unknown errors
	return -1
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