# Service Interface Contracts

This document defines the interface contracts for all service subprocesses within the Blackhole platform.

## Core Service Interfaces

### Base Service Process Interface

All service subprocesses implement this base interface:

```go
package services

import (
    "context"
    "google.golang.org/grpc"
)

// ServiceProcess represents a service subprocess
type ServiceProcess interface {
    // Lifecycle management
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Service identification
    Name() string
    Version() string
    PID() int
    
    // Health monitoring via gRPC
    Health() HealthStatus
    Ready() bool
    
    // Configuration
    Configure(config interface{}) error
    
    // RPC server setup
    RegisterGRPCHandlers(server *grpc.Server) error
    GetRPCPort() int
    GetUnixSocket() string
}

// HealthStatus represents service health
type HealthStatus struct {
    Status    string            `json:"status"` // healthy, degraded, unhealthy
    Message   string            `json:"message"`
    Timestamp time.Time         `json:"timestamp"`
    PID       int               `json:"pid"`
    Details   map[string]string `json:"details"`
}
```

### Process Registry Interface

```go
package services

type ProcessRegistry interface {
    // Process registration
    RegisterProcess(service string, info ProcessInfo) error
    UnregisterProcess(pid int) error
    
    // Process discovery
    GetProcess(service string) *ProcessInfo
    GetProcessByPID(pid int) *ProcessInfo
    ListProcesses() []ProcessInfo
    
    // Health monitoring
    UpdateHealth(pid int, status HealthStatus) error
    GetProcessHealth(pid int) *HealthStatus
}

type ProcessInfo struct {
    Service    string
    PID        int
    StartTime  time.Time
    UnixSocket string
    RPCPort    int
    Status     ProcessStatus
    Resources  ResourceUsage
}

type ProcessStatus int
const (
    ProcessStarting ProcessStatus = iota
    ProcessRunning
    ProcessStopping
    ProcessStopped
    ProcessFailed
)
```

## Identity Service Interface

```go
package identity

import (
    "context"
    "github.com/blackhole/blackhole/pkg/types"
)

// IdentityService manages decentralized identities (subprocess)
type IdentityService interface {
    services.ServiceProcess
    
    // DID operations
    CreateDID(ctx context.Context, params CreateDIDParams) (*types.DID, error)
    ResolveDID(ctx context.Context, did string) (*types.DIDDocument, error)
    UpdateDID(ctx context.Context, did string, updates UpdateDIDParams) error
    RevokeDID(ctx context.Context, did string) error
    
    // Authentication
    Authenticate(ctx context.Context, credentials AuthCredentials) (*AuthToken, error)
    ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
    RefreshToken(ctx context.Context, refreshToken string) (*AuthToken, error)
    
    // Authorization
    CheckPermission(ctx context.Context, subject, resource, action string) (bool, error)
    GrantPermission(ctx context.Context, grant PermissionGrant) error
    RevokePermission(ctx context.Context, revoke PermissionRevoke) error
    
    // Registry operations
    RegisterService(ctx context.Context, service ServiceRegistration) error
    LookupService(ctx context.Context, query ServiceQuery) ([]ServiceRecord, error)
    UpdateService(ctx context.Context, serviceID string, updates ServiceUpdate) error
    
    // Signing operations
    SignTransaction(ctx context.Context, did string, transaction interface{}) ([]byte, error)
    SignMessage(ctx context.Context, did string, message []byte) ([]byte, error)
    VerifySignature(ctx context.Context, did string, message []byte, signature []byte) (bool, error)
    
    // Wallet access
    GrantWalletAccess(ctx context.Context, did string, walletID types.WalletID) (*WalletAccessToken, error)
    RevokeWalletAccess(ctx context.Context, did string, walletID types.WalletID) error
    
    // Events
    SubscribeIdentityEvents(ctx context.Context, filter EventFilter) (<-chan IdentityEvent, error)
}

// Types
type CreateDIDParams struct {
    Method       string                 `json:"method"`
    PublicKey    []byte                 `json:"public_key"`
    ServiceEndpoints []ServiceEndpoint  `json:"service_endpoints"`
    Metadata     map[string]interface{} `json:"metadata"`
}

type AuthCredentials struct {
    DID       string `json:"did"`
    Challenge string `json:"challenge"`
    Signature []byte `json:"signature"`
}

type AuthToken struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    TokenType    string    `json:"token_type"`
    ExpiresIn    int       `json:"expires_in"`
    IssuedAt     time.Time `json:"issued_at"`
}
```

