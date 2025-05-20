// Package supervision provides process supervision functionality for the Process Orchestrator.
// It handles process monitoring, restart logic, and exponential backoff calculations.
package supervision

import (
	"fmt"
	"math"
	"math/rand"
	"os/exec"
	"time"

	"github.com/handcraftdev/blackhole/internal/core/config"
	"github.com/handcraftdev/blackhole/internal/core/process/types"
	"go.uber.org/zap"
)

// SupervisorConfig contains configuration for the supervisor
type SupervisorConfig struct {
	// AutoRestart enables automatic restart of failed processes
	AutoRestart bool
	// MaxRestartAttempts limits the number of restart attempts
	MaxRestartAttempts int
	// InitialBackoffMs is the initial backoff delay in milliseconds
	InitialBackoffMs int
	// MaxBackoffMs is the maximum backoff delay in milliseconds
	MaxBackoffMs int
}

// ProcessInfo contains information about a supervised process
type ProcessInfo struct {
	Name      string
	Command   types.ProcessCmd
	State     types.ProcessState
	PID       int
	Restarts  int
	LastError error
	StopCh    chan struct{}
	Started   time.Time
}

// Supervisor handles process supervision for services
type Supervisor struct {
	spawner     ProcessSpawner
	config      SupervisorConfig
	logger      *zap.Logger
	maxAttempts int
}

// ProcessSpawner defines an interface for process spawning
type ProcessSpawner interface {
	SpawnProcess(name string) error
}

// NewSupervisor creates a new process supervisor
func NewSupervisor(spawner ProcessSpawner, config SupervisorConfig, logger *zap.Logger) *Supervisor {
	// Set default values if not specified
	if config.MaxRestartAttempts == 0 {
		config.MaxRestartAttempts = 10
	}
	if config.InitialBackoffMs == 0 {
		config.InitialBackoffMs = 1000
	}
	if config.MaxBackoffMs == 0 {
		config.MaxBackoffMs = 30000
	}

	return &Supervisor{
		spawner: spawner,
		config:  config,
		logger:  logger,
	}
}

// Supervise monitors a process and restarts it if it fails
func (s *Supervisor) Supervise(process *ProcessInfo, isShuttingDown func() bool) {
	s.logger.Debug("Starting supervision for service", zap.String("service", process.Name))
	
	if process == nil || process.Command == nil {
		s.logger.Error("Invalid process state for supervision", 
			zap.String("service", process.Name))
		return
	}
	
	// Mark as running
	process.State = types.ProcessStateRunning
	
	// Wait for either process exit or stop signal
	exitChan := make(chan error, 1)
	go func() {
		exitErr := process.Command.Wait()
		exitChan <- exitErr
	}()
	
	// Wait for exit or stop signal
	var exitErr error
	select {
	case exitErr = <-exitChan:
		// Process exited on its own
	case <-process.StopCh:
		// Stop requested, return immediately
		return
	}
	
	// Check if shutting down
	if isShuttingDown() {
		s.logger.Info("Service exited during shutdown",
			zap.String("service", process.Name))
		return
	}
	
	// Process exited unexpectedly
	exitCode := 0
	if exitErr != nil {
		if exitError, ok := exitErr.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}
	
	s.logger.Warn("Service exited unexpectedly",
		zap.String("service", process.Name),
		zap.Int("exit_code", exitCode),
		zap.Error(exitErr))
	
	// Update status to failed and store error
	process.State = types.ProcessStateFailed
	process.LastError = fmt.Errorf("service exited with code %d: %w", exitCode, exitErr)
	
	// Check if restart is enabled
	if !s.config.AutoRestart {
		s.logger.Info("Auto-restart disabled, not restarting service",
			zap.String("service", process.Name))
		return
	}
	
	// Check if maximum restart limit is reached
	if process.Restarts >= s.config.MaxRestartAttempts {
		s.logger.Error("Service reached maximum restart attempts, not restarting",
			zap.String("service", process.Name),
			zap.Int("restarts", process.Restarts))
		return
	}
	
	// Calculate exponential backoff
	backoffDelay := CalculateBackoffDelay(process.Restarts, s.config.InitialBackoffMs, s.config.MaxBackoffMs)
	
	s.logger.Info("Restarting service after backoff",
		zap.String("service", process.Name),
		zap.Duration("backoff", backoffDelay),
		zap.Int("restart_count", process.Restarts))
		
	// Wait for backoff period or stop signal
	select {
	case <-time.After(backoffDelay):
		// Backoff completed, restart service
	case <-process.StopCh:
		// Stop requested during backoff, exit
		return
	}
	
	// Restart the service
	if err := s.spawner.SpawnProcess(process.Name); err != nil {
		s.logger.Error("Failed to restart service",
			zap.String("service", process.Name),
			zap.Error(err))
	}
}

// CalculateBackoffDelay implements exponential backoff with jitter
func CalculateBackoffDelay(restartCount, initialDelay, maxDelay int) time.Duration {
	// Calculate exponential backoff
	delayMs := math.Min(
		float64(initialDelay) * math.Pow(2, float64(restartCount)),
		float64(maxDelay),
	)
	
	// Add jitter (± 10%)
	jitterFactor := 0.9 + (0.2 * rand.Float64())
	delayMs = delayMs * jitterFactor
	
	return time.Duration(delayMs) * time.Millisecond
}