// Package service provides service lifecycle management for the Process Orchestrator.
// It handles starting, stopping, and restarting services.
package service

import (
	"errors"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/handcraftdev/blackhole/internal/core/config/types"
	processTypes "github.com/handcraftdev/blackhole/internal/core/process/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// Setup test dependencies

// MockServiceInfo represents service information for testing
type MockServiceInfo struct {
	Name      string
	State     string
	StopCh    chan struct{}
	CommandWait func() error
}

// MockProcessSpawner mocks the process spawning functionality
type MockProcessSpawner struct {
	SpawnFunc       func(string) error
	SpawnedServices []string
}

func (m *MockProcessSpawner) SpawnProcess(name string) error {
	m.SpawnedServices = append(m.SpawnedServices, name)
	if m.SpawnFunc != nil {
		return m.SpawnFunc(name)
	}
	return nil
}

// TestManager_StartService tests the StartService method
func TestManager_StartService(t *testing.T) {
	// Create test logger
	logger := zaptest.NewLogger(t)

	t.Run("Service enabled and not running", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		processes := map[string]*MockServiceInfo{}
		var processLock sync.RWMutex
		manager := NewManager(services, nil, &processLock, logger)

		// Create mock spawner
		spawner := &MockProcessSpawner{}

		// Call StartService
		err := manager.StartService("test-service", spawner.SpawnProcess)
		require.NoError(t, err)

		// Verify spawner was called
		assert.Contains(t, spawner.SpawnedServices, "test-service")
	})

	t.Run("Service disabled", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: false,
			},
		}
		processes := map[string]*MockServiceInfo{}
		var processLock sync.RWMutex
		manager := NewManager(services, nil, &processLock, logger)

		// Create mock spawner
		spawner := &MockProcessSpawner{}

		// Call StartService
		err := manager.StartService("test-service", spawner.SpawnProcess)
		require.NoError(t, err)

		// Verify spawner was not called
		assert.Empty(t, spawner.SpawnedServices)
	})

	t.Run("Service already running", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		processes := map[string]*processTypes.ServiceInfo{
			"test-service": {
				Name:  "test-service",
				State: string(processTypes.ProcessStateRunning),
			},
		}
		var processLock sync.RWMutex
		manager := NewManager(services, processes, &processLock, logger)

		// Create mock spawner
		spawner := &MockProcessSpawner{}

		// Call StartService
		err := manager.StartService("test-service", spawner.SpawnProcess)
		require.NoError(t, err)

		// Verify spawner was not called
		assert.Empty(t, spawner.SpawnedServices)
	})

	t.Run("Service not configured", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{}
		processes := map[string]*processTypes.ServiceInfo{}
		var processLock sync.RWMutex
		manager := NewManager(services, nil, &processLock, logger)

		// Create mock spawner
		spawner := &MockProcessSpawner{}

		// Call StartService
		err := manager.StartService("test-service", spawner.SpawnProcess)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no configuration found")

		// Verify spawner was not called
		assert.Empty(t, spawner.SpawnedServices)
	})

	t.Run("Spawn error", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		processes := map[string]*processTypes.ServiceInfo{}
		var processLock sync.RWMutex
		manager := NewManager(services, nil, &processLock, logger)

		// Create mock spawner with error
		expectedErr := errors.New("spawn error")
		spawner := &MockProcessSpawner{
			SpawnFunc: func(name string) error {
				return expectedErr
			},
		}

		// Call StartService
		err := manager.StartService("test-service", spawner.SpawnProcess)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)

		// Verify spawner was called
		assert.Contains(t, spawner.SpawnedServices, "test-service")
	})
}

