// Package p2p provides P2P networking functionality for the node service.
// It implements libp2p-based peer-to-peer communication, discovery, and protocol handling.
package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"go.uber.org/zap"

	"github.com/blackhole-pro/blackhole/core/internal/services/node/types"
)

// LibP2PHost implements the P2PHost interface using libp2p
type LibP2PHost struct {
	// Configuration
	config *types.P2PConfig
	logger *zap.Logger

	// libp2p components
	host     host.Host
	dht      *dht.IpfsDHT
	ping     *ping.PingService
	discovery *routing.RoutingDiscovery

	// Protocol handlers
	protocolHandlers map[protocol.ID]types.ProtocolHandler
	handlersMutex    sync.RWMutex

	// State management
	ctx       context.Context
	cancel    context.CancelFunc
	started   bool
	startedMu sync.RWMutex
}

// NewLibP2PHost creates a new P2P host with the given configuration
func NewLibP2PHost(config *types.P2PConfig, logger *zap.Logger) (*LibP2PHost, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	h := &LibP2PHost{
		config:           config,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		protocolHandlers: make(map[protocol.ID]types.ProtocolHandler),
	}

	return h, nil
}

// Start initializes and starts the P2P host
func (h *LibP2PHost) Start(ctx context.Context) error {
	h.startedMu.Lock()
	defer h.startedMu.Unlock()

	if h.started {
		return fmt.Errorf("P2P host already started")
	}

	h.logger.Info("Starting P2P host")

	// Generate or load private key
	privateKey, err := h.getOrCreatePrivateKey()
	if err != nil {
		return fmt.Errorf("failed to get private key: %w", err)
	}

	// Create libp2p host options
	opts := []libp2p.Option{
		libp2p.Identity(privateKey),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
	}

	// Add listen addresses
	if len(h.config.ListenAddresses) > 0 {
		opts = append(opts, libp2p.ListenAddrStrings(h.config.ListenAddresses...))
	}

	// Add connection manager if needed
	// For now, use default connection manager

	// Create libp2p host
	h.host, err = libp2p.New(opts...)
	if err != nil {
		return fmt.Errorf("failed to create libp2p host: %w", err)
	}

	h.logger.Info("P2P host created",
		zap.String("peer_id", h.host.ID().String()),
		zap.Strings("addresses", h.getHostAddresses()))

	// Initialize DHT if enabled
	if h.config.EnableDHT {
		if err := h.initializeDHT(ctx); err != nil {
			return fmt.Errorf("failed to initialize DHT: %w", err)
		}
	}

	// Initialize ping service
	h.ping = ping.NewPingService(h.host)

	// Setup protocol handlers
	h.setupProtocolHandlers()

	// Connect to bootstrap peers
	if len(h.config.BootstrapPeers) > 0 {
		go h.connectToBootstrapPeers(ctx)
	}

	h.started = true
	h.logger.Info("P2P host started successfully")

	return nil
}

// Stop gracefully shuts down the P2P host
func (h *LibP2PHost) Stop(ctx context.Context) error {
	h.startedMu.Lock()
	defer h.startedMu.Unlock()

	if !h.started {
		return nil
	}

	h.logger.Info("Stopping P2P host")

	// Cancel context to stop background operations
	h.cancel()

	// Close DHT
	if h.dht != nil {
		if err := h.dht.Close(); err != nil {
			h.logger.Warn("Error closing DHT", zap.Error(err))
		}
	}

	// Close host
	if h.host != nil {
		if err := h.host.Close(); err != nil {
			h.logger.Warn("Error closing host", zap.Error(err))
		}
	}

	h.started = false
	h.logger.Info("P2P host stopped")

	return nil
}

// Host returns the underlying libp2p host
func (h *LibP2PHost) Host() host.Host {
	return h.host
}

// Connect establishes a connection to a peer
func (h *LibP2PHost) Connect(ctx context.Context, addr peer.AddrInfo) error {
	if h.host == nil {
		return fmt.Errorf("P2P host not initialized")
	}

	addrs, _ := peer.AddrInfoToP2pAddrs(&addr)
	addrStrs := make([]string, len(addrs))
	for i, a := range addrs {
		addrStrs[i] = a.String()
	}
	h.logger.Debug("Connecting to peer",
		zap.String("peer_id", addr.ID.String()),
		zap.Strings("addresses", addrStrs))

	if err := h.host.Connect(ctx, addr); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %w", addr.ID, err)
	}

	h.logger.Info("Connected to peer", zap.String("peer_id", addr.ID.String()))
	return nil
}

// Disconnect closes connection to a peer
func (h *LibP2PHost) Disconnect(ctx context.Context, peerID peer.ID) error {
	if h.host == nil {
		return fmt.Errorf("P2P host not initialized")
	}

	h.logger.Debug("Disconnecting from peer", zap.String("peer_id", peerID.String()))

	if err := h.host.Network().ClosePeer(peerID); err != nil {
		return fmt.Errorf("failed to disconnect from peer %s: %w", peerID, err)
	}

	h.logger.Info("Disconnected from peer", zap.String("peer_id", peerID.String()))
	return nil
}

// GetPeers returns list of connected peers
func (h *LibP2PHost) GetPeers() []peer.ID {
	if h.host == nil {
		return nil
	}

	return h.host.Network().Peers()
}

