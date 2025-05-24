// Package resolver implements DID resolution functionality for the identity service.
package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	
	"github.com/blackhole-pro/blackhole/core/internal/services/identity/types"
)

// ipfsClient defines the interface for IPFS operations needed by the resolver
type ipfsClient interface {
	// Get retrieves content from IPFS
	Get(ctx context.Context, cid string, ownerDID string, name string, isPrivate bool) ([]byte, error)
}

// DIDResolver implements the types.DIDResolver interface using IPFS for storage
type DIDResolver struct {
	// ipfs is the IPFS client for content retrieval
	ipfs ipfsClient
}

// NewDIDResolver creates a new DIDResolver
func NewDIDResolver(ipfs ipfsClient) *DIDResolver {
	return &DIDResolver{
		ipfs: ipfs,
	}
}

// Resolve resolves a DID to its DID Document
func (r *DIDResolver) Resolve(ctx context.Context, did string) (*types.DIDDocument, error) {
	return r.ResolveWithOptions(ctx, did, &types.ResolveOptions{})
}

// ResolveWithOptions resolves a DID with privacy options
func (r *DIDResolver) ResolveWithOptions(ctx context.Context, did string, options *types.ResolveOptions) (*types.DIDDocument, error) {
	// Validate DID format
	if !isValidDID(did) {
		return nil, types.NewAuthError(types.ErrInvalidDIDFormat, did, "", "", "DID format is invalid")
	}
	
	// Normalize DID for path usage (replace colons with underscores in path but not in returned document)
	normalizedDID := normalizeDIDForPath(did)
	
	// Get DID Document from IPFS
	// The document is stored at /dids/[normalized_did]/document
	documentBytes, err := r.ipfs.Get(ctx, "", normalizedDID, "document", false)
	if err != nil {
		return nil, types.NewAuthError(types.ErrDIDResolutionFailed, did, "", "", fmt.Sprintf("failed to retrieve DID document: %v", err))
	}
	
	// Parse the DID Document
	var document types.DIDDocument
	if err := json.Unmarshal(documentBytes, &document); err != nil {
		return nil, types.NewAuthError(types.ErrDIDResolutionFailed, did, "", "", fmt.Sprintf("failed to parse DID document: %v", err))
	}
	
	// Apply privacy options if requested
	if options.MinimalDisclosure {
		document = applyMinimalDisclosure(document, options.RequestedVerificationMethod, options.IncludeService)
	}
	
	return &document, nil
}

// isValidDID validates the format of a DID
func isValidDID(did string) bool {
	// Basic validation: did:method:idstring
	parts := strings.Split(did, ":")
	return len(parts) >= 3 && parts[0] == "did"
}

// normalizeDIDForPath normalizes a DID for use in a file path
func normalizeDIDForPath(did string) string {
	// Replace colons with underscores for file path safety
	return strings.ReplaceAll(did, ":", "_")
}

// applyMinimalDisclosure creates a minimal version of the DID Document based on options
func applyMinimalDisclosure(doc types.DIDDocument, requestedMethod string, includeService bool) types.DIDDocument {
	// Create a new document with the same ID and context
	minimalDoc := types.DIDDocument{
		Context: doc.Context,
		ID:      doc.ID,
	}
	
	// Include controller if present
	if len(doc.Controller) > 0 {
		minimalDoc.Controller = doc.Controller
	}
	
	// If a specific verification method is requested, include only that one
	if requestedMethod != "" {
		for _, vm := range doc.VerificationMethod {
			if vm.ID == requestedMethod {
				minimalDoc.VerificationMethod = []types.VerificationMethod{vm}
				break
			}
		}
		
		// Include authentication reference if it matches
		for _, auth := range doc.Authentication {
			if auth == requestedMethod {
				minimalDoc.Authentication = []string{auth}
				break
			}
		}
	}
	
	// Otherwise include all authentication methods (but not other methods)
	if requestedMethod == "" {
		// Map to track which verification methods to include
		includeVM := make(map[string]bool)
		
		// Mark all authentication methods for inclusion
		for _, auth := range doc.Authentication {
			includeVM[auth] = true
		}
		
		// Include only the marked verification methods
		for _, vm := range doc.VerificationMethod {
			if includeVM[vm.ID] {
				minimalDoc.VerificationMethod = append(minimalDoc.VerificationMethod, vm)
			}
		}
		
		minimalDoc.Authentication = doc.Authentication
	}
	
	// Include services if requested
	if includeService {
		minimalDoc.Service = doc.Service
	}
	
	return minimalDoc
}