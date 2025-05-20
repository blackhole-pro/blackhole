# Blackhole Identity Service Architecture

This document provides a comprehensive overview of the Blackhole identity service architecture as a dedicated subprocess within the Blackhole architecture, integrated with the unified security model and zero-knowledge proof capabilities.

## Introduction

The Blackhole identity service runs as an isolated subprocess, providing secure identity management through a decentralized architecture built on W3C standards. As a subprocess, it communicates with other services via gRPC while maintaining process-level isolation. The service integrates seamlessly with the unified Security Manager, providing privacy-preserving authentication through zero-knowledge proofs while maintaining self-sovereign identity principles.

## Architecture Overview

### Subprocess Design

The Identity Service runs as a dedicated subprocess with its own binary entry point:

```mermaid
graph TD
    subgraph Orchestrator
        Orch[Process Manager]
        SD[Service Discovery]
        Mon[Monitor]
    end
    
    subgraph Identity Subprocess
        gRPC[gRPC Server :9001]
        DID[DID System]
        Cred[Credential Manager]
        ZK[ZK Proof Generator]
        Key[Key Management]
    end
    
    subgraph Security Subprocess
        SecGRPC[gRPC Server :9002]
        Auth[DID Authenticator]
        ZKVerify[ZK Verifier]
        PermMap[Permission Mapper]
    end
    
    Orch -->|spawn| Identity Subprocess
    SD -->|register| gRPC
    Mon -->|health check| gRPC
    
    Identity Subprocess -->|gRPC :9002| Security Subprocess
    Security Subprocess -->|gRPC :9001| Identity Subprocess
```

### Service Entry Point

```go
// cmd/blackhole/service/identity/main.go
package main

import (
    "context"
    "flag"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/blackhole/internal/services/identity"
    "github.com/blackhole/pkg/api/identity/v1"
    "google.golang.org/grpc"
)

var (
    port       = flag.Int("port", 9001, "gRPC port")
    unixSocket = flag.String("unix-socket", "/tmp/blackhole-identity.sock", "Unix socket path")
    config     = flag.String("config", "", "Configuration file path")
)

func main() {
    flag.Parse()
    
    // Initialize service
    cfg, err := identity.LoadConfig(*config)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    service, err := identity.New(cfg)
    if err != nil {
        log.Fatalf("Failed to create service: %v", err)
    }
    
    // Create gRPC server
    grpcServer := grpc.NewServer(
        grpc.MaxRecvMsgSize(10 * 1024 * 1024), // 10MB
        grpc.MaxSendMsgSize(10 * 1024 * 1024),
    )
    
    // Register service
    identityv1.RegisterIdentityServiceServer(grpcServer, service)
    
    // Listen on Unix socket for local communication
    unixListener, err := net.Listen("unix", *unixSocket)
    if err != nil {
        log.Fatalf("Failed to listen on unix socket: %v", err)
    }
    defer os.Remove(*unixSocket)
    
    // Listen on TCP for remote communication
    tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
    if err != nil {
        log.Fatalf("Failed to listen on TCP: %v", err)
    }
    
    // Handle shutdown gracefully
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    
    go func() {
        <-sigChan
        log.Println("Shutting down identity service...")
        grpcServer.GracefulStop()
        cancel()
    }()
    
    // Start serving
    go func() {
        log.Printf("Identity service listening on Unix socket: %s", *unixSocket)
        if err := grpcServer.Serve(unixListener); err != nil {
            log.Fatalf("Failed to serve Unix socket: %v", err)
        }
    }()
    
    log.Printf("Identity service listening on TCP port: %d", *port)
    if err := grpcServer.Serve(tcpListener); err != nil {
        log.Fatalf("Failed to serve TCP: %v", err)
    }
}
```

### Integration with Unified Security Model

