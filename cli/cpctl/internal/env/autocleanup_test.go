package env

import (
	"context"
	"testing"
	"time"
)

func TestAutoCleanupManager_New(t *testing.T) {
	store, _ := NewMetadataStore(t.TempDir())
	manager := NewAutoCleanupManager(store, "tofu/mirror")

	if manager == nil {
		t.Fatal("NewAutoCleanupManager returned nil")
	}
	if manager.metadataStore != store {
		t.Error("metadataStore not set correctly")
	}
	if manager.tofuDir != "tofu/mirror" {
		t.Errorf("tofuDir: got %q, want %q", manager.tofuDir, "tofu/mirror")
	}
	if manager.checkInterval != 1*time.Minute {
		t.Errorf("checkInterval: got %v, want 1m", manager.checkInterval)
	}
	if manager.running {
		t.Error("manager should not be running initially")
	}
}

func TestAutoCleanupManager_ScheduleCleanup(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	// Save initial metadata
	metadata := &EnvironmentMetadata{
		Name:       "test-env",
		CreatedAt:  time.Now(),
		Status:     "ready",
		TTLSeconds: 0, // No TTL initially
	}
	if err := store.Save("test-env", metadata); err != nil {
		t.Fatalf("Save: %v", err)
	}

	manager := NewAutoCleanupManager(store, "tofu/mirror")

	// Schedule cleanup with 1 hour TTL
	ttlSeconds := 3600
	if err := manager.ScheduleCleanup("test-env", ttlSeconds); err != nil {
		t.Fatalf("ScheduleCleanup: %v", err)
	}

	// Load metadata and verify cleanup was scheduled
	loaded, err := store.Load("test-env")
	if err != nil {
		t.Fatalf("Load after schedule: %v", err)
	}

	if loaded.TTLSeconds != ttlSeconds {
		t.Errorf("TTLSeconds: got %d, want %d", loaded.TTLSeconds, ttlSeconds)
	}
	if !loaded.AutoCleanupScheduled {
		t.Error("AutoCleanupScheduled should be true")
	}
	if loaded.CleanupAt == nil {
		t.Error("CleanupAt should be set")
	}

	// Verify cleanup time is correct
	expectedCleanupAt := loaded.CreatedAt.Add(time.Duration(ttlSeconds) * time.Second)
	if !loaded.CleanupAt.Equal(expectedCleanupAt) {
		t.Errorf("CleanupAt: got %v, want %v", loaded.CleanupAt, expectedCleanupAt)
	}
}

func TestAutoCleanupManager_CancelCleanup(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	// Save metadata with cleanup scheduled
	cleanupAt := time.Now().Add(1 * time.Hour)
	metadata := &EnvironmentMetadata{
		Name:              "test-env",
		CreatedAt:         time.Now(),
		Status:            "ready",
		TTLSeconds:        3600,
		AutoCleanupScheduled: true,
		CleanupAt:         &cleanupAt,
	}
	if err := store.Save("test-env", metadata); err != nil {
		t.Fatalf("Save: %v", err)
	}

	manager := NewAutoCleanupManager(store, "tofu/mirror")

	// Cancel cleanup
	if err := manager.CancelCleanup("test-env"); err != nil {
		t.Fatalf("CancelCleanup: %v", err)
	}

	// Load metadata and verify cleanup was cancelled
	loaded, err := store.Load("test-env")
	if err != nil {
		t.Fatalf("Load after cancel: %v", err)
	}

	if loaded.AutoCleanupScheduled {
		t.Error("AutoCleanupScheduled should be false after cancel")
	}
	if loaded.CleanupAt != nil {
		t.Error("CleanupAt should be nil after cancel")
	}
}

func TestAutoCleanupManager_GetTimeUntilCleanup(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	now := time.Now()
	futureTime := now.Add(30 * time.Minute)

	// Save metadata with cleanup scheduled
	metadata := &EnvironmentMetadata{
		Name:              "test-env",
		CreatedAt:         now,
		Status:            "ready",
		TTLSeconds:        1800, // 30 minutes
		AutoCleanupScheduled: true,
		CleanupAt:         &futureTime,
	}
	if err := store.Save("test-env", metadata); err != nil {
		t.Fatalf("Save: %v", err)
	}

	manager := NewAutoCleanupManager(store, "tofu/mirror")

	// Get time until cleanup
	remaining, err := manager.GetTimeUntilCleanup("test-env")
	if err != nil {
		t.Fatalf("GetTimeUntilCleanup: %v", err)
	}

	// Should be approximately 30 minutes
	expected := 30 * time.Minute
	tolerance := 5 * time.Second
	if remaining < expected-tolerance || remaining > expected+tolerance {
		t.Errorf("Time until cleanup: got %v, want ~%v", remaining, expected)
	}
}

func TestAutoCleanupManager_GetTimeUntilCleanup_NoSchedule(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	// Save metadata without cleanup schedule
	metadata := &EnvironmentMetadata{
		Name:   "test-env",
		Status: "ready",
	}
	if err := store.Save("test-env", metadata); err != nil {
		t.Fatalf("Save: %v", err)
	}

	manager := NewAutoCleanupManager(store, "tofu/mirror")

	// Get time until cleanup should fail
	_, err = manager.GetTimeUntilCleanup("test-env")
	if err == nil {
		t.Error("GetTimeUntilCleanup should fail when no cleanup scheduled")
	}
}

