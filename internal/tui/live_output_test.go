package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
)

// ===========================================================================
// EventBuffer tests
// ===========================================================================

func TestEventBuffer_NewEventBuffer_DefaultCapacity(t *testing.T) {
	buf := NewEventBuffer(0)
	assert.Equal(t, 0, buf.Len())
	// Zero capacity defaults to 1000
	buf.Append("test")
	assert.Equal(t, 1, buf.Len())
}

func TestEventBuffer_Append_And_Lines(t *testing.T) {
	buf := NewEventBuffer(5)
	buf.Append("line1")
	buf.Append("line2")
	buf.Append("line3")

	assert.Equal(t, 3, buf.Len())
	lines := buf.Lines()
	assert.Equal(t, []string{"line1", "line2", "line3"}, lines)
}

func TestEventBuffer_Overflow_DropsOldest(t *testing.T) {
	buf := NewEventBuffer(3)
	buf.Append("a")
	buf.Append("b")
	buf.Append("c")
	buf.Append("d") // pushes out "a"
	buf.Append("e") // pushes out "b"

	assert.Equal(t, 3, buf.Len())
	lines := buf.Lines()
	assert.Equal(t, []string{"c", "d", "e"}, lines)
}

func TestEventBuffer_Empty(t *testing.T) {
	buf := NewEventBuffer(10)
	assert.Equal(t, 0, buf.Len())
	assert.Nil(t, buf.Lines())
}

func TestEventBuffer_SingleCapacity(t *testing.T) {
	buf := NewEventBuffer(1)
	buf.Append("first")
	assert.Equal(t, 1, buf.Len())
	assert.Equal(t, []string{"first"}, buf.Lines())

	buf.Append("second")
	assert.Equal(t, 1, buf.Len())
	assert.Equal(t, []string{"second"}, buf.Lines())
}

func TestEventBuffer_ExactCapacity(t *testing.T) {
	buf := NewEventBuffer(3)
	buf.Append("a")
	buf.Append("b")
	buf.Append("c")

	assert.Equal(t, 3, buf.Len())
	assert.Equal(t, []string{"a", "b", "c"}, buf.Lines())
}

func TestEventBuffer_WrapAround_Multiple(t *testing.T) {
	buf := NewEventBuffer(3)
	// Fill and wrap multiple times
	for i := 0; i < 10; i++ {
		buf.Append(fmt.Sprintf("line%d", i))
	}
	assert.Equal(t, 3, buf.Len())
	lines := buf.Lines()
	assert.Equal(t, []string{"line7", "line8", "line9"}, lines)
}

// ===========================================================================
// DisplayFlags / shouldFormat tests
// ===========================================================================

func TestShouldFormat_DefaultMode(t *testing.T) {
	flags := DisplayFlags{}

	// Default mode shows lifecycle events
	assert.True(t, shouldFormat(event.Event{State: event.StateStarted, StepID: "step-1"}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateStarted, TotalSteps: 3}, flags))
	// Pipeline-level started without TotalSteps is a duplicate info event — skip it
	assert.False(t, shouldFormat(event.Event{State: event.StateStarted}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateRunning}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateCompleted}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateFailed}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateContractValidating}, flags))

	// Default mode hides verbose/debug events
	assert.False(t, shouldFormat(event.Event{State: event.StateStreamActivity}, flags))
	assert.False(t, shouldFormat(event.Event{State: event.StateStepProgress}, flags))
	assert.False(t, shouldFormat(event.Event{State: event.StateETAUpdated}, flags))
	assert.False(t, shouldFormat(event.Event{State: event.StateCompactionProgress}, flags))
}

func TestShouldFormat_VerboseMode(t *testing.T) {
	flags := DisplayFlags{Verbose: true}

	assert.True(t, shouldFormat(event.Event{State: event.StateStreamActivity}, flags))
	// Still shows default events
	assert.True(t, shouldFormat(event.Event{State: event.StateStarted, StepID: "step-1"}, flags))
	// Still hides debug events
	assert.False(t, shouldFormat(event.Event{State: event.StateStepProgress}, flags))
}

func TestShouldFormat_DebugMode(t *testing.T) {
	flags := DisplayFlags{Debug: true}

	// Empty heartbeats are skipped even in debug mode (matches CLI behavior)
	assert.False(t, shouldFormat(event.Event{State: event.StateStepProgress}, flags))
	// Heartbeats with data are shown
	assert.True(t, shouldFormat(event.Event{State: event.StateStepProgress, TokensIn: 100}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateStepProgress, CurrentAction: "Executing"}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateStepProgress, Progress: 50}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateETAUpdated}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateCompactionProgress}, flags))
	// Still shows default events
	assert.True(t, shouldFormat(event.Event{State: event.StateStarted, StepID: "step-1"}, flags))
	// Still hides verbose events
	assert.False(t, shouldFormat(event.Event{State: event.StateStreamActivity}, flags))
}

