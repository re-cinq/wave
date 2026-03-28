package forge

import (
	"context"
	"errors"
	"testing"
)

func TestGitHubClient_ImplementsClient(t *testing.T) {
	// Compile-time interface check
	var _ Client = (*GitHubClient)(nil)
}

func TestUnsupportedClient_ImplementsClient(t *testing.T) {
	var _ Client = (*UnsupportedClient)(nil)
}

func TestUnsupportedClient_ReturnsErrNotSupported(t *testing.T) {
	ctx := context.Background()
	client := NewUnsupportedClient(ForgeGitLab)

	if client.ForgeType() != ForgeGitLab {
		t.Errorf("ForgeType() = %v, want %v", client.ForgeType(), ForgeGitLab)
	}

	_, err := client.GetIssue(ctx, "owner", "repo", 1)
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("GetIssue() error = %v, want ErrNotSupported", err)
	}

	_, err = client.ListIssues(ctx, "owner", "repo", ListIssuesOptions{})
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("ListIssues() error = %v, want ErrNotSupported", err)
	}

	_, err = client.GetPullRequest(ctx, "owner", "repo", 1)
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("GetPullRequest() error = %v, want ErrNotSupported", err)
	}

	_, err = client.ListPullRequests(ctx, "owner", "repo", ListPullRequestsOptions{})
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("ListPullRequests() error = %v, want ErrNotSupported", err)
	}

	_, err = client.GetCommitChecks(ctx, "owner", "repo", "abc123")
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("GetCommitChecks() error = %v, want ErrNotSupported", err)
	}

	_, err = client.ListIssueComments(ctx, "owner", "repo", 1, 10)
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("ListIssueComments() error = %v, want ErrNotSupported", err)
	}
}

func TestUnsupportedClient_ErrorContainsForgeType(t *testing.T) {
	client := NewUnsupportedClient(ForgeBitbucket)
	_, err := client.GetIssue(context.Background(), "o", "r", 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("expected ErrNotSupported, got %v", err)
	}
	if got := err.Error(); got == "" {
		t.Error("expected non-empty error message")
	}
}
