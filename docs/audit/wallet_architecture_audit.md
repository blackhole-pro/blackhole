# Wallet Architecture Audit

## Executive Summary

The wallet architecture for the Blackhole platform shows significant architectural inconsistencies with the subprocess model defined in PROJECT.md. While the design contains valuable concepts around zero-knowledge proofs, IPFS credential storage, and advanced security features, it lacks crucial subprocess architecture requirements and contains conflicting design patterns.

## Critical Issues

### 1. Missing gRPC Service Definition

**Severity**: CRITICAL

The wallet architecture lacks a complete gRPC service definition despite the subprocess architecture requirement:

- `wallet_architecture.md` shows a partial protobuf definition (lines 138-169) but is incomplete
- No implementation of required gRPC communication patterns
- Missing service registration with orchestrator
- No Unix socket/TCP communication setup
- Conflicts with PROJECT.md requirement: "All inter-service communication uses gRPC" (line 226)

**Required**: Complete gRPC service definition with all wallet operations.

### 2. Resource Allocation Mismatch

**Severity**: CRITICAL

Resource specifications conflict between documents:

- `wallet_architecture.md` specifies "memory_mb: 512" (line 394)
- PROJECT.md allocates wallet service 1GB memory (not explicitly shown but inferred from pattern)
- CPU allocation undefined in wallet docs but should follow PROJECT.md pattern
- Missing IO weight specifications

**Required**: Align with PROJECT.md resource patterns.

### 3. Subprocess Entry Point Violation

**Severity**: CRITICAL

The wallet service lacks proper subprocess implementation:

- `wallet_architecture.md` shows incorrect entry point structure (lines 74-131)
- Missing proper process management hooks
- No graceful shutdown implementation
- Incomplete health check integration
- Violates PROJECT.md subprocess pattern

**Required**: Implement proper subprocess lifecycle management.

### 4. Web Interface Architecture Conflict

**Severity**: CRITICAL

Multiple documents suggest browser-based interfaces:

- `wallet_workflow_diagrams.md` references direct web interactions
- `expanded_wallet_architecture.md` mentions browser compatibility
- Conflicts with subprocess architecture that requires gRPC-only communication
- Violates process boundary requirements

**Required**: Remove all web interface references and use gRPC exclusively.

## Important Issues

### 5. Overly Complex Architecture

**Severity**: IMPORTANT

The wallet service design is overly complex:

- Multiple wallet types (decentralized, self-managed) add unnecessary complexity
- Hardware wallet integration might require external process communication
- Enterprise features exceed subprocess boundaries
- Too many advanced features for initial implementation

**Recommendation**: Simplify to core wallet functionality first.

### 6. IPFS Integration Assumptions

**Severity**: IMPORTANT

The IPFS credential storage design makes assumptions:

- `ipfs_credential_storage.md` assumes direct IPFS access from wallet
- Should integrate through Storage Service instead
- Creates tight coupling with IPFS implementation
- Bypasses service boundaries

**Recommendation**: Route IPFS operations through Storage Service via gRPC.

### 7. Unclear Service Boundaries

**Severity**: IMPORTANT

Service boundaries are poorly defined:

- Direct blockchain interaction instead of through Ledger Service
- Credential management overlaps with Identity Service
- Key management might conflict with Identity Service responsibilities
- Unclear separation of concerns

**Recommendation**: Clarify service responsibilities and use proper service delegation.

### 8. Missing Monitoring Integration

**Severity**: IMPORTANT

Limited monitoring and telemetry integration:

- Basic Prometheus metrics mentioned but incomplete
- No integration with Telemetry Service
- Missing distributed tracing setup
- Inadequate health monitoring

**Recommendation**: Full integration with platform telemetry system.

## Deferrable Issues

### 9. Advanced Security Features

**Severity**: DEFERRABLE

Many security features are premature:

- Multi-party computation
- Homomorphic encryption
- Quantum-resistant algorithms
- Advanced biometrics

**Recommendation**: Implement after core functionality is stable.

### 10. Complex Recovery Mechanisms

**Severity**: DEFERRABLE

Social recovery and multi-signature recovery are complex:

- Can be added later
- Core recovery via seed phrase is sufficient initially
- Adds unnecessary complexity to initial release

**Recommendation**: Start with basic recovery, add advanced features later.

### 11. Cross-Chain Support

**Severity**: DEFERRABLE

Multi-blockchain support adds complexity:

- Start with Root Network only
- Add cross-chain later
- Simplifies initial implementation

**Recommendation**: Single chain first, multi-chain later.

## Architectural Recommendations

### 1. Proper gRPC Implementation

