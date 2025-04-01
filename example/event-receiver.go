package example

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/jonsabados/go-lambda-demo/correlation"
)

const CorrelationIDAttribute = "correlationID"

type IDGenerator func() string

type SQSClient interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type EventReceiverOption func(er *EventReceiver)

func WithIDGenerator(idGenerator IDGenerator) EventReceiverOption {
	return func(er *EventReceiver) {
		er.idGenerator = idGenerator
	}
}

type EventReceiver struct {
	client         SQSClient
	targetQueueURL string
	idGenerator    IDGenerator
}

// NewEventReceiver creates an EventReceiver instance. By default, it will use github.com/google/uuid.NewString() to generate
// event IDs, however this may be overridden by providing a WithIDGenerator option.
func NewEventReceiver(client SQSClient, targetQueueURL string, options ...EventReceiverOption) *EventReceiver {
	ret := &EventReceiver{
		client:         client,
		targetQueueURL: targetQueueURL,
		idGenerator:    uuid.NewString,
	}
	for _, o := range options {
		o(ret)
	}
	return ret
}

func (e *EventReceiver) ReceiveInboundEvent(ctx context.Context, source, message string) error {
	toPublish := SomeDomainEvent{
		ID:      e.idGenerator(),
		Message: message,
		Source:  source,
	}
	payload, err := json.Marshal(toPublish)
	if err != nil {
		return fmt.Errorf("error marshalling event: %w", err)
	}
	res, err := e.client.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody:  aws.String(string(payload)),
		QueueUrl:     aws.String(e.targetQueueURL),
		DelaySeconds: 0,
		MessageAttributes: map[string]types.MessageAttributeValue{
			CorrelationIDAttribute: {
				DataType:    aws.String("String"),
				StringValue: aws.String(correlation.FromContext(ctx)),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error publishing event: %w", err)
	}
	zerolog.Ctx(ctx).Trace().Interface("event", toPublish).Interface("result", res).Msg("published event")
	return nil
}
