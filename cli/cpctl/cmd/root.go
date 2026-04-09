package cmd

import (
	"log/slog"
	"os"

	"cpctl/internal/config"
	"cpctl/internal/logger"

	"github.com/spf13/cobra"
)

var (
	loglevel  string
	logFormat string
)

var rootCmd = &cobra.Command{
	Use:   "cpctl",
	Short: "Cloud Playground controller",
	Long:  "cpctl manages the local cloud-playground lifecycle (kind + localstack).",
}

func Execute() {
	cobra.OnInitialize(func() {
		config.Init()

		log := logger.New(
			logger.Level(loglevel),
			logger.Format(logFormat),
		)

		// Logger global setzen (nur CLI!)
		slog.SetDefault(log)
	})

	if err := rootCmd.Execute(); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "config file (default: birdy.yaml)")
	rootCmd.PersistentFlags().StringVarP(
		&loglevel,
		"loglevel",
		"l",
		"info",
		"set the logging level (quiet, info, debug)",
	)
	rootCmd.PersistentFlags().StringVar(
		&logFormat,
		"log-format",
		"text",
		"log format: text, toon or json",
	)

}
