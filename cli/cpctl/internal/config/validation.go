package config

import (
	"fmt"
	"net/url"
	"strings"
)

// validateConfig validates the configuration and returns an error if any validation fails.
// It aggregates multiple validation errors into a single error message.
func validateConfig(cfg *Config) error {
	var errors []string

	// Validate playground.name
	if strings.TrimSpace(cfg.Playground.Name) == "" {
		errors = append(errors, "playground.name is required and cannot be empty")
	}

	// Validate playground.data_dir
	if strings.TrimSpace(cfg.Playground.DataDir) == "" {
		errors = append(errors, "playground.data_dir is required and cannot be empty")
	}

	// Validate development.stage
	validStages := map[string]bool{
		"moto":       true,
		"localstack": true,
		"mirror":     true,
	}
	if stage := strings.TrimSpace(cfg.Development.Stage); stage != "" {
		if !validStages[stage] {
			errors = append(errors, fmt.Sprintf("development.stage must be one of: moto, localstack, mirror (got: %s)", stage))
		}
	}

	// Validate AI configuration when enabled
	if cfg.AI.Enabled {
		// Validate ai.endpoint
		endpoint := strings.TrimSpace(cfg.AI.Endpoint)
		if endpoint == "" {
			errors = append(errors, "ai.endpoint is required when ai.enabled is true")
		} else {
			if u, err := url.Parse(endpoint); err != nil {
				errors = append(errors, fmt.Sprintf("ai.endpoint must be a valid URL: %v", err))
			} else if u.Scheme != "http" && u.Scheme != "https" {
				errors = append(errors, "ai.endpoint must use http:// or https:// scheme")
			}
		}

		// Validate ai.model
		if strings.TrimSpace(cfg.AI.Model) == "" {
			errors = append(errors, "ai.model is required when ai.enabled is true")
		}
	}

	// Return aggregated errors if any
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  • %s", strings.Join(errors, "\n  • "))
	}

	return nil
}