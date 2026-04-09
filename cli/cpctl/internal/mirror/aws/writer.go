package aws

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// NormalizeKey converts an SSM parameter name into a flat Kubernetes-safe key.
// Example:
//
//	/birdy/services/JWT_SECRET -> JWT_SECRET
func NormalizeKey(path, full string) (string, error) {
	key := strings.TrimPrefix(full, path)
	key = strings.TrimPrefix(key, "/")

	if strings.Contains(key, "/") {
		return "", fmt.Errorf(
			"nested SSM parameter paths are not supported: %s",
			full,
		)
	}

	return key, nil
}

// writeSecretYAML renders a Kubernetes Secret manifest from parameters
// and writes it to disk in a deterministic and secure way.
func writeSecretYAML(
	outDir string,
	namespace string,
	name string,
	params []Parameter,
) error {

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	encoded := map[string][]byte{}
	for _, p := range params {
		encoded[p.Key] = []byte(
			base64.StdEncoding.EncodeToString([]byte(p.Value)),
		)
	}

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"birdy.io/source": "aws-ssm",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: encoded,
	}

	out, err := yaml.Marshal(secret)
	if err != nil {
		return err
	}

	path := filepath.Join(outDir, name+".yaml")
	return os.WriteFile(path, out, 0o600)
}
