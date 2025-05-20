# Blackhole Indexer Architecture

This document provides a comprehensive overview of the Blackhole indexer service architecture, which enables efficient querying and discovery of content, transactions, and blockchain data across the distributed platform.

## Overview

The Blackhole indexer service runs as a dedicated subprocess, providing a critical data accessibility layer that bridges the gap between raw blockchain data, distributed content storage, and user-facing applications. As an isolated subprocess, it transforms low-level blockchain events and distributed storage metadata into queryable, structured data that powers search, discovery, and analytics features while communicating with other services via gRPC.

## Core Design Principles

1. **Decentralized Indexing**: Maintain the platform's decentralized nature through distributed indexing nodes
2. **Real-time Processing**: Index blockchain events and content updates as they occur
3. **Multi-dimensional Queries**: Support complex queries across different data dimensions
4. **Scalable Architecture**: Handle growing data volumes and query loads
5. **Data Integrity**: Ensure consistency between indexed data and source systems
6. **Performance Optimized**: Provide sub-second query responses for common operations
7. **Protocol Agnostic**: Support multiple blockchain networks and storage systems
8. **Process Isolation**: Runs as independent subprocess with dedicated resources

## Subprocess Architecture

The Indexer Service runs as an isolated subprocess with dedicated resources for blockchain and content indexing:

```mermaid
graph TD
    subgraph Orchestrator
        Orch[Process Manager]
        SD[Service Discovery]
        Mon[Monitor]
    end
    
    subgraph Indexer Subprocess
        gRPC[gRPC Server :9007]
        EventListener[Event Listener]
        DataFetcher[Data Fetcher]
        Transform[Transform Engine]
        StoreMgr[Store Manager]
        QueryAPI[Query Engine]
    end
    
    subgraph External Sources
        RootNet[Root Network]
        IPFS[IPFS Network]
        Services[Platform Services]
    end
    
    Orch -->|spawn| Indexer Subprocess
    SD -->|register| gRPC
    Mon -->|health check| gRPC
    
    Indexer Subprocess -->|WebSocket| RootNet
    Indexer Subprocess -->|HTTP| IPFS
    Indexer Subprocess -->|gRPC| Services
```

### Service Entry Point

```go
// cmd/blackhole/service/indexer/main.go
package main

import (
    "context"
    "flag"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/blackhole/internal/services/indexer"
    "github.com/blackhole/pkg/api/indexer/v1"
    "google.golang.org/grpc"
)

var (
    port       = flag.Int("port", 9007, "gRPC port")
    unixSocket = flag.String("unix-socket", "/tmp/blackhole-indexer.sock", "Unix socket path")
    config     = flag.String("config", "", "Configuration file path")
)

func main() {
    flag.Parse()
    
    // Initialize service
    cfg, err := indexer.LoadConfig(*config)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    service, err := indexer.New(cfg)
    if err != nil {
        log.Fatalf("Failed to create service: %v", err)
    }
    
    // Initialize SubQuery node
    if err := service.InitializeSubQuery(context.Background()); err != nil {
        log.Fatalf("Failed to initialize SubQuery: %v", err)
    }
    
    // Connect to blockchain
    if err := service.ConnectToRootNetwork(context.Background()); err != nil {
        log.Fatalf("Failed to connect to Root Network: %v", err)
    }
    
    // Create gRPC server
    grpcServer := grpc.NewServer(
        grpc.MaxRecvMsgSize(50 * 1024 * 1024), // 50MB for large queries
        grpc.MaxSendMsgSize(50 * 1024 * 1024),
    )
    
    // Register service
    indexerv1.RegisterIndexerServiceServer(grpcServer, service)
    indexerv1.RegisterQueryServiceServer(grpcServer, service)
    
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
    
    // Start indexing pipeline
    go service.StartEventListener(context.Background())
    go service.StartDataFetcher(context.Background())
    go service.StartTransformEngine(context.Background())
    
    // Start GraphQL server for queries
    go service.StartGraphQLServer()
    
    // Handle shutdown gracefully
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    
    go func() {
        <-sigChan
        log.Println("Shutting down indexer service...")
        service.Shutdown()
        grpcServer.GracefulStop()
        cancel()
    }()
    
    // Start serving
    go func() {
        log.Printf("Indexer service listening on Unix socket: %s", *unixSocket)
        if err := grpcServer.Serve(unixListener); err != nil {
            log.Fatalf("Failed to serve Unix socket: %v", err)
        }
    }()
    
    log.Printf("Indexer service listening on TCP port: %d", *port)
    if err := grpcServer.Serve(tcpListener); err != nil {
        log.Fatalf("Failed to serve TCP: %v", err)
    }
}
```

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Data Sources                               │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │    Root     │  │    IPFS/    │  │   Social    │             │
│  │   Network   │  │   Filecoin  │  │  Activity   │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                 │                    │
└─────────┴────────────────┴─────────────────┴────────────────────┘
          │                │                 │
          ▼                ▼                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Indexing Pipeline                             │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Event     │  │    Data     │  │  Transform  │             │
