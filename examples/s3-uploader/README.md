# S3 Uploader Example

This example demonstrates how to use AWS S3 with LocalStack for local development.

## Prerequisites

1. LocalStack running: `task localstack:up`
2. Python 3.7+ with boto3 installed

## Installation

```bash
cd examples/s3-uploader
pip install boto3
```

## Usage

Run the example:

```bash
python s3_uploader.py
```

## What it does

1. Connects to LocalStack S3 service
2. Creates a bucket if it doesn't exist
3. Generates test data and uploads it to S3
4. Lists all objects in the bucket
5. Downloads the uploaded file
6. Verifies the downloaded data

## Configuration

The script can be configured to use either:
- **LocalStack** (default): For local development
- **AWS Playground**: For testing in AWS playground environment

To use AWS playground:

```python
uploader = LocalStackS3Uploader(use_localstack=False)
```

Make sure to configure AWS credentials first:
```bash
task aws:playground-auth
```

## Integration with Kubernetes

This example can be deployed to the local Kubernetes cluster:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: s3-uploader-job
spec:
  template:
    spec:
      containers:
      - name: s3-uploader
        image: python:3.9-slim
        command: ["python", "/app/s3_uploader.py"]
        env:
        - name: AWS_ENDPOINT_URL
          value: "http://host.docker.internal:4566"
        - name: AWS_ACCESS_KEY_ID
          value: "test"
        - name: AWS_SECRET_ACCESS_KEY
          value: "test"
        - name: AWS_DEFAULT_REGION
          value: "eu-central-1"
        volumeMounts:
        - name: app-volume
          mountPath: /app
      volumes:
      - name: app-volume
        configMap:
          name: s3-uploader-script
      restartPolicy: Never
```

## Testing

Run the example to verify your LocalStack setup:

```bash
# Start LocalStack if not running
task localstack:up

# Run the example
cd examples/s3-uploader
python s3_uploader.py
```

Expected output:
```
============================================================
AWS LocalStack S3 Uploader Demo
============================================================
Bucket 'development-bucket' already exists

============================================================
Step 1: Creating and uploading test data
============================================================
Created test file: test_data.json
Uploaded 'test_data.json' to 'development-bucket/test-data.json'
Presigned URL (valid for 1 hour): http://localhost:4566/development-bucket/test-data.json?...

============================================================
Step 2: Listing bucket contents
============================================================

Objects in bucket 'development-bucket':
  - test-data.json (123 bytes, last modified: 2024-01-15 10:30:00)

============================================================
Step 3: Downloading file
============================================================
Downloaded 'test-data.json' to 'downloaded_test_data.json'

Downloaded data verification:
  Timestamp: 2024-01-15T10:30:00.123456
  Environment: localstack

Cleaned up temporary files

============================================================
Demo completed successfully!
============================================================
```
