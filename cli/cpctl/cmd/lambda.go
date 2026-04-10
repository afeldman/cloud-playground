package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"cpctl/internal/config"
	"cpctl/internal/env"
	"cpctl/internal/lambda"

	"github.com/spf13/cobra"
)

var (
	lambdaStage string // Flag for --stage
	lambdaNewClient = lambda.NewClient
)

// lambdaCmd represents the lambda command group
var lambdaCmd = &cobra.Command{
	Use:   "lambda",
	Short: "Manage Lambda functions (deploy, logs, invoke)",
	Long: `Deploy, test, and monitor Lambda functions in local or mirror-cloud environments.

Examples:
  cpctl lambda deploy ./function.zip --name my-function --runtime python3.11 --handler index.handler --role arn:aws:iam::123456789012:role/lambda-role
  cpctl lambda logs my-function --tail --lines 100
  cpctl lambda invoke my-function --payload '{"key":"value"}'
  cpctl lambda invoke my-function --async --payload @payload.json
`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// lambdaDeployCmd deploys a Lambda function
var lambdaDeployCmd = &cobra.Command{
	Use:   "deploy [zip-file]",
	Short: "Deploy a Lambda function from a ZIP file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		zipPath := args[0]
		
		// Get stage
		stage, err := getStage(cmd)
		if err != nil {
			return fmt.Errorf("failed to get stage: %w", err)
		}

		slog.Info("deploying Lambda function", 
			"zip", zipPath, 
			"stage", stage,
			"name", cmd.Flag("name").Value.String(),
		)

		// Read ZIP file
		_, err = os.ReadFile(zipPath)
		if err != nil {
			return fmt.Errorf("failed to read ZIP file %s: %w", zipPath, err)
		}

		// Create lambda client
		client, err := lambdaNewClient(stage)
		if err != nil {
			return fmt.Errorf("failed to create Lambda client: %w", err)
		}

		// Parse timeout and memory
		timeout := 30
		if cmd.Flag("timeout").Changed {
			timeout, err = cmd.Flags().GetInt("timeout")
			if err != nil {
				return fmt.Errorf("invalid timeout value: %w", err)
			}
		}

		memory := 128
		if cmd.Flag("memory").Changed {
			memory, err = cmd.Flags().GetInt("memory")
			if err != nil {
				return fmt.Errorf("invalid memory value: %w", err)
			}
		}

		// Create function definition
		fn := &lambda.Function{
			Name:        cmd.Flag("name").Value.String(),
			Runtime:     cmd.Flag("runtime").Value.String(),
			Handler:     cmd.Flag("handler").Value.String(),
			Role:        cmd.Flag("role").Value.String(),
			CodePath:    zipPath,
			Timeout:     timeout,
			Memory:      memory,
			Environment: make(map[string]string),
		}

		// Deploy function
		result, err := client.Deploy(ctx, fn)
		if err != nil {
			return fmt.Errorf("failed to deploy Lambda function: %w", err)
		}

		// Print success message
		fmt.Fprintf(out, `
✅ Lambda function deployed successfully!

📊 Deployment Details:
   Function Name:  %s
   Function ARN:   %s
   Version:        %s
   Runtime:        %s
   Handler:        %s
   Memory:         %d MB
   Timeout:        %d seconds
   Stage:          %s

💡 Next Steps:
   cpctl lambda invoke %s --payload '{}'
   cpctl lambda logs %s --tail
`,
			result.FunctionName,
			result.FunctionArn,
			result.Version,
			fn.Runtime,
			fn.Handler,
			fn.Memory,
			fn.Timeout,
			stage,
			fn.Name,
			fn.Name,
		)

		return nil
	},
}

