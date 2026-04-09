package lambda

import (
	"context"
	"fmt"
)

// Function represents a Lambda function
type Function struct {
	Name        string
	Runtime     string
	Handler     string
	Role        string
	CodePath    string
	Description string
	Timeout     int // seconds
	Memory      int // MB
	Environment map[string]string
}

// DeployResult contains deployment output
type DeployResult struct {
	FunctionArn string
	FunctionName string
	Version     string
	CodeSha256  string
	LastModified string
}

// InvokeRequest represents a Lambda invocation
type InvokeRequest struct {
	FunctionName string
	Payload      []byte
	Async        bool
}

// InvokeResult contains invocation output
type InvokeResult struct {
	StatusCode   int
	FunctionErr  string
	ExecutedVersion string
	LogResult    string
	Payload      []byte
}

// LogEntry represents a single log line
type LogEntry struct {
	Timestamp int64
	Message   string
	Level     string // INFO, ERROR, WARNING
}

// Client is the interface for Lambda operations
type Client interface {
	// Deploy creates or updates a Lambda function
	Deploy(ctx context.Context, fn *Function) (*DeployResult, error)

	// GetFunction retrieves function metadata
	GetFunction(ctx context.Context, name string) (*Function, error)

	// ListFunctions lists all Lambda functions in the stage
	ListFunctions(ctx context.Context) ([]*Function, error)

	// DeleteFunction removes a Lambda function
	DeleteFunction(ctx context.Context, name string) error

	// Invoke executes a Lambda function
	Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResult, error)

	// GetLogs retrieves function logs
	GetLogs(ctx context.Context, functionName string, limit int) ([]LogEntry, error)

	// TailLogs streams logs in real-time
	TailLogs(ctx context.Context, functionName string, logChan chan LogEntry) error

	// UpdateCode updates function code without changing config
	UpdateCode(ctx context.Context, functionName, codePath string) error

	// UpdateConfig updates function configuration
	UpdateConfig(ctx context.Context, fn *Function) error
}

// NewClient creates the appropriate Lambda client based on stage
func NewClient(stage string) (Client, error) {
	switch stage {
	case "localstack":
		return NewLocalStackClient()
	case "mirror":
		return NewAWSClient()
	default:
		return nil, fmt.Errorf("unsupported stage: %s", stage)
	}
}

// PrettyPrintFunction returns formatted function info
func PrettyPrintFunction(fn *Function) string {
	return fmt.Sprintf(`
Lambda Function: %s
  Runtime:      %s
  Handler:      %s
  Memory:       %d MB
  Timeout:      %d sec
  Role:         %s
  Description:  %s
`,
		fn.Name,
		fn.Runtime,
		fn.Handler,
		fn.Memory,
		fn.Timeout,
		fn.Role,
		fn.Description,
	)
}
