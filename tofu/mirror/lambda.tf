# Mirror-Cloud: AWS Lambda configuration
# Functions, IAM roles, and CloudWatch integration

resource "aws_iam_role" "lambda_execution" {
  count = var.enable_compute ? 1 : 0

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name = "mirror-lambda-execution-role"
  }
}

resource "aws_iam_role_policy_attachment" "lambda_basic_execution" {
  count = var.enable_compute ? 1 : 0

  role       = aws_iam_role.lambda_execution[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "lambda_vpc_execution" {
  count = var.enable_compute ? 1 : 0

  role       = aws_iam_role.lambda_execution[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

# Add permissions for SSM Parameter Store access
resource "aws_iam_role_policy" "lambda_ssm_access" {
  count = var.enable_compute ? 1 : 0

  role = aws_iam_role.lambda_execution[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:GetParameters",
          "ssm:GetParametersByPath"
        ]
        Resource = "arn:aws:ssm:${var.aws_region}:*:parameter/mirror/*"
      }
    ]
  })
}

# CloudWatch Log Group for Lambda functions
resource "aws_cloudwatch_log_group" "lambda" {
  count = var.enable_compute ? 1 : 0

  name              = "/aws/lambda/mirror"
  retention_in_days = 7

  tags = {
    Name = "mirror-lambda-logs"
  }
}

# Example Lambda function (placeholder for testing)
resource "aws_lambda_function" "example" {
  count = var.enable_compute ? 1 : 0

  filename         = "lambda_placeholder.zip"  # Must exist or use data archive
  function_name    = "mirror-example-function"
  role             = aws_iam_role.lambda_execution[0].arn
  handler          = "index.handler"
  source_code_hash = filebase64sha256("lambda_placeholder.zip")
  runtime          = "python3.11"
  timeout          = 60
  memory_size      = 256

  vpc_config {
    subnet_ids         = aws_subnet.public[*].id
    security_group_ids = [aws_security_group.lambda[0].id]
  }

  environment {
    variables = {
      ENVIRONMENT = "mirror"
      LOG_LEVEL   = "DEBUG"
    }
  }

  tags = {
    Name = "mirror-example-function"
  }

  depends_on = [
    aws_iam_role_policy_attachment.lambda_basic_execution,
    aws_iam_role_policy_attachment.lambda_vpc_execution
  ]
}

resource "aws_security_group" "lambda" {
  count = var.enable_compute ? 1 : 0

  vpc_id = aws_vpc.mirror.id
  name   = "mirror-lambda-sg"

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "mirror-lambda-sg"
  }
}

output "lambda_execution_role_arn" {
  description = "Lambda Execution Role ARN"
  value       = try(aws_iam_role.lambda_execution[0].arn, null)
}

output "lambda_log_group" {
  description = "CloudWatch Log Group for Lambda"
  value       = try(aws_cloudwatch_log_group.lambda[0].name, null)
}
