package registry_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/registry"
)

// MockMarketplaceClient implements the MarketplaceClient interface for testing
type MockMarketplaceClient struct {
	FetchFunc   func(id string) (plugins.PluginSpec, error)
	PublishFunc func(spec plugins.PluginSpec) error
}

func (m *MockMarketplaceClient) FetchPlugin(id string) (plugins.PluginSpec, error) {
	if m.FetchFunc != nil {
		return m.FetchFunc(id)
	}
	return plugins.PluginSpec{}, nil
}

func (m *MockMarketplaceClient) PublishPlugin(spec plugins.PluginSpec) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(spec)
	}
	return nil
}

func TestPluginRegistry_RegisterAndUnregister(t *testing.T) {
	r := registry.New(nil)

	// Create test plugin info
	info := plugins.PluginInfo{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		Author:      "Test Author",
		License:     "MIT",
		Status:      plugins.PluginStatusStopped,
		LoadTime:    time.Now(),
		Capabilities: []plugins.PluginCapability{
			plugins.CapabilityStorage,
			plugins.CapabilityAnalytics,
		},
	}

	// Test registration
	err := r.RegisterPlugin(info)
	require.NoError(t, err)

	// Test duplicate registration
	err = r.RegisterPlugin(info)
	assert.Error(t, err)

	// Test unregistration
	err = r.UnregisterPlugin("test-plugin")
	require.NoError(t, err)

	// Test unregistering non-existent plugin
	err = r.UnregisterPlugin("non-existent")
	assert.Error(t, err)
}

func TestPluginRegistry_SearchPlugins(t *testing.T) {
	r := registry.New(nil)

	// Register multiple test plugins
	plugins := []plugins.PluginInfo{
		{
			Name:         "storage-plugin",
			Version:      "1.0.0",
			Author:       "Storage Team",
			Capabilities: []plugins.PluginCapability{plugins.CapabilityStorage},
		},
		{
			Name:         "analytics-plugin",
			Version:      "2.0.0",
			Author:       "Analytics Team",
			Capabilities: []plugins.PluginCapability{plugins.CapabilityAnalytics},
		},
		{
			Name:         "multi-plugin",
			Version:      "1.5.0",
			Author:       "Storage Team",
			Capabilities: []plugins.PluginCapability{plugins.CapabilityStorage, plugins.CapabilityAnalytics},
		},
	}

	for _, p := range plugins {
		err := r.RegisterPlugin(p)
		require.NoError(t, err)
	}

	// Test search by capability
	results, err := r.SearchPlugins(plugins.SearchCriteria{
		Capabilities: []plugins.PluginCapability{plugins.CapabilityStorage},
	})
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Test search by author
	results, err = r.SearchPlugins(plugins.SearchCriteria{
		Author: "Storage",
	})
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Test search by name
	results, err = r.SearchPlugins(plugins.SearchCriteria{
		Name: "multi",
	})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "multi-plugin", results[0].Name)
}

func TestPluginRegistry_DiscoverPlugins(t *testing.T) {
	r := registry.New(nil)

	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create test plugin.json file
	pluginSpec := `{
		"name": "test-plugin",
		"version": "1.0.0",
		"source": {
			"type": "local",
			"path": "./test-plugin.so"
		},
		"isolation": "process",
		"resources": {
			"cpu": 50,
			"memory": 128
		}
	}`

	pluginFile := filepath.Join(tmpDir, "plugin.json")
	err := os.WriteFile(pluginFile, []byte(pluginSpec), 0644)
	require.NoError(t, err)

	// Test discovery
	specs, err := r.DiscoverPlugins(tmpDir)
	require.NoError(t, err)
	assert.Len(t, specs, 1)
	assert.Equal(t, "test-plugin", specs[0].Name)
	assert.Equal(t, "1.0.0", specs[0].Version)
	assert.Equal(t, plugins.IsolationProcess, specs[0].Isolation)
}

func TestPluginRegistry_Marketplace(t *testing.T) {
	mockMarketplace := &MockMarketplaceClient{
		FetchFunc: func(id string) (plugins.PluginSpec, error) {
			return plugins.PluginSpec{
				Name:    "marketplace-plugin",
				Version: "1.0.0",
				Source: plugins.PluginSource{
					Type: plugins.SourceTypeMarketplace,
					Path: id,
				},
				Isolation: plugins.IsolationProcess,
			}, nil
		},
		PublishFunc: func(spec plugins.PluginSpec) error {
			return nil
		},
	}

	r := registry.New(mockMarketplace)

	// Test fetch from marketplace
	spec, err := r.FetchFromMarketplace("test-plugin-id")
	require.NoError(t, err)
	assert.Equal(t, "marketplace-plugin", spec.Name)

	// Test publish to marketplace
	publishSpec := plugins.PluginSpec{
		Name:    "my-plugin",
		Version: "1.0.0",
		Source: plugins.PluginSource{
			Type: plugins.SourceTypeLocal,
			Path: "./my-plugin.so",
		},
		Isolation: plugins.IsolationProcess,
	}
	err = r.PublishToMarketplace(publishSpec)
	require.NoError(t, err)
}