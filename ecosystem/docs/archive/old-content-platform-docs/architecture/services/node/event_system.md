# Event System Architecture

## Overview

The Event System provides event-driven communication patterns for the Blackhole platform's subprocess architecture. It handles both intra-process events (within a service) and inter-process communication (between services via RPC). Each service subprocess has its own local event bus, while cross-service events are transmitted through gRPC messaging.

## Architecture Overview

### Subprocess Event Model

The event system operates at two levels:
1. **Local Events**: Within a single service process
2. **RPC Events**: Between service processes via gRPC

### Core Components

#### Local Event Bus (Per Service)
- **In-Process Hub**: Routes events within a service
- **Topic Management**: Organizes local events by topic
- **Subscription Registry**: Tracks event listeners within the process
- **Event Queue**: Buffers pending local events
- **Delivery Manager**: Ensures local event delivery

#### RPC Event Bridge
- **gRPC Service**: Handles cross-process event transmission
- **Event Serialization**: Protobuf message conversion
- **Service Discovery**: Finds target service processes
- **Delivery Guarantees**: Ensures cross-process delivery
- **Error Handling**: Manages RPC failures

#### Event Store
- **Per-Service Journal**: Local event log for each service
- **Shared Storage**: Optional centralized event persistence
- **Event Replay**: Historical event access
- **Compaction**: Storage optimization
- **Archive**: Long-term storage

## Event Model

### Event Structure

#### Base Event
```go
type Event struct {
    ID           string            `json:"id"`              // Unique event identifier
    Type         string            `json:"type"`            // Event type/category
    Source       string            `json:"source"`          // Event origin (service process name)
    ProcessID    int               `json:"process_id"`      // OS process ID
    ServiceName  string            `json:"service_name"`    // Service subprocess name
    Timestamp    int64             `json:"timestamp"`       // Event time (unix nano)
    Version      string            `json:"version"`         // Event schema version
    Data         interface{}       `json:"data"`            // Event payload
    Metadata     *Metadata         `json:"metadata"`        // Additional context
    IsLocal      bool              `json:"is_local"`        // Local vs RPC event
    TargetService string           `json:"target_service"`  // For RPC events
}

type Metadata struct {
    UserID       string            `json:"user_id,omitempty"`      // User context
    SessionID    string            `json:"session_id,omitempty"`   // Session context
    TraceID      string            `json:"trace_id,omitempty"`     // Distributed trace
    Tags         []string          `json:"tags,omitempty"`         // Event tags
    Priority     int               `json:"priority,omitempty"`     // Event priority
    TTL          int64             `json:"ttl,omitempty"`          // Time to live
}

// RPC Event Message
type RPCEvent struct {
    Event        *Event            `protobuf:"bytes,1,opt,name=event,proto3"`
    DeliveryMode string            `protobuf:"bytes,2,opt,name=delivery_mode,proto3"`
    RetryPolicy  *RetryPolicy      `protobuf:"bytes,3,opt,name=retry_policy,proto3"`
}
```

### Event Types

#### System Events (Subprocess Level)
- **Process Lifecycle**: Service subprocess start/stop
- **Service Status**: Service health changes
- **IPC Events**: Unix socket connection events
- **Resource Events**: Process resource limits
- **Health Events**: Service health status

#### Cross-Process Events (RPC)
- **Service Discovery**: Service registration/deregistration
- **State Sync**: State synchronization between services
- **Request Events**: Incoming RPC requests
- **Response Events**: RPC responses
- **Error Events**: Cross-service failures

#### Application Events
- **User Events**: User actions (handled by specific services)
- **Content Events**: Content operations (Storage service)
- **Transaction Events**: Blockchain operations (Ledger service)
- **Social Events**: Social interactions (Social service)
- **Analytics Events**: Metric updates (Analytics service)

#### Process Management Events
- **Spawn Events**: New subprocess creation
- **Restart Events**: Service restart notifications
- **Resource Events**: CPU/Memory limit changes
- **Crash Events**: Process crash notifications
- **Update Events**: Service hot-reload events

