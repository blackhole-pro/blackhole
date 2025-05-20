// Package process provides the implementation of the Process Orchestrator,
// which is the central component responsible for managing service processes
// in the Blackhole platform. It handles service discovery, process spawning,
// lifecycle management, supervision with automatic restart, resource isolation,
// and structured output handling.
//
// The orchestrator follows a component-based architecture with specialized
// subpackages for different responsibilities, providing a clean separation
// of concerns and improved testability. It uses interface-based design
// patterns to abstract OS-level functionality and enable comprehensive testing.
//
// Key responsibilities include:
// - Discovering and managing service binaries in the services directory
// - Spawning and monitoring service processes using their individual binaries
// - Implementing robust restart logic with exponential backoff
// - Managing process lifecycle (start, stop, restart)
// - Providing process isolation and resource controls
// - Handling structured output logging from processes
package process

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/handcraftdev/blackhole/internal/core/config"
	configtypes "github.com/handcraftdev/blackhole/internal/core/config/types"
	"github.com/handcraftdev/blackhole/internal/core/process/executor"
	"github.com/handcraftdev/blackhole/internal/core/process/isolation"
	"github.com/handcraftdev/blackhole/internal/core/process/output"
	"github.com/handcraftdev/blackhole/internal/core/process/service"
	"github.com/handcraftdev/blackhole/internal/core/process/supervision"
	"github.com/handcraftdev/blackhole/internal/core/process/types"
	"go.uber.org/zap"
)

// ServiceProcess represents a running service process with state management
type ServiceProcess struct {
	Name        string
	Command     types.ProcessCmd
	CommandWait func() error
	PID         int
	State       types.ProcessState
	Started     time.Time
	Restarts    int
	LastError   error
	StopCh      chan struct{}
}

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

// OrchestratorOption is a functional option type that allows configuring the
// Orchestrator with optional settings during initialization. This approach
// provides a flexible and extensible way to set dependencies and configuration.
type OrchestratorOption func(*Orchestrator)

// WithLogger sets a custom structured logger for the Orchestrator.
// If not provided, the Orchestrator will create a default logger based on
// the configured log level. The logger is used for all operational logging
// and is also passed to child components and services.
//
// Example:
//
//   logger, _ := zap.NewProduction()
//   orch, err := NewOrchestrator(configManager, WithLogger(logger))
func WithLogger(logger *zap.Logger) OrchestratorOption {
	return func(o *Orchestrator) {
		o.logger = logger
	}
}

// WithExecutor sets a custom process executor implementation.
// The executor is responsible for creating and managing OS processes.
// This option is particularly useful for testing, where you might want
// to provide a mock executor that doesn't actually spawn real processes.
//
// If not provided, a default executor that uses os/exec will be used.
//
// Example:
//
//   mockExec := testing.NewMockExecutor()
//   orch, err := NewOrchestrator(configManager, WithExecutor(mockExec))
func WithExecutor(exec types.ProcessExecutor) OrchestratorOption {
	return func(o *Orchestrator) {
		o.executor = exec
	}
}

// NewOrchestrator creates a new Process Orchestrator instance.
//
// It initializes the orchestrator with the provided configuration manager,
// sets up signal handling, verifies the services directory exists, and
// subscribes to configuration changes. The orchestrator is responsible for
// discovering service binaries, spawning processes, and managing their lifecycle.
//
// The configManager provides the initial configuration and notifies of changes.
// Optional OrchestratorOptions can be provided to customize the orchestrator,
// such as providing a custom logger or process executor.
//
// Returns an initialized Orchestrator and any error that occurred during setup.
//
// Example:
//
//   orch, err := NewOrchestrator(configManager, WithLogger(customLogger))
//   if err != nil {
//     // handle error
//   }
//   // use orchestrator
func NewOrchestrator(configManager *config.ConfigManager, options ...OrchestratorOption) (*Orchestrator, error) {
	// Get complete configuration 
	cfg := configManager.GetConfig()
	
	// Initialize orchestrator with configuration
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
	
	// Initialize service manager and info provider
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
	
	// Verify services directory exists
	if !dirExists(o.config.ServicesDir) {
		return nil, fmt.Errorf("services directory not found: %s", o.config.ServicesDir)
	}
	
	// Subscribe to configuration changes
	configManager.SubscribeToChanges(func(newConfig *configtypes.Config) {
		o.handleConfigChange(newConfig)
	})
	
	return o, nil
}

