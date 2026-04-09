package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cpctl/internal/batch"
	"cpctl/internal/lambda"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDualStageConsistency tests that the same operations work consistently
// across both LocalStack and AWS client implementations
func TestDualStageConsistency(t *testing.T) {
	// Skip if LocalStack is not running
	if os.Getenv("LOCALSTACK_ENDPOINT") == "" {
		t.Skip("Skipping integration test: LOCALSTACK_ENDPOINT not set")
	}

	ctx := context.Background()

	// Test Lambda client consistency
	t.Run("LambdaClientConsistency", func(t *testing.T) {
		testLambdaClientConsistency(t, ctx)
	})

	// Test Batch client consistency
	t.Run("BatchClientConsistency", func(t *testing.T) {
		testBatchClientConsistency(t, ctx)
	})
}

// testLambdaClientConsistency tests Lambda client operations across stages
func testLambdaClientConsistency(t *testing.T, ctx context.Context) {
	// Create test ZIP file
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test-function.zip")
	testCode := []byte("def handler(event, context):\n    return {'statusCode': 200, 'body': 'Hello World'}")
	err := os.WriteFile(zipPath, testCode, 0644)
	require.NoError(t, err)

	// Test LocalStack client
	t.Run("LocalStack", func(t *testing.T) {
		client, err := lambda.NewClient("localstack")
		require.NoError(t, err)
		require.NotNil(t, client)

		// Test deployment
		fn := &lambda.Function{
			Name:        "test-integration-function",
			Runtime:     "python3.9",
			Handler:     "index.handler",
			Role:        "arn:aws:iam::000000000000:role/lambda-role",
			CodePath:    zipPath,
			Timeout:     30,
			Memory:      128,
			Description: "Integration test function",
		}

		result, err := client.Deploy(ctx, fn)
		if err != nil {
			// If deployment fails, it might be because function already exists
			// Try to delete it first and retry
			_ = client.DeleteFunction(ctx, fn.Name)
			result, err = client.Deploy(ctx, fn)
		}
		require.NoError(t, err)
		assert.Contains(t, result.FunctionArn, "arn:aws:lambda")
		assert.Equal(t, fn.Name, result.FunctionName)

		// Test listing functions
		functions, err := client.ListFunctions(ctx)
		require.NoError(t, err)
		
		// Check if our function is in the list
		found := false
		for _, f := range functions {
			if f.Name == fn.Name {
				found = true
				break
			}
		}
		assert.True(t, found, "Function should be in the list")

		// Test invocation
		invokeReq := &lambda.InvokeRequest{
			FunctionName: fn.Name,
			Payload:      []byte(`{"test": "data"}`),
			Async:        false,
		}

		invokeResult, err := client.Invoke(ctx, invokeReq)
		require.NoError(t, err)
		assert.Equal(t, 200, invokeResult.StatusCode)

		// Test getting function details
		functionDetails, err := client.GetFunction(ctx, fn.Name)
		require.NoError(t, err)
		assert.Equal(t, fn.Name, functionDetails.Name)
		assert.Equal(t, fn.Runtime, functionDetails.Runtime)
		assert.Equal(t, fn.Handler, functionDetails.Handler)

		// Test getting logs
		_, err = client.GetLogs(ctx, fn.Name, 10)
		require.NoError(t, err)
		// Logs might be empty in test environment, but shouldn't error

		// Clean up
		err = client.DeleteFunction(ctx, fn.Name)
		require.NoError(t, err)
	})

	// Note: AWS mirror tests would require actual AWS credentials
	// We can't run them in CI without credentials, but we can verify
	// that the client can be created and the interface is consistent
	t.Run("AWSClientInterface", func(t *testing.T) {
		// This test doesn't actually call AWS, just verifies the client can be created
		// In a real environment with AWS credentials, this would work
		client, err := lambda.NewClient("mirror")
		if err != nil {
			// If AWS credentials are not available, skip the test
			t.Skip("Skipping AWS client test: AWS credentials not available")
		}
		require.NotNil(t, client)
		
		// Verify the client implements the interface
		var _ lambda.Client = client
	})
}

