// Package loader provides mesh-aware plugin loading
package loader

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/executor"
)

// MeshPluginLoader loads plugins that communicate via mesh network
type MeshPluginLoader struct {
	localPath    string
	cacheDir     string
	tempDir      string
	// meshClient   mesh.Client // TODO: implement mesh client
	socketDir    string
	logger       *zap.Logger
}

// MeshLoaderConfig configures the mesh plugin loader
type MeshLoaderConfig struct {
	LocalPath  string
	CacheDir   string
	TempDir    string
	SocketDir  string
	// MeshClient mesh.Client // TODO: implement mesh client
	Logger     *zap.Logger
}

// NewMeshPluginLoader creates a new mesh-aware plugin loader
func NewMeshPluginLoader(config MeshLoaderConfig) *MeshPluginLoader {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	if config.SocketDir == "" {
		config.SocketDir = "/tmp/blackhole/plugins"
	}
	if config.LocalPath == "" {
		config.LocalPath = "/usr/local/lib/blackhole/plugins"
	}
	if config.CacheDir == "" {
		config.CacheDir = "/var/cache/blackhole/plugins"
	}
	if config.TempDir == "" {
		config.TempDir = "/tmp/blackhole/plugin-staging"
	}

	return &MeshPluginLoader{
		localPath:  config.LocalPath,
		cacheDir:   config.CacheDir,
		tempDir:    config.TempDir,
		socketDir:  config.SocketDir,
		// meshClient: config.MeshClient, // TODO: add when mesh client available
		logger:     config.Logger,
	}
}

// LoadPlugin loads a plugin that will communicate via mesh
func (l *MeshPluginLoader) LoadPlugin(spec plugins.PluginSpec) (plugins.Plugin, error) {
	l.logger.Info("Loading mesh plugin",
		zap.String("name", spec.Name),
		zap.String("version", spec.Version),
		zap.String("source", string(spec.Source.Type)))

	// Determine binary path based on source
	binaryPath, err := l.resolveBinaryPath(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve binary path: %w", err)
	}

	// Verify the binary exists and is executable
	if err := l.verifyBinary(binaryPath); err != nil {
		return nil, fmt.Errorf("binary verification failed: %w", err)
	}

	// Create socket path for this plugin
	socketPath := filepath.Join(l.socketDir, fmt.Sprintf("%s.sock", spec.Name))

	// Create mesh isolation config
	isolationConfig := executor.MeshIsolationConfig{
		// MeshClient:     l.meshClient, // TODO: add when mesh client available
		SocketDir:      l.socketDir,
		Logger:         l.logger,
		// DefaultTimeout: spec.Timeout, // TODO: add timeout field to spec
	}

	// Create the mesh plugin
	plugin := executor.NewMeshPlugin(spec, binaryPath, isolationConfig)

	l.logger.Info("Mesh plugin loaded",
		zap.String("name", spec.Name),
		zap.String("binary", binaryPath),
		zap.String("socket", socketPath))

	return plugin, nil
}

// ValidatePlugin validates a plugin specification
func (l *MeshPluginLoader) ValidatePlugin(spec plugins.PluginSpec) error {
	// Basic validation
	if spec.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if spec.Version == "" {
		return fmt.Errorf("plugin version is required")
	}

	// Validate source
	switch spec.Source.Type {
	case plugins.SourceTypeLocal:
		if spec.Source.Path == "" {
			return fmt.Errorf("local source requires path")
		}
	case plugins.SourceTypeRemote:
		if spec.Source.Path == "" { // Remote sources use Path for URL
			return fmt.Errorf("remote source requires path (URL)")
		}
	case plugins.SourceTypeMarketplace:
		if spec.Source.Path == "" { // Marketplace sources use Path for ID
			return fmt.Errorf("marketplace source requires path (ID)")
		}
	default:
		return fmt.Errorf("invalid source type: %s", spec.Source.Type)
	}

	// Validate resources
	if spec.Resources.CPU < 0 {
		return fmt.Errorf("CPU shares cannot be negative")
	}
	if spec.Resources.Memory < 0 {
		return fmt.Errorf("memory limit cannot be negative")
	}

	// Validate isolation level for mesh plugins
	switch spec.Isolation {
	case plugins.IsolationProcess:
		// This is the recommended level for mesh plugins
	case plugins.IsolationContainer, plugins.IsolationVM:
		// These are also supported
	case plugins.IsolationNone, plugins.IsolationThread:
		return fmt.Errorf("mesh plugins require process, container, or VM isolation")
	default:
		return fmt.Errorf("invalid isolation level: %s", spec.Isolation)
	}

	return nil
}

// UnloadPlugin is called when a plugin is being unloaded
func (l *MeshPluginLoader) UnloadPlugin(plugin plugins.Plugin) error {
	// Clean up any cached resources
	// The actual process stopping is handled by the plugin itself
	
	info := plugin.Info()
	l.logger.Info("Unloading mesh plugin",
		zap.String("name", info.Name),
		zap.String("version", info.Version))

	// Remove socket file if it exists
	socketPath := filepath.Join(l.socketDir, fmt.Sprintf("%s.sock", info.Name))
	os.Remove(socketPath)

	return nil
}

// GetPluginPath returns the path where a plugin would be installed
func (l *MeshPluginLoader) GetPluginPath(spec plugins.PluginSpec) string {
	return filepath.Join(l.localPath, spec.Name, spec.Version, "plugin")
}

// Private helper methods

func (l *MeshPluginLoader) resolveBinaryPath(spec plugins.PluginSpec) (string, error) {
	switch spec.Source.Type {
	case plugins.SourceTypeLocal:
		// Use the provided path directly
		return spec.Source.Path, nil

	case plugins.SourceTypeRemote:
		// Download and cache the plugin
		cachedPath := filepath.Join(l.cacheDir, spec.Name, spec.Version, "plugin")
		if _, err := os.Stat(cachedPath); err == nil {
			// Already cached
			return cachedPath, nil
		}

		// Download the plugin
		if err := l.downloadPlugin(spec, cachedPath); err != nil {
			return "", fmt.Errorf("failed to download plugin: %w", err)
		}
		return cachedPath, nil

	case plugins.SourceTypeMarketplace:
		// Look in the standard plugin directory
		standardPath := l.GetPluginPath(spec)
		if _, err := os.Stat(standardPath); err == nil {
			return standardPath, nil
		}

		// Try to download from marketplace
		if err := l.downloadFromMarketplace(spec, standardPath); err != nil {
			return "", fmt.Errorf("failed to download from marketplace: %w", err)
		}
		return standardPath, nil

	default:
		return "", fmt.Errorf("unsupported source type: %s", spec.Source.Type)
	}
}

func (l *MeshPluginLoader) verifyBinary(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("binary not found: %w", err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", path)
	}

	// Check if it's executable
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("binary is not executable: %s", path)
	}

	// Verify it's a valid binary
	cmd := exec.Command(path, "--version")
	if err := cmd.Run(); err != nil {
		// Some plugins might not support --version, so just check if it's executable
		cmd = exec.Command(path, "--help")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("binary verification failed: %w", err)
		}
	}

	return nil
}

func (l *MeshPluginLoader) downloadPlugin(spec plugins.PluginSpec, targetPath string) error {
	// This would implement actual plugin downloading
	// For now, return an error
	return fmt.Errorf("remote plugin download not yet implemented")
}

func (l *MeshPluginLoader) downloadFromMarketplace(spec plugins.PluginSpec, targetPath string) error {
	// This would implement marketplace integration
	// For now, return an error
	return fmt.Errorf("marketplace download not yet implemented")
}