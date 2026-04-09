package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"cpctl/internal/config"
	"cpctl/internal/env"

	"github.com/spf13/cobra"
)

var (
	envStage string // Flag for --stage
	envTTL   string // Flag for --ttl (Mirror-Cloud only)
	envScale int    // Flag for --scale (Mirror-Cloud only)
)

// envCmd represents the environment command group
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage cloud environments (LocalStack / Mirror-Cloud)",
	Long: `Manage the full lifecycle of development environments:
  - LocalStack: cost-free local AWS emulation
  - Mirror-Cloud: ephemeral AWS infrastructure with auto-teardown

Examples:
  cpctl env up                     # Provision environment based on config stage
  cpctl env up mirror --ttl 4h     # Provision Mirror-Cloud with 4-hour TTL
  cpctl env status                 # Check environment health
  cpctl env status mirror          # Check Mirror-Cloud status
  cpctl env scale mirror --workers 5 # Scale Mirror-Cloud to 5 workers
  cpctl env down                   # Teardown environment`,
}

// envUpCmd provisions the environment
var envUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Provision infrastructure (LocalStack or Mirror-Cloud)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		slog.Info("provisioning environment")

		// Create environment manager
		manager, err := env.NewEnvironmentManager("data/environments", &config.Cfg)
		if err != nil {
			return fmt.Errorf("failed to create environment manager: %w", err)
		}

		// Override stage if flag provided
		if envStage != "" {
			if err := manager.SetStage(env.Stage(envStage)); err != nil {
				return err
			}
		}

		stageName := manager.GetStageName()
		fmt.Printf("🚀 Provisioning %s environment...\n", stageName)

		// Handle Mirror-Cloud specific setup
		if manager.GetStage() == env.StageMirror {
			return handleMirrorCloudUp(ctx, manager, envTTL)
		}

		// LocalStack provisioning (existing logic)
		return handleLocalStackUp(ctx, manager)
	},
}

// envStatusCmd shows environment health
var envStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check environment health and resource status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		slog.Info("checking environment status")

		// Create environment manager
		manager, err := env.NewEnvironmentManager("data/environments", &config.Cfg)
		if err != nil {
			return fmt.Errorf("failed to create environment manager: %w", err)
		}

		// Override stage if flag provided
		if envStage != "" {
			if err := manager.SetStage(env.Stage(envStage)); err != nil {
				return err
			}
		}

		// Handle Mirror-Cloud status
		if manager.GetStage() == env.StageMirror {
			return handleMirrorCloudStatus(ctx, manager)
		}

		// LocalStack status (existing logic)
		return handleLocalStackStatus(ctx, manager)
	},
}

// envDownCmd teardowns the environment
var envDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Teardown infrastructure (cleanup all resources)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		slog.Info("tearing down environment")

		// Create environment manager
		manager, err := env.NewEnvironmentManager("data/environments", &config.Cfg)
		if err != nil {
			return fmt.Errorf("failed to create environment manager: %w", err)
		}

		// Override stage if flag provided
		if envStage != "" {
			if err := manager.SetStage(env.Stage(envStage)); err != nil {
				return err
			}
		}

		stageName := manager.GetStageName()
		fmt.Printf("🛑 Tearing down %s environment...\n", stageName)

		// Handle Mirror-Cloud teardown
		if manager.GetStage() == env.StageMirror {
			return handleMirrorCloudDown(ctx, manager)
		}

		// LocalStack teardown (existing logic)
		return handleLocalStackDown(ctx, manager)
	},
}

// envScaleCmd scales Mirror-Cloud resources
var envScaleCmd = &cobra.Command{
	Use:   "scale",
	Short: "Scale Mirror-Cloud resources (workers, memory, instance type)",
	Args:  cobra.ExactArgs(1), // Requires environment name
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		envName := args[0]

		slog.Info("scaling environment", "env", envName, "workers", envScale)

		// Only Mirror-Cloud supports scaling
		if envStage != "" && env.Stage(envStage) != env.StageMirror {
			return fmt.Errorf("scaling only supported for Mirror-Cloud environments")
		}

		return handleMirrorCloudScale(ctx, envName, envScale)
	},
}

