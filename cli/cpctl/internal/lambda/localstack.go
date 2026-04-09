package lambda

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// execCommandContext is a variable that can be mocked in tests
var execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, arg...)
}

// LocalStackClient implements Client for LocalStack
type LocalStackClient struct {
	endpoint string
	region   string
}

// NewLocalStackClient creates a LocalStack Lambda client
func NewLocalStackClient() (*LocalStackClient, error) {
	endpoint := os.Getenv("LOCALSTACK_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4566"
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-central-1"
	}

	return &LocalStackClient{
		endpoint: endpoint,
		region:   region,
	}, nil
}

// Deploy creates or updates a function in LocalStack
func (c *LocalStackClient) Deploy(ctx context.Context, fn *Function) (*DeployResult, error) {
	slog.Info("deploying lambda function to localstack", "name", fn.Name)

	// Create or update function using aws-cli against LocalStack
	cmd := execCommandContext(ctx, "aws", "lambda", "create-function",
		"--function-name", fn.Name,
		"--runtime", fn.Runtime,
		"--role", fn.Role,
		"--handler", fn.Handler,
		"--zip-file", fmt.Sprintf("fileb://%s", fn.CodePath),
		"--timeout", fmt.Sprintf("%d", fn.Timeout),
		"--memory-size", fmt.Sprintf("%d", fn.Memory),
		"--endpoint-url", c.endpoint,
		"--region", c.region,
		"--output", "json",
	)

	// If function exists, update it instead
	checkCmd := execCommandContext(ctx, "aws", "lambda", "get-function",
		"--function-name", fn.Name,
		"--endpoint-url", c.endpoint,
		"--region", c.region,
	)
	if err := checkCmd.Run(); err == nil {
		// Function exists, update code
		cmd = execCommandContext(ctx, "aws", "lambda", "update-function-code",
			"--function-name", fn.Name,
			"--zip-file", fmt.Sprintf("fileb://%s", fn.CodePath),
			"--endpoint-url", c.endpoint,
			"--region", c.region,
			"--output", "json",
		)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aws lambda create-function failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse aws response: %w", err)
	}

	slog.Info("lambda function deployed", "name", fn.Name, "arn", result["FunctionArn"])

	return &DeployResult{
		FunctionArn:  fmt.Sprintf("%v", result["FunctionArn"]),
		FunctionName: fn.Name,
		Version:      fmt.Sprintf("%v", result["Version"]),
		CodeSha256:   fmt.Sprintf("%v", result["CodeSha256"]),
		LastModified: fmt.Sprintf("%v", result["LastModified"]),
	}, nil
}

// GetFunction retrieves function metadata
func (c *LocalStackClient) GetFunction(ctx context.Context, name string) (*Function, error) {
	cmd := execCommandContext(ctx, "aws", "lambda", "get-function",
		"--function-name", name,
		"--endpoint-url", c.endpoint,
		"--region", c.region,
		"--output", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aws lambda get-function failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse aws response: %w", err)
	}

	config := result["Configuration"].(map[string]interface{})

	fn := &Function{
		Name:     fmt.Sprintf("%v", config["FunctionName"]),
		Runtime:  fmt.Sprintf("%v", config["Runtime"]),
		Handler:  fmt.Sprintf("%v", config["Handler"]),
		Role:     fmt.Sprintf("%v", config["Role"]),
		Timeout:  int(config["Timeout"].(float64)),
		Memory:   int(config["MemorySize"].(float64)),
	}

	return fn, nil
}

// ListFunctions lists all functions in LocalStack
func (c *LocalStackClient) ListFunctions(ctx context.Context) ([]*Function, error) {
	cmd := execCommandContext(ctx, "aws", "lambda", "list-functions",
		"--endpoint-url", c.endpoint,
		"--region", c.region,
		"--output", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aws lambda list-functions failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse aws response: %w", err)
	}

	functions := result["Functions"].([]interface{})
	var fns []*Function

	for _, f := range functions {
		fn := f.(map[string]interface{})
		fns = append(fns, &Function{
			Name:    fmt.Sprintf("%v", fn["FunctionName"]),
			Runtime: fmt.Sprintf("%v", fn["Runtime"]),
			Handler: fmt.Sprintf("%v", fn["Handler"]),
		})
	}

	return fns, nil
}

