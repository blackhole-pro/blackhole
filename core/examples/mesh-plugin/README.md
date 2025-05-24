# Mesh-Based Plugin Example

This example demonstrates how plugins communicate via the mesh network instead of stdin/stdout.

## Architecture Comparison

### Old: stdin/stdout Communication
```
Plugin Manager --[stdin]--> Plugin Process --[stdout]--> Plugin Manager
```

### New: Mesh Network Communication
```
Plugin Manager --[Mesh/gRPC]--> Plugin Service --[Mesh/gRPC]--> Plugin Manager
                                      |
                                 Unix Socket
                                      |
                              Other Services/Apps
```

## Key Differences

1. **Process Management Only**: The plugin manager only starts/stops processes
2. **Service Discovery**: Plugins register themselves on the mesh network
3. **gRPC Communication**: All communication uses type-safe gRPC interfaces
4. **Multi-Client Support**: Multiple clients can connect to the same plugin
5. **Location Transparency**: Plugins can run locally or remotely

## How It Works

### 1. Plugin Startup
```go
// Plugin manager starts the process
cmd := exec.Command("/path/to/plugin")
cmd.Env = []string{
    "PLUGIN_SOCKET=/tmp/plugin.sock",
    "PLUGIN_NAME=my-plugin",
}
cmd.Start()
```

### 2. Plugin Registration
```go
// Plugin starts gRPC server on socket
listener, _ := net.Listen("unix", os.Getenv("PLUGIN_SOCKET"))
server := grpc.NewServer()
myv1.RegisterMyPluginServer(server, &MyPlugin{})
server.Serve(listener)
```

### 3. Mesh Registration
```go
// Plugin manager registers with mesh
protocolRouter.RegisterService("plugin.my-plugin", Endpoint{
    Socket: "/tmp/plugin.sock",
    IsLocal: true,
})
```

### 4. Client Connection
```go
// Applications connect via mesh
conn, _ := mesh.Connect("plugin.my-plugin")
client := myv1.NewMyPluginClient(conn)
response, _ := client.DoSomething(ctx, request)
```

## Benefits

1. **Type Safety**: Full gRPC type safety instead of JSON over stdin
2. **Performance**: Direct socket communication, no serialization overhead
3. **Debugging**: Can use standard gRPC tools (grpcurl, etc.)
4. **Scaling**: Plugins can have multiple instances with load balancing
5. **Monitoring**: Built-in metrics and tracing via mesh

## Example Plugin

See the `echo-plugin/` directory for a complete example of a mesh-based plugin.