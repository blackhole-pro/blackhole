// Package executor implements plugin execution with isolation support.
package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
)

// Common errors
var (
	ErrInvalidIsolationLevel = errors.New("invalid isolation level")
	ErrExecutionTimeout      = errors.New("plugin execution timeout")
	ErrResourceLimitExceeded = errors.New("resource limit exceeded")
	ErrPluginNotResponding   = errors.New("plugin not responding")
)

// ExecutionEnvironment represents a plugin's execution environment
type ExecutionEnvironment struct {
	Plugin    plugins.Plugin
	Isolation plugins.IsolationBoundary
	StartTime time.Time
	LastCheck time.Time
}

// executionEnvironment implements the plugins.ExecutionEnvironment interface
type executionEnvironment struct {
	pluginName string
	resources  plugins.PluginResources
	workDir    string
	tempDir    string
	envVars    map[string]string
}

// GetResourceLimits returns the resource limits
func (e *executionEnvironment) GetResourceLimits() plugins.PluginResources {
	return e.resources
}

// GetEnvironmentVariables returns environment variables
func (e *executionEnvironment) GetEnvironmentVariables() map[string]string {
	return e.envVars
}

// GetWorkingDirectory returns the working directory
func (e *executionEnvironment) GetWorkingDirectory() string {
	return e.workDir
}

// GetTempDirectory returns the temp directory
func (e *executionEnvironment) GetTempDirectory() string {
	return e.tempDir
}

// pluginExecutor implements the PluginExecutor interface
type pluginExecutor struct {
	isolationFactory IsolationFactory
	environments     map[string]plugins.ExecutionEnvironment
	resourceMonitor  ResourceMonitor
	defaultTimeout   time.Duration
	mu               sync.RWMutex
}

// IsolationFactory creates isolation boundaries
type IsolationFactory interface {
	CreateIsolation(level plugins.IsolationLevel, resources plugins.PluginResources) (plugins.IsolationBoundary, error)
}

// ResourceMonitor monitors plugin resource usage
type ResourceMonitor interface {
	StartMonitoring(pluginName string) error
	StopMonitoring(pluginName string) error
	GetUsage(pluginName string) (plugins.PluginResourceUsage, error)
}

// Config holds executor configuration
type Config struct {
	DefaultTimeout   time.Duration
	ResourceMonitor  ResourceMonitor
	IsolationFactory IsolationFactory
}

// New creates a new plugin executor
func New(config Config) plugins.PluginExecutor {
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Second
	}

	return &pluginExecutor{
		isolationFactory: config.IsolationFactory,
		environments:     make(map[string]plugins.ExecutionEnvironment),
		resourceMonitor:  config.ResourceMonitor,
		defaultTimeout:   config.DefaultTimeout,
	}
}

// NewExecutor creates a new plugin executor with default configuration
func NewExecutor(maxConcurrent int, updateInterval time.Duration) plugins.PluginExecutor {
	return New(Config{
		DefaultTimeout:  30 * time.Second,
		ResourceMonitor: NewResourceMonitor(updateInterval),
		IsolationFactory: &ProcessIsolationFactory{
			MaxConcurrent: maxConcurrent,
		},
	})
}

// ExecutePlugin executes a plugin request with proper isolation
func (e *pluginExecutor) ExecutePlugin(plugin plugins.Plugin, request plugins.PluginRequest) (plugins.PluginResponse, error) {
	// Create execution context with timeout
	timeout := e.defaultTimeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Get or create execution environment
	_, err := e.GetExecutionEnvironment(plugin)
	if err != nil {
		return plugins.PluginResponse{}, fmt.Errorf("failed to get execution environment: %w", err)
	}

	// Start resource monitoring if available
	if e.resourceMonitor != nil {
		if err := e.resourceMonitor.StartMonitoring(plugin.Info().Name); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to start resource monitoring for %s: %v\n", plugin.Info().Name, err)
		}
	}

	// Execute the request
	startTime := time.Now()
	
	// Create response channel for timeout handling
	type result struct {
		response plugins.PluginResponse
		err      error
	}
	resultChan := make(chan result, 1)

	go func() {
		response, err := plugin.Handle(ctx, request)
		resultChan <- result{response: response, err: err}
	}()

	// Wait for response or timeout
	select {
	case r := <-resultChan:
		// Set execution metadata
		r.response.Metadata.ProcessingTime = time.Since(startTime)
		
		// Get resource usage if available
		if e.resourceMonitor != nil {
			usage, err := e.resourceMonitor.GetUsage(plugin.Info().Name)
			if err == nil {
				r.response.Metadata.ResourceUsage.CPU = usage.CPU
				r.response.Metadata.ResourceUsage.Memory = usage.Memory
			}
		}

		return r.response, r.err

	case <-ctx.Done():
		return plugins.PluginResponse{
			Success: false,
			Error:   ErrExecutionTimeout.Error(),
		}, ErrExecutionTimeout
	}
}

