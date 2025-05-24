// Package client provides a simple client library for creating mesh-connected plugins
package client

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"go.uber.org/zap"
)

// PluginClient helps plugins connect to the mesh network
type PluginClient struct {
	// Plugin metadata
	name        string
	version     string
	description string
	
	// Configuration
	config      Config
	
	// gRPC server
	grpcServer  *grpc.Server
	listener    net.Listener
	
	// Health checking
	healthServer *health.Server
	
	// Logging
	logger      *zap.Logger
	
	// Lifecycle
	started     bool
	stopCh      chan struct{}
	doneCh      chan struct{}
}

// Config configures the plugin client
type Config struct {
	// Socket path for the plugin to listen on
	SocketPath string
	
	// Plugin metadata
	Name        string
	Version     string
	Description string
	
	// Options
	EnableReflection bool          // Enable gRPC reflection for debugging
	EnableHealth     bool          // Enable health checking endpoint
	GracefulTimeout  time.Duration // Graceful shutdown timeout
	
	// Callbacks
	OnStart    func() error           // Called when plugin starts
	OnStop     func() error           // Called when plugin stops
	OnConnect  func(peer string)      // Called when a client connects
	
	// Logger
	Logger *zap.Logger
}

// DefaultConfig returns a default configuration
func DefaultConfig(name string) Config {
	return Config{
		Name:             name,
		Version:          "1.0.0",
		Description:      fmt.Sprintf("%s plugin", name),
		SocketPath:       fmt.Sprintf("/tmp/blackhole/plugins/%s.sock", name),
		EnableReflection: true,
		EnableHealth:     true,
		GracefulTimeout:  10 * time.Second,
		Logger:           zap.NewNop(),
	}
}

// New creates a new plugin client
func New(config Config) (*PluginClient, error) {
	// Validate config
	if config.Name == "" {
		return nil, fmt.Errorf("plugin name is required")
	}
	if config.SocketPath == "" {
		config.SocketPath = fmt.Sprintf("/tmp/blackhole/plugins/%s.sock", config.Name)
	}
	if config.GracefulTimeout == 0 {
		config.GracefulTimeout = 10 * time.Second
	}
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}

	client := &PluginClient{
		name:        config.Name,
		version:     config.Version,
		description: config.Description,
		config:      config,
		logger:      config.Logger.With(zap.String("plugin", config.Name)),
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}

	return client, nil
}

// NewFromEnv creates a plugin client from environment variables
func NewFromEnv() (*PluginClient, error) {
	config := Config{
		Name:        os.Getenv("PLUGIN_NAME"),
		Version:     os.Getenv("PLUGIN_VERSION"),
		SocketPath:  os.Getenv("PLUGIN_SOCKET"),
		Description: os.Getenv("PLUGIN_DESCRIPTION"),
	}

	if config.Name == "" {
		return nil, fmt.Errorf("PLUGIN_NAME environment variable is required")
	}

	// Set defaults
	if config.Version == "" {
		config.Version = "1.0.0"
	}
	if config.SocketPath == "" {
		config.SocketPath = fmt.Sprintf("/tmp/blackhole/plugins/%s.sock", config.Name)
	}

	return New(config)
}

// Start starts the plugin and begins listening for connections
func (c *PluginClient) Start(ctx context.Context) error {
	if c.started {
		return fmt.Errorf("plugin already started")
	}

	c.logger.Info("Starting plugin",
		zap.String("version", c.version),
		zap.String("socket", c.config.SocketPath))

	// Create socket directory
	socketDir := filepath.Dir(c.config.SocketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket
	os.Remove(c.config.SocketPath)

	// Create listener
	listener, err := net.Listen("unix", c.config.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}
	c.listener = listener

	// Create gRPC server
	c.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(c.unaryInterceptor),
		grpc.StreamInterceptor(c.streamInterceptor),
	)

	// Enable reflection for debugging
	if c.config.EnableReflection {
		reflection.Register(c.grpcServer)
		c.logger.Debug("gRPC reflection enabled")
	}

	// Enable health checking
	if c.config.EnableHealth {
		c.healthServer = health.NewServer()
		grpc_health_v1.RegisterHealthServer(c.grpcServer, c.healthServer)
		c.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
		c.logger.Debug("Health checking enabled")
	}

	// Call OnStart callback
	if c.config.OnStart != nil {
		if err := c.config.OnStart(); err != nil {
			listener.Close()
			return fmt.Errorf("OnStart callback failed: %w", err)
		}
	}

	// Start serving in background
	go c.serve()

	c.started = true
	c.logger.Info("Plugin started successfully")

	return nil
}

