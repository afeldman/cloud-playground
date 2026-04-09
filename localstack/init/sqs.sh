#!/usr/bin/env bash
# Creates SQS queues in LocalStack.
# Mirrors terraform/localstack/sqs.tf — use this for quick setup without Terraform.

set -euo pipefail

AWS="aws --endpoint-url http://localhost:4566"

create_queue() {
  local name=$1
  shift
  if $AWS sqs get-queue-url --queue-name "$name" 2>/dev/null; then
    echo "  ✓ $name (already exists)"
  else
    $AWS sqs create-queue --queue-name "$name" "$@"
    echo "  ✓ $name (created)"
  fi
}

echo "Creating SQS queues..."
create_queue "dead-letter-queue"
create_queue "development-queue"
create_queue "test-queue"
echo "Done."
