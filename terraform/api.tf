locals {
  api_host_name   = "${local.workspace_prefix}go-lambda-demo"
  api_domain_name = "${local.api_host_name}.${data.aws_route53_zone.route53_zone.name}"
}

// IAM role for our lambda
resource "aws_iam_role" "api_lambda" {
  name = "${local.workspace_prefix}GoLambdaDemoAPILambda"
  // this allows lambdas to assume this role
  assume_role_policy = data.aws_iam_policy_document.assume_lambda_role_policy.json
}

// policy document for the lambdas role
data "aws_iam_policy_document" "api_lambda" {
  statement {
    sid    = "AllowLogging"
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = [
      "${aws_cloudwatch_log_group.api_lambda_logs.arn}:*"
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
      "sqs:SendMessage"
    ]
    resources = [
      aws_sqs_queue.events.arn
    ]
  }
}

// permissions for the lambda's role
resource "aws_iam_role_policy" "api_lambda" {
  role   = aws_iam_role.api_lambda.name
  policy = data.aws_iam_policy_document.api_lambda.json
}

// the lambda that will process events
resource "aws_lambda_function" "api_lambda" {
  filename         = "../dist/apiLambda.zip"
  source_code_hash = filebase64sha256("../dist/apiLambda.zip")
  timeout          = 15
  // setting reserved concurrent executions super low cause personal account & don't want to make it too easy for someone to grief my wallet by pounding on things
  reserved_concurrent_executions = 2
  runtime                        = "provided.al2"
  handler                        = "bootstrap"
  architectures                  = ["arm64"]
  function_name                  = "${local.workspace_prefix}GoLambdaDemoApi"
  role                           = aws_iam_role.api_lambda.arn

  tracing_config {
    mode = "Active"
  }

  environment {
    variables = {
      LOG_LEVEL       = "info"
      EVENT_QUEUE_URL = aws_sqs_queue.events.url
    }
  }
}

// cloudwatch log group so lambda logs don't just go to /dev/null
resource "aws_cloudwatch_log_group" "api_lambda_logs" {
  name              = "/aws/lambda/${aws_lambda_function.api_lambda.function_name}"
  retention_in_days = 7
}

// now lets expose the lambda via API Gateway
data "aws_route53_zone" "route53_zone" {
  name = var.route53_domain
}

// ACM managed certificate
module "api_cert" {
  source  = "terraform-aws-modules/acm/aws"
  version = "~> 4.0"

  domain_name = local.api_domain_name
  zone_id     = data.aws_route53_zone.route53_zone.id

  validation_method = "DNS"

  wait_for_validation = true
}

// The API Gateway
resource "aws_api_gateway_rest_api" "api" {
  name = "${local.workspace_prefix}GoLambdaDemoApi"

  tags = {
    Workspace = terraform.workspace
  }
}

// Associate the API gateway to a DNS name
resource "aws_api_gateway_domain_name" "api" {
  domain_name     = local.api_domain_name
  certificate_arn = module.api_cert.acm_certificate_arn
}

// register the DNS name
resource "aws_route53_record" "api" {
  name    = local.api_host_name
  type    = "CNAME"
  zone_id = data.aws_route53_zone.route53_zone.id
  records = [aws_api_gateway_domain_name.api.cloudfront_domain_name]
  ttl     = 300
}

// API endpoint
// First we have our resource (path)
resource "aws_api_gateway_resource" "event" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "event"
}

// now the POST method for our event resource
resource "aws_api_gateway_method" "event_post" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.event.id
  http_method   = "POST"
  authorization = "NONE"
}

// permissions for API gateway to invoke our lambda
resource "aws_lambda_permission" "api_gateway_api_lambda" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.api_lambda.function_name
  principal     = "apigateway.amazonaws.com"

  source_arn = "arn:aws:execute-api:us-east-1:${data.aws_caller_identity.current.account_id}:${aws_api_gateway_rest_api.api.id}/*/POST/${aws_api_gateway_resource.event.path_part}"
}

// and finally connect our lambda to the endpoint
resource "aws_api_gateway_integration" "event_post" {
  rest_api_id             = aws_api_gateway_rest_api.api.id
  resource_id             = aws_api_gateway_resource.event.id
  http_method             = aws_api_gateway_method.event_post.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.api_lambda.invoke_arn
}

// and then fun with API gateway stage hell
resource "aws_api_gateway_deployment" "api" {
  // note, you need to depend on all of your integrations
  depends_on = [
    aws_api_gateway_integration.event_post,
  ]
  rest_api_id = aws_api_gateway_rest_api.api.id

  // just redeploy the deployment everytime we run terraform. Easier than having to fuss with tainting things if we add new endpoints.
  variables = {
    "deployed_at" : timestamp()
  }

  // this is just needed... theres a reason, but its one of those things I just accept and move on rather than think to hard.
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_api_gateway_stage" "api" {
  deployment_id = aws_api_gateway_deployment.api.id
  rest_api_id   = aws_api_gateway_rest_api.api.id
  stage_name    = "${local.workspace_prefix}go-api-demo-main"
}

// and finally, connect our DNS mapping to our stage
resource "aws_api_gateway_base_path_mapping" "test" {
  api_id      = aws_api_gateway_rest_api.api.id
  stage_name  = aws_api_gateway_stage.api.stage_name
  domain_name = aws_api_gateway_domain_name.api.domain_name
}

output "api_url" {
  value = "https://${local.api_domain_name}"
}