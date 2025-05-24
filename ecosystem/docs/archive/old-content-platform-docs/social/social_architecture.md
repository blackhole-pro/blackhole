# Social Services Architecture

## Overview

The Blackhole Social Service runs as a dedicated subprocess, providing decentralized social networking capabilities built on ActivityPub protocols. As an isolated subprocess, it enables peer-to-peer social interactions while maintaining user privacy and data sovereignty. The service communicates with other platform components via gRPC to create a comprehensive social experience around content creation and consumption.

## Core Principles

1. **Decentralization**: No central authority controls social data or relationships
2. **Interoperability**: Full ActivityPub compliance for federation with other platforms
3. **Privacy-First**: End-to-end encryption for private conversations, local-first data processing
4. **User Sovereignty**: Users own their social graph and activity history
5. **Content-Centric**: Social interactions revolve around content creation and discovery
6. **Scalability**: Distributed architecture supporting millions of users
7. **Moderation**: Community-driven moderation with decentralized governance
8. **Process Isolation**: Runs as independent subprocess with dedicated resources

## Subprocess Architecture

The Social Service runs as an isolated subprocess with dedicated resources for social networking operations:

```mermaid
graph TD
    subgraph Orchestrator
        Orch[Process Manager]
        SD[Service Discovery]
        Mon[Monitor]
    end
    
    subgraph Social Subprocess
        gRPC[gRPC Server :9005]
        AP[ActivityPub Engine]
        FedMgr[Federation Manager]
        GraphDB[Social Graph]
        FeedGen[Feed Generator]
        NotifSvc[Notification Service]
    end
    
    subgraph Identity Subprocess
        IDgRPC[gRPC Server :9001]
        DID[DID System]
    end
    
    subgraph Storage Subprocess
        StgRPC[gRPC Server :9003]
        Content[Content Store]
    end
    
    Orch -->|spawn| Social Subprocess
    SD -->|register| gRPC
    Mon -->|health check| gRPC
    
    Social Subprocess -->|gRPC :9001| Identity Subprocess
    Social Subprocess -->|gRPC :9003| Storage Subprocess
    Social Subprocess -->|HTTPS| External ActivityPub
```

### Service Entry Point

```go
// cmd/blackhole/service/social/main.go
package main

import (
    "context"
    "flag"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/blackhole/internal/services/social"
    "github.com/blackhole/pkg/api/social/v1"
    "google.golang.org/grpc"
)

var (
    port       = flag.Int("port", 9005, "gRPC port")
    unixSocket = flag.String("unix-socket", "/tmp/blackhole-social.sock", "Unix socket path")
    config     = flag.String("config", "", "Configuration file path")
)

func main() {
    flag.Parse()
    
    // Initialize service
    cfg, err := social.LoadConfig(*config)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    service, err := social.New(cfg)
    if err != nil {
        log.Fatalf("Failed to create service: %v", err)
    }
    
    // Initialize ActivityPub federation
    if err := service.InitializeFederation(context.Background()); err != nil {
        log.Fatalf("Failed to initialize federation: %v", err)
    }
    
    // Create gRPC server
    grpcServer := grpc.NewServer(
        grpc.MaxRecvMsgSize(50 * 1024 * 1024), // 50MB for rich media
        grpc.MaxSendMsgSize(50 * 1024 * 1024),
    )
    
    // Register service
    socialv1.RegisterSocialServiceServer(grpcServer, service)
    
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
    
    // Start ActivityPub HTTP server for federation
    go service.StartActivityPubServer()
    
    // Handle shutdown gracefully
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    
    go func() {
        <-sigChan
        log.Println("Shutting down social service...")
        service.StopActivityPubServer()
        grpcServer.GracefulStop()
        cancel()
    }()
    
    // Start serving
    go func() {
        log.Printf("Social service listening on Unix socket: %s", *unixSocket)
        if err := grpcServer.Serve(unixListener); err != nil {
            log.Fatalf("Failed to serve Unix socket: %v", err)
        }
    }()
    
    log.Printf("Social service listening on TCP port: %d", *port)
    if err := grpcServer.Serve(tcpListener); err != nil {
        log.Fatalf("Failed to serve TCP: %v", err)
    }
}
```

## Architecture Components

### 1. ActivityPub Protocol Layer

```yaml
ActivityPub Core:
  Inbox:
    - Receives incoming activities from federated servers
    - Validates activity signatures
    - Processes activities based on type
    - Updates local state
    
  Outbox:
    - Queues outgoing activities
    - Signs activities with actor keys
    - Delivers to recipient inboxes
    - Handles delivery failures
    
  Collections:
    - Following/Followers lists
    - Liked content
    - Shared content
    - Blocked actors
    
  Object Types:
    - Note (posts)
    - Article (long-form content)
    - Image/Video/Audio
    - Comment
    - Question (polls)
```

