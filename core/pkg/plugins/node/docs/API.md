# Node Plugin API Reference

The Node Plugin provides a gRPC API for P2P networking and distributed communication.

## Service Definition

```protobuf
service NodeService {
    // Node information
    rpc GetInfo(GetInfoRequest) returns (GetInfoResponse);
    rpc GetStatus(GetStatusRequest) returns (GetStatusResponse);
    
    // Peer management
    rpc ListPeers(ListPeersRequest) returns (ListPeersResponse);
    rpc Connect(ConnectRequest) returns (ConnectResponse);
    rpc Disconnect(DisconnectRequest) returns (DisconnectResponse);
    rpc GetPeer(GetPeerRequest) returns (GetPeerResponse);
    
    // Messaging
    rpc Send(SendRequest) returns (SendResponse);
    rpc Publish(PublishRequest) returns (PublishResponse);
    rpc Subscribe(SubscribeRequest) returns (stream SubscribeResponse);
    
    // Network operations
    rpc Ping(PingRequest) returns (PingResponse);
    rpc FindPeer(FindPeerRequest) returns (FindPeerResponse);
    rpc FindProviders(FindProvidersRequest) returns (FindProvidersResponse);
    
    // Content routing
    rpc Provide(ProvideRequest) returns (ProvideResponse);
    rpc StartProviding(StartProvidingRequest) returns (StartProvidingResponse);
    rpc StopProviding(StopProvidingRequest) returns (StopProvidingResponse);
    
    // Plugin lifecycle
    rpc Health(HealthRequest) returns (HealthResponse);
    rpc Shutdown(ShutdownRequest) returns (ShutdownResponse);
}
```

## Methods

### GetInfo

Get information about the node.

**Request:**
```protobuf
message GetInfoRequest {}
```

**Response:**
```protobuf
message GetInfoResponse {
    string id = 1;              // Peer ID
    repeated string addresses = 2;  // Multiaddresses
    repeated string protocols = 3;  // Supported protocols
    string agent_version = 4;   // Agent version string
    string protocol_version = 5; // Protocol version
}
```

**Example:**
```go
info, err := client.GetInfo(ctx, &nodepb.GetInfoRequest{})
```

### GetStatus

Get current node status and statistics.

**Request:**
```protobuf
message GetStatusRequest {}
```

**Response:**
```protobuf
message GetStatusResponse {
    string status = 1;          // "online", "offline", "syncing"
    int32 peer_count = 2;       // Number of connected peers
    NetworkStats network = 3;    // Network statistics
    repeated string errors = 4;  // Current errors
}

message NetworkStats {
    int64 bytes_sent = 1;
    int64 bytes_received = 2;
    int64 messages_sent = 3;
    int64 messages_received = 4;
}
```

### ListPeers

List all connected peers.

**Request:**
```protobuf
message ListPeersRequest {
    // Optional filters
    bool connected_only = 1;
    repeated string protocols = 2;  // Filter by protocol
}
```

**Response:**
```protobuf
message ListPeersResponse {
    repeated Peer peers = 1;
}

message Peer {
    string id = 1;
    repeated string addresses = 2;
    repeated string protocols = 3;
    int64 connected_at = 4;     // Unix timestamp
    NetworkStats stats = 5;
    map<string, string> metadata = 6;
}
```

### Connect

Connect to a specific peer.

**Request:**
```protobuf
message ConnectRequest {
    string peer_id = 1;         // Required
    repeated string addresses = 2;  // Known addresses
    int32 timeout = 3;          // Connection timeout in seconds
}
```

**Response:**
```protobuf
message ConnectResponse {
    bool success = 1;
    string error = 2;           // Error message if failed
    Peer peer = 3;              // Connected peer info
}
```

### Disconnect

Disconnect from a peer.

**Request:**
```protobuf
message DisconnectRequest {
    string peer_id = 1;
}
```

**Response:**
```protobuf
message DisconnectResponse {
    bool success = 1;
    string error = 2;
}
```

### Send

Send a direct message to a peer.

**Request:**
```protobuf
message SendRequest {
    string peer_id = 1;
    bytes data = 2;
    string protocol = 3;        // Protocol to use
    int32 timeout = 4;          // Timeout in seconds
}
```

**Response:**
```protobuf
message SendResponse {
    bool success = 1;
    string error = 2;
    bytes response = 3;         // Response data if any
}
```

### Publish

Publish a message to a topic.

**Request:**
```protobuf
message PublishRequest {
    string topic = 1;
    bytes data = 2;
    map<string, string> headers = 3;  // Optional headers
}
```

