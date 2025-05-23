// Package mesh provides the service mesh components for the Blackhole platform.
// It implements protocol-level routing, connection pooling, and resource management
// for efficient inter-service communication using gRPC.
package mesh

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/handcraftdev/blackhole/internal/core/mesh/pool"
)

// ProtocolRouter implements protocol-level gRPC routing with intelligent resource management
type ProtocolRouter struct {
	// Service discovery and routing
	services         map[string][]Endpoint                         // service -> endpoints
	connectionPools  map[string]*pool.ProtocolLevelConnectionPool  // service -> connection pool
	serviceHealth    map[string]HealthStatus                       // service -> health status

	// Resource management
	resourceDetector *pool.ResourceDetector
	resourceManager  *pool.ResourceManager

	// Synchronization
	mutex  sync.RWMutex
	logger *zap.Logger

	// Configuration
	defaultUtilization int
}

// NewProtocolRouter creates a new protocol-level router
func NewProtocolRouter(logger *zap.Logger) *ProtocolRouter {
	resourceDetector := pool.NewResourceDetector(logger)
	
	// Start with 80% utilization as requested
	limits, err := resourceDetector.CalculateResourceLimits(80)
	if err != nil {
		logger.Warn("Failed to calculate resource limits, using defaults", zap.Error(err))
		// Fallback to conservative defaults
		limits = &pool.ResourceLimits{
			MaxTotalConnections:      100,
			MaxConnectionsPerService: 10,
			MaxConcurrentRequests:    500,
			MaxRequestsPerSecond:     1000,
			ConnectionIdleTimeout:    5 * time.Minute,
			ConnectionMaxAge:         30 * time.Minute,
		}
	}

	resourceManager := pool.NewResourceManager(limits, logger)

	return &ProtocolRouter{
		services:           make(map[string][]Endpoint),
		connectionPools:    make(map[string]*pool.ProtocolLevelConnectionPool),
		serviceHealth:      make(map[string]HealthStatus),
		resourceDetector:   resourceDetector,
		resourceManager:    resourceManager,
		logger:             logger,
		defaultUtilization: 80,
	}
}

// RegisterService registers a service endpoint for protocol-level routing
func (pr *ProtocolRouter) RegisterService(serviceName string, endpoint Endpoint) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	// Add endpoint to service list
	if endpoints, exists := pr.services[serviceName]; exists {
		// Check if endpoint already exists
		for i, existing := range endpoints {
			if existing.Socket == endpoint.Socket && existing.Address == endpoint.Address {
				// Update existing endpoint
				pr.services[serviceName][i] = endpoint
				pr.logger.Info("Updated service endpoint",
					zap.String("service", serviceName),
					zap.Bool("is_local", endpoint.IsLocal))
				return nil
			}
		}
		// Add new endpoint
		pr.services[serviceName] = append(endpoints, endpoint)
	} else {
		pr.services[serviceName] = []Endpoint{endpoint}
		pr.serviceHealth[serviceName] = HealthStatusUnknown
	}

	// Create connection pool for this service
	connectionPool, err := pool.NewProtocolLevelConnectionPool(
		serviceName,
		endpoint,
		pr.resourceManager,
		pr.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create connection pool for service %s: %w", serviceName, err)
	}

	pr.connectionPools[serviceName] = connectionPool

	pr.logger.Info("Registered service for protocol-level routing",
		zap.String("service", serviceName),
		zap.Bool("is_local", endpoint.IsLocal),
		zap.Int("total_services", len(pr.services)))

	return nil
}

// RouteRequest routes a gRPC request using protocol-level routing
func (pr *ProtocolRouter) RouteRequest(ctx context.Context, serviceName, fullMethod string, requestData []byte) ([]byte, error) {
	start := time.Now()

	// Get connection pool for service
	pr.mutex.RLock()
	connectionPool, exists := pr.connectionPools[serviceName]
	pr.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service %s not registered", serviceName)
	}

	// Parse the full method to extract service and method information
	// Format: /package.ServiceName/MethodName
	if !strings.HasPrefix(fullMethod, "/") {
		return nil, fmt.Errorf("invalid gRPC method format: %s", fullMethod)
	}

	// Route the request through the connection pool
	responseData, err := connectionPool.InvokeMethod(ctx, fullMethod, requestData)
	if err != nil {
		// Update service health on failure
		pr.updateServiceHealth(serviceName, HealthStatusDegraded)
		return nil, fmt.Errorf("failed to route request to %s: %w", serviceName, err)
	}

	// Update service health on success
	pr.updateServiceHealth(serviceName, HealthStatusHealthy)

	duration := time.Since(start)
	pr.logger.Debug("Request routed successfully",
		zap.String("service", serviceName),
		zap.String("method", fullMethod),
		zap.Duration("latency", duration))

	return responseData, nil
}

