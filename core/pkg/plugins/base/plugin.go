// Package base provides the base interfaces and types for all Blackhole plugins
package base

import (
	"context"
	"time"
)

// Plugin is the base interface that all plugins must implement
type Plugin interface {
	// Initialize initializes the plugin with the given configuration
	Initialize(ctx context.Context, config map[string]interface{}) error
	
	// Start starts the plugin
	Start(ctx context.Context) error
	
	// Stop stops the plugin
	Stop(ctx context.Context) error
	
	// HealthCheck checks if the plugin is healthy
	HealthCheck(ctx context.Context) error
	
	// Info returns information about the plugin
	Info() PluginInfo
}

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	Name         string
	Version      string
	Description  string
	Author       string
	License      string
	Homepage     string
	Repository   string
	Capabilities []string
	Permissions  []Permission
}

// Permission describes a resource permission required by the plugin
type Permission struct {
	Resource    string
	Actions     []string
	Description string
}

// PluginStatus represents the current status of a plugin
type PluginStatus string

const (
	PluginStatusUnknown  PluginStatus = "unknown"
	PluginStatusLoaded   PluginStatus = "loaded"
	PluginStatusStarting PluginStatus = "starting"
	PluginStatusRunning  PluginStatus = "running"
	PluginStatusStopping PluginStatus = "stopping"
	PluginStatusStopped  PluginStatus = "stopped"
	PluginStatusError    PluginStatus = "error"
)

// MeshPlugin extends Plugin with mesh networking capabilities
type MeshPlugin interface {
	Plugin
	
	// RegisterWithMesh registers the plugin with the mesh network
	RegisterWithMesh(ctx context.Context, meshEndpoint string) error
	
	// GetServiceName returns the service name for mesh registration
	GetServiceName() string
	
	// GetEndpoints returns the endpoints this plugin listens on
	GetEndpoints() []Endpoint
}

// Endpoint represents a network endpoint
type Endpoint struct {
	Protocol string // unix, tcp, grpc
	Address  string
	Secure   bool
}

// StatefulPlugin extends Plugin with state management capabilities
type StatefulPlugin interface {
	Plugin
	
	// ExportState exports the plugin's current state
	ExportState(ctx context.Context) ([]byte, error)
	
	// ImportState imports a previously exported state
	ImportState(ctx context.Context, state []byte) error
}

// PluginMetrics contains runtime metrics for a plugin
type PluginMetrics struct {
	StartTime         time.Time
	RequestsHandled   int64
	RequestsFailed    int64
	AverageLatencyMs  float64
	MemoryUsageMB     float64
	CPUUsagePercent   float64
	LastHealthCheck   time.Time
	HealthCheckPassed bool
}