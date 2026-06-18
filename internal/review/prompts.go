package review

import (
	"encoding/json"
	"fmt"
	"strings"
)

const outputRules = `Return only valid compact JSON. Do not wrap the JSON in markdown. Do not invent facts that are not grounded in the diff.`

func diffContext(input Input) string {
	return fmt.Sprintf("PR title: %s\nRepository: %s\nPR URL: %s\nDescription: %s\n\nDiff:\n%s",
		emptyAsUnset(input.Title),
		emptyAsUnset(input.Repository),
		emptyAsUnset(input.PullRequestURL),
		emptyAsUnset(input.Description),
		input.Diff,
	)
}

func SummaryPrompt(input Input) string {
	return fmt.Sprintf(`You are the summarizer agent for PRPilot.

Summarize the pull request change in plain English and list the main changes.

Expected schema:
{"summary":"...","main_changes":["..."]}

%s

%s`, outputRules, diffContext(input))
}

func RiskPrompt(input Input, summary SummaryResult) string {
	return fmt.Sprintf(`You are the risk analysis agent for PRPilot.

Identify practical risks in this pull request. Look for missing edge cases, error handling gaps, breaking API changes, performance problems, concurrency problems, and weak assumptions.

Expected schema:
{"risk_level":"low|medium|high","potential_issues":["..."]}

Previous summary:
%s

%s

%s`, mustJSON(summary), outputRules, diffContext(input))
}

func SecurityPrompt(input Input) string {
	return fmt.Sprintf(`You are the security review agent for PRPilot.

Check for hardcoded secrets, missing authentication or authorization, unsafe token handling, injection risk, sensitive logging, insecure error messages, and weak input validation.

Expected schema:
{"security_notes":["..."]}

If there are no obvious security concerns, return an empty list.

%s

%s`, outputRules, diffContext(input))
}

func TestPrompt(input Input, summary SummaryResult, risk RiskResult, security SecurityResult) string {
	return fmt.Sprintf(`You are the test suggestion agent for PRPilot.

Suggest targeted unit, integration, and edge-case tests based on the diff and previous review stages.

Expected schema:
{"suggested_tests":["..."]}

Previous stages:
%s

%s

%s`, mustJSON(map[string]any{
		"summary":  summary,
		"risk":     risk,
		"security": security,
	}), outputRules, diffContext(input))
}

func FinalPrompt(input Input, summary SummaryResult, risk RiskResult, security SecurityResult, tests TestResult) string {
	return fmt.Sprintf(`You are the final reviewer agent for PRPilot.

Combine the previous stage outputs into one concise, professional pull request review comment. The tone should be helpful and similar to a teammate's review.

Expected schema:
{"final_comment":"..."}

Previous stages:
%s

%s

%s`, mustJSON(map[string]any{
		"summary":  summary,
		"risk":     risk,
		"security": security,
		"tests":    tests,
	}), outputRules, diffContext(input))
}

func CorrectionPrompt(originalPrompt, invalidResponse, parseError string) string {
	return fmt.Sprintf(`The previous model response could not be parsed as valid JSON.

Parse error: %s

Invalid response:
%s

Original task:
%s

Return only corrected JSON matching the requested schema.`, parseError, invalidResponse, originalPrompt)
}

func mustJSON(value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func emptyAsUnset(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(not provided)"
	}
	return value
}
