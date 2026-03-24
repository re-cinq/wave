package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
)

func TestHandleAPIPRs_NoGitHubClient(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/prs", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPRs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp PRListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.PullRequests) != 0 {
		t.Errorf("expected 0 PRs, got %d", len(resp.PullRequests))
	}
	if resp.Message == "" {
		t.Error("expected informative message when GitHub client is not configured")
	}
}

func TestHandlePRsPage_NoGitHubClient(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/prs", nil)
	rec := httptest.NewRecorder()
	srv.handlePRsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}

func TestHandleAPIPRs_DefaultState(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/prs", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPRs(rec, req)

	var resp PRListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.FilterState != "open" {
		t.Errorf("expected default filter_state 'open', got %q", resp.FilterState)
	}
	if resp.Page != 1 {
		t.Errorf("expected default page 1, got %d", resp.Page)
	}
}

func TestHandleAPIPRs_StateParam(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		expectedState string
		expectedPage  int
	}{
		{"open", "state=open", "open", 1},
		{"closed", "state=closed", "closed", 1},
		{"all", "state=all", "all", 1},
		{"invalid defaults to open", "state=invalid", "open", 1},
		{"page 2", "state=open&page=2", "open", 2},
		{"page 0 defaults to 1", "state=closed&page=0", "closed", 1},
		{"no state with page", "page=3", "open", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := testServer(t)

			req := httptest.NewRequest("GET", "/api/prs?"+tt.query, nil)
			rec := httptest.NewRecorder()
			srv.handleAPIPRs(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}

			var resp PRListResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.FilterState != tt.expectedState {
				t.Errorf("expected filter_state %q, got %q", tt.expectedState, resp.FilterState)
			}
			if resp.Page != tt.expectedPage {
				t.Errorf("expected page %d, got %d", tt.expectedPage, resp.Page)
			}
		})
	}
}

func TestHandlePRsPage_StateParam(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/prs?state=closed", nil)
	rec := httptest.NewRecorder()
	srv.handlePRsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "closed") {
		t.Errorf("expected response to contain filter state 'closed', got: %s", body)
	}
}

func TestHandleAPIPRs_EmptyRepoSlug(t *testing.T) {
	srv, _ := testServer(t)
	// Set a non-nil GitHub client but empty repo slug
	srv.repoSlug = ""

	req := httptest.NewRequest("GET", "/api/prs", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIPRs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp PRListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Message == "" {
		t.Error("expected informative message when GitHub is not configured")
	}
}

func TestRunToSummary_WithBranchName(t *testing.T) {
	now := time.Now()
	completed := now.Add(5 * time.Minute)
	record := state.RunRecord{
		RunID:        "test-run-123",
		PipelineName: "impl-issue",
		Status:       "completed",
		Input:        "https://github.com/example/repo/issues/42",
		TotalTokens:  5000,
		StartedAt:    now,
		CompletedAt:  &completed,
		BranchName:   "impl/issue-42",
		Tags:         []string{"automated"},
	}

	summary := runToSummary(record)

	if summary.BranchName != "impl/issue-42" {
		t.Errorf("expected branch name 'impl/issue-42', got %q", summary.BranchName)
	}
	if summary.RunID != "test-run-123" {
		t.Errorf("expected run ID 'test-run-123', got %q", summary.RunID)
	}
	if summary.Duration == "" {
		t.Error("expected non-empty duration for completed run")
	}
	if summary.InputPreview == "" {
		t.Error("expected non-empty input preview")
	}
}

func TestRunToSummary_RunningNoDuration(t *testing.T) {
	record := state.RunRecord{
		RunID:        "test-run-456",
		PipelineName: "test-pipeline",
		Status:       "running",
		StartedAt:    time.Now().Add(-30 * time.Second),
	}

	summary := runToSummary(record)

	if summary.Duration == "" {
		t.Error("expected non-empty duration for running run")
	}
}

func TestRunToSummary_LongInputTruncated(t *testing.T) {
	longInput := strings.Repeat("a", 200)
	record := state.RunRecord{
		RunID:        "test-run-789",
		PipelineName: "test-pipeline",
		Status:       "pending",
		Input:        longInput,
		StartedAt:    time.Now(),
	}

	summary := runToSummary(record)

	if len(summary.InputPreview) > 84 { // 80 + "..."
		t.Errorf("expected truncated input preview, got length %d", len(summary.InputPreview))
	}
	if !strings.HasSuffix(summary.InputPreview, "...") {
		t.Error("expected truncated input to end with '...'")
	}
}
