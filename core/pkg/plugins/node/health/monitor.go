// Package health implements network health monitoring
package health

import (
	"sync"
	"time"

	"node/types"
	"go.uber.org/zap"
)

// Monitor tracks and reports network health
type Monitor struct {
	mu            sync.RWMutex
	health        *types.NetworkHealth
	config        *types.NodeConfig
	logger        *zap.Logger
	minPeers      int
	lastUpdate    time.Time
}

// NewMonitor creates a new health monitor
func NewMonitor(config *types.NodeConfig, logger *zap.Logger) *Monitor {
	minPeers := 5
	if config.MaxPeers < minPeers {
		minPeers = config.MaxPeers
	}

	return &Monitor{
		config:   config,
		logger:   logger,
		minPeers: minPeers,
		health: &types.NetworkHealth{
			Status:      "healthy",
			HealthScore: 1.0,
			LastUpdated: time.Now(),
		},
	}
}

// GetHealth returns the current network health
func (m *Monitor) GetHealth() *types.NetworkHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	healthCopy := *m.health
	return &healthCopy
}

// UpdateHealth updates the network health state
func (m *Monitor) UpdateHealth(health *types.NetworkHealth) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.health = health
	m.lastUpdate = time.Now()
}

// CheckPeerHealth evaluates peer connectivity health
func (m *Monitor) CheckPeerHealth(peers map[string]*types.PeerInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	activePeers := 0
	totalLatency := time.Duration(0)
	latencyCount := 0

	// Count active peers and calculate average latency
	for _, peer := range peers {
		if peer.Status == "connected" {
			activePeers++
			if peer.Latency > 0 {
				totalLatency += peer.Latency
				latencyCount++
			}
		}
	}

	// Update health metrics
	m.health.ActivePeers = activePeers
	m.health.TotalPeers = len(peers)
	m.health.LastUpdated = now

	// Calculate average latency
	if latencyCount > 0 {
		m.health.AverageLatency = totalLatency / time.Duration(latencyCount)
	}

	// Calculate health score
	m.health.HealthScore = m.CalculateHealthScore()

	// Determine status based on score
	m.updateStatus()

	m.logger.Debug("Health check completed",
		zap.Int("activePeers", activePeers),
		zap.Int("totalPeers", len(peers)),
		zap.Float64("healthScore", m.health.HealthScore),
		zap.String("status", m.health.Status))

	return nil
}

// CalculateHealthScore computes the overall health score
func (m *Monitor) CalculateHealthScore() float64 {
	// Peer connectivity score (40% weight)
	peerScore := m.calculatePeerScore()

	// Network performance score (30% weight)
	performanceScore := m.calculatePerformanceScore()

	// Stability score (30% weight)
	stabilityScore := m.calculateStabilityScore()

	// Combined weighted score
	totalScore := (peerScore * 0.4) + (performanceScore * 0.3) + (stabilityScore * 0.3)

	// Ensure score is between 0 and 1
	if totalScore < 0 {
		totalScore = 0
	} else if totalScore > 1 {
		totalScore = 1
	}

	return totalScore
}

// calculatePeerScore evaluates peer connectivity
func (m *Monitor) calculatePeerScore() float64 {
	if m.minPeers == 0 {
		return 1.0
	}

	peerRatio := float64(m.health.ActivePeers) / float64(m.minPeers)
	if peerRatio > 1.0 {
		peerRatio = 1.0
	}

	return peerRatio
}

// calculatePerformanceScore evaluates network performance
func (m *Monitor) calculatePerformanceScore() float64 {
	score := 1.0

	// Penalize high latency
	if m.health.AverageLatency > 0 {
		// Consider > 500ms as poor performance
		if m.health.AverageLatency > 500*time.Millisecond {
			score -= 0.5
		} else if m.health.AverageLatency > 200*time.Millisecond {
			score -= 0.2
		}
	}

	// Penalize packet loss
	if m.health.PacketLossRate > 0.05 { // > 5% loss
		score -= 0.3
	} else if m.health.PacketLossRate > 0.01 { // > 1% loss
		score -= 0.1
	}

	if score < 0 {
		score = 0
	}

	return score
}

// calculateStabilityScore evaluates network stability
func (m *Monitor) calculateStabilityScore() float64 {
	// For now, base on peer count stability
	// In production, track peer churn rate
	if m.health.TotalPeers == 0 {
		return 0.5 // No peers, but not necessarily unstable
	}

	connectedRatio := float64(m.health.ActivePeers) / float64(m.health.TotalPeers)
	return connectedRatio
}

// updateStatus updates the health status based on score
func (m *Monitor) updateStatus() {
	switch {
	case m.health.HealthScore >= 0.8:
		m.health.Status = "healthy"
	case m.health.HealthScore >= 0.5:
		m.health.Status = "degraded"
	default:
		m.health.Status = "unhealthy"
	}
}

// SetBandwidthUsage updates the bandwidth usage metric
func (m *Monitor) SetBandwidthUsage(bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.health.BandwidthUsage = bytes
}

// SetDiscoveredPeers updates the discovered peers count
func (m *Monitor) SetDiscoveredPeers(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.health.DiscoveredPeers = count
}

// SetPacketLossRate updates the packet loss rate
func (m *Monitor) SetPacketLossRate(rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.health.PacketLossRate = rate
}

// IsHealthy returns true if the network is healthy
func (m *Monitor) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.health.Status == "healthy"
}

// GetStatus returns the current health status
func (m *Monitor) GetStatus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.health.Status
}