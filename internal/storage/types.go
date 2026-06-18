package storage

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("review not found")

type Store interface {
	SaveReview(ctx context.Context, record ReviewRecord) error
	GetReview(ctx context.Context, reviewID string) (ReviewRecord, error)
}

type ReviewRecord struct {
	ReviewID         string    `dynamodbav:"review_id" json:"review_id"`
	CreatedAt        time.Time `dynamodbav:"created_at" json:"created_at"`
	Source           string    `dynamodbav:"source" json:"source"`
	Repository       string    `dynamodbav:"repository" json:"repository,omitempty"`
	PullRequestURL   string    `dynamodbav:"pull_request_url" json:"pull_request_url,omitempty"`
	Title            string    `dynamodbav:"title" json:"title"`
	RiskLevel        string    `dynamodbav:"risk_level" json:"risk_level"`
	Summary          string    `dynamodbav:"summary" json:"summary"`
	MainChanges      []string  `dynamodbav:"main_changes" json:"main_changes"`
	PotentialIssues  []string  `dynamodbav:"potential_issues" json:"potential_issues"`
	SecurityNotes    []string  `dynamodbav:"security_notes" json:"security_notes"`
	SuggestedTests   []string  `dynamodbav:"suggested_tests" json:"suggested_tests"`
	FinalComment     string    `dynamodbav:"final_comment" json:"final_comment"`
	RawModelResponse []string  `dynamodbav:"raw_model_response" json:"raw_model_response,omitempty"`
}
