package env

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeFakeTofu writes a fake "tofu" script that exits 0 and prints a marker.
// Returns the dir that should be prepended to PATH.
func writeFakeTofu(t *testing.T, exitCode int) string {
	t.Helper()
	dir := t.TempDir()

	var content string
	if runtime.GOOS == "windows" {
		t.Skip("fake tofu not supported on Windows in this test")
	}
	content = "#!/bin/sh\necho 'fake-tofu' \"$@\"\nexit " + itoa(exitCode) + "\n"

	script := filepath.Join(dir, "tofu")
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatalf("write fake tofu: %v", err)
	}

	// Prepend to PATH so exec.Command finds it
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	return dir
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	return "1"
}

func newTestProvisioner(t *testing.T) (*TofuProvisioner, string) {
	t.Helper()
	dir := t.TempDir()
	tp := NewTofuProvisioner(dir, "", "")
	return tp, dir
}

func TestTofuProvisioner_Init_Success(t *testing.T) {
	writeFakeTofu(t, 0)
	tp, _ := newTestProvisioner(t)
	if err := tp.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}
}

func TestTofuProvisioner_Plan_Success(t *testing.T) {
	writeFakeTofu(t, 0)
	tp, _ := newTestProvisioner(t)
	if err := tp.Plan(context.Background()); err != nil {
		t.Fatalf("Plan: %v", err)
	}
}

func TestTofuProvisioner_Apply_Success(t *testing.T) {
	writeFakeTofu(t, 0)
	tp, _ := newTestProvisioner(t)
	if err := tp.Apply(context.Background()); err != nil {
		t.Fatalf("Apply: %v", err)
	}
}

func TestTofuProvisioner_Destroy_Success(t *testing.T) {
	writeFakeTofu(t, 0)
	tp, _ := newTestProvisioner(t)
	if err := tp.Destroy(context.Background()); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
}

func TestTofuProvisioner_Validate_Success(t *testing.T) {
	writeFakeTofu(t, 0)
	tp, _ := newTestProvisioner(t)
	if err := tp.Validate(context.Background()); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestTofuProvisioner_Init_Failure(t *testing.T) {
	writeFakeTofu(t, 1)
	tp, _ := newTestProvisioner(t)
	if err := tp.Init(context.Background()); err == nil {
		t.Fatal("expected error when tofu exits non-zero")
	}
}

func TestTofuProvisioner_Apply_WithVarFile(t *testing.T) {
	writeFakeTofu(t, 0)
	tp, dir := newTestProvisioner(t)

	varFile := filepath.Join(dir, "test.tfvars.json")
	if err := os.WriteFile(varFile, []byte(`{"key": "val"}`), 0644); err != nil {
		t.Fatalf("write varfile: %v", err)
	}
	tp.varFile = varFile

	if err := tp.Apply(context.Background()); err != nil {
		t.Fatalf("Apply with varfile: %v", err)
	}
}

func TestTofuProvisioner_NewDefaults(t *testing.T) {
	tp := NewTofuProvisioner("/some/dir", "", "")
	if tp.stateFile != "terraform.tfstate" {
		t.Errorf("expected default stateFile, got %q", tp.stateFile)
	}
	if tp.timeout <= 0 {
		t.Error("expected positive default timeout")
	}
}
