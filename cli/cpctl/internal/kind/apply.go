package kind

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"cpctl/internal/exec"
)

func ApplyYAML(yaml string) error {
	if yaml == "" {
		return nil
	}

	return exec.RunWithStdin(
		yaml,
		"kubectl",
		"apply",
		"-f",
		"-",
	)
}

func ApplyManifests(root string) error {
	var files []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if len(files) == 0 {
		slog.Warn("no manifests found", "root", root)
		return nil
	}

	for _, file := range files {
		slog.Info("applying manifest", "file", file)
		if err := exec.Run("kubectl", "apply", "-f", file); err != nil {
			return err
		}
	}

	return nil
}
