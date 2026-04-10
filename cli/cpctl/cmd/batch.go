package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cpctl/internal/batch"
	"cpctl/internal/config"
	"cpctl/internal/env"

	"github.com/spf13/cobra"
)

var (
	batchStage string // Flag for --stage
	batchNewClient = batch.NewClient
)

// batchCmd represents the batch command group
var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Manage AWS Batch jobs (submit, watch, logs)",
	Long: `Submit, monitor, and debug Batch jobs in local or mirror-cloud environments.

Examples:
  cpctl batch register my-job --image python:3.11 --vcpus 1 --memory 512
  cpctl batch submit my-job-instance my-job-definition my-job-queue --command "python,main.py"
  cpctl batch watch job-123456789012 --interval 5s
  cpctl batch logs job-123456789012 --tail --lines 100
  cpctl batch list --queue my-queue --status RUNNING
`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// batchRegisterCmd registers a new job definition
var batchRegisterCmd = &cobra.Command{
	Use:   "register <definition-name>",
	Short: "Register a new job definition",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		definitionName := args[0]

		// Get stage
		stage, err := getBatchStage(cmd)
		if err != nil {
			return fmt.Errorf("failed to get stage: %w", err)
		}

		slog.Info("registering job definition",
			"name", definitionName,
			"stage", stage,
			"image", cmd.Flag("image").Value.String(),
		)

		// Create batch client
		client, err := batchNewClient(stage)
		if err != nil {
			return fmt.Errorf("failed to create batch client: %w", err)
		}

		// Parse vcpus and memory
		vcpus := 1
		if cmd.Flag("vcpus").Changed {
			vcpus, err = cmd.Flags().GetInt("vcpus")
			if err != nil {
				return fmt.Errorf("invalid vcpus value: %w", err)
			}
		}

		memory := 512
		if cmd.Flag("memory").Changed {
			memory, err = cmd.Flags().GetInt("memory")
			if err != nil {
				return fmt.Errorf("invalid memory value: %w", err)
			}
		}

		// Parse command (comma-separated)
		var command []string
		if cmd.Flag("command").Changed {
			cmdStr := cmd.Flag("command").Value.String()
			if cmdStr != "" {
				command = strings.Split(cmdStr, ",")
			}
		}

		// Build job definition
		def := &batch.JobDefinition{
			Name:       definitionName,
			Image:      cmd.Flag("image").Value.String(),
			Vcpus:      vcpus,
			Memory:     memory,
			Command:    command,
			JobRole:    cmd.Flag("role").Value.String(),
		}

		// Register job definition
		arn, err := client.RegisterJobDefinition(ctx, def)
		if err != nil {
			return fmt.Errorf("failed to register job definition: %w", err)
		}

		// Print success
		fmt.Fprintln(out, "✓ Job definition registered successfully")
		fmt.Fprintf(out, "  Name:     %s\n", definitionName)
		fmt.Fprintf(out, "  ARN:      %s\n", arn)
		fmt.Fprintf(out, "  Stage:    %s\n", stage)

		return nil
	},
}

