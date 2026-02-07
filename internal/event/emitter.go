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
	DurationMs int64     `json:"duration_ms"`
	Message    string    `json:"message,omitempty"`
	Persona    string    `json:"persona,omitempty"`
	Artifacts  []string  `json:"artifacts,omitempty"`
	TokensUsed int       `json:"tokens_used,omitempty"`

	// Progress tracking fields (optional, for enhanced visualization)
	Progress        int     `json:"progress,omitempty"`          // 0-100 percentage for step progress
	CurrentAction   string  `json:"current_action,omitempty"`    // Current action being performed
	TotalSteps      int     `json:"total_steps,omitempty"`       // Total steps in pipeline
	CompletedSteps  int     `json:"completed_steps,omitempty"`   // Number of completed steps
	EstimatedTimeMs int64   `json:"estimated_time_ms,omitempty"` // ETA in milliseconds
	ValidationPhase string  `json:"validation_phase,omitempty"`  // Contract validation phase
	CompactionStats *string `json:"compaction_stats,omitempty"`  // Compaction statistics (JSON)

	// Stream event fields (real-time Claude Code activity)
	ToolName   string `json:"tool_name,omitempty"`   // Tool being used (Read, Write, Bash, etc.)
	ToolTarget string `json:"tool_target,omitempty"` // Target (file path, command, pattern)
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
	humanReadable   bool
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
		humanReadable:   false,
		suppressJSON:    false,
		progressEmitter: nil,
	}
}

func NewNDJSONEmitterWithHumanReadable() *NDJSONEmitter {
	return &NDJSONEmitter{
		encoder:         json.NewEncoder(os.Stdout),
		humanReadable:   true,
		suppressJSON:    false,
		progressEmitter: nil,
	}
}

// NewNDJSONEmitterWithProgress creates an emitter with dual-stream support.
// NDJSON events go to stdout, enhanced progress visualization goes to stderr.
func NewNDJSONEmitterWithProgress(progressEmitter ProgressEmitter) *NDJSONEmitter {
	return &NDJSONEmitter{
		encoder:         json.NewEncoder(os.Stdout),
		humanReadable:   false,
		suppressJSON:    false,
		progressEmitter: progressEmitter,
	}
}

// NewProgressOnlyEmitter creates an emitter that only shows progress (no JSON logs).
// Progress goes to stderr, JSON logs are suppressed entirely.
func NewProgressOnlyEmitter(progressEmitter ProgressEmitter) *NDJSONEmitter {
	return &NDJSONEmitter{
		encoder:         json.NewEncoder(os.Stdout), // Still needs encoder but won't use it
		humanReadable:   false,
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

	// Emit NDJSON to stdout unless suppressed (backward compatibility)
	if e.suppressJSON {
		// Don't emit JSON when --no-logs is used
		return
	}

	if e.humanReadable {
		// Skip heartbeat events - they're for progress displays only
		if event.State == StateStepProgress ||
			event.State == StateETAUpdated ||
			event.State == StateCompactionProgress {
			return
		}

		dim := "\033[90m"
		reset := "\033[0m"

		// Stream activity events get compact dim rendering
		if event.State == StateStreamActivity && event.ToolName != "" {
			ts := event.Timestamp.Format("15:04:05")
			target := event.ToolTarget
			if len(target) > 60 {
				target = target[:60] + "..."
			}
			fmt.Printf("%s[%s]            %-20s %s → %s%s\n",
				dim, ts, event.StepID,
				event.ToolName, target, reset)
			return
		}

		stateColors := map[string]string{
			"started":             "\033[36m", // Primary (cyan)
			"running":             "\033[33m", // Warning (yellow)
			"completed":           "\033[32m", // Success (green)
			"failed":              "\033[31m", // Error (red)
			"retrying":            "\033[33m", // Warning (yellow)
			"contract_validating": "\033[36m", // Primary (cyan)
		}
		color := stateColors[event.State]
		if color == "" {
			color = "\033[0m"
		}

		ts := event.Timestamp.Format("15:04:05")
		if event.StepID != "" {
			// Base format: timestamp, state, stepID
			fmt.Printf("%s[%s]%s %s%-10s%s %-20s",
				dim, ts, reset,
				color, event.State, reset,
				event.StepID)

			if event.Persona != "" {
				fmt.Printf(" (%s)", event.Persona)
			}

			if event.DurationMs > 0 {
				secs := float64(event.DurationMs) / 1000.0
				if secs < 10 {
					fmt.Printf(" %5.1fs", secs)
				} else {
					fmt.Printf(" %5.0fs", secs)
				}
			}

			if event.TokensUsed > 0 {
				if event.TokensUsed >= 1000 {
					fmt.Printf(" %4.1fk", float64(event.TokensUsed)/1000.0)
				} else {
					fmt.Printf(" %d", event.TokensUsed)
				}
			}

			if len(event.Artifacts) > 0 {
				fmt.Printf(" → %v", event.Artifacts)
			}
			if event.Message != "" {
				fmt.Printf(" %s", event.Message)
			}
			fmt.Println()
		} else {
			fmt.Printf("%s[%s]%s %s%-10s%s %s %s\n", dim, ts, reset, color, event.State, reset, event.PipelineID, event.Message)
		}
	} else {
		e.encoder.Encode(event)
	}
}
