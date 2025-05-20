# Security Architecture - Process-Based Security Model

## Overview

The Blackhole platform implements a process-based security model where each service runs as an independent OS process with its own security boundaries. This architecture leverages OS-level isolation and process separation to provide robust security while supporting privacy-preserving authentication through zero-knowledge proofs.

## Architecture Principles

1. **Process Isolation**: Each service runs in its own OS process with separated memory
2. **mTLS Communication**: All inter-process communication secured with mutual TLS
3. **Resource Boundaries**: OS-enforced resource limits per service process
4. **Privacy by Design**: Zero-knowledge proofs for privacy-preserving authentication
5. **Defense in Depth**: Multiple layers of security from OS to application level

## Core Components

### Process Security Model

Each service process has independent security boundaries:

```go
type ProcessSecurity struct {
    // Process isolation
    ProcessID      int
    UserID         int
    GroupID        int
    Capabilities   []string
    
    // Network security
    NetworkPolicy  NetworkPolicy
    TLSConfig      *tls.Config
    
    // File system isolation
    RootDir        string
    AllowedPaths   []string
    
    // Resource limits
    ResourceLimits ResourceLimits
}
```

### Service Authentication

Services authenticate each other using mutual TLS:

```go
type ServiceAuth struct {
    // Service identity
    ServiceName    string
    Certificate    *x509.Certificate
    PrivateKey     crypto.PrivateKey
    
    // Trust configuration
    CACertPool     *x509.CertPool
    AllowedDNs     []string
    
    // mTLS configuration
    ClientAuth     tls.ClientAuthType
    MinTLSVersion  uint16
}

func (s *ServiceAuth) CreateTLSConfig() *tls.Config {
    return &tls.Config{
        Certificates: []tls.Certificate{{
            Certificate: [][]byte{s.Certificate.Raw},
            PrivateKey:  s.PrivateKey,
        }},
        ClientAuth:     tls.RequireAndVerifyClientCert,
        ClientCAs:      s.CACertPool,
        MinVersion:     s.MinTLSVersion,
        VerifyPeerCertificate: s.verifyServiceIdentity,
    }
}
```

### Zero-Knowledge Authentication

Privacy-preserving authentication for users:

```go
type ZKAuthManager struct {
    // Circuit verification
    circuitStore   map[string]ZKCircuit
    verifier       ZKVerifier
    
    // Proof management
    proofCache     *ProofCache
    proofTimeout   time.Duration
    
    // Service integration
    rpcClients     map[string]*grpc.ClientConn
}

// Authenticate user without revealing identity
func (m *ZKAuthManager) AuthenticateWithProof(proof ZKProof) (*AuthContext, error) {
    // Verify proof without accessing private data
    result, err := m.verifier.Verify(proof)
    if err != nil {
        return nil, err
    }
    
    // Create auth context from proven attributes
    return &AuthContext{
        ProvenClaims:   result.Claims,
        Permissions:    m.derivePermissions(result.Claims),
        IsAnonymous:    true,
        ValidUntil:     time.Now().Add(m.proofTimeout),
    }, nil
}
```

## Process-Level Security

### Process Isolation

Each service runs with specific security constraints:

```go
type ProcessIsolation struct {
    // User/Group isolation
    User  string
    Group string
    
    // Capability restrictions
    DropCapabilities []string
    KeepCapabilities []string
    
    // Seccomp filters
    SeccompProfile string
    
    // Namespace isolation
    Namespaces []string // pid, net, mount, user, ipc
}

func (i *ProcessIsolation) ApplyToCommand(cmd *exec.Cmd) error {
    // Set user/group
    if i.User != "" {
        uid, gid, err := i.getUserAndGroup()
        if err != nil {
            return err
        }
        cmd.SysProcAttr.Credential = &syscall.Credential{
            Uid: uid,
            Gid: gid,
        }
    }
    
    // Apply capability restrictions
    if len(i.DropCapabilities) > 0 {
        cmd.SysProcAttr.AmbientCaps = i.filterCapabilities()
    }
    
    // Apply namespace isolation
    if len(i.Namespaces) > 0 {
        cmd.SysProcAttr.Cloneflags = i.getCloneFlags()
    }
    
    return nil
}
```

### Network Security

Service-specific network policies:

