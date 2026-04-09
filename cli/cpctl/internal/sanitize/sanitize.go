package sanitize

import (
	"log/slog"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

func SanitizeDir(dir string) error {
	slog.Info("🧼 sanitizing manifests in", slog.String("dir", dir))

	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		return sanitizeFile(path)
	})
}

func sanitizeFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var obj map[string]interface{}
	if err := yaml.Unmarshal(raw, &obj); err != nil {
		return err
	}

	kind, _ := obj["kind"].(string)

	// ─────────────────────────────────────────
	// Handle kind: List
	// ─────────────────────────────────────────
	if kind == "List" {
		items, ok := obj["items"].([]interface{})
		if !ok {
			return nil
		}

		newItems := []interface{}{}

		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			if sanitizeObject(m) {
				newItems = append(newItems, m)
			}
		}

		if len(newItems) == 0 {
			slog.Info("🗑 removing empty List: ",
				slog.String("path", path))
			return os.Remove(path)
		}

		obj["items"] = newItems
		out, _ := yaml.Marshal(obj)
		return os.WriteFile(path, out, 0644)
	}

	// ─────────────────────────────────────────
	// Single object
	// ─────────────────────────────────────────
	if !sanitizeObject(obj) {
		slog.Info("🗑 removing object:", slog.String("kind", kind), slog.String("path", path))
		return os.Remove(path)
	}

	out, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}
