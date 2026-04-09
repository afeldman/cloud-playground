package obs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cpctl/internal/exec"
)

// HelmDeployer manages Helm chart deployments for observability stack
type HelmDeployer struct {
	helmPath string
	timeout  time.Duration
}

// NewHelmDeployer creates a new Helm deployer
func NewHelmDeployer() *HelmDeployer {
	return &HelmDeployer{
		helmPath: "helm",
		timeout:  300 * time.Second, // 5 minutes for deployment
	}
}

// DeployPrometheus deploys Prometheus stack to the specified namespace
func (d *HelmDeployer) DeployPrometheus(ctx context.Context, namespace string) error {
	slog.Info("deploying Prometheus stack", "namespace", namespace)

	// Add prometheus-community repo if not already added
	if err := d.ensureRepoAdded("prometheus-community", "https://prometheus-community.github.io/helm-charts"); err != nil {
		return fmt.Errorf("failed to add prometheus repo: %w", err)
	}

	// Update repos
	if err := d.runHelm(ctx, "repo", "update"); err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	// Create namespace if it doesn't exist
	if err := d.createNamespace(ctx, namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Install kube-prometheus-stack which includes Prometheus, AlertManager, and Grafana
	// Using a simple values override to reduce resource usage for local development
	values := `
prometheus:
  prometheusSpec:
    resources:
      requests:
        memory: "256Mi"
        cpu: "100m"
      limits:
        memory: "512Mi"
        cpu: "500m"
  service:
    type: NodePort
    nodePort: 30900

alertmanager:
  alertmanagerSpec:
    resources:
      requests:
        memory: "128Mi"
        cpu: "50m"
      limits:
        memory: "256Mi"
        cpu: "200m"
  service:
    type: NodePort
    nodePort: 30903

grafana:
  enabled: true
  adminPassword: "admin"
  service:
    type: NodePort
    nodePort: 30901
  resources:
    requests:
      memory: "128Mi"
      cpu: "50m"
    limits:
      memory: "256Mi"
      cpu: "200m"
`

	// Install the chart
	args := []string{
		"install", "prometheus", "prometheus-community/kube-prometheus-stack",
		"--namespace", namespace,
		"--create-namespace",
		"--version", "58.0.0", // Pin to a known working version
		"--wait",
		"--timeout", d.timeout.String(),
		"--set", "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.storageClassName=standard",
		"--set", "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=2Gi",
	}

	// Add values file if we have custom values
	if values != "" {
		// For simplicity, we'll pass values directly via --set flags
		// In a more robust implementation, we'd write to a temp file
		args = append(args,
			"--set", "prometheus.prometheusSpec.resources.requests.memory=256Mi",
			"--set", "prometheus.prometheusSpec.resources.requests.cpu=100m",
			"--set", "prometheus.prometheusSpec.resources.limits.memory=512Mi",
			"--set", "prometheus.prometheusSpec.resources.limits.cpu=500m",
			"--set", "prometheus.service.type=NodePort",
			"--set", "prometheus.service.nodePort=30900",
			"--set", "alertmanager.alertmanagerSpec.resources.requests.memory=128Mi",
			"--set", "alertmanager.alertmanagerSpec.resources.requests.cpu=50m",
			"--set", "alertmanager.alertmanagerSpec.resources.limits.memory=256Mi",
			"--set", "alertmanager.alertmanagerSpec.resources.limits.cpu=200m",
			"--set", "alertmanager.service.type=NodePort",
			"--set", "alertmanager.service.nodePort=30903",
			"--set", "grafana.enabled=true",
			"--set", "grafana.adminPassword=admin",
			"--set", "grafana.service.type=NodePort",
			"--set", "grafana.service.nodePort=30901",
			"--set", "grafana.resources.requests.memory=128Mi",
			"--set", "grafana.resources.requests.cpu=50m",
			"--set", "grafana.resources.limits.memory=256Mi",
			"--set", "grafana.resources.limits.cpu=200m",
		)
	}

	if err := d.runHelm(ctx, args...); err != nil {
		return fmt.Errorf("failed to install Prometheus stack: %w", err)
	}

	slog.Info("Prometheus stack deployed successfully", "namespace", namespace)
	return nil
}

// DeployGrafana deploys standalone Grafana (alternative to kube-prometheus-stack Grafana)
func (d *HelmDeployer) DeployGrafana(ctx context.Context, namespace string) error {
	slog.Info("deploying standalone Grafana", "namespace", namespace)

	// Add grafana repo if not already added
	if err := d.ensureRepoAdded("grafana", "https://grafana.github.io/helm-charts"); err != nil {
		return fmt.Errorf("failed to add grafana repo: %w", err)
	}

	// Update repos
	if err := d.runHelm(ctx, "repo", "update"); err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	// Create namespace if it doesn't exist
	if err := d.createNamespace(ctx, namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Install Grafana
	args := []string{
		"install", "grafana", "grafana/grafana",
		"--namespace", namespace,
		"--create-namespace",
		"--version", "7.3.6", // Pin to a known working version
		"--wait",
		"--timeout", d.timeout.String(),
		"--set", "adminPassword=admin",
		"--set", "service.type=NodePort",
		"--set", "service.nodePort=30901",
		"--set", "resources.requests.memory=128Mi",
		"--set", "resources.requests.cpu=50m",
		"--set", "resources.limits.memory=256Mi",
		"--set", "resources.limits.cpu=200m",
		"--set", "persistence.enabled=true",
		"--set", "persistence.size=2Gi",
	}

	if err := d.runHelm(ctx, args...); err != nil {
		return fmt.Errorf("failed to install Grafana: %w", err)
	}

	slog.Info("Grafana deployed successfully", "namespace", namespace)
	return nil
}

// DeployAlertManager deploys standalone AlertManager
func (d *HelmDeployer) DeployAlertManager(ctx context.Context, namespace string) error {
	slog.Info("deploying AlertManager", "namespace", namespace)

	// Add prometheus-community repo if not already added
	if err := d.ensureRepoAdded("prometheus-community", "https://prometheus-community.github.io/helm-charts"); err != nil {
		return fmt.Errorf("failed to add prometheus repo: %w", err)
	}

	// Update repos
	if err := d.runHelm(ctx, "repo", "update"); err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	// Create namespace if it doesn't exist
	if err := d.createNamespace(ctx, namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Install AlertManager
	args := []string{
		"install", "alertmanager", "prometheus-community/alertmanager",
		"--namespace", namespace,
		"--create-namespace",
		"--version", "1.0.0", // Pin to a known working version
		"--wait",
		"--timeout", d.timeout.String(),
		"--set", "service.type=NodePort",
		"--set", "service.nodePort=30903",
		"--set", "resources.requests.memory=128Mi",
		"--set", "resources.requests.cpu=50m",
		"--set", "resources.limits.memory=256Mi",
		"--set", "resources.limits.cpu=200m",
		"--set", "persistence.enabled=true",
		"--set", "persistence.size=1Gi",
	}

	if err := d.runHelm(ctx, args...); err != nil {
		return fmt.Errorf("failed to install AlertManager: %w", err)
	}

	slog.Info("AlertManager deployed successfully", "namespace", namespace)
	return nil
}

// WaitForReadiness waits for all pods in the namespace to be ready
func (d *HelmDeployer) WaitForReadiness(ctx context.Context, namespace string) error {
	slog.Info("waiting for pods to be ready", "namespace", namespace)

	timeout := 60 * time.Second
	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Check if all pods are ready
			output, err := exec.Capture("kubectl", "get", "pods", "-n", namespace, "-o", "jsonpath={.items[*].status.conditions[?(@.type=='Ready')].status}")
			if err != nil {
				slog.Debug("failed to check pod status", "error", err)
				continue
			}

			statuses := strings.Fields(output)
			allReady := true
			for _, status := range statuses {
				if status != "True" {
					allReady = false
					break
				}
			}

			if allReady && len(statuses) > 0 {
				slog.Info("all pods are ready", "namespace", namespace, "duration", time.Since(start))
				return nil
			}

			if time.Since(start) > timeout {
				return fmt.Errorf("timeout waiting for pods to be ready after %v", timeout)
			}

			time.Sleep(2 * time.Second)
		}
	}
}

// GetEndpoints returns service endpoints for Prometheus, Grafana, and AlertManager
func (d *HelmDeployer) GetEndpoints(ctx context.Context, namespace string) (map[string]string, error) {
	endpoints := make(map[string]string)

	// Get Prometheus endpoint
	prometheusPort, err := exec.Capture("kubectl", "get", "svc", "-n", namespace, "prometheus-kube-prometheus-prometheus", "-o", "jsonpath={.spec.ports[0].nodePort}")
	if err == nil && prometheusPort != "" {
		endpoints["prometheus"] = fmt.Sprintf("http://localhost:%s", prometheusPort)
	} else {
		// Try alternative service name
		prometheusPort, err = exec.Capture("kubectl", "get", "svc", "-n", namespace, "prometheus-prometheus-kube-prometheus-prometheus", "-o", "jsonpath={.spec.ports[0].nodePort}")
		if err == nil && prometheusPort != "" {
			endpoints["prometheus"] = fmt.Sprintf("http://localhost:%s", prometheusPort)
		}
	}

	// Get Grafana endpoint
	grafanaPort, err := exec.Capture("kubectl", "get", "svc", "-n", namespace, "prometheus-grafana", "-o", "jsonpath={.spec.ports[0].nodePort}")
	if err == nil && grafanaPort != "" {
		endpoints["grafana"] = fmt.Sprintf("http://localhost:%s", grafanaPort)
	} else {
		// Try standalone Grafana
		grafanaPort, err = exec.Capture("kubectl", "get", "svc", "-n", namespace, "grafana", "-o", "jsonpath={.spec.ports[0].nodePort}")
		if err == nil && grafanaPort != "" {
			endpoints["grafana"] = fmt.Sprintf("http://localhost:%s", grafanaPort)
		}
	}

	// Get AlertManager endpoint
	alertmanagerPort, err := exec.Capture("kubectl", "get", "svc", "-n", namespace, "prometheus-kube-prometheus-alertmanager", "-o", "jsonpath={.spec.ports[0].nodePort}")
	if err == nil && alertmanagerPort != "" {
		endpoints["alertmanager"] = fmt.Sprintf("http://localhost:%s", alertmanagerPort)
	} else {
		// Try standalone AlertManager
		alertmanagerPort, err = exec.Capture("kubectl", "get", "svc", "-n", namespace, "alertmanager", "-o", "jsonpath={.spec.ports[0].nodePort}")
		if err == nil && alertmanagerPort != "" {
			endpoints["alertmanager"] = fmt.Sprintf("http://localhost:%s", alertmanagerPort)
		}
	}

	return endpoints, nil
}

// ensureRepoAdded ensures a Helm repo is added
func (d *HelmDeployer) ensureRepoAdded(name, url string) error {
	// Check if repo already exists
	repos, err := exec.Capture(d.helmPath, "repo", "list")
	if err != nil {
		return fmt.Errorf("failed to list helm repos: %w", err)
	}

	if strings.Contains(repos, name) {
		return nil // Repo already exists
	}

	// Add the repo
	if err := d.runHelm(context.Background(), "repo", "add", name, url); err != nil {
		return fmt.Errorf("failed to add helm repo %s: %w", name, err)
	}

	return nil
}

// createNamespace creates a Kubernetes namespace if it doesn't exist
func (d *HelmDeployer) createNamespace(ctx context.Context, namespace string) error {
	// Check if namespace exists
	_, err := exec.Capture("kubectl", "get", "namespace", namespace)
	if err == nil {
		return nil // Namespace already exists
	}

	// Create namespace
	if err := exec.RunQuiet("kubectl", "create", "namespace", namespace); err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
	}

	return nil
}

// runHelm runs a Helm command
func (d *HelmDeployer) runHelm(ctx context.Context, args ...string) error {
	slog.Debug("running helm command", "args", args)
	return exec.Run(d.helmPath, args...)
}