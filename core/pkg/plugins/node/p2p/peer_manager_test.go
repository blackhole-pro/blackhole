package p2p_test

import (
	"context"
	"testing"
	"time"

	"node/p2p"
	"node/types"
	"go.uber.org/zap/zaptest"
)

func TestPeerManager_Connect(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := p2p.NewPeerManager(5, 30*time.Second, logger)

	ctx := context.Background()

	// Test successful connection
	err := pm.Connect(ctx, "peer1", "192.168.1.100:4001")
	if err != nil {
		t.Errorf("Failed to connect peer: %v", err)
	}

	// Test already connected
	err = pm.Connect(ctx, "peer1", "192.168.1.100:4001")
	if err != types.ErrPeerAlreadyConnected {
		t.Errorf("Expected ErrPeerAlreadyConnected, got: %v", err)
	}

	// Test max peers limit
	for i := 2; i <= 5; i++ {
		err = pm.Connect(ctx, string(rune('0'+i)), "192.168.1.100:4001")
		if err != nil {
			t.Errorf("Failed to connect peer %d: %v", i, err)
		}
	}

	// Should fail on 6th peer
	err = pm.Connect(ctx, "peer6", "192.168.1.100:4001")
	if err != types.ErrMaxPeersReached {
		t.Errorf("Expected ErrMaxPeersReached, got: %v", err)
	}

	// Test invalid peer ID
	err = pm.Connect(ctx, "", "192.168.1.100:4001")
	if err != types.ErrInvalidPeerID {
		t.Errorf("Expected ErrInvalidPeerID, got: %v", err)
	}
}

func TestPeerManager_Disconnect(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := p2p.NewPeerManager(5, 30*time.Second, logger)

	ctx := context.Background()

	// Connect a peer first
	err := pm.Connect(ctx, "peer1", "192.168.1.100:4001")
	if err != nil {
		t.Fatalf("Failed to connect peer: %v", err)
	}

	// Test successful disconnect
	err = pm.Disconnect(ctx, "peer1", "test")
	if err != nil {
		t.Errorf("Failed to disconnect peer: %v", err)
	}

	// Test disconnect non-existent peer
	err = pm.Disconnect(ctx, "peer1", "test")
	if err == nil {
		t.Error("Expected error for non-existent peer")
	}
}

func TestPeerManager_GetPeer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := p2p.NewPeerManager(5, 30*time.Second, logger)

	ctx := context.Background()

	// Connect a peer
	err := pm.Connect(ctx, "peer1", "192.168.1.100:4001")
	if err != nil {
		t.Fatalf("Failed to connect peer: %v", err)
	}

	// Get existing peer
	peer, err := pm.GetPeer("peer1")
	if err != nil {
		t.Errorf("Failed to get peer: %v", err)
	}
	if peer.ID != "peer1" {
		t.Errorf("Expected peer ID 'peer1', got: %s", peer.ID)
	}
	if peer.Status != "connected" {
		t.Errorf("Expected status 'connected', got: %s", peer.Status)
	}

	// Get non-existent peer
	_, err = pm.GetPeer("peer2")
	if err == nil {
		t.Error("Expected error for non-existent peer")
	}
}

func TestPeerManager_ListPeers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := p2p.NewPeerManager(10, 30*time.Second, logger)

	ctx := context.Background()

	// Connect multiple peers
	for i := 1; i <= 5; i++ {
		peerID := string(rune('0' + i))
		err := pm.Connect(ctx, peerID, "192.168.1.100:4001")
		if err != nil {
			t.Fatalf("Failed to connect peer %s: %v", peerID, err)
		}
	}

	// Test listing all peers
	filter := types.PeerFilter{
		Limit: 10,
	}
	peers, err := pm.ListPeers(filter)
	if err != nil {
		t.Errorf("Failed to list peers: %v", err)
	}
	if len(peers) != 5 {
		t.Errorf("Expected 5 peers, got: %d", len(peers))
	}

	// Test pagination
	filter = types.PeerFilter{
		Limit:  2,
		Offset: 0,
	}
	peers, err = pm.ListPeers(filter)
	if err != nil {
		t.Errorf("Failed to list peers with pagination: %v", err)
	}
	if len(peers) != 2 {
		t.Errorf("Expected 2 peers with limit, got: %d", len(peers))
	}

	// Test offset
	filter = types.PeerFilter{
		Limit:  2,
		Offset: 3,
	}
	peers, err = pm.ListPeers(filter)
	if err != nil {
		t.Errorf("Failed to list peers with offset: %v", err)
	}
	if len(peers) != 2 {
		t.Errorf("Expected 2 peers with offset, got: %d", len(peers))
	}

	// Test status filter
	filter = types.PeerFilter{
		Status: "disconnected",
		Limit:  10,
	}
	peers, err = pm.ListPeers(filter)
	if err != nil {
		t.Errorf("Failed to list peers with status filter: %v", err)
	}
	if len(peers) != 0 {
		t.Errorf("Expected 0 disconnected peers, got: %d", len(peers))
	}
}

