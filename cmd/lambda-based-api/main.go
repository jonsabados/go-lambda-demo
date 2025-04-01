package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"

	"github.com/jonsabados/go-lambda-demo/cmd"
)

func main() {
	handler := cmd.CreateAPI()
	lambda.Start(httpadapter.New(handler).ProxyWithContext)
}
