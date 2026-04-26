package display

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
)

// canonicalBasicCLIEvents returns deterministic events used to lock the
// byte-exact stderr output of BasicProgressDisplay.EmitProgress.
//
// Each event uses a fixed timestamp so the [HH:MM:SS] prefix is reproducible.
func canonicalBasicCLIEvents() []event.Event {
	ts := time.Date(2026, 4, 26, 12, 34, 56, 0, time.UTC)
	return []event.Event{
		// Pipeline-level warning (StepID empty)
		{Timestamp: ts, State: "warning", Message: "global warning"},

		// started — with persona + model + adapter
		{Timestamp: ts, StepID: "specify", State: "started", Persona: "navigator", Model: "claude-opus-4-7", Adapter: "claude"},
		// started — persona only
		{Timestamp: ts, StepID: "plan", State: "started", Persona: "craftsman"},
		// started — bare (no line emitted because Persona empty)
		{Timestamp: ts, StepID: "tasks", State: "started"},
		// running — with persona (emits line)
		{Timestamp: ts, StepID: "tasks", State: "running", Persona: "navigator"},

		// completed — duration + tokens in/out
		{Timestamp: ts, StepID: "specify", State: "completed", DurationMs: 5000, TokensIn: 50000, TokensOut: 3200},
		// completed — duration + tokens used
		{Timestamp: ts, StepID: "plan", State: "completed", DurationMs: 1500, TokensUsed: 12345},

		// failed
		{Timestamp: ts, StepID: "implement", State: "failed", Message: "context exhaustion"},

		// retrying
		{Timestamp: ts, StepID: "implement", State: "retrying", Message: "attempt 2/3"},

		// step_progress — with action
		{Timestamp: ts, StepID: "specify", State: "step_progress", CurrentAction: "Executing agent"},
		// step_progress — without action (no line)
		{Timestamp: ts, StepID: "specify", State: "step_progress"},

		// warning at step level
		{Timestamp: ts, StepID: "specify", State: "warning", Message: "minor warning"},

		// validating contract
		{Timestamp: ts, StepID: "plan", State: "contract_validating", ValidationPhase: "json_schema"},
		// contract_passed/failed/soft (no direct line; updates handover info)
		{Timestamp: ts, StepID: "plan", State: "contract_passed"},
		{Timestamp: ts, StepID: "tasks", State: "contract_failed"},
		{Timestamp: ts, StepID: "checklist", State: "contract_soft_failure"},
	}
}

// canonicalBasicCLIStreamEvents returns events for verbose stream activity
// rendering (only emits when verbose=true and step is running).
func canonicalBasicCLIStreamEvents() []event.Event {
	ts := time.Date(2026, 4, 26, 12, 34, 56, 0, time.UTC)
	return []event.Event{
		// Mark step as running so stream activity guard passes
		{Timestamp: ts, StepID: "specify", State: "started", Persona: "navigator"},
		// Stream activity — short target
		{Timestamp: ts, StepID: "specify", State: "stream_activity", ToolName: "Read", ToolTarget: ".agents/artifacts/spec.md"},
		// Stream activity — long target (truncated to terminal width)
		{Timestamp: ts, StepID: "specify", State: "stream_activity", ToolName: "Bash", ToolTarget: strings.Repeat("a", 200)},
	}
}

// runBasicProgressGolden emits events through a BasicProgressDisplay and returns
// the captured stderr output.
func runBasicProgressGolden(verbose bool, evts []event.Event) string {
	var buf bytes.Buffer
	bpd := &BasicProgressDisplay{
		writer:       &buf,
		verbose:      verbose,
		termInfo:     NewTerminalInfo(),
		handoverInfo: make(map[string]*HandoverInfo),
		stepStates:   make(map[string]string),
	}
	for _, ev := range evts {
		_ = bpd.EmitProgress(ev)
	}
	return buf.String()
}

func TestBasicProgressDisplay_GoldenLifecycle(t *testing.T) {
	got := runBasicProgressGolden(false, canonicalBasicCLIEvents())

	goldenPath := filepath.Join("testdata", "basic_progress.golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
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
		t.Errorf("BasicProgressDisplay output drifted from golden.\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

// TestBasicProgressDisplay_GoldenStream covers the verbose stream-activity
// branch with a fixed-width terminal so truncation is reproducible.
func TestBasicProgressDisplay_GoldenStream(t *testing.T) {
	// Force a deterministic terminal width so the truncation arithmetic in
	// EmitProgress produces a stable golden line. We override after construction
	// to avoid coupling to env vars.
	var buf bytes.Buffer
	bpd := &BasicProgressDisplay{
		writer:       &buf,
		verbose:      true,
		termInfo:     newFixedWidthTerminalInfo(120),
		handoverInfo: make(map[string]*HandoverInfo),
		stepStates:   make(map[string]string),
	}
	for _, ev := range canonicalBasicCLIStreamEvents() {
		_ = bpd.EmitProgress(ev)
	}
	got := buf.String()

	goldenPath := filepath.Join("testdata", "basic_progress_stream.golden")
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
		t.Errorf("BasicProgressDisplay stream output drifted from golden.\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

// newFixedWidthTerminalInfo returns a TerminalInfo whose GetWidth() returns the
// supplied value. Used to make truncation arithmetic deterministic in golden
// tests.
func newFixedWidthTerminalInfo(width int) *TerminalInfo {
	return &TerminalInfo{
		capabilities: &TerminalCapabilities{
			Width:  width,
			Height: 40,
		},
	}
}
