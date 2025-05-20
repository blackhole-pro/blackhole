// Package process provides the implementation of the process orchestrator
// which manages service processes for the Blackhole platform.
package process

import (
	"errors"
	"testing"
	"time"

	"github.com/handcraftdev/blackhole/internal/core/config/types"
	processTypes "github.com/handcraftdev/blackhole/internal/core/process/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetServiceInfo tests the GetServiceInfo method in more detail
func TestGetServiceInfo(t *testing.T) {
	// Create test orchestrator
	orch, _, _, _ := setupTestOrchestrator(t)
	
	// Test case 1: Get info for running service
	t.Run("Running service", func(t *testing.T) {
		// Setup a running process with all fields populated
		startTime := time.Now().Add(-10 * time.Minute)
		orch.processLock.Lock()
		orch.processes["service1"] = &ServiceProcess{
			Name:     "service1",
			State:    processTypes.ProcessStateRunning,
			PID:      1000,
			Started:  startTime,
			Restarts: 3,
			LastError: errors.New("previous error"),
		}
		orch.processLock.Unlock()
		
		// Get service info
		info, err := orch.GetServiceInfo("service1")
		require.NoError(t, err)
		
		// Verify all fields
		assert.Equal(t, "service1", info.Name)
		assert.True(t, info.Configured)
		assert.True(t, info.Enabled)
		assert.Equal(t, string(processTypes.ProcessStateRunning), info.State)
		assert.Equal(t, 1000, info.PID)
		assert.Equal(t, 3, info.Restarts)
		assert.Contains(t, info.LastError, "previous error")
		
		// Uptime should be approximately 10 minutes
		expectedUptime := 10 * time.Minute
		assert.True(t, info.Uptime >= expectedUptime-time.Second && 
			info.Uptime <= expectedUptime+time.Second,
			"Uptime should be ~10 minutes, got %v", info.Uptime)
	})
	
	// Test case 2: Get info for a service that's failed
	t.Run("Failed service", func(t *testing.T) {
		// Setup a failed process
		orch.processLock.Lock()
		orch.processes["service1"] = &ServiceProcess{
			Name:     "service1",
			State:    processTypes.ProcessStateFailed,
			PID:      1000,
			Started:  time.Now(),
			Restarts: 5,
			LastError: &processTypes.ProcessError{
				Service:  "service1",
				Err:      errors.New("fatal error"),
				ExitCode: 1,
			},
		}
		orch.processLock.Unlock()
		
		// Get service info
		info, err := orch.GetServiceInfo("service1")
		require.NoError(t, err)
		
		// Verify critical fields
		assert.Equal(t, string(processTypes.ProcessStateFailed), info.State)
		assert.Equal(t, 5, info.Restarts)
		assert.Contains(t, info.LastError, "fatal error")
		assert.Contains(t, info.LastError, "exit code: 1")
	})
	
	// Test case 3: Get info for configured but not running service
	t.Run("Configured but not running service", func(t *testing.T) {
		// Remove the process but keep the service config
		orch.processLock.Lock()
		delete(orch.processes, "service1")
		orch.services["service1"] = &types.ServiceConfig{
			Enabled: true,
		}
		orch.processLock.Unlock()
		
		// Get service info
		info, err := orch.GetServiceInfo("service1")
		require.NoError(t, err)
		
		// Verify it shows as configured but stopped
		assert.Equal(t, "service1", info.Name)
		assert.True(t, info.Configured)
		assert.True(t, info.Enabled)
		assert.Equal(t, string(processTypes.ProcessStateStopped), info.State)
		assert.Zero(t, info.PID)
		assert.Zero(t, info.Restarts)
		assert.Empty(t, info.LastError)
	})
	
	// Test case 4: Get info for disabled service
	t.Run("Disabled service", func(t *testing.T) {
		// Setup a disabled service
		orch.processLock.Lock()
		orch.services["disabled-service"] = &types.ServiceConfig{
			Enabled: false,
		}
		orch.processLock.Unlock()
		
		// Get service info
		info, err := orch.GetServiceInfo("disabled-service")
		require.NoError(t, err)
		
		// Verify it shows as disabled
		assert.Equal(t, "disabled-service", info.Name)
		assert.True(t, info.Configured)
		assert.False(t, info.Enabled)
		assert.Equal(t, string(processTypes.ProcessStateStopped), info.State)
	})
	
	// Test case 5: Get info for non-existent service
	t.Run("Non-existent service", func(t *testing.T) {
		// Get info for a service that doesn't exist
		info, err := orch.GetServiceInfo("nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not configured")
		assert.Nil(t, info)
	})
}

// TestGetAllServices tests the GetAllServices method in more detail
func TestGetAllServices(t *testing.T) {
	// Create test orchestrator
	orch, _, _, _ := setupTestOrchestrator(t)
	
	// Test case 1: Get all services with mixed states
	t.Run("Mixed service states", func(t *testing.T) {
		// Setup multiple services in different states
		orch.processLock.Lock()
		
		// Configure services
		orch.services = map[string]*types.ServiceConfig{
			"running": {
				Enabled: true,
			},
			"failed": {
				Enabled: true,
			},
			"stopped": {
				Enabled: true,
			},
			"disabled": {
				Enabled: false,
			},
			"notstarted": {
				Enabled: true,
			},
		}
		
		// Setup processes
		orch.processes = map[string]*ServiceProcess{
			"running": {
				Name:     "running",
				State:    processTypes.ProcessStateRunning,
				PID:      1001,
				Started:  time.Now().Add(-5 * time.Minute),
				Restarts: 0,
			},
			"failed": {
				Name:     "failed",
				State:    processTypes.ProcessStateFailed,
				PID:      1002,
				Started:  time.Now().Add(-2 * time.Minute),
				Restarts: 3,
				LastError: errors.New("crash error"),
			},
			"stopped": {
				Name:     "stopped",
				State:    processTypes.ProcessStateStopped,
				PID:      0,
				Started:  time.Now().Add(-10 * time.Minute),
				Restarts: 1,
			},
		}
		orch.processLock.Unlock()
		
		// Get all services
		services, err := orch.GetAllServices()
		require.NoError(t, err)
		
		// Verify we got all five services
		assert.Len(t, services, 5)
		
		// Check running service
		assert.Contains(t, services, "running")
		assert.Equal(t, string(processTypes.ProcessStateRunning), services["running"].State)
		assert.Equal(t, 1001, services["running"].PID)
		assert.Equal(t, 0, services["running"].Restarts)
		
		// Check failed service
		assert.Contains(t, services, "failed")
		assert.Equal(t, string(processTypes.ProcessStateFailed), services["failed"].State)
		assert.Equal(t, 1002, services["failed"].PID)
		assert.Equal(t, 3, services["failed"].Restarts)
		assert.Contains(t, services["failed"].LastError, "crash error")
		
		// Check stopped service
		assert.Contains(t, services, "stopped")
		assert.Equal(t, string(processTypes.ProcessStateStopped), services["stopped"].State)
		assert.Equal(t, 0, services["stopped"].PID)
		assert.Equal(t, 1, services["stopped"].Restarts)
		
		// Check disabled service
		assert.Contains(t, services, "disabled")
		assert.Equal(t, string(processTypes.ProcessStateStopped), services["disabled"].State)
		assert.False(t, services["disabled"].Enabled)
		
		// Check not-started service
		assert.Contains(t, services, "notstarted")
		assert.Equal(t, string(processTypes.ProcessStateStopped), services["notstarted"].State)
		assert.True(t, services["notstarted"].Enabled)
	})
	
	// Test case 2: No services configured
	t.Run("No services", func(t *testing.T) {
		// Clear all services and processes
		orch.processLock.Lock()
		orch.services = map[string]*types.ServiceConfig{}
		orch.processes = map[string]*ServiceProcess{}
		orch.processLock.Unlock()
		
		// Get all services
		services, err := orch.GetAllServices()
		require.NoError(t, err)
		
		// Verify we got an empty map
		assert.Empty(t, services)
	})
	
	// Test case 3: Processes without configs
	t.Run("Orphaned processes", func(t *testing.T) {
		// Setup processes without configs
		orch.processLock.Lock()
		orch.services = map[string]*types.ServiceConfig{}
		orch.processes = map[string]*ServiceProcess{
			"orphan": {
				Name:  "orphan",
				State: processTypes.ProcessStateRunning,
				PID:   1005,
			},
		}
		orch.processLock.Unlock()
		
		// Get all services
		services, err := orch.GetAllServices()
		require.NoError(t, err)
		
		// Verify no services are returned (only configured services)
		assert.Empty(t, services)
	})
}