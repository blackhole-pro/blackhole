// Package types defines the core type definitions and interfaces for the
// node component. It provides the contract that implementations must follow
// and ensures consistency across the node subsystem.
package types

import (
	"context"
	"time"
	
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// NodeStatus represents the operational status of a node
type NodeStatus string

const (
	NodeStatusStarting   NodeStatus = "starting"
	NodeStatusHealthy    NodeStatus = "healthy"
	NodeStatusDegraded   NodeStatus = "degraded"
	NodeStatusOffline    NodeStatus = "offline"
	NodeStatusShutdown   NodeStatus = "shutdown"
)

// PeerStatus represents the connection status to a peer
type PeerStatus string

const (
	PeerStatusConnected    PeerStatus = "connected"
	PeerStatusConnecting   PeerStatus = "connecting"
	PeerStatusDisconnected PeerStatus = "disconnected"
	PeerStatusFailed       PeerStatus = "failed"
)

// NetworkHealth represents overall network connectivity health
type NetworkHealth string

const (
	NetworkHealthGood     NetworkHealth = "good"
	NetworkHealthDegraded NetworkHealth = "degraded"
	NetworkHealthPoor     NetworkHealth = "poor"
)

// NodeConfig contains configuration for the node service
type NodeConfig struct {
	// Node identification
	NodeID   string `yaml:"node_id"`
	Version  string `yaml:"version"`
	
	// Network configuration
	ListenPort    int      `yaml:"listen_port"`
	ListenAddress string   `yaml:"listen_address"`
	ExternalIP    string   `yaml:"external_ip"`
	
	// P2P configuration
	P2P              P2PConfig     `yaml:"p2p"`
	MaxPeers         int           `yaml:"max_peers"`
	MinPeers         int           `yaml:"min_peers"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
	PingInterval     time.Duration `yaml:"ping_interval"`
	
	// Discovery configuration
	BootstrapPeers   []string      `yaml:"bootstrap_peers"`
	DiscoveryMethods []string      `yaml:"discovery_methods"`
	DHT              DHTConfig     `yaml:"dht"`
	
	// Performance configuration
	BandwidthLimit   int64         `yaml:"bandwidth_limit"`
	MessageQueueSize int           `yaml:"message_queue_size"`
	
	// Security configuration
	EnableTLS        bool          `yaml:"enable_tls"`
	TLSCertPath      string        `yaml:"tls_cert_path"`
	TLSKeyPath       string        `yaml:"tls_key_path"`
}

// DHTConfig contains DHT-specific configuration
type DHTConfig struct {
	Enabled         bool          `yaml:"enabled"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	BucketSize      int           `yaml:"bucket_size"`
}

// Peer represents a connected peer
type Peer struct {
	ID          string            `json:"id"`
	Address     string            `json:"address"`
	Status      PeerStatus        `json:"status"`
	ConnectedAt time.Time         `json:"connected_at"`
	LastSeen    time.Time         `json:"last_seen"`
	BytesSent   int64             `json:"bytes_sent"`
	BytesRecv   int64             `json:"bytes_received"`
	Latency     time.Duration     `json:"latency"`
	Metadata    map[string]string `json:"metadata"`
}

// NodeMetrics contains operational metrics for the node
type NodeMetrics struct {
	// Connection metrics
	TotalConnections  int64     `json:"total_connections"`
	ActiveConnections int64     `json:"active_connections"`
	FailedConnections int64     `json:"failed_connections"`
	
	// Data transfer metrics
	BytesSent     int64     `json:"bytes_sent"`
	BytesReceived int64     `json:"bytes_received"`
	MessagesSent  int64     `json:"messages_sent"`
	MessagesRecv  int64     `json:"messages_received"`
	
	// Performance metrics
	AverageLatency time.Duration `json:"average_latency"`
	PacketLoss     float64       `json:"packet_loss"`
	Uptime         time.Duration `json:"uptime"`
	
	// Resource metrics
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage int64   `json:"memory_usage"`
	
	// Last updated
	LastUpdated time.Time `json:"last_updated"`
}

// NetworkState represents the overall state of the network
type NetworkState struct {
	ConnectedPeers      int             `json:"connected_peers"`
	DiscoveredPeers     int             `json:"discovered_peers"`
	NetworkHealth       NetworkHealth   `json:"network_health"`
	HealthScore         float64         `json:"health_score"`
	AverageLatency      time.Duration   `json:"average_latency"`
	TotalBandwidthUsed  int64          `json:"total_bandwidth_used"`
	LastUpdated         time.Time       `json:"last_updated"`
}

// PeerConnectionRequest represents a request to connect to a peer
type PeerConnectionRequest struct {
	Address     string            `json:"address"`
	Timeout     time.Duration     `json:"timeout"`
	Metadata    map[string]string `json:"metadata"`
}

// PeerConnectionResponse represents the result of a connection attempt
type PeerConnectionResponse struct {
	Success bool   `json:"success"`
	PeerID  string `json:"peer_id,omitempty"`
	Message string `json:"message,omitempty"`
	Error   error  `json:"error,omitempty"`
}

// DiscoveryRequest represents a peer discovery request
type DiscoveryRequest struct {
	Method      string        `json:"method"`       // "dht", "bootstrap", "local"
	MaxPeers    int           `json:"max_peers"`
	Timeout     time.Duration `json:"timeout"`
	FilterFunc  func(string) bool `json:"-"` // Function to filter discovered peers
}

// DiscoveryResponse represents the result of peer discovery
type DiscoveryResponse struct {
	DiscoveredAddresses []string `json:"discovered_addresses"`
	TotalDiscovered     int      `json:"total_discovered"`
	MethodUsed          string   `json:"method_used"`
	Duration            time.Duration `json:"duration"`
}

// P2PHost defines the interface for P2P host operations
type P2PHost interface {
	// Host returns the underlying libp2p host
	Host() host.Host
	
	// Start initializes and starts the P2P host
	Start(ctx context.Context) error
	
	// Stop gracefully shuts down the P2P host
	Stop(ctx context.Context) error
	
	// Connect establishes a connection to a peer
	Connect(ctx context.Context, addr peer.AddrInfo) error
	
	// Disconnect closes connection to a peer
	Disconnect(ctx context.Context, peerID peer.ID) error
	
	// GetPeers returns list of connected peers
	GetPeers() []peer.ID
	
	// RegisterProtocolHandler registers a handler for a protocol
	RegisterProtocolHandler(protocolID protocol.ID, handler ProtocolHandler)
	
	// SendMessage sends a message to a peer using specified protocol
	SendMessage(ctx context.Context, peerID peer.ID, protocolID protocol.ID, data []byte) error
	
	// GetLocalPeerInfo returns information about the local peer
	GetLocalPeerInfo() *LocalPeerInfo
}

// ProtocolHandler defines the interface for handling protocol messages
type ProtocolHandler interface {
	HandleProtocol(ctx context.Context, stream StreamHandler) error
}

// StreamHandler defines the interface for handling streams
type StreamHandler interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	Protocol() protocol.ID
	RemotePeer() peer.ID
}

