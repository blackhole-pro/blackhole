// Package testing provides test utilities and mocks for the app package
package testing

// This package contains mock implementations of interfaces defined in the app/types
// package. These mocks can be used to test code that depends on these interfaces
// without needing to use real implementations. Each mock tracks method calls and
// allows customizing behavior via function fields.

import (
	"sync"

	"github.com/handcraftdev/blackhole/internal/core/app/types"
	"go.uber.org/zap"
)

// MockService is a mock implementation of the Service interface
type MockService struct {
	StartFunc  func() error
	StopFunc   func() error
	NameFunc   func() string
	HealthFunc func() bool
	
	// State tracking
	name        string
	started     bool
	stopped     bool
	healthy     bool
	startCalled int
	stopCalled  int
	mu          sync.RWMutex
}

// Start implements the Service.Start method
func (m *MockService) Start() error {
	m.mu.Lock()
	m.startCalled++
	m.started = true
	m.mu.Unlock()
	
	if m.StartFunc != nil {
		return m.StartFunc()
	}
	return nil
}

// Stop implements the Service.Stop method
func (m *MockService) Stop() error {
	m.mu.Lock()
	m.stopCalled++
	m.started = false
	m.stopped = true
	m.mu.Unlock()
	
	if m.StopFunc != nil {
		return m.StopFunc()
	}
	return nil
}

// Name implements the Service.Name method
func (m *MockService) Name() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	
	if m.name != "" {
		return m.name
	}
	
	return "mock-service"
}

// Health implements the Service.Health method
func (m *MockService) Health() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.HealthFunc != nil {
		return m.HealthFunc()
	}
	
	return m.healthy
}

// NewMockService creates a new MockService with the given name
func NewMockService(name string) *MockService {
	return &MockService{
		name: name,
		healthy: true,
	}
}

// MockApplication is a mock implementation of the Application interface
type MockApplication struct {
	StartFunc             func() error
	StopFunc              func() error
	GetProcessManagerFunc func() types.ProcessManager
	GetConfigManagerFunc  func() types.ConfigManager
	RegisterServiceFunc   func(service types.Service) error
	GetServiceFunc        func(name string) (types.Service, bool)
	
	// State tracking
	services     map[string]types.Service
	started      bool
	stopped      bool
	startCalled  int
	stopCalled   int
	mu           sync.RWMutex
}

// Start implements the Application.Start method
func (m *MockApplication) Start() error {
	m.mu.Lock()
	m.startCalled++
	m.started = true
	m.mu.Unlock()
	
	if m.StartFunc != nil {
		return m.StartFunc()
	}
	return nil
}

// Stop implements the Application.Stop method
func (m *MockApplication) Stop() error {
	m.mu.Lock()
	m.stopCalled++
	m.started = false
	m.stopped = true
	m.mu.Unlock()
	
	if m.StopFunc != nil {
		return m.StopFunc()
	}
	return nil
}

// GetProcessManager implements the Application.GetProcessManager method
func (m *MockApplication) GetProcessManager() types.ProcessManager {
	if m.GetProcessManagerFunc != nil {
		return m.GetProcessManagerFunc()
	}
	return NewMockProcessManager()
}

// GetConfigManager implements the Application.GetConfigManager method
func (m *MockApplication) GetConfigManager() types.ConfigManager {
	if m.GetConfigManagerFunc != nil {
		return m.GetConfigManagerFunc()
	}
	return NewMockConfigManager()
}

// RegisterService implements the Application.RegisterService method
func (m *MockApplication) RegisterService(service types.Service) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.RegisterServiceFunc != nil {
		return m.RegisterServiceFunc(service)
	}
	
	if m.services == nil {
		m.services = make(map[string]types.Service)
	}
	
	m.services[service.Name()] = service
	return nil
}

// GetService implements the Application.GetService method
func (m *MockApplication) GetService(name string) (types.Service, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.GetServiceFunc != nil {
		return m.GetServiceFunc(name)
	}
	
	if m.services == nil {
		return nil, false
	}
	
	service, exists := m.services[name]
	return service, exists
}

// NewMockApplication creates a new MockApplication
func NewMockApplication() *MockApplication {
	return &MockApplication{
		services: make(map[string]types.Service),
	}
}