## Event Flow

### Publishing Events

#### Local Event Creation (Within Service)
```go
// Create local event within service process
func (s *IdentityService) PublishUserCreated(user *User) {
    event := &Event{
        ID:          generateID(),
        Type:        "user.created",
        Source:      "identity",
        ProcessID:   os.Getpid(),
        ServiceName: "identity",
        Timestamp:   time.Now().UnixNano(),
        Version:     "1.0",
        IsLocal:     true,
        Data: map[string]interface{}{
            "userId": user.ID,
            "email":  user.Email,
            "role":   user.Role,
        },
        Metadata: &Metadata{
            UserID:  user.ID,
            TraceID: getCurrentTraceID(),
            Priority: 5,
        },
    }
    
    // Publish to local event bus
    s.eventBus.Publish(event)
}
```

#### Cross-Process Event (RPC)
```go
// Send event to another service process
func (s *IdentityService) NotifyUserCreated(user *User) error {
    event := &Event{
        ID:            generateID(),
        Type:          "user.created",
        Source:        "identity", 
        ProcessID:     os.Getpid(),
        ServiceName:   "identity",
        TargetService: "analytics",
        Timestamp:     time.Now().UnixNano(),
        Version:       "1.0",
        IsLocal:       false,
        Data:          user,
    }
    
    // Send via gRPC to analytics service
    ctx := context.Background()
    _, err := s.analyticsClient.PublishEvent(ctx, &RPCEvent{
        Event:        event,
        DeliveryMode: "at_least_once",
    })
    
    return err
}
```

#### Publishing Patterns
- **Local Fire-and-Forget**: Async within process
- **RPC Request-Reply**: Synchronous cross-process
- **Local Topic**: In-process pub/sub
- **Service Broadcast**: RPC to multiple services
- **Process Fanout**: Send to all service instances

### Subscribing to Events

#### Local Subscription (Within Service)
```go
// Subscribe to local events within service process
func (s *StorageService) StartEventHandlers() {
    // Topic subscription
    s.eventBus.Subscribe("user.*", func(event *Event) {
        log.Printf("User event in storage service: %v", event)
    })
    
    // Filtered subscription
    s.eventBus.Subscribe("content.uploaded", SubscriptionOptions{
        Filter: func(event *Event) bool {
            size, ok := event.Data.(map[string]interface{})["size"].(int64)
            return ok && size > 1024*1024 // Only large files
        },
        Handler: s.processLargeFile,
    })
}
```

#### Cross-Process Subscription (RPC)
```go
// Implement gRPC event service in each subprocess
type EventServiceServer struct {
    service Service
}

func (s *EventServiceServer) PublishEvent(ctx context.Context, 
    req *RPCEvent) (*EventResponse, error) {
    
    // Receive event from another service
    event := req.Event
    
    // Process based on event type
    switch event.Type {
    case "user.created":
        return s.handleUserCreated(ctx, event)
    case "content.deleted":
        return s.handleContentDeleted(ctx, event)
    default:
        return &EventResponse{Success: false}, 
            fmt.Errorf("unknown event type: %s", event.Type)
    }
}
```

#### Subscription Options
- **Local Pattern Matching**: Within process wildcards
- **RPC Event Filters**: Service-level filtering
- **Process Priority**: OS-level process priority
- **Event Batching**: Accumulate before processing
- **Error Recovery**: Process-level resilience

## Event Routing

### Topic Management

#### Service-Scoped Topics
```
# Process lifecycle events
process.identity.started
process.storage.stopped
process.*.health_check

# Service-specific application events
identity.user.created
storage.content.uploaded
ledger.transaction.confirmed
social.post.published
analytics.metric.recorded
```

#### Subprocess Topic Rules
- **Service Prefix**: Each topic starts with service name
- **Process Scope**: Events scoped to subprocess
- **Local Wildcards**: Pattern matching within process
- **RPC Topics**: Special topics for cross-process
- **System Topics**: Reserved for orchestrator

