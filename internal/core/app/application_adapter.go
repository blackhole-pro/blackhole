package app

import (
	"os"
	"sync"

	"github.com/handcraftdev/blackhole/internal/core"
	"github.com/handcraftdev/blackhole/internal/core/config"
	"github.com/handcraftdev/blackhole/internal/core/process"
	"go.uber.org/zap"
)

// Application represents the main blackhole application
type Application struct {
	// Core components
	logger         *zap.Logger
	configManager  *config.ConfigManager
	processManager *process.Orchestrator
	
	// Synchronization
	mu       sync.RWMutex
	doneCh   chan struct{}
	isActive bool
}

// NewApplication creates a new application instance
func NewApplication() *Application {
	// Create a default logger
	logger, err := zap.NewProduction()
	if err != nil {
		// If we can't create a logger, panic - we need logging
		panic("failed to initialize logger: " + err.Error())
	}
	
	// Create a config manager
	configManager := config.NewConfigManager()
	
	// Return new application
	return &Application{
		logger:        logger,
		configManager: configManager,
		doneCh:        make(chan struct{}),
	}
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
	
	// Initialize process manager
	orchestrator, err := process.NewOrchestrator(a.configManager, process.WithLogger(a.logger))
	if err != nil {
		return err
	}
	a.processManager = orchestrator
	
	// Start the orchestrator
	if err := orchestrator.Start(); err != nil {
		return err
	}
	
	// Start all enabled services
	if err := orchestrator.StartAll(); err != nil {
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
	
	// Close the done channel
	close(a.doneCh)
	
	a.logger.Info("Blackhole application stopped")
	return nil
}

// GetProcessManager returns the process manager
func (a *Application) GetProcessManager() core.ProcessManager {
	return a.processManager
}

// GetConfigManager returns the configuration manager
func (a *Application) GetConfigManager() core.ConfigManager {
	return a.configManager
}

// ApplicationAdapter adapts app.Application to core.Application
type ApplicationAdapter struct {
	app *Application
}

// NewApplicationAdapter creates a new ApplicationAdapter
func NewApplicationAdapter(app *Application) *ApplicationAdapter {
	return &ApplicationAdapter{app: app}
}

// Start implements core.Application.Start
func (a *ApplicationAdapter) Start() error {
	return a.app.Start()
}

// Stop implements core.Application.Stop
func (a *ApplicationAdapter) Stop() error {
	return a.app.Stop()
}

// GetProcessManager implements core.Application.GetProcessManager
func (a *ApplicationAdapter) GetProcessManager() core.ProcessManager {
	return a.app.GetProcessManager()
}

// GetConfigManager implements core.Application.GetConfigManager
func (a *ApplicationAdapter) GetConfigManager() core.ConfigManager {
	return a.app.GetConfigManager()
}