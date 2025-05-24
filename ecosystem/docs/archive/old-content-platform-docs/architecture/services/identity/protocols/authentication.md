# Blackhole Authentication Protocols

This document outlines the authentication protocols and flows within the Blackhole platform, providing a comprehensive framework for secure, decentralized authentication based on DIDs and verifiable credentials.

## Introduction

Authentication in Blackhole is built upon the principles of self-sovereign identity, leveraging DIDs and verifiable credentials to provide secure, privacy-preserving authentication without reliance on traditional centralized identity providers. The authentication system enables flexible authentication mechanisms across various contexts while maintaining user control and minimizing data exposure.

## Design Principles

1. **User Sovereignty**: Users control their identities and authentication factors
2. **Privacy By Design**: Minimal disclosure of information during authentication
3. **Contextual Authentication**: Security levels appropriate to the context
4. **Cross-Platform Consistency**: Uniform experience across devices and applications
5. **Standards Compliance**: Adoption of industry standards for interoperability
6. **Flexibility**: Support for multiple authentication methods and factors
7. **Resilience**: Maintenance of authentication capabilities during network disruptions
8. **Revocability**: Ability to revoke authentication capabilities when compromised

## Core Components

### 1. Authentication Protocol Suite

- **Basic Challenge-Response**: Simple cryptographic proof of DID control
- **Enhanced Authentication**: Multi-factor authentication with elevated assurance
- **Continuous Authentication**: Session maintenance with periodic re-verification
- **Delegated Authentication**: Acting on behalf of other identities with proper authorization
- **Zero-Knowledge Authentication**: Proving attributes without revealing values

### 2. Credential-Based Authentication

- **Credential Verification**: Authentication based on credential possession
- **Trust Framework**: Rules for accepting credentials from specific issuers
- **Attribute-Based Access**: Authentication based on specific credential attributes
- **Anonymous Credentials**: Authentication without disclosing identity
- **Revocation Checking**: Verification of credential validity at authentication time

### 3. Session Management

- **Session Establishment**: Secure session creation after successful authentication
- **Session Security**: Protection against hijacking and replay attacks
- **Session Persistence**: Appropriate timeouts and refresh mechanisms
- **Cross-Device Sessions**: Session mobility across user devices
- **Session Metadata**: Contextual information about active sessions

### 4. Authorization Framework

- **Permission Model**: Granular permissions system
- **Consent Management**: Explicit user consent for information sharing
- **Capability Delegation**: Secure delegation of specific capabilities
- **Resource Authorization**: Control over specific resource access
- **Revocation Mechanisms**: Immediate revocation of granted permissions

### 5. Authentication API

- **Service Integration**: Easy integration for service providers
- **Client Libraries**: Simplified client-side authentication libraries
- **Security Best Practices**: Enforced through API design
- **Protocol Negotiation**: Dynamic selection of appropriate authentication methods
- **Error Handling**: Secure error reporting without information leakage

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│                Client Applications                      │
│                                                         │
└───────────────────────┬─────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────┐
│                                                         │
│              Authentication Gateway                     │
│                                                         │
│  ┌──────────────┐ ┌───────────────┐ ┌───────────────┐  │
│  │  Protocol    │ │    Session    │ │  Permission   │  │
│  │  Handler     │ │    Manager    │ │    Manager    │  │
│  └──────────────┘ └───────────────┘ └───────────────┘  │
│                                                         │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                                                         │
│             Authentication Services                     │
│                                                         │
│  ┌──────────────┐ ┌───────────────┐ ┌───────────────┐  │
│  │  Challenge   │ │  Credential   │ │  Signature    │  │
│  │  Generator   │ │  Verifier     │ │  Validator    │  │
│  └──────────────┘ └───────────────┘ └───────────────┘  │
│                                                         │
│  ┌──────────────┐ ┌───────────────┐ ┌───────────────┐  │
│  │   Token      │ │  Multi-Factor │ │    Audit      │  │
│  │   Service    │ │  Service      │ │    Logger     │  │
│  └──────────────┘ └───────────────┘ └───────────────┘  │
│                                                         │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                                                         │
│                 Identity Systems                        │
│                                                         │
│  ┌──────────────┐ ┌───────────────┐ ┌───────────────┐  │
│  │     DID      │ │   Credential  │ │      Key      │  │
│  │   Resolver   │ │    System     │ │  Management   │  │
│  └──────────────┘ └───────────────┘ └───────────────┘  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Authentication Protocols

