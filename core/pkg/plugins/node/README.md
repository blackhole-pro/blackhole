# Node Plugin

The Node plugin provides P2P networking capabilities for the Blackhole Foundation platform.

## Versions

This plugin provides two implementations:

1. **Direct RPC Version** (`main.go`) - Legacy implementation using direct RPC
2. **Mesh-Compliant Version** (`main_mesh.go`) - New implementation using mesh communication

See [MESH_COMPLIANCE.md](MESH_COMPLIANCE.md) for details on the mesh-compliant version.

## Scope

This plugin is responsible for:
- **P2P Networking**: Managing peer-to-peer connections using libp2p
- **Peer Discovery**: Automatic peer discovery via mDNS, DHT, and bootstrap nodes
- **Network Health Monitoring**: Monitoring network connectivity and peer health

## NOT in Scope

This plugin does NOT handle:
- Identity management (handled by identity plugin)
- Data storage (handled by storage plugin)
- Content routing (handled by indexer plugin)
- Economic transactions (handled by ledger plugin)

## Configuration

```json
{
  "nodeId": "unique-node-identifier",
  "version": "1.0.0",
  "p2pPort": 4001,
  "listenAddresses": ["/ip4/0.0.0.0/tcp/4001"],
  "bootstrapPeers": ["peer1-address", "peer2-address"],
  "enableDiscovery": true,
  "discoveryMethod": "bootstrap",
  "discoveryInterval": "30s",
  "healthCheckInterval": "10s",
  "peerTimeout": "60s",
  "maxPeers": 50,
  "maxBandwidthMbps": 100,
  "connectionTimeout": "30s",
  "enableEncryption": true,
  "privateKeyPath": "/path/to/key"
}
```

### Mesh-Compliant Configuration
For the mesh-compliant version, add:
```json
{
  "mesh": {
    "router_address": "localhost:50000",
    "retry_interval": "5s",
    "max_retries": 10
  }
}
```

## Methods

### listPeers
List connected peers with optional filtering and pagination.

Parameters:
- `status` (string, optional): Filter by peer status
- `limit` (number, optional): Maximum peers to return (default: 50)
- `offset` (number, optional): Pagination offset

### connectPeer
Connect to a specific peer.

Parameters:
- `peerId` (string, required): Peer identifier
- `address` (string, optional): Peer address

### disconnectPeer
Disconnect from a specific peer.

Parameters:
- `peerId` (string, required): Peer identifier
- `reason` (string, optional): Disconnect reason

### getNetworkStatus
Get current network status and health metrics.

No parameters required.

### discoverPeers
Discover new peers using the specified method.

Parameters:
- `method` (string, optional): Discovery method (mdns, dht, bootstrap)
- `maxPeers` (number, optional): Maximum peers to discover

## Building

### Direct RPC Version
```bash
cd core/pkg/plugins/node
go build -o node-plugin
```

### Mesh-Compliant Version
```bash
cd core/pkg/plugins/node
GOWORK=off go build -o node-mesh main_mesh.go grpc_server.go
```

## Running

The plugin is designed to be executed by the Blackhole plugin framework via RPC.

Environment variables:
- `NODE_ID`: Node identifier
- `PLUGIN_CONFIG_PATH`: Path to configuration file
- `BOOTSTRAP_PEERS`: Comma-separated list of bootstrap peers

### Direct RPC Version
```bash
./node-plugin
```

### Mesh-Compliant Version
```bash
./node-mesh --config=config.yaml
```

## Testing

Run all tests:
```bash
# Unit tests
GOWORK=off go test ./... -v

# Integration tests
GOWORK=off go test -v grpc_integration_test.go grpc_server.go main_mesh.go
```

## Development

To implement actual P2P functionality:

1. Add libp2p dependencies to go.mod
2. Replace mock implementations with real libp2p code
3. Implement proper peer connection management
4. Add protocol handlers for Blackhole-specific protocols
5. Integrate with the identity plugin for peer authentication

## Code Organization

```
node/
├── main.go                 # Direct RPC entry point
├── main_mesh.go           # Mesh-compliant entry point
├── grpc_server.go         # gRPC service implementation
├── plugin.yaml            # Plugin manifest
├── plugin-mesh.yaml       # Mesh-compliant manifest
├── go.mod                 # Go module definition
├── Makefile               # Build automation
├── README.md              # This file
├── MESH_COMPLIANCE.md     # Mesh compliance documentation
├── REFACTORING_PLAN.md    # Refactoring plan
├── proto/
│   └── v1/
│       ├── node.proto     # gRPC service definition
│       ├── node.pb.go     # Generated protobuf code
│       └── node_grpc.pb.go # Generated gRPC code
├── types/                 # Type definitions
│   ├── types.go           # Core types
│   └── errors.go          # Typed errors
├── plugin/                # Main plugin logic
│   ├── node.go            # Plugin implementation
│   └── config.go          # Configuration validation
├── p2p/                   # P2P functionality
│   ├── peer_manager.go    # Peer management
│   └── peer_manager_test.go
├── discovery/             # Peer discovery
│   ├── discovery.go       # Discovery implementation
│   └── discovery_test.go
├── health/                # Health monitoring
│   ├── monitor.go         # Health monitor
│   └── monitor_test.go
├── network/               # Network management
│   ├── manager.go         # Network manager
│   └── manager_test.go
├── handlers/              # Message handlers
│   ├── handler.go         # Handler implementation
│   └── handler_test.go
└── mesh/                  # Mesh communication
    ├── client.go          # Mesh client interface
    └── client_test.go     # Mesh client tests
```

## Compliance

This plugin complies with:
- [Development Guidelines](../../../../ecosystem/docs/06_guides/06_01-development_guidelines.md)
- [Plugin Architecture](../README.md)
- Mesh communication patterns (mesh-compliant version)