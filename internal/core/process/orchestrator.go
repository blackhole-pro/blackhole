package process

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"blackhole/internal/core/config"
	"go.uber.org/zap"
)

// ServiceProcess represents a running service process with state management
type ServiceProcess struct {
	Name        string
	Command     ProcessCmd
	PID         int
	State       ProcessState
	Started     time.Time
	Restarts    int
	LastError   error
	StopCh      chan struct{}
}

// Orchestrator manages service processes
type Orchestrator struct {
	// Configuration
	config        *OrchestratorConfig
	services      map[string]*ServiceConfig
	
	// Process tracking
	processes     map[string]*ServiceProcess
	processLock   sync.RWMutex
	
	// Communication channels
	sigCh         chan os.Signal
	doneCh        chan struct{}
	
	// Dependencies
	logger        *zap.Logger
	executor      ProcessExecutor
	
	// Control flags
	isShuttingDown atomic.Bool
}

// OrchestratorConfig contains configuration for the process orchestrator
type OrchestratorConfig struct {
	ServicesDir      string        `mapstructure:"services_dir"`
	SocketDir        string        `mapstructure:"socket_dir"`
	LogLevel         string        `mapstructure:"log_level"`
	AutoRestart      bool          `mapstructure:"auto_restart"`
	ShutdownTimeout  int           `mapstructure:"shutdown_timeout"`
}

// ServiceConfig contains configuration for a service
type ServiceConfig struct {
	Enabled      bool              `mapstructure:"enabled"`
	BinaryPath   string            `mapstructure:"binary_path"`
	DataDir      string            `mapstructure:"data_dir"`
	Args         []string          `mapstructure:"args"`
	Environment  map[string]string `mapstructure:"environment"`
	MemoryLimit  int               `mapstructure:"memory_limit"`
	CPUShares    int               `mapstructure:"cpu_shares"`
	IOWeight     int               `mapstructure:"io_weight"`
}

// OrchestratorOption allows configuring the orchestrator with functional options
type OrchestratorOption func(*Orchestrator)

// WithLogger sets a custom logger
func WithLogger(logger *zap.Logger) OrchestratorOption {
	return func(o *Orchestrator) {
		o.logger = logger
	}
}

// WithExecutor sets a custom process executor
func WithExecutor(executor ProcessExecutor) OrchestratorOption {
	return func(o *Orchestrator) {
		o.executor = executor
	}
}

// NewOrchestrator creates a new process orchestrator
func NewOrchestrator(config *config.Config, options ...OrchestratorOption) (*Orchestrator, error) {
	// Extract orchestrator config from app config
	orchestratorConfig := &OrchestratorConfig{
		ServicesDir:     "/var/lib/blackhole/services",
		SocketDir:       "/var/run/blackhole",
		LogLevel:        config.Server.LogLevel,
		AutoRestart:     true,
		ShutdownTimeout: 30,
	}
	
	// Extract service configs from app config
	serviceConfigs := make(map[string]*ServiceConfig)
	
	// Add identity service if enabled
	if config.Services.Identity.Enabled {
		serviceConfigs["identity"] = &ServiceConfig{
			Enabled:     true,
			DataDir:     filepath.Join(config.Storage.DataDir, "identity"),
			Environment: make(map[string]string),
		}
	}
	
	// Add storage service if enabled
	if config.Services.Storage.Enabled {
		serviceConfigs["storage"] = &ServiceConfig{
			Enabled:     true,
			DataDir:     filepath.Join(config.Storage.DataDir, "storage"),
			Environment: make(map[string]string),
		}
	}
	
	// Add other services similarly...
	if config.Services.Ledger.Enabled {
		serviceConfigs["ledger"] = &ServiceConfig{
			Enabled:     true,
			DataDir:     filepath.Join(config.Storage.DataDir, "ledger"),
			Environment: make(map[string]string),
		}
	}
	
	if config.Services.Indexer.Enabled {
		serviceConfigs["indexer"] = &ServiceConfig{
			Enabled:     true,
			DataDir:     filepath.Join(config.Storage.DataDir, "indexer"),
			Environment: make(map[string]string),
		}
	}
	
	if config.Services.Social.Enabled {
		serviceConfigs["social"] = &ServiceConfig{
			Enabled:     true, 
			DataDir:     filepath.Join(config.Storage.DataDir, "social"),
			Environment: make(map[string]string),
		}
	}
	
	if config.Services.Analytics.Enabled {
		serviceConfigs["analytics"] = &ServiceConfig{
			Enabled:     true,
			DataDir:     filepath.Join(config.Storage.DataDir, "analytics"),
			Environment: make(map[string]string),
		}
	}
	
	if config.Services.Telemetry.Enabled {
		serviceConfigs["telemetry"] = &ServiceConfig{
			Enabled:     true,
			DataDir:     filepath.Join(config.Storage.DataDir, "telemetry"),
			Environment: make(map[string]string),
		}
	}
	
	// Initialize orchestrator with configuration
	o := &Orchestrator{
		config:      orchestratorConfig,
		services:    serviceConfigs,
		processes:   make(map[string]*ServiceProcess),
		doneCh:      make(chan struct{}),
		executor:    &DefaultProcessExecutor{},
	}
	
	// Apply options
	for _, option := range options {
		option(o)
	}
	
	// Initialize logger if not provided
	if o.logger == nil {
		logger, err := initLogger(o.config.LogLevel)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize logger: %w", err)
		}
		o.logger = logger
	}
	
	// Setup signal handling
	o.setupSignals()
	
	// Verify services directory exists
	if !dirExists(o.config.ServicesDir) {
		o.logger.Warn("Services directory not found, creating it",
			zap.String("path", o.config.ServicesDir))
		if err := os.MkdirAll(o.config.ServicesDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create services directory: %w", err)
		}
	}
	
	// Verify socket directory exists
	if !dirExists(o.config.SocketDir) {
		o.logger.Warn("Socket directory not found, creating it",
			zap.String("path", o.config.SocketDir))
		if err := os.MkdirAll(o.config.SocketDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create socket directory: %w", err)
		}
	}
	
	o.logger.Info("Process orchestrator initialized",
		zap.Int("num_services", len(o.services)))
	
	return o, nil
}

// initLogger creates a new logger with the specified level
func initLogger(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	
	// Set log level
	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn", "warning":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	
	// Create logger
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	
	return logger, nil
}

// setupSignals initializes signal handling
func (o *Orchestrator) setupSignals() {
	o.sigCh = make(chan os.Signal, 1)
	signal.Notify(o.sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		sig := <-o.sigCh
		o.logger.Info("Received signal", zap.String("signal", sig.String()))
		o.Stop()
	}()
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// fileExists checks if a file exists and is not a directory
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// isExecutable checks if a file is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	
	// Check if file is executable by owner
	return info.Mode()&0100 != 0
}

// IsShuttingDown returns true if the orchestrator is in the process of shutting down
func (o *Orchestrator) IsShuttingDown() bool {
	return o.isShuttingDown.Load()
}