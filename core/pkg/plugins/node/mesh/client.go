// Package mesh provides mesh network client for plugins
package mesh

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
)

// Client provides mesh network connectivity for plugins
type Client interface {
	// RegisterService registers this plugin as a service on the mesh
	RegisterService(name string, endpoint string) error
	
	// GetConnection returns a gRPC connection to another service on the mesh
	GetConnection(serviceName string) (*grpc.ClientConn, error)
	
	// PublishEvent publishes an event to the mesh event bus
	PublishEvent(event Event) error
	
	// Subscribe subscribes to events matching the pattern
	Subscribe(pattern string) (<-chan Event, error)
	
	// Close closes the mesh client
	Close() error
}

// Event represents an event on the mesh network
type Event struct {
	Type      string                 `json:"type"`      // e.g., "node.peer.connected"
	Source    string                 `json:"source"`    // Plugin/service that generated the event
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`      // Event-specific data
}

// Config contains mesh client configuration
type Config struct {
	// NodeID is the unique identifier for this node
	NodeID string
	
	// MeshEndpoint is the address of the mesh router
	MeshEndpoint string
	
	// ServiceName is the name of this service/plugin
	ServiceName string
	
	// ServiceEndpoint is where this service listens
	ServiceEndpoint string
}

// MockClient provides a mock implementation for testing
type MockClient struct {
	services map[string]string
	events   chan Event
}

// NewMockClient creates a new mock mesh client
func NewMockClient() *MockClient {
	return &MockClient{
		services: make(map[string]string),
		events:   make(chan Event, 100),
	}
}

// RegisterService registers a service
func (m *MockClient) RegisterService(name string, endpoint string) error {
	m.services[name] = endpoint
	return nil
}

// GetConnection returns a mock connection
func (m *MockClient) GetConnection(serviceName string) (*grpc.ClientConn, error) {
	endpoint, ok := m.services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}
	
	// In real implementation, this would connect through mesh router
	// For now, return direct connection for testing
	return grpc.Dial(endpoint, grpc.WithInsecure())
}

// PublishEvent publishes an event
func (m *MockClient) PublishEvent(event Event) error {
	select {
	case m.events <- event:
		return nil
	default:
		return fmt.Errorf("event buffer full")
	}
}

// Subscribe subscribes to events
func (m *MockClient) Subscribe(pattern string) (<-chan Event, error) {
	// In real implementation, this would filter by pattern
	return m.events, nil
}

// Close closes the client
func (m *MockClient) Close() error {
	close(m.events)
	return nil
}

// NewClient creates a new mesh client
// TODO: Implement real mesh client that connects to mesh router
func NewClient(ctx context.Context, config Config) (Client, error) {
	// For now, return mock client
	// Real implementation would:
	// 1. Connect to mesh router at config.MeshEndpoint
	// 2. Register this service with config.ServiceName and config.ServiceEndpoint
	// 3. Set up event publishing and subscription
	return NewMockClient(), nil
}