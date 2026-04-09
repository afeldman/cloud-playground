package lambda

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// AWSClient implements Client for real AWS Lambda
type AWSClient struct {
	client *lambda.Client
	region string
}

// NewAWSClient creates a real AWS Lambda client
func NewAWSClient() (*AWSClient, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-central-1"
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSClient{
		client: lambda.NewFromConfig(cfg),
		region: region,
	}, nil
}

// Deploy creates or updates a function in AWS
func (c *AWSClient) Deploy(ctx context.Context, fn *Function) (*DeployResult, error) {
	slog.Info("deploying lambda function to aws", "name", fn.Name, "region", c.region)

	// Read function code
	codeBytes, err := os.ReadFile(fn.CodePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read code file: %w", err)
	}

	// Try to create function
	createOutput, err := c.client.CreateFunction(ctx, &lambda.CreateFunctionInput{
		FunctionName: aws.String(fn.Name),
		Runtime:      types.Runtime(fn.Runtime),
		Role:         aws.String(fn.Role),
		Handler:      aws.String(fn.Handler),
		Code: &types.FunctionCode{
			ZipFile: codeBytes,
		},
		Timeout:     aws.Int32(int32(fn.Timeout)),
		MemorySize:  aws.Int32(int32(fn.Memory)),
		Description: aws.String(fn.Description),
	})

	// If function exists, update it
	if err != nil {
		updateOutput, err := c.client.UpdateFunctionCode(ctx, &lambda.UpdateFunctionCodeInput{
			FunctionName: aws.String(fn.Name),
			ZipFile:      codeBytes,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update function code: %w", err)
		}

		slog.Info("lambda function updated", "name", fn.Name, "arn", *updateOutput.FunctionArn)

		return &DeployResult{
			FunctionArn:  *updateOutput.FunctionArn,
			FunctionName: *updateOutput.FunctionName,
			Version:      *updateOutput.Version,
			CodeSha256:   *updateOutput.CodeSha256,
			LastModified: *updateOutput.LastModified,
		}, nil
	}

	slog.Info("lambda function deployed", "name", fn.Name, "arn", *createOutput.FunctionArn)

	return &DeployResult{
		FunctionArn:  *createOutput.FunctionArn,
		FunctionName: *createOutput.FunctionName,
		Version:      *createOutput.Version,
		CodeSha256:   *createOutput.CodeSha256,
		LastModified: *createOutput.LastModified,
	}, nil
}

// GetFunction retrieves function metadata
func (c *AWSClient) GetFunction(ctx context.Context, name string) (*Function, error) {
	output, err := c.client.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(name),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	config := output.Configuration
	fn := &Function{
		Name:     *config.FunctionName,
		Runtime:  string(config.Runtime),
		Handler:  *config.Handler,
		Role:     *config.Role,
		Timeout:  int(*config.Timeout),
		Memory:   int(*config.MemorySize),
	}

	if config.Description != nil {
		fn.Description = *config.Description
	}

	return fn, nil
}

// ListFunctions lists all Lambda functions
func (c *AWSClient) ListFunctions(ctx context.Context) ([]*Function, error) {
	paginator := lambda.NewListFunctionsPaginator(c.client, &lambda.ListFunctionsInput{})

	var functions []*Function

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list functions: %w", err)
		}

		for _, f := range page.Functions {
			functions = append(functions, &Function{
				Name:    *f.FunctionName,
				Runtime: string(f.Runtime),
				Handler: *f.Handler,
				Memory:  int(*f.MemorySize),
				Timeout: int(*f.Timeout),
			})
		}
	}

	return functions, nil
}

// DeleteFunction removes a function from AWS
func (c *AWSClient) DeleteFunction(ctx context.Context, name string) error {
	_, err := c.client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: aws.String(name),
	})
	if err != nil {
		return fmt.Errorf("failed to delete function: %w", err)
	}

	slog.Info("lambda function deleted", "name", name)
	return nil
}

// Invoke executes a function in AWS
func (c *AWSClient) Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResult, error) {
	invokeType := types.InvocationTypeRequestResponse
	if req.Async {
		invokeType = types.InvocationTypeEvent
	}

	output, err := c.client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(req.FunctionName),
		InvocationType: invokeType,
		Payload:        req.Payload,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invoke function: %w", err)
	}

	result := &InvokeResult{
		StatusCode: int(output.StatusCode),
	}

	if output.FunctionError != nil {
		result.FunctionErr = *output.FunctionError
	}

	if output.ExecutedVersion != nil {
		result.ExecutedVersion = *output.ExecutedVersion
	}

	if output.LogResult != nil {
		result.LogResult = *output.LogResult
	}

	// Payload is already []byte, no need to use io.ReadAll
	result.Payload = output.Payload

	return result, nil
}

// GetLogs retrieves function logs from CloudWatch
func (c *AWSClient) GetLogs(ctx context.Context, functionName string, limit int) ([]LogEntry, error) {
	// Note: Requires CloudWatch Logs client
	// This is simplified - full implementation would query CloudWatch Logs
	slog.Info("getting logs for function", "name", functionName)

	return []LogEntry{}, nil
}

// TailLogs streams function logs in real-time
func (c *AWSClient) TailLogs(ctx context.Context, functionName string, logChan chan LogEntry) error {
	// Note: Requires CloudWatch Logs client for streaming
	// This is simplified for now
	slog.Info("tailing logs for function", "name", functionName)

	return nil
}

// UpdateCode updates function code
func (c *AWSClient) UpdateCode(ctx context.Context, functionName, codePath string) error {
	codeBytes, err := os.ReadFile(codePath)
	if err != nil {
		return fmt.Errorf("failed to read code file: %w", err)
	}

	_, err = c.client.UpdateFunctionCode(ctx, &lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(functionName),
		ZipFile:      codeBytes,
	})
	if err != nil {
		return fmt.Errorf("failed to update function code: %w", err)
	}

	slog.Info("lambda function code updated", "name", functionName)
	return nil
}

// UpdateConfig updates function configuration
func (c *AWSClient) UpdateConfig(ctx context.Context, fn *Function) error {
	_, err := c.client.UpdateFunctionConfiguration(ctx, &lambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(fn.Name),
		Runtime:      types.Runtime(fn.Runtime),
		Handler:      aws.String(fn.Handler),
		Timeout:      aws.Int32(int32(fn.Timeout)),
		MemorySize:   aws.Int32(int32(fn.Memory)),
		Description:  aws.String(fn.Description),
	})
	if err != nil {
		return fmt.Errorf("failed to update function configuration: %w", err)
	}

	slog.Info("lambda function config updated", "name", fn.Name)
	return nil
}
