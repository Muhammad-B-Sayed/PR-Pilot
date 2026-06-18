package review

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type Pipeline struct {
	llm    LLMClient
	logger *slog.Logger
}

func NewPipeline(llm LLMClient, logger *slog.Logger) *Pipeline {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pipeline{llm: llm, logger: logger}
}

func (p *Pipeline) Run(ctx context.Context, input Input) (Result, error) {
	start := time.Now()
	rawResponses := make([]string, 0, 5)

	summary, raw, err := generateJSON[SummaryResult](ctx, p.llm, SummaryPrompt(input))
	if err != nil {
		return Result{}, fmt.Errorf("summary stage: %w", err)
	}
	rawResponses = append(rawResponses, raw)

	risk, raw, err := generateJSON[RiskResult](ctx, p.llm, RiskPrompt(input, summary))
	if err != nil {
		return Result{}, fmt.Errorf("risk stage: %w", err)
	}
	risk.RiskLevel = NormalizeRiskLevel(risk.RiskLevel)
	rawResponses = append(rawResponses, raw)

	security, raw, err := generateJSON[SecurityResult](ctx, p.llm, SecurityPrompt(input))
	if err != nil {
		return Result{}, fmt.Errorf("security stage: %w", err)
	}
	rawResponses = append(rawResponses, raw)

	tests, raw, err := generateJSON[TestResult](ctx, p.llm, TestPrompt(input, summary, risk, security))
	if err != nil {
		return Result{}, fmt.Errorf("test suggestion stage: %w", err)
	}
	rawResponses = append(rawResponses, raw)

	final, raw, err := generateJSON[FinalResult](ctx, p.llm, FinalPrompt(input, summary, risk, security, tests))
	if err != nil {
		return Result{}, fmt.Errorf("final reviewer stage: %w", err)
	}
	rawResponses = append(rawResponses, raw)

	source := input.Source
	if source == "" {
		source = "api"
	}

	result := Result{
		Title:           input.Title,
		Repository:      input.Repository,
		PullRequestURL:  input.PullRequestURL,
		Source:          source,
		Summary:         summary.Summary,
		RiskLevel:       risk.RiskLevel,
		MainChanges:     summary.MainChanges,
		PotentialIssues: risk.PotentialIssues,
		SecurityNotes:   security.SecurityNotes,
		SuggestedTests:  tests.SuggestedTests,
		FinalComment:    final.FinalComment,
		RawResponses:    rawResponses,
	}

	p.logger.InfoContext(ctx, "review pipeline completed",
		"risk_level", result.RiskLevel,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return result, nil
}

func generateJSON[T any](ctx context.Context, llm LLMClient, prompt string) (T, string, error) {
	var zero T
	raw, err := llm.Generate(ctx, prompt)
	if err != nil {
		return zero, "", err
	}
	parsed, err := ParseJSON[T](raw)
	if err == nil {
		return parsed, raw, nil
	}

	correctedRaw, correctionErr := llm.Generate(ctx, CorrectionPrompt(prompt, raw, err.Error()))
	if correctionErr != nil {
		return zero, raw, correctionErr
	}
	parsed, err = ParseJSON[T](correctedRaw)
	if err != nil {
		return zero, correctedRaw, err
	}
	return parsed, correctedRaw, nil
}
