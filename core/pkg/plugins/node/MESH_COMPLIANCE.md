# Node Plugin Mesh Compliance Documentation

## Overview

The node plugin has been refactored to comply with the Blackhole mesh communication architecture. This document describes the changes made and how to use the mesh-compliant version.

## Architecture Changes

### Before (Direct RPC)
- Plugin communicated directly with other services via RPC
- Tight coupling between services
- No event-driven architecture
- Limited scalability

### After (Mesh-Compliant)
- All communication goes through the mesh network
- Event-driven architecture with publish/subscribe
- Location transparency for distributed deployment
- Better scalability and fault tolerance

## Key Components

### 1. Mesh Client (`mesh/client.go`)
Provides interface for mesh communication:
```go
type Client interface {
    RegisterService(name string, endpoint string) error
    GetConnection(serviceName string) (*grpc.ClientConn, error)
    PublishEvent(event Event) error
    Subscribe(pattern string) (<-chan Event, error)
    Unsubscribe(pattern string) error
    Close() error
}
```

### 2. gRPC Server (`grpc_server.go`)
Implements the NodePlugin service defined in proto:
- Handles all plugin operations (Start, Stop, Connect, etc.)
- Publishes events for important state changes
- Integrates with mesh client for event publishing

### 3. Mesh Entry Point (`main_mesh.go`)
New entry point that:
- Connects to mesh router
- Registers the node service
- Sets up gRPC server
- Subscribes to relevant events
- Handles graceful shutdown

## Event Patterns

### Published Events
- `node.started` - When plugin starts
- `node.stopped` - When plugin stops
- `node.peer.connected` - When a peer connects
- `node.peer.disconnected` - When a peer disconnects
- `node.peer.discovered` - When new peers are discovered
- `node.network.status.changed` - When network status changes

### Subscribed Events
- `storage.content.*` - Storage content events
- `identity.peer.*` - Identity verification events
- `mesh.topology.*` - Mesh topology changes

## Building and Running

### Build Mesh-Compliant Version
```bash
cd core/pkg/plugins/node
GOWORK=off go build -o node-mesh main_mesh.go grpc_server.go
```

### Configuration
The mesh-compliant version requires additional configuration:
```yaml
mesh:
  router_address: "localhost:50000"
  retry_interval: "5s"
  max_retries: 10
```

### Running
```bash
./node-mesh --config=config.yaml
```

## Testing

### Unit Tests
```bash
# Test mesh client
GOWORK=off go test ./mesh -v

# Test gRPC server integration
GOWORK=off go test -v grpc_integration_test.go grpc_server.go main_mesh.go
```

### Integration Testing
1. Start mesh router
2. Start mesh-compliant node plugin
3. Use grpcurl to test operations:
```bash
# List services
grpcurl -plaintext localhost:PORT list

# Get plugin info
grpcurl -plaintext localhost:PORT blackhole.node.v1.NodePlugin/GetInfo
```

## Migration Guide

### For Plugin Developers
1. Replace direct RPC calls with mesh client calls
2. Publish events for important state changes
3. Subscribe to relevant events from other plugins
4. Use gRPC service definition from proto

### For Users
1. Update configuration to include mesh settings
2. Ensure mesh router is running before starting plugin
3. Monitor events for debugging and monitoring

## Benefits

1. **Decoupling**: Services no longer need direct connections
2. **Scalability**: Can distribute plugins across multiple nodes
3. **Resilience**: Mesh handles connection failures and retries
4. **Observability**: All events flow through mesh for monitoring
5. **Flexibility**: Can add new event consumers without changing publishers

## Known Limitations

1. Mesh client implementation is currently a mock
2. Need actual mesh router for production use
3. Event schema validation not yet implemented
4. No event persistence/replay capabilities yet

## Future Improvements

1. Implement real mesh client with router connection
2. Add event schema validation
3. Implement event persistence and replay
4. Add metrics for mesh communication
5. Implement circuit breakers for resilience
6. Add distributed tracing support

## Troubleshooting

### Plugin Won't Start
- Check mesh router is running
- Verify mesh configuration is correct
- Check logs for connection errors

### Events Not Being Received
- Verify subscription patterns are correct
- Check mesh router logs
- Ensure publisher and subscriber are connected

### Performance Issues
- Monitor event queue sizes
- Check network latency to mesh router
- Consider event batching for high-frequency events