// Package plugins provides the plugin management framework for Blackhole.
package plugins

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Common errors
var (
	ErrPluginNotFound    = errors.New("plugin not found")
	ErrPluginNotLoaded   = errors.New("plugin not loaded")
	ErrPluginAlreadyLoaded = errors.New("plugin already loaded")
	ErrInvalidState      = errors.New("invalid plugin state")
)

// managedPlugin wraps a plugin with management metadata
type managedPlugin struct {
	plugin      Plugin
	spec        PluginSpec
	startTime   time.Time
	restartCount int
	lastError   error
	mu          sync.RWMutex
}

// pluginManager implements the PluginManager interface
type pluginManager struct {
	registry  PluginRegistry
	loader    PluginLoader
	executor  PluginExecutor
	state     StateManager
	lifecycle PluginLifecycle
	
	plugins   map[string]*managedPlugin
	mu        sync.RWMutex
}

// NewManager creates a new plugin manager
func NewManager(
	registry PluginRegistry,
	loader PluginLoader,
	executor PluginExecutor,
	state StateManager,
	lifecycle PluginLifecycle,
) PluginManager {
	return &pluginManager{
		registry:  registry,
		loader:    loader,
		executor:  executor,
		state:     state,
		lifecycle: lifecycle,
		plugins:   make(map[string]*managedPlugin),
	}
}

// LoadPlugin loads a plugin according to its specification
func (m *pluginManager) LoadPlugin(spec PluginSpec) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already loaded
	if _, exists := m.plugins[spec.Name]; exists {
		return ErrPluginAlreadyLoaded
	}

	// Validate the specification
	if err := m.loader.ValidatePlugin(spec); err != nil {
		return fmt.Errorf("plugin validation failed: %w", err)
	}

	// Load the plugin
	plugin, err := m.loader.LoadPlugin(spec)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Create managed plugin
	mp := &managedPlugin{
		plugin: plugin,
		spec:   spec,
	}

	// Notify lifecycle
	if m.lifecycle != nil {
		if err := m.lifecycle.OnPluginLoad(plugin); err != nil {
			// Clean up on lifecycle error
			m.loader.UnloadPlugin(plugin)
			return fmt.Errorf("lifecycle onload failed: %w", err)
		}
	}

	// Initialize the plugin
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := plugin.Start(ctx); err != nil {
		// Clean up on start error
		m.loader.UnloadPlugin(plugin)
		return fmt.Errorf("failed to start plugin: %w", err)
	}

	mp.startTime = time.Now()

	// Store the managed plugin
	m.plugins[spec.Name] = mp

	// Register in registry
	info := plugin.Info()
	info.LoadTime = mp.startTime
	info.Status = PluginStatusRunning
	
	if err := m.registry.RegisterPlugin(info); err != nil {
		// Log error but don't fail - plugin is already loaded
		fmt.Printf("Warning: failed to register plugin %s: %v\n", spec.Name, err)
	}

	// Notify lifecycle
	if m.lifecycle != nil {
		if err := m.lifecycle.OnPluginStart(plugin); err != nil {
			// Log error but don't fail
			fmt.Printf("Warning: lifecycle onstart failed for %s: %v\n", spec.Name, err)
		}
	}

	return nil
}

// UnloadPlugin unloads a plugin
func (m *pluginManager) UnloadPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mp, exists := m.plugins[name]
	if !exists {
		return ErrPluginNotFound
	}

	// Notify lifecycle
	if m.lifecycle != nil {
		if err := m.lifecycle.OnPluginStop(mp.plugin); err != nil {
			// Log but continue with unload
			fmt.Printf("Warning: lifecycle onstop failed for %s: %v\n", name, err)
		}
	}

	// Stop the plugin
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := mp.plugin.Stop(ctx); err != nil {
		// Log but continue with unload
		fmt.Printf("Warning: failed to stop plugin %s: %v\n", name, err)
	}

	// Unload the plugin
	if err := m.loader.UnloadPlugin(mp.plugin); err != nil {
		// Log but continue
		fmt.Printf("Warning: failed to unload plugin %s: %v\n", name, err)
	}

	// Notify lifecycle
	if m.lifecycle != nil {
		if err := m.lifecycle.OnPluginUnload(mp.plugin); err != nil {
			// Log error
			fmt.Printf("Warning: lifecycle onunload failed for %s: %v\n", name, err)
		}
	}

	// Remove from registry
	if err := m.registry.UnregisterPlugin(name); err != nil {
		// Log error
		fmt.Printf("Warning: failed to unregister plugin %s: %v\n", name, err)
	}

	// Remove from managed plugins
	delete(m.plugins, name)

	return nil
}

// ReloadPlugin reloads a plugin
func (m *pluginManager) ReloadPlugin(name string) error {
	m.mu.RLock()
	mp, exists := m.plugins[name]
	if !exists {
		m.mu.RUnlock()
		return ErrPluginNotFound
	}
	spec := mp.spec
	m.mu.RUnlock()

	// Unload the current plugin
	if err := m.UnloadPlugin(name); err != nil {
		return fmt.Errorf("failed to unload plugin: %w", err)
	}

	// Load the plugin again
	if err := m.LoadPlugin(spec); err != nil {
		return fmt.Errorf("failed to reload plugin: %w", err)
	}

	return nil
}

