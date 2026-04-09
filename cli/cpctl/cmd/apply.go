package cmd

import (
	"log/slog"
	"path/filepath"

	"cpctl/internal/config"
	"cpctl/internal/kind"
	"cpctl/internal/sanitize"

	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply Kubernetes manifests to the playground cluster",
	RunE: func(cmd *cobra.Command, args []string) error {

		root := config.RepoRoot()

		bootstrap := filepath.Join(root, "manifests", "bootstrap")
		sanitizedServices := filepath.Join(root, "manifests", "sanitized", "services")

		// 1️⃣ Sanitize ONLY directories with manifests
		slog.Info("🧼 sanitizing manifests")
		if err := sanitize.SanitizeDir(sanitizedServices); err != nil {
			return err
		}

		// 2️⃣ Apply bootstrap (namespaces, etc.)
		slog.Info("▶ applying bootstrap manifests")
		if err := kind.ApplyManifests(bootstrap); err != nil {
			return err
		}

		// 3️⃣ Apply sanitized workloads
		slog.Info("▶ applying sanitized manifests")
		if err := kind.ApplyManifests(sanitizedServices); err != nil {
			return err
		}

		slog.Info("✅ manifests applied")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
}
