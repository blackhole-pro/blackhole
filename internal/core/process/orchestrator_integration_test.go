package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"blackhole/internal/core/config"
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
	config := createTestConfig(t, tempDir)
	
	// Create test logger
	logger := zaptest.NewLogger(t)
	
	// Set services directory in config
	config.Storage.DataDir = tempDir
	
	// Create test service binaries
	successBin := buildTestBinary(t, tempDir, "success", 0)
	failureBin := buildTestBinary(t, tempDir, "failure", 1)
	
	// Configure services
	config.Services.Identity.Enabled = true
	config.Services.Storage.Enabled = true
	
	// Create orchestrator
	orch, err := NewOrchestrator(config, WithLogger(logger))
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
		orch.services["success"] = &ServiceConfig{
			Enabled:    true,
			BinaryPath: successBin,
			DataDir:    filepath.Join(tempDir, "services", "success", "data"),
		}
		
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
		orch.services["failure"] = &ServiceConfig{
			Enabled:    true,
			BinaryPath: failureBin,
			DataDir:    filepath.Join(tempDir, "services", "failure", "data"),
		}
		
		// Create data directory
		err := os.MkdirAll(filepath.Join(tempDir, "services", "failure", "data"), 0755)
		require.NoError(t, err)
		
		// Enable auto-restart
		orch.config.AutoRestart = true
		
		// Start service
		err = orch.StartService("failure")
		require.NoError(t, err)
		
		// Wait for it to fail and attempt restart (up to 2 seconds)
		deadline := time.Now().Add(2 * time.Second)
		var info *ServiceInfo
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