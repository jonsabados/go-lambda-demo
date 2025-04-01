locals {
  workspace_prefix = terraform.workspace == "default" ? "" : "${terraform.workspace}-"
}