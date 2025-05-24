# Using Plugins in Your Application

This guide shows how to use Blackhole plugins in your application via the mesh network.

## Overview

Plugins in Blackhole are standalone services that communicate via gRPC over the mesh network. Your application connects to plugins just like any other service.

## Connecting to Plugins

### 1. Direct Connection (Development)

For development and testing, you can connect directly to a plugin's socket:

```go
import (
    "google.golang.org/grpc"
    nodev1 "github.com/blackhole-prodev/blackhole/core/pkg/plugins/node/proto/v1"
)

// Connect directly to plugin socket
conn, err := grpc.Dial("unix:///tmp/blackhole/plugins/node.sock", 
    grpc.WithInsecure())
if err != nil {
    return err
}
defer conn.Close()

// Create client
nodeClient := nodev1.NewNodePluginClient(conn)

// Use the plugin
info, err := nodeClient.GetInfo(ctx, &nodev1.GetInfoRequest{})
```

### 2. Via Mesh Network (Production)

In production, connect through the mesh network for service discovery and load balancing:

```go
import (
    "github.com/blackhole-prodev/blackhole/core/internal/framework/mesh"
    nodev1 "github.com/blackhole-prodev/blackhole/core/pkg/plugins/node/proto/v1"
)

// Connect via mesh
conn, err := mesh.Connect(ctx, "plugin.node")
if err != nil {
    return err
}
defer conn.Close()

// Create client
nodeClient := nodev1.NewNodePluginClient(conn)
```

### 3. Via Plugin Manager (Managed)

If your application uses the plugin manager:

```go
// Get connection from plugin manager
conn, err := pluginManager.GetPluginConnection("node")
if err != nil {
    return err
}

// Create client
nodeClient := nodev1.NewNodePluginClient(conn)
```

## Example: Content Distribution Application

Here's a complete example showing how to orchestrate multiple plugins:

```go
package main

import (
    "context"
    "fmt"
    
    "google.golang.org/grpc"
    
    nodev1 "github.com/blackhole-prodev/blackhole/core/pkg/plugins/node/proto/v1"
    storagev1 "github.com/blackhole-prodev/blackhole/core/pkg/plugins/storage/proto/v1"
    identityv1 "github.com/blackhole-prodev/blackhole/core/pkg/plugins/identity/proto/v1"
)

type ContentApp struct {
    nodeClient     nodev1.NodePluginClient
    storageClient  storagev1.StoragePluginClient
    identityClient identityv1.IdentityPluginClient
}

func NewContentApp(ctx context.Context) (*ContentApp, error) {
    // Connect to plugins
    nodeConn, err := grpc.Dial("unix:///tmp/blackhole/plugins/node.sock", 
        grpc.WithInsecure())
    if err != nil {
        return nil, err
    }
    
    storageConn, err := grpc.Dial("unix:///tmp/blackhole/plugins/storage.sock", 
        grpc.WithInsecure())
    if err != nil {
        return nil, err
    }
    
    identityConn, err := grpc.Dial("unix:///tmp/blackhole/plugins/identity.sock", 
        grpc.WithInsecure())
    if err != nil {
        return nil, err
    }
    
    return &ContentApp{
        nodeClient:     nodev1.NewNodePluginClient(nodeConn),
        storageClient:  storagev1.NewStoragePluginClient(storageConn),
        identityClient: identityv1.NewIdentityPluginClient(identityConn),
    }, nil
}

func (app *ContentApp) ShareFile(ctx context.Context, userID, filePath string, recipients []string) error {
    // 1. Verify user identity
    authResp, err := app.identityClient.Authenticate(ctx, &identityv1.AuthenticateRequest{
        UserId: userID,
    })
    if err != nil {
        return fmt.Errorf("authentication failed: %w", err)
    }
    if !authResp.Authenticated {
        return fmt.Errorf("user not authenticated")
    }
    
    // 2. Store the file
    storeResp, err := app.storageClient.Store(ctx, &storagev1.StoreRequest{
        Key:  filePath,
        Data: readFile(filePath),
    })
    if err != nil {
        return fmt.Errorf("storage failed: %w", err)
    }
    
    // 3. Share via P2P network
    for _, recipient := range recipients {
        // Find recipient's peer ID
        peerResp, err := app.nodeClient.ResolvePeer(ctx, &nodev1.ResolvePeerRequest{
            Identity: recipient,
        })
        if err != nil {
            continue // Skip if can't resolve
        }
        
        // Send notification
        _, err = app.nodeClient.SendMessage(ctx, &nodev1.SendMessageRequest{
            PeerId: peerResp.PeerId,
            Message: &nodev1.Message{
                Type: "file_share",
                Data: map[string]string{
                    "file_id": storeResp.FileId,
                    "sender":  userID,
                },
            },
        })
    }
    
    return nil
}
```

## Plugin Communication Patterns

### 1. Request-Response

Standard RPC calls:

```go
// Simple request-response
resp, err := nodeClient.ListPeers(ctx, &nodev1.ListPeersRequest{
    StatusFilter: "connected",
    Limit:        10,
})
```

