package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/executor"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/loader"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/registry"
)

func TestPluginFramework_BasicFlow(t *testing.T) {
	// Create registry
	reg := registry.New(nil)

	// Create loader
	ldr := loader.New()

	// Create executor
	exec := executor.New(executor.Config{
		DefaultTimeout:  30 * time.Second,
		ResourceMonitor: executor.NewBasicResourceMonitor(),
	})

	// Create manager
	manager := plugins.NewManager(reg, ldr, exec, nil, nil)

	// Test plugin spec
	spec := plugins.PluginSpec{
		Name:    "test-plugin",
		Version: "1.0.0",
		Source: plugins.PluginSource{
			Type: plugins.SourceTypeLocal,
			Path: filepath.Join("testdata", "test-plugin.so"),
		},
		Isolation: plugins.IsolationNone,
		Resources: plugins.PluginResources{
			CPU:    50,
			Memory: 128,
		},
	}

	// Test loading plugin
	t.Run("LoadPlugin", func(t *testing.T) {
		// For this test, we would need an actual plugin binary
		// This is a placeholder showing how the framework would be used
		t.Skip("Requires actual plugin binary")

		err := manager.LoadPlugin(spec)
		require.NoError(t, err)

		// Verify plugin is loaded
		info, err := manager.GetPlugin("test-plugin")
		require.NoError(t, err)
		assert.Equal(t, "test-plugin", info.Name)
		assert.Equal(t, "1.0.0", info.Version)
		assert.Equal(t, plugins.PluginStatusRunning, info.Status)
	})

	// Test plugin execution
	t.Run("ExecutePlugin", func(t *testing.T) {
		t.Skip("Requires actual plugin binary")

		request := plugins.PluginRequest{
			ID:     "test-req-1",
			Method: "greet",
			Params: map[string]interface{}{
				"name": "Blackhole",
			},
			Context: plugins.RequestContext{
				Timestamp: time.Now(),
			},
		}

		response, err := manager.ExecutePlugin("test-plugin", request)
		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, "test-req-1", response.ID)
		assert.NotEmpty(t, response.Result)
	})

	// Test plugin hot-swap
	t.Run("HotSwapPlugin", func(t *testing.T) {
		t.Skip("Requires actual plugin binary")

		// Export current state
		state, err := manager.ExportPluginState("test-plugin")
		require.NoError(t, err)
		assert.NotEmpty(t, state)

		// Perform hot swap
		err = manager.HotSwapPlugin("test-plugin", "2.0.0")
		require.NoError(t, err)

		// Verify new version
		info, err := manager.GetPlugin("test-plugin")
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", info.Version)
		assert.Equal(t, plugins.PluginStatusRunning, info.Status)
	})

	// Test unloading plugin
	t.Run("UnloadPlugin", func(t *testing.T) {
		t.Skip("Requires actual plugin binary")

		err := manager.UnloadPlugin("test-plugin")
		require.NoError(t, err)

		// Verify plugin is unloaded
		_, err = manager.GetPlugin("test-plugin")
		assert.Error(t, err)
	})
}

func TestPluginRegistry_Discovery(t *testing.T) {
	reg := registry.New(nil)

	// Test discovering plugins from directory
	pluginDir := filepath.Join("..", "..", "examples", "plugins")
	specs, err := reg.DiscoverPlugins(pluginDir)
	require.NoError(t, err)

	// Should find the hello plugin
	found := false
	for _, spec := range specs {
		if spec.Name == "hello-plugin" {
			found = true
			assert.Equal(t, "1.0.0", spec.Version)
			assert.Equal(t, plugins.IsolationProcess, spec.Isolation)
			assert.Equal(t, 10, spec.Resources.CPU)
			assert.Equal(t, 64, spec.Resources.Memory)
		}
	}
	assert.True(t, found, "hello-plugin should be discovered")
}

func TestPluginExecution_ProcessIsolation(t *testing.T) {
	// This test would require building the hello plugin as an executable
	t.Skip("Requires building plugin binary")

	// Build the hello plugin
	// cmd := exec.Command("go", "build", "-o", "hello-plugin", "../examples/plugins/hello/main.go")
	// err := cmd.Run()
	// require.NoError(t, err)

	// Create process plugin
	spec := plugins.PluginSpec{
		Name:    "hello-plugin",
		Version: "1.0.0",
		Source: plugins.PluginSource{
			Type: plugins.SourceTypeLocal,
			Path: "./hello-plugin",
		},
		Isolation: plugins.IsolationProcess,
	}

	plugin := executor.NewProcessPlugin(spec, "./hello-plugin")

	// Start plugin
	ctx := context.Background()
	err := plugin.Start(ctx)
	require.NoError(t, err)
	defer plugin.Stop(ctx)

	// Test plugin execution
	request := plugins.PluginRequest{
		ID:     "test-1",
		Method: "greet",
		Params: map[string]interface{}{
			"name": "World",
		},
	}

	response, err := plugin.Handle(ctx, request)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Hello, World!", response.Result["greeting"])
	assert.Equal(t, float64(1), response.Result["count"])

	// Test state export
	state, err := plugin.ExportState()
	require.NoError(t, err)
	assert.NotEmpty(t, state)
}