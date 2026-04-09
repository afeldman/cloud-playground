package cmd

import (
	"bytes"
	"context"
	"testing"
	"time"

	"cpctl/internal/batch"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBatchClient is a mock for testing CLI commands
type mockBatchClient struct {
	registerJobDefinitionFunc func(ctx context.Context, def *batch.JobDefinition) (string, error)
	submitJobFunc            func(ctx context.Context, req *batch.SubmitJobRequest) (*batch.SubmitJobResult, error)
	describeJobFunc          func(ctx context.Context, jobID string) (*batch.Job, error)
	listJobsFunc             func(ctx context.Context, queueName string, status string) ([]*batch.Job, error)
	listJobDefinitionsFunc   func(ctx context.Context) ([]*batch.JobDefinition, error)
	deregisterJobDefinitionFunc func(ctx context.Context, name string) error
	createJobQueueFunc       func(ctx context.Context, queue *batch.JobQueue) error
	listJobQueuesFunc        func(ctx context.Context) ([]*batch.JobQueue, error)
	terminateJobFunc         func(ctx context.Context, jobID string, reason string) error
	getLogsFunc              func(ctx context.Context, jobID string) ([]string, error)
	tailLogsFunc             func(ctx context.Context, jobID string, logChan chan string) error
}

func (m *mockBatchClient) RegisterJobDefinition(ctx context.Context, def *batch.JobDefinition) (string, error) {
	return m.registerJobDefinitionFunc(ctx, def)
}

func (m *mockBatchClient) SubmitJob(ctx context.Context, req *batch.SubmitJobRequest) (*batch.SubmitJobResult, error) {
	return m.submitJobFunc(ctx, req)
}

func (m *mockBatchClient) DescribeJob(ctx context.Context, jobID string) (*batch.Job, error) {
	return m.describeJobFunc(ctx, jobID)
}

func (m *mockBatchClient) ListJobs(ctx context.Context, queueName string, status string) ([]*batch.Job, error) {
	return m.listJobsFunc(ctx, queueName, status)
}

func (m *mockBatchClient) ListJobDefinitions(ctx context.Context) ([]*batch.JobDefinition, error) {
	return m.listJobDefinitionsFunc(ctx)
}

func (m *mockBatchClient) DeregisterJobDefinition(ctx context.Context, name string) error {
	return m.deregisterJobDefinitionFunc(ctx, name)
}

func (m *mockBatchClient) CreateJobQueue(ctx context.Context, queue *batch.JobQueue) error {
	return m.createJobQueueFunc(ctx, queue)
}

func (m *mockBatchClient) ListJobQueues(ctx context.Context) ([]*batch.JobQueue, error) {
	return m.listJobQueuesFunc(ctx)
}

func (m *mockBatchClient) TerminateJob(ctx context.Context, jobID string, reason string) error {
	return m.terminateJobFunc(ctx, jobID, reason)
}

func (m *mockBatchClient) GetLogs(ctx context.Context, jobID string) ([]string, error) {
	return m.getLogsFunc(ctx, jobID)
}

func (m *mockBatchClient) TailLogs(ctx context.Context, jobID string, logChan chan string) error {
	return m.tailLogsFunc(ctx, jobID, logChan)
}

func resetBatchCommandFlags() {
	for _, command := range []*cobra.Command{batchRegisterCmd, batchSubmitCmd, batchWatchCmd, batchLogsCmd, batchListCmd} {
		command.Flags().VisitAll(func(f *pflag.Flag) {
			f.Changed = false
		})
	}
	batchCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
	})
}