// DeleteFunction removes a function from LocalStack
func (c *LocalStackClient) DeleteFunction(ctx context.Context, name string) error {
	cmd := execCommandContext(ctx, "aws", "lambda", "delete-function",
		"--function-name", name,
		"--endpoint-url", c.endpoint,
		"--region", c.region,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws lambda delete-function failed: %w", err)
	}

	slog.Info("lambda function deleted", "name", name)
	return nil
}

// Invoke executes a function in LocalStack
func (c *LocalStackClient) Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResult, error) {
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("lambda-payload-%d.json", time.Now().UnixNano()))
	if err := os.WriteFile(tempFile, req.Payload, 0644); err != nil {
		return nil, fmt.Errorf("failed to write payload: %w", err)
	}
	defer os.Remove(tempFile)

	invokeType := "RequestResponse"
	if req.Async {
		invokeType = "Event"
	}

	cmd := execCommandContext(ctx, "aws", "lambda", "invoke",
		"--function-name", req.FunctionName,
		"--invocation-type", invokeType,
		"--payload", fmt.Sprintf("fileb://%s", tempFile),
		"--endpoint-url", c.endpoint,
		"--region", c.region,
		filepath.Join(os.TempDir(), "lambda-response.json"),
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aws lambda invoke failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse invoke response: %w", err)
	}

	// Read response payload
	respFile := filepath.Join(os.TempDir(), "lambda-response.json")
	payload, _ := os.ReadFile(respFile)
	os.Remove(respFile)

	return &InvokeResult{
		StatusCode:      int(result["StatusCode"].(float64)),
		FunctionErr:     fmt.Sprintf("%v", result["FunctionError"]),
		ExecutedVersion: fmt.Sprintf("%v", result["ExecutedVersion"]),
		Payload:         payload,
	}, nil
}

// GetLogs retrieves function logs from CloudWatch (LocalStack)
func (c *LocalStackClient) GetLogs(ctx context.Context, functionName string, limit int) ([]LogEntry, error) {
	logGroupName := fmt.Sprintf("/aws/lambda/%s", functionName)

	cmd := execCommandContext(ctx, "aws", "logs", "tail",
		logGroupName,
		"--max-items", fmt.Sprintf("%d", limit),
		"--endpoint-url", c.endpoint,
		"--region", c.region,
		"--output", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		// Return empty list if no logs found
		return []LogEntry{}, nil
	}

	var events []struct {
		Timestamp int64  `json:"timestamp"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal(output, &events); err != nil {
		return []LogEntry{}, nil
	}

	var logs []LogEntry
	for _, e := range events {
		logs = append(logs, LogEntry{
			Timestamp: e.Timestamp,
			Message:   e.Message,
		})
	}

	return logs, nil
}

// TailLogs streams function logs in real-time
func (c *LocalStackClient) TailLogs(ctx context.Context, functionName string, logChan chan LogEntry) error {
	logGroupName := fmt.Sprintf("/aws/lambda/%s", functionName)

	cmd := execCommandContext(ctx, "aws", "logs", "tail",
		logGroupName,
		"--follow",
		"--endpoint-url", c.endpoint,
		"--region", c.region,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Stream logs line by line
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		logChan <- LogEntry{
			Timestamp: time.Now().Unix(),
			Message:   line,
		}
	}

	return cmd.Wait()
}

// UpdateCode updates function code
func (c *LocalStackClient) UpdateCode(ctx context.Context, functionName, codePath string) error {
	cmd := execCommandContext(ctx, "aws", "lambda", "update-function-code",
		"--function-name", functionName,
		"--zip-file", fmt.Sprintf("fileb://%s", codePath),
		"--endpoint-url", c.endpoint,
		"--region", c.region,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws lambda update-function-code failed: %w", err)
	}

	slog.Info("lambda function code updated", "name", functionName)
	return nil
}

// UpdateConfig updates function configuration
func (c *LocalStackClient) UpdateConfig(ctx context.Context, fn *Function) error {
	cmd := execCommandContext(ctx, "aws", "lambda", "update-function-configuration",
		"--function-name", fn.Name,
		"--runtime", fn.Runtime,
		"--handler", fn.Handler,
		"--timeout", fmt.Sprintf("%d", fn.Timeout),
		"--memory-size", fmt.Sprintf("%d", fn.Memory),
		"--endpoint-url", c.endpoint,
		"--region", c.region,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws lambda update-function-configuration failed: %w", err)
	}

	slog.Info("lambda function config updated", "name", fn.Name)
	return nil
}
