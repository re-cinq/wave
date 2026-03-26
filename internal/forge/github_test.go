package forge

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/github"
)

func TestConvertGitHubIssue(t *testing.T) {
	now := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
	closed := now.Add(time.Hour)

	gi := &github.Issue{
		Number:    42,
		Title:     "Test issue",
		Body:      "Issue body",
		State:     "closed",
		User:      &github.User{Login: "author"},
		Labels:    []*github.Label{{Name: "bug"}, {Name: "critical"}},
		Assignees: []*github.User{{Login: "dev1"}, {Login: "dev2"}},
		Comments:  5,
		CreatedAt: now,
		UpdatedAt: now.Add(30 * time.Minute),
		ClosedAt:  &closed,
		HTMLURL:   "https://github.com/owner/repo/issues/42",
	}

	issue := convertGitHubIssue(gi)

	if issue.Number != 42 {
		t.Errorf("Number = %d, want 42", issue.Number)
	}
	if issue.Title != "Test issue" {
		t.Errorf("Title = %q, want %q", issue.Title, "Test issue")
	}
	if issue.Author != "author" {
		t.Errorf("Author = %q, want %q", issue.Author, "author")
	}
	if len(issue.Labels) != 2 {
		t.Fatalf("Labels = %d, want 2", len(issue.Labels))
	}
	if issue.Labels[0] != "bug" || issue.Labels[1] != "critical" {
		t.Errorf("Labels = %v, want [bug critical]", issue.Labels)
	}
	if len(issue.Assignees) != 2 {
		t.Fatalf("Assignees = %d, want 2", len(issue.Assignees))
	}
	if issue.ClosedAt == nil {
		t.Error("ClosedAt should not be nil")
	}
	if issue.IsPR {
		t.Error("IsPR should be false for a regular issue")
	}
}

func TestConvertGitHubIssue_NilUser(t *testing.T) {
	gi := &github.Issue{
		Number: 1,
		User:   nil,
	}
	issue := convertGitHubIssue(gi)
	if issue.Author != "" {
		t.Errorf("Author = %q, want empty for nil user", issue.Author)
	}
}

func TestConvertGitHubIssue_NilLabels(t *testing.T) {
	gi := &github.Issue{
		Number: 1,
		Labels: []*github.Label{nil, {Name: "valid"}, nil},
	}
	issue := convertGitHubIssue(gi)
	if len(issue.Labels) != 1 {
		t.Errorf("Labels count = %d, want 1 (nil labels filtered)", len(issue.Labels))
	}
}

func TestConvertGitHubIssue_IsPR(t *testing.T) {
	gi := &github.Issue{
		Number: 1,
		PullRequest: &struct {
			URL     string `json:"url"`
			HTMLURL string `json:"html_url"`
		}{URL: "https://api.github.com/repos/o/r/pulls/1"},
	}
	issue := convertGitHubIssue(gi)
	if !issue.IsPR {
		t.Error("IsPR should be true when PullRequest is set")
	}
}

func TestConvertGitHubPR(t *testing.T) {
	now := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
	merged := now.Add(2 * time.Hour)

	gp := &github.PullRequest{
		Number:       99,
		Title:        "Test PR",
		Body:         "PR body",
		State:        "closed",
		User:         &github.User{Login: "prauthor"},
		Labels:       []*github.Label{{Name: "enhancement"}},
		Draft:        false,
		Merged:       true,
		Head:         &github.GitRef{Ref: "feat/branch"},
		Base:         &github.GitRef{Ref: "main"},
		Additions:    100,
		Deletions:    50,
		ChangedFiles: 10,
		Commits:      3,
		Comments:     7,
		CreatedAt:    now,
		UpdatedAt:    now.Add(time.Hour),
		MergedAt:     &merged,
		HTMLURL:      "https://github.com/owner/repo/pull/99",
	}

	pr := convertGitHubPR(gp)

	if pr.Number != 99 {
		t.Errorf("Number = %d, want 99", pr.Number)
	}
	if pr.Author != "prauthor" {
		t.Errorf("Author = %q, want %q", pr.Author, "prauthor")
	}
	if pr.HeadBranch != "feat/branch" {
		t.Errorf("HeadBranch = %q, want %q", pr.HeadBranch, "feat/branch")
	}
	if pr.BaseBranch != "main" {
		t.Errorf("BaseBranch = %q, want %q", pr.BaseBranch, "main")
	}
	if !pr.Merged {
		t.Error("Merged should be true")
	}
	if pr.MergedAt == nil {
		t.Error("MergedAt should not be nil")
	}
	if pr.Additions != 100 {
		t.Errorf("Additions = %d, want 100", pr.Additions)
	}
}

func TestConvertGitHubPR_NilHeadBase(t *testing.T) {
	gp := &github.PullRequest{
		Number: 1,
		Head:   nil,
		Base:   nil,
		User:   nil,
	}
	pr := convertGitHubPR(gp)
	if pr.HeadBranch != "" {
		t.Errorf("HeadBranch = %q, want empty for nil head", pr.HeadBranch)
	}
	if pr.BaseBranch != "" {
		t.Errorf("BaseBranch = %q, want empty for nil base", pr.BaseBranch)
	}
	if pr.Author != "" {
		t.Errorf("Author = %q, want empty for nil user", pr.Author)
	}
}

func TestGitHubClient_ForgeType(t *testing.T) {
	client := NewGitHubClient(nil)
	if client.ForgeType() != ForgeGitHub {
		t.Errorf("ForgeType() = %v, want %v", client.ForgeType(), ForgeGitHub)
	}
}

func TestGitHubClient_UnwrapGitHub(t *testing.T) {
	ghClient := &github.Client{}
	client := NewGitHubClient(ghClient)
	if client.UnwrapGitHub() != ghClient {
		t.Error("UnwrapGitHub() should return the underlying client")
	}
}
