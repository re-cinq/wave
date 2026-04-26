package doctor

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/github"
	"github.com/recinq/wave/internal/suggest"
)

// CodebaseOptions configures codebase health analysis.
type CodebaseOptions struct {
	ForgeInfo   forge.ForgeInfo
	ForgeClient forge.Client

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
func AnalyzeCodebase(ctx context.Context, opts CodebaseOptions) (*suggest.CodebaseHealth, error) {
	if opts.ForgeInfo.Type != forge.ForgeGitHub {
		return nil, nil
	}

	health := &suggest.CodebaseHealth{}

	if opts.ForgeClient != nil {
		prs, err := opts.ForgeClient.ListPullRequests(ctx, opts.ForgeInfo.Owner, opts.ForgeInfo.Repo, forge.ListPullRequestsOptions{
			State:   "open",
			PerPage: 100,
		})
		if err != nil {
			log.Printf("[doctor] failed to list pull requests: %v", err)
		} else {
			health.PRs = analyzePRs(prs, opts.now())
		}

		issues, err := opts.ForgeClient.ListIssues(ctx, opts.ForgeInfo.Owner, opts.ForgeInfo.Repo, forge.ListIssuesOptions{
			State:   "open",
			PerPage: 100,
		})
		if err != nil {
			log.Printf("[doctor] failed to list issues: %v", err)
		} else {
			health.Issues = analyzeIssues(ctx, issues, opts.ForgeClient)
		}
	}

	// Analyze CI via gh CLI (more reliable than API for workflow runs)
	health.CI = analyzeCIStatus(opts)

	return health, nil
}

func analyzePRs(prs []*forge.PullRequest, now time.Time) suggest.PRSummary {
	summary := suggest.PRSummary{Open: len(prs)}
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

func analyzeIssues(ctx context.Context, issues []*forge.Issue, client forge.Client) suggest.IssueSummary {
	var summary suggest.IssueSummary

	// Issue quality analysis requires the underlying GitHub client
	var analyzer *github.Analyzer
	if gc, ok := client.(*forge.GitHubClient); ok {
		analyzer = github.NewAnalyzer(gc.UnwrapGitHub())
	}

	for _, issue := range issues {
		if issue.IsPR {
			continue
		}
		summary.Open++

		if len(issue.Assignees) == 0 {
			summary.Unassigned++
		}

		if analyzer != nil {
			// Convert forge.Issue back to github.Issue for the analyzer
			ghIssue := forgeIssueToGitHub(issue)
			analysis := analyzer.AnalyzeIssue(ctx, ghIssue)
			if analysis.QualityScore < 50 {
				summary.PoorQuality++
			}
		}
	}
	return summary
}

// forgeIssueToGitHub converts a forge.Issue to a github.Issue for GitHub-specific analysis.
func forgeIssueToGitHub(fi *forge.Issue) *github.Issue {
	gi := &github.Issue{
		Number:    fi.Number,
		Title:     fi.Title,
		Body:      fi.Body,
		State:     fi.State,
		Comments:  fi.Comments,
		CreatedAt: fi.CreatedAt,
		UpdatedAt: fi.UpdatedAt,
		ClosedAt:  fi.ClosedAt,
		HTMLURL:   fi.HTMLURL,
	}
	if fi.Author != "" {
		gi.User = &github.User{Login: fi.Author}
	}
	for _, l := range fi.Labels {
		gi.Labels = append(gi.Labels, &github.Label{Name: l.Name, Color: l.Color})
	}
	for _, login := range fi.Assignees {
		gi.Assignees = append(gi.Assignees, &github.User{Login: login})
	}
	return gi
}

// ghRunResult matches the JSON output of `gh run list --json`.
type ghRunResult struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

func analyzeCIStatus(opts CodebaseOptions) suggest.CIStatus {
	out, err := opts.runGHCmd("run", "list", "--limit", "5", "--json", "status,conclusion")
	if err != nil {
		return suggest.CIStatus{Status: "unknown"}
	}

	var runs []ghRunResult
	if err := json.Unmarshal(out, &runs); err != nil {
		return suggest.CIStatus{Status: "unknown"}
	}

	if len(runs) == 0 {
		return suggest.CIStatus{Status: "unknown"}
	}

	ci := suggest.CIStatus{
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
