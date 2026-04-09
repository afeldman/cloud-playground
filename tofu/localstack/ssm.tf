resource "aws_ssm_parameter" "db_host" {
  name  = "/development/database/host"
  type  = "String"
  value = "localhost"
}

resource "aws_ssm_parameter" "db_port" {
  name  = "/development/database/port"
  type  = "String"
  value = "5432"
}

resource "aws_ssm_parameter" "api_key" {
  name  = "/development/api/key"
  type  = "SecureString"
  value = "test-api-key-123"
}
