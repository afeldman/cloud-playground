package sync

import (
	"fmt"
	"log/slog"
	"cpctl/internal/exec"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func DiffConfigMap(ns string, data map[string]string) error {
	live, err := exec.Capture(
		"kubectl", "-n", ns,
		"get", "configmap", "birdy-config", "-o", "yaml",
	)
	if err != nil {
		slog.Info("ℹ️  ConfigMap does not exist yet")
	}

	rendered, err := RenderConfigMapYAML(ns, data)
	if err != nil {
		return err
	}

	return PrintDiff(live, rendered)
}

func PrintDiff(live string, rendered string) error {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(live, rendered, false)

	if len(diffs) == 1 && diffs[0].Type == diffmatchpatch.DiffEqual {
		slog.Info("✅ no changes")
		return nil
	}

	fmt.Println("🔍 diff:")
	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffInsert:
			printLines("+", d.Text)
		case diffmatchpatch.DiffDelete:
			printLines("-", d.Text)
		}
	}

	return nil
}

func printLines(prefix, text string) {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fmt.Printf("%s %s\n", prefix, line)
	}
}

type DiffResult struct {
	HasChanges bool
}
