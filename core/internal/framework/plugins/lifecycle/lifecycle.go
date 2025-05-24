// Package lifecycle provides plugin lifecycle management for the Blackhole Framework.
package lifecycle

import (
	"log"
	"sync"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
)

// lifecycleManager implements the PluginLifecycle interface
type lifecycleManager struct {
	handlers []LifecycleHandler
	mu       sync.RWMutex
}

// LifecycleHandler is a callback interface for lifecycle events
type LifecycleHandler interface {
	OnLoad(plugin plugins.Plugin) error
	OnStart(plugin plugins.Plugin) error
	OnStop(plugin plugins.Plugin) error
	OnUnload(plugin plugins.Plugin) error
	OnError(plugin plugins.Plugin, err error)
	OnCrash(plugin plugins.Plugin) error
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager() plugins.PluginLifecycle {
	return &lifecycleManager{
		handlers: make([]LifecycleHandler, 0),
	}
}

// RegisterHandler adds a lifecycle handler
func (m *lifecycleManager) RegisterHandler(handler LifecycleHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

// OnPluginLoad is called when a plugin is loaded
func (m *lifecycleManager) OnPluginLoad(plugin plugins.Plugin) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, handler := range m.handlers {
		if err := handler.OnLoad(plugin); err != nil {
			return err
		}
	}
	
	log.Printf("[Lifecycle] Plugin loaded: %s", plugin.Info().Name)
	return nil
}

// OnPluginStart is called when a plugin starts
func (m *lifecycleManager) OnPluginStart(plugin plugins.Plugin) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, handler := range m.handlers {
		if err := handler.OnStart(plugin); err != nil {
			return err
		}
	}
	
	log.Printf("[Lifecycle] Plugin started: %s", plugin.Info().Name)
	return nil
}

// OnPluginStop is called when a plugin stops
func (m *lifecycleManager) OnPluginStop(plugin plugins.Plugin) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, handler := range m.handlers {
		if err := handler.OnStop(plugin); err != nil {
			return err
		}
	}
	
	log.Printf("[Lifecycle] Plugin stopped: %s", plugin.Info().Name)
	return nil
}

// OnPluginUnload is called when a plugin is unloaded
func (m *lifecycleManager) OnPluginUnload(plugin plugins.Plugin) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, handler := range m.handlers {
		if err := handler.OnUnload(plugin); err != nil {
			return err
		}
	}
	
	log.Printf("[Lifecycle] Plugin unloaded: %s", plugin.Info().Name)
	return nil
}

// OnPluginError is called when a plugin encounters an error
func (m *lifecycleManager) OnPluginError(plugin plugins.Plugin, err error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, handler := range m.handlers {
		handler.OnError(plugin, err)
	}
	
	log.Printf("[Lifecycle] Plugin error in %s: %v", plugin.Info().Name, err)
	return nil
}

// OnPluginCrash is called when a plugin crashes
func (m *lifecycleManager) OnPluginCrash(plugin plugins.Plugin) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, handler := range m.handlers {
		if err := handler.OnCrash(plugin); err != nil {
			return err
		}
	}
	
	log.Printf("[Lifecycle] Plugin crashed: %s", plugin.Info().Name)
	return nil
}

// DefaultLifecycleHandler provides a default implementation of LifecycleHandler
type DefaultLifecycleHandler struct{}

// OnLoad is called when a plugin is loaded
func (h *DefaultLifecycleHandler) OnLoad(plugin plugins.Plugin) error {
	return nil
}

// OnStart is called when a plugin starts
func (h *DefaultLifecycleHandler) OnStart(plugin plugins.Plugin) error {
	return nil
}

// OnStop is called when a plugin stops  
func (h *DefaultLifecycleHandler) OnStop(plugin plugins.Plugin) error {
	return nil
}

// OnUnload is called when a plugin is unloaded
func (h *DefaultLifecycleHandler) OnUnload(plugin plugins.Plugin) error {
	return nil
}

// OnError is called when a plugin encounters an error
func (h *DefaultLifecycleHandler) OnError(plugin plugins.Plugin, err error) {
	// Default: log the error
	log.Printf("Plugin %s error: %v", plugin.Info().Name, err)
}

// OnCrash is called when a plugin crashes
func (h *DefaultLifecycleHandler) OnCrash(plugin plugins.Plugin) error {
	// Default: try to restart the plugin
	log.Printf("Plugin %s crashed, attempting restart", plugin.Info().Name)
	return nil
}