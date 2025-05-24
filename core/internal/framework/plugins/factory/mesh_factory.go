// Package factory provides factory methods for creating mesh-based plugin managers
package factory

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/blackhole-pro/blackhole/core/internal/framework/mesh"
	"github.com/blackhole-pro/blackhole/core/internal/framework/mesh/routing"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/lifecycle"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/loader"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/registry"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/state"
)

// MeshPluginManagerFactory creates mesh-based plugin managers
type MeshPluginManagerFactory struct {
	meshNetwork    mesh.MeshNetwork
	protocolRouter *routing.ProtocolRouter
	config         MeshPluginConfig
	logger         *zap.Logger
}

// MeshPluginConfig configures the mesh plugin system
type MeshPluginConfig struct {
	// Directories
	PluginDir  string // Where plugins are installed
	CacheDir   string // Where downloaded plugins are cached
	StateDir   string // Where plugin state is stored
	SocketDir  string // Where plugin sockets are created
	TempDir    string // Temporary directory for operations
	
	// Networking
	EnableDiscovery bool   // Enable automatic plugin discovery
	MeshEndpoint   string // Endpoint for mesh network
	
	// Defaults
	DefaultIsolation plugins.IsolationLevel
	DefaultTimeout   int // seconds
}

// DefaultMeshPluginConfig returns default configuration
func DefaultMeshPluginConfig() MeshPluginConfig {
	return MeshPluginConfig{
		PluginDir:        "/usr/local/lib/blackhole/plugins",
		CacheDir:         "/var/cache/blackhole/plugins",
		StateDir:         "/var/lib/blackhole/plugins",
		SocketDir:        "/var/run/blackhole/plugins",
		TempDir:          "/tmp/blackhole/plugins",
		EnableDiscovery:  true,
		DefaultIsolation: plugins.IsolationLevelProcess,
		DefaultTimeout:   30,
	}
}

// NewMeshPluginManagerFactory creates a new factory for mesh-based plugin managers
func NewMeshPluginManagerFactory(
	meshNetwork mesh.MeshNetwork,
	protocolRouter *routing.ProtocolRouter,
	config MeshPluginConfig,
	logger *zap.Logger,
) *MeshPluginManagerFactory {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &MeshPluginManagerFactory{
		meshNetwork:    meshNetwork,
		protocolRouter: protocolRouter,
		config:         config,
		logger:         logger,
	}
}

// CreatePluginManager creates a new mesh-based plugin manager
func (f *MeshPluginManagerFactory) CreatePluginManager() (plugins.PluginManager, error) {
	// Create registry
	registryConfig := registry.Config{
		LocalPath:     f.config.PluginDir,
		ScanInterval:  0, // Disable automatic scanning for now
		MarketplaceURL: "", // No marketplace yet
	}
	pluginRegistry := registry.New(registryConfig)

	// Create mesh-aware loader
	loaderConfig := loader.MeshLoaderConfig{
		LocalPath:  f.config.PluginDir,
		CacheDir:   f.config.CacheDir,
		TempDir:    f.config.TempDir,
		SocketDir:  f.config.SocketDir,
		MeshClient: nil, // TODO: Create mesh client
		Logger:     f.logger.With(zap.String("component", "loader")),
	}
	pluginLoader := loader.NewMeshPluginLoader(loaderConfig)

	// Create state manager
	stateStorage := state.NewFileStorage(f.config.StateDir)
	stateManager := state.NewManager(stateStorage, state.NewJSONSerializer())

	// Create lifecycle manager
	lifecycleManager := lifecycle.New()

	// Create the mesh plugin manager
	managerConfig := plugins.MeshPluginManagerConfig{
		Registry:       pluginRegistry,
		Loader:         pluginLoader,
		StateManager:   stateManager,
		Lifecycle:      lifecycleManager,
		MeshNetwork:    f.meshNetwork,
		ProtocolRouter: f.protocolRouter,
		SocketDir:      f.config.SocketDir,
		Logger:         f.logger.With(zap.String("component", "manager")),
	}

	manager := plugins.NewMeshPluginManager(managerConfig)

	f.logger.Info("Created mesh-based plugin manager",
		zap.String("plugin_dir", f.config.PluginDir),
		zap.String("socket_dir", f.config.SocketDir),
		zap.Bool("discovery", f.config.EnableDiscovery))

	return manager, nil
}

// CreateMockPluginManager creates a mock plugin manager for testing
func (f *MeshPluginManagerFactory) CreateMockPluginManager() plugins.PluginManager {
	return NewMockPluginManager()
}

// GetDefaultPluginSpec creates a default plugin specification
func GetDefaultPluginSpec(name, version string) plugins.PluginSpec {
	return plugins.PluginSpec{
		Name:        name,
		Version:     version,
		Description: fmt.Sprintf("%s plugin", name),
		Author:      "Unknown",
		License:     "Unknown",
		Source: plugins.PluginSource{
			Type: plugins.SourceTypeLocal,
			Path: fmt.Sprintf("/usr/local/lib/blackhole/plugins/%s/%s/plugin", name, version),
		},
		Isolation: plugins.IsolationLevelProcess,
		Resources: plugins.PluginResources{
			CPUShares:    50,  // 50% of one CPU
			MemoryMB:     256, // 256MB RAM
			MaxGoroutines: 100,
		},
		Permissions: []plugins.PluginPermission{
			{
				Resource: "network",
				Actions:  []string{"connect"},
			},
		},
	}
}

// LoadPluginFromPath creates a plugin spec from a local path
func LoadPluginFromPath(path string) (plugins.PluginSpec, error) {
	// This would load plugin.json or similar from the path
	// For now, return a basic spec
	return plugins.PluginSpec{
		Name:    "unknown",
		Version: "0.0.0",
		Source: plugins.PluginSource{
			Type: plugins.SourceTypeLocal,
			Path: path,
		},
		Isolation: plugins.IsolationLevelProcess,
	}, nil
}