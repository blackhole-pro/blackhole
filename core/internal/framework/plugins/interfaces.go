// Package plugins provides the plugin management domain for the Blackhole Framework.
// This domain handles plugin loading, execution, isolation, and hot-swapping.
package plugins

import (
	"context"
	"time"
)

// PluginManager is the main interface for plugin management.
type PluginManager interface {
	// Plugin lifecycle management
	LoadPlugin(spec PluginSpec) error
	UnloadPlugin(name string) error
	ReloadPlugin(name string) error
	
	// Plugin execution
	ExecutePlugin(name string, request PluginRequest) (PluginResponse, error)
	
	// Plugin information
	ListPlugins() []PluginInfo
	GetPlugin(name string) (PluginInfo, error)
	
	// Hot swapping
	HotSwapPlugin(name string, newVersion string) error
	
	// Plugin state management
	ExportPluginState(name string) ([]byte, error)
	ImportPluginState(name string, state []byte) error
}

// Plugin represents a loaded plugin instance.
type Plugin interface {
	// Plugin metadata
	Info() PluginInfo
	
	// Plugin lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	
	// Plugin execution
	Handle(ctx context.Context, request PluginRequest) (PluginResponse, error)
	
	// Health and status
	HealthCheck() error
	GetStatus() PluginStatus
	
	// Hot swapping support
	PrepareShutdown() error
	ExportState() ([]byte, error)
	ImportState(state []byte) error
}

// PluginSpec defines how to load and configure a plugin.
type PluginSpec struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Source       PluginSource           `json:"source"`
	Config       map[string]interface{} `json:"config"`
	Dependencies []PluginDependency     `json:"dependencies"`
	Resources    PluginResources        `json:"resources"`
	Isolation    IsolationLevel         `json:"isolation"`
}

// PluginSource defines where to load the plugin from.
type PluginSource struct {
	Type SourceType `json:"type"` // local, remote, marketplace
	Path string     `json:"path"` // File path, URL, or marketplace ID
	Hash string     `json:"hash"` // Verification hash
}

// SourceType defines the type of plugin source.
type SourceType string

const (
	SourceTypeLocal       SourceType = "local"
	SourceTypeRemote      SourceType = "remote"
	SourceTypeMarketplace SourceType = "marketplace"
)

// PluginDependency defines a plugin dependency.
type PluginDependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Optional bool  `json:"optional"`
}

// PluginResources defines resource requirements for a plugin.
type PluginResources struct {
	CPU    int    `json:"cpu"`     // CPU percentage
	Memory int    `json:"memory"`  // Memory in MB
	Disk   int    `json:"disk"`    // Disk space in MB
	Network int   `json:"network"` // Network bandwidth in Mbps
}

// IsolationLevel defines the level of isolation for a plugin.
type IsolationLevel string

const (
	IsolationNone       IsolationLevel = "none"       // Same process
	IsolationThread     IsolationLevel = "thread"     // Separate thread
	IsolationProcess    IsolationLevel = "process"    // Separate process
	IsolationContainer  IsolationLevel = "container"  // Container isolation
	IsolationVM         IsolationLevel = "vm"         // VM isolation
)

// PluginInfo provides information about a plugin.
type PluginInfo struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	Author      string         `json:"author"`
	License     string         `json:"license"`
	Homepage    string         `json:"homepage"`
	Repository  string         `json:"repository"`
	
	// Runtime information
	Status      PluginStatus   `json:"status"`
	LoadTime    time.Time      `json:"load_time"`
	Uptime      time.Duration  `json:"uptime"`
	LastError   string         `json:"last_error,omitempty"`
	
	// Resource usage
	ResourceUsage PluginResourceUsage `json:"resource_usage"`
	
	// Capabilities
	Capabilities []PluginCapability `json:"capabilities"`
	Permissions  []PluginPermission `json:"permissions"`
}

// PluginStatus represents the current status of a plugin.
type PluginStatus string

const (
	PluginStatusUnknown  PluginStatus = "unknown"
	PluginStatusLoading  PluginStatus = "loading"
	PluginStatusLoaded   PluginStatus = "loaded"
	PluginStatusStarting PluginStatus = "starting"
	PluginStatusRunning  PluginStatus = "running"
	PluginStatusStopping PluginStatus = "stopping"
	PluginStatusStopped  PluginStatus = "stopped"
	PluginStatusFailed   PluginStatus = "failed"
)

// PluginResourceUsage represents current resource usage by a plugin.
type PluginResourceUsage struct {
	CPU     float64 `json:"cpu_percent"`
	Memory  uint64  `json:"memory_bytes"`
	Disk    uint64  `json:"disk_bytes"`
	Network struct {
		BytesIn  uint64 `json:"bytes_in"`
		BytesOut uint64 `json:"bytes_out"`
	} `json:"network"`
}

