// Package p2p implements peer-to-peer networking functionality
package p2p

import (
	"context"
	"sync"
	"time"

	"node/types"
	"go.uber.org/zap"
)

// PeerManager manages peer connections and lifecycle
type PeerManager struct {
	mu            sync.RWMutex
	peers         map[string]*types.PeerInfo
	maxPeers      int
	timeout       time.Duration
	logger        *zap.Logger
	metricsUpdate chan<- types.MetricsUpdate
}

// NewPeerManager creates a new peer manager instance
func NewPeerManager(maxPeers int, timeout time.Duration, logger *zap.Logger) *PeerManager {
	return &PeerManager{
		peers:    make(map[string]*types.PeerInfo),
		maxPeers: maxPeers,
		timeout:  timeout,
		logger:   logger,
	}
}

// SetMetricsChannel sets the channel for metrics updates
func (pm *PeerManager) SetMetricsChannel(ch chan<- types.MetricsUpdate) {
	pm.metricsUpdate = ch
}

// Connect establishes a connection to a peer
func (pm *PeerManager) Connect(ctx context.Context, peerID string, address string) error {
	if peerID == "" {
		return types.ErrInvalidPeerID
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if already connected
	if peer, exists := pm.peers[peerID]; exists && peer.Status == "connected" {
		pm.logger.Debug("Peer already connected",
			zap.String("peerId", peerID),
			zap.String("address", peer.Address))
		return types.ErrPeerAlreadyConnected
	}

	// Check peer limit
	if len(pm.peers) >= pm.maxPeers {
		pm.logger.Warn("Max peers limit reached",
			zap.Int("currentPeers", len(pm.peers)),
			zap.Int("maxPeers", pm.maxPeers))
		return types.ErrMaxPeersReached
	}

	// Create connection with timeout
	connectCtx, cancel := context.WithTimeout(ctx, pm.timeout)
	defer cancel()

	// Simulate connection (in real implementation, use libp2p)
	select {
	case <-connectCtx.Done():
		pm.sendMetricsUpdate(types.MetricsUpdate{ConnectionsFailed: 1})
		return types.NewNetworkError("connect", address, types.ErrConnectionTimeout)
	case <-time.After(100 * time.Millisecond): // Simulate connection time
		// Connection successful
	}

	// Create peer info
	now := time.Now()
	pm.peers[peerID] = &types.PeerInfo{
		ID:          peerID,
		Address:     address,
		Status:      "connected",
		ConnectedAt: now,
		LastSeen:    now,
		Protocols:   []string{"/ipfs/1.0.0", "/blackhole/1.0.0"},
		UserAgent:   "blackhole-node/1.0.0",
	}

	pm.logger.Info("Peer connected",
		zap.String("peerId", peerID),
		zap.String("address", address))

	// Update metrics
	pm.sendMetricsUpdate(types.MetricsUpdate{ConnectionsAdded: 1})

	return nil
}

// Disconnect terminates a peer connection
func (pm *PeerManager) Disconnect(ctx context.Context, peerID string, reason string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	peer, exists := pm.peers[peerID]
	if !exists {
		return types.NewPeerError(peerID, "disconnect", types.ErrPeerNotFound)
	}

	// Update peer status
	peer.Status = "disconnected"
	delete(pm.peers, peerID)

	pm.logger.Info("Peer disconnected",
		zap.String("peerId", peerID),
		zap.String("reason", reason))

	// Update metrics
	pm.sendMetricsUpdate(types.MetricsUpdate{ConnectionsAdded: -1})

	return nil
}

// GetPeer retrieves information about a specific peer
func (pm *PeerManager) GetPeer(peerID string) (*types.PeerInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peer, exists := pm.peers[peerID]
	if !exists {
		return nil, types.NewPeerError(peerID, "get", types.ErrPeerNotFound)
	}

	// Return a copy to prevent external modification
	peerCopy := *peer
	return &peerCopy, nil
}

// ListPeers returns a filtered list of peers
func (pm *PeerManager) ListPeers(filter types.PeerFilter) ([]*types.PeerInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Apply filters
	filtered := make([]*types.PeerInfo, 0)
	for _, peer := range pm.peers {
		if filter.Status == "" || peer.Status == filter.Status {
			peerCopy := *peer
			filtered = append(filtered, &peerCopy)
		}
	}

	// Apply pagination
	start := filter.Offset
	end := start + filter.Limit
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) || filter.Limit <= 0 {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

// GetPeerCount returns the count of active and total peers
func (pm *PeerManager) GetPeerCount() (active int, total int) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	total = len(pm.peers)
	for _, peer := range pm.peers {
		if peer.Status == "connected" {
			active++
		}
	}

	return active, total
}

// CheckHealth checks the health of all peers
func (pm *PeerManager) CheckHealth(timeout time.Duration) map[string]*types.PeerInfo {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	unhealthyPeers := make(map[string]*types.PeerInfo)

	for id, peer := range pm.peers {
		if now.Sub(peer.LastSeen) > timeout {
			peer.Status = "disconnected"
			unhealthyPeers[id] = peer
			pm.logger.Debug("Peer marked as unhealthy",
				zap.String("peerId", id),
				zap.Duration("lastSeenAgo", now.Sub(peer.LastSeen)))
		}
	}

	return unhealthyPeers
}

// UpdatePeerMetrics updates metrics for a specific peer
func (pm *PeerManager) UpdatePeerMetrics(peerID string, bytesRecv, bytesSent, msgsRecv, msgsSent int64) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	peer, exists := pm.peers[peerID]
	if !exists {
		return types.NewPeerError(peerID, "update metrics", types.ErrPeerNotFound)
	}

	peer.BytesReceived += bytesRecv
	peer.BytesSent += bytesSent
	peer.MessagesRecv += msgsRecv
	peer.MessagesSent += msgsSent
	peer.LastSeen = time.Now()

	// Send global metrics update
	pm.sendMetricsUpdate(types.MetricsUpdate{
		BytesReceived:    bytesRecv,
		BytesSent:        bytesSent,
		MessagesReceived: msgsRecv,
		MessagesSent:     msgsSent,
	})

	return nil
}

// DisconnectAll disconnects all peers
func (pm *PeerManager) DisconnectAll() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	count := len(pm.peers)
	for id := range pm.peers {
		delete(pm.peers, id)
	}

	pm.logger.Info("Disconnected all peers", zap.Int("count", count))

	// Update metrics
	if count > 0 {
		pm.sendMetricsUpdate(types.MetricsUpdate{ConnectionsAdded: -int64(count)})
	}
}

// GetPeerMap returns a copy of the peer map
func (pm *PeerManager) GetPeerMap() map[string]*types.PeerInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peersCopy := make(map[string]*types.PeerInfo, len(pm.peers))
	for id, peer := range pm.peers {
		peerCopy := *peer
		peersCopy[id] = &peerCopy
	}

	return peersCopy
}

// sendMetricsUpdate sends metrics update if channel is set
func (pm *PeerManager) sendMetricsUpdate(update types.MetricsUpdate) {
	if pm.metricsUpdate != nil {
		select {
		case pm.metricsUpdate <- update:
		default:
			// Channel full, metrics dropped
			pm.logger.Warn("Metrics channel full, update dropped")
		}
	}
}