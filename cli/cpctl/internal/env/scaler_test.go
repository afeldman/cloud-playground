package env

import (
	"context"
	"testing"
	"time"
)

func TestScaler_ValidateScaleOptions(t *testing.T) {
	store, _ := NewMetadataStore(t.TempDir())
	scaler := NewScaler(store, "tofu/mirror")

	testCases := []struct {
		name    string
		opts    ScaleOptions
		wantErr bool
	}{
		{
			name: "valid scaling",
			opts: ScaleOptions{
				EnvName: "test-env",
				Workers: 5,
			},
			wantErr: false,
		},
		{
			name: "too few workers",
			opts: ScaleOptions{
				EnvName: "test-env",
				Workers: 0,
			},
			wantErr: true,
		},
		{
			name: "too many workers",
			opts: ScaleOptions{
				EnvName: "test-env",
				Workers: 25,
			},
			wantErr: true,
		},
		{
			name: "valid with instance type",
			opts: ScaleOptions{
				EnvName:      "test-env",
				Workers:      3,
				InstanceType: "t3.medium",
			},
			wantErr: false,
		},
		{
			name: "invalid instance type",
			opts: ScaleOptions{
				EnvName:      "test-env",
				Workers:      3,
				InstanceType: "invalid-type",
			},
			wantErr: true,
		},
		{
			name: "valid memory",
			opts: ScaleOptions{
				EnvName:  "test-env",
				Workers:  3,
				MemoryMB: 1024,
			},
			wantErr: false,
		},
		{
			name: "too little memory",
			opts: ScaleOptions{
				EnvName:  "test-env",
				Workers:  3,
				MemoryMB: 64,
			},
			wantErr: true,
		},
		{
			name: "too much memory",
			opts: ScaleOptions{
				EnvName:  "test-env",
				Workers:  3,
				MemoryMB: 20480,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := scaler.ValidateScaleOptions(tc.opts)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestScaler_GetCurrentScale(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	// Save metadata with scale vars
	metadata := &EnvironmentMetadata{
		Name:   "test-env",
		Status: "ready",
		Vars: map[string]string{
			"instance_count": "3",
			"instance_type":  "t3.medium",
			"memory_mb":      "1024",
			"other_var":      "value", // Should be ignored
		},
	}
	if err := store.Save("test-env", metadata); err != nil {
		t.Fatalf("Save: %v", err)
	}

	scaler := NewScaler(store, "tofu/mirror")
	scaleInfo, err := scaler.GetCurrentScale("test-env")
	if err != nil {
		t.Fatalf("GetCurrentScale: %v", err)
	}

	// Verify only scale-related vars are returned
	expectedVars := map[string]string{
		"instance_count": "3",
		"instance_type":  "t3.medium",
		"memory_mb":      "1024",
	}

	if len(scaleInfo) != len(expectedVars) {
		t.Errorf("scaleInfo length: got %d, want %d", len(scaleInfo), len(expectedVars))
	}

	for key, expectedValue := range expectedVars {
		actualValue, ok := scaleInfo[key]
		if !ok {
			t.Errorf("Missing scale var: %s", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Scale var %s: got %q, want %q", key, actualValue, expectedValue)
		}
	}

	// Verify other_var is not included
	if _, ok := scaleInfo["other_var"]; ok {
		t.Error("other_var should not be included in scale info")
	}
}

func TestScaler_GetCurrentScale_NoMetadata(t *testing.T) {
	store, _ := NewMetadataStore(t.TempDir())
	scaler := NewScaler(store, "tofu/mirror")

	// Environment doesn't exist
	scaleInfo, err := scaler.GetCurrentScale("non-existent-env")
	if err == nil {
		t.Error("GetCurrentScale should fail for non-existent environment")
	}
	if scaleInfo != nil {
		t.Error("scaleInfo should be nil on error")
	}
}

func TestScaler_GetScalingLimits(t *testing.T) {
	store, _ := NewMetadataStore(t.TempDir())
	scaler := NewScaler(store, "tofu/mirror")

	limits := scaler.GetScalingLimits()

	// Verify limits structure
	requiredKeys := []string{
		"max_workers",
		"min_workers",
		"max_memory_mb",
		"min_memory_mb",
		"max_ttl_hours",
		"confirmation_at",
		"allowed_instance_types",
	}

	for _, key := range requiredKeys {
		if _, ok := limits[key]; !ok {
			t.Errorf("Missing limit key: %s", key)
		}
	}

	// Verify specific values
	if maxWorkers, ok := limits["max_workers"].(int); !ok || maxWorkers != 20 {
		t.Errorf("max_workers: got %v, want 20", limits["max_workers"])
	}
	if minWorkers, ok := limits["min_workers"].(int); !ok || minWorkers != 1 {
		t.Errorf("min_workers: got %v, want 1", limits["min_workers"])
	}
	if confirmationAt, ok := limits["confirmation_at"].(int); !ok || confirmationAt != 10 {
		t.Errorf("confirmation_at: got %v, want 10", limits["confirmation_at"])
	}
}

func TestScaler_Scale_EnvironmentNotReady(t *testing.T) {
	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	// Save metadata with non-ready status
	metadata := &EnvironmentMetadata{
		Name:   "test-env",
		Status: "provisioning", // Not ready for scaling
	}
	if err := store.Save("test-env", metadata); err != nil {
		t.Fatalf("Save: %v", err)
	}

	scaler := NewScaler(store, "tofu/mirror")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := ScaleOptions{
		EnvName: "test-env",
		Workers: 5,
		Confirm: true, // Skip confirmation for test
	}

	err = scaler.Scale(ctx, opts)
	if err == nil {
		t.Error("Scale should fail for non-ready environment")
	}
}

// Note: Testing the actual Scale method with tofu operations would require
// mocking the tofu orchestrator, which is more complex. The above tests
// cover the validation and metadata interaction parts.

func TestNewScaler(t *testing.T) {
	store, _ := NewMetadataStore(t.TempDir())
	scaler := NewScaler(store, "tofu/mirror")

	if scaler == nil {
		t.Fatal("NewScaler returned nil")
	}
	if scaler.metadataStore != store {
		t.Error("metadataStore not set correctly")
	}
	if scaler.tofuDir != "tofu/mirror" {
		t.Errorf("tofuDir: got %q, want %q", scaler.tofuDir, "tofu/mirror")
	}
}