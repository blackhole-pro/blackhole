# Complete Service Implementation Guide

This guide provides a comprehensive step-by-step approach to implementing a new service in the Blackhole distributed content sharing platform. Each service follows the subprocess architecture pattern where services run as independent OS processes communicating via gRPC.

## Prerequisites

- Go 1.21+ installed
- Protocol Buffers compiler (`protoc`) with Go plugins
- Understanding of gRPC and service mesh concepts
- Familiarity with the Blackhole architecture

## Step 1: Directory Structure Setup

Create the service directory structure following the established pattern:

```bash
mkdir -p internal/services/myservice/{types,testing}
mkdir -p configs
```

The standard service structure:
```
internal/services/myservice/
├── main.go           # Service entry point
├── go.mod            # Service module definition  
├── service.go        # Core service implementation
├── handlers.go       # gRPC request handlers
├── clients.go        # Inter-service communication clients
├── config.go         # Configuration management
├── middleware.go     # gRPC middleware stack
├── types/
│   ├── types.go      # Service-specific types
│   └── errors.go     # Service error definitions
└── testing/
    └── mocks.go      # Test mocks and utilities
```

## Step 2: Protocol Buffer Definitions

Create the gRPC service definition in `internal/rpc/proto/myservice/v1/service.proto`:

```protobuf
syntax = "proto3";

package myservice.v1;

option go_package = "github.com/blackhole/blackhole/internal/rpc/gen/myservice/v1";

import "common/v1/common.proto";

// Define your service interface
service MyService {
  rpc GetStatus(GetStatusRequest) returns (GetStatusResponse);
  rpc StartOperation(StartOperationRequest) returns (StartOperationResponse);
  rpc StopOperation(StopOperationRequest) returns (StopOperationResponse);
  // Add service-specific methods
}

// Request/Response message definitions
message GetStatusRequest {}

message GetStatusResponse {
  string status = 1;
  int64 uptime_seconds = 2;
  repeated string active_operations = 3;
}

message StartOperationRequest {
  string operation_id = 1;
  map<string, string> parameters = 2;
}

message StartOperationResponse {
  bool success = 1;
  string message = 2;
}

message StopOperationRequest {
  string operation_id = 1;
}

message StopOperationResponse {
  bool success = 1;
  string message = 2;
}
```

## Step 3: Service-Specific Types

Create `internal/services/myservice/types/types.go`:

```go
package types

import (
    "time"
)

// ServiceStatus represents the current state of the service
type ServiceStatus int

const (
    StatusUnknown ServiceStatus = iota
    StatusStarting
    StatusRunning
    StatusStopping
    StatusStopped
    StatusError
)

// ServiceConfig holds all configuration for the service
type ServiceConfig struct {
    // Network configuration
    Network NetworkConfig `yaml:"network"`
    
    // Service-specific settings
    Operations OperationsConfig `yaml:"operations"`
    
    // Client configurations for other services
    Clients ClientsConfig `yaml:"clients"`
    
    // Security settings
    Security SecurityConfig `yaml:"security"`
}

type NetworkConfig struct {
    ListenAddress string `yaml:"listen_address"`
    Port          int    `yaml:"port"`
    MaxConnections int   `yaml:"max_connections"`
}

type OperationsConfig struct {
    MaxConcurrent int           `yaml:"max_concurrent"`
    Timeout       time.Duration `yaml:"timeout"`
    RetryAttempts int           `yaml:"retry_attempts"`
}

type ClientsConfig struct {
    Identity IdentityClientConfig `yaml:"identity"`
    Storage  StorageClientConfig  `yaml:"storage"`
}

type IdentityClientConfig struct {
    SocketPath string        `yaml:"socket_path"`
    Timeout    time.Duration `yaml:"timeout"`
}

type StorageClientConfig struct {
    SocketPath string        `yaml:"socket_path"`
    Timeout    time.Duration `yaml:"timeout"`
}

type SecurityConfig struct {
    EnableTLS      bool   `yaml:"enable_tls"`
    CertFile       string `yaml:"cert_file"`
    KeyFile        string `yaml:"key_file"`
    RequireAuth    bool   `yaml:"require_auth"`
    AllowedOrigins []string `yaml:"allowed_origins"`
}

// DefaultServiceConfig returns a configuration with sensible defaults
func DefaultServiceConfig() *ServiceConfig {
    return &ServiceConfig{
        Network: NetworkConfig{
            ListenAddress:  "127.0.0.1",
            Port:          8080,
            MaxConnections: 100,
        },
        Operations: OperationsConfig{
            MaxConcurrent: 10,
            Timeout:       30 * time.Second,
            RetryAttempts: 3,
        },
        Clients: ClientsConfig{
            Identity: IdentityClientConfig{
                SocketPath: "/tmp/blackhole/identity.sock",
                Timeout:    5 * time.Second,
            },
            Storage: StorageClientConfig{
                SocketPath: "/tmp/blackhole/storage.sock",
                Timeout:    10 * time.Second,
            },
        },
        Security: SecurityConfig{
            EnableTLS:   false,
            RequireAuth: false,
        },
    }
}
```

