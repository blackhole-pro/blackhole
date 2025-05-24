// Package main implements the node plugin for Blackhole Foundation.
// This plugin handles P2P networking, peer discovery, and network health monitoring.
// It does NOT handle identity, storage, or any other service responsibilities.
//
// Scope:
// - P2P Networking: Manages peer-to-peer connections using libp2p
// - Peer Discovery: Discovers and maintains connections with network peers
// - Network Health Monitoring: Monitors network connectivity and peer health
//
// NOT in scope:
// - Identity management (handled by identity plugin)
// - Data storage (handled by storage plugin)
// - Content routing (handled by indexer plugin)
// - Economic transactions (handled by ledger plugin)
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
)

// NodePlugin implements the P2P networking and peer management functionality
// It adheres to the Plugin interface defined in the framework
type NodePlugin struct {
	// Configuration
	config NodeConfig
	
	// Plugin state
	mu            sync.RWMutex
	status        plugins.PluginStatus
	startTime     time.Time
	healthStatus  string
	version       string
	
	// Network components (in real implementation, these would be libp2p components)
	// host          host.Host          // libp2p host
	// dht           *dht.IpfsDHT       // DHT for peer discovery
	// pubsub        *pubsub.PubSub     // PubSub for message broadcasting
	
	// Network state
	peers         map[string]*PeerInfo
	networkHealth NetworkHealth
	metrics       NetworkMetrics
	
	// Channels
	shutdownCh    chan struct{}
	
	// Background workers
	workerWg      sync.WaitGroup
	
	// Resource limits
	maxPeers      int
	maxBandwidth  int64 // bytes per second
}

// NodeConfig defines configuration for the node plugin
type NodeConfig struct {
	// Node identification
	NodeID      string   `json:"nodeId"`
	Version     string   `json:"version"`
	
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
	MaxPeers            int   `json:"maxPeers"`
	MaxBandwidthMbps    int   `json:"maxBandwidthMbps"`
	ConnectionTimeout   time.Duration `json:"connectionTimeout"`
	
	// Security
	EnableEncryption    bool   `json:"enableEncryption"`
	PrivateKeyPath      string `json:"privateKeyPath"`
}

// PeerInfo represents information about a connected peer
type PeerInfo struct {
	ID            string    `json:"id"`
	Address       string    `json:"address"`
	Status        string    `json:"status"` // connected, connecting, disconnected
	ConnectedAt   time.Time `json:"connectedAt"`
	LastSeen      time.Time `json:"lastSeen"`
	Latency       time.Duration `json:"latency"`
	
	// Metrics
	BytesReceived int64 `json:"bytesReceived"`
	BytesSent     int64 `json:"bytesSent"`
	MessagesRecv  int64 `json:"messagesReceived"`
	MessagesSent  int64 `json:"messagesSent"`
	
	// Capabilities
	Protocols     []string `json:"protocols"`
	UserAgent     string   `json:"userAgent"`
}

// NetworkHealth represents overall network connectivity health
type NetworkHealth struct {
	Status        string    `json:"status"` // healthy, degraded, unhealthy
	ActivePeers   int       `json:"activePeers"`
	TotalPeers    int       `json:"totalPeers"`
	HealthScore   float64   `json:"healthScore"` // 0.0 to 1.0
	LastUpdated   time.Time `json:"lastUpdated"`
	
	// Detailed health indicators
	AverageLatency    time.Duration `json:"averageLatency"`
	PacketLossRate    float64       `json:"packetLossRate"`
	BandwidthUsage    int64         `json:"bandwidthUsage"`
	DiscoveredPeers   int           `json:"discoveredPeers"`
}

// NetworkMetrics tracks network performance metrics
type NetworkMetrics struct {
	TotalConnections  int64         `json:"totalConnections"`
	ActiveConnections int64         `json:"activeConnections"`
	FailedConnections int64         `json:"failedConnections"`
	BytesReceived     int64         `json:"bytesReceived"`
	BytesSent         int64         `json:"bytesSent"`
	MessagesReceived  int64         `json:"messagesReceived"`
	MessagesSent      int64         `json:"messagesSent"`
	LastReset         time.Time     `json:"lastReset"`
}

