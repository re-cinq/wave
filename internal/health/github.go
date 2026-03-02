package health

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

// GitHubAnalyzer implements ForgeAnalyzer for GitHub repositories using the
// gh CLI tool.
type GitHubAnalyzer struct {
	// cmdRunner allows test injection; defaults to exec.CommandContext.
	cmdRunner func(ctx context.Context, name string, args ...string) *exec.Cmd
	// repoPath is the local git repo path for setting Cmd.Dir.
	repoPath string
	// opts contains analysis configuration.
	opts AnalyzeOptions
}

// NewGitHubAnalyzer creates a GitHubAnalyzer for the given local repository
// path with the supplied analysis options.
func NewGitHubAnalyzer(repoPath string, opts AnalyzeOptions) *GitHubAnalyzer {
	return &GitHubAnalyzer{
		cmdRunner: exec.CommandContext,
		repoPath:  repoPath,
		opts:      opts,
	}
}

// conventionalCommitRe matches conventional commit prefixes like "feat(pipeline):"
// and captures the scope in group 1.
var conventionalCommitRe = regexp.MustCompile(`^\w+\(([^)]+)\):`)

// runGH executes a gh command with the given arguments, setting the working
// directory to the analyzer's repoPath. It returns stdout bytes on success
// or an error containing stderr context on failure.
func (a *GitHubAnalyzer) runGH(ctx context.Context, args ...string) ([]byte, error) {
	cmd := a.cmdRunner(ctx, "gh", args...)
	cmd.Dir = a.repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}

	return stdout.Bytes(), nil
}

// AnalyzeCommits fetches recent commits within the configured window and
// returns activity statistics including author breakdown, areas of activity,
// and commit frequency.
func (a *GitHubAnalyzer) AnalyzeCommits(ctx context.Context, repo string) (*CommitAnalysis, error) {
	windowDays := a.opts.CommitWindowDays
	if windowDays <= 0 {
		windowDays = DefaultCommitWindowDays
	}

	since := time.Now().UTC().AddDate(0, 0, -windowDays).Format(time.RFC3339)
	perPage := a.opts.MaxItems
	if perPage <= 0 {
		perPage = DefaultMaxItems
	}

	endpoint := fmt.Sprintf("repos/%s/commits?per_page=%d&since=%s", repo, perPage, since)
	data, err := a.runGH(ctx, "api", endpoint)
	if err != nil {
		return nil, fmt.Errorf("analyze commits: %w", err)
	}

	// GitHub returns an empty array or a JSON array of commit objects.
	var commits []struct {
		Sha    string `json:"sha"`
		Commit struct {
			Author struct {
				Name string `json:"name"`
				Date string `json:"date"`
			} `json:"author"`
			Message string `json:"message"`
		} `json:"commit"`
	}

	if err := json.Unmarshal(data, &commits); err != nil {
		return nil, fmt.Errorf("analyze commits: parse response: %w", err)
	}

	// Count commits per author.
	authorCounts := make(map[string]int)
	for _, c := range commits {
		name := c.Commit.Author.Name
		if name == "" {
			name = "unknown"
		}
		authorCounts[name]++
	}

	authors := make([]AuthorActivity, 0, len(authorCounts))
	for name, count := range authorCounts {
		authors = append(authors, AuthorActivity{
			Name:        name,
			CommitCount: count,
		})
	}
	// Sort by commit count descending, then name ascending for stability.
	sort.Slice(authors, func(i, j int) bool {
		if authors[i].CommitCount != authors[j].CommitCount {
			return authors[i].CommitCount > authors[j].CommitCount
		}
		return authors[i].Name < authors[j].Name
	})

	// Extract areas of activity from conventional commit scopes.
	areaSet := make(map[string]struct{})
	for _, c := range commits {
		msg := c.Commit.Message
		// Take only the first line of the commit message.
		if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
			msg = msg[:idx]
		}
		matches := conventionalCommitRe.FindStringSubmatch(msg)
		if len(matches) >= 2 {
			areaSet[matches[1]] = struct{}{}
		} else {
			areaSet["uncategorized"] = struct{}{}
		}
	}

	areas := make([]string, 0, len(areaSet))
	for area := range areaSet {
		areas = append(areas, area)
	}
	sort.Strings(areas)

	totalCount := len(commits)
	frequencyPerDay := 0.0
	if windowDays > 0 {
		frequencyPerDay = math.Round(float64(totalCount)/float64(windowDays)*100) / 100
	}

	return &CommitAnalysis{
		TotalCount:      totalCount,
		WindowDays:      windowDays,
		Authors:         authors,
		AreasOfActivity: areas,
		FrequencyPerDay: frequencyPerDay,
	}, nil
}

