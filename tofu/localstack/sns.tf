resource "aws_sns_topic" "development" {
  name = "development-topic"
}

resource "aws_sns_topic" "notifications" {
  name = "notifications-topic"
}
