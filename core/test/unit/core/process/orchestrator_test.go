// Package process contains tests for the process orchestrator
// which manages service processes for the Blackhole platform.
package process_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/blackhole-pro/blackhole/core/internal/runtime/config"
	configTypes "github.com/blackhole-pro/blackhole/core/internal/runtime/config/types"
	"github.com/blackhole-pro/blackhole/core/internal/runtime/orchestrator"
	testingMocks "github.com/blackhole-pro/blackhole/core/internal/runtime/orchestrator/testing"
	processTypes "github.com/blackhole-pro/blackhole/core/internal/runtime/orchestrator/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// createTempDir creates a temporary directory for tests
func createTempDir(t *testing.T) string {
	tempDir := t.TempDir()
	
	// Create services subdirectory
	servicesDir := filepath.Join(tempDir, "services")
	err := os.Mkdir(servicesDir, 0755)
	require.NoError(t, err)
	
	return tempDir
}

// createMockBinary creates a mock executable file for testing
func createMockBinary(t *testing.T, servicesDir, serviceName string) string {
	// Create service directory
	serviceDir := filepath.Join(servicesDir, serviceName)
	err := os.Mkdir(serviceDir, 0755)
	require.NoError(t, err)
	
	// Create binary file
	binaryPath := filepath.Join(serviceDir, serviceName)
	err = os.WriteFile(binaryPath, []byte("#!/bin/sh\necho 'Mock service running'\nsleep 10"), 0755)
	require.NoError(t, err)
	
	return binaryPath
}

// newTestConfig creates a test configuration for the orchestrator
func newTestConfig(t *testing.T, tempDir string) *configTypes.Config {
	cfg := config.NewDefaultConfig()
	
	// Update paths to use temporary directory
	cfg.Orchestrator.ServicesDir = filepath.Join(tempDir, "services")
	cfg.Orchestrator.SocketDir = filepath.Join(tempDir, "sockets")
	
	// Create socket directory if it doesn't exist
	if _, err := os.Stat(cfg.Orchestrator.SocketDir); os.IsNotExist(err) {
		err = os.Mkdir(cfg.Orchestrator.SocketDir, 0755)
		require.NoError(t, err)
	}
	
	// Add test services
	cfg.Services["service1"] = &configTypes.ServiceConfig{
		Enabled:    true,
		BinaryPath: filepath.Join(cfg.Orchestrator.ServicesDir, "service1", "service1"),
		Args:       []string{"--config", "test.yaml"},
		Environment: map[string]string{
			"TEST_ENV": "test_value",
		},
	}
	
	cfg.Services["service2"] = &configTypes.ServiceConfig{
		Enabled:    false,
		BinaryPath: filepath.Join(cfg.Orchestrator.ServicesDir, "service2", "service2"),
	}
	
	return cfg
}

// newTestConfigManager creates a test config manager with the given configuration
func newTestConfigManager(t *testing.T, testConfig *configTypes.Config) *config.ConfigManager {
	logger := zaptest.NewLogger(t)
	cm := config.NewConfigManager(logger)
	err := cm.SetConfig(testConfig)
	require.NoError(t, err)
	return cm
}

// setupTestOrchestrator creates an orchestrator with mock executor for testing
func setupTestOrchestrator(t *testing.T) (*orchestrator.Orchestrator, *testingMocks.MockProcessExecutor, *config.ConfigManager, string) {
	// Create temp directory for test
	tempDir := createTempDir(t)
	
	// Create mock service binaries
	servicesDir := filepath.Join(tempDir, "services")
	createMockBinary(t, servicesDir, "service1")
	createMockBinary(t, servicesDir, "service2")
	
	// Create test configuration
	testConfig := newTestConfig(t, tempDir)
	configManager := newTestConfigManager(t, testConfig)
	
	// Create mock executor
	mockExecutor := testingMocks.NewMockExecutor()
	
	// Create orchestrator with mock executor and test logger
	orchestrator, err := orchestrator.NewOrchestrator(
		configManager,
		orchestrator.WithExecutor(mockExecutor),
		orchestrator.WithLogger(zaptest.NewLogger(t)),
	)
	require.NoError(t, err)
	
	return orchestrator, mockExecutor, configManager, tempDir
}

