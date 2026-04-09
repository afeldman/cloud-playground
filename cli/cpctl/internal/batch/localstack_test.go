package batch

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCmd is a test helper that simulates exec.CommandContext
type mockCmd struct {
	*exec.Cmd
	output []byte
	err    error
}

// mockExec creates a simple mock for execCommandContext
func mockExec(outputs map[string][]byte, errors map[string]error) func(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cmdKey := name + " " + strings.Join(arg, " ")
		
		output := outputs[cmdKey]
		if len(output) == 0 {
			for key, candidate := range outputs {
				if strings.Contains(cmdKey, key) {
					output = candidate
					break
				}
			}
		}
		_ = errors[cmdKey]
		
		// Create a command that will return our mocked output
		// We use echo to output the mock response
		echoCmd := exec.CommandContext(ctx, "echo", string(output))
		return echoCmd
	}
}

func TestRegisterJobDefinition(t *testing.T) {
	ctx := context.Background()
	
	// Mock response for register-job-definition
	registerResponse := map[string]interface{}{
		"jobDefinitionArn": "arn:aws:batch:us-east-1:000000000000:job-definition/test-job-def:1",
		"jobDefinitionName": "test-job-def",
		"revision": 1,
	}
	
	registerJSON, err := json.Marshal(registerResponse)
	require.NoError(t, err)
	
	// Save original exec function and restore after test
	originalExec := execCommandContext
	defer func() { execCommandContext = originalExec }()
	
	execCommandContext = mockExec(
		map[string][]byte{
			"register-job-definition": registerJSON,
		},
		map[string]error{},
	)
	
	client, err := NewLocalStackClient()
	require.NoError(t, err)
	
	def := &JobDefinition{
		Name:    "test-job-def",
		Type:    "container",
		Image:   "alpine:latest",
		Vcpus:   1,
		Memory:  512,
		Command: []string{"echo", "hello"},
	}
	
	arn, err := client.RegisterJobDefinition(ctx, def)
	require.NoError(t, err)
	assert.Equal(t, "arn:aws:batch:us-east-1:000000000000:job-definition/test-job-def:1", arn)
}

func TestSubmitJob(t *testing.T) {
	ctx := context.Background()
	
	// Mock response for submit-job
	submitResponse := map[string]interface{}{
		"jobId":  "test-job-id",
		"jobArn": "arn:aws:batch:us-east-1:000000000000:job/test-job-id",
	}
	
	submitJSON, err := json.Marshal(submitResponse)
	require.NoError(t, err)
	
	// Save original exec function and restore after test
	originalExec := execCommandContext
	defer func() { execCommandContext = originalExec }()
	
	execCommandContext = mockExec(
		map[string][]byte{
			"submit-job": submitJSON,
		},
		map[string]error{},
	)
	
	client, err := NewLocalStackClient()
	require.NoError(t, err)
	
	req := &SubmitJobRequest{
		JobName:       "test-job",
		JobDefinition: "test-job-def:1",
		JobQueue:      "test-queue",
	}
	
	result, err := client.SubmitJob(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "test-job-id", result.JobID)
	assert.Equal(t, "arn:aws:batch:us-east-1:000000000000:job/test-job-id", result.JobArn)
}

func TestDescribeJob(t *testing.T) {
	ctx := context.Background()
	
	now := time.Now()
	// Mock response for describe-jobs
	describeResponse := map[string]interface{}{
		"jobs": []map[string]interface{}{
			{
				"jobId":         "test-job-id",
				"jobName":       "test-job",
				"status":        "SUCCEEDED",
				"jobDefinition": "test-def:1",
				"createdAt":     now.UnixMilli(),
				"startedAt":     now.Add(5 * time.Second).UnixMilli(),
				"stoppedAt":     now.Add(10 * time.Second).UnixMilli(),
				"container": map[string]interface{}{
					"exitCode":       0,
					"logStreamName": "log-stream",
					"reason":        "Completed",
				},
			},
		},
	}
	
	describeJSON, err := json.Marshal(describeResponse)
	require.NoError(t, err)
	
	// Save original exec function and restore after test
	originalExec := execCommandContext
	defer func() { execCommandContext = originalExec }()
	
	execCommandContext = mockExec(
		map[string][]byte{
			"describe-jobs": describeJSON,
		},
		map[string]error{},
	)
	
	client, err := NewLocalStackClient()
	require.NoError(t, err)
	
	job, err := client.DescribeJob(ctx, "test-job-id")
	require.NoError(t, err)
	assert.Equal(t, "test-job-id", job.JobID)
	assert.Equal(t, "test-job", job.JobName)
	assert.Equal(t, "SUCCEEDED", job.Status)
	assert.Equal(t, "test-def:1", job.JobDefinition)
}

