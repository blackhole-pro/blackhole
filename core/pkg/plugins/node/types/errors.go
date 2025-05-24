// Package types defines error types for the node plugin
package types

import (
	"errors"
	"fmt"
)

// Common errors
var (
	// Plugin lifecycle errors
	ErrPluginNotRunning     = errors.New("plugin not running")
	ErrPluginAlreadyRunning = errors.New("plugin already running")
	ErrPluginShutdown       = errors.New("plugin is shutting down")
	
	// Peer connection errors
	ErrPeerNotFound         = errors.New("peer not found")
	ErrPeerAlreadyConnected = errors.New("peer already connected")
	ErrMaxPeersReached      = errors.New("maximum peers limit reached")
	ErrConnectionTimeout    = errors.New("connection timeout")
	ErrInvalidPeerID        = errors.New("invalid peer ID")
	
	// Configuration errors
	ErrInvalidConfig        = errors.New("invalid configuration")
	ErrMissingNodeID        = errors.New("node ID is required")
	ErrInvalidPort          = errors.New("invalid P2P port")
	ErrInvalidDiscoveryMethod = errors.New("invalid discovery method")
	
	// Resource errors
	ErrBandwidthExceeded    = errors.New("bandwidth limit exceeded")
	ErrResourceLimitReached = errors.New("resource limit reached")
	
	// Discovery errors
	ErrDiscoveryDisabled    = errors.New("peer discovery is disabled")
	ErrDiscoveryFailed      = errors.New("peer discovery failed")
	
	// Network errors
	ErrNetworkUnhealthy     = errors.New("network is unhealthy")
	ErrNoBootstrapPeers     = errors.New("no bootstrap peers configured")
)

// PeerError represents an error related to peer operations
type PeerError struct {
	PeerID    string
	Operation string
	Err       error
}

// Error implements the error interface
func (e *PeerError) Error() string {
	return fmt.Sprintf("peer %s: %s failed: %v", e.PeerID, e.Operation, e.Err)
}

// Unwrap returns the underlying error
func (e *PeerError) Unwrap() error {
	return e.Err
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field string
	Value interface{}
	Err   error
}

// Error implements the error interface
func (e *ConfigError) Error() string {
	return fmt.Sprintf("config field %s with value %v: %v", e.Field, e.Value, e.Err)
}

// Unwrap returns the underlying error
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// NetworkError represents a network-related error
type NetworkError struct {
	Operation string
	Address   string
	Err       error
}

// Error implements the error interface
func (e *NetworkError) Error() string {
	if e.Address != "" {
		return fmt.Sprintf("network %s on %s failed: %v", e.Operation, e.Address, e.Err)
	}
	return fmt.Sprintf("network %s failed: %v", e.Operation, e.Err)
}

// Unwrap returns the underlying error
func (e *NetworkError) Unwrap() error {
	return e.Err
}

// DiscoveryError represents a peer discovery error
type DiscoveryError struct {
	Method string
	Err    error
}

// Error implements the error interface
func (e *DiscoveryError) Error() string {
	return fmt.Sprintf("discovery via %s failed: %v", e.Method, e.Err)
}

// Unwrap returns the underlying error
func (e *DiscoveryError) Unwrap() error {
	return e.Err
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

// Helper functions for creating errors

// NewPeerError creates a new peer error
func NewPeerError(peerID, operation string, err error) error {
	return &PeerError{
		PeerID:    peerID,
		Operation: operation,
		Err:       err,
	}
}

// NewConfigError creates a new configuration error
func NewConfigError(field string, value interface{}, err error) error {
	return &ConfigError{
		Field: field,
		Value: value,
		Err:   err,
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(operation, address string, err error) error {
	return &NetworkError{
		Operation: operation,
		Address:   address,
		Err:       err,
	}
}

// NewDiscoveryError creates a new discovery error
func NewDiscoveryError(method string, err error) error {
	return &DiscoveryError{
		Method: method,
		Err:    err,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}