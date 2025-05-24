// Package loader implements plugin loading functionality for the Blackhole Framework.
package loader

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"plugin"
	"sync"
	"time"
	"context"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/validator"
)

// Common errors
var (
	ErrInvalidPlugin       = errors.New("invalid plugin")
	ErrPluginNotFound      = errors.New("plugin not found")
	ErrInvalidSource       = errors.New("invalid plugin source")
	ErrVerificationFailed  = errors.New("plugin verification failed")
	ErrDependencyMissing   = errors.New("plugin dependency missing")
	ErrIncompatibleVersion = errors.New("incompatible plugin version")
)

// pluginLoader implements the PluginLoader interface
type pluginLoader struct {
	validators []PluginValidator
	loaders    map[plugins.SourceType]SourceLoader
	cache      *PluginCache
	mu         sync.RWMutex
}

// PluginValidator validates a plugin before loading
type PluginValidator interface {
	Validate(spec plugins.PluginSpec, binaryPath string) error
}

// SourceLoader loads plugins from different sources
type SourceLoader interface {
	Load(source plugins.PluginSource) (string, error) // Returns local path to plugin binary
}

// PluginCache caches loaded plugins
type PluginCache struct {
	mu    sync.RWMutex
	cache map[string]CacheEntry
}

// CacheEntry represents a cached plugin
type CacheEntry struct {
	Plugin     plugins.Plugin
	BinaryPath string
	LoadTime   int64
}

// New creates a new plugin loader
func New() plugins.PluginLoader {
	return NewWithOptions(false)
}

// NewWithOptions creates a new plugin loader with options
func NewWithOptions(strictCompliance bool) plugins.PluginLoader {
	loader := &pluginLoader{
		validators: []PluginValidator{
			&hashValidator{},
			&dependencyValidator{},
			newComplianceValidator(strictCompliance),
		},
		loaders: make(map[plugins.SourceType]SourceLoader),
		cache:   &PluginCache{cache: make(map[string]CacheEntry)},
	}

	// Register default source loaders
	loader.loaders[plugins.SourceTypeLocal] = &localSourceLoader{}
	loader.loaders[plugins.SourceTypeRemote] = &remoteSourceLoader{
		httpClient: &http.Client{},
		cacheDir:   filepath.Join(os.TempDir(), "blackhole-plugins"),
	}

	return loader
}

// LoadPlugin loads a plugin according to its specification
func (l *pluginLoader) LoadPlugin(spec plugins.PluginSpec) (plugins.Plugin, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check cache first
	if cached, ok := l.cache.Get(spec.Name); ok {
		return cached.Plugin, nil
	}

	// Get the appropriate source loader
	sourceLoader, ok := l.loaders[spec.Source.Type]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidSource, spec.Source.Type)
	}

	// Load the plugin binary
	binaryPath, err := sourceLoader.Load(spec.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin source: %w", err)
	}

	// Validate the plugin
	for _, validator := range l.validators {
		if err := validator.Validate(spec, binaryPath); err != nil {
			return nil, fmt.Errorf("plugin validation failed: %w", err)
		}
	}

	// Create the plugin instance based on isolation level
	var p plugins.Plugin
	switch spec.Isolation {
	case plugins.IsolationNone:
		p, err = l.loadInProcessPlugin(spec, binaryPath)
	case plugins.IsolationProcess:
		p, err = l.loadProcessPlugin(spec, binaryPath)
	default:
		return nil, fmt.Errorf("unsupported isolation level: %s", spec.Isolation)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create plugin instance: %w", err)
	}

	// Cache the loaded plugin
	l.cache.Put(spec.Name, p, binaryPath)

	return p, nil
}

// loadInProcessPlugin loads a plugin in the same process (Go plugin)
func (l *pluginLoader) loadInProcessPlugin(spec plugins.PluginSpec, binaryPath string) (plugins.Plugin, error) {
	// Load the Go plugin
	p, err := plugin.Open(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for the plugin factory function
	symbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return nil, fmt.Errorf("plugin missing NewPlugin function: %w", err)
	}

	// Cast to expected function type
	newPlugin, ok := symbol.(func() plugins.Plugin)
	if !ok {
		return nil, errors.New("NewPlugin has incorrect signature")
	}

	// Create plugin instance
	return newPlugin(), nil
}

// loadProcessPlugin loads a plugin as a separate process
func (l *pluginLoader) loadProcessPlugin(spec plugins.PluginSpec, binaryPath string) (plugins.Plugin, error) {
	// Import the executor package for process plugin
	// Note: This creates a circular dependency that needs to be resolved
	// For now, we'll create a simple process plugin inline
	return &processPlugin{
		spec:       spec,
		binaryPath: binaryPath,
		info: plugins.PluginInfo{
			Name:        spec.Name,
			Version:     spec.Version,
			Description: "Process-isolated plugin",
			Status:      plugins.PluginStatusStopped,
		},
	}, nil
}

