package state

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
)

// RollbackManager manages rollback operations
type RollbackManager interface {
	// CreateCheckpoint creates a rollback checkpoint
	CreateCheckpoint(ctx context.Context, pluginID string) (string, error)
	
	// Rollback rolls back to a checkpoint
	Rollback(ctx context.Context, pluginID string, checkpointID string) error
	
	// CleanupCheckpoint removes a checkpoint
	CleanupCheckpoint(ctx context.Context, checkpointID string) error
}

// PluginManager defines the interface for plugin management operations
type PluginManager interface {
	// GetPlugin returns plugin information
	GetPlugin(pluginID string) (*plugins.PluginInfo, error)
	
	// StopPlugin stops a running plugin
	StopPlugin(ctx context.Context, pluginID string) error
	
	// StartPlugin starts a plugin
	StartPlugin(ctx context.Context, pluginID string) error
	
	// LoadPlugin loads a new plugin version
	LoadPlugin(ctx context.Context, spec plugins.PluginSpec) error
	
	// UnloadPlugin unloads a plugin
	UnloadPlugin(ctx context.Context, pluginID string) error
	
	// DrainRequests drains pending requests from a plugin
	DrainRequests(ctx context.Context, pluginID string, timeout time.Duration) error
}

// HotSwapStatus represents the status of a hot-swap operation
type HotSwapStatus struct {
	PluginID      string
	OldVersion    string
	NewVersion    string
	Status        string
	StartTime     time.Time
	EndTime       time.Time
	Error         error
	CheckpointID  string
}

// hotSwapCoordinator coordinates hot-swap operations
type hotSwapCoordinator struct {
	manager      PluginManager
	state        *stateManager
	rollback     RollbackManager
	operations   map[string]*HotSwapStatus
	mu           sync.RWMutex
}

// NewHotSwapCoordinator creates a new hot-swap coordinator
func NewHotSwapCoordinator(manager PluginManager, state *stateManager, rollback RollbackManager) *hotSwapCoordinator {
	return &hotSwapCoordinator{
		manager:    manager,
		state:      state,
		rollback:   rollback,
		operations: make(map[string]*HotSwapStatus),
	}
}

// HotSwap performs a zero-downtime plugin update
func (c *hotSwapCoordinator) HotSwap(ctx context.Context, pluginID string, newSpec plugins.PluginSpec) error {
	// Create operation status
	status := &HotSwapStatus{
		PluginID:   pluginID,
		NewVersion: newSpec.Version,
		Status:     "initializing",
		StartTime:  time.Now(),
	}
	
	c.mu.Lock()
	c.operations[pluginID] = status
	c.mu.Unlock()
	
	// Defer status update
	defer func() {
		status.EndTime = time.Now()
		if status.Error != nil {
			status.Status = "failed"
		} else {
			status.Status = "completed"
		}
	}()
	
	// Get current plugin info
	oldPlugin, err := c.manager.GetPlugin(pluginID)
	if err != nil {
		status.Error = fmt.Errorf("failed to get plugin info: %w", err)
		return status.Error
	}
	status.OldVersion = oldPlugin.Version
	
	// Create rollback checkpoint
	status.Status = "creating_checkpoint"
	checkpointID, err := c.rollback.CreateCheckpoint(ctx, pluginID)
	if err != nil {
		status.Error = fmt.Errorf("failed to create checkpoint: %w", err)
		return status.Error
	}
	status.CheckpointID = checkpointID
	
	// Perform hot-swap with rollback on failure
	if err := c.performHotSwap(ctx, status, oldPlugin, newSpec); err != nil {
		status.Status = "rolling_back"
		// Attempt rollback
		if rollbackErr := c.rollback.Rollback(ctx, pluginID, checkpointID); rollbackErr != nil {
			status.Error = fmt.Errorf("hot-swap failed and rollback failed: swap error: %w, rollback error: %v", err, rollbackErr)
		} else {
			status.Error = fmt.Errorf("hot-swap failed, rolled back successfully: %w", err)
		}
		return status.Error
	}
	
	// Cleanup checkpoint on success
	_ = c.rollback.CleanupCheckpoint(ctx, checkpointID)
	
	return nil
}

