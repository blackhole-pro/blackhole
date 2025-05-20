package process

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"blackhole/internal/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// MockProcessExecutor mocks the ProcessExecutor interface
type MockProcessExecutor struct {
	CommandFunc func(path string, args ...string) ProcessCmd
}

func (m *MockProcessExecutor) Command(path string, args ...string) ProcessCmd {
	if m.CommandFunc != nil {
		return m.CommandFunc(path, args...)
	}
	return &MockProcessCmd{}
}

// MockProcessCmd mocks the ProcessCmd interface
type MockProcessCmd struct {
	StartFunc   func() error
	WaitFunc    func() error
	ProcessFunc func() Process
	
	// Track method calls for verification
	StartCalled  bool
	WaitCalled   bool
	Env          []string
	Dir          string
	Stdout       io.Writer
	Stderr       io.Writer
}

func (m *MockProcessCmd) Start() error {
	m.StartCalled = true
	if m.StartFunc != nil {
		return m.StartFunc()
	}
	return nil
}

func (m *MockProcessCmd) Wait() error {
	m.WaitCalled = true
	if m.WaitFunc != nil {
		return m.WaitFunc()
	}
	return nil
}

func (m *MockProcessCmd) SetEnv(env []string) {
	m.Env = env
}

func (m *MockProcessCmd) SetDir(dir string) {
	m.Dir = dir
}

func (m *MockProcessCmd) SetOutput(stdout, stderr io.Writer) {
	m.Stdout = stdout
	m.Stderr = stderr
}

func (m *MockProcessCmd) Signal(sig os.Signal) error {
	return nil
}

func (m *MockProcessCmd) Process() Process {
	if m.ProcessFunc != nil {
		return m.ProcessFunc()
	}
	return &MockProcess{pid: 1000}
}

// MockProcess mocks the Process interface
type MockProcess struct {
	pid      int
	KillFunc func() error
}

func (m *MockProcess) Pid() int {
	return m.pid
}

func (m *MockProcess) Kill() error {
	if m.KillFunc != nil {
		return m.KillFunc()
	}
	return nil
}

// MockConfigManager mocks the ConfigManager interface
type MockConfigManager struct {
	Config   *config.Config
	GetFunc  func() *config.Config
	SetFunc  func(*config.Config) error
	SaveFunc func() error
}

func (m *MockConfigManager) Get() *config.Config {
	if m.GetFunc != nil {
		return m.GetFunc()
	}
	return m.Config
}

func (m *MockConfigManager) Set(cfg *config.Config) error {
	if m.SetFunc != nil {
		return m.SetFunc(cfg)
	}
	m.Config = cfg
	return nil
}

func (m *MockConfigManager) Save() error {
	if m.SaveFunc != nil {
		return m.SaveFunc()
	}
	return nil
}

// setupTestDir creates a temporary test directory
func setupTestDir(t *testing.T) string {
	tempDir := t.TempDir()
	
	// Create services dir
	servicesDir := filepath.Join(tempDir, "services")
	require.NoError(t, os.MkdirAll(servicesDir, 0755))
	
	// Create socket dir
	socketDir := filepath.Join(tempDir, "sockets")
	require.NoError(t, os.MkdirAll(socketDir, 0755))
	
	return tempDir
}

// createTestConfig creates a test configuration
func createTestConfig(t *testing.T, tempDir string) *config.Config {
	// Create config
	cfg := config.NewConfig()
	cfg.Storage.DataDir = tempDir
	
	// Set up service configurations
	cfg.Services.Identity.Enabled = true
	cfg.Services.Storage.Enabled = true
	cfg.Services.Ledger.Enabled = true
	
	return cfg
}

// TestNewOrchestrator tests the creation of a new orchestrator
func TestNewOrchestrator(t *testing.T) {
	// Setup test directory
	tempDir := setupTestDir(t)
	config := createTestConfig(t, tempDir)
	
	// Create test logger
	logger := zaptest.NewLogger(t)
	
	// Create orchestrator
	orch, err := NewOrchestrator(config, WithLogger(logger))
	
	// Verify orchestrator created successfully
	require.NoError(t, err)
	require.NotNil(t, orch)
	
	// Verify orchestrator properties
	assert.NotNil(t, orch.logger)
	assert.NotNil(t, orch.executor)
	assert.NotNil(t, orch.processes)
	assert.NotNil(t, orch.services)
	assert.NotNil(t, orch.doneCh)
	assert.NotNil(t, orch.sigCh)
	
	// Verify service configurations
	assert.Contains(t, orch.services, "identity")
	assert.Contains(t, orch.services, "storage")
	assert.Contains(t, orch.services, "ledger")
}

