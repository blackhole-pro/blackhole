package client

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"go.uber.org/zap/zaptest"
)

func TestPluginClient(t *testing.T) {
	// Create temporary directory for sockets
	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, "test-plugin.sock")

	// Create config
	config := Config{
		Name:             "test-plugin",
		Version:          "1.0.0",
		Description:      "Test plugin",
		SocketPath:       socketPath,
		EnableReflection: true,
		EnableHealth:     true,
		GracefulTimeout:  2 * time.Second,
		Logger:           zaptest.NewLogger(t),
	}

	// Track lifecycle events
	var started, stopped, connected bool
	config.OnStart = func() error {
		started = true
		return nil
	}
	config.OnStop = func() error {
		stopped = true
		return nil
	}
	config.OnConnect = func(peer string) {
		connected = true
	}

	// Create plugin client
	client, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create plugin client: %v", err)
	}

	// Start the plugin
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Failed to start plugin: %v", err)
	}

	// Verify socket exists
	if _, err := os.Stat(socketPath); err != nil {
		t.Errorf("Socket file not created: %v", err)
	}

	// Verify lifecycle callback
	if !started {
		t.Error("OnStart callback not called")
	}

	// Connect to the plugin
	conn, err := grpc.Dial(
		fmt.Sprintf("unix://%s", socketPath),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(2*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to connect to plugin: %v", err)
	}
	defer conn.Close()

	// Give time for connection callback
	time.Sleep(100 * time.Millisecond)
	if !connected {
		t.Error("OnConnect callback not called")
	}

	// Test health check
	healthClient := grpc_health_v1.NewHealthClient(conn)
	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("Expected SERVING status, got %v", resp.Status)
	}

	// Stop the plugin
	if err := client.Stop(ctx); err != nil {
		t.Fatalf("Failed to stop plugin: %v", err)
	}

	// Verify lifecycle callback
	if !stopped {
		t.Error("OnStop callback not called")
	}

	// Verify socket cleaned up
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Socket file not cleaned up")
	}
}

func TestPluginClientFromEnv(t *testing.T) {
	// Set environment variables
	tempDir := t.TempDir()
	os.Setenv("PLUGIN_NAME", "env-test-plugin")
	os.Setenv("PLUGIN_VERSION", "2.0.0")
	os.Setenv("PLUGIN_SOCKET", filepath.Join(tempDir, "env-test.sock"))
	os.Setenv("PLUGIN_DESCRIPTION", "Environment test plugin")
	defer func() {
		os.Unsetenv("PLUGIN_NAME")
		os.Unsetenv("PLUGIN_VERSION")
		os.Unsetenv("PLUGIN_SOCKET")
		os.Unsetenv("PLUGIN_DESCRIPTION")
	}()

	// Create plugin from environment
	client, err := NewFromEnv()
	if err != nil {
		t.Fatalf("Failed to create plugin from env: %v", err)
	}

	// Verify configuration
	if client.name != "env-test-plugin" {
		t.Errorf("Expected name 'env-test-plugin', got '%s'", client.name)
	}
	if client.version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", client.version)
	}
}

func TestHealthStatus(t *testing.T) {
	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, "health-test.sock")

	config := DefaultConfig("health-test")
	config.SocketPath = socketPath
	config.Logger = zaptest.NewLogger(t)

	client, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create plugin client: %v", err)
	}

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Failed to start plugin: %v", err)
	}
	defer client.Stop(ctx)

	// Connect to the plugin
	conn, err := grpc.Dial(
		fmt.Sprintf("unix://%s", socketPath),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(2*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to connect to plugin: %v", err)
	}
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)

	// Test default status
	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("Expected SERVING status, got %v", resp.Status)
	}

	// Change health status
	client.SetHealthStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Verify status changed
	resp, err = healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		t.Errorf("Expected NOT_SERVING status, got %v", resp.Status)
	}

	// Test service-specific health
	client.SetHealthStatus("my-service", grpc_health_v1.HealthCheckResponse_SERVING)
	
	resp, err = healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "my-service",
	})
	if err != nil {
		t.Fatalf("Service health check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("Expected SERVING status for my-service, got %v", resp.Status)
	}
}

func TestGracefulShutdown(t *testing.T) {
	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, "shutdown-test.sock")

	config := DefaultConfig("shutdown-test")
	config.SocketPath = socketPath
	config.GracefulTimeout = 1 * time.Second
	config.Logger = zaptest.NewLogger(t)

	client, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create plugin client: %v", err)
	}

	// Add a slow handler to test graceful shutdown
	slowService := &slowTestService{delay: 2 * time.Second}
	RegisterSlowTestServiceServer(client.GetGRPCServer(), slowService)

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Failed to start plugin: %v", err)
	}

	// Connect and start a slow request
	conn, err := grpc.Dial(
		fmt.Sprintf("unix://%s", socketPath),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(1*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to connect to plugin: %v", err)
	}
	defer conn.Close()

	testClient := NewSlowTestServiceClient(conn)
	
	// Start slow request in background
	errCh := make(chan error, 1)
	go func() {
		_, err := testClient.SlowMethod(context.Background(), &SlowRequest{})
		errCh <- err
	}()

	// Give request time to start
	time.Sleep(100 * time.Millisecond)

	// Stop the plugin (should wait for graceful timeout)
	stopStart := time.Now()
	if err := client.Stop(ctx); err != nil {
		t.Fatalf("Failed to stop plugin: %v", err)
	}
	stopDuration := time.Since(stopStart)

	// Should have waited for graceful timeout
	if stopDuration < config.GracefulTimeout {
		t.Errorf("Stop returned too quickly: %v < %v", stopDuration, config.GracefulTimeout)
	}

	// The slow request should have been interrupted
	select {
	case err := <-errCh:
		if err == nil {
			t.Error("Expected error from interrupted request")
		}
	case <-time.After(3 * time.Second):
		t.Error("Request did not complete")
	}
}

// Test service definitions for graceful shutdown test

type SlowRequest struct{}
type SlowResponse struct{}

type slowTestService struct {
	delay time.Duration
}

func (s *slowTestService) SlowMethod(ctx context.Context, req *SlowRequest) (*SlowResponse, error) {
	select {
	case <-time.After(s.delay):
		return &SlowResponse{}, nil
	case <-ctx.Done():
		return nil, status.Error(codes.Canceled, "request canceled")
	}
}

// Mock gRPC registration (in real code, this would be generated from proto)
type SlowTestServiceServer interface {
	SlowMethod(context.Context, *SlowRequest) (*SlowResponse, error)
}

type SlowTestServiceClient interface {
	SlowMethod(ctx context.Context, in *SlowRequest, opts ...grpc.CallOption) (*SlowResponse, error)
}

func RegisterSlowTestServiceServer(s *grpc.Server, srv SlowTestServiceServer) {
	// Mock registration
}

func NewSlowTestServiceClient(cc *grpc.ClientConn) SlowTestServiceClient {
	// Mock client
	return nil
}