package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/event"
)

// canonicalLiveEvents returns the deterministic event sequence used to lock
// byte-exact output of formatEventLine. Covers every state branch.
func canonicalLiveEvents() []event.Event {
	return []event.Event{
		// Started — with persona + model
		{StepID: "specify", State: event.StateStarted, Persona: "navigator", Model: "claude-opus-4-7"},
		// Started — persona only
		{StepID: "plan", State: event.StateStarted, Persona: "craftsman"},
		// Started — model only
		{StepID: "tasks", State: event.StateStarted, Model: "claude-haiku-4-5"},
		// Started — bare
		{StepID: "checklist", State: event.StateStarted},
		// Started — pipeline-level (uses PipelineID)
		{PipelineID: "pipe-1", State: event.StateStarted},

		// Running — with message
		{StepID: "specify", State: event.StateRunning, Message: "agent invoked"},
		// Running — bare
		{StepID: "plan", State: event.StateRunning},

		// Completed — duration only
		{StepID: "tasks", State: event.StateCompleted, DurationMs: 42000},
		// Completed — duration + tokens in/out
		{StepID: "checklist", State: event.StateCompleted, DurationMs: 5000, TokensIn: 50000, TokensOut: 3200},
		// Completed — duration + tokens used
		{StepID: "analyze", State: event.StateCompleted, DurationMs: 1500, TokensUsed: 12345},
		// Completed — no duration
		{StepID: "implement", State: event.StateCompleted},

		// Failed
		{StepID: "implement", State: event.StateFailed, Message: "context exhaustion"},

		// Retrying — with message
		{StepID: "implement", State: event.StateRetrying, Message: "attempt 2/3"},
		// Retrying — bare
		{StepID: "implement", State: event.StateRetrying},

		// Warning
		{StepID: "implement", State: "warning", Message: "workspace cleanup failed"},

		// Contract validating — with phase
		{StepID: "plan", State: event.StateContractValidating, ValidationPhase: "PASSED"},
		// Contract validating — bare
		{StepID: "plan", State: event.StateContractValidating},

		// Contract passed/failed/soft
		{StepID: "plan", State: "contract_passed"},
		{StepID: "plan", State: "contract_failed"},
		{StepID: "plan", State: "contract_soft_failure"},

		// Stream activity — short target
		{StepID: "specify", State: event.StateStreamActivity, ToolName: "Read", ToolTarget: ".agents/artifacts/spec.md"},
		// Stream activity — long target (truncated)
		{StepID: "specify", State: event.StateStreamActivity, ToolName: "Bash", ToolTarget: strings.Repeat("a", 100)},

		// Step progress — current action
		{StepID: "specify", State: event.StateStepProgress, CurrentAction: "Executing agent"},
		// Step progress — tokens
		{StepID: "specify", State: event.StateStepProgress, TokensIn: 200000, TokensOut: 1234},
		// Step progress — progress percent
		{StepID: "specify", State: event.StateStepProgress, Progress: 42},
		// Step progress — heartbeat
		{StepID: "specify", State: event.StateStepProgress},

		// ETA updated — with estimate
		{StepID: "specify", State: event.StateETAUpdated, EstimatedTimeMs: 90000},
		// ETA updated — calculating
		{StepID: "specify", State: event.StateETAUpdated},

		// Compaction
		{StepID: "specify", State: event.StateCompactionProgress},

		// Default — message
		{StepID: "specify", State: "unknown_state", Message: "fallthrough message"},
		// Default — bare
		{StepID: "specify", State: "unknown_state"},
	}
}

func TestFormatEventLine_GoldenLiveOutput(t *testing.T) {
	// Force NO_COLOR off so symbol branches render
	_ = os.Unsetenv("NO_COLOR")

	var sb strings.Builder
	for _, evt := range canonicalLiveEvents() {
		sb.WriteString(formatEventLine(evt))
		sb.WriteString("\n")
	}
	got := sb.String()

	goldenPath := filepath.Join("testdata", "live_output.golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with UPDATE_GOLDEN=1 to create)", goldenPath, err)
	}
	if got != string(want) {
		t.Errorf("formatEventLine output drifted from golden. got:\n%s\nwant:\n%s", got, string(want))
	}
}

func TestFormatEventLine_GoldenLiveOutput_NoColor(t *testing.T) {
	_ = os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	var sb strings.Builder
	for _, evt := range canonicalLiveEvents() {
		sb.WriteString(formatEventLine(evt))
		sb.WriteString("\n")
	}
	got := sb.String()

	goldenPath := filepath.Join("testdata", "live_output_nocolor.golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with UPDATE_GOLDEN=1 to create)", goldenPath, err)
	}
	if got != string(want) {
		t.Errorf("formatEventLine NO_COLOR output drifted from golden. got:\n%s\nwant:\n%s", got, string(want))
	}
}
