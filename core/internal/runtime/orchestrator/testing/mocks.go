// Package testing provides test utilities and mocks for the process package
package testing

import (
	"io"
	"os"
	"sync"

	"github.com/blackhole-pro/blackhole/core/internal/runtime/orchestrator/types"
)

// MockProcessExecutor is a mock implementation of the ProcessExecutor interface
type MockProcessExecutor struct {
	CommandFunc func(path string, args ...string) types.ProcessCmd
}

// Command calls the mock function
func (m *MockProcessExecutor) Command(path string, args ...string) types.ProcessCmd {
	if m.CommandFunc != nil {
		return m.CommandFunc(path, args...)
	}
	return &MockProcessCmd{}
}

// MockProcessCmd is a mock implementation of the ProcessCmd interface
type MockProcessCmd struct {
	StartFunc   func() error
	WaitFunc    func() error
	SetEnvFunc  func(env []string)
	SetDirFunc  func(dir string)
	SetOutputFunc func(stdout, stderr io.Writer)
	SignalFunc  func(sig os.Signal) error
	ProcessFunc func() types.Process
	
	Env         []string
	Dir         string
	Stdout      io.Writer
	Stderr      io.Writer
}

// Start calls the mock function
func (m *MockProcessCmd) Start() error {
	if m.StartFunc != nil {
		return m.StartFunc()
	}
	return nil
}

// Wait calls the mock function
func (m *MockProcessCmd) Wait() error {
	if m.WaitFunc != nil {
		return m.WaitFunc()
	}
	return nil
}

// SetEnv calls the mock function
func (m *MockProcessCmd) SetEnv(env []string) {
	if m.SetEnvFunc != nil {
		m.SetEnvFunc(env)
		return
	}
	m.Env = env
}

// SetDir calls the mock function
func (m *MockProcessCmd) SetDir(dir string) {
	if m.SetDirFunc != nil {
		m.SetDirFunc(dir)
		return
	}
	m.Dir = dir
}

// SetOutput calls the mock function
func (m *MockProcessCmd) SetOutput(stdout, stderr io.Writer) {
	if m.SetOutputFunc != nil {
		m.SetOutputFunc(stdout, stderr)
		return
	}
	m.Stdout = stdout
	m.Stderr = stderr
}

// Signal calls the mock function
func (m *MockProcessCmd) Signal(sig os.Signal) error {
	if m.SignalFunc != nil {
		return m.SignalFunc(sig)
	}
	return nil
}

// Process calls the mock function
func (m *MockProcessCmd) Process() types.Process {
	if m.ProcessFunc != nil {
		return m.ProcessFunc()
	}
	return &MockProcess{}
}

// MockProcess is a mock implementation of the Process interface
type MockProcess struct {
	PidFunc  func() int
	KillFunc func() error
	
	pid      int
	killErr  error
	mu       sync.Mutex
}

// Pid calls the mock function
func (m *MockProcess) Pid() int {
	if m.PidFunc != nil {
		return m.PidFunc()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pid
}

// Kill calls the mock function
func (m *MockProcess) Kill() error {
	if m.KillFunc != nil {
		return m.KillFunc()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.killErr
}

// SetPid sets the process ID for testing
func (m *MockProcess) SetPid(pid int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pid = pid
}

// SetKillError sets the error to return from Kill for testing
func (m *MockProcess) SetKillError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.killErr = err
}

// NewMockExecutor creates a new MockProcessExecutor with default settings
func NewMockExecutor() *MockProcessExecutor {
	return &MockProcessExecutor{
		CommandFunc: func(path string, args ...string) types.ProcessCmd {
			return &MockProcessCmd{}
		},
	}
}