## Storage Service Interface

```go
package storage

import (
    "context"
    "io"
    "github.com/blackhole/blackhole/pkg/types"
)

// StorageService manages distributed storage and content pipeline
type StorageService interface {
    services.Service
    
    // Content operations - now processed through pipeline
    Store(ctx context.Context, content io.Reader, opts StoreOptions) (*types.ContentID, error)
    Retrieve(ctx context.Context, contentID types.ContentID) (io.ReadCloser, error)
    Stream(ctx context.Context, contentID types.ContentID, writer io.Writer, start, end int64) error
    Delete(ctx context.Context, contentID types.ContentID) error
    Pin(ctx context.Context, contentID types.ContentID) error
    Unpin(ctx context.Context, contentID types.ContentID) error
    
    // Pipeline management
    GetPipelineStats(ctx context.Context) (*PipelineStats, error)
    GetPipelineConfig(ctx context.Context) (*PipelineConfig, error)
    UpdatePipelineConfig(ctx context.Context, config PipelineConfig) error
    
    // Reed-Solomon encoding
    GetEncodingParams(ctx context.Context, contentType string) (*RSEncodingParams, error)
    UpdateEncodingParams(ctx context.Context, contentType string, params RSEncodingParams) error
    GetFragmentHealth(ctx context.Context, contentID types.ContentID) (*FragmentHealth, error)
    
    // Metadata
    GetMetadata(ctx context.Context, contentID types.ContentID) (*ContentMetadata, error)
    UpdateMetadata(ctx context.Context, contentID types.ContentID, metadata MetadataUpdate) error
    
    // Provider management
    ListProviders(ctx context.Context) ([]Provider, error)
    SelectProvider(ctx context.Context, criteria ProviderCriteria) (*Provider, error)
    GetProviderStats(ctx context.Context, providerID string) (*ProviderStats, error)
    
    // Fragment management (replaces replication)
    GetFragments(ctx context.Context, contentID types.ContentID) ([]Fragment, error)
    RepairFragments(ctx context.Context, contentID types.ContentID) error
    
    // Events
    SubscribeStorageEvents(ctx context.Context, filter EventFilter) (<-chan StorageEvent, error)
}

// Types
type StoreOptions struct {
    Encryption      bool                   `json:"encryption"`
    Compression     bool                   `json:"compression"`
    ContentType     string                 `json:"content_type"` // For optimal RS encoding
    Priority        string                 `json:"priority"`    // For pipeline processing
    Metadata        map[string]interface{} `json:"metadata"`
}

type ContentMetadata struct {
    ContentID      types.ContentID        `json:"content_id"`
    Size           int64                  `json:"size"`
    Hash           string                 `json:"hash"`
    ContentType    string                 `json:"content_type"`
    CreatedAt      time.Time              `json:"created_at"`
    UpdatedAt      time.Time              `json:"updated_at"`
    Fragments      []Fragment             `json:"fragments"`    // Replaces replicas
    EncodingParams RSEncodingParams       `json:"encoding_params"`
    CustomMetadata map[string]interface{} `json:"custom_metadata"`
}

// Pipeline types
type PipelineStats struct {
    StageLatencies   map[string]time.Duration `json:"stage_latencies"`
    ThroughputBytes  int64                    `json:"throughput_bytes"`
    RSEncodingTime   time.Duration           `json:"rs_encoding_time"`
    CacheHitRate     float64                  `json:"cache_hit_rate"`
    ActiveOperations int                      `json:"active_operations"`
}

type PipelineConfig struct {
    Stages         []StageConfig          `json:"stages"`
    CacheConfig    CacheConfig            `json:"cache_config"`
    WorkerPoolSize int                    `json:"worker_pool_size"`
    QueueSize      int                    `json:"queue_size"`
}

type StageConfig struct {
    Name    string                 `json:"name"`
    Enabled bool                   `json:"enabled"`
    Params  map[string]interface{} `json:"params"`
}

type CacheConfig struct {
    ChunkCacheSize    int           `json:"chunk_cache_size"`
    MetadataCacheSize int           `json:"metadata_cache_size"`
    FragmentCacheSize int           `json:"fragment_cache_size"`
    TTL               time.Duration `json:"ttl"`
}

// Reed-Solomon types
type RSEncodingParams struct {
    DataShards   int `json:"data_shards"`    // k parameter
    ParityShards int `json:"parity_shards"`  // n-k parameter
    TotalShards  int `json:"total_shards"`   // n parameter
    ChunkSize    int `json:"chunk_size"`     // Optimal chunk size
}

type Fragment struct {
    Index    int             `json:"index"`
    Type     FragmentType    `json:"type"` // data or parity
    CID      types.ContentID `json:"cid"`
    NodeID   types.NodeID    `json:"node_id"`
    Size     int64           `json:"size"`
    ChunkIDs []string        `json:"chunk_ids"` // For streaming
}

type FragmentType string

const (
    FragmentTypeData   FragmentType = "data"
    FragmentTypeParity FragmentType = "parity"
)

type FragmentHealth struct {
    ContentID         types.ContentID          `json:"content_id"`
    HealthyFragments  int                      `json:"healthy_fragments"`
    TotalFragments    int                      `json:"total_fragments"`
    MinRequiredShards int                      `json:"min_required_shards"`
    FragmentStatus    map[int]FragmentStatus   `json:"fragment_status"`
    HealthScore       float64                  `json:"health_score"`
}

type FragmentStatus struct {
    FragmentIndex int    `json:"fragment_index"`
    Available     bool   `json:"available"`
    NodeID        string `json:"node_id"`
    LastChecked   time.Time `json:"last_checked"`
    ErrorMessage  string `json:"error_message,omitempty"`
}
```

