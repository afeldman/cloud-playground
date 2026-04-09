package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cpctl/internal/lambda"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLambdaClient is a mock for testing CLI commands
type mockLambdaClient struct {
	deployFunc      func(ctx context.Context, fn *lambda.Function) (*lambda.DeployResult, error)
	invokeFunc      func(ctx context.Context, req *lambda.InvokeRequest) (*lambda.InvokeResult, error)
	getLogsFunc     func(ctx context.Context, functionName string, limit int) ([]lambda.LogEntry, error)
	listFunctionsFunc func(ctx context.Context) ([]*lambda.Function, error)
	getFunctionFunc func(ctx context.Context, name string) (*lambda.Function, error)
	deleteFunctionFunc func(ctx context.Context, name string) error
}

func (m *mockLambdaClient) Deploy(ctx context.Context, fn *lambda.Function) (*lambda.DeployResult, error) {
	return m.deployFunc(ctx, fn)
}

func (m *mockLambdaClient) Invoke(ctx context.Context, req *lambda.InvokeRequest) (*lambda.InvokeResult, error) {
	return m.invokeFunc(ctx, req)
}

func (m *mockLambdaClient) GetLogs(ctx context.Context, functionName string, limit int) ([]lambda.LogEntry, error) {
	return m.getLogsFunc(ctx, functionName, limit)
}

func (m *mockLambdaClient) TailLogs(ctx context.Context, functionName string, logChan chan lambda.LogEntry) error {
	return nil
}

func (m *mockLambdaClient) ListFunctions(ctx context.Context) ([]*lambda.Function, error) {
	return m.listFunctionsFunc(ctx)
}

func (m *mockLambdaClient) GetFunction(ctx context.Context, name string) (*lambda.Function, error) {
	return m.getFunctionFunc(ctx, name)
}

func (m *mockLambdaClient) DeleteFunction(ctx context.Context, name string) error {
	return m.deleteFunctionFunc(ctx, name)
}

func (m *mockLambdaClient) UpdateCode(ctx context.Context, functionName, codePath string) error {
	return nil
}

func (m *mockLambdaClient) UpdateConfig(ctx context.Context, fn *lambda.Function) error {
	return nil
}

func resetLambdaCommandFlags() {
	for _, command := range []*cobra.Command{lambdaDeployCmd, lambdaInvokeCmd, lambdaLogsCmd} {
		command.Flags().VisitAll(func(f *pflag.Flag) {
			f.Changed = false
		})
	}
}

