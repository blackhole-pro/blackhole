package state

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
)

// StateStorage defines the interface for state persistence
type StateStorage interface {
	// Save persists plugin state
	Save(ctx context.Context, pluginID string, version string, state []byte) error
	
	// Load retrieves plugin state
	Load(ctx context.Context, pluginID string, version string) ([]byte, error)
	
	// List returns available state versions for a plugin
	List(ctx context.Context, pluginID string) ([]StateVersion, error)
	
	// Delete removes a specific state version
	Delete(ctx context.Context, pluginID string, version string) error
}

// StateSerializer defines the interface for state serialization
type StateSerializer interface {
	// Serialize converts plugin state to bytes
	Serialize(state interface{}) ([]byte, error)
	
	// Deserialize converts bytes to plugin state
	Deserialize(data []byte, target interface{}) error
}

// StateMigrator defines the interface for migrating state between versions
type StateMigrator interface {
	// CanMigrate checks if migration is supported between versions
	CanMigrate(fromVersion, toVersion string) bool
	
	// Migrate performs state migration
	Migrate(ctx context.Context, fromState []byte, fromVersion, toVersion string) ([]byte, error)
}

// StateVersion represents a saved state version
type StateVersion struct {
	PluginID  string
	Version   string
	Timestamp time.Time
	Size      int64
	Checksum  string
}

// stateManager implements plugin state management
type stateManager struct {
	storage     StateStorage
	serializer  StateSerializer
	migrations  map[string]StateMigrator // pluginID -> migrator
	mu          sync.RWMutex
}

// NewStateManager creates a new state manager
func NewStateManager(storage StateStorage, serializer StateSerializer) *stateManager {
	return &stateManager{
		storage:    storage,
		serializer: serializer,
		migrations: make(map[string]StateMigrator),
	}
}

// RegisterMigrator registers a state migrator for a plugin
func (m *stateManager) RegisterMigrator(pluginID string, migrator StateMigrator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.migrations[pluginID] = migrator
}

// SaveState saves plugin state
func (m *stateManager) SaveState(ctx context.Context, pluginID string, version string, state interface{}) error {
	// Serialize state
	data, err := m.serializer.Serialize(state)
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}
	
	// Save to storage
	if err := m.storage.Save(ctx, pluginID, version, data); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}
	
	return nil
}

// LoadState loads plugin state
func (m *stateManager) LoadState(ctx context.Context, pluginID string, version string, target interface{}) error {
	// Load from storage
	data, err := m.storage.Load(ctx, pluginID, version)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}
	
	// Deserialize state
	if err := m.serializer.Deserialize(data, target); err != nil {
		return fmt.Errorf("failed to deserialize state: %w", err)
	}
	
	return nil
}

// MigrateState migrates state from one version to another
func (m *stateManager) MigrateState(ctx context.Context, pluginID string, fromVersion, toVersion string) ([]byte, error) {
	// Get migrator
	m.mu.RLock()
	migrator, exists := m.migrations[pluginID]
	m.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("no migrator registered for plugin %s", pluginID)
	}
	
	// Check if migration is supported
	if !migrator.CanMigrate(fromVersion, toVersion) {
		return nil, fmt.Errorf("migration not supported from %s to %s", fromVersion, toVersion)
	}
	
	// Load old state
	oldState, err := m.storage.Load(ctx, pluginID, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to load state for version %s: %w", fromVersion, err)
	}
	
	// Perform migration
	newState, err := migrator.Migrate(ctx, oldState, fromVersion, toVersion)
	if err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	
	// Save new state
	if err := m.storage.Save(ctx, pluginID, toVersion, newState); err != nil {
		return nil, fmt.Errorf("failed to save migrated state: %w", err)
	}
	
	return newState, nil
}

// ListVersions lists available state versions for a plugin
func (m *stateManager) ListVersions(ctx context.Context, pluginID string) ([]StateVersion, error) {
	return m.storage.List(ctx, pluginID)
}

// DeleteState deletes a specific state version
func (m *stateManager) DeleteState(ctx context.Context, pluginID string, version string) error {
	return m.storage.Delete(ctx, pluginID, version)
}

// CreateSnapshot creates a snapshot of all plugin states
func (m *stateManager) CreateSnapshot(ctx context.Context, pluginList []plugins.PluginInfo) (map[string][]byte, error) {
	snapshot := make(map[string][]byte)
	
	for _, plugin := range pluginList {
		if plugin.Status == plugins.PluginStatusRunning {
			// Export state from running plugin
			stateData, err := m.exportPluginState(ctx, plugin.Name, plugin.Version)
			if err != nil {
				// Log error but continue with other plugins
				continue
			}
			snapshot[plugin.Name] = stateData
		}
	}
	
	return snapshot, nil
}

// RestoreSnapshot restores plugin states from a snapshot
func (m *stateManager) RestoreSnapshot(ctx context.Context, snapshot map[string][]byte) error {
	for pluginID, stateData := range snapshot {
		// Import state to plugin
		if err := m.importPluginState(ctx, pluginID, stateData); err != nil {
			// Log error but continue with other plugins
			continue
		}
	}
	
	return nil
}

// exportPluginState exports state from a running plugin
func (m *stateManager) exportPluginState(ctx context.Context, pluginID, version string) ([]byte, error) {
	// This would communicate with the plugin via RPC to export its state
	// For now, we'll return a placeholder
	return json.Marshal(map[string]interface{}{
		"pluginID": pluginID,
		"version":  version,
		"exported": time.Now(),
	})
}

// importPluginState imports state to a plugin
func (m *stateManager) importPluginState(ctx context.Context, pluginID string, stateData []byte) error {
	// This would communicate with the plugin via RPC to import state
	// For now, we'll return success
	return nil
}

// JSONSerializer implements StateSerializer using JSON
type JSONSerializer struct{}

// Serialize converts state to JSON bytes
func (s *JSONSerializer) Serialize(state interface{}) ([]byte, error) {
	return json.Marshal(state)
}

// Deserialize converts JSON bytes to state
func (s *JSONSerializer) Deserialize(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}