// MockProcessManager is a mock implementation of the ProcessManager interface
type MockProcessManager struct {
	StartFunc           func() error
	StopFunc            func() error
	StartAllFunc        func() error
	StopAllFunc         func() error
	StartServiceFunc    func(name string) error
	StopServiceFunc     func(name string) error
	RestartServiceFunc  func(name string) error
	GetServiceInfoFunc  func(name string) (types.ServiceInfo, error)
	DiscoverServicesFunc func() ([]string, error)
	RefreshServicesFunc  func() ([]string, error)
	
	// State tracking
	started         bool
	stopped         bool
	services        map[string]types.ServiceInfo
	startCalled     int
	stopCalled      int
	startAllCalled  int
	stopAllCalled   int
	mu              sync.RWMutex
}

// Start implements the ProcessManager.Start method
func (m *MockProcessManager) Start() error {
	m.mu.Lock()
	m.startCalled++
	m.started = true
	m.mu.Unlock()
	
	if m.StartFunc != nil {
		return m.StartFunc()
	}
	return nil
}

// Stop implements the ProcessManager.Stop method
func (m *MockProcessManager) Stop() error {
	m.mu.Lock()
	m.stopCalled++
	m.started = false
	m.stopped = true
	m.mu.Unlock()
	
	if m.StopFunc != nil {
		return m.StopFunc()
	}
	return nil
}

// StartAll implements the ProcessManager.StartAll method
func (m *MockProcessManager) StartAll() error {
	m.mu.Lock()
	m.startAllCalled++
	m.mu.Unlock()
	
	if m.StartAllFunc != nil {
		return m.StartAllFunc()
	}
	return nil
}

// StopAll implements the ProcessManager.StopAll method
func (m *MockProcessManager) StopAll() error {
	m.mu.Lock()
	m.stopAllCalled++
	m.mu.Unlock()
	
	if m.StopAllFunc != nil {
		return m.StopAllFunc()
	}
	return nil
}

// StartService implements the ProcessManager.StartService method
func (m *MockProcessManager) StartService(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.StartServiceFunc != nil {
		return m.StartServiceFunc(name)
	}
	
	if m.services == nil {
		m.services = make(map[string]types.ServiceInfo)
	}
	
	serviceInfo := m.services[name]
	serviceInfo.Name = name
	serviceInfo.Running = true
	m.services[name] = serviceInfo
	
	return nil
}

// StopService implements the ProcessManager.StopService method
func (m *MockProcessManager) StopService(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.StopServiceFunc != nil {
		return m.StopServiceFunc(name)
	}
	
	if m.services == nil {
		return types.ErrServiceNotFound
	}
	
	serviceInfo, exists := m.services[name]
	if !exists {
		return types.ErrServiceNotFound
	}
	
	serviceInfo.Running = false
	m.services[name] = serviceInfo
	
	return nil
}

// RestartService implements the ProcessManager.RestartService method
func (m *MockProcessManager) RestartService(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.RestartServiceFunc != nil {
		return m.RestartServiceFunc(name)
	}
	
	if m.services == nil {
		return types.ErrServiceNotFound
	}
	
	serviceInfo, exists := m.services[name]
	if !exists {
		return types.ErrServiceNotFound
	}
	
	serviceInfo.Running = true
	m.services[name] = serviceInfo
	
	return nil
}

// GetServiceInfo implements the ProcessManager.GetServiceInfo method
func (m *MockProcessManager) GetServiceInfo(name string) (types.ServiceInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.GetServiceInfoFunc != nil {
		return m.GetServiceInfoFunc(name)
	}
	
	if m.services == nil {
		return types.ServiceInfo{}, types.ErrServiceNotFound
	}
	
	serviceInfo, exists := m.services[name]
	if !exists {
		return types.ServiceInfo{}, types.ErrServiceNotFound
	}
	
	return serviceInfo, nil
}

// DiscoverServices implements the ProcessManager.DiscoverServices method
func (m *MockProcessManager) DiscoverServices() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.DiscoverServicesFunc != nil {
		return m.DiscoverServicesFunc()
	}
	
	if m.services == nil {
		return []string{}, nil
	}
	
	services := make([]string, 0, len(m.services))
	for name := range m.services {
		services = append(services, name)
	}
	
	return services, nil
}

