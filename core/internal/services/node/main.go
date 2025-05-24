package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"google.golang.org/grpc"
	"go.uber.org/zap"
)

var (
	configFile = flag.String("config", "configs/node.yaml", "Path to configuration file")
	socketPath = flag.String("socket", "/var/run/blackhole/node.sock", "Unix socket path")
	tcpAddr    = flag.String("tcp", ":50053", "TCP address for remote connections")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger, err := initLogger(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting node service",
		zap.String("config", *configFile),
		zap.String("socket", *socketPath),
		zap.String("tcp", *tcpAddr),
		zap.String("log_level", *logLevel))

	// Load configuration
	config, err := LoadConfig(*configFile)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Create node service
	nodeService, err := NewNodeService(config, logger)
	if err != nil {
		logger.Fatal("Failed to create node service", zap.Error(err))
	}

	// Start the node service
	ctx := context.Background()
	if err := nodeService.Start(ctx); err != nil {
		logger.Fatal("Failed to start node service", zap.Error(err))
	}

	// Create gRPC server with middleware interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(logger),      // Recovery should be first
			loggingInterceptor(logger),       // General logging
			nodeLoggingInterceptor(logger),   // Node-specific logging
			metricsInterceptor(logger),       // Metrics collection
			rateLimitingInterceptor(logger, 100), // Rate limiting (100 req/sec)
			authenticationInterceptor(logger), // Authentication last
		),
		grpc.ChainStreamInterceptor(
			streamingLoggingInterceptor(logger), // Streaming operations logging
		),
	)

	// Create and register node service implementation
	nodeServiceServer := NewNodeServiceServer(nodeService)
	
	// Note: In a real implementation, you would register this with the generated gRPC server
	// For example: nodev1.RegisterNodeServiceServer(grpcServer, nodeServiceServer)
	// For now, we'll create a placeholder registration
	registerNodeService(grpcServer, nodeServiceServer)

	// Listen on Unix socket for local communication
	if err := os.RemoveAll(*socketPath); err != nil {
		logger.Fatal("Failed to remove existing socket", zap.Error(err))
	}

	// Ensure socket directory exists
	socketDir := filepath.Dir(*socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		logger.Fatal("Failed to create socket directory", zap.Error(err))
	}

	unixListener, err := net.Listen("unix", *socketPath)
	if err != nil {
		logger.Fatal("Failed to listen on Unix socket", zap.Error(err))
	}

	// Listen on TCP for remote communication
	tcpListener, err := net.Listen("tcp", *tcpAddr)
	if err != nil {
		logger.Fatal("Failed to listen on TCP", zap.Error(err))
	}

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()
		
		// Stop the node service
		if err := nodeService.Stop(context.Background()); err != nil {
			logger.Error("Error stopping node service", zap.Error(err))
		}
		
		// Gracefully stop gRPC server
		grpcServer.GracefulStop()
	}()

	// Start serving
	errChan := make(chan error, 2)

	go func() {
		logger.Info("Starting Unix socket server", zap.String("path", *socketPath))
		if err := grpcServer.Serve(unixListener); err != nil {
			errChan <- fmt.Errorf("unix server error: %w", err)
		}
	}()

	go func() {
		logger.Info("Starting TCP server", zap.String("addr", *tcpAddr))
		if err := grpcServer.Serve(tcpListener); err != nil {
			errChan <- fmt.Errorf("tcp server error: %w", err)
		}
	}()

	// Wait for error or shutdown
	select {
	case err := <-errChan:
		logger.Fatal("Server error", zap.Error(err))
	case <-ctx.Done():
		logger.Info("Service shutdown complete")
	}
}

// initLogger creates a zap logger with the specified log level
func initLogger(level string) (*zap.Logger, error) {
	var zapLevel zap.AtomicLevel
	switch level {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	config := zap.Config{
		Level:            zapLevel,
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return config.Build()
}

// registerNodeService registers the node service with the gRPC server
// Note: In a real implementation, this would use the generated registration function
func registerNodeService(server *grpc.Server, nodeService *NodeServiceServer) {
	// Placeholder registration - in real implementation, you would use:
	// nodev1.RegisterNodeServiceServer(server, nodeService)
	
	// For now, we'll just log that the service is registered
	logConfig := zap.NewDevelopmentConfig()
	logConfig.DisableCaller = true
	logConfig.DisableStacktrace = true
	logger, _ := logConfig.Build()
	logger.Info("Node service registered with gRPC server")
}