### Routing Strategies

#### Local Process Routing
```go
// Route within service subprocess
type LocalRouter struct {
    routes map[string][]EventHandler
    mu     sync.RWMutex
}

func (r *LocalRouter) Route(event *Event) {
    r.mu.RLock()
    handlers := r.matchHandlers(event.Type)
    r.mu.RUnlock()
    
    for _, handler := range handlers {
        go handler(event) // Async local delivery
    }
}
```

#### Cross-Process Routing (RPC)
```go
// Route between service subprocesses
type RPCRouter struct {
    services map[string]*grpc.ClientConn
    discovery *ServiceDiscovery
}

func (r *RPCRouter) RouteToService(event *Event) error {
    // Find target service process
    conn, err := r.discovery.GetService(event.TargetService)
    if err != nil {
        return fmt.Errorf("service not found: %s", event.TargetService)
    }
    
    // Send via gRPC
    client := NewEventServiceClient(conn)
    _, err = client.PublishEvent(context.Background(), &RPCEvent{
        Event: event,
    })
    
    return err
}
```

#### Process Discovery
- **Local Registry**: Unix socket paths
- **Service Mapping**: Process ID to service name
- **Health Status**: Live process detection
- **Load Balancing**: Multiple process instances
- **Failover**: Automatic retry on failure

## Event Processing

### Processing Patterns

#### Local Sequential Processing
```go
// Process events in order within service process
func (s *LedgerService) StartEventHandlers() {
    s.eventBus.Subscribe("transaction.*", SubscriptionOptions{
        Sequential: true,
        Handler: func(event *Event) {
            s.processTransaction(event)
            s.updateBalance(event)
            s.notifyCompletion(event)
        },
    })
}
```

#### Process-Level Concurrency
```go
// Concurrent processing with goroutines
func (s *StorageService) HandleUploads() {
    s.eventBus.Subscribe("content.uploaded", SubscriptionOptions{
        Concurrency: 5, // Max 5 concurrent handlers
        Handler: func(event *Event) {
            // Each runs in separate goroutine
            go s.generateThumbnail(event)
            go s.extractMetadata(event)
            go s.scanForVirus(event)
        },
    })
}
```

#### Cross-Process Stream Processing
```go
// Aggregate events across service processes
func (s *AnalyticsService) StartAggregation() {
    metrics := make(chan *Event, 1000)
    
    // Collect from multiple services via RPC
    go s.collectFromServices(metrics)
    
    // Process in windows
    ticker := time.NewTicker(5 * time.Second)
    for range ticker.C {
        batch := s.drainChannel(metrics)
        aggregated := s.aggregate(batch)
        s.publishMetrics(aggregated)
    }
}
```

### Event Transformation

#### Service-Level Mapping
```go
// Transform events within service process
func (s *IdentityService) TransformLegacyEvents() {
    s.eventBus.Subscribe("legacy.user", SubscriptionOptions{
        Transform: func(event *Event) *Event {
            // Map old format to new
            data := event.Data.(map[string]interface{})
            return &Event{
                Type: "user.created",
                Data: User{
                    ID:    data["user_id"].(string),
                    Email: data["user_email"].(string),
                    Name:  data["user_name"].(string),
                },
            }
        },
        Handler: s.processModernUser,
    })
}
```

#### Cross-Service Enrichment
```go
// Enrich events with data from other services
func (s *AnalyticsService) EnrichEvents() {
    s.eventBus.Subscribe("order.placed", SubscriptionOptions{
        Handler: func(event *Event) {
            order := event.Data.(*Order)
            
            // Call Identity service via RPC
            user, _ := s.identityClient.GetUser(ctx, &GetUserRequest{
                Id: order.UserID,
            })
            
            // Call Catalog service via RPC
            product, _ := s.catalogClient.GetProduct(ctx, &GetProductRequest{
                Id: order.ProductID,
            })
            
            // Create enriched event
            enriched := &EnrichedOrderEvent{
                Order:       order,
                UserName:    user.Name,
                ProductName: product.Name,
            }
            
            s.processEnrichedOrder(enriched)
        },
    })
}
```

