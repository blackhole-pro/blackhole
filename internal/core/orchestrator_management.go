package core

import (
	"fmt"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

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
	serviceNames := make([]string, 0, len(o.services))
	for name, cfg := range o.services {
		if cfg.Enabled {
			serviceNames = append(serviceNames, name)
		}
	}
	o.processLock.RUnlock()
	
	// Start each enabled service in parallel
	for _, serviceName := range serviceNames {
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
			len(startErrors), StringJoin(startErrors, "; "))
	}
	
	return nil
}

// Stop gracefully shuts down the orchestrator
func (o *Orchestrator) Stop() error {
	o.logger.Info("Stopping all services and shutting down orchestrator")
	
	// Mark as shutting down to prevent restarts
	o.isShuttingDown.Store(true)
	
	// Create a context with timeout for graceful shutdown
	shutdownTimeout := time.Duration(o.config.ShutdownTimeout) * time.Second
	shutdownDeadline := time.Now().Add(shutdownTimeout)
	
	// Get a list of all running services
	o.processLock.RLock()
	runningServices := make([]string, 0, len(o.processes))
	for name, process := range o.processes {
		if process.State == ProcessStateRunning || process.State == ProcessStateStarting {
			runningServices = append(runningServices, name)
		}
	}
	o.processLock.RUnlock()
	
	// Stop all services in parallel with the same deadline
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
	stopDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(stopDone)
	}()
	
	// Wait for either completion or timeout
	select {
	case <-stopDone:
		o.logger.Info("All services stopped gracefully")
	case <-time.After(time.Until(shutdownDeadline)):
		o.logger.Warn("Shutdown grace period exceeded, some services may not have stopped gracefully")
	}
	
	// Close channels and resources
	close(o.doneCh)
	
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
	shutdownTimeout := time.Duration(o.config.ShutdownTimeout) * time.Second
	
	go func() {
		o.processLock.RLock()
		proc, exists := o.processes[name]
		if !exists || proc.Command == nil {
			o.processLock.RUnlock()
			exitChan <- nil
			return
		}
		o.processLock.RUnlock()
		
		exitErr := proc.Command.Wait()
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
	case <-time.After(shutdownTimeout):
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

// sendSignal sends a signal to a process
func (o *Orchestrator) sendSignal(name string, sig syscall.Signal) error {
	o.processLock.RLock()
	process, exists := o.processes[name]
	if !exists || process.Command == nil || process.Command.Process() == nil {
		o.processLock.RUnlock()
		return fmt.Errorf("process %s not found or not running", name)
	}
	
	proc := process.Command.Process()
	o.processLock.RUnlock()
	
	if proc == nil {
		return fmt.Errorf("nil process for service %s", name)
	}
	
	return proc.Signal(sig)
}

// StringJoin joins strings with a separator
func StringJoin(elements []string, separator string) string {
	if len(elements) == 0 {
		return ""
	}
	
	result := elements[0]
	for i := 1; i < len(elements); i++ {
		result += separator + elements[i]
	}
	
	return result
}