### 2. Federation Service

```yaml
Federation Manager:
  Discovery:
    - WebFinger for actor discovery
    - NodeInfo for server capabilities
    - Instance metadata
    
  Delivery:
    - HTTP signature authentication
    - Retry logic with exponential backoff
    - Parallel delivery to multiple recipients
    - Dead letter queue for failed deliveries
    
  Security:
    - Key management for actors
    - Signature verification
    - Origin validation
    - Rate limiting
```

### 3. Social Graph Service

```yaml
Graph Database:
  Nodes:
    - Actors (users, groups, bots)
    - Content objects
    - Activities
    
  Edges:
    - Follow relationships
    - Like relationships
    - Share relationships
    - Reply chains
    - Mentions
    - Blocks/Mutes
    
  Queries:
    - Friend-of-friend recommendations
    - Content discovery through social connections
    - Trending topics in network
    - Community detection
```

### 4. Feed Generation Service

```yaml
Feed Engine:
  Algorithms:
    Chronological:
      - Simple time-based ordering
      - No algorithmic filtering
      
    Home:
      - Activities from followed accounts
      - Boosted content from network
      - Weighted by interaction history
      
    Local:
      - Activities from same instance
      - Community-focused content
      
    Federated:
      - Activities from all known instances
      - Filtered by language and interests
      
    Trending:
      - Hashtag velocity tracking
      - Engagement metrics
      - Time decay factors
      
  Personalization:
    - User preference learning
    - Content affinity scoring
    - Interaction pattern analysis
    - Privacy-preserving ML models
```

### 5. Interaction Service

```yaml
Interaction Types:
  Social Actions:
    - Follow/Unfollow
    - Like/Unlike
    - Share/Boost
    - Reply/Comment
    - Quote
    - Bookmark
    - Report
    
  Content Creation:
    - Post creation with rich media
    - Thread creation
    - Poll creation
    - Event creation
    
  Messaging:
    - Direct messages (E2E encrypted)
    - Group conversations
    - Ephemeral messages
    
  Moderation:
    - Content reporting
    - User blocking
    - Instance blocking
    - Word filtering
```

### 6. Notification Service

```yaml
Notification System:
  Types:
    - Mentions
    - Replies
    - Follows
    - Likes
    - Shares
    - Direct messages
    - Moderation actions
    
  Delivery:
    - In-app notifications
    - Push notifications (mobile)
    - Email digests (optional)
    - WebSocket real-time updates
    
  Management:
    - Notification preferences
    - Mute controls
    - Priority levels
    - Batch processing
```

### 7. Privacy & Security Layer

```yaml
Privacy Controls:
  Visibility:
    - Public
    - Followers only
    - Mentioned users only
    - Direct (private)
    
  Data Protection:
    - End-to-end encryption for DMs
    - Deleted content purging
    - Data export capabilities
    - Account migration
    
  Access Control:
    - Block lists (user and instance level)
    - Mute lists
    - Follower approval
    - Content warnings
```

### 8. Analytics Service

```yaml
Analytics Engine:
  Metrics:
    - Engagement rates
    - Reach metrics
    - Growth statistics
    - Content performance
    
  Privacy:
    - Anonymized data collection
    - Opt-in analytics
    - Local processing
    - Aggregated reporting
    
  Insights:
    - Best posting times
    - Audience demographics
    - Content recommendations
    - Trend analysis
```

## Data Models

### Actor Model

```yaml
Actor:
  identity:
    - DID (Decentralized Identifier)
    - Username
    - Display name
    - Avatar
    - Banner
    
  profile:
    - Bio
    - Links
    - Location (optional)
    - Joined date
    
  keys:
    - Public key (for signatures)
    - Private key (encrypted)
    
  preferences:
    - Language
    - Privacy settings
    - Notification settings
    - Feed preferences
    
  statistics:
    - Follower count
    - Following count
    - Post count
    - Engagement metrics
```

### Activity Model

```yaml
Activity:
  metadata:
    - ID (globally unique)
    - Type (Create, Update, Delete, Follow, etc.)
    - Actor (who performed the action)
    - Published timestamp
    - Updated timestamp
    
  targeting:
    - To (primary recipients)
    - CC (secondary recipients)
    - BCC (blind carbon copy)
    
  object:
    - Content or reference
    - Media attachments
    - Tags and mentions
    
  context:
    - Reply thread
    - Conversation ID
    - Hashtags
```