// Info returns plugin metadata
func (p *NodePlugin) Info() plugins.PluginInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return plugins.PluginInfo{
		Name:        "node",
		Version:     "1.0.0",
		Description: "P2P networking and peer management plugin",
		Author:      "Blackhole Team",
		License:     "Apache-2.0",
		Homepage:    "https://blackhole.foundation/plugins/node",
		Repository:  "https://github.com/blackhole/plugins/node",
		Status:      p.status,
		LoadTime:    p.startTime,
		Uptime:      time.Since(p.startTime),
		Capabilities: []plugins.PluginCapability{
			{
				Name:        "p2p-networking",
				Version:     "1.0",
				Description: "Peer-to-peer networking using libp2p",
			},
			{
				Name:        "peer-discovery",
				Version:     "1.0",
				Description: "Automatic peer discovery via mDNS, DHT, and bootstrap nodes",
			},
			{
				Name:        "network-health",
				Version:     "1.0",
				Description: "Network connectivity and health monitoring",
			},
		},
		Permissions: []plugins.PluginPermission{
			{
				Resource:    "network",
				Actions:     []string{"connect", "listen", "broadcast"},
				Description: "Network access for P2P communication",
			},
			{
				Resource:    "peers",
				Actions:     []string{"read", "write", "delete"},
				Description: "Manage peer connections",
			},
			{
				Resource:    "system.network",
				Actions:     []string{"read"},
				Description: "Read network interface information",
			},
		},
		ResourceRequirements: plugins.ResourceRequirements{
			MinMemoryMB: 128,
			MaxMemoryMB: 512,
			MinCPUMHz:   100,
			MaxCPUMHz:   1000,
		},
	}
}

// Start initializes and starts the plugin
func (p *NodePlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.status == plugins.PluginStatusRunning {
		return fmt.Errorf("plugin already running")
	}
	
	log.Printf("[Node Plugin] Starting with config: %+v", p.config)
	
	p.status = plugins.PluginStatusStarting
	p.startTime = time.Now()
	p.shutdownCh = make(chan struct{})
	// Initialize state
	p.peers = make(map[string]*PeerInfo)
	p.networkHealth = NetworkHealth{
		Status: "healthy",
		HealthScore: 1.0,
		LastUpdated: time.Now(),
	}
	p.metrics = NetworkMetrics{
		LastReset: time.Now(),
	}
	p.maxPeers = p.config.MaxPeers
	if p.maxPeers <= 0 {
		p.maxPeers = 50 // Default max peers
	}
	p.maxBandwidth = int64(p.config.MaxBandwidthMbps) * 1024 * 1024 / 8 // Convert Mbps to bytes/sec
	
	// Initialize P2P components (in real implementation)
	// This would include:
	// - Creating libp2p host with the configured addresses
	// - Setting up DHT for peer discovery
	// - Initializing PubSub for message broadcasting
	// - Setting up protocol handlers
	
	// Start background workers
	p.startBackgroundWorkers()
	
	// Connect to bootstrap peers if configured
	if len(p.config.BootstrapPeers) > 0 {
		go p.connectToBootstrapPeers()
	}
	
	p.status = plugins.PluginStatusRunning
	p.healthStatus = "healthy"
	
	log.Printf("[Node Plugin] Started successfully")
	return nil
}

// Stop gracefully shuts down the plugin
func (p *NodePlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.status != plugins.PluginStatusRunning {
		return fmt.Errorf("plugin not running")
	}
	
	log.Printf("[Node Plugin] Stopping...")
	
	p.status = plugins.PluginStatusStopping
	
	// Signal shutdown
	close(p.shutdownCh)
	
	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		p.workerWg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// Workers stopped gracefully
	case <-time.After(10 * time.Second):
		log.Printf("[Node Plugin] Warning: workers did not stop within timeout")
	}
	
	// Disconnect all peers
	p.disconnectAllPeers()
	
	p.status = plugins.PluginStatusStopped
	log.Printf("[Node Plugin] Stopped")
	return nil
}