// TestManager_StopService tests the StopService method
func TestManager_StopService(t *testing.T) {
	// Create test logger
	logger := zaptest.NewLogger(t)

	t.Run("Stop running service", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		
		stopCh := make(chan struct{})
		processes := map[string]*processTypes.ServiceInfo{
			"test-service": {
				Name:  "test-service",
				State: string(processTypes.ProcessStateRunning),
				PID:   1000,
				StopCh: stopCh,
			},
		}
		
		var processLock sync.RWMutex
		manager := NewManager(services, processes, &processLock, logger)

		// Mock signal function
		signalCalled := false
		signalFn := func(name string, sig syscall.Signal) error {
			signalCalled = true
			assert.Equal(t, "test-service", name)
			assert.Equal(t, syscall.SIGTERM, sig)
			return nil
		}

		// Create a wait function that returns immediately
		processes["test-service"].CommandWait = func() error {
			return nil
		}

		// Call StopService
		err := manager.StopService("test-service", signalFn, 5)
		require.NoError(t, err)

		// Verify signal was called
		assert.True(t, signalCalled)

		// Verify state was updated
		assert.Equal(t, string(processTypes.ProcessStateStopped), processes["test-service"].State)
	})

	t.Run("Service not found", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{}
		processes := map[string]*processTypes.ServiceInfo{}
		var processLock sync.RWMutex
		manager := NewManager(services, processes, &processLock, logger)

		// Mock signal function
		signalFn := func(name string, sig syscall.Signal) error {
			t.Fatal("Signal should not be called")
			return nil
		}

		// Call StopService
		err := manager.StopService("test-service", signalFn, 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Service already stopped", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		processes := map[string]*processTypes.ServiceInfo{
			"test-service": {
				Name:  "test-service",
				State: string(processTypes.ProcessStateStopped),
			},
		}
		var processLock sync.RWMutex
		manager := NewManager(services, processes, &processLock, logger)

		// Mock signal function
		signalFn := func(name string, sig syscall.Signal) error {
			t.Fatal("Signal should not be called")
			return nil
		}

		// Call StopService
		err := manager.StopService("test-service", signalFn, 5)
		require.NoError(t, err)
	})

	t.Run("SIGTERM failure with SIGKILL fallback", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		stopCh := make(chan struct{})
		processes := map[string]*processTypes.ServiceInfo{
			"test-service": {
				Name:  "test-service",
				State: string(processTypes.ProcessStateRunning),
				PID:   1000,
				StopCh: stopCh,
			},
		}
		var processLock sync.RWMutex
		manager := NewManager(services, processes, &processLock, logger)

		// Signal tracking
		signals := make(map[syscall.Signal]bool)

		// Mock signal function
		signalFn := func(name string, sig syscall.Signal) error {
			signals[sig] = true
			assert.Equal(t, "test-service", name)
			return nil
		}

		// Create a wait function that timeouts
		processes["test-service"].CommandWait = func() error {
			time.Sleep(100 * time.Millisecond) // More than our tiny timeout
			return nil
		}

		// Call StopService with very short timeout
		err := manager.StopService("test-service", signalFn, 0)
		require.NoError(t, err)

		// Verify both signals were called
		assert.True(t, signals[syscall.SIGTERM])
		assert.True(t, signals[syscall.SIGKILL])
	})

	t.Run("Signal error", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		stopCh := make(chan struct{})
		processes := map[string]*processTypes.ServiceInfo{
			"test-service": {
				Name:  "test-service",
				State: string(processTypes.ProcessStateRunning),
				PID:   1000,
				StopCh: stopCh,
			},
		}
		var processLock sync.RWMutex
		manager := NewManager(services, processes, &processLock, logger)

		// Mock signal function with error
		expectedErr := errors.New("signal error")
		signalFn := func(name string, sig syscall.Signal) error {
			return expectedErr
		}

		// Call StopService
		err := manager.StopService("test-service", signalFn, 5)
		require.NoError(t, err) // It should still continue and try to kill the process

		// Verify state was updated
		assert.Equal(t, string(processTypes.ProcessStateStopped), processes["test-service"].State)
	})
}

// TestManager_RestartService tests the RestartService method
func TestManager_RestartService(t *testing.T) {
	// Create test logger
	logger := zaptest.NewLogger(t)

	t.Run("Restart running service", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		processes := map[string]*processTypes.ServiceInfo{
			"test-service": {
				Name:  "test-service",
				State: string(processTypes.ProcessStateRunning),
			},
		}
		var processLock sync.RWMutex
		manager := NewManager(services, processes, &processLock, logger)

		// Track function calls
		stopCalled := false
		startCalled := false

		stopFn := func(name string) error {
			stopCalled = true
			assert.Equal(t, "test-service", name)
			return nil
		}

		startFn := func(name string) error {
			startCalled = true
			assert.Equal(t, "test-service", name)
			return nil
		}

		// Call RestartService
		err := manager.RestartService("test-service", stopFn, startFn)
		require.NoError(t, err)

		// Verify functions were called
		assert.True(t, stopCalled)
		assert.True(t, startCalled)

		// Verify state was updated to restarting during the operation
		assert.Equal(t, string(processTypes.ProcessStateRestarting), processes["test-service"].State)
	})

	t.Run("Stop error during restart", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		processes := map[string]*processTypes.ServiceInfo{
			"test-service": {
				Name:  "test-service",
				State: string(processTypes.ProcessStateRunning),
			},
		}
		var processLock sync.RWMutex
		manager := NewManager(services, processes, &processLock, logger)

		// Stop function fails but restart continues
		stopErr := errors.New("stop error")
		stopFn := func(name string) error {
			return stopErr
		}

		startCalled := false
		startFn := func(name string) error {
			startCalled = true
			return nil
		}

		// Call RestartService
		err := manager.RestartService("test-service", stopFn, startFn)
		require.NoError(t, err)

		// Verify start was still called despite stop error
		assert.True(t, startCalled)
	})

	t.Run("Start error during restart", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		processes := map[string]*processTypes.ServiceInfo{
			"test-service": {
				Name:  "test-service",
				State: string(processTypes.ProcessStateRunning),
			},
		}
		var processLock sync.RWMutex
		manager := NewManager(services, processes, &processLock, logger)

		// Stop succeeds but start fails
		stopCalled := false
		stopFn := func(name string) error {
			stopCalled = true
			return nil
		}

		startErr := errors.New("start error")
		startFn := func(name string) error {
			return startErr
		}

		// Call RestartService
		err := manager.RestartService("test-service", stopFn, startFn)
		require.Error(t, err)
		assert.Equal(t, startErr, err)

		// Verify stop was called
		assert.True(t, stopCalled)
	})
}

