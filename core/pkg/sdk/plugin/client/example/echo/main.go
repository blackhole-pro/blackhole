// Example echo plugin using the mesh client library
package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"go.uber.org/zap"

	"github.com/blackhole-pro/blackhole/core/pkg/sdk/plugin/client"
	echov1 "github.com/blackhole-pro/blackhole/core/pkg/sdk/plugin/client/example/echo/proto/v1"
)

// EchoPlugin implements a simple echo service
type EchoPlugin struct {
	echov1.UnimplementedEchoServiceServer
	logger *zap.Logger
}

// Echo implements the Echo RPC method
func (p *EchoPlugin) Echo(ctx context.Context, req *echov1.EchoRequest) (*echov1.EchoResponse, error) {
	p.logger.Info("Echo request received", zap.String("message", req.Message))
	
	return &echov1.EchoResponse{
		Message: fmt.Sprintf("Echo: %s", req.Message),
	}, nil
}

// Initialize implements plugin initialization
func (p *EchoPlugin) Initialize(ctx context.Context, req *echov1.InitializeRequest) (*echov1.InitializeResponse, error) {
	p.logger.Info("Plugin initializing", zap.Any("config", req.Config))
	
	return &echov1.InitializeResponse{
		Success: true,
		Message: "Echo plugin initialized",
	}, nil
}

// GetInfo returns plugin information
func (p *EchoPlugin) GetInfo(ctx context.Context, req *echov1.GetInfoRequest) (*echov1.GetInfoResponse, error) {
	return &echov1.GetInfoResponse{
		Name:        "echo",
		Version:     "1.0.0",
		Description: "Simple echo plugin demonstrating mesh connectivity",
		Author:      "Blackhole Team",
		Capabilities: []string{"echo", "example"},
	}, nil
}

func main() {
	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	// Create plugin configuration
	config := client.DefaultConfig("echo")
	config.Logger = logger
	config.Description = "Echo plugin example"
	
	// Add lifecycle callbacks
	config.OnStart = func() error {
		logger.Info("Echo plugin starting up")
		return nil
	}
	
	config.OnStop = func() error {
		logger.Info("Echo plugin shutting down")
		return nil
	}
	
	config.OnConnect = func(peer string) {
		logger.Info("Client connected", zap.String("peer", peer))
	}

	// Create plugin client
	pluginClient, err := client.New(config)
	if err != nil {
		logger.Fatal("Failed to create plugin client", zap.Error(err))
	}

	// Create echo service
	echoService := &EchoPlugin{logger: logger}

	// Register the service with gRPC
	echov1.RegisterEchoServiceServer(pluginClient.GetGRPCServer(), echoService)

	// Run the plugin (starts and waits for shutdown)
	if err := pluginClient.Run(context.Background()); err != nil {
		logger.Fatal("Plugin failed", zap.Error(err))
	}
}

// This plugin can be used in three ways:
//
// 1. Direct execution (for testing):
//    $ go run main.go
//
// 2. Started by plugin manager:
//    The plugin manager sets environment variables and starts this process
//
// 3. Connected via mesh:
//    Other services connect using:
//    conn, _ := grpc.Dial("unix:///tmp/blackhole/plugins/echo.sock", grpc.WithInsecure())
//    client := echov1.NewEchoServiceClient(conn)
//    resp, _ := client.Echo(ctx, &echov1.EchoRequest{Message: "Hello"})