### 1. DID-Auth Challenge-Response Protocol

The foundation of Blackhole authentication is a challenge-response protocol based on DIDs:

#### Protocol Flow

1. **Initiation**:
   - Service generates a random challenge
   - Service specifies required authentication parameters
   - Challenge request is sent to user's wallet

2. **Challenge Processing**:
   - Wallet displays authentication request to user
   - User consents to authentication and selects identity
   - Wallet signs challenge with appropriate verification method
   - Optional: Include credential presentations as required

3. **Response Verification**:
   - Service receives signed response
   - Service resolves user's DID document
   - Signature is verified using specified verification method
   - Optional: Credentials are verified if presented
   - Authentication result is determined

4. **Session Establishment**:
   - Upon successful verification, session is established
   - Authentication token is issued to client
   - Session parameters and permissions are set
   - Response is sent to client application

#### Protocol Data Format

```json
// Authentication Request
{
  "type": "AuthenticationRequest",
  "version": "1.0",
  "id": "auth-req-123456789",
  "created": "2023-08-15T14:30:45Z",
  "expires": "2023-08-15T14:35:45Z",
  "callbackUrl": "https://service.example.com/auth-response",
  "challenge": "a0e2e3b4c5d6e7f8a9b0c1d2e3f4a5b6",
  "domain": "service.example.com",
  "requiredProofs": {
    "identityLevel": "basic", // basic, enhanced, or strong
    "authentication": ["did:blackhole"],
    "presentationRequirements": [
      {
        "id": "account-verification",
        "purpose": "Verify account status",
        "optional": true,
        "credentialTypes": ["BlackholeAccountCredential"],
        "constraints": {
          "fields": [
            {
              "path": ["$.credentialSubject.accountStatus"],
              "filter": {
                "type": "string",
                "pattern": "active"
              }
            }
          ]
        }
      }
    ]
  }
}

// Authentication Response
{
  "type": "AuthenticationResponse",
  "version": "1.0",
  "id": "auth-resp-987654321",
  "requestId": "auth-req-123456789",
  "created": "2023-08-15T14:31:30Z",
  "subject": "did:blackhole:123456789abcdef",
  "challenge": "a0e2e3b4c5d6e7f8a9b0c1d2e3f4a5b6",
  "domain": "service.example.com",
  "proof": {
    "type": "Ed25519Signature2020",
    "created": "2023-08-15T14:31:30Z",
    "verificationMethod": "did:blackhole:123456789abcdef#key-1",
    "proofPurpose": "authentication",
    "challenge": "a0e2e3b4c5d6e7f8a9b0c1d2e3f4a5b6",
    "domain": "service.example.com",
    "proofValue": "z58DAdFfa9xyVVuJ..."
  },
  "presentations": [
    {
      // Verifiable Presentation containing requested credentials
    }
  ]
}
```

### 2. Credential-Based Authentication Protocol

For scenarios requiring more context than simple DID control, the platform supports credential-based authentication:

#### Protocol Flow

1. **Request Generation**:
   - Service identifies required credential types and attributes
   - Service generates authentication request with credential requirements
   - Request is sent to user's wallet

2. **Credential Selection**:
   - Wallet identifies matching credentials
   - User selects appropriate credentials to share
   - Wallet prepares presentation with minimal disclosure
   - Challenge is signed with appropriate verification method

3. **Verification Process**:
   - Service verifies presentation signature
   - Credential issuer DIDs are resolved and verified
   - Credential status is checked for revocation
   - Attribute values are validated against requirements
   - Trust framework rules are applied to issuer acceptance

4. **Authorization Establishment**:
   - Upon successful verification, user identity and attributes are established
   - Authorization context is created based on verified attributes
   - Permission grants are derived from credential claims
   - Session with appropriate permissions is established

### 3. Zero-Knowledge Authentication Protocol

For maximum privacy, Blackhole supports authentication without revealing specific values:

#### Protocol Flow