```go
type NetworkPolicy struct {
    // Allowed connections
    AllowedServices []string
    AllowedPorts    []int
    
    // TLS requirements
    RequireTLS      bool
    MinTLSVersion   uint16
    AllowedCiphers  []uint16
    
    // Rate limiting
    RateLimit       *RateLimitConfig
}

func (p *NetworkPolicy) CreateListener(addr string) (net.Listener, error) {
    // Create base listener
    listener, err := net.Listen("tcp", addr)
    if err != nil {
        return nil, err
    }
    
    // Wrap with TLS if required
    if p.RequireTLS {
        tlsConfig := &tls.Config{
            MinVersion:   p.MinTLSVersion,
            CipherSuites: p.AllowedCiphers,
            ClientAuth:   tls.RequireAndVerifyClientCert,
        }
        listener = tls.NewListener(listener, tlsConfig)
    }
    
    // Add rate limiting
    if p.RateLimit != nil {
        listener = &rateLimitListener{
            Listener:  listener,
            rateLimit: p.RateLimit,
        }
    }
    
    return listener, nil
}
```

### File System Security

Process-specific file access control:

```go
type FileSystemSecurity struct {
    // Root directory (chroot)
    RootDir string
    
    // Allowed paths
    ReadPaths  []string
    WritePaths []string
    
    // Denied paths
    DenyPaths []string
    
    // Mount options
    ReadOnlyMounts []string
    NoExecMounts   []string
}

func (f *FileSystemSecurity) ApplyToProcess(pid int) error {
    // Apply chroot if configured
    if f.RootDir != "" {
        if err := syscall.Chroot(f.RootDir); err != nil {
            return err
        }
    }
    
    // Apply mount restrictions
    for _, path := range f.ReadOnlyMounts {
        if err := remountReadOnly(path); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Inter-Process Security

### Secure RPC Communication

All service communication uses authenticated gRPC:

```go
type SecureRPCManager struct {
    // Service credentials
    serviceAuth    *ServiceAuth
    
    // Connection security
    tlsConfig      *tls.Config
    authPolicy     AuthPolicy
    
    // Connection management
    connections    map[string]*grpc.ClientConn
    mu             sync.RWMutex
}

func (m *SecureRPCManager) Connect(service string) (*grpc.ClientConn, error) {
    // Create TLS credentials
    creds := credentials.NewTLS(m.tlsConfig)
    
    // Add authentication interceptors
    opts := []grpc.DialOption{
        grpc.WithTransportCredentials(creds),
        grpc.WithUnaryInterceptor(m.authInterceptor),
        grpc.WithStreamInterceptor(m.streamAuthInterceptor),
    }
    
    // For local services, use Unix sockets with additional security
    if m.isLocalService(service) {
        socketPath := m.getServiceSocket(service)
        
        // Verify socket permissions
        if err := m.verifySocketSecurity(socketPath); err != nil {
            return nil, err
        }
        
        opts = append(opts, grpc.WithContextDialer(
            m.createSecureUnixDialer(socketPath),
        ))
    }
    
    return grpc.Dial(m.getServiceAddress(service), opts...)
}

// Verify service identity in RPC calls
func (m *SecureRPCManager) authInterceptor(
    ctx context.Context,
    method string,
    req, reply interface{},
    cc *grpc.ClientConn,
    invoker grpc.UnaryInvoker,
    opts ...grpc.CallOption,
) error {
    // Extract peer identity
    peer, ok := peer.FromContext(ctx)
    if !ok {
        return ErrNoPeerIdentity
    }
    
    // Verify service certificate
    tlsInfo, ok := peer.AuthInfo.(credentials.TLSInfo)
    if !ok {
        return ErrNoTLSInfo
    }
    
    if err := m.verifyServiceCertificate(tlsInfo.State); err != nil {
        return err
    }
    
    // Add service identity to context
    ctx = context.WithValue(ctx, ServiceIdentityKey, tlsInfo.State.PeerCertificates[0].Subject.CommonName)
    
    return invoker(ctx, method, req, reply, cc, opts...)
}
```

### Service Authorization

Role-based access control between services:

```go
type ServiceAuthorization struct {
    // Service permissions
    permissions map[string][]Permission
    
    // Inter-service policies
    policies    map[ServicePair]AccessPolicy
    
    // Audit logging
    auditor     *SecurityAuditor
}