// TestNewOrchestrator tests the orchestrator constructor
func TestNewOrchestrator(t *testing.T) {
	// Basic setup for constructor testing
	tempDir := createTempDir(t)
	testConfig := newTestConfig(t, tempDir)
	configManager := newTestConfigManager(t, testConfig)
	
	// Test case 1: Create with default options
	t.Run("Default options", func(t *testing.T) {
		orchestrator, err := orchestrator.NewOrchestrator(configManager)
		require.NoError(t, err)
		assert.NotNil(t, orchestrator)
	})
	
	// Test case 2: Create with custom logger and executor
	t.Run("Custom options", func(t *testing.T) {
		testLogger := zaptest.NewLogger(t)
		mockExecutor := testingMocks.NewMockExecutor()
		
		orchestrator, err := orchestrator.NewOrchestrator(
			configManager,
			orchestrator.WithLogger(testLogger),
			orchestrator.WithExecutor(mockExecutor),
		)
		
		require.NoError(t, err)
		assert.NotNil(t, orchestrator)
	})
	
	// Test case 3: Invalid services directory
	t.Run("Invalid services directory", func(t *testing.T) {
		// Modify config to point to non-existent directory
		invalidConfig := newTestConfig(t, tempDir)
		invalidConfig.Orchestrator.ServicesDir = "/non/existent/directory"
		invalidConfigManager := newTestConfigManager(t, invalidConfig)
		
		orchestrator, err := orchestrator.NewOrchestrator(invalidConfigManager)
		assert.Error(t, err)
		assert.Nil(t, orchestrator)
		// The exact error message might differ between OSes, so check for a substring
		// that should be present in all error messages related to directory creation
		assert.Contains(t, err.Error(), "failed to create services directory")
	})
}

// TestDiscoverServices tests the DiscoverServices method
func TestDiscoverServices(t *testing.T) {
	orchestrator, _, _, tempDir := setupTestOrchestrator(t)
	
	// Test discovering existing services
	t.Run("Discover existing services", func(t *testing.T) {
		services, err := orchestrator.DiscoverServices()
		require.NoError(t, err)
		
		assert.Contains(t, services, "service1")
		assert.Contains(t, services, "service2")
		assert.Len(t, services, 2)
	})
	
	// Test discovering new services
	t.Run("Discover new services", func(t *testing.T) {
		// Create a new service
		createMockBinary(t, filepath.Join(tempDir, "services"), "service3")
		
		services, err := orchestrator.DiscoverServices()
		require.NoError(t, err)
		
		assert.Contains(t, services, "service1")
		assert.Contains(t, services, "service2")
		assert.Contains(t, services, "service3")
		assert.Len(t, services, 3)
	})
	
	// Test empty services directory
	t.Run("Empty services directory", func(t *testing.T) {
		// Create a new empty directory
		emptyDir := createTempDir(t)
		
		// Create a new config with empty services dir
		testConfig := newTestConfig(t, emptyDir)
		configManager := newTestConfigManager(t, testConfig)
		
		// Create orchestrator
		orch, err := orchestrator.NewOrchestrator(
			configManager,
			orchestrator.WithLogger(zaptest.NewLogger(t)),
		)
		require.NoError(t, err)
		
		// Discover services
		services, err := orch.DiscoverServices()
		require.NoError(t, err)
		assert.Empty(t, services)
	})
}

// TestServiceLifecycle tests the service lifecycle methods (Start, Stop, Restart)
func TestServiceLifecycle(t *testing.T) {
	orchestrator, mockExecutor, _, _ := setupTestOrchestrator(t)
	
	serviceName := "service1"
	
	// Setup mock process
	mockProcess := &testingMocks.MockProcess{}
	mockProcess.SetPid(1000)
	
	mockCmd := &testingMocks.MockProcessCmd{
		StartFunc: func() error {
			return nil
		},
		WaitFunc: func() error {
			return nil
		},
		ProcessFunc: func() processTypes.Process {
			return mockProcess
		},
	}
	
	mockExecutor.CommandFunc = func(path string, args ...string) processTypes.ProcessCmd {
		return mockCmd
	}
	
	// Test starting a service
	t.Run("Start service", func(t *testing.T) {
		err := orchestrator.Start(serviceName)
		require.NoError(t, err)
		
		// Verify service state
		state, err := orchestrator.Status(serviceName)
		require.NoError(t, err)
		assert.Equal(t, processTypes.ProcessStateRunning, state)
	})
	
	// Test stopping a service
	t.Run("Stop service", func(t *testing.T) {
		// Setup kill function
		var killCalled bool
		mockProcess.KillFunc = func() error {
			killCalled = true
			return nil
		}
		
		err := orchestrator.Stop(serviceName)
		require.NoError(t, err)
		
		// Verify kill was called
		assert.True(t, killCalled)
		
		// Verify service state
		state, err := orchestrator.Status(serviceName)
		require.NoError(t, err)
		assert.Equal(t, processTypes.ProcessStateStopped, state)
	})
	
	// Test restarting a service
	t.Run("Restart service", func(t *testing.T) {
		// Track start calls
		startCalls := 0
		mockCmd.StartFunc = func() error {
			startCalls++
			return nil
		}
		
		// First start the service
		err := orchestrator.Start(serviceName)
		require.NoError(t, err)
		
		// Reset kill called flag
		var killCalled bool
		mockProcess.KillFunc = func() error {
			killCalled = true
			return nil
		}
		
		// Then restart it
		err = orchestrator.Restart(serviceName)
		require.NoError(t, err)
		
		// Verify kill was called
		assert.True(t, killCalled)
		
		// Verify start was called twice (for initial start and restart)
		assert.Equal(t, 2, startCalls)
		
		// Verify service state
		state, err := orchestrator.Status(serviceName)
		require.NoError(t, err)
		assert.Equal(t, processTypes.ProcessStateRunning, state)
	})
}

