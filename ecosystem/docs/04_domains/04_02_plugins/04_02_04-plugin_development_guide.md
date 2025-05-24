# Plugin Development Guide

## Quick Start: Your First Plugin

### Step 1: Understand What You're Building

**Key Concept**: A plugin is a standalone gRPC service that runs as a separate process and communicates via the mesh network.

```
Your Plugin Process  <--[gRPC via Mesh]-->  Applications
     (Isolated)                              (Consumers)
```

### Step 2: Create Your Plugin Structure

```bash
my-awesome-plugin/
├── proto/
│   └── v1/
│       └── awesome.proto    # Your plugin's interface
├── internal/
│   ├── service/            # Core implementation
│   ├── state/              # State management
│   └── config/             # Configuration
├── cmd/
│   └── plugin/
│       └── main.go         # Entry point
├── go.mod                  # Module definition
├── Makefile                # Build automation
├── README.md               # Documentation
└── plugin.json             # Plugin manifest
```

### Step 3: Define Your Interface (Most Important!)

**Remember**: Your proto file is your complete contract. No hidden base interfaces!

```protobuf
syntax = "proto3";

package example.plugin.awesome.v1;

option go_package = "github.com/example/awesome-plugin/proto/v1;awesomev1";

// AwesomePlugin does something awesome
service AwesomePlugin {
    // Define EXACTLY what your plugin needs
    rpc Initialize(InitializeRequest) returns (InitializeResponse);
    rpc DoAwesomeThing(AwesomeRequest) returns (AwesomeResponse);
    rpc StreamAwesomeEvents(StreamRequest) returns (stream AwesomeEvent);
}

// Your domain-specific initialization
message InitializeRequest {
    string workspace_dir = 1;     // What YOU need
    int32 worker_threads = 2;     // Not generic config!
    bool enable_turbo_mode = 3;   // Domain-specific
}
```

### Step 4: Implement Your Service

```go
package main

import (
    "context"
    "net"
    "log"
    
    "google.golang.org/grpc"
    pb "github.com/example/awesome-plugin/proto/v1"
)

type AwesomePlugin struct {
    pb.UnimplementedAwesomePluginServer
    // Your plugin state
    config Config
    // No base plugin to embed!
}

func (p *AwesomePlugin) Initialize(ctx context.Context, req *pb.InitializeRequest) (*pb.InitializeResponse, error) {
    // YOUR initialization logic
    // Not implementing some base interface!
    p.config.WorkspaceDir = req.WorkspaceDir
    p.config.WorkerThreads = req.WorkerThreads
    
    return &pb.InitializeResponse{
        Success: true,
        Message: "Awesome plugin initialized!",
    }, nil
}

func main() {
    // Start as a gRPC service
    listener, err := net.Listen("unix", "/tmp/awesome-plugin.sock")
    if err != nil {
        log.Fatal(err)
    }
    
    server := grpc.NewServer()
    pb.RegisterAwesomePluginServer(server, &AwesomePlugin{})
    
    log.Println("Awesome plugin listening on /tmp/awesome-plugin.sock")
    server.Serve(listener)
}
```

## Key Differences from Traditional Plugin Systems

### ❌ What NOT to Do

```go
// DON'T look for base interfaces
type BasePlugin interface {
    Initialize(map[string]interface{})
    Start() error
    Stop() error
}

// DON'T implement framework interfaces
func (p *MyPlugin) FrameworkInitialize(config Config) error {
    // The framework doesn't call this!
}

// DON'T expect the framework to manage your lifecycle
func (p *MyPlugin) OnFrameworkReady() {
    // This doesn't exist!
}
```

### ✅ What TO Do

```go
// DO create your own specific interface
type AwesomePlugin struct {
    // Your fields, your way
}

// DO define exactly what you need
func (p *AwesomePlugin) DoAwesomeThing(ctx context.Context, req *pb.AwesomeRequest) (*pb.AwesomeResponse, error) {
    // Your logic, your types
}

// DO run as a standalone service
func main() {
    // You control your lifecycle
    startGRPCServer()
}
```

## Plugin Lifecycle

### 1. Process Startup (Framework's Job)

```bash
# Framework starts your plugin process
$ /usr/local/lib/blackhole/plugins/awesome-plugin
```

### 2. Service Registration (Your Job)

```go
// Your plugin registers on the mesh
listener, _ := net.Listen("unix", socketPath)
server := grpc.NewServer()
pb.RegisterAwesomePluginServer(server, plugin)
server.Serve(listener)
```

### 3. Discovery (Mesh's Job)

```yaml
# Mesh makes your plugin discoverable
services:
  plugin.awesome:
    endpoint: unix:///tmp/awesome-plugin.sock
    status: healthy
```

### 4. Usage (Application's Job)

```go
// Applications connect and use your specific interface
conn, _ := mesh.Connect("plugin.awesome")
client := awesomev1.NewAwesomePluginClient(conn)
response, _ := client.DoAwesomeThing(ctx, request)
```

## Best Practices

### 1. Design Your Interface First

Before writing any code, design your protobuf interface:
- What operations does your plugin provide?
- What data types are specific to your domain?
- What events might you need to stream?

### 2. Keep Your Domain Pure

