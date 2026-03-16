package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleAPIIssues_NoGitHubClient(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/issues", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIIssues(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp IssueListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(resp.Issues))
	}
	if resp.Message == "" {
		t.Error("expected informative message when GitHub client is not configured")
	}
}

func TestHandleIssuesPage_NoGitHubClient(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/issues", nil)
	rec := httptest.NewRecorder()
	srv.handleIssuesPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}

func TestHandleAPIStartFromIssue_MissingFields(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`{"issue_url":"","pipeline_name":""}`)
	req := httptest.NewRequest("POST", "/api/issues/start", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleAPIStartFromIssue(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAPIStartFromIssue_InvalidBody(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest("POST", "/api/issues/start", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleAPIStartFromIssue(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d", rec.Code)
	}
}

func TestHandleAPIStartFromIssue_PipelineNotFound(t *testing.T) {
	srv, _ := testServer(t)

	body := strings.NewReader(`{"issue_url":"https://github.com/re-cinq/wave/issues/1","pipeline_name":"nonexistent"}`)
	req := httptest.NewRequest("POST", "/api/issues/start", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleAPIStartFromIssue(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing pipeline, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSplitRepoSlug(t *testing.T) {
	tests := []struct {
		slug      string
		wantOwner string
		wantRepo  string
	}{
		{"re-cinq/wave", "re-cinq", "wave"},
		{"owner/repo", "owner", "repo"},
		{"invalid", "", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		owner, repo := splitRepoSlug(tt.slug)
		if owner != tt.wantOwner || repo != tt.wantRepo {
			t.Errorf("splitRepoSlug(%q) = (%q, %q), want (%q, %q)", tt.slug, owner, repo, tt.wantOwner, tt.wantRepo)
		}
	}
}
