// Package process provides the implementation of the process orchestrator
// which manages service processes for the Blackhole platform.
package process

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/handcraftdev/blackhole/internal/core/config/types"
	processTypes "github.com/handcraftdev/blackhole/internal/core/process/types"
	processTesting "github.com/handcraftdev/blackhole/internal/core/process/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestSpawnService_EdgeCases tests edge cases in the SpawnService method
func TestSpawnService_EdgeCases(t *testing.T) {
	// Create test orchestrator
	orch, mockExec, _, tempDir := setupTestOrchestrator(t)

	// Test case 1: Service not found
	t.Run("Service not found", func(t *testing.T) {
		err := orch.SpawnService("nonexistent-service")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no configuration found")
	})

	// Test case 2: Binary not found
	t.Run("Binary not found", func(t *testing.T) {
		// Add a service with non-existent binary
		orch.processLock.Lock()
		orch.services["missing-binary"] = &types.ServiceConfig{
			Enabled:    true,
			BinaryPath: "/nonexistent/path/to/binary",
		}
		orch.processLock.Unlock()

		err := orch.SpawnService("missing-binary")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "binary not found")
	})

	// Test case 3: Start error
	t.Run("Start error", func(t *testing.T) {
		// Mock start error
		startErr := errors.New("start error")
		mockCmd := &processTesting.MockProcessCmd{
			StartFunc: func() error {
				return startErr
			},
		}

		mockExec.CommandFunc = func(path string, args ...string) processTypes.ProcessCmd {
			return mockCmd
		}

		err := orch.SpawnService("service1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start error")
	})

	// Test case 4: Already running
	t.Run("Already running", func(t *testing.T) {
		// Setup a running process
		orch.processLock.Lock()
		orch.processes["service1"] = &ServiceProcess{
			Name:  "service1",
			State: processTypes.ProcessStateRunning,
			PID:   1000,
		}
		orch.processLock.Unlock()

		// Mock functions shouldn't be called
		mockCmd := &processTesting.MockProcessCmd{
			StartFunc: func() error {
				t.Fatal("Start should not be called")
				return nil
			},
		}

		mockExec.CommandFunc = func(path string, args ...string) processTypes.ProcessCmd {
			return mockCmd
		}

		// Try to start the already running service
		err := orch.SpawnService("service1")
		require.NoError(t, err)

		// Verify state is unchanged
		orch.processLock.RLock()
		process := orch.processes["service1"]
		orch.processLock.RUnlock()
		assert.Equal(t, processTypes.ProcessStateRunning, process.State)
	})

	// Test case 5: Orchestrator shutting down
	t.Run("Orchestrator shutting down", func(t *testing.T) {
		// Mark orchestrator as shutting down
		orch.isShuttingDown.Store(true)
		defer orch.isShuttingDown.Store(false)

		// Mock functions shouldn't be called
		mockCmd := &processTesting.MockProcessCmd{
			StartFunc: func() error {
				t.Fatal("Start should not be called")
				return nil
			},
		}

		mockExec.CommandFunc = func(path string, args ...string) processTypes.ProcessCmd {
			return mockCmd
		}

		err := orch.SpawnService("service1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "shutting down")
	})

	// Test case 6: Restarting service
	t.Run("Restarting service", func(t *testing.T) {
		// Reset the spawner
		mockExec.CommandFunc = func(path string, args ...string) processTypes.ProcessCmd {
			return &processTesting.MockProcessCmd{
				StartFunc: func() error {
					return nil
				},
				ProcessFunc: func() processTypes.Process {
					return &processTesting.MockProcess{
						PidFunc: func() int {
							return 1001
						},
					}
				},
			}
		}

		// Setup a process in restarting state
		orch.processLock.Lock()
		orch.processes["service1"] = &ServiceProcess{
			Name:     "service1",
			State:    processTypes.ProcessStateRestarting,
			PID:      1000,
			Restarts: 1,
			StopCh:   make(chan struct{}),
		}
		orch.processLock.Unlock()

		// Spawn the service
		err := orch.SpawnService("service1")
		require.NoError(t, err)

		// Verify restarts incremented
		orch.processLock.RLock()
		process := orch.processes["service1"]
		orch.processLock.RUnlock()
		assert.Equal(t, 2, process.Restarts)
		assert.Equal(t, 1001, process.PID)
	})
}

