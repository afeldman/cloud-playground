package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cpctl/internal/assistant"
	"cpctl/internal/config"
	"cpctl/internal/llm"
	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Local AI assistant for cloud debugging",
	Long: `Chat with a local LLM (Ollama / LM Studio) for cloud playground diagnostics.

The AI assistant has context about:
- Your Kubernetes cluster (pod status, logs, resource usage)
- LocalStack state (queues, tables, functions)
- Mirror-Cloud infrastructure (if active)
- Past commands and execution logs

Examples:
  cpctl ai chat                      # Interactive REPL
  cpctl ai debug --error "OOM killed" # Troubleshoot specific error
  cpctl ai suggest --issue tunnel    # Get suggestions for tunnel problems
`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var aiChatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Interactive chat with local LLM",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("starting AI chat session")

		client, err := newAIClient()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
		defer cancel()

		snapshot := assistant.BuildSnapshot(ctx, "chat")
		messages := []llm.Message{{
			Role:    "system",
			Content: assistant.DefaultSystemPrompt(config.Cfg.AI.SystemPrompt),
		}, {
			Role:    "user",
			Content: "Initial environment snapshot:\n" + snapshot,
		}}

		out := cmd.OutOrStdout()
		fmt.Fprintln(out, "AI Assistant ready")
		fmt.Fprintln(out, "Type 'quit' or 'exit' to stop")
		fmt.Fprintln(out)

		scanner := bufio.NewScanner(cmd.InOrStdin())
		for {
			fmt.Fprint(out, "> ")
			if !scanner.Scan() {
				fmt.Fprintln(out)
				return nil
			}

			question := strings.TrimSpace(scanner.Text())
			if question == "" {
				continue
			}
			if question == "quit" || question == "exit" {
				return nil
			}

			messages = append(messages, llm.Message{Role: "user", Content: question})
			response, err := client.Chat(cmd.Context(), trimHistory(messages))
			if err != nil {
				return fmt.Errorf("ai chat failed: %w", err)
			}

			messages = append(messages, llm.Message{Role: "assistant", Content: response})
			fmt.Fprintln(out, response)
			fmt.Fprintln(out)
		}
	},
}

var aiDebugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug specific error with AI",
	RunE: func(cmd *cobra.Command, args []string) error {
		errorMsg := cmd.Flag("error").Value.String()
		if strings.TrimSpace(errorMsg) == "" {
			return fmt.Errorf("error message is required")
		}
		slog.Info("debugging error with AI", "error", errorMsg)

		return runOneShotAI(cmd, "debug", inferTopic(errorMsg), errorMsg)
	},
}

var aiSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Get suggestions for a specific playground issue",
	RunE: func(cmd *cobra.Command, args []string) error {
		issue := cmd.Flag("issue").Value.String()
		if strings.TrimSpace(issue) == "" {
			return fmt.Errorf("issue topic is required")
		}
		slog.Info("requesting suggestions", "issue", issue)

		return runOneShotAI(cmd, "suggest", issue, "Provide actionable suggestions for issue: "+issue)
	},
}

var aiDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check AI endpoint and configured model",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAIClient()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		report, diagnoseErr := client.Diagnose(ctx)
		out := cmd.OutOrStdout()

		fmt.Fprintln(out, "AI Doctor")
		fmt.Fprintf(out, "Configured endpoint: %s\n", report.ConfiguredEndpoint)
		fmt.Fprintf(out, "Primary probe:      %s\n", report.PrimaryEndpoint)
		if strings.TrimSpace(report.FallbackEndpoint) != "" {
			fmt.Fprintf(out, "Fallback probe:     %s\n", report.FallbackEndpoint)
		}
		fmt.Fprintf(out, "Detected backend:   %s\n", report.Backend)
		fmt.Fprintf(out, "Configured model:   %s\n", report.Model)
		fmt.Fprintf(out, "Reachable:          %t\n", report.Reachable)
		fmt.Fprintf(out, "Model available:    %t\n", report.ModelAvailable)

		if len(report.AvailableModels) > 0 {
			fmt.Fprintln(out, "Available models:")
			for _, model := range report.AvailableModels {
				fmt.Fprintf(out, "- %s\n", model)
			}
		}

		if len(report.Details) > 0 {
			fmt.Fprintln(out, "Details:")
			for _, detail := range report.Details {
				fmt.Fprintf(out, "- %s\n", detail)
			}
		}

		if diagnoseErr != nil {
			return fmt.Errorf("ai doctor failed: %w", diagnoseErr)
		}
		if !report.ModelAvailable {
			return fmt.Errorf("configured model %q is not available on the detected backend", report.Model)
		}
		return nil
	},
}

func runOneShotAI(cmd *cobra.Command, mode, topic, input string) error {
	client, err := newAIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	snapshot := assistant.BuildSnapshot(ctx, topic)
	prompt := assistant.BuildPrompt(mode, input, snapshot)
	response, err := client.Chat(ctx, []llm.Message{
		{Role: "system", Content: assistant.DefaultSystemPrompt(config.Cfg.AI.SystemPrompt)},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return fmt.Errorf("ai request failed: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), response)
	return nil
}

func inferTopic(text string) string {
	value := strings.ToLower(text)
	switch {
	case strings.Contains(value, "tunnel") || strings.Contains(value, "port-forward"):
		return "tunnel"
	case strings.Contains(value, "lambda"):
		return "lambda"
	case strings.Contains(value, "batch") || strings.Contains(value, "job"):
		return "batch"
	case strings.Contains(value, "localstack"):
		return "localstack"
	default:
		return "chat"
	}
}

func trimHistory(messages []llm.Message) []llm.Message {
	if len(messages) <= 8 {
		return messages
	}
	trimmed := make([]llm.Message, 0, 8)
	trimmed = append(trimmed, messages[0])
	trimmed = append(trimmed, messages[len(messages)-7:]...)
	return trimmed
}

// newAIClient creates an LLM client from the current configuration.
// If providers are configured it uses the multi-provider path with ordered
// fallback; otherwise it falls back to the legacy single endpoint config.
func newAIClient() (*llm.Client, error) {
	cfg := config.Cfg.AI
	if len(cfg.Providers) > 0 {
		pcs := make([]llm.ProviderConfig, 0, len(cfg.Providers))
		for _, p := range cfg.Providers {
			pcs = append(pcs, llm.ProviderConfig{
				Type:   llm.ProviderType(p.Type),
				URL:    p.URL,
				APIKey: p.APIKey,
				Model:  p.Model,
			})
		}
		return llm.NewClientFromProviders(pcs)
	}
	return llm.NewClient(cfg.Endpoint, cfg.Model)
}

func init() {
	var errorMsg, issue string

	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(aiChatCmd)
	aiCmd.AddCommand(aiDebugCmd)
	aiCmd.AddCommand(aiDoctorCmd)
	aiCmd.AddCommand(aiSuggestCmd)

	aiDebugCmd.Flags().StringVar(&errorMsg, "error", "", "Error message to debug")
	aiSuggestCmd.Flags().StringVar(&issue, "issue", "", "Issue topic (tunnel, batch, lambda, localstack)")
	aiDebugCmd.MarkFlagRequired("error")
	aiSuggestCmd.MarkFlagRequired("issue")
}