type ServicePair struct {
    From string
    To   string
}

type AccessPolicy struct {
    AllowedMethods []string
    RateLimit      *RateLimitConfig
    RequiredClaims []string
}

func (a *ServiceAuthorization) Authorize(ctx context.Context, service, method string) error {
    // Get calling service identity
    caller, ok := ctx.Value(ServiceIdentityKey).(string)
    if !ok {
        return ErrNoServiceIdentity
    }
    
    // Check service pair policy
    pair := ServicePair{From: caller, To: service}
    policy, exists := a.policies[pair]
    if !exists {
        a.auditor.LogDenied(caller, service, method, "no_policy")
        return ErrAccessDenied
    }
    
    // Verify method is allowed
    if !contains(policy.AllowedMethods, method) {
        a.auditor.LogDenied(caller, service, method, "method_not_allowed")
        return ErrMethodNotAllowed
    }
    
    // Check rate limits
    if policy.RateLimit != nil {
        if !policy.RateLimit.Allow(caller, method) {
            a.auditor.LogDenied(caller, service, method, "rate_limit_exceeded")
            return ErrRateLimitExceeded
        }
    }
    
    // Audit successful authorization
    a.auditor.LogAllowed(caller, service, method)
    
    return nil
}
```

## Security Zones

Process-based security zones with different privilege levels:

```go
type SecurityZone string

const (
    ZonePublic     SecurityZone = "public"     // External-facing services
    ZoneApplication SecurityZone = "application" // Core application services
    ZoneData       SecurityZone = "data"       // Data storage services
    ZoneControl    SecurityZone = "control"    // Control plane services
)

type ZonePolicy struct {
    Zone           SecurityZone
    AllowedUsers   []string
    ProcessLimits  ProcessLimits
    NetworkPolicy  NetworkPolicy
    FilePolicy     FileSystemSecurity
}

var ZonePolicies = map[SecurityZone]ZonePolicy{
    ZonePublic: {
        Zone: ZonePublic,
        ProcessLimits: ProcessLimits{
            MaxMemory: 1 * GB,
            MaxCPU:    0.5,
            MaxFiles:  1000,
        },
        NetworkPolicy: NetworkPolicy{
            AllowedPorts: []int{443, 80},
            RequireTLS:   true,
        },
        FilePolicy: FileSystemSecurity{
            ReadPaths:  []string{"/var/lib/blackhole/public"},
            WritePaths: []string{"/var/log/blackhole"},
        },
    },
    // ... other zones
}
```

## Zero-Knowledge Integration

### Supported Circuits

Privacy-preserving authentication circuits:

```go
type ZKCircuit struct {
    ID          string
    Type        CircuitType
    Parameters  map[string]interface{}
    ProverKey   []byte
    VerifierKey []byte
}

var StandardCircuits = map[string]ZKCircuit{
    "age_verification": {
        ID:   "age_verification_v1",
        Type: CircuitTypeGroth16,
        Parameters: map[string]interface{}{
            "min_age": 18,
            "max_age": 120,
        },
    },
    "kyc_verification": {
        ID:   "kyc_verification_v1",
        Type: CircuitTypePLONK,
        Parameters: map[string]interface{}{
            "accepted_providers": []string{"provider1", "provider2"},
        },
    },
    // ... other circuits
}
```

### Proof Verification Service

Dedicated service for ZK proof verification:

```go
// Runs as separate process for security isolation
type ProofVerificationService struct {
    grpcServer *grpc.Server
    verifier   *ZKVerifier
    circuits   map[string]ZKCircuit
    auditLog   *AuditLogger
}

func (s *ProofVerificationService) VerifyProof(
    ctx context.Context,
    req *VerifyProofRequest,
) (*VerifyProofResponse, error) {
    // Get circuit
    circuit, exists := s.circuits[req.CircuitID]
    if !exists {
        return nil, ErrUnknownCircuit
    }
    
    // Verify proof
    result, err := s.verifier.Verify(
        req.Proof,
        req.PublicInputs,
        circuit.VerifierKey,
    )
    
    // Audit the verification
    s.auditLog.LogVerification(req.CircuitID, result, err)
    
    if err != nil {
        return nil, err
    }
    
    return &VerifyProofResponse{
        Valid:  result.Valid,
        Claims: result.ExtractedClaims,
    }, nil
}
```

## Security Monitoring

### Process Security Monitoring

Monitor security events across all processes:

```go
type SecurityMonitor struct {
    processes   map[string]*ProcessMonitor
    eventBus    *EventBus
    alerting    *AlertManager
    metrics     *SecurityMetrics
}