## Node Service Interface

```go
package node

import (
    "context"
    "github.com/blackhole/blackhole/pkg/types"
)

// NodeService manages node operations and P2P networking
type NodeService interface {
    services.Service
    
    // Node operations
    GetNodeID(ctx context.Context) (types.NodeID, error)
    GetNodeInfo(ctx context.Context) (*NodeInfo, error)
    UpdateNodeStatus(ctx context.Context, status NodeStatus) error
    
    // Network operations
    Connect(ctx context.Context, peerAddr string) error
    Disconnect(ctx context.Context, peerID types.PeerID) error
    ListPeers(ctx context.Context) ([]Peer, error)
    JoinNetwork(ctx context.Context, bootstrapPeers []string) error
    LeaveNetwork(ctx context.Context) error
    
    // Messaging
    SendMessage(ctx context.Context, peerID types.PeerID, msg Message) error
    Broadcast(ctx context.Context, msg Message) error
    Subscribe(ctx context.Context, topic string) (<-chan Message, error)
    Publish(ctx context.Context, topic string, data []byte) error
    
    // Discovery
    FindNodes(ctx context.Context, criteria NodeCriteria) ([]Node, error)
    FindPeers(ctx context.Context, query PeerQuery) ([]Peer, error)
    Advertise(ctx context.Context, service string) error
    FindProviders(ctx context.Context, contentID types.ContentID) ([]types.PeerID, error)
    
    // DHT operations
    Put(ctx context.Context, key string, value []byte) error
    Get(ctx context.Context, key string) ([]byte, error)
    
    // Events
    SubscribeNodeEvents(ctx context.Context, filter EventFilter) (<-chan NodeEvent, error)
}

// Types
type NodeInfo struct {
    ID           types.NodeID           `json:"id"`
    Version      string                 `json:"version"`
    NetworkID    string                 `json:"network_id"`
    Capabilities []string               `json:"capabilities"`
    Status       NodeStatus             `json:"status"`
    StartTime    time.Time              `json:"start_time"`
    Metadata     map[string]interface{} `json:"metadata"`
}

type Peer struct {
    ID        types.PeerID           `json:"id"`
    Addresses []string               `json:"addresses"`
    Protocols []string               `json:"protocols"`
    Metadata  map[string]interface{} `json:"metadata"`
    ConnectedAt time.Time            `json:"connected_at"`
}

type Message struct {
    From      types.PeerID           `json:"from"`
    To        types.PeerID           `json:"to"`
    Topic     string                 `json:"topic"`
    Data      []byte                 `json:"data"`
    Timestamp time.Time              `json:"timestamp"`
    Headers   map[string]string      `json:"headers"`
}
```

