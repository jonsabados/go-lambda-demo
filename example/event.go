package example

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/rs/zerolog"
)

type SomeDomainEvent struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Source  string `json:"source"`
}

type DynamoClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

type DynamoStore struct {
	client      DynamoClient
	targetTable string
}

func NewDynamoStore(client *dynamodb.Client, targetTable string) *DynamoStore {
	return &DynamoStore{client: client, targetTable: targetTable}
}

func (d *DynamoStore) PersistEvent(ctx context.Context, event SomeDomainEvent) error {
	av, err := attributevalue.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("error marshalling event: %w", err)
	}
	res, err := d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.targetTable),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("error persisting event: %w", err)
	}
	zerolog.Ctx(ctx).Trace().Interface("event", event).Interface("result", res).Msg("persisted event")
	return nil
}
