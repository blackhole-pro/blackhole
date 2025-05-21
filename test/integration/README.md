# Integration Tests

This directory contains integration tests for the Blackhole platform.

## Running Integration Tests

Integration tests validate the system's behavior with real processes and components. These tests create temporary directories, build test binaries, and verify the system's behavior in a more realistic environment.

### Prerequisites

- Go 1.21 or later
- The main Blackhole module and its dependencies installed

### Running Tests

From the project root, run:

```sh
# Run all integration tests
go test ./test/integration/... -v

# Run a specific integration test
go test ./test/integration -run TestAppAdapter_Integration -v

# Skip integration tests when running unit tests
go test ./... -short
```

### Test Categories

1. **App Adapter Tests** - Test the application adapter that wraps the process orchestrator.
   - Service discovery
   - Service lifecycle (start, stop)
   - Configuration management

2. **Process Orchestrator Tests** - Test the core process orchestrator directly.
   - Service process management
   - Auto-restart capability
   - Graceful shutdown

## Test Structure

Each integration test typically:

1. Creates a temporary test directory
2. Builds test service binaries
3. Sets up test configuration
4. Initializes the system components
5. Exercises the functionality being tested
6. Verifies the results

## Adding New Integration Tests

To add a new integration test:

1. Create a new test file in this directory
2. Use the existing helper functions for setup when possible
3. Structure your test to clean up resources when done
4. Add your test to the appropriate test category in this README

## Test Services

The `test-service` directory contains a simple service used for integration testing. It:

- Logs startup information
- Creates status files in its data directory
- Responds to shutdown signals gracefully
- Can be built and run by the test framework