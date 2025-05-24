package types

import (
	"fmt"
)

// Authentication-specific error types
var (
	// ErrInvalidDID indicates the DID format is invalid
	ErrInvalidDID = fmt.Errorf("invalid DID format")

	// ErrDIDNotFound indicates the DID could not be resolved
	ErrDIDNotFound = fmt.Errorf("DID not found")

	// ErrChallengeExpired indicates the authentication challenge has expired
	ErrChallengeExpired = fmt.Errorf("authentication challenge expired")

	// ErrChallengeNotFound indicates the referenced challenge does not exist
	ErrChallengeNotFound = fmt.Errorf("authentication challenge not found")

	// ErrInvalidSignature indicates the signature verification failed
	ErrInvalidSignature = fmt.Errorf("invalid signature")

	// ErrVerificationMethodNotFound indicates the specified verification method was not found
	ErrVerificationMethodNotFound = fmt.Errorf("verification method not found")

	// ErrUnsupportedSignatureType indicates the signature type is not supported
	ErrUnsupportedSignatureType = fmt.Errorf("unsupported signature type")

	// ErrInvalidNonce indicates the nonce in the response does not match the challenge
	ErrInvalidNonce = fmt.Errorf("invalid nonce")

	// ErrInvalidDIDFormat indicates the DID format is invalid
	ErrInvalidDIDFormat = fmt.Errorf("invalid DID format")

	// ErrUnauthorized indicates the requester is not authorized to perform the operation
	ErrUnauthorized = fmt.Errorf("unauthorized")

	// ErrDIDResolutionFailed indicates the DID resolution process failed
	ErrDIDResolutionFailed = fmt.Errorf("DID resolution failed")
)

// AuthError represents an authentication-specific error with context
type AuthError struct {
	// Err is the underlying error
	Err error

	// DID is the DID related to the error
	DID string

	// ChallengeID is the challenge ID related to the error, if any
	ChallengeID string

	// VerificationMethodID is the verification method ID related to the error, if any
	VerificationMethodID string

	// Message is an additional context message
	Message string
}

// Error returns the error message
func (e *AuthError) Error() string {
	msg := ""
	if e.Err != nil {
		msg = e.Err.Error()
	}

	if e.Message != "" {
		if msg != "" {
			msg = fmt.Sprintf("%s: %s", msg, e.Message)
		} else {
			msg = e.Message
		}
	}

	details := ""
	if e.DID != "" {
		details = fmt.Sprintf("DID=%s", e.DID)
	}

	if e.ChallengeID != "" {
		if details != "" {
			details = fmt.Sprintf("%s, ChallengeID=%s", details, e.ChallengeID)
		} else {
			details = fmt.Sprintf("ChallengeID=%s", e.ChallengeID)
		}
	}

	if e.VerificationMethodID != "" {
		if details != "" {
			details = fmt.Sprintf("%s, VerificationMethodID=%s", details, e.VerificationMethodID)
		} else {
			details = fmt.Sprintf("VerificationMethodID=%s", e.VerificationMethodID)
		}
	}

	if details != "" {
		msg = fmt.Sprintf("%s (%s)", msg, details)
	}

	return msg
}

// Unwrap returns the underlying error
func (e *AuthError) Unwrap() error {
	return e.Err
}

// NewAuthError creates a new AuthError
func NewAuthError(err error, did, challengeID, verificationMethodID, message string) *AuthError {
	return &AuthError{
		Err:                 err,
		DID:                 did,
		ChallengeID:         challengeID,
		VerificationMethodID: verificationMethodID,
		Message:             message,
	}
}