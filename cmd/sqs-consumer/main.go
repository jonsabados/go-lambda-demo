package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/v2/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"

	"github.com/jonsabados/go-lambda-demo/correlation"
	"github.com/jonsabados/go-lambda-demo/example"
)

type EventStore interface {
	PersistEvent(ctx context.Context, event example.SomeDomainEvent) error
}

type lambdaConfig struct {
	LogLevel   string `envconfig:"LOG_LEVEL" required:"true"`
	EventTable string `envconfig:"EVENT_TABLE" required:"true"`
}

func main() {
	ctx := context.Background()
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.LevelFieldName = "severity"
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	logger.Info().Msg("starting lambda instance")

	// load our config from environmental variables
	var cfg lambdaConfig
	err := envconfig.Process("", &cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("error loading config")
	}

	// set the log level per our config
	logLevel, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		logger.Fatal().Str("input", cfg.LogLevel).Err(err).Msg("error parsing log level")
	}
	logger = logger.Level(logLevel)

	// initialize x-ray
	err = xray.Configure(xray.Config{
		LogLevel: "warn",
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("error configuring x-ray")
	}

	// get x-ray goodness going with the http client we will be using
	httpClient := xray.Client(http.DefaultClient)

	// get our AWS environment setup
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithHTTPClient(httpClient))
	if err != nil {
		logger.Fatal().Err(err).Msg("error loading default config")
	}

	// add x-ray instrumentation to all the AWS clients
	awsv2.AWSV2Instrumentor(&awsCfg.APIOptions)

	dynamoClient := dynamodb.NewFromConfig(awsCfg)
	eventStore := example.NewDynamoStore(dynamoClient, cfg.EventTable)

	lambda.Start(NewHandler(logger, eventStore))
}

// NewHandler is a higher order function allowing us to do dependency injection for a lambda event handler. The handler itself
// is responsible for knowing about SQS and converting the inbound SQS request into domain specific constructs & then handing
// those off to proper business logic.
// Note: we are taking an interface to our eventStore because this handler is the kind of thing that should have unit tests using mocks on it,
// if it were not just a demo app (or if the author had a bit more time to whip up this demo).
func NewHandler(logger zerolog.Logger, eventStore EventStore) func(ctx context.Context, request events.SQSEvent) (events.SQSEventResponse, error) {
	return func(ctx context.Context, request events.SQSEvent) (events.SQSEventResponse, error) {
		// attach our logger to the context so anything that has a handle on the context can log stuff
		ctx = logger.WithContext(ctx)

		zerolog.Ctx(ctx).Debug().Interface("request", request).Msg("processing inbound request")
		ret := events.SQSEventResponse{}
		// SQS sends messages to lambdas in batches, so lets process each one (note, this is configurable, and you may set a batch size of 1 but the request still comes in as an array)
		for _, ev := range request.Records {
			// get the correlation id going on logs for the message
			ctx := ctx
			if correlationIDAttr, ok := ev.MessageAttributes[example.CorrelationIDAttribute]; ok {
				if correlationIDAttr.StringValue != nil {
					ctx = correlation.WithContext(ctx, *correlationIDAttr.StringValue)
				}
			}
			// excessive, but for demoing log correlation over SQS
			zerolog.Ctx(ctx).Info().Msg("processing event")
			// process the event in its own sub-trace - we are throwing the error on the floor because it is not needed after it has been returned to the xray.Capture call.
			// However, we could also kick back an error from the handler function to fail the entire batch (opposed to adding failed entries to our response)
			_ = xray.Capture(ctx, "processEvent", func(ctx context.Context) error {
				var event example.SomeDomainEvent
				err := json.Unmarshal([]byte(ev.Body), &event)
				if err != nil {
					zerolog.Ctx(ctx).Err(err).Interface("sqs event", ev).Msg("error unmarshalling event")
					ret.BatchItemFailures = append(ret.BatchItemFailures, events.SQSBatchItemFailure{
						ItemIdentifier: ev.MessageId,
					})
					return err
				}
				// delegate the rest of the processing to our "business logic" (in this case just persisting it)
				err = eventStore.PersistEvent(ctx, event)
				if err != nil {
					zerolog.Ctx(ctx).Err(err).Interface("event", event).Msg("error persisting event")
					ret.BatchItemFailures = append(ret.BatchItemFailures, events.SQSBatchItemFailure{
						ItemIdentifier: ev.MessageId,
					})
					return err
				}
				return nil
			})
		}

		zerolog.Ctx(ctx).Trace().Interface("response", ret).Msg("returning response")
		return ret, nil
	}
}
