package sync

import "cpctl/internal/kind"

func EnsureNamespace(name string) error {
	yaml := `
apiVersion: v1
kind: Namespace
metadata:
  name: ` + name

	return kind.ApplyYAML(yaml)
}
