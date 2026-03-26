package forge

import (
	"context"

	"github.com/recinq/wave/internal/github"
)

// GitHubClient adapts *github.Client to the forge.Client interface.
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient wraps an existing github.Client.
func NewGitHubClient(client *github.Client) *GitHubClient {
	return &GitHubClient{client: client}
}

// UnwrapGitHub returns the underlying *github.Client for GitHub-specific operations.
func (g *GitHubClient) UnwrapGitHub() *github.Client {
	return g.client
}

func (g *GitHubClient) ForgeType() ForgeType {
	return ForgeGitHub
}

func (g *GitHubClient) GetIssue(ctx context.Context, owner, repo string, number int) (*Issue, error) {
	gi, err := g.client.GetIssue(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}
	return convertGitHubIssue(gi), nil
}

func (g *GitHubClient) ListIssues(ctx context.Context, owner, repo string, opts ListIssuesOptions) ([]*Issue, error) {
	ghOpts := github.ListIssuesOptions{
		State:   opts.State,
		Labels:  opts.Labels,
		Sort:    opts.Sort,
		PerPage: opts.PerPage,
		Page:    opts.Page,
	}
	ghIssues, err := g.client.ListIssues(ctx, owner, repo, ghOpts)
	if err != nil {
		return nil, err
	}
	result := make([]*Issue, 0, len(ghIssues))
	for _, gi := range ghIssues {
		result = append(result, convertGitHubIssue(gi))
	}
	return result, nil
}

func (g *GitHubClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	gp, err := g.client.GetPullRequest(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}
	return convertGitHubPR(gp), nil
}

func (g *GitHubClient) ListPullRequests(ctx context.Context, owner, repo string, opts ListPullRequestsOptions) ([]*PullRequest, error) {
	ghOpts := github.ListPullRequestsOptions{
		State:   opts.State,
		Sort:    opts.Sort,
		PerPage: opts.PerPage,
		Page:    opts.Page,
	}
	ghPRs, err := g.client.ListPullRequests(ctx, owner, repo, ghOpts)
	if err != nil {
		return nil, err
	}
	result := make([]*PullRequest, 0, len(ghPRs))
	for _, gp := range ghPRs {
		result = append(result, convertGitHubPR(gp))
	}
	return result, nil
}

func convertGitHubIssue(gi *github.Issue) *Issue {
	issue := &Issue{
		Number:    gi.Number,
		Title:     gi.Title,
		Body:      gi.Body,
		State:     gi.State,
		Comments:  gi.Comments,
		CreatedAt: gi.CreatedAt,
		UpdatedAt: gi.UpdatedAt,
		ClosedAt:  gi.ClosedAt,
		HTMLURL:   gi.HTMLURL,
		IsPR:      gi.IsPullRequest(),
	}
	if gi.User != nil {
		issue.Author = gi.User.Login
	}
	for _, l := range gi.Labels {
		if l != nil {
			issue.Labels = append(issue.Labels, l.Name)
		}
	}
	for _, a := range gi.Assignees {
		if a != nil {
			issue.Assignees = append(issue.Assignees, a.Login)
		}
	}
	return issue
}

func convertGitHubPR(gp *github.PullRequest) *PullRequest {
	pr := &PullRequest{
		Number:       gp.Number,
		Title:        gp.Title,
		Body:         gp.Body,
		State:        gp.State,
		Draft:        gp.Draft,
		Merged:       gp.Merged,
		Additions:    gp.Additions,
		Deletions:    gp.Deletions,
		ChangedFiles: gp.ChangedFiles,
		Commits:      gp.Commits,
		Comments:     gp.Comments,
		CreatedAt:    gp.CreatedAt,
		UpdatedAt:    gp.UpdatedAt,
		ClosedAt:     gp.ClosedAt,
		MergedAt:     gp.MergedAt,
		HTMLURL:      gp.HTMLURL,
	}
	if gp.User != nil {
		pr.Author = gp.User.Login
	}
	for _, l := range gp.Labels {
		if l != nil {
			pr.Labels = append(pr.Labels, l.Name)
		}
	}
	if gp.Head != nil {
		pr.HeadBranch = gp.Head.Ref
	}
	if gp.Base != nil {
		pr.BaseBranch = gp.Base.Ref
	}
	return pr
}
