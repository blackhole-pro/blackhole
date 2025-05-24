# Blackhole Wallet Service Architecture

This document outlines the architecture and implementation details of the Wallet Service within the Blackhole subprocess architecture, where it runs as an independent OS process communicating with other services via gRPC.

## Introduction

The Blackhole Wallet Service runs as a dedicated subprocess, serving as the user's secure interface to their digital identity, managing DIDs, cryptographic keys, verifiable credentials, zero-knowledge proofs, and authentication capabilities. As a separate process, it communicates with the Identity Service and other services through gRPC while maintaining process-level isolation.

The wallet architecture follows a dual implementation approach:
1. **Decentralized Network Wallet (Default)**: Wallet data stored securely across the P2P node network
2. **Self-Managed Wallet (Advanced Option)**: Complete user control with local storage

## Design Goals

1. **Self-Sovereign Control**: Empower users with complete control over their identity assets
2. **Privacy by Default**: Integrate zero-knowledge proofs for privacy-preserving authentication
3. **Process Isolation**: Run as independent service with clear boundaries
4. **Key Security**: Provide robust security for cryptographic key material
5. **Cross-Platform Consistency**: Maintain consistent experience across all devices
6. **Extensibility**: Support for multiple identity mechanisms and credential types
7. **User-Centric Design**: Simple, intuitive interface with appropriate security prompts
8. **Offline Functionality**: Core operations available without network connectivity
9. **Recovery Mechanisms**: Multiple secure options for identity recovery
10. **Standards Compliance**: Adherence to emerging standards for identity wallets

## Subprocess Architecture

### Service Design

The Wallet Service runs as an isolated subprocess:

```mermaid
graph TD
    subgraph Orchestrator Process
        O[Orchestrator]
        SD[Service Discovery]
        Mon[Monitor]
    end
    
    subgraph Wallet Subprocess
        gRPC[gRPC Server :9006]
        Store[Secure Storage]
        KeyMgmt[Key Management]
        CredMgmt[Credential Manager]
        ZKGen[ZK Proof Generator]
        RecSvc[Recovery Service]
        HW[Hardware Wallet Interface]
    end
    
    subgraph Identity Subprocess
        IDgRPC[gRPC Server :9001]
        DID[DID System]
        ZKVerify[ZK Verifier]
    end
    
    subgraph Storage Subprocess  
        STgRPC[gRPC Server :9002]
        IPFS[IPFS Client]
        Encrypt[Encryption Service]
    end
    
    O -->|spawn| Wallet Subprocess
    SD -->|register| gRPC
    Mon -->|health check| gRPC
    
    Wallet Subprocess -->|gRPC :9001| Identity Subprocess
    Wallet Subprocess -->|gRPC :9002| Storage Subprocess
```

### Service Entry Point

```go
// cmd/blackhole/service/wallet/main.go
package main

import (
    "context"
    "flag"
    "net"
    "os"
    "os/signal"
    
    "google.golang.org/grpc"
    pb "github.com/blackhole/proto/wallet"
    "github.com/blackhole/internal/services/wallet"
)

func main() {
    var (
        grpcPort     = flag.String("grpc-port", "9006", "gRPC server port")
        unixSocket   = flag.String("unix-socket", "/var/run/blackhole/wallet.sock", "Unix socket path")
        configPath   = flag.String("config", "/etc/blackhole/wallet.yaml", "Config file path")
    )
    flag.Parse()
    
    // Initialize service
    svc, err := wallet.NewService(*configPath)
    if err != nil {
        log.Fatalf("Failed to create wallet service: %v", err)
    }
    
    // Create gRPC server
    grpcServer := grpc.NewServer(
        grpc.MaxRecvMsgSize(10 * 1024 * 1024),
        grpc.UnaryInterceptor(authInterceptor),
    )
    
    // Register service
    pb.RegisterWalletServiceServer(grpcServer, svc)
    
    // Listen on Unix socket for local communication
    listener, err := net.Listen("unix", *unixSocket)
    if err != nil {
        log.Fatalf("Failed to listen on Unix socket: %v", err)
    }
    
    // Start gRPC server
    go func() {
        if err := grpcServer.Serve(listener); err != nil {
            log.Fatalf("Failed to serve: %v", err)
        }
    }()
    
    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt)
    <-sigChan
    
    // Graceful shutdown
    grpcServer.GracefulStop()
}
```

