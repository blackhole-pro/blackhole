package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	identityv1 "github.com/blackhole-pro/blackhole/core/internal/rpc/gen/identity/auth/v1"
)

// AuthServiceImpl implements the AuthService gRPC interface
type AuthServiceImpl struct {
	identityv1.UnimplementedAuthServiceServer
	logger *zap.Logger
}

// NewAuthService creates a new AuthService implementation
func NewAuthService(logger *zap.Logger) *AuthServiceImpl {
	return &AuthServiceImpl{
		logger: logger,
	}
}

// GenerateChallenge creates an authentication challenge for a DID
func (s *AuthServiceImpl) GenerateChallenge(ctx context.Context, req *identityv1.GenerateChallengeRequest) (*identityv1.Challenge, error) {
	s.logger.Info("Generating challenge", zap.String("did", req.GetDid()))

	// Validate request
	if req.GetDid() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "DID is required")
	}

	// Generate random nonce
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		s.logger.Error("Failed to generate nonce", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to generate challenge")
	}

	// Generate challenge ID
	challengeID := fmt.Sprintf("chal_%x", time.Now().UnixNano())

	now := time.Now()
	expires := now.Add(5 * time.Minute)

	challenge := &identityv1.Challenge{
		Id:      challengeID,
		Did:     req.GetDid(),
		Nonce:   fmt.Sprintf("%x", nonce),
		Domain:  "blackhole.local",
		Created: timestamppb.New(now),
		Expires: timestamppb.New(expires),
		Purpose: req.GetPurpose(),
	}

	s.logger.Info("Challenge generated successfully", 
		zap.String("challenge_id", challengeID),
		zap.String("did", req.GetDid()))

	return challenge, nil
}

// VerifyResponse verifies an authentication response against a challenge
func (s *AuthServiceImpl) VerifyResponse(ctx context.Context, req *identityv1.AuthResponse) (*identityv1.AuthResult, error) {
	s.logger.Info("Verifying authentication response", 
		zap.String("did", req.GetDid()),
		zap.String("nonce", req.GetNonce()))

	// Validate request
	if req.GetDid() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "DID is required")
	}
	if req.GetNonce() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Nonce is required")
	}
	if len(req.GetSignature()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Signature is required")
	}

	// For demo purposes, we'll do basic validation
	// In production, this would verify the cryptographic signature
	verified := s.validateSignature(req)

	result := &identityv1.AuthResult{
		Verified:               verified,
		Did:                    req.GetDid(),
		VerificationMethodId:   req.GetVerificationMethodId(),
		AuthenticationDateTime: timestamppb.New(time.Now()),
	}

	if !verified {
		result.Error = "Invalid signature or challenge verification failed"
		s.logger.Warn("Authentication verification failed", 
			zap.String("did", req.GetDid()),
			zap.String("reason", result.Error))
	} else {
		s.logger.Info("Authentication verification successful", 
			zap.String("did", req.GetDid()))
	}

	return result, nil
}

// validateSignature performs basic signature validation for demo purposes
func (s *AuthServiceImpl) validateSignature(req *identityv1.AuthResponse) bool {
	// Basic validation for demo purposes
	// Check that signature looks like a proper signature
	signature := string(req.GetSignature())
	
	// Basic format validation
	if len(signature) < 10 {
		return false
	}

	// Check if DID is properly formatted
	if !s.isValidDID(req.GetDid()) {
		return false
	}

	// Check if nonce is hex encoded and reasonable length
	if len(req.GetNonce()) < 32 {
		return false
	}

	// For demo purposes, signatures starting with "sig_" are considered valid
	if len(signature) > 4 && signature[:4] == "sig_" {
		return true
	}

	return false
}

// isValidDID checks if a DID has the correct format
func (s *AuthServiceImpl) isValidDID(did string) bool {
	// Basic DID format validation
	if len(did) < 10 {
		return false
	}

	// Should start with "did:"
	if did[:4] != "did:" {
		return false
	}

	// Should have at least 3 parts separated by ":"
	parts := len(did)
	colonCount := 0
	for i := 0; i < parts; i++ {
		if did[i] == ':' {
			colonCount++
		}
	}

	return colonCount >= 2
}