// batchSubmitCmd submits a job to AWS Batch
var batchSubmitCmd = &cobra.Command{
	Use:   "submit <job-name> <job-definition> <job-queue>",
	Short: "Submit a job to AWS Batch",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		jobName := args[0]
		jobDefinition := args[1]
		jobQueue := args[2]

		// Get stage
		stage, err := getBatchStage(cmd)
		if err != nil {
			return fmt.Errorf("failed to get stage: %w", err)
		}

		slog.Info("submitting batch job",
			"name", jobName,
			"definition", jobDefinition,
			"queue", jobQueue,
			"stage", stage,
		)

		// Create batch client
		client, err := batchNewClient(stage)
		if err != nil {
			return fmt.Errorf("failed to create batch client: %w", err)
		}

		// Parse container overrides
		overrides := &batch.ContainerOverrides{}

		// Parse command (comma-separated)
		if cmd.Flag("command").Changed {
			cmdStr := cmd.Flag("command").Value.String()
			if cmdStr != "" {
				overrides.Command = strings.Split(cmdStr, ",")
			}
		}

		// Parse environment variables
		if cmd.Flag("env").Changed {
			envVars, err := cmd.Flags().GetStringArray("env")
			if err != nil {
				return fmt.Errorf("failed to parse environment variables: %w", err)
			}
			overrides.Environment = make(map[string]string)
			for _, envVar := range envVars {
				parts := strings.SplitN(envVar, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid environment variable format: %s (expected KEY=VALUE)", envVar)
				}
				overrides.Environment[parts[0]] = parts[1]
			}
		}

		// Parse vcpus override
		if cmd.Flag("vcpus").Changed {
			vcpus, err := cmd.Flags().GetInt("vcpus")
			if err != nil {
				return fmt.Errorf("invalid vcpus value: %w", err)
			}
			overrides.Vcpus = vcpus
		}

		// Parse memory override
		if cmd.Flag("memory").Changed {
			memory, err := cmd.Flags().GetInt("memory")
			if err != nil {
				return fmt.Errorf("invalid memory value: %w", err)
			}
			overrides.Memory = memory
		}

		// Build submit request
		req := &batch.SubmitJobRequest{
			JobName:           jobName,
			JobDefinition:     jobDefinition,
			JobQueue:          jobQueue,
			ContainerOverrides: overrides,
		}

		// Submit job
		result, err := client.SubmitJob(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to submit job: %w", err)
		}

		// Print success
		fmt.Fprintln(out, "✓ Job submitted successfully")
		fmt.Fprintf(out, "  Job ID:   %s\n", result.JobID)
		fmt.Fprintf(out, "  Job ARN:  %s\n", result.JobArn)
		fmt.Fprintf(out, "  Status:   %s\n", result.Status)
		fmt.Fprintf(out, "  Stage:    %s\n", stage)
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "To watch job progress:")
		fmt.Fprintf(out, "  cpctl batch watch %s\n", result.JobID)
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "To view logs:")
		fmt.Fprintf(out, "  cpctl batch logs %s\n", result.JobID)

		return nil
	},
}

// batchWatchCmd watches a job's progress
var batchWatchCmd = &cobra.Command{
	Use:   "watch <job-id>",
	Short: "Watch a job's progress",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		jobID := args[0]

		// Get stage
		stage, err := getBatchStage(cmd)
		if err != nil {
			return fmt.Errorf("failed to get stage: %w", err)
		}

		// Parse interval and timeout
		interval := 5 * time.Second
		if cmd.Flag("interval").Changed {
			intervalStr := cmd.Flag("interval").Value.String()
			duration, err := time.ParseDuration(intervalStr)
			if err != nil {
				return fmt.Errorf("invalid interval format: %w", err)
			}
			interval = duration
		}

		timeout := 3600 * time.Second
		if cmd.Flag("timeout").Changed {
			timeoutStr := cmd.Flag("timeout").Value.String()
			duration, err := time.ParseDuration(timeoutStr)
			if err != nil {
				return fmt.Errorf("invalid timeout format: %w", err)
			}
			timeout = duration
		}

		slog.Info("watching batch job",
			"job_id", jobID,
			"stage", stage,
			"interval", interval,
			"timeout", timeout,
		)

		// Create batch client
		client, err := batchNewClient(stage)
		if err != nil {
			return fmt.Errorf("failed to create batch client: %w", err)
		}

		// Set up context with timeout
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Set up signal handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Track previous status for change detection
		prevStatus := ""
		startTime := time.Now()
		statusIcons := map[string]string{
			"SUBMITTED": "•",
			"PENDING":   "•",
			"RUNNABLE":  "•",
			"STARTING":  "•",
			"RUNNING":   "•",
			"SUCCEEDED": "✓",
			"FAILED":    "✗",
		}

		fmt.Fprintf(out, "Watching job %s (stage: %s)\n", jobID, stage)
		fmt.Fprintf(out, "Polling every %v, timeout after %v\n\n", interval, timeout)

		for {
			select {
			case <-ctx.Done():
				fmt.Fprintf(out, "\n⏰ Timeout reached after %v\n", time.Since(startTime))
				return fmt.Errorf("watch timeout")
			case sig := <-sigChan:
				fmt.Fprintf(out, "\n⚠️  Received signal: %v\n", sig)
				return nil
			case <-time.After(interval):
				// Get job status
				job, err := client.DescribeJob(ctx, jobID)
				if err != nil {
					fmt.Fprintf(out, "\n❌ Error getting job status: %v\n", err)
					return err
				}

				// Print status if changed
				if job.Status != prevStatus {
					icon := statusIcons[job.Status]
					if icon == "" {
						icon = "?"
					}

					duration := time.Since(startTime).Round(time.Second)
					fmt.Fprintf(out, "[%s] %s %s (elapsed: %v)\n", 
						time.Now().Format("15:04:05"), 
						icon, 
						job.Status, 
						duration)

					prevStatus = job.Status

					// Check for terminal status
					if job.Status == "SUCCEEDED" || job.Status == "FAILED" {
						fmt.Fprintln(out)
						if job.Status == "SUCCEEDED" {
							fmt.Fprintf(out, "✅ Job completed successfully in %v\n", duration)
						} else {
							fmt.Fprintf(out, "❌ Job failed after %v\n", duration)
							if job.Reason != "" {
								fmt.Fprintf(out, "   Reason: %s\n", job.Reason)
							}
						}
						return nil
					}
				}
			}
		}
	},
}

