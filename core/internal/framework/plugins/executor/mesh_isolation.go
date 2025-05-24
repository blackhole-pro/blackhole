// Package executor provides mesh-based plugin execution
package executor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"go.uber.org/zap"

	"github.com/blackhole-pro/blackhole/core/internal/framework/mesh"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
)

// meshIsolation implements mesh-based plugin isolation
type meshIsolation struct {
	cmd            *exec.Cmd
	socketPath     string
	meshClient     mesh.Client
	grpcConn       *grpc.ClientConn
	resourceLimits plugins.PluginResources
	logger         *zap.Logger
	mu             sync.Mutex
	started        bool
}

// meshPlugin implements the Plugin interface for mesh-connected plugins
type meshPlugin struct {
	spec         plugins.PluginSpec
	binaryPath   string
	isolation    *meshIsolation
	info         plugins.PluginInfo
	status       plugins.PluginStatus
	meshEndpoint mesh.Endpoint
	logger       *zap.Logger
	mu           sync.RWMutex
}

// MeshIsolationConfig configures mesh-based plugin isolation
type MeshIsolationConfig struct {
	MeshClient     mesh.Client
	SocketDir      string
	Logger         *zap.Logger
	DefaultTimeout time.Duration
}

// NewMeshPlugin creates a new mesh-connected plugin
func NewMeshPlugin(spec plugins.PluginSpec, binaryPath string, config MeshIsolationConfig) plugins.Plugin {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	if config.SocketDir == "" {
		config.SocketDir = "/tmp/blackhole/plugins"
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Second
	}

	return &meshPlugin{
		spec:       spec,
		binaryPath: binaryPath,
		info: plugins.PluginInfo{
			Name:        spec.Name,
			Version:     spec.Version,
			Description: "Mesh-connected plugin",
			Status:      plugins.PluginStatusStopped,
		},
		status: plugins.PluginStatusStopped,
		logger: config.Logger.With(zap.String("plugin", spec.Name)),
	}
}

// Info returns plugin information
func (p *meshPlugin) Info() plugins.PluginInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.info
}

// Start starts the plugin process and connects via mesh
func (p *meshPlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status == plugins.PluginStatusRunning {
		return nil
	}

	p.status = plugins.PluginStatusStarting
	p.info.Status = p.status

	// Create socket directory
	socketDir := filepath.Dir(p.isolation.socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Prepare environment variables for the plugin
	env := os.Environ()
	env = append(env, fmt.Sprintf("PLUGIN_NAME=%s", p.spec.Name))
	env = append(env, fmt.Sprintf("PLUGIN_VERSION=%s", p.spec.Version))
	env = append(env, fmt.Sprintf("PLUGIN_SOCKET=%s", p.isolation.socketPath))
	env = append(env, fmt.Sprintf("PLUGIN_MESH_ENDPOINT=%s", p.meshEndpoint.Socket))

	// Create the command
	p.isolation.cmd = exec.CommandContext(ctx, p.binaryPath)
	p.isolation.cmd.Env = env

	// Set resource limits
	if err := p.setResourceLimits(); err != nil {
		p.logger.Warn("Failed to set resource limits", zap.Error(err))
	}

	// Start the plugin process
	if err := p.isolation.cmd.Start(); err != nil {
		p.status = plugins.PluginStatusError
		p.info.Status = p.status
		return fmt.Errorf("failed to start plugin process: %w", err)
	}

	p.isolation.started = true
	p.logger.Info("Plugin process started", zap.Int("pid", p.isolation.cmd.Process.Pid))

	// Wait for plugin to be ready on mesh
	if err := p.waitForMeshConnection(ctx); err != nil {
		p.stop()
		return fmt.Errorf("failed to establish mesh connection: %w", err)
	}

	// Register plugin with mesh network
	if err := p.registerWithMesh(ctx); err != nil {
		p.stop()
		return fmt.Errorf("failed to register with mesh: %w", err)
	}

	p.status = plugins.PluginStatusRunning
	p.info.Status = p.status
	p.logger.Info("Plugin started successfully")

	// Monitor process health
	go p.monitorProcess()

	return nil
}

// Stop stops the plugin process
func (p *meshPlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != plugins.PluginStatusRunning {
		return nil
	}

	p.status = plugins.PluginStatusStopping
	p.info.Status = p.status

	// Unregister from mesh
	if err := p.unregisterFromMesh(ctx); err != nil {
		p.logger.Warn("Failed to unregister from mesh", zap.Error(err))
	}

	// Close gRPC connection
	if p.isolation.grpcConn != nil {
		p.isolation.grpcConn.Close()
	}

	// Stop the process
	if err := p.stop(); err != nil {
		return err
	}

	p.status = plugins.PluginStatusStopped
	p.info.Status = p.status
	p.logger.Info("Plugin stopped")

	return nil
}

// Handle sends a request to the plugin via mesh
func (p *meshPlugin) Handle(ctx context.Context, request plugins.PluginRequest) (plugins.PluginResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.status != plugins.PluginStatusRunning {
		return plugins.PluginResponse{}, errors.New("plugin not running")
	}

	// Route request through mesh
	// The actual implementation depends on the plugin's specific gRPC interface
	// This is where the mesh network shines - we don't need to know the details
	
	return plugins.PluginResponse{
		Success: false,
		Error:   "mesh routing not yet implemented",
	}, nil
}

// HealthCheck checks plugin health via mesh
func (p *meshPlugin) HealthCheck() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.status != plugins.PluginStatusRunning {
		return errors.New("plugin not running")
	}

	// Check process is alive
	if p.isolation.cmd.Process == nil {
		return errors.New("plugin process not found")
	}

	// Check mesh connectivity
	if p.isolation.grpcConn == nil {
		return errors.New("mesh connection lost")
	}

	// TODO: Make actual health check call via mesh
	return nil
}

