package cmd

import (
	"fmt"

	"cpctl/internal/mirror/aws"

	"github.com/spf13/cobra"
)

var (
	mirrorProfile string
	mirrorSSMPath string
)

var mirrorAwsCmd = &cobra.Command{
	Use:   "aws",
	Short: "Mirror configuration from AWS into local files",
	RunE: func(cmd *cobra.Command, args []string) error {

		if mirrorSSMPath == "" {
			return fmt.Errorf("--ssm-path is required")
		}

		profile := mirrorProfile
		if profile == "" {
			profile = "example-profile"
		}

		return aws.Run(profile, mirrorSSMPath)
	},
}

func init() {
	mirrorCmd := &cobra.Command{
		Use:   "mirror",
		Short: "Mirror external systems into local playground",
	}

	mirrorAwsCmd.Flags().StringVar(
		&mirrorProfile,
		"profile",
		"",
		"AWS profile to use",
	)

	mirrorAwsCmd.Flags().StringVar(
		&mirrorSSMPath,
		"ssm-path",
		"",
		"SSM parameter path (e.g. /dev/services)",
	)

	mirrorCmd.AddCommand(mirrorAwsCmd)
	rootCmd.AddCommand(mirrorCmd)
}