func TestListJobs(t *testing.T) {
	ctx := context.Background()
	
	// Mock response for list-jobs
	listResponse := map[string]interface{}{
		"jobSummaryList": []map[string]interface{}{
			{
				"jobId":   "job-1",
				"jobName": "job-one",
				"status":  "RUNNING",
			},
			{
				"jobId":   "job-2",
				"jobName": "job-two",
				"status":  "SUCCEEDED",
			},
		},
	}
	
	listJSON, err := json.Marshal(listResponse)
	require.NoError(t, err)
	
	// Save original exec function and restore after test
	originalExec := execCommandContext
	defer func() { execCommandContext = originalExec }()
	
	execCommandContext = mockExec(
		map[string][]byte{
			"list-jobs": listJSON,
		},
		map[string]error{},
	)
	
	client, err := NewLocalStackClient()
	require.NoError(t, err)
	
	jobs, err := client.ListJobs(ctx, "test-queue", "RUNNING")
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	
	assert.Equal(t, "job-1", jobs[0].JobID)
	assert.Equal(t, "job-one", jobs[0].JobName)
	assert.Equal(t, "RUNNING", jobs[0].Status)
	
	assert.Equal(t, "job-2", jobs[1].JobID)
	assert.Equal(t, "job-two", jobs[1].JobName)
	assert.Equal(t, "SUCCEEDED", jobs[1].Status)
}

func TestTerminateJob(t *testing.T) {
	ctx := context.Background()
	
	// Mock response for terminate-job
	terminateResponse := map[string]interface{}{}
	
	terminateJSON, err := json.Marshal(terminateResponse)
	require.NoError(t, err)
	
	// Save original exec function and restore after test
	originalExec := execCommandContext
	defer func() { execCommandContext = originalExec }()
	
	execCommandContext = mockExec(
		map[string][]byte{
			"terminate-job": terminateJSON,
		},
		map[string]error{},
	)
	
	client, err := NewLocalStackClient()
	require.NoError(t, err)
	
	err = client.TerminateJob(ctx, "test-job-id", "User requested")
	require.NoError(t, err)
}

func TestListJobDefinitions(t *testing.T) {
	ctx := context.Background()
	
	// Mock response for describe-job-definitions
	listResponse := map[string]interface{}{
		"jobDefinitions": []map[string]interface{}{
			{
				"jobDefinitionName": "def-1",
				"type":              "container",
				"containerProperties": map[string]interface{}{
					"image":  "alpine:latest",
					"vcpus":  1,
					"memory": 512,
				},
			},
			{
				"jobDefinitionName": "def-2",
				"type":              "container",
				"containerProperties": map[string]interface{}{
					"image":  "ubuntu:latest",
					"vcpus":  2,
					"memory": 1024,
				},
			},
		},
	}
	
	listJSON, err := json.Marshal(listResponse)
	require.NoError(t, err)
	
	// Save original exec function and restore after test
	originalExec := execCommandContext
	defer func() { execCommandContext = originalExec }()
	
	execCommandContext = mockExec(
		map[string][]byte{
			"describe-job-definitions": listJSON,
		},
		map[string]error{},
	)
	
	client, err := NewLocalStackClient()
	require.NoError(t, err)
	
	defs, err := client.ListJobDefinitions(ctx)
	require.NoError(t, err)
	require.Len(t, defs, 2)
	
	assert.Equal(t, "def-1", defs[0].Name)
	assert.Equal(t, "container", defs[0].Type)
	
	assert.Equal(t, "def-2", defs[1].Name)
	assert.Equal(t, "container", defs[1].Type)
}

func TestDeregisterJobDefinition(t *testing.T) {
	ctx := context.Background()
	
	// Mock response for deregister-job-definition
	deregisterResponse := map[string]interface{}{}
	
	deregisterJSON, err := json.Marshal(deregisterResponse)
	require.NoError(t, err)
	
	// Save original exec function and restore after test
	originalExec := execCommandContext
	defer func() { execCommandContext = originalExec }()
	
	execCommandContext = mockExec(
		map[string][]byte{
			"deregister-job-definition": deregisterJSON,
		},
		map[string]error{},
	)
	
	client, err := NewLocalStackClient()
	require.NoError(t, err)
	
	err = client.DeregisterJobDefinition(ctx, "test-definition")
	require.NoError(t, err)
}

func TestLocalStackClientCreation(t *testing.T) {
	client, err := NewLocalStackClient()
	if err != nil {
		t.Fatalf("NewLocalStackClient() error = %v", err)
	}
	
	if client == nil {
		t.Fatal("NewLocalStackClient() returned nil client")
	}
	
	if client.endpoint == "" {
		t.Error("LocalStackClient endpoint should not be empty")
	}
	
	if client.region == "" {
		t.Error("LocalStackClient region should not be empty")
	}
}

func TestLocalStackClientInterface(t *testing.T) {
	var _ Client = &LocalStackClient{}
}