## Step 4: Service Error Definitions

Create `internal/services/myservice/types/errors.go`:

```go
package types

import (
    "fmt"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// ServiceError represents service-specific errors
type ServiceError struct {
    Code       ErrorCode
    Message    string
    Cause      error
    Retryable  bool
}

func (e *ServiceError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Cause)
    }
    return e.Message
}

// ErrorCode represents specific error types
type ErrorCode int

const (
    ErrUnknown ErrorCode = iota
    ErrInvalidRequest
    ErrOperationNotFound
    ErrOperationAlreadyExists
    ErrOperationInProgress
    ErrOperationFailed
    ErrServiceUnavailable
    ErrConfigurationError
    ErrAuthenticationRequired
    ErrPermissionDenied
    ErrRateLimitExceeded
    ErrResourceExhausted
    ErrTimeout
    ErrNetworkError
    ErrInternalError
)

// Error creation helpers
func NewInvalidRequestError(message string) *ServiceError {
    return &ServiceError{
        Code:      ErrInvalidRequest,
        Message:   message,
        Retryable: false,
    }
}

func NewOperationNotFoundError(operationID string) *ServiceError {
    return &ServiceError{
        Code:      ErrOperationNotFound,
        Message:   fmt.Sprintf("operation not found: %s", operationID),
        Retryable: false,
    }
}

func NewInternalError(message string, cause error) *ServiceError {
    return &ServiceError{
        Code:      ErrInternalError,
        Message:   message,
        Cause:     cause,
        Retryable: true,
    }
}

// ToGRPCStatus converts service errors to gRPC status
func (e *ServiceError) ToGRPCStatus() *status.Status {
    var code codes.Code
    
    switch e.Code {
    case ErrInvalidRequest:
        code = codes.InvalidArgument
    case ErrOperationNotFound:
        code = codes.NotFound
    case ErrOperationAlreadyExists:
        code = codes.AlreadyExists
    case ErrAuthenticationRequired:
        code = codes.Unauthenticated
    case ErrPermissionDenied:
        code = codes.PermissionDenied
    case ErrRateLimitExceeded:
        code = codes.ResourceExhausted
    case ErrTimeout:
        code = codes.DeadlineExceeded
    case ErrServiceUnavailable:
        code = codes.Unavailable
    default:
        code = codes.Internal
    }
    
    return status.New(code, e.Message)
}
```

## Step 5: Configuration Management

Create `internal/services/myservice/config.go`:

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    
    "gopkg.in/yaml.v3"
    "github.com/blackhole/blackhole/internal/services/myservice/types"
)

// LoadConfig loads and validates the service configuration
func LoadConfig(configPath string) (*types.ServiceConfig, error) {
    // Start with defaults
    config := types.DefaultServiceConfig()
    
    // If no config file specified, return defaults
    if configPath == "" {
        return config, nil
    }
    
    // Check if config file exists
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        return nil, fmt.Errorf("config file not found: %s", configPath)
    }
    
    // Read config file
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }
    
    // Parse YAML
    if err := yaml.Unmarshal(data, config); err != nil {
        return nil, fmt.Errorf("failed to parse config file: %w", err)
    }
    
    // Validate configuration
    if err := validateConfig(config); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }
    
    return config, nil
}

