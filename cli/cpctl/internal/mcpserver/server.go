package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"time"

	"cpctl/internal/tui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"sigs.k8s.io/yaml"
)

// Serve starts the MCP server over stdio (for Claude Desktop integration).
func Serve(root, clusterName string) error {
	profile := mcpProfileFromEnv()
	allowMutating := profileAllowsMutating(profile)

	s := server.NewMCPServer(
		"birdy-playground",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	registerStatusTools(s, root, clusterName)
	registerTerraformReadOnlyTools(s, root)
	registerLambdaBatchTools(s, root)
	if allowMutating {
		registerLifecycleTools(s, root, clusterName)
		registerSyncApplyTools(s, root, clusterName)
		registerTerraformMutatingTools(s, root)
		registerLambdaBatchMutatingTools(s, root)
	}

	return server.ServeStdio(s)
}

// ── Status ────────────────────────────────────────────────────────────────────

func registerStatusTools(s *server.MCPServer, root, clusterName string) {
	s.AddTool(
		mcp.NewTool("playground_status",
			mcp.WithDescription("Show current status of Kind cluster, LocalStack, and Terraform resources"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			st := tui.CheckAll(root, clusterName)
			return mcp.NewToolResultText(formatStatus(st)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("localstack_logs",
			mcp.WithDescription("Fetch recent LocalStack container logs"),
			mcp.WithNumber("lines",
				mcp.Description("Number of log lines to return (default 50)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			lines := req.GetInt("lines", 50)
			compose := filepath.Join(root, "localstack", "docker-compose.yml")
			out, err := capture("docker", "compose", "-f", compose, "logs", "--no-color", "--tail", fmt.Sprintf("%d", lines))
			if err != nil {
				return mcp.NewToolResultText("error fetching logs:\n" + out), nil
			}
			return mcp.NewToolResultText(out), nil
		},
	)
}

// ── Lambda & Batch ───────────────────────────────────────────────────────────

func registerLambdaBatchTools(s *server.MCPServer, root string) {
	s.AddTool(
		mcp.NewTool("lambda_logs",
			mcp.WithDescription("Read Lambda logs via cpctl without mutating infrastructure"),
			mcp.WithString("function_name", mcp.Required(), mcp.Description("Lambda function name")),
			mcp.WithString("stage",
				mcp.Description("Target stage (default localstack)"),
				mcp.Enum("localstack", "mirror"),
			),
			mcp.WithNumber("lines", mcp.Description("Number of log lines (default 50)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			functionName := strings.TrimSpace(req.GetString("function_name", ""))
			if functionName == "" {
				return mcp.NewToolResultText("function_name is required"), nil
			}

			stage := req.GetString("stage", "localstack")
			lines := req.GetInt("lines", 50)
			out, err := captureSelf(root,
				"lambda", "logs", functionName,
				"--stage", stage,
				"--lines", fmt.Sprintf("%d", lines),
			)
			if err != nil {
				return mcp.NewToolResultText("lambda logs failed:\n" + out), nil
			}
			return mcp.NewToolResultText(out), nil
		},
	)

	s.AddTool(
		mcp.NewTool("batch_watch",
			mcp.WithDescription("Read AWS Batch job status progression"),
			mcp.WithString("job_id", mcp.Required(), mcp.Description("Batch job ID")),
			mcp.WithString("stage",
				mcp.Description("Target stage (default localstack)"),
				mcp.Enum("localstack", "mirror"),
			),
			mcp.WithString("interval", mcp.Description("Polling interval (default 5s)")),
			mcp.WithString("timeout", mcp.Description("Timeout duration (default 60s for MCP call)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			jobID := strings.TrimSpace(req.GetString("job_id", ""))
			if jobID == "" {
				return mcp.NewToolResultText("job_id is required"), nil
			}

			stage := req.GetString("stage", "localstack")
			interval := req.GetString("interval", "5s")
			timeout := req.GetString("timeout", "60s")
			out, err := captureSelf(root,
				"batch", "watch", jobID,
				"--stage", stage,
				"--interval", interval,
				"--timeout", timeout,
			)
			if err != nil {
				return mcp.NewToolResultText("batch watch failed:\n" + out), nil
			}
			return mcp.NewToolResultText(out), nil
		},
	)

	s.AddTool(
		mcp.NewTool("batch_logs",
			mcp.WithDescription("Read AWS Batch logs for a job"),
			mcp.WithString("job_id", mcp.Required(), mcp.Description("Batch job ID")),
			mcp.WithString("stage",
				mcp.Description("Target stage (default localstack)"),
				mcp.Enum("localstack", "mirror"),
			),
			mcp.WithNumber("lines", mcp.Description("Number of log lines (default 50)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			jobID := strings.TrimSpace(req.GetString("job_id", ""))
			if jobID == "" {
				return mcp.NewToolResultText("job_id is required"), nil
			}

			stage := req.GetString("stage", "localstack")
			lines := req.GetInt("lines", 50)
			out, err := captureSelf(root,
				"batch", "logs", jobID,
				"--stage", stage,
				"--lines", fmt.Sprintf("%d", lines),
			)
			if err != nil {
				return mcp.NewToolResultText("batch logs failed:\n" + out), nil
			}
			return mcp.NewToolResultText(out), nil
		},
	)
}

func registerLambdaBatchMutatingTools(s *server.MCPServer, root string) {
	s.AddTool(
		mcp.NewTool("lambda_invoke",
			mcp.WithDescription("[MUTATING] Invoke a Lambda function through cpctl"),
			mcp.WithString("function_name", mcp.Required(), mcp.Description("Lambda function name")),
			mcp.WithString("payload", mcp.Description("Inline JSON payload")),
			mcp.WithBoolean("async", mcp.Description("Async invocation")),
			mcp.WithString("stage",
				mcp.Description("Target stage (default localstack)"),
				mcp.Enum("localstack", "mirror"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			functionName := strings.TrimSpace(req.GetString("function_name", ""))
			if functionName == "" {
				return mcp.NewToolResultText("function_name is required"), nil
			}

			stage := req.GetString("stage", "localstack")
			args := []string{"lambda", "invoke", functionName, "--stage", stage}
			if payload := strings.TrimSpace(req.GetString("payload", "")); payload != "" {
				args = append(args, "--payload", payload)
			}
			if req.GetBool("async", false) {
				args = append(args, "--async")
			}

			out, err := captureSelf(root, args...)
			if err != nil {
				return mcp.NewToolResultText("lambda invoke failed:\n" + out), nil
			}
			return mcp.NewToolResultText(out), nil
		},
	)

	s.AddTool(
		mcp.NewTool("lambda_deploy",
			mcp.WithDescription("[MUTATING] Deploy a Lambda ZIP through cpctl"),
			mcp.WithString("zip_file", mcp.Required(), mcp.Description("Path to ZIP artifact")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Function name")),
			mcp.WithString("runtime", mcp.Required(), mcp.Description("Lambda runtime")),
			mcp.WithString("handler", mcp.Required(), mcp.Description("Lambda handler path")),
			mcp.WithString("role", mcp.Required(), mcp.Description("IAM role ARN")),
			mcp.WithNumber("timeout", mcp.Description("Timeout in seconds (default 30)")),
			mcp.WithNumber("memory", mcp.Description("Memory in MB (default 128)")),
			mcp.WithString("stage",
				mcp.Description("Target stage (default localstack)"),
				mcp.Enum("localstack", "mirror"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			zipFile := strings.TrimSpace(req.GetString("zip_file", ""))
			name := strings.TrimSpace(req.GetString("name", ""))
			runtime := strings.TrimSpace(req.GetString("runtime", ""))
			handler := strings.TrimSpace(req.GetString("handler", ""))
			role := strings.TrimSpace(req.GetString("role", ""))
			if zipFile == "" || name == "" || runtime == "" || handler == "" || role == "" {
				return mcp.NewToolResultText("zip_file, name, runtime, handler and role are required"), nil
			}

			stage := req.GetString("stage", "localstack")
			args := []string{
				"lambda", "deploy", zipFile,
				"--name", name,
				"--runtime", runtime,
				"--handler", handler,
				"--role", role,
				"--stage", stage,
			}
			if timeout := req.GetInt("timeout", 0); timeout > 0 {
				args = append(args, "--timeout", fmt.Sprintf("%d", timeout))
			}
			if memory := req.GetInt("memory", 0); memory > 0 {
				args = append(args, "--memory", fmt.Sprintf("%d", memory))
			}

			out, err := captureSelf(root, args...)
			if err != nil {
				return mcp.NewToolResultText("lambda deploy failed:\n" + out), nil
			}
			return mcp.NewToolResultText(out), nil
		},
	)

	s.AddTool(
		mcp.NewTool("batch_submit",
			mcp.WithDescription("[MUTATING] Submit a Batch job through cpctl"),
			mcp.WithString("job_name", mcp.Required(), mcp.Description("Batch job name")),
			mcp.WithString("job_definition", mcp.Required(), mcp.Description("Job definition")),
			mcp.WithString("job_queue", mcp.Required(), mcp.Description("Job queue")),
			mcp.WithString("stage",
				mcp.Description("Target stage (default localstack)"),
				mcp.Enum("localstack", "mirror"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			jobName := strings.TrimSpace(req.GetString("job_name", ""))
			jobDefinition := strings.TrimSpace(req.GetString("job_definition", ""))
			jobQueue := strings.TrimSpace(req.GetString("job_queue", ""))
			if jobName == "" || jobDefinition == "" || jobQueue == "" {
				return mcp.NewToolResultText("job_name, job_definition and job_queue are required"), nil
			}

			stage := req.GetString("stage", "localstack")
			out, err := captureSelf(root,
				"batch", "submit", jobName, jobDefinition, jobQueue,
				"--stage", stage,
			)
			if err != nil {
				return mcp.NewToolResultText("batch submit failed:\n" + out), nil
			}
			return mcp.NewToolResultText(out), nil
		},
	)
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

func registerLifecycleTools(s *server.MCPServer, root, clusterName string) {
	compose := filepath.Join(root, "localstack", "docker-compose.yml")
	kindCfg := filepath.Join(root, "kind", "cluster-config.yaml")

	s.AddTool(
		mcp.NewTool("playground_up",
			mcp.WithDescription("Start the playground: create Kind cluster, start LocalStack, run Terraform apply"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var log strings.Builder

			step(&log, "deleting existing Kind cluster (clean start)")
			capture("kind", "delete", "cluster", "--name", clusterName) //nolint:errcheck

			step(&log, "creating Kind cluster")
			if out, err := capture("kind", "create", "cluster", "--name", clusterName, "--config", kindCfg); err != nil {
				return result(&log, "kind create failed: "+out), nil
			}

			step(&log, "starting LocalStack")
			if out, err := capture("docker", "compose", "-f", compose, "up", "-d"); err != nil {
				return result(&log, "docker compose failed: "+out), nil
			}

			step(&log, "waiting for LocalStack (up to 60s)")
			if err := waitLocalStack(60 * time.Second); err != nil {
				return result(&log, "localstack not ready: "+err.Error()), nil
			}
			log.WriteString("  LocalStack ready\n")

			tfDir := filepath.Join(root, "terraform", "localstack")
			step(&log, "terraform init")
			if out, err := capture("terraform", "-chdir="+tfDir, "init", "-no-color"); err != nil {
				return result(&log, "terraform init failed:\n"+out), nil
			}

			step(&log, "terraform apply")
			if out, err := capture("terraform", "-chdir="+tfDir, "apply", "-auto-approve", "-no-color"); err != nil {
				return result(&log, "terraform apply failed:\n"+out), nil
			}

			log.WriteString("\nPlayground is up.\n")
			return mcp.NewToolResultText(log.String()), nil
		},
	)

	s.AddTool(
		mcp.NewTool("playground_down",
			mcp.WithDescription("Stop and destroy the playground: remove Kind cluster, stop LocalStack (including volumes)"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var log strings.Builder
			step(&log, "stopping LocalStack")
			capture("docker", "compose", "-f", compose, "down", "-v") //nolint:errcheck
			step(&log, "deleting Kind cluster")
			capture("kind", "delete", "cluster", "--name", clusterName) //nolint:errcheck
			log.WriteString("\nPlayground is down.\n")
			return mcp.NewToolResultText(log.String()), nil
		},
	)

	s.AddTool(
		mcp.NewTool("playground_update",
			mcp.WithDescription("Update all playground components: pull Kind node images, LocalStack images, reconfigure AWS profiles, upgrade Terraform providers"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var log strings.Builder
			awsConfig := filepath.Join(root, "aws-local", "aws-config.sh")
			tfDir := filepath.Join(root, "terraform", "localstack")

			for _, image := range kindNodeImages(filepath.Join(root, "kind", "cluster-config.yaml")) {
				step(&log, "docker pull "+image)
				if out, err := capture("docker", "pull", image); err != nil {
					log.WriteString("  warning: " + out + "\n")
				}
			}

			step(&log, "pulling LocalStack images")
			if out, err := capture("docker", "compose", "-f", compose, "pull"); err != nil {
				log.WriteString("  warning: " + out + "\n")
			}

			step(&log, "reconfiguring AWS profiles")
			if out, err := capture("bash", awsConfig); err != nil {
				log.WriteString("  warning: " + out + "\n")
			}

			step(&log, "upgrading Terraform providers")
			if out, err := capture("terraform", "-chdir="+tfDir, "init", "-upgrade", "-no-color"); err != nil {
				return result(&log, "terraform init -upgrade failed:\n"+out), nil
			}

			log.WriteString("\nUpdate complete.\n")
			return mcp.NewToolResultText(log.String()), nil
		},
	)
}

// ── Sync & Apply ─────────────────────────────────────────────────────────────

func registerSyncApplyTools(s *server.MCPServer, root, clusterName string) {
	s.AddTool(
		mcp.NewTool("playground_sync",
			mcp.WithDescription("Sync config and secrets into Kubernetes from local files"),
			mcp.WithString("source",
				mcp.Description("Sync source: local (default), aws-ssm, aws-sm"),
				mcp.Enum("local", "aws-ssm", "aws-sm"),
			),
			mcp.WithBoolean("dry_run",
				mcp.Description("Preview changes without applying"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := []string{"sync"}
			if src := req.GetString("source", "local"); src != "" {
				args = append(args, "--source", src)
			}
			if req.GetBool("dry_run", false) {
				args = append(args, "--dry-run")
			}
			out, err := captureSelf(root, args...)
			if err != nil {
				return mcp.NewToolResultText("sync failed:\n" + out), nil
			}
			return mcp.NewToolResultText(out), nil
		},
	)

	s.AddTool(
		mcp.NewTool("playground_apply",
			mcp.WithDescription("Sanitize manifests and apply them to the Kind cluster"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			out, err := captureSelf(root, "apply")
			if err != nil {
				return mcp.NewToolResultText("apply failed:\n" + out), nil
			}
			return mcp.NewToolResultText(out), nil
		},
	)
}

// ── Terraform ─────────────────────────────────────────────────────────────────

func registerTerraformReadOnlyTools(s *server.MCPServer, root string) {
	tfDir := filepath.Join(root, "tofu", "localstack")

	s.AddTool(
		mcp.NewTool("terraform_plan",
			mcp.WithDescription("Preview OpenTofu infrastructure changes without applying them"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			out, err := capture("tofu", "-chdir="+tfDir, "plan", "-no-color")
			if err != nil {
				return mcp.NewToolResultText("terraform plan failed:\n" + out), nil
			}
			return mcp.NewToolResultText(out), nil
		},
	)
}

func registerTerraformMutatingTools(s *server.MCPServer, root string) {
	tfDir := filepath.Join(root, "tofu", "localstack")

	s.AddTool(
		mcp.NewTool("terraform_apply",
			mcp.WithDescription("[MUTATING] Apply OpenTofu infrastructure changes to LocalStack"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			out, err := capture("tofu", "-chdir="+tfDir, "apply", "-auto-approve", "-no-color")
			if err != nil {
				return mcp.NewToolResultText("terraform apply failed:\n" + out), nil
			}
			return mcp.NewToolResultText(out), nil
		},
	)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// capture runs a command and returns combined stdout+stderr as a string.
func capture(name string, args ...string) (string, error) {
	cmd := osexec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return strings.TrimSpace(buf.String()), err
}

// captureSelf re-invokes the current cpctl binary with the given args.
// Falls back to "go run ./cli/cpctl" when the binary is not installed.
func captureSelf(root string, args ...string) (string, error) {
	self, err := osexec.LookPath("cpctl")
	if err != nil {
		// dev fallback: run via go run from repo root
		goArgs := append([]string{"run", "./cli/cpctl"}, args...)
		cmd := osexec.Command("go", goArgs...)
		cmd.Dir = root
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		return strings.TrimSpace(buf.String()), cmd.Run()
	}
	return capture(self, args...)
}

func step(log *strings.Builder, msg string) {
	log.WriteString("→ " + msg + "\n")
}

func result(log *strings.Builder, errMsg string) *mcp.CallToolResult {
	log.WriteString("\nERROR: " + errMsg + "\n")
	return mcp.NewToolResultText(log.String())
}

func waitLocalStack(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:4566/_localstack/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("localstack not ready after %s", timeout)
}

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

func formatStatus(st tui.PlaygroundStatus) string {
	var b strings.Builder
	b.WriteString("=== birdy-playground status ===\n\n")

	// Kind
	k := st.Kind
	if k.Running {
		b.WriteString(fmt.Sprintf("Kind cluster  ● %s  (%d/%d nodes ready)\n", k.Name, k.Ready, k.Nodes))
	} else {
		b.WriteString(fmt.Sprintf("Kind cluster  ✗ not running  (%s)\n", k.Name))
	}

	// LocalStack
	ls := st.LocalStack
	if ls.Running {
		b.WriteString("LocalStack    ● http://localhost:4566\n")
		if len(ls.Services) > 0 {
			b.WriteString("\nServices:\n")
			for name, status := range ls.Services {
				icon := "✓"
				if status != "running" && status != "available" {
					icon = "✗"
				}
				b.WriteString(fmt.Sprintf("  %s %-14s %s\n", icon, name, status))
			}
		}
	} else {
		b.WriteString("LocalStack    ✗ not running\n")
	}

	// Terraform
	tf := st.Terraform
	if tf.Applied {
		b.WriteString(fmt.Sprintf("Terraform     ● %d resources applied\n", tf.Resources))
	} else {
		b.WriteString("Terraform     ✗ no state\n")
	}

	b.WriteString(fmt.Sprintf("\nChecked at: %s\n", st.CheckedAt.Format(time.RFC3339)))
	return b.String()
}

func mcpProfileFromEnv() string {
	profile := strings.TrimSpace(strings.ToLower(os.Getenv("CPCTL_MCP_PROFILE")))
	if profile == "" {
		return "read-only"
	}
	return profile
}

func profileAllowsMutating(profile string) bool {
	switch profile {
	case "operator", "mutating", "full":
		return true
	default:
		return false
	}
}
