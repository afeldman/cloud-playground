package batch

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"
)

// execCommandContext is a variable that can be mocked in tests
var execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, arg...)
}

// LocalStackClient implements Client for LocalStack Batch
type LocalStackClient struct {
	endpoint string
	region   string
}

// NewLocalStackClient creates a LocalStack Batch client
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

// RegisterJobDefinition creates or updates a job definition in LocalStack
func (c *LocalStackClient) RegisterJobDefinition(ctx context.Context, def *JobDefinition) (string, error) {
	slog.Info("registering batch job definition", "name", def.Name)

	containerProps := map[string]interface{}{
		"image":   def.Image,
		"vcpus":   def.Vcpus,
		"memory":  def.Memory,
		"command": def.Command,
	}

	propsJSON, _ := json.Marshal(containerProps)

	cmd := execCommandContext(ctx, "aws", "batch", "register-job-definition",
		"--job-definition-name", def.Name,
		"--type", def.Type,
		"--container-properties", string(propsJSON),
		"--endpoint-url", c.endpoint,
		"--region", c.region,
		"--output", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("aws batch register-job-definition failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse aws response: %w", err)
	}

	arn := fmt.Sprintf("%v", result["jobDefinitionArn"])
	slog.Info("batch job definition registered", "name", def.Name, "arn", arn)
	return arn, nil
}

// ListJobDefinitions lists all job definitions
func (c *LocalStackClient) ListJobDefinitions(ctx context.Context) ([]*JobDefinition, error) {
	cmd := execCommandContext(ctx, "aws", "batch", "describe-job-definitions",
		"--endpoint-url", c.endpoint,
		"--region", c.region,
		"--output", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aws batch describe-job-definitions failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse aws response: %w", err)
	}

	definitions := result["jobDefinitions"].([]interface{})
	var defs []*JobDefinition

	for _, d := range definitions {
		def := d.(map[string]interface{})
		defs = append(defs, &JobDefinition{
			Name: fmt.Sprintf("%v", def["jobDefinitionName"]),
			Type: fmt.Sprintf("%v", def["type"]),
		})
	}

	return defs, nil
}

// DeregisterJobDefinition removes a job definition
func (c *LocalStackClient) DeregisterJobDefinition(ctx context.Context, name string) error {
	cmd := execCommandContext(ctx, "aws", "batch", "deregister-job-definition",
		"--job-definition", name,
		"--endpoint-url", c.endpoint,
		"--region", c.region,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws batch deregister-job-definition failed: %w", err)
	}

	slog.Info("batch job definition deregistered", "name", name)
	return nil
}

// CreateJobQueue creates a job queue
func (c *LocalStackClient) CreateJobQueue(ctx context.Context, queue *JobQueue) error {
	cmd := execCommandContext(ctx, "aws", "batch", "create-job-queue",
		"--job-queue-name", queue.Name,
		"--state", "ENABLED",
		"--priority", fmt.Sprintf("%d", queue.Priority),
		"--compute-environment-order", fmt.Sprintf("order=%d", 1),
		"--endpoint-url", c.endpoint,
		"--region", c.region,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws batch create-job-queue failed: %w", err)
	}

	slog.Info("batch job queue created", "name", queue.Name)
	return nil
}

// ListJobQueues lists all job queues
func (c *LocalStackClient) ListJobQueues(ctx context.Context) ([]*JobQueue, error) {
	cmd := execCommandContext(ctx, "aws", "batch", "describe-job-queues",
		"--endpoint-url", c.endpoint,
		"--region", c.region,
		"--output", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aws batch describe-job-queues failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse aws response: %w", err)
	}

	queues := result["jobQueues"].([]interface{})
	var jqs []*JobQueue

	for _, q := range queues {
		queue := q.(map[string]interface{})
		jqs = append(jqs, &JobQueue{
			Name:     fmt.Sprintf("%v", queue["jobQueueName"]),
			Priority: int(queue["priority"].(float64)),
		})
	}

	return jqs, nil
}

// SubmitJob submits a job to Batch
func (c *LocalStackClient) SubmitJob(ctx context.Context, req *SubmitJobRequest) (*SubmitJobResult, error) {
	cmd := execCommandContext(ctx, "aws", "batch", "submit-job",
		"--job-name", req.JobName,
		"--job-definition", req.JobDefinition,
		"--job-queue", req.JobQueue,
		"--endpoint-url", c.endpoint,
		"--region", c.region,
		"--output", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aws batch submit-job failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse aws response: %w", err)
	}

	slog.Info("batch job submitted", "name", req.JobName, "jobId", result["jobId"])

	return &SubmitJobResult{
		JobID:  fmt.Sprintf("%v", result["jobId"]),
		JobArn: fmt.Sprintf("%v", result["jobArn"]),
	}, nil
}

// DescribeJob gets job details
func (c *LocalStackClient) DescribeJob(ctx context.Context, jobID string) (*Job, error) {
	cmd := execCommandContext(ctx, "aws", "batch", "describe-jobs",
		"--jobs", jobID,
		"--endpoint-url", c.endpoint,
		"--region", c.region,
		"--output", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aws batch describe-jobs failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse aws response: %w", err)
	}

	jobs := result["jobs"].([]interface{})
	if len(jobs) == 0 {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	j := jobs[0].(map[string]interface{})

	return &Job{
		JobID:         fmt.Sprintf("%v", j["jobId"]),
		JobName:       fmt.Sprintf("%v", j["jobName"]),
		Status:        fmt.Sprintf("%v", j["status"]),
		JobDefinition: fmt.Sprintf("%v", j["jobDefinition"]),
		JobQueue:      fmt.Sprintf("%v", j["jobQueue"]),
	}, nil
}

// ListJobs lists jobs in a queue
func (c *LocalStackClient) ListJobs(ctx context.Context, queueName string, status string) ([]*Job, error) {
	cmd := execCommandContext(ctx, "aws", "batch", "list-jobs",
		"--job-queue", queueName,
		"--job-status", status,
		"--endpoint-url", c.endpoint,
		"--region", c.region,
		"--output", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aws batch list-jobs failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse aws response: %w", err)
	}

	summaries := result["jobSummaryList"].([]interface{})
	var jobs []*Job

	for _, s := range summaries {
		summary := s.(map[string]interface{})
		jobs = append(jobs, &Job{
			JobID:        fmt.Sprintf("%v", summary["jobId"]),
			JobName:      fmt.Sprintf("%v", summary["jobName"]),
			Status:       fmt.Sprintf("%v", summary["status"]),
			JobQueue:     fmt.Sprintf("%v", summary["jobQueue"]),
			SubmittedAt:  time.Now().Unix(), // LocalStack may not return this
		})
	}

	return jobs, nil
}

// TerminateJob stops a running job
func (c *LocalStackClient) TerminateJob(ctx context.Context, jobID string, reason string) error {
	cmd := execCommandContext(ctx, "aws", "batch", "terminate-job",
		"--job-id", jobID,
		"--reason", reason,
		"--endpoint-url", c.endpoint,
		"--region", c.region,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws batch terminate-job failed: %w", err)
	}

	slog.Info("batch job terminated", "jobId", jobID)
	return nil
}

// GetLogs retrieves job logs
func (c *LocalStackClient) GetLogs(ctx context.Context, jobID string) ([]string, error) {
	// LocalStack Batch logs would come from CloudWatch
	return []string{}, nil
}

// TailLogs streams job logs in real-time
func (c *LocalStackClient) TailLogs(ctx context.Context, jobID string, logChan chan string) error {
	// LocalStack logs streaming
	return nil
}
