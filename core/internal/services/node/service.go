package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"github.com/blackhole-pro/blackhole/core/internal/services/node/p2p"
	"github.com/blackhole-pro/blackhole/core/internal/services/node/types"
)

// NodeService implements the core node functionality
type NodeService struct {
	// Configuration
	config *types.NodeConfig
	
	// Service state
	nodeID     string
	status     types.NodeStatus
	startedAt  time.Time
	
	// P2P components
	p2pHost   types.P2PHost
	discovery *p2p.DiscoveryService
	
	// Peer management (legacy - will be replaced by P2P host)
	peers      map[string]*types.Peer
	peersMutex sync.RWMutex
	
	// Metrics
	metrics *types.NodeMetrics
	
	// Service clients for communicating with other services
	clients *ServiceClients
	
	// Network state
	networkState *types.NetworkState
	
	// Channels for coordination
	shutdownCh chan struct{}
	
	// Dependencies
	logger *zap.Logger
	
	// Background workers
	workerWg sync.WaitGroup
}

// NewNodeService creates a new node service instance
func NewNodeService(config *types.NodeConfig, logger *zap.Logger) (*NodeService, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	
	// Create P2P host
	p2pHost, err := p2p.NewLibP2PHost(&config.P2P, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create P2P host: %w", err)
	}
	
	service := &NodeService{
		config:     config,
		nodeID:     config.NodeID,
		status:     types.NodeStatusStarting,
		startedAt:  time.Now(),
		p2pHost:    p2pHost,
		peers:      make(map[string]*types.Peer),
		shutdownCh: make(chan struct{}),
		logger:     logger,
		
		metrics: &types.NodeMetrics{
			LastUpdated: time.Now(),
		},
		
		networkState: &types.NetworkState{
			NetworkHealth: types.NetworkHealthGood,
			HealthScore:   1.0,
			LastUpdated:   time.Now(),
		},
	}
	
	// Initialize service clients
	clientConfig := GetDefaultClientConfig()
	service.clients = NewServiceClients(logger)
	
	if err := service.clients.InitializeClients(clientConfig); err != nil {
		logger.Warn("Failed to initialize some service clients", zap.Error(err))
		// Continue without clients - they're optional for basic functionality
	}
	
	return service, nil
}

// Start starts the node service
func (ns *NodeService) Start(ctx context.Context) error {
	ns.logger.Info("Starting node service",
		zap.String("node_id", ns.nodeID),
		zap.String("version", ns.config.Version))
	
	// Start P2P host
	if err := ns.p2pHost.Start(ctx); err != nil {
		return fmt.Errorf("failed to start P2P host: %w", err)
	}
	
	// Initialize discovery service
	if libp2pHost, ok := ns.p2pHost.(*p2p.LibP2PHost); ok {
		ns.discovery = p2p.NewDiscoveryService(libp2pHost.Host(), nil, ns.logger)
		if err := ns.discovery.Start(ctx); err != nil {
			ns.logger.Warn("Failed to start discovery service", zap.Error(err))
		}
	}
	
	// Update status
	ns.status = types.NodeStatusHealthy
	
	// Start background workers
	ns.startBackgroundWorkers()
	
	// Setup protocol handlers
	ns.setupProtocolHandlers()
	
	// Connect to bootstrap peers if configured
	if len(ns.config.BootstrapPeers) > 0 {
		ns.connectToBootstrapPeersP2P()
	}
	
	ns.logger.Info("Node service started successfully",
		zap.String("node_id", ns.nodeID),
		zap.String("peer_id", ns.p2pHost.GetLocalPeerInfo().PeerID.String()))
	
	return nil
}

// Stop stops the node service
func (ns *NodeService) Stop(ctx context.Context) error {
	ns.logger.Info("Stopping node service",
		zap.String("node_id", ns.nodeID))
	
	// Update status
	ns.status = types.NodeStatusShutdown
	
	// Signal shutdown to background workers
	close(ns.shutdownCh)
	
	// Stop discovery service
	if ns.discovery != nil {
		if err := ns.discovery.Stop(); err != nil {
			ns.logger.Warn("Error stopping discovery service", zap.Error(err))
		}
	}
	
	// Stop P2P host
	if err := ns.p2pHost.Stop(ctx); err != nil {
		ns.logger.Warn("Error stopping P2P host", zap.Error(err))
	}
	
	// Disconnect from all peers (legacy)
	ns.disconnectAllPeers()
	
	// Close service clients
	if ns.clients != nil {
		if err := ns.clients.Close(); err != nil {
			ns.logger.Warn("Error closing service clients", zap.Error(err))
		}
	}
	
	// Wait for background workers to finish
	ns.workerWg.Wait()
	
	ns.logger.Info("Node service stopped")
	return nil
}

