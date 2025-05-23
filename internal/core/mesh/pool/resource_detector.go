package pool

import (
	"fmt"
	"runtime"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// SystemResources represents detected system resource limits
type SystemResources struct {
	// CPU cores available
	CPUCores int

	// Memory available in bytes
	MemoryBytes int64

	// File descriptor limit
	FileDescriptorLimit int64

	// Network connection capacity estimate
	NetworkCapacity int

	// Last detection time
	DetectedAt time.Time
}

// ResourceDetector detects system resources and calculates safe limits
type ResourceDetector struct {
	logger          *zap.Logger
	cachedResources *SystemResources
	cacheExpiry     time.Duration
}

// NewResourceDetector creates a new resource detector
func NewResourceDetector(logger *zap.Logger) *ResourceDetector {
	return &ResourceDetector{
		logger:      logger,
		cacheExpiry: 5 * time.Minute, // Cache results for 5 minutes
	}
}

// DetectSystemResources detects current system resource limits
func (rd *ResourceDetector) DetectSystemResources() (*SystemResources, error) {
	// Check cache first
	if rd.cachedResources != nil && time.Since(rd.cachedResources.DetectedAt) < rd.cacheExpiry {
		return rd.cachedResources, nil
	}

	// Detect CPU cores
	cpuCores := runtime.NumCPU()

	// Detect available memory
	memoryBytes, err := rd.detectMemory()
	if err != nil {
		rd.logger.Warn("Failed to detect memory, using default", zap.Error(err))
		memoryBytes = 1024 * 1024 * 1024 // 1GB default
	}

	// Detect file descriptor limit
	fdLimit, err := rd.detectFileDescriptorLimit()
	if err != nil {
		rd.logger.Warn("Failed to detect file descriptor limit, using default", zap.Error(err))
		fdLimit = 1024 // Default limit
	}

	// Calculate network capacity based on system resources
	networkCapacity := rd.calculateNetworkCapacity(cpuCores, memoryBytes, fdLimit)

	resources := &SystemResources{
		CPUCores:            cpuCores,
		MemoryBytes:         memoryBytes,
		FileDescriptorLimit: fdLimit,
		NetworkCapacity:     networkCapacity,
		DetectedAt:          time.Now(),
	}

	// Cache the results
	rd.cachedResources = resources

	rd.logger.Info("System resources detected",
		zap.Int("cpu_cores", cpuCores),
		zap.Int64("memory_mb", memoryBytes/(1024*1024)),
		zap.Int64("fd_limit", fdLimit),
		zap.Int("network_capacity", networkCapacity))

	return resources, nil
}

// CalculateResourceLimits calculates safe resource limits based on system capacity
func (rd *ResourceDetector) CalculateResourceLimits(utilizationPercent int) (*ResourceLimits, error) {
	if utilizationPercent <= 0 || utilizationPercent > 100 {
		return nil, fmt.Errorf("utilizationPercent must be between 1-100, got %d", utilizationPercent)
	}

	resources, err := rd.DetectSystemResources()
	if err != nil {
		return nil, fmt.Errorf("failed to detect system resources: %w", err)
	}

	utilizationFactor := float64(utilizationPercent) / 100.0

	// Calculate limits based on system capacity
	limits := &ResourceLimits{
		// Total connections based on file descriptor limit and network capacity
		MaxTotalConnections: int(float64(min(int(resources.FileDescriptorLimit/4), resources.NetworkCapacity)) * utilizationFactor),

		// Connections per service based on CPU cores (more cores = more concurrent services)
		MaxConnectionsPerService: max(1, int(float64(resources.CPUCores*2)*utilizationFactor)),

		// Concurrent requests based on memory and CPU
		MaxConcurrentRequests: int(float64(resources.CPUCores*10) * utilizationFactor),

		// Request rate based on system capacity
		MaxRequestsPerSecond: int(float64(resources.CPUCores*50) * utilizationFactor),

		// Connection timeouts
		ConnectionIdleTimeout: 5 * time.Minute,
		ConnectionMaxAge:      30 * time.Minute,

		// System resources for reference
		DetectedCPUCores:    resources.CPUCores,
		DetectedMemoryMB:    resources.MemoryBytes / (1024 * 1024),
		DetectedFDLimit:     resources.FileDescriptorLimit,
		UtilizationPercent:  utilizationPercent,
	}

	rd.logger.Info("Resource limits calculated",
		zap.Int("max_total_connections", limits.MaxTotalConnections),
		zap.Int("max_connections_per_service", limits.MaxConnectionsPerService),
		zap.Int("max_concurrent_requests", limits.MaxConcurrentRequests),
		zap.Int("max_requests_per_second", limits.MaxRequestsPerSecond),
		zap.Int("utilization_percent", utilizationPercent))

	return limits, nil
}

// detectMemory detects available system memory (simplified for cross-platform)
func (rd *ResourceDetector) detectMemory() (int64, error) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	// For cross-platform compatibility, use a reasonable default based on the system
	// In production, this could use platform-specific calls
	// For now, estimate based on runtime stats
	estimatedRAM := int64(8 * 1024 * 1024 * 1024) // 8GB default
	
	// If we can get heap size, use it as a baseline
	if stats.Sys > 0 {
		// Estimate total system memory as ~10x current heap usage
		estimatedRAM = int64(stats.Sys) * 10
		
		// Reasonable bounds
		if estimatedRAM < 1024*1024*1024 {
			estimatedRAM = 1024 * 1024 * 1024 // Minimum 1GB
		}
		if estimatedRAM > 64*1024*1024*1024 {
			estimatedRAM = 64 * 1024 * 1024 * 1024 // Maximum 64GB
		}
	}
	
	return estimatedRAM, nil
}

// detectFileDescriptorLimit detects the file descriptor limit
func (rd *ResourceDetector) detectFileDescriptorLimit() (int64, error) {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return 0, fmt.Errorf("failed to get file descriptor limit: %w", err)
	}

	// Use soft limit (current limit) rather than hard limit
	return int64(rLimit.Cur), nil
}

// calculateNetworkCapacity estimates network connection capacity
func (rd *ResourceDetector) calculateNetworkCapacity(cpuCores int, memoryBytes int64, fdLimit int64) int {
	// Base capacity on multiple factors
	memoryMB := memoryBytes / (1024 * 1024)

	// Estimate based on CPU cores (each core can handle ~500 concurrent connections)
	cpuBasedCapacity := cpuCores * 500

	// Estimate based on memory (assume ~1MB per 100 connections for buffers)
	memoryBasedCapacity := int(memoryMB / 10)

	// Estimate based on file descriptors (need 1 FD per connection, reserve some for files)
	fdBasedCapacity := int(fdLimit * 3 / 4) // Use 75% of FD limit for connections

	// Take the minimum as the bottleneck
	capacity := min(cpuBasedCapacity, min(memoryBasedCapacity, fdBasedCapacity))

	// Ensure reasonable bounds
	if capacity < 10 {
		capacity = 10 // Minimum viable capacity
	}
	if capacity > 10000 {
		capacity = 10000 // Reasonable upper bound for single node
	}

	return capacity
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}