type SecurityEvent struct {
    Timestamp   time.Time
    Process     string
    EventType   SecurityEventType
    Severity    Severity
    Details     map[string]interface{}
}

func (m *SecurityMonitor) MonitorProcess(process string) {
    monitor := &ProcessMonitor{
        PID:       m.processes[process].PID,
        EventChan: make(chan SecurityEvent, 100),
    }
    
    // Monitor system calls
    go monitor.MonitorSyscalls()
    
    // Monitor network connections
    go monitor.MonitorNetwork()
    
    // Monitor file access
    go monitor.MonitorFileAccess()
    
    // Process events
    for event := range monitor.EventChan {
        m.processSecurityEvent(process, event)
    }
}

func (m *SecurityMonitor) processSecurityEvent(process string, event SecurityEvent) {
    // Update metrics
    m.metrics.RecordEvent(event)
    
    // Check security policies
    if m.isViolation(event) {
        m.alerting.SendAlert(Alert{
            Process:  process,
            Event:    event,
            Severity: event.Severity,
        })
    }
    
    // Publish to event bus
    m.eventBus.Publish(event)
}
```

### Audit Logging

Comprehensive security audit trail:

```go
type SecurityAuditor struct {
    logFile     *os.File
    encryptor   *LogEncryptor
    signer      *LogSigner
    rotation    *LogRotation
}

type AuditEntry struct {
    Timestamp   time.Time
    Process     string
    Action      string
    Subject     string
    Object      string
    Result      Result
    Details     map[string]interface{}
    Signature   []byte
}

func (a *SecurityAuditor) LogSecurityEvent(entry AuditEntry) error {
    // Add timestamp
    entry.Timestamp = time.Now()
    
    // Sign the entry
    signature, err := a.signer.Sign(entry)
    if err != nil {
        return err
    }
    entry.Signature = signature
    
    // Encrypt sensitive fields
    encrypted, err := a.encryptor.Encrypt(entry)
    if err != nil {
        return err
    }
    
    // Write to log
    return a.writeEntry(encrypted)
}
```

## Configuration

Security configuration per service:

```yaml
security:
  # Process isolation
  processes:
    identity:
      user: blackhole-identity
      group: blackhole
      capabilities:
        drop: [ALL]
        keep: [NET_BIND_SERVICE]
      seccomp: default
      
    storage:
      user: blackhole-storage
      group: blackhole
      capabilities:
        drop: [ALL]
        keep: [SYS_ADMIN]  # For mounting
      
  # Network policies
  network:
    require_tls: true
    min_tls_version: "1.3"
    allowed_ciphers:
      - TLS_AES_256_GCM_SHA384
      - TLS_CHACHA20_POLY1305_SHA256
      
  # Service authentication
  service_auth:
    ca_cert: /etc/blackhole/ca.crt
    cert_dir: /etc/blackhole/certs
    key_dir: /etc/blackhole/keys
    
  # Zero-knowledge circuits
  zk_circuits:
    - id: age_verification_v1
      enabled: true
      cache_proofs: true
      cache_duration: 1h
      
  # Security monitoring
  monitoring:
    enable_syscall_monitoring: true
    enable_network_monitoring: true
    enable_file_monitoring: true
    alert_webhook: https://security.example.com/alerts
```

## Best Practices

1. **Process Isolation**: Run each service with minimal privileges
2. **Network Security**: Always use mTLS for inter-process communication via gRPC
3. **Resource Limits**: Apply strict resource limits to prevent DoS
4. **Audit Everything**: Log all security-relevant events
5. **Defense in Depth**: Multiple security layers at different levels
6. **Zero Trust**: Verify everything, trust nothing

## Conclusion

The process-based security architecture provides:

- **Strong Isolation**: OS-level process boundaries
- **Secure Communication**: mTLS between all services
- **Privacy Protection**: Zero-knowledge proofs for users
- **Comprehensive Monitoring**: Security events across all processes
- **Flexible Policies**: Per-service security configuration

This design ensures robust security while maintaining operational simplicity and supporting advanced privacy features.