// AnalyzePRs retrieves open pull requests and categorizes them by review
// state, staleness, and recent activity.
func (a *GitHubAnalyzer) AnalyzePRs(ctx context.Context, repo string) (*PRSummary, error) {
	maxItems := a.opts.MaxItems
	if maxItems <= 0 {
		maxItems = DefaultMaxItems
	}

	stalenessThreshold := a.opts.StalenessThresholdDays
	if stalenessThreshold <= 0 {
		stalenessThreshold = DefaultStalenessThresholdDays
	}

	data, err := a.runGH(ctx,
		"pr", "list",
		"--repo", repo,
		"--state", "open",
		"--json", "number,title,author,updatedAt,reviewDecision",
		"--limit", fmt.Sprintf("%d", maxItems),
	)
	if err != nil {
		return nil, fmt.Errorf("analyze PRs: %w", err)
	}

	var prs []struct {
		Number         int    `json:"number"`
		Title          string `json:"title"`
		Author         struct {
			Login string `json:"login"`
		} `json:"author"`
		UpdatedAt      time.Time `json:"updatedAt"`
		ReviewDecision string    `json:"reviewDecision"`
	}

	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, fmt.Errorf("analyze PRs: parse response: %w", err)
	}

	now := time.Now().UTC()
	staleThreshold := now.AddDate(0, 0, -stalenessThreshold)
	recentThreshold := now.AddDate(0, 0, -7)

	byReviewState := make(map[string]int)
	var stale []StalePR
	recentActivity := 0

	for _, pr := range prs {
		// Categorize review state.
		state := pr.ReviewDecision
		if state == "" {
			state = "REVIEW_REQUIRED"
		}
		byReviewState[state]++

		// Check for staleness.
		if pr.UpdatedAt.Before(staleThreshold) {
			daysSince := int(now.Sub(pr.UpdatedAt).Hours() / 24)
			stale = append(stale, StalePR{
				Number:          pr.Number,
				Title:           pr.Title,
				Author:          pr.Author.Login,
				DaysSinceUpdate: daysSince,
			})
		}

		// Check for recent activity.
		if pr.UpdatedAt.After(recentThreshold) {
			recentActivity++
		}
	}

	// Sort stale PRs by days since update descending for readability.
	sort.Slice(stale, func(i, j int) bool {
		return stale[i].DaysSinceUpdate > stale[j].DaysSinceUpdate
	})

	return &PRSummary{
		TotalOpen:      len(prs),
		ByReviewState:  byReviewState,
		Stale:          stale,
		RecentActivity: recentActivity,
	}, nil
}

