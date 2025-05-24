package main

import (
	"fmt"
	"os"
	"time"

	"github.com/blackhole-pro/blackhole/core/internal/services/node/types"
	"gopkg.in/yaml.v3"
)

// LoadConfig loads the node service configuration from a file
func LoadConfig(configPath string) (*types.NodeConfig, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default configuration if file doesn't exist
		return getDefaultConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config types.NodeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate and apply defaults
	if err := validateAndApplyDefaults(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// getDefaultConfig returns a default configuration
func getDefaultConfig() *types.NodeConfig {
	return &types.NodeConfig{
		NodeID:        generateNodeID(),
		Version:       "1.0.0",
		ListenPort:    9000,
		ListenAddress: "0.0.0.0",
		ExternalIP:    "",
		
		MaxPeers:          50,
		MinPeers:          5,
		ConnectionTimeout: 30 * time.Second,
		PingInterval:      10 * time.Second,
		
		BootstrapPeers:   []string{},
		DiscoveryMethods: []string{"bootstrap", "dht"},
		DHT: types.DHTConfig{
			Enabled:         true,
			RefreshInterval: 5 * time.Minute,
			BucketSize:      20,
		},
		
		BandwidthLimit:   0, // No limit
		MessageQueueSize: 1000,
		
		EnableTLS:   false,
		TLSCertPath: "",
		TLSKeyPath:  "",
	}
}

// validateAndApplyDefaults validates configuration and applies defaults for missing values
func validateAndApplyDefaults(config *types.NodeConfig) error {
	// Validate required fields
	if config.NodeID == "" {
		config.NodeID = generateNodeID()
	}
	
	if config.Version == "" {
		config.Version = "1.0.0"
	}
	
	// Validate ports
	if config.ListenPort <= 0 || config.ListenPort > 65535 {
		return types.NewInvalidConfigError("listen_port", config.ListenPort)
	}
	
	// Validate peer limits
	if config.MaxPeers <= 0 {
		config.MaxPeers = 50
	}
	
	if config.MinPeers < 0 {
		config.MinPeers = 0
	}
	
	if config.MinPeers > config.MaxPeers {
		return types.NewInvalidConfigError("min_peers", 
			fmt.Sprintf("min_peers (%d) cannot be greater than max_peers (%d)", 
				config.MinPeers, config.MaxPeers))
	}
	
	// Validate timeouts
	if config.ConnectionTimeout <= 0 {
		config.ConnectionTimeout = 30 * time.Second
	}
	
	if config.PingInterval <= 0 {
		config.PingInterval = 10 * time.Second
	}
	
	// Validate discovery methods
	if len(config.DiscoveryMethods) == 0 {
		config.DiscoveryMethods = []string{"bootstrap"}
	}
	
	for _, method := range config.DiscoveryMethods {
		if !isValidDiscoveryMethod(method) {
			return types.NewInvalidConfigError("discovery_methods", method)
		}
	}
	
	// Validate DHT config if enabled
	if config.DHT.Enabled {
		if config.DHT.RefreshInterval <= 0 {
			config.DHT.RefreshInterval = 5 * time.Minute
		}
		
		if config.DHT.BucketSize <= 0 {
			config.DHT.BucketSize = 20
		}
	}
	
	// Validate TLS config
	if config.EnableTLS {
		if config.TLSCertPath == "" {
			return types.NewInvalidConfigError("tls_cert_path", "required when TLS is enabled")
		}
		
		if config.TLSKeyPath == "" {
			return types.NewInvalidConfigError("tls_key_path", "required when TLS is enabled")
		}
		
		// Check if cert files exist
		if _, err := os.Stat(config.TLSCertPath); os.IsNotExist(err) {
			return types.NewInvalidConfigError("tls_cert_path", "file does not exist")
		}
		
		if _, err := os.Stat(config.TLSKeyPath); os.IsNotExist(err) {
			return types.NewInvalidConfigError("tls_key_path", "file does not exist")
		}
	}
	
	// Validate message queue size
	if config.MessageQueueSize <= 0 {
		config.MessageQueueSize = 1000
	}
	
	return nil
}

// isValidDiscoveryMethod checks if a discovery method is valid
func isValidDiscoveryMethod(method string) bool {
	validMethods := []string{"bootstrap", "dht", "local", "static"}
	for _, valid := range validMethods {
		if method == valid {
			return true
		}
	}
	return false
}

// generateNodeID generates a unique node ID
func generateNodeID() string {
	// Simple implementation - in production, use more sophisticated method
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	timestamp := time.Now().Unix()
	
	return fmt.Sprintf("node-%s-%d-%d", hostname, pid, timestamp)
}

// SaveConfig saves the configuration to a file
func SaveConfig(config *types.NodeConfig, configPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// GetConfigValue gets a specific configuration value by key
func GetConfigValue(config *types.NodeConfig, key string) interface{} {
	switch key {
	case "node_id":
		return config.NodeID
	case "version":
		return config.Version
	case "listen_port":
		return config.ListenPort
	case "listen_address":
		return config.ListenAddress
	case "max_peers":
		return config.MaxPeers
	case "min_peers":
		return config.MinPeers
	case "connection_timeout":
		return config.ConnectionTimeout
	case "ping_interval":
		return config.PingInterval
	case "bootstrap_peers":
		return config.BootstrapPeers
	case "discovery_methods":
		return config.DiscoveryMethods
	case "bandwidth_limit":
		return config.BandwidthLimit
	case "message_queue_size":
		return config.MessageQueueSize
	case "enable_tls":
		return config.EnableTLS
	default:
		return nil
	}
}