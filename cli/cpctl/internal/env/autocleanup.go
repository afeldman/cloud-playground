package env

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AutoCleanupManager manages TTL-based automatic teardown of environments
type AutoCleanupManager struct {
	metadataStore *MetadataStore
	tofuDir       string
	checkInterval time.Duration
	mu            sync.RWMutex
	cancelFunc    context.CancelFunc
	running       bool
}

// NewAutoCleanupManager creates a new auto-cleanup manager
func NewAutoCleanupManager(metadataStore *MetadataStore, tofuDir string) *AutoCleanupManager {
	return &AutoCleanupManager{
		metadataStore: metadataStore,
		tofuDir:       tofuDir,
		checkInterval: 1 * time.Minute, // Check every minute
		running:       false,
	}
}

// Start begins the auto-cleanup background goroutine
func (acm *AutoCleanupManager) Start(ctx context.Context) error {
	acm.mu.Lock()
	defer acm.mu.Unlock()

	if acm.running {
		return fmt.Errorf("auto-cleanup manager already running")
	}

	// Create cancellable context
	cancelCtx, cancel := context.WithCancel(ctx)
	acm.cancelFunc = cancel

	// Start background goroutine
	go acm.run(cancelCtx)
	acm.running = true

	slog.Info("auto-cleanup manager started", "check_interval", acm.checkInterval)
	return nil
}

// Stop stops the auto-cleanup background goroutine
func (acm *AutoCleanupManager) Stop() {
	acm.mu.Lock()
	defer acm.mu.Unlock()

	if !acm.running {
		return
	}

	if acm.cancelFunc != nil {
		acm.cancelFunc()
		acm.cancelFunc = nil
	}

	acm.running = false
	slog.Info("auto-cleanup manager stopped")
}

// run is the main background loop
func (acm *AutoCleanupManager) run(ctx context.Context) {
	ticker := time.NewTicker(acm.checkInterval)
	defer ticker.Stop()

	slog.Debug("auto-cleanup loop started")

	for {
		select {
		case <-ctx.Done():
			slog.Debug("auto-cleanup loop stopped")
			return
		case <-ticker.C:
			acm.checkAndCleanup(ctx)
		}
	}
}

// checkAndCleanup checks for expired environments and cleans them up
func (acm *AutoCleanupManager) checkAndCleanup(ctx context.Context) {
	expiredEnvs, err := acm.metadataStore.GetExpiredEnvironments()
	if err != nil {
		slog.Error("failed to get expired environments", "error", err)
		return
	}

	if len(expiredEnvs) == 0 {
		return
	}

	slog.Info("found expired environments", "count", len(expiredEnvs), "environments", expiredEnvs)

	for _, envName := range expiredEnvs {
		acm.cleanupEnvironment(ctx, envName)
	}
}

// cleanupEnvironment tears down a single expired environment
func (acm *AutoCleanupManager) cleanupEnvironment(ctx context.Context, envName string) {
	slog.Info("cleaning up expired environment", "env", envName)

	// Load metadata to get environment details
	metadata, err := acm.metadataStore.Load(envName)
	if err != nil {
		slog.Error("failed to load metadata for cleanup", "env", envName, "error", err)
		return
	}

	// Update status to "cleaning"
	acm.metadataStore.Update(envName, map[string]interface{}{
		"status": "cleaning",
	})

	// Create tofu orchestrator for this environment
	tofuOrchestrator := NewTofuOrchestrator(acm.tofuDir, "", "")
	
	// Set destroy timeout
	tofuOrchestrator.SetTimeout(10 * time.Minute)

	// Load computed vars from metadata
	if metadata.Vars != nil {
		if err := tofuOrchestrator.GenerateComputedVars(metadata.Vars); err != nil {
			slog.Error("failed to generate computed vars for cleanup", "env", envName, "error", err)
			acm.metadataStore.Update(envName, map[string]interface{}{
				"status": "cleanup_failed",
				"error":  err.Error(),
			})
			return
		}
	}

	// Run tofu destroy
	if err := tofuOrchestrator.Destroy(ctx); err != nil {
		slog.Error("failed to destroy environment", "env", envName, "error", err)
		acm.metadataStore.Update(envName, map[string]interface{}{
			"status": "cleanup_failed",
			"error":  err.Error(),
		})
		return
	}

	// Clean up PID files
	acm.cleanupPIDFiles(envName)

	// Delete metadata
	if err := acm.metadataStore.Delete(envName); err != nil {
		slog.Warn("failed to delete metadata", "env", envName, "error", err)
	}

	slog.Info("environment auto-cleaned up", "env", envName, "ttl", metadata.TTLSeconds)
}