// validateConfig performs configuration validation
func validateConfig(config *types.ServiceConfig) error {
    // Validate network settings
    if config.Network.Port <= 0 || config.Network.Port > 65535 {
        return fmt.Errorf("invalid port: %d", config.Network.Port)
    }
    
    if config.Network.MaxConnections <= 0 {
        return fmt.Errorf("max_connections must be positive")
    }
    
    // Validate operations settings
    if config.Operations.MaxConcurrent <= 0 {
        return fmt.Errorf("max_concurrent must be positive")
    }
    
    if config.Operations.RetryAttempts < 0 {
        return fmt.Errorf("retry_attempts cannot be negative")
    }
    
    // Validate TLS settings if enabled
    if config.Security.EnableTLS {
        if config.Security.CertFile == "" {
            return fmt.Errorf("cert_file required when TLS is enabled")
        }
        if config.Security.KeyFile == "" {
            return fmt.Errorf("key_file required when TLS is enabled")
        }
        
        // Check if cert files exist
        if _, err := os.Stat(config.Security.CertFile); os.IsNotExist(err) {
            return fmt.Errorf("cert file not found: %s", config.Security.CertFile)
        }
        if _, err := os.Stat(config.Security.KeyFile); os.IsNotExist(err) {
            return fmt.Errorf("key file not found: %s", config.Security.KeyFile)
        }
    }
    
    // Validate client socket paths
    if config.Clients.Identity.SocketPath != "" {
        dir := filepath.Dir(config.Clients.Identity.SocketPath)
        if _, err := os.Stat(dir); os.IsNotExist(err) {
            return fmt.Errorf("identity socket directory does not exist: %s", dir)
        }
    }
    
    if config.Clients.Storage.SocketPath != "" {
        dir := filepath.Dir(config.Clients.Storage.SocketPath)
        if _, err := os.Stat(dir); os.IsNotExist(err) {
            return fmt.Errorf("storage socket directory does not exist: %s", dir)
        }
    }
    
    return nil
}
```

## Step 6: Core Service Implementation

Create `internal/services/myservice/service.go`:

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/blackhole/blackhole/internal/services/myservice/types"
)

// Service represents the core service implementation
type Service struct {
    config     *types.ServiceConfig
    status     types.ServiceStatus
    statusMu   sync.RWMutex
    
    // Service state
    operations map[string]*Operation
    operationsMu sync.RWMutex
    
    // Background workers
    workerCtx    context.Context
    workerCancel context.CancelFunc
    workerWg     sync.WaitGroup
    
    // Client connections
    identityClient IdentityClient
    storageClient  StorageClient
    
    startTime time.Time
}

type Operation struct {
    ID         string
    Status     string
    Parameters map[string]string
    StartTime  time.Time
    ctx        context.Context
    cancel     context.CancelFunc
}

// NewService creates a new service instance
func NewService(config *types.ServiceConfig) *Service {
    ctx, cancel := context.WithCancel(context.Background())
    
    return &Service{
        config:       config,
        status:       types.StatusStopped,
        operations:   make(map[string]*Operation),
        workerCtx:    ctx,
        workerCancel: cancel,
        startTime:    time.Now(),
    }
}

// Start initializes and starts the service
func (s *Service) Start(ctx context.Context) error {
    s.statusMu.Lock()
    defer s.statusMu.Unlock()
    
    if s.status != types.StatusStopped {
        return fmt.Errorf("service already started")
    }
    
    s.status = types.StatusStarting
    
    // Initialize client connections
    if err := s.initializeClients(); err != nil {
        s.status = types.StatusError
        return fmt.Errorf("failed to initialize clients: %w", err)
    }
    
    // Start background workers
    s.startBackgroundWorkers()
    
    s.status = types.StatusRunning
    return nil
}

// Stop gracefully shuts down the service
func (s *Service) Stop(ctx context.Context) error {
    s.statusMu.Lock()
    defer s.statusMu.Unlock()
    
    if s.status == types.StatusStopped {
        return nil
    }
    
    s.status = types.StatusStopping
    
    // Cancel all operations
    s.operationsMu.Lock()
    for _, op := range s.operations {
        op.cancel()
    }
    s.operationsMu.Unlock()
    
    // Stop background workers
    s.workerCancel()
    s.workerWg.Wait()
    
    // Close client connections
    s.closeClients()
    
    s.status = types.StatusStopped
    return nil
}

// GetStatus returns current service status
func (s *Service) GetStatus() (types.ServiceStatus, time.Duration, []string) {
    s.statusMu.RLock()
    status := s.status
    s.statusMu.RUnlock()
    
    uptime := time.Since(s.startTime)
    
    s.operationsMu.RLock()
    var activeOps []string
    for id, op := range s.operations {
        if op.Status == "running" {
            activeOps = append(activeOps, id)
        }
    }
    s.operationsMu.RUnlock()
    
    return status, uptime, activeOps
}

// StartOperation begins a new operation
func (s *Service) StartOperation(operationID string, parameters map[string]string) error {
    s.operationsMu.Lock()
    defer s.operationsMu.Unlock()
    
    // Check if operation already exists
    if _, exists := s.operations[operationID]; exists {
        return types.NewOperationAlreadyExistsError(operationID)
    }
    
    // Check concurrent operation limit
    activeCount := 0
    for _, op := range s.operations {
        if op.Status == "running" {
            activeCount++
        }
    }
    
    if activeCount >= s.config.Operations.MaxConcurrent {
        return &types.ServiceError{
            Code:      types.ErrResourceExhausted,
            Message:   "maximum concurrent operations reached",
            Retryable: true,
        }
    }
    
    // Create operation
    ctx, cancel := context.WithTimeout(s.workerCtx, s.config.Operations.Timeout)
    operation := &Operation{
        ID:         operationID,
        Status:     "running",
        Parameters: parameters,
        StartTime:  time.Now(),
        ctx:        ctx,
        cancel:     cancel,
    }
    
    s.operations[operationID] = operation
    
    // Start operation in background
    s.workerWg.Add(1)
    go s.runOperation(operation)
    
    return nil
}

// StopOperation cancels a running operation
func (s *Service) StopOperation(operationID string) error {
    s.operationsMu.Lock()
    defer s.operationsMu.Unlock()
    
    operation, exists := s.operations[operationID]
    if !exists {
        return types.NewOperationNotFoundError(operationID)
    }
    
    if operation.Status != "running" {
        return types.NewInvalidRequestError("operation is not running")
    }
    
    operation.cancel()
    operation.Status = "cancelled"
    
    return nil
}

// Private methods

func (s *Service) initializeClients() error {
    var err error
    
    // Initialize Identity client
    s.identityClient, err = NewIdentityClient(s.config.Clients.Identity)
    if err != nil {
        return fmt.Errorf("failed to create identity client: %w", err)
    }
    
    // Initialize Storage client  
    s.storageClient, err = NewStorageClient(s.config.Clients.Storage)
    if err != nil {
        return fmt.Errorf("failed to create storage client: %w", err)
    }
    
    return nil
}

func (s *Service) closeClients() {
    if s.identityClient != nil {
        s.identityClient.Close()
    }
    if s.storageClient != nil {
        s.storageClient.Close()
    }
}

func (s *Service) startBackgroundWorkers() {
    // Status monitoring worker
    s.workerWg.Add(1)
    go s.statusMonitorWorker()
    
    // Cleanup worker for completed operations
    s.workerWg.Add(1)
    go s.cleanupWorker()
}

func (s *Service) statusMonitorWorker() {
    defer s.workerWg.Done()
    
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-s.workerCtx.Done():
            return
        case <-ticker.C:
            s.performHealthCheck()
        }
    }
}

func (s *Service) cleanupWorker() {
    defer s.workerWg.Done()
    
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-s.workerCtx.Done():
            return
        case <-ticker.C:
            s.cleanupCompletedOperations()
        }
    }
}

func (s *Service) performHealthCheck() {
    // Implement health check logic
    // Check client connections, system resources, etc.
}

func (s *Service) cleanupCompletedOperations() {
    s.operationsMu.Lock()
    defer s.operationsMu.Unlock()
    
    cutoff := time.Now().Add(-1 * time.Hour)
    
    for id, op := range s.operations {
        if op.Status != "running" && op.StartTime.Before(cutoff) {
            delete(s.operations, id)
        }
    }
}

func (s *Service) runOperation(operation *Operation) {
    defer s.workerWg.Done()
    
    // Implement operation logic here
    // This is where the actual work happens
    
    select {
    case <-operation.ctx.Done():
        if operation.ctx.Err() == context.DeadlineExceeded {
            operation.Status = "timeout"
        } else {
            operation.Status = "cancelled"
        }
    case <-time.After(time.Duration(len(operation.Parameters)) * time.Second):
        // Simulate work completion
        operation.Status = "completed"
    }
}
```

