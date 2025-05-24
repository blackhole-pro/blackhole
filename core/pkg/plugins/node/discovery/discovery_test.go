package discovery_test

import (
	"context"
	"testing"
	"time"

	"node/discovery"
	"node/types"
	"go.uber.org/zap/zaptest"
)

func TestDiscovery_DiscoverPeers(t *testing.T) {
	tests := []struct {
		name      string
		config    *types.NodeConfig
		method    string
		maxPeers  int
		wantErr   bool
		errType   error
	}{
		{
			name: "discovery disabled",
			config: &types.NodeConfig{
				EnableDiscovery: false,
			},
			method:   "mdns",
			maxPeers: 10,
			wantErr:  true,
			errType:  types.ErrDiscoveryDisabled,
		},
		{
			name: "mdns discovery",
			config: &types.NodeConfig{
				EnableDiscovery: true,
				DiscoveryMethod: "mdns",
			},
			method:   "mdns",
			maxPeers: 10,
			wantErr:  false,
		},
		{
			name: "dht discovery",
			config: &types.NodeConfig{
				EnableDiscovery: true,
				DiscoveryMethod: "dht",
			},
			method:   "dht",
			maxPeers: 10,
			wantErr:  false,
		},
		{
			name: "bootstrap discovery with peers",
			config: &types.NodeConfig{
				EnableDiscovery: true,
				DiscoveryMethod: "bootstrap",
				BootstrapPeers:  []string{"peer1", "peer2", "peer3"},
			},
			method:   "bootstrap",
			maxPeers: 2,
			wantErr:  false,
		},
		{
			name: "bootstrap discovery without peers",
			config: &types.NodeConfig{
				EnableDiscovery: true,
				DiscoveryMethod: "bootstrap",
				BootstrapPeers:  []string{},
			},
			method:   "bootstrap",
			maxPeers: 10,
			wantErr:  true,
		},
		{
			name: "invalid discovery method",
			config: &types.NodeConfig{
				EnableDiscovery: true,
			},
			method:   "invalid",
			maxPeers: 10,
			wantErr:  true,
		},
		{
			name: "default method from config",
			config: &types.NodeConfig{
				EnableDiscovery: true,
				DiscoveryMethod: "mdns",
			},
			method:   "", // Use default from config
			maxPeers: 10,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			d := discovery.NewDiscovery(tt.config, logger)

			ctx := context.Background()
			peers, err := d.DiscoverPeers(ctx, tt.method, tt.maxPeers)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check results for specific methods
			switch tt.method {
			case "bootstrap", "":
				if tt.config.DiscoveryMethod == "bootstrap" && len(tt.config.BootstrapPeers) > 0 {
					if len(peers) == 0 {
						t.Error("Expected to discover bootstrap peers")
					}
					if len(peers) > tt.maxPeers {
						t.Errorf("Expected at most %d peers, got %d", tt.maxPeers, len(peers))
					}
				}
			default:
				// For mdns and dht, we just check that some peers were returned
				if len(peers) == 0 {
					t.Error("Expected to discover some peers")
				}
			}
		})
	}
}

func TestDiscovery_StartStop(t *testing.T) {
	config := &types.NodeConfig{
		EnableDiscovery:   true,
		DiscoveryMethod:   "mdns",
		DiscoveryInterval: 100 * time.Millisecond,
	}

	logger := zaptest.NewLogger(t)
	d := discovery.NewDiscovery(config, logger)

	ctx := context.Background()

	// Test starting discovery
	err := d.StartDiscovery(ctx)
	if err != nil {
		t.Fatalf("Failed to start discovery: %v", err)
	}

	// Test starting when already running
	err = d.StartDiscovery(ctx)
	if err == nil {
		t.Error("Expected error when starting already running discovery")
	}

	// Let it run for a bit
	time.Sleep(250 * time.Millisecond)

	// Check that peers were discovered
	peers := d.GetDiscoveredPeers()
	if len(peers) == 0 {
		t.Error("Expected to discover some peers during continuous discovery")
	}

	// Test stopping discovery
	err = d.StopDiscovery(ctx)
	if err != nil {
		t.Errorf("Failed to stop discovery: %v", err)
	}

	// Test stopping when not running
	err = d.StopDiscovery(ctx)
	if err == nil {
		t.Error("Expected error when stopping non-running discovery")
	}
}

