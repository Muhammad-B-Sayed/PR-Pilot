package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	prgithub "github.com/muhammad/prpilot/internal/github"
	"github.com/muhammad/prpilot/internal/queue"
	"github.com/muhammad/prpilot/internal/review"
	"github.com/muhammad/prpilot/internal/storage"
)

type Handler struct {
	Pipeline     *review.Pipeline
	Store        storage.Store
	GitHub       *prgithub.Client
	WebhookQueue queue.Enqueuer
	Logger       *slog.Logger
	MaxDiffChars int
	Clock        func() time.Time
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.HealthHTTP)
	mux.HandleFunc("POST /review", h.ReviewHTTP)
	mux.HandleFunc("GET /reviews/{review_id}", h.GetReviewHTTP)
	mux.HandleFunc("POST /webhook/github", h.GitHubWebhookHTTP)
	return requestLogger(h.logger(), mux)
}

func (h *Handler) HealthHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "prpilot"})
}

func (h *Handler) ReviewHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req ReviewRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, int64(h.maxDiffChars()+4096))).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request body")
		return
	}
	result, err := h.ProcessReview(r.Context(), req, "api")
	if err != nil {
		status := http.StatusInternalServerError
		if isValidationError(err) {
			status = http.StatusBadRequest
		}
		h.logger().ErrorContext(r.Context(), "review request failed", "error", err)
		writeError(w, status, safeError(err, status))
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) GetReviewHTTP(w http.ResponseWriter, r *http.Request) {
	reviewID := r.PathValue("review_id")
	record, err := h.Store.GetReview(r.Context(), reviewID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "review not found")
			return
		}
		h.logger().ErrorContext(r.Context(), "get review failed", "review_id", reviewID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch review")
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (h *Handler) GitHubWebhookHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := prgithub.ReadWebhookBody(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook body")
		return
	}
	if h.GitHub != nil && h.GitHub.WebhookSecret != "" {
		if !prgithub.VerifySignature(body, r.Header.Get("X-Hub-Signature-256"), h.GitHub.WebhookSecret) {
			writeError(w, http.StatusUnauthorized, "invalid webhook signature")
			return
		}
	}
	if h.WebhookQueue != nil {
		if handled, err := h.QueueGitHubWebhook(r.Context(), body); err != nil {
			h.logger().ErrorContext(r.Context(), "github webhook enqueue failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to queue github webhook")
			return
		} else if !handled {
			writeJSON(w, http.StatusAccepted, map[string]string{"status": "ignored"})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
		return
	}

	result, handled, err := h.ProcessGitHubWebhook(r.Context(), body)
	if err != nil {
		h.logger().ErrorContext(r.Context(), "github webhook failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to process github webhook")
		return
	}
	if !handled {
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "ignored"})
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) HandleLambda(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	body, err := lambdaBody(req)
	if err != nil {
		return lambdaJSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
	}
	path := lambdaPath(req)

	switch {
	case req.RequestContext.HTTP.Method == http.MethodGet && path == "/health":
		return lambdaJSON(http.StatusOK, map[string]string{"status": "ok", "service": "prpilot"})
	case req.RequestContext.HTTP.Method == http.MethodPost && path == "/review":
		var reviewReq ReviewRequest
		if err := json.Unmarshal(body, &reviewReq); err != nil {
			return lambdaJSON(http.StatusBadRequest, ErrorResponse{Error: "invalid JSON request body"})
		}
		result, err := h.ProcessReview(ctx, reviewReq, "api")
		if err != nil {
			status := http.StatusInternalServerError
			if isValidationError(err) {
				status = http.StatusBadRequest
			}
			return lambdaJSON(status, ErrorResponse{Error: safeError(err, status)})
		}
		return lambdaJSON(http.StatusCreated, result)
	case req.RequestContext.HTTP.Method == http.MethodGet && strings.HasPrefix(path, "/reviews/"):
		reviewID := strings.TrimPrefix(path, "/reviews/")
		record, err := h.Store.GetReview(ctx, reviewID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return lambdaJSON(http.StatusNotFound, ErrorResponse{Error: "review not found"})
			}
			return lambdaJSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch review"})
		}
		return lambdaJSON(http.StatusOK, record)
	case req.RequestContext.HTTP.Method == http.MethodPost && path == "/webhook/github":
		if h.GitHub != nil && h.GitHub.WebhookSecret != "" {
			if !prgithub.VerifySignature(body, lambdaHeader(req.Headers, "x-hub-signature-256"), h.GitHub.WebhookSecret) {
				h.logger().WarnContext(ctx, "github webhook signature validation failed",
					"github_event", lambdaHeader(req.Headers, "x-github-event"),
					"github_delivery", lambdaHeader(req.Headers, "x-github-delivery"),
				)
				return lambdaJSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid webhook signature"})
			}
		}
		if h.WebhookQueue != nil {
			handled, err := h.QueueGitHubWebhook(ctx, body)
			if err != nil {
				h.logger().ErrorContext(ctx, "github webhook enqueue failed",
					"github_event", lambdaHeader(req.Headers, "x-github-event"),
					"github_delivery", lambdaHeader(req.Headers, "x-github-delivery"),
					"error", err,
				)
				return lambdaJSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to queue github webhook"})
			}
			if !handled {
				return lambdaJSON(http.StatusAccepted, map[string]string{"status": "ignored"})
			}
			return lambdaJSON(http.StatusAccepted, map[string]string{"status": "queued"})
		}
		result, handled, err := h.ProcessGitHubWebhook(ctx, body)
		if err != nil {
			h.logger().ErrorContext(ctx, "github webhook failed",
				"github_event", lambdaHeader(req.Headers, "x-github-event"),
				"github_delivery", lambdaHeader(req.Headers, "x-github-delivery"),
				"error", err,
			)
			return lambdaJSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to process github webhook"})
		}
		if !handled {
			return lambdaJSON(http.StatusAccepted, map[string]string{"status": "ignored"})
		}
		return lambdaJSON(http.StatusCreated, result)
	default:
		return lambdaJSON(http.StatusNotFound, ErrorResponse{Error: "route not found"})
	}
}