## Core Architecture Components

### gRPC Service Definition

```protobuf
// proto/wallet.proto
service WalletService {
    // Key management
    rpc GenerateKey(GenerateKeyRequest) returns (GenerateKeyResponse);
    rpc ImportKey(ImportKeyRequest) returns (ImportKeyResponse);
    rpc ExportKey(ExportKeyRequest) returns (ExportKeyResponse);
    rpc SignData(SignDataRequest) returns (SignDataResponse);
    
    // DID operations
    rpc CreateDID(CreateDIDRequest) returns (CreateDIDResponse);
    rpc ResolveDID(ResolveDIDRequest) returns (ResolveDIDResponse);
    rpc UpdateDID(UpdateDIDRequest) returns (UpdateDIDResponse);
    
    // Credential management
    rpc StoreCredential(StoreCredentialRequest) returns (StoreCredentialResponse);
    rpc GetCredential(GetCredentialRequest) returns (GetCredentialResponse);
    rpc ShareCredential(ShareCredentialRequest) returns (ShareCredentialResponse);
    
    // Zero-knowledge proofs
    rpc GenerateProof(GenerateProofRequest) returns (GenerateProofResponse);
    rpc GetProvableAttributes(GetProvableAttributesRequest) returns (GetProvableAttributesResponse);
    
    // Wallet operations
    rpc Backup(BackupRequest) returns (BackupResponse);
    rpc Restore(RestoreRequest) returns (RestoreResponse);
    rpc Sync(SyncRequest) returns (stream SyncResponse);
    
    // Health check
    rpc Health(HealthRequest) returns (HealthResponse);
}
```

### Wallet Service Implementation

```go
type WalletService struct {
    // Service configuration
    config      *Config
    
    // RPC clients for other services
    identity    pb.IdentityServiceClient
    storage     pb.StorageServiceClient
    
    // Core components
    keyManager  *KeyManager
    credManager *CredentialManager
    zkProver    *ZKProofGenerator
    syncEngine  *SyncEngine
    
    // Security
    encryptor   *Encryptor
    authenticator *Authenticator
    
    // Monitoring
    metrics     *prometheus.Registry
    logger      *zap.Logger
}

func NewService(configPath string) (*WalletService, error) {
    config, err := LoadConfig(configPath)
    if err != nil {
        return nil, fmt.Errorf("load config: %w", err)
    }
    
    // Connect to other services
    identityConn, err := grpc.Dial(
        "unix:///var/run/blackhole/identity.sock",
        grpc.WithInsecure(),
    )
    if err != nil {
        return nil, fmt.Errorf("connect to identity: %w", err)
    }
    
    storageConn, err := grpc.Dial(
        "unix:///var/run/blackhole/storage.sock",
        grpc.WithInsecure(),
    )
    if err != nil {
        return nil, fmt.Errorf("connect to storage: %w", err)
    }
    
    return &WalletService{
        config:      config,
        identity:    pb.NewIdentityServiceClient(identityConn),
        storage:     pb.NewStorageServiceClient(storageConn),
        keyManager:  NewKeyManager(config.KeyStore),
        credManager: NewCredentialManager(),
        zkProver:    NewZKProofGenerator(),
        syncEngine:  NewSyncEngine(config.Sync),
        encryptor:   NewEncryptor(config.Encryption),
        logger:      zap.NewProduction(),
    }, nil
}
```

## Zero-Knowledge Proof Integration

