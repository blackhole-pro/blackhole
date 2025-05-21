// Package adapter provides adapters between different components.
package adapter

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/handcraftdev/blackhole/internal/core"
	"github.com/handcraftdev/blackhole/internal/core/app"
	"github.com/handcraftdev/blackhole/internal/core/app/types"
	coretypes "github.com/handcraftdev/blackhole/internal/core/config/types"
	processtypes "github.com/handcraftdev/blackhole/internal/core/process/types"
)

// NoServicesError is returned when no services are available to start
type NoServicesError struct {
	message string
}

// Error implements the error interface
func (e *NoServicesError) Error() string {
	return e.message
}

// IsNoServicesError checks if an error is a NoServicesError
func IsNoServicesError(err error) bool {
	_, ok := err.(*NoServicesError)
	return ok
}

// ApplicationAdapter adapts app.Application to core.Application
type ApplicationAdapter struct {
	app *app.Application
	mu  sync.Mutex
}

// NewApplicationAdapter creates a new ApplicationAdapter
func NewApplicationAdapter(app *app.Application) *ApplicationAdapter {
	return &ApplicationAdapter{app: app}
}

// Start implements the core.Application.Start method
func (a *ApplicationAdapter) Start() error {
	return a.app.Start()
}

// Stop implements the core.Application.Stop method
func (a *ApplicationAdapter) Stop() error {
	return a.app.Stop()
}

// GetProcessManager implements the core.Application.GetProcessManager method
func (a *ApplicationAdapter) GetProcessManager() core.ProcessManager {
	// Initialize the process manager without starting services
	if err := a.app.InitializeProcessManager(); err != nil {
		fmt.Printf("Error initializing process manager: %v\n", err)
		return nil
	}
	
	// Get the process manager after initialization
	appProcessManager := a.app.GetProcessManager()
	if appProcessManager == nil {
		fmt.Println("Process manager is nil after initialization")
		return nil
	}
	
	return NewProcessManagerAdapter(appProcessManager)
}

// ProcessManagerAdapter adapts types.ProcessManager to core.ProcessManager
type ProcessManagerAdapter struct {
	manager types.ProcessManager
}

// NewProcessManagerAdapter creates a new ProcessManagerAdapter
func NewProcessManagerAdapter(manager types.ProcessManager) *ProcessManagerAdapter {
	return &ProcessManagerAdapter{manager: manager}
}

// Start starts a service by name
func (a *ProcessManagerAdapter) Start(name string) error {
	return a.manager.StartService(name)
}

// Stop stops a service by name
func (a *ProcessManagerAdapter) Stop(name string) error {
	return a.manager.StopService(name)
}

// Restart restarts a service by name
func (a *ProcessManagerAdapter) Restart(name string) error {
	return a.manager.RestartService(name)
}

// StartAll starts all configured services
func (a *ProcessManagerAdapter) StartAll() error {
	// First check if the process manager is properly initialized
	if a.manager == nil {
		return fmt.Errorf("process manager not initialized")
	}

	// Check if there are any services available to start by trying to discover services
	services, err := a.DiscoverServices()
	if err != nil {
		fmt.Printf("Warning: failed to discover services: %v\n", err)
	}

	if len(services) == 0 {
		// Return a special error message that the caller can check to display appropriate message
		// Don't print here to avoid duplicate messages
		return &NoServicesError{message: "No service binaries found"}
	}

	// We have services to start, proceed with normal StartAll
	return a.manager.StartAll()
}

// StopAll stops all running services
func (a *ProcessManagerAdapter) StopAll() error {
	if a.manager == nil {
		return fmt.Errorf("process manager not initialized")
	}
	return a.manager.StopAll()
}

// DiscoverServices implements ProcessManager.DiscoverServices
// This method discovers available services by finding service binaries
func (a *ProcessManagerAdapter) DiscoverServices() ([]string, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("process manager not initialized")
	}
	
	// This would be a direct call if the interface is updated
	// but we'll keep the reflection approach for backward compatibility
	managerVal := reflect.ValueOf(a.manager)
	discoverMethod := managerVal.MethodByName("DiscoverServices")
	
	if discoverMethod.IsValid() {
		results := discoverMethod.Call([]reflect.Value{})
		if len(results) >= 2 { // Should return ([]string, error)
			// Get the services slice
			servicesVal := results[0]
			var services []string
			if !servicesVal.IsNil() {
				services = make([]string, servicesVal.Len())
				for i := 0; i < servicesVal.Len(); i++ {
					services[i] = servicesVal.Index(i).Interface().(string)
				}
			}
			
			// Check for error
			errVal := results[1]
			if !errVal.IsNil() {
				return services, errVal.Interface().(error)
			}
			return services, nil
		}
	}
	
	// Fallback: Extract services from GetAllServices
	servicesMap, err := a.GetAllServices()
	if err != nil {
		return nil, err
	}
	
	// Extract service names
	services := make([]string, 0, len(servicesMap))
	for name := range servicesMap {
		services = append(services, name)
	}
	
	return services, nil
}