// processPlugin is a minimal implementation for process-isolated plugins
// The full implementation is in the executor package
type processPlugin struct {
	spec       plugins.PluginSpec
	binaryPath string
	info       plugins.PluginInfo
}

func (p *processPlugin) Info() plugins.PluginInfo {
	return p.info
}

func (p *processPlugin) Start(ctx context.Context) error {
	// Minimal implementation - full version in executor package
	p.info.Status = plugins.PluginStatusRunning
	return nil
}

func (p *processPlugin) Stop(ctx context.Context) error {
	p.info.Status = plugins.PluginStatusStopped
	return nil
}

func (p *processPlugin) Handle(ctx context.Context, request plugins.PluginRequest) (plugins.PluginResponse, error) {
	return plugins.PluginResponse{
		ID:      request.ID,
		Success: false,
		Error:   "process plugin execution requires executor package",
	}, errors.New("not implemented in loader")
}

func (p *processPlugin) HealthCheck() error {
	return nil
}

func (p *processPlugin) GetStatus() plugins.PluginStatus {
	return p.info.Status
}

func (p *processPlugin) PrepareShutdown() error {
	return nil
}

func (p *processPlugin) ExportState() ([]byte, error) {
	return nil, nil
}

func (p *processPlugin) ImportState(state []byte) error {
	return nil
}

// UnloadPlugin unloads a plugin
func (l *pluginLoader) UnloadPlugin(p plugins.Plugin) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Remove from cache
	l.cache.Remove(p.Info().Name)

	// Stop the plugin
	return p.Stop(nil)
}

// ValidatePlugin validates a plugin specification
func (l *pluginLoader) ValidatePlugin(spec plugins.PluginSpec) error {
	// Basic validation
	if spec.Name == "" {
		return errors.New("plugin name is required")
	}

	if spec.Version == "" {
		return errors.New("plugin version is required")
	}

	if spec.Source.Path == "" {
		return errors.New("plugin source path is required")
	}

	// Validate source type
	if _, ok := l.loaders[spec.Source.Type]; !ok {
		return fmt.Errorf("%w: %s", ErrInvalidSource, spec.Source.Type)
	}

	return nil
}

// hashValidator validates plugin hash
type hashValidator struct{}

func (v *hashValidator) Validate(spec plugins.PluginSpec, binaryPath string) error {
	if spec.Source.Hash == "" {
		// Hash validation is optional
		return nil
	}

	// Calculate file hash
	file, err := os.Open(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin binary: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != spec.Source.Hash {
		return fmt.Errorf("%w: expected %s, got %s", ErrVerificationFailed, spec.Source.Hash, actualHash)
	}

	return nil
}

// dependencyValidator validates plugin dependencies
type dependencyValidator struct{}

func (v *dependencyValidator) Validate(spec plugins.PluginSpec, binaryPath string) error {
	// TODO: Implement dependency checking
	// For now, just check that required dependencies are specified
	for _, dep := range spec.Dependencies {
		if !dep.Optional && dep.Name == "" {
			return fmt.Errorf("%w: %s", ErrDependencyMissing, dep.Name)
		}
	}
	return nil
}

// localSourceLoader loads plugins from local filesystem
type localSourceLoader struct{}

func (l *localSourceLoader) Load(source plugins.PluginSource) (string, error) {
	// Check if file exists
	if _, err := os.Stat(source.Path); err != nil {
		if os.IsNotExist(err) {
			return "", ErrPluginNotFound
		}
		return "", fmt.Errorf("failed to stat plugin: %w", err)
	}

	// Return the path as-is for local files
	return source.Path, nil
}

// remoteSourceLoader loads plugins from remote URLs
type remoteSourceLoader struct {
	httpClient *http.Client
	cacheDir   string
}

func (l *remoteSourceLoader) Load(source plugins.PluginSource) (string, error) {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(l.cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Generate cache filename based on URL
	hasher := sha256.New()
	hasher.Write([]byte(source.Path))
	cacheFile := filepath.Join(l.cacheDir, hex.EncodeToString(hasher.Sum(nil)))

	// Check if already cached
	if _, err := os.Stat(cacheFile); err == nil {
		return cacheFile, nil
	}

	// Download the plugin
	resp, err := l.httpClient.Get(source.Path)
	if err != nil {
		return "", fmt.Errorf("failed to download plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download plugin: HTTP %d", resp.StatusCode)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp(l.cacheDir, "plugin-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Copy content
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to save plugin: %w", err)
	}
	tmpFile.Close()

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to make plugin executable: %w", err)
	}

	// Move to final location
	if err := os.Rename(tmpPath, cacheFile); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to cache plugin: %w", err)
	}

	return cacheFile, nil
}

// PluginCache methods

func (c *PluginCache) Get(name string) (CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.cache[name]
	return entry, ok
}

func (c *PluginCache) Put(name string, p plugins.Plugin, binaryPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[name] = CacheEntry{
		Plugin:     p,
		BinaryPath: binaryPath,
		LoadTime:   time.Now().Unix(),
	}
}

func (c *PluginCache) Remove(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, name)
}