// handleMirrorCloudUp provisions a Mirror-Cloud environment
func handleMirrorCloudUp(ctx context.Context, manager *env.EnvironmentManager, ttl string) error {
	// Get Mirror configuration
	mirrorConfig := manager.GetMirrorConfig()
	if mirrorConfig == nil || !mirrorConfig.Enabled {
		return fmt.Errorf("Mirror-Cloud is not enabled in configuration")
	}

	envName := manager.GetEnvName()
	fmt.Printf("📝 Environment name: %s\n", envName)

	// Parse TTL
	ttlSeconds, err := parseTTL(ttl)
	if err != nil {
		return fmt.Errorf("invalid TTL format: %w", err)
	}

	// Use default TTL if not specified
	if ttlSeconds == 0 && mirrorConfig.DefaultTTL != "" {
		ttlSeconds, err = parseTTL(mirrorConfig.DefaultTTL)
		if err != nil {
			return fmt.Errorf("invalid default TTL in config: %w", err)
		}
	}

	// Apply cost safeguards
	if ttlSeconds > 24*3600 { // 24 hours
		return fmt.Errorf("TTL cannot exceed 24 hours (cost safeguard)")
	}

	// Create metadata store
	metadataStore, err := env.NewMetadataStore("data/envs")
	if err != nil {
		return fmt.Errorf("failed to create metadata store: %w", err)
	}

	// Create tofu orchestrator
	tofuDir := manager.GetTofuDir()
	tofuOrchestrator := env.NewTofuOrchestrator(tofuDir, "", "")

	// Generate computed variables
	vars := map[string]string{
		"aws_profile":          mirrorConfig.AWSProfile,
		"aws_region":           mirrorConfig.AWSRegion,
		"auto_teardown_ttl":    formatTTL(ttlSeconds),
		"enable_compute":       strconv.FormatBool(mirrorConfig.Resources["compute"]),
		"enable_database":      strconv.FormatBool(mirrorConfig.Resources["database"]),
		"enable_batch":         strconv.FormatBool(mirrorConfig.Resources["batch"]),
		"instance_count":       "3", // Default instance count
	}

	// Step 1: Validate configuration
	fmt.Println("\n📝 Validating configuration...")
	if err := tofuOrchestrator.Validate(ctx); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	fmt.Println("✅ Configuration valid")

	// Step 2: Initialize Tofu backend
	fmt.Println("\n🔧 Initializing Tofu backend...")
	if err := tofuOrchestrator.Init(ctx); err != nil {
		return fmt.Errorf("tofu initialization failed: %w", err)
	}
	fmt.Println("✅ Backend initialized")

	// Step 3: Plan deployment
	fmt.Println("\n📊 Planning deployment...")
	if err := tofuOrchestrator.Plan(ctx, vars); err != nil {
		return fmt.Errorf("deployment plan failed: %w", err)
	}
	fmt.Println("✅ Plan successful")

	// Step 4: Apply configuration
	fmt.Println("\n🔨 Applying configuration (this may take 5-10 minutes)...")
	if err := tofuOrchestrator.Apply(ctx, vars); err != nil {
		// Save error state
		metadata := &env.EnvironmentMetadata{
			Name:       envName,
			Status:     "error",
			Error:      err.Error(),
			LastUpdated: time.Now(),
		}
		metadataStore.Save(envName, metadata)
		return fmt.Errorf("provisioning failed: %w", err)
	}
	fmt.Println("✅ Infrastructure provisioned")

	// Step 5: Get outputs
	fmt.Println("\n📡 Retrieving resource information...")
	outputs, err := tofuOrchestrator.GetOutputs(ctx)
	if err != nil {
		slog.Warn("failed to get tofu outputs", "error", err)
		outputs = make(map[string]string)
	}

	// Step 6: Create metadata
	metadata := &env.EnvironmentMetadata{
		Name:              envName,
		CreatedAt:         time.Now(),
		TTLSeconds:        ttlSeconds,
		Status:            "ready",
		AWSProfile:        mirrorConfig.AWSProfile,
		TofuOutputs:       outputs,
		ResourcesCreated:  getResourcesFromConfig(mirrorConfig.Resources),
		Vars:              vars,
		LastUpdated:       time.Now(),
		AutoCleanupScheduled: ttlSeconds > 0,
	}

	if err := metadataStore.Save(envName, metadata); err != nil {
		return fmt.Errorf("failed to save environment metadata: %w", err)
	}

	// Step 7: Start auto-cleanup if TTL is set
	if ttlSeconds > 0 {
		autoCleanupManager := env.NewAutoCleanupManager(metadataStore, tofuDir)
		if err := autoCleanupManager.Start(ctx); err != nil {
			slog.Warn("failed to start auto-cleanup", "error", err)
		} else {
			if err := autoCleanupManager.ScheduleCleanup(envName, ttlSeconds); err != nil {
				slog.Warn("failed to schedule cleanup", "error", err)
			}
		}
	}

	// Success summary
	cleanupTime := metadata.CreatedAt.Add(time.Duration(ttlSeconds) * time.Second)
	fmt.Printf(`
✅ Mirror-Cloud Environment Ready!

📊 Environment Details:
   Name:        %s
   Status:      %s
   AWS Profile: %s
   AWS Region:  %s
   TTL:         %s
   Cleanup At:  %s

🔗 Resources Created:
   %s

💡 Next Steps:
   cpctl env status %s    # Check detailed status
   cpctl env scale %s --workers 5  # Scale resources
   cpctl env down %s      # Manual teardown
`,
		envName,
		metadata.Status,
		metadata.AWSProfile,
		mirrorConfig.AWSRegion,
		formatTTL(ttlSeconds),
		cleanupTime.Format("2006-01-02 15:04:05"),
		strings.Join(metadata.ResourcesCreated, "\n   "),
		envName,
		envName,
		envName,
	)

	return nil
}

