// Package types defines error types used in the process orchestrator component.
// It provides domain-specific error types with contextual information that
// helps with debugging and error handling across the process subsystem.
// These error types support the error wrapping introduced in Go 1.13+.
package types

import (
	"errors"
	"fmt"
)

// Common process-related error types for classification and handling
var (
	// ErrServiceNotFound indicates that a requested service doesn't exist
	ErrServiceNotFound = errors.New("service not found")

	// ErrServiceDisabled indicates that a service is configured but disabled
	ErrServiceDisabled = errors.New("service is disabled")

	// ErrAlreadyRunning indicates that a service is already running
	ErrAlreadyRunning = errors.New("service is already running")

	// ErrNotRunning indicates that a service is not currently running
	ErrNotRunning = errors.New("service is not running")

	// ErrShuttingDown indicates that the orchestrator is in shutdown mode
	ErrShuttingDown = errors.New("orchestrator is shutting down")

	// ErrConfigChanged indicates that a configuration change was detected
	ErrConfigChanged = errors.New("configuration changed")

	// ErrBinaryNotFound indicates that a service binary was not found
	ErrBinaryNotFound = errors.New("service binary not found")

	// ErrTimeout indicates a timeout occurred during an operation
	ErrTimeout = errors.New("operation timed out")

	// ErrMaxRestartsExceeded indicates a service has exceeded its restart limit
	ErrMaxRestartsExceeded = errors.New("maximum restart attempts exceeded")
)

// ProcessError provides contextual information about process errors
type ProcessError struct {
	Service  string
	Err      error
	ExitCode int
	PID      int
	Context  string
}

// Error implements the error interface
func (e *ProcessError) Error() string {
	msg := fmt.Sprintf("service %s: %v", e.Service, e.Err)
	
	if e.Context != "" {
		msg += " (" + e.Context + ")"
	}
	
	if e.ExitCode != 0 {
		msg += fmt.Sprintf(" (exit code: %d)", e.ExitCode)
	}
	
	if e.PID > 0 {
		msg += fmt.Sprintf(" (pid: %d)", e.PID)
	}
	
	return msg
}

// Unwrap returns the underlying error
func (e *ProcessError) Unwrap() error {
	return e.Err
}

// NewProcessError creates a new ProcessError with the given service and error
func NewProcessError(service string, err error) *ProcessError {
	return &ProcessError{
		Service: service,
		Err:     err,
	}
}

// WithExitCode adds an exit code to the error
func (e *ProcessError) WithExitCode(code int) *ProcessError {
	e.ExitCode = code
	return e
}

// WithPID adds a process ID to the error
func (e *ProcessError) WithPID(pid int) *ProcessError {
	e.PID = pid
	return e
}

// WithContext adds context information to the error
func (e *ProcessError) WithContext(ctx string) *ProcessError {
	e.Context = ctx
	return e
}

// IsServiceNotFound checks if an error indicates a service not found condition
func IsServiceNotFound(err error) bool {
	return errors.Is(err, ErrServiceNotFound)
}

// IsShuttingDown checks if an error indicates that shutdown is in progress
func IsShuttingDown(err error) bool {
	return errors.Is(err, ErrShuttingDown)
}

// IsBinaryNotFound checks if an error indicates a missing binary
func IsBinaryNotFound(err error) bool {
	return errors.Is(err, ErrBinaryNotFound)
}

// IsTimeout checks if an error indicates a timeout
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsMaxRestartsExceeded checks if a service has exceeded its restart limit
func IsMaxRestartsExceeded(err error) bool {
	return errors.Is(err, ErrMaxRestartsExceeded)
}