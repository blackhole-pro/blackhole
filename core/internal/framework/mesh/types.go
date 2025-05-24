package mesh

import (
	"time"
)

// ServiceEndpoint represents a service endpoint for routing
type ServiceEndpoint struct {
	Socket      string            `json:"socket,omitempty"`
	Address     string            `json:"address,omitempty"`
	IsLocal     bool              `json:"is_local"`
	LastUpdated time.Time         `json:"last_updated"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// HealthStatus represents the health status of a service
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// ServiceInfo represents information about a registered service
type ServiceInfo struct {
	Name        string            `json:"name"`
	Endpoints   []ServiceEndpoint `json:"endpoints"`
	Health      HealthStatus      `json:"health"`
	LastSeen    time.Time         `json:"last_seen"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// RouteRequest represents a routing request
type RouteRequest struct {
	ServiceName string      `json:"service_name"`
	Method      string      `json:"method"`
	Payload     interface{} `json:"payload"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// RouteResponse represents a routing response
type RouteResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data,omitempty"`
	Error      string      `json:"error,omitempty"`
	Latency    time.Duration `json:"latency"`
	ServerInfo string      `json:"server_info,omitempty"`
}