```protobuf
// ✅ Good: Domain-specific types
message ImageProcessingRequest {
    bytes image_data = 1;
    enum Format {
        JPEG = 0;
        PNG = 1;
        WEBP = 2;
    }
    Format output_format = 2;
    int32 quality = 3;
}

// ❌ Bad: Generic types
message ProcessRequest {
    bytes data = 1;
    map<string, string> options = 2;
}
```

### 3. Handle Your Own State

```go
type AwesomePlugin struct {
    // Your plugin manages its own state
    db         *sql.DB
    cache      *Cache
    workers    []*Worker
    
    // No framework state management!
}
```

### 4. Communicate Via Mesh

Need another plugin's functionality?

```go
// Connect through mesh, not direct calls
cryptoConn, _ := mesh.Connect("plugin.crypto")
cryptoClient := cryptov1.NewCryptoPluginClient(cryptoConn)

// Use their specific interface
hash, _ := cryptoClient.Hash(ctx, &cryptov1.HashRequest{
    Data: myData,
    Algorithm: "sha256",
})
```

## Common Patterns

### Initialization Pattern

```go
func (p *Plugin) Initialize(ctx context.Context, req *pb.InitRequest) (*pb.InitResponse, error) {
    // 1. Validate configuration
    if err := p.validateConfig(req.Config); err != nil {
        return nil, err
    }
    
    // 2. Set up resources
    if err := p.setupResources(); err != nil {
        return nil, err
    }
    
    // 3. Connect to dependencies
    if err := p.connectDependencies(); err != nil {
        return nil, err
    }
    
    return &pb.InitResponse{
        Success: true,
        Capabilities: p.getCapabilities(),
    }, nil
}
```

### Health Check Pattern

```go
func (p *Plugin) HealthCheck(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
    health := &pb.HealthResponse{
        Status: "healthy",
    }
    
    // Check your specific health indicators
    if !p.isDatabaseConnected() {
        health.Status = "unhealthy"
        health.Issues = append(health.Issues, "database disconnected")
    }
    
    if p.getQueueDepth() > p.maxQueueDepth {
        health.Status = "degraded"
        health.Issues = append(health.Issues, "queue backlog")
    }
    
    return health, nil
}
```

### Streaming Pattern

```go
func (p *Plugin) StreamEvents(req *pb.StreamRequest, stream pb.Plugin_StreamEventsServer) error {
    // Set up event subscription
    events := p.subscribeToEvents(req.Filter)
    defer p.unsubscribe(events)
    
    // Stream events to client
    for event := range events {
        if err := stream.Send(event); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Testing Your Plugin

### 1. Unit Tests

```go
func TestAwesomePlugin_DoAwesomeThing(t *testing.T) {
    plugin := &AwesomePlugin{}
    plugin.Initialize(context.Background(), &pb.InitRequest{
        WorkspaceDir: "/tmp/test",
    })
    
    resp, err := plugin.DoAwesomeThing(context.Background(), &pb.AwesomeRequest{
        Input: "test",
    })
    
    assert.NoError(t, err)
    assert.True(t, resp.Success)
}
```

### 2. Integration Tests

```go
func TestPluginIntegration(t *testing.T) {
    // Start your plugin
    cmd := exec.Command("./awesome-plugin")
    cmd.Start()
    defer cmd.Process.Kill()
    
    // Connect as a client
    conn, _ := grpc.Dial("unix:///tmp/awesome-plugin.sock")
    client := pb.NewAwesomePluginClient(conn)
    
    // Test via gRPC
    resp, err := client.DoAwesomeThing(ctx, request)
    assert.NoError(t, err)
}
```

## Debugging Tips

### 1. Your Plugin Doesn't Start?
- Check process logs: `journalctl -u blackhole-plugin-awesome`
- Verify socket path exists and has permissions
- Ensure no other process is using the socket

### 2. Can't Connect to Plugin?
- Verify plugin is registered with mesh: `blackhole mesh list`
- Check mesh connectivity: `blackhole mesh ping plugin.awesome`
- Ensure gRPC service is registered correctly

### 3. Methods Not Found?
- Regenerate protobuf code: `make proto`
- Ensure server implements all service methods
- Check for version mismatches

## Advanced Topics

### Multi-Instance Plugins

```go
// Support multiple instances with unique endpoints
socketPath := fmt.Sprintf("/tmp/awesome-plugin-%s.sock", instanceID)
```

### Plugin Discovery

```go
// Register additional metadata
mesh.Register("plugin.awesome", endpoint, map[string]string{
    "version": "1.0.0",
    "capabilities": "image-processing,thumbnail-generation",
})
```

### Hot Configuration Reload

```go
func (p *Plugin) ReloadConfig(ctx context.Context, req *pb.ReloadRequest) (*pb.ReloadResponse, error) {
    newConfig, err := loadConfig(req.ConfigPath)
    if err != nil {
        return nil, err
    }
    
    // Apply without restart
    p.applyConfig(newConfig)
    
    return &pb.ReloadResponse{Success: true}, nil
}
```

## Remember

1. **You own your interface** - Design it for your domain
2. **You run independently** - No framework coupling
3. **You communicate via mesh** - Location transparent
4. **You manage your state** - Full control

Happy plugin development! 🚀