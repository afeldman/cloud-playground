package cmd

import (
	"log/slog"
	"os"
	"path/filepath"

	"cpctl/internal/config"
	"cpctl/internal/exec"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update playground components (Kind images, LocalStack, AWS profiles, Terraform providers)",
	RunE: func(cmd *cobra.Command, args []string) error {
		root := config.RepoRoot()
		compose := filepath.Join(root, "localstack", "docker-compose.yml")
		tfDir := filepath.Join(root, "terraform", "localstack")
		awsConfig := filepath.Join(root, "aws-local", "aws-config.sh")
		kindCfg := filepath.Join(root, "kind", "cluster-config.yaml")

		slog.Info("pulling Kind node images")
		for _, image := range kindNodeImages(kindCfg) {
			slog.Info("docker pull", "image", image)
			if err := exec.Run("docker", "pull", image); err != nil {
				return err
			}
		}

		slog.Info("pulling latest LocalStack images")
		if err := exec.Run("docker", "compose", "-f", compose, "pull"); err != nil {
			return err
		}

		slog.Info("reconfiguring AWS local profiles")
		if err := exec.Run("bash", awsConfig); err != nil {
			return err
		}

		slog.Info("upgrading Terraform providers")
		if err := exec.Run("terraform", "-chdir="+tfDir, "init", "-upgrade", "-no-color"); err != nil {
			return err
		}

		slog.Info("update complete")
		return nil
	},
}

// kindNodeImages parses the Kind cluster config and returns the unique node images.
func kindNodeImages(cfgPath string) []string {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil
	}

	var cfg struct {
		Nodes []struct {
			Image string `json:"image"`
		} `json:"nodes"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	seen := map[string]bool{}
	var images []string
	for _, n := range cfg.Nodes {
		if n.Image != "" && !seen[n.Image] {
			seen[n.Image] = true
			images = append(images, n.Image)
		}
	}
	return images
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