// TestStartService tests starting a service
func TestStartService(t *testing.T) {
	// Setup test directory
	tempDir := setupTestDir(t)
	config := createTestConfig(t, tempDir)
	
	// Create mock process executor
	mockCmd := &MockProcessCmd{
		StartFunc: func() error { return nil },
		ProcessFunc: func() Process {
			return &MockProcess{pid: 1000}
		},
	}
	
	mockExec := &MockProcessExecutor{
		CommandFunc: func(path string, args ...string) ProcessCmd {
			return mockCmd
		},
	}
	
	// Create orchestrator with mock executor
	orch, err := NewOrchestrator(config, WithExecutor(mockExec), WithLogger(zaptest.NewLogger(t)))
	require.NoError(t, err)
	
	// Create test service binary
	serviceName := "identity"
	serviceDir := filepath.Join(tempDir, "services", serviceName)
	require.NoError(t, os.MkdirAll(serviceDir, 0755))
	
	serviceBin := filepath.Join(serviceDir, serviceName)
	require.NoError(t, os.WriteFile(serviceBin, []byte("#!/bin/sh\necho test"), 0755))
	
	// Start service
	err = orch.StartService(serviceName)
	require.NoError(t, err)
	
	// Verify service was started
	assert.True(t, mockCmd.StartCalled)
	
	// Verify service state
	state, err := orch.Status(serviceName)
	require.NoError(t, err)
	assert.Equal(t, ProcessStateRunning, state)
	
	// Verify service info
	info, err := orch.GetServiceInfo(serviceName)
	require.NoError(t, err)
	assert.Equal(t, serviceName, info.Name)
	assert.Equal(t, string(ProcessStateRunning), info.State)
	assert.Equal(t, 1000, info.PID)
}

// TestStopService tests stopping a service
func TestStopService(t *testing.T) {
	// Setup test directory
	tempDir := setupTestDir(t)
	config := createTestConfig(t, tempDir)
	
	// Create wait channel to control test flow
	waitCh := make(chan struct{})
	
	// Create mock process executor
	mockCmd := &MockProcessCmd{
		StartFunc: func() error { return nil },
		WaitFunc: func() error {
			<-waitCh // Wait until signaled by test
			return nil
		},
		ProcessFunc: func() Process {
			return &MockProcess{pid: 1000}
		},
	}
	
	mockExec := &MockProcessExecutor{
		CommandFunc: func(path string, args ...string) ProcessCmd {
			return mockCmd
		},
	}
	
	// Create orchestrator with mock executor
	orch, err := NewOrchestrator(config, WithExecutor(mockExec), WithLogger(zaptest.NewLogger(t)))
	require.NoError(t, err)
	
	// Create test service binary
	serviceName := "identity"
	serviceDir := filepath.Join(tempDir, "services", serviceName)
	require.NoError(t, os.MkdirAll(serviceDir, 0755))
	
	serviceBin := filepath.Join(serviceDir, serviceName)
	require.NoError(t, os.WriteFile(serviceBin, []byte("#!/bin/sh\necho test"), 0755))
	
	// Start service
	err = orch.StartService(serviceName)
	require.NoError(t, err)
	
	// Start goroutine to stop service
	var stopErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		stopErr = orch.StopService(serviceName)
	}()
	
	// Let the stop process begin
	time.Sleep(50 * time.Millisecond)
	
	// Signal wait function to complete
	close(waitCh)
	
	// Wait for stop to complete
	wg.Wait()
	
	// Verify stop was successful
	require.NoError(t, stopErr)
	
	// Verify service state
	state, err := orch.Status(serviceName)
	require.NoError(t, err)
	assert.Equal(t, ProcessStateStopped, state)
}