### ZK Proof Generation

The wallet includes built-in capabilities for generating zero-knowledge proofs:

```go
type ZKProofGenerator struct {
    circuits    map[string]Circuit
    prover      Prover
    keyManager  KeyManager
}

type WalletZKCapabilities struct {
    generator   *ZKProofGenerator
    supportedProofs []ZKProofType
    circuitStore   CircuitStore
}

// Generate ZK proof without revealing private data
func (w *WalletService) GenerateProof(
    ctx context.Context,
    req *pb.GenerateProofRequest,
) (*pb.GenerateProofResponse, error) {
    // Validate request
    if err := w.validateProofRequest(req); err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
    }
    
    // Get circuit for proof type
    circuit, ok := w.zkProver.circuits[req.ProofType]
    if !ok {
        return nil, status.Errorf(codes.NotFound, "unsupported proof type: %s", req.ProofType)
    }
    
    // Gather private inputs from wallet
    privateInputs, err := w.gatherPrivateInputs(req)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "gather inputs: %v", err)
    }
    
    // Generate proof
    proof, publicSignals, err := w.zkProver.GenerateProof(
        circuit,
        privateInputs,
        req.PublicInputs,
    )
    if err != nil {
        return nil, status.Errorf(codes.Internal, "generate proof: %v", err)
    }
    
    return &pb.GenerateProofResponse{
        Proof:         proof,
        PublicSignals: publicSignals,
        ProofType:     req.ProofType,
        ExpiresAt:     time.Now().Add(1 * time.Hour).Unix(),
    }, nil
}
```

### Privacy-Preserving Authentication

```go
// Authenticate using zero-knowledge proof
func (w *WalletService) AuthenticateWithZK(
    ctx context.Context,
    req *pb.ZKAuthRequest,
) (*pb.ZKAuthResponse, error) {
    // Generate authentication proof
    proof, err := w.GenerateProof(ctx, &pb.GenerateProofRequest{
        ProofType: "auth_proof",
        PublicInputs: map[string]string{
            "challenge": req.Challenge,
            "timestamp": fmt.Sprintf("%d", time.Now().Unix()),
        },
    })
    if err != nil {
        return nil, err
    }
    
    // Send proof to identity service for verification
    verifyResp, err := w.identity.VerifyZKProof(ctx, &pb.VerifyZKProofRequest{
        Proof:         proof.Proof,
        PublicSignals: proof.PublicSignals,
        ProofType:     proof.ProofType,
    })
    if err != nil {
        return nil, status.Errorf(codes.Internal, "verify proof: %v", err)
    }
    
    if !verifyResp.Valid {
        return nil, status.Error(codes.Unauthenticated, "invalid proof")
    }
    
    return &pb.ZKAuthResponse{
        Token:     verifyResp.AuthToken,
        ExpiresAt: verifyResp.ExpiresAt,
    }, nil
}
```

## Key Management System

### Secure Key Storage

```go
type KeyManager struct {
    store       KeyStore
    encryptor   Encryptor
    hardware    HardwareInterface
}

type KeyStore interface {
    Store(keyID string, encryptedKey []byte) error
    Retrieve(keyID string) ([]byte, error)
    Delete(keyID string) error
    List() ([]string, error)
}

// Hierarchical key derivation
func (km *KeyManager) DeriveKey(
    masterKey []byte,
    path string,
) (*DerivedKey, error) {
    // Parse derivation path
    segments, err := parsePath(path)
    if err != nil {
        return nil, err
    }
    
    // Derive key using BIP32
    key := masterKey
    for _, segment := range segments {
        key, err = deriveChild(key, segment)
        if err != nil {
            return nil, err
        }
    }
    
    return &DerivedKey{
        Key:        key,
        Path:       path,
        ParentKey:  masterKey,
    }, nil
}
```

## Process Security

### Resource Isolation

The wallet subprocess runs with specific resource constraints:

