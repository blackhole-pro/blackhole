// +build ignore

package resolver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	commonv1 "github.com/blackhole-pro/blackhole/core/internal/rpc/gen/common/v1"
	// storagev1 "github.com/blackhole-pro/blackhole/core/internal/rpc/gen/storage/v1" // TODO: Uncomment when storage service is implemented
	"github.com/blackhole-pro/blackhole/core/internal/services/identity/types"
)

// Common errors
var (
	ErrDIDNotFound       = errors.New("DID document not found")
	ErrInvalidDID        = errors.New("invalid DID format")
	ErrConnectionFailed  = errors.New("failed to connect to storage service")
	ErrRetrievalFailed   = errors.New("failed to retrieve DID document")
	ErrMethodNotFound    = errors.New("verification method not found")
	ErrInvalidDocument   = errors.New("invalid DID document")
	ErrServiceUnavailable = errors.New("DID document service unavailable")
)

// GRPCDIDResolver implements the DIDResolver interface for resolving DID documents via gRPC
type GRPCDIDResolver struct {
	storageServiceAddress string
	conn                  *grpc.ClientConn
	client                storagev1.DIDDocumentServiceClient
}

// NewGRPCDIDResolver creates a new DID resolver that uses gRPC to communicate with the storage service
func NewGRPCDIDResolver(storageServiceAddress string) (*GRPCDIDResolver, error) {
	resolver := &GRPCDIDResolver{
		storageServiceAddress: storageServiceAddress,
	}
	
	// Connect to the storage service
	err := resolver.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to storage service: %w", err)
	}
	
	return resolver, nil
}

// connect establishes a connection to the storage service
func (r *GRPCDIDResolver) connect() error {
	var err error
	
	// Set up a connection to the storage service
	r.conn, err = grpc.Dial(r.storageServiceAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to storage service: %w", err)
	}
	
	// Create a client for the DID document service
	r.client = storagev1.NewDIDDocumentServiceClient(r.conn)
	
	return nil
}

// Close closes the connection to the storage service
func (r *GRPCDIDResolver) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// Resolve resolves a DID to its DID Document
func (r *GRPCDIDResolver) Resolve(ctx context.Context, did string) (*types.DIDDocument, error) {
	return r.ResolveWithOptions(ctx, did, &types.ResolveOptions{})
}

// ResolveWithOptions resolves a DID with privacy options
func (r *GRPCDIDResolver) ResolveWithOptions(ctx context.Context, did string, options *types.ResolveOptions) (*types.DIDDocument, error) {
	// Convert options to ResolutionOptions
	resOptions := &types.ResolutionOptions{
		ProtectionLevel: types.ProtectionStandard,
		AcceptCache: true,
	}
	
	if options != nil {
		if options.MinimalDisclosure {
			resOptions.ProtectionLevel = types.ProtectionEnhanced
		}
	}
	
	// Use ResolveDID to get the document
	document, err := r.ResolveDID(ctx, did, resOptions)
	if err != nil {
		return nil, err
	}
	
	// Convert to the legacy DIDDocument format
	legacyDoc := convertToLegacyDocument(document, options)
	
	return legacyDoc, nil
}

// ResolveDID resolves a DID to a DID document
func (r *GRPCDIDResolver) ResolveDID(ctx context.Context, did string, options *types.ResolutionOptions) (*types.DIDDocument, error) {
	// Validate DID format
	if err := validateDID(did); err != nil {
		return nil, err
	}
	
	// Ensure we have a connection to the storage service
	if r.client == nil {
		if err := r.connect(); err != nil {
			return nil, ErrConnectionFailed
		}
	}
	
	// Create request with authentication info if available
	authInfo := &commonv1.AuthInfo{}
	if options != nil && options.AuthInfo != nil {
		authInfo = convertAuthInfo(options.AuthInfo)
	}
	
	// Call the storage service to get the DID document
	req := &storagev1.GetDIDDocumentRequest{
		Did:      did,
		AuthInfo: authInfo,
		// Pass version ID if specified in options
		VersionId: getVersionID(options),
	}
	
	// Set a timeout for the request
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	// Call the storage service
	resp, err := r.client.GetDIDDocument(timeoutCtx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRetrievalFailed, err)
	}
	
	// Check if the operation was successful
	if !resp.Success {
		if resp.Error != nil {
			return nil, fmt.Errorf("%w: %s", ErrRetrievalFailed, resp.Error.Message)
		}
		return nil, ErrRetrievalFailed
	}
	
	// Convert the gRPC DID document to our internal representation
	document, err := convertDIDDocument(resp.Document)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidDocument, err)
	}
	
	return document, nil
}

// GetVerificationMethod retrieves a specific verification method from a DID document
func (r *GRPCDIDResolver) GetVerificationMethod(ctx context.Context, did string, id string, options *types.ResolutionOptions) (*types.VerificationMethod, error) {
	// Resolve the DID document
	doc, err := r.ResolveDID(ctx, did, options)
	if err != nil {
		return nil, err
	}
	
	// Find the specified verification method
	for _, method := range doc.VerificationMethods {
		if method.ID == id {
			return method, nil
		}
	}
	
	return nil, ErrMethodNotFound
}

