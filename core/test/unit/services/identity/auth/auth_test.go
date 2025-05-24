package auth_test

import (
	"context"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	
	"github.com/blackhole-pro/blackhole/core/internal/services/identity/auth_disabled"
	"github.com/blackhole-pro/blackhole/core/internal/services/identity/types"
)

// MockKeyManager implements the types.KeyManager interface for testing
type MockKeyManager struct {
	mock.Mock
}

func (m *MockKeyManager) Verify(ctx context.Context, did string, verificationMethodID string, message []byte, signature []byte) (bool, error) {
	args := m.Called(ctx, did, verificationMethodID, message, signature)
	return args.Bool(0), args.Error(1)
}

func (m *MockKeyManager) GetVerificationMethods(ctx context.Context, did string) ([]types.VerificationMethod, error) {
	args := m.Called(ctx, did)
	return args.Get(0).([]types.VerificationMethod), args.Error(1)
}

// MockDIDResolver implements the types.DIDResolver interface for testing
type MockDIDResolver struct {
	mock.Mock
}

func (m *MockDIDResolver) Resolve(ctx context.Context, did string) (*types.DIDDocument, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.DIDDocument), args.Error(1)
}

func (m *MockDIDResolver) ResolveWithOptions(ctx context.Context, did string, options *types.ResolveOptions) (*types.DIDDocument, error) {
	args := m.Called(ctx, did, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.DIDDocument), args.Error(1)
}

func TestAuthenticationProvider_GenerateChallenge(t *testing.T) {
	// Setup
	mockKeyManager := new(MockKeyManager)
	mockDIDResolver := new(MockDIDResolver)
	
	config := &auth.Config{
		ChallengeTTL:  5 * time.Minute,
		Domain:        "test.domain",
		NonceLength:   16,
		MaxChallenges: 10,
	}
	
	authProvider := auth.NewAuthenticationProvider(config, mockKeyManager, mockDIDResolver)
	
	// Test DID
	testDID := "did:blackhole:test123"
	
	// Mock DID resolution
	mockDIDResolver.On("Resolve", mock.Anything, testDID).Return(&types.DIDDocument{
		ID: testDID,
		VerificationMethod: []types.VerificationMethod{
			{
				ID:         testDID + "#key-1",
				Type:       "Ed25519VerificationKey2020",
				Controller: testDID,
			},
		},
	}, nil)
	
	// Act
	ctx := context.Background()
	challenge, err := authProvider.GenerateChallenge(ctx, testDID)
	
	// Assert
	require.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, testDID, challenge.DID)
	assert.Equal(t, config.Domain, challenge.Domain)
	assert.NotEmpty(t, challenge.Nonce)
	assert.NotEmpty(t, challenge.ID)
	assert.Equal(t, "authentication", challenge.Purpose)
	
	// Verify time constraints
	assert.WithinDuration(t, time.Now(), challenge.Created, 2*time.Second)
	assert.WithinDuration(t, time.Now().Add(config.ChallengeTTL), challenge.Expires, 2*time.Second)
	
	// Verify mock expectations
	mockDIDResolver.AssertExpectations(t)
}

func TestAuthenticationProvider_VerifyResponse(t *testing.T) {
	// Setup
	mockKeyManager := new(MockKeyManager)
	mockDIDResolver := new(MockDIDResolver)
	
	config := &auth.Config{
		ChallengeTTL:  5 * time.Minute,
		Domain:        "test.domain",
		NonceLength:   16,
		MaxChallenges: 10,
	}
	
	authProvider := auth.NewAuthenticationProvider(config, mockKeyManager, mockDIDResolver)
	
	// Test DID and verification method
	testDID := "did:blackhole:test123"
	testVerificationMethodID := testDID + "#key-1"
	
	// Generate a real challenge
	ctx := context.Background()
	mockDIDResolver.On("Resolve", mock.Anything, testDID).Return(&types.DIDDocument{
		ID: testDID,
		VerificationMethod: []types.VerificationMethod{
			{
				ID:         testVerificationMethodID,
				Type:       "Ed25519VerificationKey2020",
				Controller: testDID,
			},
		},
	}, nil)
	
	challenge, err := authProvider.GenerateChallenge(ctx, testDID)
	require.NoError(t, err)
	
	// Create response
	response := &types.AuthResponse{
		ID:                  "resp-123",
		ChallengeID:         challenge.ID,
		DID:                 testDID,
		Nonce:               challenge.Nonce,
		Created:             time.Now(),
		VerificationMethodID: testVerificationMethodID,
		SignatureType:       "Ed25519VerificationKey2020",
		Signature:           []byte("dummy-signature"),
	}
	
	// Mock signature verification
	message := []byte(challenge.Nonce + challenge.Domain)
	mockKeyManager.On("Verify", mock.Anything, testDID, testVerificationMethodID, message, response.Signature).Return(true, nil)
	
	// Mock DID resolution for verification method details
	mockDIDResolver.On("Resolve", mock.Anything, testDID).Return(&types.DIDDocument{
		ID: testDID,
		VerificationMethod: []types.VerificationMethod{
			{
				ID:         testVerificationMethodID,
				Type:       "Ed25519VerificationKey2020",
				Controller: testDID,
				PublicKeyMultibase: "z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
			},
		},
		Authentication: []string{testVerificationMethodID},
	}, nil).Once()
	
	// Act
	result, err := authProvider.VerifyResponse(ctx, response)
	
	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Verified)
	assert.Equal(t, testDID, result.DID)
	assert.Equal(t, testVerificationMethodID, result.VerificationMethodID)
	assert.NotNil(t, result.VerificationMethod)
	
	// Verify mock expectations
	mockKeyManager.AssertExpectations(t)
	mockDIDResolver.AssertExpectations(t)
	
	// Test failed verification
	mockKeyManager.On("Verify", mock.Anything, testDID, testVerificationMethodID, message, response.Signature).Return(false, nil)
	
	// Act
	result, err = authProvider.VerifyResponse(ctx, response)
	
	// Assert
	require.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Verified)
	assert.Equal(t, testDID, result.DID)
	assert.Equal(t, testVerificationMethodID, result.VerificationMethodID)
	assert.Contains(t, result.Error, "invalid signature")
	
	// Verify mock expectations again
	mockKeyManager.AssertExpectations(t)
}

func TestAuthenticationProvider_InvalidChallenge(t *testing.T) {
	// Setup
	mockKeyManager := new(MockKeyManager)
	mockDIDResolver := new(MockDIDResolver)
	
	config := &auth.Config{
		ChallengeTTL:  5 * time.Minute,
		Domain:        "test.domain",
		NonceLength:   16,
		MaxChallenges: 10,
	}
	
	authProvider := auth.NewAuthenticationProvider(config, mockKeyManager, mockDIDResolver)
	
	// Test invalid challenge ID
	response := &types.AuthResponse{
		ID:          "resp-123",
		ChallengeID: "non-existent-challenge",
		DID:         "did:blackhole:test123",
	}
	
	// Act
	ctx := context.Background()
	result, err := authProvider.VerifyResponse(ctx, response)
	
	// Assert
	require.Error(t, err)
	assert.Equal(t, types.ErrChallengeNotFound, err.(*types.AuthError).Err)
	assert.False(t, result.Verified)
}