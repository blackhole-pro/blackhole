package state

import (
	"context"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
)

// StateManagerWrapper wraps the internal state manager to implement plugins.StateManager
type StateManagerWrapper struct {
	manager *stateManager
}

// NewStateManagerWrapper creates a new state manager wrapper
func NewStateManagerWrapper(storage StateStorage, serializer StateSerializer) plugins.StateManager {
	return &StateManagerWrapper{
		manager: NewStateManager(storage, serializer),
	}
}

// SaveState saves plugin state
func (w *StateManagerWrapper) SaveState(plugin plugins.Plugin) error {
	info := plugin.Info()
	state, err := plugin.ExportState()
	if err != nil {
		return err
	}
	
	ctx := context.Background()
	return w.manager.SaveState(ctx, info.Name, info.Version, state)
}

// LoadState loads plugin state
func (w *StateManagerWrapper) LoadState(plugin plugins.Plugin) error {
	info := plugin.Info()
	ctx := context.Background()
	
	// Load state data
	data, err := w.manager.storage.Load(ctx, info.Name, info.Version)
	if err != nil {
		return err
	}
	
	// Import state to plugin
	return plugin.ImportState(data)
}

// MigrateState migrates plugin state between versions
func (w *StateManagerWrapper) MigrateState(plugin plugins.Plugin, fromVersion, toVersion string) error {
	info := plugin.Info()
	ctx := context.Background()
	
	// Migrate state
	migratedState, err := w.manager.MigrateState(ctx, info.Name, fromVersion, toVersion)
	if err != nil {
		return err
	}
	
	// Import migrated state
	return plugin.ImportState(migratedState)
}

// ExportState exports plugin state
func (w *StateManagerWrapper) ExportState(plugin plugins.Plugin) ([]byte, error) {
	return plugin.ExportState()
}

// ImportState imports plugin state
func (w *StateManagerWrapper) ImportState(plugin plugins.Plugin, state []byte) error {
	return plugin.ImportState(state)
}