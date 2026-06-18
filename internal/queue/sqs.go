package queue

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type Enqueuer interface {
	Enqueue(ctx context.Context, body []byte) error
}

type SQSQueue struct {
	client   *sqs.Client
	queueURL string
	logger   *slog.Logger
}

func NewSQSQueue(client *sqs.Client, queueURL string, logger *slog.Logger) *SQSQueue {
	if logger == nil {
		logger = slog.Default()
	}
	return &SQSQueue{client: client, queueURL: queueURL, logger: logger}
}

func (q *SQSQueue) Enqueue(ctx context.Context, body []byte) error {
	_, err := q.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(q.queueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		q.logger.ErrorContext(ctx, "sqs enqueue failed", "error", err)
		return err
	}
	q.logger.InfoContext(ctx, "github webhook queued")
	return nil
}
