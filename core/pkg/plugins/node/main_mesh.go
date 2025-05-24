// Package main implements the node plugin using the mesh client library
package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/blackhole-pro/blackhole/core/pkg/sdk/plugin/client"
	nodev1 "github.com/blackhole-pro/blackhole/core/pkg/plugins/node/proto/v1"
)

// NodePluginService implements the node plugin gRPC service
type NodePluginService struct {
	nodev1.UnimplementedNodePluginServer
	
	config NodeConfig
	logger *zap.Logger
	
	// State
	peers         map[string]*nodev1.PeerInfo
	networkHealth *nodev1.NetworkHealth
	metrics       *nodev1.NetworkMetrics
	started       time.Time
}

// NewNodePluginService creates a new node plugin service
func NewNodePluginService(logger *zap.Logger) *NodePluginService {
	return &NodePluginService{
		logger:  logger,
		peers:   make(map[string]*nodev1.PeerInfo),
		started: time.Now(),
		networkHealth: &nodev1.NetworkHealth{
			Status: "healthy",
			Score:  1.0,
		},
		metrics: &nodev1.NetworkMetrics{
			StartedAt: timestampNow(),
		},
	}
}

// Initialize initializes the plugin
func (s *NodePluginService) Initialize(ctx context.Context, req *nodev1.InitializeRequest) (*nodev1.InitializeResponse, error) {
	s.logger.Info("Initializing node plugin", zap.Any("config", req.Config))
	
	// Store configuration
	s.config = *req.Config
	
	// TODO: Initialize libp2p host with the provided configuration
	
	return &nodev1.InitializeResponse{
		Success:    true,
		Message:    "Node plugin initialized",
		PeerId:     s.config.NodeId, // In real implementation, generate from libp2p
		Multiaddrs: s.config.ListenAddresses,
	}, nil
}

// Start starts the plugin
func (s *NodePluginService) Start(ctx context.Context, req *nodev1.StartRequest) (*nodev1.StartResponse, error) {
	s.logger.Info("Starting node plugin")
	
	// TODO: Start libp2p host and services
	
	endpoints := make([]*nodev1.Endpoint, 0)
	for _, addr := range s.config.ListenAddresses {
		endpoints = append(endpoints, &nodev1.Endpoint{
			Protocol: "tcp",
			Address:  addr,
			Secure:   s.config.EnableEncryption,
		})
	}
	
	return &nodev1.StartResponse{
		Success:   true,
		Message:   "Node plugin started",
		Endpoints: endpoints,
		Readiness: &nodev1.NetworkReadiness{
			IsReachable: true,
			HasPublicIp: true,
			NatType:     "none",
			PublicAddrs: s.config.ListenAddresses,
		},
	}, nil
}

// Stop stops the plugin
func (s *NodePluginService) Stop(ctx context.Context, req *nodev1.StopRequest) (*nodev1.StopResponse, error) {
	s.logger.Info("Stopping node plugin", 
		zap.Bool("force", req.Force),
		zap.String("reason", req.Reason))
	
	// TODO: Stop libp2p host and clean up
	
	connectionsClosed := int32(len(s.peers))
	s.peers = make(map[string]*nodev1.PeerInfo)
	
	return &nodev1.StopResponse{
		Success:           true,
		Message:           "Node plugin stopped",
		ConnectionsClosed: connectionsClosed,
	}, nil
}

// HealthCheck checks plugin health
func (s *NodePluginService) HealthCheck(ctx context.Context, req *nodev1.HealthCheckRequest) (*nodev1.HealthCheckResponse, error) {
	healthy := len(s.peers) > 0 || time.Since(s.started) < 30*time.Second
	
	resp := &nodev1.HealthCheckResponse{
		Healthy: healthy,
		Status:  "healthy",
	}
	
	if !healthy {
		resp.Status = "degraded"
	}
	
	if req.IncludeDiagnostics {
		resp.Diagnostics = &nodev1.NetworkDiagnostics{
			ActiveConnections:   int32(len(s.peers)),
			PendingConnections:  0,
			BandwidthUsageMbps:  0.0,
			CpuUsagePercent:     5.0,
			MemoryUsageMb:       50.0,
			Issues:              make(map[string]string),
		}
		
		if len(s.peers) == 0 {
			resp.Diagnostics.Issues["peers"] = "No connected peers"
		}
	}
	
	return resp, nil
}