// batchLogsCmd retrieves job logs
var batchLogsCmd = &cobra.Command{
	Use:   "logs <job-id>",
	Short: "Retrieve job logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		jobID := args[0]

		// Get stage
		stage, err := getBatchStage(cmd)
		if err != nil {
			return fmt.Errorf("failed to get stage: %w", err)
		}

		// Parse lines
		lines := 50
		if cmd.Flag("lines").Changed {
			lines, err = cmd.Flags().GetInt("lines")
			if err != nil {
				return fmt.Errorf("invalid lines value: %w", err)
			}
		}

		slog.Info("retrieving batch logs",
			"job_id", jobID,
			"stage", stage,
			"tail", cmd.Flag("tail").Changed,
			"lines", lines,
		)

		// Create batch client
		client, err := batchNewClient(stage)
		if err != nil {
			return fmt.Errorf("failed to create batch client: %w", err)
		}

		// Check if tail mode
		if cmd.Flag("tail").Changed {
			fmt.Fprintf(out, "Streaming logs for job %s (stage: %s)\n", jobID, stage)
			fmt.Fprintln(out, "Press Ctrl+C to stop")
			fmt.Fprintln(out)

			// Set up signal handling
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			// Create log channel
			logChan := make(chan string, 100)

			// Start tailing logs in goroutine
			go func() {
				if err := client.TailLogs(ctx, jobID, logChan); err != nil {
					fmt.Fprintf(out, "Error tailing logs: %v\n", err)
				}
			}()

			// Process logs
			for {
				select {
				case <-sigChan:
					fmt.Fprintln(out, "")
					fmt.Fprintln(out, "Stopping log stream...")
					return nil
				case logLine, ok := <-logChan:
					if !ok {
						return nil // Channel closed
					}
					fmt.Fprintln(out, logLine)
				}
			}
		} else {
			// Static log retrieval
			fmt.Fprintf(out, "Logs for job %s (stage: %s)\n", jobID, stage)
			fmt.Fprintf(out, "Showing last %d lines\n\n", lines)

			logs, err := client.GetLogs(ctx, jobID)
			if err != nil {
				return fmt.Errorf("failed to get logs: %w", err)
			}

			// Print logs
			for _, logLine := range logs {
				fmt.Fprintln(out, logLine)
			}

			if len(logs) == 0 {
				fmt.Fprintln(out, "No logs available")
			}
		}

		return nil
	},
}

