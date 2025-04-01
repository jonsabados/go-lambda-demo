# go-lambda-demo

## What is this
This is an example serverless application deployed to AWS via terraform. It consists of a rest based application stuffed into a Lambda behind API Gateway with a single `/event` endpoint that takes a json payload with a message and a source attribute. These events are placed on a SQS queue, which is consumed by the [SQS Consumer Lambda](#sqs-consumer) which stores the event in [DynamoDB](https://aws.amazon.com/dynamodb/).

### Components

#### Stand Alone Version of the Rest API
One of the challenges of serverless development is it being generally difficult to do things like running the stack locally. However, since we are using a technique in which we stuff a traditional application inside a lambda for our HTTP ingress it means that we can also have a standalone version without too much effort. The [standalone version](cmd/standalone-api/main.go) provides this. In order to run it does need to know about the target SQS queue - there is a make target, `make run-rest-api`, which will source this value via terraform outputs & then start the application listening on port 8080.

#### API
There is an example of [running a traditional application within a lambda](cmd/lambda-based-api/main.go). This Lambda uses the AWS Labs [Go API Proxy](https://github.com/awslabs/aws-lambda-go-api-proxy), and is deployed behind API Gateway. The terraform defining all of this may be found [here](terraform/api.tf).

#### SQS Consumer
There is an example [SQS consumer lambda](cmd/sqs-consumer/main.go) which takes messages off of a queue and persists them in DynamoDB. The terraform defining the dynamo table, queue, dead letter queue, and lambda can be found [here](terraform/event-store.tf).

## Prerequisites
### Go
You will need go installed. This has been tested with `go1.24.1`, however any relatively recent version of go should suffice.

### make
A Makefile is used to build the project, as such you will need make installed on your machine. GN Make 3.81 is known to work.

### AWS account
You will need an AWS account, and should have your environment [configured](https://docs.aws.amazon.com/cli/latest/reference/configure/) so that the aws cli has access to this account. You may test this by executing a command like `aws s3 list`.

### Terraform
You will need a recent version of terraform. If you do not have terraform installed [tfenv](https://github.com/tfutils/tfenv) is a good way to go.

### S3 Bucket for Terraform State
While almost everything in this example is codified, one thing that cannot be is the place where state is stored. You will need to create this bucket, which can be done through the [AWS console](https://us-east-1.console.aws.amazon.com/s3/home?region=us-east-1) (note, link assumes you are deploying to us-east-1). This bucket must have a globally unique name, such as `<your-aws-account-number>-lambda-demo-tfstate`, and it should not be world readable. Bucket versioning is also a good thing to have enabled for your state bucket (this can come in handy if things go terribly wrong).

### Route53 Domain
Similar to the [state bucket](#s3-bucket-for-terraform-state) you must manually register a domain in Route53, if you do not have one readily available. You will need to specify the terraform variable `route53_domain` in whatever method you wish (edit the [vars.tf](terraform/vars.tf) file, set a TF_VAR_route53_domain env var etc). See the [terraform docs](https://developer.hashicorp.com/terraform/language/values/variables) for more info.

## Deploying
1) First execute `make build` from the root of the project, this will create binaries for the various lambdas to be deployed in dist/.
2) Next go to the `terraform` directory and execute `terraform init`. You will be prompted for the S3 bucket where state will be stored, enter the name of the bucket you [created](#s3-bucket-for-terraform-state). This is a one time operation.
3) Execute `terraform apply`
4) Send post requests to `https://go-lambda-demo.<route53 domain name here>` with a body like `{"message":"some message here", "source": "some source here"}`

### A note about workspaces
This project is setup to support terraform workspaces. You may deploy multiple instances of the application by executing `terraform workspace new <workspace_name>` and then doing a terraform apply. All resources created will have identifiers prefixed by the workspace name. This is useful for deploying multiple environments to the same AWS account. Once deployed you will be able to access your workspace via the url specified by the `api_url` output.