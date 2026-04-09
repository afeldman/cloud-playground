package env

import (
	"context"
	"path/filepath"
	"testing"

	"cpctl/internal/config"
)

func newTestManager(t *testing.T) (*EnvironmentManager, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{}
	cfg.Playground.DataDir = dir
	cfg.Development.Stage = "localstack"

	em, err := NewEnvironmentManager(filepath.Join(dir, "envs"), cfg)
	if err != nil {
		t.Fatalf("NewEnvironmentManager: %v", err)
	}
	return em, dir
}

func TestEnvironmentManager_DefaultStage(t *testing.T) {
	em, _ := newTestManager(t)
	if em.GetStage() != StageLocalStack {
		t.Errorf("expected stage localstack, got %v", em.GetStage())
	}
}

func TestEnvironmentManager_SetStage(t *testing.T) {
	em, _ := newTestManager(t)

	if err := em.SetStage(StageMirror); err != nil {
		t.Fatalf("SetStage mirror: %v", err)
	}
	if em.GetStage() != StageMirror {
		t.Errorf("expected mirror, got %v", em.GetStage())
	}
}

func TestEnvironmentManager_SetStage_Invalid(t *testing.T) {
	em, _ := newTestManager(t)
	if err := em.SetStage("production"); err == nil {
		t.Error("expected error for invalid stage")
	}
}

func TestEnvironmentManager_GetTofuDir(t *testing.T) {
	em, _ := newTestManager(t)

	cases := []struct {
		stage Stage
		want  string
	}{
		{StageLocalStack, "tofu/localstack"},
		{StageMirror, "tofu/mirror"},
	}

	for _, tc := range cases {
		em.stage = tc.stage
		got := em.GetTofuDir()
		if got != tc.want {
			t.Errorf("stage=%s: got %q want %q", tc.stage, got, tc.want)
		}
	}
}

func TestEnvironmentManager_WriteAndReadState(t *testing.T) {
	em, _ := newTestManager(t)
	ctx := context.Background()

	state := &EnvironmentState{
		Stage:         StageLocalStack,
		Status:        "ready",
		ResourceCount: 5,
		TunnelCount:   2,
	}

	if err := em.WriteState(ctx, state); err != nil {
		t.Fatalf("WriteState: %v", err)
	}

	got, err := em.ReadState(ctx)
	if err != nil {
		t.Fatalf("ReadState: %v", err)
	}

	if got.Status != "ready" {
		t.Errorf("Status: got %q want %q", got.Status, "ready")
	}
	if got.ResourceCount != 5 {
		t.Errorf("ResourceCount: got %d want %d", got.ResourceCount, 5)
	}
	if got.TunnelCount != 2 {
		t.Errorf("TunnelCount: got %d want %d", got.TunnelCount, 2)
	}
}

func TestEnvironmentManager_ReadState_NonExistent(t *testing.T) {
	em, _ := newTestManager(t)
	ctx := context.Background()

	state, err := em.ReadState(ctx)
	if err != nil {
		t.Fatalf("ReadState for missing file should not error: %v", err)
	}
	if state.Status != "down" {
		t.Errorf("expected default status 'down', got %q", state.Status)
	}
}

func TestEnvironmentManager_DeleteState(t *testing.T) {
	em, _ := newTestManager(t)
	ctx := context.Background()

	if err := em.WriteState(ctx, &EnvironmentState{Status: "ready"}); err != nil {
		t.Fatalf("WriteState: %v", err)
	}
	if err := em.DeleteState(ctx); err != nil {
		t.Fatalf("DeleteState: %v", err)
	}

	// After deletion, ReadState should return default "down"
	state, err := em.ReadState(ctx)
	if err != nil {
		t.Fatalf("ReadState after delete: %v", err)
	}
	if state.Status != "down" {
		t.Errorf("expected 'down' after delete, got %q", state.Status)
	}
}

func TestEnvironmentManager_DeleteState_Idempotent(t *testing.T) {
	em, _ := newTestManager(t)
	ctx := context.Background()
	if err := em.DeleteState(ctx); err != nil {
		t.Errorf("DeleteState non-existent should not error: %v", err)
	}
}

func TestEnvironmentManager_GetStageName(t *testing.T) {
	em, _ := newTestManager(t)

	em.stage = StageLocalStack
	if name := em.GetStageName(); name == "" {
		t.Error("expected non-empty stage name for localstack")
	}

	em.stage = StageMirror
	if name := em.GetStageName(); name == "" {
		t.Error("expected non-empty stage name for mirror")
	}
}