// RegisterProtocolHandler registers a handler for a protocol
func (h *LibP2PHost) RegisterProtocolHandler(protocolID protocol.ID, handler types.ProtocolHandler) {
	h.handlersMutex.Lock()
	defer h.handlersMutex.Unlock()

	h.protocolHandlers[protocolID] = handler

	if h.host != nil {
		h.host.SetStreamHandler(protocolID, h.createStreamHandler(protocolID, handler))
	}

	h.logger.Info("Registered protocol handler", zap.String("protocol", string(protocolID)))
}

// SendMessage sends a message to a peer using specified protocol
func (h *LibP2PHost) SendMessage(ctx context.Context, peerID peer.ID, protocolID protocol.ID, data []byte) error {
	if h.host == nil {
		return fmt.Errorf("P2P host not initialized")
	}

	stream, err := h.host.NewStream(ctx, peerID, protocolID)
	if err != nil {
		return fmt.Errorf("failed to create stream to peer %s: %w", peerID, err)
	}
	defer stream.Close()

	if _, err := stream.Write(data); err != nil {
		return fmt.Errorf("failed to write data to stream: %w", err)
	}

	h.logger.Debug("Sent message to peer",
		zap.String("peer_id", peerID.String()),
		zap.String("protocol", string(protocolID)),
		zap.Int("bytes", len(data)))

	return nil
}

// GetLocalPeerInfo returns information about the local peer
func (h *LibP2PHost) GetLocalPeerInfo() *types.LocalPeerInfo {
	if h.host == nil {
		return nil
	}

	addresses := h.getHostAddresses()
	protocols := h.getHostProtocols()

	return &types.LocalPeerInfo{
		PeerID:    h.host.ID(),
		Addresses: addresses,
		Protocols: protocols,
	}
}

// Private helper methods

func (h *LibP2PHost) getOrCreatePrivateKey() (crypto.PrivKey, error) {
	// For now, generate a new key each time
	// In production, you would load from file or generate and save
	privateKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	return privateKey, nil
}

func (h *LibP2PHost) createConnectionManager() (interface{}, error) {
	// Use default connection manager with configured limits
	return nil, nil // libp2p will use default if nil
}

func (h *LibP2PHost) initializeDHT(ctx context.Context) error {
	var mode dht.ModeOpt
	switch h.config.DHTMode {
	case "client":
		mode = dht.ModeClient
	case "server":
		mode = dht.ModeServer
	default:
		mode = dht.ModeAuto
	}

	var err error
	h.dht, err = dht.New(ctx, h.host, dht.Mode(mode))
	if err != nil {
		return fmt.Errorf("failed to create DHT: %w", err)
	}

	if err := h.dht.Bootstrap(ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Create discovery service
	h.discovery = routing.NewRoutingDiscovery(h.dht)

	h.logger.Info("DHT initialized", zap.String("mode", h.config.DHTMode))
	return nil
}

func (h *LibP2PHost) setupProtocolHandlers() {
	h.handlersMutex.RLock()
	defer h.handlersMutex.RUnlock()

	for protocolID, handler := range h.protocolHandlers {
		h.host.SetStreamHandler(protocolID, h.createStreamHandler(protocolID, handler))
	}
}

func (h *LibP2PHost) createStreamHandler(protocolID protocol.ID, handler types.ProtocolHandler) network.StreamHandler {
	return func(stream network.Stream) {
		defer stream.Close()

		ctx, cancel := context.WithTimeout(h.ctx, 30*time.Second)
		defer cancel()

		streamWrapper := &streamWrapper{
			stream:   stream,
			protocol: protocolID,
		}

		if err := handler.HandleProtocol(ctx, streamWrapper); err != nil {
			h.logger.Error("Protocol handler error",
				zap.String("protocol", string(protocolID)),
				zap.String("peer", stream.Conn().RemotePeer().String()),
				zap.Error(err))
		}
	}
}

func (h *LibP2PHost) connectToBootstrapPeers(ctx context.Context) {
	for _, peerAddr := range h.config.BootstrapPeers {
		addr, err := peer.AddrInfoFromString(peerAddr)
		if err != nil {
			h.logger.Warn("Invalid bootstrap peer address",
				zap.String("address", peerAddr),
				zap.Error(err))
			continue
		}

		if err := h.Connect(ctx, *addr); err != nil {
			h.logger.Warn("Failed to connect to bootstrap peer",
				zap.String("peer_id", addr.ID.String()),
				zap.Error(err))
		}
	}
}

func (h *LibP2PHost) getHostAddresses() []string {
	addrs := h.host.Addrs()
	result := make([]string, len(addrs))
	for i, addr := range addrs {
		result[i] = addr.String()
	}
	return result
}

func (h *LibP2PHost) getHostProtocols() []string {
	protocols := h.host.Mux().Protocols()
	result := make([]string, len(protocols))
	for i, p := range protocols {
		result[i] = string(p)
	}
	return result
}

// streamWrapper implements the StreamHandler interface
type streamWrapper struct {
	stream   network.Stream
	protocol protocol.ID
}

func (s *streamWrapper) Read(b []byte) (int, error) {
	return s.stream.Read(b)
}

func (s *streamWrapper) Write(b []byte) (int, error) {
	return s.stream.Write(b)
}

func (s *streamWrapper) Close() error {
	return s.stream.Close()
}

func (s *streamWrapper) Protocol() protocol.ID {
	return s.protocol
}

func (s *streamWrapper) RemotePeer() peer.ID {
	return s.stream.Conn().RemotePeer()
}