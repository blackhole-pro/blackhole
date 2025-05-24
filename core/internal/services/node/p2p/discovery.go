package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"go.uber.org/zap"

	"github.com/blackhole-pro/blackhole/core/internal/services/node/types"
)

// DiscoveryService handles peer discovery using various methods
type DiscoveryService struct {
	host      host.Host
	dht       *dht.IpfsDHT
	discovery *routing.RoutingDiscovery
	logger    *zap.Logger

	// Configuration
	namespace string
	
	// State
	ctx       context.Context
	cancel    context.CancelFunc
	started   bool
	startedMu sync.RWMutex
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(host host.Host, dht *dht.IpfsDHT, logger *zap.Logger) *DiscoveryService {
	ctx, cancel := context.WithCancel(context.Background())
	
	var discovery *routing.RoutingDiscovery
	if dht != nil {
		discovery = routing.NewRoutingDiscovery(dht)
	}

	return &DiscoveryService{
		host:      host,
		dht:       dht,
		discovery: discovery,
		logger:    logger,
		namespace: "blackhole",
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins the discovery service
func (d *DiscoveryService) Start(ctx context.Context) error {
	d.startedMu.Lock()
	defer d.startedMu.Unlock()

	if d.started {
		return fmt.Errorf("discovery service already started")
	}

	d.logger.Info("Starting peer discovery service")

	// Start advertising this node
	if d.discovery != nil {
		go d.advertiseLoop()
	}

	d.started = true
	d.logger.Info("Peer discovery service started")

	return nil
}

// Stop stops the discovery service
func (d *DiscoveryService) Stop() error {
	d.startedMu.Lock()
	defer d.startedMu.Unlock()

	if !d.started {
		return nil
	}

	d.logger.Info("Stopping peer discovery service")
	d.cancel()
	d.started = false
	d.logger.Info("Peer discovery service stopped")

	return nil
}

// DiscoverPeers discovers peers using the specified method
func (d *DiscoveryService) DiscoverPeers(ctx context.Context, request *types.DiscoveryRequest) (*types.DiscoveryResponse, error) {
	start := time.Now()
	
	d.logger.Info("Starting peer discovery",
		zap.String("method", request.Method),
		zap.Int("max_peers", request.MaxPeers))

	var discoveredAddresses []string
	var err error

	switch request.Method {
	case "dht":
		discoveredAddresses, err = d.discoverViaDHT(ctx, request)
	case "bootstrap":
		discoveredAddresses, err = d.discoverViaBootstrap(ctx, request)
	case "connected":
		discoveredAddresses, err = d.discoverConnectedPeers(ctx, request)
	default:
		return nil, fmt.Errorf("unsupported discovery method: %s", request.Method)
	}

	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	// Apply filter if provided
	if request.FilterFunc != nil {
		filtered := make([]string, 0, len(discoveredAddresses))
		for _, addr := range discoveredAddresses {
			if request.FilterFunc(addr) {
				filtered = append(filtered, addr)
			}
		}
		discoveredAddresses = filtered
	}

	// Limit results if requested
	if request.MaxPeers > 0 && len(discoveredAddresses) > request.MaxPeers {
		discoveredAddresses = discoveredAddresses[:request.MaxPeers]
	}

	duration := time.Since(start)

	d.logger.Info("Peer discovery completed",
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

// GetConnectedPeers returns currently connected peers
func (d *DiscoveryService) GetConnectedPeers() []peer.ID {
	return d.host.Network().Peers()
}

// Private methods

func (d *DiscoveryService) advertiseLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	// Advertise immediately
	d.advertise()

	for {
		select {
		case <-ticker.C:
			d.advertise()
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *DiscoveryService) advertise() {
	if d.discovery == nil {
		return
	}

	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()

	ttl, err := d.discovery.Advertise(ctx, d.namespace)
	if err != nil {
		d.logger.Warn("Failed to advertise node", zap.Error(err))
		return
	}

	d.logger.Debug("Advertised node in DHT",
		zap.String("namespace", d.namespace),
		zap.Duration("ttl", ttl))
}

func (d *DiscoveryService) discoverViaDHT(ctx context.Context, request *types.DiscoveryRequest) ([]string, error) {
	if d.discovery == nil {
		return nil, fmt.Errorf("DHT discovery not available")
	}

	// Set timeout for discovery
	discoveryCtx := ctx
	if request.Timeout > 0 {
		var cancel context.CancelFunc
		discoveryCtx, cancel = context.WithTimeout(ctx, request.Timeout)
		defer cancel()
	}

	// Find peers
	peerChan, err := d.discovery.FindPeers(discoveryCtx, d.namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to start peer discovery: %w", err)
	}

	var discoveredAddresses []string
	maxPeers := request.MaxPeers
	if maxPeers <= 0 {
		maxPeers = 50 // Default limit
	}

	for peer := range peerChan {
		if len(discoveredAddresses) >= maxPeers {
			break
		}

		// Skip self
		if peer.ID == d.host.ID() {
			continue
		}

		// Convert peer info to addresses
		for _, addr := range peer.Addrs {
			fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), peer.ID.String())
			discoveredAddresses = append(discoveredAddresses, fullAddr)
		}
	}

	return discoveredAddresses, nil
}

func (d *DiscoveryService) discoverViaBootstrap(ctx context.Context, request *types.DiscoveryRequest) ([]string, error) {
	// This would typically return configured bootstrap peers
	// For now, return empty list as bootstrap peers are handled during connection
	return []string{}, nil
}

func (d *DiscoveryService) discoverConnectedPeers(ctx context.Context, request *types.DiscoveryRequest) ([]string, error) {
	peers := d.host.Network().Peers()
	discoveredAddresses := make([]string, 0, len(peers))

	for _, peerID := range peers {
		// Get peer addresses from peerstore
		addrs := d.host.Peerstore().Addrs(peerID)
		for _, addr := range addrs {
			fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), peerID.String())
			discoveredAddresses = append(discoveredAddresses, fullAddr)
		}
	}

	return discoveredAddresses, nil
}

// StartPeriodicDiscovery provides utility functions for peer discovery
func (d *DiscoveryService) StartPeriodicDiscovery(ctx context.Context, namespace string) error {
	if d.discovery == nil {
		return fmt.Errorf("DHT discovery not available")
	}

	// Start periodic advertising
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := d.discovery.Advertise(ctx, namespace); err != nil {
					d.logger.Warn("Failed to advertise", zap.Error(err))
				}
			}
		}
	}()

	// Start discovering peers
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				peerChan, err := d.discovery.FindPeers(ctx, namespace)
				if err != nil {
					d.logger.Warn("Failed to discover peers", zap.Error(err))
					continue
				}

				for peerInfo := range peerChan {
					if peerInfo.ID == d.host.ID() {
						continue
					}

					d.logger.Info("Discovered peer",
						zap.String("peer_id", peerInfo.ID.String()),
						zap.Int("addresses", len(peerInfo.Addrs)))

					// Attempt to connect to discovered peer
					connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					if err := d.host.Connect(connectCtx, peerInfo); err != nil {
						d.logger.Debug("Failed to connect to discovered peer",
							zap.String("peer_id", peerInfo.ID.String()),
							zap.Error(err))
					} else {
						d.logger.Info("Connected to discovered peer",
							zap.String("peer_id", peerInfo.ID.String()))
					}
					cancel()
				}
			}
		}
	}()

	return nil
}