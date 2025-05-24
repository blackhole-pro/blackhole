// Package types defines the core type definitions for configuration
// used throughout the Blackhole platform.
package types

// Config represents the application configuration
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:"server" yaml:"server" json:"server"`
	
	// Service configurations
	Services ServicesConfig `mapstructure:"services" yaml:"services" json:"services"`
	
	// Network configuration
	Network NetworkConfig `mapstructure:"network" yaml:"network" json:"network"`
	
	// Security configuration
	Security SecurityConfig `mapstructure:"security" yaml:"security" json:"security"`
	
	// Orchestrator configuration
	Orchestrator OrchestratorConfig `mapstructure:"orchestrator" yaml:"orchestrator" json:"orchestrator"`
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	HTTPAddr  string `mapstructure:"http_addr" yaml:"http_addr" json:"http_addr"`
	GRPCAddr  string `mapstructure:"grpc_addr" yaml:"grpc_addr" json:"grpc_addr"`
	LogLevel  string `mapstructure:"log_level" yaml:"log_level" json:"log_level"`
	EnableTLS bool   `mapstructure:"enable_tls" yaml:"enable_tls" json:"enable_tls"`
}

// ServicesConfig contains service-specific configurations
type ServicesConfig map[string]*ServiceConfig

// ServiceConfig contains common service configuration
type ServiceConfig struct {
	Enabled     bool              `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	MaxWorkers  int               `mapstructure:"max_workers" yaml:"max_workers" json:"max_workers"`
	Timeout     string            `mapstructure:"timeout" yaml:"timeout" json:"timeout"`
	Options     map[string]string `mapstructure:"options" yaml:"options" json:"options"`
	
	// Fields needed for the orchestrator
	BinaryPath  string            `mapstructure:"binary_path" yaml:"binary_path" json:"binary_path"`
	DataDir     string            `mapstructure:"data_dir" yaml:"data_dir" json:"data_dir"`
	Args        []string          `mapstructure:"args" yaml:"args" json:"args"`
	Environment map[string]string `mapstructure:"environment" yaml:"environment" json:"environment"`
	MemoryLimit int               `mapstructure:"memory_limit" yaml:"memory_limit" json:"memory_limit"`
	CPUShares   int               `mapstructure:"cpu_shares" yaml:"cpu_shares" json:"cpu_shares"`
	IOWeight    int               `mapstructure:"io_weight" yaml:"io_weight" json:"io_weight"`
}

// NetworkConfig contains networking configuration
type NetworkConfig struct {
	P2PEnabled     bool     `mapstructure:"p2p_enabled" yaml:"p2p_enabled" json:"p2p_enabled"`
	BootstrapNodes []string `mapstructure:"bootstrap_nodes" yaml:"bootstrap_nodes" json:"bootstrap_nodes"`
	ListenAddrs    []string `mapstructure:"listen_addrs" yaml:"listen_addrs" json:"listen_addrs"`
	EnableRelay    bool     `mapstructure:"enable_relay" yaml:"enable_relay" json:"enable_relay"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	TLSCertFile string `mapstructure:"tls_cert_file" yaml:"tls_cert_file" json:"tls_cert_file"`
	TLSKeyFile  string `mapstructure:"tls_key_file" yaml:"tls_key_file" json:"tls_key_file"`
	JWTSecret   string `mapstructure:"jwt_secret" yaml:"jwt_secret" json:"jwt_secret"`
	EnableAuth  bool   `mapstructure:"enable_auth" yaml:"enable_auth" json:"enable_auth"`
}

// OrchestratorConfig contains configuration for the process orchestrator
type OrchestratorConfig struct {
	ServicesDir     string `mapstructure:"services_dir" yaml:"services_dir" json:"services_dir"`
	SocketDir       string `mapstructure:"socket_dir" yaml:"socket_dir" json:"socket_dir"`
	LogLevel        string `mapstructure:"log_level" yaml:"log_level" json:"log_level"`
	AutoRestart     bool   `mapstructure:"auto_restart" yaml:"auto_restart" json:"auto_restart"`
	ShutdownTimeout int    `mapstructure:"shutdown_timeout" yaml:"shutdown_timeout" json:"shutdown_timeout"`
}