// lambdaInvokeCmd invokes a Lambda function
var lambdaInvokeCmd = &cobra.Command{
	Use:   "invoke [function-name]",
	Short: "Invoke a Lambda function with payload",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		functionName := args[0]
		
		// Get stage
		stage, err := getStage(cmd)
		if err != nil {
			return fmt.Errorf("failed to get stage: %w", err)
		}

		slog.Info("invoking Lambda function", 
			"function", functionName, 
			"stage", stage,
			"async", cmd.Flag("async").Value.String(),
		)

		// Parse payload
		var payload []byte
		if cmd.Flag("payload").Changed {
			payloadStr := cmd.Flag("payload").Value.String()
			// Check if payload is a file reference (starts with @)
			if len(payloadStr) > 0 && payloadStr[0] == '@' {
				filePath := payloadStr[1:]
				payload, err = os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("failed to read payload file %s: %w", filePath, err)
				}
			} else {
				payload = []byte(payloadStr)
			}
		}

		// Create lambda client
		client, err := lambdaNewClient(stage)
		if err != nil {
			return fmt.Errorf("failed to create Lambda client: %w", err)
		}

		// Create invoke request
		req := &lambda.InvokeRequest{
			FunctionName: functionName,
			Payload:      payload,
			Async:        cmd.Flag("async").Value.String() == "true",
		}

		// Invoke function
		result, err := client.Invoke(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to invoke Lambda function: %w", err)
		}

		// Print results
		statusIcon := "✅"
		if result.StatusCode >= 400 {
			statusIcon = "❌"
		}

		fmt.Fprintf(out, `
%s Lambda function invoked

📊 Invocation Details:
   Function:      %s
   Status Code:   %d
   Executed Version: %s
   Stage:         %s
   Async:         %v
`,
			statusIcon,
			functionName,
			result.StatusCode,
			result.ExecutedVersion,
			stage,
			req.Async,
		)

		if result.FunctionErr != "" {
			fmt.Fprintf(out, "   Error:         %s\n", result.FunctionErr)
		}

		if result.LogResult != "" {
			fmt.Fprintf(out, "\n📝 Log Result:\n%s\n", result.LogResult)
		}

		if len(result.Payload) > 0 {
			fmt.Fprintln(out, "")
			fmt.Fprintln(out, "📦 Response Payload:")
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, result.Payload, "", "  "); err == nil {
				fmt.Fprintln(out, prettyJSON.String())
			} else {
				fmt.Fprintln(out, string(result.Payload))
			}
		}

		return nil
	},
}

// lambdaLogsCmd shows Lambda function logs
var lambdaLogsCmd = &cobra.Command{
	Use:   "logs [function-name]",
	Short: "Show Lambda function logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		functionName := args[0]
		
		// Get stage
		stage, err := getStage(cmd)
		if err != nil {
			return fmt.Errorf("failed to get stage: %w", err)
		}

		slog.Info("fetching Lambda logs", 
			"function", functionName, 
			"stage", stage,
			"tail", cmd.Flag("tail").Value.String(),
			"lines", cmd.Flag("lines").Value.String(),
		)

		// Create lambda client
		client, err := lambdaNewClient(stage)
		if err != nil {
			return fmt.Errorf("failed to create Lambda client: %w", err)
		}

		// Parse lines limit
		lines := 50
		if cmd.Flag("lines").Changed {
			lines, err = cmd.Flags().GetInt("lines")
			if err != nil {
				return fmt.Errorf("invalid lines value: %w", err)
			}
		}

		// Check if tail mode
		if cmd.Flag("tail").Value.String() == "true" {
			fmt.Fprintf(out, "🔍 Tailing logs for %s (stage: %s)...\n", functionName, stage)
			fmt.Fprintln(out, "Press Ctrl+C to stop")
			fmt.Fprintln(out)

			logChan := make(chan lambda.LogEntry, 100)
			go func() {
				if err := client.TailLogs(ctx, functionName, logChan); err != nil {
					slog.Error("failed to tail logs", "error", err)
				}
			}()

			for logEntry := range logChan {
				timestamp := time.Unix(logEntry.Timestamp, 0).Format("2006-01-02 15:04:05")
				levelIcon := "ℹ️"
				if logEntry.Level == "ERROR" {
					levelIcon = "❌"
				} else if logEntry.Level == "WARNING" {
					levelIcon = "⚠️"
				}
				fmt.Fprintf(out, "%s [%s] %s: %s\n", levelIcon, timestamp, logEntry.Level, logEntry.Message)
			}

			return nil
		}

		// Get logs (non-tail mode)
		fmt.Fprintf(out, "📝 Logs for %s (stage: %s):\n\n", functionName, stage)
		
		logs, err := client.GetLogs(ctx, functionName, lines)
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}

		if len(logs) == 0 {
			fmt.Fprintln(out, "No logs found.")
			return nil
		}

		for _, logEntry := range logs {
			timestamp := time.Unix(logEntry.Timestamp, 0).Format("2006-01-02 15:04:05")
			levelIcon := "ℹ️"
			if logEntry.Level == "ERROR" {
				levelIcon = "❌"
			} else if logEntry.Level == "WARNING" {
				levelIcon = "⚠️"
			}
			fmt.Fprintf(out, "%s [%s] %s: %s\n", levelIcon, timestamp, logEntry.Level, logEntry.Message)
		}

		fmt.Fprintf(out, "\n📊 Total logs: %d\n", len(logs))

		return nil
	},
}

