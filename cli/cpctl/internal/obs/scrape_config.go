package obs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"cpctl/internal/config"
	"cpctl/internal/exec"
	"gopkg.in/yaml.v3"
)

// PrometheusConfig represents Prometheus scrape configuration
type PrometheusConfig struct {
	Global struct {
		ScrapeInterval     string `yaml:"scrape_interval"`
		EvaluationInterval string `yaml:"evaluation_interval"`
	} `yaml:"global"`
	Alerting struct {
		Alertmanagers []struct {
			StaticConfigs []struct {
				Targets []string `yaml:"targets"`
			} `yaml:"static_configs"`
		} `yaml:"alertmanagers"`
	} `yaml:"alerting"`
	RuleFiles []string `yaml:"rule_files"`
	ScrapeConfigs []ScrapeConfig `yaml:"scrape_configs"`
}

// ScrapeConfig represents a single scrape configuration
type ScrapeConfig struct {
	JobName         string                 `yaml:"job_name"`
	StaticConfigs   []StaticConfig         `yaml:"static_configs"`
	MetricsPath     string                 `yaml:"metrics_path,omitempty"`
	Scheme          string                 `yaml:"scheme,omitempty"`
	ScrapeInterval  string                 `yaml:"scrape_interval,omitempty"`
	ScrapeTimeout   string                 `yaml:"scrape_timeout,omitempty"`
	HonorLabels     bool                   `yaml:"honor_labels,omitempty"`
	HonorTimestamps bool                   `yaml:"honor_timestamps,omitempty"`
	Params          map[string]string      `yaml:"params,omitempty"`
	RelabelConfigs  []interface{}          `yaml:"relabel_configs,omitempty"`
	MetricRelabelConfigs []interface{}     `yaml:"metric_relabel_configs,omitempty"`
	TLSConfig       *TLSConfig             `yaml:"tls_config,omitempty"`
	BearerTokenFile string                 `yaml:"bearer_token_file,omitempty"`
}

// StaticConfig represents static configuration for targets
type StaticConfig struct {
	Targets []string `yaml:"targets"`
}

