package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Checkpoint represents a rollback checkpoint
type Checkpoint struct {
	ID        string                 `json:"id"`
	PluginID  string                 `json:"plugin_id"`
	Timestamp time.Time              `json:"timestamp"`
	State     []byte                 `json:"-"` // Not included in JSON
	Metadata  map[string]interface{} `json:"metadata"`
}

// FileRollbackManager implements RollbackManager using the filesystem
type FileRollbackManager struct {
	baseDir      string
	stateStorage StateStorage
	mu           sync.RWMutex
}

// NewFileRollbackManager creates a new file-based rollback manager
func NewFileRollbackManager(baseDir string, stateStorage StateStorage) (*FileRollbackManager, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rollback directory: %w", err)
	}
	
	return &FileRollbackManager{
		baseDir:      baseDir,
		stateStorage: stateStorage,
	}, nil
}

// CreateCheckpoint creates a rollback checkpoint
func (m *FileRollbackManager) CreateCheckpoint(ctx context.Context, pluginID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Generate checkpoint ID
	checkpointID := uuid.New().String()
	
	// Get current plugin state
	// In a real implementation, this would get the latest version
	// For now, we'll use a placeholder version
	version := "current"
	state, err := m.stateStorage.Load(ctx, pluginID, version)
	if err != nil {
		// If no state exists, create empty checkpoint
		state = []byte("{}")
	}
	
	// Create checkpoint
	checkpoint := Checkpoint{
		ID:        checkpointID,
		PluginID:  pluginID,
		Timestamp: time.Now(),
		State:     state,
		Metadata: map[string]interface{}{
			"version": version,
		},
	}
	
	// Save checkpoint metadata
	metadataPath := filepath.Join(m.baseDir, fmt.Sprintf("%s.meta.json", checkpointID))
	metadataData, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal checkpoint metadata: %w", err)
	}
	
	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		return "", fmt.Errorf("failed to write checkpoint metadata: %w", err)
	}
	
	// Save checkpoint state
	statePath := filepath.Join(m.baseDir, fmt.Sprintf("%s.state", checkpointID))
	if err := os.WriteFile(statePath, state, 0644); err != nil {
		// Clean up metadata on failure
		_ = os.Remove(metadataPath)
		return "", fmt.Errorf("failed to write checkpoint state: %w", err)
	}
	
	return checkpointID, nil
}

// Rollback rolls back to a checkpoint
func (m *FileRollbackManager) Rollback(ctx context.Context, pluginID string, checkpointID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Load checkpoint metadata
	metadataPath := filepath.Join(m.baseDir, fmt.Sprintf("%s.meta.json", checkpointID))
	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to read checkpoint metadata: %w", err)
	}
	
	var checkpoint Checkpoint
	if err := json.Unmarshal(metadataData, &checkpoint); err != nil {
		return fmt.Errorf("failed to unmarshal checkpoint metadata: %w", err)
	}
	
	// Verify plugin ID
	if checkpoint.PluginID != pluginID {
		return fmt.Errorf("checkpoint %s is for plugin %s, not %s", checkpointID, checkpoint.PluginID, pluginID)
	}
	
	// Load checkpoint state
	statePath := filepath.Join(m.baseDir, fmt.Sprintf("%s.state", checkpointID))
	state, err := os.ReadFile(statePath)
	if err != nil {
		return fmt.Errorf("failed to read checkpoint state: %w", err)
	}
	
	// Restore state
	version := "current"
	if v, ok := checkpoint.Metadata["version"].(string); ok {
		version = v
	}
	
	if err := m.stateStorage.Save(ctx, pluginID, version, state); err != nil {
		return fmt.Errorf("failed to restore state: %w", err)
	}
	
	return nil
}

// CleanupCheckpoint removes a checkpoint
func (m *FileRollbackManager) CleanupCheckpoint(ctx context.Context, checkpointID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Remove metadata file
	metadataPath := filepath.Join(m.baseDir, fmt.Sprintf("%s.meta.json", checkpointID))
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove checkpoint metadata: %w", err)
	}
	
	// Remove state file
	statePath := filepath.Join(m.baseDir, fmt.Sprintf("%s.state", checkpointID))
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove checkpoint state: %w", err)
	}
	
	return nil
}