// performHotSwap performs the actual hot-swap operation
func (c *hotSwapCoordinator) performHotSwap(ctx context.Context, status *HotSwapStatus, oldPlugin *plugins.PluginInfo, newSpec plugins.PluginSpec) error {
	// Step 1: Load new plugin version
	status.Status = "loading_new_version"
	if err := c.manager.LoadPlugin(ctx, newSpec); err != nil {
		return fmt.Errorf("failed to load new plugin version: %w", err)
	}
	
	// Step 2: Export state from old plugin
	status.Status = "exporting_state"
	stateData, err := c.exportState(ctx, oldPlugin)
	if err != nil {
		// Unload new plugin
		_ = c.manager.UnloadPlugin(ctx, newSpec.Name)
		return fmt.Errorf("failed to export state: %w", err)
	}
	
	// Step 3: Migrate state if needed
	if oldPlugin.Version != newSpec.Version {
		status.Status = "migrating_state"
		migratedState, err := c.state.MigrateState(ctx, oldPlugin.Name, oldPlugin.Version, newSpec.Version)
		if err != nil {
			// Unload new plugin
			_ = c.manager.UnloadPlugin(ctx, newSpec.Name)
			return fmt.Errorf("failed to migrate state: %w", err)
		}
		stateData = migratedState
	}
	
	// Step 4: Drain requests from old plugin
	status.Status = "draining_requests"
	drainTimeout := 30 * time.Second
	if err := c.manager.DrainRequests(ctx, oldPlugin.Name, drainTimeout); err != nil {
		// Continue anyway - requests may have timed out
	}
	
	// Step 5: Stop old plugin
	status.Status = "stopping_old_version"
	if err := c.manager.StopPlugin(ctx, oldPlugin.Name); err != nil {
		// Unload new plugin
		_ = c.manager.UnloadPlugin(ctx, newSpec.Name)
		return fmt.Errorf("failed to stop old plugin: %w", err)
	}
	
	// Step 6: Start new plugin
	status.Status = "starting_new_version"
	if err := c.manager.StartPlugin(ctx, newSpec.Name); err != nil {
		// Try to restart old plugin
		_ = c.manager.StartPlugin(ctx, oldPlugin.Name)
		// Unload new plugin
		_ = c.manager.UnloadPlugin(ctx, newSpec.Name)
		return fmt.Errorf("failed to start new plugin: %w", err)
	}
	
	// Step 7: Import state to new plugin
	status.Status = "importing_state"
	if err := c.importState(ctx, newSpec.Name, stateData); err != nil {
		// Stop new plugin and restart old
		_ = c.manager.StopPlugin(ctx, newSpec.Name)
		_ = c.manager.StartPlugin(ctx, oldPlugin.Name)
		// Unload new plugin
		_ = c.manager.UnloadPlugin(ctx, newSpec.Name)
		return fmt.Errorf("failed to import state: %w", err)
	}
	
	// Step 8: Unload old plugin
	status.Status = "unloading_old_version"
	if err := c.manager.UnloadPlugin(ctx, oldPlugin.Name); err != nil {
		// Non-critical error, log but continue
	}
	
	status.Status = "completed"
	return nil
}

// exportState exports state from a plugin
func (c *hotSwapCoordinator) exportState(ctx context.Context, plugin *plugins.PluginInfo) ([]byte, error) {
	// This would communicate with the plugin to export its state
	// For now, return placeholder
	return c.state.exportPluginState(ctx, plugin.Name, plugin.Version)
}

// importState imports state to a plugin
func (c *hotSwapCoordinator) importState(ctx context.Context, pluginID string, stateData []byte) error {
	// This would communicate with the plugin to import state
	// For now, return success
	return c.state.importPluginState(ctx, pluginID, stateData)
}

// GetStatus returns the status of a hot-swap operation
func (c *hotSwapCoordinator) GetStatus(pluginID string) (*HotSwapStatus, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	status, exists := c.operations[pluginID]
	return status, exists
}

// ListOperations lists all hot-swap operations
func (c *hotSwapCoordinator) ListOperations() []*HotSwapStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	operations := make([]*HotSwapStatus, 0, len(c.operations))
	for _, op := range c.operations {
		operations = append(operations, op)
	}
	
	return operations
}

// CleanupOldOperations removes completed operations older than the specified duration
func (c *hotSwapCoordinator) CleanupOldOperations(maxAge time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	for id, op := range c.operations {
		if op.EndTime.Before(cutoff) && (op.Status == "completed" || op.Status == "failed") {
			delete(c.operations, id)
		}
	}
}