## Ledger Service Interface

```go
package ledger

import (
    "context"
    "math/big"
    "github.com/blackhole/blackhole/pkg/types"
)

// LedgerService manages blockchain interactions
// Note: All transactions must be pre-signed - signing is handled by Identity Service
type LedgerService interface {
    services.Service
    
    // Transaction operations (accepts pre-signed transactions)
    SendSignedTransaction(ctx context.Context, signedTx SignedTransaction) (*TxReceipt, error)
    GetTransaction(ctx context.Context, txHash string) (*Transaction, error)
    GetTransactionReceipt(ctx context.Context, txHash string) (*TxReceipt, error)
    
    // Smart contract operations
    DeployContract(ctx context.Context, contract Contract) (*ContractAddress, error)
    CallContract(ctx context.Context, call ContractCall) ([]byte, error)
    EstimateGas(ctx context.Context, call ContractCall) (*big.Int, error)
    
    // Account operations
    GetBalance(ctx context.Context, address string) (*big.Int, error)
    GetNonce(ctx context.Context, address string) (uint64, error)
    
    // Block operations
    GetBlock(ctx context.Context, blockNumber *big.Int) (*Block, error)
    GetLatestBlock(ctx context.Context) (*Block, error)
    
    // Token operations
    TokenizeContent(ctx context.Context, params TokenizeParams) (*TokenInfo, error)
    TransferToken(ctx context.Context, transfer TokenTransfer) (*TxReceipt, error)
    GetTokenInfo(ctx context.Context, tokenID string) (*TokenInfo, error)
    
    // Events
    SubscribeLedgerEvents(ctx context.Context, filter EventFilter) (<-chan LedgerEvent, error)
}

// Types
type Transaction struct {
    From     string                 `json:"from"`
    To       string                 `json:"to"`
    Value    *big.Int               `json:"value"`
    Gas      uint64                 `json:"gas"`
    GasPrice *big.Int               `json:"gas_price"`
    Data     []byte                 `json:"data"`
    Nonce    uint64                 `json:"nonce"`
}

type SignedTransaction struct {
    Transaction Transaction            `json:"transaction"`
    Signature   []byte                 `json:"signature"`
    SignerDID   types.DID              `json:"signer_did"`
}

type TokenizeParams struct {
    ContentID   types.ContentID        `json:"content_id"`
    Supply      *big.Int               `json:"supply"`
    Metadata    TokenMetadata          `json:"metadata"`
    Royalties   []Royalty              `json:"royalties"`
}
```

## Social Service Interface

