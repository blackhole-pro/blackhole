package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	
	"node/mesh"
	"node/plugin"
	nodev1 "node/proto/v1"
	"node/types"
)

// Mock dependencies
type mockPlugin struct {
	mock.Mock
	peerManager   *mockPeerManager
	netManager    *mockNetManager
	healthMonitor *mockHealthMonitor
	discovery     *mockDiscovery
}

func (m *mockPlugin) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockPlugin) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockPlugin) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockPlugin) Info() types.PluginInfo {
	args := m.Called()
	return args.Get(0).(types.PluginInfo)
}

func (m *mockPlugin) GetPeerManager() types.PeerManager {
	return m.peerManager
}

func (m *mockPlugin) GetNetManager() types.NetworkManager {
	return m.netManager
}

func (m *mockPlugin) GetHealthMonitor() types.HealthMonitor {
	return m.healthMonitor
}

func (m *mockPlugin) GetDiscovery() *mockDiscovery {
	return m.discovery
}

func (m *mockPlugin) GetNetworkMetrics() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

type mockPeerManager struct {
	mock.Mock
}

func (m *mockPeerManager) Connect(ctx context.Context, peerID, address string) error {
	args := m.Called(ctx, peerID, address)
	return args.Error(0)
}

func (m *mockPeerManager) Disconnect(ctx context.Context, peerID, reason string) error {
	args := m.Called(ctx, peerID, reason)
	return args.Error(0)
}