// GetNodeInfo returns information about this node
func (ns *NodeService) GetNodeInfo() *types.NodeMetrics {
	ns.updateMetrics()
	
	return &types.NodeMetrics{
		TotalConnections:  ns.metrics.TotalConnections,
		ActiveConnections: ns.getActivePeerCount(),
		FailedConnections: ns.metrics.FailedConnections,
		BytesSent:        ns.metrics.BytesSent,
		BytesReceived:    ns.metrics.BytesReceived,
		MessagesSent:     ns.metrics.MessagesSent,
		MessagesRecv:     ns.metrics.MessagesRecv,
		AverageLatency:   ns.calculateAverageLatency(),
		PacketLoss:       ns.metrics.PacketLoss,
		Uptime:          time.Since(ns.startedAt),
		CPUUsage:        ns.metrics.CPUUsage,
		MemoryUsage:     ns.metrics.MemoryUsage,
		LastUpdated:     time.Now(),
	}
}

// ListPeers returns a list of connected peers
func (ns *NodeService) ListPeers(filter types.PeerStatus, limit, offset int) ([]*types.Peer, int) {
	ns.peersMutex.RLock()
	defer ns.peersMutex.RUnlock()
	
	var filteredPeers []*types.Peer
	
	for _, peer := range ns.peers {
		if filter == "" || peer.Status == filter {
			filteredPeers = append(filteredPeers, peer)
		}
	}
	
	totalCount := len(filteredPeers)
	
	// Apply pagination
	start := offset
	end := offset + limit
	
	if start >= len(filteredPeers) {
		return []*types.Peer{}, totalCount
	}
	
	if end > len(filteredPeers) {
		end = len(filteredPeers)
	}
	
	return filteredPeers[start:end], totalCount
}

// ConnectToPeer connects to a new peer using P2P host
func (ns *NodeService) ConnectToPeer(ctx context.Context, request *types.PeerConnectionRequest) (*types.PeerConnectionResponse, error) {
	ns.logger.Info("Attempting to connect to peer",
		zap.String("address", request.Address))
	
	// Parse peer address
	addr, err := peer.AddrInfoFromString(request.Address)
	if err != nil {
		return &types.PeerConnectionResponse{
			Success: false,
			Message: "invalid peer address",
			Error:   err,
		}, nil
	}
	
	// Check if already connected
	connectedPeers := ns.p2pHost.GetPeers()
	for _, peerID := range connectedPeers {
		if peerID == addr.ID {
			return &types.PeerConnectionResponse{
				Success: true,
				PeerID:  peerID.String(),
				Message: "already connected",
			}, nil
		}
	}
	
	// Check peer limits
	if len(connectedPeers) >= ns.config.MaxPeers {
		return &types.PeerConnectionResponse{
			Success: false,
			Message: "maximum peer limit reached",
			Error:   fmt.Errorf("max peers limit (%d) reached", ns.config.MaxPeers),
		}, nil
	}
	
	// Set timeout for connection
	connectCtx := ctx
	if request.Timeout > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, request.Timeout)
		defer cancel()
	}
	
	// Connect via P2P host
	if err := ns.p2pHost.Connect(connectCtx, *addr); err != nil {
		ns.logger.Warn("Failed to connect to peer",
			zap.String("peer_id", addr.ID.String()),
			zap.Error(err))
		
		return &types.PeerConnectionResponse{
			Success: false,
			Message: "connection failed",
			Error:   err,
		}, nil
	}
	
	// Update metrics
	ns.metrics.TotalConnections++
	
	ns.logger.Info("Successfully connected to peer", zap.String("peer_id", addr.ID.String()))
	
	return &types.PeerConnectionResponse{
		Success: true,
		PeerID:  addr.ID.String(),
		Message: "connection established",
	}, nil
}

