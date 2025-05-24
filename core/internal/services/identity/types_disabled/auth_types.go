package types

import "context"

// ProtectionLevel represents the level of privacy protection
type ProtectionLevel int

const (
	// ProtectionUnspecified is the default unspecified protection level
	ProtectionUnspecified ProtectionLevel = iota
	
	// ProtectionMinimal provides basic identity verification
	ProtectionMinimal
	
	// ProtectionStandard provides standard authentication
	ProtectionStandard
	
	// ProtectionEnhanced provides enhanced privacy protection
	ProtectionEnhanced
	
	// ProtectionMaximum provides maximum privacy protection
	ProtectionMaximum
)

// AuthInfo represents authentication information
type AuthInfo struct {
	// DID of the authenticated entity
	DID string
	
	// Authentication token or signature
	AuthToken string
	
	// Verification method ID used for authentication
	VerificationMethodID string
	
	// Timestamp when authentication occurred (RFC 3339 date-time)
	AuthenticationTime string
	
	// Pseudonymous identifier for privacy-preserving operations
	PseudonymousIdentifier string
	
	// Authentication provider (e.g., "wallet", "oauth")
	AuthProvider string
	
	// For OAuth, the provider used (e.g., "google", "facebook", "apple")
	OAuthProvider string
	
	// Protection level for this authentication
	ProtectionLevel ProtectionLevel
	
	// Additional authentication metadata
	Metadata map[string]string
}

// ResolutionOptions represents options for DID resolution
type ResolutionOptions struct {
	// Version ID for retrieving a specific version of a DID document
	VersionID string
	
	// Authentication information for authorized access
	AuthInfo *AuthInfo
	
	// Whether to accept cached responses
	AcceptCache bool
	
	// Protection level for privacy-sensitive operations
	ProtectionLevel ProtectionLevel
}

// DIDDocument represents a W3C compliant DID document
type DIDDocument struct {
	// JSON-LD context for the DID document
	Context []string `json:"@context,omitempty"`
	
	// ID is the DID this document describes (same as DID field for compatibility)
	ID string `json:"id"`
	
	// The DID this document describes
	DID string
	
	// The controller DIDs that control this DID
	Controllers []string
	
	// Also Known As - alternative identifiers for this DID
	AlsoKnownAs []string
	
	// Verification methods associated with this DID
	VerificationMethods []*VerificationMethod
	
	// Authentication verification method references
	Authentication []string
	
	// Assertion method verification method references
	AssertionMethod []string
	
	// Key agreement verification method references
	KeyAgreement []string
	
	// Capability invocation verification method references
	CapabilityInvocation []string
	
	// Capability delegation verification method references
	CapabilityDelegation []string
	
	// Service endpoints associated with this DID
	Services []*DIDService
	
	// Creation timestamp as RFC 3339 date-time string
	Created string
	
	// Last updated timestamp as RFC 3339 date-time string
	Updated string
	
	// Version identifier for the document
	VersionID string
	
	// Additional properties as key-value pairs
	Properties map[string]string
}

// VerificationMethod represents a cryptographic verification method in a DID document
type VerificationMethod struct {
	// Unique identifier for this verification method
	ID string
	
	// Type of the verification method (e.g., Ed25519VerificationKey2020)
	Type string
	
	// Controller DID that controls this verification method
	Controller string
	
	// Public key material, encoded as specified by the verification method type
	PublicKeyMultibase string
	
	// Public key in JWK format (alternative to PublicKeyMultibase)
	PublicKeyJwk map[string]interface{}
}

// DIDService represents a service endpoint in a DID document
type DIDService struct {
	// Unique identifier for this service
	ID string
	
	// Type of service
	Type string
	
	// Service endpoint URL
	ServiceEndpoint string
	
	// Additional properties as key-value pairs
	Properties map[string]string
}

// DIDResolver defines the interface for resolving DIDs to DID documents
type DIDResolver interface {
	// Resolve resolves a DID to its DID Document
	Resolve(ctx context.Context, did string) (*DIDDocument, error)
	
	// ResolveDID resolves a DID to a DID document
	ResolveDID(ctx context.Context, did string, options *ResolutionOptions) (*DIDDocument, error)
	
	// GetVerificationMethod retrieves a specific verification method from a DID document
	GetVerificationMethod(ctx context.Context, did string, id string, options *ResolutionOptions) (*VerificationMethod, error)
	
	// DIDExists checks if a DID document exists
	DIDExists(ctx context.Context, did string) (bool, error)
}

// Missing types that other files expect

// ResolveOptions provides options for DID resolution
type ResolveOptions struct {
	// Accept media type for the resolution
	Accept string
	
	// Enable privacy protection
	PrivacyProtection bool
}

// AuthChallenge represents an authentication challenge
type AuthChallenge struct {
	// ID uniquely identifies this challenge
	ID string `json:"id"`
	
	// DID is the DID being authenticated
	DID string `json:"did"`
	
	// Nonce is a random value to prevent replay attacks
	Nonce string `json:"nonce"`
	
	// Challenge data
	Challenge string `json:"challenge"`
	
	// Expires at timestamp
	ExpiresAt int64 `json:"expires_at"`
}

// AuthResponse represents a response to an authentication challenge
type AuthResponse struct {
	// Challenge ID this response relates to
	ChallengeID string `json:"challenge_id"`
	
	// DID being authenticated
	DID string `json:"did"`
	
	// Signature over the challenge
	Signature string `json:"signature"`
	
	// Verification method used
	VerificationMethod string `json:"verification_method"`
}

// AuthResult represents the result of authentication
type AuthResult struct {
	// Whether authentication was successful
	Success bool `json:"success"`
	
	// DID that was authenticated
	DID string `json:"did"`
	
	// Error message if authentication failed
	Error string `json:"error,omitempty"`
	
	// Authentication timestamp
	AuthenticatedAt int64 `json:"authenticated_at"`
}

// KeyManager defines the interface for key management operations
type KeyManager interface {
	// GenerateKeyPair generates a new key pair
	GenerateKeyPair(keyType string) ([]byte, []byte, error)
	
	// Sign signs data with a private key
	Sign(privateKey []byte, data []byte) ([]byte, error)
	
	// Verify verifies a signature
	Verify(publicKey []byte, data []byte, signature []byte) error
}

// AuthenticationProvider provides authentication functionality
type AuthenticationProvider interface {
	// GenerateChallenge generates a new authentication challenge
	GenerateChallenge(did string) (*AuthChallenge, error)
	
	// VerifyResponse verifies an authentication response
	VerifyResponse(response *AuthResponse) (*AuthResult, error)
}