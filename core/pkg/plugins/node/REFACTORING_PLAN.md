# Node Plugin Mesh Compliance Refactoring Plan

## Current Issues

The node plugin currently violates the core plugin architecture principles:

1. **Direct RPC Communication**: Uses Unix socket RPC instead of mesh network
2. **No gRPC Service**: Doesn't implement the `NodePlugin` gRPC service from proto
3. **No Event Publishing**: Doesn't publish events to mesh event bus
4. **No Mesh Client**: Doesn't use mesh network for communication
5. **Location Dependent**: Tied to local Unix socket, not network transparent

## Required Changes

### 1. Implement gRPC Service

Replace the current RPC implementation with proper gRPC service:

```go
// Implement the NodePlugin service from proto/v1/node.proto
type nodePluginServer struct {
    nodev1.UnimplementedNodePluginServer
    plugin *plugin.Plugin
    meshClient mesh.Client
    eventPublisher mesh.EventPublisher
}
```

### 2. Use Mesh Client for Communication

- Remove direct Unix socket RPC
- Use mesh client for all external communication
- Register service with mesh network on startup

### 3. Implement Event Publishing

Publish events to mesh event bus:
- `node.peer.connected` - When a peer connects
- `node.peer.disconnected` - When a peer disconnects  
- `node.network.status.changed` - When network health changes
- `node.peer.discovered` - When new peers are discovered

### 4. Update Main Entry Point

```go
func main() {
    // 1. Create mesh client
    meshClient, err := mesh.NewClient(config)
    
    // 2. Create plugin with mesh integration
    plugin := createNodePlugin(meshClient)
    
    // 3. Create gRPC server
    server := grpc.NewServer()
    nodev1.RegisterNodePluginServer(server, plugin)
    
    // 4. Register with mesh
    meshClient.RegisterService("node", server)
    
    // 5. Start serving through mesh
    meshClient.Serve()
}
```

### 5. Event Publishing Implementation

```go
func (p *Plugin) publishPeerEvent(eventType string, peerID string, data map[string]interface{}) {
    event := mesh.Event{
        Type: fmt.Sprintf("node.peer.%s", eventType),
        Source: p.config.NodeID,
        Data: data,
        Timestamp: time.Now(),
    }
    
    p.eventPublisher.Publish(event)
}
```

### 6. Consume Events from Other Plugins

Subscribe to relevant events:
- Storage content announcements
- Identity verification events
- Analytics requests

## Implementation Steps

### Phase 1: Create Mesh-Compliant Structure
1. Create new `grpc_server.go` implementing the gRPC service
2. Add mesh client integration
3. Keep existing functionality but wrap in gRPC methods

### Phase 2: Add Event Publishing
1. Add event publisher to plugin struct
2. Publish events at key points (connect, disconnect, etc)
3. Create event types and schemas

### Phase 3: Remove Old RPC
1. Remove the generic RPC server code
2. Update main.go to use mesh client
3. Remove Unix socket handling

### Phase 4: Add Event Consumption
1. Subscribe to relevant events from other plugins
2. React to storage and identity events
3. Implement event handlers

## Benefits After Refactoring

1. **Location Transparency**: Plugin can run anywhere on the mesh
2. **Service Discovery**: Other plugins can find node service via mesh
3. **Load Balancing**: Multiple node instances with automatic balancing
4. **Security**: Mesh handles authentication and encryption
5. **Monitoring**: All communication observable through mesh
6. **Event-Driven**: Loose coupling with other plugins

## Testing Strategy

1. Create mesh client mock for unit tests
2. Test gRPC service implementation
3. Verify event publishing
4. Integration tests with mesh network
5. Multi-instance testing

## Migration Path

To avoid breaking existing functionality:
1. Implement new gRPC service alongside old RPC
2. Add feature flag to switch between implementations
3. Test thoroughly with mesh
4. Remove old implementation

## Example Usage After Refactoring

```go
// Other plugins can now use the node plugin via mesh
meshClient := mesh.Connect()
nodeClient := nodev1.NewNodePluginClient(meshClient.GetConnection("node"))

// Make calls through mesh
status, err := nodeClient.GetNetworkStatus(ctx, &nodev1.GetNetworkStatusRequest{})

// Subscribe to events
events := meshClient.Subscribe("node.peer.*")
for event := range events {
    // Handle peer events
}
```

This refactoring will make the node plugin fully compliant with the Blackhole plugin architecture.