// DiscoverService returns endpoint information for a service
func (pr *ProtocolRouter) DiscoverService(serviceName string) (Endpoint, error) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	endpoints, exists := pr.services[serviceName]
	if !exists || len(endpoints) == 0 {
		return Endpoint{}, fmt.Errorf("service %s not found", serviceName)
	}

	// Return the first healthy endpoint
	for _, endpoint := range endpoints {
		// In a full implementation, you'd check endpoint health here
		return endpoint, nil
	}

	return endpoints[0], nil
}

// ListServices returns all registered services
func (pr *ProtocolRouter) ListServices() map[string][]Endpoint {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string][]Endpoint)
	for service, endpoints := range pr.services {
		endpointsCopy := make([]Endpoint, len(endpoints))
		copy(endpointsCopy, endpoints)
		result[service] = endpointsCopy
	}

	return result
}

// GetHealth returns the health status of a service
func (pr *ProtocolRouter) GetHealth(serviceName string) (HealthStatus, error) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	health, exists := pr.serviceHealth[serviceName]
	if !exists {
		return HealthStatusUnknown, fmt.Errorf("service %s not found", serviceName)
	}

	return health, nil
}

// UpdateHealth updates the health status of a service
func (pr *ProtocolRouter) UpdateHealth(serviceName string, health HealthStatus) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if _, exists := pr.services[serviceName]; !exists {
		return fmt.Errorf("service %s not registered", serviceName)
	}

	pr.serviceHealth[serviceName] = health
	pr.logger.Debug("Service health updated",
		zap.String("service", serviceName),
		zap.String("health", string(health)))

	return nil
}

// GetResourceUsage returns current resource utilization
func (pr *ProtocolRouter) GetResourceUsage() pool.ResourceUsage {
	return pr.resourceManager.GetResourceUsage()
}

// GetPoolStats returns statistics for all connection pools
func (pr *ProtocolRouter) GetPoolStats() map[string]pool.PoolStats {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	stats := make(map[string]pool.PoolStats)
	for serviceName, connectionPool := range pr.connectionPools {
		stats[serviceName] = connectionPool.GetPoolStats()
	}

	return stats
}

// UpdateResourceLimits updates resource limits with new utilization percentage
func (pr *ProtocolRouter) UpdateResourceLimits(utilizationPercent int) error {
	if utilizationPercent <= 0 || utilizationPercent > 100 {
		return fmt.Errorf("utilization percent must be between 1-100, got %d", utilizationPercent)
	}

	// Calculate new limits
	newLimits, err := pr.resourceDetector.CalculateResourceLimits(utilizationPercent)
	if err != nil {
		return fmt.Errorf("failed to calculate new resource limits: %w", err)
	}

	// Update resource manager
	pr.resourceManager.UpdateLimits(newLimits)

	pr.defaultUtilization = utilizationPercent
	pr.logger.Info("Resource limits updated",
		zap.Int("utilization_percent", utilizationPercent),
		zap.Int("max_total_connections", newLimits.MaxTotalConnections),
		zap.Int("max_connections_per_service", newLimits.MaxConnectionsPerService))

	return nil
}

// Close shuts down the protocol router and cleans up resources
func (pr *ProtocolRouter) Close() error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	var lastError error
	for serviceName, connectionPool := range pr.connectionPools {
		if err := connectionPool.Close(); err != nil {
			lastError = err
			pr.logger.Warn("Failed to close connection pool",
				zap.String("service", serviceName),
				zap.Error(err))
		}
	}

	// Clear all data
	pr.services = make(map[string][]Endpoint)
	pr.connectionPools = make(map[string]*pool.ProtocolLevelConnectionPool)
	pr.serviceHealth = make(map[string]HealthStatus)

	pr.logger.Info("Protocol router closed")
	return lastError
}

// updateServiceHealth is an internal method to update service health
func (pr *ProtocolRouter) updateServiceHealth(serviceName string, health HealthStatus) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()
	pr.serviceHealth[serviceName] = health
}

// IsRunning returns whether the protocol router is running
func (pr *ProtocolRouter) IsRunning() bool {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()
	return len(pr.services) > 0 || len(pr.connectionPools) > 0
}

// GetActiveMiddleware returns active middleware (placeholder for future implementation)
func (pr *ProtocolRouter) GetActiveMiddleware() []string {
	// Future implementation would return actual middleware
	return []string{"resource-management", "health-monitoring", "rate-limiting"}
}