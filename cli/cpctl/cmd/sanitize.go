package cmd

import (
	"path/filepath"

	"cpctl/internal/config"
	"cpctl/internal/sanitize"

	"github.com/spf13/cobra"
)

var sanitizeCmd = &cobra.Command{
	Use:   "sanitize",
	Short: "Sanitize Kubernetes manifests for local playground",
	RunE: func(cmd *cobra.Command, args []string) error {
		root := config.RepoRoot()
		dir := filepath.Join(root, "manifests", "sanitized", "services")

		return sanitize.SanitizeDir(dir)
	},
}

func init() {
	rootCmd.AddCommand(sanitizeCmd)
}