// DisconnectFromPeer disconnects from a peer using P2P host
func (ns *NodeService) DisconnectFromPeer(peerIDStr, reason string) error {
	ns.logger.Info("Disconnecting from peer",
		zap.String("peer_id", peerIDStr),
		zap.String("reason", reason))
	
	// Parse peer ID
	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}
	
	// Disconnect via P2P host
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := ns.p2pHost.Disconnect(ctx, peerID); err != nil {
		return fmt.Errorf("failed to disconnect from peer: %w", err)
	}
	
	ns.logger.Info("Disconnected from peer", zap.String("peer_id", peerIDStr))
	
	return nil
}

// GetNetworkStatus returns the current network status
func (ns *NodeService) GetNetworkStatus() *types.NetworkState {
	ns.updateNetworkState()
	
	return &types.NetworkState{
		ConnectedPeers:     int(ns.getActivePeerCount()),
		DiscoveredPeers:    len(ns.peers),
		NetworkHealth:      ns.networkState.NetworkHealth,
		HealthScore:        ns.networkState.HealthScore,
		AverageLatency:     ns.calculateAverageLatency(),
		TotalBandwidthUsed: ns.metrics.BytesSent + ns.metrics.BytesReceived,
		LastUpdated:        time.Now(),
	}
}

// DiscoverPeers initiates peer discovery using the discovery service
func (ns *NodeService) DiscoverPeers(ctx context.Context, request *types.DiscoveryRequest) (*types.DiscoveryResponse, error) {
	ns.logger.Info("Starting peer discovery",
		zap.String("method", request.Method),
		zap.Int("max_peers", request.MaxPeers))
	
	// Use discovery service if available
	if ns.discovery != nil {
		return ns.discovery.DiscoverPeers(ctx, request)
	}
	
	// Fallback to basic discovery methods
	start := time.Now()
	var discoveredAddresses []string
	
	switch request.Method {
	case "bootstrap":
		discoveredAddresses = ns.discoverViaBootstrap(ctx, request)
	case "connected":
		discoveredAddresses = ns.discoverConnectedPeers(ctx, request)
	case "dht":
		// DHT discovery requires the discovery service
		return nil, fmt.Errorf("DHT discovery not available - discovery service not initialized")
	default:
		return nil, types.NewDiscoveryFailedError(request.Method, 
			fmt.Errorf("unsupported discovery method"))
	}
	
	// Filter discovered addresses if filter function is provided
	if request.FilterFunc != nil {
		var filtered []string
		for _, addr := range discoveredAddresses {
			if request.FilterFunc(addr) {
				filtered = append(filtered, addr)
			}
		}
		discoveredAddresses = filtered
	}
	
	duration := time.Since(start)
	
	ns.logger.Info("Peer discovery completed",
		zap.String("method", request.Method),
		zap.Int("discovered", len(discoveredAddresses)),
		zap.Duration("duration", duration))
	
	return &types.DiscoveryResponse{
		DiscoveredAddresses: discoveredAddresses,
		TotalDiscovered:     len(discoveredAddresses),
		MethodUsed:          request.Method,
		Duration:            duration,
	}, nil
}

// Private helper methods

func (ns *NodeService) startBackgroundWorkers() {
	// Start metrics updater
	ns.workerWg.Add(1)
	go ns.metricsUpdater()
	
	// Start peer health checker
	ns.workerWg.Add(1)
	go ns.peerHealthChecker()
	
	// Start network state monitor
	ns.workerWg.Add(1)
	go ns.networkStateMonitor()
}

func (ns *NodeService) metricsUpdater() {
	defer ns.workerWg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ns.updateMetrics()
		case <-ns.shutdownCh:
			return
		}
	}
}

func (ns *NodeService) peerHealthChecker() {
	defer ns.workerWg.Done()
	
	ticker := time.NewTicker(ns.config.PingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ns.checkPeerHealth()
		case <-ns.shutdownCh:
			return
		}
	}
}

func (ns *NodeService) networkStateMonitor() {
	defer ns.workerWg.Done()
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ns.updateNetworkState()
		case <-ns.shutdownCh:
			return
		}
	}
}

