package env

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetadataStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	metadata := &EnvironmentMetadata{
		Name:       "test-env",
		CreatedAt:  time.Now(),
		TTLSeconds: 3600,
		Status:     "ready",
		AWSProfile: "test-profile",
		TofuOutputs: map[string]string{
			"vpc_id": "vpc-12345678",
		},
		ResourcesCreated: []string{"vpc", "lambda"},
		Vars: map[string]string{
			"instance_count": "3",
		},
	}

	// Save metadata
	if err := store.Save("test-env", metadata); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load metadata
	loaded, err := store.Load("test-env")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Verify loaded metadata
	if loaded.Name != metadata.Name {
		t.Errorf("Name: got %q, want %q", loaded.Name, metadata.Name)
	}
	if loaded.Status != metadata.Status {
		t.Errorf("Status: got %q, want %q", loaded.Status, metadata.Status)
	}
	if loaded.TTLSeconds != metadata.TTLSeconds {
		t.Errorf("TTLSeconds: got %d, want %d", loaded.TTLSeconds, metadata.TTLSeconds)
	}
	if loaded.AWSProfile != metadata.AWSProfile {
		t.Errorf("AWSProfile: got %q, want %q", loaded.AWSProfile, metadata.AWSProfile)
	}
	if len(loaded.TofuOutputs) != len(metadata.TofuOutputs) {
		t.Errorf("TofuOutputs length: got %d, want %d", len(loaded.TofuOutputs), len(metadata.TofuOutputs))
	}
	if len(loaded.ResourcesCreated) != len(metadata.ResourcesCreated) {
		t.Errorf("ResourcesCreated length: got %d, want %d", len(loaded.ResourcesCreated), len(metadata.ResourcesCreated))
	}
	if len(loaded.Vars) != len(metadata.Vars) {
		t.Errorf("Vars length: got %d, want %d", len(loaded.Vars), len(metadata.Vars))
	}

	// Verify cleanup was scheduled
	if !loaded.AutoCleanupScheduled {
		t.Error("AutoCleanupScheduled should be true when TTLSeconds > 0")
	}
	if loaded.CleanupAt == nil {
		t.Error("CleanupAt should be set when TTLSeconds > 0")
	}
}

func TestMetadataStore_Update(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	// Save initial metadata
	metadata := &EnvironmentMetadata{
		Name:       "test-env",
		CreatedAt:  time.Now(),
		Status:     "provisioning",
		TTLSeconds: 3600,
	}
	if err := store.Save("test-env", metadata); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Update metadata
	updates := map[string]interface{}{
		"status": "ready",
		"tofu_outputs": map[string]string{
			"batch_compute_env_arn": "arn:aws:batch:...",
		},
		"error": "",
	}
	if err := store.Update("test-env", updates); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Load and verify updates
	loaded, err := store.Load("test-env")
	if err != nil {
		t.Fatalf("Load after update: %v", err)
	}

	if loaded.Status != "ready" {
		t.Errorf("Status after update: got %q, want %q", loaded.Status, "ready")
	}
	if len(loaded.TofuOutputs) != 1 {
		t.Errorf("TofuOutputs after update: got %d items, want 1", len(loaded.TofuOutputs))
	}
	if loaded.Error != "" {
		t.Errorf("Error after update: got %q, want empty", loaded.Error)
	}
}

func TestMetadataStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	// Save metadata
	metadata := &EnvironmentMetadata{
		Name:   "test-env",
		Status: "ready",
	}
	if err := store.Save("test-env", metadata); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify it exists
	if _, err := store.Load("test-env"); err != nil {
		t.Fatalf("Load before delete: %v", err)
	}

	// Delete
	if err := store.Delete("test-env"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify it's gone
	if _, err := store.Load("test-env"); err == nil {
		t.Error("Load after delete should fail")
	}
}

func TestMetadataStore_List(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	// Save multiple environments
	envs := []string{"env1", "env2", "env3"}
	for _, envName := range envs {
		metadata := &EnvironmentMetadata{
			Name:   envName,
			Status: "ready",
		}
		if err := store.Save(envName, metadata); err != nil {
			t.Fatalf("Save %s: %v", envName, err)
		}
	}

	// List environments
	list, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Verify all environments are listed
	if len(list) != len(envs) {
		t.Errorf("List length: got %d, want %d", len(list), len(envs))
	}

	// Check that all expected environments are in the list
	envMap := make(map[string]bool)
	for _, env := range list {
		envMap[env] = true
	}
	for _, expectedEnv := range envs {
		if !envMap[expectedEnv] {
			t.Errorf("Environment %q not found in list", expectedEnv)
		}
	}
}

func TestMetadataStore_GetExpiredEnvironments(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	now := time.Now()

	// Create environments with different TTLs
	testCases := []struct {
		name     string
		created  time.Time
		ttl      int
		expired  bool
	}{
		{"expired-env", now.Add(-2 * time.Hour), 3600, true},    // Created 2h ago, TTL 1h
		{"active-env", now.Add(-30 * time.Minute), 3600, false}, // Created 30m ago, TTL 1h
		{"no-ttl-env", now, 0, false},                           // No TTL
	}

	for _, tc := range testCases {
		metadata := &EnvironmentMetadata{
			Name:       tc.name,
			CreatedAt:  tc.created,
			TTLSeconds: tc.ttl,
			Status:     "ready",
		}
		if err := store.Save(tc.name, metadata); err != nil {
			t.Fatalf("Save %s: %v", tc.name, err)
		}
	}

	// Get expired environments
	expired, err := store.GetExpiredEnvironments()
	if err != nil {
		t.Fatalf("GetExpiredEnvironments: %v", err)
	}

	// Verify only expired-env is in the list
	if len(expired) != 1 {
		t.Errorf("Expired environments: got %d, want 1", len(expired))
	}
	if len(expired) > 0 && expired[0] != "expired-env" {
		t.Errorf("Expired environment: got %q, want %q", expired[0], "expired-env")
	}
}

func TestMetadataStore_LockFile(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	// Create first lock
	lockFile1, err := store.LockFile("test-env")
	if err != nil {
		t.Fatalf("First LockFile: %v", err)
	}
	defer lockFile1.Close()

	// Try to create second lock (should fail)
	if _, err := store.LockFile("test-env"); err == nil {
		t.Error("Second LockFile should fail")
	}

	// Release first lock
	lockFile1.Close()
	os.Remove(filepath.Join(store.GetEnvironmentDir("test-env"), ".lock"))

	// Should be able to create lock after release
	lockFile2, err := store.LockFile("test-env")
	if err != nil {
		t.Fatalf("LockFile after release: %v", err)
	}
	lockFile2.Close()
}

func TestMetadataStore_AtomicSave(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	// Save metadata multiple times to test atomicity
	metadata := &EnvironmentMetadata{
		Name:   "test-env",
		Status: "ready",
	}

	// Save 10 times concurrently (in test we do sequentially)
	for i := 0; i < 10; i++ {
		metadata.TTLSeconds = i * 3600
		if err := store.Save("test-env", metadata); err != nil {
			t.Fatalf("Save iteration %d: %v", i, err)
		}

		// Load and verify
		loaded, err := store.Load("test-env")
		if err != nil {
			t.Fatalf("Load iteration %d: %v", i, err)
		}
		if loaded.TTLSeconds != i*3600 {
			t.Errorf("Iteration %d: TTLSeconds got %d, want %d", i, loaded.TTLSeconds, i*3600)
		}
	}
}