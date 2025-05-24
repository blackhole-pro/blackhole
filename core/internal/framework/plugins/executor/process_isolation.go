// Package executor provides process isolation for plugin execution.
package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
)

// processIsolation implements process-level isolation for plugins
type processIsolation struct {
	cmd            *exec.Cmd
	stdin          io.WriteCloser
	stdout         io.ReadCloser
	stderr         io.ReadCloser
	encoder        *json.Encoder
	decoder        *json.Decoder
	resourceLimits plugins.PluginResources
	mu             sync.Mutex
	started        bool
}

// processPlugin implements the Plugin interface for process-isolated plugins
type processPlugin struct {
	spec       plugins.PluginSpec
	binaryPath string
	isolation  *processIsolation
	info       plugins.PluginInfo
	status     plugins.PluginStatus
	mu         sync.RWMutex
}

// RPC message types
type rpcMessage struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// NewProcessPlugin creates a new process-isolated plugin
func NewProcessPlugin(spec plugins.PluginSpec, binaryPath string) plugins.Plugin {
	return &processPlugin{
		spec:       spec,
		binaryPath: binaryPath,
		info: plugins.PluginInfo{
			Name:        spec.Name,
			Version:     spec.Version,
			Description: "Process-isolated plugin",
			Status:      plugins.PluginStatusStopped,
		},
		status: plugins.PluginStatusStopped,
	}
}

// Info returns plugin information
func (p *processPlugin) Info() plugins.PluginInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.info
}

// Start starts the plugin process
func (p *processPlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status == plugins.PluginStatusRunning {
		return nil
	}

	p.status = plugins.PluginStatusStarting
	p.info.Status = p.status

	// Create process isolation
	isolation := &processIsolation{
		resourceLimits: p.spec.Resources,
	}

	// Create the command
	isolation.cmd = exec.CommandContext(ctx, p.binaryPath)
	
	// Set up pipes for communication
	stdin, err := isolation.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	isolation.stdin = stdin

	stdout, err := isolation.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	isolation.stdout = stdout

	stderr, err := isolation.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	isolation.stderr = stderr

	// Set up JSON encoding/decoding
	isolation.encoder = json.NewEncoder(stdin)
	isolation.decoder = json.NewDecoder(stdout)

	// Set process attributes for resource limits
	isolation.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group
	}

	// Set environment variables
	isolation.cmd.Env = append(os.Environ(),
		fmt.Sprintf("PLUGIN_NAME=%s", p.spec.Name),
		fmt.Sprintf("PLUGIN_VERSION=%s", p.spec.Version),
		"PLUGIN_MODE=subprocess",
	)

	// Start the process
	if err := isolation.cmd.Start(); err != nil {
		p.status = plugins.PluginStatusFailed
		p.info.Status = p.status
		return fmt.Errorf("failed to start plugin process: %w", err)
	}

	isolation.started = true
	p.isolation = isolation

	// Start error monitoring
	go p.monitorStderr()

	// Initialize the plugin
	initReq := rpcMessage{
		ID:     "init",
		Method: "initialize",
		Params: json.RawMessage(fmt.Sprintf(`{"name":"%s","version":"%s"}`, p.spec.Name, p.spec.Version)),
	}

	resp, err := p.sendRequest(initReq)
	if err != nil {
		p.stop()
		p.status = plugins.PluginStatusFailed
		p.info.Status = p.status
		return fmt.Errorf("plugin initialization failed: %w", err)
	}

	if resp.Error != "" {
		p.stop()
		p.status = plugins.PluginStatusFailed
		p.info.Status = p.status
		return fmt.Errorf("plugin initialization error: %s", resp.Error)
	}

	// Update status
	p.status = plugins.PluginStatusRunning
	p.info.Status = p.status
	p.info.LoadTime = time.Now()

	return nil
}

// Stop stops the plugin process
func (p *processPlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != plugins.PluginStatusRunning {
		return nil
	}

	p.status = plugins.PluginStatusStopping
	p.info.Status = p.status

	// Send shutdown request
	shutdownReq := rpcMessage{
		ID:     "shutdown",
		Method: "shutdown",
	}

	// Try graceful shutdown first
	if _, err := p.sendRequest(shutdownReq); err == nil {
		// Wait for process to exit
		done := make(chan error, 1)
		go func() {
			done <- p.isolation.cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(10 * time.Second):
			// Force kill after timeout
			p.isolation.cmd.Process.Kill()
			<-done
		}
	} else {
		// If graceful shutdown fails, force kill
		p.stop()
	}

	p.status = plugins.PluginStatusStopped
	p.info.Status = p.status
	p.isolation = nil

	return nil
}

