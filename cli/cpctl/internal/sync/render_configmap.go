package sync

import (
	"fmt"
	"cpctl/internal/exec"

	"sigs.k8s.io/yaml"
)

func RenderConfigMap(namespace, name string, data map[string]string) string {
	checksum := HashMap(data)

	yaml := fmt.Sprintf(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  namespace: %s
  annotations:
    birdy.io/checksum: "%s"
data:
`, name, namespace, checksum)

	for k, v := range data {
		yaml += fmt.Sprintf("  %s: %q\n", k, v)
	}

	return yaml
}

func RenderConfigMapYAML(namespace string, data map[string]string) (string, error) {
	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "birdy-config",
			"namespace": namespace,
			"annotations": map[string]string{
				"birdy.io/checksum": HashMap(data),
			},
		},
		"data": data,
	}

	out, err := yaml.Marshal(obj)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func GetLiveConfigMapYAML(ns, name string) (string, error) {
	return exec.Capture(
		"kubectl", "-n", ns,
		"get", "configmap", name,
		"-o", "yaml",
	)
}