**Response:**
```protobuf
message PublishResponse {
    bool success = 1;
    string error = 2;
    string message_id = 3;      // Published message ID
}
```

### Subscribe

Subscribe to messages on a topic (streaming).

**Request:**
```protobuf
message SubscribeRequest {
    string topic = 1;
    bool include_self = 2;      // Include own messages
}
```

**Response (stream):**
```protobuf
message SubscribeResponse {
    string message_id = 1;
    string peer_id = 2;         // Sender peer ID
    bytes data = 3;
    string topic = 4;
    int64 timestamp = 5;        // Unix timestamp
    map<string, string> headers = 6;
}
```

**Example:**
```go
stream, err := client.Subscribe(ctx, &nodepb.SubscribeRequest{
    Topic: "my-topic",
})

for {
    msg, err := stream.Recv()
    if err == io.EOF {
        break
    }
    // Process message
}
```

### Ping

Ping a peer to check connectivity.

**Request:**
```protobuf
message PingRequest {
    string peer_id = 1;
}
```

**Response:**
```protobuf
message PingResponse {
    bool success = 1;
    int64 latency = 2;          // Round trip time in milliseconds
    string error = 3;
}
```

### FindPeer

Find a peer in the network.

**Request:**
```protobuf
message FindPeerRequest {
    string peer_id = 1;
    int32 timeout = 2;          // Search timeout in seconds
}
```

**Response:**
```protobuf
message FindPeerResponse {
    bool found = 1;
    Peer peer = 2;
    repeated string closer_peers = 3;  // Peers closer to target
}
```

### FindProviders

Find peers providing specific content.

**Request:**
```protobuf
message FindProvidersRequest {
    string cid = 1;             // Content ID
    int32 count = 2;            // Max number of providers
    int32 timeout = 3;          // Search timeout
}
```

**Response:**
```protobuf
message FindProvidersResponse {
    repeated Provider providers = 1;
}

message Provider {
    string peer_id = 1;
    repeated string addresses = 2;
    int64 last_seen = 3;        // Unix timestamp
}
```

### Provide

Announce that this node provides content.

**Request:**
```protobuf
message ProvideRequest {
    string cid = 1;             // Content ID
    bool recursive = 2;         // Provide all blocks recursively
}
```

**Response:**
```protobuf
message ProvideResponse {
    bool success = 1;
    string error = 2;
}
```

### Health

Check plugin health status.

**Request:**
```protobuf
message HealthRequest {}
```

**Response:**
```protobuf
message HealthResponse {
    bool healthy = 1;
    string status = 2;          // "healthy", "degraded", "unhealthy"
    map<string, string> checks = 3;  // Individual health checks
}
```

### Shutdown

Gracefully shutdown the plugin.

**Request:**
```protobuf
message ShutdownRequest {
    int32 timeout = 1;          // Graceful shutdown timeout
}
```

**Response:**
```protobuf
message ShutdownResponse {
    bool success = 1;
    string error = 2;
}
```

## Error Handling

All methods may return gRPC errors with the following codes:

- `INVALID_ARGUMENT`: Invalid request parameters
- `NOT_FOUND`: Peer or content not found
- `UNAVAILABLE`: Service temporarily unavailable
- `DEADLINE_EXCEEDED`: Operation timeout
- `RESOURCE_EXHAUSTED`: Resource limits exceeded
- `INTERNAL`: Internal plugin error

Example error handling:

```go
info, err := client.GetInfo(ctx, &nodepb.GetInfoRequest{})
if err != nil {
    if status.Code(err) == codes.Unavailable {
        // Service is down
    }
    return err
}
```

## Events

The plugin emits events that can be subscribed to via the mesh network:

- `node.peer.connected` - Peer connected
- `node.peer.disconnected` - Peer disconnected
- `node.peer.discovered` - New peer discovered
- `node.message.received` - Message received
- `node.network.changed` - Network topology changed

## Rate Limiting

The API implements rate limiting for certain operations:

- Connect: 10 requests per minute per peer
- Send: 100 messages per minute per peer
- Publish: 1000 messages per minute per topic
- FindPeer: 30 requests per minute

## Best Practices

1. **Connection Management**
   - Reuse connections when possible
   - Implement exponential backoff for reconnects
   - Monitor connection health

2. **Message Handling**
   - Use appropriate message sizes (< 1MB recommended)
   - Implement message deduplication
   - Handle stream disconnections gracefully

3. **Resource Usage**
   - Close unused subscriptions
   - Limit concurrent operations
   - Monitor memory usage

4. **Security**
   - Validate all peer IDs
   - Sanitize message content
   - Implement application-level encryption if needed