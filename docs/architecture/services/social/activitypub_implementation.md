# ActivityPub Implementation Guide

## Overview

This document details the implementation of ActivityPub protocol within the Blackhole social services. ActivityPub is a W3C standard for decentralized social networking that enables federation between different platforms while maintaining user autonomy and data portability.

## Protocol Fundamentals

### Actor Model

```yaml
Actor Structure:
  Required Properties:
    - id: Unique URI identifying the actor
    - type: Person, Group, Application, Service, or Organization
    - inbox: URI for receiving activities
    - outbox: URI for sending activities
    - preferredUsername: Unique username within the instance
    
  Optional Properties:
    - following: Collection of actors this actor follows
    - followers: Collection of actors following this actor
    - liked: Collection of objects this actor has liked
    - streams: Additional activity streams
    - endpoints: Service endpoints (sharedInbox, proxyUrl, etc.)
    - publicKey: Public key for HTTP signatures
```

### Activity Types

```yaml
Core Activities:
  Object Management:
    Create:
      - Creates a new object (post, image, etc.)
      - Object included in activity
      
    Update:
      - Modifies existing object
      - Includes updated object
      
    Delete:
      - Removes object
      - May include tombstone
      
  Social Interactions:
    Follow:
      - Request to follow another actor
      - Target actor can Accept or Reject
      
    Accept:
      - Accepts a follow request
      - Establishes follower relationship
      
    Reject:
      - Rejects a follow request
      - No relationship established
      
    Like:
      - Expresses interest in an object
      - Adds to liked collection
      
    Announce:
      - Shares/boosts an object
      - Redistributes to followers
      
    Undo:
      - Reverses previous activity
      - References original activity
```

### Object Types

```yaml
Content Objects:
  Note:
    - Short-form text content
    - Similar to tweets or toots
    - May include attachments
    
  Article:
    - Long-form content
    - Includes title and body
    - Rich text formatting
    
  Image:
    - Visual content
    - Includes URL and metadata
    - Alt text for accessibility
    
  Video:
    - Video content
    - Streaming URL
    - Duration and format info
    
  Audio:
    - Audio content
    - Podcast episodes, music
    - Duration and bitrate
    
  Event:
    - Calendar events
    - Start/end times
    - Location information
    
  Question:
    - Polls and surveys
    - Multiple choice options
    - End time for voting
```

## Implementation Architecture

### 1. Protocol Handler

```yaml
HTTP Endpoints:
  Actor Endpoints:
    GET /.well-known/webfinger:
      - Actor discovery via email-like addresses
      - Returns actor URI and profile info
      
    GET /users/{username}:
      - Returns actor object
      - Content negotiation for ActivityPub
      
    GET /users/{username}/inbox:
      - Actor's inbox (usually requires auth)
      - Returns OrderedCollection
      
    POST /users/{username}/inbox:
      - Receive activities from other servers
      - Validate signatures
      - Process based on activity type
      
    GET /users/{username}/outbox:
      - Actor's public activities
      - Paginated OrderedCollection
      
    POST /users/{username}/outbox:
      - Client-to-server activity creation
      - Validate authorization
      - Distribute to followers
```

### 2. Signature Verification

```yaml
HTTP Signatures:
  Request Signing:
    headers:
      - (request-target): POST /inbox
      - host: example.com
      - date: Tue, 07 Jun 2024 20:51:35 GMT
      - digest: SHA-256=...
      - signature: keyId="...", algorithm="rsa-sha256", headers="...", signature="..."
      
  Verification Steps:
    1. Extract signature parameters
    2. Fetch actor's public key
    3. Verify signature matches
    4. Check date within acceptable range
    5. Verify digest if present
```

### 3. Activity Processing

```yaml
Inbox Processing:
  Create Activity:
    1. Verify actor signature
    2. Validate object properties
    3. Store object in database
    4. Add to relevant timelines
    5. Generate notifications
    6. Return 201 Created
    
  Follow Activity:
    1. Verify actor signature
    2. Check if already following
    3. Create follow relationship
    4. Send Accept activity back
    5. Add to followers collection
    6. Return 201 Created
    
  Like Activity:
    1. Verify actor signature
    2. Find target object
    3. Add to likes collection
    4. Update like count
    5. Notify object owner
    6. Return 201 Created
    
  Announce Activity:
    1. Verify actor signature
    2. Find announced object
    3. Create share record
    4. Add to timeline
    5. Notify original author
    6. Return 201 Created
```

