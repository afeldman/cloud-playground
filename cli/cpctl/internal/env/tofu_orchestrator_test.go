package env

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestOrchestrator(t *testing.T) (*TofuOrchestrator, string) {
	t.Helper()
	dir := t.TempDir()
	to := NewTofuOrchestrator(dir, "", "")
	return to, dir
}

func TestTofuOrchestrator_GenerateComputedVars(t *testing.T) {
	to, dir := newTestOrchestrator(t)

	vars := map[string]string{
		"aws_profile":       "test-profile",
		"aws_region":        "eu-central-1",
		"instance_count":    "3",
		"enable_compute":    "true",
		"auto_teardown_ttl": "4h",
	}

	if err := to.GenerateComputedVars(vars); err != nil {
		t.Fatalf("GenerateComputedVars: %v", err)
	}

	// Verify file was created
	computedFile := filepath.Join(dir, "computed.tfvars")
	if _, err := os.Stat(computedFile); os.IsNotExist(err) {
		t.Fatal("computed.tfvars file not created")
	}

	// Read and verify content
	content, err := os.ReadFile(computedFile)
	if err != nil {
		t.Fatalf("Read computed.tfvars: %v", err)
	}

	contentStr := string(content)
	expectedLines := []string{
		"aws_profile = \"test-profile\"",
		"aws_region = \"eu-central-1\"",
		"instance_count = 3",
		"enable_compute = true",
		"auto_teardown_ttl = \"4h\"",
	}

	for _, expectedLine := range expectedLines {
		if !containsLine(contentStr, expectedLine) {
			t.Errorf("computed.tfvars missing line: %s", expectedLine)
		}
	}
}

func TestTofuOrchestrator_GetCurrentVars(t *testing.T) {
	to, dir := newTestOrchestrator(t)

	// Create computed.tfvars file
	computedFile := filepath.Join(dir, "computed.tfvars")
	content := `aws_profile = "test-profile"
aws_region = "eu-central-1"
instance_count = 3
enable_compute = true
auto_teardown_ttl = "4h"
`
	if err := os.WriteFile(computedFile, []byte(content), 0644); err != nil {
		t.Fatalf("Write computed.tfvars: %v", err)
	}

	// Get current vars
	vars, err := to.GetCurrentVars()
	if err != nil {
		t.Fatalf("GetCurrentVars: %v", err)
	}

	// Verify vars
	expectedVars := map[string]string{
		"aws_profile":       "test-profile",
		"aws_region":        "eu-central-1",
		"instance_count":    "3",
		"enable_compute":    "true",
		"auto_teardown_ttl": "4h",
	}

	for key, expectedValue := range expectedVars {
		actualValue, ok := vars[key]
		if !ok {
			t.Errorf("Missing variable: %s", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Variable %s: got %q, want %q", key, actualValue, expectedValue)
		}
	}
}

func TestTofuOrchestrator_Init(t *testing.T) {
	writeFakeTofu(t, 0)
	to, _ := newTestOrchestrator(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := to.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
}

func TestTofuOrchestrator_Plan(t *testing.T) {
	writeFakeTofu(t, 0)
	to, _ := newTestOrchestrator(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	vars := map[string]string{
		"instance_count": "3",
	}

	if err := to.Plan(ctx, vars); err != nil {
		t.Fatalf("Plan: %v", err)
	}
}

func TestTofuOrchestrator_Apply(t *testing.T) {
	writeFakeTofu(t, 0)
	to, _ := newTestOrchestrator(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	vars := map[string]string{
		"instance_count": "3",
	}

	if err := to.Apply(ctx, vars); err != nil {
		t.Fatalf("Apply: %v", err)
	}
}

func TestTofuOrchestrator_Destroy(t *testing.T) {
	writeFakeTofu(t, 0)
	to, _ := newTestOrchestrator(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := to.Destroy(ctx); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
}

func TestTofuOrchestrator_Validate(t *testing.T) {
	writeFakeTofu(t, 0)
	to, _ := newTestOrchestrator(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := to.Validate(ctx); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestTofuOrchestrator_GetOutputs(t *testing.T) {
	// Create a fake tofu that outputs JSON
	dir := t.TempDir()
	script := filepath.Join(dir, "tofu")
	content := `#!/bin/sh
if [ "$1" = "output" ] && [ "$2" = "-json" ]; then
  echo '{
    "vpc_id": {
      "value": "vpc-12345678"
    },
    "batch_compute_env_arn": {
      "value": "arn:aws:batch:eu-central-1:123456789012:compute-environment/test"
    },
    "public_subnets": {
      "value": ["subnet-123", "subnet-456"]
    }
  }'
  exit 0
else
  echo "fake-tofu $@"
  exit 0
fi
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatalf("write fake tofu: %v", err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	to, _ := newTestOrchestrator(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	outputs, err := to.GetOutputs(ctx)
	if err != nil {
		t.Fatalf("GetOutputs: %v", err)
	}

	// Verify outputs
	expectedOutputs := map[string]string{
		"vpc_id":                 "vpc-12345678",
		"batch_compute_env_arn":  "arn:aws:batch:eu-central-1:123456789012:compute-environment/test",
		"public_subnets":         `["subnet-123","subnet-456"]`,
	}

	for key, expectedValue := range expectedOutputs {
		actualValue, ok := outputs[key]
		if !ok {
			t.Errorf("Missing output: %s", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Output %s: got %q, want %q", key, actualValue, expectedValue)
		}
	}
}

func TestTofuOrchestrator_Timeout(t *testing.T) {
	// Create a fake tofu that sleeps to trigger timeout
	dir := t.TempDir()
	script := filepath.Join(dir, "tofu")
	content := `#!/bin/sh
sleep 10
echo "fake-tofu $@"
exit 0
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatalf("write fake tofu: %v", err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	to, _ := newTestOrchestrator(t)
	to.SetTimeout(100 * time.Millisecond) // Very short timeout

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := to.Apply(ctx, nil); err == nil {
		t.Fatal("Apply should timeout")
	}
}

func TestTofuOrchestrator_GetResourceCount(t *testing.T) {
	// Create a fake tofu that outputs JSON state
	dir := t.TempDir()
	script := filepath.Join(dir, "tofu")
	content := `#!/bin/sh
if [ "$1" = "show" ] && [ "$2" = "-json" ]; then
  echo '{
    "values": {
      "root_module": {
        "resources": [
          {"address": "aws_vpc.test"},
          {"address": "aws_subnet.public[0]"},
          {"address": "aws_subnet.public[1]"},
          {"address": "aws_batch_compute_environment.test"}
        ]
      }
    }
  }'
  exit 0
else
  echo "fake-tofu $@"
  exit 0
fi
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatalf("write fake tofu: %v", err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	to, _ := newTestOrchestrator(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := to.GetResourceCount(ctx)
	if err != nil {
		t.Fatalf("GetResourceCount: %v", err)
	}

	if count != 4 {
		t.Errorf("Resource count: got %d, want 4", count)
	}
}

// Helper function to check if a line exists in content
func containsLine(content, line string) bool {
	lines := splitLines(content)
	for _, l := range lines {
		if l == line {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}