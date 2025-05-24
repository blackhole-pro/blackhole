// Package auth implements DID-based authentication for the identity service.
package auth

import (
	"context"
	"time"
	
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	
	authv1 "github.com/blackhole-pro/blackhole/core/internal/rpc/gen/identity/auth/v1"
	"github.com/blackhole-pro/blackhole/core/internal/services/identity/types"
)

// RateLimiter defines the interface for rate limiting functionality
type RateLimiter interface {
	// AllowRequest checks if a request is allowed based on key and limit
	AllowRequest(key string, requestType string) bool
}

// Service implements the gRPC AuthService
type Service struct {
	auth.UnimplementedAuthServiceServer
	authProvider types.AuthenticationProvider
	rateLimiter  RateLimiter
}

// NewService creates a new auth service
func NewService(authProvider types.AuthenticationProvider, rateLimiter RateLimiter) *Service {
	return &Service{
		authProvider: authProvider,
		rateLimiter:  rateLimiter,
	}
}

// RegisterService registers the auth service with a gRPC server
func RegisterService(server *grpc.Server, authProvider types.AuthenticationProvider, rateLimiter RateLimiter) {
	service := NewService(authProvider, rateLimiter)
	auth.RegisterAuthServiceServer(server, service)
}

// GenerateChallenge creates an authentication challenge for a DID
func (s *Service) GenerateChallenge(ctx context.Context, req *auth.GenerateChallengeRequest) (*auth.Challenge, error) {
	// Check if request is allowed by rate limiter
	if s.rateLimiter != nil {
		clientIP, _ := getClientIP(ctx)
		
		// Rate limit by IP
		if !s.rateLimiter.AllowRequest(clientIP, "challenge") {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded for client IP")
		}
		
		// Rate limit by DID
		if !s.rateLimiter.AllowRequest(req.Did, "challenge") {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded for DID")
		}
	}
	
	// Generate challenge using the authProvider
	challenge, err := s.authProvider.GenerateChallenge(ctx, req.Did)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	
	// Convert to proto message
	return &auth.Challenge{
		Id:      challenge.ID,
		Did:     challenge.DID,
		Nonce:   challenge.Nonce,
		Domain:  challenge.Domain,
		Created: timestamppb.New(challenge.Created),
		Expires: timestamppb.New(challenge.Expires),
		Purpose: challenge.Purpose,
	}, nil
}

// VerifyResponse verifies an authentication response against a challenge
func (s *Service) VerifyResponse(ctx context.Context, req *auth.AuthResponse) (*auth.AuthResult, error) {
	// Check if request is allowed by rate limiter
	if s.rateLimiter != nil {
		// Rate limit by DID
		if !s.rateLimiter.AllowRequest(req.Did, "verify") {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded for DID")
		}
	}
	
	// Convert proto message to internal type
	response := &types.AuthResponse{
		ID:                  req.Id,
		ChallengeID:         req.ChallengeId,
		DID:                 req.Did,
		Nonce:               req.Nonce,
		Created:             req.Created.AsTime(),
		VerificationMethodID: req.VerificationMethodId,
		SignatureType:       req.SignatureType,
		Signature:           req.Signature,
		ProtectionLevel:     convertProtectionLevel(req.ProtectionLevel),
	}
	
	// Verify response using the authProvider
	result, err := s.authProvider.VerifyResponse(ctx, response)
	if err != nil {
		// Create a result with error information but still return the error
		// so that the gRPC status code is set correctly
		errorResult := &auth.AuthResult{
			Verified:              false,
			Did:                   req.Did,
			VerificationMethodId:  req.VerificationMethodId,
			Error:                 err.Error(),
			AuthenticationDateTime: timestamppb.New(time.Now()),
		}
		
		// Check for specific error types and return appropriate gRPC status
		switch {
		case types.ErrInvalidDID.Error() == err.Error() || 
			types.ErrInvalidDIDFormat.Error() == err.Error():
			return errorResult, status.Error(codes.InvalidArgument, err.Error())
		case types.ErrDIDNotFound.Error() == err.Error():
			return errorResult, status.Error(codes.NotFound, err.Error())
		case types.ErrChallengeExpired.Error() == err.Error() || 
			types.ErrChallengeNotFound.Error() == err.Error():
			return errorResult, status.Error(codes.InvalidArgument, err.Error())
		case types.ErrInvalidSignature.Error() == err.Error() || 
			types.ErrUnauthorized.Error() == err.Error():
			return errorResult, status.Error(codes.Unauthenticated, err.Error())
		default:
			return errorResult, status.Error(codes.Internal, err.Error())
		}
	}
	
	// Convert successful result to proto message
	return &auth.AuthResult{
		Verified:              result.Verified,
		Did:                   result.DID,
		VerificationMethodId:  result.VerificationMethodID,
		Error:                 result.Error,
		AuthenticationDateTime: timestamppb.New(result.AuthenticationDateTime),
		PseudonymousIdentifier: result.PseudonymousIdentifier,
	}, nil
}

// Helper functions

// getClientIP extracts the client IP from the context
// In a real implementation, this would extract from gRPC metadata
func getClientIP(ctx context.Context) (string, error) {
	// This is a simplified implementation
	// In production, extract from X-Forwarded-For or other metadata
	return "127.0.0.1", nil
}

// convertProtectionLevel converts from proto enum to internal type
func convertProtectionLevel(level auth.ProtectionLevel) types.ProtectionLevel {
	switch level {
	case auth.ProtectionLevel_PROTECTION_PSEUDONYMOUS:
		return types.ProtectionPseudonymous
	case auth.ProtectionLevel_PROTECTION_ANONYMOUS:
		return types.ProtectionAnonymous
	default:
		return types.ProtectionStandard
	}
}