```go
package social

import (
    "context"
    "github.com/blackhole/blackhole/pkg/types"
)

// SocialService manages social networking features
type SocialService interface {
    services.Service
    
    // Profile operations
    CreateProfile(ctx context.Context, profile Profile) (*types.ProfileID, error)
    UpdateProfile(ctx context.Context, profileID types.ProfileID, updates ProfileUpdate) error
    GetProfile(ctx context.Context, profileID types.ProfileID) (*Profile, error)
    DeleteProfile(ctx context.Context, profileID types.ProfileID) error
    
    // Relationship management
    Follow(ctx context.Context, followerID, followeeID types.ProfileID) error
    Unfollow(ctx context.Context, followerID, followeeID types.ProfileID) error
    GetFollowers(ctx context.Context, profileID types.ProfileID) ([]types.ProfileID, error)
    GetFollowing(ctx context.Context, profileID types.ProfileID) ([]types.ProfileID, error)
    
    // Content operations
    CreatePost(ctx context.Context, post Post) (*types.PostID, error)
    GetPost(ctx context.Context, postID types.PostID) (*Post, error)
    DeletePost(ctx context.Context, postID types.PostID) error
    GetFeed(ctx context.Context, profileID types.ProfileID, filter FeedFilter) ([]Post, error)
    
    // Interaction operations
    Like(ctx context.Context, profileID types.ProfileID, contentID types.ContentID) error
    Unlike(ctx context.Context, profileID types.ProfileID, contentID types.ContentID) error
    Comment(ctx context.Context, comment Comment) (*types.CommentID, error)
    
    // ActivityPub federation
    SendActivity(ctx context.Context, activity Activity) error
    ReceiveActivity(ctx context.Context, activity Activity) error
    GetActor(ctx context.Context, actorID string) (*Actor, error)
    
    // Events
    SubscribeSocialEvents(ctx context.Context, filter EventFilter) (<-chan SocialEvent, error)
}

// Types
type Profile struct {
    ID          types.ProfileID        `json:"id"`
    DID         string                 `json:"did"`
    Username    string                 `json:"username"`
    DisplayName string                 `json:"display_name"`
    Bio         string                 `json:"bio"`
    Avatar      types.ContentID        `json:"avatar"`
    Banner      types.ContentID        `json:"banner"`
    Metadata    map[string]interface{} `json:"metadata"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

