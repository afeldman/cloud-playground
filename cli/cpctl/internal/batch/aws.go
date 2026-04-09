package batch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/batch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

// AWSClient implements Client for real AWS Batch
type AWSClient struct {
	batchClient *batch.Client
	logsClient  *cloudwatchlogs.Client
}

// NewAWSClient creates an AWS Batch client
func NewAWSClient(batchSvc *batch.Client, logsSvc *cloudwatchlogs.Client) *AWSClient {
	return &AWSClient{
		batchClient: batchSvc,
		logsClient:  logsSvc,
	}
}

// RegisterJobDefinition creates or updates a job definition in AWS
func (c *AWSClient) RegisterJobDefinition(ctx context.Context, def *JobDefinition) (string, error) {
	slog.Info("registering batch job definition", "name", def.Name)

	containerProps := &types.ContainerProperties{
		Image:   aws.String(def.Image),
		Vcpus:   aws.Int32(int32(def.Vcpus)),
		Memory:  aws.Int32(int32(def.Memory)),
		Command: def.Command,
	}

	input := &batch.RegisterJobDefinitionInput{
		JobDefinitionName: aws.String(def.Name),
		Type:              types.JobDefinitionType(def.Type),
		ContainerProperties: containerProps,
	}

	result, err := c.batchClient.RegisterJobDefinition(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to register job definition: %w", err)
	}

	slog.Info("batch job definition registered",
		"name", def.Name,
		"arn", *result.JobDefinitionArn,
		"revision", *result.Revision,
	)

	return *result.JobDefinitionArn, nil
}

// ListJobDefinitions lists all job definitions
func (c *AWSClient) ListJobDefinitions(ctx context.Context) ([]*JobDefinition, error) {
	input := &batch.DescribeJobDefinitionsInput{}

	result, err := c.batchClient.DescribeJobDefinitions(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list job definitions: %w", err)
	}

	var defs []*JobDefinition
	for _, jd := range result.JobDefinitions {
		containerProps := jd.ContainerProperties
		defs = append(defs, &JobDefinition{
			Name:        *jd.JobDefinitionName,
			Type:        *jd.Type,
			Image:       *containerProps.Image,
			Vcpus:       int(*containerProps.Vcpus),
			Memory:      int(*containerProps.Memory),
		})
	}

	return defs, nil
}

// DeregisterJobDefinition removes a job definition
func (c *AWSClient) DeregisterJobDefinition(ctx context.Context, name string) error {
	input := &batch.DeregisterJobDefinitionInput{
		JobDefinition: aws.String(name),
	}

	_, err := c.batchClient.DeregisterJobDefinition(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to deregister job definition: %w", err)
	}

	slog.Info("batch job definition deregistered", "name", name)
	return nil
}

// CreateJobQueue creates a job queue
func (c *AWSClient) CreateJobQueue(ctx context.Context, queue *JobQueue) error {
	// Note: In real AWS, you need to specify compute environment orders
	// This is a simplified version for the example
	input := &batch.CreateJobQueueInput{
		JobQueueName: aws.String(queue.Name),
		Priority:     aws.Int32(int32(queue.Priority)),
		State:        types.JQStateEnabled,
		ComputeEnvironmentOrder: []types.ComputeEnvironmentOrder{
			{
				Order:               aws.Int32(1),
				ComputeEnvironment:  aws.String("default"),
			},
		},
	}

	_, err := c.batchClient.CreateJobQueue(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create job queue: %w", err)
	}

	slog.Info("batch job queue created", "name", queue.Name)
	return nil
}

// ListJobQueues lists all job queues
func (c *AWSClient) ListJobQueues(ctx context.Context) ([]*JobQueue, error) {
	input := &batch.DescribeJobQueuesInput{}

	result, err := c.batchClient.DescribeJobQueues(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list job queues: %w", err)
	}

	var queues []*JobQueue
	for _, jq := range result.JobQueues {
		queues = append(queues, &JobQueue{
			Name:     *jq.JobQueueName,
			Priority: int(*jq.Priority),
		})
	}

	return queues, nil
}

