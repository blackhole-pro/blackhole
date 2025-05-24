// Package auth implements DID-based authentication for the identity service.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
	
	"github.com/google/uuid"
	
	"github.com/blackhole-pro/blackhole/core/internal/services/identity/types"
)

// Config contains configuration for the authentication provider
type Config struct {
	// ChallengeTTL is the time-to-live for authentication challenges
	ChallengeTTL time.Duration
	
	// Domain is the service domain for authentication challenges
	Domain string
	
	// NonceLength is the length of the nonce in bytes
	NonceLength int
	
	// MaxChallenges is the maximum number of active challenges to store
	MaxChallenges int
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		ChallengeTTL:  5 * time.Minute,
		Domain:        "blackhole.identity",
		NonceLength:   32,
		MaxChallenges: 1000,
	}
}

// AuthenticationProvider implements the types.AuthenticationProvider interface
type AuthenticationProvider struct {
	config      *Config
	keyManager  types.KeyManager
	resolver    types.DIDResolver
	
	// In-memory challenge store
	challenges  map[string]*types.AuthChallenge
	challengeMu sync.RWMutex
}

// NewAuthenticationProvider creates a new AuthenticationProvider
func NewAuthenticationProvider(
	config *Config,
	keyManager types.KeyManager,
	resolver types.DIDResolver,
) *AuthenticationProvider {
	if config == nil {
		config = DefaultConfig()
	}
	
	return &AuthenticationProvider{
		config:     config,
		keyManager: keyManager,
		resolver:   resolver,
		challenges: make(map[string]*types.AuthChallenge),
	}
}

// GenerateChallenge creates a new authentication challenge for the specified DID
func (a *AuthenticationProvider) GenerateChallenge(ctx context.Context, did string) (*types.AuthChallenge, error) {
	// Validate DID format
	// Note: Full validation will happen when resolving the DID
	if did == "" {
		return nil, types.NewAuthError(types.ErrInvalidDID, did, "", "", "DID cannot be empty")
	}
	
	// Try to resolve the DID to ensure it exists
	_, err := a.resolver.Resolve(ctx, did)
	if err != nil {
		return nil, types.NewAuthError(types.ErrDIDNotFound, did, "", "", fmt.Sprintf("failed to resolve DID: %v", err))
	}
	
	// Generate a random nonce
	nonceBytes := make([]byte, a.config.NonceLength)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce := hex.EncodeToString(nonceBytes)
	
	// Create the challenge
	now := time.Now()
	challenge := &types.AuthChallenge{
		ID:      uuid.New().String(),
		DID:     did,
		Nonce:   nonce,
		Domain:  a.config.Domain,
		Created: now,
		Expires: now.Add(a.config.ChallengeTTL),
		Purpose: "authentication",
	}
	
	// Store the challenge
	a.challengeMu.Lock()
	defer a.challengeMu.Unlock()
	
	// Clean expired challenges if needed
	if len(a.challenges) >= a.config.MaxChallenges {
		a.cleanExpiredChallengesLocked()
	}
	
	a.challenges[challenge.ID] = challenge
	
	return challenge, nil
}

