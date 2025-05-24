// Package runtime provides the foundational layer for the Blackhole Framework.
// This layer handles process orchestration, lifecycle management, and system health.
package runtime

import (
	"context"
	"time"
)

// Runtime is the main interface for the runtime layer.
// It provides process orchestration and system management capabilities.
type Runtime interface {
	// Service lifecycle management
	StartService(name string, config ServiceConfig) error
	StopService(name string) error
	RestartService(name string) error
	
	// Service status and health
	GetServiceStatus(name string) ServiceStatus
	ListServices() []ServiceInfo
	
	// Health monitoring
	RegisterHealthCheck(name string, check HealthCheck) error
	GetSystemHealth() SystemHealth
	
	// Configuration management
	LoadConfiguration(path string) error
	ReloadConfiguration() error
	GetConfiguration() Configuration
}

// ServiceConfig defines the configuration for a service.
type ServiceConfig struct {
	Name         string            `yaml:"name"`
	Command      string            `yaml:"command"`
	Args         []string          `yaml:"args"`
	Environment  map[string]string `yaml:"environment"`
	Resources    ResourceLimits    `yaml:"resources"`
	HealthCheck  HealthCheckConfig `yaml:"health_check"`
	Dependencies []string          `yaml:"dependencies"`
}

// ServiceStatus represents the current status of a service.
type ServiceStatus int

const (
	ServiceStatusUnknown ServiceStatus = iota
	ServiceStatusStarting
	ServiceStatusRunning
	ServiceStatusStopping
	ServiceStatusStopped
	ServiceStatusFailed
)

func (s ServiceStatus) String() string {
	switch s {
	case ServiceStatusStarting:
		return "STARTING"
	case ServiceStatusRunning:
		return "RUNNING"
	case ServiceStatusStopping:
		return "STOPPING"
	case ServiceStatusStopped:
		return "STOPPED"
	case ServiceStatusFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// ServiceInfo provides information about a service.
type ServiceInfo struct {
	Name      string        `json:"name"`
	Status    ServiceStatus `json:"status"`
	PID       int           `json:"pid,omitempty"`
	StartTime *time.Time    `json:"start_time,omitempty"`
	Uptime    time.Duration `json:"uptime"`
	Restarts  int           `json:"restarts"`
	LastError string        `json:"last_error,omitempty"`
}

// ResourceLimits defines resource constraints for a service.
type ResourceLimits struct {
	CPU    int    `yaml:"cpu"`     // CPU percentage (100 = 1 core)
	Memory int    `yaml:"memory"`  // Memory in MB
	IOWeight int  `yaml:"io_weight"` // I/O priority weight
}

// HealthCheck defines how to check service health.
type HealthCheck interface {
	Check(ctx context.Context) error
}

// HealthCheckConfig defines health check configuration.
type HealthCheckConfig struct {
	Type     string        `yaml:"type"`     // http, tcp, exec
	Target   string        `yaml:"target"`   // URL, address, or command
	Interval time.Duration `yaml:"interval"` // Check interval
	Timeout  time.Duration `yaml:"timeout"`  // Check timeout
	Retries  int           `yaml:"retries"`  // Retry attempts
}

// SystemHealth represents overall system health.
type SystemHealth struct {
	Status       HealthStatus           `json:"status"`
	Services     map[string]ServiceInfo `json:"services"`
	ResourceUsage ResourceUsage         `json:"resource_usage"`
	Uptime       time.Duration          `json:"uptime"`
	LastCheck    time.Time              `json:"last_check"`
}

// HealthStatus represents health status.
type HealthStatus int

const (
	HealthStatusUnknown HealthStatus = iota
	HealthStatusHealthy
	HealthStatusDegraded
	HealthStatusUnhealthy
)

func (h HealthStatus) String() string {
	switch h {
	case HealthStatusHealthy:
		return "HEALTHY"
	case HealthStatusDegraded:
		return "DEGRADED"
	case HealthStatusUnhealthy:
		return "UNHEALTHY"
	default:
		return "UNKNOWN"
	}
}

// ResourceUsage represents current resource usage.
type ResourceUsage struct {
	CPU    float64 `json:"cpu_percent"`
	Memory uint64  `json:"memory_bytes"`
	Disk   uint64  `json:"disk_bytes"`
	Network struct {
		BytesIn  uint64 `json:"bytes_in"`
		BytesOut uint64 `json:"bytes_out"`
	} `json:"network"`
}

// Configuration represents the runtime configuration.
type Configuration interface {
	GetServiceConfig(name string) (ServiceConfig, error)
	SetServiceConfig(name string, config ServiceConfig) error
	GetGlobalConfig() GlobalConfig
	SetGlobalConfig(config GlobalConfig) error
}

// GlobalConfig represents global runtime configuration.
type GlobalConfig struct {
	LogLevel    string        `yaml:"log_level"`
	SocketDir   string        `yaml:"socket_dir"`
	ServicesDir string        `yaml:"services_dir"`
	PIDFile     string        `yaml:"pid_file"`
	Timeouts    TimeoutConfig `yaml:"timeouts"`
}

// TimeoutConfig defines various timeout settings.
type TimeoutConfig struct {
	ServiceStart    time.Duration `yaml:"service_start"`
	ServiceStop     time.Duration `yaml:"service_stop"`
	ServiceRestart  time.Duration `yaml:"service_restart"`
	HealthCheck     time.Duration `yaml:"health_check"`
}