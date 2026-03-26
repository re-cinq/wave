package forge

import (
	"context"
	"errors"
)

// ErrNotConfigured is returned when a forge client has no valid token.
var ErrNotConfigured = errors.New("forge client not configured: no authentication token found")

// ErrNotSupported is returned by stub implementations for unsupported forges.
var ErrNotSupported = errors.New("forge type not yet supported")

// Client is a read-only interface for forge issue/PR operations.
type Client interface {
	GetIssue(ctx context.Context, owner, repo string, number int) (*Issue, error)
	ListIssues(ctx context.Context, owner, repo string, opts ListIssuesOptions) ([]*Issue, error)
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error)
	ListPullRequests(ctx context.Context, owner, repo string, opts ListPullRequestsOptions) ([]*PullRequest, error)
	ForgeType() ForgeType
}
