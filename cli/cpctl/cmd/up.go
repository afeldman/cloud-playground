package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"cpctl/internal/config"
	"cpctl/internal/exec"
	"cpctl/internal/localstack"
	"cpctl/internal/moto"
	"cpctl/internal/tui"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start local playground",
	RunE: func(cmd *cobra.Command, args []string) error {

		for _, b := range []string{"kind", "docker", "tofu"} {
			if !exec.Exists(b) {
				return fmt.Errorf("missing dependency: %s", b)
			}
		}

		root := config.RepoRoot()
		name := config.Cfg.Playground.Name
		stage := config.Cfg.Development.Stage
		kindCfg := filepath.Join(root, "kind", "cluster-config.yaml")
		tfDir := filepath.Join(root, "tofu", "localstack") // moto + localstack teilen tofu-Config

		compose, emulatorLabel, waitFn := emulatorConfig(root, stage)

		steps := []tui.Step{
			{
				Label: "Removing previous cluster",
				Run: func() error {
					_ = exec.RunQuiet("kind", "delete", "cluster", "--name", name)
					return nil
				},
			},
			{
				Label: "Creating Kind cluster",
				Run: func() error {
					return exec.RunQuiet("kind", "create", "cluster", "--name", name, "--config", kindCfg)
				},
			},
			{
				Label: "Starting " + emulatorLabel,
				Run: func() error {
					return exec.RunQuiet("docker", "compose", "-f", compose, "up", "-d")
				},
			},
			{
				Label: "Waiting for " + emulatorLabel,
				Run:   waitFn,
			},
			{
				Label: "OpenTofu init",
				Run: func() error {
					return exec.RunQuiet("tofu", "-chdir="+tfDir, "init", "-upgrade", "-no-color")
				},
			},
			{
				Label: "OpenTofu apply",
				Run: func() error {
					return exec.RunQuiet("tofu", "-chdir="+tfDir, "apply", "-auto-approve", "-no-color")
				},
			},
		}

		m := tui.NewProgress("Starting playground", steps)
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		if pm, ok := finalModel.(tui.ProgressModel); ok && pm.Err() != nil {
			return pm.Err()
		}

		printPlaygroundSummary(name)
		return nil
	},
}

// emulatorConfig returns compose file path, display label and wait function for the given stage.
func emulatorConfig(root, stage string) (compose, label string, waitFn func() error) {
	switch stage {
	case "localstack":
		return filepath.Join(root, "localstack", "docker-compose.yml"),
			"LocalStack",
			func() error { return localstack.WaitReady(60 * time.Second) }
	default: // "moto" and future stages
		return filepath.Join(root, "moto", "docker-compose.yml"),
			"Moto",
			func() error { return moto.WaitReady(60 * time.Second) }
	}
}

func printPlaygroundSummary(name string) {
	stage := config.Cfg.Development.Stage
	fmt.Printf("\nPlayground is up:\n")
	fmt.Printf("  Kind cluster  : %s\n", name)
	fmt.Printf("  AWS emulator  : http://localhost:4566 (%s)\n", stage)
	fmt.Printf("  OpenTofu state: tofu/localstack/terraform.tfstate\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cpctl sync   # push config+secrets to cluster\n")
	fmt.Printf("  cpctl down   # destroy everything\n")
}

func init() {
	rootCmd.AddCommand(upCmd)
}
