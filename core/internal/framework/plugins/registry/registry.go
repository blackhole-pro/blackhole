// Package registry implements the plugin registry for discovering and managing plugins.
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
)

// Common errors
var (
	ErrPluginNotFound      = errors.New("plugin not found")
	ErrPluginAlreadyExists = errors.New("plugin already exists")
	ErrInvalidPluginSpec   = errors.New("invalid plugin specification")
	ErrInvalidSearchCriteria = errors.New("invalid search criteria")
)

// pluginRegistry implements the PluginRegistry interface
type pluginRegistry struct {
	mu          sync.RWMutex
	plugins     map[string]*plugins.PluginInfo
	searchIndex map[string][]string // capability -> plugin names
	marketplace MarketplaceClient
}

// MarketplaceClient interface for marketplace integration
type MarketplaceClient interface {
	FetchPlugin(id string) (plugins.PluginSpec, error)
	PublishPlugin(spec plugins.PluginSpec) error
}

// New creates a new plugin registry
func New(marketplace MarketplaceClient) plugins.PluginRegistry {
	return &pluginRegistry{
		plugins:     make(map[string]*plugins.PluginInfo),
		searchIndex: make(map[string][]string),
		marketplace: marketplace,
	}
}

// DiscoverPlugins scans a directory for plugin specifications
func (r *pluginRegistry) DiscoverPlugins(path string) ([]plugins.PluginSpec, error) {
	var specs []plugins.PluginSpec

	// Walk the directory tree
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Look for plugin specification files
		if !d.IsDir() && strings.HasSuffix(path, "plugin.json") {
			spec, err := r.loadPluginSpec(path)
			if err != nil {
				// Log error but continue scanning
				fmt.Printf("Warning: failed to load plugin spec from %s: %v\n", path, err)
				return nil
			}
			specs = append(specs, spec)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to discover plugins: %w", err)
	}

	return specs, nil
}

// loadPluginSpec loads a plugin specification from a JSON file
func (r *pluginRegistry) loadPluginSpec(path string) (plugins.PluginSpec, error) {
	var spec plugins.PluginSpec

	data, err := os.ReadFile(path)
	if err != nil {
		return spec, fmt.Errorf("failed to read plugin spec: %w", err)
	}

	if err := json.Unmarshal(data, &spec); err != nil {
		return spec, fmt.Errorf("failed to parse plugin spec: %w", err)
	}

	// Validate the specification
	if err := r.validatePluginSpec(spec); err != nil {
		return spec, fmt.Errorf("invalid plugin spec: %w", err)
	}

	// If source path is relative, make it absolute based on spec file location
	if spec.Source.Type == plugins.SourceTypeLocal && !filepath.IsAbs(spec.Source.Path) {
		spec.Source.Path = filepath.Join(filepath.Dir(path), spec.Source.Path)
	}

	return spec, nil
}

// validatePluginSpec validates a plugin specification
func (r *pluginRegistry) validatePluginSpec(spec plugins.PluginSpec) error {
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
	switch spec.Source.Type {
	case plugins.SourceTypeLocal, plugins.SourceTypeRemote, plugins.SourceTypeMarketplace:
		// Valid source types
	default:
		return fmt.Errorf("invalid source type: %s", spec.Source.Type)
	}

	// Validate isolation level
	switch spec.Isolation {
	case plugins.IsolationNone, plugins.IsolationThread, plugins.IsolationProcess,
		plugins.IsolationContainer, plugins.IsolationVM:
		// Valid isolation levels
	default:
		return fmt.Errorf("invalid isolation level: %s", spec.Isolation)
	}

	return nil
}

// SearchPlugins searches for plugins matching the given criteria
func (r *pluginRegistry) SearchPlugins(criteria plugins.SearchCriteria) ([]plugins.PluginInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []plugins.PluginInfo

	// If searching by capability, use the index
	if len(criteria.Capabilities) > 0 {
		pluginNames := make(map[string]bool)
		for _, capability := range criteria.Capabilities {
			for _, name := range r.searchIndex[string(capability)] {
				pluginNames[name] = true
			}
		}

		for name := range pluginNames {
			if info, exists := r.plugins[name]; exists {
				if r.matchesCriteria(info, criteria) {
					results = append(results, *info)
				}
			}
		}
	} else {
		// Search all plugins
		for _, info := range r.plugins {
			if r.matchesCriteria(info, criteria) {
				results = append(results, *info)
			}
		}
	}

	return results, nil
}

