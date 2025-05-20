# Node Service Architecture

This directory contains the comprehensive architecture documentation for the Node service, which manages P2P networking, node operations, lifecycle, and communication protocols in the Blackhole platform.

## Documentation Structure

### Core Architecture
- [Node Core Architecture](./node_core_architecture.md) - Core node operations and lifecycle management
- [State Management](./state_management.md) - Node state management and distributed state coordination
- [Bootstrap Sequence](./bootstrap_sequence.md) - Node startup, initialization, and bootstrap processes

### Networking Protocols
- [P2P Networking](./p2p_networking.md) - Complete P2P networking stack built on libp2p
- [Network Synchronization](./network_synchronization.md) - State synchronization and data consistency protocols
- [Message Routing](./message_routing.md) - Message delivery and routing architecture

### Communication Systems
- [Event System](./event_system.md) - Event-driven communication and pub/sub architecture

## Service Overview

The Node service is responsible for:

1. **P2P Networking**
   - Manages network connections using libp2p
   - Handles peer discovery and DHT operations
   - Maintains network topology and routing

2. **Node Operations**
   - Manages node lifecycle and configuration
   - Handles bootstrap and shutdown sequences
   - Monitors node health and performance

3. **Network Protocols**
   - Implements network synchronization protocols
   - Manages consensus coordination
   - Handles gossip protocol dissemination

4. **Message Routing**
   - Routes messages between peers
   - Handles protocol upgrades and fallbacks
   - Manages connection pooling and optimization

5. **Event System**
   - Provides event-driven communication
   - Manages pub/sub messaging
   - Handles distributed event coordination

## Technical Architecture

The Node service runs as an independent OS process communicating via gRPC:

```go
// Node service interface
type NodeService interface {
    // Networking
    ConnectPeer(context.Context, *PeerInfo) error
    DisconnectPeer(context.Context, *PeerID) error
    GetConnectedPeers(context.Context) ([]*PeerInfo, error)
    
    // Messaging
    SendMessage(context.Context, *Message) error
    Subscribe(topic string) (chan Message, error)
    
    // Synchronization  
    SyncState(context.Context, *PeerID) error
    GetNetworkState(context.Context) (*NetworkState, error)
}
```

## Integration Points

- **Storage Service**: For content persistence and retrieval
- **Identity Service**: For peer authentication and authorization
- **Telemetry Service**: For monitoring and metrics
- **Ledger Service**: For consensus operations