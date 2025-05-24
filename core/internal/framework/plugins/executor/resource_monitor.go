package executor

import (
	"sync"
	"time"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
)

// resourceMonitor implements ResourceMonitor interface
type resourceMonitor struct {
	usage          map[string]plugins.PluginResourceUsage
	updateInterval time.Duration
	mu             sync.RWMutex
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(updateInterval time.Duration) ResourceMonitor {
	return &resourceMonitor{
		usage:          make(map[string]plugins.PluginResourceUsage),
		updateInterval: updateInterval,
	}
}

// StartMonitoring starts monitoring a plugin's resource usage
func (m *resourceMonitor) StartMonitoring(pluginName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Initialize with zero usage
	m.usage[pluginName] = plugins.PluginResourceUsage{}
	
	// TODO: Start actual monitoring goroutine
	return nil
}

// StopMonitoring stops monitoring a plugin's resource usage
func (m *resourceMonitor) StopMonitoring(pluginName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.usage, pluginName)
	return nil
}

// GetUsage returns a plugin's current resource usage
func (m *resourceMonitor) GetUsage(pluginName string) (plugins.PluginResourceUsage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	usage, exists := m.usage[pluginName]
	if !exists {
		return plugins.PluginResourceUsage{}, nil
	}
	
	return usage, nil
}