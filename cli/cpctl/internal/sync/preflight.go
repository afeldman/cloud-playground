package sync

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"cpctl/internal/exec"
)

func Preflight(input SyncInput) error {
	if input.DryRun {
		slog.Info("🧪 dry-run: skipping preflight checks")
		return nil
	}

	// 1️⃣ kubectl vorhanden?
	if err := exec.Run("kubectl", "version", "--client"); err != nil {
		return errors.New("kubectl not available")
	}

	// 2️⃣ Context prüfen
	out, err := exec.Capture("kubectl", "config", "current-context")
	if err != nil {
		return errors.New("cannot determine kubectl context")
	}

	ctx := strings.TrimSpace(out)
	if !strings.Contains(ctx, "birdy-playground") {
		return fmt.Errorf("unsafe kubectl context: %s", ctx)
	}

	// 3️⃣ Cluster erreichbar?
	if err := exec.Run("kubectl", "get", "ns"); err != nil {
		return errors.New("cluster not reachable")
	}

	slog.Info("✅ preflight checks passed")
	return nil
}
