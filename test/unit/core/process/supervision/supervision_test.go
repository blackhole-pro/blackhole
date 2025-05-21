// Package supervision_test provides tests for the process supervision functionality.
package supervision_test

import (
	"errors"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/handcraftdev/blackhole/internal/core/process/supervision"
	"github.com/handcraftdev/blackhole/internal/core/process/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// MockProcessCmd implements types.ProcessCmd for testing
type MockProcessCmd struct {
	StartFn   func() error
	WaitFn    func() error
	ProcessFn func() types.Process
	SignalFn  func(sig os.Signal) error
	
	// Other methods not used in tests
	SetEnvFn    func(env []string)
	SetDirFn    func(dir string)
	SetOutputFn func(stdout, stderr io.Writer)
}

func (m *MockProcessCmd) Start() error {
	if m.StartFn != nil {
		return m.StartFn()
	}
	return nil
}

func (m *MockProcessCmd) Wait() error {
	if m.WaitFn != nil {
		return m.WaitFn()
	}
	return nil
}

func (m *MockProcessCmd) Process() types.Process {
	if m.ProcessFn != nil {
		return m.ProcessFn()
	}
	return nil
}

func (m *MockProcessCmd) Signal(sig os.Signal) error {
	if m.SignalFn != nil {
		return m.SignalFn(sig)
	}
	return nil
}

func (m *MockProcessCmd) SetEnv(env []string) {
	if m.SetEnvFn != nil {
		m.SetEnvFn(env)
	}
}

func (m *MockProcessCmd) SetDir(dir string) {
	if m.SetDirFn != nil {
		m.SetDirFn(dir)
	}
}

func (m *MockProcessCmd) SetOutput(stdout, stderr io.Writer) {
	if m.SetOutputFn != nil {
		m.SetOutputFn(stdout, stderr)
	}
}

// MockProcess implements types.Process for testing
type MockProcess struct {
	PidFn  func() int
	KillFn func() error
}

func (m *MockProcess) Pid() int {
	if m.PidFn != nil {
		return m.PidFn()
	}
	return 0
}

func (m *MockProcess) Kill() error {
	if m.KillFn != nil {
		return m.KillFn()
	}
	return nil
}

// MockProcessSpawner implements ProcessSpawner for testing
type MockProcessSpawner struct {
	mu            sync.Mutex
	SpawnFn       func(string) error
	SpawnedCount  int
	LastSpawned   string
	SpawnedReturn error
}

func (m *MockProcessSpawner) SpawnProcess(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.SpawnedCount++
	m.LastSpawned = name
	
	if m.SpawnFn != nil {
		return m.SpawnFn(name)
	}
	return m.SpawnedReturn
}

// TestSupervisor_Supervise tests the Supervise method
func TestSupervisor_Supervise(t *testing.T) {
	t.Run("Process exits successfully", func(t *testing.T) {
		// Create a logger
		logger := zaptest.NewLogger(t)
		
		// Create mock spawner
		spawner := &MockProcessSpawner{}
		
		// Create supervisor
		supervisor := supervision.NewSupervisor(spawner, supervision.SupervisorConfig{}, logger)
		
		// Create mock command that exits without error
		cmd := &MockProcessCmd{
			WaitFn: func() error {
				return nil // Process exits normally
			},
		}
		
		// Create process info
		processInfo := &supervision.ProcessInfo{
			Name:     "test-service",
			Command:  cmd,
			State:    types.ProcessStateStarting,
			PID:      1000,
			Restarts: 0,
			StopCh:   make(chan struct{}),
			Started:  time.Now(),
		}
		
		// Create a shutdown flag function
		isShuttingDown := func() bool {
			return false
		}
		
		// Run supervise in a goroutine
		done := make(chan struct{})
		go func() {
			supervisor.Supervise(processInfo, isShuttingDown)
			close(done)
		}()
		
		// Wait for supervision to complete
		select {
		case <-done:
			// Good, it should complete
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Supervision did not complete in time")
		}
		
		// Verify state was updated to running
		assert.Equal(t, types.ProcessStateRunning, processInfo.State)
		
		// Verify spawner was not called (no restart needed)
		assert.Equal(t, 0, spawner.SpawnedCount)
	})
	
	t.Run("Process fails and gets restarted", func(t *testing.T) {
		// Create a logger
		logger := zaptest.NewLogger(t)
		
		// Create mock spawner
		spawner := &MockProcessSpawner{}
		
		// Create supervisor with auto-restart enabled
		supervisor := supervision.NewSupervisor(spawner, supervision.SupervisorConfig{
			AutoRestart:       true,
			MaxRestartAttempts: 3,
			InitialBackoffMs:  1, // Very short for testing
			MaxBackoffMs:      10,
		}, logger)
		
		// Create mock command that fails
		cmd := &MockProcessCmd{
			WaitFn: func() error {
				return errors.New("process failed")
			},
		}
		
		// Create process info
		processInfo := &supervision.ProcessInfo{
			Name:     "test-service",
			Command:  cmd,
			State:    types.ProcessStateStarting,
			PID:      1000,
			Restarts: 0,
			StopCh:   make(chan struct{}),
			Started:  time.Now(),
		}
		
		// Create a shutdown flag function
		isShuttingDown := func() bool {
			return false
		}
		
		// Run supervise in a goroutine
		done := make(chan struct{})
		go func() {
			supervisor.Supervise(processInfo, isShuttingDown)
			close(done)
		}()
		
		// Wait for restart
		time.Sleep(50 * time.Millisecond)
		
		// Verify state was updated to failed
		assert.Equal(t, types.ProcessStateFailed, processInfo.State)
		
		// Verify spawner was called (restart happened)
		assert.Equal(t, 1, spawner.SpawnedCount)
		assert.Equal(t, "test-service", spawner.LastSpawned)
		
		// Verify error was stored
		assert.NotNil(t, processInfo.LastError)
		assert.Contains(t, processInfo.LastError.Error(), "process failed")
		
		// Close stop channel to end supervision
		close(processInfo.StopCh)
		
		// Wait for supervision to complete
		select {
		case <-done:
			// Good, it should complete
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Supervision did not complete in time")
		}
	})
	
	t.Run("Process fails during shutdown", func(t *testing.T) {
		// Create a logger
		logger := zaptest.NewLogger(t)
		
		// Create mock spawner
		spawner := &MockProcessSpawner{}
		
		// Create supervisor
		supervisor := supervision.NewSupervisor(spawner, supervision.SupervisorConfig{
			AutoRestart: true,
		}, logger)
		
		// Create mock command that fails
		cmd := &MockProcessCmd{
			WaitFn: func() error {
				return errors.New("process failed")
			},
		}
		
		// Create process info
		processInfo := &supervision.ProcessInfo{
			Name:     "test-service",
			Command:  cmd,
			State:    types.ProcessStateStarting,
			PID:      1000,
			Restarts: 0,
			StopCh:   make(chan struct{}),
			Started:  time.Now(),
		}
		
		// Create a shutdown flag function that returns true
		isShuttingDown := func() bool {
			return true // System is shutting down
		}
		
		// Run supervise in a goroutine
		done := make(chan struct{})
		go func() {
			supervisor.Supervise(processInfo, isShuttingDown)
			close(done)
		}()
		
		// Wait for supervision to complete
		select {
		case <-done:
			// Good, it should complete
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Supervision did not complete in time")
		}
		
		// Verify spawner was not called (no restart during shutdown)
		assert.Equal(t, 0, spawner.SpawnedCount)
	})
	
	t.Run("Maximum restart attempts exceeded", func(t *testing.T) {
		// Create a logger
		logger := zaptest.NewLogger(t)
		
		// Create mock spawner
		spawner := &MockProcessSpawner{}
		
		// Create supervisor with limited restart attempts
		supervisor := supervision.NewSupervisor(spawner, supervision.SupervisorConfig{
			AutoRestart:       true,
			MaxRestartAttempts: 2, // Only allow 2 restarts
			InitialBackoffMs:  1,  // Very short for testing
			MaxBackoffMs:      10,
		}, logger)
		
		// Create mock command that fails
		cmd := &MockProcessCmd{
			WaitFn: func() error {
				return errors.New("process failed")
			},
		}
		
		// Create process info that already has restarts
		processInfo := &supervision.ProcessInfo{
			Name:     "test-service",
			Command:  cmd,
			State:    types.ProcessStateStarting,
			PID:      1000,
			Restarts: 2, // Already at max
			StopCh:   make(chan struct{}),
			Started:  time.Now(),
		}
		
		// Create a shutdown flag function
		isShuttingDown := func() bool {
			return false
		}
		
		// Run supervise in a goroutine
		done := make(chan struct{})
		go func() {
			supervisor.Supervise(processInfo, isShuttingDown)
			close(done)
		}()
		
		// Wait for supervision to complete
		select {
		case <-done:
			// Good, it should complete without restarting
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Supervision did not complete in time")
		}
		
		// Verify state was updated to failed
		assert.Equal(t, types.ProcessStateFailed, processInfo.State)
		
		// Verify spawner was not called (max restarts exceeded)
		assert.Equal(t, 0, spawner.SpawnedCount)
	})
	
	t.Run("Process stopped by stop channel", func(t *testing.T) {
		// Create a logger
		logger := zaptest.NewLogger(t)
		
		// Create mock spawner
		spawner := &MockProcessSpawner{}
		
		// Create supervisor
		supervisor := supervision.NewSupervisor(spawner, supervision.SupervisorConfig{}, logger)
		
		// Create a channel for the wait function
		waitCh := make(chan struct{})
		
		// Create mock command with a wait function that blocks
		cmd := &MockProcessCmd{
			WaitFn: func() error {
				<-waitCh
				return nil
			},
		}
		
		// Create process info
		processInfo := &supervision.ProcessInfo{
			Name:     "test-service",
			Command:  cmd,
			State:    types.ProcessStateStarting,
			PID:      1000,
			Restarts: 0,
			StopCh:   make(chan struct{}),
			Started:  time.Now(),
		}
		
		// Create a shutdown flag function
		isShuttingDown := func() bool {
			return false
		}
		
		// Run supervise in a goroutine
		done := make(chan struct{})
		go func() {
			supervisor.Supervise(processInfo, isShuttingDown)
			close(done)
		}()
		
		// Wait a bit for supervision to start
		time.Sleep(20 * time.Millisecond)
		
		// Close stop channel to signal stop
		close(processInfo.StopCh)
		
		// Wait for supervision to complete
		select {
		case <-done:
			// Good, it should complete
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Supervision did not complete in time")
		}
		
		// Clean up
		close(waitCh)
		
		// Verify spawner was not called
		assert.Equal(t, 0, spawner.SpawnedCount)
	})
}

// TestCalculateBackoffDelay tests the backoff calculation function
func TestCalculateBackoffDelay(t *testing.T) {
	t.Run("Initial backoff", func(t *testing.T) {
		delay := supervision.CalculateBackoffDelay(0, 1000, 30000)
		// Account for jitter (±10%)
		assert.GreaterOrEqual(t, delay.Milliseconds(), int64(900))
		assert.LessOrEqual(t, delay.Milliseconds(), int64(1100))
	})
	
	t.Run("Exponential backoff", func(t *testing.T) {
		// After 3 restarts: 1000 * 2^3 = 8000ms (±10%)
		delay := supervision.CalculateBackoffDelay(3, 1000, 30000)
		assert.GreaterOrEqual(t, delay.Milliseconds(), int64(7200))
		assert.LessOrEqual(t, delay.Milliseconds(), int64(8800))
	})
	
	t.Run("Maximum backoff cap", func(t *testing.T) {
		// After 10 restarts, would be 1000 * 2^10 = 1,024,000ms, but capped at 30000ms
		delay := supervision.CalculateBackoffDelay(10, 1000, 30000)
		assert.GreaterOrEqual(t, delay.Milliseconds(), int64(27000))
		assert.LessOrEqual(t, delay.Milliseconds(), int64(33000))
	})
	
	t.Run("Different initial values", func(t *testing.T) {
		delay := supervision.CalculateBackoffDelay(2, 500, 10000)
		// After 2 restarts: 500 * 2^2 = 2000ms (±10%)
		assert.GreaterOrEqual(t, delay.Milliseconds(), int64(1800))
		assert.LessOrEqual(t, delay.Milliseconds(), int64(2200))
	})
}

// TestNewSupervisor tests the supervisor constructor
func TestNewSupervisor(t *testing.T) {
	t.Run("Default configuration", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		spawner := &MockProcessSpawner{}
		
		supervisor := supervision.NewSupervisor(spawner, supervision.SupervisorConfig{}, logger)
		require.NotNil(t, supervisor)
		
		// Since we can't access private fields directly, we'll just verify the constructor doesn't panic
		assert.NotNil(t, supervisor)
	})
	
	t.Run("Custom configuration", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		spawner := &MockProcessSpawner{}
		
		config := supervision.SupervisorConfig{
			AutoRestart:       true,
			MaxRestartAttempts: 5,
			InitialBackoffMs:  2000,
			MaxBackoffMs:      20000,
		}
		
		supervisor := supervision.NewSupervisor(spawner, config, logger)
		require.NotNil(t, supervisor)
		
		// Since we can't access private fields directly, we'll just verify the constructor doesn't panic
		assert.NotNil(t, supervisor)
	})
}