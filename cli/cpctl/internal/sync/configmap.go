package sync

import (
	"log/slog"
	"strings"

	"cpctl/internal/exec"
)

func ApplyConfigMap(input SyncInput) error {
	yaml := RenderConfigMap(input.Namespace, "birdy-config", input.Config)

	slog.Info("📦 applying ConfigMap birdy-config")
	return exec.RunWithStdin(yaml, "kubectl", "apply", "-f", "-")
}

func GetConfigMapChecksum(namespace, name string) (string, error) {
	out, err := exec.Capture(
		"kubectl", "get", "configmap", name,
		"-n", namespace,
		"-o", "jsonpath={.metadata.annotations.birdy\\.io/checksum}",
	)
	if err != nil {
		return "", nil // ConfigMap existiert nicht
	}
	return strings.TrimSpace(out), nil
}