### Content Model

```yaml
Content:
  core:
    - ID
    - Type (Note, Article, Image, etc.)
    - Author (Actor reference)
    - Created timestamp
    - Modified timestamp
    
  body:
    - Text content
    - Media attachments
    - Embed data
    - Content warnings
    
  metadata:
    - Language
    - Visibility
    - Sensitive flag
    - License
    
  interactions:
    - Like count
    - Share count
    - Reply count
    - View count
```

## Integration Points

### Storage Service

- Profile and banner image storage via IPFS
- Media attachment storage and retrieval
- Content persistence for federated objects
- Backup and archive functionality

### Identity Service

- DID-based actor identification
- Verifiable credentials for verification badges
- Multi-factor authentication
- Account recovery mechanisms

### Indexer Service

- Full-text search of social content
- Hashtag indexing
- Mention tracking
- Trending topic calculation

### Analytics Service

- Social interaction metrics
- Content performance tracking
- Network growth monitoring
- Federation health metrics

### Ledger Service

- Tokenized social interactions
- Reward mechanisms for quality content
- Micropayments for premium features
- NFT integration for digital collectibles

## Performance Considerations

### Caching Strategy

```yaml
Cache Layers:
  Edge Cache:
    - Static assets (avatars, banners)
    - Public timeline data
    - Popular content
    
  Application Cache:
    - User sessions
    - Recent activities
    - Relationship data
    
  Database Cache:
    - Query results
    - Computed feeds
    - Aggregated statistics
```

### Scalability Measures

```yaml
Scaling Strategies:
  Horizontal Scaling:
    - Activity processing workers
    - Federation delivery workers
    - Feed generation workers
    
  Data Partitioning:
    - Shard by actor ID
    - Time-based partitioning for activities
    - Geographic distribution
    
  Async Processing:
    - Background job queues
    - Event-driven architecture
    - Message broker for federation
```

## Security Considerations

### Threat Mitigation

```yaml
Security Measures:
  Authentication:
    - OAuth 2.0 for API access
    - HTTP signatures for federation
    - DID-based authentication
    
  Authorization:
    - Role-based access control
    - Capability-based permissions
    - Federation allow/block lists
    
  Protection:
    - Rate limiting
    - DDoS protection
    - Spam detection
    - Content validation
```

### Privacy Safeguards

```yaml
Privacy Features:
  Data Minimization:
    - Collect only necessary data
    - Automatic data expiration
    - User-controlled retention
    
  Encryption:
    - TLS for all communications
    - E2E encryption for DMs
    - Encrypted storage for sensitive data
    
  Anonymization:
    - Remove PII from analytics
    - Aggregate reporting only
    - No tracking across platforms
```

## Moderation Framework

### Community Moderation

```yaml
Moderation Tools:
  User Level:
    - Block/mute capabilities
    - Report content/users
    - Content warnings
    - Word filters
    
  Instance Level:
    - Admin controls
    - Moderation queues
    - Auto-moderation rules
    - Federation policies
    
  Network Level:
    - Shared block lists
    - Reputation systems
    - Community governance
    - Appeal processes
```

### AI-Assisted Moderation

```yaml
AI Moderation:
  Content Analysis:
    - Hate speech detection
    - NSFW content detection
    - Spam identification
    - Language classification
    
  Behavioral Analysis:
    - Bot detection
    - Coordinated campaigns
    - Abuse pattern recognition
    - Velocity tracking
    
  Human Override:
    - Always allow appeals
    - Transparent decisions
    - Community review
    - Bias monitoring
```

## Resource Management

The Social Service subprocess has dedicated resources tuned for social networking operations:

### Process Resource Configuration

```go
// Social service resource limits
type SocialServiceConfig struct {
    ProcessLimits ProcessResourceLimits {
        CPUQuota    "200%"         // 2 CPU cores max
        MemoryLimit "2GB"          // 2GB memory limit
        IOWeight    100            // Standard IO priority
        Nice        0              // Standard scheduling priority
    }
    
    // gRPC clients for other services
    IdentityClient *grpc.ClientConn
    StorageClient  *grpc.ClientConn
    IndexerClient  *grpc.ClientConn
}

// Monitor federation health and resources
func (s *SocialService) MonitorResourceHealth(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            stats := s.getProcessStats()
            if stats.MemoryMB > s.config.MemoryWarning {
                log.Warnf("Social service memory usage high: %d MB", stats.MemoryMB)
            }
            
            // Monitor federation connections
            federatedPeers := s.federation.GetActivePeers()
            s.metrics.RecordFederationHealth(len(federatedPeers))
            
        case <-ctx.Done():
            return
        }
    }
}
```

