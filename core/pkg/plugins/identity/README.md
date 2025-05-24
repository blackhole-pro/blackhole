# Identity Plugin Architecture

## Design Philosophy: Cohesive Domain Plugin

The identity plugin is designed as a **cohesive domain plugin** that includes all closely-related identity functionality while keeping clear boundaries.

## What's Included in Identity Plugin

### Core Identity Operations ✓
- User registration and profile management
- Authentication (password, MFA, OAuth, WebAuthn)
- Session management
- Token generation and validation (JWT, refresh tokens)
- Permission and role management

### Why These Belong Together
- **Security boundary**: All identity operations need consistent security
- **Shared state**: Sessions, tokens, and auth state are interconnected
- **Atomic operations**: Login creates session + token in one transaction
- **Performance**: Avoid mesh overhead for tightly-coupled operations

## What's NOT in Identity Plugin

### Cryptographic Operations ✗
- **Separate plugin**: `crypto-plugin`
- **Why separate**: Many plugins need crypto (storage encryption, node signatures)
- **Interface**: Identity plugin uses crypto plugin via mesh

### Key Management ✗
- **Separate plugin**: `keystore-plugin` 
- **Why separate**: System-wide key management (TLS certs, encryption keys)
- **Interface**: Identity plugin stores user keys in keystore

### Distributed Identity (DID) ✗
- **Separate plugin**: `did-plugin`
- **Why separate**: Blockchain/DID is optional, complex subsystem
- **Interface**: Identity plugin can optionally integrate with DID

## Architecture Example

```
┌─────────────────────────────────────────────────────┐
│                 Identity Plugin                      │
│                                                      │
│  ┌─────────────┐  ┌──────────┐  ┌────────────────┐ │
│  │    Auth     │  │ Sessions │  │     Tokens     │ │
│  │             │  │          │  │                │ │
│  │ • Password  │  │ • Create │  │ • JWT Generate │ │
│  │ • OAuth     │  │ • Store  │  │ • Validate     │ │
│  │ • WebAuthn  │  │ • Revoke │  │ • Refresh      │ │
│  └──────┬──────┘  └─────┬────┘  └───────┬────────┘ │
│         │               │                │          │
│         └───────────────┴────────────────┘          │
│                         │                           │
│                    Central State                    │
│                  (User Database)                    │
└─────────────────────────┬───────────────────────────┘
                          │
                     Mesh Network
                          │
         ┌────────────────┼────────────────┐
         ▼                ▼                ▼
    Crypto Plugin    Keystore Plugin   DID Plugin
    (Encryption)     (Key Storage)    (Blockchain)
```

## Interface Design

```protobuf
service IdentityPlugin {
  // User Management
  rpc RegisterUser(RegisterUserRequest) returns (RegisterUserResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
  
  // Authentication
  rpc Authenticate(AuthenticateRequest) returns (AuthenticateResponse);
  rpc VerifyMFA(VerifyMFARequest) returns (VerifyMFAResponse);
  
  // Session Management
  rpc CreateSession(CreateSessionRequest) returns (CreateSessionResponse);
  rpc ValidateSession(ValidateSessionRequest) returns (ValidateSessionResponse);
  rpc RevokeSession(RevokeSessionRequest) returns (RevokeSessionResponse);
  
  // Token Operations
  rpc GenerateToken(GenerateTokenRequest) returns (GenerateTokenResponse);
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);
  
  // Authorization
  rpc CheckPermission(CheckPermissionRequest) returns (CheckPermissionResponse);
  rpc GrantRole(GrantRoleRequest) returns (GrantRoleResponse);
}
```

## When to Split a Plugin

Split when:
1. **Different applications need different subsets** (crypto is needed by many)
2. **Different update cycles** (crypto algorithms vs auth flows)
3. **Different security boundaries** (public crypto vs private keys)
4. **Different performance requirements** (heavy crypto vs light auth checks)

Keep together when:
1. **Operations are tightly coupled** (login creates session + token)
2. **Shared state is required** (user → session → token)
3. **Atomic operations needed** (auth + session in one transaction)
4. **Security boundary is the same** (all identity operations)

## Benefits of This Approach

1. **Right-sized plugins**: Not too big, not too small
2. **Clear boundaries**: Based on domain cohesion
3. **Performance**: Tightly-coupled operations avoid mesh overhead
4. **Flexibility**: Apps can still choose which plugins to load
5. **Maintainability**: Related code stays together