package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/muhammad/prpilot/internal/api"
	"github.com/muhammad/prpilot/internal/bedrock"
	"github.com/muhammad/prpilot/internal/config"
	prgithub "github.com/muhammad/prpilot/internal/github"
	"github.com/muhammad/prpilot/internal/queue"
	"github.com/muhammad/prpilot/internal/review"
	"github.com/muhammad/prpilot/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		logger.Error("failed to load aws config", "error", err)
		os.Exit(1)
	}

	handler := &api.Handler{
		Pipeline: review.NewPipeline(
			bedrock.NewClient(bedrockruntime.NewFromConfig(awsCfg), cfg.BedrockModelID, logger),
			logger,
		),
		Store:        storage.NewDynamoStore(dynamodb.NewFromConfig(awsCfg), cfg.DynamoDBTableName, logger),
		GitHub:       prgithub.NewClient(cfg.GitHubToken, cfg.GitHubWebhookSecret),
		Logger:       logger,
		MaxDiffChars: cfg.MaxDiffChars,
	}
	if cfg.GitHubWebhookQueueURL != "" {
		handler.WebhookQueue = queue.NewSQSQueue(sqs.NewFromConfig(awsCfg), cfg.GitHubWebhookQueueURL, logger)
	}

	lambda.Start(func(ctx context.Context, raw json.RawMessage) (any, error) {
		return routeEvent(ctx, raw, handler, logger)
	})
}

func routeEvent(ctx context.Context, raw json.RawMessage, handler *api.Handler, logger *slog.Logger) (any, error) {
	var probe eventProbe
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	if len(probe.Records) > 0 && probe.Records[0].EventSource == "aws:sqs" {
		var event events.SQSEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			return nil, err
		}
		return handleSQSEvent(ctx, event, handler, logger)
	}
	if probe.RequestContext.HTTP.Method != "" {
		var req events.APIGatewayV2HTTPRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, err
		}
		return handler.HandleLambda(ctx, req)
	}
	return nil, errors.New("unsupported lambda event")
}

func handleSQSEvent(ctx context.Context, event events.SQSEvent, handler *api.Handler, logger *slog.Logger) (map[string]string, error) {
	for _, message := range event.Records {
		record, handled, err := handler.ProcessGitHubWebhook(ctx, []byte(message.Body))
		if err != nil {
			logger.ErrorContext(ctx, "sqs github webhook processing failed",
				"message_id", message.MessageId,
				"error", err,
			)
			return nil, err
		}
		if handled {
			logger.InfoContext(ctx, "sqs github webhook processed",
				"message_id", message.MessageId,
				"review_id", record.ReviewID,
				"repository", record.Repository,
			)
		}
	}
	return map[string]string{"status": "processed"}, nil
}

type eventProbe struct {
	Records []struct {
		EventSource string `json:"eventSource"`
	} `json:"Records"`
	RequestContext struct {
		HTTP struct {
			Method string `json:"method"`
		} `json:"http"`
	} `json:"requestContext"`
}
