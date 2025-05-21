// Package config provides configuration management functionality for the
// Blackhole platform, including loading, validating, and updating configuration.
package config

import (
	"fmt"
	"sync"
	
	"github.com/handcraftdev/blackhole/internal/core/config/types"
	"go.uber.org/zap"
)

// ConfigManager manages configuration with change notifications
type ConfigManager struct {
	config      *types.Config
	mutex       sync.RWMutex
	subscribers []func(*types.Config)
	logger      *zap.Logger
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(logger *zap.Logger) *ConfigManager {
	if logger == nil {
		// Default logger if not provided
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			// If we can't create a logger, fall back to no logging
			logger = zap.NewNop()
		}
	}
	
	return &ConfigManager{
		config:      NewDefaultConfig(),
		subscribers: make([]func(*types.Config), 0),
		logger:      logger.With(zap.String("component", "config_manager")),
	}
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *types.Config {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.config
}

// SetConfig updates the configuration and notifies subscribers
func (cm *ConfigManager) SetConfig(config *types.Config) error {
	if config == nil {
		return fmt.Errorf("cannot set nil configuration")
	}
	
	// Validate configuration
	if err := ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	
	cm.mutex.Lock()
	cm.config = config
	cm.mutex.Unlock()
	
	cm.logger.Info("Configuration updated")
	
	// Notify subscribers
	cm.notifySubscribers()
	
	return nil
}

// SubscribeToChanges registers a callback function to be called when the configuration changes
func (cm *ConfigManager) SubscribeToChanges(callback func(*types.Config)) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.subscribers = append(cm.subscribers, callback)
	cm.logger.Debug("New configuration subscriber registered", 
		zap.Int("total_subscribers", len(cm.subscribers)))
}

// notifySubscribers notifies all subscribers of configuration changes
func (cm *ConfigManager) notifySubscribers() {
	cm.mutex.RLock()
	config := cm.config
	subscribers := make([]func(*types.Config), len(cm.subscribers))
	copy(subscribers, cm.subscribers)
	cm.mutex.RUnlock()

	for i, callback := range subscribers {
		callback(config)
		cm.logger.Debug("Notified subscriber of configuration change", 
			zap.Int("subscriber_index", i))
	}
}

// LoadFromFile loads configuration from a YAML file
func (cm *ConfigManager) LoadFromFile(path string) error {
	loader := NewFileLoader(path)
	config, err := loader.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration from file %s: %w", path, err)
	}
	
	return cm.SetConfig(config)
}

// SaveToFile saves the current configuration to a YAML file
func (cm *ConfigManager) SaveToFile(path string) error {
	cm.mutex.RLock()
	config := cm.config
	cm.mutex.RUnlock()
	
	writer := NewFileWriter(path)
	if err := writer.Write(config); err != nil {
		return fmt.Errorf("failed to save configuration to file %s: %w", path, err)
	}
	
	cm.logger.Info("Configuration saved to file", zap.String("path", path))
	return nil
}