// SpawnProcess implements the ProcessSpawner interface
func (o *Orchestrator) SpawnProcess(name string) error {
	return o.SpawnService(name)
}

// handleConfigChange updates orchestrator with new configuration.
//
// This method is called when the configuration manager detects a configuration change.
// It updates the orchestrator's configuration, handles service removal, and updates
// service configurations. If a service is removed from configuration but still running,
// it will be stopped asynchronously.
//
// Parameters:
//   - newConfig: The new configuration to apply
func (o *Orchestrator) handleConfigChange(newConfig *configtypes.Config) {
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
			if exists && process.State != types.ProcessStateStopped {
				// Schedule async stop to avoid deadlock (we already hold the lock)
				go func(serviceName string) {
					if err := o.Stop(serviceName); err != nil {
						o.logger.Error("Failed to stop removed service", 
							zap.String("service", serviceName),
							zap.Error(err))
					}
				}(name)
			}
		}
	}
	
	// Update service configurations
	o.services = make(map[string]*configtypes.ServiceConfig)
	for name, svcCfg := range newConfig.Services {
		o.services[name] = svcCfg
	}
	
	o.logger.Info("Configuration updated", 
		zap.Int("num_services", len(o.services)))
}

// Start starts a specific service by name.
//
// This method implements the ProcessManager interface and provides the public API
// for starting services. It delegates to the StartService method for the actual
// implementation. The service must be defined in the configuration.
//
// If the service is already running, this is a no-op and returns nil.
// If the service is disabled in configuration, it will be skipped and return nil.
//
// Parameters:
//   - name: The name of the service to start
//
// Returns:
//   - error: Any error that occurred during the start operation
//
// Example:
//
//   err := orchestrator.Start("identity")
//   if err != nil {
//     // handle error
//   }
func (o *Orchestrator) Start(name string) error {
	o.logger.Info("Starting service", zap.String("service", name))
	return o.StartService(name)
}

// Stop stops a running service.
//
// This method implements the ProcessManager interface and provides the public API
// for stopping services. It delegates to the StopService method for the actual
// implementation.
//
// The method sends a SIGTERM signal to the service process and waits for it to exit
// gracefully. If the service doesn't exit within the configured shutdown timeout,
// it will be forcefully terminated with SIGKILL.
//
// Parameters:
//   - name: The name of the service to stop
//
// Returns:
//   - error: Any error that occurred during the stop operation
//
// Example:
//
//   err := orchestrator.Stop("identity")
//   if err != nil {
//     // handle error
//   }
func (o *Orchestrator) Stop(name string) error {
	o.logger.Info("Stopping service", zap.String("service", name))
	return o.StopService(name)
}

// Restart restarts a service.
//
// This method implements the ProcessManager interface and provides the public API
// for restarting services. It delegates to the RestartService method for the actual
// implementation.
//
// The method stops the service if it's running and then starts it again. If the service
// is not running, it will just start it.
//
// Parameters:
//   - name: The name of the service to restart
//
// Returns:
//   - error: Any error that occurred during the restart operation
//
// Example:
//
//   err := orchestrator.Restart("identity")
//   if err != nil {
//     // handle error
//   }
func (o *Orchestrator) Restart(name string) error {
	o.logger.Info("Restarting service", zap.String("service", name))
	return o.RestartService(name)
}

// Status gets the current state of a service.
//
// This method implements the ProcessManager interface and returns the process state
// for a given service. Possible states include:
// - ProcessStateStopped: Service is not running
// - ProcessStateStarting: Service is in the process of starting
// - ProcessStateRunning: Service is running normally
// - ProcessStateFailed: Service has failed and is not running
// - ProcessStateRestarting: Service is being restarted
//
// Parameters:
//   - name: The name of the service to check
//
// Returns:
//   - ProcessState: The current state of the service
//   - error: If the service doesn't exist or an error occurs during the operation
//
// Example:
//
//   state, err := orchestrator.Status("identity")
//   if err != nil {
//     // handle error
//   }
//   if state == types.ProcessStateRunning {
//     // service is running
//   }
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

