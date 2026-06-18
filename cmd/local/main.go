package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/muhammad/prpilot/internal/api"
	"github.com/muhammad/prpilot/internal/bedrock"
	"github.com/muhammad/prpilot/internal/config"
	prgithub "github.com/muhammad/prpilot/internal/github"
	"github.com/muhammad/prpilot/internal/review"
	"github.com/muhammad/prpilot/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()
	ctx := context.Background()

	llm := review.LLMClient(bedrock.MockClient{})
	store := storage.Store(storage.NewMemoryStore())

	if cfg.UseBedrock {
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
		if err != nil {
			logger.Error("failed to load aws config", "error", err)
			os.Exit(1)
		}
		llm = bedrock.NewClient(bedrockruntime.NewFromConfig(awsCfg), cfg.BedrockModelID, logger)
		if cfg.UseDynamoDB {
			store = storage.NewDynamoStore(dynamodb.NewFromConfig(awsCfg), cfg.DynamoDBTableName, logger)
		}
	}

	handler := &api.Handler{
		Pipeline:     review.NewPipeline(llm, logger),
		Store:        store,
		GitHub:       prgithub.NewClient(cfg.GitHubToken, cfg.GitHubWebhookSecret),
		Logger:       logger,
		MaxDiffChars: cfg.MaxDiffChars,
	}

	logger.Info("starting local server", "address", cfg.HTTPAddress)
	if err := http.ListenAndServe(cfg.HTTPAddress, handler.Routes()); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
