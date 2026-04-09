package config

import (
	"fmt"
	"log/slog"
	"strings"
)

// MigrationRule defines a rule for detecting and handling legacy configuration
type MigrationRule struct {
	// Check returns true if the rule applies to the given config
	Check func(cfg *Config) bool
	// Warning returns a warning message if the rule applies (optional)
	Warning func(cfg *Config) string
	// Error returns an error message if the rule applies (optional)
	Error func(cfg *Config) string
	// SuggestedFix returns a suggested fix message (optional)
	SuggestedFix func(cfg *Config) string
}

// migrationRules contains all known migration rules
var migrationRules = []MigrationRule{
	// Rule 1: Detect AI endpoint patterns that might not work with current endpoint resolution logic
	{
		Check: func(cfg *Config) bool {
			if !cfg.AI.Enabled {
				return false
			}
			endpoint := strings.TrimSpace(cfg.AI.Endpoint)
			if endpoint == "" {
				return false
			}
			
			// Based on resolveEndpoints logic in llm/client.go:
			// The function handles these patterns well:
			// 1. Ends with /v1/chat/completions
			// 2. Ends with /v1  
			// 3. Ends with /api/chat
			// 4. Contains port 11434 (looksLikeOllamaBase returns true)
			// 5. Everything else gets /v1/chat/completions appended
			
			// So we should only warn for endpoints that:
			// - Don't contain 11434 (not Ollama-like)
			// - Don't end with /v1, /api/chat, or /v1/chat/completions
			// - Might get unexpected transformation
			
			trimmed := strings.TrimRight(endpoint, "/")
			
			// Check if it has a known good suffix
			hasGoodSuffix := strings.HasSuffix(trimmed, "/v1") ||
				strings.HasSuffix(trimmed, "/api/chat") ||
				strings.HasSuffix(trimmed, "/v1/chat/completions")
			
			// Check if it looks like Ollama base (contains 11434)
			looksLikeOllama := strings.Contains(endpoint, "11434")
			
			// Warn if it doesn't have a good suffix AND doesn't look like Ollama
			return !hasGoodSuffix && !looksLikeOllama
		},
		Warning: func(cfg *Config) string {
			return fmt.Sprintf("AI endpoint '%s' may not work correctly with current endpoint resolution logic", cfg.AI.Endpoint)
		},
		SuggestedFix: func(cfg *Config) string {
			return "For best results, update AI endpoint to either: 1) Add '/v1' suffix for OpenAI-compatible API, 2) Add '/api/chat' for Ollama native API, or 3) Use full path '/v1/chat/completions'. Endpoints containing '11434' are automatically handled."
		},
	},
	// Rule 2: Detect configuration that assumes terraform directory structure
	{
		Check: func(cfg *Config) bool {
			// Check Playground.DataDir for references to old terraform directory
			if strings.Contains(cfg.Playground.DataDir, "terraform") {
				return true
			}
			
			// Check if any other string fields in config might reference terraform
			// This is a simple check - in a real implementation we might need to
			// check more fields or use reflection
			return false
		},
		Warning: func(cfg *Config) string {
			return "Configuration contains references to 'terraform' directory structure"
		},
		SuggestedFix: func(cfg *Config) string {
			return "Update configuration to use 'tofu' directory instead of 'terraform'. The project has migrated from Terraform to OpenTofu."
		},
	},
	// Rule 3: Detect impossible configurations that require manual intervention
	{
		Check: func(cfg *Config) bool {
			// This rule would check for configurations that cannot be automatically
			// migrated and require manual changes
			// For now, we'll use a simple example: if AI is enabled but endpoint
			// is set to a known incompatible old value
			if !cfg.AI.Enabled {
				return false
			}
			endpoint := strings.TrimSpace(cfg.AI.Endpoint)
			
			// Example of an impossible mapping: old endpoint format that doesn't
			// exist anymore or can't be resolved
			if endpoint == "http://localhost:11434/v0" || // v0 API doesn't exist
			   endpoint == "http://localhost:11434/old-api" || // old API path
			   strings.Contains(endpoint, "terraform-api") { // hypothetical old service
				return true
			}
			
			return false
		},
		Error: func(cfg *Config) string {
			return fmt.Sprintf("AI endpoint '%s' uses a format that is no longer supported", cfg.AI.Endpoint)
		},
		SuggestedFix: func(cfg *Config) string {
			return "Update to a supported endpoint format: use '/v1' for OpenAI-compatible API or '/api/chat' for Ollama native API"
		},
	},
}

// checkMigrationGuardrails checks for legacy configuration patterns and provides
// warnings or errors with guidance for migration.
func checkMigrationGuardrails(cfg *Config) error {
	var warnings []string
	var errors []string

	for _, rule := range migrationRules {
		if rule.Check(cfg) {
			if rule.Warning != nil {
				msg := rule.Warning(cfg)
				if rule.SuggestedFix != nil {
					msg += ". " + rule.SuggestedFix(cfg)
				}
				warnings = append(warnings, msg)
			}
			if rule.Error != nil {
				msg := rule.Error(cfg)
				if rule.SuggestedFix != nil {
					msg += ". " + rule.SuggestedFix(cfg)
				}
				errors = append(errors, msg)
			}
		}
	}

	// Log all warnings
	for _, warning := range warnings {
		slog.Warn("config migration warning", "message", warning)
	}

	// Return aggregated errors if any
	if len(errors) > 0 {
		return fmt.Errorf("configuration migration errors:\n  • %s", strings.Join(errors, "\n  • "))
	}

	return nil
}