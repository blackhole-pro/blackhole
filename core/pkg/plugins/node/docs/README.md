# Node Plugin

The Node Plugin provides P2P networking and distributed communication capabilities for the Blackhole platform.

## Features

- **P2P Networking**: Full libp2p-based peer-to-peer networking
- **Peer Discovery**: Automatic peer discovery via mDNS and DHT
- **Message Routing**: Efficient message routing between peers
- **NAT Traversal**: Automatic NAT traversal and relay support
- **Distributed Hash Table**: DHT for decentralized peer discovery
- **PubSub Messaging**: Topic-based publish/subscribe messaging
- **Peer Reputation**: Track and manage peer reputation scores

## Installation

### From Package

```bash
# Install from local package
blackhole plugin install node-v1.0.0.plugin

# Install from URL
blackhole plugin install https://plugins.blackhole.io/node-v1.0.0.plugin

# Install from marketplace
blackhole plugin install node@1.0.0
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/blackhole/core
cd core/pkg/plugins/node

# Build for local platform
make build-local

# Build for all platforms
make build

# Create plugin package
make package
```

## Configuration

The node plugin is configured via the Blackhole configuration file:

```yaml
plugins:
  node:
    p2p:
      listen_addresses:
        - "/ip4/0.0.0.0/tcp/0"
        - "/ip6/::/tcp/0"
      bootstrap_peers:
        - "/ip4/192.168.1.100/tcp/4001/p2p/QmPeer1"
        - "/ip4/192.168.1.101/tcp/4001/p2p/QmPeer2"
      max_connections: 100
      enable_relay: true
      enable_nat: true
    
    discovery:
      mdns:
        enabled: true
        interval: "1m"
      dht:
        enabled: true
        mode: "auto"  # client, server, or auto
    
    storage:
      path: "~/.blackhole/node/data"
      max_size: "1GB"
```

## Usage

### Basic Example

```go
package main

import (
    "context"
    "log"
    
    nodepb "github.com/blackhole/core/pkg/plugins/node/proto/v1"
    "google.golang.org/grpc"
)

func main() {
    // Connect to the node plugin
    conn, err := grpc.Dial("unix:///tmp/blackhole/plugins/node.sock", 
        grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    // Create client
    client := nodepb.NewNodeServiceClient(conn)
    
    // Get node info
    info, err := client.GetInfo(context.Background(), 
        &nodepb.GetInfoRequest{})
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Node ID: %s", info.Id)
    log.Printf("Addresses: %v", info.Addresses)
    log.Printf("Protocols: %v", info.Protocols)
}
```

### Publishing Messages

```go
// Publish a message to a topic
_, err = client.Publish(context.Background(), &nodepb.PublishRequest{
    Topic: "my-topic",
    Data:  []byte("Hello, distributed world!"),
})
```

### Subscribing to Topics

```go
// Subscribe to a topic
stream, err := client.Subscribe(context.Background(), 
    &nodepb.SubscribeRequest{
        Topic: "my-topic",
    })
if err != nil {
    log.Fatal(err)
}

// Handle incoming messages
for {
    msg, err := stream.Recv()
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Received from %s: %s", 
        msg.PeerId, string(msg.Data))
}
```

### Peer Management

```go
// List connected peers
peers, err := client.ListPeers(context.Background(), 
    &nodepb.ListPeersRequest{})

// Connect to a specific peer
_, err = client.Connect(context.Background(), 
    &nodepb.ConnectRequest{
        PeerId:    "QmPeerID...",
        Addresses: []string{"/ip4/1.2.3.4/tcp/4001"},
    })

// Disconnect from a peer
_, err = client.Disconnect(context.Background(), 
    &nodepb.DisconnectRequest{
        PeerId: "QmPeerID...",
    })
```

## API Reference

See [API.md](API.md) for the complete API documentation.

## Development

### Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (protoc)
- Make

### Building

```bash
# Generate protobuf code
make proto

# Build for local development
make dev

# Run tests
make test

# Create release package
make release
```

### Project Structure

```
node/
├── main.go           # Plugin entry point
├── plugin.yaml       # Plugin manifest
├── Makefile         # Build configuration
├── proto/           # Protocol buffer definitions
│   └── v1/
│       └── node.proto
├── internal/        # Internal implementation
│   ├── service.go   # Service implementation
│   ├── p2p.go       # P2P networking
│   └── discovery.go # Peer discovery
├── docs/            # Documentation
│   ├── README.md    # This file
│   └── API.md       # API reference
└── examples/        # Usage examples
```

## Troubleshooting

### Common Issues

1. **Cannot connect to peers**
   - Check firewall settings
   - Ensure NAT traversal is enabled
   - Verify bootstrap peers are reachable

2. **High memory usage**
   - Reduce max_connections
   - Disable DHT server mode
   - Check for message loops

3. **Slow peer discovery**
   - Enable mDNS for local discovery
   - Add more bootstrap peers
   - Check network connectivity

### Debug Mode

Enable debug logging:

```yaml
plugins:
  node:
    logging:
      level: debug
      output: stdout
```

### Performance Tuning

For high-performance deployments:

```yaml
plugins:
  node:
    p2p:
      max_connections: 500
      connection_manager:
        high_water: 400
        low_water: 200
        grace_period: "30s"
    
    transport:
      tcp:
        listen_backlog: 1024
        no_delay: true
      
      quic:
        enabled: true
        max_streams: 100
```

## Security Considerations

- All peer connections are authenticated using peer IDs
- Support for encrypted transports (TLS, Noise)
- Message signing and verification
- Peer reputation tracking
- Connection limits and rate limiting

## License

MIT License - see LICENSE file for details.

## Support

- Documentation: https://docs.blackhole.io/plugins/node
- Issues: https://github.com/blackhole/core/issues
- Community: https://discord.gg/blackhole