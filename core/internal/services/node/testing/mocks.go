package testing

import (
	"context"
	"time"

	"github.com/blackhole-pro/blackhole/core/internal/services/node/types"
)

// MockNodeService provides a mock implementation for testing
type MockNodeService struct {
	// Configuration
	Config   *types.NodeConfig
	NodeID   string
	Status   types.NodeStatus
	
	// Mock data
	Peers        map[string]*types.Peer
	Metrics      *types.NodeMetrics
	NetworkState *types.NetworkState
	
	// Control flags for testing
	ShouldFailConnect    bool
	ShouldFailDisconnect bool
	ShouldFailDiscovery  bool
	
	// Call tracking
	ConnectCalls    int
	DisconnectCalls int
	DiscoveryCalls  int
}

// NewMockNodeService creates a new mock node service
func NewMockNodeService() *MockNodeService {
	return &MockNodeService{
		Config: &types.NodeConfig{
			NodeID:      "mock-node-123",
			Version:     "1.0.0-test",
			ListenPort:  9000,
			MaxPeers:    10,
			MinPeers:    2,
		},
		NodeID: "mock-node-123",
		Status: types.NodeStatusHealthy,
		Peers:  make(map[string]*types.Peer),
		Metrics: &types.NodeMetrics{
			TotalConnections:  5,
			ActiveConnections: 3,
			BytesSent:        1024000,
			BytesReceived:    2048000,
			MessagesSent:     100,
			MessagesRecv:     150,
			AverageLatency:   50 * time.Millisecond,
			PacketLoss:       0.01,
			Uptime:          2 * time.Hour,
			CPUUsage:        25.5,
			MemoryUsage:     512 * 1024 * 1024, // 512MB
			LastUpdated:     time.Now(),
		},
		NetworkState: &types.NetworkState{
			ConnectedPeers:      3,
			DiscoveredPeers:     5,
			NetworkHealth:       types.NetworkHealthGood,
			HealthScore:         0.85,
			AverageLatency:      50 * time.Millisecond,
			TotalBandwidthUsed:  3072000,
			LastUpdated:         time.Now(),
		},
	}
}

// Start implements the NodeService interface
func (m *MockNodeService) Start(ctx context.Context) error {
	m.Status = types.NodeStatusHealthy
	return nil
}

// Stop implements the NodeService interface  
func (m *MockNodeService) Stop(ctx context.Context) error {
	m.Status = types.NodeStatusShutdown
	return nil
}

// GetNodeInfo implements the NodeService interface
func (m *MockNodeService) GetNodeInfo() *types.NodeMetrics {
	return m.Metrics
}

// ListPeers implements the NodeService interface
func (m *MockNodeService) ListPeers(filter types.PeerStatus, limit, offset int) ([]*types.Peer, int) {
	var filtered []*types.Peer
	
	for _, peer := range m.Peers {
		if filter == "" || peer.Status == filter {
			filtered = append(filtered, peer)
		}
	}
	
	totalCount := len(filtered)
	
	// Apply pagination
	start := offset
	end := offset + limit
	
	if start >= len(filtered) {
		return []*types.Peer{}, totalCount
	}
	
	if end > len(filtered) {
		end = len(filtered)
	}
	
	return filtered[start:end], totalCount
}

// ConnectToPeer implements the NodeService interface
func (m *MockNodeService) ConnectToPeer(ctx context.Context, request *types.PeerConnectionRequest) (*types.PeerConnectionResponse, error) {
	m.ConnectCalls++
	
	if m.ShouldFailConnect {
		return &types.PeerConnectionResponse{
			Success: false,
			Message: "mock connection failure",
			Error:   types.NewConnectionFailedError(request.Address, nil),
		}, types.NewConnectionFailedError(request.Address, nil)
	}
	
	// Create mock peer
	peerID := "mock-peer-" + request.Address
	peer := &types.Peer{
		ID:          peerID,
		Address:     request.Address,
		Status:      types.PeerStatusConnected,
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
		BytesSent:   0,
		BytesRecv:   0,
		Latency:     30 * time.Millisecond,
		Metadata:    request.Metadata,
	}
	
	m.Peers[peerID] = peer
	
	return &types.PeerConnectionResponse{
		Success: true,
		PeerID:  peerID,
		Message: "connected successfully",
	}, nil
}

