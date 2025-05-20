// Package app provides the main application implementation for the Blackhole platform.
// It ties together all core components and manages the application lifecycle.
package app

import (
	"sync"

	"github.com/handcraftdev/blackhole/internal/core/app/adapter"
	"github.com/handcraftdev/blackhole/internal/core/app/factory"
	"github.com/handcraftdev/blackhole/internal/core/app/types"
	"github.com/handcraftdev/blackhole/internal/core/config"
	"go.uber.org/zap"
)

// Application represents the main application instance implementing the types.Application interface
type Application struct {
	// Core components
	logger                *zap.Logger
	coreConfigManager     *config.ConfigManager
	configManagerAdapter  types.ConfigManager
	processManager        types.ProcessManager
	
	// Factories
	processManagerFactory types.ProcessManagerFactory
	
	// Service registry
	services       map[string]types.Service
	servicesMutex  sync.RWMutex
	
	// Synchronization
	mu            sync.RWMutex
	doneCh        chan struct{}
	isActive      bool
}

// ApplicationOption is a functional option for configuring the Application
type ApplicationOption func(*Application)

// WithLogger sets a custom logger for the application
func WithLogger(logger *zap.Logger) ApplicationOption {
	return func(a *Application) {
		a.logger = logger
	}
}

// WithConfigManager sets a custom config manager for the application
func WithConfigManager(configManager *config.ConfigManager) ApplicationOption {
	return func(a *Application) {
		a.coreConfigManager = configManager
		// Create a new adapter with the custom core config manager
		a.configManagerAdapter = adapter.NewConfigManagerAdapter(configManager, a.logger)
	}
}

// WithProcessManagerFactory sets a custom factory for creating the process manager.
// This is particularly useful for testing, where you might want to inject a mock
// factory that creates mock process managers instead of real ones.
//
// Example:
//
//	app, err := NewApplication(
//		WithProcessManagerFactory(mockFactory),
//	)
func WithProcessManagerFactory(factory types.ProcessManagerFactory) ApplicationOption {
	return func(a *Application) {
		a.processManagerFactory = factory
	}
}

// NewApplication creates a new application instance
func NewApplication(options ...ApplicationOption) (*Application, error) {
	// Create a default logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, &types.AppError{
			Message: "failed to initialize logger",
			Err:     err,
		}
	}
	
	// Create a default config manager
	coreConfigManager := config.NewConfigManager(logger)
	
	// Create config manager adapter
	configManagerAdapter := adapter.NewConfigManagerAdapter(coreConfigManager, logger)
	
	// Create the application
	app := &Application{
		logger:                logger,
		coreConfigManager:     coreConfigManager,
		configManagerAdapter:  configManagerAdapter,
		doneCh:                make(chan struct{}),
		services:              make(map[string]types.Service),
		processManagerFactory: factory.NewDefaultProcessManagerFactory(),
	}
	
	// Apply options
	for _, option := range options {
		option(app)
	}
	
	return app, nil
}

// RegisterService registers a service with the application
func (a *Application) RegisterService(service types.Service) error {
	if service == nil {
		return &types.AppError{Message: "cannot register nil service"}
	}
	
	serviceName := service.Name()
	if serviceName == "" {
		return &types.AppError{Message: "service name cannot be empty"}
	}
	
	a.servicesMutex.Lock()
	defer a.servicesMutex.Unlock()
	
	// Check if service is already registered
	if _, exists := a.services[serviceName]; exists {
		return types.ErrServiceAlreadyRegistered
	}
	
	a.services[serviceName] = service
	a.logger.Info("Service registered", zap.String("service", serviceName))
	return nil
}

// Start starts the application
func (a *Application) Start() error {
	a.mu.Lock()
	if a.isActive {
		a.mu.Unlock()
		return nil // Already started
	}
	a.isActive = true
	a.mu.Unlock()

	a.logger.Info("Starting Blackhole application")
	
	// Create process manager using the factory
	processManager, err := a.processManagerFactory.CreateProcessManager(
		a.configManagerAdapter,
		a.logger,
	)
	if err != nil {
		return &types.AppError{
			Message: "failed to create process manager",
			Err:     err,
		}
	}
	a.processManager = processManager
	
	// Start the process manager
	if err := a.processManager.Start(); err != nil {
		return &types.AppError{
			Message: "failed to start process manager",
			Err:     err,
		}
	}
	
	// Start all enabled services
	if err := a.processManager.StartAll(); err != nil {
		a.logger.Warn("Some services failed to start", zap.Error(err))
		// Continue with the services that did start
	}
	
	a.logger.Info("Blackhole application started successfully")
	return nil
}

// Stop stops the application
func (a *Application) Stop() error {
	a.mu.Lock()
	if !a.isActive {
		a.mu.Unlock()
		return nil // Already stopped
	}
	a.isActive = false
	a.mu.Unlock()

	a.logger.Info("Stopping Blackhole application")
	
	// Stop the orchestrator if it exists
	if a.processManager != nil {
		if err := a.processManager.Stop(); err != nil {
			a.logger.Error("Error while stopping orchestrator", zap.Error(err))
			// Continue with shutdown anyway
		}
	}
	
	// Stop all registered services
	a.servicesMutex.RLock()
	services := make([]types.Service, 0, len(a.services))
	for _, service := range a.services {
		services = append(services, service)
	}
	a.servicesMutex.RUnlock()
	
	for _, service := range services {
		if err := service.Stop(); err != nil {
			a.logger.Error("Error while stopping service",
				zap.String("service", service.Name()),
				zap.Error(err))
			// Continue stopping other services
		}
	}
	
	// Close the done channel
	close(a.doneCh)
	
	a.logger.Info("Blackhole application stopped")
	return nil
}

// GetProcessManager returns the process manager
func (a *Application) GetProcessManager() types.ProcessManager {
	return a.processManager
}

// GetConfigManager returns the configuration manager
func (a *Application) GetConfigManager() types.ConfigManager {
	return a.configManagerAdapter
}

// GetService returns a service by name
func (a *Application) GetService(name string) (types.Service, bool) {
	a.servicesMutex.RLock()
	defer a.servicesMutex.RUnlock()
	
	service, exists := a.services[name]
	return service, exists
}