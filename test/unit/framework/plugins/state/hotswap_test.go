package state_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/state"
)

// MockPluginManager implements state.PluginManager for testing
type MockPluginManager struct {
	mock.Mock
}

func (m *MockPluginManager) GetPlugin(pluginID string) (*plugins.PluginInfo, error) {
	args := m.Called(pluginID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plugins.PluginInfo), args.Error(1)
}

func (m *MockPluginManager) StopPlugin(ctx context.Context, pluginID string) error {
	args := m.Called(ctx, pluginID)
	return args.Error(0)
}

func (m *MockPluginManager) StartPlugin(ctx context.Context, pluginID string) error {
	args := m.Called(ctx, pluginID)
	return args.Error(0)
}

func (m *MockPluginManager) LoadPlugin(ctx context.Context, spec plugins.PluginSpec) error {
	args := m.Called(ctx, spec)
	return args.Error(0)
}

func (m *MockPluginManager) UnloadPlugin(ctx context.Context, pluginID string) error {
	args := m.Called(ctx, pluginID)
	return args.Error(0)
}

func (m *MockPluginManager) DrainRequests(ctx context.Context, pluginID string, timeout time.Duration) error {
	args := m.Called(ctx, pluginID, timeout)
	return args.Error(0)
}

// MockRollbackManager implements state.RollbackManager for testing
type MockRollbackManager struct {
	mock.Mock
}

func (m *MockRollbackManager) CreateCheckpoint(ctx context.Context, pluginID string) (string, error) {
	args := m.Called(ctx, pluginID)
	return args.String(0), args.Error(1)
}

func (m *MockRollbackManager) Rollback(ctx context.Context, pluginID string, checkpointID string) error {
	args := m.Called(ctx, pluginID, checkpointID)
	return args.Error(0)
}

func (m *MockRollbackManager) CleanupCheckpoint(ctx context.Context, checkpointID string) error {
	args := m.Called(ctx, checkpointID)
	return args.Error(0)
}

func TestHotSwapCoordinator_SuccessfulHotSwap(t *testing.T) {
	// Setup
	mockManager := new(MockPluginManager)
	storage := state.NewMemoryStateStorage()
	serializer := &state.JSONSerializer{}
	stateManager := state.NewStateManager(storage, serializer)
	mockRollback := new(MockRollbackManager)
	
	coordinator := state.NewHotSwapCoordinator(mockManager, stateManager, mockRollback)
	
	pluginID := "test-plugin"
	oldVersion := "1.0.0"
	newVersion := "2.0.0"
	checkpointID := "checkpoint-123"
	
	oldPlugin := &plugins.PluginInfo{
		ID:      pluginID,
		Version: oldVersion,
	}
	
	newSpec := plugins.PluginSpec{
		Name:    pluginID,
		Version: newVersion,
	}
	
	// Setup expectations
	mockManager.On("GetPlugin", pluginID).Return(oldPlugin, nil)
	mockRollback.On("CreateCheckpoint", mock.Anything, pluginID).Return(checkpointID, nil)
	mockManager.On("LoadPlugin", mock.Anything, newSpec).Return(nil)
	mockManager.On("DrainRequests", mock.Anything, pluginID, 30*time.Second).Return(nil)
	mockManager.On("StopPlugin", mock.Anything, pluginID).Return(nil)
	mockManager.On("StartPlugin", mock.Anything, pluginID).Return(nil)
	mockManager.On("UnloadPlugin", mock.Anything, pluginID).Return(nil)
	mockRollback.On("CleanupCheckpoint", mock.Anything, checkpointID).Return(nil)
	
	// Execute
	ctx := context.Background()
	err := coordinator.HotSwap(ctx, pluginID, newSpec)
	
	// Verify
	require.NoError(t, err)
	mockManager.AssertExpectations(t)
	mockRollback.AssertExpectations(t)
	
	// Check status
	status, exists := coordinator.GetStatus(pluginID)
	assert.True(t, exists)
	assert.Equal(t, pluginID, status.PluginID)
	assert.Equal(t, oldVersion, status.OldVersion)
	assert.Equal(t, newVersion, status.NewVersion)
	assert.Equal(t, "completed", status.Status)
	assert.Nil(t, status.Error)
}

