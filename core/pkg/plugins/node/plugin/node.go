// Package plugin implements the main node plugin structure
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"node/discovery"
	"node/handlers"
	"node/health"
	"node/network"
	"node/p2p"
	"node/types"
	"go.uber.org/zap"
)

// Plugin implements the P2P networking and peer management functionality
type Plugin struct {
	// Configuration
	config *types.NodeConfig
	
	// Plugin state
	mu            sync.RWMutex
	status        types.PluginStatus
	startTime     time.Time
	version       string
	
	// Components
	peerManager   *p2p.PeerManager
	discovery     *discovery.Discovery
	healthMonitor *health.Monitor
	netManager    *network.Manager
	handler       *handlers.Handler
	logger        *zap.Logger
	
	// Channels
	shutdownCh    chan struct{}
	metricsUpdate chan types.MetricsUpdate
	
	// Background workers
	workerWg      sync.WaitGroup
}

// NewPlugin creates a new node plugin instance
func NewPlugin(config *types.NodeConfig, logger *zap.Logger) (*Plugin, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	plugin := &Plugin{
		config:        config,
		version:       "1.0.0",
		status:        types.PluginStatusLoaded,
		logger:        logger,
		metricsUpdate: make(chan types.MetricsUpdate, 100),
	}

	// Initialize components
	plugin.peerManager = p2p.NewPeerManager(
		config.MaxPeers,
		config.ConnectionTimeout,
		logger.Named("peer"),
	)
	plugin.peerManager.SetMetricsChannel(plugin.metricsUpdate)

	plugin.discovery = discovery.NewDiscovery(config, logger.Named("discovery"))
	plugin.healthMonitor = health.NewMonitor(config, logger.Named("health"))
	plugin.netManager = network.NewManager(config, logger.Named("network"))
	
	plugin.handler = handlers.NewHandler(
		plugin.peerManager,
		plugin.discovery,
		plugin.healthMonitor,
		plugin.netManager,
		logger.Named("handler"),
	)

	return plugin, nil
}

// Info returns plugin metadata
func (p *Plugin) Info() types.PluginInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return types.PluginInfo{
		Name:        "node",
		Version:     p.version,
		Description: "P2P networking and peer management plugin",
		Author:      "Blackhole Team",
		License:     "Apache-2.0",
		Homepage:    "https://blackhole.foundation/plugins/node",
		Repository:  "https://github.com/blackhole/plugins/node",
		Status:      p.status,
		LoadTime:    p.startTime,
		Uptime:      time.Since(p.startTime),
		Capabilities: []types.PluginCapability{
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
		Permissions: []types.PluginPermission{
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
		ResourceRequirements: types.ResourceRequirements{
			MinMemoryMB: 128,
			MaxMemoryMB: 512,
			MinCPUMHz:   100,
			MaxCPUMHz:   1000,
		},
	}
}

// Start initializes and starts the plugin
func (p *Plugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.status == types.PluginStatusRunning {
		return types.ErrPluginAlreadyRunning
	}
	
	p.logger.Info("Starting node plugin",
		zap.String("nodeId", p.config.NodeID),
		zap.Int("p2pPort", p.config.P2PPort))
	
	p.status = types.PluginStatusStarting
	p.startTime = time.Now()
	p.shutdownCh = make(chan struct{})
	
	// Start background workers
	p.startBackgroundWorkers()
	
	// Connect to bootstrap peers if configured
	if len(p.config.BootstrapPeers) > 0 {
		p.workerWg.Add(1)
		go p.connectToBootstrapPeers(ctx)
	}
	
	// Start discovery if enabled
	if p.config.EnableDiscovery {
		if err := p.discovery.StartDiscovery(ctx); err != nil {
			p.logger.Error("Failed to start discovery", zap.Error(err))
		}
	}
	
	p.status = types.PluginStatusRunning
	p.logger.Info("Node plugin started successfully")
	
	return nil
}

// Stop gracefully shuts down the plugin
func (p *Plugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.status != types.PluginStatusRunning {
		return types.ErrPluginNotRunning
	}
	
	p.logger.Info("Stopping node plugin")
	p.status = types.PluginStatusStopping
	
	// Stop discovery
	if p.config.EnableDiscovery {
		if err := p.discovery.StopDiscovery(ctx); err != nil {
			p.logger.Error("Error stopping discovery", zap.Error(err))
		}
	}
	
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
		p.logger.Debug("All workers stopped gracefully")
	case <-time.After(10 * time.Second):
		p.logger.Warn("Workers did not stop within timeout")
	case <-ctx.Done():
		p.logger.Warn("Stop context cancelled")
	}
	
	// Disconnect all peers
	p.peerManager.DisconnectAll()
	
	p.status = types.PluginStatusStopped
	p.logger.Info("Node plugin stopped")
	
	return nil
}