type Post struct {
    ID          types.PostID           `json:"id"`
    AuthorID    types.ProfileID        `json:"author_id"`
    Content     string                 `json:"content"`
    MediaIDs    []types.ContentID      `json:"media_ids"`
    Tags        []string               `json:"tags"`
    Visibility  string                 `json:"visibility"`
    ReplyTo     *types.PostID          `json:"reply_to,omitempty"`
    Metadata    map[string]interface{} `json:"metadata"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}
```

## Wallet Service Interface

```go
package wallet

import (
    "context"
    "github.com/blackhole/blackhole/pkg/types"
)

// WalletService manages decentralized and self-managed wallets
// Note: Wallets are accessed through DID authentication - DIDs control wallet access
type WalletService interface {
    services.Service
    
    // Wallet operations (require DID authentication)
    CreateWallet(ctx context.Context, params CreateWalletParams) (*types.WalletID, error)
    GetWallet(ctx context.Context, walletID types.WalletID) (*Wallet, error)
    UpdateWallet(ctx context.Context, walletID types.WalletID, updates WalletUpdate) error
    DeleteWallet(ctx context.Context, walletID types.WalletID) error
    
    // Access control (DIDs decrypt and access wallets)
    UnlockWallet(ctx context.Context, walletID types.WalletID, accessToken WalletAccessToken) error
    LockWallet(ctx context.Context, walletID types.WalletID) error
    
    // Transaction preparation (signing delegated to Identity Service)
    PrepareTransaction(ctx context.Context, walletID types.WalletID, params TransactionParams) (*Transaction, error)
    SubmitSignedTransaction(ctx context.Context, walletID types.WalletID, signedTx []byte) (*TxReceipt, error)
    
    // Credential management
    StoreCredential(ctx context.Context, walletID types.WalletID, credential VerifiableCredential) (*types.CredentialID, error)
    GetCredential(ctx context.Context, walletID types.WalletID, credentialID types.CredentialID) (*VerifiableCredential, error)
    ListCredentials(ctx context.Context, walletID types.WalletID, filter CredentialFilter) ([]VerifiableCredential, error)
    CreatePresentation(ctx context.Context, walletID types.WalletID, params PresentationParams) (*VerifiablePresentation, error)
    
    // DID association (wallets are controlled by DIDs)
    GetControllingDID(ctx context.Context, walletID types.WalletID) (*types.DID, error)
    ListAssociatedDIDs(ctx context.Context, walletID types.WalletID) ([]types.DID, error)
    
    // Synchronization (for decentralized wallets)
    SyncWallet(ctx context.Context, walletID types.WalletID) error
    GetSyncStatus(ctx context.Context, walletID types.WalletID) (*SyncStatus, error)
    EnableSync(ctx context.Context, walletID types.WalletID, enabled bool) error
    
    // Recovery
    ExportRecovery(ctx context.Context, walletID types.WalletID) (*RecoveryData, error)
    ImportRecovery(ctx context.Context, recovery RecoveryData) (*types.WalletID, error)
    GenerateRecoveryPhrase(ctx context.Context, walletID types.WalletID) ([]string, error)
    RecoverFromPhrase(ctx context.Context, phrase []string) (*types.WalletID, error)
    
    // Events
    SubscribeWalletEvents(ctx context.Context, filter EventFilter) (<-chan WalletEvent, error)
}

// Types
type CreateWalletParams struct {
    Type        WalletType             `json:"type"` // decentralized or self-managed
    DID         types.DID              `json:"did"`  // DID that will control this wallet
    Name        string                 `json:"name"`
    Metadata    map[string]interface{} `json:"metadata"`
}

type Wallet struct {
    ID             types.WalletID         `json:"id"`
    Type           WalletType             `json:"type"`
    Name           string                 `json:"name"`
    ControllingDID types.DID              `json:"controlling_did"` // Primary DID that controls this wallet
    AssociatedDIDs []types.DID            `json:"associated_dids"` // Additional DIDs with access
    CreatedAt      time.Time              `json:"created_at"`
    UpdatedAt      time.Time              `json:"updated_at"`
    SyncEnabled    bool                   `json:"sync_enabled"`
    Metadata       map[string]interface{} `json:"metadata"`
}

type WalletAccessToken struct {
    Token     string    `json:"token"`
    WalletID  types.WalletID `json:"wallet_id"`
    DID       types.DID `json:"did"`
    ExpiresAt time.Time `json:"expires_at"`
}

type TransactionParams struct {
    To       string                 `json:"to"`
    Value    string                 `json:"value"`
    Data     []byte                 `json:"data,omitempty"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type WalletType string

const (
    WalletTypeDecentralized WalletType = "decentralized"
    WalletTypeSelfManaged   WalletType = "self-managed"
)

type KeyInfo struct {
    ID         string                 `json:"id"`
    Type       KeyType                `json:"type"`
    PublicKey  []byte                 `json:"public_key"`
    Algorithm  string                 `json:"algorithm"`
    CreatedAt  time.Time              `json:"created_at"`
    Metadata   map[string]interface{} `json:"metadata"`
}

type VerifiableCredential struct {
    ID             types.CredentialID     `json:"id"`
    Issuer         string                 `json:"issuer"`
    Subject        string                 `json:"subject"`
    Type           []string               `json:"type"`
    IssuanceDate   time.Time              `json:"issuance_date"`
    ExpirationDate *time.Time             `json:"expiration_date,omitempty"`
    CredentialData map[string]interface{} `json:"credential_data"`
    Proof          interface{}            `json:"proof"`
}

type SyncStatus struct {
    Enabled      bool      `json:"enabled"`
    LastSync     time.Time `json:"last_sync"`
    InProgress   bool      `json:"in_progress"`
    TotalItems   int       `json:"total_items"`
    SyncedItems  int       `json:"synced_items"`
    PendingItems int       `json:"pending_items"`
    Errors       []string  `json:"errors,omitempty"`
}
```

## Analytics Service Interface

```go
package analytics

import (
    "context"
    "time"
    "github.com/blackhole/blackhole/pkg/types"
)

// AnalyticsService manages data analytics
type AnalyticsService interface {
    services.Service
    
    // Event tracking
    TrackEvent(ctx context.Context, event Event) error
    BatchTrackEvents(ctx context.Context, events []Event) error
    
    // Metrics
    RecordMetric(ctx context.Context, metric Metric) error
    GetMetrics(ctx context.Context, query MetricQuery) ([]MetricResult, error)
    
    // Reports
    GenerateReport(ctx context.Context, params ReportParams) (*Report, error)
    ScheduleReport(ctx context.Context, schedule ReportSchedule) (*ScheduleID, error)
    GetReport(ctx context.Context, reportID types.ReportID) (*Report, error)
    
    // User analytics
    GetUserActivity(ctx context.Context, userID types.ProfileID, period TimePeriod) (*UserActivity, error)
    GetContentPerformance(ctx context.Context, contentID types.ContentID) (*ContentPerformance, error)
    
    // Real-time analytics
    GetRealtimeStats(ctx context.Context) (*RealtimeStats, error)
    SubscribeRealtimeUpdates(ctx context.Context) (<-chan RealtimeUpdate, error)
}

// Types
type Event struct {
    Type       string                 `json:"type"`
    UserID     types.ProfileID        `json:"user_id"`
    Properties map[string]interface{} `json:"properties"`
    Timestamp  time.Time              `json:"timestamp"`
}

type Metric struct {
    Name       string                 `json:"name"`
    Value      float64                `json:"value"`
    Tags       map[string]string      `json:"tags"`
    Timestamp  time.Time              `json:"timestamp"`
}
```

## Service Mesh Integration

All services integrate with the internal service mesh:

```go
// ServiceMesh handles internal communication
type ServiceMesh interface {
    // Service calls
    Call(ctx context.Context, service, method string, req interface{}) (interface{}, error)
    AsyncCall(ctx context.Context, service, method string, req interface{}) <-chan Response
    
    // Events
    Publish(ctx context.Context, event Event) error
    Subscribe(ctx context.Context, eventType string) (<-chan Event, error)
    
    // Middleware
    Use(middleware Middleware)
    
    // Circuit breaker
    GetCircuitBreaker(service string) CircuitBreaker
}

// Example service mesh integration
func (s *StorageService) RegisterHandlers(mesh ServiceMesh) error {
    mesh.HandleFunc("storage.store", s.handleStore)
    mesh.HandleFunc("storage.retrieve", s.handleRetrieve)
    mesh.HandleFunc("storage.delete", s.handleDelete)
    return nil
}
```

## Event Bus Integration

Services communicate asynchronously via the event bus:

```go
// EventBus handles asynchronous messaging
type EventBus interface {
    // Publishing
    Publish(event Event) error
    PublishAsync(event Event) <-chan error
    
    // Subscription
    Subscribe(pattern string, handler EventHandler) (Subscription, error)
    SubscribeOnce(pattern string, handler EventHandler) (Subscription, error)
    
    // Management
    Unsubscribe(sub Subscription) error
    Stats() EventBusStats
}

// Example event bus integration
func (s *IdentityService) RegisterEvents(bus EventBus) error {
    // Publish events
    bus.Subscribe("storage.content.created", s.handleContentCreated)
    bus.Subscribe("ledger.token.minted", s.handleTokenMinted)
    
    // Subscribe to events
    s.events = bus
    return nil
}
```

## Error Handling

Standard error types across all services:

```go
// ServiceError represents a service-level error
type ServiceError struct {
    Service string      `json:"service"`
    Method  string      `json:"method"`
    Code    ErrorCode   `json:"code"`
    Message string      `json:"message"`
    Details interface{} `json:"details,omitempty"`
}

// Common error codes
type ErrorCode string

const (
    ErrNotFound         ErrorCode = "NOT_FOUND"
    ErrAlreadyExists    ErrorCode = "ALREADY_EXISTS"
    ErrInvalidInput     ErrorCode = "INVALID_INPUT"
    ErrUnauthorized     ErrorCode = "UNAUTHORIZED"
    ErrForbidden        ErrorCode = "FORBIDDEN"
    ErrTimeout          ErrorCode = "TIMEOUT"
    ErrInternalError    ErrorCode = "INTERNAL_ERROR"
    ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)
```

These interface contracts ensure consistent behavior across all services while maintaining clean boundaries and enabling proper service mesh integration.