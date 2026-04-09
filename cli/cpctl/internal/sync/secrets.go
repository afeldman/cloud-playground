package sync

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"cpctl/internal/exec"

	"sigs.k8s.io/yaml"
)

/*
|--------------------------------------------------------------------------
| APPLY
|--------------------------------------------------------------------------
*/

func ApplySecrets(namespace string, data map[string]string) error {
	yamlOut, checksum, err := RenderSecret(namespace, data)
	if err != nil {
		return err
	}

	current, _ := exec.Capture(
		"kubectl", "get", "secret", "birdy-secrets",
		"-n", namespace,
		"-o", "jsonpath={.metadata.annotations.birdy\\.io/checksum}",
	)

	if current == checksum {
		fmt.Println("⏭️ Secret unchanged – skipping apply")
		return nil
	}

	fmt.Println("🔐 applying Secret birdy-secrets")
	return exec.RunWithStdin(yamlOut, "kubectl", "apply", "-f", "-")
}

/*
|--------------------------------------------------------------------------
| VIEW: live cluster
|--------------------------------------------------------------------------
*/

func BuildSecretViewFromCluster(ns, name string) (*SecretView, error) {
	out, err := exec.Capture(
		"kubectl", "-n", ns,
		"get", "secret", name,
		"-o", "json",
	)
	if err != nil {
		return nil, err
	}

	var obj struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Data map[string]string `json:"data"`
	}

	if err := json.Unmarshal([]byte(out), &obj); err != nil {
		return nil, err
	}

	view := &SecretView{
		Name:      obj.Metadata.Name,
		Namespace: ns,
		Keys:      map[string]string{},
	}

	for k, v := range obj.Data {
		raw, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			continue
		}
		sum := sha256.Sum256(raw)
		view.Keys[k] = hex.EncodeToString(sum[:])
	}

	return view, nil
}

/*
|--------------------------------------------------------------------------
| VIEW: rendered (local input)
|--------------------------------------------------------------------------
*/

func BuildSecretViewFromRendered(data map[string]string) SecretView {
	keys := map[string]string{}

	for k, v := range data {
		sum := sha256.Sum256([]byte(v))
		keys[k] = hex.EncodeToString(sum[:])
	}

	return SecretView{
		Keys: keys,
	}
}

/*
|--------------------------------------------------------------------------
| RENDER
|--------------------------------------------------------------------------
*/

func RenderSecret(ns string, data map[string]string) (string, string, error) {
	encoded := map[string]string{}

	for k, v := range data {
		encoded[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}

	checksum := HashMap(encoded)

	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"type":       "Opaque",
		"metadata": map[string]interface{}{
			"name":      "birdy-secrets",
			"namespace": ns,
			"annotations": map[string]string{
				"birdy.io/checksum": checksum,
			},
		},
		"data": encoded,
	}

	y, err := yaml.Marshal(obj)
	return string(y), checksum, err
}
