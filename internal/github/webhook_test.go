package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	secret := "top-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !VerifySignature(body, signature, secret) {
		t.Fatal("expected signature to verify")
	}
	if VerifySignature(body, signature, "wrong") {
		t.Fatal("expected wrong secret to fail")
	}
}

func TestPullRequestEventShouldReview(t *testing.T) {
	event, err := ParsePullRequestEvent([]byte(`{
		"action":"synchronize",
		"repository":{"full_name":"owner/repo"},
		"pull_request":{"title":"Update auth","diff_url":"https://example.com/diff"}
	}`))
	if err != nil {
		t.Fatalf("ParsePullRequestEvent returned error: %v", err)
	}
	if !event.ShouldReview() {
		t.Fatal("expected synchronize event to be reviewable")
	}
}

func TestParsePullRequestEventSupportsFormPayload(t *testing.T) {
	raw := `{"action":"opened","repository":{"full_name":"owner/repo"},"pull_request":{"title":"Update auth","diff_url":"https://example.com/diff"}}`
	form := url.Values{"payload": []string{raw}}.Encode()

	event, err := ParsePullRequestEvent([]byte(form))
	if err != nil {
		t.Fatalf("ParsePullRequestEvent returned error: %v", err)
	}
	if event.Action != "opened" {
		t.Fatalf("expected opened action, got %q", event.Action)
	}
	if !event.ShouldReview() {
		t.Fatal("expected form payload event to be reviewable")
	}
}
