package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/muhammad/prpilot/internal/bedrock"
	"github.com/muhammad/prpilot/internal/review"
	"github.com/muhammad/prpilot/internal/storage"
)

func TestReviewHTTPStoresReview(t *testing.T) {
	store := storage.NewMemoryStore()
	handler := &Handler{
		Pipeline:     review.NewPipeline(bedrock.MockClient{}, nil),
		Store:        store,
		MaxDiffChars: 50000,
		Clock:        func() time.Time { return time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC) },
	}
	body := bytes.NewBufferString(`{"title":"Add JWT auth","diff":"diff --git a/auth.go b/auth.go"}`)

	req := httptest.NewRequest(http.MethodPost, "/review", body)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var response storage.ReviewRecord
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.ReviewID == "" {
		t.Fatal("expected review id")
	}
	if response.RiskLevel != "medium" {
		t.Fatalf("expected medium risk, got %q", response.RiskLevel)
	}

	stored, err := store.GetReview(req.Context(), response.ReviewID)
	if err != nil {
		t.Fatalf("review was not stored: %v", err)
	}
	if stored.Title != "Add JWT auth" {
		t.Fatalf("unexpected stored title: %q", stored.Title)
	}
}
