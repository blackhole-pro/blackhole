// Package factory provides plugin manager factory functions for easy instantiation.
package factory

import (
	"time"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/executor"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/lifecycle"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/loader"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/registry"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/state"
)

// Config holds configuration for creating a plugin manager
type Config struct {
	// Registry configuration
	MarketplaceURL string
	
	// Loader configuration
	CachePath      string
	TempPath       string
	
	// Executor configuration
	MaxConcurrentPlugins int
	ResourceUpdateInterval time.Duration
	
	// State configuration
	StatePath      string
	EnableAutoSave bool
	AutoSaveInterval time.Duration
}

// DefaultConfig returns default plugin manager configuration
func DefaultConfig() *Config {
	return &Config{
		MarketplaceURL: "https://marketplace.blackhole.io",
		CachePath:      "/tmp/blackhole/plugin-cache",
		TempPath:       "/tmp/blackhole/plugin-temp",
		MaxConcurrentPlugins: 10,
		ResourceUpdateInterval: 5 * time.Second,
		StatePath:      "/tmp/blackhole/plugin-state",
		EnableAutoSave: true,
		AutoSaveInterval: 5 * time.Minute,
	}
}

// NewPluginManager creates a new plugin manager with all required components
func NewPluginManager(config *Config) plugins.PluginManager {
	if config == nil {
		config = DefaultConfig()
	}
	
	// Create registry
	marketplaceClient := registry.NewMockMarketplaceClient() // TODO: Replace with real client
	pluginRegistry := registry.New(marketplaceClient)
	
	// Create loader
	pluginLoader := loader.New()
	
	// Create executor
	pluginExecutor := executor.NewExecutor(
		config.MaxConcurrentPlugins,
		config.ResourceUpdateInterval,
	)
	
	// Create state manager
	var stateStorage state.StateStorage
	fileStorage, err := state.NewFileStateStorage(config.StatePath)
	if err != nil {
		// Use memory storage as fallback
		stateStorage = state.NewMemoryStateStorage()
	} else {
		stateStorage = fileStorage
	}
	stateManager := state.NewStateManagerWrapper(stateStorage, state.NewJSONStateSerializer())
	
	// Create lifecycle manager
	lifecycleManager := lifecycle.NewLifecycleManager()
	
	// Create plugin manager
	return plugins.NewManager(
		pluginRegistry,
		pluginLoader,
		pluginExecutor,
		stateManager,
		lifecycleManager,
	)
}

// NewMockPluginManager creates a plugin manager with mock components for testing
func NewMockPluginManager() plugins.PluginManager {
	// Create mock implementations
	mockRegistry := &mockRegistry{}
	mockLoader := &mockLoader{}
	mockExecutor := &mockExecutor{}
	mockStateManager := &mockStateManager{}
	mockLifecycle := &mockLifecycle{}
	
	return plugins.NewManager(
		mockRegistry,
		mockLoader,
		mockExecutor,
		mockStateManager,
		mockLifecycle,
	)
}

// Mock implementations for testing

type mockRegistry struct{}

func (m *mockRegistry) DiscoverPlugins(path string) ([]plugins.PluginSpec, error) { return nil, nil }
func (m *mockRegistry) SearchPlugins(criteria plugins.SearchCriteria) ([]plugins.PluginInfo, error) { return nil, nil }
func (m *mockRegistry) RegisterPlugin(info plugins.PluginInfo) error { return nil }
func (m *mockRegistry) UnregisterPlugin(name string) error { return nil }
func (m *mockRegistry) FetchFromMarketplace(id string) (plugins.PluginSpec, error) {
	return plugins.PluginSpec{}, nil
}
func (m *mockRegistry) PublishToMarketplace(spec plugins.PluginSpec) error { return nil }

type mockLoader struct{}

func (m *mockLoader) LoadPlugin(spec plugins.PluginSpec) (plugins.Plugin, error) { return nil, nil }
func (m *mockLoader) UnloadPlugin(plugin plugins.Plugin) error { return nil }
func (m *mockLoader) ValidatePlugin(spec plugins.PluginSpec) error { return nil }
func (m *mockLoader) GetLoadedPlugins() []string { return nil }

type mockExecutor struct{}

func (m *mockExecutor) StartPlugin(plugin plugins.Plugin) error { return nil }
func (m *mockExecutor) StopPlugin(plugin plugins.Plugin) error { return nil }
func (m *mockExecutor) ExecutePlugin(plugin plugins.Plugin, request plugins.PluginRequest) (plugins.PluginResponse, error) {
	return plugins.PluginResponse{}, nil
}
func (m *mockExecutor) GetResourceUsage(plugin plugins.Plugin) plugins.PluginResourceUsage {
	return plugins.PluginResourceUsage{}
}
func (m *mockExecutor) EnforceResourceLimits(plugin plugins.Plugin) error { return nil }
func (m *mockExecutor) IsPluginRunning(plugin plugins.Plugin) bool { return false }
func (m *mockExecutor) GetExecutionEnvironment(plugin plugins.Plugin) (plugins.ExecutionEnvironment, error) {
	return nil, nil
}
func (m *mockExecutor) CreateIsolationBoundary(level plugins.IsolationLevel) (plugins.IsolationBoundary, error) {
	return nil, nil
}

type mockStateManager struct{}

func (m *mockStateManager) SaveState(plugin plugins.Plugin) error { return nil }
func (m *mockStateManager) LoadState(plugin plugins.Plugin) error { return nil }
func (m *mockStateManager) MigrateState(plugin plugins.Plugin, fromVersion, toVersion string) error { return nil }
func (m *mockStateManager) ExportState(plugin plugins.Plugin) ([]byte, error) { return nil, nil }
func (m *mockStateManager) ImportState(plugin plugins.Plugin, state []byte) error { return nil }

type mockLifecycle struct{}

func (m *mockLifecycle) OnPluginLoad(plugin plugins.Plugin) error { return nil }
func (m *mockLifecycle) OnPluginStart(plugin plugins.Plugin) error { return nil }
func (m *mockLifecycle) OnPluginStop(plugin plugins.Plugin) error { return nil }
func (m *mockLifecycle) OnPluginUnload(plugin plugins.Plugin) error { return nil }
func (m *mockLifecycle) OnPluginError(plugin plugins.Plugin, err error) error { return nil }