## Step 7: gRPC Middleware Stack

Create `internal/services/myservice/middleware.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "runtime/debug"
    "time"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/grpc/metadata"
)

// CreateMiddlewareChain creates the complete gRPC middleware stack
func CreateMiddlewareChain() []grpc.UnaryServerInterceptor {
    return []grpc.UnaryServerInterceptor{
        recoveryInterceptor,
        loggingInterceptor,
        metricsInterceptor,
        rateLimitInterceptor,
        authInterceptor,
        tracingInterceptor,
    }
}

// recoveryInterceptor recovers from panics and converts them to gRPC errors
func recoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Panic recovered in %s: %v\nStack: %s", info.FullMethod, r, debug.Stack())
        }
    }()
    
    return handler(ctx, req)
}

// loggingInterceptor logs all gRPC requests and responses
func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    start := time.Now()
    
    log.Printf("gRPC request started: method=%s", info.FullMethod)
    
    resp, err := handler(ctx, req)
    
    duration := time.Since(start)
    status := "OK"
    if err != nil {
        status = "ERROR"
    }
    
    log.Printf("gRPC request completed: method=%s status=%s duration=%v", 
        info.FullMethod, status, duration)
    
    return resp, err
}

// metricsInterceptor collects metrics for gRPC requests
func metricsInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    start := time.Now()
    
    resp, err := handler(ctx, req)
    
    duration := time.Since(start)
    
    // Record metrics (implement with your metrics system)
    recordRequestMetrics(info.FullMethod, err, duration)
    
    return resp, err
}

// rateLimitInterceptor implements rate limiting per client
func rateLimitInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    // Extract client identifier from metadata
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.InvalidArgument, "missing metadata")
    }
    
    clientID := "unknown"
    if ids := md.Get("client-id"); len(ids) > 0 {
        clientID = ids[0]
    }
    
    // Check rate limit (implement with your rate limiter)
    if !checkRateLimit(clientID, info.FullMethod) {
        return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
    }
    
    return handler(ctx, req)
}

// authInterceptor handles authentication and authorization
func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    // Skip auth for health check methods
    if info.FullMethod == "/myservice.v1.MyService/GetStatus" {
        return handler(ctx, req)
    }
    
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "missing metadata")
    }
    
    // Extract and validate authentication token
    tokens := md.Get("authorization")
    if len(tokens) == 0 {
        return nil, status.Error(codes.Unauthenticated, "missing authorization token")
    }
    
    token := tokens[0]
    if !validateToken(token) {
        return nil, status.Error(codes.Unauthenticated, "invalid token")
    }
    
    // Add user context for downstream handlers
    userCtx := context.WithValue(ctx, "user_id", extractUserID(token))
    
    return handler(userCtx, req)
}

// tracingInterceptor adds distributed tracing support
func tracingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    // Extract trace context from metadata
    md, ok := metadata.FromIncomingContext(ctx)
    if ok {
        if traceIDs := md.Get("trace-id"); len(traceIDs) > 0 {
            ctx = context.WithValue(ctx, "trace_id", traceIDs[0])
        }
        if spanIDs := md.Get("span-id"); len(spanIDs) > 0 {
            ctx = context.WithValue(ctx, "span_id", spanIDs[0])
        }
    }
    
    // Create new span for this request
    spanCtx := createSpan(ctx, info.FullMethod)
    defer finishSpan(spanCtx)
    
    return handler(spanCtx, req)
}

// Helper functions (implement these based on your specific systems)

func recordRequestMetrics(method string, err error, duration time.Duration) {
    // Implement metrics recording
    // Example: increment counters, record histograms, etc.
}

func checkRateLimit(clientID, method string) bool {
    // Implement rate limiting logic
    // Return false if rate limit exceeded
    return true
}

func validateToken(token string) bool {
    // Implement token validation
    // Check signature, expiration, etc.
    return token != ""
}

func extractUserID(token string) string {
    // Extract user ID from validated token
    return "user123"
}

func createSpan(ctx context.Context, operationName string) context.Context {
    // Implement tracing span creation
    return ctx
}

func finishSpan(ctx context.Context) {
    // Implement tracing span completion
}
```

