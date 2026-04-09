package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"
	"time"

	"cpctl/internal/project"

	"github.com/spf13/cobra"
)

// projectCmd represents the project command group
var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage DHW2 ecosystem projects",
	Long: `Manage sequential development of DHW2 ecosystem projects.

DHW2 has 13 projects: airbyte/, datalynq-alfa/, zeroetl/, ed4-batch-*.
All projects share the same structure: terraform/, charts/ (Helm), Taskfile.yml.

Commands:
  list       List all available projects
  deploy     Deploy a project to Kind cluster
  teardown   Teardown a deployed project
  status     Show status of a deployed project
  logs       Stream logs from a deployed project

Workflow:
  1. List projects: cpctl project list
  2. Deploy one: cpctl project deploy airbyte
  3. Work on it
  4. Teardown: cpctl project teardown airbyte
  5. Deploy next project

Only one project can be deployed at a time (sequential development).`,
}

// projectListCmd lists all projects
var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all DHW2 projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = cmd.Context() // Context available for future use
		slog.Info("listing projects")

		// Load registry
		registry, err := project.NewRegistry()
		if err != nil {
			return fmt.Errorf("failed to load project registry: %w", err)
		}

		// Get flag values
		showDetails, _ := cmd.Flags().GetBool("details")

		// List projects
		projects := registry.ListProjects()
		if len(projects) == 0 {
			fmt.Println("No projects found. Run 'cpctl project list --discover' to scan for projects.")
			return nil
		}

		if showDetails {
			printDetailedProjects(projects)
		} else {
			printSummaryProjects(projects)
		}

		return nil
	},
}

// projectDeployCmd deploys a project
var projectDeployCmd = &cobra.Command{
	Use:   "deploy [project-name]",
	Short: "Deploy a project to Kind cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		slog.Info("deploying project", "name", projectName)

		// TODO: Implement deployment logic
		fmt.Printf("Deploying project: %s\n", projectName)
		fmt.Println("(Deployment logic will be implemented in Phase 9.2)")

		return nil
	},
}

// projectTeardownCmd tears down a deployed project
var projectTeardownCmd = &cobra.Command{
	Use:   "teardown [project-name]",
	Short: "Teardown a deployed project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		slog.Info("tearing down project", "name", projectName)

		// TODO: Implement teardown logic
		fmt.Printf("Tearing down project: %s\n", projectName)
		fmt.Println("(Teardown logic will be implemented in Phase 9.2)")

		return nil
	},
}

// projectStatusCmd shows status of a deployed project
var projectStatusCmd = &cobra.Command{
	Use:   "status [project-name]",
	Short: "Show status of a deployed project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		slog.Info("checking project status", "name", projectName)

		// TODO: Implement status logic
		fmt.Printf("Status for project: %s\n", projectName)
		fmt.Println("(Status logic will be implemented in Phase 9.3)")

		return nil
	},
}

// projectLogsCmd streams logs from a deployed project
var projectLogsCmd = &cobra.Command{
	Use:   "logs [project-name]",
	Short: "Stream logs from a deployed project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		slog.Info("streaming project logs", "name", projectName)

		// TODO: Implement logs logic
		fmt.Printf("Logs for project: %s\n", projectName)
		fmt.Println("(Logs logic will be implemented in Phase 9.3)")

		return nil
	},
}

// printSummaryProjects prints a summary table of projects
func printSummaryProjects(projects []*project.Project) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tNAMESPACE\tLAST ACTIVE")
	fmt.Fprintln(w, "----\t------\t---------\t-----------")

	for _, p := range projects {
		lastActive := "never"
		if !p.LastActive.IsZero() {
			lastActive = formatDuration(time.Since(p.LastActive))
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			p.Name,
			p.Status,
			p.Namespace,
			lastActive,
		)
	}

	w.Flush()
}

// printDetailedProjects prints detailed information about projects
func printDetailedProjects(projects []*project.Project) {
	for i, p := range projects {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("Project: %s\n", p.Name)
		fmt.Printf("  Path:           %s\n", p.Path)
		fmt.Printf("  Status:         %s\n", p.Status)
		fmt.Printf("  Namespace:      %s\n", p.Namespace)
		fmt.Printf("  Terraform:      %s\n", p.TerraformPath)
		fmt.Printf("  Helm Charts:    %s\n", p.HelmChartPath)

		if !p.LastActive.IsZero() {
			fmt.Printf("  Last Active:    %s (%s ago)\n",
				p.LastActive.Format("2006-01-02 15:04:05"),
				formatDuration(time.Since(p.LastActive)),
			)
		}

		if !p.LastDeployed.IsZero() {
			fmt.Printf("  Last Deployed:  %s (%s ago)\n",
				p.LastDeployed.Format("2006-01-02 15:04:05"),
				formatDuration(time.Since(p.LastDeployed)),
			)
		}

		if p.DeployedBy != "" {
			fmt.Printf("  Deployed By:    %s\n", p.DeployedBy)
		}
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.0fh", d.Hours())
	}
	return fmt.Sprintf("%.0fd", d.Hours()/24)
}

func init() {
	// Add subcommands
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectDeployCmd)
	projectCmd.AddCommand(projectTeardownCmd)
	projectCmd.AddCommand(projectStatusCmd)
	projectCmd.AddCommand(projectLogsCmd)

	// Add flags
	projectListCmd.Flags().Bool("details", false, "Show detailed project information")

	// Register with root
	rootCmd.AddCommand(projectCmd)
}