// Package plugins provides a mesh-based plugin manager
package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/blackhole-pro/blackhole/core/internal/framework/mesh"
	"github.com/blackhole-pro/blackhole/core/internal/framework/mesh/routing"
)

// MeshPluginManager manages plugins using mesh network communication
type MeshPluginManager struct {
	// Core components
	registry  PluginRegistry
	loader    PluginLoader
	state     StateManager
	lifecycle PluginLifecycle
	
	// Mesh networking
	meshNetwork    mesh.MeshNetwork
	protocolRouter *routing.ProtocolRouter
	
	// Plugin tracking
	plugins   map[string]*ManagedMeshPlugin
	mu        sync.RWMutex
	
	// Configuration
	socketDir string
	logger    *zap.Logger
}

// ManagedMeshPlugin wraps a plugin with mesh connectivity
type ManagedMeshPlugin struct {
	// Plugin info
	spec        PluginSpec
	plugin      Plugin
	
	// Mesh connectivity
	endpoint    mesh.ServiceEndpoint
	grpcConn    *grpc.ClientConn
	serviceName string
	
	// Management metadata
	startTime    time.Time
	restartCount int
	lastError    error
	
	// Process management
	process     *PluginProcess
	
	mu          sync.RWMutex
}

// PluginProcess represents a plugin OS process
type PluginProcess struct {
	PID        int
	SocketPath string
	Started    time.Time
	Status     string
}

// MeshPluginManagerConfig configures the mesh-based plugin manager
type MeshPluginManagerConfig struct {
	Registry       PluginRegistry
	Loader         PluginLoader
	StateManager   StateManager
	Lifecycle      PluginLifecycle
	MeshNetwork    mesh.MeshNetwork
	ProtocolRouter *routing.ProtocolRouter
	SocketDir      string
	Logger         *zap.Logger
}

// NewMeshPluginManager creates a new mesh-based plugin manager
func NewMeshPluginManager(config MeshPluginManagerConfig) *MeshPluginManager {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	if config.SocketDir == "" {
		config.SocketDir = "/tmp/blackhole/plugins"
	}

	return &MeshPluginManager{
		registry:       config.Registry,
		loader:         config.Loader,
		state:          config.StateManager,
		lifecycle:      config.Lifecycle,
		meshNetwork:    config.MeshNetwork,
		protocolRouter: config.ProtocolRouter,
		plugins:        make(map[string]*ManagedMeshPlugin),
		socketDir:      config.SocketDir,
		logger:         config.Logger,
	}
}

// LoadPlugin loads a plugin and connects it to the mesh
func (m *MeshPluginManager) LoadPlugin(spec PluginSpec) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already loaded
	if _, exists := m.plugins[spec.Name]; exists {
		return ErrPluginAlreadyLoaded
	}

	m.logger.Info("Loading plugin", 
		zap.String("name", spec.Name),
		zap.String("version", spec.Version))

	// Validate the specification
	if err := m.loader.ValidatePlugin(spec); err != nil {
		return fmt.Errorf("plugin validation failed: %w", err)
	}

	// Load the plugin binary/configuration
	plugin, err := m.loader.LoadPlugin(spec)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Create managed plugin
	mp := &ManagedMeshPlugin{
		spec:        spec,
		plugin:      plugin,
		serviceName: fmt.Sprintf("plugin.%s", spec.Name),
	}

	// Start the plugin process
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := plugin.Start(ctx); err != nil {
		return fmt.Errorf("failed to start plugin: %w", err)
	}

	mp.startTime = time.Now()

	// Wait for mesh registration
	if err := m.waitForPluginRegistration(ctx, mp); err != nil {
		plugin.Stop(context.Background())
		return fmt.Errorf("plugin failed to register with mesh: %w", err)
	}

	// Connect to plugin via mesh
	if err := m.connectToPlugin(ctx, mp); err != nil {
		plugin.Stop(context.Background())
		return fmt.Errorf("failed to connect to plugin: %w", err)
	}

	// Notify lifecycle
	if m.lifecycle != nil {
		if err := m.lifecycle.OnPluginLoad(plugin); err != nil {
			m.logger.Warn("Lifecycle onload failed", zap.Error(err))
		}
	}

	// Store the managed plugin
	m.plugins[spec.Name] = mp

	m.logger.Info("Plugin loaded successfully",
		zap.String("name", spec.Name),
		zap.String("service", mp.serviceName))

	return nil
}

// UnloadPlugin unloads a plugin and removes it from mesh
func (m *MeshPluginManager) UnloadPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mp, exists := m.plugins[name]
	if !exists {
		return ErrPluginNotFound
	}

	m.logger.Info("Unloading plugin", zap.String("name", name))

	// Notify lifecycle
	if m.lifecycle != nil {
		if err := m.lifecycle.OnPluginUnload(mp.plugin); err != nil {
			m.logger.Warn("Lifecycle onunload failed", zap.Error(err))
		}
	}

	// Disconnect from mesh
	if mp.grpcConn != nil {
		mp.grpcConn.Close()
	}

	// Stop the plugin
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := mp.plugin.Stop(ctx); err != nil {
		m.logger.Warn("Error stopping plugin", zap.Error(err))
	}

	// Remove from registry
	delete(m.plugins, name)

	m.logger.Info("Plugin unloaded", zap.String("name", name))
	return nil
}

