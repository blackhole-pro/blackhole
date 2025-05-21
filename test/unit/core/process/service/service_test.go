// Package service_test provides tests for service lifecycle management functionality.
package service_test

import (
	"errors"
	"io"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/handcraftdev/blackhole/internal/core/config/types"
	"github.com/handcraftdev/blackhole/internal/core/process/service"
	processTypes "github.com/handcraftdev/blackhole/internal/core/process/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// MockProcessCmd is a simple mock for testing
type MockProcessCmd struct{}

func (m *MockProcessCmd) Start() error { return nil }
func (m *MockProcessCmd) Wait() error { return nil }
func (m *MockProcessCmd) SetEnv(env []string) {}
func (m *MockProcessCmd) SetDir(dir string) {}
func (m *MockProcessCmd) SetOutput(stdout, stderr io.Writer) {}
func (m *MockProcessCmd) Signal(sig os.Signal) error { return nil }
func (m *MockProcessCmd) Process() processTypes.Process { return nil }

// MockProcess is a simple mock for testing
type MockProcess struct{}

func (m *MockProcess) Pid() int { return 1000 }
func (m *MockProcess) Kill() error { return nil }

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
		processes := map[string]*service.ServiceProcess{}
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

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
		processes := map[string]*service.ServiceProcess{}
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

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
		processes := map[string]*service.ServiceProcess{
			"test-service": {
				Name:    "test-service",
				State:   processTypes.ProcessStateRunning,
				Command: &MockProcessCmd{},
				StopCh:  make(chan struct{}),
			},
		}
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

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
		processes := map[string]*service.ServiceProcess{}
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

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
		processes := map[string]*service.ServiceProcess{}
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

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
		processes := map[string]*service.ServiceProcess{
			"test-service": {
				Name:    "test-service",
				State:   processTypes.ProcessStateRunning,
				PID:     1000,
				Command: &MockProcessCmd{},
				StopCh:  stopCh,
			},
		}
		
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

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
		assert.Equal(t, processTypes.ProcessStateStopped, processes["test-service"].State)
	})

	t.Run("Service not found", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{}
		processes := map[string]*service.ServiceProcess{}
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

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
		processes := map[string]*service.ServiceProcess{
			"test-service": {
				Name:    "test-service",
				State:   processTypes.ProcessStateStopped,
				Command: &MockProcessCmd{},
			},
		}
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

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
		processes := map[string]*service.ServiceProcess{
			"test-service": {
				Name:    "test-service",
				State:   processTypes.ProcessStateRunning,
				PID:     1000,
				Command: &MockProcessCmd{},
				StopCh:  stopCh,
			},
		}
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

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
		processes := map[string]*service.ServiceProcess{
			"test-service": {
				Name:    "test-service",
				State:   processTypes.ProcessStateRunning,
				PID:     1000,
				Command: &MockProcessCmd{},
				StopCh:  make(chan struct{}),
			},
		}
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

		// Expected error
		expectedErr := errors.New("signal error")

		// Mock signal function
		signalFn := func(name string, sig syscall.Signal) error {
			if sig == syscall.SIGKILL {
				return expectedErr
			}
			return nil
		}

		// Create a wait function that times out
		processes["test-service"].CommandWait = func() error {
			time.Sleep(100 * time.Millisecond) // More than our tiny timeout
			return nil
		}

		// Call StopService with very short timeout
		err := manager.StopService("test-service", signalFn, 0)
		require.Error(t, err)
		assert.Equal(t, expectedErr, errors.Unwrap(err))
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
		processes := map[string]*service.ServiceProcess{
			"test-service": {
				Name:    "test-service",
				State:   processTypes.ProcessStateRunning,
				Command: &MockProcessCmd{},
				StopCh:  make(chan struct{}),
			},
		}
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

		// Track function calls
		stopCalled := false
		startCalled := false

		stopFn := func(name string) error {
			assert.Equal(t, "test-service", name)
			stopCalled = true
			return nil
		}

		startFn := func(name string) error {
			assert.Equal(t, "test-service", name)
			startCalled = true
			return nil
		}

		// Call RestartService
		err := manager.RestartService("test-service", stopFn, startFn)
		require.NoError(t, err)

		// Verify functions were called
		assert.True(t, stopCalled)
		assert.True(t, startCalled)

		// Verify state was updated to restarting before the functions were called
		assert.Equal(t, processTypes.ProcessStateRestarting, processes["test-service"].State)
	})

	t.Run("Stop error during restart", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		processes := map[string]*service.ServiceProcess{
			"test-service": {
				Name:    "test-service",
				State:   processTypes.ProcessStateRunning,
				Command: &MockProcessCmd{},
				StopCh:  make(chan struct{}),
			},
		}
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

		// Track function calls
		startCalled := false

		stopFn := func(name string) error {
			return errors.New("stop error")
		}

		startFn := func(name string) error {
			startCalled = true
			return nil
		}

		// Call RestartService
		err := manager.RestartService("test-service", stopFn, startFn)
		require.NoError(t, err) // Errors from stopFn are logged but not returned

		// Start should still be called even with stop error
		assert.True(t, startCalled)
	})

	t.Run("Start error during restart", func(t *testing.T) {
		// Setup
		services := map[string]*types.ServiceConfig{
			"test-service": {
				Enabled: true,
			},
		}
		processes := map[string]*service.ServiceProcess{
			"test-service": {
				Name:    "test-service",
				State:   processTypes.ProcessStateRunning,
				Command: &MockProcessCmd{},
				StopCh:  make(chan struct{}),
			},
		}
		var processLock sync.RWMutex
		manager := service.NewManager(services, processes, &processLock, logger)

		// Track function calls
		stopCalled := false
		expectedErr := errors.New("start error")

		stopFn := func(name string) error {
			stopCalled = true
			return nil
		}

		startFn := func(name string) error {
			return expectedErr
		}

		// Call RestartService
		err := manager.RestartService("test-service", stopFn, startFn)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)

		// Stop should be called
		assert.True(t, stopCalled)
	})
}