// TestBatchRegisterCommand tests the batch register command
func TestBatchRegisterCommand(t *testing.T) {
	resetBatchCommandFlags()

	// Mock client creation
	originalNewClient := batchNewClient
	defer func() { batchNewClient = originalNewClient }()

	var capturedDefinition *batch.JobDefinition

	batchNewClient = func(stage string) (batch.Client, error) {
		return &mockBatchClient{
			registerJobDefinitionFunc: func(ctx context.Context, def *batch.JobDefinition) (string, error) {
				capturedDefinition = def
				return "arn:aws:batch:us-east-1:123456789012:job-definition/test-job:1", nil
			},
		}, nil
	}

	// Create command and execute
	cmd := batchRegisterCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Set flags
	cmd.Flags().Set("image", "alpine:latest")
	cmd.Flags().Set("vcpus", "1")
	cmd.Flags().Set("memory", "512")
	cmd.Flags().Set("command", "echo,hello,world")
	cmd.Flags().Set("stage", "localstack")

	// Execute command
	err := cmd.RunE(cmd, []string{"test-job"})
	require.NoError(t, err)

	// Verify definition parameters
	require.NotNil(t, capturedDefinition)
	assert.Equal(t, "test-job", capturedDefinition.Name)
	assert.Equal(t, "alpine:latest", capturedDefinition.Image)
	assert.Equal(t, 1, capturedDefinition.Vcpus)
	assert.Equal(t, 512, capturedDefinition.Memory)
	assert.Equal(t, []string{"echo", "hello", "world"}, capturedDefinition.Command)

	// Verify output contains success message
	outputStr := output.String()
	assert.Contains(t, outputStr, "Job definition registered successfully")
	assert.Contains(t, outputStr, "test-job")
}

// TestBatchSubmitCommand tests the batch submit command
func TestBatchSubmitCommand(t *testing.T) {
	resetBatchCommandFlags()

	// Mock client creation
	originalNewClient := batchNewClient
	defer func() { batchNewClient = originalNewClient }()

	var capturedRequest *batch.SubmitJobRequest

	batchNewClient = func(stage string) (batch.Client, error) {
		return &mockBatchClient{
			submitJobFunc: func(ctx context.Context, req *batch.SubmitJobRequest) (*batch.SubmitJobResult, error) {
				capturedRequest = req
				return &batch.SubmitJobResult{
					JobID:  "test-job-id",
					JobArn: "arn:aws:batch:us-east-1:123456789012:job/test-job-id",
				}, nil
			},
		}, nil
	}

	// Create command and execute
	cmd := batchSubmitCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Set flags
	cmd.Flags().Set("command", "python,main.py,--arg,value")
	cmd.Flags().Set("stage", "mirror")

	// Execute command
	err := cmd.RunE(cmd, []string{"test-job-instance", "test-job-definition", "test-queue"})
	require.NoError(t, err)

	// Verify request parameters
	require.NotNil(t, capturedRequest)
	assert.Equal(t, "test-job-instance", capturedRequest.JobName)
	assert.Equal(t, "test-job-definition", capturedRequest.JobDefinition)
	assert.Equal(t, "test-queue", capturedRequest.JobQueue)
	require.NotNil(t, capturedRequest.ContainerOverrides)
	assert.Equal(t, []string{"python", "main.py", "--arg", "value"}, capturedRequest.ContainerOverrides.Command)

	// Verify output contains success message
	outputStr := output.String()
	assert.Contains(t, outputStr, "Job submitted successfully")
	assert.Contains(t, outputStr, "test-job-id")
}

// TestBatchSubmitCommandNoOverrides tests the batch submit command without command override
func TestBatchSubmitCommandNoOverrides(t *testing.T) {
	resetBatchCommandFlags()

	// Mock client creation
	originalNewClient := batchNewClient
	defer func() { batchNewClient = originalNewClient }()

	var capturedRequest *batch.SubmitJobRequest

	batchNewClient = func(stage string) (batch.Client, error) {
		return &mockBatchClient{
			submitJobFunc: func(ctx context.Context, req *batch.SubmitJobRequest) (*batch.SubmitJobResult, error) {
				capturedRequest = req
				return &batch.SubmitJobResult{
					JobID:  "test-job-id",
					JobArn: "arn:aws:batch:us-east-1:123456789012:job/test-job-id",
				}, nil
			},
		}, nil
	}

	// Create command and execute
	cmd := batchSubmitCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Execute command without command flag
	err := cmd.RunE(cmd, []string{"test-job-instance", "test-job-definition", "test-queue"})
	require.NoError(t, err)

	// Verify request parameters
	require.NotNil(t, capturedRequest)
	assert.Equal(t, "test-job-instance", capturedRequest.JobName)
	assert.Equal(t, "test-job-definition", capturedRequest.JobDefinition)
	assert.Equal(t, "test-queue", capturedRequest.JobQueue)
	require.NotNil(t, capturedRequest.ContainerOverrides)
	assert.Empty(t, capturedRequest.ContainerOverrides.Command)
}

