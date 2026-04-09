package cmd

import (
	"cpctl/internal/config"
	"cpctl/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show playground status (interactive TUI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		m := tui.New(config.RepoRoot(), config.Cfg.Playground.Name)
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err := p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
