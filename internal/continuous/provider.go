package continuous

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

// WorkItem represents a single item of work to process in continuous mode.
type WorkItem struct {
	Key    string   // Stable identifier (e.g., normalized issue URL)
	Input  string   // Pipeline input string (e.g., issue URL)
	Labels []string // Associated labels
	URL    string   // Source URL
}

// WorkItemProvider returns the next work item to process.
// Returns nil when no more items are available.
type WorkItemProvider interface {
	Next(ctx context.Context) (*WorkItem, error)
}

// GitHubProvider fetches open GitHub issues as work items using the gh CLI.
type GitHubProvider struct {
	repo         string
	labelFilter  string
	store        ProcessedItemTracker
	pipelineName string
}

// NewGitHubProvider creates a provider that lists open issues from a GitHub repo.
func NewGitHubProvider(repo, labelFilter string, store ProcessedItemTracker, pipelineName string) *GitHubProvider {
	return &GitHubProvider{
		repo:         repo,
		labelFilter:  labelFilter,
		store:        store,
		pipelineName: pipelineName,
	}
}

// ghIssue represents the JSON output from gh issue list.
type ghIssue struct {
	Number int       `json:"number"`
	Title  string    `json:"title"`
	URL    string    `json:"url"`
	Labels []ghLabel `json:"labels"`
}

type ghLabel struct {
	Name string `json:"name"`
}

// Next returns the next unprocessed issue, or nil if none remain.
func (g *GitHubProvider) Next(ctx context.Context) (*WorkItem, error) {
	issues, err := g.fetchIssues(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issues: %w", err)
	}

	for _, issue := range issues {
		key := BuildItemKey(issue.URL)

		if g.store != nil {
			processed, err := g.store.IsItemProcessed(g.pipelineName, key)
			if err != nil {
				return nil, fmt.Errorf("failed to check processed state: %w", err)
			}
			if processed {
				continue
			}
		}

		labels := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			labels[i] = l.Name
		}

		return &WorkItem{
			Key:    key,
			Input:  issue.URL,
			Labels: labels,
			URL:    issue.URL,
		}, nil
	}

	return nil, nil
}

// fetchIssues calls gh issue list and returns parsed issues.
func (g *GitHubProvider) fetchIssues(ctx context.Context) ([]ghIssue, error) {
	args := []string{"issue", "list", "--repo", g.repo, "--state", "open",
		"--json", "number,title,url,labels", "--limit", "50",
		"--sort", "created", "--order", "asc"}

	if g.labelFilter != "" {
		args = append(args, "--label", g.labelFilter)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("gh issue list failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh issue list failed: %w", err)
	}

	return parseGHIssues(out)
}

// parseGHIssues parses the JSON output from gh issue list.
func parseGHIssues(data []byte) ([]ghIssue, error) {
	var issues []ghIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse gh issue list output: %w", err)
	}
	return issues, nil
}
