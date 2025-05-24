// Package factory provides factory implementations for creating core components
// of the application. It includes factories for process managers and other
// dependencies that improve testability through dependency injection.
package factory

import (
	"github.com/blackhole-pro/blackhole/core/internal/core/app/adapter"
	"github.com/blackhole-pro/blackhole/core/internal/core/app/types"
	"github.com/blackhole-pro/blackhole/core/internal/runtime/config"
	"github.com/blackhole-pro/blackhole/core/internal/runtime/orchestrator"
	"go.uber.org/zap"
)

// DefaultProcessManagerFactory is the default implementation of ProcessManagerFactory
// that creates real process.Orchestrator instances for production use.
type DefaultProcessManagerFactory struct{}

// NewDefaultProcessManagerFactory creates a new DefaultProcessManagerFactory
func NewDefaultProcessManagerFactory() *DefaultProcessManagerFactory {
	return &DefaultProcessManagerFactory{}
}

// CreateProcessManager implements the ProcessManagerFactory interface by creating
// a new process.Orchestrator instance configured with the provided dependencies.
//
// Parameters:
//   - configManager: The configuration manager to use for orchestrator initialization
//   - logger: The logger to use for logging
//
// Returns:
//   - A ProcessManager instance (implemented by process.Orchestrator)
//   - An error if creation fails
func (f *DefaultProcessManagerFactory) CreateProcessManager(
	configManager types.ConfigManager, 
	logger *zap.Logger,
) (types.ProcessManager, error) {
	// We need to get the core ConfigManager from our app ConfigManager adapter
	// First we need to check if it's our adapter type
	var coreConfigManager *config.ConfigManager
	
	// Check if we have our adapter to extract the core manager
	if adapter, ok := configManager.(*adapter.ConfigManagerAdapter); ok {
		coreConfigManager = adapter.GetCoreManager()
	}
	
	// If not, create a temporary core ConfigManager
	if coreConfigManager == nil {
		logger.Warn("Failed to get core ConfigManager from adapter, creating a new one with defaults")
		coreConfigManager = config.NewConfigManager(logger)
	} else {
		logger.Info("Using configured ConfigManager from adapter")
	}

	// Create a new process orchestrator with the core config manager and logger
	processOrchestrator, err := orchestrator.NewOrchestrator(
		coreConfigManager,
		orchestrator.WithLogger(logger),
	)
	if err != nil {
		return nil, err
	}
	
	// Create an adapter to convert between interfaces
	return adapter.NewProcessManagerAdapter(processOrchestrator, logger), nil
}