#### Process-Level Aggregation
```go
// Aggregate events within service subprocess
func (s *LedgerService) AggregatePayments() {
    aggregator := NewAggregator(time.Minute)
    
    s.eventBus.Subscribe("payment.*", SubscriptionOptions{
        Handler: func(event *Event) {
            payment := event.Data.(*Payment)
            
            // Group by order ID
            aggregator.Add(payment.OrderID, payment)
            
            // Process when window closes
            if aggregator.ShouldFlush() {
                summary := aggregator.Flush()
                s.processPaymentSummary(summary)
            }
        },
    })
}
```

## Event Persistence

### Subprocess Event Storage

#### Per-Service Storage
Each service subprocess maintains its own event storage:
- **Local WAL**: Service-specific write-ahead log
- **Process Journal**: Events scoped to subprocess
- **Shared Storage**: Optional centralized persistence
- **Cross-Service Query**: RPC-based event queries

#### Event Journal Implementation
```go
// Per-service event journal
type ServiceEventJournal struct {
    serviceName string
    processID   int
    walPath     string
    db          *bbolt.DB  // Embedded database
}

func (j *ServiceEventJournal) Append(event *Event) error {
    // Serialize event
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    // Write to WAL first
    if err := j.writeToWAL(data); err != nil {
        return err
    }
    
    // Store in embedded DB
    return j.db.Update(func(tx *bbolt.Tx) error {
        bucket := tx.Bucket([]byte("events"))
        return bucket.Put([]byte(event.ID), data)
    })
}

func (j *ServiceEventJournal) Query(criteria QueryCriteria) ([]*Event, error) {
    var events []*Event
    
    err := j.db.View(func(tx *bbolt.Tx) error {
        bucket := tx.Bucket([]byte("events"))
        
        cursor := bucket.Cursor()
        for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
            var event Event
            if err := json.Unmarshal(v, &event); err != nil {
                continue
            }
            
            if criteria.Matches(&event) {
                events = append(events, &event)
            }
        }
        return nil
    })
    
    return events, err
}
```

### Event Sourcing

#### Event Sourcing Pattern
```typescript
// Aggregate events into state
class OrderAggregate {
  private events: Event[] = [];
  private state: OrderState;

  apply(event: Event) {
    this.events.push(event);
    this.state = this.reducer(this.state, event);
  }

  private reducer(state: OrderState, event: Event): OrderState {
    switch (event.type) {
      case 'order.created':
        return { ...state, status: 'created' };
      case 'order.paid':
        return { ...state, status: 'paid' };
      case 'order.shipped':
        return { ...state, status: 'shipped' };
      default:
        return state;
    }
  }
}
```

#### Snapshots
```typescript
// State snapshots for performance
interface Snapshot {
  aggregateId: string;
  version: number;
  state: any;
  timestamp: number;
}

// Load aggregate with snapshot
async function loadAggregate(id: string) {
  const snapshot = await getLatestSnapshot(id);
  const events = await getEventsSince(id, snapshot.version);
  
  const aggregate = new OrderAggregate();
  aggregate.loadSnapshot(snapshot);
  events.forEach(event => aggregate.apply(event));
  
  return aggregate;
}
```

### Event Replay

#### Replay Scenarios
- **System Recovery**: Rebuild state
- **Debugging**: Reproduce issues
- **Analytics**: Historical analysis
- **Migration**: Data transformation
- **Testing**: Scenario replay

#### Replay Implementation
```typescript
// Replay events from point in time
async function replayEvents(from: Date, to: Date) {
  const events = await eventStore.query({
    timeRange: { from, to },
    orderBy: 'timestamp'
  });

  for (const event of events) {
    await eventBus.replay(event);
  }
}

// Selective replay
async function replayForAggregate(aggregateId: string) {
  const events = await eventStore.query({
    aggregateId,
    orderBy: 'version'
  });

  const aggregate = new OrderAggregate();
  events.forEach(event => aggregate.apply(event));
  
  return aggregate;
}
```

