package batch

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

// JobDefinition represents an AWS Batch job definition
type JobDefinition struct {
	Name       string
	Type       string // container, multinode
	Image      string
	Vcpus      int
	Memory     int // MB
	JobRole    string
	Command    []string
	Environment map[string]string
}

// JobQueue represents an AWS Batch job queue
type JobQueue struct {
	Name     string
	Enabled  bool
	Priority int
	ComputeEnvironmentOrder []string
}

// Job represents a submitted Batch job
type Job struct {
	JobID          string
	JobName        string
	JobDefinition  string
	JobQueue       string
	Status         string // SUBMITTED, PENDING, RUNNABLE, RUNNING, SUCCEEDED, FAILED
	ExitCode       int
	Reason         string
	SubmittedAt    int64
	StartedAt      int64
	StoppedAt      int64
	LogStreamName  string
}

// SubmitJobRequest represents a job submission
type SubmitJobRequest struct {
	JobName       string
	JobDefinition string
	JobQueue      string
	ContainerOverrides *ContainerOverrides
}

// ContainerOverrides allows overriding job container parameters
type ContainerOverrides struct {
	Command     []string
	Environment map[string]string
	Vcpus       int
	Memory      int
}

// SubmitJobResult contains job submission output
type SubmitJobResult struct {
	JobID   string
	JobArn  string
	Status  string
}

// Client is the interface for Batch operations
type Client interface {
	// RegisterJobDefinition creates or updates a job definition
	RegisterJobDefinition(ctx context.Context, def *JobDefinition) (string, error)

	// ListJobDefinitions lists all job definitions
	ListJobDefinitions(ctx context.Context) ([]*JobDefinition, error)

	// DeregisterJobDefinition removes a job definition
	DeregisterJobDefinition(ctx context.Context, name string) error

	// CreateJobQueue creates a job queue
	CreateJobQueue(ctx context.Context, queue *JobQueue) error

	// ListJobQueues lists all job queues
	ListJobQueues(ctx context.Context) ([]*JobQueue, error)

	// SubmitJob submits a job to a queue
	SubmitJob(ctx context.Context, req *SubmitJobRequest) (*SubmitJobResult, error)

	// DescribeJob gets job details and current status
	DescribeJob(ctx context.Context, jobID string) (*Job, error)

	// ListJobs lists jobs in a specific queue or status
	ListJobs(ctx context.Context, queueName string, status string) ([]*Job, error)

	// TerminateJob stops a running job
	TerminateJob(ctx context.Context, jobID string, reason string) error

	// GetLogs retrieves job logs from CloudWatch
	GetLogs(ctx context.Context, jobID string) ([]string, error)

	// TailLogs streams job logs in real-time
	TailLogs(ctx context.Context, jobID string, logChan chan string) error
}

// NewClient creates the appropriate Batch client based on stage
func NewClient(stage string) (Client, error) {
	switch stage {
	case "localstack":
		return NewLocalStackClient()
	case "mirror":
		return newAWSClient()
	default:
		return nil, fmt.Errorf("unsupported stage: %s", stage)
	}
}

// newAWSClient creates an AWS Batch client with default configuration
func newAWSClient() (*AWSClient, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-central-1"
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	batchClient := batch.NewFromConfig(cfg)
	logsClient := cloudwatchlogs.NewFromConfig(cfg)

	return NewAWSClient(batchClient, logsClient), nil
}

// PrettyPrintJob returns formatted job info
func PrettyPrintJob(job *Job) string {
	return fmt.Sprintf(`
Batch Job: %s (%s)
  Job ID:        %s
  Definition:    %s
  Queue:         %s
  Status:        %s
  Exit Code:     %d
  Reason:        %s
  Submitted:     %d
  Started:       %d
  Stopped:       %d
`,
		job.JobName,
		job.JobID,
		job.JobID,
		job.JobDefinition,
		job.JobQueue,
		job.Status,
		job.ExitCode,
		job.Reason,
		job.SubmittedAt,
		job.StartedAt,
		job.StoppedAt,
	)
}
