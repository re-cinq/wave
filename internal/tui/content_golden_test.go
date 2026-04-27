package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/state"
)

// canonicalStoredRecords returns the deterministic LogRecord sequence used to
// lock byte-exact output of formatStoredEvent. Covers every state branch.
func canonicalStoredRecords() []state.LogRecord {
	return []state.LogRecord{
		// started — with persona
		{StepID: "specify", State: "started", Persona: "navigator"},
		// started — bare
		{StepID: "plan", State: "started"},
		// started — pipeline-level (StepID empty -> "pipeline" prefix)
		{State: "started"},

		// running — message
		{StepID: "specify", State: "running", Message: "agent invoked"},
		// running — bare
		{StepID: "plan", State: "running"},

		// completed — duration + tokens
		{StepID: "tasks", State: "completed", DurationMs: 5000, TokensUsed: 12345},
		// completed — duration only
		{StepID: "tasks", State: "completed", DurationMs: 42000},
		// completed — bare
		{StepID: "tasks", State: "completed"},

		// failed — with message
		{StepID: "implement", State: "failed", Message: "context exhaustion"},
		// failed — empty message ("unknown error")
		{StepID: "implement", State: "failed"},

		// retrying — with message
		{StepID: "implement", State: "retrying", Message: "attempt 2/3"},
		// retrying — bare
		{StepID: "implement", State: "retrying"},

		// warning
		{StepID: "implement", State: "warning", Message: "workspace cleanup failed"},

		// contract validating — phase via Message
		{StepID: "plan", State: "contract_validating", Message: "json_schema"},
		// contract validating — bare
		{StepID: "plan", State: "contract_validating"},

		// contract passed/failed/soft
		{StepID: "plan", State: "contract_passed"},
		{StepID: "plan", State: "contract_failed"},
		{StepID: "plan", State: "contract_soft_failure"},

		// stream activity
		{StepID: "specify", State: "stream_activity", Message: "Read .agents/artifacts/spec.md"},

		// step progress — with action
		{StepID: "specify", State: "step_progress", Message: "Executing agent"},
		// step progress — bare
		{StepID: "specify", State: "step_progress"},

		// default — message
		{StepID: "specify", State: "unknown_state", Message: "fallthrough message"},
		// default — bare
		{StepID: "specify", State: "unknown_state"},
	}
}

func TestFormatStoredEvent_GoldenStoredOutput(t *testing.T) {
	_ = os.Unsetenv("NO_COLOR")

	var sb strings.Builder
	for _, rec := range canonicalStoredRecords() {
		sb.WriteString(formatStoredEvent(rec))
		sb.WriteString("\n")
	}
	got := sb.String()

	goldenPath := filepath.Join("testdata", "stored_events.golden")
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
		t.Errorf("formatStoredEvent output drifted from golden. got:\n%s\nwant:\n%s", got, string(want))
	}
}

func TestFormatStoredEvent_GoldenStoredOutput_NoColor(t *testing.T) {
	_ = os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	var sb strings.Builder
	for _, rec := range canonicalStoredRecords() {
		sb.WriteString(formatStoredEvent(rec))
		sb.WriteString("\n")
	}
	got := sb.String()

	goldenPath := filepath.Join("testdata", "stored_events_nocolor.golden")
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
		t.Errorf("formatStoredEvent NO_COLOR output drifted from golden. got:\n%s\nwant:\n%s", got, string(want))
	}
}
