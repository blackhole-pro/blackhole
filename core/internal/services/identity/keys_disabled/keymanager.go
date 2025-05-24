// Package keys implements key management functionality for the identity service.
package keys

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
	
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/multiformats/go-multibase"
	
	"github.com/blackhole-pro/blackhole/core/internal/services/identity/types"
)

// KeyManager implements the types.KeyManager interface
type KeyManager struct {
	resolver types.DIDResolver
}

// NewKeyManager creates a new KeyManager
func NewKeyManager(resolver types.DIDResolver) *KeyManager {
	return &KeyManager{
		resolver: resolver,
	}
}

// Verify verifies a signature using the appropriate verification method from a DID
func (k *KeyManager) Verify(ctx context.Context, did string, verificationMethodID string, message []byte, signature []byte) (bool, error) {
	// Resolve the DID document
	doc, err := k.resolver.Resolve(ctx, did)
	if err != nil {
		return false, types.NewAuthError(types.ErrDIDResolutionFailed, did, "", verificationMethodID, "failed to resolve DID for verification")
	}
	
	// Find the verification method
	var method *types.VerificationMethod
	for _, m := range doc.VerificationMethod {
		if m.ID == verificationMethodID {
			method = &m
			break
		}
	}
	
	if method == nil {
		return false, types.NewAuthError(types.ErrVerificationMethodNotFound, did, "", verificationMethodID, "verification method not found in DID document")
	}
	
	// Verify based on the verification method type
	switch method.Type {
	case "Ed25519VerificationKey2020":
		return verifyEd25519(method, message, signature)
	case "EcdsaSecp256k1VerificationKey2019":
		return verifySecp256k1(method, message, signature)
	default:
		return false, types.NewAuthError(types.ErrUnsupportedSignatureType, did, "", verificationMethodID, fmt.Sprintf("unsupported signature type: %s", method.Type))
	}
}

// GetVerificationMethods returns available verification methods for a DID
func (k *KeyManager) GetVerificationMethods(ctx context.Context, did string) ([]types.VerificationMethod, error) {
	// Resolve the DID document
	doc, err := k.resolver.Resolve(ctx, did)
	if err != nil {
		return nil, types.NewAuthError(types.ErrDIDResolutionFailed, did, "", "", "failed to resolve DID for verification methods")
	}
	
	// Find authentication verification methods
	authMethods := make(map[string]bool)
	for _, auth := range doc.Authentication {
		authMethods[auth] = true
	}
	
	// Filter to include only authentication methods
	var methods []types.VerificationMethod
	for _, method := range doc.VerificationMethod {
		if authMethods[method.ID] {
			methods = append(methods, method)
		}
	}
	
	if len(methods) == 0 {
		return nil, types.NewAuthError(types.ErrVerificationMethodNotFound, did, "", "", "no authentication verification methods found in DID document")
	}
	
	return methods, nil
}

// verifyEd25519 verifies an Ed25519 signature
func verifyEd25519(method *types.VerificationMethod, message []byte, signature []byte) (bool, error) {
	var publicKey ed25519.PublicKey
	
	// Extract the public key based on available format
	if method.PublicKeyMultibase != "" {
		// Decode the multibase-encoded public key
		_, decoded, err := multibase.Decode(method.PublicKeyMultibase)
		if err != nil {
			return false, fmt.Errorf("failed to decode multibase public key: %w", err)
		}
		publicKey = ed25519.PublicKey(decoded)
	} else if method.PublicKeyJwk != nil {
		// Extract x and y coordinates from JWK
		x, ok := method.PublicKeyJwk["x"].(string)
		if !ok {
			return false, fmt.Errorf("missing or invalid 'x' coordinate in JWK")
		}
		
		// Decode base64url-encoded coordinate
		decoded, err := base64.RawURLEncoding.DecodeString(x)
		if err != nil {
			return false, fmt.Errorf("failed to decode JWK coordinate: %w", err)
		}
		
		publicKey = ed25519.PublicKey(decoded)
	} else {
		return false, fmt.Errorf("no public key data available in verification method")
	}
	
	// Verify the signature
	return ed25519.Verify(publicKey, message, signature), nil
}

