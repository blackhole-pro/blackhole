package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"

	"github.com/blackhole-pro/blackhole/core/internal/framework/mesh"
)

func main() {
	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()

	// Create protocol router with intelligent resource management
	router := mesh.NewProtocolRouter(logger)
	defer router.Close()

	// Register a local service (Unix socket)
	localEndpoint := mesh.Endpoint{
		Socket:      "sockets/identity.sock",
		IsLocal:     true,
		LastUpdated: time.Now(),
		Metadata: map[string]string{
			"version": "1.0.0",
			"type":    "identity",
		},
	}

	err = router.RegisterService("identity", localEndpoint)
	if err != nil {
		logger.Fatal("Failed to register local service", zap.Error(err))
	}

	// Register a remote service (TCP)
	remoteEndpoint := mesh.Endpoint{
		Address:     "api.example.com:443",
		IsLocal:     false,
		LastUpdated: time.Now(),
		Metadata: map[string]string{
			"version": "2.1.0",
			"type":    "external-api",
		},
	}

	err = router.RegisterService("external-api", remoteEndpoint)
	if err != nil {
		logger.Fatal("Failed to register remote service", zap.Error(err))
	}

	// Show resource usage
	usage := router.GetResourceUsage()
	logger.Info("Initial resource usage",
		zap.Int("total_connections", usage.TotalConnections),
		zap.Int("max_connections", usage.MaxTotalConnections),
		zap.Float64("connection_utilization", usage.ConnectionUtilization))

	// List registered services
	services := router.ListServices()
	logger.Info("Registered services",
		zap.Int("service_count", len(services)))

	for serviceName, endpoints := range services {
		logger.Info("Service details",
			zap.String("service", serviceName),
			zap.Int("endpoints", len(endpoints)),
			zap.Bool("is_local", endpoints[0].IsLocal))
	}

	// Demonstrate service discovery
	identityEndpoint, err := router.DiscoverService("identity")
	if err != nil {
		logger.Error("Failed to discover identity service", zap.Error(err))
	} else {
		logger.Info("Discovered identity service",
			zap.String("socket", identityEndpoint.Socket),
			zap.Bool("is_local", identityEndpoint.IsLocal))
	}

	// Show connection pool statistics
	poolStats := router.GetPoolStats()
	for serviceName, stats := range poolStats {
		logger.Info("Pool statistics",
			zap.String("service", serviceName),
			zap.Int("total_connections", stats.TotalConnections),
			zap.Int("healthy_connections", stats.HealthyConnections),
			zap.Int("max_connections", stats.MaxConnections),
			zap.Float64("success_rate", stats.SuccessRate))
	}

	// Demonstrate runtime resource limit adjustment
	logger.Info("Updating resource limits to 60% utilization")
	err = router.UpdateResourceLimits(60)
	if err != nil {
		logger.Error("Failed to update resource limits", zap.Error(err))
	}

	// Show updated resource usage
	updatedUsage := router.GetResourceUsage()
	logger.Info("Updated resource usage",
		zap.Int("max_connections", updatedUsage.MaxTotalConnections),
		zap.Int("max_requests", updatedUsage.MaxConcurrentRequests))

	// Example of protocol-level request routing (would fail without actual service)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This would route a gRPC call: /identity.auth.v1.AuthService/GenerateChallenge
	requestData := []byte(`{"did": "did:example:123", "purpose": "authentication"}`)
	_, err = router.RouteRequest(ctx, "identity", "/identity.auth.v1.AuthService/GenerateChallenge", requestData)
	if err != nil {
		// Expected to fail since no actual service is running
		logger.Info("Protocol routing example (expected to fail)", zap.Error(err))
	}

	logger.Info("Protocol router example completed successfully")
}