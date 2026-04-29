// Package metrics owns persistence and queries for pipeline performance
// metrics and retrospectives. It was extracted from internal/state so the
// state package can stay focused on run/step lifecycle (issue #62).
//
// The metrics package reads and writes the `performance_metric` and
// `retrospective` tables. Schema migrations for those tables still live in
// internal/state's migration runner — there is a single migration runner per
// database and the metrics package is intentionally a query layer only.
package metrics

import "time"

// PerformanceMetricRecord holds historical performance data for a step.
type PerformanceMetricRecord struct {
	ID                 int64
	RunID              string
	StepID             string
	PipelineName       string
	Persona            string
	StartedAt          time.Time
	CompletedAt        *time.Time
	DurationMs         int64
	TokensUsed         int
	FilesModified      int
	ArtifactsGenerated int
	MemoryBytes        int64
	Success            bool
	ErrorMessage       string
}

// PerformanceQueryOptions specifies filters for performance queries.
type PerformanceQueryOptions struct {
	PipelineName string
	StepID       string
	Persona      string
	Since        time.Time
	Limit        int
}

// StepPerformanceStats holds aggregated performance statistics for a step.
type StepPerformanceStats struct {
	StepID           string
	Persona          string
	TotalRuns        int
	SuccessfulRuns   int
	FailedRuns       int
	AvgDurationMs    int64
	MinDurationMs    int64
	MaxDurationMs    int64
	AvgTokensUsed    int
	TotalTokensUsed  int
	AvgFilesModified int
	AvgArtifacts     int
	LastRunAt        time.Time
	TokenBurnRate    float64 // tokens per second
}

// RetrospectiveRecord holds metadata for a stored retrospective.
type RetrospectiveRecord struct {
	ID           int64
	RunID        string
	PipelineName string
	Smoothness   string
	Status       string // "quantitative", "complete" (has narrative)
	FilePath     string
	CreatedAt    time.Time
}

// ListRetrosOptions specifies filters for listing retrospectives.
type ListRetrosOptions struct {
	PipelineName string
	SinceUnix    int64
	Limit        int
}
