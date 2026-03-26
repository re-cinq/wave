package forge

import (
	"context"
	"fmt"
)

// UnsupportedClient returns ErrNotSupported for all operations.
type UnsupportedClient struct {
	forgeType ForgeType
}

// NewUnsupportedClient creates a stub client for an unsupported forge type.
func NewUnsupportedClient(ft ForgeType) *UnsupportedClient {
	return &UnsupportedClient{forgeType: ft}
}

func (u *UnsupportedClient) ForgeType() ForgeType {
	return u.forgeType
}

func (u *UnsupportedClient) GetIssue(_ context.Context, _, _ string, _ int) (*Issue, error) {
	return nil, fmt.Errorf("%w: %s", ErrNotSupported, u.forgeType)
}

func (u *UnsupportedClient) ListIssues(_ context.Context, _, _ string, _ ListIssuesOptions) ([]*Issue, error) {
	return nil, fmt.Errorf("%w: %s", ErrNotSupported, u.forgeType)
}

func (u *UnsupportedClient) GetPullRequest(_ context.Context, _, _ string, _ int) (*PullRequest, error) {
	return nil, fmt.Errorf("%w: %s", ErrNotSupported, u.forgeType)
}

func (u *UnsupportedClient) ListPullRequests(_ context.Context, _, _ string, _ ListPullRequestsOptions) ([]*PullRequest, error) {
	return nil, fmt.Errorf("%w: %s", ErrNotSupported, u.forgeType)
}
