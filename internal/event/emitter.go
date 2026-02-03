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
)

type EventEmitter interface {
	Emit(event Event)
}

type NDJSONEmitter struct {
	encoder         *json.Encoder
	humanReadable   bool
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
		progressEmitter: nil,
	}
}

func NewNDJSONEmitterWithHumanReadable() *NDJSONEmitter {
	return &NDJSONEmitter{
		encoder:         json.NewEncoder(os.Stdout),
		humanReadable:   true,
		progressEmitter: nil,
	}
}

// NewNDJSONEmitterWithProgress creates an emitter with dual-stream support.
// NDJSON events go to stdout, enhanced progress visualization goes to stderr.
func NewNDJSONEmitterWithProgress(progressEmitter ProgressEmitter) *NDJSONEmitter {
	return &NDJSONEmitter{
		encoder:         json.NewEncoder(os.Stdout),
		humanReadable:   false,
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

	// Always emit NDJSON to stdout (backward compatibility)
	if e.humanReadable {
		stateColors := map[string]string{
			"started":              "\033[36m",
			"running":              "\033[33m",
			"completed":            "\033[32m",
			"failed":               "\033[31m",
			"retrying":             "\033[35m",
			"step_progress":        "\033[36m",
			"eta_updated":          "\033[90m",
			"contract_validating":  "\033[35m",
			"compaction_progress":  "\033[90m",
		}
		color := stateColors[event.State]
		if color == "" {
			color = "\033[0m"
		}
		reset := "\033[0m"

		ts := event.Timestamp.Format("15:04:05")
		if event.StepID != "" {
			fmt.Printf("%s[%s]%s %s%-10s%s %s", "\033[90m", ts, reset, color, event.State, reset, event.StepID)
			if event.Persona != "" {
				fmt.Printf(" (%s)", event.Persona)
			}
			if event.Progress > 0 {
				fmt.Printf(" %d%%", event.Progress)
			}
			if event.CurrentAction != "" {
				fmt.Printf(" [%s]", event.CurrentAction)
			}
			if event.DurationMs > 0 {
				secs := float64(event.DurationMs) / 1000.0
				fmt.Printf(" %.1fs", secs)
			}
			if event.TokensUsed > 0 {
				fmt.Printf(" %dk tokens", event.TokensUsed/1000)
			}
			if len(event.Artifacts) > 0 {
				fmt.Printf(" → %v", event.Artifacts)
			}
			if event.Message != "" {
				fmt.Printf(" %s", event.Message)
			}
			fmt.Println()
		} else {
			fmt.Printf("%s[%s]%s %s%-10s%s %s %s\n", "\033[90m", ts, reset, color, event.State, reset, event.PipelineID, event.Message)
		}
	} else {
		e.encoder.Encode(event)
	}
}
