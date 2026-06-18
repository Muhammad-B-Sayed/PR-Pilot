package api

import (
	"errors"
	"strings"
)

type ReviewRequest struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	Diff           string `json:"diff"`
	Repository     string `json:"repository,omitempty"`
	PullRequestURL string `json:"pull_request_url,omitempty"`
}

func (r ReviewRequest) Validate(maxDiffChars int) error {
	if strings.TrimSpace(r.Title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(r.Diff) == "" {
		return errors.New("diff is required")
	}
	if len(r.Diff) > maxDiffChars {
		return errors.New("diff exceeds maximum size")
	}
	return nil
}

type ErrorResponse struct {
	Error string `json:"error"`
}
