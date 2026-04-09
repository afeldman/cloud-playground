resource "aws_cloudwatch_log_group" "lambda_development" {
  name = "/aws/lambda/development"
}

resource "aws_cloudwatch_log_group" "lambda_test" {
  name = "/aws/lambda/test"
}
