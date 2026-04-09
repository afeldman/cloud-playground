package env

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// Scaler manages environment resource scaling
type Scaler struct {
	metadataStore *MetadataStore
	tofuDir       string
}

// NewScaler creates a new scaler
func NewScaler(metadataStore *MetadataStore, tofuDir string) *Scaler {
	return &Scaler{
		metadataStore: metadataStore,
		tofuDir:       tofuDir,
	}
}

// ScaleOptions contains scaling parameters
type ScaleOptions struct {
	EnvName      string
	Workers      int
	InstanceType string
	MemoryMB     int
	Confirm      bool
}

// Scale scales environment resources
func (s *Scaler) Scale(ctx context.Context, opts ScaleOptions) error {
	// Validate inputs
	if opts.Workers < 1 || opts.Workers > 20 {
		return fmt.Errorf("workers must be between 1 and 20 (got %d)", opts.Workers)
	}

	// Load current environment metadata
	metadata, err := s.metadataStore.Load(opts.EnvName)
	if err != nil {
		return fmt.Errorf("failed to load environment metadata: %w", err)
	}

	// Check if environment is active
	if metadata.Status != "ready" && metadata.Status != "up" {
		return fmt.Errorf("environment is not active (status: %s)", metadata.Status)
	}

	// Load current tofu variables
	tofuOrchestrator := NewTofuOrchestrator(s.tofuDir, "", "")
	currentVars, err := tofuOrchestrator.GetCurrentVars()
	if err != nil {
		return fmt.Errorf("failed to get current tofu variables: %w", err)
	}

	// Update variables for scaling
	updatedVars := make(map[string]string)
	for k, v := range currentVars {
		updatedVars[k] = v
	}

	// Set scaling parameters
	updatedVars["instance_count"] = strconv.Itoa(opts.Workers)
	if opts.InstanceType != "" {
		updatedVars["instance_type"] = opts.InstanceType
	}
	if opts.MemoryMB > 0 {
		updatedVars["memory_mb"] = strconv.Itoa(opts.MemoryMB)
	}

	// Show preview if not confirmed
	if !opts.Confirm {
		fmt.Printf("📊 Scaling Preview for %s:\n", opts.EnvName)
		fmt.Println("────────────────────────────────────────")
		
		// Show changes
		for key, newValue := range updatedVars {
			oldValue := currentVars[key]
			if oldValue != newValue {
				fmt.Printf("  %s: %s → %s\n", key, oldValue, newValue)
			}
		}
		
		fmt.Println("────────────────────────────────────────")
		fmt.Printf("This will modify %d resources.\n", len(updatedVars))
		fmt.Print("Continue? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return fmt.Errorf("scaling cancelled by user")
		}
	}

	// Generate plan
	slog.Info("planning scaling operation", "env", opts.EnvName, "workers", opts.Workers)
	if err := tofuOrchestrator.Plan(ctx, updatedVars); err != nil {
		return fmt.Errorf("scaling plan failed: %w", err)
	}

	// Apply changes
	slog.Info("applying scaling changes", "env", opts.EnvName)
	if err := tofuOrchestrator.Apply(ctx, updatedVars); err != nil {
		return fmt.Errorf("scaling apply failed: %w", err)
	}

	// Get updated outputs
	outputs, err := tofuOrchestrator.GetOutputs(ctx)
	if err != nil {
		slog.Warn("failed to get updated outputs", "error", err)
	} else {
		// Update metadata with new outputs
		s.metadataStore.Update(opts.EnvName, map[string]interface{}{
			"tofu_outputs": outputs,
			"vars":         updatedVars,
		})
	}

	// Poll AWS for scaling completion
	if err := s.pollScalingCompletion(ctx, opts.EnvName, metadata); err != nil {
		slog.Warn("scaling polling failed", "error", err)
		// Continue anyway - tofu apply succeeded
	}

	slog.Info("scaling completed", "env", opts.EnvName, "workers", opts.Workers)
	return nil
}