// LocalPeerInfo contains information about the local peer
type LocalPeerInfo struct {
	PeerID    peer.ID    `json:"peer_id"`
	Addresses []string   `json:"addresses"`
	Protocols []string   `json:"protocols"`
}

// P2PConfig contains P2P-specific configuration
type P2PConfig struct {
	// Identity configuration
	PrivateKeyPath string `yaml:"private_key_path"`
	
	// Transport configuration
	EnableTCP       bool     `yaml:"enable_tcp"`
	EnableQUIC      bool     `yaml:"enable_quic"`
	EnableWebSocket bool     `yaml:"enable_websocket"`
	ListenAddresses []string `yaml:"listen_addresses"`
	
	// Security configuration
	EnableNoise  bool `yaml:"enable_noise"`
	EnableTLS    bool `yaml:"enable_tls"`
	
	// Discovery configuration
	EnableMDNS       bool          `yaml:"enable_mdns"`
	MDNSInterval     time.Duration `yaml:"mdns_interval"`
	EnableDHT        bool          `yaml:"enable_dht"`
	DHTMode          string        `yaml:"dht_mode"` // "client", "server", "auto"
	BootstrapPeers   []string      `yaml:"bootstrap_peers"`
	
	// Connection management
	LowWaterMark  int `yaml:"low_water_mark"`
	HighWaterMark int `yaml:"high_water_mark"`
	GracePeriod   time.Duration `yaml:"grace_period"`
	
	// Resource management
	MaxStreams         int   `yaml:"max_streams"`
	MaxInboundStreams  int   `yaml:"max_inbound_streams"`
	MaxOutboundStreams int   `yaml:"max_outbound_streams"`
	MaxMemory          int64 `yaml:"max_memory"`
}