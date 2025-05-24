package pool

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/handcraftdev/blackhole/internal/core/mesh"
)

// ProtocolLevelConnectionPool manages multiple gRPC connections for protocol-level routing
type ProtocolLevelConnectionPool struct {
	serviceName    string
	endpoint       mesh.Endpoint
	
	// Connection pool
	connections     []*PooledConnection
	maxSize         int
	currentSize     int
	roundRobinIndex int
	
	// Resource management
	resourceManager *ResourceManager
	
	// Synchronization
	mutex  sync.RWMutex
	logger *zap.Logger
	
	// Health monitoring
	healthCheckInterval time.Duration
	stopHealthCheck     chan struct{}
	healthCheckStopped  chan struct{}
}

// PooledConnection wraps a gRPC connection with usage tracking
type PooledConnection struct {
	conn            *grpc.ClientConn
	activeRequests  int32
	createdAt       time.Time
	lastUsed        time.Time
	healthy         bool
	
	// Metrics
	totalRequests   int64
	failedRequests  int64
	avgLatency      time.Duration
	
	mutex          sync.RWMutex
}

// NewProtocolLevelConnectionPool creates a new connection pool for protocol-level routing
func NewProtocolLevelConnectionPool(
	serviceName string,
	endpoint mesh.Endpoint,
	resourceManager *ResourceManager,
	logger *zap.Logger,
) (*ProtocolLevelConnectionPool, error) {
	
	pool := &ProtocolLevelConnectionPool{
		serviceName:         serviceName,
		endpoint:           endpoint,
		connections:        make([]*PooledConnection, 0),
		maxSize:           resourceManager.limits.MaxConnectionsPerService,
		resourceManager:   resourceManager,
		logger:            logger,
		healthCheckInterval: 30 * time.Second,
		stopHealthCheck:    make(chan struct{}),
		healthCheckStopped: make(chan struct{}),
	}
	
	// Start health check routine
	go pool.healthCheckRoutine()
	
	return pool, nil
}

// GetConnection returns an available connection from the pool
func (p *ProtocolLevelConnectionPool) GetConnection(ctx context.Context) (*PooledConnection, error) {
	// Check rate limiting and concurrency limits
	if err := p.resourceManager.CanMakeRequest(); err != nil {
		return nil, fmt.Errorf("resource limit exceeded: %w", err)
	}
	
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	// Find the best available connection
	bestConn := p.selectBestConnection()
	
	// If no good connection exists and we're under the limit, create a new one
	if bestConn == nil && p.currentSize < p.maxSize {
		if err := p.resourceManager.CanCreateConnection(p.serviceName); err != nil {
			return nil, fmt.Errorf("cannot create new connection: %w", err)
		}
		
		newConn, err := p.createNewConnection(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create new connection: %w", err)
		}
		
		p.connections = append(p.connections, newConn)
		p.currentSize++
		p.resourceManager.TrackConnectionCreated(p.serviceName)
		bestConn = newConn
		
		p.logger.Debug("Created new connection",
			zap.String("service", p.serviceName),
			zap.Int("pool_size", p.currentSize))
	}
	
	if bestConn == nil {
		return nil, fmt.Errorf("no available connections and pool at maximum size (%d)", p.maxSize)
	}
	
	// Track request start
	bestConn.mutex.Lock()
	bestConn.activeRequests++
	bestConn.lastUsed = time.Now()
	bestConn.mutex.Unlock()
	
	p.resourceManager.TrackRequestStarted()
	
	return bestConn, nil
}

// ReleaseConnection returns a connection to the pool after use
func (p *ProtocolLevelConnectionPool) ReleaseConnection(conn *PooledConnection, requestLatency time.Duration, success bool) {
	conn.mutex.Lock()
	if conn.activeRequests > 0 {
		conn.activeRequests--
	}
	conn.totalRequests++
	if !success {
		conn.failedRequests++
	}
	
	// Update average latency (simple moving average)
	if conn.avgLatency == 0 {
		conn.avgLatency = requestLatency
	} else {
		conn.avgLatency = (conn.avgLatency + requestLatency) / 2
	}
	conn.mutex.Unlock()
	
	p.resourceManager.TrackRequestCompleted()
}

// selectBestConnection chooses the best available connection based on load and health
func (p *ProtocolLevelConnectionPool) selectBestConnection() *PooledConnection {
	var bestConn *PooledConnection
	lowestLoad := int32(^uint32(0) >> 1) // Max int32
	
	for _, conn := range p.connections {
		// Skip unhealthy connections
		if !conn.isHealthy() {
			continue
		}
		
		conn.mutex.RLock()
		activeRequests := conn.activeRequests
		conn.mutex.RUnlock()
		
		// Choose connection with lowest active requests
		if activeRequests < lowestLoad {
			bestConn = conn
			lowestLoad = activeRequests
		}
	}
	
	return bestConn
}

// createNewConnection creates a new gRPC connection
func (p *ProtocolLevelConnectionPool) createNewConnection(ctx context.Context) (*PooledConnection, error) {
	// Create gRPC connection based on endpoint type
	var conn *grpc.ClientConn
	var err error
	
	if p.endpoint.IsLocal && p.endpoint.Socket != "" {
		// Local service with Unix socket
		dialOpts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
				dialer := &net.Dialer{}
				return dialer.DialContext(ctx, "unix", addr)
			}),
		}
		
		conn, err = grpc.DialContext(ctx, p.endpoint.Socket, dialOpts...)
	} else if p.endpoint.Address != "" {
		// Remote service with TCP address
		dialOpts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		
		conn, err = grpc.DialContext(ctx, p.endpoint.Address, dialOpts...)
	} else {
		return nil, fmt.Errorf("invalid endpoint: requires either socket or address")
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to dial service: %w", err)
	}
	
	pooledConn := &PooledConnection{
		conn:      conn,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		healthy:   true,
	}
	
	return pooledConn, nil
}

