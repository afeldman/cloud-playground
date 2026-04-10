package env

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EnvironmentMetadata stores comprehensive environment state
type EnvironmentMetadata struct {
	Name              string                 `json:"name"`
	CreatedAt         time.Time              `json:"created_at"`
	TTLSeconds        int                    `json:"ttl_seconds"`
	Status            string                 `json:"status"`
	AWSProfile        string                 `json:"aws_profile"`
	TofuOutputs       map[string]string      `json:"tofu_outputs"`
	ResourcesCreated  []string               `json:"resources_created"`
	Vars              map[string]string      `json:"vars"`
	LastUpdated       time.Time              `json:"last_updated"`
	Error             string                 `json:"error,omitempty"`
	AutoCleanupScheduled bool                `json:"auto_cleanup_scheduled"`
	CleanupAt         *time.Time             `json:"cleanup_at,omitempty"`
}

// MetadataStore manages environment metadata persistence
type MetadataStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewMetadataStore creates a new metadata store
func NewMetadataStore(baseDir string) (*MetadataStore, error) {
	if baseDir == "" {
		baseDir = "data/envs"
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	return &MetadataStore{
		baseDir: baseDir,
	}, nil
}

// Save saves environment metadata to disk
func (ms *MetadataStore) Save(envName string, metadata *EnvironmentMetadata) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Update timestamps
	metadata.LastUpdated = time.Now()
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = metadata.LastUpdated
	}

	// Calculate cleanup time if TTL is set
	if metadata.TTLSeconds > 0 && metadata.CleanupAt == nil {
		cleanupAt := metadata.CreatedAt.Add(time.Duration(metadata.TTLSeconds) * time.Second)
		metadata.CleanupAt = &cleanupAt
		metadata.AutoCleanupScheduled = true
	}

	// Create environment directory
	envDir := filepath.Join(ms.baseDir, envName)
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return fmt.Errorf("failed to create environment directory: %w", err)
	}

	// Write to temp file first, then rename for atomicity
	tempFile := filepath.Join(envDir, "metadata.json.tmp")
	finalFile := filepath.Join(envDir, "metadata.json")

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp metadata file: %w", err)
	}

	// Rename to final file
	if err := os.Rename(tempFile, finalFile); err != nil {
		// Clean up temp file
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename metadata file: %w", err)
	}

	slog.Debug("metadata saved", "env", envName, "file", finalFile)
	return nil
}

// Load loads environment metadata from disk
func (ms *MetadataStore) Load(envName string) (*EnvironmentMetadata, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	metadataFile := filepath.Join(ms.baseDir, envName, "metadata.json")
	
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("metadata not found for environment: %s", envName)
		}
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata EnvironmentMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// Update updates specific fields in environment metadata
func (ms *MetadataStore) Update(envName string, updates map[string]interface{}) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Load existing metadata without acquiring the lock again
	metadata, err := ms.loadInternal(envName)
	if err != nil {
		return err
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "status":
			if str, ok := value.(string); ok {
				metadata.Status = str
			}
		case "tofu_outputs":
			if outputs, ok := value.(map[string]string); ok {
				metadata.TofuOutputs = outputs
			}
		case "resources_created":
			if resources, ok := value.([]string); ok {
				metadata.ResourcesCreated = resources
			}
		case "vars":
			if vars, ok := value.(map[string]string); ok {
				metadata.Vars = vars
			}
		case "error":
			if str, ok := value.(string); ok {
				metadata.Error = str
			}
		case "ttl_seconds":
			if ttl, ok := value.(int); ok {
				metadata.TTLSeconds = ttl
				// Recalculate cleanup time
				if ttl > 0 {
					cleanupAt := metadata.CreatedAt.Add(time.Duration(ttl) * time.Second)
					metadata.CleanupAt = &cleanupAt
					metadata.AutoCleanupScheduled = true
				}
			}
		case "auto_cleanup_scheduled":
			if b, ok := value.(bool); ok {
				metadata.AutoCleanupScheduled = b
			}
		case "cleanup_at":
			if value == nil {
				metadata.CleanupAt = nil
			} else if t, ok := value.(*time.Time); ok {
				metadata.CleanupAt = t
			}
		}
	}

	metadata.LastUpdated = time.Now()

	// Save updated metadata
	return ms.saveInternal(envName, metadata)
}

