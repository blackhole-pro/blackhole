// Package types defines the core type definitions and interfaces for the
// identity service. It provides the contract that implementations must follow
// and ensures consistency across the identity subsystem.
package types

import "context"

// Service defines the interface for the identity service
type Service interface {
	// Start starts the service with the given context
	Start(ctx context.Context) error
	
	// Stop stops the service
	Stop() error
	
	// Name returns the service name
	Name() string
	
	// Health returns the health status of the service
	Health() bool
}

// IdentityProvider defines the interface for identity creation and management
type IdentityProvider interface {
	// CreateIdentity creates a new identity
	CreateIdentity(ctx context.Context, info IdentityInfo) (string, error)
	
	// GetIdentity retrieves an identity by its ID
	GetIdentity(ctx context.Context, id string) (*IdentityInfo, error)
	
	// ListIdentities lists all identities with pagination
	ListIdentities(ctx context.Context, offset, limit int) ([]IdentityInfo, error)
	
	// DeleteIdentity deletes an identity by its ID
	DeleteIdentity(ctx context.Context, id string) error
}

// CredentialProvider defines the interface for credential operations
type CredentialProvider interface {
	// IssueCredential issues a new credential
	IssueCredential(ctx context.Context, credential Credential) (string, error)
	
	// VerifyCredential verifies a credential
	VerifyCredential(ctx context.Context, id string) (bool, error)
	
	// RevokeCredential revokes a credential
	RevokeCredential(ctx context.Context, id string) error
}

// IdentityInfo contains information about an identity
type IdentityInfo struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Controller  string            `json:"controller,omitempty"`
	Created     int64             `json:"created"`
	Updated     int64             `json:"updated"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	PublicKey   string            `json:"public_key,omitempty"`
	VerifyKey   string            `json:"verify_key,omitempty"`
	Status      string            `json:"status"`
}

// Credential represents a verifiable credential
type Credential struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Issuer        string            `json:"issuer"`
	Subject       string            `json:"subject"`
	IssuanceDate  int64             `json:"issuance_date"`
	ExpirationDate int64            `json:"expiration_date,omitempty"`
	Claims        map[string]string `json:"claims"`
	Proof         string            `json:"proof,omitempty"`
	Status        string            `json:"status"`
}