// Handle processes plugin requests
func (p *Plugin) Handle(ctx context.Context, request types.PluginRequest) (types.PluginResponse, error) {
	p.mu.RLock()
	if p.status != types.PluginStatusRunning {
		p.mu.RUnlock()
		return types.PluginResponse{
			Success: false,
			Error:   types.ErrPluginNotRunning.Error(),
		}, nil
	}
	p.mu.RUnlock()
	
	return p.handler.HandleRequest(ctx, request)
}

// HealthCheck returns the health status of the plugin
func (p *Plugin) HealthCheck() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.status != types.PluginStatusRunning {
		return types.ErrPluginNotRunning
	}
	
	if !p.healthMonitor.IsHealthy() {
		return types.ErrNetworkUnhealthy
	}
	
	return nil
}

// GetStatus returns the current plugin status
func (p *Plugin) GetStatus() types.PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// GetPeerManager returns the peer manager
func (p *Plugin) GetPeerManager() types.PeerManager {
	return p.peerManager
}

// GetDiscovery returns the discovery service
func (p *Plugin) GetDiscovery() types.PeerDiscovery {
	return p.discovery
}

// GetHealthMonitor returns the health monitor
func (p *Plugin) GetHealthMonitor() types.HealthMonitor {
	return p.healthMonitor
}

// GetNetManager returns the network manager
func (p *Plugin) GetNetManager() types.NetworkManager {
	return p.netManager
}

// PrepareShutdown prepares the plugin for shutdown
func (p *Plugin) PrepareShutdown() error {
	p.logger.Info("Preparing for shutdown")
	
	// Export state for persistence
	state, err := p.ExportState()
	if err != nil {
		p.logger.Error("Failed to export state", zap.Error(err))
		return err
	}
	
	// Save state to file (optional)
	// This could be saved to a persistent location
	p.logger.Info("State exported", zap.Int("stateSize", len(state)))
	
	return nil
}

// ExportState exports the plugin state
func (p *Plugin) ExportState() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	state := struct {
		Config        *types.NodeConfig     `json:"config"`
		Peers         map[string]*types.PeerInfo `json:"peers"`
		NetworkHealth *types.NetworkHealth  `json:"networkHealth"`
		Metrics       *types.NetworkMetrics `json:"metrics"`
	}{
		Config:        p.config,
		Peers:         p.peerManager.GetPeerMap(),
		NetworkHealth: p.healthMonitor.GetHealth(),
		Metrics:       p.netManager.GetMetrics(),
	}
	
	return json.Marshal(state)
}

// ImportState imports plugin state
func (p *Plugin) ImportState(data []byte) error {
	var state struct {
		Config        *types.NodeConfig     `json:"config"`
		Peers         map[string]*types.PeerInfo `json:"peers"`
		NetworkHealth *types.NetworkHealth  `json:"networkHealth"`
		Metrics       *types.NetworkMetrics `json:"metrics"`
	}
	
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Update components with imported state
	if state.NetworkHealth != nil {
		p.healthMonitor.UpdateHealth(state.NetworkHealth)
	}
	
	// Note: We don't import peers as connections need to be re-established
	// Config should come from initialization, not import
	
	return nil
}

