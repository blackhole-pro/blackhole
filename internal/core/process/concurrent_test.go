// Package process provides the implementation of the process orchestrator
// which manages service processes for the Blackhole platform.
package process

import (
	"sync"
	"testing"
	"time"

	processTypes "github.com/handcraftdev/blackhole/internal/core/process/types"
	processTesting "github.com/handcraftdev/blackhole/internal/core/process/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrchestrator_ConcurrentOperations tests the orchestrator under concurrent access
func TestOrchestrator_ConcurrentOperations(t *testing.T) {
	// Create test orchestrator
	orch, mockExec, _, _ := setupTestOrchestrator(t)
	
	// Helper function to set up mock process command
	setupMockCmd := func() *processTesting.MockProcessCmd {
		return &processTesting.MockProcessCmd{
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
	}
	
	// Set up mock executor
	mockExec.CommandFunc = func(path string, args ...string) processTypes.ProcessCmd {
		return setupMockCmd()
	}
	
	// Test case 1: Concurrent starts of multiple services
	t.Run("Concurrent starts", func(t *testing.T) {
		// Create multiple services
		serviceNames := []string{"svc1", "svc2", "svc3", "svc4", "svc5"}
		
		// Add services to configuration
		orch.processLock.Lock()
		for _, name := range serviceNames {
			orch.services[name] = &types.ServiceConfig{
				Enabled: true,
			}
		}
		orch.processLock.Unlock()
		
		// Start all services concurrently
		var wg sync.WaitGroup
		errCh := make(chan error, len(serviceNames))
		
		for _, name := range serviceNames {
			wg.Add(1)
			go func(serviceName string) {
				defer wg.Done()
				if err := orch.Start(serviceName); err != nil {
					errCh <- err
				}
			}(name)
		}
		
		// Wait for all starts to complete
		wg.Wait()
		close(errCh)
		
		// Check for errors
		for err := range errCh {
			t.Errorf("Error starting service: %v", err)
		}
		
		// Verify all services were started
		orch.processLock.RLock()
		for _, name := range serviceNames {
			proc, exists := orch.processes[name]
			if !assert.True(t, exists, "Service %s should exist", name) {
				continue
			}
			assert.Equal(t, processTypes.ProcessStateRunning, proc.State, "Service %s should be running", name)
		}
		orch.processLock.RUnlock()
	})
	
	// Test case 2: Concurrent starts and stops
	t.Run("Concurrent starts and stops", func(t *testing.T) {
		// Reset process map
		orch.processLock.Lock()
		orch.processes = make(map[string]*ServiceProcess)
		orch.processLock.Unlock()
		
		// Create services
		serviceNames := []string{"svc1", "svc2", "svc3", "svc4", "svc5"}
		
		// Add services to configuration
		orch.processLock.Lock()
		for _, name := range serviceNames {
			orch.services[name] = &types.ServiceConfig{
				Enabled: true,
			}
		}
		orch.processLock.Unlock()
		
		// Launch goroutines to start and stop services concurrently
		var wg sync.WaitGroup
		for i, name := range serviceNames {
			wg.Add(1)
			go func(serviceName string, index int) {
				defer wg.Done()
				
				// Start the service
				err := orch.Start(serviceName)
				if err != nil {
					t.Errorf("Error starting service %s: %v", serviceName, err)
					return
				}
				
				// For even indices, also stop the service
				if index%2 == 0 {
					// Small delay to ensure start completes first
					time.Sleep(10 * time.Millisecond)
					
					err := orch.Stop(serviceName)
					if err != nil {
						t.Errorf("Error stopping service %s: %v", serviceName, err)
					}
				}
			}(name, i)
		}
		
		// Wait for all operations to complete
		wg.Wait()
		
		// Verify services with even indices are stopped, odd indices are running
		orch.processLock.RLock()
		for i, name := range serviceNames {
			proc, exists := orch.processes[name]
			if !assert.True(t, exists, "Service %s should exist", name) {
				continue
			}
			
			if i%2 == 0 {
				assert.Equal(t, processTypes.ProcessStateStopped, proc.State, "Service %s should be stopped", name)
			} else {
				assert.Equal(t, processTypes.ProcessStateRunning, proc.State, "Service %s should be running", name)
			}
		}
		orch.processLock.RUnlock()
	})
	
	// Test case 3: Concurrent restarts
	t.Run("Concurrent restarts", func(t *testing.T) {
		// Reset process map
		orch.processLock.Lock()
		orch.processes = make(map[string]*ServiceProcess)
		orch.processLock.Unlock()
		
		// Create and start a service
		serviceName := "restart-test"
		
		// Add service to configuration
		orch.processLock.Lock()
		orch.services[serviceName] = &types.ServiceConfig{
			Enabled: true,
		}
		orch.processLock.Unlock()
		
		// Start the service
		err := orch.Start(serviceName)
		require.NoError(t, err, "Failed to start service")
		
		// Launch concurrent restarts
		var wg sync.WaitGroup
		restartCount := 5
		errCh := make(chan error, restartCount)
		
		for i := 0; i < restartCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := orch.Restart(serviceName); err != nil {
					errCh <- err
				}
			}()
		}
		
		// Wait for all restarts to complete
		wg.Wait()
		close(errCh)
		
		// Check for errors - we expect some errors as restarts will conflict
		// but the service should end up in a valid state
		errCount := 0
		for range errCh {
			errCount++
		}
		
		// Verify service is in running state at the end
		orch.processLock.RLock()
		proc, exists := orch.processes[serviceName]
		orch.processLock.RUnlock()
		
		assert.True(t, exists, "Service should exist")
		assert.Equal(t, processTypes.ProcessStateRunning, proc.State, "Service should be running after restarts")
	})
	
	// Test case 4: Concurrent status and info operations during lifecycle changes
	t.Run("Concurrent status and lifecycle operations", func(t *testing.T) {
		// Reset process map
		orch.processLock.Lock()
		orch.processes = make(map[string]*ServiceProcess)
		orch.processLock.Unlock()
		
		// Create service
		serviceName := "info-test"
		
		// Add service to configuration
		orch.processLock.Lock()
		orch.services[serviceName] = &types.ServiceConfig{
			Enabled: true,
		}
		orch.processLock.Unlock()
		
		// Launch a goroutine that continuously checks status and info
		var wg sync.WaitGroup
		stopCh := make(chan struct{})
		
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for {
				select {
				case <-stopCh:
					return
				default:
					// Get status
					_, _ = orch.Status(serviceName)
					
					// Get service info
					_, _ = orch.GetServiceInfo(serviceName)
					
					// Check if running
					_ = orch.IsRunning(serviceName)
					
					// Small sleep to avoid tight loop
					time.Sleep(1 * time.Millisecond)
				}
			}
		}()
		
		// Perform lifecycle operations
		err := orch.Start(serviceName)
		require.NoError(t, err, "Failed to start service")
		
		time.Sleep(10 * time.Millisecond)
		
		err = orch.Restart(serviceName)
		require.NoError(t, err, "Failed to restart service")
		
		time.Sleep(10 * time.Millisecond)
		
		err = orch.Stop(serviceName)
		require.NoError(t, err, "Failed to stop service")
		
		// Stop the status checking goroutine
		close(stopCh)
		wg.Wait()
		
		// Verify service ends up stopped
		orch.processLock.RLock()
		proc, exists := orch.processes[serviceName]
		orch.processLock.RUnlock()
		
		assert.True(t, exists, "Service should exist")
		assert.Equal(t, processTypes.ProcessStateStopped, proc.State, "Service should be stopped")
	})
}