// cleanupPIDFiles removes any PID files associated with the environment
func (acm *AutoCleanupManager) cleanupPIDFiles(envName string) {
	envDir := acm.metadataStore.GetEnvironmentDir(envName)
	
	// Look for PID files
	pidFiles := []string{
		filepath.Join(envDir, "tunnel.pid"),
		filepath.Join(envDir, "process.pid"),
		filepath.Join(envDir, ".lock"),
	}

	for _, pidFile := range pidFiles {
		if _, err := os.Stat(pidFile); err == nil {
			if err := os.Remove(pidFile); err != nil {
				slog.Warn("failed to remove PID file", "file", pidFile, "error", err)
			} else {
				slog.Debug("removed PID file", "file", pidFile)
			}
		}
	}
}

// ScheduleCleanup schedules cleanup for a specific environment
func (acm *AutoCleanupManager) ScheduleCleanup(envName string, ttlSeconds int) error {
	metadata, err := acm.metadataStore.Load(envName)
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	// Update TTL and schedule cleanup
	metadata.TTLSeconds = ttlSeconds
	cleanupAt := metadata.CreatedAt.Add(time.Duration(ttlSeconds) * time.Second)
	metadata.CleanupAt = &cleanupAt
	metadata.AutoCleanupScheduled = true

	if err := acm.metadataStore.Save(envName, metadata); err != nil {
		return fmt.Errorf("failed to save metadata with cleanup schedule: %w", err)
	}

	slog.Info("cleanup scheduled", "env", envName, "ttl_seconds", ttlSeconds, "cleanup_at", cleanupAt.Format(time.RFC3339))
	return nil
}

// CancelCleanup cancels scheduled cleanup for an environment
func (acm *AutoCleanupManager) CancelCleanup(envName string) error {
	return acm.metadataStore.Update(envName, map[string]interface{}{
		"auto_cleanup_scheduled": false,
		"cleanup_at":             nil,
	})
}

// GetTimeUntilCleanup returns time remaining until cleanup
func (acm *AutoCleanupManager) GetTimeUntilCleanup(envName string) (time.Duration, error) {
	metadata, err := acm.metadataStore.Load(envName)
	if err != nil {
		return 0, fmt.Errorf("failed to load metadata: %w", err)
	}

	if metadata.CleanupAt == nil {
		return 0, fmt.Errorf("no cleanup scheduled for environment")
	}

	remaining := time.Until(*metadata.CleanupAt)
	if remaining < 0 {
		return 0, nil
	}

	return remaining, nil
}

// ResumeCleanupOnStartup resumes cleanup for environments that were being cleaned up
// when the process was interrupted
func (acm *AutoCleanupManager) ResumeCleanupOnStartup(ctx context.Context) error {
	envs, err := acm.metadataStore.List()
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	for _, envName := range envs {
		metadata, err := acm.metadataStore.Load(envName)
		if err != nil {
			slog.Warn("failed to load metadata for resume", "env", envName, "error", err)
			continue
		}

		// Check if environment was in cleanup process
		if metadata.Status == "cleaning" || metadata.Status == "cleanup_failed" {
			slog.Info("resuming cleanup for environment", "env", envName, "status", metadata.Status)
			go acm.cleanupEnvironment(ctx, envName)
		}

		// Check if cleanup is overdue
		if metadata.CleanupAt != nil && time.Now().After(*metadata.CleanupAt) {
			slog.Info("cleanup overdue for environment", "env", envName)
			go acm.cleanupEnvironment(ctx, envName)
		}
	}

	return nil
}

// IsRunning returns whether the auto-cleanup manager is running
func (acm *AutoCleanupManager) IsRunning() bool {
	acm.mu.RLock()
	defer acm.mu.RUnlock()
	return acm.running
}