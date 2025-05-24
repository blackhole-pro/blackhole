# Node Plugin

The Node plugin provides P2P networking capabilities for the Blackhole Foundation platform.

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

```bash
cd core/pkg/plugins/node
go build -o node-plugin
```

## Running

The plugin is designed to be executed by the Blackhole plugin framework via RPC.

Environment variables:
- `NODE_ID`: Node identifier
- `PLUGIN_CONFIG_PATH`: Path to configuration file
- `BOOTSTRAP_PEERS`: Comma-separated list of bootstrap peers

## Development

To implement actual P2P functionality:

1. Add libp2p dependencies to go.mod
2. Replace mock implementations with real libp2p code
3. Implement proper peer connection management
4. Add protocol handlers for Blackhole-specific protocols
5. Integrate with the identity plugin for peer authentication