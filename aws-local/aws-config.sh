#!/bin/bash
# AWS Local Profile Configuration
# This script sets up AWS CLI profiles for local development

set -e

echo "Setting up AWS local profiles..."

# Create AWS config directory if it doesn't exist
mkdir -p ~/.aws

# Configure localstack profile
cat > ~/.aws/config << EOF
[profile localstack]
region = eu-central-1
output = json
endpoint_url = http://localhost:4566

[profile aws-playground]
region = eu-central-1
output = json
# Add your playground credentials here when needed
# aws_access_key_id = 
# aws_secret_access_key = 

[profile aws-local]
region = eu-central-1
output = json
# This profile can be used for local AWS SDK development
EOF

# Configure credentials for localstack (dummy credentials)
cat > ~/.aws/credentials << EOF
[localstack]
aws_access_key_id = test
aws_secret_access_key = test

[aws-playground]
# Add your playground credentials here when needed
# aws_access_key_id = 
# aws_secret_access_key = 

[aws-local]
aws_access_key_id = local
aws_secret_access_key = local
EOF

echo "AWS profiles configured:"
echo "1. localstack - For LocalStack development"
echo "2. aws-playground - For AWS playground (configure credentials when needed)"
echo "3. aws-local - For local AWS SDK development"

# Test localstack connection
echo -e "\nTesting LocalStack connection..."
if curl -s http://localhost:4566/_localstack/health > /dev/null 2>&1; then
    echo "✓ LocalStack is running"
    
    # Create a test S3 bucket
    echo "Creating test S3 bucket..."
    AWS_PROFILE=localstack aws --endpoint-url=http://localhost:4566 s3 mb s3://test-bucket-local || echo "LocalStack might not be fully started yet"
else
    echo "⚠ LocalStack is not running. Start it with: task localstack:up"
fi

echo -e "\nUsage examples:"
echo "  # Use localstack profile"
echo "  AWS_PROFILE=localstack aws --endpoint-url=http://localhost:4566 s3 ls"
echo ""
echo "  # Use aws-playground profile (configure credentials first)"
echo "  AWS_PROFILE=aws-playground aws s3 ls"
echo ""
echo "  # Use aws-local profile for SDK development"
echo "  AWS_PROFILE=aws-local aws s3 ls --endpoint-url=http://localhost:4566"
