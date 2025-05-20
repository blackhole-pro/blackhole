// Package types defines error types used in the process orchestrator component.
// It provides domain-specific error types with contextual information that
// helps with debugging and error handling across the process subsystem.
// These error types support the error wrapping introduced in Go 1.13+.
package types

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessError tests the ProcessError type
func TestProcessError(t *testing.T) {
	// Test basic error creation and formatting
	t.Run("Basic error creation", func(t *testing.T) {
		err := &ProcessError{
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
		err := &ProcessError{
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
		err := &ProcessError{
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
		err := NewProcessError("example", baseErr).
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
		require.NotNil(t, ErrServiceNotFound)
		require.NotNil(t, ErrServiceDisabled)
		require.NotNil(t, ErrAlreadyRunning)
		require.NotNil(t, ErrNotRunning)
		require.NotNil(t, ErrShuttingDown)
		require.NotNil(t, ErrConfigChanged)
		require.NotNil(t, ErrBinaryNotFound)
		require.NotNil(t, ErrTimeout)
		require.NotNil(t, ErrMaxRestartsExceeded)
	})

	// Test error checking functions
	t.Run("Error checking functions", func(t *testing.T) {
		// Direct error check
		assert.True(t, IsServiceNotFound(ErrServiceNotFound))
		assert.True(t, IsShuttingDown(ErrShuttingDown))
		assert.True(t, IsBinaryNotFound(ErrBinaryNotFound))
		assert.True(t, IsTimeout(ErrTimeout))
		assert.True(t, IsMaxRestartsExceeded(ErrMaxRestartsExceeded))
		
		// Non-matching errors
		assert.False(t, IsServiceNotFound(ErrTimeout))
		assert.False(t, IsShuttingDown(ErrServiceNotFound))
		
		// Wrapped errors
		wrappedErr := fmt.Errorf("wrapped: %w", ErrServiceNotFound)
		assert.True(t, IsServiceNotFound(wrappedErr))
		assert.False(t, IsShuttingDown(wrappedErr))
		
		// Decorated errors
		procErr := NewProcessError("test", ErrTimeout)
		assert.True(t, IsTimeout(procErr))
		assert.False(t, IsServiceNotFound(procErr))
	})

	// Test error wrapping with ProcessError
	t.Run("Error wrapping", func(t *testing.T) {
		// Create a base error
		baseErr := ErrBinaryNotFound
		
		// Wrap in ProcessError
		procErr := NewProcessError("test-service", baseErr)
		
		// Check if wrapping preserves error identity
		assert.True(t, IsBinaryNotFound(procErr))
		assert.False(t, IsTimeout(procErr))
		
		// Double wrapping
		doubleWrapped := fmt.Errorf("additional context: %w", procErr) 
		assert.True(t, IsBinaryNotFound(doubleWrapped))
	})
}