```mermaid
graph TD
    subgraph User Layer
        Wallet[Identity Wallet]
        ZKGen[ZK Proof Generator]
    end
    
    subgraph Security Manager Subprocess
        Auth[DID Authenticator]
        ZKVerify[ZK Verifier]
        PermMap[Permission Mapper]
    end
    
    subgraph Identity Service Subprocess
        DIDReg[DID Registry]
        CredStore[Credential Store]
        KeyMgmt[Key Management]
        ProofGen[Proof Generation]
    end
    
    Wallet -->|gRPC| Auth
    ZKGen -->|gRPC| ZKVerify
    Auth -->|gRPC| DIDReg
    ZKVerify -->|gRPC| CredStore
    PermMap -->|gRPC| Identity Service Subprocess
```

## Core Components with Resource Management

The Identity Service runs as an isolated subprocess with OS-level resource controls. Resource management happens at the process level through cgroups and system limits.

### Process Resource Configuration

```go
// Identity service resource limits
type IdentityServiceConfig struct {
    ProcessLimits ProcessResourceLimits {
        CPUQuota    "200%"         // 2 CPU cores max
        MemoryLimit "1GB"          // 1GB memory limit
        IOWeight    100            // Standard IO priority
        Nice        0              // Standard scheduling priority
    }
    
    // gRPC client for resource manager
    ResourceClient *grpc.ClientConn
}

// Resource monitoring through gRPC
func (s *IdentityService) ReportResourceUsage(ctx context.Context) error {
    stats := s.getProcessStats()
    
    _, err := s.resourceClient.ReportUsage(ctx, &ResourceUsage{
        Service:   "identity",
        CPUUsage:  stats.CPUPercent,
        MemUsage:  stats.MemoryMB,
        IOOps:     stats.IOOperations,
        Timestamp: time.Now(),
    })
    
    return err
}
```

### 1. DID System with Privacy Features

The DID system includes privacy-preserving capabilities with process-isolated resource management:

```go
type DIDSystem struct {
    resolver      DIDResolver
    registry      DIDRegistry
    ipfsStore     IPFSStore
    zkGenerator   ZKProofGenerator
    resourceMon   ResourceMonitor   // Process resource monitoring
}

// Privacy-preserving DID authentication with subprocess resource monitoring
func (d *DIDSystem) AuthenticateWithPrivacy(ctx context.Context, didRequest DIDAuthRequest) (*PrivateAuthResult, error) {
    // Monitor ZK proof generation resource usage
    startTime := time.Now()
    startCPU := d.resourceMon.GetCPUUsage()
    startMem := d.resourceMon.GetMemoryUsage()
    
    // Generate DID commitment for anonymous auth
    commitment := d.generateDIDCommitment(didRequest.DID)
    
    // Create ZK proof of DID ownership (CPU intensive)
    proof, err := d.zkGenerator.ProveOwnership(
        didRequest.DID,
        didRequest.PrivateKey,
        commitment,
    )
    
    if err != nil {
        return nil, err
    }
    
    // Report resource usage for this operation
    d.resourceMon.ReportOperation("zk_proof_generation", OperationMetrics{
        Duration:  time.Since(startTime),
        CPUDelta:  d.resourceMon.GetCPUUsage() - startCPU,
        MemDelta:  d.resourceMon.GetMemoryUsage() - startMem,
    })
    
    return &PrivateAuthResult{
        DIDCommitment: commitment,
        Proof:         proof,
        ValidUntil:    time.Now().Add(time.Hour),
    }, nil
}
```

### 2. Zero-Knowledge Credential System

Verifiable credentials with privacy preservation:

```go
type ZKCredentialSystem struct {
    store        CredentialStore
    verifier     ZKVerifier
    issuerTrust  IssuerTrustRegistry
}

// Prove credential attributes without revealing the credential
func (c *ZKCredentialSystem) ProveAttributes(request AttributeProofRequest) (*AttributeProof, error) {
    // Fetch encrypted credential
    credential, err := c.store.Get(request.CredentialID)
    if err != nil {
        return nil, err
    }
    
    // Generate ZK proof for requested attributes
    proof, err := c.generateAttributeProof(
        credential,
        request.RequestedAttributes,
        request.ProofType,
    )
    
    return &AttributeProof{
        Proof:              proof,
        ProvenAttributes:   request.RequestedAttributes,
        IssuerCommitment:   c.hashIssuer(credential.Issuer),
        ValidUntil:         time.Now().Add(time.Hour),
    }, nil
}
```

