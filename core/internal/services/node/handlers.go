package main

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/blackhole-pro/blackhole/core/internal/services/node/types"
	// Import the generated protobuf types (we'll need to generate these)
	// For now, using placeholder imports
)

// NodeServiceServer implements the gRPC NodeService interface
type NodeServiceServer struct {
	// Embed the unimplemented server for forward compatibility
	// UnimplementedNodeServiceServer
	
	service *NodeService
}

// NewNodeServiceServer creates a new gRPC server for the node service
func NewNodeServiceServer(service *NodeService) *NodeServiceServer {
	return &NodeServiceServer{
		service: service,
	}
}

// GetNodeInfo returns information about this node
func (s *NodeServiceServer) GetNodeInfo(ctx context.Context, req *GetNodeInfoRequest) (*NodeInfo, error) {
	// Get node metrics
	metrics := s.service.GetNodeInfo()
	
	// Convert to protobuf response
	nodeInfo := &NodeInfo{
		NodeId:    s.service.nodeID,
		Version:   s.service.config.Version,
		Addresses: []string{s.service.config.ListenAddress},
		Port:      int32(s.service.config.ListenPort),
		StartedAt: timestamppb.New(s.service.startedAt),
		Status:    string(s.service.status),
		Metrics: &NodeMetrics{
			TotalConnections:  metrics.TotalConnections,
			ActiveConnections: metrics.ActiveConnections,
			BytesSent:        metrics.BytesSent,
			BytesReceived:    metrics.BytesReceived,
			MessagesSent:     metrics.MessagesSent,
			MessagesReceived: metrics.MessagesRecv,
			LastUpdated:      timestamppb.New(metrics.LastUpdated),
		},
	}
	
	// Include peers if requested
	if req.IncludePeers {
		peers, _ := s.service.ListPeers("", 100, 0) // Get up to 100 peers
		for _, peer := range peers {
			nodeInfo.Peers = append(nodeInfo.Peers, &PeerInfo{
				PeerId:      peer.ID,
				Address:     peer.Address,
				Status:      string(peer.Status),
				ConnectedAt: timestamppb.New(peer.ConnectedAt),
				BytesSent:   peer.BytesSent,
				BytesReceived: peer.BytesRecv,
			})
		}
	}
	
	return nodeInfo, nil
}

// ListPeers returns list of connected peers
func (s *NodeServiceServer) ListPeers(ctx context.Context, req *ListPeersRequest) (*ListPeersResponse, error) {
	// Set defaults
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 50
	}
	
	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}
	
	// Parse status filter
	var statusFilter types.PeerStatus
	if req.StatusFilter != "" {
		statusFilter = types.PeerStatus(req.StatusFilter)
	}
	
	// Get peers from service
	peers, totalCount := s.service.ListPeers(statusFilter, limit, offset)
	
	// Convert to protobuf response
	response := &ListPeersResponse{
		TotalCount: int32(totalCount),
	}
	
	for _, peer := range peers {
		response.Peers = append(response.Peers, &PeerInfo{
			PeerId:      peer.ID,
			Address:     peer.Address,
			Status:      string(peer.Status),
			ConnectedAt: timestamppb.New(peer.ConnectedAt),
			BytesSent:   peer.BytesSent,
			BytesReceived: peer.BytesRecv,
		})
	}
	
	return response, nil
}

// ConnectToPeer establishes connection to a peer
func (s *NodeServiceServer) ConnectToPeer(ctx context.Context, req *ConnectToPeerRequest) (*ConnectToPeerResponse, error) {
	// Validate request
	if req.PeerAddress == "" {
		return nil, status.Errorf(codes.InvalidArgument, "peer address is required")
	}
	
	// Set timeout
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = s.service.config.ConnectionTimeout
	}
	
	// Create connection request
	connectionReq := &types.PeerConnectionRequest{
		Address:  req.PeerAddress,
		Timeout:  timeout,
		Metadata: req.Metadata,
	}
	
	// Attempt connection
	response, err := s.service.ConnectToPeer(ctx, connectionReq)
	if err != nil {
		// Convert internal error to gRPC error
		if nodeErr, ok := err.(*types.NodeError); ok {
			switch nodeErr.Code {
			case types.ErrorCodeConnectionFailed:
				return nil, status.Errorf(codes.Unavailable, nodeErr.Message)
			case types.ErrorCodeConnectionTimeout:
				return nil, status.Errorf(codes.DeadlineExceeded, nodeErr.Message)
			case types.ErrorCodePeerInvalidAddress:
				return nil, status.Errorf(codes.InvalidArgument, nodeErr.Message)
			default:
				return nil, status.Errorf(codes.Internal, nodeErr.Message)
			}
		}
		return nil, status.Errorf(codes.Internal, "connection failed: %v", err)
	}
	
	// Return successful response
	return &ConnectToPeerResponse{
		Success: response.Success,
		Message: response.Message,
		PeerId:  response.PeerID,
	}, nil
}

// DisconnectFromPeer removes connection to a peer
func (s *NodeServiceServer) DisconnectFromPeer(ctx context.Context, req *DisconnectFromPeerRequest) (*DisconnectFromPeerResponse, error) {
	// Validate request
	if req.PeerId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "peer ID is required")
	}
	
	// Attempt disconnection
	err := s.service.DisconnectFromPeer(req.PeerId, req.Reason)
	if err != nil {
		// Convert internal error to gRPC error
		if nodeErr, ok := err.(*types.NodeError); ok {
			switch nodeErr.Code {
			case types.ErrorCodePeerNotFound:
				return nil, status.Errorf(codes.NotFound, nodeErr.Message)
			default:
				return nil, status.Errorf(codes.Internal, nodeErr.Message)
			}
		}
		return nil, status.Errorf(codes.Internal, "disconnection failed: %v", err)
	}
	
	return &DisconnectFromPeerResponse{
		Success: true,
		Message: "peer disconnected successfully",
	}, nil
}