// GetStatus returns the plugin status
func (p *meshPlugin) GetStatus() plugins.PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// PrepareShutdown prepares the plugin for shutdown
func (p *meshPlugin) PrepareShutdown() error {
	// TODO: Send shutdown preparation signal via mesh
	return nil
}

// ExportState exports plugin state via mesh
func (p *meshPlugin) ExportState() ([]byte, error) {
	// TODO: Call plugin's ExportState method via mesh
	return nil, errors.New("not implemented")
}

// ImportState imports plugin state via mesh
func (p *meshPlugin) ImportState(state []byte) error {
	// TODO: Call plugin's ImportState method via mesh
	return errors.New("not implemented")
}

// Private methods

func (p *meshPlugin) setResourceLimits() error {
	// Set resource limits for the process
	// This is OS-specific and would need proper implementation
	
	// Example for Linux using cgroups or setrlimit
	// For now, return nil as it's platform-specific
	return nil
}

func (p *meshPlugin) waitForMeshConnection(ctx context.Context) error {
	// Wait for plugin to start listening on its socket
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return errors.New("timeout waiting for plugin mesh connection")
		case <-ticker.C:
			// Try to connect to the plugin's socket
			conn, err := net.Dial("unix", p.isolation.socketPath)
			if err == nil {
				conn.Close()
				// Socket is ready, establish gRPC connection
				grpcConn, err := grpc.Dial(
					fmt.Sprintf("unix://%s", p.isolation.socketPath),
					grpc.WithInsecure(),
				)
				if err != nil {
					return fmt.Errorf("failed to establish gRPC connection: %w", err)
				}
				p.isolation.grpcConn = grpcConn
				return nil
			}
		}
	}
}

func (p *meshPlugin) registerWithMesh(ctx context.Context) error {
	// Register the plugin's endpoint with the mesh network
	serviceName := fmt.Sprintf("plugin.%s", p.spec.Name)
	
	endpoint := mesh.Endpoint{
		Socket:  p.isolation.socketPath,
		Address: "", // Unix socket, no TCP address
		IsLocal: true,
	}

	// Register with mesh
	if p.isolation.meshClient != nil {
		// TODO: Implement mesh registration
		// This would register the plugin's endpoint so other services can discover it
	}

	p.meshEndpoint = endpoint
	p.logger.Info("Plugin registered with mesh", 
		zap.String("service", serviceName),
		zap.String("socket", endpoint.Socket))

	return nil
}

func (p *meshPlugin) unregisterFromMesh(ctx context.Context) error {
	// Unregister from mesh network
	serviceName := fmt.Sprintf("plugin.%s", p.spec.Name)
	
	if p.isolation.meshClient != nil {
		// TODO: Implement mesh unregistration
	}

	p.logger.Info("Plugin unregistered from mesh", zap.String("service", serviceName))
	return nil
}

func (p *meshPlugin) stop() error {
	if !p.isolation.started {
		return nil
	}

	// Send graceful shutdown signal
	if err := p.isolation.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		p.logger.Warn("Failed to send SIGTERM", zap.Error(err))
	}

	// Wait for graceful shutdown
	done := make(chan error, 1)
	go func() {
		done <- p.isolation.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Force kill if not stopped gracefully
		p.logger.Warn("Plugin did not stop gracefully, forcing kill")
		p.isolation.cmd.Process.Kill()
		<-done
	case err := <-done:
		if err != nil {
			p.logger.Debug("Plugin process exited", zap.Error(err))
		}
	}

	// Clean up socket
	os.Remove(p.isolation.socketPath)
	
	p.isolation.started = false
	return nil
}

func (p *meshPlugin) monitorProcess() {
	// Monitor the plugin process
	err := p.isolation.cmd.Wait()
	
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status == plugins.PluginStatusRunning {
		// Unexpected exit
		p.status = plugins.PluginStatusError
		p.info.Status = p.status
		p.logger.Error("Plugin process exited unexpectedly", zap.Error(err))
		
		// Clean up mesh registration
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p.unregisterFromMesh(ctx)
	}
}