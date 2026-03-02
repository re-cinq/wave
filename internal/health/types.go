package health

import (
	"context"
	"time"
)

// Configuration defaults for health analysis.
const (
	DefaultCommitWindowDays       = 30
	DefaultStalenessThresholdDays = 14
	DefaultMaxItems               = 100
)

// ForgeType identifies the source code forge hosting a repository.
type ForgeType string

const (
	GitHub    ForgeType = "github"
	GitLab    ForgeType = "gitlab"
	Bitbucket ForgeType = "bitbucket"
	Gitea     ForgeType = "gitea"
	Unknown   ForgeType = "unknown"
)

// ForgeAnalyzer defines the interface for forge-specific repository analysis.
type ForgeAnalyzer interface {
	AnalyzeCommits(ctx context.Context, repo string) (*CommitAnalysis, error)
	AnalyzePRs(ctx context.Context, repo string) (*PRSummary, error)
	AnalyzeIssues(ctx context.Context, repo string) (*IssueSummary, error)
	AnalyzeCIStatus(ctx context.Context, repo string) (*CIStatus, error)
}

// HealthReport is the top-level artifact produced by health analysis.
type HealthReport struct {
	ForgeType    ForgeType       `json:"forge_type"`
	Repository   string          `json:"repository"`
	AnalyzedAt   time.Time       `json:"analyzed_at"`
	Commits      *CommitAnalysis `json:"commits"`
	PullRequests *PRSummary      `json:"pull_requests"`
	Issues       *IssueSummary   `json:"issues"`
	CIStatus     *CIStatus       `json:"ci_status,omitempty"`
}

// CommitAnalysis summarises recent commit activity within a configurable window.
type CommitAnalysis struct {
	TotalCount      int              `json:"total_count"`
	WindowDays      int              `json:"window_days"`
	Authors         []AuthorActivity `json:"authors"`
	AreasOfActivity []string         `json:"areas_of_activity"`
	FrequencyPerDay float64          `json:"frequency_per_day"`
}

// AuthorActivity records the number of commits attributed to an individual author.
type AuthorActivity struct {
	Name        string `json:"name"`
	CommitCount int    `json:"commit_count"`
}

// PRSummary describes the state of open pull requests.
type PRSummary struct {
	TotalOpen      int            `json:"total_open"`
	ByReviewState  map[string]int `json:"by_review_state"`
	Stale          []StalePR      `json:"stale"`
	RecentActivity int            `json:"recent_activity"` // PRs with comments in last 7 days
}

// StalePR identifies a pull request that has not been updated recently.
type StalePR struct {
	Number          int    `json:"number"`
	Title           string `json:"title"`
	Author          string `json:"author"`
	DaysSinceUpdate int    `json:"days_since_update"`
}

// IssueSummary categorises open issues for prioritisation.
type IssueSummary struct {
	TotalOpen  int               `json:"total_open"`
	ByCategory map[string]int    `json:"by_category"`
	ByPriority map[string]int    `json:"by_priority"`
	Actionable []ActionableIssue `json:"actionable"`
}

// ActionableIssue represents an issue considered ready for immediate work.
type ActionableIssue struct {
	Number   int      `json:"number"`
	Title    string   `json:"title"`
	Labels   []string `json:"labels"`
	Priority string   `json:"priority"`
}

// CIStatus captures continuous integration health signals.
type CIStatus struct {
	RecentRuns    int        `json:"recent_runs"`
	PassRate      float64    `json:"pass_rate"`
	LastRunStatus string     `json:"last_run_status"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
}

// AnalyzeOptions controls the behaviour of the health analysis orchestrator.
type AnalyzeOptions struct {
	CommitWindowDays       int
	StalenessThresholdDays int
	MaxItems               int
}
