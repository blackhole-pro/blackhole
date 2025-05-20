// Package types defines error types used in the identity service
package types

import "fmt"

// IdentityError provides contextual information about identity-related errors
type IdentityError struct {
	ID       string
	Code     string
	Message  string
	Err      error
}

// Error implements the error interface
func (e *IdentityError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("identity error [%s] for ID %s: %s (%v)", e.Code, e.ID, e.Message, e.Err)
	}
	return fmt.Sprintf("identity error [%s]: %s (%v)", e.Code, e.Message, e.Err)
}

// Unwrap returns the underlying error
func (e *IdentityError) Unwrap() error {
	return e.Err
}

// CredentialError provides contextual information about credential-related errors
type CredentialError struct {
	ID       string
	Code     string
	Message  string
	Err      error
}

// Error implements the error interface
func (e *CredentialError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("credential error [%s] for ID %s: %s (%v)", e.Code, e.ID, e.Message, e.Err)
	}
	return fmt.Sprintf("credential error [%s]: %s (%v)", e.Code, e.Message, e.Err)
}

// Unwrap returns the underlying error
func (e *CredentialError) Unwrap() error {
	return e.Err
}

// Error codes for the identity service
const (
	ErrNotFound             = "NOT_FOUND"
	ErrInvalidInput         = "INVALID_INPUT"
	ErrAlreadyExists        = "ALREADY_EXISTS"
	ErrUnauthorized         = "UNAUTHORIZED"
	ErrInternal             = "INTERNAL"
	ErrExpired              = "EXPIRED"
	ErrRevoked              = "REVOKED"
	ErrInvalidSignature     = "INVALID_SIGNATURE"
	ErrUnsupportedOperation = "UNSUPPORTED_OPERATION"
)