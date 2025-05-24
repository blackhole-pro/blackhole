// Package discovery implements peer discovery mechanisms
package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"node/types"
	"go.uber.org/zap"
)

// Discovery handles peer discovery operations
type Discovery struct {
	mu               sync.RWMutex
	config           *types.NodeConfig
	logger           *zap.Logger
	discoveredPeers  []types.DiscoveredPeer
	isRunning        bool
	stopCh           chan struct{}
	discoveryWorkers sync.WaitGroup
}

// NewDiscovery creates a new discovery instance
func NewDiscovery(config *types.NodeConfig, logger *zap.Logger) *Discovery {
	return &Discovery{
		config:          config,
		logger:          logger,
		discoveredPeers: make([]types.DiscoveredPeer, 0),
	}
}

// DiscoverPeers performs peer discovery using the specified method
func (d *Discovery) DiscoverPeers(ctx context.Context, method string, maxPeers int) ([]types.DiscoveredPeer, error) {
	if !d.config.EnableDiscovery {
		return nil, types.ErrDiscoveryDisabled
	}

	if method == "" {
		method = d.config.DiscoveryMethod
	}

	d.logger.Info("Starting peer discovery",
		zap.String("method", method),
		zap.Int("maxPeers", maxPeers))

	var peers []types.DiscoveredPeer
	var err error

	switch method {
	case "mdns":
		peers, err = d.discoverMDNS(ctx, maxPeers)
	case "dht":
		peers, err = d.discoverDHT(ctx, maxPeers)
	case "bootstrap":
		peers, err = d.discoverBootstrap(ctx, maxPeers)
	default:
		return nil, types.NewDiscoveryError(method, 
			fmt.Errorf("%w: %s", types.ErrInvalidDiscoveryMethod, method))
	}

	if err != nil {
		return nil, err
	}

	// Store discovered peers
	d.mu.Lock()
	d.discoveredPeers = append(d.discoveredPeers, peers...)
	d.deduplicatePeers()
	d.mu.Unlock()

	return peers, nil
}

// StartDiscovery starts continuous peer discovery
func (d *Discovery) StartDiscovery(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isRunning {
		return fmt.Errorf("discovery already running")
	}

	if !d.config.EnableDiscovery {
		return types.ErrDiscoveryDisabled
	}

	d.isRunning = true
	d.stopCh = make(chan struct{})

	// Start discovery worker
	d.discoveryWorkers.Add(1)
	go d.discoveryWorker(ctx)

	d.logger.Info("Discovery started",
		zap.String("method", d.config.DiscoveryMethod),
		zap.Duration("interval", d.config.DiscoveryInterval))

	return nil
}

// StopDiscovery stops continuous peer discovery
func (d *Discovery) StopDiscovery(ctx context.Context) error {
	d.mu.Lock()
	if !d.isRunning {
		d.mu.Unlock()
		return fmt.Errorf("discovery not running")
	}

	d.isRunning = false
	close(d.stopCh)
	d.mu.Unlock()

	// Wait for workers to stop
	done := make(chan struct{})
	go func() {
		d.discoveryWorkers.Wait()
		close(done)
	}()

	select {
	case <-done:
		d.logger.Info("Discovery stopped gracefully")
	case <-time.After(5 * time.Second):
		d.logger.Warn("Discovery stop timeout")
	case <-ctx.Done():
		d.logger.Warn("Discovery stop cancelled")
	}

	return nil
}

// GetDiscoveredPeers returns the list of discovered peers
func (d *Discovery) GetDiscoveredPeers() []types.DiscoveredPeer {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return a copy
	peers := make([]types.DiscoveredPeer, len(d.discoveredPeers))
	copy(peers, d.discoveredPeers)
	return peers
}

