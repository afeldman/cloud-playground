package config

import (
	"strings"
	"testing"
)

func makeValidationConfig(name, dataDir, stage string, aiEnabled bool, aiEndpoint, aiModel string) Config {
	c := Config{}
	c.Playground.Name = name
	c.Playground.DataDir = dataDir
	c.Development.Stage = stage
	c.AI.Enabled = aiEnabled
	c.AI.Endpoint = aiEndpoint
	c.AI.Model = aiModel
	return c
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantErr     bool
		errContains []string
	}{
		{
			name:    "valid config with AI enabled",
			config:  makeValidationConfig("test-playground", "./data", "localstack", true, "http://localhost:11434", "llama3.2"),
			wantErr: false,
		},
		{
			name:    "valid config with AI disabled",
			config:  makeValidationConfig("test-playground", "./data", "mirror", false, "", ""),
			wantErr: false,
		},
		{
			name:        "empty playground name",
			config:      makeValidationConfig("", "./data", "localstack", false, "", ""),
			wantErr:     true,
			errContains: []string{"playground.name is required"},
		},
		{
			name:        "empty playground data_dir",
			config:      makeValidationConfig("test-playground", "", "localstack", false, "", ""),
			wantErr:     true,
			errContains: []string{"playground.data_dir is required"},
		},
		{
			name:        "invalid development stage",
			config:      makeValidationConfig("test-playground", "./data", "invalid-stage", false, "", ""),
			wantErr:     true,
			errContains: []string{"development.stage must be one of: localstack, mirror"},
		},
		{
			name:        "AI enabled but empty endpoint",
			config:      makeValidationConfig("test-playground", "./data", "localstack", true, "", "llama3.2"),
			wantErr:     true,
			errContains: []string{"ai.endpoint is required"},
		},
		{
			name:        "AI enabled but URL without scheme",
			config:      makeValidationConfig("test-playground", "./data", "localstack", true, "localhost:11434", "llama3.2"),
			wantErr:     true,
			errContains: []string{"ai.endpoint must use http:// or https:// scheme"},
		},
		{
			name:        "AI enabled but empty model",
			config:      makeValidationConfig("test-playground", "./data", "localstack", true, "http://localhost:11434", ""),
			wantErr:     true,
			errContains: []string{"ai.model is required"},
		},
		{
			name: "multiple validation errors",
			config: func() Config {
				c := Config{}
				c.Playground.Name = ""
				c.Playground.DataDir = ""
				c.Development.Stage = "invalid"
				c.AI.Enabled = true
				c.AI.Endpoint = ""
				c.AI.Model = ""
				return c
			}(),
			wantErr: true,
			errContains: []string{
				"playground.name is required",
				"playground.data_dir is required",
				"development.stage must be one of",
				"ai.endpoint is required",
				"ai.model is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				errStr := err.Error()
				for _, contain := range tt.errContains {
					if !strings.Contains(errStr, contain) {
						t.Errorf("validateConfig() error = %v, should contain %v", errStr, contain)
					}
				}
			}
		})
	}
}