// getStage returns the stage from flag or environment manager
func getStage(cmd *cobra.Command) (string, error) {
	// Check if stage flag is provided
	if cmd.Flag("stage").Changed {
		stage := cmd.Flag("stage").Value.String()
		// Validate stage
		if stage != "localstack" && stage != "mirror" && stage != "moto" {
			return "", fmt.Errorf("invalid stage: %s (must be 'localstack', 'moto' or 'mirror')", stage)
		}
		return stage, nil
	}

	// Get stage from environment manager
	manager, err := env.NewEnvironmentManager("data/environments", &config.Cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create environment manager: %w", err)
	}

	stage := manager.GetStage()
	switch stage {
	case env.StageMoto:
		return "moto", nil
	case env.StageLocalStack:
		return "localstack", nil
	case env.StageMirror:
		return "mirror", nil
	default:
		return "", fmt.Errorf("unknown stage: %v", stage)
	}
}

func init() {
	// Add subcommands
	lambdaCmd.AddCommand(lambdaDeployCmd)
	lambdaCmd.AddCommand(lambdaInvokeCmd)
	lambdaCmd.AddCommand(lambdaLogsCmd)

	// Add flags to deploy command
	lambdaDeployCmd.Flags().String("name", "", "Function name (required)")
	lambdaDeployCmd.Flags().String("runtime", "", "Runtime (e.g., python3.11, go1.x, nodejs18.x) (required)")
	lambdaDeployCmd.Flags().String("handler", "", "Handler path (e.g., index.handler) (required)")
	lambdaDeployCmd.Flags().String("role", "", "IAM role ARN (required)")
	lambdaDeployCmd.Flags().Int("timeout", 30, "Timeout in seconds (default: 30)")
	lambdaDeployCmd.Flags().Int("memory", 128, "Memory in MB (default: 128)")
	lambdaDeployCmd.Flags().StringVar(&lambdaStage, "stage", "", "Override stage (localstack|moto|mirror)")
	
	// Mark required flags
	lambdaDeployCmd.MarkFlagRequired("name")
	lambdaDeployCmd.MarkFlagRequired("runtime")
	lambdaDeployCmd.MarkFlagRequired("handler")
	lambdaDeployCmd.MarkFlagRequired("role")

	// Add flags to invoke command
	lambdaInvokeCmd.Flags().String("payload", "", "JSON payload (inline or @file.json)")
	lambdaInvokeCmd.Flags().Bool("async", false, "Async invocation (default: sync)")
	lambdaInvokeCmd.Flags().StringVar(&lambdaStage, "stage", "", "Override stage (localstack|moto|mirror)")

	// Add flags to logs command
	lambdaLogsCmd.Flags().Bool("tail", false, "Stream logs live (default: static)")
	lambdaLogsCmd.Flags().Int("lines", 50, "Number of log lines to retrieve")
	lambdaLogsCmd.Flags().String("since", "", "Time filter (e.g., 1h, 10m)")
	lambdaLogsCmd.Flags().StringVar(&lambdaStage, "stage", "", "Override stage (localstack|moto|mirror)")

	// Register with root
	rootCmd.AddCommand(lambdaCmd)
}