// TestLambdaDeployCommand tests the lambda deploy command
func TestLambdaDeployCommand(t *testing.T) {
	resetLambdaCommandFlags()

	// Create a temporary test ZIP file
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")
	err := os.WriteFile(zipPath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Mock client creation
	originalNewClient := lambdaNewClient
	defer func() { lambdaNewClient = originalNewClient }()

	var capturedStage string
	var capturedFunction *lambda.Function

	lambdaNewClient = func(stage string) (lambda.Client, error) {
		capturedStage = stage
		return &mockLambdaClient{
			deployFunc: func(ctx context.Context, fn *lambda.Function) (*lambda.DeployResult, error) {
				capturedFunction = fn
				return &lambda.DeployResult{
					FunctionArn:  "arn:aws:lambda:us-east-1:123456789012:function:test-function",
					FunctionName: "test-function",
					Version:      "$LATEST",
					CodeSha256:   "abc123",
					LastModified: time.Now().Format(time.RFC3339),
				}, nil
			},
		}, nil
	}

	// Create command and execute
	cmd := lambdaDeployCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Set flags
	cmd.Flags().Set("name", "test-function")
	cmd.Flags().Set("runtime", "python3.9")
	cmd.Flags().Set("handler", "index.handler")
	cmd.Flags().Set("role", "arn:aws:iam::123456789012:role/lambda-role")
	cmd.Flags().Set("timeout", "30")
	cmd.Flags().Set("memory", "128")
	cmd.Flags().Set("stage", "localstack")

	// Execute command
	err = cmd.RunE(cmd, []string{zipPath})
	require.NoError(t, err)

	// Verify stage was captured
	assert.Equal(t, "localstack", capturedStage)

	// Verify function parameters
	require.NotNil(t, capturedFunction)
	assert.Equal(t, "test-function", capturedFunction.Name)
	assert.Equal(t, "python3.9", capturedFunction.Runtime)
	assert.Equal(t, "index.handler", capturedFunction.Handler)
	assert.Equal(t, "arn:aws:iam::123456789012:role/lambda-role", capturedFunction.Role)
	assert.Equal(t, zipPath, capturedFunction.CodePath)
	assert.Equal(t, 30, capturedFunction.Timeout)
	assert.Equal(t, 128, capturedFunction.Memory)

	// Verify output contains success message
	outputStr := output.String()
	assert.Contains(t, outputStr, "Lambda function deployed successfully")
}

// TestLambdaDeployCommandMissingFile tests error handling for missing file
func TestLambdaDeployCommandMissingFile(t *testing.T) {
	resetLambdaCommandFlags()

	cmd := lambdaDeployCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Set flags
	cmd.Flags().Set("name", "test-function")
	cmd.Flags().Set("runtime", "python3.9")
	cmd.Flags().Set("handler", "index.handler")
	cmd.Flags().Set("role", "arn:aws:iam::123456789012:role/lambda-role")

	// Execute command with non-existent file
	err := cmd.RunE(cmd, []string{"/non/existent/file.zip"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read ZIP file")
}

// TestLambdaInvokeCommand tests the lambda invoke command
func TestLambdaInvokeCommand(t *testing.T) {
	resetLambdaCommandFlags()

	// Mock client creation
	originalNewClient := lambdaNewClient
	defer func() { lambdaNewClient = originalNewClient }()

	var capturedRequest *lambda.InvokeRequest

	lambdaNewClient = func(stage string) (lambda.Client, error) {
		return &mockLambdaClient{
			invokeFunc: func(ctx context.Context, req *lambda.InvokeRequest) (*lambda.InvokeResult, error) {
				capturedRequest = req
				return &lambda.InvokeResult{
					StatusCode:   200,
					Payload:      []byte(`{"result": "success"}`),
					ExecutedVersion: "$LATEST",
					LogResult:    "LOG RESULT",
				}, nil
			},
		}, nil
	}

	// Create command and execute
	cmd := lambdaInvokeCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Set flags with inline JSON payload
	cmd.Flags().Set("payload", `{"key": "value"}`)
	cmd.Flags().Set("stage", "mirror")

	// Execute command
	err := cmd.RunE(cmd, []string{"test-function"})
	require.NoError(t, err)

	// Verify request parameters
	require.NotNil(t, capturedRequest)
	assert.Equal(t, "test-function", capturedRequest.FunctionName)
	assert.Equal(t, []byte(`{"key": "value"}`), capturedRequest.Payload)
	assert.False(t, capturedRequest.Async)

	// Verify output contains result
	outputStr := output.String()
	assert.Contains(t, outputStr, "Status Code:   200")
	assert.Contains(t, outputStr, `"result": "success"`)
}

// TestLambdaInvokeCommandAsync tests the lambda invoke command with async flag
func TestLambdaInvokeCommandAsync(t *testing.T) {
	resetLambdaCommandFlags()

	// Mock client creation
	originalNewClient := lambdaNewClient
	defer func() { lambdaNewClient = originalNewClient }()

	var capturedRequest *lambda.InvokeRequest

	lambdaNewClient = func(stage string) (lambda.Client, error) {
		return &mockLambdaClient{
			invokeFunc: func(ctx context.Context, req *lambda.InvokeRequest) (*lambda.InvokeResult, error) {
				capturedRequest = req
				return &lambda.InvokeResult{
					StatusCode: 202,
				}, nil
			},
		}, nil
	}

	// Create command and execute
	cmd := lambdaInvokeCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Set flags
	cmd.Flags().Set("payload", `{}`)
	cmd.Flags().Set("async", "true")
	cmd.Flags().Set("stage", "localstack")

	// Execute command
	err := cmd.RunE(cmd, []string{"test-function"})
	require.NoError(t, err)

	// Verify async flag
	require.NotNil(t, capturedRequest)
	assert.True(t, capturedRequest.Async)
}

// TestLambdaInvokeCommandFilePayload tests the lambda invoke command with file payload
func TestLambdaInvokeCommandFilePayload(t *testing.T) {
	resetLambdaCommandFlags()

	// Create a temporary test file
	tempDir := t.TempDir()
	payloadFile := filepath.Join(tempDir, "payload.json")
	payloadContent := `{"file": "content"}`
	err := os.WriteFile(payloadFile, []byte(payloadContent), 0644)
	require.NoError(t, err)

	// Mock client creation
	originalNewClient := lambdaNewClient
	defer func() { lambdaNewClient = originalNewClient }()

	var capturedRequest *lambda.InvokeRequest

	lambdaNewClient = func(stage string) (lambda.Client, error) {
		return &mockLambdaClient{
			invokeFunc: func(ctx context.Context, req *lambda.InvokeRequest) (*lambda.InvokeResult, error) {
				capturedRequest = req
				return &lambda.InvokeResult{
					StatusCode: 200,
					Payload:    []byte(`{"result": "from file"}`),
				}, nil
			},
		}, nil
	}

	// Create command and execute
	cmd := lambdaInvokeCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Set flags with file payload
	cmd.Flags().Set("payload", "@"+payloadFile)

	// Execute command
	err = cmd.RunE(cmd, []string{"test-function"})
	require.NoError(t, err)

	// Verify file content was read
	require.NotNil(t, capturedRequest)
	assert.Equal(t, []byte(payloadContent), capturedRequest.Payload)
}

// TestLambdaLogsCommand tests the lambda logs command
func TestLambdaLogsCommand(t *testing.T) {
	resetLambdaCommandFlags()

	// Mock client creation
	originalNewClient := lambdaNewClient
	defer func() { lambdaNewClient = originalNewClient }()

	var capturedFunctionName string
	var capturedLimit int

	lambdaNewClient = func(stage string) (lambda.Client, error) {
		return &mockLambdaClient{
			getLogsFunc: func(ctx context.Context, functionName string, limit int) ([]lambda.LogEntry, error) {
				capturedFunctionName = functionName
				capturedLimit = limit
				return []lambda.LogEntry{
					{
						Timestamp: time.Now().Unix(),
						Message:   "START RequestId: abc-123",
						Level:     "INFO",
					},
					{
						Timestamp: time.Now().Unix() + 1,
						Message:   "END RequestId: abc-123",
						Level:     "INFO",
					},
				}, nil
			},
		}, nil
	}

	// Create command and execute
	cmd := lambdaLogsCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Set flags
	cmd.Flags().Set("lines", "50")
	cmd.Flags().Set("stage", "mirror")

	// Execute command
	err := cmd.RunE(cmd, []string{"test-function"})
	require.NoError(t, err)

	// Verify parameters
	assert.Equal(t, "test-function", capturedFunctionName)
	assert.Equal(t, 50, capturedLimit)

	// Verify output contains logs
	outputStr := output.String()
	assert.Contains(t, outputStr, "START RequestId: abc-123")
	assert.Contains(t, outputStr, "END RequestId: abc-123")
}

// TestLambdaLogsCommandDefaultLimit tests the lambda logs command with default limit
func TestLambdaLogsCommandDefaultLimit(t *testing.T) {
	resetLambdaCommandFlags()

	// Mock client creation
	originalNewClient := lambdaNewClient
	defer func() { lambdaNewClient = originalNewClient }()

	var capturedLimit int

	lambdaNewClient = func(stage string) (lambda.Client, error) {
		return &mockLambdaClient{
			getLogsFunc: func(ctx context.Context, functionName string, limit int) ([]lambda.LogEntry, error) {
				capturedLimit = limit
				return []lambda.LogEntry{}, nil
			},
		}, nil
	}

	// Create command and execute
	cmd := lambdaLogsCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Execute command without setting lines flag
	err := cmd.RunE(cmd, []string{"test-function"})
	require.NoError(t, err)

	// Verify default limit
	assert.Equal(t, 50, capturedLimit)
}