1. **ZKP Preparation**:
   - Service requests proof of specific attributes without values
   - Requirements are specified in zero-knowledge format
   - Challenge incorporates the proof requirements

2. **Proof Generation**:
   - Wallet identifies credentials containing required attributes
   - Zero-knowledge proofs are generated to prove attribute properties
   - Proofs are combined into a presentation with minimal disclosure
   - ZKP presentation is signed and sent to service

3. **Proof Verification**:
   - Service verifies the mathematical validity of ZKPs
   - Service confirms proofs satisfy required attribute constraints
   - No actual values are revealed, only their properties
   - Verification result determines authentication success

4. **Anonymous Session**:
   - Upon success, pseudonymous session is established
   - Permissions are based on proven attributes
   - Session maintains minimal correlation with user identity
   - Service tracks only necessary attributes for functionality

## Authentication Levels

The Blackhole authentication system supports multiple levels of authentication assurance:

### 1. Basic Authentication

- **Purpose**: Low-risk operations and basic identity association
- **Requirements**: DID control verification via single factor
- **Use Cases**: Content browsing, public interactions, low-value transactions
- **Implementation**: Simple challenge-response with DID signature

### 2. Standard Authentication

- **Purpose**: General platform operations with personal account access
- **Requirements**: DID control plus account verification
- **Use Cases**: Content creation, social interactions, profile management
- **Implementation**: Challenge-response with account credential verification

### 3. Enhanced Authentication

- **Purpose**: Higher-value operations and sensitive data access
- **Requirements**: Multi-factor authentication with registered factors
- **Use Cases**: Financial transactions, private content access, account settings
- **Implementation**: Multiple verification methods plus credential verification

### 4. Strong Authentication

- **Purpose**: Critical operations with highest security requirements
- **Requirements**: Multiple factors across different channels with biometrics
- **Use Cases**: Key recovery, high-value transfers, security setting changes
- **Implementation**: Multi-factor, multi-channel verification with timestamped audit

## Session Management

### Session Types

1. **Ephemeral Sessions**:
   - Short-lived with minimal persistence
   - Limited permission scope
   - Automatically expire after short period
   - No refresh capabilities

2. **Standard Sessions**:
   - Regular user sessions for normal operations
   - Moderate timeout periods
   - Refresh token capabilities
   - Limited to originating device

3. **Extended Sessions**:
   - Longer duration for trusted devices
   - Device registration required
   - Periodic step-up verification
   - Revocable from user dashboard

4. **Cross-Device Sessions**:
   - Accessible from multiple authenticated devices
   - Secure session state synchronization
   - Device-specific capabilities and restrictions
   - Full activity history accessible to user

### Session Security Features

1. **Token Security**:
   - Short-lived access tokens
   - Cryptographically signed tokens
   - Context-bound usage constraints
   - Hardware-backed security where available

2. **Context Validation**:
   - Device fingerprinting for anomaly detection
   - Location awareness for unusual access patterns
   - Activity pattern monitoring
   - Risk-based adaptive authentication

3. **Session Monitoring**:
   - Real-time session activity tracking
   - User-accessible session dashboard
   - Unusual activity notifications
   - Emergency session termination capabilities

## Cross-Platform Authentication

The authentication system provides consistent experiences across platforms while leveraging platform-specific security capabilities:

### Web Platforms

- **Authentication Flow**: Redirect or popup-based flows
- **Session Storage**: Secure cookies and local storage
- **Security Features**: CSP, SameSite cookies, anti-CSRF measures
- **Integration**: JavaScript SDK with security best practices

### Mobile Applications

- **Authentication Flow**: Native app deep linking
- **Session Storage**: Secure enclave / keystore integration
- **Security Features**: Biometric integration, app attestation
- **Integration**: Native SDKs for iOS and Android

### Desktop Applications

- **Authentication Flow**: Custom protocol handlers
- **Session Storage**: OS keychain integration
- **Security Features**: Hardware security integration where available
- **Integration**: Electron and native application SDKs

## Implementation Structure

The authentication system follows the standardized project structure with clear separation between node implementation, client SDK, and shared components:

### Node Implementation