## Event Ordering

### Ordering Guarantees

#### FIFO Ordering
- **Per-Topic**: Order within topic
- **Per-Partition**: Order within partition
- **Per-Producer**: Order from same source
- **Global**: Total order (expensive)
- **Causal**: Preserves causality

#### Ordering Implementation
```typescript
// Ensure ordered processing
eventBus.subscribe('payment.*', {
  ordered: true,
  partitionKey: (event) => event.data.userId,
  handler: async (event) => {
    // Events for same user processed in order
    await processPayment(event);
  }
});
```

### Concurrent Processing

#### Parallelism Control
```typescript
// Control concurrency level
eventBus.subscribe('image.process', {
  concurrency: 10,
  ordered: false,
  handler: async (event) => {
    // Up to 10 images processed in parallel
    await processImage(event);
  }
});

// Ordered within partition
eventBus.subscribe('user.action', {
  concurrency: 5,
  partitionKey: (event) => event.data.userId,
  ordered: true,
  handler: async (event) => {
    // Parallel processing but ordered per user
    await processUserAction(event);
  }
});
```

## Error Handling

### Retry Strategies

#### Exponential Backoff
```typescript
eventBus.subscribe('external.api', {
  retry: {
    maxAttempts: 5,
    initialDelay: 1000,
    maxDelay: 60000,
    multiplier: 2,
    jitter: true
  },
  handler: async (event) => {
    await callExternalAPI(event);
  }
});
```

#### Dead Letter Queue
```typescript
// Failed events go to DLQ
eventBus.subscribe('critical.operation', {
  retry: {
    maxAttempts: 3,
    deadLetterQueue: 'dlq.critical'
  },
  handler: async (event) => {
    await criticalOperation(event);
  }
});

// Process dead letters
eventBus.subscribe('dlq.critical', {
  handler: async (event) => {
    await notifyAdmins(event);
    await logFailure(event);
  }
});
```

### Error Recovery

#### Circuit Breaker
```typescript
eventBus.subscribe('fragile.service', {
  circuitBreaker: {
    failureThreshold: 0.5,
    resetTimeout: 30000,
    monitoringPeriod: 60000
  },
  handler: async (event) => {
    await callFragileService(event);
  }
});
```

#### Fault Tolerance
```typescript
// Fallback on failure
eventBus.subscribe('user.notification', {
  fallback: async (event, error) => {
    // Primary notification failed
    await sendEmailFallback(event);
  },
  handler: async (event) => {
    await sendPushNotification(event);
  }
});
```

## Performance Optimization

### Batching

#### Batch Processing
```typescript
// Process events in batches
eventBus.subscribe('metric.reported', {
  batch: {
    size: 100,
    timeout: 5000
  },
  handler: async (events) => {
    await bulkInsertMetrics(events);
  }
});
```

#### Micro-Batching
```typescript
// Small, frequent batches
eventBus.subscribe('log.entry', {
  microBatch: {
    size: 10,
    timeout: 100
  },
  handler: async (events) => {
    await writeLogBatch(events);
  }
});
```

### Caching

#### Event Caching
```typescript
// Cache frequently accessed events
const eventCache = new LRUCache<string, Event>({
  max: 10000,
  ttl: 3600000 // 1 hour
});

eventBus.use(async (event, next) => {
  // Cache read events
  if (event.type.startsWith('read.')) {
    const cached = eventCache.get(event.data.id);
    if (cached) {
      return cached;
    }
  }
  
  const result = await next();
  
  // Cache results
  if (event.type.startsWith('read.')) {
    eventCache.set(event.data.id, result);
  }
  
  return result;
});
```

### Compression