func TestHotSwapCoordinator_RollbackOnFailure(t *testing.T) {
	// Setup
	mockManager := new(MockPluginManager)
	storage := state.NewMemoryStateStorage()
	serializer := &state.JSONSerializer{}
	stateManager := state.NewStateManager(storage, serializer)
	mockRollback := new(MockRollbackManager)
	
	coordinator := state.NewHotSwapCoordinator(mockManager, stateManager, mockRollback)
	
	pluginID := "test-plugin"
	oldVersion := "1.0.0"
	newVersion := "2.0.0"
	checkpointID := "checkpoint-123"
	
	oldPlugin := &plugins.PluginInfo{
		ID:      pluginID,
		Version: oldVersion,
	}
	
	newSpec := plugins.PluginSpec{
		Name:    pluginID,
		Version: newVersion,
	}
	
	loadError := errors.New("failed to load new plugin")
	
	// Setup expectations
	mockManager.On("GetPlugin", pluginID).Return(oldPlugin, nil)
	mockRollback.On("CreateCheckpoint", mock.Anything, pluginID).Return(checkpointID, nil)
	mockManager.On("LoadPlugin", mock.Anything, newSpec).Return(loadError)
	mockRollback.On("Rollback", mock.Anything, pluginID, checkpointID).Return(nil)
	
	// Execute
	ctx := context.Background()
	err := coordinator.HotSwap(ctx, pluginID, newSpec)
	
	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rolled back successfully")
	mockManager.AssertExpectations(t)
	mockRollback.AssertExpectations(t)
	
	// Check status
	status, exists := coordinator.GetStatus(pluginID)
	assert.True(t, exists)
	assert.Equal(t, "failed", status.Status)
	assert.NotNil(t, status.Error)
}

func TestHotSwapCoordinator_ListOperations(t *testing.T) {
	// Setup
	mockManager := new(MockPluginManager)
	storage := state.NewMemoryStateStorage()
	serializer := &state.JSONSerializer{}
	stateManager := state.NewStateManager(storage, serializer)
	mockRollback := new(MockRollbackManager)
	
	coordinator := state.NewHotSwapCoordinator(mockManager, stateManager, mockRollback)
	
	// Create multiple operations
	for i := 0; i < 3; i++ {
		pluginID := fmt.Sprintf("plugin-%d", i)
		
		oldPlugin := &plugins.PluginInfo{
			ID:      pluginID,
			Version: "1.0.0",
		}
		
		newSpec := plugins.PluginSpec{
			Name:    pluginID,
			Version: "2.0.0",
		}
		
		mockManager.On("GetPlugin", pluginID).Return(oldPlugin, nil).Once()
		mockRollback.On("CreateCheckpoint", mock.Anything, pluginID).Return("checkpoint-"+pluginID, nil).Once()
		
		if i == 1 {
			// Make one fail
			mockManager.On("LoadPlugin", mock.Anything, newSpec).Return(errors.New("load failed")).Once()
			mockRollback.On("Rollback", mock.Anything, pluginID, "checkpoint-"+pluginID).Return(nil).Once()
		} else {
			// Others succeed
			mockManager.On("LoadPlugin", mock.Anything, newSpec).Return(nil).Once()
			mockManager.On("DrainRequests", mock.Anything, pluginID, 30*time.Second).Return(nil).Once()
			mockManager.On("StopPlugin", mock.Anything, pluginID).Return(nil).Once()
			mockManager.On("StartPlugin", mock.Anything, pluginID).Return(nil).Once()
			mockManager.On("UnloadPlugin", mock.Anything, pluginID).Return(nil).Once()
			mockRollback.On("CleanupCheckpoint", mock.Anything, "checkpoint-"+pluginID).Return(nil).Once()
		}
		
		ctx := context.Background()
		_ = coordinator.HotSwap(ctx, pluginID, newSpec)
	}
	
	// List operations
	operations := coordinator.ListOperations()
	assert.Len(t, operations, 3)
	
	// Count statuses
	completed := 0
	failed := 0
	for _, op := range operations {
		switch op.Status {
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}
	
	assert.Equal(t, 2, completed)
	assert.Equal(t, 1, failed)
}

func TestHotSwapCoordinator_CleanupOldOperations(t *testing.T) {
	// Setup
	mockManager := new(MockPluginManager)
	storage := state.NewMemoryStateStorage()
	serializer := &state.JSONSerializer{}
	stateManager := state.NewStateManager(storage, serializer)
	mockRollback := new(MockRollbackManager)
	
	coordinator := state.NewHotSwapCoordinator(mockManager, stateManager, mockRollback)
	
	// Create an old operation by manually setting its status
	// This is a simplified test - in real code we'd need access to internal state
	operations := coordinator.ListOperations()
	assert.Len(t, operations, 0)
	
	// Test cleanup (should not panic even with no operations)
	coordinator.CleanupOldOperations(24 * time.Hour)
}