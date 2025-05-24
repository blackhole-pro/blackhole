package state

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileStateStorage implements StateStorage using the filesystem
type FileStateStorage struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFileStateStorage creates a new file-based state storage
func NewFileStateStorage(baseDir string) (*FileStateStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}
	
	return &FileStateStorage{
		baseDir: baseDir,
	}, nil
}

// Save persists plugin state to a file
func (s *FileStateStorage) Save(ctx context.Context, pluginID string, version string, state []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Create plugin directory
	pluginDir := filepath.Join(s.baseDir, pluginID)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}
	
	// Calculate checksum
	hash := sha256.Sum256(state)
	checksum := hex.EncodeToString(hash[:])
	
	// Create metadata
	metadata := StateMetadata{
		PluginID:  pluginID,
		Version:   version,
		Timestamp: time.Now(),
		Size:      int64(len(state)),
		Checksum:  checksum,
	}
	
	// Save metadata
	metadataPath := filepath.Join(pluginDir, fmt.Sprintf("%s.meta.json", version))
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}
	
	// Save state
	statePath := filepath.Join(pluginDir, fmt.Sprintf("%s.state", version))
	if err := os.WriteFile(statePath, state, 0644); err != nil {
		// Clean up metadata on failure
		_ = os.Remove(metadataPath)
		return fmt.Errorf("failed to write state: %w", err)
	}
	
	return nil
}

// Load retrieves plugin state from a file
func (s *FileStateStorage) Load(ctx context.Context, pluginID string, version string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Read metadata
	metadataPath := filepath.Join(s.baseDir, pluginID, fmt.Sprintf("%s.meta.json", version))
	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("state not found for plugin %s version %s", pluginID, version)
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}
	
	var metadata StateMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	
	// Read state
	statePath := filepath.Join(s.baseDir, pluginID, fmt.Sprintf("%s.state", version))
	state, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %w", err)
	}
	
	// Verify checksum
	hash := sha256.Sum256(state)
	checksum := hex.EncodeToString(hash[:])
	if checksum != metadata.Checksum {
		return nil, fmt.Errorf("state checksum mismatch: expected %s, got %s", metadata.Checksum, checksum)
	}
	
	return state, nil
}

// List returns available state versions for a plugin
func (s *FileStateStorage) List(ctx context.Context, pluginID string) ([]StateVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	pluginDir := filepath.Join(s.baseDir, pluginID)
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []StateVersion{}, nil
		}
		return nil, fmt.Errorf("failed to read plugin directory: %w", err)
	}
	
	var versions []StateVersion
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		// Look for metadata files
		if filepath.Ext(entry.Name()) == ".json" && filepath.Ext(filepath.Base(entry.Name())[:len(entry.Name())-5]) == ".meta" {
			// Read metadata
			metadataPath := filepath.Join(pluginDir, entry.Name())
			metadataData, err := os.ReadFile(metadataPath)
			if err != nil {
				continue
			}
			
			var metadata StateMetadata
			if err := json.Unmarshal(metadataData, &metadata); err != nil {
				continue
			}
			
			versions = append(versions, StateVersion{
				PluginID:  metadata.PluginID,
				Version:   metadata.Version,
				Timestamp: metadata.Timestamp,
				Size:      metadata.Size,
				Checksum:  metadata.Checksum,
			})
		}
	}
	
	return versions, nil
}

// Delete removes a specific state version
func (s *FileStateStorage) Delete(ctx context.Context, pluginID string, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Remove metadata file
	metadataPath := filepath.Join(s.baseDir, pluginID, fmt.Sprintf("%s.meta.json", version))
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove metadata: %w", err)
	}
	
	// Remove state file
	statePath := filepath.Join(s.baseDir, pluginID, fmt.Sprintf("%s.state", version))
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state: %w", err)
	}
	
	// Remove plugin directory if empty
	pluginDir := filepath.Join(s.baseDir, pluginID)
	entries, _ := os.ReadDir(pluginDir)
	if len(entries) == 0 {
		_ = os.Remove(pluginDir)
	}
	
	return nil
}

