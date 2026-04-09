package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"cpctl/internal/config"
	"cpctl/internal/tunnels"

	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Manage database tunnels (PostgreSQL, RDS)",
	Long: `Manage persistent SSH/port-forward tunnels to databases.

Examples:
  cpctl tunnel start pg              # PostgreSQL in-cluster (port-forward)
  cpctl tunnel start rds             # RDS via SSM Session Manager
  cpctl tunnel list                  # Show active tunnels with PIDs
  cpctl tunnel status                # Health check all tunnels
  cpctl tunnel stop pg               # Terminate tunnel by name
`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var tunnelStartCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start a tunnel (pg, rds, or other configured name)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnelName := args[0]
		slog.Info("starting tunnel", "name", tunnelName)

		tunnelCfg, ok := config.Cfg.Tunnels[tunnelName]
		if !ok {
			return fmt.Errorf("tunnel '%s' not found in config", tunnelName)
		}

		// Create appropriate tunnel based on type
		var tunnel tunnels.Tunnel
		var err error

		switch tunnelCfg.Type {
		case "kubernetes":
			tunnel, err = tunnels.NewKubernetesTunnel(&config.Cfg, tunnelName)
		case "aws-ssm":
			tunnel, err = tunnels.NewSSMTunnel(&config.Cfg, tunnelName)
		default:
			return fmt.Errorf("unsupported tunnel type: %s", tunnelCfg.Type)
		}

		if err != nil {
			return fmt.Errorf("failed to create tunnel: %w", err)
		}

		// Setup signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

		// Start tunnel
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		if err := tunnel.Start(ctx); err != nil {
			return fmt.Errorf("failed to start tunnel: %w", err)
		}

		fmt.Printf("✅ Tunnel '%s' started successfully\n", tunnelName)
		fmt.Printf("   Local: localhost:%d\n", tunnelCfg.LocalPort)
		fmt.Println("Press Ctrl+C to stop the tunnel")

		// Wait for signal or context cancellation
		select {
		case sig := <-sigChan:
			fmt.Printf("\nReceived signal %v, shutting down tunnel...\n", sig)
			
			// Stop the tunnel
			if err := tunnel.Stop(); err != nil {
				slog.Error("failed to stop tunnel", "error", err)
				return fmt.Errorf("failed to stop tunnel: %w", err)
			}
			
			fmt.Printf("✅ Tunnel '%s' stopped gracefully\n", tunnelName)
			
			// If it was SIGTERM or SIGINT, exit with success
			if sig == syscall.SIGINT || sig == syscall.SIGTERM {
				return nil
			}
			// For other signals, return error
			return fmt.Errorf("tunnel stopped by signal: %v", sig)
			
		case <-ctx.Done():
			fmt.Println("\nContext cancelled, shutting down tunnel...")
			if err := tunnel.Stop(); err != nil {
				slog.Error("failed to stop tunnel on context cancellation", "error", err)
				return fmt.Errorf("failed to stop tunnel: %w", err)
			}
			fmt.Printf("✅ Tunnel '%s' stopped due to context cancellation\n", tunnelName)
			return ctx.Err()
		}
	},
}

var tunnelStopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop a tunnel by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnelName := args[0]
		slog.Info("stopping tunnel", "name", tunnelName)

		mgr := tunnels.New(config.Cfg.Playground.DataDir)
		info, err := mgr.Load(tunnelName)
		if err != nil {
			return fmt.Errorf("tunnel not found: %s", tunnelName)
		}

		// Kill process by PID (cross-platform way)
		proc, err := os.FindProcess(info.PID)
		if err != nil {
			slog.Error("process not found", "pid", info.PID, "error", err)
			return fmt.Errorf("process not found: %w", err)
		}

		if err := proc.Kill(); err != nil {
			slog.Error("failed to kill tunnel process", "pid", info.PID, "error", err)
			return fmt.Errorf("failed to kill tunnel: %w", err)
		}

		// Remove PID file
		if err := mgr.Delete(tunnelName); err != nil {
			slog.Error("failed to remove tunnel metadata", "error", err)
		}

		fmt.Printf("✅ Tunnel '%s' stopped\n", tunnelName)
		return nil
	},
}

var tunnelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active tunnels with PIDs",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("listing active tunnels")

		mgr := tunnels.New(config.Cfg.Playground.DataDir)
		activeTunnels, err := mgr.List()
		if err != nil {
			return fmt.Errorf("failed to list tunnels: %w", err)
		}

		if len(activeTunnels) == 0 {
			fmt.Println("No active tunnels")
			return nil
		}

		fmt.Println("\nActive Tunnels:")
		fmt.Println("┌──────────────┬──────┬────────────┬──────────────────┬─────────────┐")
		fmt.Println("│ Name         │ PID  │ Local Port │ Remote Host      │ Remote Port │")
		fmt.Println("├──────────────┼──────┼────────────┼──────────────────┼─────────────┤")

		for _, tunnel := range activeTunnels {
			fmt.Printf("│ %-12s │ %-4d │ %-10d │ %-16s │ %-11d │\n",
				tunnel.Name, tunnel.PID, tunnel.LocalPort, tunnel.RemoteHost, tunnel.RemotePort)
		}

		fmt.Println("└──────────────┴──────┴────────────┴──────────────────┴─────────────┘")
		return nil
	},
}

var tunnelStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Health check all active tunnels",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("checking tunnel health")

		mgr := tunnels.New(config.Cfg.Playground.DataDir)
		activeTunnels, err := mgr.List()
		if err != nil {
			return fmt.Errorf("failed to list tunnels: %w", err)
		}

		if len(activeTunnels) == 0 {
			fmt.Println("No active tunnels")
			return nil
		}

		// Use health checker to verify connectivity
		healthChecker := tunnels.NewHealthChecker()
		healthResults := healthChecker.CheckAllTunnels(activeTunnels)

		fmt.Println("\nTunnel Status:")
		fmt.Println("┌──────────────┬──────────────┬─────────────────────────────────────┐")
		fmt.Println("│ Name         │ PID          │ Health Status                      │")
		fmt.Println("├──────────────┼──────────────┼─────────────────────────────────────┤")

		for _, tunnel := range activeTunnels {
			status := healthResults[tunnel.Name]
			// Truncate status if too long for display
			if len(status) > 35 {
				status = status[:32] + "..."
			}
			fmt.Printf("│ %-12s │ %-12d │ %-35s │\n", tunnel.Name, tunnel.PID, status)
		}

		fmt.Println("└──────────────┴──────────────┴─────────────────────────────────────┘")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tunnelCmd)
	tunnelCmd.AddCommand(tunnelStartCmd)
	tunnelCmd.AddCommand(tunnelStopCmd)
	tunnelCmd.AddCommand(tunnelListCmd)
	tunnelCmd.AddCommand(tunnelStatusCmd)
}