// DisconnectFromPeer implements the NodeService interface
func (m *MockNodeService) DisconnectFromPeer(peerID, reason string) error {
	m.DisconnectCalls++
	
	if m.ShouldFailDisconnect {
		return types.NewPeerNotFoundError(peerID)
	}
	
	peer, exists := m.Peers[peerID]
	if !exists {
		return types.NewPeerNotFoundError(peerID)
	}
	
	peer.Status = types.PeerStatusDisconnected
	delete(m.Peers, peerID)
	
	return nil
}

// GetNetworkStatus implements the NodeService interface
func (m *MockNodeService) GetNetworkStatus() *types.NetworkState {
	// Update connected peers count based on current peers
	connectedCount := 0
	for _, peer := range m.Peers {
		if peer.Status == types.PeerStatusConnected {
			connectedCount++
		}
	}
	
	m.NetworkState.ConnectedPeers = connectedCount
	m.NetworkState.DiscoveredPeers = len(m.Peers)
	m.NetworkState.LastUpdated = time.Now()
	
	return m.NetworkState
}

// DiscoverPeers implements the NodeService interface
func (m *MockNodeService) DiscoverPeers(ctx context.Context, request *types.DiscoveryRequest) (*types.DiscoveryResponse, error) {
	m.DiscoveryCalls++
	
	if m.ShouldFailDiscovery {
		return nil, types.NewDiscoveryFailedError(request.Method, nil)
	}
	
	// Return mock discovered addresses based on method
	var addresses []string
	switch request.Method {
	case "bootstrap":
		addresses = []string{
			"bootstrap1.example.com:9000",
			"bootstrap2.example.com:9000",
		}
	case "dht":
		addresses = []string{
			"192.168.1.100:9000",
			"192.168.1.101:9000",
			"192.168.1.102:9000",
		}
	case "local":
		addresses = []string{
			"127.0.0.1:9001",
			"127.0.0.1:9002",
		}
	default:
		addresses = []string{"default.peer.com:9000"}
	}
	
	// Limit to requested max peers
	if request.MaxPeers > 0 && len(addresses) > request.MaxPeers {
		addresses = addresses[:request.MaxPeers]
	}
	
	return &types.DiscoveryResponse{
		DiscoveredAddresses: addresses,
		TotalDiscovered:     len(addresses),
		MethodUsed:          request.Method,
		Duration:            100 * time.Millisecond,
	}, nil
}

// Helper methods for testing

// AddMockPeer adds a mock peer for testing
func (m *MockNodeService) AddMockPeer(address string, status types.PeerStatus) string {
	peerID := "mock-peer-" + address
	peer := &types.Peer{
		ID:          peerID,
		Address:     address,
		Status:      status,
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
		BytesSent:   1000,
		BytesRecv:   2000,
		Latency:     25 * time.Millisecond,
		Metadata:    map[string]string{"type": "mock"},
	}
	
	m.Peers[peerID] = peer
	return peerID
}

// SetNetworkHealth sets the network health for testing
func (m *MockNodeService) SetNetworkHealth(health types.NetworkHealth, score float64) {
	m.NetworkState.NetworkHealth = health
	m.NetworkState.HealthScore = score
}

// SetPeerLatency sets latency for a specific peer
func (m *MockNodeService) SetPeerLatency(peerID string, latency time.Duration) {
	if peer, exists := m.Peers[peerID]; exists {
		peer.Latency = latency
	}
}

