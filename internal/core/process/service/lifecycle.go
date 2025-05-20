// Package service provides service lifecycle management for the Process Orchestrator.
// It handles starting, stopping, and restarting services.
package service

import (
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/handcraftdev/blackhole/internal/core/config"
	"github.com/handcraftdev/blackhole/internal/core/process/types"
	"go.uber.org/zap"
)

// Manager handles service lifecycle operations
type Manager struct {
	services    map[string]*config.ServiceConfig
	processes   map[string]*types.ServiceInfo
	processLock *sync.RWMutex
	logger      *zap.Logger
}

// NewManager creates a new service lifecycle manager
func NewManager(
	services map[string]*config.ServiceConfig, 
	processes map[string]*types.ServiceInfo,
	processLock *sync.RWMutex,
	logger *zap.Logger,
) *Manager {
	return &Manager{
		services:    services,
		processes:   processes,
		processLock: processLock,
		logger:      logger,
	}
}

// StartService initiates a service by checking its configuration and status
func (m *Manager) StartService(name string, spawnFn func(string) error) error {
	m.processLock.RLock()
	// Get service configuration 
	serviceCfg, exists := m.services[name]
	if !exists {
		m.processLock.RUnlock()
		return fmt.Errorf("no configuration found for service %s", name)
	}
	
	// Skip disabled services
	if !serviceCfg.Enabled {
		m.logger.Info("Skipping disabled service", zap.String("service", name))
		m.processLock.RUnlock()
		return nil
	}
	
	// Check if service is already running
	process, exists := m.processes[name]
	m.processLock.RUnlock()
	
	if exists && process.State == string(types.ProcessStateRunning) {
		m.logger.Info("Service already running", zap.String("service", name))
		return nil
	}
	
	// Spawn the service process
	return spawnFn(name)
}

// StopService gracefully stops a running service
func (m *Manager) StopService(
	name string, 
	signalFn func(string, syscall.Signal) error,
	shutdownTimeout int,
) error {
	// Find process in process map
	m.processLock.Lock()
	process, exists := m.processes[name]
	if !exists {
		m.processLock.Unlock()
		return fmt.Errorf("service %s not found", name)
	}
	
	// Check if already stopped
	if process.State == string(types.ProcessStateStopped) {
		m.processLock.Unlock()
		return nil
	}
	
	// Update status and get process info
	process.State = string(types.ProcessStateStopped)
	stopCh := process.StopCh
	pid := process.PID
	m.processLock.Unlock()
	
	// Signal supervision goroutine to stop
	if stopCh != nil {
		close(stopCh)
	}
	
	m.logger.Info("Sending SIGTERM to service", 
		zap.String("service", name),
		zap.Int("pid", pid))
	
	// Send SIGTERM for graceful shutdown
	if err := signalFn(name, syscall.SIGTERM); err != nil {
		m.logger.Warn("Failed to send SIGTERM", 
			zap.String("service", name),
			zap.Error(err))
	}
	
	// Wait for process to exit with timeout
	exitChan := make(chan error, 1)
	go func() {
		m.processLock.RLock()
		process, exists := m.processes[name]
		if !exists || process.CommandWait == nil {
			m.processLock.RUnlock()
			exitChan <- nil
			return
		}
		waitFn := process.CommandWait
		m.processLock.RUnlock()
		
		exitErr := waitFn()
		exitChan <- exitErr
	}()
	
	// Wait for exit or timeout
	select {
	case err := <-exitChan:
		if err != nil {
			m.logger.Warn("Error waiting for service to exit",
				zap.String("service", name),
				zap.Error(err))
		}
	case <-time.After(time.Duration(shutdownTimeout) * time.Second):
		// Timeout occurred, force kill
		m.logger.Warn("Service did not exit gracefully, sending SIGKILL",
			zap.String("service", name),
			zap.Int("pid", pid))
		
		if err := signalFn(name, syscall.SIGKILL); err != nil {
			m.logger.Error("Failed to send SIGKILL",
				zap.String("service", name),
				zap.Error(err))
			return fmt.Errorf("failed to forcefully terminate service %s: %w", name, err)
		}
	}
	
	m.logger.Info("Service stopped", zap.String("service", name))
	return nil
}

// RestartService restarts a running service
func (m *Manager) RestartService(
	name string, 
	stopFn func(string) error,
	startFn func(string) error,
) error {
	m.logger.Info("Restarting service", zap.String("service", name))
	
	m.processLock.Lock()
	process, exists := m.processes[name]
	if exists {
		process.State = string(types.ProcessStateRestarting)
	}
	m.processLock.Unlock()
	
	// Stop the service
	if err := stopFn(name); err != nil {
		// Log but continue, as we'll try to start anyway
		m.logger.Warn("Error stopping service during restart",
			zap.String("service", name),
			zap.Error(err))
	}
	
	// Start the service
	return startFn(name)
}