// InvokeMethod invokes a gRPC method using protocol-level routing
func (p *ProtocolLevelConnectionPool) InvokeMethod(ctx context.Context, fullMethod string, reqBytes []byte) ([]byte, error) {
	start := time.Now()
	
	// Get connection from pool
	conn, err := p.GetConnection(ctx)
	if err != nil {
		return nil, err
	}
	
	// Execute the gRPC call
	var respBytes []byte
	err = conn.conn.Invoke(ctx, fullMethod, reqBytes, &respBytes)
	
	// Record metrics and release connection
	duration := time.Since(start)
	success := err == nil
	p.ReleaseConnection(conn, duration, success)
	
	if err != nil {
		return nil, fmt.Errorf("gRPC invocation failed: %w", err)
	}
	
	return respBytes, nil
}

// Close closes all connections in the pool
func (p *ProtocolLevelConnectionPool) Close() error {
	// Stop health check routine
	close(p.stopHealthCheck)
	<-p.healthCheckStopped
	
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	var lastError error
	for _, conn := range p.connections {
		if err := conn.conn.Close(); err != nil {
			lastError = err
			p.logger.Warn("Failed to close connection",
				zap.String("service", p.serviceName),
				zap.Error(err))
		}
		p.resourceManager.TrackConnectionClosed(p.serviceName)
	}
	
	p.connections = nil
	p.currentSize = 0
	
	p.logger.Info("Connection pool closed", zap.String("service", p.serviceName))
	return lastError
}

// GetPoolStats returns current pool statistics
func (p *ProtocolLevelConnectionPool) GetPoolStats() PoolStats {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	var totalRequests, failedRequests int64
	var totalActiveRequests int32
	var avgLatency time.Duration
	healthyConnections := 0
	
	for _, conn := range p.connections {
		conn.mutex.RLock()
		totalRequests += conn.totalRequests
		failedRequests += conn.failedRequests
		totalActiveRequests += conn.activeRequests
		if conn.avgLatency > 0 {
			if avgLatency == 0 {
				avgLatency = conn.avgLatency
			} else {
				avgLatency = (avgLatency + conn.avgLatency) / 2
			}
		}
		if conn.healthy {
			healthyConnections++
		}
		conn.mutex.RUnlock()
	}
	
	var successRate float64
	if totalRequests > 0 {
		successRate = float64(totalRequests-failedRequests) / float64(totalRequests) * 100
	}
	
	return PoolStats{
		ServiceName:         p.serviceName,
		TotalConnections:    p.currentSize,
		HealthyConnections:  healthyConnections,
		MaxConnections:      p.maxSize,
		ActiveRequests:      int(totalActiveRequests),
		TotalRequests:       totalRequests,
		FailedRequests:      failedRequests,
		SuccessRate:         successRate,
		AverageLatency:      avgLatency,
	}
}

// PoolStats represents connection pool statistics
type PoolStats struct {
	ServiceName        string        `json:"service_name"`
	TotalConnections   int           `json:"total_connections"`
	HealthyConnections int           `json:"healthy_connections"`
	MaxConnections     int           `json:"max_connections"`
	ActiveRequests     int           `json:"active_requests"`
	TotalRequests      int64         `json:"total_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	SuccessRate        float64       `json:"success_rate_percent"`
	AverageLatency     time.Duration `json:"average_latency"`
}

// isHealthy checks if a connection is healthy
func (pc *PooledConnection) isHealthy() bool {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	
	// Check if connection is marked as healthy and gRPC state is good
	if !pc.healthy {
		return false
	}
	
	state := pc.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// healthCheckRoutine periodically checks connection health
func (p *ProtocolLevelConnectionPool) healthCheckRoutine() {
	defer close(p.healthCheckStopped)
	
	ticker := time.NewTicker(p.healthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.stopHealthCheck:
			return
		case <-ticker.C:
			p.performHealthCheck()
		}
	}
}

// performHealthCheck checks and maintains connection health
func (p *ProtocolLevelConnectionPool) performHealthCheck() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	now := time.Now()
	connectionsToRemove := make([]int, 0)
	
	for i, conn := range p.connections {
		conn.mutex.Lock()
		
		// Check if connection is too old
		if now.Sub(conn.createdAt) > p.resourceManager.limits.ConnectionMaxAge {
			conn.healthy = false
			connectionsToRemove = append(connectionsToRemove, i)
			conn.mutex.Unlock()
			continue
		}
		
		// Check if connection is idle for too long
		if now.Sub(conn.lastUsed) > p.resourceManager.limits.ConnectionIdleTimeout {
			conn.healthy = false
			connectionsToRemove = append(connectionsToRemove, i)
			conn.mutex.Unlock()
			continue
		}
		
		// Check gRPC connection state
		state := conn.conn.GetState()
		if state == connectivity.TransientFailure || state == connectivity.Shutdown {
			conn.healthy = false
			connectionsToRemove = append(connectionsToRemove, i)
		}
		
		conn.mutex.Unlock()
	}
	
	// Remove unhealthy connections (in reverse order to maintain indices)
	for i := len(connectionsToRemove) - 1; i >= 0; i-- {
		idx := connectionsToRemove[i]
		conn := p.connections[idx]
		
		// Close the connection
		conn.conn.Close()
		p.resourceManager.TrackConnectionClosed(p.serviceName)
		
		// Remove from slice
		p.connections = append(p.connections[:idx], p.connections[idx+1:]...)
		p.currentSize--
		
		p.logger.Debug("Removed unhealthy connection",
			zap.String("service", p.serviceName),
			zap.Int("remaining_connections", p.currentSize))
	}
}