// SubmitJob submits a job to Batch
func (c *AWSClient) SubmitJob(ctx context.Context, req *SubmitJobRequest) (*SubmitJobResult, error) {
	input := &batch.SubmitJobInput{
		JobName:       aws.String(req.JobName),
		JobDefinition: aws.String(req.JobDefinition),
		JobQueue:      aws.String(req.JobQueue),
	}

	// Handle container overrides if provided
	if req.ContainerOverrides != nil && req.ContainerOverrides.Command != nil {
		input.ContainerOverrides = &types.ContainerOverrides{
			Command: req.ContainerOverrides.Command,
		}
	}

	result, err := c.batchClient.SubmitJob(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to submit job: %w", err)
	}

	slog.Info("batch job submitted",
		"name", req.JobName,
		"jobId", *result.JobId,
	)

	return &SubmitJobResult{
		JobID:  *result.JobId,
		JobArn: *result.JobArn,
	}, nil
}

// DescribeJob gets job details
func (c *AWSClient) DescribeJob(ctx context.Context, jobID string) (*Job, error) {
	input := &batch.DescribeJobsInput{
		Jobs: []string{jobID},
	}

	result, err := c.batchClient.DescribeJobs(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe job: %w", err)
	}

	if len(result.Jobs) == 0 {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	j := result.Jobs[0]

	// AWS SDK JobDetail has different fields than our internal Job struct
	// Use available fields and default zeros for missing ones
	return &Job{
		JobID:         *j.JobId,
		JobName:       *j.JobName,
		Status:        string(j.Status),
		JobDefinition: *j.JobDefinition,
		JobQueue:      "", // Not available in JobDetail
		SubmittedAt:   0,  // Not available in JobDetail
		StartedAt:     0,  // Not available in JobDetail
	}, nil
}

// ListJobs lists jobs in a queue
func (c *AWSClient) ListJobs(ctx context.Context, queueName string, status string) ([]*Job, error) {
	input := &batch.ListJobsInput{
		JobQueue: aws.String(queueName),
		Filters: []types.KeyValuesPair{
			{
				Name:   aws.String("name"),
				Values: []string{status},
			},
		},
	}

	result, err := c.batchClient.ListJobs(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	var jobs []*Job
	for _, s := range result.JobSummaryList {
		jobs = append(jobs, &Job{
			JobID:       *s.JobId,
			JobName:     *s.JobName,
			Status:      string(s.Status),
			JobQueue:    queueName, // Pass the queue name from the parameter
			SubmittedAt: 0,          // Not available in JobSummary
		})
	}

	return jobs, nil
}

// TerminateJob stops a running job
func (c *AWSClient) TerminateJob(ctx context.Context, jobID string, reason string) error {
	input := &batch.TerminateJobInput{
		JobId:  aws.String(jobID),
		Reason: aws.String(reason),
	}

	_, err := c.batchClient.TerminateJob(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to terminate job: %w", err)
	}

	slog.Info("batch job terminated", "jobId", jobID)
	return nil
}

// GetLogs retrieves job logs from CloudWatch
func (c *AWSClient) GetLogs(ctx context.Context, jobID string) ([]string, error) {
	// In AWS, Batch logs are in a CloudWatch log group
	// Log group name is typically /aws/batch/job
	logGroupName := "/aws/batch/job"
	logStreamName := jobID

	input := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(logGroupName),
		LogStreamName: aws.String(logStreamName),
	}

	result, err := c.logsClient.GetLogEvents(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	var logLines []string
	for _, event := range result.Events {
		logLines = append(logLines, *event.Message)
	}

	return logLines, nil
}

// TailLogs streams job logs in real-time (simplified version)
func (c *AWSClient) TailLogs(ctx context.Context, jobID string, logChan chan string) error {
	logGroupName := "/aws/batch/job"
	logStreamName := jobID

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastTimestamp := time.Now().UnixMilli()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			input := &cloudwatchlogs.FilterLogEventsInput{
				LogGroupName:   aws.String(logGroupName),
				LogStreamNames: []string{logStreamName},
				StartTime:      aws.Int64(lastTimestamp),
			}

			result, err := c.logsClient.FilterLogEvents(ctx, input)
			if err != nil {
				// Log stream might not exist yet
				continue
			}

			for _, event := range result.Events {
				logChan <- *event.Message
				lastTimestamp = *event.Timestamp
			}
		}
	}
}
