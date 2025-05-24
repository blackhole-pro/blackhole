package pool

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ResourceLimits defines resource constraints for connection management
type ResourceLimits struct {
	// Connection limits
	MaxTotalConnections      int `yaml:"max_total_connections" json:"max_total_connections"`
	MaxConnectionsPerService int `yaml:"max_connections_per_service" json:"max_connections_per_service"`

	// Request limits
	MaxConcurrentRequests int `yaml:"max_concurrent_requests" json:"max_concurrent_requests"`
	MaxRequestsPerSecond  int `yaml:"max_requests_per_second" json:"max_requests_per_second"`

	// Connection lifecycle
	ConnectionIdleTimeout time.Duration `yaml:"connection_idle_timeout" json:"connection_idle_timeout"`
	ConnectionMaxAge      time.Duration `yaml:"connection_max_age" json:"connection_max_age"`

	// System information (read-only, for monitoring)
	DetectedCPUCores   int   `yaml:"-" json:"detected_cpu_cores"`
	DetectedMemoryMB   int64 `yaml:"-" json:"detected_memory_mb"`
	DetectedFDLimit    int64 `yaml:"-" json:"detected_fd_limit"`
	UtilizationPercent int   `yaml:"-" json:"utilization_percent"`
}

// ResourceManager manages runtime resource limits and tracking
type ResourceManager struct {
	limits *ResourceLimits
	mutex  sync.RWMutex

	// Runtime tracking
	currentTotalConnections int
	currentActiveRequests   int
	connectionCounts        map[string]int // service -> connection count

	// Rate limiting
	requestCounter *RateLimiter

	logger *zap.Logger
}

// NewResourceManager creates a new resource manager
func NewResourceManager(limits *ResourceLimits, logger *zap.Logger) *ResourceManager {
	return &ResourceManager{
		limits:           limits,
		connectionCounts: make(map[string]int),
		requestCounter:   NewRateLimiter(limits.MaxRequestsPerSecond),
		logger:           logger,
	}
}

// CanCreateConnection checks if a new connection can be created
func (rm *ResourceManager) CanCreateConnection(serviceName string) error {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	// Check total connection limit
	if rm.currentTotalConnections >= rm.limits.MaxTotalConnections {
		return fmt.Errorf("total connection limit reached (%d/%d)",
			rm.currentTotalConnections, rm.limits.MaxTotalConnections)
	}

	// Check per-service connection limit
	serviceConnections := rm.connectionCounts[serviceName]
	if serviceConnections >= rm.limits.MaxConnectionsPerService {
		return fmt.Errorf("service connection limit reached for %s (%d/%d)",
			serviceName, serviceConnections, rm.limits.MaxConnectionsPerService)
	}

	return nil
}

// CanMakeRequest checks if a new request can be made (rate limiting + concurrency)
func (rm *ResourceManager) CanMakeRequest() error {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	// Check concurrent request limit
	if rm.currentActiveRequests >= rm.limits.MaxConcurrentRequests {
		return fmt.Errorf("concurrent request limit reached (%d/%d)",
			rm.currentActiveRequests, rm.limits.MaxConcurrentRequests)
	}

	// Check rate limit
	if !rm.requestCounter.Allow() {
		return fmt.Errorf("request rate limit exceeded (%d requests/second)",
			rm.limits.MaxRequestsPerSecond)
	}

	return nil
}

// TrackConnectionCreated records a new connection
func (rm *ResourceManager) TrackConnectionCreated(serviceName string) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.currentTotalConnections++
	rm.connectionCounts[serviceName]++

	rm.logger.Debug("Connection created",
		zap.String("service", serviceName),
		zap.Int("total_connections", rm.currentTotalConnections),
		zap.Int("service_connections", rm.connectionCounts[serviceName]))
}

// TrackConnectionClosed records a closed connection
func (rm *ResourceManager) TrackConnectionClosed(serviceName string) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if rm.currentTotalConnections > 0 {
		rm.currentTotalConnections--
	}

	if rm.connectionCounts[serviceName] > 0 {
		rm.connectionCounts[serviceName]--
	}

	rm.logger.Debug("Connection closed",
		zap.String("service", serviceName),
		zap.Int("total_connections", rm.currentTotalConnections),
		zap.Int("service_connections", rm.connectionCounts[serviceName]))
}

