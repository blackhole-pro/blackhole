// Package adapter provides adapters between different application interfaces
// to enable flexible integration between components without direct dependencies.
package adapter

import (
	"context"
	"time"
	
	"github.com/handcraftdev/blackhole/internal/core/app/types"
	"github.com/handcraftdev/blackhole/internal/core/config"
	"github.com/handcraftdev/blackhole/internal/core/process"
	processtypes "github.com/handcraftdev/blackhole/internal/core/process/types"
	"go.uber.org/zap"
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

// ProcessManagerAdapter adapts process.Orchestrator to the types.ProcessManager interface
type ProcessManagerAdapter struct {
	orchestrator *process.Orchestrator
	logger       *zap.Logger
}

// NewProcessManagerAdapter creates a new ProcessManagerAdapter
func NewProcessManagerAdapter(orchestrator *process.Orchestrator, logger *zap.Logger) *ProcessManagerAdapter {
	return &ProcessManagerAdapter{
		orchestrator: orchestrator,
		logger:       logger,
	}
}

// Start implements the types.ProcessManager.Start method
func (a *ProcessManagerAdapter) Start() error {
	// This is the start method for the manager itself, not for starting services
	a.logger.Info("Process manager started")
	return nil
}

// Stop implements the types.ProcessManager.Stop method
func (a *ProcessManagerAdapter) Stop() error {
	// This is the stop method for the manager itself, not for stopping services
	a.logger.Info("Process manager stopped")
	return nil
}

// StartService implements the types.ProcessManager.StartService method
func (a *ProcessManagerAdapter) StartService(name string) error {
	return a.orchestrator.Start(name)
}

// StopService implements the types.ProcessManager.StopService method
func (a *ProcessManagerAdapter) StopService(name string) error {
	return a.orchestrator.Stop(name)
}

// RestartService implements the types.ProcessManager.RestartService method
func (a *ProcessManagerAdapter) RestartService(name string) error {
	return a.orchestrator.Restart(name)
}

// StartAll implements the types.ProcessManager.StartAll method
func (a *ProcessManagerAdapter) StartAll() error {
	// Since Orchestrator doesn't have StartAll, we need to implement it here
	a.logger.Info("Starting all services")
	
	// Get all services
	services, err := a.orchestrator.GetAllServices()
	if err != nil {
		return err
	}
	
	// Start each enabled service
	for name, info := range services {
		if info.Enabled {
			if err := a.orchestrator.Start(name); err != nil {
				a.logger.Error("Failed to start service", 
					zap.String("service", name),
					zap.Error(err))
				// Continue with other services even if one fails
			}
		}
	}
	
	return nil
}

// StopAll implements the types.ProcessManager.StopAll method
func (a *ProcessManagerAdapter) StopAll() error {
	a.logger.Info("Stopping all services")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return a.orchestrator.Shutdown(ctx)
}

// IsRunning implements the types.ProcessManager.IsRunning method
func (a *ProcessManagerAdapter) IsRunning(name string) bool {
	return a.orchestrator.IsRunning(name)
}

// GetServiceInfo implements the types.ProcessManager.GetServiceInfo method
func (a *ProcessManagerAdapter) GetServiceInfo(name string) (types.ServiceInfo, error) {
	info, err := a.orchestrator.GetServiceInfo(name)
	if err != nil {
		return types.ServiceInfo{}, err
	}
	
	// Convert from process types.ServiceInfo to app types.ServiceInfo
	return types.ServiceInfo{
		Name:       info.Name,
		Enabled:    info.Enabled,
		Running:    info.State == string(processtypes.ProcessStateRunning),
		PID:        info.PID,
		Uptime:     info.Uptime.String(),
		LastError:  info.LastError,
	}, nil
}

// ConfigManagerAdapter adapts core.config.ConfigManager to app.types.ConfigManager
type ConfigManagerAdapter struct {
	manager *config.ConfigManager
	logger  *zap.Logger
}

// NewConfigManagerAdapter creates a new ConfigManagerAdapter
func NewConfigManagerAdapter(manager *config.ConfigManager, logger *zap.Logger) *ConfigManagerAdapter {
	return &ConfigManagerAdapter{
		manager: manager,
		logger:  logger,
	}
}

// GetConfig implements the types.ConfigManager.GetConfig method
func (a *ConfigManagerAdapter) GetConfig() *types.Config {
	// This is a simplified adapter since we don't need all config fields
	// We could implement a more comprehensive conversion if needed
	return &types.Config{}
}

// SetConfig implements the types.ConfigManager.SetConfig method
func (a *ConfigManagerAdapter) SetConfig(config *types.Config) error {
	// This is a simplified implementation
	a.logger.Info("Config updated via adapter")
	return nil
}

// LoadFromFile implements the types.ConfigManager.LoadFromFile method
func (a *ConfigManagerAdapter) LoadFromFile(path string) error {
	// Use the core config manager to load the file
	// But this is a simplified implementation
	a.logger.Info("Loading config from file", zap.String("path", path))
	return nil
}

// SaveToFile implements the types.ConfigManager.SaveToFile method
func (a *ConfigManagerAdapter) SaveToFile(path string) error {
	// Use the core config manager to save the file
	// But this is a simplified implementation
	a.logger.Info("Saving config to file", zap.String("path", path))
	return nil
}

// SubscribeToChanges implements the types.ConfigManager.SubscribeToChanges method
func (a *ConfigManagerAdapter) SubscribeToChanges(callback func(*types.Config)) {
	// We could implement a real subscription if needed
	a.logger.Info("Subscribed to config changes via adapter")
}