// Handle handles a plugin request
func (p *processPlugin) Handle(ctx context.Context, request plugins.PluginRequest) (plugins.PluginResponse, error) {
	p.mu.RLock()
	if p.status != plugins.PluginStatusRunning {
		p.mu.RUnlock()
		return plugins.PluginResponse{}, fmt.Errorf("plugin not running: status=%s", p.status)
	}
	p.mu.RUnlock()

	// Convert request to RPC message
	params, err := json.Marshal(request)
	if err != nil {
		return plugins.PluginResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	req := rpcMessage{
		ID:     request.ID,
		Method: "handle",
		Params: params,
	}

	// Send request
	resp, err := p.sendRequest(req)
	if err != nil {
		return plugins.PluginResponse{}, err
	}

	if resp.Error != "" {
		return plugins.PluginResponse{
			ID:      request.ID,
			Success: false,
			Error:   resp.Error,
		}, errors.New(resp.Error)
	}

	// Parse response
	var pluginResp plugins.PluginResponse
	if err := json.Unmarshal(resp.Result, &pluginResp); err != nil {
		return plugins.PluginResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return pluginResp, nil
}

// HealthCheck checks if the plugin is healthy
func (p *processPlugin) HealthCheck() error {
	p.mu.RLock()
	if p.status != plugins.PluginStatusRunning {
		p.mu.RUnlock()
		return fmt.Errorf("plugin not running")
	}
	p.mu.RUnlock()

	req := rpcMessage{
		ID:     "health",
		Method: "healthcheck",
	}

	resp, err := p.sendRequest(req)
	if err != nil {
		return err
	}

	if resp.Error != "" {
		return errors.New(resp.Error)
	}

	return nil
}

// GetStatus returns the plugin status
func (p *processPlugin) GetStatus() plugins.PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// PrepareShutdown prepares the plugin for shutdown
func (p *processPlugin) PrepareShutdown() error {
	req := rpcMessage{
		ID:     "prepare-shutdown",
		Method: "prepare_shutdown",
	}

	resp, err := p.sendRequest(req)
	if err != nil {
		return err
	}

	if resp.Error != "" {
		return errors.New(resp.Error)
	}

	return nil
}

// ExportState exports the plugin state
func (p *processPlugin) ExportState() ([]byte, error) {
	req := rpcMessage{
		ID:     "export-state",
		Method: "export_state",
	}

	resp, err := p.sendRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}

	// Result should be base64 encoded state
	var state string
	if err := json.Unmarshal(resp.Result, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return []byte(state), nil
}

// ImportState imports plugin state
func (p *processPlugin) ImportState(state []byte) error {
	params, err := json.Marshal(string(state))
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	req := rpcMessage{
		ID:     "import-state",
		Method: "import_state",
		Params: params,
	}

	resp, err := p.sendRequest(req)
	if err != nil {
		return err
	}

	if resp.Error != "" {
		return errors.New(resp.Error)
	}

	return nil
}

// sendRequest sends an RPC request to the plugin process
func (p *processPlugin) sendRequest(req rpcMessage) (*rpcResponse, error) {
	p.isolation.mu.Lock()
	defer p.isolation.mu.Unlock()

	// Send request
	if err := p.isolation.encoder.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	var resp rpcResponse
	if err := p.isolation.decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Verify response ID matches request
	if resp.ID != req.ID {
		return nil, fmt.Errorf("response ID mismatch: expected %s, got %s", req.ID, resp.ID)
	}

	return &resp, nil
}

// monitorStderr monitors the plugin's stderr for logging
func (p *processPlugin) monitorStderr() {
	scanner := bufio.NewScanner(p.isolation.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		// Log plugin stderr output
		fmt.Printf("[Plugin %s] %s\n", p.spec.Name, line)
	}
}

// stop forcefully stops the plugin process
func (p *processPlugin) stop() {
	if p.isolation != nil && p.isolation.started {
		// Close pipes
		p.isolation.stdin.Close()
		p.isolation.stdout.Close()
		p.isolation.stderr.Close()

		// Kill process group
		if p.isolation.cmd.Process != nil {
			syscall.Kill(-p.isolation.cmd.Process.Pid, syscall.SIGKILL)
		}
	}
}

// processIsolationFactory creates process isolation boundaries
type processIsolationFactory struct{}

// NewProcessIsolationFactory creates a new process isolation factory
func NewProcessIsolationFactory() IsolationFactory {
	return &processIsolationFactory{}
}

func (f *processIsolationFactory) CreateIsolation(level plugins.IsolationLevel, resources plugins.PluginResources) (plugins.IsolationBoundary, error) {
	switch level {
	case plugins.IsolationProcess:
		return &processIsolationBoundary{
			resources: resources,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidIsolationLevel, level)
	}
}

// processIsolationBoundary implements process-level isolation boundary
type processIsolationBoundary struct {
	resources plugins.PluginResources
	pid       int
}

func (b *processIsolationBoundary) Enter() error {
	// Process isolation is handled by exec.Command
	return nil
}

func (b *processIsolationBoundary) Exit() error {
	// Cleanup is handled by process termination
	return nil
}

func (b *processIsolationBoundary) EnforceResourceLimits(limits plugins.PluginResources) error {
	// TODO: Implement resource limits using cgroups on Linux
	// For now, just store the limits
	b.resources = limits
	return nil
}

func (b *processIsolationBoundary) GetResourceUsage() (plugins.PluginResourceUsage, error) {
	// TODO: Implement actual resource usage monitoring
	return plugins.PluginResourceUsage{
		CPU:    0,
		Memory: 0,
	}, nil
}