// StateMetadata represents metadata for a saved state
type StateMetadata struct {
	PluginID  string    `json:"plugin_id"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size"`
	Checksum  string    `json:"checksum"`
}

// MemoryStateStorage implements StateStorage in memory (for testing)
type MemoryStateStorage struct {
	states map[string]map[string]*storedState
	mu     sync.RWMutex
}

type storedState struct {
	data     []byte
	metadata StateMetadata
}

// NewMemoryStateStorage creates a new in-memory state storage
func NewMemoryStateStorage() *MemoryStateStorage {
	return &MemoryStateStorage{
		states: make(map[string]map[string]*storedState),
	}
}

// Save persists plugin state in memory
func (s *MemoryStateStorage) Save(ctx context.Context, pluginID string, version string, state []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Calculate checksum
	hash := sha256.Sum256(state)
	checksum := hex.EncodeToString(hash[:])
	
	// Create plugin map if not exists
	if _, exists := s.states[pluginID]; !exists {
		s.states[pluginID] = make(map[string]*storedState)
	}
	
	// Store state and metadata
	s.states[pluginID][version] = &storedState{
		data: append([]byte(nil), state...), // Make a copy
		metadata: StateMetadata{
			PluginID:  pluginID,
			Version:   version,
			Timestamp: time.Now(),
			Size:      int64(len(state)),
			Checksum:  checksum,
		},
	}
	
	return nil
}

// Load retrieves plugin state from memory
func (s *MemoryStateStorage) Load(ctx context.Context, pluginID string, version string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	pluginStates, exists := s.states[pluginID]
	if !exists {
		return nil, fmt.Errorf("no states found for plugin %s", pluginID)
	}
	
	stored, exists := pluginStates[version]
	if !exists {
		return nil, fmt.Errorf("state not found for plugin %s version %s", pluginID, version)
	}
	
	// Return a copy
	return append([]byte(nil), stored.data...), nil
}

// List returns available state versions for a plugin
func (s *MemoryStateStorage) List(ctx context.Context, pluginID string) ([]StateVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	pluginStates, exists := s.states[pluginID]
	if !exists {
		return []StateVersion{}, nil
	}
	
	versions := make([]StateVersion, 0, len(pluginStates))
	for _, stored := range pluginStates {
		versions = append(versions, StateVersion{
			PluginID:  stored.metadata.PluginID,
			Version:   stored.metadata.Version,
			Timestamp: stored.metadata.Timestamp,
			Size:      stored.metadata.Size,
			Checksum:  stored.metadata.Checksum,
		})
	}
	
	return versions, nil
}

// Delete removes a specific state version
func (s *MemoryStateStorage) Delete(ctx context.Context, pluginID string, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if pluginStates, exists := s.states[pluginID]; exists {
		delete(pluginStates, version)
		if len(pluginStates) == 0 {
			delete(s.states, pluginID)
		}
	}
	
	return nil
}

// StreamingStateStorage wraps a StateStorage to support streaming large states
type StreamingStateStorage struct {
	backend StateStorage
}

// NewStreamingStateStorage creates a new streaming state storage wrapper
func NewStreamingStateStorage(backend StateStorage) *StreamingStateStorage {
	return &StreamingStateStorage{
		backend: backend,
	}
}

// SaveStream saves plugin state from a reader
func (s *StreamingStateStorage) SaveStream(ctx context.Context, pluginID string, version string, reader io.Reader) error {
	// Read all data (in production, this could be chunked)
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read state stream: %w", err)
	}
	
	return s.backend.Save(ctx, pluginID, version, data)
}

// LoadStream loads plugin state to a writer
func (s *StreamingStateStorage) LoadStream(ctx context.Context, pluginID string, version string, writer io.Writer) error {
	data, err := s.backend.Load(ctx, pluginID, version)
	if err != nil {
		return err
	}
	
	_, err = writer.Write(data)
	return err
}