package process

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// DiscoverServices finds all service binaries in the services directory
func (o *Orchestrator) DiscoverServices() ([]string, error) {
	var services []string
	
	// Read services directory
	entries, err := os.ReadDir(o.config.ServicesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read services directory: %w", err)
	}
	
	// Check each entry for service binary
	for _, entry := range entries {
		if entry.IsDir() {
			serviceName := entry.Name()
			serviceBinaryPath := filepath.Join(o.config.ServicesDir, serviceName, serviceName)
			
			// Check if binary exists and is executable
			if fileExists(serviceBinaryPath) && isExecutable(serviceBinaryPath) {
				services = append(services, serviceName)
				o.logger.Debug("Discovered service binary",
					zap.String("service", serviceName),
					zap.String("path", serviceBinaryPath))
			}
		}
	}
	
	if len(services) == 0 {
		o.logger.Warn("No service binaries found in services directory",
			zap.String("directory", o.config.ServicesDir))
	} else {
		o.logger.Info("Discovered service binaries",
			zap.Int("count", len(services)),
			zap.Strings("services", services))
	}
	
	return services, nil
}

// RefreshServices re-discovers services in the services directory
func (o *Orchestrator) RefreshServices() ([]string, error) {
	services, err := o.DiscoverServices()
	if err != nil {
		return nil, err
	}
	
	o.processLock.Lock()
	defer o.processLock.Unlock()
	
	// Log any newly discovered services
	var newServices []string
	for _, serviceName := range services {
		if _, exists := o.services[serviceName]; !exists {
			newServices = append(newServices, serviceName)
			
			// Add to services map with default configuration
			o.services[serviceName] = &ServiceConfig{
				Enabled:    true,
				DataDir:    filepath.Join(o.config.ServicesDir, serviceName, "data"),
				BinaryPath: filepath.Join(o.config.ServicesDir, serviceName, serviceName),
			}
		}
	}
	
	if len(newServices) > 0 {
		o.logger.Info("Discovered new services",
			zap.Strings("services", newServices))
	}
	
	return services, nil
}

// GetServiceBinaryPath returns the binary path for a service
func (o *Orchestrator) GetServiceBinaryPath(name string) (string, error) {
	o.processLock.RLock()
	defer o.processLock.RUnlock()
	
	// Check if service is configured
	serviceCfg, exists := o.services[name]
	if !exists {
		return "", fmt.Errorf("service %s not configured", name)
	}
	
	// Use configured binary path if available
	if serviceCfg.BinaryPath != "" {
		return serviceCfg.BinaryPath, nil
	}
	
	// Use default path: servicesDir/name/name
	defaultPath := filepath.Join(o.config.ServicesDir, name, name)
	
	// Verify it exists and is executable
	if !fileExists(defaultPath) {
		return "", fmt.Errorf("service binary not found at %s", defaultPath)
	}
	
	if !isExecutable(defaultPath) {
		return "", fmt.Errorf("service binary is not executable: %s", defaultPath)
	}
	
	return defaultPath, nil
}

// GetServiceInfo returns diagnostic information about a service
func (o *Orchestrator) GetServiceInfo(name string) (*ServiceInfo, error) {
	o.processLock.RLock()
	defer o.processLock.RUnlock()
	
	// Check if service is configured
	serviceCfg, exists := o.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not configured", name)
	}
	
	// Get process info if running
	process, exists := o.processes[name]
	if !exists {
		return &ServiceInfo{
			Name:       name,
			Configured: true,
			Enabled:    serviceCfg.Enabled,
			State:      string(ProcessStateStopped),
		}, nil
	}
	
	// Calculate uptime in seconds
	var uptime int64
	if !process.Started.IsZero() {
		uptime = int64(time.Since(process.Started).Seconds())
	}
	
	// Build service info
	info := &ServiceInfo{
		Name:       name,
		Configured: true,
		Enabled:    serviceCfg.Enabled,
		State:      string(process.State),
		PID:        process.PID,
		Uptime:     uptime,
		Restarts:   process.Restarts,
	}
	
	// Add error info if available
	if process.LastError != nil {
		if procErr, ok := process.LastError.(*ProcessError); ok {
			info.LastExitCode = procErr.ExitCode
			info.LastError = procErr.Error()
		} else {
			info.LastError = process.LastError.Error()
		}
	}
	
	return info, nil
}

// GetAllServices returns information about all services
func (o *Orchestrator) GetAllServices() (map[string]*ServiceInfo, error) {
	o.processLock.RLock()
	defer o.processLock.RUnlock()
	
	services := make(map[string]*ServiceInfo)
	
	// Add all configured services
	for name, cfg := range o.services {
		// Calculate uptime in seconds
		var uptime int64
		
		info := &ServiceInfo{
			Name:       name,
			Configured: true,
			Enabled:    cfg.Enabled,
			State:      string(ProcessStateStopped),
		}
		
		// Add process info if running
		if process, exists := o.processes[name]; exists {
			info.State = string(process.State)
			info.PID = process.PID
			
			if !process.Started.IsZero() {
				info.Uptime = int64(time.Since(process.Started).Seconds())
			}
			
			info.Restarts = process.Restarts
			
			// Add error info if available
			if process.LastError != nil {
				if procErr, ok := process.LastError.(*ProcessError); ok {
					info.LastExitCode = procErr.ExitCode
					info.LastError = procErr.Error()
				} else {
					info.LastError = process.LastError.Error()
				}
			}
		}
		
		services[name] = info
	}
	
	return services, nil
}

// Status returns the current status of a service
func (o *Orchestrator) Status(name string) (ProcessState, error) {
	o.processLock.RLock()
	defer o.processLock.RUnlock()
	
	// Check if service is configured
	if _, exists := o.services[name]; !exists {
		return "", fmt.Errorf("service %s not configured", name)
	}
	
	// Get process info if running
	process, exists := o.processes[name]
	if !exists {
		return ProcessStateStopped, nil
	}
	
	return process.State, nil
}

// IsRunning checks if a service is running
func (o *Orchestrator) IsRunning(name string) bool {
	o.processLock.RLock()
	defer o.processLock.RUnlock()
	
	// Get process info if running
	process, exists := o.processes[name]
	if !exists {
		return false
	}
	
	return process.State == ProcessStateRunning
}