package github

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/url"
	"strings"
)

type PullRequestEvent struct {
	Action     string `json:"action"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest struct {
		Title            string `json:"title"`
		Body             string `json:"body"`
		HTMLURL          string `json:"html_url"`
		DiffURL          string `json:"diff_url"`
		IssueCommentsURL string `json:"comments_url"`
		Number           int    `json:"number"`
	} `json:"pull_request"`
}

func ReadWebhookBody(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, io.LimitReader(r, 1<<20)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ParsePullRequestEvent(body []byte) (PullRequestEvent, error) {
	body = normalizeWebhookBody(body)
	var event PullRequestEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return PullRequestEvent{}, err
	}
	return event, nil
}

func normalizeWebhookBody(body []byte) []byte {
	body = bytes.TrimSpace(body)
	if !bytes.HasPrefix(body, []byte("payload=")) {
		return body
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return body
	}
	payload := values.Get("payload")
	if payload == "" {
		return body
	}
	return []byte(payload)
}

func (e PullRequestEvent) ShouldReview() bool {
	switch e.Action {
	case "opened", "synchronize", "reopened":
		return e.PullRequest.DiffURL != ""
	default:
		return false
	}
}

func VerifySignature(body []byte, signatureHeader, secret string) bool {
	if secret == "" || signatureHeader == "" {
		return false
	}
	signatureHeader = strings.TrimPrefix(signatureHeader, "sha256=")
	expectedMAC := hmac.New(sha256.New, []byte(secret))
	expectedMAC.Write(body)
	expected := expectedMAC.Sum(nil)
	actual, err := hex.DecodeString(signatureHeader)
	if err != nil {
		return false
	}
	return hmac.Equal(actual, expected)
}
