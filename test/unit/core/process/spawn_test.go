// Package process contains tests for the process orchestrator
// which manages service processes for the Blackhole platform.
package process_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/handcraftdev/blackhole/internal/core/config"
	configTypes "github.com/handcraftdev/blackhole/internal/core/config/types"
	"github.com/handcraftdev/blackhole/internal/core/process"
	testingMocks "github.com/handcraftdev/blackhole/internal/core/process/testing"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestSpawnServiceEdgeCases tests edge cases in the SpawnService method
func TestSpawnServiceEdgeCases(t *testing.T) {
	// These tests should exercise the SpawnService method
	// We're just making sure the test file structure is correct
	// Detailed test implementation can be added later
	t.Skip("Test implementation pending detailed work on public API testing approach")
}

// TestStopServiceEdgeCases tests edge cases in the StopService method
func TestStopServiceEdgeCases(t *testing.T) {
	// These tests should exercise the StopService method
	// We're just making sure the test file structure is correct
	// Detailed test implementation can be added later
	t.Skip("Test implementation pending detailed work on public API testing approach")
}

// Helper functions for test setup - copied from orchestrator_test.go
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
func setupTestOrchestrator(t *testing.T) (*process.Orchestrator, *testingMocks.MockProcessExecutor, *config.ConfigManager, string) {
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
	orchestrator, err := process.NewOrchestrator(
		configManager,
		process.WithExecutor(mockExecutor),
		process.WithLogger(zaptest.NewLogger(t)),
	)
	require.NoError(t, err)
	
	return orchestrator, mockExecutor, configManager, tempDir
}