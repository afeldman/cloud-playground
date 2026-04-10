package env

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TofuOrchestrator handles OpenTofu operations with enhanced features
type TofuOrchestrator struct {
	tofuDir      string
	stateFile    string
	varFile      string
	computedFile string
	timeout      time.Duration
}

// NewTofuOrchestrator creates a new Tofu orchestrator
func NewTofuOrchestrator(tofuDir, stateFile, varFile string) *TofuOrchestrator {
	computedFile := filepath.Join(tofuDir, "computed.tfvars")
	
	return &TofuOrchestrator{
		tofuDir:      tofuDir,
		stateFile:    stateFile,
		varFile:      varFile,
		computedFile: computedFile,
		timeout:      30 * time.Minute, // Default apply timeout
	}
}

// SetTimeout sets the operation timeout
func (to *TofuOrchestrator) SetTimeout(d time.Duration) {
	to.timeout = d
}

// Init initializes Tofu working directory
func (to *TofuOrchestrator) Init(ctx context.Context) error {
	slog.Info("initializing tofu", "dir", to.tofuDir)

	cmd := exec.CommandContext(ctx, "tofu", "init", "-no-color")
	cmd.Dir = to.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tofu init failed: %w", err)
	}

	slog.Info("tofu initialized", "dir", to.tofuDir)
	return nil
}

// GenerateComputedVars generates computed.tfvars with runtime variables
func (to *TofuOrchestrator) GenerateComputedVars(vars map[string]string) error {
	// Convert map to HCL format
	var hclLines []string
	for key, value := range vars {
		// Handle different value types
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			// Array value
			hclLines = append(hclLines, fmt.Sprintf("%s = %s", key, value))
		} else if value == "true" || value == "false" {
			// Boolean value
			hclLines = append(hclLines, fmt.Sprintf("%s = %s", key, value))
		} else if _, err := strconv.Atoi(value); err == nil {
			// Integer value (strict: entire string must be numeric)
			hclLines = append(hclLines, fmt.Sprintf("%s = %s", key, value))
		} else {
			// String value
			hclLines = append(hclLines, fmt.Sprintf("%s = \"%s\"", key, value))
		}
	}

	content := strings.Join(hclLines, "\n") + "\n"
	
	if err := os.WriteFile(to.computedFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write computed.tfvars: %w", err)
	}

	slog.Debug("computed.tfvars generated", "file", to.computedFile, "vars", vars)
	return nil
}

// Plan validates Tofu configuration and generates plan
func (to *TofuOrchestrator) Plan(ctx context.Context, vars map[string]string) error {
	slog.Info("planning tofu deployment", "dir", to.tofuDir)

	// Generate computed vars if provided
	if vars != nil {
		if err := to.GenerateComputedVars(vars); err != nil {
			return fmt.Errorf("failed to generate computed vars: %w", err)
		}
	}

	cmd := exec.CommandContext(ctx, "tofu", "plan", "-no-color")

	// Add var files
	if to.varFile != "" && fileExists(to.varFile) {
		cmd.Args = append(cmd.Args, "-var-file", to.varFile)
	}
	if fileExists(to.computedFile) {
		cmd.Args = append(cmd.Args, "-var-file", to.computedFile)
	}

	cmd.Dir = to.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tofu plan failed: %w", err)
	}

	slog.Info("tofu plan successful", "dir", to.tofuDir)
	return nil
}

// Apply applies Tofu configuration
func (to *TofuOrchestrator) Apply(ctx context.Context, vars map[string]string) error {
	slog.Info("applying tofu configuration", "dir", to.tofuDir)

	// Generate computed vars if provided
	if vars != nil {
		if err := to.GenerateComputedVars(vars); err != nil {
			return fmt.Errorf("failed to generate computed vars: %w", err)
		}
	}

	cmd := exec.CommandContext(ctx, "tofu", "apply", "-auto-approve", "-no-color")

	// Add var files
	if to.varFile != "" && fileExists(to.varFile) {
		cmd.Args = append(cmd.Args, "-var-file", to.varFile)
	}
	if fileExists(to.computedFile) {
		cmd.Args = append(cmd.Args, "-var-file", to.computedFile)
	}

	cmd.Dir = to.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Wrap context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, to.timeout)
	defer cancel()
	cmd = exec.CommandContext(timeoutCtx, cmd.Args[0], cmd.Args[1:]...)
	cmd.Dir = to.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tofu apply failed: %w", err)
	}

	slog.Info("tofu apply successful", "dir", to.tofuDir)
	return nil
}