│  │  Listener   │  │   Fetcher   │  │   Engine    │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                 │                    │
│         └────────────────┴─────────────────┘                    │
│                          │                                      │
│                    ┌─────▼─────┐                                │
│                    │  Mapping  │                                │
│                    │  Handler  │                                │
│                    └─────┬─────┘                                │
│                          │                                      │
│                    ┌─────▼─────┐                                │
│                    │   Store   │                                │
│                    │  Manager  │                                │
│                    └─────┬─────┘                                │
└──────────────────────────┴──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Storage Layer                                │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │    Index    │  │    Query    │  │   Cache     │             │
│  │  Database   │  │    Store    │  │   Layer     │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                 │                    │
└─────────┴────────────────┴─────────────────┴────────────────────┘
          │                │                 │
          ▼                ▼                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Query Layer                                │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   GraphQL   │  │    REST     │  │  WebSocket  │             │
│  │     API     │  │     API     │  │   Stream    │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

#### 1. Data Sources

The indexer ingests data from multiple sources:

- **Root Network**: Blockchain events, transactions, token operations
- **IPFS/Filecoin**: Content metadata, storage proofs, availability status
- **Social Activity**: Comments, likes, follows from ActivityPub integration
- **Identity System**: DID operations, credential verifications
- **Analytics Events**: Usage metrics, performance data

#### 2. Event Listener

Monitors blockchain and system events in real-time:

- Block production monitoring
- Transaction event filtering
- Smart contract event decoding
- System event subscription
- State change detection

#### 3. Data Fetcher

Retrieves additional data needed for indexing:

- IPFS content metadata resolution
- Blockchain state queries
- External API integrations
- Cross-chain data fetching
- Batch data retrieval

#### 4. Transform Engine

Processes raw data into structured formats:

- Event decoding and parsing
- Data normalization
- Field extraction and mapping
- Relationship establishment
- Computed field generation

#### 5. Mapping Handler

Applies business logic to transformed data:

- Entity creation and updates
- Relationship mapping
- Aggregation calculations
- Validation and sanitization
- Error handling

#### 6. Store Manager

Handles data persistence operations:

- Transaction management
- Batch write optimization
- Index maintenance
- Data versioning
- Conflict resolution

#### 7. Storage Layer

Provides persistent storage with multiple components:

- **Index Database**: Primary data store (PostgreSQL with extensions)
- **Query Store**: Optimized read replicas
- **Cache Layer**: Redis for hot data and query results
- **Archive Storage**: Long-term historical data

#### 8. Query Layer

Exposes indexed data through multiple interfaces:

- **GraphQL API**: Flexible queries for complex data relationships
- **REST API**: Simple HTTP endpoints for basic operations
- **WebSocket Stream**: Real-time data subscriptions
- **Bulk Export**: Large-scale data extraction

## SubQuery Integration

SubQuery has been selected as the primary indexing framework for Blackhole due to its native Substrate support and EVM compatibility.