### 4. Federation Queue

```yaml
Delivery System:
  Queue Structure:
    - Priority queues by instance
    - Retry logic with backoff
    - Dead letter queue
    - Delivery reports
    
  Worker Configuration:
    - Concurrent delivery workers
    - Rate limiting per instance
    - Timeout handling
    - Error categorization
    
  Optimization:
    - Shared inbox delivery
    - Batch processing
    - Connection pooling
    - Circuit breakers
```

### 5. Collections Management

```yaml
Collection Types:
  OrderedCollection:
    - Time-ordered items
    - Used for outbox, inbox
    - Supports pagination
    
  Collection Pages:
    - first: First page URI
    - last: Last page URI
    - next: Next page URI
    - prev: Previous page URI
    - partOf: Parent collection
    - items: Array of items
    
  Special Collections:
    following:
      - Actors this actor follows
      - May be private
      
    followers:
      - Actors following this actor
      - Usually public
      
    liked:
      - Objects actor has liked
      - Configurable privacy
      
    featured:
      - Pinned/featured content
      - Highlighted items
```

## Advanced Features

### 1. Shared Inbox

```yaml
Shared Inbox Optimization:
  Configuration:
    - Single endpoint per instance
    - Reduces delivery overhead
    - Batched activity processing
    
  Implementation:
    endpoint: /inbox
    processing:
      1. Receive activity
      2. Verify signature
      3. Determine recipients
      4. Distribute to actor inboxes
      5. Process once per instance
```

### 2. Relay Support

```yaml
Relay Configuration:
  Purpose:
    - Improve federation reach
    - Reduce direct connections
    - Content distribution
    
  Implementation:
    relay_actor:
      - Special actor type
      - Follows all local actors
      - Announces public content
      
    subscription:
      - Follow relay actors
      - Receive announcements
      - Filter relevant content
```

### 3. Custom Extensions

```yaml
Blackhole Extensions:
  Additional Properties:
    blackhole:tokenizedContent:
      - Links to tokenized versions
      - Smart contract addresses
      
    blackhole:contentLicense:
      - Specific license terms
      - Rights management info
      
    blackhole:storageProof:
      - IPFS content hash
      - Filecoin deal proof
      
  Custom Activities:
    TokenizeContent:
      - Create NFT from content
      - Include contract details
      
    PurchaseContent:
      - Buy access to content
      - Payment confirmation
```

### 4. Performance Optimizations

```yaml
Caching Strategy:
  Actor Cache:
    - Cache remote actor objects
    - TTL based on update frequency
    - Invalidate on Update activity
    
  Public Key Cache:
    - Cache actor public keys
    - Longer TTL (keys change rarely)
    - Essential for signature verification
    
  Media Cache:
    - Cache remote media files
    - Progressive loading
    - CDN integration
    
  Timeline Cache:
    - Pre-computed timelines
    - Redis-backed storage
    - Incremental updates
```

## Security Considerations

### 1. Signature Security

```yaml
Security Measures:
  Key Management:
    - Regular key rotation
    - Secure key storage
    - Key revocation support
    
  Signature Validation:
    - Time window enforcement
    - Replay attack prevention
    - Origin verification
    
  Rate Limiting:
    - Per-actor limits
    - Per-instance limits
    - Adaptive throttling
```

### 2. Content Security

```yaml
Content Validation:
  Input Sanitization:
    - HTML filtering
    - Script removal
    - URL validation
    
  Media Security:
    - File type validation
    - Size limits
    - Virus scanning
    
  Federation Policies:
    - Instance block lists
    - Content filtering
    - Spam detection
```

### 3. Privacy Controls

```yaml
Privacy Features:
  Visibility Scopes:
    - Public: Visible to all
    - Unlisted: Not in public timelines
    - Followers: Only followers can see
    - Private: Only mentioned users
    
  Data Protection:
    - Selective federation
    - Content expiration
    - Right to deletion
    
  User Controls:
    - Block/mute lists
    - Follower approval
    - Instance selection
```

