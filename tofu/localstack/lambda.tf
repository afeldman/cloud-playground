# LocalStack: AWS Lambda functions
# LocalStack supports Lambda with limited runtime flexibility
# Can be used for testing containerized functions

resource "aws_lambda_function" "localstack_example" {
  filename      = "lambda_placeholder.zip"  # Placeholder — can be overridden
  function_name = "birdy-example-function"
  role          = aws_iam_role.lambda_localstack.arn
  handler       = "index.handler"
  runtime       = "python3.11"
  timeout       = 60
  memory_size   = 256

  environment {
    variables = {
      ENVIRONMENT = "local"
      LOG_LEVEL   = "DEBUG"
    }
  }

  tags = {
    Name        = "birdy-example-function"
    Environment = "local"
  }
}

resource "aws_iam_role" "lambda_localstack" {
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
    Name = "birdy-lambda-role"
  }
}

resource "aws_iam_role_policy" "lambda_localstack_ssm" {
  role = aws_iam_role.lambda_localstack.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:GetParameters"
        ]
        Resource = "*"
      }
    ]
  })
}

output "lambda_function_arn" {
  description = "LocalStack Lambda function ARN"
  value       = aws_lambda_function.localstack_example.arn
}

output "lambda_function_name" {
  description = "LocalStack Lambda function name"
  value       = aws_lambda_function.localstack_example.function_name
}