// GetInfo returns plugin information
func (s *NodePluginService) GetInfo(ctx context.Context, req *nodev1.GetInfoRequest) (*nodev1.GetInfoResponse, error) {
	return &nodev1.GetInfoResponse{
		Name:        "node",
		Version:     "1.0.0",
		Description: "P2P networking and peer management plugin",
		PeerId:      s.config.NodeId,
		Protocols:   []string{"/ipfs/1.0.0", "/blackhole/1.0.0"},
		ListenAddrs: s.config.ListenAddresses,
		Capabilities: &nodev1.NodeCapabilities{
			SupportsRelay:        true,
			SupportsNatTraversal: true,
			SupportsDht:          true,
			SupportsPubsub:       true,
			TransportProtocols:   []string{"tcp", "quic", "websocket"},
		},
		Status: &nodev1.NodeStatus{
			State:         "running",
			StartedAt:     timestampNow(),
			UptimeSeconds: int64(time.Since(s.started).Seconds()),
			Version:       "1.0.0",
		},
	}, nil
}

// ListPeers lists connected peers
func (s *NodePluginService) ListPeers(ctx context.Context, req *nodev1.ListPeersRequest) (*nodev1.ListPeersResponse, error) {
	peers := make([]*nodev1.PeerInfo, 0, len(s.peers))
	
	for _, peer := range s.peers {
		if req.StatusFilter == "" || peer.Status == req.StatusFilter {
			peers = append(peers, peer)
		}
	}
	
	// Apply pagination
	start := int(req.Offset)
	end := start + int(req.Limit)
	if start > len(peers) {
		start = len(peers)
	}
	if end > len(peers) || req.Limit == 0 {
		end = len(peers)
	}
	
	return &nodev1.ListPeersResponse{
		Peers:      peers[start:end],
		TotalCount: int32(len(peers)),
		Stats: &nodev1.PeerStats{
			TotalKnown:   int32(len(s.peers)),
			Connected:    int32(len(peers)),
			Connecting:   0,
			Disconnected: 0,
		},
	}, nil
}

// Additional methods would be implemented similarly...

func timestampNow() *Timestamp {
	// This would use the actual protobuf timestamp
	return nil
}

type Timestamp struct{}

func main() {
	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	// Create plugin configuration
	config := client.DefaultConfig("node")
	config.Logger = logger
	config.Description = "P2P networking and peer management plugin"
	
	// Override socket path from environment if set
	if socket := os.Getenv("PLUGIN_SOCKET"); socket != "" {
		config.SocketPath = socket
	}
	
	// Add lifecycle callbacks
	config.OnStart = func() error {
		logger.Info("Node plugin starting")
		// Initialize P2P networking
		return nil
	}
	
	config.OnStop = func() error {
		logger.Info("Node plugin stopping")
		// Clean up P2P connections
		return nil
	}

	// Create plugin client
	pluginClient, err := client.New(config)
	if err != nil {
		logger.Fatal("Failed to create plugin client", zap.Error(err))
	}

	// Create and register the node service
	nodeService := NewNodePluginService(logger)
	nodev1.RegisterNodePluginServer(pluginClient.GetGRPCServer(), nodeService)

	// Set initial health status
	pluginClient.SetHealthStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Run the plugin
	logger.Info("Node plugin ready to start")
	if err := pluginClient.Run(context.Background()); err != nil {
		logger.Fatal("Plugin failed", zap.Error(err))
	}
}