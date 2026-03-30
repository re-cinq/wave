package webui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/timeouts"
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

func TestHandleAPIIssues_DefaultState(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/issues", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIIssues(rec, req)

	var resp IssueListResponse
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

func TestHandleAPIIssues_StateParam(t *testing.T) {
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

			req := httptest.NewRequest("GET", "/api/issues?"+tt.query, nil)
			rec := httptest.NewRecorder()
			srv.handleAPIIssues(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}

			var resp IssueListResponse
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

func TestHandleIssuesPage_StateParam(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/issues?state=closed", nil)
	rec := httptest.NewRecorder()
	srv.handleIssuesPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "closed") {
		t.Errorf("expected response to contain filter state 'closed', got: %s", body)
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

// deadlineCapturingForge is a minimal forge.Client that captures the context
// deadline passed to ListIssues so tests can verify the correct timeout is used.
type deadlineCapturingForge struct {
	forge.Client // embed to satisfy interface; unused methods will panic
	deadline     time.Time
	ok           bool
}

func (f *deadlineCapturingForge) ListIssues(ctx context.Context, _, _ string, _ forge.ListIssuesOptions) ([]*forge.Issue, error) {
	f.deadline, f.ok = ctx.Deadline()
	return nil, nil
}

func TestGetIssueListData_UsesForgeAPIListTimeout(t *testing.T) {
	srv, _ := testServer(t)
	fc := &deadlineCapturingForge{}
	srv.forgeClient = fc
	srv.repoSlug = "owner/repo"

	before := time.Now()
	srv.getIssueListData("open", 1)
	after := time.Now()

	if !fc.ok {
		t.Fatal("expected context to have a deadline")
	}

	// The deadline should be approximately now + ForgeAPIList (30s).
	// Allow 2s tolerance for test execution jitter.
	wantMin := before.Add(timeouts.ForgeAPIList - 2*time.Second)
	wantMax := after.Add(timeouts.ForgeAPIList + 2*time.Second)
	if fc.deadline.Before(wantMin) || fc.deadline.After(wantMax) {
		t.Errorf("deadline %v not within expected range [%v, %v] for ForgeAPIList=%v",
			fc.deadline, wantMin, wantMax, timeouts.ForgeAPIList)
	}
}
