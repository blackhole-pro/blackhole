package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"go.uber.org/zap"
)

var (
	configFile = flag.String("config", "configs/identity.yaml", "Path to configuration file")
	socketPath = flag.String("socket", "/var/run/blackhole/identity.sock", "Unix socket path")
	tcpAddr    = flag.String("tcp", ":50051", "TCP address for remote connections")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting identity service",
		zap.String("config", *configFile),
		zap.String("socket", *socketPath),
		zap.String("tcp", *tcpAddr),
	)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// TODO: Register identity service implementation
	// identityv1.RegisterIdentityServiceServer(grpcServer, service)

	// Listen on Unix socket for local communication
	if err := os.RemoveAll(*socketPath); err != nil {
		logger.Fatal("Failed to remove existing socket", zap.Error(err))
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
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
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