### 3. Unified Authentication Flow

Integration with Security Manager via gRPC:

```go
type IdentityService struct {
    securityClient  securityv1.SecurityServiceClient // gRPC client
    didSystem       *DIDSystem
    credentialMgr   *CredentialManager
    zkProofGen      *ZKProofGenerator
}

// Initialize gRPC client to Security Manager
func (i *IdentityService) InitSecurityClient(ctx context.Context) error {
    conn, err := grpc.DialContext(ctx, "unix:///tmp/blackhole-security.sock",
        grpc.WithInsecure(),
        grpc.WithDefaultCallOptions(
            grpc.MaxCallRecvMsgSize(10 * 1024 * 1024),
            grpc.MaxCallSendMsgSize(10 * 1024 * 1024),
        ),
    )
    if err != nil {
        return fmt.Errorf("failed to connect to security service: %w", err)
    }
    
    i.securityClient = securityv1.NewSecurityServiceClient(conn)
    return nil
}

// Authenticate through Security Manager subprocess
func (i *IdentityService) Authenticate(ctx context.Context, authReq AuthRequest) (*SecurityContext, error) {
    // Route through Security Manager subprocess via gRPC
    resp, err := i.securityClient.AuthenticateWithDID(ctx, &securityv1.DIDAuthRequest{
        Did:       authReq.DID,
        Challenge: authReq.Challenge,
        Signature: authReq.Signature,
        ProofType: authReq.ProofType,
    })
    if err != nil {
        return nil, fmt.Errorf("security service auth failed: %w", err)
    }
    
    return &SecurityContext{
        UserID: resp.UserId,
        Permissions: resp.Permissions,
        Zone: SecurityZone(resp.Zone),
        ValidUntil: resp.ValidUntil.AsTime(),
    }, nil
}

// Generate ZK proofs for Security Manager
func (i *IdentityService) GenerateZKProof(proofReq ZKProofRequest) (*ZKProof, error) {
    switch proofReq.Type {
    case ProofOfAge:
        return i.generateAgeProof(proofReq)
    case ProofOfCredential:
        return i.generateCredentialProof(proofReq)
    case ProofOfMembership:
        return i.generateMembershipProof(proofReq)
    default:
        return nil, ErrUnsupportedProofType
    }
}
```

## Privacy-Preserving Features

### 1. Anonymous Authentication

Users can authenticate without revealing their DID:

```go
func (i *IdentityService) AnonymousAuth(anonReq AnonymousAuthRequest) (*AnonymousSession, error) {
    // Generate ephemeral identifier
    ephemeralID := i.generateEphemeralID()
    
    // Create ZK proof of valid DID ownership
    proof, err := i.zkProofGen.ProveValidDID(
        anonReq.DIDCommitment,
        anonReq.Challenge,
    )
    
    if err != nil {
        return nil, err
    }
    
    // Create anonymous session
    session := &AnonymousSession{
        EphemeralID: ephemeralID,
        Proof:       proof,
        Permissions: i.getAnonymousPermissions(proof),
        ExpiresAt:   time.Now().Add(time.Hour),
    }
    
    return session, nil
}
```

### 2. Selective Disclosure

Choose which attributes to reveal:

```go
type SelectiveDisclosure struct {
    credential    VerifiableCredential
    disclosedAttrs []string
    hiddenAttrs   []string
    proof        ZKProof
}

func (i *IdentityService) SelectivelyDisclose(
    credentialID string,
    attributesToReveal []string,
) (*SelectiveDisclosure, error) {
    cred, err := i.credentialMgr.Get(credentialID)
    if err != nil {
        return nil, err
    }
    
    // Generate proof for selective disclosure
    proof, err := i.zkProofGen.ProveSelectedAttributes(
        cred,
        attributesToReveal,
    )
    
    return &SelectiveDisclosure{
        credential:     cred,
        disclosedAttrs: attributesToReveal,
        hiddenAttrs:    i.getHiddenAttributes(cred, attributesToReveal),
        proof:          proof,
    }, nil
}
```

