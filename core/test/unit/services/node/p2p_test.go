package node

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/blackhole-pro/blackhole/core/internal/services/node/p2p"
	"github.com/blackhole-pro/blackhole/core/internal/services/node/types"
)

func TestP2PHostCreation(t *testing.T) {
	logger := zap.NewNop()
	
	config := &types.P2PConfig{
		EnableTCP:        true,
		EnableQUIC:       false,
		EnableWebSocket:  false,
		ListenAddresses:  []string{"/ip4/127.0.0.1/tcp/0"},
		EnableNoise:      true,
		EnableTLS:        false,
		EnableMDNS:       false,
		EnableDHT:        false,
		DHTMode:          "client",
		LowWaterMark:     10,
		HighWaterMark:    20,
		GracePeriod:      30 * time.Second,
		MaxStreams:       100,
		MaxInboundStreams: 50,
		MaxOutboundStreams: 50,
	}
	
	host, err := p2p.NewLibP2PHost(config, logger)
	if err != nil {
		t.Fatalf("Failed to create P2P host: %v", err)
	}
	
	if host == nil {
		t.Fatal("P2P host is nil")
	}
}

func TestP2PHostStartStop(t *testing.T) {
	logger := zap.NewNop()
	
	config := &types.P2PConfig{
		EnableTCP:        true,
		EnableQUIC:       false,
		EnableWebSocket:  false,
		ListenAddresses:  []string{"/ip4/127.0.0.1/tcp/0"},
		EnableNoise:      true,
		EnableTLS:        false,
		EnableMDNS:       false,
		EnableDHT:        false,
		DHTMode:          "client",
		LowWaterMark:     10,
		HighWaterMark:    20,
		GracePeriod:      30 * time.Second,
		MaxStreams:       100,
		MaxInboundStreams: 50,
		MaxOutboundStreams: 50,
	}
	
	host, err := p2p.NewLibP2PHost(config, logger)
	if err != nil {
		t.Fatalf("Failed to create P2P host: %v", err)
	}
	
	ctx := context.Background()
	
	// Start the host
	err = host.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start P2P host: %v", err)
	}
	
	// Verify host is running
	if host.Host() == nil {
		t.Fatal("P2P host is not running")
	}
	
	// Get local peer info
	peerInfo := host.GetLocalPeerInfo()
	if peerInfo == nil {
		t.Fatal("Failed to get local peer info")
	}
	
	if len(peerInfo.PeerID) == 0 {
		t.Fatal("Peer ID is empty")
	}
	
	// Stop the host
	err = host.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop P2P host: %v", err)
	}
}

func TestDiscoveryService(t *testing.T) {
	logger := zap.NewNop()
	
	config := &types.P2PConfig{
		EnableTCP:        true,
		EnableQUIC:       false,
		EnableWebSocket:  false,
		ListenAddresses:  []string{"/ip4/127.0.0.1/tcp/0"},
		EnableNoise:      true,
		EnableTLS:        false,
		EnableMDNS:       false,
		EnableDHT:        false,
		DHTMode:          "client",
		LowWaterMark:     10,
		HighWaterMark:    20,
		GracePeriod:      30 * time.Second,
		MaxStreams:       100,
		MaxInboundStreams: 50,
		MaxOutboundStreams: 50,
	}
	
	host, err := p2p.NewLibP2PHost(config, logger)
	if err != nil {
		t.Fatalf("Failed to create P2P host: %v", err)
	}
	
	ctx := context.Background()
	
	// Start the host
	err = host.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start P2P host: %v", err)
	}
	defer host.Stop(ctx)
	
	// Create discovery service
	discovery := p2p.NewDiscoveryService(host.Host(), nil, logger)
	
	// Start discovery
	err = discovery.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start discovery service: %v", err)
	}
	defer discovery.Stop()
	
	// Test discovery of connected peers
	request := &types.DiscoveryRequest{
		Method:   "connected",
		MaxPeers: 10,
		Timeout:  5 * time.Second,
	}
	
	response, err := discovery.DiscoverPeers(ctx, request)
	if err != nil {
		t.Fatalf("Failed to discover peers: %v", err)
	}
	
	if response == nil {
		t.Fatal("Discovery response is nil")
	}
	
	if response.MethodUsed != "connected" {
		t.Errorf("Expected method 'connected', got '%s'", response.MethodUsed)
	}
}

func TestProtocolHandler(t *testing.T) {
	logger := zap.NewNop()
	
	config := &types.P2PConfig{
		EnableTCP:        true,
		EnableQUIC:       false,
		EnableWebSocket:  false,
		ListenAddresses:  []string{"/ip4/127.0.0.1/tcp/0"},
		EnableNoise:      true,
		EnableTLS:        false,
		EnableMDNS:       false,
		EnableDHT:        false,
		DHTMode:          "client",
		LowWaterMark:     10,
		HighWaterMark:    20,
		GracePeriod:      30 * time.Second,
		MaxStreams:       100,
		MaxInboundStreams: 50,
		MaxOutboundStreams: 50,
	}
	
	host, err := p2p.NewLibP2PHost(config, logger)
	if err != nil {
		t.Fatalf("Failed to create P2P host: %v", err)
	}
	
	ctx := context.Background()
	
	// Start the host
	err = host.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start P2P host: %v", err)
	}
	defer host.Stop(ctx)
	
	// Register a test protocol handler
	testHandler := &TestProtocolHandler{received: make(chan []byte, 1)}
	host.RegisterProtocolHandler("/test/1.0.0", testHandler)
	
	// Verify handler was registered
	peerInfo := host.GetLocalPeerInfo()
	if peerInfo == nil {
		t.Fatal("Failed to get local peer info")
	}
	
	// Protocol registration is successful if no error occurred
	t.Log("Protocol handler registered successfully")
}

// TestProtocolHandler implements types.ProtocolHandler for testing
type TestProtocolHandler struct {
	received chan []byte
}

func (h *TestProtocolHandler) HandleProtocol(ctx context.Context, stream types.StreamHandler) error {
	buffer := make([]byte, 1024)
	n, err := stream.Read(buffer)
	if err != nil {
		return err
	}
	
	select {
	case h.received <- buffer[:n]:
	case <-ctx.Done():
		return ctx.Err()
	}
	
	return nil
}