package tui

import (
	"context"
	"strings"
	"time"

	"github.com/recinq/wave/internal/github"
)

// PRData is a TUI-specific projection of a GitHub pull request.
type PRData struct {
	Number       int
	Title        string
	State        string
	Author       string
	Labels       []string
	Draft        bool
	Merged       bool
	Additions    int
	Deletions    int
	ChangedFiles int
	Comments     int
	Commits      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Body         string
	HTMLURL      string
	HeadBranch   string
	BaseBranch   string
}

// PRDataProvider fetches pull request data for the TUI.
type PRDataProvider interface {
	FetchPRs() ([]PRData, error)
}

// DefaultPRDataProvider uses the GitHub client to fetch pull requests.
type DefaultPRDataProvider struct {
	client   *github.Client
	repoSlug string // owner/repo format
}

// NewDefaultPRDataProvider creates a new PR data provider.
func NewDefaultPRDataProvider(client *github.Client, repoSlug string) *DefaultPRDataProvider {
	return &DefaultPRDataProvider{client: client, repoSlug: repoSlug}
}

// FetchPRs retrieves open pull requests from the configured repository.
func (p *DefaultPRDataProvider) FetchPRs() ([]PRData, error) {
	if p.repoSlug == "" {
		return nil, nil
	}
	owner, repo, ok := strings.Cut(p.repoSlug, "/")
	if !ok {
		return nil, nil
	}
	prs, err := p.client.ListPullRequests(context.Background(), owner, repo, github.ListPullRequestsOptions{
		State:   "open",
		PerPage: 50,
		Sort:    "updated",
	})
	if err != nil {
		return nil, err
	}
	var result []PRData
	for _, pr := range prs {
		d := PRData{
			Number:       pr.Number,
			Title:        pr.Title,
			State:        pr.State,
			Body:         pr.Body,
			Draft:        pr.Draft,
			Merged:       pr.Merged,
			Additions:    pr.Additions,
			Deletions:    pr.Deletions,
			ChangedFiles: pr.ChangedFiles,
			Comments:     pr.Comments,
			Commits:      pr.Commits,
			CreatedAt:    pr.CreatedAt,
			UpdatedAt:    pr.UpdatedAt,
			HTMLURL:      pr.HTMLURL,
		}
		if pr.User != nil {
			d.Author = pr.User.Login
		}
		for _, l := range pr.Labels {
			d.Labels = append(d.Labels, l.Name)
		}
		if pr.Head != nil {
			d.HeadBranch = pr.Head.Ref
		}
		if pr.Base != nil {
			d.BaseBranch = pr.Base.Ref
		}
		result = append(result, d)
	}
	return result, nil
}
