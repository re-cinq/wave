package event

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/recinq/wave/internal/state"
)

type Event struct {
	Timestamp  time.Time `json:"timestamp"`
	PipelineID string    `json:"pipeline_id"`
	StepID     string    `json:"step_id,omitempty"`
	State      string    `json:"state"`
	DurationMs int64     `json:"duration_ms,omitempty"`
	Message    string    `json:"message,omitempty"`
	Persona    string    `json:"persona,omitempty"`
	Artifacts  []string  `json:"artifacts,omitempty"`
	TokensUsed int       `json:"tokens_used,omitempty"`
	TokensIn   int       `json:"tokens_in,omitempty"`  // Input tokens (prompt + cache creation)
	TokensOut  int       `json:"tokens_out,omitempty"` // Output tokens (completion)

	// Progress tracking fields (optional, for enhanced visualization)
	Progress        int     `json:"progress,omitempty"`         // 0-100 percentage for step progress
	CurrentAction   string  `json:"current_action,omitempty"`   // Current action being performed
	TotalSteps      int     `json:"total_steps,omitempty"`      // Total steps in pipeline
	CompletedSteps  int     `json:"completed_steps,omitempty"`  // Number of completed steps
	EstimatedTimeMs int64   `json:"estimated_time_ms"`          // ETA in milliseconds (0 = no estimate)
	ValidationPhase string  `json:"validation_phase,omitempty"` // Contract validation phase
	CompactionStats *string `json:"compaction_stats,omitempty"` // Compaction statistics (JSON)

	// Error classification fields (context exhaustion handling)
	FailureReason string `json:"failure_reason,omitempty"` // "timeout", "context_exhaustion", "general_error"
	FailureClass  string `json:"failure_class,omitempty"`  // Pipeline-level failure classification (transient, deterministic, etc.)
	Remediation   string `json:"remediation,omitempty"`    // Actionable suggestion for the user

	// Stream event fields (real-time Claude Code activity)
	ToolName   string `json:"tool_name,omitempty"`   // Tool being used (Read, Write, Bash, etc.)
	ToolTarget string `json:"tool_target,omitempty"` // Target (file path, command, pattern)

	// Step metadata fields (FR-010: model and adapter type in step-start events)
	Model           string `json:"model,omitempty"`            // Resolved model ID (e.g., "claude-haiku-4-5")
	ConfiguredModel string `json:"configured_model,omitempty"` // Tier from pipeline config (e.g., "cheapest")
	Adapter         string `json:"adapter,omitempty"`          // Adapter type (e.g., "claude")
	Temperature float64 `json:"temperature,omitempty"` // Temperature setting for this step

	// Recovery hints (populated on failure events only)
	RecoveryHints []RecoveryHintJSON `json:"recovery_hints,omitempty"`

	// Structured outcomes (populated on final completion event only)
	Outcomes *OutcomesJSON `json:"outcomes,omitempty"`

	// Continuous loop iteration metadata (optional, for --continuous mode)
	Iteration      int    `json:"iteration,omitempty"`
	TotalProcessed int    `json:"total_processed,omitempty"`
	WorkItemID     string `json:"work_item_id,omitempty"`
}

// RecoveryHintJSON is the JSON-serializable representation of a recovery hint.
type RecoveryHintJSON struct {
	Label   string `json:"label"`
	Command string `json:"command"`
	Type    string `json:"type"`
}

// OutcomesJSON is the structured outcome data included in the final JSON completion event.
type OutcomesJSON struct {
	Branch       string            `json:"branch,omitempty"`
	Pushed       bool              `json:"pushed"`
	RemoteRef    string            `json:"remote_ref,omitempty"`
	PushError    string            `json:"push_error,omitempty"`
	PullRequests []OutcomeLinkJSON `json:"pull_requests"`
	Issues       []OutcomeLinkJSON `json:"issues"`
	Deployments  []OutcomeLinkJSON `json:"deployments"`
	Deliverables []DeliverableJSON `json:"deliverables"`
}

// OutcomeLinkJSON represents a URL outcome in JSON format.
type OutcomeLinkJSON struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// DeliverableJSON represents a deliverable in JSON format.
type DeliverableJSON struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
	StepID      string `json:"step_id"`
}

