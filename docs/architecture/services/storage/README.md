# Storage Service Architecture

This directory contains the comprehensive architecture documentation for the Storage service, which handles content persistence, replication, and retrieval in the Blackhole platform.

## Documentation Structure

### Core Architecture
- [Storage Architecture](./storage_architecture.md) - Overall storage service design and implementation
- [Data Persistence](./data_persistence.md) - Data persistence strategies and mechanisms
- [Content Lifecycle Management](./content_lifecycle_management.md) - Content lifecycle from creation to deletion

### Content Processing
- [Content Addressing](./content_addressing.md) - Content pipeline architecture with high-parity Reed-Solomon encoding
- [Reed-Solomon Architecture](./reed_solomon_architecture.md) - Detailed Reed-Solomon encoding system for durability

### Persistence & Replication
- [Content Persistence & Replication](./content_persistence_replication.md) - Content storage and replication strategies
- [Storage Provider Selection](./storage_provider_selection.md) - Algorithm for optimal storage provider selection

### External Integration
- [Expanded Architecture](./expanded_architecture.md) - Extended storage architecture with advanced features
- [Filecoin Integration](./filecoin_integration.md) - Integration with Filecoin for decentralized storage

## Service Overview

The Storage service is responsible for:

1. **Content Storage**
   - Manages content persistence to IPFS
   - Handles Filecoin storage deals
   - Implements local caching strategies

2. **Reed-Solomon Encoding**
   - Provides high-parity encoding for durability
   - Manages fragment distribution
   - Handles streaming encoding/decoding

3. **Content Pipeline**
   - Processes content through validation, encryption, encoding
   - Manages chunking and optimal fragment size
   - Handles metadata extraction and indexing

4. **Replication Management**
   - Ensures content redundancy
   - Manages replication factors
   - Handles geographic distribution

5. **Content Retrieval**
   - Optimizes content fetching
   - Implements retrieval strategies
   - Manages reconstruction from fragments

## Technical Architecture

The Storage service runs as an independent OS process communicating via gRPC:

```go
// Storage service interface
type StorageService interface {
    // Content operations
    Store(context.Context, *Content) (*ContentID, error)
    Retrieve(context.Context, *ContentID) (*Content, error)
    Delete(context.Context, *ContentID) error
    
    // Reed-Solomon operations
    EncodeRS(context.Context, *Content) (*EncodedFragments, error)
    DecodeRS(context.Context, *EncodedFragments) (*Content, error)
    
    // Replication management
    SetReplicationFactor(context.Context, *ContentID, int) error
    GetStorageStatus(context.Context, *ContentID) (*StorageStatus, error)
}
```

## Integration Points

- **Node Service**: For P2P content distribution
- **Identity Service**: For content ownership and access control  
- **Ledger Service**: For storage economics and payments
- **Indexer Service**: For content discovery
- **Analytics Service**: For storage metrics and optimization