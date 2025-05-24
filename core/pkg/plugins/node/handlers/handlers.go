// Package handlers implements request handlers for the node plugin
package handlers

import (
	"context"
	"fmt"

	"node/types"
	"go.uber.org/zap"
)

// Handler processes plugin requests
type Handler struct {
	peerManager   types.PeerManager
	discovery     types.PeerDiscovery
	healthMonitor types.HealthMonitor
	netManager    types.NetworkManager
	logger        *zap.Logger
}

// NewHandler creates a new request handler
func NewHandler(
	peerManager types.PeerManager,
	discovery types.PeerDiscovery,
	healthMonitor types.HealthMonitor,
	netManager types.NetworkManager,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		peerManager:   peerManager,
		discovery:     discovery,
		healthMonitor: healthMonitor,
		netManager:    netManager,
		logger:        logger,
	}
}

// HandleRequest routes requests to appropriate handlers
func (h *Handler) HandleRequest(ctx context.Context, request types.PluginRequest) (types.PluginResponse, error) {
	h.logger.Debug("Handling request",
		zap.String("method", request.Method),
		zap.Any("params", request.Params))

	switch request.Method {
	case "listPeers":
		return h.HandleListPeers(ctx, request)
	case "connectPeer":
		return h.HandleConnectPeer(ctx, request)
	case "disconnectPeer":
		return h.HandleDisconnectPeer(ctx, request)
	case "getNetworkStatus":
		return h.HandleGetNetworkStatus(ctx, request)
	case "discoverPeers":
		return h.HandleDiscoverPeers(ctx, request)
	default:
		return types.PluginResponse{
			Success: false,
			Error:   fmt.Sprintf("unknown method: %s", request.Method),
		}, nil
	}
}

// HandleListPeers handles peer listing requests
func (h *Handler) HandleListPeers(ctx context.Context, request types.PluginRequest) (types.PluginResponse, error) {
	// Parse parameters
	filter := h.parsePeerFilter(request.Params)

	// Get peers
	peers, err := h.peerManager.ListPeers(filter)
	if err != nil {
		h.logger.Error("Failed to list peers", zap.Error(err))
		return types.PluginResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to list peers: %v", err),
		}, nil
	}

	// Get total count for pagination
	_, total := h.peerManager.GetPeerCount()

	return types.PluginResponse{
		Success: true,
		Data: map[string]interface{}{
			"peers":      peers,
			"count":      len(peers),
			"totalCount": total,
			"offset":     filter.Offset,
			"limit":      filter.Limit,
		},
	}, nil
}

// HandleConnectPeer handles peer connection requests
func (h *Handler) HandleConnectPeer(ctx context.Context, request types.PluginRequest) (types.PluginResponse, error) {
	// Validate parameters
	peerID, address, err := h.validateConnectParams(request.Params)
	if err != nil {
		return types.PluginResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Check if already connected
	if peer, err := h.peerManager.GetPeer(peerID); err == nil && peer.Status == "connected" {
		return types.PluginResponse{
			Success: true,
			Data: map[string]interface{}{
				"message": "already connected",
				"peer":    peer,
			},
		}, nil
	}

	// Connect to peer
	if err := h.peerManager.Connect(ctx, peerID, address); err != nil {
		h.logger.Error("Failed to connect peer",
			zap.String("peerId", peerID),
			zap.Error(err))
		return types.PluginResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to connect: %v", err),
		}, nil
	}

	return types.PluginResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "peer connected",
			"peerId":  peerID,
		},
	}, nil
}

// HandleDisconnectPeer handles peer disconnection requests
func (h *Handler) HandleDisconnectPeer(ctx context.Context, request types.PluginRequest) (types.PluginResponse, error) {
	// Validate parameters
	peerID, ok := request.Params["peerId"].(string)
	if !ok || peerID == "" {
		return types.PluginResponse{
			Success: false,
			Error:   "peerId is required",
		}, nil
	}

	// Get disconnect reason
	reason, _ := request.Params["reason"].(string)
	if reason == "" {
		reason = "requested"
	}

	// Disconnect peer
	if err := h.peerManager.Disconnect(ctx, peerID, reason); err != nil {
		h.logger.Error("Failed to disconnect peer",
			zap.String("peerId", peerID),
			zap.Error(err))
		return types.PluginResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to disconnect: %v", err),
		}, nil
	}

	return types.PluginResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "peer disconnected",
			"peerId":  peerID,
			"reason":  reason,
		},
	}, nil
}

// HandleGetNetworkStatus handles network status requests
func (h *Handler) HandleGetNetworkStatus(ctx context.Context, request types.PluginRequest) (types.PluginResponse, error) {
	// Get network status
	status := h.netManager.GetNetworkStatus()

	// Add health information
	health := h.healthMonitor.GetHealth()
	status["networkHealth"] = health

	// Add peer counts
	active, total := h.peerManager.GetPeerCount()
	status["peers"] = map[string]int{
		"active": active,
		"total":  total,
	}

	return types.PluginResponse{
		Success: true,
		Data:    status,
	}, nil
}

// HandleDiscoverPeers handles peer discovery requests
func (h *Handler) HandleDiscoverPeers(ctx context.Context, request types.PluginRequest) (types.PluginResponse, error) {
	// Parse parameters
	method, maxPeers := h.parseDiscoveryParams(request.Params)

	// Perform discovery
	peers, err := h.discovery.DiscoverPeers(ctx, method, maxPeers)
	if err != nil {
		h.logger.Error("Failed to discover peers",
			zap.String("method", method),
			zap.Error(err))
		return types.PluginResponse{
			Success: false,
			Error:   fmt.Sprintf("discovery failed: %v", err),
		}, nil
	}

	return types.PluginResponse{
		Success: true,
		Data: map[string]interface{}{
			"discovered": peers,
			"count":      len(peers),
			"method":     method,
		},
	}, nil
}

// Helper methods

func (h *Handler) parsePeerFilter(params map[string]interface{}) types.PeerFilter {
	filter := types.PeerFilter{
		Limit: 50, // Default limit
	}

	if status, ok := params["status"].(string); ok {
		filter.Status = status
	}

	if limit, ok := params["limit"].(float64); ok && limit > 0 {
		filter.Limit = int(limit)
	}

	if offset, ok := params["offset"].(float64); ok && offset >= 0 {
		filter.Offset = int(offset)
	}

	return filter
}

func (h *Handler) validateConnectParams(params map[string]interface{}) (peerID, address string, err error) {
	peerID, ok := params["peerId"].(string)
	if !ok || peerID == "" {
		return "", "", fmt.Errorf("peerId is required")
	}

	address, _ = params["address"].(string)
	if address == "" {
		address = peerID // Fallback to ID as address
	}

	return peerID, address, nil
}

func (h *Handler) parseDiscoveryParams(params map[string]interface{}) (method string, maxPeers int) {
	method, _ = params["method"].(string)
	maxPeers = 10 // Default

	if max, ok := params["maxPeers"].(float64); ok && max > 0 {
		maxPeers = int(max)
	}

	return method, maxPeers
}