// Handle processes plugin requests
func (p *NodePlugin) Handle(ctx context.Context, request plugins.PluginRequest) (plugins.PluginResponse, error) {
	p.mu.RLock()
	if p.status != plugins.PluginStatusRunning {
		p.mu.RUnlock()
		return plugins.PluginResponse{
			Success: false,
			Error:   "plugin not running",
		}, nil
	}
	p.mu.RUnlock()
	
	switch request.Method {
	case "listPeers":
		return p.handleListPeers(ctx, request)
	case "connectPeer":
		return p.handleConnectPeer(ctx, request)
	case "disconnectPeer":
		return p.handleDisconnectPeer(ctx, request)
	case "getNetworkStatus":
		return p.handleGetNetworkStatus(ctx, request)
	case "discoverPeers":
		return p.handleDiscoverPeers(ctx, request)
	default:
		return plugins.PluginResponse{
			Success: false,
			Error:   fmt.Sprintf("unknown method: %s", request.Method),
		}, nil
	}
}

// HealthCheck returns the health status of the plugin
func (p *NodePlugin) HealthCheck() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.status != plugins.PluginStatusRunning {
		return fmt.Errorf("plugin not running")
	}
	
	if p.healthStatus != "healthy" {
		return fmt.Errorf("plugin unhealthy: %s", p.healthStatus)
	}
	
	return nil
}

// GetStatus returns the current plugin status
func (p *NodePlugin) GetStatus() plugins.PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// PrepareShutdown prepares the plugin for shutdown
func (p *NodePlugin) PrepareShutdown() error {
	log.Printf("[Node Plugin] Preparing for shutdown")
	// Save any pending state
	return nil
}

// ExportState exports the plugin state
func (p *NodePlugin) ExportState() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	state := struct {
		Config        NodeConfig    `json:"config"`
		Peers         map[string]*PeerInfo `json:"peers"`
		NetworkHealth NetworkHealth `json:"networkHealth"`
	}{
		Config:        p.config,
		Peers:         p.peers,
		NetworkHealth: p.networkHealth,
	}
	
	return json.Marshal(state)
}

// ImportState imports plugin state
func (p *NodePlugin) ImportState(data []byte) error {
	var state struct {
		Config        NodeConfig    `json:"config"`
		Peers         map[string]*PeerInfo `json:"peers"`
		NetworkHealth NetworkHealth `json:"networkHealth"`
	}
	
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Only import peer list and network health
	// Config should come from initialization
	p.peers = state.Peers
	p.networkHealth = state.NetworkHealth
	
	return nil
}

// Background workers

func (p *NodePlugin) startBackgroundWorkers() {
	// Health monitor
	if p.config.HealthCheckInterval > 0 {
		p.workerWg.Add(1)
		go p.healthMonitor()
	}
	
	// Peer discovery
	if p.config.EnableDiscovery && p.config.DiscoveryInterval > 0 {
		p.workerWg.Add(1)
		go p.discoveryWorker()
	}
	
	// Network state updater
	p.workerWg.Add(1)
	go p.networkStateUpdater()
}

func (p *NodePlugin) healthMonitor() {
	defer p.workerWg.Done()
	
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.checkPeerHealth()
		case <-p.shutdownCh:
			return
		}
	}
}

func (p *NodePlugin) discoveryWorker() {
	defer p.workerWg.Done()
	
	ticker := time.NewTicker(p.config.DiscoveryInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.discoverNewPeers()
		case <-p.shutdownCh:
			return
		}
	}
}

func (p *NodePlugin) networkStateUpdater() {
	defer p.workerWg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.updateNetworkState()
		case <-p.shutdownCh:
			return
		}
	}
}

// Helper methods

