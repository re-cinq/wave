package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/state"
)

// mockForgeClient implements forge.Client with configurable responses.
type mockForgeClient struct {
	listPRs   func(ctx context.Context, owner, repo string, opts forge.ListPullRequestsOptions) ([]*forge.PullRequest, error)
	getPR     func(ctx context.Context, owner, repo string, number int) (*forge.PullRequest, error)
}

func (m *mockForgeClient) GetIssue(context.Context, string, string, int) (*forge.Issue, error) {
	return nil, forge.ErrNotSupported
}

func (m *mockForgeClient) ListIssues(context.Context, string, string, forge.ListIssuesOptions) ([]*forge.Issue, error) {
	return nil, forge.ErrNotSupported
}

func (m *mockForgeClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*forge.PullRequest, error) {
	if m.getPR != nil {
		return m.getPR(ctx, owner, repo, number)
	}
	return nil, forge.ErrNotSupported
}

func (m *mockForgeClient) ListPullRequests(ctx context.Context, owner, repo string, opts forge.ListPullRequestsOptions) ([]*forge.PullRequest, error) {
	if m.listPRs != nil {
		return m.listPRs(ctx, owner, repo, opts)
	}
	return nil, forge.ErrNotSupported
}

func (m *mockForgeClient) ListPullRequestCommits(context.Context, string, string, int) ([]*forge.Commit, error) {
	return nil, forge.ErrNotSupported
}

func (m *mockForgeClient) GetCommitChecks(context.Context, string, string, string) ([]*forge.CheckRun, error) {
	return nil, forge.ErrNotSupported
}

func (m *mockForgeClient) ListIssueComments(context.Context, string, string, int, int) ([]*forge.Comment, error) {
	return nil, forge.ErrNotSupported
}

func (m *mockForgeClient) CreatePullRequestReview(context.Context, string, string, int, string, string) error {
	return forge.ErrNotSupported
}

func (m *mockForgeClient) ForgeType() forge.ForgeType {
	return forge.ForgeGitHub
}

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

func TestGetPRListData_EnrichedStats(t *testing.T) {
	srv, _ := testServer(t)
	srv.repoSlug = "owner/repo"
	srv.forgeClient = &mockForgeClient{
		listPRs: func(_ context.Context, _, _ string, _ forge.ListPullRequestsOptions) ([]*forge.PullRequest, error) {
			return []*forge.PullRequest{
				{Number: 1, Title: "PR 1", State: "open", Author: "alice", CreatedAt: time.Now()},
				{Number: 2, Title: "PR 2", State: "open", Author: "bob", CreatedAt: time.Now()},
			}, nil
		},
		getPR: func(_ context.Context, _, _ string, number int) (*forge.PullRequest, error) {
			switch number {
			case 1:
				return &forge.PullRequest{Number: 1, Additions: 10, Deletions: 3, ChangedFiles: 2}, nil
			case 2:
				return &forge.PullRequest{Number: 2, Additions: 50, Deletions: 20, ChangedFiles: 8}, nil
			default:
				return nil, fmt.Errorf("unknown PR %d", number)
			}
		},
	}

	data := srv.getPRListData("open", 1)

	if len(data.PullRequests) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(data.PullRequests))
	}

	pr1 := data.PullRequests[0]
	if pr1.Additions != 10 || pr1.Deletions != 3 || pr1.ChangedFiles != 2 {
		t.Errorf("PR #1: expected additions=10 deletions=3 changed=2, got additions=%d deletions=%d changed=%d",
			pr1.Additions, pr1.Deletions, pr1.ChangedFiles)
	}

	pr2 := data.PullRequests[1]
	if pr2.Additions != 50 || pr2.Deletions != 20 || pr2.ChangedFiles != 8 {
		t.Errorf("PR #2: expected additions=50 deletions=20 changed=8, got additions=%d deletions=%d changed=%d",
			pr2.Additions, pr2.Deletions, pr2.ChangedFiles)
	}
}

func TestGetPRListData_PartialEnrichmentFailure(t *testing.T) {
	srv, _ := testServer(t)
	srv.repoSlug = "owner/repo"
	srv.forgeClient = &mockForgeClient{
		listPRs: func(_ context.Context, _, _ string, _ forge.ListPullRequestsOptions) ([]*forge.PullRequest, error) {
			return []*forge.PullRequest{
				{Number: 1, Title: "Good PR", State: "open", Author: "alice", CreatedAt: time.Now()},
				{Number: 2, Title: "Bad PR", State: "open", Author: "bob", CreatedAt: time.Now()},
			}, nil
		},
		getPR: func(_ context.Context, _, _ string, number int) (*forge.PullRequest, error) {
			if number == 1 {
				return &forge.PullRequest{Number: 1, Additions: 15, Deletions: 5, ChangedFiles: 3}, nil
			}
			return nil, fmt.Errorf("API error for PR #%d", number)
		},
	}

	data := srv.getPRListData("open", 1)

	if len(data.PullRequests) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(data.PullRequests))
	}

	pr1 := data.PullRequests[0]
	if pr1.Additions != 15 || pr1.Deletions != 5 || pr1.ChangedFiles != 3 {
		t.Errorf("PR #1: expected enriched stats, got additions=%d deletions=%d changed=%d",
			pr1.Additions, pr1.Deletions, pr1.ChangedFiles)
	}

	pr2 := data.PullRequests[1]
	if pr2.Additions != 0 || pr2.Deletions != 0 || pr2.ChangedFiles != 0 {
		t.Errorf("PR #2: expected zero stats on failure, got additions=%d deletions=%d changed=%d",
			pr2.Additions, pr2.Deletions, pr2.ChangedFiles)
	}
}

func TestGetPRListData_Labels(t *testing.T) {
	srv, _ := testServer(t)
	srv.repoSlug = "owner/repo"
	srv.forgeClient = &mockForgeClient{
		listPRs: func(_ context.Context, _, _ string, _ forge.ListPullRequestsOptions) ([]*forge.PullRequest, error) {
			return []*forge.PullRequest{
				{
					Number:    1,
					Title:     "Labeled PR",
					State:     "open",
					Author:    "alice",
					Labels:    []string{"bug", "priority:high"},
					CreatedAt: time.Now(),
				},
				{
					Number:    2,
					Title:     "No Labels",
					State:     "open",
					Author:    "bob",
					CreatedAt: time.Now(),
				},
			}, nil
		},
		getPR: func(_ context.Context, _, _ string, number int) (*forge.PullRequest, error) {
			return &forge.PullRequest{Number: number}, nil
		},
	}

	data := srv.getPRListData("open", 1)

	if len(data.PullRequests) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(data.PullRequests))
	}

	pr1 := data.PullRequests[0]
	if len(pr1.Labels) != 2 || pr1.Labels[0] != "bug" || pr1.Labels[1] != "priority:high" {
		t.Errorf("PR #1: expected labels [bug, priority:high], got %v", pr1.Labels)
	}

	pr2 := data.PullRequests[1]
	if len(pr2.Labels) != 0 {
		t.Errorf("PR #2: expected no labels, got %v", pr2.Labels)
	}
}