// Event state constants for pipeline and step lifecycle
const (
	// Event-specific states
	StateStarted = "started"

	// Step lifecycle states — canonical source: state.StepState
	StateRunning   = string(state.StateRunning)
	StateCompleted = string(state.StateCompleted)
	StateFailed    = string(state.StateFailed)
	StateRetrying  = string(state.StateRetrying)
	StateSkipped   = string(state.StateSkipped)
	StateReworking = string(state.StateReworking)

	// Progress tracking states
	StateStepProgress       = "step_progress"       // Step is making progress (with percentage)
	StateETAUpdated         = "eta_updated"         // Estimated time remaining updated
	StateContractValidating = "contract_validating" // Contract validation in progress
	StateCompactionProgress = "compaction_progress" // Context compaction in progress
	StateStreamActivity     = "stream_activity"     // Real-time tool activity from Claude Code

	// Sequence lifecycle states (pipeline composition)
	StateSequenceStarted   = "sequence_started"   // Sequence execution begun
	StateSequenceProgress  = "sequence_progress"  // Individual pipeline within sequence starting
	StateSequenceCompleted = "sequence_completed" // All pipelines in sequence completed
	StateSequenceFailed    = "sequence_failed"    // Sequence stopped due to pipeline failure

	// Parallel execution states
	StateParallelStageStarted   = "parallel_stage_started"   // Parallel stage begun
	StateParallelStageCompleted = "parallel_stage_completed" // Parallel stage finished
	StateParallelStageFailed    = "parallel_stage_failed"    // Parallel stage had failures

	// Composition primitive states
	StateIterationStarted   = "iteration_started"   // Iterate step begun
	StateIterationProgress  = "iteration_progress"  // Individual item processing
	StateIterationCompleted = "iteration_completed" // All items processed
	StateBranchEvaluated    = "branch_evaluated"    // Branch condition resolved
	StateGateWaiting        = "gate_waiting"        // Gate step blocking
	StateGateResolved       = "gate_resolved"       // Gate condition met
	StateLoopIteration      = "loop_iteration"      // Loop iteration started
	StateLoopCompleted      = "loop_completed"      // Loop terminated
	StateAggregateCompleted = "aggregate_completed" // Aggregation finished

	// Continuous loop states
	StateLoopStart             = "loop_start"
	StateLoopIterationStart    = "loop_iteration_start"
	StateLoopIterationComplete = "loop_iteration_complete"
	StateLoopIterationFailed   = "loop_iteration_failed"
	StateLoopSummary           = "loop_summary"

	// Ontology lifecycle states
	StateOntologyInject  = "ontology_inject"  // Ontology contexts injected into step
	StateOntologyLineage = "ontology_lineage" // Ontology decision lineage recorded

	// Hook lifecycle states
	StateHookStarted = "hook_started" // Hook execution begun
	StateHookPassed  = "hook_passed"  // Hook execution passed
	StateHookFailed  = "hook_failed"  // Hook execution failed
)

type EventEmitter interface {
	Emit(event Event)
}

type NDJSONEmitter struct {
	encoder         *json.Encoder
	suppressJSON    bool // When true, suppresses JSON output to stdout
	debugVerbose    bool // When true, emits additional internal state transition events
	mu              sync.Mutex
	progressEmitter ProgressEmitter // Optional enhanced progress emitter
}

// SetDebugVerbosity enables or disables emission of additional internal state
// transition events (step state changes, prompt assembly, artifact injection details).
func (e *NDJSONEmitter) SetDebugVerbosity(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.debugVerbose = enabled
}

// DebugVerbose returns whether debug verbosity is enabled.
func (e *NDJSONEmitter) DebugVerbose() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.debugVerbose
}

// ProgressEmitter is an optional interface for enhanced progress visualization.
// If set, it receives events on stderr while NDJSON continues to stdout.
type ProgressEmitter interface {
	EmitProgress(event Event) error
}

func NewNDJSONEmitter() *NDJSONEmitter {
	return &NDJSONEmitter{
		encoder:         json.NewEncoder(os.Stdout),
		suppressJSON:    false,
		progressEmitter: nil,
	}
}

// NewNDJSONEmitterWithProgress creates an emitter with dual-stream support.
// NDJSON events go to stdout, enhanced progress visualization goes to stderr.
func NewNDJSONEmitterWithProgress(progressEmitter ProgressEmitter) *NDJSONEmitter {
	return &NDJSONEmitter{
		encoder:         json.NewEncoder(os.Stdout),
		suppressJSON:    false,
		progressEmitter: progressEmitter,
	}
}

// NewProgressOnlyEmitter creates an emitter that only shows progress (no JSON logs).
// Progress goes to stderr, JSON logs are suppressed entirely.
func NewProgressOnlyEmitter(progressEmitter ProgressEmitter) *NDJSONEmitter {
	return &NDJSONEmitter{
		encoder:         json.NewEncoder(os.Stdout), // Still needs encoder but won't use it
		suppressJSON:    true,                       // Suppress JSON output to stdout
		progressEmitter: progressEmitter,
	}
}

// SetProgressEmitter sets or updates the progress emitter for enhanced visualization.
func (e *NDJSONEmitter) SetProgressEmitter(progressEmitter ProgressEmitter) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progressEmitter = progressEmitter
}

func (e *NDJSONEmitter) Emit(event Event) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// If progress emitter is configured, send enhanced progress events to it
	// This allows dual-stream output: NDJSON to stdout, enhanced display to stderr
	if e.progressEmitter != nil {
		// Send to progress emitter (stderr) - errors are logged but don't block
		if err := e.progressEmitter.EmitProgress(event); err != nil {
			// Log error to stderr without blocking main event flow
			fmt.Fprintf(os.Stderr, "Warning: progress emitter error: %v\n", err)
		}
	}

	// Emit NDJSON to stdout unless suppressed
	if e.suppressJSON {
		return
	}

	_ = e.encoder.Encode(event)
}