// RefreshServices implements the ProcessManager.RefreshServices method
func (m *MockProcessManager) RefreshServices() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.RefreshServicesFunc != nil {
		return m.RefreshServicesFunc()
	}
	
	if m.services == nil {
		return []string{}, nil
	}
	
	services := make([]string, 0, len(m.services))
	for name := range m.services {
		services = append(services, name)
	}
	
	return services, nil
}

// NewMockProcessManager creates a new MockProcessManager
func NewMockProcessManager() *MockProcessManager {
	return &MockProcessManager{
		services: make(map[string]types.ServiceInfo),
	}
}

// MockConfigManager is a mock implementation of the ConfigManager interface
type MockConfigManager struct {
	GetConfigFunc       func() *types.Config
	SetConfigFunc       func(config *types.Config) error
	LoadFromFileFunc    func(path string) error
	SaveToFileFunc      func(path string) error
	SubscribeToChangesFunc func(callback func(*types.Config))
	
	// State tracking
	config        *types.Config
	subscribers   []func(*types.Config)
	loadCalled    int
	saveCalled    int
	mu            sync.RWMutex
}

// GetConfig implements the ConfigManager.GetConfig method
func (m *MockConfigManager) GetConfig() *types.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.GetConfigFunc != nil {
		return m.GetConfigFunc()
	}
	
	if m.config == nil {
		return &types.Config{}
	}
	
	return m.config
}

// SetConfig implements the ConfigManager.SetConfig method
func (m *MockConfigManager) SetConfig(config *types.Config) error {
	m.mu.Lock()
	m.config = config
	subscribers := make([]func(*types.Config), len(m.subscribers))
	copy(subscribers, m.subscribers)
	m.mu.Unlock()
	
	if m.SetConfigFunc != nil {
		return m.SetConfigFunc(config)
	}
	
	// Notify subscribers
	for _, subscriber := range subscribers {
		subscriber(config)
	}
	
	return nil
}

// LoadFromFile implements the ConfigManager.LoadFromFile method
func (m *MockConfigManager) LoadFromFile(path string) error {
	m.mu.Lock()
	m.loadCalled++
	m.mu.Unlock()
	
	if m.LoadFromFileFunc != nil {
		return m.LoadFromFileFunc(path)
	}
	
	return nil
}

// SaveToFile implements the ConfigManager.SaveToFile method
func (m *MockConfigManager) SaveToFile(path string) error {
	m.mu.Lock()
	m.saveCalled++
	m.mu.Unlock()
	
	if m.SaveToFileFunc != nil {
		return m.SaveToFileFunc(path)
	}
	
	return nil
}

// SubscribeToChanges implements the ConfigManager.SubscribeToChanges method
func (m *MockConfigManager) SubscribeToChanges(callback func(*types.Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.SubscribeToChangesFunc != nil {
		m.SubscribeToChangesFunc(callback)
		return
	}
	
	m.subscribers = append(m.subscribers, callback)
}

// NewMockConfigManager creates a new MockConfigManager
func NewMockConfigManager() *MockConfigManager {
	return &MockConfigManager{
		config: &types.Config{},
		subscribers: make([]func(*types.Config), 0),
	}
}

// MockProcessManagerFactory is a mock implementation of the ProcessManagerFactory interface
type MockProcessManagerFactory struct {
	// Function field to customize behavior
	CreateProcessManagerFunc func(configManager types.ConfigManager, logger *zap.Logger) (types.ProcessManager, error)
	
	// State tracking
	CreateCalled int
	LastConfig   types.ConfigManager
	LastLogger   *zap.Logger
	mu           sync.RWMutex
}

// CreateProcessManager implements the ProcessManagerFactory interface
func (m *MockProcessManagerFactory) CreateProcessManager(
	configManager types.ConfigManager,
	logger *zap.Logger,
) (types.ProcessManager, error) {
	m.mu.Lock()
	m.CreateCalled++
	m.LastConfig = configManager
	m.LastLogger = logger
	m.mu.Unlock()
	
	if m.CreateProcessManagerFunc != nil {
		return m.CreateProcessManagerFunc(configManager, logger)
	}
	
	// Return a default mock process manager
	return NewMockProcessManager(), nil
}

// NewMockProcessManagerFactory creates a new MockProcessManagerFactory
func NewMockProcessManagerFactory() *MockProcessManagerFactory {
	return &MockProcessManagerFactory{}
}