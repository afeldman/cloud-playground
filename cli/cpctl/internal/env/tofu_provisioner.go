package env

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

// TofuProvisioner handles OpenTofu provisioning
type TofuProvisioner struct {
	tofuDir   string
	stateFile string
	varFile   string
	timeout   time.Duration
}

// NewTofuProvisioner creates a new Tofu provisioner
func NewTofuProvisioner(tofuDir, stateFile, varFile string) *TofuProvisioner {
	if stateFile == "" {
		stateFile = "terraform.tfstate"
	}

	return &TofuProvisioner{
		tofuDir:   tofuDir,
		stateFile: stateFile,
		varFile:   varFile,
		timeout:   15 * time.Minute, // Default timeout
	}
}

// SetTimeout sets the provisioning timeout
func (tp *TofuProvisioner) SetTimeout(d time.Duration) {
	tp.timeout = d
}

// Init initializes Tofu working directory
func (tp *TofuProvisioner) Init(ctx context.Context) error {
	slog.Info("initializing tofu", "dir", tp.tofuDir)

	cmd := exec.CommandContext(ctx, "tofu", "init",
		"-backend-config", fmt.Sprintf("path=%s", tp.stateFile),
	)
	cmd.Dir = tp.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tofu init failed: %w", err)
	}

	slog.Info("tofu initialized", "dir", tp.tofuDir)
	return nil
}

// Plan validates Tofu configuration and generates plan
func (tp *TofuProvisioner) Plan(ctx context.Context) error {
	slog.Info("planning tofu deployment", "dir", tp.tofuDir)

	cmd := exec.CommandContext(ctx, "tofu", "plan")

	// Add var file if exists
	if tp.varFile != "" && fileExists(tp.varFile) {
		cmd.Args = append(cmd.Args, "-var-file", tp.varFile)
	}

	cmd.Dir = tp.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tofu plan failed: %w", err)
	}

	slog.Info("tofu plan successful", "dir", tp.tofuDir)
	return nil
}

// Apply applies Tofu configuration
func (tp *TofuProvisioner) Apply(ctx context.Context) error {
	slog.Info("applying tofu configuration", "dir", tp.tofuDir)

	cmd := exec.CommandContext(ctx, "tofu", "apply", "-auto-approve")

	// Add var file if exists
	if tp.varFile != "" && fileExists(tp.varFile) {
		cmd.Args = append(cmd.Args, "-var-file", tp.varFile)
	}

	cmd.Dir = tp.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Wrap context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, tp.timeout)
	defer cancel()
	cmd = exec.CommandContext(timeoutCtx, cmd.Args[0], cmd.Args[1:]...)
	cmd.Dir = tp.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tofu apply failed: %w", err)
	}

	slog.Info("tofu apply successful", "dir", tp.tofuDir)
	return nil
}

// Destroy removes infrastructure managed by Tofu
func (tp *TofuProvisioner) Destroy(ctx context.Context) error {
	slog.Info("destroying tofu infrastructure", "dir", tp.tofuDir)

	cmd := exec.CommandContext(ctx, "tofu", "destroy", "-auto-approve")

	// Add var file if exists
	if tp.varFile != "" && fileExists(tp.varFile) {
		cmd.Args = append(cmd.Args, "-var-file", tp.varFile)
	}

	cmd.Dir = tp.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	timeoutCtx, cancel := context.WithTimeout(ctx, tp.timeout)
	defer cancel()
	cmd = exec.CommandContext(timeoutCtx, cmd.Args[0], cmd.Args[1:]...)
	cmd.Dir = tp.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tofu destroy failed: %w", err)
	}

	slog.Info("tofu destroy successful", "dir", tp.tofuDir)
	return nil
}

// Validate checks Tofu configuration validity
func (tp *TofuProvisioner) Validate(ctx context.Context) error {
	slog.Info("validating tofu configuration", "dir", tp.tofuDir)

	cmd := exec.CommandContext(ctx, "tofu", "validate")
	cmd.Dir = tp.tofuDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tofu validate failed: %w", err)
	}

	slog.Info("tofu validation successful", "dir", tp.tofuDir)
	return nil
}

// Output reads an output value from Tofu state
func (tp *TofuProvisioner) Output(ctx context.Context, key string) (string, error) {
	cmd := exec.CommandContext(ctx, "tofu", "output", "-raw", key)
	cmd.Dir = tp.tofuDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tofu output failed for key %s: %w", key, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetResourceCount returns number of managed resources (estimated from state)
func (tp *TofuProvisioner) GetResourceCount(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "tofu", "show", "-json")
	cmd.Dir = tp.tofuDir

	output, err := cmd.Output()
	if err != nil {
		// If state doesn't exist yet, return 0
		if strings.Contains(err.Error(), "No state file") {
			return 0, nil
		}
		return 0, fmt.Errorf("tofu show failed: %w", err)
	}

	// Count lines with "address" to estimate resource count
	resourceCount := strings.Count(string(output), `"address"`)
	return resourceCount, nil
}


