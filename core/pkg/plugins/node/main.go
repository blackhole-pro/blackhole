// Package main implements the node plugin entry point
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"node/plugin"
	"node/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// rpcPlugin wraps the plugin for RPC communication
type rpcPlugin struct {
	plugin *plugin.Plugin
	ctx    context.Context
	cancel context.CancelFunc
}

// RPC method implementations

// Info returns plugin information
func (r *rpcPlugin) Info(args *struct{}, reply *types.PluginInfo) error {
	info := r.plugin.Info()
	*reply = info
	return nil
}

// Start starts the plugin
func (r *rpcPlugin) Start(args *struct{}, reply *bool) error {
	err := r.plugin.Start(r.ctx)
	*reply = err == nil
	return err
}

// Stop stops the plugin
func (r *rpcPlugin) Stop(args *struct{}, reply *bool) error {
	err := r.plugin.Stop(r.ctx)
	*reply = err == nil
	return err
}

// Handle processes a request
func (r *rpcPlugin) Handle(args *types.PluginRequest, reply *types.PluginResponse) error {
	resp, err := r.plugin.Handle(r.ctx, *args)
	*reply = resp
	return err
}

// HealthCheck performs a health check
func (r *rpcPlugin) HealthCheck(args *struct{}, reply *bool) error {
	err := r.plugin.HealthCheck()
	*reply = err == nil
	return err
}

// GetStatus returns the plugin status
func (r *rpcPlugin) GetStatus(args *struct{}, reply *types.PluginStatus) error {
	status := r.plugin.GetStatus()
	*reply = status
	return nil
}

// ExportState exports the plugin state
func (r *rpcPlugin) ExportState(args *struct{}, reply *[]byte) error {
	state, err := r.plugin.ExportState()
	if err != nil {
		return err
	}
	*reply = state
	return nil
}

// ImportState imports plugin state
func (r *rpcPlugin) ImportState(args *[]byte, reply *bool) error {
	err := r.plugin.ImportState(*args)
	*reply = err == nil
	return err
}

func main() {
	// Initialize logger
	logger := initLogger()
	defer logger.Sync()

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Create plugin instance
	plugin, err := plugin.NewPlugin(config, logger)
	if err != nil {
		logger.Fatal("Failed to create plugin", zap.Error(err))
	}

	// Run plugin with RPC
	if err := runPlugin(plugin, logger); err != nil {
		logger.Fatal("Plugin failed", zap.Error(err))
	}
}

func runPlugin(p *plugin.Plugin, logger *zap.Logger) error {
	// Create context for plugin lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Create RPC wrapper
	wrapper := &rpcPlugin{
		plugin: p,
		ctx:    ctx,
		cancel: cancel,
	}

	// Register RPC service
	if err := rpc.Register(wrapper); err != nil {
		return fmt.Errorf("failed to register RPC service: %w", err)
	}

	// Get socket path from environment
	socketPath := os.Getenv("PLUGIN_SOCKET")
	if socketPath == "" {
		return fmt.Errorf("PLUGIN_SOCKET environment variable not set")
	}

	// Remove existing socket
	os.Remove(socketPath)

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket %s: %w", socketPath, err)
	}
	defer listener.Close()

	logger.Info("Plugin started",
		zap.String("socket", socketPath),
		zap.String("version", p.Info().Version))

	// Start accepting connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					logger.Error("Failed to accept connection", zap.Error(err))
					continue
				}
			}
			go rpc.ServeConn(conn)
		}
	}()

	// Wait for shutdown signal
	select {
	case sig := <-sigCh:
		logger.Info("Received signal", zap.String("signal", sig.String()))
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	// Prepare for shutdown
	if err := p.PrepareShutdown(); err != nil {
		logger.Error("Error preparing for shutdown", zap.Error(err))
	}

	// Stop the plugin
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()

	if err := p.Stop(stopCtx); err != nil {
		logger.Error("Error stopping plugin", zap.Error(err))
	}

	return nil
}

func initLogger() *zap.Logger {
	// Check log level from environment
	logLevel := zapcore.InfoLevel
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		switch strings.ToLower(level) {
		case "debug":
			logLevel = zapcore.DebugLevel
		case "info":
			logLevel = zapcore.InfoLevel
		case "warn":
			logLevel = zapcore.WarnLevel
		case "error":
			logLevel = zapcore.ErrorLevel
		}
	}

	// Create logger configuration
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(logLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// Customize encoder config
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.StacktraceKey = ""

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger.With(zap.String("plugin", "node"))
}

func loadConfig() (*types.NodeConfig, error) {
	// Default configuration
	config := &types.NodeConfig{
		NodeID:              os.Getenv("NODE_ID"),
		Version:             "1.0.0",
		P2PPort:             4001,
		ListenAddresses:     []string{"/ip4/0.0.0.0/tcp/4001"},
		EnableDiscovery:     true,
		DiscoveryMethod:     "bootstrap",
		DiscoveryInterval:   30 * time.Second,
		HealthCheckInterval: 10 * time.Second,
		PeerTimeout:         60 * time.Second,
		MaxPeers:            50,
		MaxBandwidthMbps:    100,
		ConnectionTimeout:   30 * time.Second,
		EnableEncryption:    true,
	}

	// Try to load config from file
	configPath := os.Getenv("PLUGIN_CONFIG_PATH")
	if configPath == "" {
		configPath = "/etc/blackhole/plugins/node.json"
	}

	if configData, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(configData, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables
	if nodeID := os.Getenv("NODE_ID"); nodeID != "" {
		config.NodeID = nodeID
	}

	if port := os.Getenv("P2P_PORT"); port != "" {
		// Parse port from string
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			config.P2PPort = p
		}
	}

	if bootstrapPeers := os.Getenv("BOOTSTRAP_PEERS"); bootstrapPeers != "" {
		config.BootstrapPeers = strings.Split(bootstrapPeers, ",")
	}

	// Validate configuration
	if err := plugin.ValidateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}