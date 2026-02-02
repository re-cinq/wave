package state

import "time"

// RunRecord holds a pipeline run record.
type RunRecord struct {
	RunID        string
	PipelineName string
	Status       string
	Input        string
	CurrentStep  string
	TotalTokens  int
	StartedAt    time.Time
	CompletedAt  *time.Time
	CancelledAt  *time.Time
	ErrorMessage string
}

// ListRunsOptions specifies filters for listing runs.
type ListRunsOptions struct {
	PipelineName string
	Status       string
	OlderThan    time.Duration
	Limit        int
}

// LogRecord holds an event log entry.
type LogRecord struct {
	ID         int64
	RunID      string
	Timestamp  time.Time
	StepID     string
	State      string
	Persona    string
	Message    string
	TokensUsed int
	DurationMs int64
}

// EventQueryOptions specifies filters for log queries.
type EventQueryOptions struct {
	StepID     string
	ErrorsOnly bool
	Limit      int
	Offset     int
}

// ArtifactRecord holds artifact metadata.
type ArtifactRecord struct {
	ID        int64
	RunID     string
	StepID    string
	Name      string
	Path      string
	Type      string
	SizeBytes int64
	CreatedAt time.Time
}

// CancellationRecord holds cancellation request info.
type CancellationRecord struct {
	RunID       string
	RequestedAt time.Time
	Force       bool
}