### SubQuery Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    SubQuery Framework                           │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Project   │  │   Mapping   │  │    Query    │             │
│  │   Config    │  │  Functions  │  │   Service   │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                 │                    │
│         └────────────────┴─────────────────┘                    │
│                          │                                      │
│                    ┌─────▼─────┐                                │
│                    │  SubQuery │                                │
│                    │    Node    │                                │
│                    └─────┬─────┘                                │
│                          │                                      │
│  ┌─────────────┐  ┌─────▼─────┐  ┌─────────────┐               │
│  │    Store    │  │   Query    │  │   Cache     │               │
│  │  (Postgres) │  │   Engine   │  │   (Redis)   │               │
│  └─────────────┘  └───────────┘  └─────────────┘               │
└─────────────────────────────────────────────────────────────────┘
```

### SubQuery Components

1. **Project Configuration**: Defines data sources, schema, and deployment settings
2. **Mapping Functions**: Transform blockchain events into entities
3. **Query Service**: GraphQL API server with built-in caching
4. **SubQuery Node**: Core indexing engine that processes blocks
5. **Store**: PostgreSQL database for indexed data
6. **Cache**: Redis layer for query performance

### Why SubQuery?

1. **Native Substrate Support**: Optimized for Substrate-based chains like Root Network
2. **EVM Compatibility**: Full support for Ethereum-compatible smart contracts
3. **Performance**: Built-in optimizations for blockchain indexing
4. **Developer Experience**: Comprehensive tooling and documentation
5. **Decentralization Options**: Can run as centralized or decentralized service
6. **Multi-chain Support**: Index multiple chains in a single project

## Data Models

### Entity Schemas

#### Content Entity

```
Content
├── id: String (CID)
├── creator: Account
├── createdAt: DateTime
├── metadata: ContentMetadata
├── storage: StorageInfo
├── token: Token (optional)
├── licenses: [License]
├── analytics: ContentAnalytics
└── socialStats: SocialStats
```

#### Token Entity

```
Token
├── id: String
├── tokenId: String
├── type: TokenType
├── creator: Account
├── owner: Account
├── contentCid: String
├── metadata: TokenMetadata
├── royalties: RoyaltyConfig
├── transferHistory: [Transfer]
├── licenses: [License]
└── marketData: MarketData
```

#### License Entity

```
License
├── id: String
├── token: Token
├── licensor: Account
├── licensee: Account
├── type: LicenseType
├── terms: LicenseTerms
├── issuedAt: DateTime
├── expiresAt: DateTime
├── status: LicenseStatus
└── revocationReason: String
```

#### Account Entity

```
Account
├── id: String (Address/DID)
├── did: String
├── createdContent: [Content]
├── ownedTokens: [Token]
├── grantedLicenses: [License]
├── receivedLicenses: [License]
├── royaltyEarnings: [RoyaltyPayment]
├── socialProfile: SocialProfile
└── reputation: ReputationScore
```

#### Transaction Entity

```
Transaction
├── id: String (txHash)
├── blockNumber: BigInt
├── timestamp: DateTime
├── from: Account
├── to: Account
├── type: TransactionType
├── tokenId: String
├── value: BigInt
├── fee: BigInt
├── status: TransactionStatus
└── metadata: Object
```

## Indexing Pipeline

### Event Processing Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                   Event Processing Pipeline                     │
│                                                                 │
│  Block Event ──► Filter ──► Decode ──► Transform ──► Store      │
│                    │          │           │            │        │
│                    ▼          ▼           ▼            ▼        │
│                  Rules      Schema     Mapping      Database    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Processing Stages

1. **Event Detection**
   - Monitor new blocks
   - Filter relevant transactions
   - Identify smart contract events
   - Queue for processing

2. **Event Decoding**
   - Parse transaction data
   - Decode event parameters
   - Extract relevant fields
   - Validate data types

3. **Data Transformation**
   - Map to entity schemas
   - Calculate derived fields
   - Establish relationships
   - Apply business rules

4. **Storage Operations**
   - Create/update entities
   - Maintain indices
   - Update aggregations
   - Trigger side effects

5. **Post-Processing**
   - Clear caches
   - Update search indices
   - Emit notifications
   - Trigger workflows

## Query Architecture

### GraphQL Schema

```graphql
type Query {
  # Content queries
  content(id: ID!): Content
  contents(filter: ContentFilter, orderBy: ContentOrderBy, first: Int): [Content!]
  contentSearch(query: String!, filters: SearchFilters): SearchResults
  
  # Token queries
  token(id: ID!): Token
  tokens(filter: TokenFilter, orderBy: TokenOrderBy, first: Int): [Token!]
  tokensByCreator(creator: String!, first: Int): [Token!]
  
  # License queries
  license(id: ID!): License
  licenses(filter: LicenseFilter, first: Int): [License!]
  activeLicenses(tokenId: String!): [License!]
  
  # Account queries
  account(id: ID!): Account
  accounts(filter: AccountFilter, first: Int): [Account!]
  accountByDID(did: String!): Account
  
  # Market queries
  listings(filter: ListingFilter, orderBy: ListingOrderBy, first: Int): [Listing!]
  auctions(status: AuctionStatus, first: Int): [Auction!]
  
  # Analytics queries
  contentAnalytics(contentId: String!, timeframe: Timeframe): Analytics
  platformStats(timeframe: Timeframe): PlatformStats
  
  # Transaction queries
  transaction(id: ID!): Transaction
  transactions(filter: TransactionFilter, first: Int): [Transaction!]
  transactionHistory(accountId: String!, first: Int): [Transaction!]
}

