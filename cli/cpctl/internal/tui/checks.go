package tui

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cpctl/internal/exec"
)

type KindStatus struct {
	Running bool
	Name    string
	Nodes   int
	Ready   int
}

type LocalStackStatus struct {
	Running  bool
	Services map[string]string
}

type TerraformStatus struct {
	Applied   bool
	Resources int
}

type PlaygroundStatus struct {
	Kind       KindStatus
	LocalStack LocalStackStatus
	Terraform  TerraformStatus
	CheckedAt  time.Time
}

func CheckAll(root, clusterName string) PlaygroundStatus {
	return PlaygroundStatus{
		Kind:       checkKind(clusterName),
		LocalStack: checkLocalStack(),
		Terraform:  checkTerraform(root),
		CheckedAt:  time.Now(),
	}
}

func checkKind(name string) KindStatus {
	out, err := exec.Capture("kind", "get", "clusters")
	if err != nil {
		return KindStatus{Name: name}
	}

	running := false
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == name {
			running = true
			break
		}
	}
	if !running {
		return KindStatus{Name: name}
	}

	ctx := "kind-" + name
	nodeOut, err := exec.Capture("kubectl", "--context", ctx, "get", "nodes", "--no-headers")
	if err != nil {
		return KindStatus{Running: true, Name: name}
	}

	lines := strings.Split(strings.TrimSpace(nodeOut), "\n")
	total := 0
	ready := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		total++
		fields := strings.Fields(l)
		if len(fields) >= 2 && fields[1] == "Ready" {
			ready++
		}
	}

	return KindStatus{Running: true, Name: name, Nodes: total, Ready: ready}
}

func checkLocalStack() LocalStackStatus {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://localhost:4566/_localstack/health")
	if err != nil {
		return LocalStackStatus{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return LocalStackStatus{Running: true}
	}

	var payload struct {
		Services map[string]string `json:"services"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return LocalStackStatus{Running: true}
	}

	return LocalStackStatus{Running: true, Services: payload.Services}
}

func checkTerraform(root string) TerraformStatus {
	tfstatePath := filepath.Join(root, "terraform", "localstack", "terraform.tfstate")
	data, err := os.ReadFile(tfstatePath)
	if err != nil {
		return TerraformStatus{}
	}

	var state struct {
		Resources []json.RawMessage `json:"resources"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return TerraformStatus{Applied: true}
	}

	return TerraformStatus{Applied: true, Resources: len(state.Resources)}
}