// AnalyzeIssues retrieves open issues and categorizes them by labels and
// priority, identifying actionable items ready for immediate work.
func (a *GitHubAnalyzer) AnalyzeIssues(ctx context.Context, repo string) (*IssueSummary, error) {
	maxItems := a.opts.MaxItems
	if maxItems <= 0 {
		maxItems = DefaultMaxItems
	}

	data, err := a.runGH(ctx,
		"issue", "list",
		"--repo", repo,
		"--state", "open",
		"--json", "number,title,labels,updatedAt",
		"--limit", fmt.Sprintf("%d", maxItems),
	)
	if err != nil {
		return nil, fmt.Errorf("analyze issues: %w", err)
	}

	var issues []struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		Labels    []struct {
			Name string `json:"name"`
		} `json:"labels"`
		UpdatedAt time.Time `json:"updatedAt"`
	}

	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, fmt.Errorf("analyze issues: parse response: %w", err)
	}

	byCategory := make(map[string]int)
	byPriority := make(map[string]int)
	var actionable []ActionableIssue

	// actionableLabels are labels that indicate an issue is ready for work.
	actionableLabels := map[string]bool{
		"bug":         true,
		"enhancement": true,
		"feature":     true,
	}

	for _, issue := range issues {
		labelNames := make([]string, 0, len(issue.Labels))
		isActionable := false
		priority := ""

		for _, l := range issue.Labels {
			name := l.Name
			labelNames = append(labelNames, name)
			byCategory[name]++

			// Extract priority from labels containing "priority".
			lower := strings.ToLower(name)
			if strings.Contains(lower, "priority") {
				// Extract the priority level: "priority: high" -> "high",
				// "priority/critical" -> "critical", etc.
				p := extractPriority(name)
				if p != "" {
					priority = p
					byPriority[p]++
				}
			}

			// Check if this label marks the issue as actionable.
			if actionableLabels[strings.ToLower(name)] {
				isActionable = true
			}
		}

		if isActionable {
			actionable = append(actionable, ActionableIssue{
				Number:   issue.Number,
				Title:    issue.Title,
				Labels:   labelNames,
				Priority: priority,
			})
		}
	}

	return &IssueSummary{
		TotalOpen:  len(issues),
		ByCategory: byCategory,
		ByPriority: byPriority,
		Actionable: actionable,
	}, nil
}

// extractPriority parses a priority label and returns the priority level.
// It handles formats like "priority: high", "priority/critical",
// "priority-low", etc.
func extractPriority(label string) string {
	lower := strings.ToLower(label)

	// Try common separators: ": ", ":", "/", "-", " ".
	for _, sep := range []string{": ", ":", "/", "-", " "} {
		idx := strings.Index(lower, "priority"+sep)
		if idx >= 0 {
			p := strings.TrimSpace(lower[idx+len("priority"+sep):])
			if p != "" {
				return p
			}
		}
	}

	// If the label is exactly "priority" with no value, skip.
	return ""
}

// AnalyzeCIStatus fetches recent CI workflow runs and computes pass rate
// and last run information.
func (a *GitHubAnalyzer) AnalyzeCIStatus(ctx context.Context, repo string) (*CIStatus, error) {
	endpoint := fmt.Sprintf("repos/%s/actions/runs?per_page=20&status=completed", repo)
	data, err := a.runGH(ctx, "api", endpoint)
	if err != nil {
		return nil, fmt.Errorf("analyze CI status: %w", err)
	}

	var response struct {
		TotalCount   int `json:"total_count"`
		WorkflowRuns []struct {
			Conclusion string    `json:"conclusion"`
			CreatedAt  time.Time `json:"created_at"`
			Status     string    `json:"status"`
		} `json:"workflow_runs"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("analyze CI status: parse response: %w", err)
	}

	runs := response.WorkflowRuns
	if len(runs) == 0 {
		return &CIStatus{
			RecentRuns:    0,
			PassRate:      0,
			LastRunStatus: "",
			LastRunAt:     nil,
		}, nil
	}

	successCount := 0
	for _, run := range runs {
		if run.Conclusion == "success" {
			successCount++
		}
	}

	passRate := math.Round(float64(successCount)/float64(len(runs))*10000) / 100

	// Runs are returned newest first by the API.
	lastRun := runs[0]
	lastRunAt := lastRun.CreatedAt

	return &CIStatus{
		RecentRuns:    len(runs),
		PassRate:      passRate,
		LastRunStatus: lastRun.Conclusion,
		LastRunAt:     &lastRunAt,
	}, nil
}

// Compile-time interface assertion.
var _ ForgeAnalyzer = (*GitHubAnalyzer)(nil)