// GetCallCounts returns the number of calls made to each method
func (m *MockNodeService) GetCallCounts() map[string]int {
	return map[string]int{
		"connect":    m.ConnectCalls,
		"disconnect": m.DisconnectCalls,
		"discovery":  m.DiscoveryCalls,
	}
}

// Reset resets the mock to initial state
func (m *MockNodeService) Reset() {
	m.Peers = make(map[string]*types.Peer)
	m.ConnectCalls = 0
	m.DisconnectCalls = 0
	m.DiscoveryCalls = 0
	m.ShouldFailConnect = false
	m.ShouldFailDisconnect = false
	m.ShouldFailDiscovery = false
	m.Status = types.NodeStatusHealthy
}

// MockServiceClients provides mock service clients for testing
type MockServiceClients struct {
	IdentityClient *MockIdentityClient
	StorageClient  *MockStorageClient
}

// NewMockServiceClients creates new mock service clients
func NewMockServiceClients() *MockServiceClients {
	return &MockServiceClients{
		IdentityClient: NewMockIdentityClient(),
		StorageClient:  NewMockStorageClient(),
	}
}

// MockIdentityClient provides a mock identity client
type MockIdentityClient struct {
	ShouldFailChallenge bool
	ShouldFailVerify    bool
	ChallengeCalls      int
	VerifyCalls         int
}

// NewMockIdentityClient creates a new mock identity client
func NewMockIdentityClient() *MockIdentityClient {
	return &MockIdentityClient{}
}

// GenerateChallenge mocks the identity service challenge generation
func (m *MockIdentityClient) GenerateChallenge(ctx context.Context, did string, purpose string) (interface{}, error) {
	m.ChallengeCalls++
	
	if m.ShouldFailChallenge {
		return nil, types.NewInternalError("generate challenge", nil)
	}
	
	// Return mock challenge
	challenge := map[string]interface{}{
		"id":      "mock-challenge-123",
		"did":     did,
		"nonce":   "mock-nonce-456",
		"domain":  "blackhole.test",
		"purpose": purpose,
	}
	
	return challenge, nil
}

// MockStorageClient provides a mock storage client
type MockStorageClient struct {
	ShouldFailGet   bool
	ShouldFailStore bool
	GetCalls        int
	StoreCalls      int
}

// NewMockStorageClient creates a new mock storage client
func NewMockStorageClient() *MockStorageClient {
	return &MockStorageClient{}
}

// GetDIDDocument mocks getting a DID document from storage
func (m *MockStorageClient) GetDIDDocument(ctx context.Context, did string) (interface{}, error) {
	m.GetCalls++
	
	if m.ShouldFailGet {
		return nil, types.NewInternalError("get DID document", nil)
	}
	
	// Return mock DID document
	document := map[string]interface{}{
		"id":               did,
		"context":          []string{"https://www.w3.org/ns/did/v1"},
		"verificationMethod": []map[string]interface{}{
			{
				"id":   did + "#key-1",
				"type": "Ed25519VerificationKey2020",
				"controller": did,
				"publicKeyMultibase": "mock-public-key",
			},
		},
	}
	
	return document, nil
}

// Example test helper functions

// CreateTestNodeConfig creates a test configuration
func CreateTestNodeConfig() *types.NodeConfig {
	return &types.NodeConfig{
		NodeID:            "test-node-123",
		Version:           "1.0.0-test",
		ListenPort:        19000,
		ListenAddress:     "127.0.0.1",
		MaxPeers:          5,
		MinPeers:          1,
		ConnectionTimeout: 5 * time.Second,
		PingInterval:      2 * time.Second,
		BootstrapPeers:    []string{"test-bootstrap:9000"},
		DiscoveryMethods:  []string{"bootstrap", "local"},
		DHT: types.DHTConfig{
			Enabled:         false, // Disable for testing
			RefreshInterval: 1 * time.Minute,
			BucketSize:      10,
		},
		BandwidthLimit:   1000000, // 1MB/s
		MessageQueueSize: 100,
		EnableTLS:        false,
	}
}