func TestShouldFormat_OutputOnlyMode(t *testing.T) {
	flags := DisplayFlags{OutputOnly: true, Verbose: true, Debug: true}

	// Output-only overrides everything
	assert.True(t, shouldFormat(event.Event{State: event.StateCompleted}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateFailed}, flags))

	// All other events hidden
	assert.False(t, shouldFormat(event.Event{State: event.StateStarted}, flags))
	assert.False(t, shouldFormat(event.Event{State: event.StateRunning}, flags))
	assert.False(t, shouldFormat(event.Event{State: event.StateStreamActivity}, flags))
	assert.False(t, shouldFormat(event.Event{State: event.StateStepProgress}, flags))
}

// ===========================================================================
// formatEventLine tests
// ===========================================================================

func TestFormatEventLine_Started(t *testing.T) {
	evt := event.Event{
		StepID:  "specify",
		State:   event.StateStarted,
		Persona: "navigator",
		Model:   "opus",
	}
	line := formatEventLine(evt)
	assert.Contains(t, line, "[specify]")
	assert.Contains(t, line, "Starting...")
	assert.Contains(t, line, "navigator")
	assert.Contains(t, line, "opus")
}

func TestFormatEventLine_Completed(t *testing.T) {
	evt := event.Event{
		StepID:     "plan",
		State:      event.StateCompleted,
		DurationMs: 42000,
	}
	line := formatEventLine(evt)
	assert.Contains(t, line, "[plan]")
	assert.Contains(t, line, "Completed")
	assert.Contains(t, line, "42s")
}

func TestFormatEventLine_Failed(t *testing.T) {
	evt := event.Event{
		StepID:  "plan",
		State:   event.StateFailed,
		Message: "context exhaustion",
	}
	line := formatEventLine(evt)
	assert.Contains(t, line, "[plan]")
	assert.Contains(t, line, "Failed")
	assert.Contains(t, line, "context exhaustion")
}

func TestFormatEventLine_StreamActivity(t *testing.T) {
	evt := event.Event{
		StepID:     "specify",
		State:      event.StateStreamActivity,
		ToolName:   "Read",
		ToolTarget: ".wave/artifacts/spec.md",
	}
	line := formatEventLine(evt)
	assert.Contains(t, line, "[specify]")
	assert.Contains(t, line, "Read")
	assert.Contains(t, line, ".wave/artifacts/spec.md")
}

func TestFormatEventLine_StreamActivity_TruncatesLongTarget(t *testing.T) {
	longTarget := strings.Repeat("a", 100)
	evt := event.Event{
		StepID:     "step1",
		State:      event.StateStreamActivity,
		ToolName:   "Bash",
		ToolTarget: longTarget,
	}
	line := formatEventLine(evt)
	assert.Contains(t, line, "...")
	assert.True(t, len(line) < len(longTarget)+20)
}

func TestFormatEventLine_StepProgress_WithTokens(t *testing.T) {
	evt := event.Event{
		StepID:    "specify",
		State:     event.StateStepProgress,
		TokensIn:  200000,
		TokensOut: 1234,
	}
	line := formatEventLine(evt)
	assert.Contains(t, line, "[specify]")
	assert.Contains(t, line, "tokens:")
	assert.Contains(t, line, "200.0k in")
	assert.Contains(t, line, "1.2k out")
}

func TestFormatEventLine_ContractValidating(t *testing.T) {
	evt := event.Event{
		StepID:          "plan",
		State:           event.StateContractValidating,
		ValidationPhase: "PASSED",
	}
	line := formatEventLine(evt)
	assert.Contains(t, line, "[plan]")
	assert.Contains(t, line, "Contract:")
	assert.Contains(t, line, "PASSED")
}

func TestFormatEventLine_Warning(t *testing.T) {
	evt := event.Event{
		StepID:  "plan",
		State:   "warning",
		Message: "workspace cleanup failed",
	}
	line := formatEventLine(evt)
	assert.Contains(t, line, "[plan]")
	assert.Contains(t, line, "workspace cleanup failed")
}

func TestFormatEventLine_Retrying(t *testing.T) {
	evt := event.Event{
		StepID:  "plan",
		State:   event.StateRetrying,
		Message: "attempt 2/3",
	}
	line := formatEventLine(evt)
	assert.Contains(t, line, "[plan]")
	assert.Contains(t, line, "Retrying")
	assert.Contains(t, line, "attempt 2/3")
}

func TestFormatEventLine_ContractPassed(t *testing.T) {
	line := formatEventLine(event.Event{StepID: "plan", State: "contract_passed"})
	assert.Contains(t, line, "[plan]")
	assert.Contains(t, line, "Contract: passed")
}

func TestFormatEventLine_ContractFailed(t *testing.T) {
	line := formatEventLine(event.Event{StepID: "plan", State: "contract_failed"})
	assert.Contains(t, line, "[plan]")
	assert.Contains(t, line, "Contract: failed")
}