func TestAutoCleanupManager_GetExpiredEnvironments(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	now := time.Now()

	// Create environments with different TTL statuses
	testCases := []struct {
		name     string
		created  time.Time
		ttl      int
		expired  bool
	}{
		{"expired-1", now.Add(-2 * time.Hour), 3600, true},    // Expired 1 hour ago
		{"expired-2", now.Add(-90 * time.Minute), 3600, true}, // Expired 30 minutes ago
		{"active-1", now.Add(-30 * time.Minute), 3600, false}, // Expires in 30 minutes
		{"active-2", now, 3600, false},                       // Expires in 1 hour
		{"no-ttl", now, 0, false},                            // No TTL
	}

	for _, tc := range testCases {
		cleanupAt := tc.created.Add(time.Duration(tc.ttl) * time.Second)
		metadata := &EnvironmentMetadata{
			Name:              tc.name,
			CreatedAt:         tc.created,
			TTLSeconds:        tc.ttl,
			Status:            "ready",
			AutoCleanupScheduled: tc.ttl > 0,
			CleanupAt:         &cleanupAt,
		}
		if err := store.Save(tc.name, metadata); err != nil {
			t.Fatalf("Save %s: %v", tc.name, err)
		}
	}

	manager := NewAutoCleanupManager(store, "tofu/mirror")

	// Get expired environments
	expired, err := manager.metadataStore.GetExpiredEnvironments()
	if err != nil {
		t.Fatalf("GetExpiredEnvironments: %v", err)
	}

	// Verify only expired environments are returned
	if len(expired) != 2 {
		t.Errorf("Expired environments: got %d, want 2", len(expired))
	}

	// Check that expired-1 and expired-2 are in the list
	expiredMap := make(map[string]bool)
	for _, env := range expired {
		expiredMap[env] = true
	}

	if !expiredMap["expired-1"] {
		t.Error("expired-1 should be in expired list")
	}
	if !expiredMap["expired-2"] {
		t.Error("expired-2 should be in expired list")
	}
	if expiredMap["active-1"] {
		t.Error("active-1 should not be in expired list")
	}
	if expiredMap["active-2"] {
		t.Error("active-2 should not be in expired list")
	}
	if expiredMap["no-ttl"] {
		t.Error("no-ttl should not be in expired list")
	}
}

func TestAutoCleanupManager_StartStop(t *testing.T) {
	store, _ := NewMetadataStore(t.TempDir())
	manager := NewAutoCleanupManager(store, "tofu/mirror")

	// Should not be running initially
	if manager.IsRunning() {
		t.Error("manager should not be running initially")
	}

	// Start
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Should be running
	if !manager.IsRunning() {
		t.Error("manager should be running after Start")
	}

	// Stop
	manager.Stop()

	// Should not be running
	if manager.IsRunning() {
		t.Error("manager should not be running after Stop")
	}
}

func TestAutoCleanupManager_StartAlreadyRunning(t *testing.T) {
	store, _ := NewMetadataStore(t.TempDir())
	manager := NewAutoCleanupManager(store, "tofu/mirror")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start first time
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("First Start: %v", err)
	}

	// Try to start again (should fail)
	if err := manager.Start(ctx); err == nil {
		t.Error("Second Start should fail")
	}
}

// Note: Testing the actual cleanupEnvironment method would require
// mocking the tofu orchestrator, which is more complex. The above tests
// cover the scheduling and management logic.

func TestAutoCleanupManager_ResumeCleanupOnStartup(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	// Save environments in different states
	envs := []struct {
		name   string
		status string
	}{
		{"cleaning-env", "cleaning"},
		{"failed-env", "cleanup_failed"},
		{"ready-env", "ready"},
		{"expired-ready", "ready"}, // Will have expired cleanup time
	}

	now := time.Now()
	for _, env := range envs {
		cleanupAt := now.Add(-1 * time.Hour) // Expired 1 hour ago
		metadata := &EnvironmentMetadata{
			Name:              env.name,
			CreatedAt:         now.Add(-2 * time.Hour),
			Status:            env.status,
			TTLSeconds:        3600,
			AutoCleanupScheduled: true,
			CleanupAt:         &cleanupAt,
		}
		if err := store.Save(env.name, metadata); err != nil {
			t.Fatalf("Save %s: %v", env.name, err)
		}
	}

	manager := NewAutoCleanupManager(store, "tofu/mirror")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Resume cleanup
	if err := manager.ResumeCleanupOnStartup(ctx); err != nil {
		t.Fatalf("ResumeCleanupOnStartup: %v", err)
	}

	// Note: In a real test, we would verify that cleanupEnvironment was called
	// for cleaning-env, failed-env, and expired-ready. However, since
	// cleanupEnvironment requires tofu mocking, we can't easily test that here.
	// The test above at least verifies the function doesn't panic and handles
	// all environment states.
}