// ExecutePlugin executes a plugin request via mesh
func (m *MeshPluginManager) ExecutePlugin(name string, request PluginRequest) (PluginResponse, error) {
	m.mu.RLock()
	mp, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return PluginResponse{}, ErrPluginNotFound
	}

	// Route through mesh network
	// The specific method depends on the plugin's gRPC interface
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// For now, we use the plugin's Handle method
	// In reality, each plugin type would have its own gRPC interface
	return mp.plugin.Handle(ctx, request)
}

// ListPlugins returns information about all loaded plugins
func (m *MeshPluginManager) ListPlugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]PluginInfo, 0, len(m.plugins))
	for _, mp := range m.plugins {
		info := mp.plugin.Info()
		// Add mesh-specific information
		info.LoadTime = mp.startTime
		info.Uptime = time.Since(mp.startTime)
		infos = append(infos, info)
	}
	return infos
}

// GetPlugin returns information about a specific plugin
func (m *MeshPluginManager) GetPlugin(name string) (PluginInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp, exists := m.plugins[name]
	if !exists {
		return PluginInfo{}, ErrPluginNotFound
	}

	info := mp.plugin.Info()
	info.LoadTime = mp.startTime
	info.Uptime = time.Since(mp.startTime)
	return info, nil
}

// GetPluginConnection returns the gRPC connection to a plugin
func (m *MeshPluginManager) GetPluginConnection(name string) (*grpc.ClientConn, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp, exists := m.plugins[name]
	if !exists {
		return nil, ErrPluginNotFound
	}

	if mp.grpcConn == nil {
		return nil, fmt.Errorf("plugin %s has no mesh connection", name)
	}

	return mp.grpcConn, nil
}

// ReloadPlugin reloads a plugin by stopping and starting it
func (m *MeshPluginManager) ReloadPlugin(name string) error {
	m.mu.RLock()
	mp, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return ErrPluginNotFound
	}

	// Save the spec for reloading
	spec := mp.spec

	// Unload the plugin
	if err := m.UnloadPlugin(name); err != nil {
		return fmt.Errorf("failed to unload plugin: %w", err)
	}

	// Reload the plugin
	if err := m.LoadPlugin(spec); err != nil {
		return fmt.Errorf("failed to reload plugin: %w", err)
	}

	return nil
}

// HotSwapPlugin performs a hot swap of a plugin to a new version
func (m *MeshPluginManager) HotSwapPlugin(name string, newVersion string) error {
	// This would involve:
	// 1. Loading the new version
	// 2. Exporting state from old version
	// 3. Importing state to new version
	// 4. Switching mesh routing
	// 5. Stopping old version
	
	// For now, return not implemented
	return fmt.Errorf("hot swap not yet implemented for mesh plugins")
}

// Private helper methods

func (m *MeshPluginManager) waitForPluginRegistration(ctx context.Context, mp *ManagedMeshPlugin) error {
	// Wait for the plugin to register itself with the mesh network
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if plugin is registered in mesh
			if m.protocolRouter != nil {
				// Check if the service is registered
				// This is a simplified check - real implementation would query mesh registry
				return nil
			}
		}
	}
}

func (m *MeshPluginManager) connectToPlugin(ctx context.Context, mp *ManagedMeshPlugin) error {
	// Connect to the plugin via mesh network
	
	// In a real implementation, this would:
	// 1. Query mesh for plugin's endpoint
	// 2. Establish gRPC connection
	// 3. Verify plugin is responsive
	
	// For now, we'll simulate this
	socketPath := fmt.Sprintf("%s/%s.sock", m.socketDir, mp.spec.Name)
	
	conn, err := grpc.DialContext(ctx,
		fmt.Sprintf("unix://%s", socketPath),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to plugin socket: %w", err)
	}

	mp.grpcConn = conn
	mp.endpoint = mesh.ServiceEndpoint{
		Socket:  socketPath,
		IsLocal: true,
	}

	// Register the plugin's endpoint with protocol router
	if m.protocolRouter != nil {
		err = m.protocolRouter.RegisterService(mp.serviceName, mp.endpoint)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to register with protocol router: %w", err)
		}
	}

	return nil
}

// ExportPluginState exports state from a plugin
func (m *MeshPluginManager) ExportPluginState(name string) ([]byte, error) {
	m.mu.RLock()
	mp, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrPluginNotFound
	}

	return mp.plugin.ExportState()
}

// ImportPluginState imports state into a plugin
func (m *MeshPluginManager) ImportPluginState(name string, state []byte) error {
	m.mu.RLock()
	mp, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return ErrPluginNotFound
	}

	return mp.plugin.ImportState(state)
}