### Resource Isolation Benefits

1. **Process Isolation**: ActivityPub operations don't affect other services
2. **Memory Management**: Large media content handled independently
3. **CPU Control**: Feed generation doesn't impact system performance
4. **Network Isolation**: Federation traffic isolated from internal RPC
5. **Crash Recovery**: Social service failures don't bring down platform

## gRPC Integration

The Social Service communicates with other services via gRPC:

```go
type SocialService struct {
    // gRPC clients
    identityClient identityv1.IdentityServiceClient
    storageClient  storagev1.StorageServiceClient
    indexerClient  indexerv1.IndexerServiceClient
    
    // Core components
    activityPub    *ActivityPubEngine
    federation     *FederationManager
    socialGraph    *GraphDatabase
    feedGenerator  *FeedGenerator
}

// Authenticate user via Identity service
func (s *SocialService) AuthenticateUser(ctx context.Context, req *AuthRequest) error {
    resp, err := s.identityClient.AuthenticateDID(ctx, &identityv1.DIDAuthRequest{
        Did:       req.UserDID,
        Challenge: req.Challenge,
        Signature: req.Signature,
    })
    
    if err != nil {
        return fmt.Errorf("identity service auth failed: %w", err)
    }
    
    // Verify permissions for social actions
    if !resp.HasPermission("social:post") {
        return ErrInsufficientPermissions
    }
    
    return nil
}

// Store media content via Storage service
func (s *SocialService) StoreMedia(ctx context.Context, media *MediaContent) (*ContentRef, error) {
    resp, err := s.storageClient.Upload(ctx, &storagev1.UploadRequest{
        Content:     media.Data,
        ContentType: media.Type,
        Metadata:    media.Metadata,
    })
    
    if err != nil {
        return nil, fmt.Errorf("storage service upload failed: %w", err)
    }
    
    return &ContentRef{
        CID:      resp.Cid,
        URL:      resp.Url,
        MimeType: media.Type,
    }, nil
}
```

## Service Configuration

```yaml
social_service:
  # Service configuration
  service:
    name: "social"
    port: 9005
    unix_socket: "/tmp/blackhole-social.sock"
    log_level: "info"
    
  # Process management
  process:
    cpu_limit: "200%"          # 2 CPU cores
    memory_limit: "2GB"        # 2GB memory limit
    restart_policy: "always"
    restart_delay: "5s"
    health_check_interval: "30s"
    
  # ActivityPub configuration
  activitypub:
    hostname: "blackhole.social"
    federation_enabled: true
    max_federation_connections: 1000
    inbox_size: 10000
    outbox_size: 10000
    
  # Federation settings
  federation:
    relay_enabled: true
    allowed_instances: []  # Empty = all allowed
    blocked_instances: []
    auto_accept_follows: false
    
  # Social graph database
  graph:
    type: "neo4j"
    connection_string: "bolt://localhost:7687"
    max_connections: 50
    
  # Feed generation
  feed:
    cache_size: 10000
    max_timeline_length: 500
    refresh_interval: "5m"
    
  # Notification settings
  notifications:
    queue_size: 10000
    batch_size: 100
    delivery_interval: "10s"
```

## Future Enhancements

### Planned Features

1. **Advanced Federation**
   - Cross-protocol bridging (Matrix, XMPP)
   - Improved relay systems
   - Federation monitoring dashboard

2. **Enhanced Privacy**
   - Zero-knowledge proofs for interactions
   - Homomorphic encryption for private feeds
   - Decentralized identity verification

3. **Rich Media**
   - Live streaming integration
   - 360° content support
   - AR/VR social experiences

4. **Monetization**
   - Creator subscriptions
   - Tipping mechanisms
   - Paid promotions (privacy-preserving)

5. **AI Integration**
   - Personalized content recommendations
   - Natural language processing
   - Automated content tagging

## Subprocess Benefits

1. **Isolation**: Social operations don't impact other services
2. **Resource Control**: Dedicated CPU/memory for federation
3. **Fault Tolerance**: Crashes don't affect storage or identity
4. **Independent Scaling**: Can run multiple social instances
5. **Security**: Process-level security boundaries
6. **Monitoring**: Process-specific metrics and health checks

## Conclusion

The Blackhole Social Service provides a comprehensive, decentralized social networking solution that prioritizes user privacy and data sovereignty while maintaining compatibility with the broader federated social web. Running as an isolated subprocess with dedicated resources ensures reliable federation while the gRPC architecture enables seamless integration with identity, storage, and indexing services.