// Background workers

func (p *Plugin) startBackgroundWorkers() {
	// Health monitor
	if p.config.HealthCheckInterval > 0 {
		p.workerWg.Add(1)
		go p.runHealthMonitor()
	}
	
	// Metrics processor
	p.workerWg.Add(1)
	go p.metricsProcessor()
	
	// Network state updater
	p.workerWg.Add(1)
	go p.networkStateUpdater()
}

func (p *Plugin) runHealthMonitor() {
	defer p.workerWg.Done()
	
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Check peer health
			unhealthyPeers := p.peerManager.CheckHealth(p.config.PeerTimeout)
			if len(unhealthyPeers) > 0 {
				p.logger.Debug("Found unhealthy peers",
					zap.Int("count", len(unhealthyPeers)))
			}
			
			// Update overall health
			peers := p.peerManager.GetPeerMap()
			if err := p.healthMonitor.CheckPeerHealth(peers); err != nil {
				p.logger.Error("Health check failed", zap.Error(err))
			}
			
		case <-p.shutdownCh:
			return
		}
	}
}

func (p *Plugin) metricsProcessor() {
	defer p.workerWg.Done()
	
	for {
		select {
		case update := <-p.metricsUpdate:
			p.netManager.UpdateMetrics(update)
			
		case <-p.shutdownCh:
			return
		}
	}
}

func (p *Plugin) networkStateUpdater() {
	defer p.workerWg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Update network state
			active, total := p.peerManager.GetPeerCount()
			health := p.healthMonitor.GetHealth()
			health.ActivePeers = active
			health.TotalPeers = total
			
			// Update discovered peers count
			discovered := p.discovery.GetDiscoveredPeers()
			p.healthMonitor.SetDiscoveredPeers(len(discovered))
			
		case <-p.shutdownCh:
			return
		}
	}
}

func (p *Plugin) connectToBootstrapPeers(ctx context.Context) {
	defer p.workerWg.Done()
	
	p.logger.Info("Connecting to bootstrap peers",
		zap.Int("count", len(p.config.BootstrapPeers)))
	
	for i, peer := range p.config.BootstrapPeers {
		peerID := fmt.Sprintf("bootstrap-%d", i)
		
		if err := p.peerManager.Connect(ctx, peerID, peer); err != nil {
			p.logger.Error("Failed to connect to bootstrap peer",
				zap.String("address", peer),
				zap.Error(err))
		} else {
			p.logger.Info("Connected to bootstrap peer",
				zap.String("peerId", peerID),
				zap.String("address", peer))
		}
		
		// Small delay between connections
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return
		case <-p.shutdownCh:
			return
		}
	}
}

// ValidateConfig validates the plugin configuration
func ValidateConfig(config *types.NodeConfig) error {
	if config.NodeID == "" {
		return types.NewValidationError("nodeId", "is required")
	}
	
	if config.P2PPort <= 0 || config.P2PPort > 65535 {
		return types.NewValidationError("p2pPort", "must be between 1 and 65535")
	}
	
	if config.MaxPeers < 0 {
		return types.NewValidationError("maxPeers", "cannot be negative")
	}
	
	if config.MaxBandwidthMbps < 0 {
		return types.NewValidationError("maxBandwidthMbps", "cannot be negative")
	}
	
	if config.DiscoveryMethod != "" && 
	   config.DiscoveryMethod != "mdns" && 
	   config.DiscoveryMethod != "dht" && 
	   config.DiscoveryMethod != "bootstrap" {
		return types.NewValidationError("discoveryMethod", 
			fmt.Sprintf("invalid method: %s", config.DiscoveryMethod))
	}
	
	return nil
}