// matchesCriteria checks if a plugin matches the search criteria
func (r *pluginRegistry) matchesCriteria(info *plugins.PluginInfo, criteria plugins.SearchCriteria) bool {
	// Check name
	if criteria.Name != "" && !strings.Contains(strings.ToLower(info.Name), strings.ToLower(criteria.Name)) {
		return false
	}

	// Check author
	if criteria.Author != "" && !strings.Contains(strings.ToLower(info.Author), strings.ToLower(criteria.Author)) {
		return false
	}

	// Check license
	if criteria.License != "" && info.License != criteria.License {
		return false
	}

	// Check version range
	if criteria.MinVersion != "" {
		// TODO: Implement proper version comparison
		if info.Version < criteria.MinVersion {
			return false
		}
	}

	if criteria.MaxVersion != "" {
		// TODO: Implement proper version comparison
		if info.Version > criteria.MaxVersion {
			return false
		}
	}

	// Check capabilities
	if len(criteria.Capabilities) > 0 {
		hasCapability := false
		for _, requiredCap := range criteria.Capabilities {
			for _, pluginCap := range info.Capabilities {
				if pluginCap == requiredCap {
					hasCapability = true
					break
				}
			}
			if !hasCapability {
				return false
			}
		}
	}

	return true
}

// RegisterPlugin registers a plugin in the registry
func (r *pluginRegistry) RegisterPlugin(info plugins.PluginInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[info.Name]; exists {
		return ErrPluginAlreadyExists
	}

	// Add to main registry
	r.plugins[info.Name] = &info

	// Update search index
	for _, capability := range info.Capabilities {
		r.searchIndex[string(capability)] = append(r.searchIndex[string(capability)], info.Name)
	}

	return nil
}

// UnregisterPlugin removes a plugin from the registry
func (r *pluginRegistry) UnregisterPlugin(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.plugins[name]
	if !exists {
		return ErrPluginNotFound
	}

	// Remove from main registry
	delete(r.plugins, name)

	// Update search index
	for _, capability := range info.Capabilities {
		r.removeFromIndex(string(capability), name)
	}

	return nil
}

// removeFromIndex removes a plugin name from a capability index
func (r *pluginRegistry) removeFromIndex(capability, name string) {
	names := r.searchIndex[capability]
	for i, n := range names {
		if n == name {
			// Remove by swapping with last element and truncating
			names[i] = names[len(names)-1]
			r.searchIndex[capability] = names[:len(names)-1]
			break
		}
	}
}

// FetchFromMarketplace fetches a plugin specification from the marketplace
func (r *pluginRegistry) FetchFromMarketplace(id string) (plugins.PluginSpec, error) {
	if r.marketplace == nil {
		return plugins.PluginSpec{}, errors.New("marketplace client not configured")
	}

	return r.marketplace.FetchPlugin(id)
}

// PublishToMarketplace publishes a plugin to the marketplace
func (r *pluginRegistry) PublishToMarketplace(spec plugins.PluginSpec) error {
	if r.marketplace == nil {
		return errors.New("marketplace client not configured")
	}

	// Validate the specification before publishing
	if err := r.validatePluginSpec(spec); err != nil {
		return fmt.Errorf("cannot publish invalid plugin spec: %w", err)
	}

	return r.marketplace.PublishPlugin(spec)
}

// GetPlugin returns information about a specific plugin
func (r *pluginRegistry) GetPlugin(name string) (plugins.PluginInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.plugins[name]
	if !exists {
		return plugins.PluginInfo{}, ErrPluginNotFound
	}

	return *info, nil
}

// ListPlugins returns a list of all registered plugins
func (r *pluginRegistry) ListPlugins() []plugins.PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []plugins.PluginInfo
	for _, info := range r.plugins {
		list = append(list, *info)
	}

	return list
}