```protobuf
// proto/wallet/v1/wallet.proto
syntax = "proto3";

package blackhole.wallet.v1;

service WalletService {
    // Core wallet operations
    rpc CreateWallet(CreateWalletRequest) returns (CreateWalletResponse);
    rpc GetWallet(GetWalletRequest) returns (GetWalletResponse);
    
    // Key management (simplified)
    rpc GenerateKey(GenerateKeyRequest) returns (GenerateKeyResponse);
    rpc SignData(SignDataRequest) returns (SignDataResponse);
    
    // Credential operations (delegated to Identity)
    rpc StoreCredentialReference(StoreCredentialRefRequest) returns (StoreCredentialRefResponse);
    rpc GetCredentialReferences(GetCredentialRefsRequest) returns (GetCredentialRefsResponse);
    
    // Transaction operations (delegated to Ledger)
    rpc PrepareTransaction(PrepareTransactionRequest) returns (PrepareTransactionResponse);
    rpc SignTransaction(SignTransactionRequest) returns (SignTransactionResponse);
    
    // Health monitoring
    rpc Health(HealthRequest) returns (HealthResponse);
}
```

### 2. Subprocess Structure

```go
// internal/services/wallet/main.go
func main() {
    // Parse flags
    flags := parseFlags()
    
    // Load configuration
    cfg, err := LoadConfig(flags.ConfigPath)
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // Create service
    svc, err := wallet.New(cfg)
    if err != nil {
        log.Fatal("Failed to create service:", err)
    }
    
    // Setup gRPC server
    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(authInterceptor),
        grpc.MaxRecvMsgSize(cfg.MaxMessageSize),
    )
    
    // Register service
    walletv1.RegisterWalletServiceServer(grpcServer, svc)
    
    // Listen on Unix socket for local communication
    listener, err := net.Listen("unix", cfg.SocketPath)
    if err != nil {
        log.Fatal("Failed to listen:", err)
    }
    
    // Start server with graceful shutdown
    go func() {
        if err := grpcServer.Serve(listener); err != nil {
            log.Error("Server failed:", err)
        }
    }()
    
    // Wait for shutdown signal
    <-waitForShutdown()
    
    // Graceful shutdown
    grpcServer.GracefulStop()
}
```

### 3. Service Integration Pattern

```go
// Service should delegate to other services via gRPC
type WalletService struct {
    // Service clients
    identity pb.IdentityServiceClient
    storage  pb.StorageServiceClient
    ledger   pb.LedgerServiceClient
    
    // Local components only
    keyStore     KeyStore
    cryptoEngine CryptoEngine
    
    // Monitoring
    metrics *prometheus.Registry
    logger  *zap.Logger
}

// Example: Store credential (delegated to Identity)
func (s *WalletService) StoreCredentialReference(
    ctx context.Context,
    req *pb.StoreCredentialRefRequest,
) (*pb.StoreCredentialRefResponse, error) {
    // Wallet only stores reference, actual credential goes to Identity
    ref := CredentialReference{
        ID:        req.CredentialId,
        StorageID: req.StorageId,
        Type:      req.Type,
    }
    
    // Store reference locally
    if err := s.keyStore.StoreCredentialRef(ref); err != nil {
        return nil, err
    }
    
    // Actual credential storage via Identity Service
    _, err := s.identity.StoreCredential(ctx, &identitypb.StoreCredentialRequest{
        Credential: req.Credential,
    })
    if err != nil {
        return nil, err
    }
    
    return &pb.StoreCredentialRefResponse{
        Success: true,
    }, nil
}
```

### 4. Simplified Resource Configuration

```yaml
# Align with PROJECT.md patterns
services:
  wallet:
    enabled: true
    resources:
      cpu_percent: 100    # 1 CPU core
      memory_mb: 1024     # 1GB RAM (standard service allocation)
      io_weight: 500      # Medium I/O priority
    network:
      socket: /var/run/blackhole/wallet.sock
      port: 9006  # If TCP is needed
    storage:
      data_dir: /var/lib/blackhole/wallet
      max_size: 5GB  # Reasonable limit
```

## Risk Assessment

### High Risk Areas

1. **Incomplete subprocess implementation** - Could cause integration failures
2. **Conflicting architectures** - Web vs subprocess model confusion
3. **Service boundary violations** - Direct external service access
4. **Resource misalignment** - Could cause runtime issues

### Mitigation Strategy

1. Implement proper subprocess architecture first
2. Remove all web-based assumptions
3. Enforce service boundaries through gRPC
4. Align resources with PROJECT.md
5. Simplify to core features initially

## Implementation Priorities

1. **Week 1**: Fix subprocess architecture and gRPC implementation
2. **Week 2**: Implement core wallet operations (create, key management)
3. **Week 3**: Add credential reference management
4. **Week 4**: Integrate with Identity and Ledger services
5. **Week 5**: Add monitoring and health checks
6. **Week 6**: Testing and documentation

## Conclusion

The wallet architecture contains valuable concepts but needs significant revision to align with the Blackhole subprocess architecture. The primary focus should be on:

1. Implementing proper subprocess patterns
2. Establishing clear service boundaries
3. Using gRPC for all communication
4. Simplifying initial feature set
5. Aligning with platform standards

The advanced features (ZK proofs, complex recovery, multi-chain) should be deferred until the core subprocess implementation is stable and proven.