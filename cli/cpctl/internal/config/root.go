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
		// eindeutiger Root: birdy.yaml + kind + localstack
		if exists(filepath.Join(dir, "birdy.yaml")) &&
			exists(filepath.Join(dir, "kind")) &&
			exists(filepath.Join(dir, "localstack")) {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			panic("repo root not found (birdy.yaml, kind/, localstack/ required)")
		}
		dir = parent
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