// BuildScrapeConfig builds a Prometheus scrape configuration based on current environment
func BuildScrapeConfig() (*PrometheusConfig, error) {
	cfg := &PrometheusConfig{}

	// Global settings
	cfg.Global.ScrapeInterval = "15s"
	cfg.Global.EvaluationInterval = "15s"

	// AlertManager configuration
	cfg.Alerting.Alertmanagers = []struct {
		StaticConfigs []struct {
			Targets []string `yaml:"targets"`
		} `yaml:"static_configs"`
	}{
		{
			StaticConfigs: []struct {
				Targets []string `yaml:"targets"`
			}{
				{Targets: []string{"alertmanager:9093"}},
			},
		},
	}

	// Rule files
	cfg.RuleFiles = []string{
		"/etc/prometheus/rules/*.yaml",
	}

	// Scrape configurations
	scrapeConfigs := []ScrapeConfig{}

	// 1. Kubernetes API server
	scrapeConfigs = append(scrapeConfigs, ScrapeConfig{
		JobName: "kubernetes-apiservers",
		Scheme:  "https",
		StaticConfigs: []StaticConfig{
			{Targets: []string{"kubernetes.default.svc:443"}},
		},
		MetricsPath: "/metrics",
		TLSConfig: &TLSConfig{
			CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		},
		BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
		RelabelConfigs: []interface{}{
			map[string]interface{}{
				"source_labels": []string{"__address__"},
				"regex":         "([^:]+)(?::\\d+)?",
				"target_label":  "cluster",
				"replacement":   "$1",
			},
		},
	})

	// 2. Kubernetes nodes
	scrapeConfigs = append(scrapeConfigs, ScrapeConfig{
		JobName: "kubernetes-nodes",
		Scheme:  "https",
		StaticConfigs: []StaticConfig{
			{Targets: []string{"kubernetes.default.svc:443"}},
		},
		MetricsPath: "/api/v1/nodes",
		Params: map[string]string{
			"resource": "nodes",
		},
		TLSConfig: &TLSConfig{
			CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		},
		BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
		RelabelConfigs: []interface{}{
			map[string]interface{}{
				"action": "labelmap",
				"regex":  "__meta_kubernetes_node_label_(.+)",
			},
			map[string]interface{}{
				"target_label": "__address__",
				"replacement":  "kubernetes.default.svc:443",
			},
			map[string]interface{}{
				"source_labels": []string{"__meta_kubernetes_node_name"},
				"regex":         "(.+)",
				"target_label":  "__metrics_path__",
				"replacement":   "/api/v1/nodes/${1}/proxy/metrics",
			},
		},
	})

	// 3. Kubernetes pods
	scrapeConfigs = append(scrapeConfigs, ScrapeConfig{
		JobName: "kubernetes-pods",
		Scheme:  "https",
		StaticConfigs: []StaticConfig{
			{Targets: []string{"kubernetes.default.svc:443"}},
		},
		MetricsPath: "/api/v1/pods",
		Params: map[string]string{
			"resource": "pods",
		},
		TLSConfig: &TLSConfig{
			CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		},
		BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
		RelabelConfigs: []interface{}{
			map[string]interface{}{
				"action": "keep",
				"regex":  "true",
				"source_labels": []string{
					"__meta_kubernetes_pod_annotation_prometheus_io_scrape",
				},
			},
			map[string]interface{}{
				"action": "replace",
				"regex":  "(.+)",
				"source_labels": []string{
					"__meta_kubernetes_pod_annotation_prometheus_io_path",
				},
				"target_label": "__metrics_path__",
			},
			map[string]interface{}{
				"action": "replace",
				"regex":  "([^:]+)(?::\\d+)?;?(\\d+)?",
				"source_labels": []string{
					"__address__",
					"__meta_kubernetes_pod_annotation_prometheus_io_port",
				},
				"target_label": "__address__",
			},
			map[string]interface{}{
				"action": "labelmap",
				"regex":  "__meta_kubernetes_pod_label_(.+)",
			},
			map[string]interface{}{
				"action": "replace",
				"source_labels": []string{"__meta_kubernetes_namespace"},
				"target_label":  "kubernetes_namespace",
			},
			map[string]interface{}{
				"action": "replace",
				"source_labels": []string{"__meta_kubernetes_pod_name"},
				"target_label":  "kubernetes_pod_name",
			},
		},
	})

	// 4. LocalStack metrics (if enabled)
	if config.Cfg.LocalStack.Enabled {
		localstackTarget := fmt.Sprintf("%s:%d", 
			strings.TrimPrefix(config.Cfg.LocalStack.Endpoint, "http://"), 
			config.Cfg.LocalStack.Port)
		
		scrapeConfigs = append(scrapeConfigs, ScrapeConfig{
			JobName: "localstack",
			StaticConfigs: []StaticConfig{
				{Targets: []string{localstackTarget}},
			},
			MetricsPath: "/_localstack/metrics",
			Scheme:      "http",
			ScrapeInterval: "30s",
			ScrapeTimeout:  "10s",
		})
	}

	// 5. cpctl metrics endpoint
	scrapeConfigs = append(scrapeConfigs, ScrapeConfig{
		JobName: "cpctl",
		StaticConfigs: []StaticConfig{
			{Targets: []string{"host.docker.internal:9090"}}, // Default metrics port
		},
		MetricsPath: "/metrics",
		Scheme:      "http",
		ScrapeInterval: "15s",
		ScrapeTimeout:  "5s",
	})

	// 6. PostgreSQL exporter (if tunnel is active)
	// We'll check if PostgreSQL tunnel is running
	if isPostgresTunnelActive() {
		scrapeConfigs = append(scrapeConfigs, ScrapeConfig{
			JobName: "postgres",
			StaticConfigs: []StaticConfig{
				{Targets: []string{"host.docker.internal:9187"}}, // Default PostgreSQL exporter port
			},
			MetricsPath: "/metrics",
			Scheme:      "http",
			ScrapeInterval: "30s",
			ScrapeTimeout:  "10s",
			Params: map[string]string{
				"sslmode": "disable",
			},
		})
	}

	cfg.ScrapeConfigs = scrapeConfigs
	return cfg, nil
}

// UpdateConfigMap updates the Prometheus ConfigMap with new configuration
func UpdateConfigMap(ctx context.Context, namespace string, config *PrometheusConfig) error {
	slog.Info("updating Prometheus ConfigMap", "namespace", namespace)

	// Convert config to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Prometheus config: %w", err)
	}

	// Create or update ConfigMap
	configMapYAML := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: %s
data:
  prometheus.yml: |
%s`, namespace, indent(string(yamlData), 4))

	// Apply ConfigMap
	if err := exec.RunWithStdin(configMapYAML, "kubectl", "apply", "-f", "-"); err != nil {
		return fmt.Errorf("failed to apply Prometheus ConfigMap: %w", err)
	}

	// Restart Prometheus pod to pick up new config
	podName, err := exec.Capture("kubectl", "get", "pods", "-n", namespace, "-l", "app.kubernetes.io/name=prometheus", "-o", "jsonpath={.items[0].metadata.name}")
	if err == nil && podName != "" {
		if err := exec.RunQuiet("kubectl", "delete", "pod", "-n", namespace, podName); err != nil {
			slog.Warn("failed to restart Prometheus pod", "error", err)
		} else {
			slog.Info("restarted Prometheus pod to apply new config", "pod", podName)
		}
	}

	return nil
}

// isPostgresTunnelActive checks if PostgreSQL tunnel is running
func isPostgresTunnelActive() bool {
	// Check if tunnel PID file exists or process is running
	// For now, return false - we'll implement proper check later
	return false
}

// indent adds indentation to each line
func indent(s string, spaces int) string {
	indentStr := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = indentStr + line
	}
	return strings.Join(lines, "\n")
}

// TLSConfig represents TLS configuration for scrape jobs
type TLSConfig struct {
	CAFile string `yaml:"ca_file"`
}

// BearerTokenFile field for authentication
type BearerTokenFile string