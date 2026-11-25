package display_test

import (
	"testing"

	"github.com/recinq/wave/internal/display"
)

// TestProgressStateConstants verifies progress state constants.
func TestProgressStateConstants(t *testing.T) {
	tests := []struct {
		name     string
		state    display.ProgressState
		expected string
	}{
		{"not started", display.StateNotStarted, "not_started"},
		{"running", display.StateRunning, "running"},
		{"completed", display.StateCompleted, "completed"},
		{"failed", display.StateFailed, "failed"},
		{"skipped", display.StateSkipped, "skipped"},
		{"cancelled", display.StateCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("expected state %q, got %q", tt.expected, string(tt.state))
			}
		})
	}
}

// TestAnimationTypeConstants verifies animation type constants.
func TestAnimationTypeConstants(t *testing.T) {
	tests := []struct {
		name      string
		animation display.AnimationType
		expected  string
	}{
		{"dots", display.AnimationDots, "dots"},
		{"line", display.AnimationLine, "line"},
		{"bars", display.AnimationBars, "bars"},
		{"spinner", display.AnimationSpinner, "spinner"},
		{"clock", display.AnimationClock, "clock"},
		{"bouncing_bar", display.AnimationBouncingBar, "bouncing_bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.animation) != tt.expected {
				t.Errorf("expected animation %q, got %q", tt.expected, string(tt.animation))
			}
		})
	}
}

// TestPipelineContext tests pipeline context structure.
func TestPipelineContext(t *testing.T) {
	ctx := display.PipelineContext{
		ManifestPath:      "wave.yaml",
		PipelineName:      "test-pipeline",
		WorkspacePath:     ".agents/workspaces/test",
		TotalSteps:        10,
		CurrentStepNum:    3,
		CompletedSteps:    2,
		FailedSteps:       0,
		SkippedSteps:      0,
		OverallProgress:   20,
		EstimatedTimeMs:   60000,
		CurrentStepID:     "step-3",
		CurrentPersona:    "craftsman",
		CurrentAction:     "building",
		CurrentStepName:   "Step 3",
		PipelineStartTime: 1000000,
		CurrentStepStart:  1005000,
		AverageStepTimeMs: 5000,
		ElapsedTimeMs:     10000,
		StepStatuses:      make(map[string]display.ProgressState),
		Message:           "Processing step 3",
	}

	if ctx.TotalSteps != 10 {
		t.Errorf("expected TotalSteps 10, got %d", ctx.TotalSteps)
	}

	if ctx.OverallProgress != 20 {
		t.Errorf("expected OverallProgress 20, got %d", ctx.OverallProgress)
	}

	if ctx.EstimatedTimeMs != 60000 {
		t.Errorf("expected EstimatedTimeMs 60000, got %d", ctx.EstimatedTimeMs)
	}

	if ctx.CurrentPersona != "craftsman" {
		t.Errorf("expected CurrentPersona 'craftsman', got %q", ctx.CurrentPersona)
	}

	if ctx.StepStatuses == nil {
		t.Error("expected StepStatuses map to be initialized")
	}
}