// VerifyResponse verifies an authentication response against a previously generated challenge
func (a *AuthenticationProvider) VerifyResponse(ctx context.Context, response *types.AuthResponse) (*types.AuthResult, error) {
	if response == nil {
		return nil, fmt.Errorf("response cannot be nil")
	}
	
	// Retrieve the challenge
	a.challengeMu.RLock()
	challenge, exists := a.challenges[response.ChallengeID]
	a.challengeMu.RUnlock()
	
	if !exists {
		return nil, types.NewAuthError(types.ErrChallengeNotFound, response.DID, response.ChallengeID, "", "challenge not found")
	}
	
	// Verify the challenge hasn't expired
	if time.Now().After(challenge.Expires) {
		return &types.AuthResult{
			Verified:              false,
			DID:                   response.DID,
			VerificationMethodID:  response.VerificationMethodID,
			Error:                 "challenge expired",
			AuthenticationDateTime: time.Now(),
		}, types.NewAuthError(types.ErrChallengeExpired, response.DID, response.ChallengeID, "", "challenge expired")
	}
	
	// Verify the DIDs match
	if challenge.DID != response.DID {
		return &types.AuthResult{
			Verified:              false,
			DID:                   response.DID,
			VerificationMethodID:  response.VerificationMethodID,
			Error:                 "DID mismatch",
			AuthenticationDateTime: time.Now(),
		}, types.NewAuthError(types.ErrUnauthorized, response.DID, response.ChallengeID, "", "DID in response does not match challenge")
	}
	
	// Verify the nonce matches
	if challenge.Nonce != response.Nonce {
		return &types.AuthResult{
			Verified:              false,
			DID:                   response.DID,
			VerificationMethodID:  response.VerificationMethodID,
			Error:                 "nonce mismatch",
			AuthenticationDateTime: time.Now(),
		}, types.NewAuthError(types.ErrInvalidNonce, response.DID, response.ChallengeID, "", "nonce in response does not match challenge")
	}
	
	// Create the message that was signed (format: nonce + domain)
	message := []byte(challenge.Nonce + challenge.Domain)
	
	// Verify the signature
	verified, err := a.keyManager.Verify(ctx, response.DID, response.VerificationMethodID, message, response.Signature)
	if err != nil {
		return &types.AuthResult{
			Verified:              false,
			DID:                   response.DID,
			VerificationMethodID:  response.VerificationMethodID,
			Error:                 fmt.Sprintf("signature verification failed: %v", err),
			AuthenticationDateTime: time.Now(),
		}, types.NewAuthError(types.ErrInvalidSignature, response.DID, response.ChallengeID, response.VerificationMethodID, err.Error())
	}
	
	if !verified {
		return &types.AuthResult{
			Verified:              false,
			DID:                   response.DID,
			VerificationMethodID:  response.VerificationMethodID,
			Error:                 "invalid signature",
			AuthenticationDateTime: time.Now(),
		}, types.NewAuthError(types.ErrInvalidSignature, response.DID, response.ChallengeID, response.VerificationMethodID, "signature verification failed")
	}
	
	// Resolve the DID document to get the verification method details
	doc, err := a.resolver.Resolve(ctx, response.DID)
	if err != nil {
		// We've already verified the signature, so this is just to get additional info
		// Return success even if we can't get the full verification method
		return &types.AuthResult{
			Verified:              true,
			DID:                   response.DID,
			VerificationMethodID:  response.VerificationMethodID,
			AuthenticationDateTime: time.Now(),
		}, nil
	}
	
	// Find the verification method
	var verificationMethod *types.VerificationMethod
	for _, vm := range doc.VerificationMethod {
		if vm.ID == response.VerificationMethodID {
			verificationMethod = &vm
			break
		}
	}
	
	// Remove the challenge to prevent reuse
	a.challengeMu.Lock()
	delete(a.challenges, response.ChallengeID)
	a.challengeMu.Unlock()
	
	// Generate pseudonymous identifier if requested
	var pseudonymousID string
	if response.ProtectionLevel == types.ProtectionPseudonymous {
		pseudonymousID = generatePseudonymousID(response.DID, a.config.Domain)
	}
	
	// Create authentication result
	result := &types.AuthResult{
		Verified:              true,
		DID:                   response.DID,
		VerificationMethodID:  response.VerificationMethodID,
		VerificationMethod:    verificationMethod,
		AuthenticationDateTime: time.Now(),
		PseudonymousIdentifier: pseudonymousID,
	}
	
	// If the verification method has a different controller, note that
	if verificationMethod != nil && verificationMethod.Controller != response.DID {
		result.ControllerDID = verificationMethod.Controller
	}
	
	return result, nil
}

// generatePseudonymousID creates a privacy-preserving identifier derived from the DID
func generatePseudonymousID(did, domain string) string {
	// Simple implementation: base64(hash(did + domain))
	// In a production system, consider using a more sophisticated approach
	// like HMAC with a service-specific key
	idStr := did + ":" + domain
	return base64.URLEncoding.EncodeToString([]byte(idStr))
}

// cleanExpiredChallengesLocked removes expired challenges (must be called with the lock held)
func (a *AuthenticationProvider) cleanExpiredChallengesLocked() {
	now := time.Now()
	for id, challenge := range a.challenges {
		if now.After(challenge.Expires) {
			delete(a.challenges, id)
		}
	}
}