// IsRunning checks if a service is running.
//
// This method implements the ProcessManager interface and provides a convenient
// way to check if a service is in the running state. It's a shorthand for checking
// if the Status method returns ProcessStateRunning.
//
// Parameters:
//   - name: The name of the service to check
//
// Returns:
//   - bool: true if the service is running, false otherwise
//
// Example:
//
//   if orchestrator.IsRunning("identity") {
//     // service is running
//   }
func (o *Orchestrator) IsRunning(name string) bool {
	state, err := o.Status(name)
	if err != nil {
		return false
	}
	return state == types.ProcessStateRunning
}

// StartService starts a specific service by name.
//
// This is an internal implementation method that delegates to the service manager.
// It checks if the service is configured and enabled, then spawns the process if needed.
// Public API users should use the Start method instead.
//
// Parameters:
//   - name: The name of the service to start
//
// Returns:
//   - error: Any error that occurred during the start operation
func (o *Orchestrator) StartService(name string) error {
	return o.serviceManager.StartService(name, func(serviceName string) error {
		return o.SpawnService(serviceName)
	})
}

// StopService stops a running service with graceful shutdown.
//
// This is an internal implementation method that delegates to the service manager.
// It handles the graceful shutdown process with SIGTERM and SIGKILL fallback.
// Public API users should use the Stop method instead.
//
// Parameters:
//   - name: The name of the service to stop
//
// Returns:
//   - error: Any error that occurred during the stop operation
func (o *Orchestrator) StopService(name string) error {
	return o.serviceManager.StopService(name, o.sendSignal, o.config.ShutdownTimeout)
}

// RestartService restarts a running service by stopping and starting it.
//
// This is an internal implementation method that delegates to the service manager.
// It first stops the service if it's running, then starts it again.
// Public API users should use the Restart method instead.
//
// Parameters:
//   - name: The name of the service to restart
//
// Returns:
//   - error: Any error that occurred during the restart operation
func (o *Orchestrator) RestartService(name string) error {
	return o.serviceManager.RestartService(name, o.StopService, o.StartService)
}

// SpawnService starts a new service process by creating a new OS process.
//
// This method handles the actual process spawning, environment setup, output handling,
// and supervision initialization. It's called by StartService and is not part of the
// public API.
//
// The method:
// 1. Verifies that the service is configured and the binary exists
// 2. Sets up command-line arguments and environment
// 3. Creates a process record and starts the OS process
// 4. Sets up process output handling and isolation
// 5. Initializes supervision in a separate goroutine
//
// Parameters:
//   - name: The name of the service to spawn
//
// Returns:
//   - error: Any error that occurred during the spawn operation
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

// sendSignal sends a signal to a service process.
//
// This helper method sends an OS signal like SIGTERM or SIGKILL to a running service.
// It's used by the StopService method to gracefully stop or forcefully terminate processes.
//
// Parameters:
//   - name: The name of the service to signal
//   - sig: The signal to send (e.g., syscall.SIGTERM, syscall.SIGKILL)
//
// Returns:
//   - error: Any error that occurred while sending the signal
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

// Shutdown gracefully shuts down the orchestrator and all managed services.
//
// This method stops all running services and performs cleanup operations.
// It respects the provided context for cancellation and timeout control.
// The method will attempt to stop all services gracefully, but will force
// kill any services that don't exit within the shutdown timeout.
//
// Once the orchestrator is shut down, it cannot be restarted. A new instance
// must be created.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//
// Returns:
//   - error: Any error that occurred during the shutdown process
//
// Example:
//
//   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//   defer cancel()
//   err := orchestrator.Shutdown(ctx)
//   if err != nil {
//     // handle error
//   }
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
	
	// Check for errors
	close(errCh)
	var errs []string
	for err := range errCh {
		errs = append(errs, err.Error())
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %s", fmt.Sprintf("%v", errs))
	}
	
	// Close channels
	close(o.doneCh)
	
	return nil
}