// RefreshServices implements ProcessManager.RefreshServices
// This method re-discovers services and updates service configurations
func (a *ProcessManagerAdapter) RefreshServices() ([]string, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("process manager not initialized")
	}

	// This would be a direct call if the interface is updated
	// but we'll keep the reflection approach for backward compatibility
	managerVal := reflect.ValueOf(a.manager)
	refreshMethod := managerVal.MethodByName("RefreshServices")
	
	if refreshMethod.IsValid() {
		results := refreshMethod.Call([]reflect.Value{})
		if len(results) >= 2 { // Should return ([]string, error)
			// Get the services slice
			servicesVal := results[0]
			var services []string
			if !servicesVal.IsNil() {
				services = make([]string, servicesVal.Len())
				for i := 0; i < servicesVal.Len(); i++ {
					services[i] = servicesVal.Index(i).Interface().(string)
				}
			}
			
			// Check for error
			errVal := results[1]
			if !errVal.IsNil() {
				return services, errVal.Interface().(error)
			}
			return services, nil
		}
	}
	
	// If RefreshServices is not available, fall back to DiscoverServices
	// This maintains compatibility with implementations that don't have RefreshServices yet
	return a.DiscoverServices()
}

// Status returns the current state of a service
func (a *ProcessManagerAdapter) Status(name string) (processtypes.ProcessState, error) {
	info, err := a.manager.GetServiceInfo(name)
	if err != nil {
		return "", err
	}
	
	if info.Running {
		return processtypes.ProcessStateRunning, nil
	}
	return processtypes.ProcessStateStopped, nil
}

// IsRunning checks if a service is running
func (a *ProcessManagerAdapter) IsRunning(name string) bool {
	info, err := a.manager.GetServiceInfo(name)
	if err != nil {
		return false
	}
	return info.Running
}

// GetServiceInfo returns information about a service
func (a *ProcessManagerAdapter) GetServiceInfo(name string) (*core.ServiceInfo, error) {
	info, err := a.manager.GetServiceInfo(name)
	if err != nil {
		return nil, err
	}
	
	// Determine state string based on running status
	state := "stopped"
	if info.Running {
		state = "running"
	}
	
	// Convert from types.ServiceInfo to core.ServiceInfo
	coreInfo := &core.ServiceInfo{
		Name:      info.Name,
		Enabled:   info.Enabled,
		State:     state,
		PID:       info.PID,
		LastError: info.LastError,
	}
	
	// Parse uptime string to duration if present
	if info.Uptime != "" {
		// Simplistic approach - uptime is likely in a simple format
		// In a real implementation, you'd parse this properly
		coreInfo.Uptime = time.Hour // placeholder
	}
	
	// Add Configured field (not in original type)
	coreInfo.Configured = true
	
	return coreInfo, nil
}

// GetAllServices returns information about all services
func (a *ProcessManagerAdapter) GetAllServices() (map[string]*core.ServiceInfo, error) {
	// Get process manager reference
	if a.manager == nil {
		return nil, fmt.Errorf("process manager not initialized")
	}
	
	// First discover services to ensure we're using the ones actually available
	services, err := a.DiscoverServices()
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}
	
	result := make(map[string]*core.ServiceInfo)
	for _, name := range services {
		info, err := a.GetServiceInfo(name)
		if err == nil && info != nil {
			result[name] = info
		}
	}
	
	return result, nil
}

// GetConfigManager implements the core.Application.GetConfigManager method
func (a *ApplicationAdapter) GetConfigManager() core.ConfigManager {
	// Get the app-level config manager and adapt it
	appConfigManager := a.app.GetConfigManager()
	return NewConfigManagerAdapter(appConfigManager)
}

// ConfigManagerAdapter adapts types.ConfigManager to core.ConfigManager
type ConfigManagerAdapter struct {
	manager types.ConfigManager
}

// NewConfigManagerAdapter creates a new ConfigManagerAdapter
func NewConfigManagerAdapter(manager types.ConfigManager) *ConfigManagerAdapter {
	return &ConfigManagerAdapter{manager: manager}
}

// Get implements the core.ConfigManager.Get method
func (a *ConfigManagerAdapter) Get() *coretypes.Config {
	// Create a new default config since we can't directly convert
	// In a proper implementation, you'd convert between config types
	return &coretypes.Config{}
}

// Set implements the core.ConfigManager.Set method
func (a *ConfigManagerAdapter) Set(cfg *coretypes.Config) error {
	// In a proper implementation, you'd convert between config types
	// and call a.manager.SetConfig with the converted value
	return nil
}

// Save implements the core.ConfigManager.Save method
func (a *ConfigManagerAdapter) Save() error {
	// In a proper implementation, you'd call SaveToFile with a path
	return nil
}