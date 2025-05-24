package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
	
	"node/mesh"
	"node/plugin"
	nodev1 "node/proto/v1"
	"node/types"
)

// Create a test implementation of mesh client
type testMeshClient struct {
	events []mesh.Event
}

func (c *testMeshClient) RegisterService(name string, endpoint string) error {
	return nil
}

func (c *testMeshClient) GetConnection(serviceName string) (*grpc.ClientConn, error) {
	return nil, nil
}

func (c *testMeshClient) PublishEvent(event mesh.Event) error {
	c.events = append(c.events, event)
	return nil
}

func (c *testMeshClient) Subscribe(pattern string) (<-chan mesh.Event, error) {
	ch := make(chan mesh.Event)
	close(ch)
	return ch, nil
}

func (c *testMeshClient) Unsubscribe(pattern string) error {
	return nil
}

func (c *testMeshClient) Close() error {
	return nil
}

func TestNodePluginServer_Integration(t *testing.T) {
	// Create a real plugin
	logger := zap.NewNop()
	config := &types.NodeConfig{
		NodeID:            "test-node",
		P2PPort:           4001,
		ListenAddresses:   []string{"/ip4/0.0.0.0/tcp/4001"},
		EnableDiscovery:   true,
		DiscoveryMethod:   "mdns",
		DiscoveryInterval: 60 * time.Second,
		MaxPeers:          50,
		MaxBandwidthMbps:  100,
		ConnectionTimeout: 30 * time.Second,
		EnableEncryption:  true,
		PrivateKeyPath:    "/path/to/key",
	}
	
	pluginInstance, err := plugin.NewPlugin(config, logger)
	assert.NoError(t, err)
	
	// Create mesh client
	meshClient := &testMeshClient{}
	
	// Create server
	server := NewNodePluginServer(pluginInstance, meshClient)
	
	// Test Initialize
	t.Run("Initialize", func(t *testing.T) {
		req := &nodev1.InitializeRequest{
			Config: &nodev1.NodeConfig{
				NodeId:           "test-node-2",
				P2PPort:          4002,
				ListenAddresses:  []string{"/ip4/0.0.0.0/tcp/4002"},
				BootstrapPeers:   []string{},
				EnableDiscovery:  true,
				DiscoveryMethod:  "mdns",
				MaxPeers:         50,
				MaxBandwidthMbps: 100,
				ConnectionTimeout: durationpb.New(30 * time.Second),
				EnableEncryption: true,
				PrivateKeyPath:   "/path/to/key",
			},
		}
		
		resp, err := server.Initialize(context.Background(), req)
		
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.PeerId, "12D3KooW")
		assert.Len(t, resp.Multiaddrs, 1)
	})
	
	// Test Start
	t.Run("Start", func(t *testing.T) {
		req := &nodev1.StartRequest{}
		resp, err := server.Start(context.Background(), req)
		
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Endpoints)
		assert.NotNil(t, resp.Readiness)
		
		// Check that start event was published
		assert.Len(t, meshClient.events, 1)
		assert.Equal(t, "node.started", meshClient.events[0].Type)
	})
	
	// Test Health Check
	t.Run("HealthCheck", func(t *testing.T) {
		req := &nodev1.HealthCheckRequest{
			IncludeDiagnostics: true,
		}
		resp, err := server.HealthCheck(context.Background(), req)
		
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Healthy)
		assert.NotNil(t, resp.Diagnostics)
	})
	
	// Test Get Info
	t.Run("GetInfo", func(t *testing.T) {
		req := &nodev1.GetInfoRequest{}
		resp, err := server.GetInfo(context.Background(), req)
		
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "node", resp.Name)
		assert.NotEmpty(t, resp.Version)
		assert.NotNil(t, resp.Capabilities)
		assert.NotNil(t, resp.Status)
	})
	
	// Test List Peers
	t.Run("ListPeers", func(t *testing.T) {
		req := &nodev1.ListPeersRequest{
			StatusFilter: "connected",
			Limit:        10,
			Offset:       0,
		}
		resp, err := server.ListPeers(context.Background(), req)
		
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Stats)
	})
	
	// Test Stop
	t.Run("Stop", func(t *testing.T) {
		req := &nodev1.StopRequest{
			Reason: "test shutdown",
		}
		resp, err := server.Stop(context.Background(), req)
		
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		
		// Check that stop event was published
		found := false
		for _, event := range meshClient.events {
			if event.Type == "node.stopped" {
				found = true
				break
			}
		}
		assert.True(t, found, "Stop event should have been published")
	})
}

// Test concurrent operations
func TestNodePluginServer_Concurrent(t *testing.T) {
	// Create plugin
	logger := zap.NewNop()
	config := &types.NodeConfig{
		NodeID:            "test-concurrent",
		P2PPort:           4003,
		ListenAddresses:   []string{"/ip4/0.0.0.0/tcp/4003"},
		EnableDiscovery:   false,
		DiscoveryInterval: 60 * time.Second,
		MaxPeers:          100,
	}
	
	pluginInstance, err := plugin.NewPlugin(config, logger)
	assert.NoError(t, err)
	
	meshClient := &testMeshClient{}
	server := NewNodePluginServer(pluginInstance, meshClient)
	
	// Initialize and start
	_, err = server.Initialize(context.Background(), &nodev1.InitializeRequest{
		Config: &nodev1.NodeConfig{
			NodeId:          "test-concurrent",
			P2PPort:         4003,
			ListenAddresses: []string{"/ip4/0.0.0.0/tcp/4003"},
		},
	})
	assert.NoError(t, err)
	
	_, err = server.Start(context.Background(), &nodev1.StartRequest{})
	assert.NoError(t, err)
	
	// Run concurrent operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	done := make(chan bool, 4)
	
	// Concurrent health checks
	go func() {
		for i := 0; i < 10; i++ {
			_, err := server.HealthCheck(ctx, &nodev1.HealthCheckRequest{})
			assert.NoError(t, err)
		}
		done <- true
	}()
	
	// Concurrent list peers
	go func() {
		for i := 0; i < 10; i++ {
			_, err := server.ListPeers(ctx, &nodev1.ListPeersRequest{})
			assert.NoError(t, err)
		}
		done <- true
	}()
	
	// Concurrent get info
	go func() {
		for i := 0; i < 10; i++ {
			_, err := server.GetInfo(ctx, &nodev1.GetInfoRequest{})
			assert.NoError(t, err)
		}
		done <- true
	}()
	
	// Concurrent network status
	go func() {
		for i := 0; i < 10; i++ {
			_, err := server.GetNetworkStatus(ctx, &nodev1.GetNetworkStatusRequest{})
			assert.NoError(t, err)
		}
		done <- true
	}()
	
	// Wait for completion
	for i := 0; i < 4; i++ {
		select {
		case <-done:
		case <-ctx.Done():
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
	
	// Clean shutdown
	_, err = server.Stop(context.Background(), &nodev1.StopRequest{})
	assert.NoError(t, err)
}