// TestBatchWatchCommand tests the batch watch command
func TestBatchWatchCommand(t *testing.T) {
	resetBatchCommandFlags()

	// Mock client creation
	originalNewClient := batchNewClient
	defer func() { batchNewClient = originalNewClient }()

	var capturedJobID string
	callCount := 0

	batchNewClient = func(stage string) (batch.Client, error) {
		return &mockBatchClient{
			describeJobFunc: func(ctx context.Context, jobID string) (*batch.Job, error) {
				capturedJobID = jobID
				callCount++
				
				// Simulate job progressing through states
				status := "PENDING"
				if callCount > 1 {
					status = "RUNNING"
				}
				if callCount > 2 {
					status = "SUCCEEDED"
				}
				
				return &batch.Job{
					JobID:   jobID,
					JobName: "test-job",
					Status:  status,
				}, nil
			},
		}, nil
	}

	// Create command and execute with short interval
	cmd := batchWatchCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Set flags
	cmd.Flags().Set("interval", "100ms")
	cmd.Flags().Set("stage", "localstack")

	// We need to run this in a goroutine and cancel after a short time
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	
	cmd.SetContext(ctx)
	
	// Execute command
	err := cmd.RunE(cmd, []string{"test-job-id"})
	// Command will be cancelled by timeout, which is expected
	if err != nil && err != context.DeadlineExceeded {
		require.NoError(t, err)
	}

	// Verify job ID was captured
	assert.Equal(t, "test-job-id", capturedJobID)
	
	// Verify describe was called multiple times (polling)
	assert.Greater(t, callCount, 1)
}

// TestBatchLogsCommand tests the batch logs command
func TestBatchLogsCommand(t *testing.T) {
	resetBatchCommandFlags()

	// Mock client creation
	originalNewClient := batchNewClient
	defer func() { batchNewClient = originalNewClient }()

	var capturedJobID string

	batchNewClient = func(stage string) (batch.Client, error) {
		return &mockBatchClient{
			getLogsFunc: func(ctx context.Context, jobID string) ([]string, error) {
				capturedJobID = jobID
				return []string{
					"2024-01-01T00:00:00Z START RequestId: abc-123",
					"2024-01-01T00:00:01Z Processing event",
					"2024-01-01T00:00:02Z END RequestId: abc-123",
				}, nil
			},
		}, nil
	}

	// Create command and execute
	cmd := batchLogsCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Set flags
	cmd.Flags().Set("stage", "mirror")

	// Execute command
	err := cmd.RunE(cmd, []string{"test-job-id"})
	require.NoError(t, err)

	// Verify job ID
	assert.Equal(t, "test-job-id", capturedJobID)

	// Verify output contains logs
	outputStr := output.String()
	assert.Contains(t, outputStr, "START RequestId: abc-123")
	assert.Contains(t, outputStr, "Processing event")
	assert.Contains(t, outputStr, "END RequestId: abc-123")
}

// TestBatchListCommand tests the batch list command
func TestBatchListCommand(t *testing.T) {
	resetBatchCommandFlags()

	// Mock client creation
	originalNewClient := batchNewClient
	defer func() { batchNewClient = originalNewClient }()

	var capturedQueueName string
	var capturedStatus string

	batchNewClient = func(stage string) (batch.Client, error) {
		return &mockBatchClient{
			listJobsFunc: func(ctx context.Context, queueName string, status string) ([]*batch.Job, error) {
				capturedQueueName = queueName
				capturedStatus = status
				return []*batch.Job{
					{
						JobID:   "job-1",
						JobName: "job-one",
						Status:  "RUNNING",
					},
					{
						JobID:   "job-2",
						JobName: "job-two",
						Status:  "SUCCEEDED",
					},
				}, nil
			},
		}, nil
	}

	// Create command and execute
	cmd := batchListCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Set flags
	cmd.Flags().Set("queue", "test-queue")
	cmd.Flags().Set("status", "RUNNING")
	cmd.Flags().Set("stage", "localstack")

	// Execute command
	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Verify parameters
	assert.Equal(t, "test-queue", capturedQueueName)
	assert.Equal(t, "RUNNING", capturedStatus)

	// Verify output contains job list
	outputStr := output.String()
	assert.Contains(t, outputStr, "job-1")
	assert.Contains(t, outputStr, "job-one")
	assert.Contains(t, outputStr, "RUNNING")
	assert.Contains(t, outputStr, "job-2")
	assert.Contains(t, outputStr, "job-two")
}
