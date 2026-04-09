package cmd

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"cpctl/internal/config"
	"cpctl/internal/exec"
	"cpctl/internal/localstack"

	"github.com/spf13/cobra"
)

func init() {
	localstackCmd := &cobra.Command{
		Use:   "localstack",
		Short: "Manage LocalStack (AWS-only, no cluster needed)",
	}

	localstackUpCmd := &cobra.Command{
		Use:   "up",
		Short: "Start LocalStack",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := config.RepoRoot()
			compose := filepath.Join(root, "localstack", "docker-compose.yml")
			if err := exec.Run("docker", "compose", "-f", compose, "up", "-d"); err != nil {
				return err
			}
			if err := localstack.WaitReady(60 * time.Second); err != nil {
				return err
			}

			tfDir := filepath.Join(root, "tofu", "localstack")
			slog.Info("⏳ running tofu apply")
			if err := exec.Run("tofu", "-chdir="+tfDir, "init", "-upgrade", "-no-color"); err != nil {
				return err
			}
			if err := exec.Run("tofu", "-chdir="+tfDir, "apply", "-auto-approve", "-no-color"); err != nil {
				return err
			}

			fmt.Println("LocalStack is up:")
			fmt.Println("  Endpoint        : http://localhost:4566")
			fmt.Println("  UI              : http://localhost:8080")
			fmt.Println("  OpenTofu state  : tofu/localstack/terraform.tfstate")
			return nil
		},
	}

	localstackDownCmd := &cobra.Command{
		Use:   "down",
		Short: "Stop LocalStack",
		RunE: func(cmd *cobra.Command, args []string) error {
			compose := filepath.Join(config.RepoRoot(), "localstack", "docker-compose.yml")
			return exec.Run("docker", "compose", "-f", compose, "down", "-v")
		},
	}

	localstackCmd.AddCommand(localstackUpCmd, localstackDownCmd)
	rootCmd.AddCommand(localstackCmd)
}
