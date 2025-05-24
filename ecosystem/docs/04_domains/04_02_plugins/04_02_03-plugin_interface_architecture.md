# Plugin Interface Architecture

## Overview

This document explains a fundamental architectural decision in Blackhole Foundation: **why plugins don't share a common base interface** and how this enables maximum flexibility and type safety.

## The Journey: From Base Interface to Domain-Specific Interfaces

### Initial Assumption: Common Base Interface

When first designing the plugin system, a natural assumption was that all plugins would need common methods:

```protobuf
// Initial thinking: All plugins need these
service BasePlugin {
    rpc Initialize(InitializeRequest) returns (InitializeResponse);
    rpc Start(StartRequest) returns (StartResponse);
    rpc Stop(StopRequest) returns (StopResponse);
    rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}
```

This seemed logical because:
- The framework needs to manage plugin lifecycle
- Consistency across plugins would be good
- Type safety for framework-plugin interaction

### The Problem: Different Domains, Different Needs

However, when implementing actual plugins, we discovered fundamental incompatibilities:

#### Storage Plugin Initialization
```protobuf
message InitializeRequest {
    string data_dir = 1;        // Where to store files
    int64 max_size_gb = 2;      // Storage quota
    string storage_backend = 3;  // "filesystem", "s3", "ipfs"
    bool enable_encryption = 4;  // Encryption at rest
}
```

#### Node Plugin Initialization
```protobuf
message InitializeRequest {
    string node_id = 1;                  // Unique identifier
    repeated string bootstrap_peers = 2;  // P2P network peers
    int32 p2p_port = 3;                  // Listen port
    string discovery_method = 4;         // "mdns", "dht", "manual"
}
```

#### Identity Plugin Initialization
```protobuf
message InitializeRequest {
    string database_url = 1;      // User database
    int32 session_timeout = 2;    // Session duration
    repeated string oauth_providers = 3;  // "google", "github"
    bool enable_mfa = 4;          // Multi-factor auth
}
```

**These are completely different!** Forcing a common structure would require:
- Generic `map<string, any>` configuration (loss of type safety)
- Massive optional field lists (confusing API)
- Lowest common denominator design (no optimization)

## The Architectural Insight

The breakthrough came from understanding how plugins actually interact with the framework:

### Traditional Plugin Architecture
```
Framework --[Direct Method Calls]--> Plugin
         Initialize()
         Start()
         Stop()
```

### Blackhole's Architecture
```
Framework --[Process Management]--> Plugin Process
                                          |
                                          v
                                    [Mesh Network]
                                          ^
                                          |
Application --[gRPC via Mesh]-----> Plugin Service
```

**Key Insight**: The framework never calls plugin methods directly!

## How It Actually Works

### 1. Framework's Role (Process Management)

```go
// The framework manages plugins at the OS process level
type PluginManager struct {
    processes map[string]*os.Process
}

func (pm *PluginManager) StartPlugin(name string) error {
    // Start the plugin process
    cmd := exec.Command("plugins/storage-plugin")
    cmd.Start()
    
    // Register with mesh
    mesh.RegisterService("plugin.storage", "unix:///tmp/storage.sock")
    
    // Monitor process health
    go pm.monitorProcess(cmd.Process)
    
    return nil
}
```

### 2. Plugin's Role (Service Provider)

```go
// Each plugin is a standalone gRPC service
func main() {
    // Storage plugin main
    server := grpc.NewServer()
    storagev1.RegisterStoragePluginServer(server, &StoragePlugin{})
    
    // Listen on mesh endpoint
    listener, _ := net.Listen("unix", "/tmp/storage.sock")
    server.Serve(listener)
}
```

### 3. Application's Role (Service Consumer)

```go
// Applications connect through mesh
storageConn, _ := mesh.Connect("plugin.storage")
storageClient := storagev1.NewStoragePluginClient(storageConn)

// Use storage-specific interface
resp, _ := storageClient.Store(ctx, &storagev1.StoreRequest{
    Key:      "document.pdf",
    Data:     fileData,
    Encrypt:  true,
    Compress: true,
})
```

## Benefits of Domain-Specific Interfaces

### 1. Type Safety

```protobuf
// Storage gets storage-specific types
message StorageConfig {
    enum Backend {
        FILESYSTEM = 0;
        S3 = 1;
        IPFS = 2;
    }
    Backend backend = 1;
    // Not a generic string!
}
```

### 2. Domain Optimization

