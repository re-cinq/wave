package tui

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/recinq/wave/internal/github"
	"github.com/recinq/wave/internal/manifest"
)

// IssueData is a TUI-specific projection of a GitHub issue.
type IssueData struct {
	Number    int
	Title     string
	State     string
	Author    string
	Labels    []string
	Assignees []string
	Comments  int
	CreatedAt time.Time
	UpdatedAt time.Time
	Body      string
	HTMLURL   string
}

// IssueDataProvider fetches issue data for the TUI.
type IssueDataProvider interface {
	FetchIssues() ([]IssueData, error)
}

// DefaultIssueDataProvider uses the GitHub client to fetch issues.
type DefaultIssueDataProvider struct {
	client   *github.Client
	manifest *manifest.Manifest
}

// NewDefaultIssueDataProvider creates a new issue data provider.
func NewDefaultIssueDataProvider(client *github.Client, m *manifest.Manifest) *DefaultIssueDataProvider {
	return &DefaultIssueDataProvider{client: client, manifest: m}
}

// FetchIssues retrieves open issues from the configured repository.
func (p *DefaultIssueDataProvider) FetchIssues() ([]IssueData, error) {
	if p.manifest == nil || p.manifest.Metadata.Repo == "" {
		return nil, nil
	}
	owner, repo, ok := strings.Cut(p.manifest.Metadata.Repo, "/")
	if !ok {
		return nil, nil
	}
	issues, err := p.client.ListIssues(context.Background(), owner, repo, github.ListIssuesOptions{
		State:   "open",
		PerPage: 50,
		Sort:    "updated",
	})
	if err != nil {
		return nil, err
	}
	var result []IssueData
	for _, issue := range issues {
		if issue.IsPullRequest() {
			continue // Skip PRs
		}
		d := IssueData{
			Number:    issue.Number,
			Title:     issue.Title,
			State:     issue.State,
			Body:      issue.Body,
			Comments:  issue.Comments,
			CreatedAt: issue.CreatedAt,
			UpdatedAt: issue.UpdatedAt,
			HTMLURL:   issue.HTMLURL,
		}
		if issue.User != nil {
			d.Author = issue.User.Login
		}
		for _, l := range issue.Labels {
			d.Labels = append(d.Labels, l.Name)
		}
		for _, a := range issue.Assignees {
			d.Assignees = append(d.Assignees, a.Login)
		}
		result = append(result, d)
	}
	return result, nil
}

// resolveGitHubToken returns a GitHub token from environment variables,
// falling back to `gh auth token` if available.
func resolveGitHubToken() string {
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err == nil {
		if token := strings.TrimSpace(string(out)); token != "" {
			return token
		}
	}
	return ""
}
