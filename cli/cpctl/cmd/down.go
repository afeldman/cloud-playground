package cmd

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"cpctl/internal/config"
	"cpctl/internal/exec"
	"cpctl/internal/tui"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Destroy playground",
	RunE: func(cmd *cobra.Command, args []string) error {

		root := config.RepoRoot()
		name := config.Cfg.Playground.Name
		compose := filepath.Join(root, "localstack", "docker-compose.yml")

		steps := []tui.Step{
			{
				Label: "Stopping LocalStack",
				Run: func() error {
					_ = exec.RunQuiet("docker", "compose", "-f", compose, "down", "-v")
					return nil
				},
			},
			{
				Label: "Deleting Kind cluster",
				Run: func() error {
					_ = exec.RunQuiet("kind", "delete", "cluster", "--name", name)
					return nil
				},
			},
		}

		m := tui.NewProgress("Stopping playground", steps)
		p := tea.NewProgram(m)
		_, err := p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
