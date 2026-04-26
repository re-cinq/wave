package suggest

import (
	"github.com/recinq/wave/internal/forge"
)

// Status represents the severity of a check result.
type Status int

const (
	StatusOK Status = iota
	StatusWarn
	StatusErr
)

// String returns a human-readable label for the status.
func (s Status) String() string {
	switch s {
	case StatusOK:
		return "ok"
	case StatusWarn:
		return "warn"
	case StatusErr:
		return "error"
	default:
		return "unknown"
	}
}

// MarshalJSON implements json.Marshaler for Status.
func (s Status) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// CheckResult represents the outcome of a single health check.
type CheckResult struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Status   Status `json:"status"`
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
}

// Report aggregates all check results and forge detection info.
type Report struct {
	Results   []CheckResult    `json:"results"`
	Summary   Status           `json:"summary"`
	ForgeInfo *forge.ForgeInfo `json:"forge,omitempty"`
	Codebase  *CodebaseHealth  `json:"codebase,omitempty"`
}

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
	Status     string `json:"status"`
	RecentRuns int    `json:"recent_runs"`
	Failures   int    `json:"failures"`
}

// HealthCheckMsg carries the result of an async health check (Bubbletea msg form).
type HealthCheckMsg struct {
	Name    string
	Status  Status
	Message string
	Details map[string]string
}
