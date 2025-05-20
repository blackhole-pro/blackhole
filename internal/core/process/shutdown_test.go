// Package process provides the implementation of the process orchestrator
// which manages service processes for the Blackhole platform.
package process

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	processTypes "github.com/handcraftdev/blackhole/internal/core/process/types"
	processTesting "github.com/handcraftdev/blackhole/internal/core/process/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestShutdown_ContextTimeout tests the Shutdown method with context timeout
func TestShutdown_ContextTimeout(t *testing.T) {
	// Create test orchestrator
	orch, mockExec, _, _ := setupTestOrchestrator(t)
	
	// Test case 1: Normal shutdown (all services stop gracefully)
	t.Run("Normal shutdown", func(t *testing.T) {
		// Setup a running process that exits gracefully
		stopCh := make(chan struct{})
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
					return nil
				},
			},
			CommandWait: func() error {
				return nil
			},
		}
		orch.processLock.Unlock()
		
		// Create context with reasonable timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		
		// Shutdown should succeed
		err := orch.Shutdown(ctx)
		require.NoError(t, err)
		
		// Verify orchestrator is marked as shutting down
		assert.True(t, orch.isShuttingDown.Load())
		
		// Verify service is stopped
		orch.processLock.RLock()
		process := orch.processes["service1"]
		orch.processLock.RUnlock()
		assert.Equal(t, processTypes.ProcessStateStopped, process.State)
	})
	
	// Test case 2: Context cancellation
	t.Run("Context cancellation", func(t *testing.T) {
		// Reset orchestrator state
		orch.isShuttingDown.Store(false)
		
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
		
		// Create context that we'll cancel immediately
		ctx, cancel := context.WithCancel(context.Background())
		
		// Start shutdown in a goroutine as it will block
		shutdownErrCh := make(chan error)
		go func() {
			shutdownErrCh <- orch.Shutdown(ctx)
		}()
		
		// Wait a bit to ensure shutdown has started
		time.Sleep(50 * time.Millisecond)
		
		// Cancel the context
		cancel()
		
		// Wait for shutdown to return
		var err error
		select {
		case err = <-shutdownErrCh:
			// Good, shutdown should return with error
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Shutdown did not return after context cancellation")
		}
		
		// Verify shutdown returned with context canceled error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
		
		// Clean up
		close(waitCh)
	})
	
	// Test case 3: Shutdown with some services failing to stop
	t.Run("Service stop errors", func(t *testing.T) {
		// Reset orchestrator state
		orch.isShuttingDown.Store(false)
		
		// Setup multiple services, one that stops normally and one that fails
		orch.processLock.Lock()
		
		// Service that stops normally
		orch.processes["service1"] = &ServiceProcess{
			Name:    "service1",
			State:   processTypes.ProcessStateRunning,
			PID:     1000,
			StopCh:  make(chan struct{}),
			Command: &processTesting.MockProcessCmd{
				SignalFunc: func(sig os.Signal) error {
					return nil
				},
				WaitFunc: func() error {
					return nil
				},
			},
			CommandWait: func() error {
				return nil
			},
		}
		
		// Service that fails to stop
		orch.processes["service2"] = &ServiceProcess{
			Name:    "service2",
			State:   processTypes.ProcessStateRunning,
			PID:     1001,
			StopCh:  make(chan struct{}),
			Command: &processTesting.MockProcessCmd{
				SignalFunc: func(sig os.Signal) error {
					return errors.New("signal error")
				},
				WaitFunc: func() error {
					return errors.New("wait error")
				},
			},
			CommandWait: func() error {
				return errors.New("wait error")
			},
		}
		orch.processLock.Unlock()
		
		// Create context with reasonable timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		
		// Shutdown should still complete but with errors
		err := orch.Shutdown(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "errors during shutdown")
		
		// Verify both services are marked as stopped regardless of errors
		orch.processLock.RLock()
		process1 := orch.processes["service1"]
		process2 := orch.processes["service2"]
		orch.processLock.RUnlock()
		
		assert.Equal(t, processTypes.ProcessStateStopped, process1.State)
		assert.Equal(t, processTypes.ProcessStateStopped, process2.State)
	})
	
	// Test case 4: Shutdown with no running services
	t.Run("No running services", func(t *testing.T) {
		// Reset orchestrator state
		orch.isShuttingDown.Store(false)
		
		// Clear all processes
		orch.processLock.Lock()
		orch.processes = make(map[string]*ServiceProcess)
		orch.processLock.Unlock()
		
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		
		// Shutdown should succeed quickly
		err := orch.Shutdown(ctx)
		require.NoError(t, err)
		
		// Verify orchestrator is marked as shutting down
		assert.True(t, orch.isShuttingDown.Load())
	})
}

// TestIsShuttingDown tests the IsShuttingDown method
func TestIsShuttingDown(t *testing.T) {
	// Create test logger
	logger := zaptest.NewLogger(t)
	
	// Create test orchestrator
	orch, err := NewOrchestrator(
		newTestConfigManager(t, newTestConfig(t, t.TempDir())),
		WithLogger(logger),
	)
	require.NoError(t, err)
	
	// Initially not shutting down
	assert.False(t, orch.isShuttingDown.Load())
	
	// Mark as shutting down
	orch.isShuttingDown.Store(true)
	
	// IsShuttingDown should return true
	assert.True(t, orch.isShuttingDown.Load())
}