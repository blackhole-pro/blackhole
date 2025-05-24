package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// Import other service clients
	identityv1 "github.com/blackhole-pro/blackhole/core/internal/rpc/gen/identity/auth/v1"
)

// ServiceClients contains gRPC clients for other services
type ServiceClients struct {
	Identity *IdentityClient
	logger   *zap.Logger
}

// NewServiceClients creates a new set of service clients
func NewServiceClients(logger *zap.Logger) *ServiceClients {
	return &ServiceClients{
		logger: logger,
	}
}

// InitializeClients sets up connections to other services
func (sc *ServiceClients) InitializeClients(config *ServiceClientConfig) error {
	var err error
	
	// Initialize Identity client
	if config.Identity.Enabled {
		sc.Identity, err = NewIdentityClient(config.Identity, sc.logger)
		if err != nil {
			return fmt.Errorf("failed to create identity client: %w", err)
		}
	}
	
	return nil
}

// Close closes all client connections
func (sc *ServiceClients) Close() error {
	var errs []error
	
	if sc.Identity != nil {
		if err := sc.Identity.Close(); err != nil {
			errs = append(errs, fmt.Errorf("identity client close error: %w", err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("client close errors: %v", errs)
	}
	
	return nil
}

// ServiceClientConfig contains configuration for service clients
type ServiceClientConfig struct {
	Identity ClientEndpointConfig `yaml:"identity"`
}

// ClientEndpointConfig contains endpoint configuration for a service client
type ClientEndpointConfig struct {
	Enabled      bool          `yaml:"enabled"`
	UnixSocket   string        `yaml:"unix_socket"`
	TCPAddress   string        `yaml:"tcp_address"`
	Timeout      time.Duration `yaml:"timeout"`
	RetryAttempts int          `yaml:"retry_attempts"`
	UseUnixSocket bool         `yaml:"use_unix_socket"`
}

// IdentityClient wraps the identity service gRPC client
type IdentityClient struct {
	conn   *grpc.ClientConn
	client identityv1.AuthServiceClient
	config ClientEndpointConfig
	logger *zap.Logger
}

// NewIdentityClient creates a new identity service client
func NewIdentityClient(config ClientEndpointConfig, logger *zap.Logger) (*IdentityClient, error) {
	conn, err := createGRPCConnection(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}
	
	client := identityv1.NewAuthServiceClient(conn)
	
	return &IdentityClient{
		conn:   conn,
		client: client,
		config: config,
		logger: logger,
	}, nil
}

// GenerateChallenge calls the identity service to generate an authentication challenge
func (ic *IdentityClient) GenerateChallenge(ctx context.Context, did string, purpose string) (*identityv1.Challenge, error) {
	if ic.client == nil {
		return nil, fmt.Errorf("identity client not initialized")
	}
	
	req := &identityv1.GenerateChallengeRequest{
		Did:     did,
		Purpose: purpose,
	}
	
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, ic.config.Timeout)
	defer cancel()
	
	ic.logger.Debug("Calling identity service GenerateChallenge",
		zap.String("did", did),
		zap.String("purpose", purpose))
	
	resp, err := ic.client.GenerateChallenge(ctx, req)
	if err != nil {
		ic.logger.Error("GenerateChallenge failed",
			zap.String("did", did),
			zap.Error(err))
		return nil, fmt.Errorf("generate challenge failed: %w", err)
	}
	
	ic.logger.Debug("GenerateChallenge successful",
		zap.String("challenge_id", resp.Id))
	
	return resp, nil
}

// VerifyResponse calls the identity service to verify an authentication response
func (ic *IdentityClient) VerifyResponse(ctx context.Context, authResponse *identityv1.AuthResponse) (*identityv1.AuthResult, error) {
	if ic.client == nil {
		return nil, fmt.Errorf("identity client not initialized")
	}
	
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, ic.config.Timeout)
	defer cancel()
	
	ic.logger.Debug("Calling identity service VerifyResponse",
		zap.String("challenge_id", authResponse.ChallengeId),
		zap.String("did", authResponse.Did))
	
	resp, err := ic.client.VerifyResponse(ctx, authResponse)
	if err != nil {
		ic.logger.Error("VerifyResponse failed",
			zap.String("challenge_id", authResponse.ChallengeId),
			zap.Error(err))
		return nil, fmt.Errorf("verify response failed: %w", err)
	}
	
	ic.logger.Debug("VerifyResponse successful",
		zap.String("did", resp.Did),
		zap.Bool("verified", resp.Verified))
	
	return resp, nil
}

// Close closes the identity client connection
func (ic *IdentityClient) Close() error {
	if ic.conn != nil {
		return ic.conn.Close()
	}
	return nil
}


// createGRPCConnection creates a gRPC connection based on configuration
func createGRPCConnection(config ClientEndpointConfig, logger *zap.Logger) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	var err error
	
	// Set default timeout if not specified
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	if config.UseUnixSocket && config.UnixSocket != "" {
		// Connect via Unix socket
		logger.Debug("Connecting via Unix socket",
			zap.String("socket", config.UnixSocket))
		
		dialOpts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
				dialer := &net.Dialer{}
				return dialer.DialContext(ctx, "unix", addr)
			}),
		}
		
		conn, err = grpc.DialContext(ctx, config.UnixSocket, dialOpts...)
	} else if config.TCPAddress != "" {
		// Connect via TCP
		logger.Debug("Connecting via TCP",
			zap.String("address", config.TCPAddress))
		
		dialOpts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		
		conn, err = grpc.DialContext(ctx, config.TCPAddress, dialOpts...)
	} else {
		return nil, fmt.Errorf("no connection endpoint specified")
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}
	
	return conn, nil
}

// GetDefaultClientConfig returns default configuration for service clients
func GetDefaultClientConfig() *ServiceClientConfig {
	return &ServiceClientConfig{
		Identity: ClientEndpointConfig{
			Enabled:       true,
			UnixSocket:    "./sockets/identity.sock",
			TCPAddress:    "localhost:8101",
			Timeout:       10 * time.Second,
			RetryAttempts: 3,
			UseUnixSocket: true,
		},
	}
}