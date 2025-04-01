.DEFAULT_GOAL := build

dist/:
	mkdir dist

dist/sqsConsumerLambda.zip: dist/ $(shell find . -iname "*.go")
	./scripts/build-lambda.sh github.com/jonsabados/go-lambda-demo/cmd/sqs-consumer dist/sqsConsumerLambda.zip

dist/apiLambda.zip: dist/ $(shell find . -iname "*.go")
	./scripts/build-lambda.sh github.com/jonsabados/go-lambda-demo/cmd/lambda-based-api dist/apiLambda.zip

build: dist/sqsConsumerLambda.zip dist/apiLambda.zip

.PHONY: run-rest-api
run-rest-api:
	LOG_LEVEL=trace EVENT_QUEUE_URL=`terraform -chdir=terraform/ output -raw event_queue_url` go run github.com/jonsabados/go-lambda-demo/cmd/standalone-api