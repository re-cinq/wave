package state

import (
	"time"
)

// RunRecord holds a pipeline run record.
type RunRecord struct {
	RunID           string
	PipelineName    string
	Status          string
	Input           string
	CurrentStep     string
	TotalTokens     int
	StartedAt       time.Time
	CompletedAt     *time.Time
	CancelledAt     *time.Time
	ErrorMessage    string
	Tags            []string // Tags for categorization and filtering
	BranchName      string   // Worktree branch for this run
	PID             int      // OS process ID of the detached executor (0 = unknown/in-process)
	LastHeartbeat   time.Time // Last liveness ping written by the running pipeline (zero = never reported)
	ParentRunID     string   // Parent pipeline run ID (empty for top-level runs)
	ParentStepID    string   // Step ID in parent pipeline that launched this child run
	ForkedFromRunID string   // Run ID this was forked from (empty if not a fork)

	// Composition metadata (issue #1450). Set when a parent composition
	// step (iterate, aggregate, sub_pipeline, branch, loop) launches
	// this run; lets the WebUI render iterate progress, sub-pipeline
	// breadcrumbs, and run-kind chips without re-deriving from event_log.
	IterateIndex   *int   // 0-based index within an iterate step's items array (nil for non-iterate launches)
	IterateTotal   *int   // total items in the parent iterate step (nil for non-iterate launches)
	IterateMode    string // "parallel" or "serial" (empty for non-iterate launches)
	RunKind        string // "top_level" | "iterate_child" | "sub_pipeline_child" | "loop_iteration" | "branch_arm" | "resume" (empty defaults to top_level for legacy rows)
	SubPipelineRef string // Sub-pipeline name as referenced in YAML (e.g. "audit-security"); empty for top-level runs
}

// RunKind enum values for pipeline_run.run_kind. Issue #1450 introduced
// composition kinds; #1510 added "resume" so the WebUI can link a resumed
// run back to the failed parent run.
const (
	RunKindTopLevel         = "top_level"
	RunKindIterateChild     = "iterate_child"
	RunKindSubPipelineChild = "sub_pipeline_child"
	RunKindLoopIteration    = "loop_iteration"
	RunKindBranchArm        = "branch_arm"
	// RunKindResume marks a run that was launched via `wave resume <id>` /
	// `wave run --from-step ... --run <id>` / web UI resume button. The
	// `parent_run_id` column points at the original failed run.
	RunKindResume = "resume"
)

// CheckpointRecord holds checkpoint data at a step boundary for fork/rewind.
type CheckpointRecord struct {
	ID                 int64
	RunID              string
	StepID             string
	StepIndex          int
	WorkspacePath      string
	WorkspaceCommitSHA string // Git HEAD SHA for worktree workspaces (empty for non-worktree)
	ArtifactSnapshot   string // JSON map of "stepID:name" -> path for all artifacts at this point
	CreatedAt          time.Time
}

// ListRunsOptions specifies filters for listing runs.
type ListRunsOptions struct {
	PipelineName string
	Status       string
	OlderThan    time.Duration
	Limit        int
	Tags         []string // Filter runs that have any of these tags
	BeforeUnix   int64    // Cursor: only return runs started before this unix timestamp
	BeforeRunID  string   // Cursor: tie-break for runs at the same timestamp
	SinceUnix    int64    // Only return runs started after this unix timestamp
	TopLevelOnly bool     // Only return top-level runs (parent_run_id IS NULL OR ''). Issue #1450 — keeps composition children out of pipeline detail recent-runs lists.
}

// LogRecord holds an event log entry.
type LogRecord struct {
	ID              int64
	RunID           string
	Timestamp       time.Time
	StepID          string
	State           string
	Persona         string
	Message         string
	TokensUsed      int
	DurationMs      int64
	Model           string
	ConfiguredModel string // Tier name from pipeline config (e.g. "cheapest")
	Adapter         string
}

// EventQueryOptions specifies filters for log queries.
type EventQueryOptions struct {
	StepID     string
	ErrorsOnly bool
	Limit      int
	Offset     int
	AfterID    int64 // Filter events with ID > AfterID (for SSE Last-Event-ID backfill)
	SinceUnix  int64 // Only return events with timestamp >= SinceUnix
	TailLimit  int   // Return the most recent N events (applied via DESC + reverse)
	OrderDesc  bool  // Order by timestamp DESC, id DESC instead of ASC
}

// DecisionQueryOptions specifies filters for decision-log queries.
type DecisionQueryOptions struct {
	StepID   string
	Category string
}

