resource "aws_sqs_queue" "development" {
  name = "development-queue"
}

resource "aws_sqs_queue" "test" {
  name = "test-queue"
}

resource "aws_sqs_queue" "dead_letter" {
  name = "dead-letter-queue"
}
