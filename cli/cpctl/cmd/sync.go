package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"cpctl/internal/config"
	"cpctl/internal/providers/aws"
	"cpctl/internal/sync"

	"github.com/spf13/cobra"
)

var (
	namespace string
	dryRun    bool
	only      string
	diff      bool

	source   string
	ssmPath  string
	smPrefix string
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync local config & secrets into Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔄 syncing local configuration")

		// --------------------
		// Secret source select
		// --------------------
		var secrets map[string]string

		switch source {
		case "local":
			var err error
			secrets, err = sync.LoadSecrets(filepath.Join(config.RepoRoot(), "data", "secrets", "local"))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

		case "aws-ssm":
			if ssmPath == "" {
				fmt.Fprintln(os.Stderr, "--ssm-path is required for source=aws-ssm")
				os.Exit(1)
			}

			provider := &aws.SSMProvider{Path: ssmPath}
			items, err := provider.List()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			secrets = aws.NormalizeProviderSecrets(items)

		case "aws-sm":
			provider := &aws.SecretsManagerProvider{Prefix: smPrefix}
			items, err := provider.List()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			secrets = aws.NormalizeProviderSecrets(items)

		default:
			fmt.Fprintf(os.Stderr, "unknown source: %s\n", source)
			os.Exit(1)
		}

		// --------------------
		// Sync input
		// --------------------
		cfg, err := sync.LoadParams(filepath.Join(config.RepoRoot(), "data", "params", "global"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "warning: could not load data/params/global:", err)
			cfg = map[string]string{}
		}

		input := sync.SyncInput{
			Namespace: namespace,
			Config:    cfg,
			Secrets:   secrets,
			Only:      only,
			DryRun:    dryRun,
			Diff:      diff,
		}

		res, err := sync.Run(input)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if diff && res != nil && res.HasChanges {
			// CI: drift detected
			os.Exit(2)
		}

		os.Exit(0)
	},
}

func init() {
	syncCmd.Flags().StringVar(&namespace, "namespace", "services", "target namespace")
	syncCmd.Flags().StringVar(&source, "source", "local", "secret source: local | aws-ssm | aws-sm")
	syncCmd.Flags().StringVar(&ssmPath, "ssm-path", "", "SSM parameter path (e.g. /birdy/services)")
	syncCmd.Flags().StringVar(&smPrefix, "sm-prefix", "", "Secrets Manager name prefix")
	syncCmd.Flags().StringVar(&only, "only", "", "sync only config or secrets")
	syncCmd.Flags().BoolVar(&dryRun, "dry-run", false, "print manifests without applying")
	syncCmd.Flags().BoolVar(&diff, "diff", false, "show diff without applying")

	rootCmd.AddCommand(syncCmd)
}