// TestStopService_EdgeCases tests edge cases in the StopService method
func TestStopService_EdgeCases(t *testing.T) {
	// Create test orchestrator
	orch, mockExec, _, _ := setupTestOrchestrator(t)

	// Test case 1: Service not found
	t.Run("Service not found", func(t *testing.T) {
		err := orch.StopService("nonexistent-service")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	// Test case 2: Service already stopped
	t.Run("Service already stopped", func(t *testing.T) {
		// Setup a stopped process
		orch.processLock.Lock()
		orch.processes["service1"] = &ServiceProcess{
			Name:  "service1",
			State: processTypes.ProcessStateStopped,
		}
		orch.processLock.Unlock()

		err := orch.StopService("service1")
		require.NoError(t, err)
	})

	// Test case 3: Signal error
	t.Run("Signal error", func(t *testing.T) {
		// Setup a running process
		stopCh := make(chan struct{})
		orch.processLock.Lock()
		orch.processes["service1"] = &ServiceProcess{
			Name:    "service1",
			State:   processTypes.ProcessStateRunning,
			PID:     1000,
			StopCh:  stopCh,
			Command: &processTesting.MockProcessCmd{
				SignalFunc: func(sig os.Signal) error {
					return errors.New("signal error")
				},
				WaitFunc: func() error {
					return nil
				},
			},
			CommandWait: func() error {
				return nil
			},
		}
		orch.processLock.Unlock()

		// Stop should still succeed despite signal error
		err := orch.StopService("service1")
		require.NoError(t, err)

		// Verify state is updated
		orch.processLock.RLock()
		process := orch.processes["service1"]
		orch.processLock.RUnlock()
		assert.Equal(t, processTypes.ProcessStateStopped, process.State)
	})

	// Test case 4: Wait timeout
	t.Run("Wait timeout", func(t *testing.T) {
		// Set very short timeout
		orch.config.ShutdownTimeout = 1

		// Setup a running process with wait that never returns
		stopCh := make(chan struct{})
		waitCh := make(chan struct{})
		
		orch.processLock.Lock()
		orch.processes["service1"] = &ServiceProcess{
			Name:    "service1",
			State:   processTypes.ProcessStateRunning,
			PID:     1000,
			StopCh:  stopCh,
			Command: &processTesting.MockProcessCmd{
				SignalFunc: func(sig os.Signal) error {
					return nil
				},
				WaitFunc: func() error {
					<-waitCh // Block indefinitely
					return nil
				},
			},
			CommandWait: func() error {
				<-waitCh // Block indefinitely
				return nil
			},
		}
		orch.processLock.Unlock()

		// Stop should fall back to SIGKILL
		killCalled := false
		mockProcess := &processTesting.MockProcess{
			PidFunc: func() int {
				return 1000
			},
		}
		
		mockCmd := &processTesting.MockProcessCmd{
			ProcessFunc: func() processTypes.Process {
				return mockProcess
			},
			SignalFunc: func(sig os.Signal) error {
				if sig == os.Kill {
					killCalled = true
				}
				return nil
			},
		}
		
		orch.processLock.Lock()
		orch.processes["service1"].Command = mockCmd
		orch.processLock.Unlock()

		// Stop should still succeed with timeout
		err := orch.StopService("service1")
		require.NoError(t, err)

		// Verify SIGKILL was sent
		assert.True(t, killCalled)

		// Verify state is updated
		orch.processLock.RLock()
		process := orch.processes["service1"]
		orch.processLock.RUnlock()
		assert.Equal(t, processTypes.ProcessStateStopped, process.State)
		
		// Clean up
		close(waitCh)
	})
}