func (p *NodePlugin) connectToBootstrapPeers() {
	for _, peer := range p.config.BootstrapPeers {
		// Simulate connection (in real implementation, use libp2p)
		p.mu.Lock()
		p.peers[peer] = &PeerInfo{
			ID:          peer,
			Address:     peer,
			Status:      "connected",
			ConnectedAt: time.Now(),
			LastSeen:    time.Now(),
		}
		p.mu.Unlock()
		
		log.Printf("[Node Plugin] Connected to bootstrap peer: %s", peer)
	}
}

func (p *NodePlugin) disconnectAllPeers() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for id := range p.peers {
		delete(p.peers, id)
		log.Printf("[Node Plugin] Disconnected from peer: %s", id)
	}
}

func (p *NodePlugin) checkPeerHealth() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	now := time.Now()
	for id, peer := range p.peers {
		if now.Sub(peer.LastSeen) > p.config.PeerTimeout {
			peer.Status = "disconnected"
			log.Printf("[Node Plugin] Peer %s marked as disconnected (timeout)", id)
		}
	}
}

func (p *NodePlugin) discoverNewPeers() {
	// Simulate peer discovery
	log.Printf("[Node Plugin] Running peer discovery...")
}

func (p *NodePlugin) updateNetworkState() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	activePeers := 0
	for _, peer := range p.peers {
		if peer.Status == "connected" {
			activePeers++
		}
	}
	
	p.networkHealth.ActivePeers = activePeers
	p.networkHealth.TotalPeers = len(p.peers)
	p.networkHealth.LastUpdated = time.Now()
	
	// Calculate health score based on multiple factors
	minDesiredPeers := 5
	if p.maxPeers < minDesiredPeers {
		minDesiredPeers = p.maxPeers
	}
	
	// Base score on active peer ratio
	peerScore := float64(activePeers) / float64(minDesiredPeers)
	if peerScore > 1.0 {
		peerScore = 1.0
	}
	
	// Consider connection success rate
	var connectionScore float64 = 1.0
	totalAttempts := p.metrics.TotalConnections
	if totalAttempts > 0 {
		connectionScore = float64(p.metrics.ActiveConnections) / float64(totalAttempts)
	}
	
	// Combined health score
	p.networkHealth.HealthScore = (peerScore * 0.7) + (connectionScore * 0.3)
	
	// Determine status based on score
	if p.networkHealth.HealthScore >= 0.8 {
		p.networkHealth.Status = "healthy"
	} else if p.networkHealth.HealthScore >= 0.5 {
		p.networkHealth.Status = "degraded"
	} else {
		p.networkHealth.Status = "unhealthy"
	}
	
	// Update health status for plugin
	if p.networkHealth.Status == "unhealthy" {
		p.healthStatus = "unhealthy: poor network connectivity"
	} else {
		p.healthStatus = "healthy"
	}
}

// Request handlers

func (p *NodePlugin) handleListPeers(ctx context.Context, request plugins.PluginRequest) (plugins.PluginResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Parse filter parameters
	statusFilter, _ := request.Params["status"].(string)
	limit, _ := request.Params["limit"].(float64) // JSON numbers are float64
	offset, _ := request.Params["offset"].(float64)
	
	if limit <= 0 {
		limit = 50
	}
	
	// Filter and paginate peers
	filtered := make([]*PeerInfo, 0)
	for _, peer := range p.peers {
		if statusFilter == "" || peer.Status == statusFilter {
			filtered = append(filtered, peer)
		}
	}
	
	// Apply pagination
	start := int(offset)
	end := start + int(limit)
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}
	
	peers := filtered[start:end]
	
	return plugins.PluginResponse{
		Success: true,
		Data: map[string]interface{}{
			"peers":      peers,
			"count":      len(peers),
			"totalCount": len(filtered),
			"offset":     start,
			"limit":      int(limit),
		},
	}, nil
}

