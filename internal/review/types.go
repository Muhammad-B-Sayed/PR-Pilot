package review

import "time"

type Input struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	Diff           string `json:"diff"`
	Repository     string `json:"repository,omitempty"`
	PullRequestURL string `json:"pull_request_url,omitempty"`
	Source         string `json:"source,omitempty"`
}

type Result struct {
	ReviewID        string    `json:"review_id"`
	CreatedAt       time.Time `json:"created_at"`
	Title           string    `json:"title"`
	Repository      string    `json:"repository,omitempty"`
	PullRequestURL  string    `json:"pull_request_url,omitempty"`
	Source          string    `json:"source"`
	Summary         string    `json:"summary"`
	RiskLevel       string    `json:"risk_level"`
	MainChanges     []string  `json:"main_changes"`
	PotentialIssues []string  `json:"potential_issues"`
	SecurityNotes   []string  `json:"security_notes"`
	SuggestedTests  []string  `json:"suggested_tests"`
	FinalComment    string    `json:"final_comment"`
	RawResponses    []string  `json:"-"`
}

type SummaryResult struct {
	Summary     string   `json:"summary"`
	MainChanges []string `json:"main_changes"`
}

type RiskResult struct {
	RiskLevel       string   `json:"risk_level"`
	PotentialIssues []string `json:"potential_issues"`
}

type SecurityResult struct {
	SecurityNotes []string `json:"security_notes"`
}

type TestResult struct {
	SuggestedTests []string `json:"suggested_tests"`
}

type FinalResult struct {
	FinalComment string `json:"final_comment"`
}

func NormalizeRiskLevel(level string) string {
	switch level {
	case "low", "medium", "high":
		return level
	default:
		return "medium"
	}
}