// ListCheckpoints lists all checkpoints for a plugin
func (m *FileRollbackManager) ListCheckpoints(ctx context.Context, pluginID string) ([]Checkpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read rollback directory: %w", err)
	}
	
	var checkpoints []Checkpoint
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		// Look for metadata files
		if filepath.Ext(entry.Name()) == ".json" {
			// Read metadata
			metadataPath := filepath.Join(m.baseDir, entry.Name())
			metadataData, err := os.ReadFile(metadataPath)
			if err != nil {
				continue
			}
			
			var checkpoint Checkpoint
			if err := json.Unmarshal(metadataData, &checkpoint); err != nil {
				continue
			}
			
			// Filter by plugin ID
			if checkpoint.PluginID == pluginID {
				checkpoints = append(checkpoints, checkpoint)
			}
		}
	}
	
	return checkpoints, nil
}

// CleanupOldCheckpoints removes checkpoints older than the specified duration
func (m *FileRollbackManager) CleanupOldCheckpoints(ctx context.Context, maxAge time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read rollback directory: %w", err)
	}
	
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		// Look for metadata files
		if filepath.Ext(entry.Name()) == ".json" {
			// Read metadata
			metadataPath := filepath.Join(m.baseDir, entry.Name())
			metadataData, err := os.ReadFile(metadataPath)
			if err != nil {
				continue
			}
			
			var checkpoint Checkpoint
			if err := json.Unmarshal(metadataData, &checkpoint); err != nil {
				continue
			}
			
			// Check age
			if checkpoint.Timestamp.Before(cutoff) {
				// Remove checkpoint
				checkpointID := checkpoint.ID
				_ = os.Remove(metadataPath)
				_ = os.Remove(filepath.Join(m.baseDir, fmt.Sprintf("%s.state", checkpointID)))
			}
		}
	}
	
	return nil
}

// MemoryRollbackManager implements RollbackManager in memory (for testing)
type MemoryRollbackManager struct {
	checkpoints  map[string]*Checkpoint
	stateStorage StateStorage
	mu           sync.RWMutex
}

// NewMemoryRollbackManager creates a new in-memory rollback manager
func NewMemoryRollbackManager(stateStorage StateStorage) *MemoryRollbackManager {
	return &MemoryRollbackManager{
		checkpoints:  make(map[string]*Checkpoint),
		stateStorage: stateStorage,
	}
}

// CreateCheckpoint creates a rollback checkpoint
func (m *MemoryRollbackManager) CreateCheckpoint(ctx context.Context, pluginID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Generate checkpoint ID
	checkpointID := uuid.New().String()
	
	// Get current plugin state
	version := "current"
	state, err := m.stateStorage.Load(ctx, pluginID, version)
	if err != nil {
		// If no state exists, create empty checkpoint
		state = []byte("{}")
	}
	
	// Create checkpoint
	checkpoint := &Checkpoint{
		ID:        checkpointID,
		PluginID:  pluginID,
		Timestamp: time.Now(),
		State:     append([]byte(nil), state...), // Make a copy
		Metadata: map[string]interface{}{
			"version": version,
		},
	}
	
	m.checkpoints[checkpointID] = checkpoint
	
	return checkpointID, nil
}

// Rollback rolls back to a checkpoint
func (m *MemoryRollbackManager) Rollback(ctx context.Context, pluginID string, checkpointID string) error {
	m.mu.RLock()
	checkpoint, exists := m.checkpoints[checkpointID]
	m.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("checkpoint %s not found", checkpointID)
	}
	
	// Verify plugin ID
	if checkpoint.PluginID != pluginID {
		return fmt.Errorf("checkpoint %s is for plugin %s, not %s", checkpointID, checkpoint.PluginID, pluginID)
	}
	
	// Restore state
	version := "current"
	if v, ok := checkpoint.Metadata["version"].(string); ok {
		version = v
	}
	
	return m.stateStorage.Save(ctx, pluginID, version, checkpoint.State)
}

// CleanupCheckpoint removes a checkpoint
func (m *MemoryRollbackManager) CleanupCheckpoint(ctx context.Context, checkpointID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.checkpoints, checkpointID)
	return nil
}