func (p *NodePlugin) handleConnectPeer(ctx context.Context, request plugins.PluginRequest) (plugins.PluginResponse, error) {
	peerID, ok := request.Params["peerId"].(string)
	if !ok || peerID == "" {
		return plugins.PluginResponse{
			Success: false,
			Error:   "peerId is required",
		}, nil
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Check if already connected
	if peer, exists := p.peers[peerID]; exists && peer.Status == "connected" {
		return plugins.PluginResponse{
			Success: true,
			Data: map[string]interface{}{
				"message": "already connected",
				"peer":    peer,
			},
		}, nil
	}
	
	// Check resource limits
	if len(p.peers) >= p.maxPeers {
		return plugins.PluginResponse{
			Success: false,
			Error:   fmt.Sprintf("max peers limit reached (%d)", p.maxPeers),
		}, nil
	}
	
	// Get peer address
	peerAddress, _ := request.Params["address"].(string)
	if peerAddress == "" {
		peerAddress = peerID // Fallback to ID as address
	}
	
	// Simulate connection (in real implementation, use libp2p)
	p.peers[peerID] = &PeerInfo{
		ID:          peerID,
		Address:     peerAddress,
		Status:      "connected",
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
		Protocols:   []string{"/ipfs/1.0.0", "/blackhole/1.0.0"},
		UserAgent:   "blackhole-node/1.0.0",
	}
	
	// Update metrics
	p.metrics.TotalConnections++
	p.metrics.ActiveConnections++
	
	return plugins.PluginResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "peer connected",
			"peerId":  peerID,
		},
	}, nil
}

func (p *NodePlugin) handleDisconnectPeer(ctx context.Context, request plugins.PluginRequest) (plugins.PluginResponse, error) {
	peerID, ok := request.Params["peerId"].(string)
	if !ok || peerID == "" {
		return plugins.PluginResponse{
			Success: false,
			Error:   "peerId is required",
		}, nil
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if _, exists := p.peers[peerID]; !exists {
		return plugins.PluginResponse{
			Success: false,
			Error:   "peer not found",
		}, nil
	}
	
	// Get disconnect reason
	reason, _ := request.Params["reason"].(string)
	if reason == "" {
		reason = "requested"
	}
	
	// Update peer status first
	peer := p.peers[peerID]
	peer.Status = "disconnected"
	
	// Remove from active peers
	delete(p.peers, peerID)
	
	// Update metrics
	if p.metrics.ActiveConnections > 0 {
		p.metrics.ActiveConnections--
	}
	
	return plugins.PluginResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "peer disconnected",
			"peerId":  peerID,
			"reason":  reason,
		},
	}, nil
}

func (p *NodePlugin) handleGetNetworkStatus(ctx context.Context, request plugins.PluginRequest) (plugins.PluginResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Calculate additional metrics
	var totalBandwidth int64
	for _, peer := range p.peers {
		totalBandwidth += peer.BytesReceived + peer.BytesSent
	}
	p.networkHealth.BandwidthUsage = totalBandwidth
	
	return plugins.PluginResponse{
		Success: true,
		Data: map[string]interface{}{
			"networkHealth": p.networkHealth,
			"nodeId":        p.config.NodeID,
			"version":       p.version,
			"uptime":        time.Since(p.startTime).String(),
			"metrics":       p.metrics,
			"limits": map[string]interface{}{
				"maxPeers":     p.maxPeers,
				"maxBandwidth": p.maxBandwidth,
			},
		},
	}, nil
}

