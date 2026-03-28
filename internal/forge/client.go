package forge

import (
	"context"
	"errors"
)

// ErrNotSupported is returned by stub implementations for unsupported forges.
var ErrNotSupported = errors.New("forge type not yet supported")

// Client is a read-only interface for forge issue/PR operations.
type Client interface {
	GetIssue(ctx context.Context, owner, repo string, number int) (*Issue, error)
	ListIssues(ctx context.Context, owner, repo string, opts ListIssuesOptions) ([]*Issue, error)
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error)
	ListPullRequests(ctx context.Context, owner, repo string, opts ListPullRequestsOptions) ([]*PullRequest, error)
	GetCommitChecks(ctx context.Context, owner, repo, ref string) ([]*CheckRun, error)
	ListIssueComments(ctx context.Context, owner, repo string, number int, limit int) ([]*Comment, error)
	ForgeType() ForgeType
}