func TestDiscovery_StartWithDisabled(t *testing.T) {
	config := &types.NodeConfig{
		EnableDiscovery: false,
	}

	logger := zaptest.NewLogger(t)
	d := discovery.NewDiscovery(config, logger)

	ctx := context.Background()
	err := d.StartDiscovery(ctx)
	if err != types.ErrDiscoveryDisabled {
		t.Errorf("Expected ErrDiscoveryDisabled, got: %v", err)
	}
}

func TestDiscovery_GetDiscoveredPeers(t *testing.T) {
	config := &types.NodeConfig{
		EnableDiscovery: true,
		DiscoveryMethod: "mdns",
	}

	logger := zaptest.NewLogger(t)
	d := discovery.NewDiscovery(config, logger)

	// Initially should be empty
	peers := d.GetDiscoveredPeers()
	if len(peers) != 0 {
		t.Error("Expected no discovered peers initially")
	}

	// Discover some peers
	ctx := context.Background()
	_, err := d.DiscoverPeers(ctx, "mdns", 5)
	if err != nil {
		t.Fatalf("Failed to discover peers: %v", err)
	}

	// Should have some peers now
	peers = d.GetDiscoveredPeers()
	if len(peers) == 0 {
		t.Error("Expected some discovered peers after discovery")
	}

	// Test that we get a copy (modifying returned slice shouldn't affect internal state)
	originalLen := len(peers)
	peers = append(peers, types.DiscoveredPeer{ID: "test", Address: "test", Source: "test"})
	
	newPeers := d.GetDiscoveredPeers()
	if len(newPeers) != originalLen {
		t.Error("Modifying returned peers affected internal state")
	}
}

func TestDiscovery_ClearDiscoveredPeers(t *testing.T) {
	config := &types.NodeConfig{
		EnableDiscovery: true,
		DiscoveryMethod: "mdns",
	}

	logger := zaptest.NewLogger(t)
	d := discovery.NewDiscovery(config, logger)

	// Discover some peers
	ctx := context.Background()
	_, err := d.DiscoverPeers(ctx, "mdns", 5)
	if err != nil {
		t.Fatalf("Failed to discover peers: %v", err)
	}

	// Verify we have peers
	if len(d.GetDiscoveredPeers()) == 0 {
		t.Fatal("Expected to have discovered peers before clearing")
	}

	// Clear peers
	d.ClearDiscoveredPeers()

	// Verify cleared
	if len(d.GetDiscoveredPeers()) != 0 {
		t.Error("Expected no peers after clearing")
	}
}

func TestDiscovery_StopTimeout(t *testing.T) {
	config := &types.NodeConfig{
		EnableDiscovery:   true,
		DiscoveryMethod:   "mdns",
		DiscoveryInterval: 10 * time.Millisecond,
	}

	logger := zaptest.NewLogger(t)
	d := discovery.NewDiscovery(config, logger)

	ctx := context.Background()

	// Start discovery
	err := d.StartDiscovery(ctx)
	if err != nil {
		t.Fatalf("Failed to start discovery: %v", err)
	}

	// Create a context that will timeout quickly
	stopCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	// Stop should handle the timeout gracefully
	err = d.StopDiscovery(stopCtx)
	if err != nil {
		t.Errorf("Stop failed with timeout: %v", err)
	}
}

func TestDiscovery_Deduplication(t *testing.T) {
	config := &types.NodeConfig{
		EnableDiscovery:   true,
		DiscoveryMethod:   "mdns",
		DiscoveryInterval: 50 * time.Millisecond,
	}

	logger := zaptest.NewLogger(t)
	d := discovery.NewDiscovery(config, logger)

	ctx := context.Background()

	// Start continuous discovery
	err := d.StartDiscovery(ctx)
	if err != nil {
		t.Fatalf("Failed to start discovery: %v", err)
	}

	// Let it run for multiple intervals
	time.Sleep(200 * time.Millisecond)

	// Stop discovery
	err = d.StopDiscovery(ctx)
	if err != nil {
		t.Errorf("Failed to stop discovery: %v", err)
	}

	// Check that peers are deduplicated
	peers := d.GetDiscoveredPeers()
	seen := make(map[string]bool)
	for _, peer := range peers {
		key := peer.ID + peer.Address
		if seen[key] {
			t.Errorf("Found duplicate peer: %s at %s", peer.ID, peer.Address)
		}
		seen[key] = true
	}
}