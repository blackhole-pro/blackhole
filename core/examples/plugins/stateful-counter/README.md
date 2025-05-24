# Stateful Counter Plugin

This example demonstrates a stateful plugin with hot-swapping capabilities for the Blackhole Framework.

## Features

- **Stateful Operations**: Maintains counters with labels and operation history
- **State Versioning**: Supports migration from V1 to V2 state format
- **Hot-Swapping**: Can export and import state for zero-downtime updates
- **History Tracking**: Keeps track of recent operations
- **Multiple Counters**: Support for named counters with labels

## Building

```bash
go build -o stateful-counter main.go
```

## Plugin Methods

### Counter Operations

- `increment` - Increment a counter
  - Params: `counter` (string, optional, defaults to "default")
  
- `decrement` - Decrement a counter
  - Params: `counter` (string, optional, defaults to "default")
  
- `get` - Get current counter value and label
  - Params: `counter` (string, optional, defaults to "default")

### Label Operations

- `set_label` - Set a label for a counter
  - Params: `counter` (string), `label` (string)

### Bulk Operations

- `get_all` - Get all counters, labels, and last update time
  
- `get_history` - Get recent operation history
  - Params: `limit` (number, optional, defaults to 10)

## State Management

The plugin maintains state in the following format:

```json
{
  "version": "2.0.0",
  "counters": {
    "default": 10,
    "custom": 5
  },
  "labels": {
    "default": "Main Counter",
    "custom": "Custom Counter"
  },
  "history": [
    {
      "timestamp": "2025-05-24T10:00:00Z",
      "counter": "default",
      "value": 10,
      "operation": "increment"
    }
  ],
  "last_updated": "2025-05-24T10:00:00Z"
}
```

## State Migration

The plugin supports automatic migration from V1 to V2 state format:

- **V1 Format**: Simple counter map
- **V2 Format**: Counters with labels and history

When importing V1 state, the plugin automatically:
1. Preserves all counter values
2. Initializes empty labels
3. Creates migration history entries
4. Sets the current timestamp as last_updated

## Hot-Swapping Example

1. Start the plugin with the framework
2. Perform some operations to build state
3. Export state before update: `export_state`
4. Stop the old version
5. Start the new version
6. Import state: `import_state` with the exported data
7. Continue operations with preserved state

## Testing

You can test the plugin manually using the RPC protocol:

```bash
# Initialize
echo '{"id":"1","method":"initialize"}' | ./stateful-counter

# Increment default counter
echo '{"id":"2","method":"handle","params":{"method":"increment"}}' | ./stateful-counter

# Set a label
echo '{"id":"3","method":"handle","params":{"method":"set_label","params":{"counter":"default","label":"Main Counter"}}}' | ./stateful-counter

# Get history
echo '{"id":"4","method":"handle","params":{"method":"get_history","params":{"limit":5}}}' | ./stateful-counter

# Export state
echo '{"id":"5","method":"export_state"}' | ./stateful-counter
```