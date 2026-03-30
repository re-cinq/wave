package continuous

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// ghIssue is the JSON shape returned by `gh issue list --json`.
type ghIssue struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// GitHubSource fetches work items from GitHub issues using the gh CLI.
type GitHubSource struct {
	Label string
	State string
	Limit int

	items   []*WorkItem
	index   int
	fetched bool
}

// NewGitHubSource creates a GitHubSource from parsed parameters.
func NewGitHubSource(params map[string]string) (*GitHubSource, error) {
	s := &GitHubSource{
		Label: params["label"],
		State: params["state"],
		Limit: 100,
	}
	if s.State == "" {
		s.State = "open"
	}
	if limitStr, ok := params["limit"]; ok {
		n, err := strconv.Atoi(limitStr)
		if err != nil {
			return nil, fmt.Errorf("invalid limit %q: %w", limitStr, err)
		}
		s.Limit = n
	}
	return s, nil
}

func (s *GitHubSource) fetch(ctx context.Context) error {
	args := []string{"issue", "list", "--json", "number,url", "--state", s.State, "--limit", strconv.Itoa(s.Limit)}
	if s.Label != "" {
		args = append(args, "--label", s.Label)
	}
	cmd := exec.CommandContext(ctx, "gh", args...)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("gh issue list failed: %w", err)
	}

	var issues []ghIssue
	if err := json.Unmarshal(out, &issues); err != nil {
		return fmt.Errorf("failed to parse gh output: %w", err)
	}

	s.items = make([]*WorkItem, len(issues))
	for i, issue := range issues {
		s.items[i] = &WorkItem{
			ID:    strconv.Itoa(issue.Number),
			Input: issue.URL,
		}
	}
	s.fetched = true
	return nil
}

func (s *GitHubSource) Next(ctx context.Context) (*WorkItem, error) {
	if !s.fetched {
		if err := s.fetch(ctx); err != nil {
			return nil, err
		}
	}
	if s.index >= len(s.items) {
		return nil, nil
	}
	item := s.items[s.index]
	s.index++
	return item, nil
}

func (s *GitHubSource) Name() string {
	if s.Label != "" {
		return fmt.Sprintf("github(label=%s, state=%s)", s.Label, s.State)
	}
	return fmt.Sprintf("github(state=%s)", s.State)
}