func (m *mockPeerManager) GetPeer(peerID string) (*types.PeerInfo, error) {
	args := m.Called(peerID)
	if peer := args.Get(0); peer != nil {
		return peer.(*types.PeerInfo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockPeerManager) ListPeers(filter types.PeerFilter) ([]*types.PeerInfo, error) {
	args := m.Called(filter)
	if peers := args.Get(0); peers != nil {
		return peers.([]*types.PeerInfo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockPeerManager) GetPeerCount() (active, total int) {
	args := m.Called()
	return args.Int(0), args.Int(1)
}

type mockNetManager struct {
	mock.Mock
}

func (m *mockNetManager) GetMetrics() *types.NetworkMetrics {
	args := m.Called()
	if metrics := args.Get(0); metrics != nil {
		return metrics.(*types.NetworkMetrics)
	}
	return nil
}

func (m *mockNetManager) GetNetworkStatus() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

type mockHealthMonitor struct {
	mock.Mock
}

func (m *mockHealthMonitor) GetHealth() *types.HealthStatus {
	args := m.Called()
	if health := args.Get(0); health != nil {
		return health.(*types.HealthStatus)
	}
	return nil
}

func (m *mockHealthMonitor) CalculateHealthScore() float64 {
	args := m.Called()
	return args.Get(0).(float64)
}

type mockDiscovery struct {
	mock.Mock
}

func (m *mockDiscovery) DiscoverPeers(ctx context.Context, method string, limit int) ([]*types.DiscoveredPeer, error) {
	args := m.Called(ctx, method, limit)
	if peers := args.Get(0); peers != nil {
		return peers.([]*types.DiscoveredPeer), args.Error(1)
	}
	return nil, args.Error(1)
}

type mockMeshClient struct {
	mock.Mock
	events chan mesh.Event
}

func newMockMeshClient() *mockMeshClient {
	return &mockMeshClient{
		events: make(chan mesh.Event, 100),
	}
}

func (m *mockMeshClient) RegisterService(name string, endpoint string) error {
	args := m.Called(name, endpoint)
	return args.Error(0)
}

func (m *mockMeshClient) GetConnection(serviceName string) (*grpc.ClientConn, error) {
	args := m.Called(serviceName)
	if conn := args.Get(0); conn != nil {
		return conn.(*grpc.ClientConn), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMeshClient) PublishEvent(event mesh.Event) error {
	args := m.Called(event)
	// Store event for verification
	select {
	case m.events <- event:
	default:
		// Channel full
	}
	return args.Error(0)
}

func (m *mockMeshClient) Subscribe(pattern string) (<-chan mesh.Event, error) {
	args := m.Called(pattern)
	if ch := args.Get(0); ch != nil {
		return ch.(<-chan mesh.Event), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMeshClient) Unsubscribe(pattern string) error {
	args := m.Called(pattern)
	return args.Error(0)
}

func (m *mockMeshClient) Close() error {
	args := m.Called()
	close(m.events)
	return args.Error(0)
}

// Test helpers
func setupMockPlugin() *mockPlugin {
	p := new(mockPlugin)
	p.peerManager = new(mockPeerManager)
	p.netManager = new(mockNetManager)
	p.healthMonitor = new(mockHealthMonitor)
	p.discovery = new(mockDiscovery)
	return p
}

func TestNodePluginServer_Initialize(t *testing.T) {
	plugin := setupMockPlugin()
	meshClient := newMockMeshClient()
	server := NewNodePluginServer(plugin, meshClient)
	
	req := &nodev1.InitializeRequest{
		Config: &nodev1.NodeConfig{
			NodeId:           "test-node",
			P2PPort:          4001,
			ListenAddresses:  []string{"/ip4/0.0.0.0/tcp/4001"},
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
	assert.Contains(t, resp.Multiaddrs[0], "/p2p/")
}

func TestNodePluginServer_Start(t *testing.T) {
	plugin := setupMockPlugin()
	meshClient := newMockMeshClient()
	server := NewNodePluginServer(plugin, meshClient)
	
	// Initialize first
	initReq := &nodev1.InitializeRequest{
		Config: &nodev1.NodeConfig{
			NodeId:           "test-node",
			P2PPort:          4001,
			ListenAddresses:  []string{"/ip4/0.0.0.0/tcp/4001"},
			EnableDiscovery:  true,
			EnableEncryption: true,
		},
	}
	_, err := server.Initialize(context.Background(), initReq)
	assert.NoError(t, err)
	
	// Setup mocks
	plugin.On("Start", mock.Anything).Return(nil)
	meshClient.On("PublishEvent", mock.Anything).Return(nil)
	
	// Start plugin
	req := &nodev1.StartRequest{}
	resp, err := server.Start(context.Background(), req)
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Len(t, resp.Endpoints, 1)
	assert.NotNil(t, resp.Readiness)
	
	plugin.AssertExpectations(t)
	
	// Verify event was published
	meshClient.AssertCalled(t, "PublishEvent", mock.MatchedBy(func(event mesh.Event) bool {
		return event.Type == "node.started"
	}))
}

func TestNodePluginServer_HealthCheck(t *testing.T) {
	plugin := setupMockPlugin()
	meshClient := newMockMeshClient()
	server := NewNodePluginServer(plugin, meshClient)
	
	// Setup mocks
	plugin.On("HealthCheck").Return(nil)
	plugin.healthMonitor.On("GetHealth").Return(types.HealthStatus{
		Status:         "healthy",
		HealthScore:    0.95,
		BandwidthUsage: 1024 * 1024, // 1MB
		LastUpdated:    time.Now(),
	})
	plugin.netManager.On("GetMetrics").Return(types.NetworkMetrics{})
	plugin.peerManager.On("GetPeerCount").Return(5, 10)
	
	// Test without diagnostics
	req := &nodev1.HealthCheckRequest{
		IncludeDiagnostics: false,
	}
	resp, err := server.HealthCheck(context.Background(), req)
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Healthy)
	assert.Equal(t, "healthy", resp.Status)
	assert.Nil(t, resp.Diagnostics)
	
	// Test with diagnostics
	req.IncludeDiagnostics = true
	resp, err = server.HealthCheck(context.Background(), req)
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Diagnostics)
	assert.Equal(t, int32(5), resp.Diagnostics.ActiveConnections)
	
	plugin.AssertExpectations(t)
	plugin.healthMonitor.AssertExpectations(t)
	plugin.peerManager.AssertExpectations(t)
}

func TestNodePluginServer_ConnectPeer(t *testing.T) {
	plugin := setupMockPlugin()
	meshClient := newMockMeshClient()
	server := NewNodePluginServer(plugin, meshClient)
	
	// Initialize first
	server.config = &types.NodeConfig{NodeID: "test-node"}
	server.initialized = true
	
	// Setup mocks
	peerInfo := &types.PeerInfo{
		ID:        "peer-123",
		Address:   "/ip4/192.168.1.10/tcp/4001",
		Status:    "connected",
		ConnectedAt: time.Now(),
		LastSeen:  time.Now(),
	}
	
	plugin.peerManager.On("Connect", mock.Anything, "peer-123", "/ip4/192.168.1.10/tcp/4001").Return(nil)
	plugin.peerManager.On("GetPeer", "peer-123").Return(peerInfo, nil)
	meshClient.On("PublishEvent", mock.Anything).Return(nil)
	
	// Connect to peer
	req := &nodev1.ConnectPeerRequest{
		PeerId: "peer-123",
		Addrs:  []string{"/ip4/192.168.1.10/tcp/4001"},
	}
	resp, err := server.ConnectPeer(context.Background(), req)
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.PeerInfo)
	assert.Equal(t, "peer-123", resp.PeerInfo.Id)
	
	plugin.peerManager.AssertExpectations(t)
	
	// Verify peer connected event was published
	meshClient.AssertCalled(t, "PublishEvent", mock.MatchedBy(func(event mesh.Event) bool {
		return event.Type == "node.peer.connected" &&
			event.Data["peer_id"] == "peer-123"
	}))
}

func TestNodePluginServer_ListPeers(t *testing.T) {
	plugin := setupMockPlugin()
	meshClient := newMockMeshClient()
	server := NewNodePluginServer(plugin, meshClient)
	
	// Setup mocks
	peers := []*types.PeerInfo{
		{
			ID:      "peer-1",
			Address: "/ip4/192.168.1.10/tcp/4001",
			Status:  "connected",
		},
		{
			ID:      "peer-2",
			Address: "/ip4/192.168.1.11/tcp/4001",
			Status:  "connected",
		},
	}
	
	filter := types.PeerFilter{
		Status: "connected",
		Limit:  10,
		Offset: 0,
	}
	
	plugin.peerManager.On("ListPeers", filter).Return(peers, nil)
	plugin.peerManager.On("GetPeerCount").Return(2, 5)
	
	// List peers
	req := &nodev1.ListPeersRequest{
		StatusFilter: "connected",
		Limit:        10,
		Offset:       0,
	}
	resp, err := server.ListPeers(context.Background(), req)
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Peers, 2)
	assert.Equal(t, int32(5), resp.TotalCount)
	assert.NotNil(t, resp.Stats)
	assert.Equal(t, int32(2), resp.Stats.Connected)
	
	plugin.peerManager.AssertExpectations(t)
}

func TestNodePluginServer_GetNetworkStatus(t *testing.T) {
	plugin := setupMockPlugin()
	meshClient := newMockMeshClient()
	server := NewNodePluginServer(plugin, meshClient)
	
	// Initialize server config
	server.config = &types.NodeConfig{
		NodeID:           "test-node",
		MaxBandwidthMbps: 100,
	}
	
	// Setup mocks
	health := types.HealthStatus{
		Status:      "healthy",
		HealthScore: 0.95,
		LastUpdated: time.Now(),
	}
	
	metrics := types.NetworkMetrics{
		TotalConnections:  100,
		ActiveConnections: 10,
		BytesSent:         1024 * 1024,
		BytesReceived:     2048 * 1024,
		LastReset:         time.Now().Add(-1 * time.Hour),
	}
	
	networkStatus := map[string]interface{}{
		"rates": map[string]float64{
			"mbpsIn":  10.5,
			"mbpsOut": 8.3,
		},
	}
	
	plugin.healthMonitor.On("GetHealth").Return(health)
	plugin.netManager.On("GetMetrics").Return(metrics)
	plugin.netManager.On("GetNetworkStatus").Return(networkStatus)
	plugin.peerManager.On("GetPeerCount").Return(10, 20)
	meshClient.On("PublishEvent", mock.Anything).Return(nil)
	
	// Get network status
	req := &nodev1.GetNetworkStatusRequest{
		IncludeBandwidth: true,
		IncludeRouting:   true,
	}
	resp, err := server.GetNetworkStatus(context.Background(), req)
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Health)
	assert.Equal(t, "healthy", resp.Health.Status)
	assert.NotNil(t, resp.Metrics)
	assert.Equal(t, int64(100), resp.Metrics.TotalConnections)
	assert.NotNil(t, resp.Bandwidth)
	assert.Equal(t, 10.5, resp.Bandwidth.RateInMbps)
	assert.NotNil(t, resp.Routing)
	assert.Equal(t, int32(20), resp.Routing.RoutingTableSize)
	
	plugin.AssertExpectations(t)
	plugin.healthMonitor.AssertExpectations(t)
	plugin.netManager.AssertExpectations(t)
	plugin.peerManager.AssertExpectations(t)
}

func TestNodePluginServer_DiscoverPeers(t *testing.T) {
	plugin := setupMockPlugin()
	meshClient := newMockMeshClient()
	server := NewNodePluginServer(plugin, meshClient)
	
	// Initialize server config
	server.config = &types.NodeConfig{
		NodeID:          "test-node",
		DiscoveryMethod: "mdns",
	}
	
	// Setup mocks
	discoveredPeers := []*types.DiscoveredPeer{
		{
			ID:      "peer-new-1",
			Address: "/ip4/192.168.1.20/tcp/4001",
			Source:  "mdns",
		},
		{
			ID:      "peer-new-2",
			Address: "/ip4/192.168.1.21/tcp/4001",
			Source:  "mdns",
		},
	}
	
	plugin.discovery.On("DiscoverPeers", mock.Anything, "mdns", 10).Return(discoveredPeers, nil)
	meshClient.On("PublishEvent", mock.Anything).Return(nil)
	
	// Discover peers
	req := &nodev1.DiscoverPeersRequest{
		Method: "mdns",
		Limit:  10,
	}
	resp, err := server.DiscoverPeers(context.Background(), req)
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Peers, 2)
	assert.Equal(t, "mdns", resp.MethodUsed)
	assert.Equal(t, int32(2), resp.TotalDiscovered)
	
	plugin.discovery.AssertExpectations(t)
	
	// Verify discovery events were published
	meshClient.AssertNumberOfCalls(t, "PublishEvent", 2)
}

// Test event publishing verification
func TestNodePluginServer_EventPublishing(t *testing.T) {
	plugin := setupMockPlugin()
	meshClient := newMockMeshClient()
	server := NewNodePluginServer(plugin, meshClient)
	
	// Initialize server
	server.config = &types.NodeConfig{NodeID: "test-node"}
	server.initialized = true
	
	// Track published events
	publishedEvents := []mesh.Event{}
	meshClient.On("PublishEvent", mock.Anything).Run(func(args mock.Arguments) {
		event := args.Get(0).(mesh.Event)
		publishedEvents = append(publishedEvents, event)
	}).Return(nil)
	
	// Test various event publishing scenarios
	
	// 1. Service start event
	plugin.On("Start", mock.Anything).Return(nil)
	_, err := server.Start(context.Background(), &nodev1.StartRequest{})
	assert.NoError(t, err)
	
	// 2. Peer connected event
	plugin.peerManager.On("Connect", mock.Anything, "peer-1", "addr1").Return(nil)
	plugin.peerManager.On("GetPeer", "peer-1").Return(&types.PeerInfo{ID: "peer-1"}, nil)
	_, err = server.ConnectPeer(context.Background(), &nodev1.ConnectPeerRequest{
		PeerId: "peer-1",
		Addrs:  []string{"addr1"},
	})
	assert.NoError(t, err)
	
	// 3. Peer disconnected event
	plugin.peerManager.On("Disconnect", mock.Anything, "peer-1", "test").Return(nil)
	_, err = server.DisconnectPeer(context.Background(), &nodev1.DisconnectPeerRequest{
		PeerId: "peer-1",
		Reason: "test",
	})
	assert.NoError(t, err)
	
	// Verify events
	assert.Len(t, publishedEvents, 3)
	
	// Check event types
	assert.Equal(t, "node.started", publishedEvents[0].Type)
	assert.Equal(t, "node.peer.connected", publishedEvents[1].Type)
	assert.Equal(t, "node.peer.disconnected", publishedEvents[2].Type)
	
	// Check event source
	for _, event := range publishedEvents {
		assert.Equal(t, "test-node", event.Source)
		assert.NotZero(t, event.Timestamp)
		assert.NotNil(t, event.Data)
	}
}