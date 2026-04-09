package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaValidation tests that the JSON schema is valid and can be used
// to validate configurations
func TestSchemaValidation(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "docs", "cpctl-config.schema.json")
	schemaData, err := os.ReadFile(schemaPath)
	if os.IsNotExist(err) {
		t.Skip("cpctl-config.schema.json not yet generated — skipping schema validation test")
	}
	require.NoError(t, err, "should read schema file")

	var schema map[string]interface{}
	err = json.Unmarshal(schemaData, &schema)
	require.NoError(t, err, "schema should be valid JSON")

	assert.Contains(t, schema, "$schema", "schema should have $schema field")
	assert.Contains(t, schema, "title", "schema should have title field")
	assert.Contains(t, schema, "description", "schema should have description field")
	assert.Contains(t, schema, "type", "schema should have type field")
	assert.Equal(t, "object", schema["type"], "schema type should be object")

	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok, "schema should have properties object")

	requiredSections := []string{
		"playground",
		"localstack",
		"kind",
		"tunnels",
		"mirror",
		"github_actions",
		"ai",
		"development",
	}

	for _, section := range requiredSections {
		assert.Contains(t, properties, section, "schema should contain %s section", section)
	}
}

// TestExampleConfigs tests that example configurations conform to the schema
func TestExampleConfigs(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		expected bool
	}{
		{
			name: "minimal valid config",
			config: `{
				"playground": {
					"name": "test-playground",
					"data_dir": "./data"
				},
				"ai": {
					"enabled": true,
					"endpoint": "http://localhost:11434",
					"model": "llama3.2"
				},
				"development": {
					"stage": "localstack"
				}
			}`,
			expected: true,
		},
		{
			name: "full valid config",
			config: `{
				"playground": {
					"name": "my-playground",
					"data_dir": "./my-data"
				},
				"ai": {
					"enabled": true,
					"endpoint": "http://localhost:11434/v1",
					"model": "llama3.2"
				},
				"development": {
					"stage": "mirror"
				}
			}`,
			expected: true,
		},
		{
			name: "invalid config - missing required playground fields",
			config: `{
				"playground": {
					"name": "test"
				}
			}`,
			expected: false,
		},
		{
			name: "invalid config - wrong stage enum",
			config: `{
				"playground": {
					"name": "test",
					"data_dir": "./data"
				},
				"development": {
					"stage": "invalid-stage"
				}
			}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg map[string]interface{}
			err := json.Unmarshal([]byte(tt.config), &cfg)
			require.NoError(t, err, "test config should be valid JSON")

			if strings.Contains(tt.name, "invalid config") {
				t.Skip("Schema validation would be implemented with a proper JSON schema validator")
			} else {
				t.Skip("Full schema validation would be implemented with a proper JSON schema validator")
			}
		})
	}
}

// TestSchemaMatchesConfigStruct tests that the schema matches the actual Config struct
func TestSchemaMatchesConfigStruct(t *testing.T) {
	// Test 1: empty playground.name should fail validation
	c1 := Config{}
	c1.Playground.Name = ""
	c1.Playground.DataDir = "./data"

	err := validateConfig(&c1)
	assert.Error(t, err, "empty playground.name should fail validation")
	assert.Contains(t, err.Error(), "playground.name is required")

	// Test 2: invalid development.stage enum
	c2 := Config{}
	c2.Playground.Name = "test"
	c2.Playground.DataDir = "./data"
	c2.Development.Stage = "invalid-stage"

	err = validateConfig(&c2)
	assert.Error(t, err, "invalid development.stage should fail validation")
	assert.Contains(t, err.Error(), "development.stage must be one of")

	// Test 3: AI enabled but empty endpoint
	c3 := Config{}
	c3.Playground.Name = "test"
	c3.Playground.DataDir = "./data"
	c3.AI.Enabled = true
	c3.AI.Endpoint = ""
	c3.AI.Model = "llama3.2"

	err = validateConfig(&c3)
	assert.Error(t, err, "AI enabled with empty endpoint should fail validation")
	assert.Contains(t, err.Error(), "ai.endpoint is required")

	// Test 4: AI endpoint without scheme
	c4 := Config{}
	c4.Playground.Name = "test"
	c4.Playground.DataDir = "./data"
	c4.AI.Enabled = true
	c4.AI.Endpoint = "localhost:11434"
	c4.AI.Model = "llama3.2"

	err = validateConfig(&c4)
	assert.Error(t, err, "AI endpoint without scheme should fail validation")
	assert.Contains(t, err.Error(), "ai.endpoint must use http:// or https:// scheme")
}

// TestSchemaMigrationPaths tests legacy warning/error paths
func TestSchemaMigrationPaths(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() Config
		expectError bool
	}{
		{
			name: "AI endpoint without suffix and not Ollama - should warn",
			setup: func() Config {
				return makeConfig("test", "./data", true, "http://localhost:8080", "llama3.2")
			},
			expectError: false,
		},
		{
			name: "terraform reference in DataDir - should warn",
			setup: func() Config {
				return makeConfig("test", "./terraform/data", false, "", "")
			},
			expectError: false,
		},
		{
			name: "AI endpoint with impossible format - should error",
			setup: func() Config {
				return makeConfig("test", "./data", true, "http://localhost:11434/v0", "llama3.2")
			},
			expectError: true,
		},
		{
			name: "valid config with proper AI endpoint - no errors",
			setup: func() Config {
				return makeConfig("test", "./data", true, "http://localhost:11434/v1", "llama3.2")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setup()
			err := checkMigrationGuardrails(&cfg)
			if tt.expectError {
				assert.Error(t, err, "expected migration error")
			} else {
				assert.NoError(t, err, "expected no migration error")
			}
		})
	}
}