func TestPeerManager_GetPeerCount(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := p2p.NewPeerManager(10, 30*time.Second, logger)

	ctx := context.Background()

	// Initial count
	active, total := pm.GetPeerCount()
	if active != 0 || total != 0 {
		t.Errorf("Expected 0 peers initially, got active=%d, total=%d", active, total)
	}

	// Connect peers
	for i := 1; i <= 3; i++ {
		peerID := string(rune('0' + i))
		err := pm.Connect(ctx, peerID, "192.168.1.100:4001")
		if err != nil {
			t.Fatalf("Failed to connect peer: %v", err)
		}
	}

	active, total = pm.GetPeerCount()
	if active != 3 || total != 3 {
		t.Errorf("Expected 3 peers, got active=%d, total=%d", active, total)
	}
}

func TestPeerManager_CheckHealth(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := p2p.NewPeerManager(10, 30*time.Second, logger)

	ctx := context.Background()

	// Connect a peer
	err := pm.Connect(ctx, "peer1", "192.168.1.100:4001")
	if err != nil {
		t.Fatalf("Failed to connect peer: %v", err)
	}

	// Check health with short timeout
	unhealthy := pm.CheckHealth(100 * time.Millisecond)
	if len(unhealthy) != 0 {
		t.Errorf("Expected 0 unhealthy peers initially, got: %d", len(unhealthy))
	}

	// Wait and check again
	time.Sleep(200 * time.Millisecond)
	unhealthy = pm.CheckHealth(100 * time.Millisecond)
	if len(unhealthy) != 1 {
		t.Errorf("Expected 1 unhealthy peer after timeout, got: %d", len(unhealthy))
	}
}

func TestPeerManager_UpdatePeerMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := p2p.NewPeerManager(10, 30*time.Second, logger)

	// Set up metrics channel
	metricsCh := make(chan types.MetricsUpdate, 10)
	pm.SetMetricsChannel(metricsCh)

	ctx := context.Background()

	// Connect a peer
	err := pm.Connect(ctx, "peer1", "192.168.1.100:4001")
	if err != nil {
		t.Fatalf("Failed to connect peer: %v", err)
	}

	// Drain any initial metrics from connection
	select {
	case <-metricsCh:
		// Ignore initial connection metric
	default:
	}

	// Update metrics
	err = pm.UpdatePeerMetrics("peer1", 1000, 500, 10, 5)
	if err != nil {
		t.Errorf("Failed to update peer metrics: %v", err)
	}

	// Check that metrics were sent
	select {
	case update := <-metricsCh:
		if update.BytesReceived != 1000 {
			t.Errorf("Expected BytesReceived=1000, got: %d", update.BytesReceived)
		}
		if update.BytesSent != 500 {
			t.Errorf("Expected BytesSent=500, got: %d", update.BytesSent)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for metrics update")
	}

	// Update metrics for non-existent peer
	err = pm.UpdatePeerMetrics("peer2", 1000, 500, 10, 5)
	if err == nil {
		t.Error("Expected error for non-existent peer")
	}
}

func TestPeerManager_DisconnectAll(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := p2p.NewPeerManager(10, 30*time.Second, logger)

	ctx := context.Background()

	// Connect multiple peers
	for i := 1; i <= 5; i++ {
		peerID := string(rune('0' + i))
		err := pm.Connect(ctx, peerID, "192.168.1.100:4001")
		if err != nil {
			t.Fatalf("Failed to connect peer: %v", err)
		}
	}

	// Verify peers are connected
	active, total := pm.GetPeerCount()
	if active != 5 {
		t.Errorf("Expected 5 active peers before disconnect, got: %d", active)
	}

	// Disconnect all
	pm.DisconnectAll()

	// Verify all disconnected
	active, total = pm.GetPeerCount()
	if active != 0 || total != 0 {
		t.Errorf("Expected 0 peers after DisconnectAll, got active=%d, total=%d", active, total)
	}
}

func TestPeerManager_ConcurrentOperations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pm := p2p.NewPeerManager(100, 30*time.Second, logger)

	ctx := context.Background()
	done := make(chan bool)

	// Concurrent connects
	go func() {
		for i := 0; i < 10; i++ {
			peerID := string(rune('A' + i))
			pm.Connect(ctx, peerID, "192.168.1.100:4001")
		}
		done <- true
	}()

	// Concurrent disconnects
	go func() {
		for i := 0; i < 10; i++ {
			peerID := string(rune('A' + i))
			pm.Disconnect(ctx, peerID, "test")
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 20; i++ {
			pm.GetPeerCount()
			pm.ListPeers(types.PeerFilter{Limit: 10})
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should not panic or deadlock
}