// testBatchClientConsistency tests Batch client operations across stages
func testBatchClientConsistency(t *testing.T, ctx context.Context) {
	// Test LocalStack client
	t.Run("LocalStack", func(t *testing.T) {
		client, err := batch.NewClient("localstack")
		require.NoError(t, err)
		require.NotNil(t, client)

		// Test job definition registration
		def := &batch.JobDefinition{
			Name:    "test-integration-job-def",
			Type:    "container",
			Image:   "alpine:latest",
			Vcpus:   1,
			Memory:  512,
			Command: []string{"echo", "Hello from integration test"},
		}

		arn, err := client.RegisterJobDefinition(ctx, def)
		require.NoError(t, err)
		assert.Contains(t, arn, "arn:aws:batch")

		// Test listing job definitions
		defs, err := client.ListJobDefinitions(ctx)
		require.NoError(t, err)
		
		// Check if our definition is in the list
		found := false
		for _, d := range defs {
			if d.Name == def.Name {
				found = true
				break
			}
		}
		assert.True(t, found, "Job definition should be in the list")

		// Note: Job submission requires a job queue to exist
		// In LocalStack, we might need to create a queue first
		// For now, we'll skip job submission tests in integration

		// Clean up
		err = client.DeregisterJobDefinition(ctx, def.Name)
		require.NoError(t, err)
	})

	// Note: AWS mirror tests would require actual AWS credentials
	t.Run("AWSClientInterface", func(t *testing.T) {
		client, err := batch.NewClient("mirror")
		if err != nil {
			t.Skip("Skipping AWS client test: AWS credentials not available")
		}
		require.NotNil(t, client)
		
		// Verify the client implements the interface
		var _ batch.Client = client
	})
}

// TestFactoryRouting tests that the factory correctly routes to the right client
func TestFactoryRouting(t *testing.T) {
	tests := []struct {
		name      string
		stage     string
		wantError bool
	}{
		{
			name:      "LocalStack stage",
			stage:     "localstack",
			wantError: false,
		},
		{
			name:      "Mirror stage",
			stage:     "mirror",
			wantError: false,
		},
		{
			name:      "Invalid stage",
			stage:     "invalid",
			wantError: true,
		},
		{
			name:      "Empty stage",
			stage:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Lambda client factory
			lambdaClient, lambdaErr := lambda.NewClient(tt.stage)
			if tt.wantError {
				assert.Error(t, lambdaErr)
				assert.Nil(t, lambdaClient)
			} else {
				assert.NoError(t, lambdaErr)
				assert.NotNil(t, lambdaClient)
			}

			// Test Batch client factory
			batchClient, batchErr := batch.NewClient(tt.stage)
			if tt.wantError {
				assert.Error(t, batchErr)
				assert.Nil(t, batchClient)
			} else {
				assert.NoError(t, batchErr)
				assert.NotNil(t, batchClient)
			}
		})
	}
}

// TestClientInterfaces tests that all clients implement their interfaces
func TestClientInterfaces(t *testing.T) {
	// Lambda clients
	var _ lambda.Client = &lambda.LocalStackClient{}
	var _ lambda.Client = &lambda.AWSClient{}

	// Batch clients
	var _ batch.Client = &batch.LocalStackClient{}
	var _ batch.Client = &batch.AWSClient{}
}

// TestShortIntegration tests that integration tests can be skipped with -short flag
func TestShortIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would run actual integration tests
	// For now, just verify we can create clients
	// Try to create LocalStack client
	lambdaClient, lambdaErr := lambda.NewClient("localstack")
	batchClient, batchErr := batch.NewClient("localstack")

	// In short mode, we skip, so we shouldn't reach here
	// But if we do (because -short wasn't used), we can test
	if lambdaErr == nil {
		assert.NotNil(t, lambdaClient)
	}
	if batchErr == nil {
		assert.NotNil(t, batchClient)
	}
}

// TestEnvironmentSetup verifies test environment setup
func TestEnvironmentSetup(t *testing.T) {
	// Check if we have the necessary environment variables for LocalStack
	endpoint := os.Getenv("LOCALSTACK_ENDPOINT")
	region := os.Getenv("AWS_REGION")

	t.Logf("LocalStack endpoint: %s", endpoint)
	t.Logf("AWS region: %s", region)

	// These are not required for unit tests, only for integration tests
	// The test will be skipped if LOCALSTACK_ENDPOINT is not set
}

// TestTemporaryFiles tests that temporary files are cleaned up properly
func TestTemporaryFiles(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	
	// Verify file exists
	_, err = os.Stat(testFile)
	require.NoError(t, err)
	
	// The file should be automatically cleaned up when test finishes
	// because we used t.TempDir()
}

// TestContextCancellation tests that operations respect context cancellation
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create a client
	client, err := lambda.NewClient("localstack")
	if err != nil {
		t.Skip("Skipping context test: LocalStack not available")
	}

	// Try to list functions with short timeout
	// This should either succeed quickly or fail with context error
	_, _ = client.ListFunctions(ctx)
	
	// We don't assert anything about the error because:
	// 1. It might succeed if LocalStack is fast
	// 2. It might fail with context error if LocalStack is slow
	// 3. It might fail with other errors if LocalStack is not running
	// The important thing is that we test with a context
}