## Key Management with Security Zones

Key operations respect security zones:

```go
type KeyManager struct {
    securityMgr  *SecurityManager
    keyStore     KeyStore
    hsmIntegration HSMInterface
}

func (k *KeyManager) PerformKeyOperation(
    ctx context.Context,
    operation KeyOperation,
) error {
    secCtx := GetSecurityContext(ctx)
    
    // Check zone permissions for key operations
    switch operation.Type {
    case KeyGeneration:
        if secCtx.Zone < ProtectedZone {
            return ErrInsufficientPermissions
        }
    case KeyRotation:
        if secCtx.Zone < InternalZone {
            return ErrInsufficientPermissions
        }
    case KeyRecovery:
        if secCtx.Zone < RestrictedZone {
            return ErrInsufficientPermissions
        }
    }
    
    return k.executeOperation(operation)
}
```

## Credential Types and ZK Circuits

### Supported Credential Types

```go
const (
    // Identity credentials
    CredentialKYC        = "kyc_verification"
    CredentialAge        = "age_verification"
    CredentialNationality = "nationality"
    
    // Professional credentials
    CredentialEducation  = "education"
    CredentialEmployment = "employment"
    CredentialLicense    = "professional_license"
    
    // Financial credentials
    CredentialIncome     = "income_verification"
    CredentialCredit     = "credit_score"
    CredentialAssets     = "asset_ownership"
)
```

### ZK Circuit Integration

```go
type ZKCircuitManager struct {
    circuits map[string]Circuit
    prover   Prover
    verifier Verifier
}

// Register identity-specific circuits
func (z *ZKCircuitManager) RegisterIdentityCircuits() {
    z.circuits["age_over_18"] = &AgeVerificationCircuit{
        MinAge: 18,
    }
    
    z.circuits["kyc_verified"] = &KYCVerificationCircuit{
        RequiredFields: []string{"name", "address", "id_number"},
    }
    
    z.circuits["credential_ownership"] = &CredentialOwnershipCircuit{
        AcceptedIssuers: []string{"gov_id_issuer", "verified_kyc_provider"},
    }
}
```

## Integration Points

### 1. Security Manager Integration

```go
// Identity service registers with Security Manager via service discovery
func (i *IdentityService) RegisterWithServiceDiscovery(ctx context.Context) error {
    // Connect to orchestrator for service registration
    conn, err := grpc.DialContext(ctx, "unix:///tmp/blackhole-orchestrator.sock",
        grpc.WithInsecure(),
    )
    if err != nil {
        return fmt.Errorf("failed to connect to orchestrator: %w", err)
    }
    defer conn.Close()
    
    client := orchestratorv1.NewOrchestratorClient(conn)
    
    // Register identity service
    _, err = client.RegisterService(ctx, &orchestratorv1.ServiceRegistration{
        Name:         "identity",
        Version:      "1.0.0",
        Port:         9001,
        UnixSocket:   "/tmp/blackhole-identity.sock",
        HealthCheck:  "/health",
        Dependencies: []string{"security"},
        Endpoints: []string{
            "AuthenticateWithDID",
            "GenerateZKProof",
            "ManageCredentials",
        },
    })
    
    return err
}
```

### 2. Wallet Integration

```go
// Wallet generates proofs for identity operations
type IdentityWallet struct {
    keyManager  KeyManager
    proofGen    ProofGenerator
    didManager  DIDManager
}

func (w *IdentityWallet) GenerateAuthProof(challenge string) (*AuthProof, error) {
    // Sign challenge with DID key
    signature, err := w.keyManager.Sign(challenge)
    if err != nil {
        return nil, err
    }
    
    // Generate ZK proof if requested
    var zkProof *ZKProof
    if w.proofGen.IsZKEnabled() {
        zkProof, err = w.proofGen.ProveIdentity(w.didManager.GetDID())
        if err != nil {
            return nil, err
        }
    }
    
    return &AuthProof{
        DID:       w.didManager.GetDID(),
        Signature: signature,
        ZKProof:   zkProof,
    }, nil
}
```

## Security Considerations

