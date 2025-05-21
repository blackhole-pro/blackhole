// Package types_test provides tests for the process orchestrator error types.
package types_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/handcraftdev/blackhole/internal/core/process/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessError tests the ProcessError type
func TestProcessError(t *testing.T) {
	// Test basic error creation and formatting
	t.Run("Basic error creation", func(t *testing.T) {
		err := &types.ProcessError{
			Service:  "test-service",
			Err:      errors.New("test error"),
			ExitCode: 1,
		}

		// Check error formatting
		assert.Equal(t, "service test-service: test error (exit code: 1)", err.Error())

		// Test Unwrap
		assert.Equal(t, "test error", err.Unwrap().Error())
	})

	// Test error with PID
	t.Run("Error with PID", func(t *testing.T) {
		err := &types.ProcessError{
			Service:  "test-service",
			Err:      errors.New("test error"),
			ExitCode: 1,
			PID:      1000,
		}

		// Check error formatting includes PID
		assert.Equal(t, "service test-service: test error (exit code: 1) (pid: 1000)", err.Error())
	})

	// Test error with context
	t.Run("Error with context", func(t *testing.T) {
		err := &types.ProcessError{
			Service:  "test-service",
			Err:      errors.New("test error"),
			Context:  "during startup",
		}

		// Check error formatting includes context
		assert.Equal(t, "service test-service: test error (during startup)", err.Error())
	})

	// Test builder pattern
	t.Run("Builder pattern", func(t *testing.T) {
		baseErr := errors.New("base error")
		
		// Create using builder pattern
		err := types.NewProcessError("example", baseErr).
			WithExitCode(2).
			WithPID(1234).
			WithContext("initialization")
		
		// Check all fields are set correctly
		assert.Equal(t, "example", err.Service)
		assert.Equal(t, baseErr, err.Err)
		assert.Equal(t, 2, err.ExitCode)
		assert.Equal(t, 1234, err.PID)
		assert.Equal(t, "initialization", err.Context)
		
		// Check formatting
		assert.Equal(t, "service example: base error (initialization) (exit code: 2) (pid: 1234)", err.Error())
	})
}

// TestErrorTypes tests the error constants and helper functions
func TestErrorTypes(t *testing.T) {
	// Test standard error constants
	t.Run("Standard error constants", func(t *testing.T) {
		// Check that all error constants are defined
		require.NotNil(t, types.ErrServiceNotFound)
		require.NotNil(t, types.ErrServiceDisabled)
		require.NotNil(t, types.ErrAlreadyRunning)
		require.NotNil(t, types.ErrNotRunning)
		require.NotNil(t, types.ErrShuttingDown)
		require.NotNil(t, types.ErrConfigChanged)
		require.NotNil(t, types.ErrBinaryNotFound)
		require.NotNil(t, types.ErrTimeout)
		require.NotNil(t, types.ErrMaxRestartsExceeded)
	})

	// Test error checking functions
	t.Run("Error checking functions", func(t *testing.T) {
		// Direct error check
		assert.True(t, types.IsServiceNotFound(types.ErrServiceNotFound))
		assert.True(t, types.IsShuttingDown(types.ErrShuttingDown))
		assert.True(t, types.IsBinaryNotFound(types.ErrBinaryNotFound))
		assert.True(t, types.IsTimeout(types.ErrTimeout))
		assert.True(t, types.IsMaxRestartsExceeded(types.ErrMaxRestartsExceeded))
		
		// Non-matching errors
		assert.False(t, types.IsServiceNotFound(types.ErrTimeout))
		assert.False(t, types.IsShuttingDown(types.ErrServiceNotFound))
		
		// Wrapped errors
		wrappedErr := fmt.Errorf("wrapped: %w", types.ErrServiceNotFound)
		assert.True(t, types.IsServiceNotFound(wrappedErr))
		assert.False(t, types.IsShuttingDown(wrappedErr))
		
		// Decorated errors
		procErr := types.NewProcessError("test", types.ErrTimeout)
		assert.True(t, types.IsTimeout(procErr))
		assert.False(t, types.IsServiceNotFound(procErr))
	})

	// Test error wrapping with ProcessError
	t.Run("Error wrapping", func(t *testing.T) {
		// Create a base error
		baseErr := types.ErrBinaryNotFound
		
		// Wrap in ProcessError
		procErr := types.NewProcessError("test-service", baseErr)
		
		// Check if wrapping preserves error identity
		assert.True(t, types.IsBinaryNotFound(procErr))
		assert.False(t, types.IsTimeout(procErr))
		
		// Double wrapping
		doubleWrapped := fmt.Errorf("additional context: %w", procErr) 
		assert.True(t, types.IsBinaryNotFound(doubleWrapped))
	})
}