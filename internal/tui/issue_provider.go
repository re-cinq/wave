package tui

import (
	"context"
	"strings"
	"time"

	"github.com/recinq/wave/internal/forge"
)

// IssueData is a TUI-specific projection of a forge issue.
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

// DefaultIssueDataProvider uses a forge client to fetch issues.
type DefaultIssueDataProvider struct {
	client   forge.Client
	repoSlug string // owner/repo format
}

// NewDefaultIssueDataProvider creates a new issue data provider.
func NewDefaultIssueDataProvider(client forge.Client, repoSlug string) *DefaultIssueDataProvider {
	return &DefaultIssueDataProvider{client: client, repoSlug: repoSlug}
}

// FetchIssues retrieves open issues from the configured repository.
func (p *DefaultIssueDataProvider) FetchIssues() ([]IssueData, error) {
	if p.repoSlug == "" {
		return nil, nil
	}
	owner, repo, ok := strings.Cut(p.repoSlug, "/")
	if !ok {
		return nil, nil
	}
	issues, err := p.client.ListIssues(context.Background(), owner, repo, forge.ListIssuesOptions{
		State:   "open",
		PerPage: 50,
		Sort:    "updated",
	})
	if err != nil {
		return nil, err
	}
	var result []IssueData
	for _, issue := range issues {
		if issue.IsPR {
			continue
		}
		var labelNames []string
		for _, l := range issue.Labels {
			labelNames = append(labelNames, l.Name)
		}
		result = append(result, IssueData{
			Number:    issue.Number,
			Title:     issue.Title,
			State:     issue.State,
			Author:    issue.Author,
			Labels:    labelNames,
			Assignees: issue.Assignees,
			Body:      issue.Body,
			Comments:  issue.Comments,
			CreatedAt: issue.CreatedAt,
			UpdatedAt: issue.UpdatedAt,
			HTMLURL:   issue.HTMLURL,
		})
	}
	return result, nil
}

