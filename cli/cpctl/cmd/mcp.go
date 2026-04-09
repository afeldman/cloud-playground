package cmd

import (
	"cpctl/internal/config"
	"cpctl/internal/mcpserver"

	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server commands",
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server over stdio (Claude Desktop integration)",
	Long: `Starts an MCP server on stdin/stdout.

Profiles:
  - CPCTL_MCP_PROFILE=read-only (default): status/logs/plan tools
  - CPCTL_MCP_PROFILE=operator: also enables mutating tools

Add to Claude Desktop config (~/.config/claude/claude_desktop_config.json):

  {
    "mcpServers": {
      "cloud-playground": {
        "command": "cpctl",
        "args": ["mcp", "serve"]
      }
    }
  }

Read-only tools: playground_status, localstack_logs, terraform_plan,
lambda_logs, batch_watch, batch_logs.

Operator-only tools: playground_up, playground_down, playground_update,
playground_sync, playground_apply, terraform_apply, lambda_deploy,
lambda_invoke, batch_submit.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcpserver.Serve(config.RepoRoot(), config.Cfg.Playground.Name)
	},
}

func init() {
	mcpCmd.AddCommand(mcpServeCmd)
	rootCmd.AddCommand(mcpCmd)
}