// TestRestartService tests restarting a service
func TestRestartService(t *testing.T) {
	// Setup test directory
	tempDir := setupTestDir(t)
	config := createTestConfig(t, tempDir)
	
	// Create wait channel to control test flow
	waitCh := make(chan struct{})
	startCount := 0
	
	// Create mock process executor
	mockCmd := &MockProcessCmd{
		StartFunc: func() error {
			startCount++
			return nil 
		},
		WaitFunc: func() error {
			<-waitCh // Wait until signaled by test
			return nil
		},
		ProcessFunc: func() Process {
			return &MockProcess{pid: 1000}
		},
	}
	
	mockExec := &MockProcessExecutor{
		CommandFunc: func(path string, args ...string) ProcessCmd {
			return mockCmd
		},
	}
	
	// Create orchestrator with mock executor
	orch, err := NewOrchestrator(config, WithExecutor(mockExec), WithLogger(zaptest.NewLogger(t)))
	require.NoError(t, err)
	
	// Create test service binary
	serviceName := "identity"
	serviceDir := filepath.Join(tempDir, "services", serviceName)
	require.NoError(t, os.MkdirAll(serviceDir, 0755))
	
	serviceBin := filepath.Join(serviceDir, serviceName)
	require.NoError(t, os.WriteFile(serviceBin, []byte("#!/bin/sh\necho test"), 0755))
	
	// Start service
	err = orch.StartService(serviceName)
	require.NoError(t, err)
	
	// Wait for service to be marked as running
	time.Sleep(50 * time.Millisecond)
	
	// Create a new wait channel for restart
	oldWaitCh := waitCh
	waitCh = make(chan struct{})
	
	// Signal the original wait to complete
	close(oldWaitCh)
	
	// Start goroutine to restart service
	var restartErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		restartErr = orch.RestartService(serviceName)
	}()
	
	// Let the restart process begin
	time.Sleep(50 * time.Millisecond)
	
	// Signal wait function to complete for restart
	close(waitCh)
	
	// Wait for restart to complete
	wg.Wait()
	
	// Verify restart was successful
	require.NoError(t, restartErr)
	
	// Verify service state
	state, err := orch.Status(serviceName)
	require.NoError(t, err)
	assert.Equal(t, ProcessStateRunning, state)
	
	// Verify service was started twice (initial + restart)
	assert.Equal(t, 2, startCount)
	
	// Check restart count
	info, err := orch.GetServiceInfo(serviceName)
	require.NoError(t, err)
	assert.Equal(t, 1, info.Restarts)
}

// TestServiceFailureAndRestart tests service failure and automatic restart
func TestServiceFailureAndRestart(t *testing.T) {
	// Setup test directory
	tempDir := setupTestDir(t)
	config := createTestConfig(t, tempDir)
	
	// Create tracker for start count
	startCount := 0
	waitCh := make(chan struct{})
	
	// Create process mocks
	mockCmd := &MockProcessCmd{
		StartFunc: func() error {
			startCount++
			return nil 
		},
		WaitFunc: func() error {
			<-waitCh // Wait until signaled by test
			
			// Return error to simulate failure if not the first start
			if startCount > 1 {
				return nil
			}
			return errors.New("simulated process failure")
		},
		ProcessFunc: func() Process {
			return &MockProcess{pid: 1000}
		},
	}
	
	mockExec := &MockProcessExecutor{
		CommandFunc: func(path string, args ...string) ProcessCmd {
			return mockCmd
		},
	}
	
	// Create orchestrator with mock executor
	orch, err := NewOrchestrator(config, WithExecutor(mockExec), WithLogger(zaptest.NewLogger(t)))
	require.NoError(t, err)
	
	// Set auto-restart to true
	orch.config.AutoRestart = true
	
	// Reduce backoff delay for test
	oldCalcBackoff := calculateBackoffDelay
	calculateBackoffDelay = func(count int) time.Duration {
		return 10 * time.Millisecond // Very short backoff for tests
	}
	defer func() { calculateBackoffDelay = oldCalcBackoff }()
	
	// Create test service binary
	serviceName := "identity"
	serviceDir := filepath.Join(tempDir, "services", serviceName)
	require.NoError(t, os.MkdirAll(serviceDir, 0755))
	
	serviceBin := filepath.Join(serviceDir, serviceName)
	require.NoError(t, os.WriteFile(serviceBin, []byte("#!/bin/sh\necho test"), 0755))
	
	// Start service
	err = orch.StartService(serviceName)
	require.NoError(t, err)
	
	// Wait for service to be marked as running
	time.Sleep(50 * time.Millisecond)
	
	// Signal process to fail
	close(waitCh)
	
	// Create a new wait channel for restart
	waitCh = make(chan struct{})
	
	// Wait for restart to happen (should be very quick with our test backoff)
	time.Sleep(200 * time.Millisecond)
	
	// Verify service was restarted
	assert.Equal(t, 2, startCount)
	
	// Signal process to exit cleanly this time
	close(waitCh)
	
	// Check service info
	info, err := orch.GetServiceInfo(serviceName)
	require.NoError(t, err)
	assert.Equal(t, 1, info.Restarts)
	assert.NotEmpty(t, info.LastError) // Should have error from failure
}

