locals {
  event_persistor_lambda_timeout_seconds = 10
}

// IAM role for our lambda
resource "aws_iam_role" "sqs_consumer" {
  name = "${local.workspace_prefix}GoLambdaDemoSQSConsumer"
  // this allows lambdas to assume this role
  assume_role_policy = data.aws_iam_policy_document.assume_lambda_role_policy.json
}

// policy document for the lambdas role
data "aws_iam_policy_document" "sqs_consumer" {
  statement {
    sid    = "AllowLogging"
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = [
      "${aws_cloudwatch_log_group.sqs_consumer_logs.arn}:*"
    ]
  }

  statement {
    sid    = "AllowXRayWrite"
    effect = "Allow"
    actions = [
      "xray:PutTraceSegments",
      "xray:PutTelemetryRecords",
      "xray:GetSamplingRules",
      "xray:GetSamplingTargets",
      "xray:GetSamplingStatisticSummaries"
    ]
    resources = ["*"]
  }

  statement {
    sid    = "SQSAccess"
    effect = "Allow"
    actions = [
      // note, were including all the permissions needed to reset message visibility even though our example lambda doesn't.
      // That would be useful if you have a long running lambda but you want reprocessing on errors to happen quickly.
      "sqs:GetQueueAttributes",
      "sqs:GetQueueUrl",
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
      "sqs:ChangeMessageVisibility"
    ]
    resources = [
      aws_sqs_queue.events.arn
    ]
  }

  statement {
    sid    = "DynamoAccess"
    effect = "Allow"
    actions = [
      "dynamodb:PutItem",
    ]
    resources = [
      aws_dynamodb_table.event_store.arn
    ]
  }
}

// permissions for the lambda's role
resource "aws_iam_role_policy" "sqs_consumer" {
  role   = aws_iam_role.sqs_consumer.name
  policy = data.aws_iam_policy_document.sqs_consumer.json
}

// the lambda that will process events
resource "aws_lambda_function" "sqs_consumer" {
  filename         = "../dist/sqsConsumerLambda.zip"
  source_code_hash = filebase64sha256("../dist/sqsConsumerLambda.zip")
  timeout          = local.event_persistor_lambda_timeout_seconds
  runtime          = "provided.al2"
  handler          = "bootstrap"
  architectures    = ["arm64"]
  function_name    = "${local.workspace_prefix}GoLambdaDemoSQSConsumer"
  role             = aws_iam_role.sqs_consumer.arn

  tracing_config {
    mode = "Active"
  }

  environment {
    variables = {
      LOG_LEVEL   = "info"
      EVENT_TABLE = aws_dynamodb_table.event_store.name
    }
  }
}

// cloudwatch log group so lambda logs don't just go to /dev/null
resource "aws_cloudwatch_log_group" "sqs_consumer_logs" {
  name              = "/aws/lambda/${aws_lambda_function.sqs_consumer.function_name}"
  retention_in_days = 7
}

// now lets connect our queue to the lambda

// first lets grant SQS permission to invoke the lambda
resource "aws_lambda_permission" "sqs_consumer_sqs_invoke" {
  statement_id  = "AllowExecutionFromSQS"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.sqs_consumer.arn
  principal     = "events.amazonaws.com"
  source_arn    = aws_sqs_queue.events.arn
}

// and then the event source mapping
resource "aws_lambda_event_source_mapping" "sqs_consumer" {
  event_source_arn = aws_sqs_queue.events.arn
  function_name    = aws_lambda_function.sqs_consumer.arn
  // if you don't depend on the policy attachment for the role that gives the lambda SQS access its possible terraform
  // will try to create the event source mapping before the lambda has permissions & the apply will fail
  depends_on = [aws_iam_role_policy.sqs_consumer]
  batch_size = 5
  // allow batch item failure responses
  function_response_types = [
    "ReportBatchItemFailures"
  ]
}