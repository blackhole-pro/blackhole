// Package config provides configuration management functionality
package config

import (
	"fmt"
	"strings"

	"github.com/handcraftdev/blackhole/internal/core/config/types"
	"github.com/spf13/viper"
)

var (
	// Version information (set by build flags)
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// NewDefaultConfig creates a new configuration with default values
func NewDefaultConfig() *types.Config {
	return &types.Config{
		Server: types.ServerConfig{
			HTTPAddr: ":8080",
			GRPCAddr: ":9090",
			LogLevel: "info",
		},
		Services: make(types.ServicesConfig),
		Storage: types.StorageConfig{
			DataDir:   "/var/lib/blackhole",
			CacheSize: 1024 * 1024 * 1024, // 1GB
		},
		Network: types.NetworkConfig{
			P2PEnabled: true,
		},
		Orchestrator: types.OrchestratorConfig{
			ServicesDir:     "./services",
			SocketDir:       "./sockets",
			LogLevel:        "info",
			AutoRestart:     true,
			ShutdownTimeout: 30,
		},
	}
}

// FileLoader loads configuration from a file
type FileLoader struct {
	path string
}

// NewFileLoader creates a new file loader
func NewFileLoader(path string) *FileLoader {
	return &FileLoader{path: path}
}

// Load loads configuration from a file
func (l *FileLoader) Load() (*types.Config, error) {
	v := viper.New()
	
	// Set config name and paths
	if l.path != "" {
		v.SetConfigFile(l.path)
	} else {
		v.SetConfigName("blackhole")
		v.SetConfigType("yaml")
		v.AddConfigPath("/etc/blackhole")
		v.AddConfigPath("$HOME/.blackhole")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
	}
	
	// Set environment variable prefix
	v.SetEnvPrefix("BLACKHOLE")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	
	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; use defaults
		fmt.Printf("Config file not found, using defaults\n")
	} else {
		fmt.Printf("Config loaded successfully: %s\n", v.ConfigFileUsed())
		fmt.Printf("ServicesDir from config: %s\n", v.GetString("orchestrator.services_dir"))
	}
	
	// Create new config with defaults
	config := NewDefaultConfig()
	
	// Unmarshal into config struct
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return config, nil
}

// FileWriter writes configuration to a file
type FileWriter struct {
	path string
}

// NewFileWriter creates a new file writer
func NewFileWriter(path string) *FileWriter {
	return &FileWriter{path: path}
}

// Write writes configuration to a file
func (w *FileWriter) Write(config *types.Config) error {
	v := viper.New()
	
	// Set config file
	v.SetConfigFile(w.path)
	
	// Convert to map
	configMap := make(map[string]interface{})
	v.MergeConfigMap(configMap)
	
	// Set all config values
	v.Set("server", config.Server)
	v.Set("services", config.Services)
	v.Set("storage", config.Storage)
	v.Set("network", config.Network)
	v.Set("security", config.Security)
	v.Set("orchestrator", config.Orchestrator)
	
	// Write config file
	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// ValidateConfig validates the provided configuration
func ValidateConfig(config *types.Config) error {
	// Validate server config
	if config.Server.HTTPAddr == "" {
		return fmt.Errorf("server.http_addr cannot be empty")
	}
	if config.Server.GRPCAddr == "" {
		return fmt.Errorf("server.grpc_addr cannot be empty")
	}
	
	// Validate storage config
	if config.Storage.DataDir == "" {
		return fmt.Errorf("storage.data_dir cannot be empty")
	}
	
	// Validate orchestrator config
	if config.Orchestrator.ServicesDir == "" {
		return fmt.Errorf("orchestrator.services_dir cannot be empty")
	}
	if config.Orchestrator.ShutdownTimeout <= 0 {
		return fmt.Errorf("orchestrator.shutdown_timeout must be positive")
	}
	
	return nil
}