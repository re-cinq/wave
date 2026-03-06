package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/event"
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
	assert.True(t, shouldFormat(event.Event{State: event.StateStarted}, flags))
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
	assert.True(t, shouldFormat(event.Event{State: event.StateStarted}, flags))
	// Still hides debug events
	assert.False(t, shouldFormat(event.Event{State: event.StateStepProgress}, flags))
}

func TestShouldFormat_DebugMode(t *testing.T) {
	flags := DisplayFlags{Debug: true}

	assert.True(t, shouldFormat(event.Event{State: event.StateStepProgress}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateETAUpdated}, flags))
	assert.True(t, shouldFormat(event.Event{State: event.StateCompactionProgress}, flags))
	// Still shows default events
	assert.True(t, shouldFormat(event.Event{State: event.StateStarted}, flags))
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
	assert.Contains(t, line, "persona: navigator")
	assert.Contains(t, line, "model: opus")
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
	assert.Contains(t, line, "heartbeat")
	assert.Contains(t, line, "1234/200000")
}

func TestFormatEventLine_ContractValidating(t *testing.T) {
	evt := event.Event{
		StepID:          "plan",
		State:           event.StateContractValidating,
		ValidationPhase: "PASSED",
	}
	line := formatEventLine(evt)
	assert.Contains(t, line, "[plan]")
	assert.Contains(t, line, "Contract validation")
	assert.Contains(t, line, "PASSED")
}

func TestFormatEventLine_NoColor(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
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
	os.Setenv("NO_COLOR", "1")
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
	// Viewport height = 40 - 3 (header) - 2 (footer) = 35
	assert.Equal(t, 35, m.viewport.Height)
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

	view := m.View()
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