// ExecutePlugin executes a plugin request
func (m *pluginManager) ExecutePlugin(name string, request PluginRequest) (PluginResponse, error) {
	m.mu.RLock()
	mp, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return PluginResponse{}, ErrPluginNotFound
	}

	// Check plugin status
	status := mp.plugin.GetStatus()
	if status != PluginStatusRunning {
		return PluginResponse{}, fmt.Errorf("%w: plugin status is %s", ErrInvalidState, status)
	}

	// Execute through the executor if available
	if m.executor != nil {
		return m.executor.ExecutePlugin(mp.plugin, request)
	}

	// Direct execution
	ctx := context.Background()
	if request.Context.Timestamp.IsZero() {
		request.Context.Timestamp = time.Now()
	}

	startTime := time.Now()
	response, err := mp.plugin.Handle(ctx, request)
	
	// Set response metadata
	response.Metadata.ProcessingTime = time.Since(startTime)
	
	if err != nil {
		mp.mu.Lock()
		mp.lastError = err
		mp.mu.Unlock()

		// Notify lifecycle of error
		if m.lifecycle != nil {
			m.lifecycle.OnPluginError(mp.plugin, err)
		}

		return response, err
	}

	return response, nil
}

// ListPlugins returns a list of all loaded plugins
func (m *pluginManager) ListPlugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]PluginInfo, 0, len(m.plugins))
	for _, mp := range m.plugins {
		info := mp.plugin.Info()
		info.Status = mp.plugin.GetStatus()
		info.Uptime = time.Since(mp.startTime)
		
		mp.mu.RLock()
		if mp.lastError != nil {
			info.LastError = mp.lastError.Error()
		}
		mp.mu.RUnlock()

		infos = append(infos, info)
	}

	return infos
}

// GetPlugin returns information about a specific plugin
func (m *pluginManager) GetPlugin(name string) (PluginInfo, error) {
	m.mu.RLock()
	mp, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return PluginInfo{}, ErrPluginNotFound
	}

	info := mp.plugin.Info()
	info.Status = mp.plugin.GetStatus()
	info.Uptime = time.Since(mp.startTime)
	
	mp.mu.RLock()
	if mp.lastError != nil {
		info.LastError = mp.lastError.Error()
	}
	mp.mu.RUnlock()

	return info, nil
}

// HotSwapPlugin performs a hot swap of a plugin
func (m *pluginManager) HotSwapPlugin(name string, newVersion string) error {
	m.mu.RLock()
	mp, exists := m.plugins[name]
	if !exists {
		m.mu.RUnlock()
		return ErrPluginNotFound
	}
	oldSpec := mp.spec
	m.mu.RUnlock()

	// Create new spec with updated version
	newSpec := oldSpec
	newSpec.Version = newVersion

	// Export current state
	state, err := m.ExportPluginState(name)
	if err != nil {
		return fmt.Errorf("failed to export plugin state: %w", err)
	}

	// Prepare for shutdown
	if err := mp.plugin.PrepareShutdown(); err != nil {
		return fmt.Errorf("failed to prepare plugin for shutdown: %w", err)
	}

	// Load the new plugin version
	newPlugin, err := m.loader.LoadPlugin(newSpec)
	if err != nil {
		return fmt.Errorf("failed to load new plugin version: %w", err)
	}

	// Start the new plugin
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := newPlugin.Start(ctx); err != nil {
		m.loader.UnloadPlugin(newPlugin)
		return fmt.Errorf("failed to start new plugin version: %w", err)
	}

	// Import state into new plugin
	if err := newPlugin.ImportState(state); err != nil {
		// Rollback - stop and unload new plugin
		newPlugin.Stop(ctx)
		m.loader.UnloadPlugin(newPlugin)
		return fmt.Errorf("failed to import state into new plugin: %w", err)
	}

	// Swap the plugins
	m.mu.Lock()
	oldPlugin := mp.plugin
	mp.plugin = newPlugin
	mp.spec = newSpec
	mp.startTime = time.Now()
	mp.restartCount++
	m.mu.Unlock()

	// Stop and unload the old plugin
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()
	
	oldPlugin.Stop(ctx2)
	m.loader.UnloadPlugin(oldPlugin)

	// Update registry
	info := newPlugin.Info()
	info.LoadTime = mp.startTime
	info.Status = PluginStatusRunning
	m.registry.RegisterPlugin(info)

	return nil
}

// ExportPluginState exports a plugin's state
func (m *pluginManager) ExportPluginState(name string) ([]byte, error) {
	m.mu.RLock()
	mp, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrPluginNotFound
	}

	// Export through state manager if available
	if m.state != nil {
		return m.state.ExportState(mp.plugin)
	}

	// Direct export
	return mp.plugin.ExportState()
}

// ImportPluginState imports state into a plugin
func (m *pluginManager) ImportPluginState(name string, state []byte) error {
	m.mu.RLock()
	mp, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return ErrPluginNotFound
	}

	// Import through state manager if available
	if m.state != nil {
		return m.state.ImportState(mp.plugin, state)
	}

	// Direct import
	return mp.plugin.ImportState(state)
}