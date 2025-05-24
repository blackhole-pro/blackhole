// Package core provides the main components for the Blackhole platform core runtime
package core

import (
	"context"
	"time"

	"github.com/blackhole-pro/blackhole/core/internal/runtime/config/types"
	processtypes "github.com/blackhole-pro/blackhole/core/internal/runtime/orchestrator/types"
)

// Application is the main Blackhole application interface
type Application interface {
	// Start starts the application and its services
	Start() error
	
	// Stop stops the application and its services
	Stop() error
	
	// GetProcessManager returns the process manager
	GetProcessManager() ProcessManager
	
	// GetConfigManager returns the configuration manager
	GetConfigManager() ConfigManager
}

// ServiceInfo represents status information about a service
type ServiceInfo struct {
	Name       string
	Configured bool
	Enabled    bool
	State      string
	PID        int
	Uptime     time.Duration
	Restarts   int
	LastError  string
}

// ProcessManager defines the interface for managing processes
type ProcessManager interface {
	// Start starts a service by name
	Start(name string) error
	
	// Stop stops a service by name
	Stop(name string) error
	
	// Restart restarts a service by name
	Restart(name string) error
	
	// StartAll starts all configured services
	StartAll() error
	
	// StopAll stops all services and shuts down the orchestrator
	StopAll() error
	
	// Status returns the current state of a service
	Status(name string) (processtypes.ProcessState, error)
	
	// IsRunning checks if a service is running
	IsRunning(name string) bool
	
	// GetServiceInfo returns information about a service
	GetServiceInfo(name string) (*ServiceInfo, error)
	
	// GetAllServices returns information about all services
	GetAllServices() (map[string]*ServiceInfo, error)
}

// ConfigManager defines the interface for configuration management
type ConfigManager interface {
	// Get returns the current configuration
	Get() *types.Config
	
	// Set updates the configuration
	Set(cfg *types.Config) error
	
	// Save persists the configuration to disk
	Save() error
}

// NewApplication creates a new Blackhole application
func NewApplication(ctx context.Context) (Application, error) {
	// This will be implemented later with the app package
	return nil, nil
}