type Subscription {
  # Real-time subscriptions
  contentCreated(creator: String): Content
  tokenTransferred(tokenId: String): Transfer
  licenseGranted(tokenId: String): License
  marketActivity(type: MarketEventType): MarketEvent
}
```

### Query Optimization

1. **Index Strategy**
   - Primary indices on entity IDs
   - Secondary indices on common filter fields
   - Composite indices for complex queries
   - Full-text indices for search

2. **Caching Strategy**
   - Query result caching
   - Entity-level caching
   - Computed field caching
   - Cache invalidation rules

3. **Query Planning**
   - Query complexity analysis
   - Cost-based optimization
   - Pagination strategies
   - N+1 query prevention

## Search Architecture

### Full-Text Search

```
┌─────────────────────────────────────────────────────────────────┐
│                    Search Architecture                          │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Content   │  │  Metadata   │  │    User     │             │
│  │   Indexer   │  │   Indexer   │  │   Query     │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                 │                    │
│         └────────────────┼─────────────────┘                    │
│                          ▼                                      │
│                    ┌───────────┐                                │
│                    │   Search  │                                │
│                    │   Engine  │                                │
│                    └─────┬─────┘                                │
│                          │                                      │
│  ┌─────────────┐  ┌─────▼─────┐  ┌─────────────┐               │
│  │   Ranking   │  │  Filter   │  │  Results    │               │
│  │   Engine    │  │  Engine   │  │  Formatter  │               │
│  └─────────────┘  └───────────┘  └─────────────┘               │
└─────────────────────────────────────────────────────────────────┘
```

### Search Features

1. **Multi-field Search**
   - Title and description
   - Content metadata
   - Creator information
   - Tag matching

2. **Advanced Filters**
   - Content type
   - Date ranges
   - Price ranges
   - License types

3. **Relevance Ranking**
   - Text relevance
   - Popularity metrics
   - Recency weighting
   - User preferences

4. **Search Suggestions**
   - Autocomplete
   - Related searches
   - Trending queries
   - Spelling corrections

## Performance Optimization

### Indexing Performance

1. **Batch Processing**
   - Group similar operations
   - Bulk database writes
   - Parallel processing
   - Resource pooling

2. **Incremental Updates**
   - Delta synchronization
   - Partial reindexing
   - Change detection
   - Minimal data transfer

3. **Resource Management**
   - Memory limits
   - CPU throttling
   - I/O rate limiting
   - Connection pooling

### Query Performance

1. **Query Optimization**
   - Query analysis and rewriting
   - Index hint generation
   - Execution plan caching
   - Result set limiting

2. **Caching Layers**
   - Application-level cache
   - Database query cache
   - CDN integration
   - Edge caching

3. **Load Balancing**
   - Read replica distribution
   - Query routing
   - Failover handling
   - Geographic distribution

## Data Consistency

### Consistency Mechanisms

1. **Event Ordering**
   - Block height tracking
   - Transaction ordering
   - Event sequence numbers
   - Timestamp validation

2. **State Reconciliation**
   - Periodic validation
   - Checksum verification
   - Reorg handling
   - State snapshots

3. **Error Recovery**
   - Failed event retry
   - Dead letter queues
   - Manual intervention
   - Audit trails

### Data Integrity

```
┌─────────────────────────────────────────────────────────────────┐
│                    Data Integrity Flow                          │
│                                                                 │
│  Source Data ──► Validation ──► Transform ──► Verify ──► Store  │
│                      │             │            │         │     │
│                      ▼             ▼            ▼         ▼     │
│                   Schema        Business      Cross     Audit   │
│                   Check         Rules         Check     Log     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Scalability Architecture