func TestFormatEventLine_Completed_WithTokens(t *testing.T) {
	evt := event.Event{
		StepID:     "plan",
		State:      event.StateCompleted,
		DurationMs: 42000,
		TokensIn:   50000,
		TokensOut:  3200,
	}
	line := formatEventLine(evt)
	assert.Contains(t, line, "Completed")
	assert.Contains(t, line, "50.0k in")
	assert.Contains(t, line, "3.2k out")
}

func TestFormatEventLine_StepProgress_WithAction(t *testing.T) {
	evt := event.Event{
		StepID:        "specify",
		State:         event.StateStepProgress,
		CurrentAction: "Executing agent",
	}
	line := formatEventLine(evt)
	assert.Contains(t, line, "[specify]")
	assert.Contains(t, line, "Executing agent")
}

func TestShouldFormat_WarningAndRetrying(t *testing.T) {
	flags := DisplayFlags{}
	assert.True(t, shouldFormat(event.Event{State: "warning"}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateRetrying}, flags))
	assert.True(t, shouldFormat(event.Event{State: "contract_passed"}, flags))
	assert.True(t, shouldFormat(event.Event{State: "contract_failed"}, flags))
}

func TestFormatEventLine_NoColor(t *testing.T) {
	_ = os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	evt := event.Event{
		StepID: "plan",
		State:  event.StateCompleted,
	}
	line := formatEventLine(evt)
	assert.NotContains(t, line, "✓")
	assert.Contains(t, line, "Completed")
}

// ===========================================================================
// formatErrorBlock tests
// ===========================================================================

func TestFormatErrorBlock_AllFields(t *testing.T) {
	evt := event.Event{
		StepID:        "plan",
		Persona:       "craftsman",
		FailureReason: "context_exhaustion",
		Remediation:   "Consider splitting this step into smaller tasks",
		RecoveryHints: []event.RecoveryHintJSON{
			{Command: "wave run my-pipeline --from-step plan"},
			{Command: "wave run my-pipeline --model opus"},
		},
	}

	block := formatErrorBlock(evt)
	assert.Contains(t, block, "Pipeline failed")
	assert.Contains(t, block, "Step: plan (craftsman)")
	assert.Contains(t, block, "Reason: context_exhaustion")
	assert.Contains(t, block, "Remediation: Consider splitting")
	assert.Contains(t, block, "→ wave run my-pipeline --from-step plan")
	assert.Contains(t, block, "→ wave run my-pipeline --model opus")
}

func TestFormatErrorBlock_MissingOptionalFields(t *testing.T) {
	evt := event.Event{
		StepID: "plan",
	}

	block := formatErrorBlock(evt)
	assert.Contains(t, block, "Pipeline failed")
	assert.Contains(t, block, "Step: plan")
	assert.NotContains(t, block, "Reason:")
	assert.NotContains(t, block, "Remediation:")
	assert.NotContains(t, block, "Recovery hints:")
}

func TestFormatErrorBlock_NoColor(t *testing.T) {
	_ = os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	evt := event.Event{
		StepID: "plan",
	}

	block := formatErrorBlock(evt)
	assert.NotContains(t, block, "✗")
	assert.Contains(t, block, "Pipeline failed")
}

// ===========================================================================
// formatElapsed tests
// ===========================================================================

func TestFormatElapsed_UnderAnHour(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "00:00"},
		{30 * time.Second, "00:30"},
		{5*time.Minute + 23*time.Second, "05:23"},
		{59*time.Minute + 59*time.Second, "59:59"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatElapsed(tt.duration))
		})
	}
}

func TestFormatElapsed_OverAnHour(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{1 * time.Hour, "1:00:00"},
		{1*time.Hour + 23*time.Minute + 45*time.Second, "1:23:45"},
		{10*time.Hour + 5*time.Minute, "10:05:00"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatElapsed(tt.duration))
		})
	}
}

func TestFormatElapsed_NegativeDuration(t *testing.T) {
	assert.Equal(t, "00:00", formatElapsed(-5*time.Second))
}

// ===========================================================================
// LiveOutputModel tests
// ===========================================================================

func TestLiveOutputModel_Constructor(t *testing.T) {
	buf := NewEventBuffer(100)
	startedAt := time.Now()
	m := NewLiveOutputModel("run-1", "test-pipeline", buf, startedAt, 6)

	assert.Equal(t, "run-1", m.runID)
	assert.Equal(t, "test-pipeline", m.pipelineName)
	assert.True(t, m.autoScroll)
	assert.False(t, m.completed)
	assert.Equal(t, 6, m.totalSteps)
}

func TestLiveOutputModel_SetSize(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 40)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
	// Viewport height = 40 - 7 (header) - 2 (footer) = 31
	assert.Equal(t, 31, m.viewport.Height)
	assert.Equal(t, 120, m.viewport.Width)
}

func TestLiveOutputModel_Update_PipelineEventMsg(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 40)

	// Send a started event
	msg := PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{
			StepID:  "specify",
			State:   event.StateStarted,
			Persona: "navigator",
		},
	}
	m, _ = m.Update(msg)

	assert.Equal(t, 1, buf.Len())
	assert.Equal(t, "specify", m.currentStep)
	assert.Equal(t, 1, m.stepNumber)
}

