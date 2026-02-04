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
	Tags         []string // Tags for categorization and filtering
}

// ListRunsOptions specifies filters for listing runs.
type ListRunsOptions struct {
	PipelineName string
	Status       string
	OlderThan    time.Duration
	Limit        int
	Tags         []string // Filter runs that have any of these tags
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
	StepID              string
	Persona             string
	TotalRuns           int
	SuccessfulRuns      int
	FailedRuns          int
	AvgDurationMs       int64
	MinDurationMs       int64
	MaxDurationMs       int64
	AvgTokensUsed       int
	TotalTokensUsed     int
	AvgFilesModified    int
	AvgArtifacts        int
	LastRunAt           time.Time
	TokenBurnRate       float64 // tokens per second
}

// ProgressSnapshotRecord holds a point-in-time progress snapshot.
type ProgressSnapshotRecord struct {
	ID              int64
	RunID           string
	StepID          string
	Timestamp       time.Time
	Progress        int
	CurrentAction   string
	EstimatedTimeMs int64
	ValidationPhase string
	CompactionStats string
}

// StepProgressRecord holds real-time step progress information.
type StepProgressRecord struct {
	StepID                string
	RunID                 string
	Persona               string
	State                 string
	Progress              int
	CurrentAction         string
	Message               string
	StartedAt             *time.Time
	UpdatedAt             time.Time
	EstimatedCompletionMs int64
	TokensUsed            int
}

// PipelineProgressRecord holds pipeline-level progress aggregation.
type PipelineProgressRecord struct {
	RunID                 string
	TotalSteps            int
	CompletedSteps        int
	CurrentStepIndex      int
	OverallProgress       int
	EstimatedCompletionMs int64
	UpdatedAt             time.Time
}

// ArtifactMetadataRecord holds extended artifact metadata for visualization.
type ArtifactMetadataRecord struct {
	ArtifactID   int64
	RunID        string
	StepID       string
	PreviewText  string
	MimeType     string
	Encoding     string
	MetadataJSON string
	IndexedAt    time.Time
}
