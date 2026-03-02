package health

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCmdRunner returns a cmdRunner function that, when called, returns an
// exec.Cmd that will output predetermined data on stdout based on argument
// pattern matching. The outputs map keys are substrings matched against the
// joined argument list; the first match wins.
func mockCmdRunner(outputs map[string]string) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		key := strings.Join(args, " ")
		for pattern, output := range outputs {
			if strings.Contains(key, pattern) {
				return exec.CommandContext(ctx, "printf", "%s", output)
			}
		}
		// Return a command that fails for unmatched patterns.
		return exec.CommandContext(ctx, "false")
	}
}

func TestGitHubAnalyzerAnalyzeCommits(t *testing.T) {
	commitJSON := `[
		{"sha":"abc123","commit":{"author":{"name":"Alice","date":"2025-01-15T10:00:00Z"},"message":"feat(pipeline): add new step"}},
		{"sha":"def456","commit":{"author":{"name":"Bob","date":"2025-01-14T10:00:00Z"},"message":"fix(contract): validation bug"}},
		{"sha":"ghi789","commit":{"author":{"name":"Alice","date":"2025-01-13T10:00:00Z"},"message":"refactor: cleanup"}}
	]`

	analyzer := &GitHubAnalyzer{
		cmdRunner: mockCmdRunner(map[string]string{
			"api repos/": commitJSON,
		}),
		repoPath: t.TempDir(),
		opts:     AnalyzeOptions{CommitWindowDays: 30, MaxItems: 100},
	}

	result, err := analyzer.AnalyzeCommits(context.Background(), "owner/repo")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 3, result.TotalCount, "should have 3 commits")
	assert.Equal(t, 30, result.WindowDays, "window should be 30 days")

	// Authors: Alice=2, Bob=1 — sorted by count descending.
	require.Len(t, result.Authors, 2, "should have 2 distinct authors")
	assert.Equal(t, "Alice", result.Authors[0].Name)
	assert.Equal(t, 2, result.Authors[0].CommitCount)
	assert.Equal(t, "Bob", result.Authors[1].Name)
	assert.Equal(t, 1, result.Authors[1].CommitCount)

	// Areas: "contract", "pipeline", "uncategorized" (from "refactor: cleanup" which has no scope).
	assert.Contains(t, result.AreasOfActivity, "pipeline")
	assert.Contains(t, result.AreasOfActivity, "contract")
	assert.Contains(t, result.AreasOfActivity, "uncategorized")

	// FrequencyPerDay: 3 commits / 30 days = 0.10
	assert.Equal(t, 0.10, result.FrequencyPerDay)
}

func TestGitHubAnalyzerAnalyzePRs(t *testing.T) {
	recentTime := time.Now().UTC().Add(-1 * 24 * time.Hour).Format(time.RFC3339)
	staleTime1 := time.Now().UTC().Add(-60 * 24 * time.Hour).Format(time.RFC3339)
	staleTime2 := time.Now().UTC().Add(-45 * 24 * time.Hour).Format(time.RFC3339)

	prJSON := fmt.Sprintf(`[
		{"number":1,"title":"Add feature","author":{"login":"alice"},"updatedAt":"%s","reviewDecision":"APPROVED"},
		{"number":2,"title":"Fix bug","author":{"login":"bob"},"updatedAt":"%s","reviewDecision":"CHANGES_REQUESTED"},
		{"number":3,"title":"Docs update","author":{"login":"charlie"},"updatedAt":"%s","reviewDecision":""}
	]`, recentTime, staleTime1, staleTime2)

	analyzer := &GitHubAnalyzer{
		cmdRunner: mockCmdRunner(map[string]string{
			"pr list": prJSON,
		}),
		repoPath: t.TempDir(),
		opts:     AnalyzeOptions{MaxItems: 100, StalenessThresholdDays: 14},
	}

	result, err := analyzer.AnalyzePRs(context.Background(), "owner/repo")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 3, result.TotalOpen, "should have 3 open PRs")

	// Review state breakdown.
	assert.Equal(t, 1, result.ByReviewState["APPROVED"])
	assert.Equal(t, 1, result.ByReviewState["CHANGES_REQUESTED"])
	assert.Equal(t, 1, result.ByReviewState["REVIEW_REQUIRED"], "empty reviewDecision should map to REVIEW_REQUIRED")

	// Stale PRs: #2 and #3 are both older than 14 days.
	require.Len(t, result.Stale, 2, "should have 2 stale PRs")
	// Stale PRs are sorted by DaysSinceUpdate descending, so #2 (60 days) comes first.
	assert.Equal(t, 2, result.Stale[0].Number)
	assert.Equal(t, "bob", result.Stale[0].Author)
	assert.Greater(t, result.Stale[0].DaysSinceUpdate, 50)
	assert.Equal(t, 3, result.Stale[1].Number)
	assert.Equal(t, "charlie", result.Stale[1].Author)
	assert.Greater(t, result.Stale[1].DaysSinceUpdate, 35)

	// Recent activity: only #1 (updated 1 day ago) is within the 7-day window.
	assert.Equal(t, 1, result.RecentActivity, "only 1 PR should have recent activity")
}