func TestLiveOutputModel_Update_IgnoresMismatchedRunID(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 40)

	msg := PipelineEventMsg{
		RunID: "run-other",
		Event: event.Event{StepID: "specify", State: event.StateStarted},
	}
	m, _ = m.Update(msg)

	assert.Equal(t, 0, buf.Len())
}

func TestLiveOutputModel_Update_DisplayFlagToggles(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)

	assert.False(t, m.flags.Verbose)
	assert.False(t, m.flags.Debug)
	assert.False(t, m.flags.OutputOnly)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	assert.True(t, m.flags.Verbose)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.True(t, m.flags.Debug)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	assert.True(t, m.flags.OutputOnly)

	// Toggle off
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	assert.False(t, m.flags.Verbose)
}

func TestLiveOutputModel_Update_CompletionSetsCompleted(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 40)

	// Pipeline-level completion (empty StepID)
	msg := PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{State: event.StateCompleted},
	}
	m, cmd := m.Update(msg)

	assert.True(t, m.completed)
	assert.NotNil(t, cmd) // Should return transition timer cmd
	lines := buf.Lines()
	assert.True(t, len(lines) > 0)
	// Check that summary line contains completion info
	lastLine := lines[len(lines)-1]
	assert.Contains(t, lastLine, "Pipeline completed")
}

func TestLiveOutputModel_View_RendersThreeParts(t *testing.T) {
	buf := NewEventBuffer(100)
	buf.Append("[specify] Starting...")
	m := NewLiveOutputModel("run-1", "test-pipeline", buf, time.Now(), 6)
	m.showLog = true // Switch to log view to test buffer rendering
	m.SetSize(120, 30)

	view := m.View()
	assert.Contains(t, view, "test-pipeline")
	assert.Contains(t, view, "[specify] Starting...")
	// Footer should have flag indicators
	assert.Contains(t, view, "verbose")
}

func TestLiveOutputModel_View_EmptyBuffer_ShowsWaiting(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 30)

	// Dashboard mode (default): shows "Waiting for pipeline to start..."
	view := m.View()
	assert.Contains(t, view, "Waiting for pipeline to start...")

	// Log mode: shows "Waiting for events..."
	m.showLog = true
	m.updateViewportContent()
	view = m.View()
	assert.Contains(t, view, "Waiting for events...")
}

func TestLiveOutputModel_StepProgressUpdatesHeader(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 30)

	// First step starts
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateStarted, Model: "opus", TotalSteps: 6},
	})
	assert.Equal(t, "specify", m.currentStep)
	assert.Equal(t, 1, m.stepNumber)
	assert.Equal(t, "opus", m.model)

	// Second step starts
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "plan", State: event.StateStarted},
	})
	assert.Equal(t, "plan", m.currentStep)
	assert.Equal(t, 2, m.stepNumber)
}

func TestLiveOutputModel_FlagToggle_RebuildBuffer_ShowsHiddenEvents(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 40)

	// Send a stream_activity event (hidden by default, shown with verbose)
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "step1", State: event.StateStreamActivity, ToolName: "Read", ToolTarget: "file.go"},
	})
	// Should not be in buffer (verbose is off)
	assert.Equal(t, 0, buf.Len(), "stream_activity should be hidden by default")

	// Toggle verbose ON — buffer should rebuild with the event
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	assert.True(t, m.flags.Verbose)
	assert.Equal(t, 1, buf.Len(), "stream_activity should appear after verbose toggle")
	lines := buf.Lines()
	assert.Contains(t, lines[0], "Read")

	// Toggle verbose OFF — buffer should rebuild without the event
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	assert.False(t, m.flags.Verbose)
	assert.Equal(t, 0, buf.Len(), "stream_activity should disappear after verbose toggle off")
}

func TestLiveOutputModel_FlagToggle_DebugRebuild(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 40)

	// Send a step_progress event with tokens (hidden by default, shown with debug)
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "step1", State: event.StateStepProgress, TokensIn: 1000, TokensOut: 500},
	})
	assert.Equal(t, 0, buf.Len(), "step_progress should be hidden by default")

	// Toggle debug ON
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.Equal(t, 1, buf.Len(), "step_progress should appear after debug toggle")

	// Toggle debug OFF
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.Equal(t, 0, buf.Len(), "step_progress should disappear after debug toggle off")
}

func TestLiveOutputModel_OutputOnly_RebuildFilters(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 40)

	// Send a started event (visible by default, hidden in output-only)
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "step1", State: event.StateStarted, Persona: "nav"},
	})
	assert.Equal(t, 1, buf.Len(), "started should be visible by default")

	// Toggle output-only ON
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	assert.Equal(t, 0, buf.Len(), "started should be hidden in output-only mode")

	// Toggle output-only OFF
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	assert.Equal(t, 1, buf.Len(), "started should reappear after output-only toggle off")
}

