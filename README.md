# go-lambda-demo

## What is this
This is an example serverless application deployed to AWS via terraform.

### Components

#### SQS Consumer
There is an example [SQS consumer lambda](cmd/sqs-consumer) which takes messages off of a queue and persists them in DynamoDB. The terraform defining the dynamo table, queue, dead letter queue, and lambda can be found [here](terraform/event-store.tf)

## Prerequisites
### AWS account
You will need an AWS account, and should have your environment [configured](https://docs.aws.amazon.com/cli/latest/reference/configure/) so that the aws cli has access to this account. You may test this by executing a command like `aws s3 list`.

### Terraform
You will need a recent version of terraform. If you do not have terraform installed [tfenv](https://github.com/tfutils/tfenv) is a good way to go.

### S3 Bucket for Terraform State
While almost everything in this example is codified, one thing that cannot be is the place where state is stored. You will need to create this bucket, which can be done through the [AWS console](https://us-east-1.console.aws.amazon.com/s3/home?region=us-east-1) (note, link assumes you are deploying to us-east-1). This bucket must have a globally unique name, such as `<your-aws-account-number>-lambda-demo-tfstate`, and it should not be world readable. Bucket versioning is also a good thing to have enabled for your state bucket (this can come in handy if things go terribly wrong).

## Deploying
1) First execute `make build` from the root of the project, this will create binaries for the various lambdas to be deployed in dist/.
2) Next go to the `terraform` directory and execute `terraform init`. You will be prompted for the S3 bucket where state will be stored, enter the name of the bucket you [created](#s3-bucket-for-terraform-state). This is a one time operation.
3) Execute `terraform apply`
4) Profit!

### A note about workspaces
This project is setup to support terraform workspaces. You may deploy multiple instances of the application by executing `terraform workspace new <workspace_name>` and then doing a terraform apply. All resources created will have identifiers prefixed by the workspace name. This is useful for deploying multiple environments to the same AWS account.