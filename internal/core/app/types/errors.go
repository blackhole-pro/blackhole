// Package types defines the core type definitions and interfaces for the
// application component. It provides the contract that implementations must follow
// and ensures consistency across the application subsystem.
package types

import (
	"fmt"
)

// AppError is the base error type for application-related errors
type AppError struct {
	Message string
	Err     error
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// ServiceError represents an error related to service operations
type ServiceError struct {
	Service string
	Op      string
	Err     error
}

// Error implements the error interface
func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("service %s: %s operation failed: %v", e.Service, e.Op, e.Err)
	}
	return fmt.Sprintf("service %s: %s operation failed", e.Service, e.Op)
}

// Unwrap returns the underlying error
func (e *ServiceError) Unwrap() error {
	return e.Err
}

// ConfigError represents an error related to configuration operations
type ConfigError struct {
	Path string
	Op   string
	Err  error
}

// Error implements the error interface
func (e *ConfigError) Error() string {
	if e.Path != "" {
		if e.Err != nil {
			return fmt.Sprintf("config %s: %s operation failed: %v", e.Path, e.Op, e.Err)
		}
		return fmt.Sprintf("config %s: %s operation failed", e.Path, e.Op)
	}
	
	if e.Err != nil {
		return fmt.Sprintf("config: %s operation failed: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("config: %s operation failed", e.Op)
}

// Unwrap returns the underlying error
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// Known error types - using variables for consistent error comparison
var (
	ErrServiceNotFound   = &AppError{Message: "service not found"}
	ErrServiceNotRunning = &AppError{Message: "service not running"}
	ErrServiceDisabled   = &AppError{Message: "service disabled"}
	ErrServiceAlreadyRegistered = &AppError{Message: "service already registered"}
)

// NewServiceError creates a new ServiceError
func NewServiceError(service, op string, err error) *ServiceError {
	return &ServiceError{
		Service: service,
		Op:      op,
		Err:     err,
	}
}

// NewConfigError creates a new ConfigError
func NewConfigError(path, op string, err error) *ConfigError {
	return &ConfigError{
		Path: path,
		Op:   op,
		Err:  err,
	}
}