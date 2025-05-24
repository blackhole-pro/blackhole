// Package main implements the gRPC server for the node plugin
package main

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	
	"node/mesh"
	"node/plugin"
	"node/types"
	nodev1 "node/proto/v1"
)

// nodePluginServer implements the NodePlugin gRPC service
type nodePluginServer struct {
	nodev1.UnimplementedNodePluginServer
	
	plugin         *plugin.Plugin
	meshClient     mesh.Client
	config         *types.NodeConfig
	initialized    bool
}

// NewNodePluginServer creates a new gRPC server for the node plugin
func NewNodePluginServer(p *plugin.Plugin, meshClient mesh.Client) *nodePluginServer {
	return &nodePluginServer{
		plugin:     p,
		meshClient: meshClient,
	}
}

// Initialize initializes the plugin with configuration
func (s *nodePluginServer) Initialize(ctx context.Context, req *nodev1.InitializeRequest) (*nodev1.InitializeResponse, error) {
	if s.initialized {
		return nil, status.Error(codes.AlreadyExists, "plugin already initialized")
	}
	
	// Convert proto config to internal config
	config := &types.NodeConfig{
		NodeID:              req.Config.NodeId,
		P2PPort:             int(req.Config.P2PPort),
		ListenAddresses:     req.Config.ListenAddresses,
		BootstrapPeers:      req.Config.BootstrapPeers,
		EnableDiscovery:     req.Config.EnableDiscovery,
		DiscoveryMethod:     req.Config.DiscoveryMethod,
		MaxPeers:            int(req.Config.MaxPeers),
		MaxBandwidthMbps:    int(req.Config.MaxBandwidthMbps),
		ConnectionTimeout:   req.Config.ConnectionTimeout.AsDuration(),
		EnableEncryption:    req.Config.EnableEncryption,
		PrivateKeyPath:      req.Config.PrivateKeyPath,
	}
	
	s.config = config
	s.initialized = true
	
	// Generate peer ID (in real implementation, from libp2p)
	peerID := fmt.Sprintf("12D3KooW%s", config.NodeID)
	
	// Generate multiaddresses
	multiaddrs := make([]string, len(config.ListenAddresses))
	for i, addr := range config.ListenAddresses {
		multiaddrs[i] = fmt.Sprintf("%s/p2p/%s", addr, peerID)
	}
	
	return &nodev1.InitializeResponse{
		Success:    true,
		Message:    "Node plugin initialized successfully",
		PeerId:     peerID,
		Multiaddrs: multiaddrs,
	}, nil
}

