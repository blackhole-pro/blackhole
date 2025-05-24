# Protocol-Level Routing Usage Guide

## Overview

The Blackhole Service Mesh Router now implements **protocol-level routing**, which means it can route ANY gRPC method without explicit registration or code changes. This is the same approach used by industry-standard service meshes like Istio, Linkerd, and Envoy.

## Key Benefits

1. **Method Agnostic**: Any gRPC method works automatically without router code changes
2. **Zero Maintenance**: Add new methods to services without touching router code  
3. **Industry Standard**: Uses standard gRPC HTTP/2 paths for routing
4. **Service-Only Registration**: Register services once, not individual methods
5. **Middleware Support**: All middleware (logging, circuit breaker, etc.) works automatically

## How It Works

When you make a gRPC call like:
```go
response, err := client.Store(ctx, &storagepb.StoreRequest{...})
```

gRPC automatically converts it to an HTTP/2 request:
```http
POST /storage.v1.StorageService/Store HTTP/2
Content-Type: application/grpc
[protobuf binary data]
```

The router parses the path `/storage.v1.StorageService/Store` to extract:
- **Service**: `storage.v1.StorageService`
- **Method**: `Store`

Then routes to the correct service using service discovery.

## Usage Example

### 1. Register Services (One-Time Setup)

```go
// Register services by their protocol names (from .proto files)
err := router.RegisterService("storage.v1.StorageService", types.Endpoint{
    Socket:  "/tmp/blackhole/storage.sock",  // Unix socket for local services
    IsLocal: true,
})

err = router.RegisterService("identity.auth.v1.AuthService", types.Endpoint{
    Socket:  "/tmp/blackhole/identity.sock",
    IsLocal: true,
})
```

### 2. Route Any gRPC Method

```go
// Example: Storage service methods
ctx := context.Background()

// Serialize your protobuf request to bytes
reqBytes, _ := proto.Marshal(&storagepb.StoreRequest{
    Data: []byte("Hello, World!"),
})

// Route ANY method using the gRPC path convention
respBytes, err := router.RouteGRPC(ctx, "/storage.v1.StorageService/Store", reqBytes)

// Other methods work automatically:
respBytes, err = router.RouteGRPC(ctx, "/storage.v1.StorageService/Retrieve", reqBytes)
respBytes, err = router.RouteGRPC(ctx, "/storage.v1.StorageService/ListContent", reqBytes)
respBytes, err = router.RouteGRPC(ctx, "/storage.v1.StorageService/DeleteContent", reqBytes)

// Identity service methods:
respBytes, err = router.RouteGRPC(ctx, "/identity.auth.v1.AuthService/ValidateToken", reqBytes)
respBytes, err = router.RouteGRPC(ctx, "/identity.auth.v1.AuthService/GenerateChallenge", reqBytes)
```

### 3. New Methods Work Automatically

When developers add new methods to services, they work instantly:

```protobuf
// Add this to storage/v1/service.proto
service StorageService {
    // Existing methods...
    rpc Store(StoreRequest) returns (StoreResponse);
    rpc Retrieve(RetrieveRequest) returns (RetrieveResponse);
    
    // NEW METHOD - no router changes needed!
    rpc CompressContent(CompressRequest) returns (CompressResponse);
}
```

```go
// This works immediately without any router code changes:
respBytes, err := router.RouteGRPC(ctx, "/storage.v1.StorageService/CompressContent", reqBytes)
```

## Path Convention

gRPC paths follow the convention: `/{package}.{version}.{Service}/{Method}`

Examples:
- `/storage.v1.StorageService/Store`
- `/identity.auth.v1.AuthService/ValidateToken`  
- `/ledger.v1.LedgerService/SubmitTransaction`
- `/social.v1.SocialService/PostMessage`

## Service Registration

Services are registered by their **protocol name** (from .proto files), not their implementation name:

```yaml
# From storage/v1/service.proto
package storage.v1;
service StorageService { ... }

# Register as: "storage.v1.StorageService"
```

```yaml  
# From identity/auth/v1/auth.proto
package identity.auth.v1;
service AuthService { ... }

# Register as: "identity.auth.v1.AuthService"
```

## Middleware Integration

All Service Mesh middleware works automatically:

- **Circuit Breaker**: Per-service failure isolation
- **Retry Logic**: Automatic retry with exponential backoff
- **Metrics**: Request counting, latency tracking, error rates
- **Logging**: Request/response logging with correlation IDs
- **Security**: Authentication and authorization
- **Tracing**: Distributed request tracing

## Comparison with Traditional Approach

### Traditional (Explicit Method Registration)
```go
// BAD: Need to explicitly handle every method
func (r *Router) RouteRequest(service, method string, req interface{}) (interface{}, error) {
    switch service {
    case "storage":
        switch method {
        case "Store":
            return client.Store(ctx, req.(*storagepb.StoreRequest))
        case "Retrieve":
            return client.Retrieve(ctx, req.(*storagepb.RetrieveRequest))
        // Need to add EVERY method explicitly!
        }
    }
}
```

### Protocol-Level Routing (Current Implementation)
```go
// GOOD: Works for ANY method automatically
func (r *Router) RouteGRPC(fullMethod string, reqBytes []byte) ([]byte, error) {
    service, method := parseGRPCPath(fullMethod)  // Parse "/storage.v1.StorageService/Store"
    conn := getServiceConnection(service)         // Route to service
    return conn.Invoke(ctx, fullMethod, reqBytes) // Forward raw gRPC call
}
```

## Integration with Existing Services

To use protocol-level routing with existing Blackhole services:

1. **Register services** using their protocol names from .proto files
2. **Route requests** using the new `RouteGRPC` method
3. **Existing middleware** continues to work automatically
4. **Service discovery** and connection pooling work as before

## Performance

Protocol-level routing has minimal overhead compared to direct gRPC calls:

- **Parsing**: Simple string split operation
- **Routing**: Hash table lookup for service discovery
- **Forwarding**: Raw byte forwarding without deserialization
- **Middleware**: Same overhead as before

This approach is used in production by major service meshes handling millions of requests per second.

## Next Steps

This implementation provides the foundation for:

1. **Service Mesh Gateway**: Route external requests to internal services
2. **Load Balancing**: Distribute requests across service instances  
3. **A/B Testing**: Route percentage of traffic to different service versions
4. **Canary Deployments**: Gradually roll out new service versions
5. **Service Federation**: Route between different service clusters