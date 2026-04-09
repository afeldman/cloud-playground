resource "aws_s3_bucket" "development" {
  bucket = "development-bucket"
}

resource "aws_s3_bucket" "test" {
  bucket = "test-bucket"
}

resource "aws_s3_bucket" "artifacts" {
  bucket = "artifacts-bucket"
}