// PluginCapability defines what a plugin can do.
type PluginCapability string

const (
	CapabilityStorage      PluginCapability = "storage"
	CapabilityNetworking   PluginCapability = "networking"
	CapabilityComputation  PluginCapability = "computation"
	CapabilityUI           PluginCapability = "ui"
	CapabilityIntegration  PluginCapability = "integration"
	CapabilityAnalytics    PluginCapability = "analytics"
)

// PluginPermission defines what resources a plugin can access.
type PluginPermission string

const (
	PermissionFileSystem   PluginPermission = "filesystem"
	PermissionNetwork      PluginPermission = "network"
	PermissionSystem       PluginPermission = "system"
	PermissionOtherPlugins PluginPermission = "other_plugins"
	PermissionUserData     PluginPermission = "user_data"
)

// PluginRequest represents a request to a plugin.
type PluginRequest struct {
	ID      string                 `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	Data    []byte                 `json:"data,omitempty"`
	Context RequestContext         `json:"context"`
}

// PluginResponse represents a response from a plugin.
type PluginResponse struct {
	ID       string                 `json:"id"`
	Success  bool                   `json:"success"`
	Result   map[string]interface{} `json:"result,omitempty"`
	Data     []byte                 `json:"data,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata ResponseMetadata       `json:"metadata"`
}

// RequestContext provides context for plugin requests.
type RequestContext struct {
	UserID    string            `json:"user_id,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// ResponseMetadata provides metadata for plugin responses.
type ResponseMetadata struct {
	ProcessingTime time.Duration `json:"processing_time"`
	CacheHit       bool          `json:"cache_hit,omitempty"`
	ResourceUsage  struct {
		CPU    float64 `json:"cpu_ms"`
		Memory uint64  `json:"memory_bytes"`
	} `json:"resource_usage"`
}

// PluginRegistry handles plugin discovery and registration.
type PluginRegistry interface {
	// Plugin discovery
	DiscoverPlugins(path string) ([]PluginSpec, error)
	SearchPlugins(criteria SearchCriteria) ([]PluginInfo, error)
	
	// Plugin registration
	RegisterPlugin(info PluginInfo) error
	UnregisterPlugin(name string) error
	
	// Plugin marketplace integration
	FetchFromMarketplace(id string) (PluginSpec, error)
	PublishToMarketplace(spec PluginSpec) error
}

// SearchCriteria defines search criteria for plugins.
type SearchCriteria struct {
	Name         string               `json:"name,omitempty"`
	Category     string               `json:"category,omitempty"`
	Capabilities []PluginCapability   `json:"capabilities,omitempty"`
	Author       string               `json:"author,omitempty"`
	License      string               `json:"license,omitempty"`
	MinVersion   string               `json:"min_version,omitempty"`
	MaxVersion   string               `json:"max_version,omitempty"`
}

// PluginLoader handles loading and unloading of plugins.
type PluginLoader interface {
	LoadPlugin(spec PluginSpec) (Plugin, error)
	UnloadPlugin(plugin Plugin) error
	ValidatePlugin(spec PluginSpec) error
}

// PluginExecutor handles plugin execution and isolation.
type PluginExecutor interface {
	ExecutePlugin(plugin Plugin, request PluginRequest) (PluginResponse, error)
	GetExecutionEnvironment(plugin Plugin) (ExecutionEnvironment, error)
	CreateIsolationBoundary(level IsolationLevel) (IsolationBoundary, error)
}

// ExecutionEnvironment represents the environment where a plugin executes.
type ExecutionEnvironment interface {
	GetResourceLimits() PluginResources
	GetEnvironmentVariables() map[string]string
	GetWorkingDirectory() string
	GetTempDirectory() string
}

// IsolationBoundary represents an isolation boundary for plugin execution.
type IsolationBoundary interface {
	Enter() error
	Exit() error
	EnforceResourceLimits(limits PluginResources) error
	GetResourceUsage() (PluginResourceUsage, error)
}

// PluginLifecycle handles plugin lifecycle events.
type PluginLifecycle interface {
	OnPluginLoad(plugin Plugin) error
	OnPluginStart(plugin Plugin) error
	OnPluginStop(plugin Plugin) error
	OnPluginUnload(plugin Plugin) error
	OnPluginError(plugin Plugin, err error) error
}

// StateManager handles plugin state management and migration.
type StateManager interface {
	SaveState(plugin Plugin) error
	LoadState(plugin Plugin) error
	MigrateState(plugin Plugin, fromVersion, toVersion string) error
	ExportState(plugin Plugin) ([]byte, error)
	ImportState(plugin Plugin, state []byte) error
}