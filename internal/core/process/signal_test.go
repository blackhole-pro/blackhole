// Package process provides the implementation of the process orchestrator
// which manages service processes for the Blackhole platform.
package process

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupSignals tests the signal handling logic
func TestSetupSignals(t *testing.T) {
	// Create test orchestrator
	orch, _, _, _ := setupTestOrchestrator(t)
	
	// Ensure signal channel was created
	assert.NotNil(t, orch.sigCh)

	// Create a channel to track shutdown calls
	shutdownCalled := make(chan struct{})
	
	// Replace actual shutdown method with a test version
	originalShutdown := orch.Shutdown
	orch.Shutdown = func(ctx context.Context) error {
		close(shutdownCalled)
		return nil
	}
	defer func() {
		orch.Shutdown = originalShutdown
	}()
	
	// Test for each signal
	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	
	for _, sig := range signals {
		t.Run(sig.String(), func(t *testing.T) {
			// Reset shutdown tracking
			shutdownCalled = make(chan struct{})
			
			// Send signal to orchestrator
			orch.sigCh <- sig
			
			// Wait for shutdown to be called or timeout
			select {
			case <-shutdownCalled:
				// Shutdown was called as expected
			case <-time.After(100 * time.Millisecond):
				t.Fatal("Shutdown was not called after signal")
			}
		})
	}
}

// TestSetupSignals_SignalNotify tests the signal.Notify functionality
func TestSetupSignals_SignalNotify(t *testing.T) {
	// Skip in short mode to avoid affecting other tests
	if testing.Short() {
		t.Skip("Skipping signal notify test in short mode")
	}
	
	// Save original signal.Notify to restore later
	originalNotify := signal.Notify
	defer func() {
		signal.Notify = originalNotify
	}()
	
	// Track calls to signal.Notify
	var notifyCalled bool
	var notifySignals []os.Signal
	var notifyChan chan<- os.Signal
	
	// Replace signal.Notify with test function
	signal.Notify = func(c chan<- os.Signal, sig ...os.Signal) {
		notifyCalled = true
		notifySignals = sig
		notifyChan = c
	}
	
	// Create orchestrator to trigger setupSignals
	orch, err := NewOrchestrator(
		newTestConfigManager(t, newTestConfig(t, t.TempDir())),
	)
	require.NoError(t, err)
	
	// Verify signal.Notify was called
	assert.True(t, notifyCalled)
	
	// Verify correct signals were registered
	assert.Contains(t, notifySignals, syscall.SIGINT)
	assert.Contains(t, notifySignals, syscall.SIGTERM)
	
	// Verify the channel passed to signal.Notify is the orchestrator's sigCh
	assert.Equal(t, orch.sigCh, notifyChan)
}

// TestSignalHandling_Integration tests actual signal handling with a real signal
func TestSignalHandling_Integration(t *testing.T) {
	// Skip in CI or short mode to avoid affecting other tests
	if testing.Short() || os.Getenv("CI") != "" {
		t.Skip("Skipping signal handling integration test in short/CI mode")
	}
	
	// Create test orchestrator
	orch, _, _, _ := setupTestOrchestrator(t)
	
	// Create a channel to track shutdown calls
	shutdownCalled := make(chan struct{})
	
	// Replace actual shutdown method with a test version
	originalShutdown := orch.Shutdown
	orch.Shutdown = func(ctx context.Context) error {
		close(shutdownCalled)
		return nil
	}
	defer func() {
		orch.Shutdown = originalShutdown
	}()
	
	// Send a real signal to the current process
	// This is risky as it affects the whole process, but it's the most direct way to test
	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	
	// Use SIGUSR1 which is less likely to affect other tests
	// Note: this assumes we're on a Unix-like system
	err = p.Signal(syscall.SIGUSR1)
	require.NoError(t, err)
	
	// Signal should not trigger shutdown since we don't listen for SIGUSR1
	select {
	case <-shutdownCalled:
		t.Fatal("Shutdown was incorrectly called for SIGUSR1")
	case <-time.After(100 * time.Millisecond):
		// This is expected - SIGUSR1 should be ignored
	}
}