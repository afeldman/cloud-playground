package tunnels

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestManager(t *testing.T) (*TunnelManager, string) {
	t.Helper()
	dir := t.TempDir()
	tm := &TunnelManager{DataDir: filepath.Join(dir, "tunnels")}
	return tm, dir
}

func TestTunnelManager_SaveAndLoad(t *testing.T) {
	tm, _ := newTestManager(t)

	info := TunnelInfo{
		Name:       "pg",
		PID:        12345,
		Type:       "kubernetes",
		LocalPort:  5432,
		RemoteHost: "postgres.default.svc.cluster.local",
		RemotePort: 5432,
	}

	if err := tm.Save(info); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := tm.Load("pg")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.Name != info.Name {
		t.Errorf("Name: got %q want %q", got.Name, info.Name)
	}
	if got.PID != info.PID {
		t.Errorf("PID: got %d want %d", got.PID, info.PID)
	}
	if got.Type != info.Type {
		t.Errorf("Type: got %q want %q", got.Type, info.Type)
	}
	if got.LocalPort != info.LocalPort {
		t.Errorf("LocalPort: got %d want %d", got.LocalPort, info.LocalPort)
	}
	if got.RemoteHost != info.RemoteHost {
		t.Errorf("RemoteHost: got %q want %q", got.RemoteHost, info.RemoteHost)
	}
	if got.RemotePort != info.RemotePort {
		t.Errorf("RemotePort: got %d want %d", got.RemotePort, info.RemotePort)
	}
}

func TestTunnelManager_Load_NotFound(t *testing.T) {
	tm, _ := newTestManager(t)
	_, err := tm.Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing tunnel, got nil")
	}
}

func TestTunnelManager_List_Empty(t *testing.T) {
	tm, _ := newTestManager(t)
	tunnels, err := tm.List()
	if err != nil {
		t.Fatalf("List on empty dir: %v", err)
	}
	if len(tunnels) != 0 {
		t.Errorf("expected 0 tunnels, got %d", len(tunnels))
	}
}

func TestTunnelManager_List(t *testing.T) {
	tm, _ := newTestManager(t)

	for _, name := range []string{"pg", "rds"} {
		if err := tm.Save(TunnelInfo{Name: name, PID: 1, Type: "kubernetes", LocalPort: 5432}); err != nil {
			t.Fatalf("Save %s: %v", name, err)
		}
	}

	tunnels, err := tm.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tunnels) != 2 {
		t.Errorf("expected 2 tunnels, got %d", len(tunnels))
	}
}

func TestTunnelManager_Delete(t *testing.T) {
	tm, _ := newTestManager(t)

	if err := tm.Save(TunnelInfo{Name: "pg", PID: 1, Type: "kubernetes", LocalPort: 5432}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := tm.Delete("pg"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify files are gone
	pidFile := filepath.Join(tm.DataDir, "pg.pid")
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("pid file should be deleted")
	}

	// List should be empty
	tunnels, _ := tm.List()
	if len(tunnels) != 0 {
		t.Errorf("expected 0 tunnels after delete, got %d", len(tunnels))
	}
}

func TestTunnelManager_Delete_Idempotent(t *testing.T) {
	tm, _ := newTestManager(t)
	// Delete non-existent should not error
	if err := tm.Delete("nonexistent"); err != nil {
		t.Errorf("Delete of nonexistent should not error: %v", err)
	}
}
