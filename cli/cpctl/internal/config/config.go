package config

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// AIProvider describes a single LLM backend in the providers list.
type AIProvider struct {
	Type   string `mapstructure:"type"`    // openai | anthropic | openai-compatible | ollama
	URL    string `mapstructure:"url"`
	APIKey string `mapstructure:"api_key"` // supports ${ENV_VAR} expansion
	Model  string `mapstructure:"model"`
}

type Config struct {
	Playground struct {
		Name    string `mapstructure:"name"`
		DataDir string `mapstructure:"data_dir"`
	}
	AWS struct {
		Region string `mapstructure:"region"`
	}
	LocalStack struct {
		Enabled  bool   `mapstructure:"enabled"`
		Endpoint string `mapstructure:"endpoint"`
		Port     int    `mapstructure:"port"`
	}
	Kind struct {
		Enabled      bool   `mapstructure:"enabled"`
		ClusterName  string `mapstructure:"cluster_name"`
		Kubeconfig   string `mapstructure:"kubeconfig"`
	}
	Tunnels map[string]struct {
		Type       string `mapstructure:"type"`
		Method     string `mapstructure:"method"`
		Namespace  string `mapstructure:"namespace"`
		Service    string `mapstructure:"service"`
		LocalPort  int    `mapstructure:"local_port"`
		RemotePort int    `mapstructure:"remote_port"`
		RemoteHost string `mapstructure:"remote_host"`
		SSHUser    string `mapstructure:"ssh_user"`
		AutoStart  bool   `mapstructure:"auto_start"`
	}
	Mirror struct {
		Enabled           bool            `mapstructure:"enabled"`
		AWSProfile        string          `mapstructure:"aws_profile"`
		DefaultTTL        string          `mapstructure:"default_ttl"`
		Resources         map[string]bool `mapstructure:"resources"`
	}
	GitHubActions struct {
		Enabled    bool              `mapstructure:"enabled"`
		Platforms  map[string]string `mapstructure:"platforms"`
		SecretsFile string           `mapstructure:"secrets_file"`
		EventsDir  string            `mapstructure:"events_dir"`
	}
	AI struct {
		Enabled      bool         `mapstructure:"enabled"`
		Endpoint     string       `mapstructure:"endpoint"`
		Model        string       `mapstructure:"model"`
		SystemPrompt string       `mapstructure:"system_prompt"`
		Providers    []AIProvider `mapstructure:"providers"`
	}
	Development struct {
		AutoSyncSSM       bool   `mapstructure:"auto_sync_ssm"`
		AutoRebuildCLI    bool   `mapstructure:"auto_rebuild_cli"`
		AutoStartTunnels  bool   `mapstructure:"auto_start_tunnels"`
		Stage             string `mapstructure:"stage"` // localstack or mirror
	}
}

var Cfg Config

func loadDotEnv() {
	path := filepath.Join(RepoRoot(), ".env")
	f, err := os.Open(path)
	if err != nil {
		return // silent no-op if missing
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if key != "" && os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func Init() {
	loadDotEnv()

	// Try .cpctl.yaml first, fall back to birdy.yaml
	viper.SetConfigName(".cpctl")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.AddConfigPath("../..")

	viper.SetEnvPrefix("CPCTL")
	viper.AutomaticEnv()

	setDefaults()

	// Load config (silent if not found)
	if err := viper.ReadInConfig(); err != nil {
		slog.Debug("failed to read .cpctl.yaml", "error", err)
		// Try fallback birdy.yaml
		viper.SetConfigName("birdy")
		if err := viper.ReadInConfig(); err != nil {
			slog.Warn("using default config (no .cpctl.yaml or birdy.yaml)")
		} else {
			slog.Debug("loaded birdy.yaml config")
		}
	} else {
		slog.Debug("loaded .cpctl.yaml config", "file", viper.ConfigFileUsed())
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		slog.Error("failed to unmarshal config", "error", err)
		panic(err)
	}

	// Debug: Check what was loaded
	slog.Debug("config loaded", 
		"playground.name", Cfg.Playground.Name,
		"playground.data_dir", Cfg.Playground.DataDir,
		"config_file", viper.ConfigFileUsed(),
	)

	// Validate configuration
	if err := validateConfig(&Cfg); err != nil {
		slog.Error("configuration validation failed", "error", err)
		panic(err)
	}

	// Check for migration guardrails (legacy config patterns)
	if err := checkMigrationGuardrails(&Cfg); err != nil {
		slog.Error("configuration migration check failed", "error", err)
		panic(err)
	}

	slog.Debug("config loaded", "playground", Cfg.Playground.Name)
}

func setDefaults() {
	viper.SetDefault("playground.name", "birdy-playground")
	viper.SetDefault("playground.data_dir", "./data")
	viper.SetDefault("aws.region", "eu-central-1")
	viper.SetDefault("localstack.enabled", true)
	viper.SetDefault("localstack.endpoint", "http://localhost:4566")
	viper.SetDefault("localstack.port", 4566)
	viper.SetDefault("kind.enabled", true)
	viper.SetDefault("kind.cluster_name", "birdy-local")
	viper.SetDefault("kind.kubeconfig", os.ExpandEnv("$HOME/.kube/config"))
	viper.SetDefault("ai.enabled", true)
	viper.SetDefault("ai.endpoint", "http://localhost:11434")
	viper.SetDefault("ai.model", "llama3.2")
	viper.SetDefault("development.auto_sync_ssm", true)
	viper.SetDefault("development.auto_rebuild_cli", false)
	viper.SetDefault("development.auto_start_tunnels", false)
	viper.SetDefault("development.stage", "localstack")
}
