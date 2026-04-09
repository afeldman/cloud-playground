package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"cpctl/internal/exec"
)

type ConfigMapView struct {
	Name      string
	Namespace string
	Keys      map[string]string // key -> checksum
}

func BuildConfigMapViewFromCluster(ns, name string) (*ConfigMapView, error) {
	out, err := exec.Capture(
		"kubectl", "-n", ns,
		"get", "configmap", name,
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

	view := &ConfigMapView{
		Name:      obj.Metadata.Name,
		Namespace: ns,
		Keys:      map[string]string{},
	}

	for k, v := range obj.Data {
		sum := sha256.Sum256([]byte(v))
		view.Keys[k] = hex.EncodeToString(sum[:])
	}

	return view, nil
}

func BuildConfigMapViewFromRendered(data map[string]string) ConfigMapView {
	keys := map[string]string{}

	for k, v := range data {
		sum := sha256.Sum256([]byte(v))
		keys[k] = hex.EncodeToString(sum[:])
	}

	return ConfigMapView{
		Keys: keys,
	}
}