func TestGitHubAnalyzerAnalyzeIssues(t *testing.T) {
	issueJSON := `[
		{"number":10,"title":"Critical bug","labels":[{"name":"bug"},{"name":"priority: high"}],"updatedAt":"2025-01-15T10:00:00Z"},
		{"number":11,"title":"New feature","labels":[{"name":"enhancement"}],"updatedAt":"2025-01-14T10:00:00Z"},
		{"number":12,"title":"Question","labels":[{"name":"question"}],"updatedAt":"2025-01-13T10:00:00Z"}
	]`

	analyzer := &GitHubAnalyzer{
		cmdRunner: mockCmdRunner(map[string]string{
			"issue list": issueJSON,
		}),
		repoPath: t.TempDir(),
		opts:     AnalyzeOptions{MaxItems: 100},
	}

	result, err := analyzer.AnalyzeIssues(context.Background(), "owner/repo")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 3, result.TotalOpen, "should have 3 open issues")

	// By category: bug=1, enhancement=1, question=1, "priority: high"=1.
	assert.Equal(t, 1, result.ByCategory["bug"])
	assert.Equal(t, 1, result.ByCategory["enhancement"])
	assert.Equal(t, 1, result.ByCategory["question"])

	// By priority: "high"=1.
	assert.Equal(t, 1, result.ByPriority["high"])

	// Actionable: #10 (bug) and #11 (enhancement) are actionable.
	require.Len(t, result.Actionable, 2, "should have 2 actionable issues")
	assert.Equal(t, 10, result.Actionable[0].Number)
	assert.Equal(t, "Critical bug", result.Actionable[0].Title)
	assert.Equal(t, "high", result.Actionable[0].Priority)
	assert.ElementsMatch(t, []string{"bug", "priority: high"}, result.Actionable[0].Labels)
	assert.Equal(t, 11, result.Actionable[1].Number)
	assert.Equal(t, "New feature", result.Actionable[1].Title)
	assert.Empty(t, result.Actionable[1].Priority, "issue without priority label should have empty priority")
}

func TestGitHubAnalyzerAnalyzeCIStatus(t *testing.T) {
	ciJSON := `{"workflow_runs":[
		{"id":1,"conclusion":"success","created_at":"2025-01-15T10:00:00Z"},
		{"id":2,"conclusion":"failure","created_at":"2025-01-14T10:00:00Z"},
		{"id":3,"conclusion":"success","created_at":"2025-01-13T10:00:00Z"}
	]}`

	analyzer := &GitHubAnalyzer{
		cmdRunner: mockCmdRunner(map[string]string{
			"api repos/": ciJSON,
		}),
		repoPath: t.TempDir(),
		opts:     AnalyzeOptions{},
	}

	result, err := analyzer.AnalyzeCIStatus(context.Background(), "owner/repo")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 3, result.RecentRuns, "should have 3 recent runs")
	assert.InDelta(t, 66.67, result.PassRate, 0.01, "pass rate should be ~66.67%")
	assert.Equal(t, "success", result.LastRunStatus, "last run should be success")
	require.NotNil(t, result.LastRunAt, "last run time should not be nil")
}

func TestGitHubAnalyzerEmptyResults(t *testing.T) {
	analyzer := &GitHubAnalyzer{
		cmdRunner: mockCmdRunner(map[string]string{
			"api repos/": `[]`,
			"pr list":    `[]`,
			"issue list": `[]`,
		}),
		repoPath: t.TempDir(),
		opts:     AnalyzeOptions{CommitWindowDays: 30, MaxItems: 100, StalenessThresholdDays: 14},
	}

	ctx := context.Background()
	repo := "owner/repo"

	t.Run("empty commits", func(t *testing.T) {
		result, err := analyzer.AnalyzeCommits(ctx, repo)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, result.TotalCount)
		assert.Empty(t, result.Authors)
		assert.Empty(t, result.AreasOfActivity)
		assert.Equal(t, 0.0, result.FrequencyPerDay)
	})

	t.Run("empty PRs", func(t *testing.T) {
		result, err := analyzer.AnalyzePRs(ctx, repo)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, result.TotalOpen)
		assert.Empty(t, result.ByReviewState)
		assert.Empty(t, result.Stale)
		assert.Equal(t, 0, result.RecentActivity)
	})

	t.Run("empty issues", func(t *testing.T) {
		result, err := analyzer.AnalyzeIssues(ctx, repo)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, result.TotalOpen)
		assert.Empty(t, result.ByCategory)
		assert.Empty(t, result.ByPriority)
		assert.Empty(t, result.Actionable)
	})

	t.Run("empty CI runs", func(t *testing.T) {
		// CI endpoint returns an object, not an array. Empty runs = empty array within.
		ciAnalyzer := &GitHubAnalyzer{
			cmdRunner: mockCmdRunner(map[string]string{
				"api repos/": `{"workflow_runs":[]}`,
			}),
			repoPath: t.TempDir(),
			opts:     AnalyzeOptions{},
		}

		result, err := ciAnalyzer.AnalyzeCIStatus(ctx, repo)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, result.RecentRuns)
		assert.Equal(t, 0.0, result.PassRate)
		assert.Empty(t, result.LastRunStatus)
		assert.Nil(t, result.LastRunAt)
	})
}
