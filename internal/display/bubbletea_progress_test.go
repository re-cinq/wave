package display

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
)

func TestBubbleTeaProgressDisplay_ActivityCleanupOnCompletion(t *testing.T) {
	// Create a BubbleTeaProgressDisplay with minimal state for testing.
	// We bypass the constructor since it requires a TTY.
	btpd := &BubbleTeaProgressDisplay{
		steps:            make(map[string]*StepStatus),
		stepOrder:        make([]string, 0),
		stepDurations:    make(map[string]int64),
		stepStartTimes:   make(map[string]time.Time),
		stepToolActivity: make(map[string][2]string),
		handoverInfo:     make(map[string]*HandoverInfo),
		enabled:          false, // We'll call updateFromEvent directly
	}

	// Register two steps that share a workspace
	btpd.steps["step-a"] = &StepStatus{StepID: "step-a", Name: "step-a", Persona: "persona-a", State: StateNotStarted}
	btpd.steps["step-b"] = &StepStatus{StepID: "step-b", Name: "step-b", Persona: "persona-b", State: StateNotStarted}
	btpd.stepOrder = []string{"step-a", "step-b"}

	// Simulate step A starting
	btpd.updateFromEvent(event.Event{
		Timestamp: time.Now(),
		StepID:    "step-a",
		State:     "started",
	})

	if btpd.steps["step-a"].State != StateRunning {
		t.Errorf("step-a should be running, got %v", btpd.steps["step-a"].State)
	}

	// Simulate tool activity for step A
	btpd.updateFromEvent(event.Event{
		Timestamp:  time.Now(),
		StepID:     "step-a",
		State:      "stream_activity",
		ToolName:   "Read",
		ToolTarget: "file.go",
	})

	// Enable verbose to capture tool activity
	btpd.verbose = true
	btpd.updateFromEvent(event.Event{
		Timestamp:  time.Now(),
		StepID:     "step-a",
		State:      "stream_activity",
		ToolName:   "Read",
		ToolTarget: "file.go",
	})

	if _, exists := btpd.stepToolActivity["step-a"]; !exists {
		t.Error("step-a should have tool activity after stream_activity event")
	}

	// Simulate step A completing
	btpd.updateFromEvent(event.Event{
		Timestamp:  time.Now(),
		StepID:     "step-a",
		State:      "completed",
		DurationMs: 5000,
	})

	if btpd.steps["step-a"].State != StateCompleted {
		t.Errorf("step-a should be completed, got %v", btpd.steps["step-a"].State)
	}

	// Verify step A's tool activity was cleaned up
	if _, exists := btpd.stepToolActivity["step-a"]; exists {
		t.Error("step-a should NOT have tool activity after completion")
	}

	// Simulate step B starting (shares workspace)
	btpd.updateFromEvent(event.Event{
		Timestamp: time.Now(),
		StepID:    "step-b",
		State:     "started",
	})

	if btpd.steps["step-b"].State != StateRunning {
		t.Errorf("step-b should be running, got %v", btpd.steps["step-b"].State)
	}

	// Simulate tool activity for step B
	btpd.updateFromEvent(event.Event{
		Timestamp:  time.Now(),
		StepID:     "step-b",
		State:      "stream_activity",
		ToolName:   "Write",
		ToolTarget: "output.go",
	})

	// Verify only step B has activity, not step A
	if _, exists := btpd.stepToolActivity["step-a"]; exists {
		t.Error("step-a should still NOT have tool activity")
	}
	if activity, exists := btpd.stepToolActivity["step-b"]; !exists {
		t.Error("step-b should have tool activity")
	} else {
		if activity[0] != "Write" {
			t.Errorf("step-b tool name should be 'Write', got %q", activity[0])
		}
		if activity[1] != "output.go" {
			t.Errorf("step-b tool target should be 'output.go', got %q", activity[1])
		}
	}
}

func TestBubbleTeaProgressDisplay_SyntheticCompletionEvent(t *testing.T) {
	// Test that synthetic completion events (from resume) correctly
	// transition steps to completed state.
	btpd := &BubbleTeaProgressDisplay{
		steps:            make(map[string]*StepStatus),
		stepOrder:        make([]string, 0),
		stepDurations:    make(map[string]int64),
		stepStartTimes:   make(map[string]time.Time),
		stepToolActivity: make(map[string][2]string),
		handoverInfo:     make(map[string]*HandoverInfo),
		enabled:          false,
	}

	// Register a step that will receive a synthetic completion event
	btpd.steps["prior-step"] = &StepStatus{
		StepID:  "prior-step",
		Name:    "prior-step",
		Persona: "navigator",
		State:   StateNotStarted,
	}
	btpd.stepOrder = []string{"prior-step", "current-step"}

	// Emit synthetic completion event (as resume.go does)
	btpd.updateFromEvent(event.Event{
		Timestamp:  time.Now(),
		StepID:     "prior-step",
		State:      "completed",
		Persona:    "navigator",
		Message:    "completed in prior run",
		DurationMs: 0,
	})

	// Verify step transitioned to completed
	if btpd.steps["prior-step"].State != StateCompleted {
		t.Errorf("prior-step should be completed, got %v", btpd.steps["prior-step"].State)
	}
	if btpd.steps["prior-step"].Progress != 100 {
		t.Errorf("prior-step progress should be 100, got %d", btpd.steps["prior-step"].Progress)
	}

	// Verify no stale tool activity
	if _, exists := btpd.stepToolActivity["prior-step"]; exists {
		t.Error("prior-step should NOT have tool activity after synthetic completion")
	}

	// Verify duration is stored as 0 (synthetic events don't have real duration)
	if dur, exists := btpd.stepDurations["prior-step"]; exists && dur != 0 {
		t.Errorf("prior-step duration should be 0 or absent, got %d", dur)
	}
}
