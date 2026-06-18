package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type Client struct {
	runtime *bedrockruntime.Client
	modelID string
	logger  *slog.Logger
}

func NewClient(runtime *bedrockruntime.Client, modelID string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{runtime: runtime, modelID: modelID, logger: logger}
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	start := time.Now()
	body, err := json.Marshal(anthropicRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        1200,
		Temperature:      0.1,
		Messages: []anthropicMessage{
			{
				Role: "user",
				Content: []anthropicContent{
					{Type: "text", Text: prompt},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	out, err := c.runtime.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(c.modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		c.logger.ErrorContext(ctx, "bedrock invoke failed", "duration_ms", time.Since(start).Milliseconds(), "error", err)
		return "", err
	}

	var response anthropicResponse
	if err := json.Unmarshal(out.Body, &response); err != nil {
		return "", err
	}
	if len(response.Content) == 0 {
		return "", fmt.Errorf("bedrock response had no content")
	}

	c.logger.InfoContext(ctx, "bedrock invoke completed", "duration_ms", time.Since(start).Milliseconds())
	return response.Content[0].Text, nil
}

type anthropicRequest struct {
	AnthropicVersion string             `json:"anthropic_version"`
	MaxTokens        int                `json:"max_tokens"`
	Temperature      float64            `json:"temperature"`
	Messages         []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicResponse struct {
	Content []anthropicContent `json:"content"`
}