func (ns *NodeService) updateMetrics() {
	ns.metrics.ActiveConnections = ns.getActivePeerCount()
	ns.metrics.LastUpdated = time.Now()
	// In a real implementation, you would collect actual system metrics here
}

func (ns *NodeService) updateNetworkState() {
	activePeers := ns.getActivePeerCount()
	
	// Calculate health score based on connected peers
	var healthScore float64
	if ns.config.MinPeers > 0 {
		healthScore = float64(activePeers) / float64(ns.config.MinPeers)
		if healthScore > 1.0 {
			healthScore = 1.0
		}
	} else {
		healthScore = 1.0
	}
	
	// Determine network health
	var health types.NetworkHealth
	if healthScore >= 0.8 {
		health = types.NetworkHealthGood
	} else if healthScore >= 0.5 {
		health = types.NetworkHealthDegraded
	} else {
		health = types.NetworkHealthPoor
	}
	
	ns.networkState.ConnectedPeers = int(activePeers)
	ns.networkState.NetworkHealth = health
	ns.networkState.HealthScore = healthScore
	ns.networkState.LastUpdated = time.Now()
}

func (ns *NodeService) getActivePeerCount() int64 {
	ns.peersMutex.RLock()
	defer ns.peersMutex.RUnlock()
	
	var count int64
	for _, peer := range ns.peers {
		if peer.Status == types.PeerStatusConnected {
			count++
		}
	}
	return count
}

func (ns *NodeService) calculateAverageLatency() time.Duration {
	ns.peersMutex.RLock()
	defer ns.peersMutex.RUnlock()
	
	var totalLatency time.Duration
	var count int
	
	for _, peer := range ns.peers {
		if peer.Status == types.PeerStatusConnected && peer.Latency > 0 {
			totalLatency += peer.Latency
			count++
		}
	}
	
	if count == 0 {
		return 0
	}
	
	return totalLatency / time.Duration(count)
}

func (ns *NodeService) establishPeerConnection(peer *types.Peer, timeout time.Duration) {
	// Simulate connection establishment
	time.Sleep(100 * time.Millisecond) // Simulate network delay
	
	// For demo purposes, assume connection is successful
	ns.peersMutex.Lock()
	peer.Status = types.PeerStatusConnected
	peer.Latency = 50 * time.Millisecond // Simulated latency
	ns.peersMutex.Unlock()
	
	ns.logger.Info("Peer connection established",
		zap.String("peer_id", peer.ID),
		zap.String("address", peer.Address))
}

func (ns *NodeService) checkPeerHealth() {
	ns.peersMutex.Lock()
	defer ns.peersMutex.Unlock()
	
	for _, peer := range ns.peers {
		if peer.Status == types.PeerStatusConnected {
			// Update last seen time (in real implementation, this would be based on actual ping)
			peer.LastSeen = time.Now()
		}
	}
}

func (ns *NodeService) connectToBootstrapPeers() {
	for _, bootstrapAddr := range ns.config.BootstrapPeers {
		request := &types.PeerConnectionRequest{
			Address: bootstrapAddr,
			Timeout: ns.config.ConnectionTimeout,
			Metadata: map[string]string{
				"type": "bootstrap",
			},
		}
		
		_, err := ns.ConnectToPeer(context.Background(), request)
		if err != nil {
			ns.logger.Warn("Failed to connect to bootstrap peer",
				zap.String("address", bootstrapAddr),
				zap.Error(err))
		}
	}
}

func (ns *NodeService) disconnectAllPeers() {
	ns.peersMutex.RLock()
	peerIDs := make([]string, 0, len(ns.peers))
	for peerID := range ns.peers {
		peerIDs = append(peerIDs, peerID)
	}
	ns.peersMutex.RUnlock()
	
	for _, peerID := range peerIDs {
		ns.DisconnectFromPeer(peerID, "service shutdown")
	}
}

// Discovery method implementations (simplified for demo)

func (ns *NodeService) discoverViaBootstrap(ctx context.Context, request *types.DiscoveryRequest) []string {
	// Return configured bootstrap peers
	return ns.config.BootstrapPeers
}

func (ns *NodeService) discoverViaDHT(ctx context.Context, request *types.DiscoveryRequest) []string {
	// Placeholder DHT discovery implementation
	return []string{
		"192.168.1.100:9000",
		"192.168.1.101:9000",
		"192.168.1.102:9000",
	}
}

