// Package service provides service lifecycle management for the Process Orchestrator.
// This file contains functionality for retrieving service status and information.
package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/handcraftdev/blackhole/internal/core/config/types"
	processtypes "github.com/handcraftdev/blackhole/internal/core/process/types"
)

// InfoProvider retrieves information about services
type InfoProvider struct {
	services    map[string]*types.ServiceConfig
	processes   map[string]*ServiceProcess
	processLock *sync.RWMutex
}

// ServiceProcess represents a running service process with state management
type ServiceProcess struct {
	Name        string
	Command     processtypes.ProcessCmd
	CommandWait func() error
	PID         int
	State       processtypes.ProcessState
	Started     time.Time
	Restarts    int
	LastError   error
	StopCh      chan struct{}
}

// NewInfoProvider creates a new service information provider
func NewInfoProvider(
	services map[string]*types.ServiceConfig, 
	processes map[string]*ServiceProcess,
	processLock *sync.RWMutex,
) *InfoProvider {
	return &InfoProvider{
		services:    services,
		processes:   processes,
		processLock: processLock,
	}
}

// GetServiceInfo returns diagnostic information about a specific service
func (p *InfoProvider) GetServiceInfo(name string) (*processtypes.ServiceInfo, error) {
	p.processLock.RLock()
	defer p.processLock.RUnlock()
	
	// Check if service is configured
	serviceCfg, exists := p.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not configured", name)
	}
	
	// Get process info if running
	process, exists := p.processes[name]
	if !exists {
		return &processtypes.ServiceInfo{
			Name:       name,
			Configured: true,
			Enabled:    serviceCfg.Enabled,
			State:      string(processtypes.ProcessStateStopped),
		}, nil
	}
	
	// Build service info
	info := &processtypes.ServiceInfo{
		Name:       name,
		Configured: true,
		Enabled:    serviceCfg.Enabled,
		State:      string(process.State),
		PID:        process.PID,
		Uptime:     time.Since(process.Started),
		Restarts:   process.Restarts,
	}
	
	// Add error info if available
	if process.LastError != nil {
		info.LastError = process.LastError.Error()
	}
	
	return info, nil
}

// GetAllServices returns information about all configured services
func (p *InfoProvider) GetAllServices() (map[string]*processtypes.ServiceInfo, error) {
	p.processLock.RLock()
	defer p.processLock.RUnlock()
	
	services := make(map[string]*processtypes.ServiceInfo)
	
	// Add all configured services
	for name, cfg := range p.services {
		info := &processtypes.ServiceInfo{
			Name:       name,
			Configured: true,
			Enabled:    cfg.Enabled,
			State:      string(processtypes.ProcessStateStopped),
		}
		
		// Add process info if running
		if process, exists := p.processes[name]; exists {
			info.State = string(process.State)
			info.PID = process.PID
			info.Uptime = time.Since(process.Started)
			info.Restarts = process.Restarts
			
			// Add error info if available
			if process.LastError != nil {
				info.LastError = process.LastError.Error()
			}
		}
		
		services[name] = info
	}
	
	return services, nil
}