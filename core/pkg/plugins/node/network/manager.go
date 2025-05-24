// Package network implements overall network management
package network

import (
	"fmt"
	"sync"
	"time"

	"node/types"
	"go.uber.org/zap"
)

// Manager manages network metrics and operations
type Manager struct {
	mu           sync.RWMutex
	metrics      *types.NetworkMetrics
	config       *types.NodeConfig
	logger       *zap.Logger
	maxBandwidth int64 // bytes per second
}

// NewManager creates a new network manager
func NewManager(config *types.NodeConfig, logger *zap.Logger) *Manager {
	maxBandwidth := int64(config.MaxBandwidthMbps) * 1024 * 1024 / 8
	
	return &Manager{
		config:       config,
		logger:       logger,
		maxBandwidth: maxBandwidth,
		metrics: &types.NetworkMetrics{
			LastReset: time.Now(),
		},
	}
}

// GetMetrics returns current network metrics
func (m *Manager) GetMetrics() *types.NetworkMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	metricsCopy := *m.metrics
	return &metricsCopy
}

// UpdateMetrics applies a metrics update
func (m *Manager) UpdateMetrics(update types.MetricsUpdate) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update connection counts
	if update.ConnectionsAdded != 0 {
		m.metrics.ActiveConnections += update.ConnectionsAdded
		if update.ConnectionsAdded > 0 {
			m.metrics.TotalConnections += update.ConnectionsAdded
		}
	}

	if update.ConnectionsFailed > 0 {
		m.metrics.FailedConnections += update.ConnectionsFailed
	}

	// Update data transfer metrics
	m.metrics.BytesReceived += update.BytesReceived
	m.metrics.BytesSent += update.BytesSent
	m.metrics.MessagesReceived += update.MessagesReceived
	m.metrics.MessagesSent += update.MessagesSent

	m.logger.Debug("Metrics updated",
		zap.Int64("activeConnections", m.metrics.ActiveConnections),
		zap.Int64("bytesReceived", update.BytesReceived),
		zap.Int64("bytesSent", update.BytesSent))
}

// GetNetworkStatus returns comprehensive network status
func (m *Manager) GetNetworkStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.metrics.LastReset)
	
	// Calculate bandwidth usage
	totalBytes := m.metrics.BytesReceived + m.metrics.BytesSent
	bandwidthUsage := float64(0)
	if uptime.Seconds() > 0 {
		bytesPerSecond := float64(totalBytes) / uptime.Seconds()
		if m.maxBandwidth > 0 {
			bandwidthUsage = (bytesPerSecond / float64(m.maxBandwidth)) * 100
		}
	}

	return map[string]interface{}{
		"metrics": m.metrics,
		"limits": map[string]interface{}{
			"maxPeers":      m.config.MaxPeers,
			"maxBandwidth":  m.maxBandwidth,
			"bandwidthMbps": m.config.MaxBandwidthMbps,
		},
		"usage": map[string]interface{}{
			"bandwidthPercent": bandwidthUsage,
			"uptime":           uptime.String(),
		},
		"rates": m.calculateRates(uptime),
	}
}

// ValidateBandwidth checks if current bandwidth usage is within limits
func (m *Manager) ValidateBandwidth(additionalBytes int64) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.maxBandwidth <= 0 {
		// No bandwidth limit
		return nil
	}

	uptime := time.Since(m.metrics.LastReset)
	if uptime.Seconds() == 0 {
		return nil
	}

	// Calculate current rate
	totalBytes := m.metrics.BytesReceived + m.metrics.BytesSent + additionalBytes
	bytesPerSecond := float64(totalBytes) / uptime.Seconds()

	if bytesPerSecond > float64(m.maxBandwidth) {
		return fmt.Errorf("%w: current %.2f MB/s exceeds limit %.2f MB/s",
			types.ErrBandwidthExceeded,
			bytesPerSecond/1024/1024,
			float64(m.maxBandwidth)/1024/1024)
	}

	return nil
}

// ResetMetrics resets all metrics
func (m *Manager) ResetMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics = &types.NetworkMetrics{
		LastReset: time.Now(),
	}

	m.logger.Info("Network metrics reset")
}

// calculateRates calculates data transfer rates
func (m *Manager) calculateRates(uptime time.Duration) map[string]float64 {
	if uptime.Seconds() == 0 {
		return map[string]float64{
			"bytesPerSecond":    0,
			"messagesPerSecond": 0,
			"mbpsIn":            0,
			"mbpsOut":           0,
		}
	}

	seconds := uptime.Seconds()
	bytesPerSecond := float64(m.metrics.BytesReceived+m.metrics.BytesSent) / seconds
	messagesPerSecond := float64(m.metrics.MessagesReceived+m.metrics.MessagesSent) / seconds
	mbpsIn := (float64(m.metrics.BytesReceived) / seconds) * 8 / 1024 / 1024
	mbpsOut := (float64(m.metrics.BytesSent) / seconds) * 8 / 1024 / 1024

	return map[string]float64{
		"bytesPerSecond":    bytesPerSecond,
		"messagesPerSecond": messagesPerSecond,
		"mbpsIn":            mbpsIn,
		"mbpsOut":           mbpsOut,
	}
}

// GetConnectionStats returns connection statistics
func (m *Manager) GetConnectionStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	successRate := float64(0)
	if m.metrics.TotalConnections > 0 {
		successful := m.metrics.TotalConnections - m.metrics.FailedConnections
		successRate = float64(successful) / float64(m.metrics.TotalConnections) * 100
	}

	return map[string]interface{}{
		"total":       m.metrics.TotalConnections,
		"active":      m.metrics.ActiveConnections,
		"failed":      m.metrics.FailedConnections,
		"successRate": successRate,
	}
}

// ValidateResourceUsage checks if resource usage is within limits
func (m *Manager) ValidateResourceUsage() error {
	// Check bandwidth
	if err := m.ValidateBandwidth(0); err != nil {
		return err
	}

	// Additional resource checks can be added here
	// - CPU usage
	// - Memory usage
	// - Disk I/O

	return nil
}