```
@blackhole/node/
├── services/
│   └── identity/
│       └── authentication/          # Authentication service
│           ├── protocols/           # Authentication protocol implementations
│           │   ├── did-auth.ts      # DID-Auth challenge-response
│           │   ├── credential-auth.ts # Credential-based authentication
│           │   ├── zkp-auth.ts      # Zero-knowledge authentication
│           │   └── multi-factor.ts  # Multi-factor authentication
│           │
│           ├── sessions/            # Session management
│           │   ├── manager.ts       # Session CRUD operations
│           │   ├── tokens.ts        # Token generation and validation
│           │   ├── storage.ts       # Secure session storage
│           │   └── monitor.ts       # Session monitoring
│           │
│           ├── verification/        # Verification processes
│           │   ├── signature.ts     # Signature verification
│           │   ├── challenge.ts     # Challenge generation and validation
│           │   ├── credential.ts    # Credential verification for auth
│           │   └── presentation.ts  # Presentation verification
│           │
│           ├── permissions/         # Authorization framework
│           │   ├── model.ts         # Permission data model
│           │   ├── evaluator.ts     # Permission evaluation engine
│           │   ├── grants.ts        # Permission grant management
│           │   └── enforcement.ts   # Permission enforcement
│           │
│           ├── services/            # Authentication services
│           │   ├── gateway.ts       # Authentication gateway
│           │   ├── provider.ts      # Authentication provider
│           │   ├── verifier.ts      # Authentication verifier
│           │   └── auditor.ts       # Authentication audit logging
│           │
│           └── api.ts               # API endpoints
```

### Client SDK Implementation

```
@blackhole/client-sdk/
├── services/
│   └── identity/
│       └── auth-client.ts           # Authentication client
│
├── platforms/
│   ├── browser/
│   │   └── identity/
│   │       └── auth/                # Browser authentication
│   │           ├── ui.ts            # Auth UI components
│   │           └── storage.ts       # Secure storage
│   │
│   ├── react/
│   │   └── identity/
│   │       └── auth/                # React authentication
│   │           ├── hooks.ts         # React hooks
│   │           ├── context.ts       # Auth context
│   │           └── components.ts    # Auth components
│   │
│   └── react-native/
│       └── identity/
│           └── auth/                # Mobile authentication
│               ├── biometrics.ts    # Biometric integration
│               └── secure-store.ts  # Mobile secure storage
```

### Shared Types and Interfaces

```
@blackhole/shared/
└── types/
    └── identity/
        └── authentication/          # Authentication types
            ├── protocols.ts         # Protocol definitions
            ├── sessions.ts          # Session types
            ├── tokens.ts            # Token types
            └── permissions.ts       # Permission types
```

## Technical Specifications

### 1. Cryptographic Requirements

- **Key Types**: Support for Ed25519, ECDSA (secp256k1), RSA-PSS
- **Signature Suites**: Ed25519Signature2020, EcdsaSecp256k1Signature2019, RsaSignature2018
- **Challenge Format**: 128-bit minimum random value with collision resistance
- **Token Security**: JWT with EdDSA or ES256K signatures, short expiration times

### 2. Credential Verification

- **Verification Model**: Full verification chain from presentation to issuer
- **Trust Configuration**: Flexible trust rules for credential acceptance
- **Status Checking**: Real-time revocation status verification
- **Caching**: Appropriate caching of verification results with TTL

### 3. Multi-Factor Authentication

- **Factor Types**: Something you know, have, are, plus context factors
- **Factor Registration**: Secure enrollment of authentication factors
- **Factor Selection**: Smart selection of appropriate factors based on risk
- **Recovery Options**: Factor recovery without compromising security

### 4. Performance Considerations

- **Latency Targets**: Authentication completion <2s for standard flows
- **Caching Strategy**: Efficient caching of DIDs and validation results
- **Scalability**: Stateless design for horizontal scaling
- **Optimization**: Minimal network roundtrips for common operations

## Security Considerations

### 1. Threat Mitigations

- **Phishing**: Clear service identification and consent interfaces
- **Man-in-the-Middle**: Channel security and request signing
- **Replay Attacks**: One-time challenges with expiration
- **Session Hijacking**: Secure token handling and context validation
- **Brute Force**: Rate limiting and progressive delays
- **Impersonation**: Strict verification and trust framework enforcement