## Step 8: gRPC Request Handlers

Create `internal/services/myservice/handlers.go`:

```go
package main

import (
    "context"
    "fmt"
    
    pb "github.com/blackhole/blackhole/internal/rpc/gen/myservice/v1"
    "github.com/blackhole/blackhole/internal/services/myservice/types"
)

// MyServiceServer implements the gRPC service interface
type MyServiceServer struct {
    pb.UnimplementedMyServiceServer
    service *Service
}

// NewMyServiceServer creates a new gRPC server instance
func NewMyServiceServer(service *Service) *MyServiceServer {
    return &MyServiceServer{
        service: service,
    }
}

// GetStatus returns the current service status
func (s *MyServiceServer) GetStatus(ctx context.Context, req *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
    status, uptime, activeOps := s.service.GetStatus()
    
    var statusStr string
    switch status {
    case types.StatusRunning:
        statusStr = "running"
    case types.StatusStarting:
        statusStr = "starting"
    case types.StatusStopping:
        statusStr = "stopping"
    case types.StatusStopped:
        statusStr = "stopped"
    case types.StatusError:
        statusStr = "error"
    default:
        statusStr = "unknown"
    }
    
    return &pb.GetStatusResponse{
        Status:           statusStr,
        UptimeSeconds:    int64(uptime.Seconds()),
        ActiveOperations: activeOps,
    }, nil
}

// StartOperation begins a new operation
func (s *MyServiceServer) StartOperation(ctx context.Context, req *pb.StartOperationRequest) (*pb.StartOperationResponse, error) {
    // Validate request
    if req.OperationId == "" {
        return nil, types.NewInvalidRequestError("operation_id is required").ToGRPCStatus().Err()
    }
    
    // Start the operation
    err := s.service.StartOperation(req.OperationId, req.Parameters)
    if err != nil {
        if serviceErr, ok := err.(*types.ServiceError); ok {
            return &pb.StartOperationResponse{
                Success: false,
                Message: serviceErr.Message,
            }, serviceErr.ToGRPCStatus().Err()
        }
        
        return &pb.StartOperationResponse{
            Success: false,
            Message: "internal error",
        }, types.NewInternalError("failed to start operation", err).ToGRPCStatus().Err()
    }
    
    return &pb.StartOperationResponse{
        Success: true,
        Message: fmt.Sprintf("operation %s started successfully", req.OperationId),
    }, nil
}

// StopOperation cancels a running operation
func (s *MyServiceServer) StopOperation(ctx context.Context, req *pb.StopOperationRequest) (*pb.StopOperationResponse, error) {
    // Validate request
    if req.OperationId == "" {
        return nil, types.NewInvalidRequestError("operation_id is required").ToGRPCStatus().Err()
    }
    
    // Stop the operation
    err := s.service.StopOperation(req.OperationId)
    if err != nil {
        if serviceErr, ok := err.(*types.ServiceError); ok {
            return &pb.StopOperationResponse{
                Success: false,
                Message: serviceErr.Message,
            }, serviceErr.ToGRPCStatus().Err()
        }
        
        return &pb.StopOperationResponse{
            Success: false,
            Message: "internal error",
        }, types.NewInternalError("failed to stop operation", err).ToGRPCStatus().Err()
    }
    
    return &pb.StopOperationResponse{
        Success: true,
        Message: fmt.Sprintf("operation %s stopped successfully", req.OperationId),
    }, nil
}
```