// handleMirrorCloudStatus shows Mirror-Cloud environment status
func handleMirrorCloudStatus(ctx context.Context, manager *env.EnvironmentManager) error {
	metadataStore, err := env.NewMetadataStore("data/envs")
	if err != nil {
		return fmt.Errorf("failed to create metadata store: %w", err)
	}

	// List all Mirror-Cloud environments
	envs, err := metadataStore.List()
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	if len(envs) == 0 {
		fmt.Println("🔍 No Mirror-Cloud environments found")
		return nil
	}

	fmt.Println("🔍 Mirror-Cloud Environments")
	fmt.Println("────────────────────────────────────────")

	for _, envName := range envs {
		metadata, err := metadataStore.Load(envName)
		if err != nil {
			fmt.Printf("❌ %s - Error loading metadata\n", envName)
			continue
		}

		// Calculate time remaining
		timeRemaining := "N/A"
		if metadata.CleanupAt != nil {
			remaining := time.Until(*metadata.CleanupAt)
			if remaining > 0 {
				timeRemaining = formatDurationCompact(remaining)
			} else {
				timeRemaining = "EXPIRED"
			}
		}

		// Status icon
		statusIcon := "⭕"
		switch metadata.Status {
		case "ready", "up":
			statusIcon = "✅"
		case "error", "cleanup_failed":
			statusIcon = "❌"
		case "cleaning":
			statusIcon = "🔄"
		}

		fmt.Printf("%s %s - %s (TTL: %s)\n", 
			statusIcon, envName, metadata.Status, timeRemaining)

		// Show resource summary if available
		if len(metadata.TofuOutputs) > 0 {
			fmt.Printf("   ├── VPC: %s\n", getOutputOrNA(metadata.TofuOutputs, "vpc_id"))
			fmt.Printf("   ├── Batch: %s\n", getOutputOrNA(metadata.TofuOutputs, "batch_compute_env_arn"))
			fmt.Printf("   └── Lambda: %s\n", getOutputOrNA(metadata.TofuOutputs, "lambda_execution_role_arn"))
		}

		// Show scaling info if available
		if metadata.Vars != nil && metadata.Vars["instance_count"] != "" {
			fmt.Printf("   └── Workers: %s\n", metadata.Vars["instance_count"])
		}

		fmt.Println()
	}

	// Show auto-cleanup status
	autoCleanupManager := env.NewAutoCleanupManager(metadataStore, manager.GetTofuDir())
	if autoCleanupManager.IsRunning() {
		fmt.Println("🔄 Auto-cleanup: ACTIVE (checking every minute)")
	} else {
		fmt.Println("⏸️  Auto-cleanup: INACTIVE")
	}

	return nil
}

// handleMirrorCloudDown tears down a Mirror-Cloud environment
func handleMirrorCloudDown(ctx context.Context, manager *env.EnvironmentManager) error {
	metadataStore, err := env.NewMetadataStore("data/envs")
	if err != nil {
		return fmt.Errorf("failed to create metadata store: %w", err)
	}

	// Get environment name from args or use default
	envName := manager.GetEnvName()

	// Load metadata to confirm environment exists
	metadata, err := metadataStore.Load(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %s", envName)
	}

	fmt.Printf("🗑️  Tearing down environment: %s\n", envName)

	// Create tofu orchestrator
	tofuDir := manager.GetTofuDir()
	tofuOrchestrator := env.NewTofuOrchestrator(tofuDir, "", "")

	// Load vars from metadata
	vars := metadata.Vars
	if vars == nil {
		vars = make(map[string]string)
	}

	// Run tofu destroy
	fmt.Println("\n🔥 Destroying infrastructure (this may take 5-10 minutes)...")
	if err := tofuOrchestrator.Destroy(ctx); err != nil {
		// Update metadata with error
		metadataStore.Update(envName, map[string]interface{}{
			"status": "cleanup_failed",
			"error":  err.Error(),
		})
		return fmt.Errorf("teardown failed: %w", err)
	}
	fmt.Println("✅ Infrastructure destroyed")

	// Delete metadata
	if err := metadataStore.Delete(envName); err != nil {
		slog.Warn("failed to delete metadata", "env", envName, "error", err)
	}

	fmt.Printf(`
✅ Environment torn down: %s

📊 Summary:
   Name:   %s
   Status: destroyed
   TTL:    %s (was scheduled for %s)

💡 To create a new environment:
   cpctl env up mirror --ttl 4h
`,
		envName,
		envName,
		formatTTL(metadata.TTLSeconds),
		formatTimeOrNA(metadata.CleanupAt),
	)

	return nil
}

