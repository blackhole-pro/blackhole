package app_test

import (
	"errors"
	"testing"

	"github.com/handcraftdev/blackhole/internal/core/app"
	apptesting "github.com/handcraftdev/blackhole/internal/core/app/testing"
	"github.com/handcraftdev/blackhole/internal/core/app/types"
	"github.com/handcraftdev/blackhole/internal/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestNewApplication tests the creation of a new application
func TestNewApplication(t *testing.T) {
	// Test with default options
	t.Run("Default options", func(t *testing.T) {
		application, err := app.NewApplication()
		require.NoError(t, err)
		assert.NotNil(t, application)
	})

	// Test with custom logger
	t.Run("Custom logger", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		application, err := app.NewApplication(app.WithLogger(logger))
		require.NoError(t, err)
		assert.NotNil(t, application)
	})

	// Test with default options only
	t.Run("Default options with logger", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		application, err := app.NewApplication(app.WithLogger(logger))
		require.NoError(t, err)
		assert.NotNil(t, application)
	})
}

// TestRegisterService tests registering services with the application
func TestRegisterService(t *testing.T) {
	// Create test application
	logger := zaptest.NewLogger(t)
	application, err := app.NewApplication(app.WithLogger(logger))
	require.NoError(t, err)

	// Test registering a valid service
	t.Run("Valid service", func(t *testing.T) {
		mockService := apptesting.NewMockService("test-service")
		err := application.RegisterService(mockService)
		assert.NoError(t, err)

		// Verify service was registered
		service, exists := application.GetService("test-service")
		assert.True(t, exists)
		assert.Equal(t, mockService, service)
	})

	// Test registering a nil service
	t.Run("Nil service", func(t *testing.T) {
		err := application.RegisterService(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot register nil service")
	})

	// Test registering a service with empty name
	t.Run("Empty service name", func(t *testing.T) {
		mockService := apptesting.NewMockService("")
		mockService.NameFunc = func() string { return "" }
		err := application.RegisterService(mockService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service name cannot be empty")
	})

	// Test registering a service that's already registered
	t.Run("Already registered service", func(t *testing.T) {
		mockService := apptesting.NewMockService("test-service")
		err := application.RegisterService(mockService)
		assert.Error(t, err)
		assert.Equal(t, types.ErrServiceAlreadyRegistered, err)
	})
}

// TestStartStop tests starting and stopping the application
func TestStartStop(t *testing.T) {
	// Create test application with mock dependencies
	logger := zaptest.NewLogger(t)
	realConfigManager := config.NewConfigManager(logger)  // Use real config manager
	mockProcessManager := apptesting.NewMockProcessManager()
	mockFactory := apptesting.NewMockProcessManagerFactory()
	
	// Configure mock factory to return our mock process manager
	mockFactory.CreateProcessManagerFunc = func(cm types.ConfigManager, l *zap.Logger) (types.ProcessManager, error) {
		return mockProcessManager, nil
	}

	application, err := app.NewApplication(
		app.WithLogger(logger),
		app.WithConfigManager(realConfigManager),  // Pass real config manager
		app.WithProcessManagerFactory(mockFactory),
	)
	require.NoError(t, err)

	// Test starting the application
	t.Run("Start application", func(t *testing.T) {
		// Configure mock process manager methods to succeed
		mockProcessManager.StartFunc = func() error {
			return nil
		}
		
		mockProcessManager.StartAllFunc = func() error {
			return nil
		}
		
		// Start the application
		err := application.Start()
		require.NoError(t, err)
		
		// Verify our factory was called
		assert.Equal(t, 1, mockFactory.CreateCalled)
		
		// Since we can't access the unexported fields directly in the new test structure,
		// we can only verify the test passes without errors
	})
	
	// Test application startup errors - split into subtests to avoid interference
	t.Run("Start application errors", func(t *testing.T) {
		// Factory creation error test
		t.Run("Factory creation error", func(t *testing.T) {
			// Create a fresh application for this test
			logger := zaptest.NewLogger(t)
			testConfigManager := config.NewConfigManager(logger)
			errFactory := apptesting.NewMockProcessManagerFactory()
			
			// Configure factory to return an error
			factoryErr := errors.New("factory creation failed")
			errFactory.CreateProcessManagerFunc = func(cm types.ConfigManager, l *zap.Logger) (types.ProcessManager, error) {
				return nil, factoryErr
			}
			
			testApp, err := app.NewApplication(
				app.WithLogger(logger),
				app.WithConfigManager(testConfigManager),
				app.WithProcessManagerFactory(errFactory),
			)
			require.NoError(t, err)
			
			// Start should fail with factory error
			err = testApp.Start()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create process manager")
		})
		
		// Process manager start error test
		t.Run("Process manager start error", func(t *testing.T) {
			// Create a fresh application for this test
			logger := zaptest.NewLogger(t)
			testConfigManager := config.NewConfigManager(logger)
			testFactory := apptesting.NewMockProcessManagerFactory()
			
			// Configure a mock process manager that returns an error on start
			mockPM := apptesting.NewMockProcessManager()
			startErr := errors.New("process manager start failed")
			mockPM.StartFunc = func() error {
				return startErr
			}
			
			// Configure factory to return our process manager
			testFactory.CreateProcessManagerFunc = func(cm types.ConfigManager, l *zap.Logger) (types.ProcessManager, error) {
				return mockPM, nil
			}
			
			testApp, err := app.NewApplication(
				app.WithLogger(logger),
				app.WithConfigManager(testConfigManager),
				app.WithProcessManagerFactory(testFactory),
			)
			require.NoError(t, err)
			
			// Start should fail with process manager start error
			err = testApp.Start()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to start process manager")
		})
	})

	// Test stopping the application
	t.Run("Stop application", func(t *testing.T) {
		// Mock the Stop method to succeed
		mockService := apptesting.NewMockService("test-service")
		application.RegisterService(mockService)
		
		// Configure mock to succeed
		mockProcessManager.StopFunc = func() error {
			return nil
		}
		
		// Stop the application
		err := application.Stop()
		assert.NoError(t, err)
		
		// Since we can't access the unexported fields directly in the new test structure,
		// we can only verify the test passes without errors
	})
}

// TestServiceOperations tests service-related operations
func TestServiceOperations(t *testing.T) {
	// Create test application
	logger := zaptest.NewLogger(t)
	application, err := app.NewApplication(app.WithLogger(logger))
	require.NoError(t, err)

	// Register some test services
	service1 := apptesting.NewMockService("service1")
	service2 := apptesting.NewMockService("service2")
	application.RegisterService(service1)
	application.RegisterService(service2)

	// Test GetService for existing service
	t.Run("Get existing service", func(t *testing.T) {
		service, exists := application.GetService("service1")
		assert.True(t, exists)
		assert.Equal(t, service1, service)
	})

	// Test GetService for non-existent service
	t.Run("Get non-existent service", func(t *testing.T) {
		service, exists := application.GetService("non-existent")
		assert.False(t, exists)
		assert.Nil(t, service)
	})
}