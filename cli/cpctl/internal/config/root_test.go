package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepoRootUsesCPCTLRoot(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "kind"), 0o755); err != nil {
		t.Fatalf("mkdir kind: %v", err)
	}

	t.Setenv("CPCTL_ROOT", tmp)

	got := RepoRoot()
	if got != tmp {
		t.Fatalf("RepoRoot() = %q, want %q", got, tmp)
	}
}