// Destroy removes infrastructure managed by Tofu
func (to *TofuOrchestrator) Destroy(ctx context.Context) error {
	slog.Info("destroying tofu infrastructure", "dir", to.tofuDir)

	cmd := exec.CommandContext(ctx, "tofu", "destroy", "-auto-approve", "-no-color")

	// Add var files
	if to.varFile != "" && fileExists(to.varFile) {
		cmd.Args = append(cmd.Args, "-var-file", to.varFile)
	}
	if fileExists(to.computedFile) {
		cmd.Args = append(cmd.Args, "-var-file", to.computedFile)
	}

	cmd.Dir = to.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute) // Shorter timeout for destroy
	defer cancel()
	cmd = exec.CommandContext(timeoutCtx, cmd.Args[0], cmd.Args[1:]...)
	cmd.Dir = to.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tofu destroy failed: %w", err)
	}

	// Clean up computed vars file
	os.Remove(to.computedFile)

	slog.Info("tofu destroy successful", "dir", to.tofuDir)
	return nil
}

// Validate checks Tofu configuration validity
func (to *TofuOrchestrator) Validate(ctx context.Context) error {
	slog.Info("validating tofu configuration", "dir", to.tofuDir)

	cmd := exec.CommandContext(ctx, "tofu", "validate")
	cmd.Dir = to.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tofu validate failed: %w", err)
	}

	slog.Info("tofu validation successful", "dir", to.tofuDir)
	return nil
}

// Output reads an output value from Tofu state
func (to *TofuOrchestrator) Output(ctx context.Context, key string) (string, error) {
	cmd := exec.CommandContext(ctx, "tofu", "output", "-raw", key)
	cmd.Dir = to.tofuDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tofu output failed for key %s: %w", key, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetResourceCount returns number of managed resources (estimated from state)
func (to *TofuOrchestrator) GetResourceCount(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "tofu", "show", "-json")
	cmd.Dir = to.tofuDir

	output, err := cmd.Output()
	if err != nil {
		// If state doesn't exist yet, return 0
		if strings.Contains(err.Error(), "No state file") {
			return 0, nil
		}
		return 0, fmt.Errorf("tofu show failed: %w", err)
	}

	// Parse JSON to count resources properly
	var state struct {
		Values struct {
			RootModule struct {
				Resources []interface{} `json:"resources"`
			} `json:"root_module"`
		} `json:"values"`
	}

	if err := json.Unmarshal(output, &state); err != nil {
		// Fallback to string counting if JSON parsing fails
		resourceCount := strings.Count(string(output), `"address"`)
		return resourceCount, nil
	}

	return len(state.Values.RootModule.Resources), nil
}

// GetOutputs returns all outputs from Tofu state
func (to *TofuOrchestrator) GetOutputs(ctx context.Context) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "tofu", "output", "-json")
	cmd.Dir = to.tofuDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tofu output failed: %w", err)
	}

	var outputs map[string]struct {
		Value interface{} `json:"value"`
	}
	if err := json.Unmarshal(output, &outputs); err != nil {
		return nil, fmt.Errorf("failed to parse tofu outputs: %w", err)
	}

	result := make(map[string]string)
	for key, val := range outputs {
		switch v := val.Value.(type) {
		case string:
			result[key] = v
		case []interface{}:
			// Convert array to JSON string
			if jsonBytes, err := json.Marshal(v); err == nil {
				result[key] = string(jsonBytes)
			}
		case map[string]interface{}:
			// Convert map to JSON string
			if jsonBytes, err := json.Marshal(v); err == nil {
				result[key] = string(jsonBytes)
			}
		case float64:
			result[key] = fmt.Sprintf("%.0f", v)
		case bool:
			result[key] = fmt.Sprintf("%v", v)
		default:
			result[key] = fmt.Sprintf("%v", v)
		}
	}

	return result, nil
}

// GetCurrentVars reads current variables from computed.tfvars
func (to *TofuOrchestrator) GetCurrentVars() (map[string]string, error) {
	if !fileExists(to.computedFile) {
		return make(map[string]string), nil
	}

	content, err := os.ReadFile(to.computedFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read computed.tfvars: %w", err)
	}

	// Simple parsing of HCL-like format
	vars := make(map[string]string)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key = value format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes from string values
		value = strings.Trim(value, `"`)
		
		vars[key] = value
	}

	return vars, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}