## Step 9: Inter-Service Communication Clients

Create `internal/services/myservice/clients.go`:

```go
package main

import (
    "context"
    "fmt"
    "net"
    "time"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    
    identitypb "github.com/blackhole/blackhole/internal/rpc/gen/identity/auth/v1"
    storagepb "github.com/blackhole/blackhole/internal/rpc/gen/storage/v1"
    "github.com/blackhole/blackhole/internal/services/myservice/types"
)

// IdentityClient handles communication with the Identity service
type IdentityClient interface {
    ValidateToken(ctx context.Context, token string) (bool, error)
    GetUserInfo(ctx context.Context, userID string) (*UserInfo, error)
    Close() error
}

type identityClient struct {
    conn   *grpc.ClientConn
    client identitypb.AuthServiceClient
    config types.IdentityClientConfig
}

type UserInfo struct {
    ID    string
    Email string
    Roles []string
}

// NewIdentityClient creates a new Identity service client
func NewIdentityClient(config types.IdentityClientConfig) (IdentityClient, error) {
    // Create connection options
    opts := []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithTimeout(config.Timeout),
    }
    
    // Determine connection target (Unix socket or TCP)
    var target string
    if config.SocketPath != "" {
        target = fmt.Sprintf("unix://%s", config.SocketPath)
        opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
            return net.Dial("unix", config.SocketPath)
        }))
    } else {
        return nil, fmt.Errorf("socket_path is required for identity client")
    }
    
    // Establish connection
    conn, err := grpc.Dial(target, opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to identity service: %w", err)
    }
    
    return &identityClient{
        conn:   conn,
        client: identitypb.NewAuthServiceClient(conn),
        config: config,
    }, nil
}

func (c *identityClient) ValidateToken(ctx context.Context, token string) (bool, error) {
    req := &identitypb.ValidateTokenRequest{
        Token: token,
    }
    
    resp, err := c.client.ValidateToken(ctx, req)
    if err != nil {
        return false, fmt.Errorf("failed to validate token: %w", err)
    }
    
    return resp.Valid, nil
}

func (c *identityClient) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
    req := &identitypb.GetUserRequest{
        UserId: userID,
    }
    
    resp, err := c.client.GetUser(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to get user info: %w", err)
    }
    
    return &UserInfo{
        ID:    resp.User.Id,
        Email: resp.User.Email,
        Roles: resp.User.Roles,
    }, nil
}

func (c *identityClient) Close() error {
    return c.conn.Close()
}

// StorageClient handles communication with the Storage service
type StorageClient interface {
    StoreData(ctx context.Context, data []byte) (string, error)
    RetrieveData(ctx context.Context, id string) ([]byte, error)
    Close() error
}

type storageClient struct {
    conn   *grpc.ClientConn
    client storagepb.StorageServiceClient
    config types.StorageClientConfig
}

// NewStorageClient creates a new Storage service client
func NewStorageClient(config types.StorageClientConfig) (StorageClient, error) {
    // Create connection options
    opts := []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithTimeout(config.Timeout),
    }
    
    // Determine connection target (Unix socket or TCP)
    var target string
    if config.SocketPath != "" {
        target = fmt.Sprintf("unix://%s", config.SocketPath)
        opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
            return net.Dial("unix", config.SocketPath)
        }))
    } else {
        return nil, fmt.Errorf("socket_path is required for storage client")
    }
    
    // Establish connection
    conn, err := grpc.Dial(target, opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to storage service: %w", err)
    }
    
    return &storageClient{
        conn:   conn,
        client: storagepb.NewStorageServiceClient(conn),
        config: config,
    }, nil
}

func (c *storageClient) StoreData(ctx context.Context, data []byte) (string, error) {
    req := &storagepb.StoreRequest{
        Data: data,
    }
    
    resp, err := c.client.Store(ctx, req)
    if err != nil {
        return "", fmt.Errorf("failed to store data: %w", err)
    }
    
    return resp.Id, nil
}

func (c *storageClient) RetrieveData(ctx context.Context, id string) ([]byte, error) {
    req := &storagepb.RetrieveRequest{
        Id: id,
    }
    
    resp, err := c.client.Retrieve(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to retrieve data: %w", err)
    }
    
    return resp.Data, nil
}

func (c *storageClient) Close() error {
    return c.conn.Close()
}
```

## Step 10: Service Entry Point

