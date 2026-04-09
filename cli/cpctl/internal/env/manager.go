package env

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cpctl/internal/config"
)

// Stage represents the deployment stage (localstack or mirror)
type Stage string

const (
	StageLocalStack Stage = "localstack"
	StageMirror     Stage = "mirror"
)

// MirrorConfig holds Mirror-Cloud specific configuration
type MirrorConfig struct {
	AWSProfile string            `json:"aws_profile"`
	AWSRegion  string            `json:"aws_region"`
	DefaultTTL string            `json:"default_ttl"`
	Resources  map[string]bool   `json:"resources"`
	Enabled    bool              `json:"enabled"`
}

// EnvironmentManager manages infrastructure provisioning lifecycle
type EnvironmentManager struct {
	dataDir      string
	stage        Stage
	config       *config.Config
	mirrorConfig *MirrorConfig
	envName      string
}

// EnvironmentState tracks the current state of an environment
type EnvironmentState struct {
	Stage           Stage
	Status          string // provisioning, ready, downgrading, down
	LastUpdated     time.Time
	ResourceCount   int
	TunnelCount     int
	ErrorMessage    string
	TofuStateKey    string // localstack or mirror
}

// NewEnvironmentManager creates a new environment manager
func NewEnvironmentManager(dataDir string, cfg *config.Config) (*EnvironmentManager, error) {
	if dataDir == "" {
		dataDir = "data/environments"
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create environment data directory: %w", err)
	}

	em := &EnvironmentManager{
		dataDir: dataDir,
		config:  cfg,
	}

	// Detect stage from config
	if cfg.Development.Stage != "" {
		em.stage = Stage(cfg.Development.Stage)
	} else {
		em.stage = StageLocalStack // default
	}

	// Parse Mirror config if enabled
	if em.stage == StageMirror {
		mirrorCfg, err := em.parseMirrorConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to parse mirror config: %w", err)
		}
		em.mirrorConfig = mirrorCfg
		em.envName = em.generateEnvName()
	}

	return em, nil
}

// GetStage returns the current deployment stage
func (em *EnvironmentManager) GetStage() Stage {
	return em.stage
}

// SetStage sets the deployment stage
func (em *EnvironmentManager) SetStage(stage Stage) error {
	valid := stage == StageLocalStack || stage == StageMirror
	if !valid {
		return fmt.Errorf("invalid stage: %s (must be 'localstack' or 'mirror')", stage)
	}
	em.stage = stage
	return nil
}

// GetTofuDir returns the appropriate Tofu directory for the current stage
func (em *EnvironmentManager) GetTofuDir() string {
	baseDir := "tofu"
	switch em.stage {
	case StageLocalStack:
		return filepath.Join(baseDir, "localstack")
	case StageMirror:
		return filepath.Join(baseDir, "mirror")
	default:
		return baseDir
	}
}

// GetVarFile returns the var file path for the current stage
func (em *EnvironmentManager) GetVarFile() string {
	switch em.stage {
	case StageLocalStack:
		return filepath.Join("tofu", fmt.Sprintf("%s.tfvars.json", StageLocalStack))
	case StageMirror:
		return filepath.Join("tofu", fmt.Sprintf("%s.tfvars.json", StageMirror))
	default:
		return ""
	}
}

// GetStateFile returns the state file path for the current stage
func (em *EnvironmentManager) GetStateFile() string {
	return filepath.Join(em.dataDir, fmt.Sprintf("%s.tfstate", em.stage))
}

// GetEnvFile returns the environment metadata file path
func (em *EnvironmentManager) GetEnvFile() string {
	return filepath.Join(em.dataDir, fmt.Sprintf("%s.env", em.stage))
}

// ReadState reads current environment state from disk
func (em *EnvironmentManager) ReadState(ctx context.Context) (*EnvironmentState, error) {
	envFile := em.GetEnvFile()

	envState := &EnvironmentState{
		Stage:      em.stage,
		Status:     "down",
		TofuStateKey: string(em.stage),
	}

	data, err := os.ReadFile(envFile)
	if err != nil {
		if os.IsNotExist(err) {
			return envState, nil // Return default "down" state
		}
		return nil, fmt.Errorf("failed to read environment state: %w", err)
	}

	// Parse simple key=value format
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "STATUS":
			envState.Status = value
		case "RESOURCES":
			fmt.Sscanf(value, "%d", &envState.ResourceCount)
		case "TUNNELS":
			fmt.Sscanf(value, "%d", &envState.TunnelCount)
		case "ERROR":
			envState.ErrorMessage = value
		case "UPDATED":
			// Parse timestamp
			if t, err := time.Parse(time.RFC3339, value); err == nil {
				envState.LastUpdated = t
			}
		}
	}

	return envState, nil
}