// batchListCmd lists jobs in a queue
var batchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs in a queue",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()

		// Get stage
		stage, err := getBatchStage(cmd)
		if err != nil {
			return fmt.Errorf("failed to get stage: %w", err)
		}

		// Get queue name (required)
		queueName := cmd.Flag("queue").Value.String()
		if queueName == "" {
			return fmt.Errorf("queue name is required (use --queue)")
		}

		// Get status filter
		status := "RUNNING"
		if cmd.Flag("status").Changed {
			status = cmd.Flag("status").Value.String()
		}

		slog.Info("listing batch jobs",
			"stage", stage,
			"queue", queueName,
			"status", status,
		)

		// Create batch client
		client, err := batchNewClient(stage)
		if err != nil {
			return fmt.Errorf("failed to create batch client: %w", err)
		}

		// List jobs
		jobs, err := client.ListJobs(ctx, queueName, status)
		if err != nil {
			return fmt.Errorf("failed to list jobs: %w", err)
		}

		// Print table
		fmt.Fprintf(out, "Jobs in queue '%s' (status: %s, stage: %s)\n", queueName, status, stage)
		fmt.Fprintln(out)
		if len(jobs) == 0 {
			fmt.Fprintln(out, "No jobs found")
			return nil
		}

		// Print header
		fmt.Fprintf(out, "%-36s %-30s %-12s %-10s\n", "JOB ID", "NAME", "STATUS", "DURATION")
		fmt.Fprintf(out, "%-36s %-30s %-12s %-10s\n", 
			strings.Repeat("-", 36), 
			strings.Repeat("-", 30), 
			strings.Repeat("-", 12), 
			strings.Repeat("-", 10))

		// Print jobs
		for _, job := range jobs {
			duration := "N/A"
			if job.SubmittedAt > 0 {
				submittedTime := time.Unix(job.SubmittedAt, 0)
				if job.StoppedAt > 0 {
					stoppedTime := time.Unix(job.StoppedAt, 0)
					duration = stoppedTime.Sub(submittedTime).Round(time.Second).String()
				} else {
					duration = time.Since(submittedTime).Round(time.Second).String()
				}
			}

			// Truncate name if too long
			name := job.JobName
			if len(name) > 28 {
				name = name[:25] + "..."
			}

			fmt.Fprintf(out, "%-36s %-30s %-12s %-10s\n", 
				job.JobID, 
				name, 
				job.Status, 
				duration)
		}

		fmt.Fprintf(out, "\nTotal: %d job(s)\n", len(jobs))
		return nil
	},
}

// getBatchStage returns the stage from flag or environment manager
func getBatchStage(cmd *cobra.Command) (string, error) {
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
	rootCmd.AddCommand(batchCmd)
	batchCmd.AddCommand(batchRegisterCmd)
	batchCmd.AddCommand(batchSubmitCmd)
	batchCmd.AddCommand(batchWatchCmd)
	batchCmd.AddCommand(batchLogsCmd)
	batchCmd.AddCommand(batchListCmd)

	// Global stage flag
	batchCmd.PersistentFlags().StringVar(&batchStage, "stage", "", "Target stage (localstack|moto|mirror)")

	// Register command flags
	batchRegisterCmd.Flags().String("image", "", "Docker image URI (required)")
	batchRegisterCmd.Flags().Int("vcpus", 1, "vCPU count")
	batchRegisterCmd.Flags().Int("memory", 512, "Memory in MB")
	batchRegisterCmd.Flags().String("command", "", "Command override (comma-separated)")
	batchRegisterCmd.Flags().String("role", "", "Job role ARN")
	batchRegisterCmd.MarkFlagRequired("image")

	// Submit command flags
	batchSubmitCmd.Flags().String("command", "", "Override job command (comma-separated)")
	batchSubmitCmd.Flags().StringArray("env", []string{}, "Environment variables (KEY=VALUE, repeatable)")
	batchSubmitCmd.Flags().Int("vcpus", 0, "Override vCPU limit (0 = use definition default)")
	batchSubmitCmd.Flags().Int("memory", 0, "Override memory limit in MB (0 = use definition default)")

	// Watch command flags
	batchWatchCmd.Flags().String("interval", "5s", "Poll interval (e.g., 5s, 1m)")
	batchWatchCmd.Flags().String("timeout", "3600s", "Give up after timeout (e.g., 30m, 1h)")

	// Logs command flags
	batchLogsCmd.Flags().Bool("tail", false, "Stream logs live")
	batchLogsCmd.Flags().Int("lines", 50, "Number of log lines to retrieve")

	// List command flags
	batchListCmd.Flags().String("queue", "", "Job queue name (required)")
	batchListCmd.Flags().String("status", "RUNNING", "Job status filter")
	batchListCmd.MarkFlagRequired("queue")
}