Create `internal/services/myservice/main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"
    
    pb "github.com/blackhole/blackhole/internal/rpc/gen/myservice/v1"
)

func main() {
    // Load configuration
    configPath := os.Getenv("CONFIG_PATH")
    if configPath == "" {
        configPath = "configs/myservice.yaml"
    }
    
    config, err := LoadConfig(configPath)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Create service instance
    service := NewService(config)
    
    // Start service
    ctx := context.Background()
    if err := service.Start(ctx); err != nil {
        log.Fatalf("Failed to start service: %v", err)
    }
    
    // Create gRPC server
    opts := []grpc.ServerOption{
        grpc.ChainUnaryInterceptor(CreateMiddlewareChain()...),
    }
    
    server := grpc.NewServer(opts...)
    
    // Register service implementation
    pb.RegisterMyServiceServer(server, NewMyServiceServer(service))
    
    // Enable reflection for debugging
    reflection.Register(server)
    
    // Setup listeners
    var wg sync.WaitGroup
    
    // Unix socket listener
    socketPath := "/tmp/blackhole/myservice.sock"
    if err := os.MkdirAll("/tmp/blackhole", 0755); err != nil {
        log.Fatalf("Failed to create socket directory: %v", err)
    }
    
    // Remove existing socket file
    if err := os.RemoveAll(socketPath); err != nil {
        log.Printf("Warning: failed to remove existing socket: %v", err)
    }
    
    unixListener, err := net.Listen("unix", socketPath)
    if err != nil {
        log.Fatalf("Failed to listen on Unix socket: %v", err)
    }
    
    // TCP listener
    tcpAddr := fmt.Sprintf("%s:%d", config.Network.ListenAddress, config.Network.Port)
    tcpListener, err := net.Listen("tcp", tcpAddr)
    if err != nil {
        log.Fatalf("Failed to listen on TCP: %v", err)
    }
    
    // Start servers
    wg.Add(2)
    
    go func() {
        defer wg.Done()
        log.Printf("Starting gRPC server on Unix socket: %s", socketPath)
        if err := server.Serve(unixListener); err != nil {
            log.Printf("Unix socket server error: %v", err)
        }
    }()
    
    go func() {
        defer wg.Done()
        log.Printf("Starting gRPC server on TCP: %s", tcpAddr)
        if err := server.Serve(tcpListener); err != nil {
            log.Printf("TCP server error: %v", err)
        }
    }()
    
    // Setup graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    // Wait for shutdown signal
    <-sigChan
    log.Println("Shutdown signal received, stopping service...")
    
    // Graceful shutdown
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Stop gRPC server
    done := make(chan struct{})
    go func() {
        server.GracefulStop()
        close(done)
    }()
    
    select {
    case <-done:
        log.Println("gRPC server stopped gracefully")
    case <-shutdownCtx.Done():
        log.Println("Shutdown timeout, forcing stop")
        server.Stop()
    }
    
    // Stop service
    if err := service.Stop(shutdownCtx); err != nil {
        log.Printf("Error stopping service: %v", err)
    }
    
    // Wait for all goroutines
    wg.Wait()
    
    // Cleanup socket file
    os.RemoveAll(socketPath)
    
    log.Println("Service shutdown complete")
}
```

## Step 11: Test Mocks and Utilities

Create `internal/services/myservice/testing/mocks.go`:

