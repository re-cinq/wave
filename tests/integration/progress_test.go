package integration_test

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/event"
)

// TestEventToProgressConversion tests converting events to progress updates.
func TestEventToProgressConversion(t *testing.T) {
	evt := event.Event{
		Timestamp:      time.Now(),
		PipelineID:     "test-pipeline-001",
		StepID:         "step-1",
		State:          "running",
		DurationMs:     5000,
		Message:        "Executing step 1",
		Persona:        "navigator",
		Progress:       50,
		CurrentAction:  "analyzing",
		TotalSteps:     5,
		CompletedSteps: 2,
		TokensUsed:     1500,
	}

	// Verify event has progress information
	if evt.Progress != 50 {
		t.Errorf("expected Progress 50, got %d", evt.Progress)
	}

	if evt.CurrentAction != "analyzing" {
		t.Errorf("expected CurrentAction 'analyzing', got %q", evt.CurrentAction)
	}

	if evt.TotalSteps != 5 {
		t.Errorf("expected TotalSteps 5, got %d", evt.TotalSteps)
	}

	if evt.CompletedSteps != 2 {
		t.Errorf("expected CompletedSteps 2, got %d", evt.CompletedSteps)
	}
}

// TestPipelineContextTracking tests comprehensive context tracking.
func TestPipelineContextTracking(t *testing.T) {
	ctx := &display.PipelineContext{
		ManifestPath:      "wave.yaml",
		PipelineName:      "test-pipeline",
		WorkspacePath:     ".agents/workspaces/test",
		TotalSteps:        5,
		CurrentStepNum:    0,
		CompletedSteps:    0,
		FailedSteps:       0,
		SkippedSteps:      0,
		OverallProgress:   0,
		PipelineStartTime: time.Now().UnixNano(),
		StepStatuses:      make(map[string]display.ProgressState),
	}

	steps := []string{"navigate", "analyze", "implement", "validate", "document"}

	startTime := time.Now()

	for i, stepID := range steps {
		// Start step
		ctx.CurrentStepNum = i + 1
		ctx.CurrentStepID = stepID
		ctx.CurrentPersona = "persona-" + stepID
		ctx.CurrentAction = "executing"
		ctx.CurrentStepStart = time.Now().UnixNano()
		ctx.StepStatuses[stepID] = display.StateRunning

		// Simulate step execution
		time.Sleep(1 * time.Millisecond) // Minimal delay for timing

		// Complete step
		ctx.StepStatuses[stepID] = display.StateCompleted
		ctx.CompletedSteps++

		// Update average step time
		elapsed := time.Since(startTime).Milliseconds()
		if ctx.CompletedSteps > 0 {
			ctx.AverageStepTimeMs = elapsed / int64(ctx.CompletedSteps)
		}

		// Calculate overall progress
		ctx.OverallProgress = (ctx.CompletedSteps * 100) / ctx.TotalSteps

		// Calculate ETA
		if ctx.CompletedSteps > 0 && ctx.CompletedSteps < ctx.TotalSteps {
			remainingSteps := ctx.TotalSteps - ctx.CompletedSteps
			ctx.EstimatedTimeMs = ctx.AverageStepTimeMs * int64(remainingSteps)
		}

		// Verify progress
		expectedProgress := ((i + 1) * 100) / len(steps)
		if ctx.OverallProgress != expectedProgress {
			t.Errorf("step %d: expected %d%% progress, got %d%%", i+1, expectedProgress, ctx.OverallProgress)
		}
	}

	// Verify final state
	if ctx.CompletedSteps != ctx.TotalSteps {
		t.Errorf("expected %d completed steps, got %d", ctx.TotalSteps, ctx.CompletedSteps)
	}

	if ctx.OverallProgress != 100 {
		t.Errorf("expected 100%% progress, got %d%%", ctx.OverallProgress)
	}

	// Allow small timing variance (up to 10ms) due to system scheduling
	if ctx.EstimatedTimeMs > 10 {
		t.Errorf("expected near-zero ETA after completion, got %dms", ctx.EstimatedTimeMs)
	}

	// Verify all steps are tracked
	if len(ctx.StepStatuses) != ctx.TotalSteps {
		t.Errorf("expected %d step statuses, got %d", ctx.TotalSteps, len(ctx.StepStatuses))
	}

	// Verify all steps completed
	for stepID, status := range ctx.StepStatuses {
		if status != display.StateCompleted {
			t.Errorf("step %s: expected completed, got %v", stepID, status)
		}
	}
}
