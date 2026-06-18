package review

import (
	"context"
	"strings"
	"testing"
)

type scriptedLLM struct {
	responses []string
	calls     int
}

func (s *scriptedLLM) Generate(ctx context.Context, prompt string) (string, error) {
	if s.calls >= len(s.responses) {
		return "{}", nil
	}
	response := s.responses[s.calls]
	s.calls++
	return response, nil
}

func TestPipelineRetriesInvalidJSONOnce(t *testing.T) {
	llm := &scriptedLLM{responses: []string{
		"not json",
		`{"summary":"Adds auth middleware.","main_changes":["Adds middleware"]}`,
		`{"risk_level":"low","potential_issues":[]}`,
		`{"security_notes":[]}`,
		`{"suggested_tests":["Test missing token"]}`,
		`{"final_comment":"Looks good; add the missing-token test."}`,
	}}
	pipeline := NewPipeline(llm, nil)

	got, err := pipeline.Run(context.Background(), Input{
		Title: "Add auth middleware",
		Diff:  "diff --git a/auth.go b/auth.go",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if llm.calls != 6 {
		t.Fatalf("expected 6 LLM calls with one correction, got %d", llm.calls)
	}
	if !strings.Contains(got.FinalComment, "missing-token") {
		t.Fatalf("unexpected final comment: %q", got.FinalComment)
	}
}
