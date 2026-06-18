package review

import "testing"

func TestParseJSONTrimsMarkdownFence(t *testing.T) {
	got, err := ParseJSON[RiskResult]("```json\n{\"risk_level\":\"high\",\"potential_issues\":[\"missing validation\"]}\n```")
	if err != nil {
		t.Fatalf("ParseJSON returned error: %v", err)
	}
	if got.RiskLevel != "high" {
		t.Fatalf("expected high risk, got %q", got.RiskLevel)
	}
}

func TestNormalizeRiskLevelDefaultsUnknownValues(t *testing.T) {
	if got := NormalizeRiskLevel("critical"); got != "medium" {
		t.Fatalf("expected unknown risk to default to medium, got %q", got)
	}
}