// TrackRequestStarted records a request start
func (rm *ResourceManager) TrackRequestStarted() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.currentActiveRequests++
}

// TrackRequestCompleted records a request completion
func (rm *ResourceManager) TrackRequestCompleted() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	if rm.currentActiveRequests > 0 {
		rm.currentActiveRequests--
	}
}

// GetResourceUsage returns current resource usage statistics
func (rm *ResourceManager) GetResourceUsage() ResourceUsage {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	// Copy connection counts
	serviceCounts := make(map[string]int)
	for service, count := range rm.connectionCounts {
		serviceCounts[service] = count
	}

	return ResourceUsage{
		TotalConnections:        rm.currentTotalConnections,
		MaxTotalConnections:     rm.limits.MaxTotalConnections,
		ActiveRequests:          rm.currentActiveRequests,
		MaxConcurrentRequests:   rm.limits.MaxConcurrentRequests,
		ConnectionsPerService:   serviceCounts,
		MaxConnectionsPerService: rm.limits.MaxConnectionsPerService,
		
		// Utilization percentages
		ConnectionUtilization: float64(rm.currentTotalConnections) / float64(rm.limits.MaxTotalConnections) * 100,
		RequestUtilization:    float64(rm.currentActiveRequests) / float64(rm.limits.MaxConcurrentRequests) * 100,
	}
}

// UpdateLimits updates resource limits at runtime
func (rm *ResourceManager) UpdateLimits(newLimits *ResourceLimits) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	oldLimits := rm.limits
	rm.limits = newLimits

	// Update rate limiter
	rm.requestCounter.UpdateLimit(newLimits.MaxRequestsPerSecond)

	rm.logger.Info("Resource limits updated",
		zap.Int("old_max_connections", oldLimits.MaxTotalConnections),
		zap.Int("new_max_connections", newLimits.MaxTotalConnections),
		zap.Int("old_max_requests", oldLimits.MaxRequestsPerSecond),
		zap.Int("new_max_requests", newLimits.MaxRequestsPerSecond))
}

// ResourceUsage represents current resource usage
type ResourceUsage struct {
	// Connection usage
	TotalConnections         int            `json:"total_connections"`
	MaxTotalConnections      int            `json:"max_total_connections"`
	ConnectionsPerService    map[string]int `json:"connections_per_service"`
	MaxConnectionsPerService int            `json:"max_connections_per_service"`

	// Request usage
	ActiveRequests        int `json:"active_requests"`
	MaxConcurrentRequests int `json:"max_concurrent_requests"`

	// Utilization percentages
	ConnectionUtilization float64 `json:"connection_utilization_percent"`
	RequestUtilization    float64 `json:"request_utilization_percent"`
}

// IsOverloaded returns true if the system is approaching resource limits
func (ru *ResourceUsage) IsOverloaded() bool {
	// Consider overloaded if either connections or requests are >90% utilized
	return ru.ConnectionUtilization > 90.0 || ru.RequestUtilization > 90.0
}

// IsHealthy returns true if the system has reasonable resource availability
func (ru *ResourceUsage) IsHealthy() bool {
	// Consider healthy if both connections and requests are <80% utilized
	return ru.ConnectionUtilization < 80.0 && ru.RequestUtilization < 80.0
}

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	limit     int
	tokens    int
	lastRefill time.Time
	mutex     sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	return &RateLimiter{
		limit:      requestsPerSecond,
		tokens:     requestsPerSecond,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed under the rate limit
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	// Refill tokens based on elapsed time
	if elapsed >= time.Second {
		rl.tokens = rl.limit
		rl.lastRefill = now
	}

	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// UpdateLimit updates the rate limit
func (rl *RateLimiter) UpdateLimit(newLimit int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.limit = newLimit
	if rl.tokens > newLimit {
		rl.tokens = newLimit
	}
}