### Horizontal Scaling

```
┌─────────────────────────────────────────────────────────────────┐
│                   Scaled Architecture                           │
│                                                                 │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐                   │
│  │  Indexer  │  │  Indexer  │  │  Indexer  │                   │
│  │   Node 1  │  │   Node 2  │  │   Node 3  │                   │
│  └─────┬─────┘  └─────┬─────┘  └─────┬─────┘                   │
│        │              │              │                          │
│        └──────────────┴──────────────┘                          │
│                       │                                         │
│                 ┌─────▼─────┐                                   │
│                 │   Load    │                                   │
│                 │ Balancer  │                                   │
│                 └─────┬─────┘                                   │
│                       │                                         │
│  ┌───────────┐  ┌─────▼─────┐  ┌───────────┐                   │
│  │   Query   │  │  Primary  │  │   Query   │                   │
│  │  Replica  │  │    DB     │  │  Replica  │                   │
│  └───────────┘  └───────────┘  └───────────┘                   │
└─────────────────────────────────────────────────────────────────┘
```

### Scaling Strategies

1. **Indexer Scaling**
   - Multiple indexer instances
   - Work distribution
   - Leader election
   - State synchronization

2. **Database Scaling**
   - Read replicas
   - Sharding strategies
   - Connection pooling
   - Query routing

3. **Cache Scaling**
   - Distributed caching
   - Cache partitioning
   - Hot data replication
   - Cache warming

## Integration Points

### Service Integration

1. **Ledger Service**
   - Transaction monitoring
   - Token event indexing
   - Royalty tracking
   - License management

2. **Storage Service**
   - Content metadata indexing
   - Availability tracking
   - Replication status
   - Provider information

3. **Identity Service**
   - DID resolution
   - Credential indexing
   - Access control
   - Profile information

4. **Social Service**
   - Activity indexing
   - Relationship mapping
   - Engagement metrics
   - Content discovery

5. **Analytics Service**
   - Usage data collection
   - Performance metrics
   - Business intelligence
   - Trend analysis

### External Integration

1. **IPFS Gateway**
   - Content resolution
   - Metadata fetching
   - Availability checking
   - Pin status

2. **Root Network RPC**
   - Block subscription
   - State queries
   - Event filtering
   - Transaction monitoring

3. **Social Networks**
   - ActivityPub federation
   - Profile synchronization
   - Content distribution
   - Engagement tracking

## Security Considerations

### Access Control

1. **API Security**
   - Authentication requirements
   - Rate limiting
   - Query complexity limits
   - IP whitelisting

2. **Data Privacy**
   - Field-level permissions
   - User data encryption
   - GDPR compliance
   - Audit logging

3. **Query Security**
   - Input validation
   - SQL injection prevention
   - Query depth limiting
   - Resource constraints

### Operational Security

1. **Infrastructure Security**
   - Network isolation
   - Encrypted communications
   - Access logging
   - Intrusion detection

2. **Data Security**
   - Encryption at rest
   - Backup encryption
   - Secure key management
   - Access auditing

## Monitoring and Observability

### Metrics Collection

```
┌─────────────────────────────────────────────────────────────────┐
│                 Monitoring Architecture                         │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   System    │  │Application  │  │  Business   │             │
│  │   Metrics   │  │   Metrics   │  │   Metrics   │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                 │                    │
│         └────────────────┴─────────────────┘                    │
│                          │                                      │
│                    ┌─────▼─────┐                                │
│                    │  Metrics  │                                │
│                    │ Collector │                                │
│                    └─────┬─────┘                                │
│                          │                                      │
│  ┌─────────────┐  ┌─────▼─────┐  ┌─────────────┐               │
│  │   Alerts    │  │ Dashboard │  │  Analytics  │               │
│  └─────────────┘  └───────────┘  └─────────────┘               │
└─────────────────────────────────────────────────────────────────┘
```

### Key Metrics