func (h *Handler) QueueGitHubWebhook(ctx context.Context, body []byte) (bool, error) {
	event, err := prgithub.ParsePullRequestEvent(body)
	if err != nil {
		return false, err
	}
	h.logger().InfoContext(ctx, "github webhook parsed",
		"action", event.Action,
		"repository", event.Repository.FullName,
		"pull_request_number", event.PullRequest.Number,
		"diff_url_set", event.PullRequest.DiffURL != "",
	)
	if !event.ShouldReview() {
		h.logger().InfoContext(ctx, "github webhook ignored",
			"action", event.Action,
			"repository", event.Repository.FullName,
			"pull_request_number", event.PullRequest.Number,
		)
		return false, nil
	}
	if h.WebhookQueue == nil {
		return false, errors.New("github webhook queue is not configured")
	}
	if err := h.WebhookQueue.Enqueue(ctx, body); err != nil {
		return false, err
	}
	return true, nil
}

func (h *Handler) ProcessReview(ctx context.Context, req ReviewRequest, source string) (storage.ReviewRecord, error) {
	if err := req.Validate(h.maxDiffChars()); err != nil {
		return storage.ReviewRecord{}, err
	}

	result, err := h.Pipeline.Run(ctx, review.Input{
		Title:          req.Title,
		Description:    req.Description,
		Diff:           req.Diff,
		Repository:     req.Repository,
		PullRequestURL: req.PullRequestURL,
		Source:         source,
	})
	if err != nil {
		return storage.ReviewRecord{}, err
	}

	now := h.now().UTC()
	result.ReviewID = newReviewID()
	result.CreatedAt = now

	record := storage.ReviewRecord{
		ReviewID:         result.ReviewID,
		CreatedAt:        result.CreatedAt,
		Source:           result.Source,
		Repository:       result.Repository,
		PullRequestURL:   result.PullRequestURL,
		Title:            result.Title,
		RiskLevel:        result.RiskLevel,
		Summary:          result.Summary,
		MainChanges:      result.MainChanges,
		PotentialIssues:  result.PotentialIssues,
		SecurityNotes:    result.SecurityNotes,
		SuggestedTests:   result.SuggestedTests,
		FinalComment:     result.FinalComment,
		RawModelResponse: result.RawResponses,
	}
	if err := h.Store.SaveReview(ctx, record); err != nil {
		return storage.ReviewRecord{}, err
	}
	h.logger().InfoContext(ctx, "review stored", "review_id", record.ReviewID, "risk_level", record.RiskLevel)
	return record, nil
}

