package types

import (
	"fmt"
)

// NodeError represents node service specific errors
type NodeError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Cause   error     `json:"cause,omitempty"`
}

func (e *NodeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *NodeError) Unwrap() error {
	return e.Cause
}

// ErrorCode represents specific error types for node operations
type ErrorCode string

const (
	// Connection errors
	ErrorCodeConnectionFailed    ErrorCode = "CONNECTION_FAILED"
	ErrorCodeConnectionTimeout   ErrorCode = "CONNECTION_TIMEOUT" 
	ErrorCodeConnectionRefused   ErrorCode = "CONNECTION_REFUSED"
	ErrorCodeConnectionClosed    ErrorCode = "CONNECTION_CLOSED"
	
	// Peer errors
	ErrorCodePeerNotFound        ErrorCode = "PEER_NOT_FOUND"
	ErrorCodePeerAlreadyExists   ErrorCode = "PEER_ALREADY_EXISTS"
	ErrorCodePeerUnreachable     ErrorCode = "PEER_UNREACHABLE"
	ErrorCodePeerInvalidAddress  ErrorCode = "PEER_INVALID_ADDRESS"
	
	// Discovery errors
	ErrorCodeDiscoveryFailed     ErrorCode = "DISCOVERY_FAILED"
	ErrorCodeDiscoveryTimeout    ErrorCode = "DISCOVERY_TIMEOUT"
	ErrorCodeDiscoveryUnavailable ErrorCode = "DISCOVERY_UNAVAILABLE"
	
	// Network errors
	ErrorCodeNetworkUnavailable  ErrorCode = "NETWORK_UNAVAILABLE"
	ErrorCodeNetworkDegraded     ErrorCode = "NETWORK_DEGRADED"
	ErrorCodeBandwidthExceeded   ErrorCode = "BANDWIDTH_EXCEEDED"
	
	// Configuration errors
	ErrorCodeInvalidConfig       ErrorCode = "INVALID_CONFIG"
	ErrorCodeConfigNotFound      ErrorCode = "CONFIG_NOT_FOUND"
	
	// Service errors
	ErrorCodeServiceNotReady     ErrorCode = "SERVICE_NOT_READY"
	ErrorCodeServiceShutdown     ErrorCode = "SERVICE_SHUTDOWN"
	ErrorCodeInternalError       ErrorCode = "INTERNAL_ERROR"
)

// Predefined error constructors

func NewConnectionFailedError(address string, cause error) *NodeError {
	return &NodeError{
		Code:    ErrorCodeConnectionFailed,
		Message: fmt.Sprintf("failed to connect to peer at %s", address),
		Cause:   cause,
	}
}

func NewConnectionTimeoutError(address string, timeout string) *NodeError {
	return &NodeError{
		Code:    ErrorCodeConnectionTimeout,
		Message: fmt.Sprintf("connection to peer at %s timed out after %s", address, timeout),
	}
}

func NewPeerNotFoundError(peerID string) *NodeError {
	return &NodeError{
		Code:    ErrorCodePeerNotFound,
		Message: fmt.Sprintf("peer with ID %s not found", peerID),
	}
}

func NewPeerAlreadyExistsError(peerID string) *NodeError {
	return &NodeError{
		Code:    ErrorCodePeerAlreadyExists,
		Message: fmt.Sprintf("peer with ID %s already exists", peerID),
	}
}

func NewPeerUnreachableError(address string) *NodeError {
	return &NodeError{
		Code:    ErrorCodePeerUnreachable,
		Message: fmt.Sprintf("peer at %s is unreachable", address),
	}
}

func NewInvalidAddressError(address string) *NodeError {
	return &NodeError{
		Code:    ErrorCodePeerInvalidAddress,
		Message: fmt.Sprintf("invalid peer address: %s", address),
	}
}

func NewDiscoveryFailedError(method string, cause error) *NodeError {
	return &NodeError{
		Code:    ErrorCodeDiscoveryFailed,
		Message: fmt.Sprintf("peer discovery using method %s failed", method),
		Cause:   cause,
	}
}

func NewDiscoveryTimeoutError(method string, timeout string) *NodeError {
	return &NodeError{
		Code:    ErrorCodeDiscoveryTimeout,
		Message: fmt.Sprintf("peer discovery using method %s timed out after %s", method, timeout),
	}
}

func NewNetworkUnavailableError() *NodeError {
	return &NodeError{
		Code:    ErrorCodeNetworkUnavailable,
		Message: "network is unavailable",
	}
}

func NewNetworkDegradedError(reason string) *NodeError {
	return &NodeError{
		Code:    ErrorCodeNetworkDegraded,
		Message: fmt.Sprintf("network is degraded: %s", reason),
	}
}

func NewInvalidConfigError(field string, value interface{}) *NodeError {
	return &NodeError{
		Code:    ErrorCodeInvalidConfig,
		Message: fmt.Sprintf("invalid configuration value for %s: %v", field, value),
	}
}

func NewServiceNotReadyError() *NodeError {
	return &NodeError{
		Code:    ErrorCodeServiceNotReady,
		Message: "node service is not ready",
	}
}

func NewServiceShutdownError() *NodeError {
	return &NodeError{
		Code:    ErrorCodeServiceShutdown,
		Message: "node service is shutting down",
	}
}

func NewInternalError(operation string, cause error) *NodeError {
	return &NodeError{
		Code:    ErrorCodeInternalError,
		Message: fmt.Sprintf("internal error during %s", operation),
		Cause:   cause,
	}
}

// IsRetryable returns true if the error is retryable
func (e *NodeError) IsRetryable() bool {
	switch e.Code {
	case ErrorCodeConnectionTimeout,
		 ErrorCodeConnectionRefused,
		 ErrorCodePeerUnreachable,
		 ErrorCodeDiscoveryTimeout,
		 ErrorCodeNetworkDegraded:
		return true
	default:
		return false
	}
}

// IsTemporary returns true if the error is temporary
func (e *NodeError) IsTemporary() bool {
	switch e.Code {
	case ErrorCodeConnectionTimeout,
		 ErrorCodeDiscoveryTimeout,
		 ErrorCodeNetworkDegraded,
		 ErrorCodeBandwidthExceeded:
		return true
	default:
		return false
	}
}