// WriteState writes environment state to disk
func (em *EnvironmentManager) WriteState(ctx context.Context, state *EnvironmentState) error {
	envFile := em.GetEnvFile()

	content := fmt.Sprintf(`# Environment state for %s (auto-generated)
STAGE=%s
STATUS=%s
RESOURCES=%d
TUNNELS=%d
UPDATED=%s
ERROR=%s
`,
		em.stage,
		em.stage,
		state.Status,
		state.ResourceCount,
		state.TunnelCount,
		time.Now().Format(time.RFC3339),
		state.ErrorMessage,
	)

	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write environment state: %w", err)
	}

	slog.Info("environment state written", "file", envFile, "status", state.Status)
	return nil
}

// DeleteState removes environment state from disk
func (em *EnvironmentManager) DeleteState(ctx context.Context) error {
	envFile := em.GetEnvFile()

	if err := os.Remove(envFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete environment state: %w", err)
	}

	slog.Info("environment state deleted", "file", envFile)
	return nil
}

// GetStageName returns a human-readable stage name
func (em *EnvironmentManager) GetStageName() string {
	switch em.stage {
	case StageLocalStack:
		return "LocalStack (local, cost-free)"
	case StageMirror:
		return "Mirror-Cloud (ephemeral AWS)"
	default:
		return string(em.stage)
	}
}

// parseMirrorConfig parses Mirror configuration from .cpctl.yaml
func (em *EnvironmentManager) parseMirrorConfig() (*MirrorConfig, error) {
	cfg := &MirrorConfig{
		AWSProfile: em.config.Mirror.AWSProfile,
		AWSRegion:  em.config.AWS.Region, // Default to global AWS region
		DefaultTTL: em.config.Mirror.DefaultTTL,
		Resources:  em.config.Mirror.Resources,
		Enabled:    em.config.Mirror.Enabled,
	}

	// Set defaults if not provided
	if cfg.AWSProfile == "" {
		cfg.AWSProfile = "mirror-account"
	}
	if cfg.AWSRegion == "" {
		cfg.AWSRegion = "eu-central-1"
	}
	if cfg.DefaultTTL == "" {
		cfg.DefaultTTL = "4h"
	}
	if cfg.Resources == nil {
		cfg.Resources = map[string]bool{
			"vpc":     true,
			"batch":   true,
			"lambda":  true,
			"compute": true,
		}
	}

	// Validate AWS credentials are available
	if err := em.validateAWSCredentials(cfg.AWSProfile); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validateAWSCredentials checks if AWS credentials are available
func (em *EnvironmentManager) validateAWSCredentials(profile string) error {
	// Check if AWS credentials file exists
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	awsConfigPath := filepath.Join(homeDir, ".aws", "config")
	awsCredsPath := filepath.Join(homeDir, ".aws", "credentials")

	// Check if config file exists
	if _, err := os.Stat(awsConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("AWS config file not found at %s", awsConfigPath)
	}

	// Check if credentials file exists
	if _, err := os.Stat(awsCredsPath); os.IsNotExist(err) {
		return fmt.Errorf("AWS credentials file not found at %s", awsCredsPath)
	}

	// TODO: Could add actual credential validation by calling AWS STS
	// For now, just check file existence
	slog.Debug("AWS credentials validated", "profile", profile)
	return nil
}

// generateEnvName generates a unique environment name
func (em *EnvironmentManager) generateEnvName() string {
	timestamp := time.Now().Format("20060102-150405")
	hostname, _ := os.Hostname()
	shortHost := "local"
	if hostname != "" {
		parts := strings.Split(hostname, ".")
		shortHost = parts[0]
	}
	
	return fmt.Sprintf("mirror-%s-%s", shortHost, timestamp)
}

// GetMirrorConfig returns the Mirror configuration
func (em *EnvironmentManager) GetMirrorConfig() *MirrorConfig {
	return em.mirrorConfig
}

// GetEnvName returns the environment name
func (em *EnvironmentManager) GetEnvName() string {
	return em.envName
}
