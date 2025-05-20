package app

import (
	"errors"
	"testing"

	"github.com/handcraftdev/blackhole/internal/core/app/testing"
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
		app, err := NewApplication()
		require.NoError(t, err)
		assert.NotNil(t, app)
		assert.NotNil(t, app.logger)
		assert.NotNil(t, app.configManager)
		assert.NotNil(t, app.services)
		assert.NotNil(t, app.doneCh)
		assert.False(t, app.isActive)
	})

	// Test with custom logger
	t.Run("Custom logger", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		app, err := NewApplication(WithLogger(logger))
		require.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, logger, app.logger)
	})

	// Test with custom config manager
	t.Run("Custom config manager", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		configManager := config.NewConfigManager(logger)
		app, err := NewApplication(WithLogger(logger), WithConfigManager(configManager))
		require.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, configManager, app.configManager)
	})
}

// TestRegisterService tests registering services with the application
func TestRegisterService(t *testing.T) {
	// Create test application
	logger := zaptest.NewLogger(t)
	app, err := NewApplication(WithLogger(logger))
	require.NoError(t, err)

	// Test registering a valid service
	t.Run("Valid service", func(t *testing.T) {
		mockService := testing.NewMockService("test-service")
		err := app.RegisterService(mockService)
		assert.NoError(t, err)

		// Verify service was registered
		service, exists := app.GetService("test-service")
		assert.True(t, exists)
		assert.Equal(t, mockService, service)
	})

	// Test registering a nil service
	t.Run("Nil service", func(t *testing.T) {
		err := app.RegisterService(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot register nil service")
	})

	// Test registering a service with empty name
	t.Run("Empty service name", func(t *testing.T) {
		mockService := testing.NewMockService("")
		mockService.NameFunc = func() string { return "" }
		err := app.RegisterService(mockService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service name cannot be empty")
	})

	// Test registering a service that's already registered
	t.Run("Already registered service", func(t *testing.T) {
		mockService := testing.NewMockService("test-service")
		err := app.RegisterService(mockService)
		assert.Error(t, err)
		assert.Equal(t, types.ErrServiceAlreadyRegistered, err)
	})
}

// TestStartStop tests starting and stopping the application
func TestStartStop(t *testing.T) {
	// Create test application with mock dependencies
	logger := zaptest.NewLogger(t)
	mockConfigManager := testing.NewMockConfigManager()
	mockProcessManager := testing.NewMockProcessManager()
	mockFactory := testing.NewMockProcessManagerFactory()
	
	// Configure mock factory to return our mock process manager
	mockFactory.CreateProcessManagerFunc = func(cm types.ConfigManager, l *zap.Logger) (types.ProcessManager, error) {
		return mockProcessManager, nil
	}

	app, err := NewApplication(
		WithLogger(logger),
		WithConfigManager(mockConfigManager),
		WithProcessManagerFactory(mockFactory),
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
		err := app.Start()
		require.NoError(t, err)
		
		// Verify our factory was called
		assert.Equal(t, 1, mockFactory.CreateCalled)
		
		// Verify our process manager methods were called
		assert.Equal(t, 1, mockProcessManager.startCalled)
		assert.Equal(t, 1, mockProcessManager.startAllCalled)
		
		// Verify application state
		assert.True(t, app.isActive)
	})
	
	// Test application startup errors
	t.Run("Start application errors", func(t *testing.T) {
		// Create a fresh application for this test
		logger := zaptest.NewLogger(t)
		mockConfigManager := testing.NewMockConfigManager()
		errFactory := testing.NewMockProcessManagerFactory()
		
		// First test: Factory creation error
		factoryErr := errors.New("factory creation failed")
		errFactory.CreateProcessManagerFunc = func(cm types.ConfigManager, l *zap.Logger) (types.ProcessManager, error) {
			return nil, factoryErr
		}
		
		app, err := NewApplication(
			WithLogger(logger),
			WithConfigManager(mockConfigManager),
			WithProcessManagerFactory(errFactory),
		)
		require.NoError(t, err)
		
		// Start should fail with factory error
		err = app.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create process manager")
		
		// Second test: Process manager start error
		mockPM := testing.NewMockProcessManager()
		startErr := errors.New("process manager start failed")
		mockPM.StartFunc = func() error {
			return startErr
		}
		
		// Configure factory to return our process manager
		errFactory.CreateProcessManagerFunc = func(cm types.ConfigManager, l *zap.Logger) (types.ProcessManager, error) {
			return mockPM, nil
		}
		
		// Try starting again
		err = app.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start process manager")
	})

	// Test stopping the application
	t.Run("Stop application", func(t *testing.T) {
		// Mock the Stop method to succeed
		mockService := testing.NewMockService("test-service")
		app.RegisterService(mockService)
		
		// Since we set isActive in the previous test, it should still be true
		assert.True(t, app.isActive)
		
		// Configure mock to succeed
		mockProcessManager.StopFunc = func() error {
			return nil
		}
		
		// Stop the application
		err := app.Stop()
		assert.NoError(t, err)
		
		// Verify our process manager was stopped
		assert.Equal(t, 1, mockProcessManager.stopCalled)
		
		// Verify application state
		assert.False(t, app.isActive)
	})
}

// TestServiceOperations tests service-related operations
func TestServiceOperations(t *testing.T) {
	// Create test application
	logger := zaptest.NewLogger(t)
	app, err := NewApplication(WithLogger(logger))
	require.NoError(t, err)

	// Register some test services
	service1 := testing.NewMockService("service1")
	service2 := testing.NewMockService("service2")
	app.RegisterService(service1)
	app.RegisterService(service2)

	// Test GetService for existing service
	t.Run("Get existing service", func(t *testing.T) {
		service, exists := app.GetService("service1")
		assert.True(t, exists)
		assert.Equal(t, service1, service)
	})

	// Test GetService for non-existent service
	t.Run("Get non-existent service", func(t *testing.T) {
		service, exists := app.GetService("non-existent")
		assert.False(t, exists)
		assert.Nil(t, service)
	})
}