func TestLiveOutputModel_StoredRecords_RebuildOnFlagToggle(t *testing.T) {
	// Simulates detached pipeline runs where events come from SQLite polling
	// rather than in-memory PipelineEventMsg. The storedRecords field must
	// be used by rebuildBuffer() when rawEvents is empty.
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 40)

	// Simulate SQLite-polled events (as done by DetachedEventPollTickMsg handler)
	records := []state.LogRecord{
		{StepID: "step1", State: event.StateStarted, Message: "Starting... (navigator)"},
		{StepID: "step1", State: event.StateRunning, Message: "Running..."},
		{StepID: "step1", State: event.StateStreamActivity, Message: "Read file.go"},
		{StepID: "step1", State: event.StateStepProgress, Message: "tokens: 1k in / 500 out"},
		{StepID: "step1", State: event.StateCompleted, Message: "Completed (5s)"},
	}
	for _, rec := range records {
		m.storedRecords = append(m.storedRecords, rec)
		if shouldFormatRecord(rec, m.flags) {
			buf.Append(formatStoredEvent(rec))
		}
	}
	m.updateViewportContent()

	// Default flags: verbose=false, debug=false
	// Should show started, running, completed (3 events) but NOT stream_activity or step_progress
	assert.Equal(t, 3, buf.Len(), "default flags should show 3 core events")

	// Toggle verbose ON — stream_activity should appear
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	assert.True(t, m.flags.Verbose)
	assert.Equal(t, 4, buf.Len(), "verbose ON should add stream_activity event")

	// Toggle verbose OFF — back to 3
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	assert.False(t, m.flags.Verbose)
	assert.Equal(t, 3, buf.Len(), "verbose OFF should remove stream_activity event")

	// Toggle debug ON — step_progress should appear
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.True(t, m.flags.Debug)
	assert.Equal(t, 4, buf.Len(), "debug ON should add step_progress event")

	// Toggle output-only ON — only completed should remain
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	assert.True(t, m.flags.OutputOnly)
	assert.Equal(t, 1, buf.Len(), "output-only should show only completed event")

	// Toggle output-only OFF — back to debug (4 events)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	assert.False(t, m.flags.OutputOnly)
	assert.Equal(t, 4, buf.Len(), "output-only OFF should restore debug view")
}

func TestLiveOutputModel_RawEvents_StoredForAllEvents(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 40)

	// Send various events
	events := []event.Event{
		{StepID: "s1", State: event.StateStarted},
		{StepID: "s1", State: event.StateStreamActivity, ToolName: "Read"},
		{StepID: "s1", State: event.StateStepProgress, TokensIn: 100},
		{StepID: "s1", State: event.StateCompleted, DurationMs: 5000},
	}
	for _, evt := range events {
		m, _ = m.Update(PipelineEventMsg{RunID: "run-1", Event: evt})
	}

	// All raw events should be stored regardless of flags
	assert.Equal(t, 4, len(m.rawEvents), "all events should be stored in rawEvents")
}

func TestLiveOutputModel_HandoverMetadata_OnStepCompletion(t *testing.T) {
	_ = os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 3)
	m.SetSize(120, 40)

	// Step 1 starts
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateStarted, Persona: "navigator"},
	})

	// Step 2 starts (so handover target can be resolved)
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "plan", State: event.StateStarted, Persona: "navigator"},
	})

	// Contract validation for step 1
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateContractValidating, ValidationPhase: "json_schema"},
	})
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: "contract_passed"},
	})

	// Step 1 completes with artifacts
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{
			StepID:     "specify",
			State:      event.StateCompleted,
			DurationMs: 42000,
			Artifacts:  []string{".wave/artifacts/spec.md"},
		},
	})

	// Verify handover info was accumulated
	assert.Contains(t, m.handoverInfo, "specify")
	assert.Equal(t, "passed", m.handoverInfo["specify"].ContractStatus)
	assert.Equal(t, "json_schema", m.handoverInfo["specify"].ContractSchema)

	// Verify tree lines were appended to buffer
	lines := buf.Lines()
	found := false
	for _, line := range lines {
		if strings.Contains(line, "artifact:") && strings.Contains(line, "spec.md") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected artifact tree line in buffer, got: %v", lines)

	// Check for contract line
	contractFound := false
	for _, line := range lines {
		if strings.Contains(line, "contract:") && strings.Contains(line, "valid") {
			contractFound = true
			break
		}
	}
	assert.True(t, contractFound, "Expected contract tree line in buffer, got: %v", lines)

	// Check for handover line
	handoverFound := false
	for _, line := range lines {
		if strings.Contains(line, "handover") && strings.Contains(line, "plan") {
			handoverFound = true
			break
		}
	}
	assert.True(t, handoverFound, "Expected handover tree line in buffer, got: %v", lines)
}

