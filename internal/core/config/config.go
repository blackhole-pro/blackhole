package config

import (
	"fmt"
	"sync"

	"github.com/spf13/viper"
)

var (
	// Version information (set by build flags)
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
	
	// Global config instance
	globalConfig *Config
	configOnce   sync.Once
)

// Config represents the application configuration
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:"server"`
	
	// Service configurations
	Services ServicesConfig `mapstructure:"services"`
	
	// Storage configuration
	Storage StorageConfig `mapstructure:"storage"`
	
	// Network configuration
	Network NetworkConfig `mapstructure:"network"`
	
	// Security configuration
	Security SecurityConfig `mapstructure:"security"`
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	HTTPAddr  string `mapstructure:"http_addr"`
	GRPCAddr  string `mapstructure:"grpc_addr"`
	LogLevel  string `mapstructure:"log_level"`
	EnableTLS bool   `mapstructure:"enable_tls"`
}

// ServicesConfig contains service-specific configurations
type ServicesConfig struct {
	Identity  ServiceConfig `mapstructure:"identity"`
	Storage   ServiceConfig `mapstructure:"storage"`
	Ledger    ServiceConfig `mapstructure:"ledger"`
	Indexer   ServiceConfig `mapstructure:"indexer"`
	Social    ServiceConfig `mapstructure:"social"`
	Analytics ServiceConfig `mapstructure:"analytics"`
	Telemetry ServiceConfig `mapstructure:"telemetry"`
}

// ServiceConfig contains common service configuration
type ServiceConfig struct {
	Enabled    bool              `mapstructure:"enabled"`
	MaxWorkers int               `mapstructure:"max_workers"`
	Timeout    string            `mapstructure:"timeout"`
	Options    map[string]string `mapstructure:"options"`
}

// StorageConfig contains storage-related configuration
type StorageConfig struct {
	IPFSEndpoint     string `mapstructure:"ipfs_endpoint"`
	FilecoinEndpoint string `mapstructure:"filecoin_endpoint"`
	DataDir          string `mapstructure:"data_dir"`
	CacheSize        int64  `mapstructure:"cache_size"`
}

// NetworkConfig contains networking configuration
type NetworkConfig struct {
	P2PEnabled   bool     `mapstructure:"p2p_enabled"`
	BootstrapNodes []string `mapstructure:"bootstrap_nodes"`
	ListenAddrs  []string `mapstructure:"listen_addrs"`
	EnableRelay  bool     `mapstructure:"enable_relay"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	TLSCertFile string `mapstructure:"tls_cert_file"`
	TLSKeyFile  string `mapstructure:"tls_key_file"`
	JWTSecret   string `mapstructure:"jwt_secret"`
	EnableAuth  bool   `mapstructure:"enable_auth"`
}

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	configOnce.Do(func() {
		globalConfig = &Config{
			Server: ServerConfig{
				HTTPAddr: ":8080",
				GRPCAddr: ":9090",
				LogLevel: "info",
			},
			Services: ServicesConfig{
				Identity:  ServiceConfig{Enabled: true},
				Storage:   ServiceConfig{Enabled: true},
				Ledger:    ServiceConfig{Enabled: true},
				Indexer:   ServiceConfig{Enabled: true},
				Social:    ServiceConfig{Enabled: true},
				Analytics: ServiceConfig{Enabled: true},
				Telemetry: ServiceConfig{Enabled: true},
			},
			Storage: StorageConfig{
				DataDir:   "/var/lib/blackhole",
				CacheSize: 1024 * 1024 * 1024, // 1GB
			},
			Network: NetworkConfig{
				P2PEnabled: true,
			},
		}
	})
	return globalConfig
}

// GetConfig returns the global configuration instance
func GetConfig() *Config {
	return NewConfig()
}

// Load loads configuration from various sources
func (c *Config) Load() error {
	v := viper.New()
	
	// Set config name and paths
	v.SetConfigName("blackhole")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/blackhole")
	v.AddConfigPath("$HOME/.blackhole")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	
	// Set environment variable prefix
	v.SetEnvPrefix("BLACKHOLE")
	v.AutomaticEnv()
	
	// Set defaults
	v.SetDefault("server.http_addr", ":8080")
	v.SetDefault("server.grpc_addr", ":9090")
	v.SetDefault("server.log_level", "info")
	v.SetDefault("storage.data_dir", "/var/lib/blackhole")
	
	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; use defaults
	}
	
	// Unmarshal into struct
	if err := v.Unmarshal(c); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return nil
}

// Validate validates the configuration
func ValidateConfig() error {
	config := GetConfig()
	
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
	
	return nil
}