```yaml
# wallet service configuration
services:
  wallet:
    enabled: true
    resources:
      cpu_percent: 100     # 1 CPU core
      memory_mb: 512       # 512MB RAM
      io_weight: 500       # Medium I/O priority
      open_files: 1000     # File descriptor limit
    
    security:
      user: blackhole-wallet
      group: blackhole
      capabilities:
        drop: [ALL]
        keep: [NET_BIND_SERVICE]
      
    storage:
      data_dir: /var/lib/blackhole/wallet
      max_size: 10GB
```

### Communication Security

All RPC communication uses mTLS:

```go
func createTLSConfig() (*tls.Config, error) {
    // Load service certificate
    cert, err := tls.LoadX509KeyPair(
        "/etc/blackhole/certs/wallet.crt",
        "/etc/blackhole/keys/wallet.key",
    )
    if err != nil {
        return nil, err
    }
    
    // Load CA certificate
    caCert, err := os.ReadFile("/etc/blackhole/ca.crt")
    if err != nil {
        return nil, err
    }
    
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)
    
    return &tls.Config{
        Certificates: []tls.Certificate{cert},
        ClientCAs:    caCertPool,
        ClientAuth:   tls.RequireAndVerifyClientCert,
        MinVersion:   tls.VersionTLS13,
    }, nil
}
```

## Wallet Types Implementation

### Decentralized Network Wallet

```go
type NetworkWallet struct {
    walletID    string
    storage     pb.StorageServiceClient
    syncEngine  *SyncEngine
    encryptor   *Encryptor
}

func (w *NetworkWallet) Store(key string, data []byte) error {
    // Encrypt data
    encrypted, err := w.encryptor.Encrypt(data)
    if err != nil {
        return err
    }
    
    // Store in distributed network
    _, err = w.storage.Store(context.Background(), &pb.StoreRequest{
        Key:   fmt.Sprintf("wallet/%s/%s", w.walletID, key),
        Value: encrypted,
        Options: &pb.StoreOptions{
            Redundancy: 3,
            Encryption: true,
        },
    })
    
    return err
}
```

### Self-Managed Wallet

```go
type LocalWallet struct {
    basePath   string
    encryptor  *Encryptor
    fileStore  *FileStore
}

func (w *LocalWallet) Store(key string, data []byte) error {
    // Encrypt data
    encrypted, err := w.encryptor.Encrypt(data)
    if err != nil {
        return err
    }
    
    // Store locally
    path := filepath.Join(w.basePath, key)
    return w.fileStore.WriteSecure(path, encrypted)
}
```

## Recovery Mechanisms

### Multi-Signature Recovery

```go
type RecoveryService struct {
    threshold   int
    guardians   []Guardian
    shamirSplit *ShamirSecretSharing
}

func (r *RecoveryService) CreateRecoveryShares(
    masterKey []byte,
) ([]RecoveryShare, error) {
    // Split master key using Shamir's Secret Sharing
    shares, err := r.shamirSplit.Split(
        masterKey,
        len(r.guardians),
        r.threshold,
    )
    if err != nil {
        return nil, err
    }
    
    // Encrypt shares for each guardian
    var recoveryShares []RecoveryShare
    for i, guardian := range r.guardians {
        encrypted, err := guardian.PublicKey.Encrypt(shares[i])
        if err != nil {
            return nil, err
        }
        
        recoveryShares = append(recoveryShares, RecoveryShare{
            GuardianID: guardian.ID,
            Share:      encrypted,
            Index:      i,
        })
    }
    
    return recoveryShares, nil
}
```

## Integration with Other Services

### Identity Service Integration

