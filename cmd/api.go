package cmd

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-xray-sdk-go/v2/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"

	"github.com/jonsabados/go-lambda-demo/api"
	"github.com/jonsabados/go-lambda-demo/example"
)

type appCfg struct {
	LogLevel      string `envconfig:"LOG_LEVEL" required:"true"`
	EventQueueURL string `envconfig:"EVENT_QUEUE_URL" required:"true"`
}

// CreateAPI does all the DI needed to create the rest handler - putting in its own place so both the lambda based version of the app,
// and the standalone version don't have to repeat each other
func CreateAPI() http.Handler {
	ctx := context.Background()
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.LevelFieldName = "severity"
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	logger.Info().Msg("starting rest API")

	// load our config from environmental variables
	var cfg appCfg
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

	sqsClient := sqs.NewFromConfig(awsCfg)
	eventReceiver := example.NewEventReceiver(sqsClient, cfg.EventQueueURL)
	eventEndpoint := api.NewEventReceiverEndpoint(eventReceiver)
	return api.NewRestAPI(logger, uuid.NewString, eventEndpoint)
}
