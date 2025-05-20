// Package process provides the implementation of the process orchestrator
// which manages service processes for the Blackhole platform.
package process

import (
	"testing"
	"time"

	"github.com/handcraftdev/blackhole/internal/core/config/types"
	processTypes "github.com/handcraftdev/blackhole/internal/core/process/types"
	processTesting "github.com/handcraftdev/blackhole/internal/core/process/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConfigManager is a simplified implementation of ConfigManager for testing
type mockConfigManager struct {
	config     *types.Config
	subscribers []func(*types.Config)
}

func (m *mockConfigManager) GetConfig() *types.Config {
	return m.config
}

func (m *mockConfigManager) SetConfig(config *types.Config) error {
	m.config = config
	// Notify subscribers
	for _, subscriber := range m.subscribers {
		subscriber(config)
	}
	return nil
}

func (m *mockConfigManager) SubscribeToChanges(callback func(*types.Config)) {
	m.subscribers = append(m.subscribers, callback)
}

// TestHandleConfigChange_RuntimeUpdates tests configuration changes while services are running
func TestHandleConfigChange_RuntimeUpdates(t *testing.T) {
	// Create test orchestrator with mock executor
	orch, mockExec, configManager, tempDir := setupTestOrchestrator(t)
	
	// Setup mock for process commands
	mockCmd := &processTesting.MockProcessCmd{
		StartFunc: func() error {
			return nil
		},
		WaitFunc: func() error {
			return nil
		},
		ProcessFunc: func() processTypes.Process {
			return &processTesting.MockProcess{
				PidFunc: func() int {
					return 1000
				},
			}
		},
	}
	
	mockExec.CommandFunc = func(path string, args ...string) processTypes.ProcessCmd {
		return mockCmd
	}
	
	// Test case 1: Adding a new service at runtime
	t.Run("Adding a new service", func(t *testing.T) {
		// Start with empty services
		orch.processLock.Lock()
		orch.services = make(map[string]*types.ServiceConfig)
		orch.processes = make(map[string]*ServiceProcess)
		orch.processLock.Unlock()
		
		// Get current config
		currentConfig := configManager.GetConfig()
		
		// Create a new config with a new service
		newConfig := *currentConfig
		newConfig.Services = make(map[string]*types.ServiceConfig)
		newConfig.Services["new-service"] = &types.ServiceConfig{
			Enabled:   true,
			DataDir:   tempDir + "/new-service",
			BinaryPath: tempDir + "/new-service/binary",
		}
		
		// Update the config
		err := configManager.SetConfig(&newConfig)
		require.NoError(t, err)
		
		// Verify service was added
		orch.processLock.RLock()
		service, exists := orch.services["new-service"]
		orch.processLock.RUnlock()
		
		assert.True(t, exists, "New service should be added to services map")
		assert.Equal(t, tempDir+"/new-service", service.DataDir)
	})
	
	// Test case 2: Removing a running service
	t.Run("Removing a running service", func(t *testing.T) {
		// Start with a running service
		orch.processLock.Lock()
		orch.services["running-service"] = &types.ServiceConfig{
			Enabled: true,
		}
		orch.processes["running-service"] = &ServiceProcess{
			Name:    "running-service",
			State:   processTypes.ProcessStateRunning,
			PID:     1001,
			StopCh:  make(chan struct{}),
			Command: mockCmd,
		}
		orch.processLock.Unlock()
		
		// Get current config
		currentConfig := configManager.GetConfig()
		
		// Create a new config without the running service
		newConfig := *currentConfig
		newConfig.Services = make(map[string]*types.ServiceConfig)
		
		// Add other services but not the running one
		for name, service := range currentConfig.Services {
			if name != "running-service" {
				newConfig.Services[name] = service
			}
		}
		
		// Update the config
		err := configManager.SetConfig(&newConfig)
		require.NoError(t, err)
		
		// Wait a bit for async stop to complete
		time.Sleep(50 * time.Millisecond)
		
		// Verify service was removed from services map
		orch.processLock.RLock()
		_, serviceExists := orch.services["running-service"]
		process, processExists := orch.processes["running-service"]
		orch.processLock.RUnlock()
		
		assert.False(t, serviceExists, "Service should be removed from services map")
		
		// Process might still exist but should be stopped
		if processExists {
			assert.Equal(t, processTypes.ProcessStateStopped, process.State, "Process should be stopped")
		}
	})
	
	// Test case 3: Updating a service configuration
	t.Run("Updating service configuration", func(t *testing.T) {
		// Start with a configured service
		orch.processLock.Lock()
		orch.services["update-service"] = &types.ServiceConfig{
			Enabled: true,
			Args:    []string{"--old-arg"},
			Environment: map[string]string{
				"OLD_ENV": "old_value",
			},
		}
		orch.processLock.Unlock()
		
		// Get current config
		currentConfig := configManager.GetConfig()
		
		// Create a new config with updated service
		newConfig := *currentConfig
		newConfig.Services = make(map[string]*types.ServiceConfig)
		
		// Copy all services except the one we're updating
		for name, service := range currentConfig.Services {
			if name != "update-service" {
				newConfig.Services[name] = service
			}
		}
		
		// Add updated service
		newConfig.Services["update-service"] = &types.ServiceConfig{
			Enabled: true,
			Args:    []string{"--new-arg", "--debug"},
			Environment: map[string]string{
				"NEW_ENV": "new_value",
			},
		}
		
		// Update the config
		err := configManager.SetConfig(&newConfig)
		require.NoError(t, err)
		
		// Verify service was updated
		orch.processLock.RLock()
		service, exists := orch.services["update-service"]
		orch.processLock.RUnlock()
		
		assert.True(t, exists, "Service should still exist")
		assert.Equal(t, []string{"--new-arg", "--debug"}, service.Args, "Args should be updated")
		assert.Equal(t, "new_value", service.Environment["NEW_ENV"], "Environment should be updated")
		assert.NotContains(t, service.Environment, "OLD_ENV", "Old environment variables should be removed")
	})
	
	// Test case 4: Disabling an enabled service
	t.Run("Disabling a service", func(t *testing.T) {
		// Start with an enabled service
		orch.processLock.Lock()
		orch.services["enabled-service"] = &types.ServiceConfig{
			Enabled: true,
		}
		// Also start the service
		orch.processes["enabled-service"] = &ServiceProcess{
			Name:    "enabled-service",
			State:   processTypes.ProcessStateRunning,
			PID:     1002,
			StopCh:  make(chan struct{}),
			Command: mockCmd,
		}
		orch.processLock.Unlock()
		
		// Get current config
		currentConfig := configManager.GetConfig()
		
		// Create a new config with service disabled
		newConfig := *currentConfig
		newConfig.Services = make(map[string]*types.ServiceConfig)
		
		// Copy all services
		for name, service := range currentConfig.Services {
			serviceCopy := *service
			newConfig.Services[name] = &serviceCopy
		}
		
		// Disable the service
		newConfig.Services["enabled-service"].Enabled = false
		
		// Update the config
		err := configManager.SetConfig(&newConfig)
		require.NoError(t, err)
		
		// Verify service was disabled
		orch.processLock.RLock()
		service, exists := orch.services["enabled-service"]
		orch.processLock.RUnlock()
		
		assert.True(t, exists, "Service should still exist")
		assert.False(t, service.Enabled, "Service should be disabled")
		
		// Service should NOT be stopped just because it was disabled
		// Stopping happens only when services are removed entirely
	})
	
	// Test case 5: Updating orchestrator config
	t.Run("Updating orchestrator config", func(t *testing.T) {
		// Get current config
		currentConfig := configManager.GetConfig()
		
		// Create a new config with updated orchestrator settings
		newConfig := *currentConfig
		newConfig.Orchestrator.LogLevel = "debug"
		newConfig.Orchestrator.AutoRestart = !currentConfig.Orchestrator.AutoRestart
		newConfig.Orchestrator.ShutdownTimeout = 60
		
		// Update the config
		err := configManager.SetConfig(&newConfig)
		require.NoError(t, err)
		
		// Verify orchestrator config was updated
		assert.Equal(t, "debug", orch.config.LogLevel, "Log level should be updated")
		assert.Equal(t, !currentConfig.Orchestrator.AutoRestart, orch.config.AutoRestart, "AutoRestart should be toggled")
		assert.Equal(t, 60, orch.config.ShutdownTimeout, "ShutdownTimeout should be updated")
	})
	
	// Test case 6: Changing service binary path
	t.Run("Changing service binary path", func(t *testing.T) {
		// Start with a service with binary path
		orch.processLock.Lock()
		orch.services["path-service"] = &types.ServiceConfig{
			Enabled:    true,
			BinaryPath: tempDir + "/old/path",
		}
		orch.processLock.Unlock()
		
		// Get current config
		currentConfig := configManager.GetConfig()
		
		// Create a new config with updated binary path
		newConfig := *currentConfig
		newConfig.Services = make(map[string]*types.ServiceConfig)
		
		// Copy all services except the one we're updating
		for name, service := range currentConfig.Services {
			if name != "path-service" {
				newConfig.Services[name] = service
			}
		}
		
		// Add updated service
		newConfig.Services["path-service"] = &types.ServiceConfig{
			Enabled:    true,
			BinaryPath: tempDir + "/new/path",
		}
		
		// Update the config
		err := configManager.SetConfig(&newConfig)
		require.NoError(t, err)
		
		// Verify binary path was updated
		orch.processLock.RLock()
		service, exists := orch.services["path-service"]
		orch.processLock.RUnlock()
		
		assert.True(t, exists, "Service should still exist")
		assert.Equal(t, tempDir+"/new/path", service.BinaryPath, "Binary path should be updated")
	})
}