### 2. Streaming

For real-time data:

```go
// Subscribe to peer events
stream, err := nodeClient.StreamPeerEvents(ctx, &nodev1.StreamPeerEventsRequest{
    EventTypes: []string{"connected", "disconnected"},
})

for {
    event, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    
    // Handle event
    fmt.Printf("Peer %s: %s\n", event.PeerId, event.Type)
}
```

### 3. Bidirectional Streaming

For interactive communication:

```go
// Start chat session
stream, err := chatClient.Chat(ctx)

// Send messages
go func() {
    for msg := range outgoing {
        stream.Send(&ChatMessage{Text: msg})
    }
    stream.CloseSend()
}()

// Receive messages
for {
    msg, err := stream.Recv()
    if err == io.EOF {
        break
    }
    fmt.Printf("Received: %s\n", msg.Text)
}
```

## Error Handling

Always handle plugin-specific errors:

```go
resp, err := nodeClient.ConnectPeer(ctx, req)
if err != nil {
    // Check if it's a gRPC status error
    if st, ok := status.FromError(err); ok {
        switch st.Code() {
        case codes.NotFound:
            // Peer not found
        case codes.AlreadyExists:
            // Already connected
        case codes.ResourceExhausted:
            // Too many connections
        default:
            // Other error
        }
    }
    return err
}
```

## Health Checking

Monitor plugin health:

```go
import "google.golang.org/grpc/health/grpc_health_v1"

// Create health client
healthClient := grpc_health_v1.NewHealthClient(conn)

// Check overall health
resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
    // Plugin is not healthy
}

// Watch health changes
stream, err := healthClient.Watch(ctx, &grpc_health_v1.HealthCheckRequest{})
for {
    resp, err := stream.Recv()
    if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
        // Handle unhealthy plugin
    }
}
```

## Best Practices

### 1. Connection Management

```go
// Create a connection pool
type PluginPool struct {
    mu    sync.RWMutex
    conns map[string]*grpc.ClientConn
}

func (p *PluginPool) GetConnection(plugin string) (*grpc.ClientConn, error) {
    p.mu.RLock()
    if conn, ok := p.conns[plugin]; ok {
        p.mu.RUnlock()
        return conn, nil
    }
    p.mu.RUnlock()
    
    // Create new connection
    p.mu.Lock()
    defer p.mu.Unlock()
    
    // Double-check after acquiring write lock
    if conn, ok := p.conns[plugin]; ok {
        return conn, nil
    }
    
    conn, err := grpc.Dial(fmt.Sprintf("unix:///tmp/blackhole/plugins/%s.sock", plugin),
        grpc.WithInsecure(),
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            Time:    30 * time.Second,
            Timeout: 10 * time.Second,
        }),
    )
    if err != nil {
        return nil, err
    }
    
    p.conns[plugin] = conn
    return conn, nil
}
```

### 2. Retry Logic

```go
func callWithRetry(ctx context.Context, fn func() error) error {
    backoff := 100 * time.Millisecond
    maxRetries := 3
    
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        
        // Check if retryable
        if st, ok := status.FromError(err); ok {
            if st.Code() == codes.Unavailable {
                time.Sleep(backoff)
                backoff *= 2
                continue
            }
        }
        
        return err
    }
    
    return fmt.Errorf("max retries exceeded")
}
```

### 3. Context Propagation

Always pass context for cancellation and tracing:

```go
func (app *App) ProcessRequest(ctx context.Context) error {
    // Create child context with timeout
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Pass context to all plugin calls
    if err := app.processWithPlugins(ctx); err != nil {
        return err
    }
    
    return nil
}
```

## Debugging

### Using grpcurl

Test plugins directly:

```bash
# List services
grpcurl -unix -plaintext /tmp/blackhole/plugins/node.sock list

# Describe service
grpcurl -unix -plaintext /tmp/blackhole/plugins/node.sock describe NodePlugin

# Call method
grpcurl -unix -plaintext /tmp/blackhole/plugins/node.sock NodePlugin/GetInfo

# With data
grpcurl -unix -plaintext -d '{"status_filter": "connected"}' \
    /tmp/blackhole/plugins/node.sock NodePlugin/ListPeers
```

### Logging

Enable gRPC logging:

```go
import "google.golang.org/grpc/grpclog"

// Enable verbose logging
grpclog.SetLoggerV2(grpclog.NewLoggerV2(
    os.Stdout, os.Stdout, os.Stderr))
```

## Summary

1. **Plugins are services** - Connect via gRPC like any service
2. **Use the mesh** - For production service discovery
3. **Handle errors** - Check gRPC status codes
4. **Monitor health** - Use health checking
5. **Manage connections** - Pool and reuse connections
6. **Pass context** - For cancellation and tracing

The key insight is that plugins are just specialized services on the mesh network. Your application orchestrates them to build functionality, but doesn't need to know they're plugins - they're just services with well-defined interfaces.