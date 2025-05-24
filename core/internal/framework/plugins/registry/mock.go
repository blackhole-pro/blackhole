package registry

import (
	"fmt"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
)

// MockMarketplaceClient is a mock implementation of MarketplaceClient for testing
type MockMarketplaceClient struct {
	Plugins map[string]plugins.PluginSpec
}

// NewMockMarketplaceClient creates a new mock marketplace client
func NewMockMarketplaceClient() *MockMarketplaceClient {
	return &MockMarketplaceClient{
		Plugins: make(map[string]plugins.PluginSpec),
	}
}

// FetchPlugin fetches a plugin from the mock marketplace
func (m *MockMarketplaceClient) FetchPlugin(id string) (plugins.PluginSpec, error) {
	spec, exists := m.Plugins[id]
	if !exists {
		return plugins.PluginSpec{}, fmt.Errorf("plugin %s not found in marketplace", id)
	}
	return spec, nil
}

// PublishPlugin publishes a plugin to the mock marketplace
func (m *MockMarketplaceClient) PublishPlugin(spec plugins.PluginSpec) error {
	m.Plugins[spec.Name] = spec
	return nil
}