func TestLiveOutputModel_HandoverMetadata_ContractFailed(t *testing.T) {
	_ = os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 2)
	m.SetSize(120, 40)

	// Step starts
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "implement", State: event.StateStarted},
	})

	// Contract fails
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "implement", State: "contract_failed"},
	})

	// Step completes
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "implement", State: event.StateCompleted, DurationMs: 5000},
	})

	// Verify failed contract shows in output
	lines := buf.Lines()
	failedFound := false
	for _, line := range lines {
		if strings.Contains(line, "failed") && strings.Contains(line, "contract") {
			failedFound = true
			break
		}
	}
	assert.True(t, failedFound, "Expected contract failed line in buffer, got: %v", lines)
}

func TestLiveOutputModel_StepOrder_TracksCorrectly(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 3)

	// Three steps start in order
	for _, step := range []string{"specify", "plan", "implement"} {
		m, _ = m.Update(PipelineEventMsg{
			RunID: "run-1",
			Event: event.Event{StepID: step, State: event.StateStarted},
		})
	}

	assert.Equal(t, []string{"specify", "plan", "implement"}, m.stepOrder)

	// Duplicate start should not add again
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateStarted},
	})
	assert.Equal(t, []string{"specify", "plan", "implement"}, m.stepOrder)
}

// ===========================================================================
// Dashboard view tests
// ===========================================================================

func TestLiveOutputModel_DashboardIsDefault(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	assert.False(t, m.showLog, "dashboard should be the default view (showLog=false)")
	assert.NotNil(t, m.dashStepMap, "dashStepMap should be initialized")
}

func TestLiveOutputModel_DashboardState_UpdatesFromEvents(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 3)
	m.SetSize(120, 40)

	// Step 1 starts
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateStarted, Persona: "navigator", Model: "opus", Adapter: "claude"},
	})

	assert.Len(t, m.dashSteps, 1)
	s := m.dashStepMap["specify"]
	assert.Equal(t, "running", s.status)
	assert.Equal(t, "navigator", s.persona)
	assert.Equal(t, "opus", s.model)
	assert.Equal(t, "claude", s.adapter)

	// Tool activity
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateStreamActivity, ToolName: "Read", ToolTarget: "main.go"},
	})
	assert.Equal(t, "Read", m.dashStepMap["specify"].lastToolName)
	assert.Equal(t, "main.go", m.dashStepMap["specify"].lastToolTarget)

	// Step completes
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{
			StepID: "specify", State: event.StateCompleted,
			DurationMs: 30000, TokensIn: 50000, TokensOut: 3000,
			Artifacts: []string{".wave/artifacts/spec.md"},
		},
	})
	s = m.dashStepMap["specify"]
	assert.Equal(t, "completed", s.status)
	assert.Equal(t, int64(30000), s.durationMs)
	assert.Equal(t, 50000, s.tokensIn)
	assert.Equal(t, 3000, s.tokensOut)
	assert.Equal(t, []string{".wave/artifacts/spec.md"}, s.artifacts)

	// Step 2 starts and fails
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "plan", State: event.StateStarted, Persona: "navigator"},
	})
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "plan", State: event.StateFailed, Message: "context exhaustion", DurationMs: 10000},
	})
	assert.Len(t, m.dashSteps, 2)
	assert.Equal(t, "failed", m.dashStepMap["plan"].status)
	assert.Equal(t, "context exhaustion", m.dashStepMap["plan"].message)
}

func TestLiveOutputModel_DashboardState_UpdatesFromStoredRecords(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 2)

	now := time.Now()
	records := []state.LogRecord{
		{StepID: "specify", State: event.StateStarted, Persona: "navigator", Timestamp: now},
		{StepID: "specify", State: event.StateCompleted, DurationMs: 20000, TokensUsed: 5000},
		{StepID: "plan", State: event.StateStarted, Persona: "craftsman", Timestamp: now},
	}
	for _, rec := range records {
		m.updateDashStepFromRecord(rec)
	}

	assert.Len(t, m.dashSteps, 2)
	assert.Equal(t, "completed", m.dashStepMap["specify"].status)
	assert.Equal(t, "navigator", m.dashStepMap["specify"].persona)
	assert.Equal(t, int64(20000), m.dashStepMap["specify"].durationMs)
	assert.Equal(t, 5000, m.dashStepMap["specify"].tokensUsed)
	assert.Equal(t, "running", m.dashStepMap["plan"].status)
	assert.Equal(t, "craftsman", m.dashStepMap["plan"].persona)
}

func TestLiveOutputModel_RenderDashboard_CompletedStep(t *testing.T) {
	_ = os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 2)
	m.SetSize(120, 40)

	// Add a completed step
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateStarted, Persona: "navigator", Model: "opus"},
	})
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateCompleted, DurationMs: 42000, TokensIn: 50000, TokensOut: 3200},
	})

	dashboard := m.renderDashboard()
	assert.Contains(t, dashboard, "specify")
	assert.Contains(t, dashboard, "navigator")
	assert.Contains(t, dashboard, "opus")
	assert.Contains(t, dashboard, "42s")
	assert.Contains(t, dashboard, "50.0k in")
	assert.Contains(t, dashboard, "3.2k out")
}