// Start starts the plugin
func (s *nodePluginServer) Start(ctx context.Context, req *nodev1.StartRequest) (*nodev1.StartResponse, error) {
	if !s.initialized {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	
	// Start the plugin
	if err := s.plugin.Start(ctx); err != nil {
		return &nodev1.StartResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	
	// Create endpoints from config
	endpoints := make([]*nodev1.Endpoint, 0, len(s.config.ListenAddresses))
	for _, addr := range s.config.ListenAddresses {
		endpoints = append(endpoints, &nodev1.Endpoint{
			Protocol: "tcp",
			Address:  addr,
			Secure:   s.config.EnableEncryption,
		})
	}
	
	// Create readiness info
	readiness := &nodev1.NetworkReadiness{
		IsReachable: true,
		HasPublicIp: true,
		NatType:     "none",
		PublicAddrs: s.config.ListenAddresses,
	}
	
	// Publish start event
	s.publishEvent("started", map[string]interface{}{
		"endpoints": endpoints,
		"readiness": readiness,
	})
	
	return &nodev1.StartResponse{
		Success:   true,
		Message:   "Node plugin started successfully",
		Endpoints: endpoints,
		Readiness: readiness,
	}, nil
}

// Stop stops the plugin
func (s *nodePluginServer) Stop(ctx context.Context, req *nodev1.StopRequest) (*nodev1.StopResponse, error) {
	// Get peer count before stopping
	active, _ := s.plugin.GetPeerManager().GetPeerCount()
	
	// Stop the plugin
	if err := s.plugin.Stop(ctx); err != nil {
		return &nodev1.StopResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	
	// Publish stop event
	s.publishEvent("stopped", map[string]interface{}{
		"reason":             req.Reason,
		"connections_closed": active,
	})
	
	return &nodev1.StopResponse{
		Success:           true,
		Message:           "Node plugin stopped successfully",
		ConnectionsClosed: int32(active),
	}, nil
}

// HealthCheck checks plugin health
func (s *nodePluginServer) HealthCheck(ctx context.Context, req *nodev1.HealthCheckRequest) (*nodev1.HealthCheckResponse, error) {
	err := s.plugin.HealthCheck()
	health := s.plugin.GetHealthMonitor().GetHealth()
	
	resp := &nodev1.HealthCheckResponse{
		Healthy: err == nil,
		Status:  health.Status,
	}
	
	if req.IncludeDiagnostics {
		// TODO: Use metrics for actual diagnostics
		_ = s.plugin.GetNetManager().GetMetrics()
		active, _ := s.plugin.GetPeerManager().GetPeerCount()
		
		resp.Diagnostics = &nodev1.NetworkDiagnostics{
			ActiveConnections:   int32(active),
			PendingConnections:  0,
			BandwidthUsageMbps:  float64(health.BandwidthUsage) * 8 / 1024 / 1024,
			CpuUsagePercent:     5.0,  // TODO: Get actual CPU usage
			MemoryUsageMb:       50.0, // TODO: Get actual memory usage
			Issues:              make(map[string]string),
		}
		
		if health.Status != "healthy" {
			resp.Diagnostics.Issues["health"] = health.Status
		}
		if active == 0 {
			resp.Diagnostics.Issues["peers"] = "No connected peers"
		}
	}
	
	return resp, nil
}

// GetInfo returns plugin information
func (s *nodePluginServer) GetInfo(ctx context.Context, req *nodev1.GetInfoRequest) (*nodev1.GetInfoResponse, error) {
	info := s.plugin.Info()
	
	return &nodev1.GetInfoResponse{
		Name:        info.Name,
		Version:     info.Version,
		Description: info.Description,
		PeerId:      s.config.NodeID,
		Protocols:   []string{"/ipfs/1.0.0", "/blackhole/1.0.0"},
		ListenAddrs: s.config.ListenAddresses,
		Capabilities: &nodev1.NodeCapabilities{
			SupportsRelay:        true,
			SupportsNatTraversal: true,
			SupportsDht:          s.config.DiscoveryMethod == "dht",
			SupportsPubsub:       true,
			TransportProtocols:   []string{"tcp", "quic", "websocket"},
		},
		Status: &nodev1.NodeStatus{
			State:         string(info.Status),
			StartedAt:     timestamppb.New(info.LoadTime),
			UptimeSeconds: int64(info.Uptime.Seconds()),
			Version:       info.Version,
		},
	}, nil
}

// ListPeers lists connected peers
func (s *nodePluginServer) ListPeers(ctx context.Context, req *nodev1.ListPeersRequest) (*nodev1.ListPeersResponse, error) {
	filter := types.PeerFilter{
		Status: req.StatusFilter,
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	}
	
	peers, err := s.plugin.GetPeerManager().ListPeers(filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list peers: %v", err)
	}
	
	// Convert internal peers to proto
	protoPeers := make([]*nodev1.PeerInfo, len(peers))
	for i, peer := range peers {
		protoPeers[i] = s.peerInfoToProto(peer)
	}
	
	active, total := s.plugin.GetPeerManager().GetPeerCount()
	
	return &nodev1.ListPeersResponse{
		Peers:      protoPeers,
		TotalCount: int32(total),
		Stats: &nodev1.PeerStats{
			TotalKnown:   int32(total),
			Connected:    int32(active),
			Connecting:   0,
			Disconnected: int32(total - active),
		},
	}, nil
}

// ConnectPeer connects to a peer
func (s *nodePluginServer) ConnectPeer(ctx context.Context, req *nodev1.ConnectPeerRequest) (*nodev1.ConnectPeerResponse, error) {
	address := req.PeerId
	if len(req.Addrs) > 0 {
		address = req.Addrs[0]
	}
	
	err := s.plugin.GetPeerManager().Connect(ctx, req.PeerId, address)
	if err != nil {
		return &nodev1.ConnectPeerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	
	// Get peer info
	peer, err := s.plugin.GetPeerManager().GetPeer(req.PeerId)
	if err != nil {
		return &nodev1.ConnectPeerResponse{
			Success: true,
			Message: "Connected but failed to get peer info",
		}, nil
	}
	
	// Publish peer connected event
	s.publishPeerEvent("connected", req.PeerId, map[string]interface{}{
		"address":   address,
		"timestamp": time.Now(),
	})
	
	return &nodev1.ConnectPeerResponse{
		Success:  true,
		Message:  "Peer connected successfully",
		PeerInfo: s.peerInfoToProto(peer),
	}, nil
}

// DisconnectPeer disconnects from a peer
func (s *nodePluginServer) DisconnectPeer(ctx context.Context, req *nodev1.DisconnectPeerRequest) (*nodev1.DisconnectPeerResponse, error) {
	err := s.plugin.GetPeerManager().Disconnect(ctx, req.PeerId, req.Reason)
	if err != nil {
		return &nodev1.DisconnectPeerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	
	// Publish peer disconnected event
	s.publishPeerEvent("disconnected", req.PeerId, map[string]interface{}{
		"reason":    req.Reason,
		"timestamp": time.Now(),
	})
	
	return &nodev1.DisconnectPeerResponse{
		Success: true,
		Message: "Peer disconnected successfully",
	}, nil
}

// GetNetworkStatus returns network status
func (s *nodePluginServer) GetNetworkStatus(ctx context.Context, req *nodev1.GetNetworkStatusRequest) (*nodev1.GetNetworkStatusResponse, error) {
	health := s.plugin.GetHealthMonitor().GetHealth()
	metrics := s.plugin.GetNetManager().GetMetrics()
	
	resp := &nodev1.GetNetworkStatusResponse{
		Health: &nodev1.NetworkHealth{
			Status:      health.Status,
			Score:       health.HealthScore,
			Issues:      []string{},
			LastUpdated: timestamppb.New(health.LastUpdated),
		},
		Metrics: &nodev1.NetworkMetrics{
			TotalConnections:  metrics.TotalConnections,
			ActiveConnections: metrics.ActiveConnections,
			FailedConnections: metrics.FailedConnections,
			BytesSent:         metrics.BytesSent,
			BytesReceived:     metrics.BytesReceived,
			MessagesSent:      metrics.MessagesSent,
			MessagesReceived:  metrics.MessagesReceived,
			StartedAt:         timestamppb.New(metrics.LastReset),
		},
	}
	
	if req.IncludeBandwidth {
		status := s.plugin.GetNetManager().GetNetworkStatus()
		if rates, ok := status["rates"].(map[string]float64); ok {
			resp.Bandwidth = &nodev1.BandwidthStats{
				RateInMbps:  rates["mbpsIn"],
				RateOutMbps: rates["mbpsOut"],
				TotalIn:     metrics.BytesReceived,
				TotalOut:    metrics.BytesSent,
				LimitMbps:   float64(s.config.MaxBandwidthMbps),
			}
		}
	}
	
	if req.IncludeRouting {
		active, total := s.plugin.GetPeerManager().GetPeerCount()
		resp.Routing = &nodev1.RoutingInfo{
			RoutingTableSize: int32(total),
			Protocols:        []string{"/ipfs/1.0.0", "/blackhole/1.0.0"},
			IsReachable:      active > 0,
			NatStatus:        "none",
		}
	}
	
	// Publish status change event if health changed
	s.publishEvent("network.status.changed", map[string]interface{}{
		"status": health.Status,
		"score":  health.HealthScore,
	})
	
	return resp, nil
}

// DiscoverPeers discovers new peers
func (s *nodePluginServer) DiscoverPeers(ctx context.Context, req *nodev1.DiscoverPeersRequest) (*nodev1.DiscoverPeersResponse, error) {
	method := req.Method
	if method == "" {
		method = s.config.DiscoveryMethod
	}
	
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}
	
	peers, err := s.plugin.GetDiscovery().DiscoverPeers(ctx, method, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "discovery failed: %v", err)
	}
	
	// Convert to proto
	discovered := make([]*nodev1.DiscoveredPeer, len(peers))
	for i, peer := range peers {
		discovered[i] = &nodev1.DiscoveredPeer{
			Id:         peer.ID,
			Addrs:      []string{peer.Address},
			Source:     peer.Source,
			Confidence: 0.8,
			Metadata:   make(map[string]string),
		}
		
		// Publish discovered event
		s.publishPeerEvent("discovered", peer.ID, map[string]interface{}{
			"address": peer.Address,
			"source":  peer.Source,
		})
	}
	
	return &nodev1.DiscoverPeersResponse{
		Peers:           discovered,
		MethodUsed:      method,
		TotalDiscovered: int32(len(discovered)),
	}, nil
}

// StreamPeerEvents streams peer events
func (s *nodePluginServer) StreamPeerEvents(req *nodev1.StreamPeerEventsRequest, stream nodev1.NodePlugin_StreamPeerEventsServer) error {
	// Subscribe to mesh events
	pattern := "node.peer.*"
	events, err := s.meshClient.Subscribe(pattern)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe to events: %v", err)
	}
	
	// Stream events to client
	for event := range events {
		// Filter by event type if specified
		if len(req.EventTypes) > 0 {
			found := false
			for _, t := range req.EventTypes {
				if event.Type == fmt.Sprintf("node.peer.%s", t) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Extract peer ID from event data
		peerID := ""
		if id, ok := event.Data["peer_id"].(string); ok {
			peerID = id
		}
		
		// Filter by peer ID if specified
		if len(req.PeerIds) > 0 {
			found := false
			for _, id := range req.PeerIds {
				if peerID == id {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Send event to client
		peerEvent := &nodev1.PeerEvent{
			Type:      event.Type,
			PeerId:    peerID,
			Timestamp: timestamppb.New(event.Timestamp),
			// Data field would be converted to protobuf Struct
		}
		
		if err := stream.Send(peerEvent); err != nil {
			return err
		}
	}
	
	return nil
}

// StreamNetworkMetrics streams network metrics
func (s *nodePluginServer) StreamNetworkMetrics(req *nodev1.StreamNetworkMetricsRequest, stream nodev1.NodePlugin_StreamNetworkMetricsServer) error {
	ticker := time.NewTicker(req.Interval.AsDuration())
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			metrics := s.plugin.GetNetManager().GetMetrics()
			protoMetrics := &nodev1.NetworkMetrics{
				TotalConnections:  metrics.TotalConnections,
				ActiveConnections: metrics.ActiveConnections,
				FailedConnections: metrics.FailedConnections,
				BytesSent:         metrics.BytesSent,
				BytesReceived:     metrics.BytesReceived,
				MessagesSent:      metrics.MessagesSent,
				MessagesReceived:  metrics.MessagesReceived,
				StartedAt:         timestamppb.New(metrics.LastReset),
			}
			
			if err := stream.Send(protoMetrics); err != nil {
				return err
			}
			
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

// Helper methods

func (s *nodePluginServer) peerInfoToProto(peer *types.PeerInfo) *nodev1.PeerInfo {
	return &nodev1.PeerInfo{
		Id:           peer.ID,
		Addrs:        []string{peer.Address},
		Status:       peer.Status,
		ConnectedAt:  timestamppb.New(peer.ConnectedAt),
		LastSeen:     timestamppb.New(peer.LastSeen),
		Protocols:    peer.Protocols,
		AgentVersion: peer.UserAgent,
		Metadata:     make(map[string]string),
		Metrics: &nodev1.PeerMetrics{
			BytesSent:        peer.BytesSent,
			BytesReceived:    peer.BytesReceived,
			MessagesSent:     peer.MessagesSent,
			MessagesReceived: peer.MessagesRecv,
			LatencyMs:        float64(peer.Latency.Milliseconds()),
			PacketLoss:       0.0,
		},
	}
}

func (s *nodePluginServer) publishEvent(eventType string, data map[string]interface{}) {
	event := mesh.Event{
		Type:      fmt.Sprintf("node.%s", eventType),
		Source:    s.config.NodeID,
		Timestamp: time.Now(),
		Data:      data,
	}
	
	if err := s.meshClient.PublishEvent(event); err != nil {
		// Log error but don't fail the operation
		// In real implementation, use proper logging
	}
}

func (s *nodePluginServer) publishPeerEvent(eventType string, peerID string, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["peer_id"] = peerID
	
	event := mesh.Event{
		Type:      fmt.Sprintf("node.peer.%s", eventType),
		Source:    s.config.NodeID,
		Timestamp: time.Now(),
		Data:      data,
	}
	
	if err := s.meshClient.PublishEvent(event); err != nil {
		// Log error but don't fail the operation
		// In real implementation, use proper logging
	}
}