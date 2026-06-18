package storage

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoStore struct {
	client    *dynamodb.Client
	tableName string
	logger    *slog.Logger
}

func NewDynamoStore(client *dynamodb.Client, tableName string, logger *slog.Logger) *DynamoStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &DynamoStore{client: client, tableName: tableName, logger: logger}
}

func (s *DynamoStore) SaveReview(ctx context.Context, record ReviewRecord) error {
	start := time.Now()
	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return err
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "dynamodb save failed", "review_id", record.ReviewID, "error", err)
		return err
	}
	s.logger.InfoContext(ctx, "dynamodb save completed", "review_id", record.ReviewID, "duration_ms", time.Since(start).Milliseconds())
	return nil
}

func (s *DynamoStore) GetReview(ctx context.Context, reviewID string) (ReviewRecord, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"review_id": &types.AttributeValueMemberS{Value: reviewID},
		},
	})
	if err != nil {
		return ReviewRecord{}, err
	}
	if len(out.Item) == 0 {
		return ReviewRecord{}, ErrNotFound
	}
	var record ReviewRecord
	if err := attributevalue.UnmarshalMap(out.Item, &record); err != nil {
		return ReviewRecord{}, err
	}
	if record.ReviewID == "" {
		return ReviewRecord{}, errors.New("dynamodb item missing review_id")
	}
	return record, nil
}
