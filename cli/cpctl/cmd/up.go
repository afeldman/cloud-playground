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
		kindCfg := filepath.Join(root, "kind", "cluster-config.yaml")
		compose := filepath.Join(root, "localstack", "docker-compose.yml")
		tfDir := filepath.Join(root, "tofu", "localstack")

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
				Label: "Starting LocalStack",
				Run: func() error {
					return exec.RunQuiet("docker", "compose", "-f", compose, "up", "-d")
				},
			},
			{
				Label: "Waiting for LocalStack",
				Run: func() error {
					return localstack.WaitReady(60 * time.Second)
				},
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

func printPlaygroundSummary(name string) {
	fmt.Printf("\nPlayground is up:\n")
	fmt.Printf("  Kind cluster  : %s\n", name)
	fmt.Printf("  LocalStack    : http://localhost:4566\n")
	fmt.Printf("  LocalStack UI : http://localhost:8080\n")
	fmt.Printf("  OpenTofu state  : tofu/localstack/terraform.tfstate\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  lqtctl sync              # push config+secrets to cluster\n")
	fmt.Printf("  lqtctl apply             # apply manifests\n")
	fmt.Printf("  lqtctl localstack up     # AWS only, no cluster\n")
	fmt.Printf("  lqtctl down              # destroy everything\n")
}

func init() {
	rootCmd.AddCommand(upCmd)
}