// Delete removes environment metadata from disk
func (ms *MetadataStore) Delete(envName string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	envDir := filepath.Join(ms.baseDir, envName)
	
	// Remove metadata file
	metadataFile := filepath.Join(envDir, "metadata.json")
	if err := os.Remove(metadataFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete metadata file: %w", err)
	}

	// Try to remove directory if empty
	os.Remove(envDir)

	slog.Debug("metadata deleted", "env", envName)
	return nil
}

// List returns all environment names with metadata
func (ms *MetadataStore) List() ([]string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	entries, err := os.ReadDir(ms.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var envs []string
	for _, entry := range entries {
		if entry.IsDir() {
			metadataFile := filepath.Join(ms.baseDir, entry.Name(), "metadata.json")
			if _, err := os.Stat(metadataFile); err == nil {
				envs = append(envs, entry.Name())
			}
		}
	}

	return envs, nil
}

// GetExpiredEnvironments returns environments that have exceeded their TTL
func (ms *MetadataStore) GetExpiredEnvironments() ([]string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	envs, err := ms.listInternal()
	if err != nil {
		return nil, err
	}

	var expired []string
	now := time.Now()

	for _, envName := range envs {
		metadata, err := ms.loadInternal(envName)
		if err != nil {
			slog.Warn("failed to load metadata for environment", "env", envName, "error", err)
			continue
		}

		if metadata.AutoCleanupScheduled && metadata.CleanupAt != nil && now.After(*metadata.CleanupAt) {
			expired = append(expired, envName)
		}
	}

	return expired, nil
}

// loadInternal reads metadata from disk without acquiring the lock.
// Callers must hold at least a read lock.
func (ms *MetadataStore) loadInternal(envName string) (*EnvironmentMetadata, error) {
	metadataFile := filepath.Join(ms.baseDir, envName, "metadata.json")

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("metadata not found for environment: %s", envName)
		}
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata EnvironmentMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// listInternal returns all environment names without acquiring the lock.
// Callers must hold at least a read lock.
func (ms *MetadataStore) listInternal() ([]string, error) {
	entries, err := os.ReadDir(ms.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var envs []string
	for _, entry := range entries {
		if entry.IsDir() {
			metadataFile := filepath.Join(ms.baseDir, entry.Name(), "metadata.json")
			if _, err := os.Stat(metadataFile); err == nil {
				envs = append(envs, entry.Name())
			}
		}
	}

	return envs, nil
}

// saveInternal is the internal save method without locking
func (ms *MetadataStore) saveInternal(envName string, metadata *EnvironmentMetadata) error {
	envDir := filepath.Join(ms.baseDir, envName)
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return fmt.Errorf("failed to create environment directory: %w", err)
	}

	metadataFile := filepath.Join(envDir, "metadata.json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// GetEnvironmentDir returns the directory path for an environment
func (ms *MetadataStore) GetEnvironmentDir(envName string) string {
	return filepath.Join(ms.baseDir, envName)
}

// LockFile creates a lock file for an environment
func (ms *MetadataStore) LockFile(envName string) (*os.File, error) {
	envDir := ms.GetEnvironmentDir(envName)
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create environment directory: %w", err)
	}

	lockFile := filepath.Join(envDir, ".lock")
	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("environment is locked by another process")
		}
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Write PID to lock file
	pid := os.Getpid()
	if _, err := file.WriteString(fmt.Sprintf("%d\n", pid)); err != nil {
		file.Close()
		os.Remove(lockFile)
		return nil, fmt.Errorf("failed to write lock file: %w", err)
	}

	return file, nil
}