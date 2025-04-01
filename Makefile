.DEFAULT_GOAL := build

dist/:
	mkdir dist

dist/sqsConsumerLambda.zip: dist/ $(shell find . -iname "*.go")
	./scripts/build-lambda.sh github.com/jonsabados/go-lambda-demo/cmd/sqs-consumer dist/sqsConsumerLambda.zip

build: dist/sqsConsumerLambda.zip