func TestLiveOutputModel_RenderDashboard_RunningStep(t *testing.T) {
	_ = os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 2)
	m.SetSize(120, 40)

	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateStarted, Persona: "navigator"},
	})

	dashboard := m.renderDashboard()
	assert.Contains(t, dashboard, "specify")
	assert.Contains(t, dashboard, "navigator")
	// Running step should show elapsed time format (MM:SS)
	assert.Contains(t, dashboard, "00:0")
}

func TestLiveOutputModel_RenderDashboard_FailedStep(t *testing.T) {
	_ = os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 2)
	m.SetSize(120, 40)

	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "plan", State: event.StateStarted, Persona: "craftsman"},
	})
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "plan", State: event.StateFailed, DurationMs: 5000},
	})

	dashboard := m.renderDashboard()
	assert.Contains(t, dashboard, "plan")
	assert.Contains(t, dashboard, "craftsman")
	assert.Contains(t, dashboard, "5s")
}

func TestLiveOutputModel_RenderDashboard_VerboseHandover(t *testing.T) {
	_ = os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 2)
	m.SetSize(120, 40)

	// Start both steps so handover target can be resolved
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateStarted, Persona: "navigator"},
	})
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "plan", State: event.StateStarted, Persona: "navigator"},
	})

	// Contract validation
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: "contract_passed"},
	})

	// Complete with artifacts
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateCompleted, DurationMs: 5000, Artifacts: []string{".wave/artifacts/spec.md"}},
	})

	// Without verbose: no handover metadata
	dashboard := m.renderDashboard()
	assert.NotContains(t, dashboard, "artifact:")
	assert.NotContains(t, dashboard, "handover")

	// With verbose: handover metadata appears
	m.flags.Verbose = true
	dashboard = m.renderDashboard()
	assert.Contains(t, dashboard, "artifact:")
	assert.Contains(t, dashboard, "spec.md")
	assert.Contains(t, dashboard, "handover")
}

func TestLiveOutputModel_ShowLog_Toggle(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 40)

	// Default is dashboard mode
	assert.False(t, m.showLog)

	// Send an event
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateStarted, Persona: "navigator"},
	})

	// Toggle to log view
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	assert.True(t, m.showLog)

	// Viewport should contain event log content
	view := m.View()
	assert.Contains(t, view, "[specify]")
	assert.Contains(t, view, "Starting...")

	// Toggle back to dashboard
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	assert.False(t, m.showLog)

	// Viewport should contain dashboard content
	view = m.View()
	assert.Contains(t, view, "specify")
	assert.Contains(t, view, "navigator")
}

func TestLiveOutputModel_Footer_ShowsLogToggle(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 6)
	m.SetSize(120, 40)

	view := m.View()
	assert.Contains(t, view, "[ ] log")

	// Toggle log on
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	view = m.View()
	assert.Contains(t, view, "[l] log")
}

func TestLiveOutputModel_Header_ShowsCompletionCounts(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 3)
	m.SetSize(120, 40)

	// Start step 1
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateStarted, TotalSteps: 3},
	})

	// Complete step 1
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateCompleted, DurationMs: 5000},
	})

	// Start step 2
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "plan", State: event.StateStarted},
	})

	nc := noColor()
	header := m.renderHeader(nc)
	assert.Contains(t, header, "1 ok")
}

func TestLiveOutputModel_DashboardTickMsg_UpdatesViewport(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 2)
	m.SetSize(120, 40)

	// Start a running step
	m, _ = m.Update(PipelineEventMsg{
		RunID: "run-1",
		Event: event.Event{StepID: "specify", State: event.StateStarted, Persona: "navigator"},
	})

	// Dashboard tick should update and return a new tick command
	m, cmd := m.Update(DashboardTickMsg{})
	assert.NotNil(t, cmd, "dashboard tick should return a new tick command while running")

	// Complete the pipeline — tick should stop
	m.completed = true
	m, cmd = m.Update(DashboardTickMsg{})
	assert.Nil(t, cmd, "dashboard tick should return nil when pipeline is completed")
}

// ===========================================================================
// Persisted-event tailing indicator tests
// ===========================================================================

func TestLiveOutputModel_TailingPersisted_HeaderShowsIndicator(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 3)
	m.tailingPersisted = true
	m.SetSize(120, 40)

	nc := noColor()
	header := m.renderHeader(nc)
	assert.Contains(t, header, "Tailing persisted events", "header should show tailing indicator for detached runs")
}

func TestLiveOutputModel_TailingPersisted_FooterShowsSQLiteIndicator(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 3)
	m.tailingPersisted = true
	m.SetSize(120, 40)

	nc := noColor()
	footer := m.renderFooter(nc)
	assert.Contains(t, footer, "tailing from SQLite", "footer should show SQLite tailing indicator")
}

func TestLiveOutputModel_TailingPersisted_NotShownWhenCompleted(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 3)
	m.tailingPersisted = true
	m.completed = true
	m.SetSize(120, 40)

	nc := noColor()
	footer := m.renderFooter(nc)
	assert.NotContains(t, footer, "tailing from SQLite", "footer should not show tailing indicator when completed")
}