// GetExecutionEnvironment returns the execution environment for a plugin
func (e *pluginExecutor) GetExecutionEnvironment(plugin plugins.Plugin) (plugins.ExecutionEnvironment, error) {
	e.mu.RLock()
	env, exists := e.environments[plugin.Info().Name]
	e.mu.RUnlock()

	if exists {
		return env, nil
	}

	// Create new environment
	e.mu.Lock()
	defer e.mu.Unlock()

	// Double-check after acquiring write lock
	if env, exists := e.environments[plugin.Info().Name]; exists {
		return env, nil
	}

	// Create new execution environment
	env = &executionEnvironment{
		pluginName: plugin.Info().Name,
		workDir:    fmt.Sprintf("/tmp/blackhole-plugins/%s", plugin.Info().Name),
		tempDir:    fmt.Sprintf("/tmp/blackhole-plugins/%s/tmp", plugin.Info().Name),
		envVars:    make(map[string]string),
	}

	// Set default environment variables
	newEnv := env.(*executionEnvironment)
	newEnv.envVars["PLUGIN_NAME"] = plugin.Info().Name
	newEnv.envVars["PLUGIN_WORK_DIR"] = newEnv.workDir
	newEnv.envVars["PLUGIN_TEMP_DIR"] = newEnv.tempDir

	// Create directories
	if err := os.MkdirAll(newEnv.workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}
	if err := os.MkdirAll(newEnv.tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	e.environments[plugin.Info().Name] = env
	return env, nil
}

// CreateIsolationBoundary creates an isolation boundary for the given level
func (e *pluginExecutor) CreateIsolationBoundary(level plugins.IsolationLevel) (plugins.IsolationBoundary, error) {
	if e.isolationFactory == nil {
		// Return a no-op boundary if no factory is configured
		return &noOpIsolationBoundary{}, nil
	}

	// Default resources if not specified
	resources := plugins.PluginResources{
		CPU:     100, // 100% of one core
		Memory:  256, // 256 MB
		Disk:    100, // 100 MB
		Network: 10,  // 10 Mbps
	}

	return e.isolationFactory.CreateIsolation(level, resources)
}


// noOpIsolationBoundary provides no isolation (for testing or simple cases)
type noOpIsolationBoundary struct{}

func (b *noOpIsolationBoundary) Enter() error {
	return nil
}

func (b *noOpIsolationBoundary) Exit() error {
	return nil
}

func (b *noOpIsolationBoundary) EnforceResourceLimits(limits plugins.PluginResources) error {
	return nil
}

func (b *noOpIsolationBoundary) GetResourceUsage() (plugins.PluginResourceUsage, error) {
	return plugins.PluginResourceUsage{}, nil
}

// basicResourceMonitor provides basic resource monitoring
type basicResourceMonitor struct {
	usage map[string]plugins.PluginResourceUsage
	mu    sync.RWMutex
}

// NewBasicResourceMonitor creates a basic resource monitor
func NewBasicResourceMonitor() ResourceMonitor {
	return &basicResourceMonitor{
		usage: make(map[string]plugins.PluginResourceUsage),
	}
}

func (m *basicResourceMonitor) StartMonitoring(pluginName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Initialize usage tracking
	m.usage[pluginName] = plugins.PluginResourceUsage{}
	return nil
}

func (m *basicResourceMonitor) StopMonitoring(pluginName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.usage, pluginName)
	return nil
}

func (m *basicResourceMonitor) GetUsage(pluginName string) (plugins.PluginResourceUsage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	usage, exists := m.usage[pluginName]
	if !exists {
		return plugins.PluginResourceUsage{}, errors.New("plugin not monitored")
	}
	
	return usage, nil
}