// TestDiscoverServices tests service discovery
func TestDiscoverServices(t *testing.T) {
	// Setup test directory
	tempDir := setupTestDir(t)
	config := createTestConfig(t, tempDir)
	
	// Create orchestrator
	orch, err := NewOrchestrator(config, WithLogger(zaptest.NewLogger(t)))
	require.NoError(t, err)
	
	// Create test service binaries
	services := []string{"identity", "storage", "ledger"}
	for _, svc := range services {
		serviceDir := filepath.Join(tempDir, "services", svc)
		require.NoError(t, os.MkdirAll(serviceDir, 0755))
		
		serviceBin := filepath.Join(serviceDir, svc)
		require.NoError(t, os.WriteFile(serviceBin, []byte("#!/bin/sh\necho test"), 0755))
	}
	
	// Discover services
	discovered, err := orch.DiscoverServices()
	require.NoError(t, err)
	
	// Verify all services were discovered
	assert.Len(t, discovered, len(services))
	for _, svc := range services {
		assert.Contains(t, discovered, svc)
	}
	
	// Test RefreshServices
	newService := "telemetry"
	serviceDir := filepath.Join(tempDir, "services", newService)
	require.NoError(t, os.MkdirAll(serviceDir, 0755))
	
	serviceBin := filepath.Join(serviceDir, newService)
	require.NoError(t, os.WriteFile(serviceBin, []byte("#!/bin/sh\necho test"), 0755))
	
	// Refresh services
	refreshed, err := orch.RefreshServices()
	require.NoError(t, err)
	
	// Verify new service was discovered
	assert.Len(t, refreshed, len(services)+1)
	assert.Contains(t, refreshed, newService)
	
	// Verify service was added to config map
	assert.Contains(t, orch.services, newService)
}

// TestProcessIsolation tests process isolation functionality
func TestProcessIsolation(t *testing.T) {
	// Create mock command
	mockCmd := &MockProcessCmd{}
	
	// Create service config
	serviceCfg := &ServiceConfig{
		Enabled:     true,
		DataDir:     "/var/lib/test",
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
		MemoryLimit: 1024,
	}
	
	// Setup isolation
	setupProcessIsolation(mockCmd, serviceCfg)
	
	// Verify environment variables
	assert.Contains(t, mockCmd.Env, "PATH="+os.Getenv("PATH"))
	assert.Contains(t, mockCmd.Env, "HOME=/var/lib/test")
	assert.Contains(t, mockCmd.Env, "TEST_VAR=test_value")
	assert.Contains(t, mockCmd.Env, "GOMEMLIMIT=1024MiB")
	
	// Verify working directory
	assert.Equal(t, "/var/lib/test", mockCmd.Dir)
}

// TestOutputHandling tests process output handling
func TestOutputHandling(t *testing.T) {
	// Create test logger
	logger := zaptest.NewLogger(t)
	
	// Create prefixed log writer
	writer := newPrefixedLogWriter(logger, "test-service", false)
	
	// Write some output
	_, err := writer.Write([]byte("Test line 1\nTest line 2\nIncomplete"))
	require.NoError(t, err)
	
	// Write more output to complete the line
	_, err = writer.Write([]byte(" line 3\n"))
	require.NoError(t, err)
	
	// Create log buffer for testing
	buffer := NewLogBuffer("stdout", false, "test-service", logger)
	
	// Write some lines
	_, err = buffer.Write([]byte("Buffered line 1\nBuffered line 2\n"))
	require.NoError(t, err)
	
	// Get lines from buffer
	lines := buffer.GetLines()
	assert.Len(t, lines, 2)
	assert.Equal(t, "Buffered line 1", lines[0])
	assert.Equal(t, "Buffered line 2", lines[1])
	
	// Clear buffer
	buffer.Clear()
	assert.Empty(t, buffer.GetLines())
}