func TestLiveOutputModel_TailingPersisted_NotShownForLiveRuns(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 3)
	// tailingPersisted is false by default (in-process run)
	m.SetSize(120, 40)

	nc := noColor()
	header := m.renderHeader(nc)
	assert.NotContains(t, header, "Tailing persisted events", "header should not show tailing indicator for live runs")

	footer := m.renderFooter(nc)
	assert.NotContains(t, footer, "tailing from SQLite", "footer should not show SQLite indicator for live runs")
}

func TestLiveOutputModel_TailingPersisted_HeaderWithStepProgress(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 3)
	m.tailingPersisted = true
	m.stepNumber = 2
	m.totalSteps = 5
	m.currentStep = "plan"
	m.SetSize(120, 40)

	nc := noColor()
	header := m.renderHeader(nc)
	assert.Contains(t, header, "Tailing persisted events")
	assert.Contains(t, header, "step 2/5")
	assert.Contains(t, header, "plan")
}

func TestLiveOutputModel_TailingPersisted_DashboardPopulatedFromStoredRecords(t *testing.T) {
	// Verify that stored records populate the dashboard correctly
	// when tailingPersisted is true (simulating hover-selection of detached run)
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 2)
	m.tailingPersisted = true

	now := time.Now()
	records := []state.LogRecord{
		{StepID: "specify", State: event.StateStarted, Persona: "navigator", Timestamp: now},
		{StepID: "specify", State: event.StateCompleted, DurationMs: 20000, TokensUsed: 5000},
		{StepID: "plan", State: event.StateStarted, Persona: "craftsman", Timestamp: now},
	}

	// Simulate what content.go does for hover-selected detached runs
	for _, rec := range records {
		m.storedRecords = append(m.storedRecords, rec)
		m.updateDashStepFromRecord(rec)
		if shouldFormatRecord(rec, m.flags) {
			buf.Append(formatStoredEvent(rec))
		}
	}
	m.SetSize(120, 40)

	// Dashboard should show step states
	assert.Len(t, m.dashSteps, 2)
	assert.Equal(t, "completed", m.dashStepMap["specify"].status)
	assert.Equal(t, "running", m.dashStepMap["plan"].status)

	// Buffer should have events too
	assert.Equal(t, 3, buf.Len(), "buffer should contain started, completed, started events")
}

func TestLiveOutputModel_UpdateStepTrackingFromRecord(t *testing.T) {
	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "pipe", buf, time.Now(), 0)

	// Non-started events should be ignored
	m.updateStepTrackingFromRecord(state.LogRecord{
		State:  event.StateRunning,
		StepID: "step-a",
	})
	assert.Equal(t, 0, m.stepNumber)
	assert.Equal(t, "", m.currentStep)

	// Started event with StepID should increment stepNumber
	m.updateStepTrackingFromRecord(state.LogRecord{
		State:  event.StateStarted,
		StepID: "specify",
	})
	assert.Equal(t, 1, m.stepNumber)
	assert.Equal(t, "specify", m.currentStep)
	assert.Equal(t, []string{"specify"}, m.stepOrder)

	// Second step
	m.updateStepTrackingFromRecord(state.LogRecord{
		State:  event.StateStarted,
		StepID: "plan",
	})
	assert.Equal(t, 2, m.stepNumber)
	assert.Equal(t, "plan", m.currentStep)
	assert.Equal(t, []string{"specify", "plan"}, m.stepOrder)

	// Duplicate stepID should not add to stepOrder again
	m.updateStepTrackingFromRecord(state.LogRecord{
		State:  event.StateStarted,
		StepID: "specify",
	})
	assert.Equal(t, 3, m.stepNumber)
	assert.Equal(t, "specify", m.currentStep)
	assert.Equal(t, []string{"specify", "plan"}, m.stepOrder)

	// Pipeline-level started (no StepID) should be ignored
	m.updateStepTrackingFromRecord(state.LogRecord{
		State: event.StateStarted,
	})
	assert.Equal(t, 3, m.stepNumber, "pipeline-level started should not increment stepNumber")
}

func TestLiveOutputModel_TailingPersisted_HeaderShowsStepProgress(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	buf := NewEventBuffer(100)
	m := NewLiveOutputModel("run-1", "test-pipeline", buf, time.Now(), 0)
	m.tailingPersisted = true
	m.SetSize(120, 40)

	// Before step tracking, header shows generic tailing message
	header := m.renderHeader(true)
	assert.Contains(t, header, "Tailing persisted events")
	assert.NotContains(t, header, "step")

	// Populate step tracking from stored records
	m.updateStepTrackingFromRecord(state.LogRecord{
		State:  event.StateStarted,
		StepID: "specify",
	})
	m.updateStepTrackingFromRecord(state.LogRecord{
		State:  event.StateStarted,
		StepID: "plan",
	})

	header = m.renderHeader(true)
	assert.Contains(t, header, "step 2")
	assert.Contains(t, header, "plan")
}
