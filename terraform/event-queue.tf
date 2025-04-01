// dropping messages on the floor is bad, so lets create a dead letter queue for problematic messages.
// Note, in a production setup there should be monitors looking for messages in these queues!
resource "aws_sqs_queue" "events_dlq" {
  name = "${local.workspace_prefix}GoLambdaDemoEvents-dlq"

  // max the message retention
  message_retention_seconds = 1209600
}

// the queue where events that should be persisted get written to
resource "aws_sqs_queue" "events" {
  name = "${local.workspace_prefix}GoLambdaDemoEvents"

  // slight buffer on lambda execution time
  visibility_timeout_seconds = local.event_persistor_lambda_timeout_seconds + 1
  // max the message retention
  message_retention_seconds = 1209600

  // connect our DLQ
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.events_dlq.arn
    maxReceiveCount     = 4
  })
}

// give our queue permissions to send dead letters to the DLQ
resource "aws_sqs_queue_redrive_allow_policy" "events_dlq" {
  queue_url = aws_sqs_queue.events_dlq.id

  redrive_allow_policy = jsonencode({
    redrivePermission = "byQueue",
    sourceQueueArns   = [aws_sqs_queue.events.arn]
  })
}

output "event_queue_url" {
  value = aws_sqs_queue.events.url
}