// GetNetworkStatus returns network connectivity status
func (s *NodeServiceServer) GetNetworkStatus(ctx context.Context, req *GetNetworkStatusRequest) (*NetworkStatus, error) {
	// Get network state from service
	networkState := s.service.GetNetworkStatus()
	
	// Convert to protobuf response
	response := &NetworkStatus{
		Status:              string(networkState.NetworkHealth),
		ConnectedPeers:      int32(networkState.ConnectedPeers),
		TotalDiscoveredPeers: int32(networkState.DiscoveredPeers),
		NetworkHealthScore:  networkState.HealthScore,
		LastUpdated:         timestamppb.New(networkState.LastUpdated),
	}
	
	// Include detailed metrics if requested
	if req.IncludeDetailedMetrics {
		avgLatencyMs := float64(networkState.AverageLatency.Nanoseconds()) / 1e6
		
		response.Metrics = &NetworkMetrics{
			AverageLatencyMs:     avgLatencyMs,
			PacketLossRate:       0.0, // Placeholder
			TotalBandwidthUsed:   networkState.TotalBandwidthUsed,
			FailedConnections:    int32(s.service.metrics.FailedConnections),
			SuccessfulConnections: int32(s.service.metrics.TotalConnections),
		}
	}
	
	return response, nil
}

// DiscoverPeers initiates peer discovery
func (s *NodeServiceServer) DiscoverPeers(ctx context.Context, req *DiscoverPeersRequest) (*DiscoverPeersResponse, error) {
	// Validate request
	if req.DiscoveryMethod == "" {
		req.DiscoveryMethod = "bootstrap" // Default method
	}
	
	// Set defaults
	maxPeers := int(req.MaxPeers)
	if maxPeers <= 0 {
		maxPeers = 10
	}
	
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	
	// Create discovery request
	discoveryReq := &types.DiscoveryRequest{
		Method:   req.DiscoveryMethod,
		MaxPeers: maxPeers,
		Timeout:  timeout,
	}
	
	// Perform discovery
	discoveryResp, err := s.service.DiscoverPeers(ctx, discoveryReq)
	if err != nil {
		// Convert internal error to gRPC error
		if nodeErr, ok := err.(*types.NodeError); ok {
			switch nodeErr.Code {
			case types.ErrorCodeDiscoveryFailed:
				return nil, status.Errorf(codes.Unavailable, nodeErr.Message)
			case types.ErrorCodeDiscoveryTimeout:
				return nil, status.Errorf(codes.DeadlineExceeded, nodeErr.Message)
			case types.ErrorCodeDiscoveryUnavailable:
				return nil, status.Errorf(codes.Unimplemented, nodeErr.Message)
			default:
				return nil, status.Errorf(codes.Internal, nodeErr.Message)
			}
		}
		return nil, status.Errorf(codes.Internal, "discovery failed: %v", err)
	}
	
	// Return successful response
	return &DiscoverPeersResponse{
		DiscoveredAddresses:  discoveryResp.DiscoveredAddresses,
		TotalDiscovered:      int32(discoveryResp.TotalDiscovered),
		DiscoveryMethodUsed:  discoveryResp.MethodUsed,
	}, nil
}

// Placeholder types for protobuf messages (these would be generated from .proto files)
// In a real implementation, these would be imported from the generated package

type GetNodeInfoRequest struct {
	IncludePeers   bool
	IncludeMetrics bool
}

type NodeInfo struct {
	NodeId    string
	Version   string
	Addresses []string
	Port      int32
	StartedAt *timestamppb.Timestamp
	Status    string
	Metrics   *NodeMetrics
	Peers     []*PeerInfo
}

type NodeMetrics struct {
	TotalConnections  int64
	ActiveConnections int64
	BytesSent        int64
	BytesReceived    int64
	MessagesSent     int64
	MessagesReceived int64
	LastUpdated      *timestamppb.Timestamp
}

type PeerInfo struct {
	PeerId        string
	Address       string
	Status        string
	ConnectedAt   *timestamppb.Timestamp
	BytesSent     int64
	BytesReceived int64
}

type ListPeersRequest struct {
	StatusFilter string
	Limit        int32
	Offset       int32
}

type ListPeersResponse struct {
	Peers      []*PeerInfo
	TotalCount int32
}

type ConnectToPeerRequest struct {
	PeerAddress    string
	TimeoutSeconds int32
	Metadata       map[string]string
}

type ConnectToPeerResponse struct {
	Success bool
	Message string
	PeerId  string
}

type DisconnectFromPeerRequest struct {
	PeerId string
	Reason string
}

type DisconnectFromPeerResponse struct {
	Success bool
	Message string
}

type GetNetworkStatusRequest struct {
	IncludeDetailedMetrics bool
}

type NetworkStatus struct {
	Status                string
	ConnectedPeers        int32
	TotalDiscoveredPeers  int32
	NetworkHealthScore    float64
	LastUpdated           *timestamppb.Timestamp
	Metrics               *NetworkMetrics
}

type NetworkMetrics struct {
	AverageLatencyMs      float64
	PacketLossRate        float64
	TotalBandwidthUsed    int64
	FailedConnections     int32
	SuccessfulConnections int32
}

type DiscoverPeersRequest struct {
	DiscoveryMethod string
	MaxPeers        int32
	TimeoutSeconds  int32
}

type DiscoverPeersResponse struct {
	DiscoveredAddresses []string
	TotalDiscovered     int32
	DiscoveryMethodUsed string
}