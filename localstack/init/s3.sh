#!/usr/bin/env bash
# Creates S3 buckets in LocalStack.
# Mirrors terraform/localstack/s3.tf — use this for quick setup without Terraform.

set -euo pipefail

AWS="aws --endpoint-url http://localhost:4566"

create_bucket() {
  local name=$1
  if $AWS s3api head-bucket --bucket "$name" 2>/dev/null; then
    echo "  ✓ $name (already exists)"
  else
    $AWS s3 mb "s3://$name" --region eu-central-1
    echo "  ✓ $name (created)"
  fi
}

echo "Creating S3 buckets..."
create_bucket "development-bucket"
create_bucket "test-bucket"
create_bucket "artifacts-bucket"
echo "Done."