```go
package testing

import (
    "context"
    "sync"
    "time"
    
    "github.com/blackhole/blackhole/internal/services/myservice/types"
)

// MockService provides a mock implementation for testing
type MockService struct {
    mu         sync.RWMutex
    status     types.ServiceStatus
    operations map[string]*MockOperation
    startTime  time.Time
}

type MockOperation struct {
    ID         string
    Status     string
    Parameters map[string]string
    StartTime  time.Time
}

// NewMockService creates a new mock service
func NewMockService() *MockService {
    return &MockService{
        status:     types.StatusStopped,
        operations: make(map[string]*MockOperation),
        startTime:  time.Now(),
    }
}

func (m *MockService) Start(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.status = types.StatusRunning
    m.startTime = time.Now()
    return nil
}

func (m *MockService) Stop(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.status = types.StatusStopped
    return nil
}

func (m *MockService) GetStatus() (types.ServiceStatus, time.Duration, []string) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    uptime := time.Since(m.startTime)
    var activeOps []string
    
    for id, op := range m.operations {
        if op.Status == "running" {
            activeOps = append(activeOps, id)
        }
    }
    
    return m.status, uptime, activeOps
}

func (m *MockService) StartOperation(operationID string, parameters map[string]string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if _, exists := m.operations[operationID]; exists {
        return &types.ServiceError{
            Code:    types.ErrOperationAlreadyExists,
            Message: "operation already exists",
        }
    }
    
    m.operations[operationID] = &MockOperation{
        ID:         operationID,
        Status:     "running",
        Parameters: parameters,
        StartTime:  time.Now(),
    }
    
    return nil
}

func (m *MockService) StopOperation(operationID string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    operation, exists := m.operations[operationID]
    if !exists {
        return &types.ServiceError{
            Code:    types.ErrOperationNotFound,
            Message: "operation not found",
        }
    }
    
    operation.Status = "stopped"
    return nil
}

// MockIdentityClient provides a mock implementation for testing
type MockIdentityClient struct {
    ValidTokens map[string]bool
    Users       map[string]*MockUserInfo
}

type MockUserInfo struct {
    ID    string
    Email string
    Roles []string
}

func NewMockIdentityClient() *MockIdentityClient {
    return &MockIdentityClient{
        ValidTokens: make(map[string]bool),
        Users:       make(map[string]*MockUserInfo),
    }
}

func (m *MockIdentityClient) ValidateToken(ctx context.Context, token string) (bool, error) {
    return m.ValidTokens[token], nil
}

func (m *MockIdentityClient) GetUserInfo(ctx context.Context, userID string) (*MockUserInfo, error) {
    user, exists := m.Users[userID]
    if !exists {
        return nil, &types.ServiceError{
            Code:    types.ErrOperationNotFound,
            Message: "user not found",
        }
    }
    return user, nil
}

func (m *MockIdentityClient) Close() error {
    return nil
}

// MockStorageClient provides a mock implementation for testing
type MockStorageClient struct {
    mu   sync.RWMutex
    data map[string][]byte
}

func NewMockStorageClient() *MockStorageClient {
    return &MockStorageClient{
        data: make(map[string][]byte),
    }
}

func (m *MockStorageClient) StoreData(ctx context.Context, data []byte) (string, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    id := generateID()
    m.data[id] = data
    return id, nil
}

func (m *MockStorageClient) RetrieveData(ctx context.Context, id string) ([]byte, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    data, exists := m.data[id]
    if !exists {
        return nil, &types.ServiceError{
            Code:    types.ErrOperationNotFound,
            Message: "data not found",
        }
    }
    
    return data, nil
}

func (m *MockStorageClient) Close() error {
    return nil
}

// Helper functions

func generateID() string {
    return "mock-id-" + time.Now().Format("20060102-150405")
}

// Test utilities

func CreateTestConfig() *types.ServiceConfig {
    config := types.DefaultServiceConfig()
    config.Network.Port = 9999  // Use test port
    return config
}

func WaitForStatus(service *MockService, expectedStatus types.ServiceStatus, timeout time.Duration) bool {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        status, _, _ := service.GetStatus()
        if status == expectedStatus {
            return true
        }
        time.Sleep(10 * time.Millisecond)
    }
    return false
}
```

## Step 12: Build System Integration

Add your service to the `Makefile`:

```makefile
# Add to the main Makefile

# Service-specific targets
myservice:
	@echo "Building MyService..."
	cd internal/services/myservice && go build -o ../../../bin/myservice ./...

myservice-test:
	@echo "Testing MyService..."
	cd internal/services/myservice && go test -v ./...

# Add to build-services target
build-services: identity storage node ledger indexer social analytics telemetry wallet myservice

# Add to test-services target  
test-services: identity-test storage-test node-test myservice-test

# Add to clean target
clean:
	rm -rf bin/
	rm -rf sockets/
	# ... existing clean commands
```

Create a service-specific go.mod file:

```bash
# Create internal/services/myservice/go.mod
cd internal/services/myservice
go mod init github.com/blackhole/blackhole/internal/services/myservice
```

Update the workspace `go.work` file:

```go
go 1.21

use (
    .
    internal/services/identity
    internal/services/storage
    internal/services/myservice
    // ... other services
)
```

## Step 13: Configuration File

Create `configs/myservice.yaml`:

```yaml
# MyService Configuration
network:
  listen_address: "127.0.0.1"
  port: 8080
  max_connections: 100

operations:
  max_concurrent: 10
  timeout: "30s"
  retry_attempts: 3

clients:
  identity:
    socket_path: "/tmp/blackhole/identity.sock"
    timeout: "5s"
  storage:
    socket_path: "/tmp/blackhole/storage.sock"
    timeout: "10s"

security:
  enable_tls: false
  cert_file: ""
  key_file: ""
  require_auth: false
  allowed_origins: []

# Service-specific configuration
myservice_specific:
  feature_enabled: true
  cache_size: 1000
  worker_count: 4
```

## Next Steps

After implementing these components:

1. **Generate Protocol Buffers**: Run `make generate-proto` to create Go files from .proto definitions

2. **Build the Service**: Run `make myservice` to build your service binary

3. **Test the Implementation**: Run `make myservice-test` to execute unit tests

4. **Integration Testing**: Add integration tests in `test/integration/`

5. **Documentation**: Update service-specific documentation in `docs/`

6. **Configuration**: Add service to main `blackhole.yaml` configuration

7. **Service Registration**: Register with the orchestrator for automatic lifecycle management

This guide provides a complete foundation for implementing services in the Blackhole platform. Each component follows established patterns and integrates seamlessly with the subprocess architecture.