func (ns *NodeService) discoverLocal(ctx context.Context, request *types.DiscoveryRequest) []string {
	// Placeholder local network discovery implementation
	return []string{
		"127.0.0.1:9001",
		"127.0.0.1:9002",
	}
}

// setupProtocolHandlers sets up protocol handlers for P2P communication
func (ns *NodeService) setupProtocolHandlers() {
	// Register basic ping protocol
	pingHandler := &BasicProtocolHandler{
		protocolID: "/blackhole/ping/1.0.0",
		logger:     ns.logger,
	}
	ns.p2pHost.RegisterProtocolHandler("/blackhole/ping/1.0.0", pingHandler)
	
	// Register node info protocol
	nodeInfoHandler := &NodeInfoProtocolHandler{
		nodeService: ns,
		logger:      ns.logger,
	}
	ns.p2pHost.RegisterProtocolHandler("/blackhole/nodeinfo/1.0.0", nodeInfoHandler)
}

// connectToBootstrapPeersP2P connects to bootstrap peers using P2P host
func (ns *NodeService) connectToBootstrapPeersP2P() {
	for _, peerAddr := range ns.config.BootstrapPeers {
		addr, err := peer.AddrInfoFromString(peerAddr)
		if err != nil {
			ns.logger.Warn("Invalid bootstrap peer address",
				zap.String("address", peerAddr),
				zap.Error(err))
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), ns.config.ConnectionTimeout)
		if err := ns.p2pHost.Connect(ctx, *addr); err != nil {
			ns.logger.Warn("Failed to connect to bootstrap peer",
				zap.String("peer_id", addr.ID.String()),
				zap.Error(err))
		} else {
			ns.logger.Info("Connected to bootstrap peer",
				zap.String("peer_id", addr.ID.String()))
		}
		cancel()
	}
}

// discoverConnectedPeers returns currently connected peers
func (ns *NodeService) discoverConnectedPeers(ctx context.Context, request *types.DiscoveryRequest) []string {
	connectedPeers := ns.p2pHost.GetPeers()
	discoveredAddresses := make([]string, 0, len(connectedPeers))

	for _, peerID := range connectedPeers {
		// Get peer addresses from the host's peerstore
		if ns.p2pHost.Host() != nil {
			addrs := ns.p2pHost.Host().Peerstore().Addrs(peerID)
			for _, addr := range addrs {
				fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), peerID.String())
				discoveredAddresses = append(discoveredAddresses, fullAddr)
			}
		}
	}

	return discoveredAddresses
}

// BasicProtocolHandler implements a basic ping protocol handler
type BasicProtocolHandler struct {
	protocolID string
	logger     *zap.Logger
}

func (h *BasicProtocolHandler) HandleProtocol(ctx context.Context, stream types.StreamHandler) error {
	h.logger.Debug("Handling ping protocol",
		zap.String("protocol", h.protocolID),
		zap.String("peer", stream.RemotePeer().String()))

	// Read ping message
	buffer := make([]byte, 4)
	if _, err := stream.Read(buffer); err != nil {
		return fmt.Errorf("failed to read ping: %w", err)
	}

	// Send pong response
	if _, err := stream.Write([]byte("pong")); err != nil {
		return fmt.Errorf("failed to write pong: %w", err)
	}

	return nil
}

// NodeInfoProtocolHandler implements a node information protocol handler
type NodeInfoProtocolHandler struct {
	nodeService *NodeService
	logger      *zap.Logger
}

func (h *NodeInfoProtocolHandler) HandleProtocol(ctx context.Context, stream types.StreamHandler) error {
	h.logger.Debug("Handling node info protocol",
		zap.String("peer", stream.RemotePeer().String()))

	// Get node info
	nodeInfo := h.nodeService.GetNodeInfo()
	
	// For simplicity, send basic info as string
	info := fmt.Sprintf("node_id:%s,connections:%d,uptime:%s",
		h.nodeService.nodeID,
		nodeInfo.ActiveConnections,
		nodeInfo.Uptime.String())

	if _, err := stream.Write([]byte(info)); err != nil {
		return fmt.Errorf("failed to write node info: %w", err)
	}

	return nil
}