1. **Indexing Metrics**
   - Blocks processed per second
   - Event processing latency
   - Indexing queue depth
   - Error rates

2. **Query Metrics**
   - Query response times
   - Query complexity scores
   - Cache hit rates
   - Error rates

3. **System Metrics**
   - CPU utilization
   - Memory usage
   - Disk I/O
   - Network traffic

4. **Business Metrics**
   - Total indexed content
   - Active users
   - Query volume
   - API usage

## Deployment Architecture

### Container Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  Kubernetes Deployment                          │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  Indexer    │  │   Query     │  │    API      │             │
│  │    Pods     │  │    Pods     │  │    Pods     │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  Database   │  │   Cache     │  │  Message    │             │
│  │  Service    │  │  Service    │  │   Queue     │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                 │
│  ┌─────────────────────────────────────────────────┐           │
│  │            Persistent Volumes                    │           │
│  └─────────────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────────────┘
```

### Deployment Components

1. **Indexer Deployment**
   - StatefulSet for data persistence
   - Horizontal pod autoscaling
   - Rolling updates
   - Health checks

2. **Query Service Deployment**
   - Deployment with replicas
   - Load balancer service
   - Ingress configuration
   - TLS termination

3. **Database Deployment**
   - StatefulSet configuration
   - Persistent volume claims
   - Backup CronJobs
   - Connection pooling

4. **Supporting Services**
   - Redis deployment
   - Message queue setup
   - Monitoring stack
   - Log aggregation

## Implementation Roadmap

### Phase 1: Foundation (4 weeks)
- SubQuery project setup
- Basic entity schemas
- Root Network integration
- Simple GraphQL API

### Phase 2: Core Features (6 weeks)
- Complete entity mappings
- IPFS metadata indexing
- Search functionality
- Query optimization

### Phase 3: Advanced Features (6 weeks)
- Real-time subscriptions
- Analytics integration
- Performance optimization
- Caching implementation

### Phase 4: Scale & Polish (4 weeks)
- Horizontal scaling
- Monitoring setup
- Security hardening
- Documentation

## Future Enhancements

1. **Multi-chain Support**
   - Cross-chain indexing
   - Unified queries
   - Chain abstraction
   - Bridge monitoring

2. **AI Integration**
   - Content recommendations
   - Search improvements
   - Anomaly detection
   - Predictive analytics

3. **Advanced Analytics**
   - Complex aggregations
   - Time-series analysis
   - Machine learning models
   - Business intelligence

4. **Developer Tools**
   - SDK generation
   - Query builders
   - Testing frameworks
   - Performance profiling

## Conclusion

The Blackhole indexer service provides a critical infrastructure component that enables efficient data access across the distributed platform. By leveraging SubQuery's proven indexing framework and implementing a comprehensive architecture for data processing, storage, and querying, we create a scalable and performant system that maintains the platform's decentralized principles while delivering excellent user experience.

## Process Resource Management

The Indexer Service subprocess has dedicated resources optimized for blockchain indexing and query operations:

### Resource Configuration

```go
// Indexer service resource limits
type IndexerServiceConfig struct {
    ProcessLimits ProcessResourceLimits {
        CPUQuota    "300%"         // 3 CPU cores (high demand)
        MemoryLimit "4GB"          // 4GB memory limit
        IOWeight    150            // Higher IO priority
        Nice        0              // Standard scheduling priority
    }
    
    // SubQuery configuration
    SubQueryConfig struct {
        NodeURL       string
        QueryURL      string
        MaxBlockRange int    // Maximum blocks per query
        BatchSize     int    // Events per batch
    }
    
    // Connection pools
    ConnectionPools struct {
        PostgresPool  int    // Database connections
        RedisPool     int    // Cache connections
        BlockchainRPC int    // RPC connections
    }
}

