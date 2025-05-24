package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/blackhole-pro/blackhole/core/internal/core/app/types"
	"github.com/blackhole-pro/blackhole/core/internal/runtime/config"
	configTypes "github.com/blackhole-pro/blackhole/core/internal/runtime/config/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// Simplified Application interface for testing
type TestApplication interface {
	// Methods needed by our tests
	Initialize() error
	DiscoverServices() ([]string, error)
	StartService(name string) error
	StopService(name string) error
	GetServiceInfo(name string) (*types.ServiceInfo, error)
	GetAllServices() (map[string]*types.ServiceInfo, error)
	Stop() error
}

// Helper function to create a test application
func createTestApp(t *testing.T, configManager *config.ConfigManager) TestApplication {
	// In a real implementation, you would use the app factory
	// For this test, we'll create a mock application
	return &testAppImpl{
		t:             t,
		configManager: configManager,
		services:      make(map[string]*types.ServiceInfo),
	}
}

// Mock implementation for testing
type testAppImpl struct {
	t             *testing.T
	configManager *config.ConfigManager
	services      map[string]*types.ServiceInfo
}

func (a *testAppImpl) Initialize() error {
	return nil
}

func (a *testAppImpl) DiscoverServices() ([]string, error) {
	cfg := a.configManager.GetConfig()
	servicesDir := cfg.Orchestrator.ServicesDir
	
	entries, err := os.ReadDir(servicesDir)
	if err != nil {
		return nil, err
	}
	
	var services []string
	for _, entry := range entries {
		if entry.IsDir() {
			serviceName := entry.Name()
			services = append(services, serviceName)
			
			// Add to services map if not exists
			if _, exists := a.services[serviceName]; !exists {
				a.services[serviceName] = &types.ServiceInfo{
					Name: serviceName,
					Enabled: true,
				}
			}
		}
	}
	
	return services, nil
}

func (a *testAppImpl) StartService(name string) error {
	cfg := a.configManager.GetConfig()
	serviceConfig, exists := cfg.Services[name]
	if !exists {
		return types.ErrServiceNotFound
	}
	
	// Start the service as a subprocess
	cmd := exec.Command(serviceConfig.BinaryPath)
	cmd.Env = append(os.Environ(), "BLACKHOLE_SERVICE_DATA_DIR="+serviceConfig.DataDir)
	err := cmd.Start()
	if err != nil {
		return err
	}
	
	// Update service info
	a.services[name] = &types.ServiceInfo{
		Name: name,
		Enabled: true,
		Running: true,
		PID: cmd.Process.Pid,
	}
	
	return nil
}

func (a *testAppImpl) StopService(name string) error {
	info, exists := a.services[name]
	if !exists || !info.Running {
		return nil // already stopped
	}
	
	// Find process by PID
	process, err := os.FindProcess(info.PID)
	if err != nil {
		return err
	}
	
	// Send SIGTERM for graceful shutdown
	if err := process.Signal(os.Interrupt); err != nil {
		return err
	}
	
	// Update service info
	info.Running = false
	
	return nil
}

func (a *testAppImpl) GetServiceInfo(name string) (*types.ServiceInfo, error) {
	info, exists := a.services[name]
	if !exists {
		return nil, types.ErrServiceNotFound
	}
	return info, nil
}

func (a *testAppImpl) GetAllServices() (map[string]*types.ServiceInfo, error) {
	return a.services, nil
}

func (a *testAppImpl) Stop() error {
	for name := range a.services {
		_ = a.StopService(name)
	}
	return nil
}

// setupTestDir creates a temporary directory structure for tests
func setupTestDir(t *testing.T) string {
	tempDir := t.TempDir()

	// Create services directory
	servicesDir := filepath.Join(tempDir, "services")
	err := os.MkdirAll(servicesDir, 0755)
	require.NoError(t, err)

	// Create test-service directory
	testServiceDir := filepath.Join(servicesDir, "test-service")
	err = os.MkdirAll(testServiceDir, 0755)
	require.NoError(t, err)

	// Create data directory
	dataDir := filepath.Join(testServiceDir, "data")
	err = os.MkdirAll(dataDir, 0755)
	require.NoError(t, err)

	return tempDir
}

// createTestConfig creates a test configuration with the given temporary directory
func createTestConfig(t *testing.T, tempDir, testServiceBin string) *configTypes.Config {
	cfg := config.NewDefaultConfig()

	// Update paths to use the test directory
	cfg.Orchestrator.ServicesDir = filepath.Join(tempDir, "services")
	cfg.Orchestrator.LogLevel = "debug"
	cfg.Orchestrator.AutoRestart = true
	cfg.Orchestrator.ShutdownTimeout = 5

	// Create initial services map
	cfg.Services = make(configTypes.ServicesConfig)
	cfg.Services["test-service"] = &configTypes.ServiceConfig{
		Enabled:    true,
		BinaryPath: testServiceBin,
		DataDir:    filepath.Join(tempDir, "services", "test-service", "data"),
	}

	return cfg
}

