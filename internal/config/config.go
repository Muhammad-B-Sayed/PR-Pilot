package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AWSRegion             string
	BedrockModelID        string
	DynamoDBTableName     string
	GitHubToken           string
	GitHubWebhookSecret   string
	GitHubWebhookQueueURL string
	UseBedrock            bool
	UseDynamoDB           bool
	HTTPAddress           string
	MaxDiffChars          int
	RequestTimeout        time.Duration
}

func Load() Config {
	return Config{
		AWSRegion:             env("AWS_REGION", "us-east-1"),
		BedrockModelID:        env("BEDROCK_MODEL_ID", "us.anthropic.claude-haiku-4-5-20251001-v1:0"),
		DynamoDBTableName:     env("DYNAMODB_TABLE_NAME", "prpilot_reviews"),
		GitHubToken:           os.Getenv("GITHUB_TOKEN"),
		GitHubWebhookSecret:   os.Getenv("GITHUB_WEBHOOK_SECRET"),
		GitHubWebhookQueueURL: os.Getenv("GITHUB_WEBHOOK_QUEUE_URL"),
		UseBedrock:            envBool("USE_BEDROCK", false),
		UseDynamoDB:           envBool("USE_DYNAMODB", false),
		HTTPAddress:           env("HTTP_ADDRESS", ":8080"),
		MaxDiffChars:          envInt("MAX_DIFF_CHARS", 50000),
		RequestTimeout:        time.Duration(envInt("REQUEST_TIMEOUT_SECONDS", 30)) * time.Second,
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