```go
func (w *WalletService) CreateDID(
    ctx context.Context,
    req *pb.CreateDIDRequest,
) (*pb.CreateDIDResponse, error) {
    // Generate key pair
    keyPair, err := w.keyManager.GenerateKeyPair(req.KeyType)
    if err != nil {
        return nil, err
    }
    
    // Create DID through identity service
    didResp, err := w.identity.CreateDID(ctx, &pb.IdentityCreateDIDRequest{
        PublicKey: keyPair.PublicKey,
        Method:    req.Method,
    })
    if err != nil {
        return nil, err
    }
    
    // Store private key in wallet
    if err := w.keyManager.Store(didResp.Did, keyPair.PrivateKey); err != nil {
        // Rollback DID creation on storage failure
        w.identity.DeleteDID(ctx, &pb.DeleteDIDRequest{Did: didResp.Did})
        return nil, err
    }
    
    return &pb.CreateDIDResponse{
        Did:       didResp.Did,
        Document:  didResp.Document,
        CreatedAt: time.Now().Unix(),
    }, nil
}
```

## Monitoring and Health

### Service Health Check

```go
func (w *WalletService) Health(
    ctx context.Context,
    req *pb.HealthRequest,
) (*pb.HealthResponse, error) {
    var status pb.HealthStatus
    var details []string
    
    // Check key store
    if err := w.keyManager.HealthCheck(); err != nil {
        status = pb.HealthStatus_DEGRADED
        details = append(details, fmt.Sprintf("keystore: %v", err))
    }
    
    // Check service connections
    if _, err := w.identity.Health(ctx, &pb.HealthRequest{}); err != nil {
        status = pb.HealthStatus_DEGRADED
        details = append(details, fmt.Sprintf("identity: %v", err))
    }
    
    if _, err := w.storage.Health(ctx, &pb.HealthRequest{}); err != nil {
        status = pb.HealthStatus_DEGRADED
        details = append(details, fmt.Sprintf("storage: %v", err))
    }
    
    if status == "" {
        status = pb.HealthStatus_HEALTHY
    }
    
    return &pb.HealthResponse{
        Status:  status,
        Details: details,
        Uptime:  time.Since(w.startTime).Seconds(),
    }, nil
}
```

### Metrics Export

```go
func (w *WalletService) registerMetrics() {
    // RPC metrics
    rpcDuration := prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "wallet_rpc_duration_seconds",
            Help: "RPC request duration",
        },
        []string{"method", "status"},
    )
    
    // Key operation metrics
    keyOps := prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "wallet_key_operations_total",
            Help: "Total key operations",
        },
        []string{"operation", "key_type"},
    )
    
    // ZK proof metrics
    zkProofs := prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "wallet_zk_proofs_generated_total",
            Help: "Total ZK proofs generated",
        },
        []string{"proof_type"},
    )
    
    w.metrics.MustRegister(rpcDuration, keyOps, zkProofs)
}
```

## Security Considerations

1. **Process Isolation**: Wallet runs in separate process with limited privileges
2. **Key Encryption**: All keys encrypted at rest with user passphrase
3. **Memory Protection**: Sensitive data cleared from memory after use
4. **Audit Logging**: All operations logged for security monitoring
5. **Rate Limiting**: Protection against brute force attacks
6. **Secure Communication**: mTLS for all inter-process communication via gRPC

## Future Enhancements

1. **Hardware Wallet Support**: Integration with Ledger, Trezor
2. **Multi-Party Computation**: Advanced key management schemes
3. **Biometric Authentication**: Fingerprint/face recognition
4. **Social Recovery**: Friend-based recovery mechanisms
5. **Advanced ZK Circuits**: More privacy-preserving operations
6. **Cross-Chain Identity**: Support for multiple blockchain networks

## Conclusion

The Wallet Service architecture within the subprocess model provides:

- **Security**: Process-level isolation and encryption
- **Privacy**: Zero-knowledge proof integration
- **Flexibility**: Support for both network and local wallets
- **Reliability**: Service health monitoring and recovery mechanisms
- **Scalability**: Independent service scaling

This design ensures users maintain control of their identity while benefiting from the security and reliability of the Blackhole platform's subprocess architecture.