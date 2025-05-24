package auth

import (
	"context"
	"time"
	
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	
	authv1 "github.com/blackhole-pro/blackhole/core/internal/rpc/gen/identity/auth/v1"
)

// OAuthService implements the gRPC OAuthService
type OAuthService struct {
	auth.UnimplementedOAuthServiceServer
	oauthManager *OAuthManager
	rateLimiter  RateLimiter
}

// NewOAuthService creates a new OAuth service
func NewOAuthService(oauthManager *OAuthManager, rateLimiter RateLimiter) *OAuthService {
	return &OAuthService{
		oauthManager: oauthManager,
		rateLimiter:  rateLimiter,
	}
}

// GetOAuthURL generates an OAuth URL for the specified provider
func (s *OAuthService) GetOAuthURL(ctx context.Context, req *auth.OAuthURLRequest) (*auth.OAuthURLResponse, error) {
	// Apply rate limiting
	if s.rateLimiter != nil {
		clientIP, _ := getClientIP(ctx)
		
		// Rate limit by IP
		if !s.rateLimiter.AllowRequest(clientIP, "oauth_url") {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded for client IP")
		}
	}
	
	var url string
	var err error
	
	// Generate OAuth URL based on provider
	switch req.Provider {
	case "google":
		url, err = s.oauthManager.GetGoogleAuthURL(req.RedirectUri, req.State, req.Nonce)
	case "facebook":
		url, err = s.oauthManager.GetFacebookAuthURL(req.RedirectUri, req.State, req.Nonce)
	case "apple":
		url, err = s.oauthManager.GetAppleAuthURL(req.RedirectUri, req.State, req.Nonce)
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported provider")
	}
	
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	
	// Create response
	response := &auth.OAuthURLResponse{
		AuthUrl: url,
		State:   req.State,
		Expires: timestamppb.New(time.Now().Add(5 * time.Minute)), // URL expires in 5 minutes
	}
	
	return response, nil
}

// VerifyOAuthCallback verifies an OAuth callback
func (s *OAuthService) VerifyOAuthCallback(ctx context.Context, req *auth.OAuthCallbackRequest) (*auth.AuthResult, error) {
	// Apply rate limiting
	if s.rateLimiter != nil {
		clientIP, _ := getClientIP(ctx)
		
		// Rate limit by IP
		if !s.rateLimiter.AllowRequest(clientIP, "oauth_callback") {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded for client IP")
		}
	}
	
	var userData *OAuthUserData
	var err error
	
	// Verify callback based on provider
	switch req.Provider {
	case "google":
		userData, err = s.oauthManager.VerifyGoogleCallback(req.Code, req.RedirectUri)
	case "facebook":
		userData, err = s.oauthManager.VerifyFacebookCallback(req.Code, req.RedirectUri)
	case "apple":
		userData, err = s.oauthManager.VerifyAppleCallback(req.Code, req.RedirectUri)
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported provider")
	}
	
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	
	// Get or create DID for OAuth user
	did, err := s.oauthManager.GetOrCreateDIDForOAuthUser(ctx, req.Provider, userData.ID, userData)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	
	// Create authentication result
	result := &auth.AuthResult{
		Verified:                true,
		Did:                     did,
		VerificationMethodId:    did + "#oauth", // OAuth verification method
		AuthenticationDateTime:  timestamppb.New(time.Now()),
		PseudonymousIdentifier: req.Provider + ":" + userData.ID,
	}
	
	return result, nil
}

// RegisterOAuthService registers the OAuth service with a gRPC server
func RegisterOAuthService(server *grpc.Server, oauthManager *OAuthManager, rateLimiter RateLimiter) {
	service := NewOAuthService(oauthManager, rateLimiter)
	auth.RegisterOAuthServiceServer(server, service)
}