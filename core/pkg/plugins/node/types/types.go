// Package types defines the core type definitions and interfaces for the
// node plugin. It provides the contract that implementations must follow
// and ensures consistency across the node networking subsystem.
package types

import (
	"context"
	"time"
)

// Plugin represents the plugin interface that must be implemented
type Plugin interface {
	Info() PluginInfo
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Handle(ctx context.Context, request PluginRequest) (PluginResponse, error)
	HealthCheck() error
	GetStatus() PluginStatus
	PrepareShutdown() error
	ExportState() ([]byte, error)
	ImportState(data []byte) error
}

// PluginInfo contains metadata about the plugin
type PluginInfo struct {
	Name                 string                 `json:"name"`
	Version              string                 `json:"version"`
	Description          string                 `json:"description"`
	Author               string                 `json:"author"`
	License              string                 `json:"license"`
	Homepage             string                 `json:"homepage"`
	Repository           string                 `json:"repository"`
	Status               PluginStatus           `json:"status"`
	LoadTime             time.Time              `json:"loadTime"`
	Uptime               time.Duration          `json:"uptime"`
	Capabilities         []PluginCapability     `json:"capabilities"`
	Permissions          []PluginPermission     `json:"permissions"`
	ResourceRequirements ResourceRequirements   `json:"resourceRequirements"`
}

// PluginStatus represents the current state of a plugin
type PluginStatus string

const (
	PluginStatusUnknown  PluginStatus = "unknown"
	PluginStatusLoaded   PluginStatus = "loaded"
	PluginStatusStarting PluginStatus = "starting"
	PluginStatusRunning  PluginStatus = "running"
	PluginStatusStopping PluginStatus = "stopping"
	PluginStatusStopped  PluginStatus = "stopped"
	PluginStatusError    PluginStatus = "error"
)

// PluginCapability describes a capability provided by the plugin
type PluginCapability struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// PluginPermission describes a permission required by the plugin
type PluginPermission struct {
	Resource    string   `json:"resource"`
	Actions     []string `json:"actions"`
	Description string   `json:"description"`
}

// ResourceRequirements specifies resource requirements for the plugin
type ResourceRequirements struct {
	MinMemoryMB int `json:"minMemoryMB"`
	MaxMemoryMB int `json:"maxMemoryMB"`
	MinCPUMHz   int `json:"minCPUMHz"`
	MaxCPUMHz   int `json:"maxCPUMHz"`
}

// PluginRequest represents a request to the plugin
type PluginRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// PluginResponse represents a response from the plugin
type PluginResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// NodeConfig defines configuration for the node plugin
type NodeConfig struct {
	// Node identification
	NodeID  string `json:"nodeId"`
	Version string `json:"version"`

	// P2P networking
	P2PPort         int      `json:"p2pPort"`
	ListenAddresses []string `json:"listenAddresses"`
	BootstrapPeers  []string `json:"bootstrapPeers"`

	// Discovery
	EnableDiscovery   bool          `json:"enableDiscovery"`
	DiscoveryInterval time.Duration `json:"discoveryInterval"`
	DiscoveryMethod   string        `json:"discoveryMethod"` // mdns, dht, bootstrap

	// Health monitoring
	HealthCheckInterval time.Duration `json:"healthCheckInterval"`
	PeerTimeout         time.Duration `json:"peerTimeout"`

	// Resource limits
	MaxPeers          int           `json:"maxPeers"`
	MaxBandwidthMbps  int           `json:"maxBandwidthMbps"`
	ConnectionTimeout time.Duration `json:"connectionTimeout"`

	// Security
	EnableEncryption bool   `json:"enableEncryption"`
	PrivateKeyPath   string `json:"privateKeyPath"`
}

// PeerInfo represents information about a connected peer
type PeerInfo struct {
	ID          string        `json:"id"`
	Address     string        `json:"address"`
	Status      string        `json:"status"` // connected, connecting, disconnected
	ConnectedAt time.Time     `json:"connectedAt"`
	LastSeen    time.Time     `json:"lastSeen"`
	Latency     time.Duration `json:"latency"`

	// Metrics
	BytesReceived int64 `json:"bytesReceived"`
	BytesSent     int64 `json:"bytesSent"`
	MessagesRecv  int64 `json:"messagesReceived"`
	MessagesSent  int64 `json:"messagesSent"`

	// Capabilities
	Protocols []string `json:"protocols"`
	UserAgent string   `json:"userAgent"`
}

// NetworkHealth represents overall network connectivity health
type NetworkHealth struct {
	Status      string    `json:"status"` // healthy, degraded, unhealthy
	ActivePeers int       `json:"activePeers"`
	TotalPeers  int       `json:"totalPeers"`
	HealthScore float64   `json:"healthScore"` // 0.0 to 1.0
	LastUpdated time.Time `json:"lastUpdated"`

	// Detailed health indicators
	AverageLatency  time.Duration `json:"averageLatency"`
	PacketLossRate  float64       `json:"packetLossRate"`
	BandwidthUsage  int64         `json:"bandwidthUsage"`
	DiscoveredPeers int           `json:"discoveredPeers"`
}

// NetworkMetrics tracks network performance metrics
type NetworkMetrics struct {
	TotalConnections  int64     `json:"totalConnections"`
	ActiveConnections int64     `json:"activeConnections"`
	FailedConnections int64     `json:"failedConnections"`
	BytesReceived     int64     `json:"bytesReceived"`
	BytesSent         int64     `json:"bytesSent"`
	MessagesReceived  int64     `json:"messagesReceived"`
	MessagesSent      int64     `json:"messagesSent"`
	LastReset         time.Time `json:"lastReset"`
}

// PeerManager handles peer connections and management
type PeerManager interface {
	Connect(ctx context.Context, peerID string, address string) error
	Disconnect(ctx context.Context, peerID string, reason string) error
	GetPeer(peerID string) (*PeerInfo, error)
	ListPeers(filter PeerFilter) ([]*PeerInfo, error)
	GetPeerCount() (active int, total int)
}

// PeerFilter defines filtering options for peer listing
type PeerFilter struct {
	Status string
	Limit  int
	Offset int
}

// PeerDiscovery handles peer discovery operations
type PeerDiscovery interface {
	DiscoverPeers(ctx context.Context, method string, maxPeers int) ([]DiscoveredPeer, error)
	StartDiscovery(ctx context.Context) error
	StopDiscovery(ctx context.Context) error
}

// DiscoveredPeer represents a discovered peer
type DiscoveredPeer struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Source  string `json:"source"`
}

// HealthMonitor monitors network health
type HealthMonitor interface {
	GetHealth() *NetworkHealth
	UpdateHealth(health *NetworkHealth)
	CheckPeerHealth(peers map[string]*PeerInfo) error
	CalculateHealthScore() float64
}

// NetworkManager manages overall network operations
type NetworkManager interface {
	GetMetrics() *NetworkMetrics
	UpdateMetrics(update MetricsUpdate)
	GetNetworkStatus() map[string]interface{}
	ValidateBandwidth(current int64) error
}

// MetricsUpdate represents an update to network metrics
type MetricsUpdate struct {
	ConnectionsAdded   int64
	ConnectionsFailed  int64
	BytesReceived      int64
	BytesSent          int64
	MessagesReceived   int64
	MessagesSent       int64
}