// Monitor indexing health and resources
func (i *IndexerService) MonitorHealth(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            stats := i.getProcessStats()
            if stats.MemoryMB > i.config.MemoryWarning {
                log.Warnf("Indexer memory usage high: %d MB", stats.MemoryMB)
            }
            
            // Monitor indexing lag
            lag := i.getIndexingLag()
            if lag > i.config.MaxAcceptableLag {
                log.Warnf("Indexing lag high: %d blocks behind", lag)
                i.increaseWorkers()
            }
            
            // Monitor query performance
            qps := i.getQueriesPerSecond()
            if qps > i.config.QPSThreshold {
                i.scaleQueryReplicas()
            }
            
        case <-ctx.Done():
            return
        }
    }
}
```

### Resource Isolation Benefits

1. **Process Isolation**: Indexing operations don't affect services
2. **Memory Management**: Large datasets handled independently
3. **CPU Control**: Complex queries don't impact system
4. **Database Isolation**: Dedicated PostgreSQL connections
5. **Crash Recovery**: Indexing failures don't affect platform

## gRPC Integration

The Indexer Service provides query capabilities to other services:

```go
type IndexerService struct {
    // Core components
    subQuery     *SubQueryNode
    eventListener *EventListener
    transformer  *TransformEngine
    queryEngine  *QueryEngine
    
    // Data stores
    postgres     *PostgresDB
    redis        *RedisCache
    
    // RPC clients
    storageClient storagev1.StorageServiceClient
}

// GraphQL query handler via gRPC
func (i *IndexerService) ExecuteQuery(ctx context.Context, req *indexerv1.QueryRequest) (*indexerv1.QueryResponse, error) {
    // Parse GraphQL query
    query, err := i.parseQuery(req.Query)
    if err != nil {
        return nil, err
    }
    
    // Check cache first
    if cached := i.redis.Get(query.Hash()); cached != nil {
        return cached.(*indexerv1.QueryResponse), nil
    }
    
    // Execute query
    result, err := i.queryEngine.Execute(ctx, query)
    if err != nil {
        return nil, err
    }
    
    // Cache result
    i.redis.Set(query.Hash(), result, 5*time.Minute)
    
    return &indexerv1.QueryResponse{
        Data:     result.Data,
        Metadata: result.Metadata,
    }, nil
}

// Real-time subscription handler
func (i *IndexerService) Subscribe(req *indexerv1.SubscriptionRequest, stream indexerv1.Indexer_SubscribeServer) error {
    subscription := i.createSubscription(req)
    defer subscription.Close()
    
    for {
        select {
        case event := <-subscription.Events:
            if err := stream.Send(event); err != nil {
                return err
            }
        case <-stream.Context().Done():
            return stream.Context().Err()
        }
    }
}
```

## Service Configuration

```yaml
indexer_service:
  # Service configuration
  service:
    name: "indexer"
    port: 9007
    unix_socket: "/tmp/blackhole-indexer.sock"
    log_level: "info"
    
  # Process management
  process:
    cpu_limit: "300%"          # 3 CPU cores
    memory_limit: "4GB"        # 4GB memory limit
    restart_policy: "always"
    restart_delay: "5s"
    health_check_interval: "30s"
    
  # SubQuery configuration
  subquery:
    project_name: "blackhole-indexer"
    node_endpoint: "http://localhost:3000"
    query_endpoint: "http://localhost:3001"
    dictionary_endpoint: "https://api.subquery.network/sq/root-network"
    
  # Database configuration
  database:
    type: "postgresql"
    host: "localhost"
    port: 5432
    name: "blackhole_indexer"
    max_connections: 50
    
  # Cache configuration
  cache:
    type: "redis"
    host: "localhost"
    port: 6379
    db: 0
    max_connections: 100
    
  # Blockchain configuration
  blockchain:
    network: "root-network"
    rpc_endpoint: "wss://root.network/ws"
    start_block: "latest"
    block_confirmations: 6
    
  # Query performance
  query:
    max_query_depth: 10
    max_query_complexity: 1000
    default_limit: 100
    max_limit: 1000
    timeout: "30s"
```

## Subprocess Benefits

1. **High Performance**: Dedicated resources for indexing workloads
2. **Scalability**: Can allocate more CPU/memory as needed
3. **Fault Tolerance**: Crashes don't affect other services
4. **Query Isolation**: Complex queries don't impact service performance
5. **Independent Updates**: Can update indexer without affecting platform
6. **Monitoring**: Process-specific metrics for optimization

---

This indexer architecture provides Blackhole with a scalable, high-performance data access layer that maintains platform stability through process isolation. Running as a dedicated subprocess with SubQuery integration ensures efficient blockchain indexing while the gRPC interface provides fast, reliable query capabilities to all platform services.