// handleMirrorCloudScale scales Mirror-Cloud resources
func handleMirrorCloudScale(ctx context.Context, envName string, workers int) error {
	metadataStore, err := env.NewMetadataStore("data/envs")
	if err != nil {
		return fmt.Errorf("failed to create metadata store: %w", err)
	}

	// Load metadata
	metadata, err := metadataStore.Load(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %s", envName)
	}

	// Check if environment is in a valid state for scaling
	if metadata.Status != "ready" && metadata.Status != "up" {
		return fmt.Errorf("environment is not in a ready state (current status: %s)", metadata.Status)
	}

	// Create scaler
	scaler := env.NewScaler(metadataStore, "tofu/mirror")

	// Scale options
	opts := env.ScaleOptions{
		EnvName: envName,
		Workers: workers,
		Confirm: false, // Ask for confirmation
	}

	// Validate options
	if err := scaler.ValidateScaleOptions(opts); err != nil {
		return fmt.Errorf("scale validation failed: %w", err)
	}

	// Perform scaling
	fmt.Printf("📈 Scaling environment %s to %d workers...\n", envName, workers)
	if err := scaler.Scale(ctx, opts); err != nil {
		return fmt.Errorf("scaling failed: %w", err)
	}

	// Get updated scale info
	scaleInfo, err := scaler.GetCurrentScale(envName)
	if err != nil {
		slog.Warn("failed to get updated scale info", "error", err)
	} else {
		fmt.Printf("\n✅ Scaling completed successfully!\n")
		fmt.Println("📊 Current configuration:")
		for key, value := range scaleInfo {
			fmt.Printf("   %s: %s\n", key, value)
		}
	}

	return nil
}

// handleLocalStackUp delegates to the existing cpctl up command logic.
func handleLocalStackUp(_ context.Context, _ *env.EnvironmentManager) error {
	return upCmd.RunE(upCmd, nil)
}

// handleLocalStackStatus shows LocalStack + Kind cluster health.
func handleLocalStackStatus(_ context.Context, _ *env.EnvironmentManager) error {
	return statusCmd.RunE(statusCmd, nil)
}

// handleLocalStackDown delegates to the existing cpctl down command logic.
func handleLocalStackDown(_ context.Context, _ *env.EnvironmentManager) error {
	return downCmd.RunE(downCmd, nil)
}

// Helper functions

func parseTTL(ttl string) (int, error) {
	if ttl == "" {
		return 0, nil
	}

	// Parse duration string like "4h", "30m", "24h"
	duration, err := time.ParseDuration(ttl)
	if err != nil {
		return 0, err
	}

	return int(duration.Seconds()), nil
}

func formatTTL(seconds int) string {
	if seconds == 0 {
		return "never"
	}
	duration := time.Duration(seconds) * time.Second
	return duration.String()
}

func formatDurationCompact(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

func getResourcesFromConfig(resources map[string]bool) []string {
	var result []string
	for resource, enabled := range resources {
		if enabled {
			result = append(result, resource)
		}
	}
	return result
}

func getOutputOrNA(outputs map[string]string, key string) string {
	if value, ok := outputs[key]; ok && value != "" {
		// Truncate long values
		if len(value) > 30 {
			return value[:27] + "..."
		}
		return value
	}
	return "N/A"
}

func formatTimeOrNA(t *time.Time) string {
	if t == nil {
		return "N/A"
	}
	return t.Format("2006-01-02 15:04:05")
}

func init() {
	// Add subcommands
	envCmd.AddCommand(envUpCmd)
	envCmd.AddCommand(envStatusCmd)
	envCmd.AddCommand(envDownCmd)
	envCmd.AddCommand(envScaleCmd)

	// Add stage flag to commands that need it
	envUpCmd.Flags().StringVar(&envStage, "stage", "", "Override stage (localstack or mirror)")
	envStatusCmd.Flags().StringVar(&envStage, "stage", "", "Override stage (localstack or mirror)")
	envDownCmd.Flags().StringVar(&envStage, "stage", "", "Override stage (localstack or mirror)")
	envScaleCmd.Flags().StringVar(&envStage, "stage", "", "Override stage (must be mirror for scaling)")

	// Add TTL flag for Mirror-Cloud
	envUpCmd.Flags().StringVar(&envTTL, "ttl", "", "Time-to-live for Mirror-Cloud (e.g., '4h', '24h')")

	// Add scale flag
	envScaleCmd.Flags().IntVar(&envScale, "workers", 0, "Number of workers to scale to (1-20)")

	// Register with root
	rootCmd.AddCommand(envCmd)
}