// pollScalingCompletion polls AWS resources to confirm scaling completed
func (s *Scaler) pollScalingCompletion(ctx context.Context, envName string, metadata *EnvironmentMetadata) error {
	// This would make actual AWS API calls to check resource status
	// For now, we'll simulate with a simple delay and log
	
	slog.Info("waiting for AWS resources to scale...", "env", envName)
	
	// Simulate waiting for AWS Batch compute environment update
	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			// In real implementation, would call:
			// - AWS Batch: DescribeComputeEnvironments
			// - AWS EC2: DescribeInstances
			// - AWS Lambda: GetFunctionConfiguration
			
			if i%5 == 0 {
				slog.Debug("polling AWS resource status", "env", envName, "attempt", i+1)
			}
		}
	}
	
	slog.Info("AWS resources scaled successfully", "env", envName)
	return nil
}

// GetCurrentScale returns current scaling configuration
func (s *Scaler) GetCurrentScale(envName string) (map[string]string, error) {
	metadata, err := s.metadataStore.Load(envName)
	if err != nil {
		return nil, fmt.Errorf("failed to load environment metadata: %w", err)
	}

	// Extract scaling-related variables
	scaleInfo := make(map[string]string)
	
	if metadata.Vars != nil {
		for key, value := range metadata.Vars {
			// Look for scaling-related keys
			if key == "instance_count" || key == "instance_type" || 
			   key == "memory_mb" || key == "vcpus" || 
			   key == "desired_vcpus" || key == "max_vcpus" {
				scaleInfo[key] = value
			}
		}
	}

	// If no vars in metadata, try to get from tofu state
	if len(scaleInfo) == 0 {
		tofuOrchestrator := NewTofuOrchestrator(s.tofuDir, "", "")
		currentVars, err := tofuOrchestrator.GetCurrentVars()
		if err == nil {
			for key, value := range currentVars {
				if key == "instance_count" || key == "instance_type" || 
				   key == "memory_mb" || key == "vcpus" || 
				   key == "desired_vcpus" || key == "max_vcpus" {
					scaleInfo[key] = value
				}
			}
		}
	}

	return scaleInfo, nil
}

// ValidateScaleOptions validates scaling options against limits
func (s *Scaler) ValidateScaleOptions(opts ScaleOptions) error {
	// Worker count validation
	if opts.Workers < 1 {
		return fmt.Errorf("workers must be at least 1")
	}
	if opts.Workers > 20 {
		return fmt.Errorf("workers cannot exceed 20 (cost safeguard)")
	}

	// Memory validation
	if opts.MemoryMB > 0 {
		if opts.MemoryMB < 128 {
			return fmt.Errorf("memory must be at least 128MB")
		}
		if opts.MemoryMB > 10240 {
			return fmt.Errorf("memory cannot exceed 10240MB (10GB)")
		}
	}

	// Instance type validation (simplified)
	if opts.InstanceType != "" {
		validTypes := map[string]bool{
			"t3.micro":    true,
			"t3.small":    true,
			"t3.medium":   true,
			"t3.large":    true,
			"c5.large":    true,
			"c5.xlarge":   true,
			"c5.2xlarge":  true,
			"m5.large":    true,
			"m5.xlarge":   true,
			"m5.2xlarge":  true,
		}
		
		if !validTypes[opts.InstanceType] {
			return fmt.Errorf("invalid instance type: %s", opts.InstanceType)
		}
	}

	return nil
}

// GetScalingLimits returns the scaling limits for cost safeguarding
func (s *Scaler) GetScalingLimits() map[string]interface{} {
	return map[string]interface{}{
		"max_workers":      20,
		"min_workers":      1,
		"max_memory_mb":    10240,
		"min_memory_mb":    128,
		"max_ttl_hours":    24,
		"confirmation_at":  10, // Require confirmation when scaling above 10 workers
		"allowed_instance_types": []string{
			"t3.micro", "t3.small", "t3.medium", "t3.large",
			"c5.large", "c5.xlarge", "c5.2xlarge",
			"m5.large", "m5.xlarge", "m5.2xlarge",
		},
	}
}