func (p *NodePlugin) handleDiscoverPeers(ctx context.Context, request plugins.PluginRequest) (plugins.PluginResponse, error) {
	// Parse discovery parameters
	method, _ := request.Params["method"].(string)
	maxPeers, _ := request.Params["maxPeers"].(float64)
	
	if method == "" {
		method = p.config.DiscoveryMethod
	}
	if maxPeers <= 0 {
		maxPeers = 10
	}
	
	// Validate discovery is enabled
	if !p.config.EnableDiscovery {
		return plugins.PluginResponse{
			Success: false,
			Error:   "peer discovery is disabled",
		}, nil
	}
	
	// Simulate peer discovery based on method
	var discoveredPeers []map[string]interface{}
	
	switch method {
	case "mdns":
		// Local network discovery
		discoveredPeers = []map[string]interface{}{
			{"id": "local-peer-1", "address": "192.168.1.100:4001", "source": "mdns"},
			{"id": "local-peer-2", "address": "192.168.1.101:4001", "source": "mdns"},
		}
	case "dht":
		// DHT-based discovery
		discoveredPeers = []map[string]interface{}{
			{"id": "dht-peer-1", "address": "203.0.113.10:4001", "source": "dht"},
			{"id": "dht-peer-2", "address": "203.0.113.20:4001", "source": "dht"},
		}
	case "bootstrap":
		// Bootstrap nodes
		for i, addr := range p.config.BootstrapPeers {
			if i >= int(maxPeers) {
				break
			}
			discoveredPeers = append(discoveredPeers, map[string]interface{}{
				"id":      fmt.Sprintf("bootstrap-%d", i),
				"address": addr,
				"source":  "bootstrap",
			})
		}
	default:
		return plugins.PluginResponse{
			Success: false,
			Error:   fmt.Sprintf("unknown discovery method: %s", method),
		}, nil
	}
	
	// Update discovered peers count
	p.mu.Lock()
	p.networkHealth.DiscoveredPeers = len(discoveredPeers)
	p.mu.Unlock()
	
	return plugins.PluginResponse{
		Success: true,
		Data: map[string]interface{}{
			"discovered": discoveredPeers,
			"count":      len(discoveredPeers),
			"method":     method,
		},
	}, nil
}

// ValidateConfig validates the plugin configuration
func ValidateConfig(config *NodeConfig) error {
	if config.NodeID == "" {
		return errors.New("nodeId is required")
	}
	
	if config.P2PPort <= 0 || config.P2PPort > 65535 {
		return errors.New("invalid p2pPort")
	}
	
	if config.MaxPeers < 0 {
		return errors.New("maxPeers cannot be negative")
	}
	
	if config.MaxBandwidthMbps < 0 {
		return errors.New("maxBandwidthMbps cannot be negative")
	}
	
	if config.DiscoveryMethod != "" && 
	   config.DiscoveryMethod != "mdns" && 
	   config.DiscoveryMethod != "dht" && 
	   config.DiscoveryMethod != "bootstrap" {
		return fmt.Errorf("invalid discovery method: %s", config.DiscoveryMethod)
	}
	
	return nil
}

// Main entry point
func main() {
	// Read configuration from environment or file
	configPath := os.Getenv("PLUGIN_CONFIG_PATH")
	if configPath == "" {
		configPath = "/etc/blackhole/plugins/node.json"
	}
	
	// Default configuration
	config := NodeConfig{
		NodeID:              os.Getenv("NODE_ID"),
		Version:             "1.0.0",
		P2PPort:             4001,
		ListenAddresses:     []string{"/ip4/0.0.0.0/tcp/4001"},
		EnableDiscovery:     true,
		DiscoveryMethod:     "bootstrap",
		DiscoveryInterval:   30 * time.Second,
		HealthCheckInterval: 10 * time.Second,
		PeerTimeout:         60 * time.Second,
		MaxPeers:            50,
		MaxBandwidthMbps:    100,
		ConnectionTimeout:   30 * time.Second,
		EnableEncryption:    true,
	}
	
	// Try to load config from file
	if configData, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(configData, &config); err != nil {
			log.Printf("Warning: failed to parse config file: %v", err)
		}
	}
	
	// Override with environment variables
	if nodeID := os.Getenv("NODE_ID"); nodeID != "" {
		config.NodeID = nodeID
	}
	
	if bootstrapPeers := os.Getenv("BOOTSTRAP_PEERS"); bootstrapPeers != "" {
		// Parse comma-separated peers
		// config.BootstrapPeers = strings.Split(bootstrapPeers, ",")
	}
	
	// Validate configuration
	if err := ValidateConfig(&config); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	
	// Create plugin instance
	plugin := &NodePlugin{
		config:  config,
		version: "1.0.0",
		status:  plugins.PluginStatusLoaded,
	}
	
	// Run plugin using the framework's RPC protocol
	if err := plugins.Run(plugin); err != nil {
		log.Fatalf("Plugin failed: %v", err)
	}
}