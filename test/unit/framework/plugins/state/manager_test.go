package state_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/state"
)

func TestStateManager_SaveAndLoad(t *testing.T) {
	// Create memory storage and JSON serializer
	storage := state.NewMemoryStateStorage()
	serializer := &state.JSONSerializer{}
	manager := state.NewStateManager(storage, serializer)
	
	// Test data
	pluginID := "test-plugin"
	version := "1.0.0"
	testState := map[string]interface{}{
		"counter": 42,
		"name":    "test",
		"data":    []string{"a", "b", "c"},
	}
	
	// Save state
	ctx := context.Background()
	err := manager.SaveState(ctx, pluginID, version, testState)
	require.NoError(t, err)
	
	// Load state
	var loadedState map[string]interface{}
	err = manager.LoadState(ctx, pluginID, version, &loadedState)
	require.NoError(t, err)
	
	// Verify state
	assert.Equal(t, testState["counter"], int(loadedState["counter"].(float64)))
	assert.Equal(t, testState["name"], loadedState["name"])
	
	data := loadedState["data"].([]interface{})
	assert.Len(t, data, 3)
	assert.Equal(t, "a", data[0])
}

func TestStateManager_ListVersions(t *testing.T) {
	storage := state.NewMemoryStateStorage()
	serializer := &state.JSONSerializer{}
	manager := state.NewStateManager(storage, serializer)
	
	pluginID := "test-plugin"
	ctx := context.Background()
	
	// Save multiple versions
	versions := []string{"1.0.0", "1.1.0", "2.0.0"}
	for _, v := range versions {
		err := manager.SaveState(ctx, pluginID, v, map[string]string{"version": v})
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}
	
	// List versions
	versionList, err := manager.ListVersions(ctx, pluginID)
	require.NoError(t, err)
	assert.Len(t, versionList, 3)
	
	// Verify all versions are present
	versionMap := make(map[string]bool)
	for _, v := range versionList {
		versionMap[v.Version] = true
	}
	
	for _, v := range versions {
		assert.True(t, versionMap[v], "Version %s should be in list", v)
	}
}

func TestStateManager_DeleteState(t *testing.T) {
	storage := state.NewMemoryStateStorage()
	serializer := &state.JSONSerializer{}
	manager := state.NewStateManager(storage, serializer)
	
	pluginID := "test-plugin"
	version := "1.0.0"
	ctx := context.Background()
	
	// Save state
	err := manager.SaveState(ctx, pluginID, version, map[string]string{"test": "data"})
	require.NoError(t, err)
	
	// Verify it exists
	var state map[string]string
	err = manager.LoadState(ctx, pluginID, version, &state)
	require.NoError(t, err)
	
	// Delete state
	err = manager.DeleteState(ctx, pluginID, version)
	require.NoError(t, err)
	
	// Verify it's gone
	err = manager.LoadState(ctx, pluginID, version, &state)
	assert.Error(t, err)
}

func TestStateManager_Migration(t *testing.T) {
	storage := state.NewMemoryStateStorage()
	serializer := &state.JSONSerializer{}
	manager := state.NewStateManager(storage, serializer)
	
	pluginID := "test-plugin"
	oldVersion := "1.0.0"
	newVersion := "2.0.0"
	ctx := context.Background()
	
	// Create a mock migrator
	migrator := &mockMigrator{
		canMigrate: func(from, to string) bool {
			return from == oldVersion && to == newVersion
		},
		migrate: func(ctx context.Context, fromState []byte, from, to string) ([]byte, error) {
			// Simple migration: add a field
			var state map[string]interface{}
			err := serializer.Deserialize(fromState, &state)
			if err != nil {
				return nil, err
			}
			
			state["migrated"] = true
			state["migration_time"] = time.Now().Unix()
			
			return serializer.Serialize(state)
		},
	}
	
	// Register migrator
	manager.RegisterMigrator(pluginID, migrator)
	
	// Save old state
	oldState := map[string]interface{}{
		"version": oldVersion,
		"data":    "test",
	}
	err := manager.SaveState(ctx, pluginID, oldVersion, oldState)
	require.NoError(t, err)
	
	// Migrate state
	newStateBytes, err := manager.MigrateState(ctx, pluginID, oldVersion, newVersion)
	require.NoError(t, err)
	
	// Verify migrated state
	var migratedState map[string]interface{}
	err = serializer.Deserialize(newStateBytes, &migratedState)
	require.NoError(t, err)
	
	assert.Equal(t, oldVersion, migratedState["version"])
	assert.Equal(t, "test", migratedState["data"])
	assert.True(t, migratedState["migrated"].(bool))
	assert.NotNil(t, migratedState["migration_time"])
}

func TestStateManager_Snapshot(t *testing.T) {
	storage := state.NewMemoryStateStorage()
	serializer := &state.JSONSerializer{}
	manager := state.NewStateManager(storage, serializer)
	
	ctx := context.Background()
	
	// Create mock plugin infos
	plugins := []plugins.PluginInfo{
		{
			ID:      "plugin-1",
			Version: "1.0.0",
			Status: plugins.PluginStatus{
				State: plugins.PluginStateRunning,
			},
		},
		{
			ID:      "plugin-2",
			Version: "2.0.0",
			Status: plugins.PluginStatus{
				State: plugins.PluginStateRunning,
			},
		},
		{
			ID:      "plugin-3",
			Version: "1.0.0",
			Status: plugins.PluginStatus{
				State: plugins.PluginStateStopped,
			},
		},
	}
	
	// Create snapshot
	snapshot, err := manager.CreateSnapshot(ctx, plugins)
	require.NoError(t, err)
	
	// Should only include running plugins
	assert.Len(t, snapshot, 2)
	assert.NotNil(t, snapshot["plugin-1"])
	assert.NotNil(t, snapshot["plugin-2"])
	assert.Nil(t, snapshot["plugin-3"])
	
	// Restore snapshot
	err = manager.RestoreSnapshot(ctx, snapshot)
	require.NoError(t, err)
}

// Mock migrator for testing
type mockMigrator struct {
	canMigrate func(from, to string) bool
	migrate    func(ctx context.Context, fromState []byte, from, to string) ([]byte, error)
}

func (m *mockMigrator) CanMigrate(fromVersion, toVersion string) bool {
	return m.canMigrate(fromVersion, toVersion)
}

func (m *mockMigrator) Migrate(ctx context.Context, fromState []byte, fromVersion, toVersion string) ([]byte, error) {
	return m.migrate(ctx, fromState, fromVersion, toVersion)
}