func (h *Handler) ProcessGitHubWebhook(ctx context.Context, body []byte) (storage.ReviewRecord, bool, error) {
	event, err := prgithub.ParsePullRequestEvent(body)
	if err != nil {
		return storage.ReviewRecord{}, false, err
	}
	h.logger().InfoContext(ctx, "github webhook parsed",
		"action", event.Action,
		"repository", event.Repository.FullName,
		"pull_request_number", event.PullRequest.Number,
		"diff_url_set", event.PullRequest.DiffURL != "",
	)
	if !event.ShouldReview() {
		h.logger().InfoContext(ctx, "github webhook ignored",
			"action", event.Action,
			"repository", event.Repository.FullName,
			"pull_request_number", event.PullRequest.Number,
		)
		return storage.ReviewRecord{}, false, nil
	}
	if h.GitHub == nil {
		return storage.ReviewRecord{}, false, errors.New("github client is not configured")
	}
	diff, err := h.GitHub.FetchDiff(ctx, event.PullRequest.DiffURL)
	if err != nil {
		return storage.ReviewRecord{}, false, err
	}
	h.logger().InfoContext(ctx, "github diff fetched",
		"repository", event.Repository.FullName,
		"pull_request_number", event.PullRequest.Number,
		"diff_chars", len(diff),
	)
	record, err := h.ProcessReview(ctx, ReviewRequest{
		Title:          event.PullRequest.Title,
		Description:    event.PullRequest.Body,
		Diff:           diff,
		Repository:     event.Repository.FullName,
		PullRequestURL: event.PullRequest.HTMLURL,
	}, "github_webhook")
	if err != nil {
		return storage.ReviewRecord{}, false, err
	}
	if h.GitHub.CanPostComments() && event.PullRequest.IssueCommentsURL != "" {
		if err := h.GitHub.PostComment(ctx, event.PullRequest.IssueCommentsURL, record.FinalComment); err != nil {
			h.logger().ErrorContext(ctx, "failed to post github comment", "review_id", record.ReviewID, "error", err)
		}
	}
	return record, true, nil
}

func (h *Handler) maxDiffChars() int {
	if h.MaxDiffChars <= 0 {
		return 50000
	}
	return h.MaxDiffChars
}

func (h *Handler) now() time.Time {
	if h.Clock != nil {
		return h.Clock()
	}
	return time.Now()
}

func (h *Handler) logger() *slog.Logger {
	if h.Logger != nil {
		return h.Logger
	}
	return slog.Default()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

func lambdaJSON(status int, value any) (events.APIGatewayV2HTTPResponse, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusInternalServerError, Body: `{"error":"failed to encode response"}`}, nil
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       string(body),
	}, nil
}

func lambdaBody(req events.APIGatewayV2HTTPRequest) ([]byte, error) {
	if req.IsBase64Encoded {
		return base64.StdEncoding.DecodeString(req.Body)
	}
	return []byte(req.Body), nil
}

func lambdaPath(req events.APIGatewayV2HTTPRequest) string {
	path := req.RawPath
	stage := req.RequestContext.Stage
	if stage != "" && stage != "$default" {
		path = strings.TrimPrefix(path, "/"+stage)
	}
	if path == "" {
		return "/"
	}
	return path
}

func lambdaHeader(headers map[string]string, key string) string {
	for headerKey, value := range headers {
		if strings.EqualFold(headerKey, key) {
			return value
		}
	}
	return ""
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.InfoContext(r.Context(), "http request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func safeError(err error, status int) string {
	if status >= http.StatusInternalServerError {
		return "review processing failed"
	}
	return err.Error()
}

func isValidationError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "required") || strings.Contains(message, "maximum size")
}

func newReviewID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "rev_" + hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return "rev_" + hex.EncodeToString(b[:])
}