// Stop gracefully stops the plugin
func (c *PluginClient) Stop(ctx context.Context) error {
	if !c.started {
		return nil
	}

	c.logger.Info("Stopping plugin")

	// Signal shutdown
	close(c.stopCh)

	// Set health status to not serving
	if c.healthServer != nil {
		c.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}

	// Graceful shutdown of gRPC server
	done := make(chan struct{})
	go func() {
		c.grpcServer.GracefulStop()
		close(done)
	}()

	// Wait for graceful shutdown or timeout
	select {
	case <-done:
		c.logger.Debug("gRPC server stopped gracefully")
	case <-time.After(c.config.GracefulTimeout):
		c.logger.Warn("Graceful shutdown timeout, forcing stop")
		c.grpcServer.Stop()
	}

	// Close listener
	if c.listener != nil {
		c.listener.Close()
	}

	// Call OnStop callback
	if c.config.OnStop != nil {
		if err := c.config.OnStop(); err != nil {
			c.logger.Error("OnStop callback failed", zap.Error(err))
		}
	}

	// Wait for serve goroutine to finish
	<-c.doneCh

	// Clean up socket
	os.Remove(c.config.SocketPath)

	c.started = false
	c.logger.Info("Plugin stopped")

	return nil
}

// GetGRPCServer returns the gRPC server for registering services
func (c *PluginClient) GetGRPCServer() *grpc.Server {
	return c.grpcServer
}

// SetHealthStatus sets the health status of the plugin
func (c *PluginClient) SetHealthStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	if c.healthServer != nil {
		c.healthServer.SetServingStatus(service, status)
	}
}

// WaitForShutdown blocks until the plugin receives a shutdown signal
func (c *PluginClient) WaitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		c.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-c.stopCh:
		c.logger.Debug("Shutdown initiated internally")
	}
}

// Run is a convenience method that starts the plugin and waits for shutdown
func (c *PluginClient) Run(ctx context.Context) error {
	// Start the plugin
	if err := c.Start(ctx); err != nil {
		return err
	}

	// Wait for shutdown signal
	c.WaitForShutdown()

	// Stop the plugin
	return c.Stop(ctx)
}

// Private methods

func (c *PluginClient) serve() {
	defer close(c.doneCh)

	c.logger.Debug("Starting gRPC server", zap.String("socket", c.config.SocketPath))

	if err := c.grpcServer.Serve(c.listener); err != nil {
		select {
		case <-c.stopCh:
			// Expected shutdown
			c.logger.Debug("gRPC server stopped")
		default:
			// Unexpected error
			c.logger.Error("gRPC server error", zap.Error(err))
		}
	}
}

// Interceptors for logging and monitoring

func (c *PluginClient) unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	
	// Get peer info
	peer := "unknown"
	if p, ok := grpc.Peer(ctx); ok {
		peer = p.Addr.String()
	}

	// Call OnConnect for new connections
	if c.config.OnConnect != nil {
		c.config.OnConnect(peer)
	}

	// Log request
	c.logger.Debug("Handling request",
		zap.String("method", info.FullMethod),
		zap.String("peer", peer))

	// Handle the request
	resp, err := handler(ctx, req)

	// Log response
	duration := time.Since(start)
	if err != nil {
		c.logger.Error("Request failed",
			zap.String("method", info.FullMethod),
			zap.String("peer", peer),
			zap.Duration("duration", duration),
			zap.Error(err))
	} else {
		c.logger.Debug("Request completed",
			zap.String("method", info.FullMethod),
			zap.String("peer", peer),
			zap.Duration("duration", duration))
	}

	return resp, err
}

func (c *PluginClient) streamInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()
	
	// Get peer info
	peer := "unknown"
	if p, ok := grpc.Peer(ss.Context()); ok {
		peer = p.Addr.String()
	}

	// Log stream start
	c.logger.Debug("Starting stream",
		zap.String("method", info.FullMethod),
		zap.String("peer", peer),
		zap.Bool("server_stream", info.IsServerStream),
		zap.Bool("client_stream", info.IsClientStream))

	// Handle the stream
	err := handler(srv, ss)

	// Log stream end
	duration := time.Since(start)
	if err != nil {
		c.logger.Error("Stream failed",
			zap.String("method", info.FullMethod),
			zap.String("peer", peer),
			zap.Duration("duration", duration),
			zap.Error(err))
	} else {
		c.logger.Debug("Stream completed",
			zap.String("method", info.FullMethod),
			zap.String("peer", peer),
			zap.Duration("duration", duration))
	}

	return err
}