package types_test

import (
	"errors"
	"testing"

	"node/types"
)

func TestCommonErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "ErrPluginNotRunning",
			err:  types.ErrPluginNotRunning,
			want: "plugin not running",
		},
		{
			name: "ErrPluginAlreadyRunning",
			err:  types.ErrPluginAlreadyRunning,
			want: "plugin already running",
		},
		{
			name: "ErrPeerNotFound",
			err:  types.ErrPeerNotFound,
			want: "peer not found",
		},
		{
			name: "ErrMaxPeersReached",
			err:  types.ErrMaxPeersReached,
			want: "maximum peers limit reached",
		},
		{
			name: "ErrInvalidConfig",
			err:  types.ErrInvalidConfig,
			want: "invalid configuration",
		},
		{
			name: "ErrDiscoveryDisabled",
			err:  types.ErrDiscoveryDisabled,
			want: "peer discovery is disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPeerError(t *testing.T) {
	tests := []struct {
		name      string
		peerID    string
		operation string
		err       error
		want      string
	}{
		{
			name:      "basic peer error",
			peerID:    "peer123",
			operation: "connect",
			err:       errors.New("connection refused"),
			want:      "peer peer123: connect failed: connection refused",
		},
		{
			name:      "peer error with wrapped error",
			peerID:    "peer456",
			operation: "disconnect",
			err:       types.ErrPeerNotFound,
			want:      "peer peer456: disconnect failed: peer not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.NewPeerError(tt.peerID, tt.operation, tt.err)
			if got := err.Error(); got != tt.want {
				t.Errorf("PeerError.Error() = %v, want %v", got, tt.want)
			}

			// Test Unwrap
			var peerErr *types.PeerError
			if errors.As(err, &peerErr) {
				if unwrapped := peerErr.Unwrap(); unwrapped != tt.err {
					t.Errorf("PeerError.Unwrap() = %v, want %v", unwrapped, tt.err)
				}
			} else {
				t.Error("Failed to cast to PeerError")
			}
		})
	}
}

func TestConfigError(t *testing.T) {
	tests := []struct {
		name  string
		field string
		value interface{}
		err   error
		want  string
	}{
		{
			name:  "invalid port",
			field: "p2pPort",
			value: -1,
			err:   errors.New("must be positive"),
			want:  "config field p2pPort with value -1: must be positive",
		},
		{
			name:  "missing node ID",
			field: "nodeId",
			value: "",
			err:   types.ErrMissingNodeID,
			want:  "config field nodeId with value : node ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.NewConfigError(tt.field, tt.value, tt.err)
			if got := err.Error(); got != tt.want {
				t.Errorf("ConfigError.Error() = %v, want %v", got, tt.want)
			}

			// Test Unwrap
			var configErr *types.ConfigError
			if errors.As(err, &configErr) {
				if unwrapped := configErr.Unwrap(); unwrapped != tt.err {
					t.Errorf("ConfigError.Unwrap() = %v, want %v", unwrapped, tt.err)
				}
			} else {
				t.Error("Failed to cast to ConfigError")
			}
		})
	}
}

func TestNetworkError(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		address   string
		err       error
		want      string
	}{
		{
			name:      "with address",
			operation: "connect",
			address:   "192.168.1.100:4001",
			err:       errors.New("timeout"),
			want:      "network connect on 192.168.1.100:4001 failed: timeout",
		},
		{
			name:      "without address",
			operation: "listen",
			address:   "",
			err:       errors.New("port already in use"),
			want:      "network listen failed: port already in use",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.NewNetworkError(tt.operation, tt.address, tt.err)
			if got := err.Error(); got != tt.want {
				t.Errorf("NetworkError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		message string
		want    string
	}{
		{
			name:    "invalid port range",
			field:   "p2pPort",
			message: "must be between 1 and 65535",
			want:    "validation failed for p2pPort: must be between 1 and 65535",
		},
		{
			name:    "negative value",
			field:   "maxPeers",
			message: "cannot be negative",
			want:    "validation failed for maxPeers: cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.NewValidationError(tt.field, tt.message)
			if got := err.Error(); got != tt.want {
				t.Errorf("ValidationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that errors can be properly wrapped and unwrapped
	baseErr := errors.New("base error")
	peerErr := types.NewPeerError("peer1", "test", baseErr)
	
	// Should be able to check if it's a PeerError
	var pe *types.PeerError
	if !errors.As(peerErr, &pe) {
		t.Error("Expected to be able to cast to PeerError")
	}
	
	// Should be able to unwrap to base error
	if !errors.Is(peerErr, baseErr) {
		t.Error("Expected to be able to unwrap to base error")
	}
}