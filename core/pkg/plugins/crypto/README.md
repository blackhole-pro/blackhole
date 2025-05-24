# Crypto Plugin

## Purpose

The crypto plugin provides **cryptographic primitives** as a service to other plugins and applications.

## Why Separate from Identity?

1. **Used by many plugins**: Storage needs encryption, Node needs signatures, Analytics needs hashing
2. **Different security model**: Crypto operations are stateless, identity is stateful
3. **Different update cycle**: Crypto algorithms change less frequently than auth flows
4. **Performance isolation**: Heavy crypto operations don't impact identity performance

## What It Provides

```protobuf
service CryptoPlugin {
  // Hashing
  rpc Hash(HashRequest) returns (HashResponse);
  rpc VerifyHash(VerifyHashRequest) returns (VerifyHashResponse);
  
  // Symmetric Encryption
  rpc Encrypt(EncryptRequest) returns (EncryptResponse);
  rpc Decrypt(DecryptRequest) returns (DecryptResponse);
  
  // Digital Signatures
  rpc Sign(SignRequest) returns (SignResponse);
  rpc Verify(VerifyRequest) returns (VerifyResponse);
  
  // Key Derivation
  rpc DeriveKey(DeriveKeyRequest) returns (DeriveKeyResponse);
  
  // Random Generation
  rpc GenerateRandom(GenerateRandomRequest) returns (GenerateRandomResponse);
}
```

## Who Uses It

- **Identity Plugin**: Password hashing, token signing
- **Storage Plugin**: File encryption/decryption
- **Node Plugin**: Message signatures, peer verification
- **Analytics Plugin**: Data anonymization, hashing

## Example Usage

```go
// Identity plugin using crypto for password hashing
func (id *IdentityPlugin) hashPassword(password string) (string, error) {
    resp, err := id.cryptoClient.Hash(ctx, &HashRequest{
        Data:      []byte(password),
        Algorithm: "argon2id",
        Options: map[string]string{
            "memory": "64MB",
            "time":   "3",
        },
    })
    return resp.Hash, err
}
```