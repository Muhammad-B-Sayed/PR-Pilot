package bedrock

import (
	"context"
	"strings"
)

type MockClient struct{}

func (MockClient) Generate(ctx context.Context, prompt string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "summarizer agent"):
		return `{"summary":"This PR updates application code and supporting behavior based on the submitted diff.","main_changes":["Reviews the submitted diff","Highlights notable implementation changes"]}`, nil
	case strings.Contains(lower, "risk analysis agent"):
		return `{"risk_level":"medium","potential_issues":["Confirm error paths and edge cases are covered","Validate behavior with malformed or missing inputs"]}`, nil
	case strings.Contains(lower, "security review agent"):
		return `{"security_notes":["Check that secrets are not logged or committed","Verify authentication and authorization behavior around changed routes"]}`, nil
	case strings.Contains(lower, "test suggestion agent"):
		return `{"suggested_tests":["Add tests for valid and invalid inputs","Add regression tests for changed behavior","Add edge-case tests for error handling"]}`, nil
	case strings.Contains(lower, "final reviewer agent"):
		return `{"final_comment":"Nice direction overall. I would add focused tests around invalid inputs, error handling, and any auth-sensitive paths touched by this change before merging."}`, nil
	default:
		return `{"final_comment":"Review completed."}`, nil
	}
}
