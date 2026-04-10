package config

import (
	"os"
	"path/filepath"
)

func RepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	for {
		// eindeutiger Root: .cpctl.yaml oder birdy.yaml + kind/
		hasConfig := exists(filepath.Join(dir, ".cpctl.yaml")) || exists(filepath.Join(dir, "birdy.yaml"))
		if hasConfig && exists(filepath.Join(dir, "kind")) {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			panic("repo root not found (.cpctl.yaml or birdy.yaml + kind/ required)")
		}
		dir = parent
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
