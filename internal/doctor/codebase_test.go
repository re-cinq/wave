package doctor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/github"
)

func TestAnalyzeCodebase_NonGitHub(t *testing.T) {
	result, err := AnalyzeCodebase(context.Background(), CodebaseOptions{
		ForgeInfo: forge.ForgeInfo{Type: forge.ForgeGitLab},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for non-GitHub forge")
	}
}

func TestAnalyzeCodebase_WithMockServer(t *testing.T) {
	now := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	staleDate := now.AddDate(0, 0, -20)
	recentDate := now.AddDate(0, 0, -1)

	mux := http.NewServeMux()

	// Mock PRs endpoint
	mux.HandleFunc("GET /repos/test/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		prs := []github.PullRequest{
			{Number: 1, UpdatedAt: recentDate, Comments: 0}, // needs review
			{Number: 2, UpdatedAt: staleDate, Comments: 3},  // stale
			{Number: 3, UpdatedAt: recentDate, Comments: 1}, // normal
		}
		_ = json.NewEncoder(w).Encode(prs)
	})

	// Mock issues endpoint
	mux.HandleFunc("GET /repos/test/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		issues := []github.Issue{
			{
				Number:    10,
				Title:     "A well-described issue with good context",
				Body:      "This is a detailed description with steps to reproduce, expected behavior, and actual behavior.",
				State:     "open",
				Assignees: []*github.User{{Login: "user1"}},
			},
			{
				Number: 11,
				Title:  "bad",
				Body:   "",
				State:  "open",
			},
			{
				Number: 12,
				Title:  "Another good issue with plenty of detail",
				Body:   "Detailed steps and context provided here with multiple paragraphs of explanation.",
				State:  "open",
			},
		}
		_ = json.NewEncoder(w).Encode(issues)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := github.NewClient(github.ClientConfig{
		BaseURL: server.URL,
		Token:   "test-token",
	})
	forgeClient, err := forge.NewGitHubClient(client)
	if err != nil {
		t.Fatalf("NewGitHubClient: %v", err)
	}

	result, err := AnalyzeCodebase(context.Background(), CodebaseOptions{
		ForgeInfo: forge.ForgeInfo{
			Type:  forge.ForgeGitHub,
			Owner: "test",
			Repo:  "repo",
		},
		ForgeClient: forgeClient,
		Now:         func() time.Time { return now },
		RunGHCmd: func(args ...string) ([]byte, error) {
			runs := []ghRunResult{
				{Status: "completed", Conclusion: "success"},
				{Status: "completed", Conclusion: "failure"},
				{Status: "completed", Conclusion: "success"},
			}
			return json.Marshal(runs)
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// PRs
	if result.PRs.Open != 3 {
		t.Errorf("PRs.Open = %d, want 3", result.PRs.Open)
	}
	if result.PRs.Stale != 1 {
		t.Errorf("PRs.Stale = %d, want 1", result.PRs.Stale)
	}
	if result.PRs.NeedsReview != 1 {
		t.Errorf("PRs.NeedsReview = %d, want 1", result.PRs.NeedsReview)
	}

	// Issues (3 items, 1 is PR-shaped... but we're using direct issue objects not the PR endpoint)
	if result.Issues.Open != 3 {
		t.Errorf("Issues.Open = %d, want 3", result.Issues.Open)
	}
	if result.Issues.Unassigned != 2 {
		t.Errorf("Issues.Unassigned = %d, want 2", result.Issues.Unassigned)
	}

	// CI
	if result.CI.Status != "failing" {
		t.Errorf("CI.Status = %q, want \"failing\"", result.CI.Status)
	}
	if result.CI.Failures != 1 {
		t.Errorf("CI.Failures = %d, want 1", result.CI.Failures)
	}
}

func TestAnalyzeCIStatus_NoGHCLI(t *testing.T) {
	ci := analyzeCIStatus(CodebaseOptions{
		RunGHCmd: func(args ...string) ([]byte, error) {
			return nil, &exec.Error{Name: "gh", Err: exec.ErrNotFound}
		},
	})
	if ci.Status != "unknown" {
		t.Errorf("CI.Status = %q, want \"unknown\"", ci.Status)
	}
}

func TestAnalyzeCIStatus_AllPassing(t *testing.T) {
	ci := analyzeCIStatus(CodebaseOptions{
		RunGHCmd: func(args ...string) ([]byte, error) {
			runs := []ghRunResult{
				{Status: "completed", Conclusion: "success"},
				{Status: "completed", Conclusion: "success"},
			}
			return json.Marshal(runs)
		},
	})
	if ci.Status != "passing" {
		t.Errorf("CI.Status = %q, want \"passing\"", ci.Status)
	}
	if ci.Failures != 0 {
		t.Errorf("CI.Failures = %d, want 0", ci.Failures)
	}
}