## Interoperability

### 1. Mastodon Compatibility

```yaml
Mastodon Extensions:
  Additional Properties:
    - sensitive: Content warning flag
    - language: Content language
    - repliesCount: Number of replies
    - sharesCount: Number of shares
    
  Custom Fields:
    - Profile metadata
    - Verification links
    - Bot indicators
```

### 2. Pleroma/Akkoma Support

```yaml
Pleroma Features:
  Extensions:
    - Emoji reactions
    - Quote posts
    - Local-only posts
    
  API Compatibility:
    - Support both APIs
    - Feature detection
    - Graceful degradation
```

### 3. Other Platforms

```yaml
Platform Support:
  PeerTube:
    - Video federation
    - Channel subscriptions
    - Comments and likes
    
  Pixelfed:
    - Image galleries
    - Stories support
    - Collections
    
  Funkwhale:
    - Audio federation
    - Playlist sharing
    - Library access
```

## Testing & Validation

### 1. Protocol Testing

```yaml
Test Suites:
  Unit Tests:
    - Activity parsing
    - Signature generation
    - Collection management
    
  Integration Tests:
    - Federation flows
    - Activity delivery
    - Error handling
    
  Compliance Tests:
    - W3C test suite
    - Compatibility tests
    - Security audits
```

### 2. Federation Testing

```yaml
Testing Tools:
  Local Federation:
    - Multi-instance setup
    - Docker compose
    - Network simulation
    
  Test Instances:
    - Staging servers
    - Cross-platform testing
    - Load testing
    
  Monitoring:
    - Delivery success rates
    - Federation health
    - Error tracking
```

## Deployment Considerations

### 1. Infrastructure

```yaml
Deployment Architecture:
  Components:
    - Web servers (ActivityPub endpoints)
    - Worker processes (federation)
    - Queue system (Redis/RabbitMQ)
    - Database (PostgreSQL)
    - Cache layer (Redis)
    - Media storage (S3/IPFS)
    
  Scaling:
    - Horizontal scaling
    - Load balancing
    - Geographic distribution
```

### 2. Configuration

```yaml
Instance Configuration:
  Required Settings:
    - Instance domain
    - Admin contact
    - Instance description
    - Registration policy
    
  Federation Settings:
    - Relay configuration
    - Block lists
    - Media proxy
    - Shared inbox
    
  Performance Tuning:
    - Worker count
    - Queue priorities
    - Cache sizes
    - Rate limits
```

## Monitoring & Maintenance

### 1. Metrics

```yaml
Key Metrics:
  Federation Health:
    - Delivery success rate
    - Average delivery time
    - Failed deliveries
    - Retry queue size
    
  Performance:
    - Request latency
    - Worker utilization
    - Database performance
    - Cache hit rates
    
  User Activity:
    - Active users
    - Post frequency
    - Federation reach
    - Interaction rates
```

### 2. Maintenance Tasks

```yaml
Regular Maintenance:
  Data Cleanup:
    - Remove orphaned activities
    - Purge old notifications
    - Clean media cache
    
  Federation Hygiene:
    - Update instance lists
    - Refresh actor cache
    - Verify key validity
    
  Performance:
    - Analyze slow queries
    - Optimize indexes
    - Update statistics
```

## Best Practices

### 1. Implementation Guidelines

1. **Always verify signatures** before processing activities
2. **Implement proper error handling** with meaningful responses
3. **Use caching strategically** to reduce federation overhead
4. **Respect rate limits** when delivering to other instances
5. **Provide clear documentation** for API endpoints
6. **Support content negotiation** for ActivityPub clients
7. **Implement graceful degradation** for unsupported features

### 2. Federation Etiquette

1. **Deliver activities promptly** to maintain real-time feel
2. **Retry failed deliveries** with exponential backoff
3. **Respect robots.txt** and federation policies
4. **Handle deletions properly** by removing cached content
5. **Provide contact information** for instance administrators
6. **Implement abuse prevention** to be a good federation citizen

## Conclusion

This ActivityPub implementation provides Blackhole with a robust, standards-compliant social networking layer that enables seamless federation with the broader decentralized social web. By following these guidelines and best practices, we ensure reliable, secure, and performant social interactions while maintaining user privacy and autonomy.