// Package types defines the core type definitions and interfaces for the
// application component. It provides the contract that implementations must follow
// and ensures consistency across the application subsystem.
package types

import (
	"go.uber.org/zap"
)

// Service defines the common interface for all services in the platform
type Service interface {
	// Start starts the service
	Start() error
	
	// Stop stops the service
	Stop() error
	
	// Name returns the service name
	Name() string
	
	// Health returns the health status of the service
	Health() bool
}

// Application defines the interface for the main application
type Application interface {
	// Start starts the application
	Start() error
	
	// Stop stops the application
	Stop() error
	
	// GetProcessManager returns the process manager
	GetProcessManager() ProcessManager
	
	// GetConfigManager returns the configuration manager
	GetConfigManager() ConfigManager
	
	// RegisterService registers a service with the application
	RegisterService(service Service) error
	
	// GetService returns a service by name
	GetService(name string) (Service, bool)
}

// ProcessManager defines the common interface for process management
type ProcessManager interface {
	// Start starts the process manager
	Start() error
	
	// Stop stops the process manager
	Stop() error
	
	// StartAll starts all configured services
	StartAll() error
	
	// StopAll stops all running services
	StopAll() error
	
	// StartService starts a specific service
	StartService(name string) error
	
	// StopService stops a specific service
	StopService(name string) error
	
	// RestartService restarts a specific service
	RestartService(name string) error
	
	// GetServiceInfo returns information about a service
	GetServiceInfo(name string) (ServiceInfo, error)
	
	// DiscoverServices discovers available services by finding service binaries in the services directory
	DiscoverServices() ([]string, error)
	
	// RefreshServices re-discovers services and updates service configurations
	RefreshServices() ([]string, error)
}

// ProcessManagerFactory defines an interface for creating ProcessManager instances.
// This factory pattern improves testability by allowing dependency injection
// of different ProcessManager implementations.
type ProcessManagerFactory interface {
	// CreateProcessManager creates a new ProcessManager instance
	CreateProcessManager(configManager ConfigManager, logger *zap.Logger) (ProcessManager, error)
}

// ConfigManager defines the common interface for configuration management
type ConfigManager interface {
	// GetConfig returns the current configuration
	GetConfig() *Config
	
	// SetConfig updates the configuration
	SetConfig(config *Config) error
	
	// LoadFromFile loads configuration from a file
	LoadFromFile(path string) error
	
	// SaveToFile saves configuration to a file
	SaveToFile(path string) error
	
	// SubscribeToChanges registers a callback for configuration changes
	SubscribeToChanges(callback func(*Config))
}

// Config represents the application configuration
type Config struct {
	// Placeholder for config fields - specific fields will be
	// defined in the concrete implementation
}

// ServiceInfo contains information about a service
type ServiceInfo struct {
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	Running    bool   `json:"running"`
	PID        int    `json:"pid,omitempty"`
	Uptime     string `json:"uptime,omitempty"`
	LastError  string `json:"last_error,omitempty"`
}