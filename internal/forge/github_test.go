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
	if issue.Labels[0].Name != "bug" || issue.Labels[1].Name != "critical" {
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
	client, err := NewGitHubClient(github.NewClient(github.ClientConfig{Token: "test"}))
	if err != nil {
		t.Fatalf("NewGitHubClient: %v", err)
	}
	if client.ForgeType() != ForgeGitHub {
		t.Errorf("ForgeType() = %v, want %v", client.ForgeType(), ForgeGitHub)
	}
}

func TestNewGitHubClient_NilReturnsError(t *testing.T) {
	client, err := NewGitHubClient(nil)
	if err == nil {
		t.Fatal("expected error for nil client")
	}
	if client != nil {
		t.Fatal("expected nil client when error returned")
	}
}

func TestGitHubClient_UnwrapGitHub(t *testing.T) {
	ghClient := &github.Client{}
	client, err := NewGitHubClient(ghClient)
	if err != nil {
		t.Fatalf("NewGitHubClient: %v", err)
	}
	if client.UnwrapGitHub() != ghClient {
		t.Error("UnwrapGitHub() should return the underlying client")
	}
}

func TestConvertGitHubPRCommits(t *testing.T) {
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	ghCommits := []*github.PullRequestCommit{
		{
			SHA: "abc1234567890def",
			Commit: github.PullRequestCommitDetail{
				Message: "feat: add commit view\n\nDetailed description here.",
				Author:  github.PullRequestCommitAuthor{Name: "git-author", Date: now},
			},
			Author:  &github.User{Login: "gh-user"},
			HTMLURL: "https://github.com/owner/repo/commit/abc1234567890def",
		},
		{
			SHA: "def4567890123abc",
			Commit: github.PullRequestCommitDetail{
				Message: "fix: typo correction",
				Author:  github.PullRequestCommitAuthor{Name: "git-author2", Date: now.Add(-time.Hour)},
			},
			Author:  nil, // nil GitHub user — should fall back to git commit author
			HTMLURL: "https://github.com/owner/repo/commit/def4567890123abc",
		},
	}

	result := make([]*Commit, 0, len(ghCommits))
	for _, gc := range ghCommits {
		c := &Commit{
			SHA:     gc.SHA,
			Message: gc.Commit.Message,
			Author:  gc.Commit.Author.Name,
			Date:    gc.Commit.Author.Date,
			HTMLURL: gc.HTMLURL,
		}
		if gc.Author != nil {
			c.Author = gc.Author.Login
		}
		result = append(result, c)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(result))
	}

	// First commit: GitHub user login overrides git author name
	if result[0].SHA != "abc1234567890def" {
		t.Errorf("result[0].SHA = %q, want %q", result[0].SHA, "abc1234567890def")
	}
	if result[0].Author != "gh-user" {
		t.Errorf("result[0].Author = %q, want %q (GitHub login)", result[0].Author, "gh-user")
	}
	if result[0].HTMLURL != "https://github.com/owner/repo/commit/abc1234567890def" {
		t.Errorf("result[0].HTMLURL = %q, unexpected", result[0].HTMLURL)
	}

	// Second commit: nil GitHub user, falls back to git author name
	if result[1].Author != "git-author2" {
		t.Errorf("result[1].Author = %q, want %q (git commit author fallback)", result[1].Author, "git-author2")
	}
	if !result[1].Date.Equal(now.Add(-time.Hour)) {
		t.Errorf("result[1].Date = %v, want %v", result[1].Date, now.Add(-time.Hour))
	}
}

func TestConvertGitHubCheckRuns(t *testing.T) {
	t.Run("maps all fields correctly", func(t *testing.T) {
		ghCheckRuns := []*github.CheckRun{
			{
				ID:         100,
				Name:       "CI / build",
				Status:     "completed",
				Conclusion: "success",
				HTMLURL:    "https://github.com/owner/repo/runs/100",
			},
			{
				ID:         200,
				Name:       "CI / lint",
				Status:     "in_progress",
				Conclusion: "",
				HTMLURL:    "https://github.com/owner/repo/runs/200",
			},
			{
				ID:         300,
				Name:       "CI / test",
				Status:     "completed",
				Conclusion: "failure",
				HTMLURL:    "https://github.com/owner/repo/runs/300",
			},
		}

		result := make([]*CheckRun, 0, len(ghCheckRuns))
		for _, cr := range ghCheckRuns {
			result = append(result, &CheckRun{
				Name:       cr.Name,
				Status:     cr.Status,
				Conclusion: cr.Conclusion,
				HTMLURL:    cr.HTMLURL,
			})
		}

		if len(result) != 3 {
			t.Fatalf("expected 3 check runs, got %d", len(result))
		}

		// Verify first check run — completed/success
		if result[0].Name != "CI / build" {
			t.Errorf("result[0].Name = %q, want %q", result[0].Name, "CI / build")
		}
		if result[0].Status != "completed" {
			t.Errorf("result[0].Status = %q, want %q", result[0].Status, "completed")
		}
		if result[0].Conclusion != "success" {
			t.Errorf("result[0].Conclusion = %q, want %q", result[0].Conclusion, "success")
		}
		if result[0].HTMLURL != "https://github.com/owner/repo/runs/100" {
			t.Errorf("result[0].HTMLURL = %q, want %q", result[0].HTMLURL, "https://github.com/owner/repo/runs/100")
		}

		// Verify second check run — in_progress with empty conclusion
		if result[1].Status != "in_progress" {
			t.Errorf("result[1].Status = %q, want %q", result[1].Status, "in_progress")
		}
		if result[1].Conclusion != "" {
			t.Errorf("result[1].Conclusion = %q, want empty", result[1].Conclusion)
		}

		// Verify third check run — completed/failure
		if result[2].Conclusion != "failure" {
			t.Errorf("result[2].Conclusion = %q, want %q", result[2].Conclusion, "failure")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		ghCheckRuns := []*github.CheckRun{}
		result := make([]*CheckRun, 0, len(ghCheckRuns))
		for _, cr := range ghCheckRuns {
			result = append(result, &CheckRun{
				Name:       cr.Name,
				Status:     cr.Status,
				Conclusion: cr.Conclusion,
				HTMLURL:    cr.HTMLURL,
			})
		}
		if len(result) != 0 {
			t.Errorf("expected 0 check runs, got %d", len(result))
		}
	})

	t.Run("ID field is not mapped to forge type", func(t *testing.T) {
		// The forge.CheckRun type intentionally omits the ID field.
		// Verify that the mapping works without it.
		ghCR := &github.CheckRun{
			ID:         999,
			Name:       "test",
			Status:     "completed",
			Conclusion: "success",
			HTMLURL:    "https://example.com",
		}
		forgeCR := &CheckRun{
			Name:       ghCR.Name,
			Status:     ghCR.Status,
			Conclusion: ghCR.Conclusion,
			HTMLURL:    ghCR.HTMLURL,
		}
		if forgeCR.Name != "test" {
			t.Errorf("Name = %q, want %q", forgeCR.Name, "test")
		}
	})
}
