# LocalStack: AWS Batch configuration (limited support)
# LocalStack supports Job Definitions and Queues but not actual job execution
# Actual execution happens via Kubernetes Jobs on batch-worker node

terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

# LocalStack Batch configuration
resource "aws_batch_compute_environment" "localstack" {
  compute_environment_name = "birdy-batch-compute-env"
  type                     = "MANAGED"
  state                    = "ENABLED"

  tags = {
    Name        = "birdy-batch-compute-env"
    Environment = "local"
  }
}

resource "aws_batch_job_queue" "localstack" {
  name                 = "birdy-job-queue"
  state                = "ENABLED"
  priority             = 1
  compute_environment_order {
    order               = 1
    compute_environment = aws_batch_compute_environment.localstack.arn
  }

  tags = {
    Name        = "birdy-job-queue"
    Environment = "local"
  }
}

# Example Job Definition
resource "aws_batch_job_definition" "localstack_example" {
  name             = "birdy-example-job"
  type             = "container"
  revision         = 1
  container_properties = jsonencode({
    image   = "busybox"
    vcpus   = 1
    memory  = 128
    command = ["echo", "hello from localstack"]
  })

  tags = {
    Name        = "birdy-example-job"
    Environment = "local"
  }
}

output "batch_job_queue_name" {
  description = "LocalStack Batch job queue name"
  value       = aws_batch_job_queue.localstack.name
}

output "batch_job_def_arn" {
  description = "LocalStack Batch job definition ARN"
  value       = aws_batch_job_definition.localstack_example.arn
}