// DIDExists checks if a DID document exists
func (r *GRPCDIDResolver) DIDExists(ctx context.Context, did string) (bool, error) {
	// Validate DID format
	if err := validateDID(did); err != nil {
		return false, err
	}
	
	// Ensure we have a connection to the storage service
	if r.client == nil {
		if err := r.connect(); err != nil {
			return false, ErrConnectionFailed
		}
	}
	
	// Call the storage service to check if the DID document exists
	req := &storagev1.DIDDocumentExistsRequest{
		Did: did,
	}
	
	// Set a timeout for the request
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	// Call the storage service
	resp, err := r.client.DIDDocumentExists(timeoutCtx, req)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}
	
	// Check if the operation was successful
	if !resp.Success {
		if resp.Error != nil {
			return false, fmt.Errorf("%w: %s", ErrServiceUnavailable, resp.Error.Message)
		}
		return false, ErrServiceUnavailable
	}
	
	return resp.Exists, nil
}

// Helper functions

// validateDID validates the format of a DID
func validateDID(did string) error {
	// Basic validation for now - in a real implementation this would be more robust
	if did == "" {
		return ErrInvalidDID
	}
	
	// Must start with "did:"
	if len(did) < 4 || did[:4] != "did:" {
		return ErrInvalidDID
	}
	
	return nil
}

// getVersionID gets the version ID from the resolution options
func getVersionID(options *types.ResolutionOptions) string {
	if options == nil || options.VersionID == "" {
		return ""
	}
	return options.VersionID
}

// convertAuthInfo converts our internal auth info to the gRPC auth info
func convertAuthInfo(authInfo *types.AuthInfo) *commonv1.AuthInfo {
	if authInfo == nil {
		return &commonv1.AuthInfo{}
	}
	
	// Convert protection level
	protectionLevel := commonv1.AuthInfo_PROTECTION_LEVEL_UNSPECIFIED
	switch authInfo.ProtectionLevel {
	case types.ProtectionMinimal:
		protectionLevel = commonv1.AuthInfo_PROTECTION_LEVEL_MINIMAL
	case types.ProtectionStandard:
		protectionLevel = commonv1.AuthInfo_PROTECTION_LEVEL_STANDARD
	case types.ProtectionEnhanced:
		protectionLevel = commonv1.AuthInfo_PROTECTION_LEVEL_ENHANCED
	case types.ProtectionMaximum:
		protectionLevel = commonv1.AuthInfo_PROTECTION_LEVEL_MAXIMUM
	}
	
	// Convert to gRPC auth info
	return &commonv1.AuthInfo{
		Did:                    authInfo.DID,
		AuthToken:              authInfo.AuthToken,
		VerificationMethodId:   authInfo.VerificationMethodID,
		AuthenticationTime:     authInfo.AuthenticationTime,
		PseudonymousIdentifier: authInfo.PseudonymousIdentifier,
		AuthProvider:           authInfo.AuthProvider,
		OauthProvider:          authInfo.OAuthProvider,
		ProtectionLevel:        protectionLevel,
		Metadata:               authInfo.Metadata,
	}
}

// convertDIDDocument converts a gRPC DID document to our internal representation
func convertDIDDocument(doc *storagev1.DIDDocument) (*types.DIDDocument, error) {
	if doc == nil {
		return nil, errors.New("nil document")
	}
	
	// Convert verification methods
	verificationMethods := make([]*types.VerificationMethod, 0, len(doc.VerificationMethods))
	for _, method := range doc.VerificationMethods {
		verificationMethod := &types.VerificationMethod{
			ID:                  method.Id,
			Type:                convertVerificationMethodType(method.Type),
			Controller:          method.Controller,
			PublicKeyMultibase:  method.PublicKeyMultibase,
		}
		verificationMethods = append(verificationMethods, verificationMethod)
	}
	
	// Convert services
	services := make([]*types.Service, 0, len(doc.Services))
	for _, service := range doc.Services {
		svc := &types.Service{
			ID:              service.Id,
			Type:            service.Type,
			ServiceEndpoint: service.ServiceEndpoint,
			Properties:      service.Properties,
		}
		services = append(services, svc)
	}
	
	// Create the document
	return &types.DIDDocument{
		DID:                 doc.Did,
		Controllers:         doc.Controllers,
		AlsoKnownAs:         doc.AlsoKnownAs,
		VerificationMethods: verificationMethods,
		Authentication:      doc.Authentication,
		AssertionMethod:     doc.AssertionMethod,
		KeyAgreement:        doc.KeyAgreement,
		CapabilityInvocation: doc.CapabilityInvocation,
		CapabilityDelegation: doc.CapabilityDelegation,
		Services:            services,
		Created:             doc.Created,
		Updated:             doc.Updated,
		VersionID:           doc.VersionId,
		Properties:          doc.Properties,
	}, nil
}

// convertVerificationMethodType converts a gRPC verification method type to our internal type
func convertVerificationMethodType(methodType storagev1.VerificationMethodType) string {
	switch methodType {
	case storagev1.VerificationMethodType_VERIFICATION_METHOD_TYPE_ED25519:
		return "Ed25519VerificationKey2020"
	case storagev1.VerificationMethodType_VERIFICATION_METHOD_TYPE_SECP256K1:
		return "EcdsaSecp256k1VerificationKey2019"
	case storagev1.VerificationMethodType_VERIFICATION_METHOD_TYPE_RSA:
		return "RsaVerificationKey2018"
	case storagev1.VerificationMethodType_VERIFICATION_METHOD_TYPE_X25519:
		return "X25519KeyAgreementKey2020"
	default:
		return "UnknownVerificationKey"
	}
}

// convertToLegacyDocument converts new DIDDocument structure to the legacy format
func convertToLegacyDocument(doc *types.DIDDocument, options *types.ResolveOptions) *types.DIDDocument {
	// Legacy format uses a different structure, but for now we'll just return
	// the document as is and let the legacy code handle it
	
	// In a real implementation, this would convert between document formats
	
	return doc
}