#### Event Compression
```typescript
// Compress large events
eventBus.use(async (event, next) => {
  if (event.data && sizeof(event.data) > 1024) {
    event.data = await compress(event.data);
    event.metadata.compressed = true;
  }
  
  return next();
});

// Decompress on receive
eventBus.subscribe('*', {
  decompress: true,
  handler: async (event) => {
    // Event automatically decompressed
    await processEvent(event);
  }
});
```

## Monitoring and Observability

### Subprocess Event Metrics

#### Per-Service Metrics
Each service subprocess tracks its own metrics:
- **Local Event Rate**: Events/sec within process
- **RPC Event Rate**: Cross-process events/sec
- **Process Memory**: Event queue memory usage
- **Handler Latency**: Per-handler processing time
- **Process Health**: Service subprocess status

#### Cross-Process Metrics
- **RPC Latency**: Service-to-service latency
- **Failed Deliveries**: Cross-process failures
- **Service Discovery**: Process registration time
- **Connection Pool**: gRPC connection stats
- **Process Lifecycle**: Start/stop events

### Event Tracing

#### Process-Level Tracing
```go
// Trace events within service process
func (s *Service) TraceEvent(event *Event) {
    span := s.tracer.Start(fmt.Sprintf("event.%s", event.Type))
    span.SetAttributes(
        attribute.String("service", s.name),
        attribute.Int("process_id", os.Getpid()),
        attribute.String("event_id", event.ID),
    )
    defer span.End()
    
    // Process event
    if err := s.handleEvent(event); err != nil {
        span.SetStatus(codes.Error, err.Error())
    }
}
```

#### Cross-Process Correlation
```go
// Correlate events across service processes
func (s *Service) PropagateTrace(event *Event) {
    // Extract trace from incoming RPC
    ctx := extract(context.Background(), event.Metadata)
    span := s.tracer.Start(ctx, "cross_process_event")
    
    // Add subprocess info
    span.SetAttributes(
        attribute.String("source_service", event.Source),
        attribute.String("target_service", s.name),
        attribute.Int("source_pid", event.ProcessID),
        attribute.Int("target_pid", os.Getpid()),
    )
    
    // Continue trace chain
    event.Metadata.TraceID = span.SpanContext().TraceID().String()
    await processOrderWithContext(event, relatedEvents);
  }
});
```

### Event Analytics

#### Real-Time Analytics
```typescript
// Stream processing for analytics
eventBus.stream('user.action.*')
  .map(event => ({
    action: event.type,
    userId: event.data.userId,
    timestamp: event.timestamp
  }))
  .window(60000) // 1 minute windows
  .groupBy('action')
  .count()
  .subscribe(counts => {
    publishActionMetrics(counts);
  });
```

#### Event Querying
```typescript
// Query historical events
const events = await eventStore.query({
  type: 'order.completed',
  timeRange: {
    from: new Date('2024-01-01'),
    to: new Date('2024-01-31')
  },
  filter: {
    'data.amount': { $gt: 1000 }
  },
  orderBy: 'timestamp',
  limit: 100
});
```

## Integration Patterns

### External Systems

#### Webhook Integration
```typescript
// Publish events to webhooks
eventBus.subscribe('order.shipped', {
  webhook: {
    url: 'https://partner.api/webhooks/shipment',
    headers: {
      'X-API-Key': process.env.PARTNER_API_KEY
    },
    retry: {
      maxAttempts: 3,
      backoff: 'exponential'
    }
  }
});
```

#### Message Queue Bridge
```typescript
// Bridge to external message queues
eventBus.bridge({
  source: 'internal.*',
  destination: 'amqp://rabbitmq/exchange',
  transform: (event) => ({
    routingKey: event.type,
    payload: event.data,
    headers: event.metadata
  })
});
```

### Database Integration

#### Change Data Capture
```typescript
// Capture database changes as events
dbConnector.on('change', (change) => {
  const event = {
    type: `database.${change.table}.${change.operation}`,
    source: 'database',
    data: {
      table: change.table,
      operation: change.operation,
      before: change.before,
      after: change.after
    }
  };
  
  eventBus.publish(event);
});
```

