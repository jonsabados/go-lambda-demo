data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "assume_lambda_role_policy" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = [
        "lambda.amazonaws.com"
      ]

      type = "Service"
    }
    effect = "Allow"
    sid    = "AllowLambdaAssumeRole"
  }
}