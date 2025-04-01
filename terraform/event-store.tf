// the dynamo table that will store our events
resource "aws_dynamodb_table" "event_store" {
  hash_key     = "ID"
  name         = "${local.workspace_prefix}GoLambdaDemoEvents"
  billing_mode = "PAY_PER_REQUEST"

  attribute {
    name = "ID"
    type = "S"
  }
}