#### Event-Driven Cache
```typescript
// Invalidate cache on events
eventBus.subscribe('data.updated', {
  handler: async (event) => {
    await cache.invalidate(event.data.key);
    await cache.preload(event.data.key, event.data.value);
  }
});
```

## Security

### Event Authentication

#### Event Signing
```typescript
// Sign events for authenticity
eventBus.use(async (event, next) => {
  event.signature = await sign(event, privateKey);
  return next();
});

// Verify event signatures
eventBus.subscribe('external.*', {
  verify: true,
  handler: async (event) => {
    // Signature verified before handler
    await processExternalEvent(event);
  }
});
```

#### Access Control
```typescript
// Role-based event access
eventBus.subscribe('admin.*', {
  authorize: (event, context) => {
    return context.user.roles.includes('admin');
  },
  handler: async (event) => {
    await processAdminEvent(event);
  }
});
```

### Event Encryption

#### Payload Encryption
```typescript
// Encrypt sensitive events
eventBus.use(async (event, next) => {
  if (event.metadata.sensitive) {
    event.data = await encrypt(event.data, encryptionKey);
    event.metadata.encrypted = true;
  }
  
  return next();
});

// Automatic decryption
eventBus.subscribe('sensitive.*', {
  decrypt: true,
  handler: async (event) => {
    // Event automatically decrypted
    await processSensitiveEvent(event);
  }
});
```

## Testing

### Event Testing

#### Unit Testing
```typescript
// Test event handlers
describe('OrderHandler', () => {
  it('should process order events', async () => {
    const event = {
      type: 'order.created',
      data: { orderId: '123', amount: 100 }
    };
    
    const result = await orderHandler(event);
    
    expect(result.status).toBe('processed');
  });
});
```

#### Integration Testing
```typescript
// Test event flow
describe('Order Flow', () => {
  it('should complete order process', async () => {
    const bus = new TestEventBus();
    
    // Subscribe handlers
    bus.subscribe('order.created', createHandler);
    bus.subscribe('payment.completed', paymentHandler);
    
    // Publish initial event
    await bus.publish({
      type: 'order.created',
      data: { orderId: '123' }
    });
    
    // Verify events were published
    expect(bus.published).toContainEqual(
      expect.objectContaining({
        type: 'payment.requested'
      })
    );
  });
});
```

### Event Simulation

#### Load Testing
```typescript
// Simulate high event load
async function loadTest() {
  const events = generateEvents(10000);
  
  const start = Date.now();
  
  await Promise.all(
    events.map(event => eventBus.publish(event))
  );
  
  const duration = Date.now() - start;
  const rate = events.length / (duration / 1000);
  
  console.log(`Published ${rate} events/second`);
}
```

## Best Practices

### Subprocess Event Design
- **Process Isolation**: Keep events scoped to services
- **Service Identity**: Include service name and PID
- **Local vs RPC**: Distinguish event types clearly
- **Process State**: Track subprocess lifecycle
- **Versioned Schema**: Support service evolution

### Publishing Guidelines
- **Local First**: Use local events within service
- **RPC Selective**: Only cross-process when needed
- **Process Context**: Include process metadata
- **Connection Reuse**: Pool gRPC connections
- **Error Boundaries**: Isolate process failures

### Subprocess Patterns
- **Service Topics**: Prefix with service name
- **Process Health**: Monitor subprocess status
- **Resource Limits**: Respect process constraints
- **Graceful Shutdown**: Clean event handling
- **Hot Reload**: Support service updates

### Operational Practices
- **Process Monitoring**: Track subprocess metrics
- **Service Discovery**: Maintain process registry
- **Fault Isolation**: Contain process failures
- **Security Boundaries**: Process-level access control
- **Audit Trail**: Track cross-process events

### Performance Optimization
- **Local Processing**: Minimize RPC calls
- **Unix Sockets**: Use for local communication
- **Process Affinity**: Co-locate related services
- **Memory Efficiency**: Per-process event queues
- **Resource Pooling**: Share gRPC connections