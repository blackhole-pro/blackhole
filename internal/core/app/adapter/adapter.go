// Package adapter provides adapters between different application interfaces
// to enable flexible integration between components without direct dependencies.
package adapter

import (
	"github.com/handcraftdev/blackhole/internal/core/app/types"
)

// ApplicationAdapter adapts concrete Application implementations to the types.Application interface
type ApplicationAdapter struct {
	app types.Application
}

// NewApplicationAdapter creates a new ApplicationAdapter
func NewApplicationAdapter(app types.Application) *ApplicationAdapter {
	return &ApplicationAdapter{app: app}
}

// Start implements the types.Application.Start method
func (a *ApplicationAdapter) Start() error {
	return a.app.Start()
}

// Stop implements the types.Application.Stop method
func (a *ApplicationAdapter) Stop() error {
	return a.app.Stop()
}

// GetProcessManager implements the types.Application.GetProcessManager method
func (a *ApplicationAdapter) GetProcessManager() types.ProcessManager {
	return a.app.GetProcessManager()
}

// GetConfigManager implements the types.Application.GetConfigManager method
func (a *ApplicationAdapter) GetConfigManager() types.ConfigManager {
	return a.app.GetConfigManager()
}

// RegisterService implements the types.Application.RegisterService method
func (a *ApplicationAdapter) RegisterService(service types.Service) error {
	return a.app.RegisterService(service)
}

// GetService implements the types.Application.GetService method
func (a *ApplicationAdapter) GetService(name string) (types.Service, bool) {
	return a.app.GetService(name)
}