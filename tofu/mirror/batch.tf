# Mirror-Cloud: AWS Batch configuration
# Provides Job Definitions, Compute Environments, and Job Queues
# Ready for both LocalStack (via testcontainers) and real AWS

resource "aws_iam_role" "batch_service_role" {
  count = var.enable_batch ? 1 : 0

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "batch.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name = "mirror-batch-service-role"
  }
}

resource "aws_iam_role_policy_attachment" "batch_service_policy" {
  count = var.enable_batch ? 1 : 0

  role       = aws_iam_role.batch_service_role[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSBatchServiceRole"
}

resource "aws_iam_role" "ecs_instance_role" {
  count = var.enable_batch ? 1 : 0

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name = "mirror-ecs-instance-role"
  }
}

resource "aws_iam_role_policy_attachment" "ecs_instance_policy" {
  count = var.enable_batch ? 1 : 0

  role       = aws_iam_role.ecs_instance_role[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceforEC2Role"
}

resource "aws_iam_instance_profile" "ecs_instance" {
  count = var.enable_batch ? 1 : 0

  role = aws_iam_role.ecs_instance_role[0].name
}

resource "aws_security_group" "batch" {
  count = var.enable_batch ? 1 : 0

  vpc_id = aws_vpc.mirror.id
  name   = "mirror-batch-sg"

  ingress {
    from_port   = 0
    to_port     = 65535
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "mirror-batch-sg"
  }
}

resource "aws_batch_compute_environment" "mirror" {
  count = var.enable_batch ? 1 : 0

  compute_environment_name = "mirror-compute-env"
  type                     = "MANAGED"
  state                    = "ENABLED"
  service_role             = aws_iam_role.batch_service_role[0].arn

  compute_resources {
    type                = "EC2"
    allocationStrategy  = "SPOT_CAPACITY_OPTIMIZED"
    minvCpus            = 0
    maxvCpus            = 256
    desiredvCpus        = 0
    instanceRole        = aws_iam_instance_profile.ecs_instance[0].arn
    instanceTypes       = ["optimal"]
    subnets             = aws_subnet.public[*].id
    securityGroupIds    = [aws_security_group.batch[0].id]
    tags = {
      Name = "mirror-batch-worker"
    }
  }

  tags = {
    Name = "mirror-compute-env"
  }

  depends_on = [
    aws_iam_role_policy_attachment.batch_service_policy
  ]
}

resource "aws_batch_job_queue" "mirror" {
  count = var.enable_batch ? 1 : 0

  name                 = "mirror-job-queue"
  state                = "ENABLED"
  priority             = 1
  compute_environment_order {
    order               = 1
    compute_environment = aws_batch_compute_environment.mirror[0].arn
  }

  tags = {
    Name = "mirror-job-queue"
  }

  depends_on = [
    aws_batch_compute_environment.mirror
  ]
}

resource "aws_batch_job_definition" "example" {
  count = var.enable_batch ? 1 : 0

  name             = "mirror-example-job"
  type             = "container"
  revision         = 1
  container_properties = jsonencode({
    image      = "busybox"
    vcpus      = 1
    memory     = 128
    command    = ["echo", "hello world"]
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = aws_cloudwatch_log_group.batch[0].name
        awslogs-region        = var.aws_region
        awslogs-stream-prefix = "batch"
      }
    }
  })

  tags = {
    Name = "mirror-example-job"
  }

  depends_on = [
    aws_cloudwatch_log_group.batch
  ]
}

resource "aws_cloudwatch_log_group" "batch" {
  count = var.enable_batch ? 1 : 0

  name              = "/aws/batch/mirror"
  retention_in_days = 7

  tags = {
    Name = "mirror-batch-logs"
  }
}

output "batch_compute_env_arn" {
  description = "Batch Compute Environment ARN"
  value       = try(aws_batch_compute_environment.mirror[0].arn, null)
}

output "batch_job_queue_arn" {
  description = "Batch Job Queue ARN"
  value       = try(aws_batch_job_queue.mirror[0].arn, null)
}