// verifySecp256k1 verifies a secp256k1 ECDSA signature
func verifySecp256k1(method *types.VerificationMethod, message []byte, signature []byte) (bool, error) {
	var publicKeyBytes []byte
	
	// Extract the public key based on available format
	if method.PublicKeyMultibase != "" {
		// Decode the multibase-encoded public key
		_, decoded, err := multibase.Decode(method.PublicKeyMultibase)
		if err != nil {
			return false, fmt.Errorf("failed to decode multibase public key: %w", err)
		}
		publicKeyBytes = decoded
	} else if method.PublicKeyJwk != nil {
		// Extract x and y coordinates from JWK
		x, ok := method.PublicKeyJwk["x"].(string)
		if !ok {
			return false, fmt.Errorf("missing or invalid 'x' coordinate in JWK")
		}
		
		y, ok := method.PublicKeyJwk["y"].(string)
		if !ok {
			return false, fmt.Errorf("missing or invalid 'y' coordinate in JWK")
		}
		
		// Decode base64url-encoded coordinates
		xBytes, err := base64.RawURLEncoding.DecodeString(x)
		if err != nil {
			return false, fmt.Errorf("failed to decode JWK x coordinate: %w", err)
		}
		
		yBytes, err := base64.RawURLEncoding.DecodeString(y)
		if err != nil {
			return false, fmt.Errorf("failed to decode JWK y coordinate: %w", err)
		}
		
		// Combine coordinates into uncompressed public key format (0x04 | x | y)
		publicKeyBytes = make([]byte, 65)
		publicKeyBytes[0] = 0x04
		copy(publicKeyBytes[1:33], xBytes)
		copy(publicKeyBytes[33:], yBytes)
	} else {
		return false, fmt.Errorf("no public key data available in verification method")
	}
	
	// Hash the message (Ethereum style)
	messageHash := crypto.Keccak256Hash(message)
	
	// Verify the signature
	return crypto.VerifySignature(publicKeyBytes, messageHash.Bytes(), signature[:len(signature)-1]), nil
}

// GetPublicKeyFromMethod extracts the public key from a verification method
func GetPublicKeyFromMethod(method *types.VerificationMethod) ([]byte, error) {
	if method.PublicKeyMultibase != "" {
		_, decoded, err := multibase.Decode(method.PublicKeyMultibase)
		if err != nil {
			return nil, fmt.Errorf("failed to decode multibase public key: %w", err)
		}
		return decoded, nil
	} else if method.PublicKeyJwk != nil {
		if method.Type == "Ed25519VerificationKey2020" {
			x, ok := method.PublicKeyJwk["x"].(string)
			if !ok {
				return nil, fmt.Errorf("missing or invalid 'x' coordinate in JWK")
			}
			
			decoded, err := base64.RawURLEncoding.DecodeString(x)
			if err != nil {
				return nil, fmt.Errorf("failed to decode JWK coordinate: %w", err)
			}
			
			return decoded, nil
		} else if method.Type == "EcdsaSecp256k1VerificationKey2019" {
			x, ok := method.PublicKeyJwk["x"].(string)
			if !ok {
				return nil, fmt.Errorf("missing or invalid 'x' coordinate in JWK")
			}
			
			y, ok := method.PublicKeyJwk["y"].(string)
			if !ok {
				return nil, fmt.Errorf("missing or invalid 'y' coordinate in JWK")
			}
			
			xBytes, err := base64.RawURLEncoding.DecodeString(x)
			if err != nil {
				return nil, fmt.Errorf("failed to decode JWK x coordinate: %w", err)
			}
			
			yBytes, err := base64.RawURLEncoding.DecodeString(y)
			if err != nil {
				return nil, fmt.Errorf("failed to decode JWK y coordinate: %w", err)
			}
			
			publicKeyBytes := make([]byte, 65)
			publicKeyBytes[0] = 0x04
			copy(publicKeyBytes[1:33], xBytes)
			copy(publicKeyBytes[33:], yBytes)
			
			return publicKeyBytes, nil
		}
	}
	
	return nil, fmt.Errorf("no public key data available in verification method")
}