### 1. Privacy by Default
- All operations support privacy-preserving modes
- ZK proofs preferred over direct credential sharing
- Minimal data disclosure principle

### 2. Zone-Based Access
- Key operations restricted by security zone
- Credential issuance requires Internal zone
- Recovery operations need Restricted zone

### 3. Audit Privacy
- Audit logs use DID commitments
- ZK proof verifications logged without details
- Privacy-preserving analytics

## Configuration

```yaml
identity_service:
  # Service configuration
  service:
    name: "identity"
    port: 9001
    unix_socket: "/tmp/blackhole-identity.sock"
    log_level: "info"
    
  # Process management
  process:
    cpu_limit: "200%"          # 2 CPU cores max
    memory_limit: "1GB"        # 1GB memory limit
    restart_policy: "always"
    restart_delay: "5s"
    health_check_interval: "30s"
    
  # DID configuration
  did:
    method: "root"
    registry_endpoint: "https://did.rootnetwork.xyz"
    ipfs_gateway: "https://ipfs.io"
    
  # Zero-knowledge configuration
  zk_proofs:
    enabled: true
    supported_types:
      - age_verification
      - kyc_verification
      - credential_ownership
    circuit_path: "/circuits/identity"
    
  # Credential configuration
  credentials:
    storage: "ipfs"
    encryption: "aes-256-gcm"
    trusted_issuers:
      - "did:root:gov_issuer"
      - "did:root:verified_kyc"
      
  # Security integration via gRPC
  security:
    grpc_endpoint: "unix:///tmp/blackhole-security.sock"
    unified_auth: true
    zone_restrictions:
      key_generation: "protected"
      key_rotation: "internal"
      credential_issuance: "internal"
      recovery: "restricted"
```

## Resource Usage Patterns

The Identity Service subprocess has specific resource requirements managed at the OS level:

### CPU-Intensive Operations
- Zero-knowledge proof generation (can spike to 200% CPU)
- Cryptographic signature operations
- Key derivation and generation
- Credential verification

### Memory Requirements
- Base memory: ~256MB
- Credential caching: up to 512MB
- ZK circuit loading: ~256MB when active
- Peak memory usage: 1GB (hard limit)

### Process-Level Resource Management
```go
// Resource monitoring for subprocess
type IdentityResourceMonitor struct {
    pid              int
    cpuThreshold     float64
    memoryThreshold  uint64
}

func (m *IdentityResourceMonitor) CheckResourceHealth() error {
    stats, err := process.NewProcess(int32(m.pid))
    if err != nil {
        return err
    }
    
    cpuPercent, _ := stats.CPUPercent()
    memInfo, _ := stats.MemoryInfo()
    
    // Alert if approaching limits
    if cpuPercent > m.cpuThreshold {
        log.Warnf("Identity service CPU usage high: %.2f%%", cpuPercent)
    }
    
    if memInfo.RSS > m.memoryThreshold {
        log.Warnf("Identity service memory usage high: %d MB", memInfo.RSS/1024/1024)
    }
    
    return nil
}
```

### Subprocess Isolation Benefits
- OS-enforced CPU limits (cgroups)
- Hard memory boundaries (no OOM affecting other services)
- Process priority management
- Independent crash recovery
- Resource usage visibility through system tools

## Migration Path

Since no code exists yet, this architecture will be the initial implementation:

1. Build DID system with ZK capabilities
2. Implement credential system with selective disclosure
3. Integrate with Security Manager
4. Add privacy-preserving features
5. Deploy ZK circuits for common proofs
6. Enable anonymous authentication modes

## Benefits

1. **Privacy First**: ZK proofs enable authentication without identity exposure
2. **Process Isolation**: Subprocess architecture ensures service independence
3. **Resource Control**: OS-level resource limits prevent service interference
4. **Self-Sovereign**: Users control their identity and credentials
5. **Fault Tolerance**: Service crashes don't affect other components
6. **Clear Boundaries**: gRPC interfaces enforce clean service contracts
7. **Independent Scaling**: Can adjust resources per subprocess as needed
8. **Security**: Process-level isolation provides additional security boundaries