// discoveryWorker runs periodic peer discovery
func (d *Discovery) discoveryWorker(ctx context.Context) {
	defer d.discoveryWorkers.Done()

	ticker := time.NewTicker(d.config.DiscoveryInterval)
	defer ticker.Stop()

	// Initial discovery
	d.runDiscovery(ctx)

	for {
		select {
		case <-ticker.C:
			d.runDiscovery(ctx)
		case <-d.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// runDiscovery performs a single discovery round
func (d *Discovery) runDiscovery(ctx context.Context) {
	peers, err := d.DiscoverPeers(ctx, d.config.DiscoveryMethod, 10)
	if err != nil {
		d.logger.Error("Discovery failed", zap.Error(err))
		return
	}

	d.mu.Lock()
	d.discoveredPeers = append(d.discoveredPeers, peers...)
	// Keep only unique peers (simple deduplication)
	d.deduplicatePeers()
	d.mu.Unlock()

	d.logger.Debug("Discovery completed",
		zap.Int("foundPeers", len(peers)),
		zap.Int("totalDiscovered", len(d.discoveredPeers)))
}

// discoverMDNS performs mDNS-based local discovery
func (d *Discovery) discoverMDNS(ctx context.Context, maxPeers int) ([]types.DiscoveredPeer, error) {
	// Simulate mDNS discovery
	// In real implementation, use go-libp2p mDNS service

	d.logger.Debug("Running mDNS discovery")

	// Simulate finding local peers
	peers := make([]types.DiscoveredPeer, 0, maxPeers)
	for i := 0; i < maxPeers && i < 2; i++ {
		peers = append(peers, types.DiscoveredPeer{
			ID:      fmt.Sprintf("mdns-peer-%d-%d", time.Now().Unix(), i),
			Address: fmt.Sprintf("192.168.1.%d:4001", 100+i),
			Source:  "mdns",
		})
	}

	if len(peers) > maxPeers {
		peers = peers[:maxPeers]
	}

	return peers, nil
}

// discoverDHT performs DHT-based discovery
func (d *Discovery) discoverDHT(ctx context.Context, maxPeers int) ([]types.DiscoveredPeer, error) {
	// Simulate DHT discovery
	// In real implementation, use libp2p Kademlia DHT

	d.logger.Debug("Running DHT discovery")

	// Simulate finding DHT peers
	peers := make([]types.DiscoveredPeer, 0, maxPeers)
	for i := 0; i < maxPeers && i < 3; i++ {
		peers = append(peers, types.DiscoveredPeer{
			ID:      fmt.Sprintf("dht-peer-%d-%d", time.Now().Unix(), i),
			Address: fmt.Sprintf("203.0.113.%d:4001", 10+i*10),
			Source:  "dht",
		})
	}

	if len(peers) > maxPeers {
		peers = peers[:maxPeers]
	}

	return peers, nil
}

// discoverBootstrap uses bootstrap nodes for discovery
func (d *Discovery) discoverBootstrap(ctx context.Context, maxPeers int) ([]types.DiscoveredPeer, error) {
	if len(d.config.BootstrapPeers) == 0 {
		return nil, types.NewDiscoveryError("bootstrap", types.ErrNoBootstrapPeers)
	}

	d.logger.Debug("Running bootstrap discovery",
		zap.Int("bootstrapNodes", len(d.config.BootstrapPeers)))

	peers := make([]types.DiscoveredPeer, 0, len(d.config.BootstrapPeers))
	for i, addr := range d.config.BootstrapPeers {
		if i >= maxPeers {
			break
		}
		peers = append(peers, types.DiscoveredPeer{
			ID:      fmt.Sprintf("bootstrap-%d", i),
			Address: addr,
			Source:  "bootstrap",
		})
	}

	return peers, nil
}

// deduplicatePeers removes duplicate discovered peers
func (d *Discovery) deduplicatePeers() {
	seen := make(map[string]bool)
	unique := make([]types.DiscoveredPeer, 0, len(d.discoveredPeers))

	for _, peer := range d.discoveredPeers {
		key := peer.ID + peer.Address
		if !seen[key] {
			seen[key] = true
			unique = append(unique, peer)
		}
	}

	d.discoveredPeers = unique
}

// ClearDiscoveredPeers clears the discovered peers list
func (d *Discovery) ClearDiscoveredPeers() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.discoveredPeers = make([]types.DiscoveredPeer, 0)
}