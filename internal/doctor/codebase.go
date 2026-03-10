package doctor

import (
	"context"
	"encoding/json"
	"os/exec"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/github"
)

// CodebaseHealth aggregates forge-API-sourced codebase metrics.
type CodebaseHealth struct {
	PRs    PRSummary    `json:"prs"`
	Issues IssueSummary `json:"issues"`
	CI     CIStatus     `json:"ci"`
}

// PRSummary summarizes open pull request state.
type PRSummary struct {
	Open        int `json:"open"`
	NeedsReview int `json:"needs_review"`
	Stale       int `json:"stale"`
}

// IssueSummary summarizes open issue state.
type IssueSummary struct {
	Open        int `json:"open"`
	PoorQuality int `json:"poor_quality"`
	Unassigned  int `json:"unassigned"`
}

// CIStatus summarizes recent CI run results.
type CIStatus struct {
	Status     string `json:"status"` // "passing", "failing", "unknown"
	RecentRuns int    `json:"recent_runs"`
	Failures   int    `json:"failures"`
}

// CodebaseOptions configures codebase health analysis.
type CodebaseOptions struct {
	ForgeInfo forge.ForgeInfo
	GHClient  *github.Client

	// RunGHCmd overrides gh CLI execution for testing.
	RunGHCmd func(args ...string) ([]byte, error)
	// Now overrides the current time for testing.
	Now func() time.Time
}

func (o *CodebaseOptions) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o *CodebaseOptions) runGHCmd(args ...string) ([]byte, error) {
	if o.RunGHCmd != nil {
		return o.RunGHCmd(args...)
	}
	return exec.Command("gh", args...).Output()
}

// AnalyzeCodebase fetches codebase metrics from the forge API.
// Returns nil, nil for non-GitHub forges (no error, no data).
func AnalyzeCodebase(ctx context.Context, opts CodebaseOptions) (*CodebaseHealth, error) {
	if opts.ForgeInfo.Type != forge.ForgeGitHub {
		return nil, nil
	}

	health := &CodebaseHealth{}

	// Analyze PRs
	if opts.GHClient != nil {
		prs, err := opts.GHClient.ListPullRequests(ctx, opts.ForgeInfo.Owner, opts.ForgeInfo.Repo, github.ListPullRequestsOptions{
			State:   "open",
			PerPage: 100,
		})
		if err == nil {
			health.PRs = analyzePRs(prs, opts.now())
		}

		// Analyze issues (list issues API includes PRs, we filter them out)
		issues, err := opts.GHClient.ListIssues(ctx, opts.ForgeInfo.Owner, opts.ForgeInfo.Repo, github.ListIssuesOptions{
			State:   "open",
			PerPage: 100,
		})
		if err == nil {
			health.Issues = analyzeIssues(ctx, issues, opts.GHClient)
		}
	}

	// Analyze CI via gh CLI (more reliable than API for workflow runs)
	health.CI = analyzeCIStatus(opts)

	return health, nil
}

func analyzePRs(prs []*github.PullRequest, now time.Time) PRSummary {
	summary := PRSummary{Open: len(prs)}
	staleThreshold := now.AddDate(0, 0, -14)

	for _, pr := range prs {
		if pr.UpdatedAt.Before(staleThreshold) {
			summary.Stale++
		}
		// PRs with 0 comments are likely awaiting review
		if pr.Comments == 0 {
			summary.NeedsReview++
		}
	}
	return summary
}

func analyzeIssues(ctx context.Context, issues []*github.Issue, client *github.Client) IssueSummary {
	var summary IssueSummary

	analyzer := github.NewAnalyzer(client)

	for _, issue := range issues {
		if issue.IsPullRequest() {
			continue
		}
		summary.Open++

		if len(issue.Assignees) == 0 {
			summary.Unassigned++
		}

		analysis := analyzer.AnalyzeIssue(ctx, issue)
		if analysis.QualityScore < 50 {
			summary.PoorQuality++
		}
	}
	return summary
}

// ghRunResult matches the JSON output of `gh run list --json`.
type ghRunResult struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

func analyzeCIStatus(opts CodebaseOptions) CIStatus {
	out, err := opts.runGHCmd("run", "list", "--limit", "5", "--json", "status,conclusion")
	if err != nil {
		return CIStatus{Status: "unknown"}
	}

	var runs []ghRunResult
	if err := json.Unmarshal(out, &runs); err != nil {
		return CIStatus{Status: "unknown"}
	}

	if len(runs) == 0 {
		return CIStatus{Status: "unknown"}
	}

	ci := CIStatus{
		RecentRuns: len(runs),
		Status:     "passing",
	}

	for _, run := range runs {
		if run.Conclusion == "failure" {
			ci.Failures++
		}
	}

	if ci.Failures > 0 {
		ci.Status = "failing"
	}

	return ci
}