// EventAggregateStats holds aggregate metrics over a run's event log,
// limited to events in completed/failed terminal states.
type EventAggregateStats struct {
	TotalEvents   int
	TotalTokens   int
	AvgDurationMs float64
	MinDurationMs float64
	MaxDurationMs float64
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

// StepAttemptRecord holds an individual retry attempt for a pipeline step.
type StepAttemptRecord struct {
	ID           int64
	RunID        string
	StepID       string
	Attempt      int
	State        string // "failed", "succeeded"
	ErrorMessage string
	FailureClass string
	StdoutTail   string
	TokensUsed   int
	DurationMs   int64
	StartedAt    time.Time
	CompletedAt  *time.Time
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

// OntologyUsageRecord represents a single ontology context injection into a pipeline step.
type OntologyUsageRecord struct {
	ID             int64
	RunID          string
	StepID         string
	ContextName    string
	InvariantCount int
	StepStatus     string // "success", "failed", "skipped"
	ContractPassed *bool  // nil if contract not checked
	CreatedAt      time.Time
}

// OntologyStats holds aggregated statistics for a single ontology context.
type OntologyStats struct {
	ContextName string
	TotalRuns   int
	Successes   int
	Failures    int
	SuccessRate float64
	LastUsed    time.Time
}

// DecisionRecord holds an append-only decision log entry for a pipeline run.
type DecisionRecord struct {
	ID        int64
	RunID     string
	StepID    string
	Timestamp time.Time
	Category  string // "model_routing", "retry", "contract", "budget", "composition"
	Decision  string // what was decided
	Rationale string // why
	Context   string // JSON blob of relevant context data
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

// Webhook represents a registered webhook endpoint that receives
// lifecycle event notifications via HTTP POST.
type Webhook struct {
	ID        int64
	Name      string
	URL       string
	Events    []string          // e.g. ["run_completed", "step_failed", "gate_requested"]
	Matcher   string            // regex pattern for step name filtering (empty = all)
	Headers   map[string]string // custom headers to include
	Secret    string            // HMAC signing secret
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// WebhookDelivery records the outcome of a single webhook delivery attempt.
type WebhookDelivery struct {
	ID             int64
	WebhookID      int64
	RunID          string
	Event          string
	StatusCode     int
	ResponseTimeMs int64
	Error          string
	DeliveredAt    time.Time
}

// OutcomeType enumerates the kinds of pipeline outcomes tracked by the
// in-memory OutcomeTracker and persisted via the state store.
type OutcomeType string

const (
	OutcomeTypeFile       OutcomeType = "file"
	OutcomeTypeURL        OutcomeType = "url"
	OutcomeTypePR         OutcomeType = "pr"
	OutcomeTypeDeployment OutcomeType = "deployment"
	OutcomeTypeLog        OutcomeType = "log"
	OutcomeTypeContract   OutcomeType = "contract"
	OutcomeTypeArtifact   OutcomeType = "artifact"
	OutcomeTypeBranch     OutcomeType = "branch"
	OutcomeTypeIssue      OutcomeType = "issue"
	OutcomeTypeOther      OutcomeType = "other"
)

// OutcomeRecord stores a pipeline outcome that survives worktree cleanup.
// Examples: PR URL, issue URL, artifact path, branch name.
type OutcomeRecord struct {
	ID          int64
	RunID       string
	StepID      string
	Type        OutcomeType
	Label       string // human-readable label
	Value       string // the actual URL, path, or identifier
	Description string
	Metadata    map[string]any
	CreatedAt   time.Time
}

// OrchestrationDecision records a task classification → pipeline routing decision.
type OrchestrationDecision struct {
	ID           int64
	RunID        string
	InputText    string
	Domain       string
	Complexity   string
	PipelineName string
	ModelTier    string
	Reason       string
	Outcome      string // "pending", "completed", "failed", "cancelled"
	TokensUsed   int
	DurationMs   int64
	CreatedAt    time.Time
	CompletedAt  *time.Time
}

// OrchestrationStats aggregates success/failure rates for a pipeline.
type OrchestrationStats struct {
	PipelineName  string
	TotalRuns     int
	Completed     int
	Failed        int
	Cancelled     int
	AvgTokens     int
	AvgDurationMs int64
}

// OrchestrationDecisionSummary aggregates decisions grouped by domain+complexity+pipeline.
type OrchestrationDecisionSummary struct {
	Domain        string
	Complexity    string
	PipelineName  string
	Total         int
	Completed     int
	Failed        int
	SuccessRate   float64
	AvgTokens     int
	AvgDurationMs int64
}
