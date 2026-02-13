package event

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
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

	// Progress tracking fields (optional, for enhanced visualization)
	Progress        int     `json:"progress,omitempty"`          // 0-100 percentage for step progress
	CurrentAction   string  `json:"current_action,omitempty"`    // Current action being performed
	TotalSteps      int     `json:"total_steps,omitempty"`       // Total steps in pipeline
	CompletedSteps  int     `json:"completed_steps,omitempty"`   // Number of completed steps
	EstimatedTimeMs int64   `json:"estimated_time_ms"` // ETA in milliseconds (0 = no estimate)
	ValidationPhase string  `json:"validation_phase,omitempty"`  // Contract validation phase
	CompactionStats *string `json:"compaction_stats,omitempty"`  // Compaction statistics (JSON)

	// Error classification fields (context exhaustion handling)
	FailureReason string `json:"failure_reason,omitempty"` // "timeout", "context_exhaustion", "general_error"
	Remediation   string `json:"remediation,omitempty"`    // Actionable suggestion for the user

	// Stream event fields (real-time Claude Code activity)
	ToolName   string `json:"tool_name,omitempty"`   // Tool being used (Read, Write, Bash, etc.)
	ToolTarget string `json:"tool_target,omitempty"` // Target (file path, command, pattern)

	// Step metadata fields (FR-010: model and adapter type in step-start events)
	Model   string `json:"model,omitempty"`   // Model name (e.g., "opus", "sonnet")
	Adapter string `json:"adapter,omitempty"` // Adapter type (e.g., "claude")

	// Recovery hints (populated on failure events only)
	RecoveryHints []RecoveryHintJSON `json:"recovery_hints,omitempty"`
}

// RecoveryHintJSON is the JSON-serializable representation of a recovery hint.
type RecoveryHintJSON struct {
	Label   string `json:"label"`
	Command string `json:"command"`
	Type    string `json:"type"`
}

// Event state constants for pipeline and step lifecycle
const (
	// Existing states
	StateStarted   = "started"
	StateRunning   = "running"
	StateCompleted = "completed"
	StateFailed    = "failed"
	StateRetrying  = "retrying"

	// New progress tracking states
	StateStepProgress       = "step_progress"       // Step is making progress (with percentage)
	StateETAUpdated         = "eta_updated"         // Estimated time remaining updated
	StateContractValidating = "contract_validating" // Contract validation in progress
	StateCompactionProgress = "compaction_progress" // Context compaction in progress
	StateStreamActivity     = "stream_activity"     // Real-time tool activity from Claude Code
)

type EventEmitter interface {
	Emit(event Event)
}

type NDJSONEmitter struct {
	encoder         *json.Encoder
	suppressJSON    bool            // When true, suppresses JSON output to stdout
	mu              sync.Mutex
	progressEmitter ProgressEmitter // Optional enhanced progress emitter
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
		suppressJSON:    true, // Suppress JSON output to stdout
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

	e.encoder.Encode(event)
}
