package config

import (
	"strings"
	"testing"
)

func makeConfig(name, dataDir string, aiEnabled bool, aiEndpoint, aiModel string) Config {
	c := Config{}
	c.Playground.Name = name
	c.Playground.DataDir = dataDir
	c.AI.Enabled = aiEnabled
	c.AI.Endpoint = aiEndpoint
	c.AI.Model = aiModel
	return c
}

func TestCheckMigrationGuardrails(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantErr     bool
		errContains []string
		warnCount   int
	}{
		{
			name:      "valid config with proper AI endpoint",
			config:    makeConfig("test-playground", "./data", true, "http://localhost:11434/v1", "llama3.2"),
			wantErr:   false,
			warnCount: 0,
		},
		{
			name:      "AI endpoint without suffix and not Ollama - should warn",
			config:    makeConfig("test-playground", "./data", true, "http://localhost:8080", "llama3.2"),
			wantErr:   false,
			warnCount: 1,
		},
		{
			name:      "AI endpoint without suffix but AI disabled - no warning",
			config:    makeConfig("test-playground", "./data", false, "http://localhost:8080", "llama3.2"),
			wantErr:   false,
			warnCount: 0,
		},
		{
			name:      "config with terraform reference in DataDir - should warn",
			config:    makeConfig("test-playground", "./terraform/data", false, "", ""),
			wantErr:   false,
			warnCount: 1,
		},
		{
			name:        "AI endpoint with impossible format - should error",
			config:      makeConfig("test-playground", "./data", true, "http://localhost:11434/v0", "llama3.2"),
			wantErr:     true,
			errContains: []string{"AI endpoint", "no longer supported"},
			warnCount:   0,
		},
		{
			name:      "multiple issues - terraform reference and non-Ollama AI endpoint without suffix",
			config:    makeConfig("test-playground", "./terraform/data", true, "http://localhost:8080", "llama3.2"),
			wantErr:   false,
			warnCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkMigrationGuardrails(&tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkMigrationGuardrails() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				errStr := err.Error()
				for _, contain := range tt.errContains {
					if !strings.Contains(errStr, contain) {
						t.Errorf("checkMigrationGuardrails() error = %v, should contain %v", errStr, contain)
					}
				}
			}
		})
	}
}

func TestMigrationRulesCoverage(t *testing.T) {
	defaultConfig := makeConfig("birdy-playground", "./data", true, "http://localhost:11434", "llama3.2")

	err := checkMigrationGuardrails(&defaultConfig)
	if err != nil {
		t.Errorf("default config should not error, got: %v", err)
	}

	properConfig := makeConfig("birdy-playground", "./data", true, "http://localhost:11434/v1", "llama3.2")
	err = checkMigrationGuardrails(&properConfig)
	if err != nil {
		t.Errorf("config with proper endpoint should not error, got: %v", err)
	}

	nonOllamaConfig := makeConfig("birdy-playground", "./data", true, "http://localhost:8080", "llama3.2")
	err = checkMigrationGuardrails(&nonOllamaConfig)
	if err != nil {
		t.Errorf("config with non-Ollama endpoint should not error, got: %v", err)
	}
}