```protobuf
// Node plugin has P2P-specific streaming
service NodePlugin {
    rpc StreamPeerEvents(StreamRequest) returns (stream PeerEvent);
    rpc StreamNetworkMetrics(MetricsRequest) returns (stream Metrics);
}

// Storage plugin has storage-specific streaming
service StoragePlugin {
    rpc StreamStore(stream DataChunk) returns (StoreResponse);
    rpc StreamRetrieve(RetrieveRequest) returns (stream DataChunk);
}
```

### 3. Independent Evolution

Each plugin can evolve independently:
- Node plugin adds WebRTC support
- Storage plugin adds deduplication
- Identity plugin adds biometric auth
- No coordination needed!

### 4. Clear Contracts

Each plugin's protobuf is its complete contract:
```
node/proto/v1/node.proto      # Complete node interface
storage/proto/v1/storage.proto # Complete storage interface
identity/proto/v1/identity.proto # Complete identity interface
```

No hidden base requirements or inheritance confusion.

## Framework-Plugin Interaction

The framework interacts with plugins at three levels:

### 1. Process Level
- Start/stop plugin processes
- Monitor process health
- Enforce resource limits
- Handle crashes/restarts

### 2. Mesh Level
- Register plugin endpoints
- Enable service discovery
- Route requests
- Load balancing

### 3. Metrics Level
- Collect resource usage
- Monitor performance
- Track errors
- Generate alerts

**Note**: None of these require calling plugin methods!

## Design Principles

### 1. Plugins are Services
- Each plugin is a complete, standalone service
- Communicates via well-defined gRPC interface
- No inheritance or base classes

### 2. Domain-Driven Design
- Each plugin interface matches its domain
- No forced abstractions
- Optimize for specific use cases

### 3. Process Isolation
- Plugins run in separate processes
- Framework manages processes, not objects
- Communication via mesh network

### 4. Type Safety First
- Strongly typed interfaces
- No generic configurations
- Compile-time validation

## Common Patterns

While plugins don't share interfaces, they may follow similar patterns:

### Initialization Pattern
```protobuf
// Each plugin defines its own initialization
message InitializeRequest {
    // Domain-specific configuration
    DomainConfig config = 1;
    
    // Common metadata (optional)
    string instance_id = 2;
    map<string, string> labels = 3;
}
```

### Health Check Pattern
```protobuf
// Each plugin defines what "healthy" means
message HealthCheckResponse {
    bool healthy = 1;
    string status = 2;  // "healthy", "degraded", "unhealthy"
    
    // Domain-specific health details
    oneof details {
        NetworkHealth network_health = 3;
        StorageHealth storage_health = 4;
        DatabaseHealth database_health = 5;
    }
}
```

### Streaming Pattern
```protobuf
// Each plugin defines its own streaming needs
service Plugin {
    // Event streams (node plugin)
    rpc StreamEvents(StreamRequest) returns (stream Event);
    
    // Data streams (storage plugin)
    rpc StreamData(stream DataChunk) returns (Response);
    
    // Metric streams (analytics plugin)
    rpc StreamMetrics(MetricRequest) returns (stream Metric);
}
```

## Migration Guide

If you're coming from traditional plugin architectures:

### Don't Do This
```go
// ❌ Don't expect base interfaces
type Plugin interface {
    Initialize(config map[string]interface{})
    Start() error
    Stop() error
}
```

### Do This Instead
```go
// ✅ Connect to specific plugin services
nodeClient := nodev1.NewNodePluginClient(meshConn)
resp, err := nodeClient.ListPeers(ctx, &nodev1.ListPeersRequest{
    StatusFilter: "connected",
})
```

## FAQ

### Q: How does the framework manage plugins without a common interface?

A: The framework manages plugins at the **process level**, not the API level. It starts/stops processes, monitors health via process status, and enforces resource limits via OS mechanisms.

### Q: How do plugins register with the framework?

A: Plugins register their **mesh endpoints** when they start. The framework just needs to know where to find them on the mesh, not what methods they have.

### Q: What if I need common functionality across plugins?

A: Create a **shared plugin** (like crypto-plugin) that other plugins can use via mesh. Don't force it into a base interface.

### Q: How do I know what methods a plugin supports?

A: Check the plugin's **protobuf definition**. It's the complete contract. The proto file in `plugin-name/proto/v1/` defines everything the plugin can do.

## Conclusion

By avoiding artificial base interfaces and embracing domain-specific designs, Blackhole's plugin architecture achieves:

- **Maximum flexibility** - Each plugin is perfect for its domain
- **Type safety** - Strongly typed, domain-specific interfaces
- **Independent evolution** - Plugins can innovate without coordination
- **Clear contracts** - Proto files are the complete truth

This architecture is more complex than traditional plugin systems, but it enables the true vision of Blackhole: a framework where plugins are first-class services that can run anywhere, scale independently, and evolve without limits.