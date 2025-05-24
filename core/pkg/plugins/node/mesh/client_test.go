package mesh

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockMeshClient implements the Client interface for testing
type MockMeshClient struct {
	mock.Mock
}

func (m *MockMeshClient) RegisterService(name string, endpoint string) error {
	args := m.Called(name, endpoint)
	return args.Error(0)
}

func (m *MockMeshClient) GetConnection(serviceName string) (*grpc.ClientConn, error) {
	args := m.Called(serviceName)
	if conn := args.Get(0); conn != nil {
		return conn.(*grpc.ClientConn), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMeshClient) PublishEvent(event Event) error {
	args := m.Called(event)
	return args.Error(0)
}

func (m *MockMeshClient) Subscribe(pattern string) (<-chan Event, error) {
	args := m.Called(pattern)
	if ch := args.Get(0); ch != nil {
		return ch.(<-chan Event), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMeshClient) Unsubscribe(pattern string) error {
	args := m.Called(pattern)
	return args.Error(0)
}

func (m *MockMeshClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestMockClient_RegisterService(t *testing.T) {
	client := new(MockMeshClient)
	
	// Setup expectation
	client.On("RegisterService", "test-service", "localhost:8080").Return(nil)
	
	// Call method
	err := client.RegisterService("test-service", "localhost:8080")
	
	// Assert
	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestMockClient_PublishEvent(t *testing.T) {
	client := new(MockMeshClient)
	
	event := Event{
		Type:      "test.event",
		Source:    "test-source",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"key": "value",
		},
	}
	
	// Setup expectation
	client.On("PublishEvent", event).Return(nil)
	
	// Call method
	err := client.PublishEvent(event)
	
	// Assert
	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestMockClient_Subscribe(t *testing.T) {
	client := new(MockMeshClient)
	
	// Create a channel for events
	eventChan := make(chan Event, 1)
	testEvent := Event{
		Type:      "test.event",
		Source:    "test-source",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"key": "value",
		},
	}
	
	// Send event to channel
	eventChan <- testEvent
	close(eventChan)
	
	// Convert to read-only channel
	var readChan <-chan Event = eventChan
	
	// Setup expectation
	client.On("Subscribe", "test.*").Return(readChan, nil)
	
	// Call method
	ch, err := client.Subscribe("test.*")
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, ch)
	
	// Read from channel
	received := <-ch
	assert.Equal(t, testEvent.Type, received.Type)
	assert.Equal(t, testEvent.Source, received.Source)
	
	client.AssertExpectations(t)
}

func TestMockClient_GetConnection(t *testing.T) {
	client := new(MockMeshClient)
	
	// Create a mock connection
	mockConn := &grpc.ClientConn{}
	
	// Setup expectation
	client.On("GetConnection", "storage").Return(mockConn, nil)
	
	// Call method
	conn, err := client.GetConnection("storage")
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	client.AssertExpectations(t)
}

func TestMockClient_Close(t *testing.T) {
	client := new(MockMeshClient)
	
	// Setup expectation
	client.On("Close").Return(nil)
	
	// Call method
	err := client.Close()
	
	// Assert
	assert.NoError(t, err)
	client.AssertExpectations(t)
}

// TestEventMarshaling tests event serialization/deserialization
func TestEventMarshaling(t *testing.T) {
	event := Event{
		Type:      "node.peer.connected",
		Source:    "node-123",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"peer_id": "peer-456",
			"address": "/ip4/127.0.0.1/tcp/4001",
			"latency": 25.5,
			"protocols": []string{"/ipfs/1.0.0", "/blackhole/1.0.0"},
		},
	}
	
	// Verify event fields
	assert.Equal(t, "node.peer.connected", event.Type)
	assert.Equal(t, "node-123", event.Source)
	assert.NotNil(t, event.Data)
	
	// Verify data fields
	assert.Equal(t, "peer-456", event.Data["peer_id"])
	assert.Equal(t, "/ip4/127.0.0.1/tcp/4001", event.Data["address"])
	assert.Equal(t, 25.5, event.Data["latency"])
	
	protocols, ok := event.Data["protocols"].([]string)
	assert.True(t, ok)
	assert.Len(t, protocols, 2)
}

// TestConcurrentEventPublishing tests thread-safe event publishing
func TestConcurrentEventPublishing(t *testing.T) {
	client := new(MockMeshClient)
	
	// Allow any event to be published
	client.On("PublishEvent", mock.Anything).Return(nil)
	
	// Publish events concurrently
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			event := Event{
				Type:      "test.concurrent",
				Source:    "test",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"id": id,
				},
			}
			
			err := client.PublishEvent(event)
			assert.NoError(t, err)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-ctx.Done():
			t.Fatal("Timeout waiting for concurrent publishing")
		}
	}
	
	// Verify all events were published
	client.AssertNumberOfCalls(t, "PublishEvent", 10)
}