// buildTestService builds the test service binary
func buildTestService(t *testing.T, tempDir string) string {
	// Get the path to the test service source
	srcPath, err := filepath.Abs("test-service/main.go")
	require.NoError(t, err)
	
	// Create destination directory
	testServiceDir := filepath.Join(tempDir, "services", "test-service")
	binPath := filepath.Join(testServiceDir, "test-service")
	
	// Build the test service
	cmd := exec.Command("go", "build", "-o", binPath, srcPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build test service: %s", output)
	
	return binPath
}

// copyFile copies a file from src to dst
func copyFile(t *testing.T, src, dst string) {
	input, err := os.ReadFile(src)
	require.NoError(t, err)
	
	err = os.WriteFile(dst, input, 0755)
	require.NoError(t, err)
}

func TestAppAdapter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Setup test directory
	tempDir := setupTestDir(t)
	testServiceBin := buildTestService(t, tempDir)
	testConfig := createTestConfig(t, tempDir, testServiceBin)

	// Create the app
	logger := zaptest.NewLogger(t)
	configManager := config.NewConfigManager(logger)
	require.NoError(t, configManager.SetConfig(testConfig))

	application := createTestApp(t, configManager)
	require.NotNil(t, application)

	// Test app initialization, service discovery and lifecycle management
	t.Run("AppLifecycle", func(t *testing.T) {
		// Initialize the app
		err := application.Initialize()
		require.NoError(t, err)

		// Discover services
		services, err := application.DiscoverServices()
		require.NoError(t, err)
		assert.Contains(t, services, "test-service")

		// Start the service
		err = application.StartService("test-service")
		require.NoError(t, err)

		// Give the service a moment to start
		time.Sleep(500 * time.Millisecond)

		// Get service info
		info, err := application.GetServiceInfo("test-service")
		require.NoError(t, err)
		assert.True(t, info.Running)
		assert.Greater(t, info.PID, 0)

		// Check status file was created
		statusFilePath := filepath.Join(tempDir, "services", "test-service", "data", "status.txt")
		_, err = os.Stat(statusFilePath)
		assert.NoError(t, err, "Status file should be created by running service")

		// Stop the service
		err = application.StopService("test-service")
		require.NoError(t, err)

		// Check shutdown file was created
		shutdownFilePath := filepath.Join(tempDir, "services", "test-service", "data", "shutdown.txt")
		time.Sleep(500 * time.Millisecond) // Give the service time to write shutdown file
		_, err = os.Stat(shutdownFilePath)
		assert.NoError(t, err, "Shutdown file should be created during graceful shutdown")

		// Verify service is stopped
		info, err = application.GetServiceInfo("test-service")
		require.NoError(t, err)
		assert.False(t, info.Running)
	})

	// Test service discovery
	t.Run("ServiceDiscovery", func(t *testing.T) {
		// Create another test service directory
		newServiceDir := filepath.Join(tempDir, "services", "another-service")
		require.NoError(t, os.MkdirAll(newServiceDir, 0755))

		// Copy the test service binary
		newServiceBin := filepath.Join(newServiceDir, "another-service")
		copyFile(t, testServiceBin, newServiceBin)

		// Create data directory
		dataDir := filepath.Join(newServiceDir, "data")
		require.NoError(t, os.MkdirAll(dataDir, 0755))

		// Add to config
		cfg := configManager.GetConfig()
		cfg.Services["another-service"] = &configTypes.ServiceConfig{
			Enabled:    true,
			BinaryPath: newServiceBin,
			DataDir:    dataDir,
		}

		// Discover services again
		services, err := application.DiscoverServices()
		require.NoError(t, err)
		assert.Contains(t, services, "test-service")
		assert.Contains(t, services, "another-service")

		// Start the new service
		err = application.StartService("another-service")
		require.NoError(t, err)

		// Get all services
		allServices, err := application.GetAllServices()
		require.NoError(t, err)
		assert.Len(t, allServices, 2)
		assert.Contains(t, allServices, "test-service")
		assert.Contains(t, allServices, "another-service")
		assert.True(t, allServices["another-service"].Running)

		// Stop all services
		err = application.Stop()
		require.NoError(t, err)

		// Verify all services are stopped
		allServices, err = application.GetAllServices()
		require.NoError(t, err)
		for _, info := range allServices {
			assert.False(t, info.Running)
		}
	})
}