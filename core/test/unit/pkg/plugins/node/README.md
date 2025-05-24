# Node Plugin Tests

The node plugin is a self-contained module with its own go.mod file. To run tests for the node plugin:

```bash
cd core/pkg/plugins/node
GOWORK=off go test ./...
```

## Test Coverage

To run tests with coverage:

```bash
cd core/pkg/plugins/node
GOWORK=off go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Why Tests Are With The Plugin

According to the development guidelines, plugins are self-contained modules that should include their own tests. This is an exception to the general rule of placing tests in the `/test` directory because:

1. Plugins have their own `go.mod` file
2. Plugins are distributed independently
3. Plugin tests need to be run in the plugin's module context
4. This allows plugins to be developed and tested in isolation

## Test Organization

Within the plugin, tests follow the standard Go convention of placing `*_test.go` files alongside the code they test:

- `types/errors_test.go` - Tests for error types
- `p2p/peer_manager_test.go` - Tests for peer management
- `discovery/discovery_test.go` - Tests for peer discovery
- `health/monitor_test.go` - Tests for health monitoring
- `network/manager_test.go` - Tests for network management
- `handlers/handlers_test.go` - Tests for request handlers