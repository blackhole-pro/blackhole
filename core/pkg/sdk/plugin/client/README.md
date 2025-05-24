# Plugin Mesh Client Library

This library makes it easy to create mesh-connected plugins for the Blackhole Framework.

## Features

- **Simple API**: Just a few lines to create a mesh-connected plugin
- **Automatic Socket Management**: Creates and cleans up Unix sockets
- **Built-in Health Checking**: gRPC health service included
- **Reflection Support**: Enable gRPC reflection for debugging
- **Lifecycle Callbacks**: Hook into start/stop/connect events
- **Graceful Shutdown**: Handles signals and drains connections
- **Logging**: Structured logging with zap
- **Environment Support**: Configure from environment variables

## Quick Start

### 1. Define Your Service

Create your protobuf definition:

```protobuf
syntax = "proto3";

service MyPlugin {
  rpc Initialize(InitRequest) returns (InitResponse);
  rpc DoWork(WorkRequest) returns (WorkResponse);
}
```

### 2. Implement Your Plugin

```go
package main

import (
    "context"
    
    "github.com/blackhole-prodev/blackhole/core/pkg/sdk/plugin/client"
    myv1 "myproject/proto/v1"
)

type MyPlugin struct {
    myv1.UnimplementedMyPluginServer
}

func (p *MyPlugin) DoWork(ctx context.Context, req *myv1.WorkRequest) (*myv1.WorkResponse, error) {
    // Your plugin logic here
    return &myv1.WorkResponse{Result: "done"}, nil
}

func main() {
    // Create plugin client
    pluginClient, err := client.New(client.DefaultConfig("my-plugin"))
    if err != nil {
        log.Fatal(err)
    }

    // Register your service
    myv1.RegisterMyPluginServer(pluginClient.GetGRPCServer(), &MyPlugin{})

    // Run the plugin
    if err := pluginClient.Run(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

### 3. Build and Run

```bash
go build -o my-plugin
./my-plugin
```

## Configuration

### Using Config Struct

```go
config := client.Config{
    Name:             "my-plugin",
    Version:          "1.0.0",
    Description:      "My awesome plugin",
    SocketPath:       "/tmp/my-plugin.sock",
    EnableReflection: true,
    EnableHealth:     true,
    GracefulTimeout:  10 * time.Second,
    Logger:           zap.NewProduction(),
}

pluginClient, err := client.New(config)
```

### Using Environment Variables

```go
// Reads from: PLUGIN_NAME, PLUGIN_VERSION, PLUGIN_SOCKET, etc.
pluginClient, err := client.NewFromEnv()
```

### Using Defaults

```go
// Creates sensible defaults for the given plugin name
config := client.DefaultConfig("my-plugin")
pluginClient, err := client.New(config)
```

## Lifecycle Callbacks

```go
config.OnStart = func() error {
    // Called when plugin starts
    // Initialize resources, connect to databases, etc.
    return nil
}

config.OnStop = func() error {
    // Called when plugin stops
    // Clean up resources, close connections, etc.
    return nil
}

config.OnConnect = func(peer string) {
    // Called when a client connects
    log.Printf("Client connected from %s", peer)
}
```

## Health Checking

The library automatically provides gRPC health checking:

```go
// Set service health status
pluginClient.SetHealthStatus("my-service", grpc_health_v1.HealthCheckResponse_SERVING)

// The empty string "" represents overall plugin health
pluginClient.SetHealthStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
```

Clients can check health using standard gRPC health checking:

```bash
grpcurl -unix -plaintext /tmp/my-plugin.sock grpc.health.v1.Health/Check
```

## Debugging

### Enable Reflection

With reflection enabled (default), you can explore the plugin's API:

```bash
# List services
grpcurl -unix -plaintext /tmp/my-plugin.sock list

# Describe service
grpcurl -unix -plaintext /tmp/my-plugin.sock describe MyPlugin

# Call methods
grpcurl -unix -plaintext -d '{"message": "test"}' /tmp/my-plugin.sock MyPlugin/DoWork
```

### Logging

The library includes structured logging:

```go
logger, _ := zap.NewDevelopment()
config.Logger = logger
```

All requests are automatically logged with timing information.

## Advanced Usage

### Manual Lifecycle Control

Instead of using `Run()`, you can control the lifecycle manually:

```go
// Start the plugin
if err := pluginClient.Start(ctx); err != nil {
    return err
}

// Do other work...

// Wait for shutdown signal
pluginClient.WaitForShutdown()

// Stop the plugin
if err := pluginClient.Stop(ctx); err != nil {
    return err
}
```

### Custom Interceptors

Access the gRPC server to add custom interceptors:

```go
server := pluginClient.GetGRPCServer()
// Note: Do this before calling Start()
```

## Integration with Plugin Manager

The plugin manager expects plugins to:

1. Listen on the socket path specified in `PLUGIN_SOCKET`
2. Implement their service-specific gRPC interface
3. Handle graceful shutdown on SIGTERM

This library handles all of these requirements automatically.

## Example

See the `example/echo/` directory for a complete working example of a mesh-connected plugin.

## Best Practices

1. **Use structured logging** - Pass a proper logger in config
2. **Handle context cancellation** - Respect context in your RPC methods
3. **Set health status** - Update health based on your plugin's state
4. **Clean shutdown** - Use OnStop to clean up resources
5. **Version your API** - Use `/v1/` in your proto package names
6. **Document capabilities** - List what your plugin can do

## Troubleshooting

### Permission Denied

Make sure the socket directory exists and is writable:

```bash
mkdir -p /tmp/blackhole/plugins
chmod 755 /tmp/blackhole/plugins
```

### Address Already in Use

The library automatically removes existing sockets, but if issues persist:

```bash
rm /tmp/blackhole/plugins/my-plugin.sock
```

### Connection Refused

Ensure the plugin is running and the socket path is correct:

```bash
ls -la /tmp/blackhole/plugins/
```

### Debugging Communication

Use grpcurl to test the plugin directly:

```bash
grpcurl -unix -plaintext /tmp/blackhole/plugins/my-plugin.sock list
```