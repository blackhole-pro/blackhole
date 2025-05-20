// Package process provides the implementation of the process orchestrator
// which manages service processes for the Blackhole platform.
package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/handcraftdev/blackhole/internal/core/config"
	"github.com/handcraftdev/blackhole/internal/core/config/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestOrchestrator_Integration runs integration tests with real processes
func TestOrchestrator_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	// Setup test directory
	tempDir := setupTestDir(t)
	testConfig := createTestConfig(t, tempDir)
	configManager := config.NewConfigManager(zaptest.NewLogger(t))
	require.NoError(t, configManager.SetConfig(testConfig))
	
	// Create test service binaries
	successBin := buildTestBinary(t, tempDir, "success", 0)
	failureBin := buildTestBinary(t, tempDir, "failure", 1)
	
	// Create orchestrator
	orch, err := NewOrchestrator(configManager, WithLogger(zaptest.NewLogger(t)))
	require.NoError(t, err)
	
	// Override auto-restart timeout for faster testing
	oldCalcBackoff := calculateBackoffDelay
	calculateBackoffDelay = func(count int) time.Duration {
		return 100 * time.Millisecond
	}
	defer func() { calculateBackoffDelay = oldCalcBackoff }()
	
	// Test starting a successful service
	t.Run("StartSuccessService", func(t *testing.T) {
		// Add success service to services map
		orch.processLock.Lock()
		orch.services["success"] = &types.ServiceConfig{
			Enabled:    true,
			BinaryPath: successBin,
			DataDir:    filepath.Join(tempDir, "services", "success", "data"),
		}
		orch.processLock.Unlock()
		
		// Create data directory
		err := os.MkdirAll(filepath.Join(tempDir, "services", "success", "data"), 0755)
		require.NoError(t, err)
		
		// Start service
		err = orch.StartService("success")
		require.NoError(t, err)
		
		// Verify service is running
		info, err := orch.GetServiceInfo("success")
		require.NoError(t, err)
		assert.Equal(t, "running", info.State)
		assert.Greater(t, info.PID, 0)
		
		// Let it run for a moment
		time.Sleep(500 * time.Millisecond)
		
		// Stop the service
		err = orch.StopService("success")
		require.NoError(t, err)
		
		// Verify service is stopped
		info, err = orch.GetServiceInfo("success")
		require.NoError(t, err)
		assert.Equal(t, "stopped", info.State)
	})
	
	// Test service that fails and gets restarted
	t.Run("RestartFailingService", func(t *testing.T) {
		// Add failure service to services map
		orch.processLock.Lock()
		orch.services["failure"] = &types.ServiceConfig{
			Enabled:    true,
			BinaryPath: failureBin,
			DataDir:    filepath.Join(tempDir, "services", "failure", "data"),
		}
		orch.config.AutoRestart = true
		orch.processLock.Unlock()
		
		// Create data directory
		err := os.MkdirAll(filepath.Join(tempDir, "services", "failure", "data"), 0755)
		require.NoError(t, err)
		
		// Start service
		err = orch.StartService("failure")
		require.NoError(t, err)
		
		// Wait for it to fail and attempt restart (up to 2 seconds)
		deadline := time.Now().Add(2 * time.Second)
		var info *types.ServiceInfo
		restartOccurred := false
		
		for time.Now().Before(deadline) {
			info, err = orch.GetServiceInfo("failure")
			require.NoError(t, err)
			
			// If we see a restart count > 0, test passed
			if info.Restarts > 0 {
				restartOccurred = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		
		// Verify service was restarted
		assert.True(t, restartOccurred, "Service should have restarted at least once")
		assert.Greater(t, info.Restarts, 0)
		assert.Equal(t, 1, info.LastExitCode)
		
		// Stop the service
		err = orch.StopService("failure")
		require.NoError(t, err)
	})
	
	// Test orchestrator shutdown
	t.Run("OrchestratorShutdown", func(t *testing.T) {
		// Start both services
		err := orch.StartService("success")
		require.NoError(t, err)
		
		err = orch.StartService("failure")
		require.NoError(t, err)
		
		// Verify services are running
		successInfo, err := orch.GetServiceInfo("success")
		require.NoError(t, err)
		assert.Equal(t, "running", successInfo.State)
		
		failureInfo, err := orch.GetServiceInfo("failure")
		require.NoError(t, err)
		assert.Equal(t, "running", failureInfo.State)
		
		// Stop all services
		err = orch.Stop()
		require.NoError(t, err)
		
		// Verify all services are stopped
		services, err := orch.GetAllServices()
		require.NoError(t, err)
		
		for name, info := range services {
			if name == "success" || name == "failure" {
				assert.Equal(t, "stopped", info.State, "Service %s should be stopped", name)
			}
		}
	})
}

// setupTestDir creates a temporary directory structure for tests
func setupTestDir(t *testing.T) string {
	tempDir := t.TempDir()
	
	// Create services directory
	servicesDir := filepath.Join(tempDir, "services")
	err := os.MkdirAll(servicesDir, 0755)
	require.NoError(t, err)
	
	// Create sockets directory
	socketsDir := filepath.Join(tempDir, "sockets")
	err = os.MkdirAll(socketsDir, 0755)
	require.NoError(t, err)
	
	return tempDir
}

// createTestConfig creates a test configuration with the given temporary directory
func createTestConfig(t *testing.T, tempDir string) *types.Config {
	cfg := config.NewDefaultConfig()
	
	// Update paths to use the test directory
	cfg.Orchestrator.ServicesDir = filepath.Join(tempDir, "services")
	cfg.Orchestrator.SocketDir = filepath.Join(tempDir, "sockets")
	cfg.Orchestrator.LogLevel = "debug"
	cfg.Orchestrator.AutoRestart = true
	cfg.Orchestrator.ShutdownTimeout = 5
	
	// Create initial services map
	cfg.Services = make(types.ServicesConfig)
	
	return cfg
}

// buildTestBinary creates a test binary
func buildTestBinary(t *testing.T, tempDir, name string, exitCode int) string {
	// Create service directory
	serviceDir := filepath.Join(tempDir, "services", name)
	require.NoError(t, os.MkdirAll(serviceDir, 0755))
	
	// Create data directory
	dataDir := filepath.Join(serviceDir, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0755))
	
	// Create source file
	srcPath := filepath.Join(t.TempDir(), name+".go")
	src := fmt.Sprintf(`
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("Starting test service: %s")
	fmt.Println("Arguments:", os.Args)
	fmt.Println("Environment:")
	for _, env := range os.Environ() {
		fmt.Println(" ", env)
	}
	
	// Sleep briefly if not failure service
	if %d == 0 {
		time.Sleep(1 * time.Hour) // Long sleep for success service
	} else {
		time.Sleep(100 * time.Millisecond) // Brief sleep then exit
	}
	
	os.Exit(%d)
}
`, name, exitCode, exitCode)
	
	require.NoError(t, os.WriteFile(srcPath, []byte(src), 0644))
	
	// Build binary
	binPath := filepath.Join(serviceDir, name)
	cmd := exec.Command("go", "build", "-o", binPath, srcPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build test binary: %s", output)
	
	return binPath
}