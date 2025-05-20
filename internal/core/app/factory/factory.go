// Package factory provides factory implementations for creating core components
// of the application. It includes factories for process managers and other
// dependencies that improve testability through dependency injection.
package factory

import (
	"github.com/handcraftdev/blackhole/internal/core/app/types"
	"github.com/handcraftdev/blackhole/internal/core/process"
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
	// Create a new process orchestrator with the provided config manager and logger
	orchestrator, err := process.NewOrchestrator(
		configManager,
		process.WithLogger(logger),
	)
	if err != nil {
		return nil, err
	}
	
	return orchestrator, nil
}