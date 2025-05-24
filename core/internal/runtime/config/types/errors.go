// Package types defines error types related to configuration
package types

import "fmt"

// ConfigError represents errors that occur during configuration operations
type ConfigError struct {
	Operation string
	Path      string
	Err       error
}

// Error implements the error interface
func (e *ConfigError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("config error during %s for path %s: %v", e.Operation, e.Path, e.Err)
	}
	return fmt.Sprintf("config error during %s: %v", e.Operation, e.Err)
}

// Unwrap returns the underlying error
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// ValidationError represents configuration validation errors
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation error for %s: %s", e.Field, e.Message)
}