// TestInfoProvider_GetServiceInfo tests the GetServiceInfo method
func TestInfoProvider_GetServiceInfo(t *testing.T) {
	t.Run("Get running service info", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		processes := map[string]*processTypes.ServiceInfo{
			"test-service": {
				Name:      "test-service",
				Configured: true,
				Enabled:   true,
				State:     string(processTypes.ProcessStateRunning),
				PID:       1000,
				Restarts:  2,
				LastError: "test error",
			},
		}
		var processLock sync.RWMutex
		provider := NewInfoProvider(services, processes, &processLock)

		// Get service info
		info, err := provider.GetServiceInfo("test-service")
		require.NoError(t, err)

		// Verify info
		assert.Equal(t, "test-service", info.Name)
		assert.Equal(t, string(processTypes.ProcessStateRunning), info.State)
		assert.Equal(t, 1000, info.PID)
		assert.Equal(t, 2, info.Restarts)
		assert.Equal(t, "test error", info.LastError)
	})

	t.Run("Get non-running service info", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		processes := map[string]*processTypes.ServiceInfo{}
		var processLock sync.RWMutex
		provider := NewInfoProvider(services, processes, &processLock)

		// Get service info
		info, err := provider.GetServiceInfo("test-service")
		require.NoError(t, err)

		// Verify info
		assert.Equal(t, "test-service", info.Name)
		assert.Equal(t, string(processTypes.ProcessStateStopped), info.State)
		assert.True(t, info.Configured)
		assert.True(t, info.Enabled)
	})

	t.Run("Get non-existent service info", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{}
		processes := map[string]*processTypes.ServiceInfo{}
		var processLock sync.RWMutex
		provider := NewInfoProvider(services, processes, &processLock)

		// Get service info
		info, err := provider.GetServiceInfo("test-service")
		require.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "not configured")
	})
}

// TestInfoProvider_GetAllServices tests the GetAllServices method
func TestInfoProvider_GetAllServices(t *testing.T) {
	t.Run("Get all services", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"service1": {
				Enabled: true,
			},
			"service2": {
				Enabled: false,
			},
		}
		processes := map[string]*processTypes.ServiceInfo{
			"service1": {
				Name:      "service1",
				Configured: true,
				Enabled:   true,
				State:     string(processTypes.ProcessStateRunning),
				PID:       1000,
			},
		}
		var processLock sync.RWMutex
		provider := NewInfoProvider(services, processes, &processLock)

		// Get all services
		allServices, err := provider.GetAllServices()
		require.NoError(t, err)

		// Verify all services
		assert.Len(t, allServices, 2)
		
		// Verify running service
		assert.Contains(t, allServices, "service1")
		assert.Equal(t, string(processTypes.ProcessStateRunning), allServices["service1"].State)
		assert.Equal(t, 1000, allServices["service1"].PID)
		
		// Verify stopped service
		assert.Contains(t, allServices, "service2")
		assert.Equal(t, string(processTypes.ProcessStateStopped), allServices["service2"].State)
		assert.False(t, allServices["service2"].Enabled)
	})

	t.Run("No services", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{}
		processes := map[string]*processTypes.ServiceInfo{}
		var processLock sync.RWMutex
		provider := NewInfoProvider(services, processes, &processLock)

		// Get all services
		allServices, err := provider.GetAllServices()
		require.NoError(t, err)
		assert.Empty(t, allServices)
	})
}