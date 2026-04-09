package sync

import (
	"os"
	"path/filepath"
	"strings"
)

func LoadParams(dir string) (map[string]string, error) {
	out := map[string]string{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return out, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return out, err
		}

		out[e.Name()] = strings.TrimSpace(string(b))
	}

	return out, nil
}

func LoadSecrets(dir string) (map[string]string, error) {
	out := map[string]string{}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return out, nil // ← wichtig
	}
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out[e.Name()] = strings.TrimSpace(string(b))
	}

	return out, nil
}