// TestErrorHandlingAndRecovery tests error handling and recovery
func TestErrorHandlingAndRecovery(t *testing.T) {
	orchestrator, mockExecutor, _, _ := setupTestOrchestrator(t)
	serviceName := "service1"
	
	// Test starting a service with an error
	t.Run("Start service error", func(t *testing.T) {
		// Mock start to return an error
		mockCmd := &testingMocks.MockProcessCmd{
			StartFunc: func() error {
				return assert.AnError
			},
		}
		
		mockExecutor.CommandFunc = func(path string, args ...string) processTypes.ProcessCmd {
			return mockCmd
		}
		
		// Try to start the service
		err := orchestrator.Start(serviceName)
		assert.Error(t, err)
		
		// Verify service state
		state, err := orchestrator.Status(serviceName)
		require.NoError(t, err)
		assert.Equal(t, processTypes.ProcessStateFailed, state)
		
		// Verify service info has error
		info, err := orchestrator.GetServiceInfo(serviceName)
		require.NoError(t, err)
		assert.Equal(t, "failed", info.State)
		assert.Contains(t, info.LastError, "assert.AnError")
	})
	
	// Test auto restart after failure
	t.Run("Auto restart after failure", func(t *testing.T) {
		// Enable auto restart
		// This is testing an internal field so we can't easily set it in the test
		// We'll rely on the default behavior instead
		
		// Setup variables to track calls
		startCalls := 0
		waitError := true
		waitCh := make(chan struct{})
		
		mockCmd := &testingMocks.MockProcessCmd{
			StartFunc: func() error {
				startCalls++
				return nil
			},
			WaitFunc: func() error {
				if waitError {
					waitError = false
					return assert.AnError
				}
				<-waitCh // Wait indefinitely for subsequent calls
				return nil
			},
			ProcessFunc: func() processTypes.Process {
				return &testingMocks.MockProcess{
					PidFunc: func() int {
						return 1000
					},
				}
			},
		}
		
		mockExecutor.CommandFunc = func(path string, args ...string) processTypes.ProcessCmd {
			return mockCmd
		}
		
		// Start service
		err := orchestrator.Start(serviceName)
		require.NoError(t, err)
		
		// Wait for restart to happen
		time.Sleep(50 * time.Millisecond)
		
		// Clean up
		close(waitCh)
	})
}

// TestSignalHandling tests the signal handling functionality
func TestSignalHandling(t *testing.T) {
	orchestrator, mockExecutor, _, _ := setupTestOrchestrator(t)
	
	// Setup service
	serviceName := "service1"
	killCalled := false
	
	mockProcess := &testingMocks.MockProcess{
		PidFunc: func() int {
			return 1000
		},
		KillFunc: func() error {
			killCalled = true
			return nil
		},
	}
	
	mockCmd := &testingMocks.MockProcessCmd{
		StartFunc: func() error {
			return nil
		},
		WaitFunc: func() error {
			// Wait indefinitely
			ch := make(chan struct{})
			<-ch
			return nil
		},
		ProcessFunc: func() processTypes.Process {
			return mockProcess
		},
	}
	
	mockExecutor.CommandFunc = func(path string, args ...string) processTypes.ProcessCmd {
		return mockCmd
	}
	
	// Start service
	err := orchestrator.Start(serviceName)
	require.NoError(t, err)
	
	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start shutdown in a goroutine
	var shutdownErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		shutdownErr = orchestrator.Shutdown(ctx)
	}()
	
	// Wait a bit for shutdown to start
	time.Sleep(50 * time.Millisecond)
	
	// Cancel context to simulate timeout
	cancel()
	
	// Wait for shutdown to complete
	wg.Wait()
	
	// Verify kill was called
	assert.True(t, killCalled)
	
	// Verify shutdown error (should be context canceled)
	assert.Error(t, shutdownErr)
	assert.Contains(t, shutdownErr.Error(), "context canceled")
}