### 2. Key Security

- **Key Generation**: Secure random generation with sufficient entropy
- **Key Storage**: Hardware-backed where available, otherwise encrypted storage
- **Key Usage**: Purpose-limited keys with appropriate constraints
- **Key Compromise**: Quick revocation mechanisms and rotation procedures

### 3. Privacy Protection

- **Data Minimization**: Only necessary information requested and shared
- **Purpose Limitation**: Clear purpose specification for all authentications
- **Storage Limitation**: Minimal retention of authentication data
- **User Control**: Clear consent and visibility into information sharing

## Implementation Roadmap

### Phase 1: Core Authentication (Weeks 1-2)
- Implement DID-Auth challenge-response protocol
- Develop basic session management
- Create initial permission model
- Build authentication API foundations

### Phase 2: Credential Authentication (Weeks 3-4)
- Add credential verification for authentication
- Implement trust framework for credential acceptance
- Develop presentation request handling
- Create attribute-based authorization mapping

### Phase 3: Advanced Features (Weeks 5-6)
- Implement multi-factor authentication
- Add zero-knowledge authentication protocols
- Develop cross-device session capabilities
- Create comprehensive session monitoring

### Phase 4: Platform Integration (Weeks 7-8)
- Integrate with web platforms
- Develop mobile authentication flows
- Create desktop application authentication
- Implement OAuth/OIDC compatibility layer

## OAuth/OIDC Compatibility

To support integration with existing systems, Blackhole provides OAuth 2.0 and OpenID Connect compatibility:

### 1. Blackhole as Identity Provider

- **OAuth 2.0 Flows**: Support for authorization code, implicit, and PKCE flows
- **OIDC Provider**: Full OpenID Connect provider implementation
- **Claim Mapping**: Mapping of verified credentials to OIDC claims
- **Token Service**: Standard-compliant token issuance

### 2. Blackhole as Relying Party

- **OAuth Integration**: Authentication with external OAuth providers
- **Identity Linking**: Secure linking of external identities to DIDs
- **Credential Issuance**: Optional credential issuance for external identities
- **Session Unification**: Unified session across authentication methods

## Authentication API Reference

### Authentication Request Endpoint

```
POST /v1/auth/request
```

Creates a new authentication request challenge.

**Request Body:**
```json
{
  "type": "AuthenticationRequest",
  "callbackUrl": "https://service.example.com/auth-response",
  "requiredLevel": "standard",
  "presentationRequirements": [...],
  "expiresIn": 300
}
```

**Response:**
```json
{
  "id": "auth-req-123456789",
  "challenge": "a0e2e3b4c5d6e7f8a9b0c1d2e3f4a5b6",
  "domain": "service.example.com",
  "created": "2023-08-15T14:30:45Z",
  "expires": "2023-08-15T14:35:45Z",
  "requestUri": "blackhole://auth?request=eyJhbGciOiJFUzI1NksifQ..."
}
```

### Authentication Verification Endpoint

```
POST /v1/auth/verify
```

Verifies an authentication response.

**Request Body:**
```json
{
  "type": "AuthenticationResponse",
  "id": "auth-resp-987654321",
  "requestId": "auth-req-123456789",
  "subject": "did:blackhole:123456789abcdef",
  "proof": {...},
  "presentations": [...]
}
```

**Response:**
```json
{
  "verified": true,
  "subject": "did:blackhole:123456789abcdef",
  "level": "standard",
  "sessionToken": "eyJhbGciOiJFUzI1NksifQ...",
  "expiresIn": 3600,
  "permissions": ["content:read", "profile:write", ...]
}
```

## Conclusion

The Blackhole Authentication Protocols provide a comprehensive, secure, and privacy-preserving authentication system built on decentralized identity principles. By leveraging DIDs and verifiable credentials, the platform enables authentication that gives users control over their identity while providing services with sufficient assurance for access control decisions.

This architecture forms the foundation for all authentication processes in the Blackhole platform, enabling secure access to content, social features, and platform capabilities while preserving user sovereignty and privacy.

---

*This document serves as the architectural blueprint for the Authentication Protocols within the Blackhole platform and will be updated as the system evolves.*