// setupSignals initializes signal handling for the orchestrator.
//
// This method sets up a goroutine that listens for SIGINT and SIGTERM signals
// and initiates a graceful shutdown of the orchestrator when received.
// It's called during orchestrator initialization.
func (o *Orchestrator) setupSignals() {
	// Create signal channel
	o.sigCh = make(chan os.Signal, 1)
	signal.Notify(o.sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Signal handling happens in a goroutine
	go func() {
		sig := <-o.sigCh
		o.logger.Info("Received signal", zap.String("signal", sig.String()))
		
		// Create context with timeout for shutdown
		ctx, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(o.config.ShutdownTimeout) * time.Second,
		)
		defer cancel()
		
		// Shutdown gracefully
		if err := o.Shutdown(ctx); err != nil {
			o.logger.Error("Shutdown failed", zap.Error(err))
		}
	}()
}

// DiscoverServices finds all service binaries in the services directory.
//
// This method scans the configured services directory for executable service
// binaries. It looks for directories containing an executable with the same
// name as the directory (e.g., "identity/identity").
//
// The discovered services are logged at INFO level, and a warning is logged
// if no services are found.
//
// Returns:
//   - []string: List of service names that were discovered
//   - error: Any error that occurred during directory scanning
//
// Example:
//
//   services, err := orchestrator.DiscoverServices()
//   if err != nil {
//     // handle error
//   }
//   fmt.Printf("Discovered %d services: %v\n", len(services), services)
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

// GetServiceInfo returns comprehensive diagnostic information about a service.
//
// This method retrieves detailed status information for a specific service,
// including its configured state, runtime state, process ID, uptime, restart
// count, and any error information if the service has failed.
//
// Parameters:
//   - name: The name of the service to get information for
//
// Returns:
//   - *types.ServiceInfo: Detailed service information structure
//   - error: If the service doesn't exist or an error occurs during the operation
//
// Example:
//
//   info, err := orchestrator.GetServiceInfo("identity")
//   if err != nil {
//     // handle error
//   }
//   fmt.Printf("Service: %s, State: %s, PID: %d\n",
//     info.Name, info.State, info.PID)
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
		info.LastError = process.LastError.Error()
	}
	
	return info, nil
}

// GetAllServices returns diagnostic information about all configured services.
//
// This method retrieves status information for all services that are configured
// in the orchestrator, whether they are running or not. It's useful for getting
// a comprehensive view of the system state.
//
// Returns:
//   - map[string]*types.ServiceInfo: Map of service names to their information structures
//   - error: Any error that occurred during the information gathering
//
// Example:
//
//   services, err := orchestrator.GetAllServices()
//   if err != nil {
//     // handle error
//   }
//   for name, info := range services {
//     fmt.Printf("Service: %s, State: %s\n", name, info.State)
//   }
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
				info.LastError = process.LastError.Error()
			}
		}
		
		services[name] = info
	}
	
	return services, nil
}

// Helper functions for file/directory operations

// fileExists checks if a file exists at the given path.
//
// Parameters:
//   - path: The path to check
//
// Returns:
//   - bool: true if the file exists, false otherwise
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// dirExists checks if a directory exists at the given path.
//
// Parameters:
//   - path: The path to check
//
// Returns:
//   - bool: true if the directory exists, false otherwise
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// isExecutable checks if a file is executable.
//
// Parameters:
//   - path: The path to check
//
// Returns:
//   - bool: true if the file is executable, false otherwise
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return (info.Mode().Perm() & 0111) != 0
}

// initLogger creates a zap logger with the specified log level.
//
// This helper method creates a structured logger with the appropriate log level
// configuration. It's used when no custom logger is provided to the orchestrator.
//
// Parameters:
//   - level: The log level ("debug", "info", "warn", "error")
//
// Returns:
//   - *zap.Logger: The configured logger
//   - error: Any error that occurred during logger creation
func initLogger(level string) (*zap.Logger, error) {
	var zapLevel zap.AtomicLevel
	switch level {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	
	config := zap.Config{
		Level:            zapLevel,
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	
	return config.Build()
}