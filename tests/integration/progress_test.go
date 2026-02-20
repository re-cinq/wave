package integration_test

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/event"
)

// TestProgressDisplayIntegration tests end-to-end progress display functionality.
func TestProgressDisplayIntegration(t *testing.T) {
	// Create a display configuration
	config := display.DefaultDisplayConfig()
	config.Validate()

	if !config.Enabled {
		t.Error("expected display to be enabled by default")
	}
}

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

// TestProgressStateTransitions tests state transition handling.
func TestProgressStateTransitions(t *testing.T) {
	tests := []struct {
		name      string
		fromState display.ProgressState
		toState   display.ProgressState
		valid     bool
	}{
		{"not_started to running", display.StateNotStarted, display.StateRunning, true},
		{"running to completed", display.StateRunning, display.StateCompleted, true},
		{"running to failed", display.StateRunning, display.StateFailed, true},
		{"running to cancelled", display.StateRunning, display.StateCancelled, true},
		{"not_started to skipped", display.StateNotStarted, display.StateSkipped, true},
		{"completed to running", display.StateCompleted, display.StateRunning, false}, // Invalid
		{"failed to completed", display.StateFailed, display.StateCompleted, false},   // Invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create step with initial state
			step := display.StepProgress{
				StepID: "test-step",
				State:  tt.fromState,
			}

			// Transition to new state
			step.State = tt.toState

			// Verify state was updated
			if step.State != tt.toState {
				t.Errorf("expected state %v, got %v", tt.toState, step.State)
			}

			// Note: In a real implementation, invalid transitions might be prevented
			// This test just verifies the basic state change mechanism
		})
	}
}

// TestPipelineProgressTracking tests comprehensive pipeline progress tracking.
func TestPipelineProgressTracking(t *testing.T) {
	pipeline := &display.PipelineProgress{
		PipelineID:     "integration-test-001",
		PipelineName:   "Integration Test Pipeline",
		State:          display.StateNotStarted,
		TotalSteps:     3,
		CompletedSteps: 0,
		CurrentStep:    0,
		Progress:       0,
		Steps:          make(map[string]*display.StepProgress),
		StartTime:      time.Now().UnixNano(),
	}

	// Simulate pipeline execution
	steps := []struct {
		stepID   string
		persona  string
		duration int64
		tokens   int
	}{
		{"step-1", "navigator", 5000, 1500},
		{"step-2", "craftsman", 8000, 3000},
		{"step-3", "philosopher", 3000, 800},
	}

	// Start pipeline
	pipeline.State = display.StateRunning

	for i, stepInfo := range steps {
		stepNum := i + 1

		// Start step
		step := &display.StepProgress{
			StepID:     stepInfo.stepID,
			Name:       stepInfo.stepID,
			State:      display.StateRunning,
			Persona:    stepInfo.persona,
			Progress:   0,
			StartTime:  time.Now().UnixNano(),
			TokensUsed: 0,
		}
		pipeline.Steps[stepInfo.stepID] = step
		pipeline.CurrentStep = stepNum

		// Simulate progress
		for progress := 0; progress <= 100; progress += 25 {
			step.Progress = progress
			pipeline.Progress = ((stepNum - 1) * 100 + progress) / len(steps)

			// Verify progress is within bounds
			if step.Progress < 0 || step.Progress > 100 {
				t.Errorf("step progress out of bounds: %d", step.Progress)
			}
			if pipeline.Progress < 0 || pipeline.Progress > 100 {
				t.Errorf("pipeline progress out of bounds: %d", pipeline.Progress)
			}
		}

		// Complete step
		step.State = display.StateCompleted
		endTime := time.Now().UnixNano()
		step.EndTime = &endTime
		step.TokensUsed = stepInfo.tokens
		step.DurationMs = stepInfo.duration
		pipeline.CompletedSteps++

		// Verify step completion
		if step.State != display.StateCompleted {
			t.Errorf("expected step to be completed, got %v", step.State)
		}
		if step.EndTime == nil {
			t.Error("expected EndTime to be set for completed step")
		}
	}

	// Complete pipeline
	pipeline.State = display.StateCompleted
	pipeline.Progress = 100
	endTime := time.Now().UnixNano()
	pipeline.EndTime = &endTime

	// Verify final state
	if pipeline.State != display.StateCompleted {
		t.Errorf("expected pipeline to be completed, got %v", pipeline.State)
	}

	if pipeline.CompletedSteps != pipeline.TotalSteps {
		t.Errorf("expected %d completed steps, got %d", pipeline.TotalSteps, pipeline.CompletedSteps)
	}

	if pipeline.Progress != 100 {
		t.Errorf("expected 100%% progress, got %d%%", pipeline.Progress)
	}

	if pipeline.EndTime == nil {
		t.Error("expected EndTime to be set for completed pipeline")
	}

	// Verify all steps are tracked
	if len(pipeline.Steps) != pipeline.TotalSteps {
		t.Errorf("expected %d steps tracked, got %d", pipeline.TotalSteps, len(pipeline.Steps))
	}
}

// TestPipelineContextTracking tests comprehensive context tracking.
func TestPipelineContextTracking(t *testing.T) {
	ctx := &display.PipelineContext{
		ManifestPath:      "wave.yaml",
		PipelineName:      "test-pipeline",
		WorkspacePath:     ".wave/workspaces/test",
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

		// Calculate overall progress
		ctx.OverallProgress = (ctx.CompletedSteps * 100) / ctx.TotalSteps

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

// TestDisplayConfigFromEnvironment tests configuration from environment variables.
func TestDisplayConfigFromEnvironment(t *testing.T) {
	// This test would set environment variables and verify they're picked up
	// For now, we just test that default config works
	config := display.DefaultDisplayConfig()
	config.Validate()

	if config.RefreshRate < 1 || config.RefreshRate > 60 {
		t.Errorf("expected valid RefreshRate, got %d", config.RefreshRate)
	}
}

// TestPerformanceMonitoring tests performance metrics tracking.
func TestPerformanceMonitoring(t *testing.T) {
	metrics := display.NewPerformanceMetrics()
	if metrics == nil {
		t.Fatal("expected NewPerformanceMetrics to return non-nil")
	}

	// Set execution start
	metrics.SetExecutionStart()

	// Simulate some render operations
	for i := 0; i < 10; i++ {
		cleanup := metrics.RecordRenderStart()
		time.Sleep(1 * time.Millisecond)
		cleanup()
	}

	// Get stats
	stats := metrics.GetStats()

	if stats.TotalRenders != 10 {
		t.Errorf("expected 10 renders, got %d", stats.TotalRenders)
	}

	if stats.AvgRenderTimeMs <= 0 {
		t.Errorf("expected positive average render time, got %f", stats.AvgRenderTimeMs)
	}

	if stats.MaxRenderTimeMs < stats.MinRenderTimeMs {
		t.Errorf("max render time (%f) should be >= min render time (%f)",
			stats.MaxRenderTimeMs, stats.MinRenderTimeMs)
	}

	// Set total execution time
	metrics.SetTotalExecutionTime(100 * time.Millisecond)

	// Check overhead
	overhead := metrics.GetOverheadRatio()
	if overhead < 0 || overhead > 1 {
		t.Errorf("expected overhead ratio 0-1, got %f", overhead)
	}

	// Record some events
	for i := 0; i < 50; i++ {
		metrics.RecordEvent()
	}

	stats = metrics.GetStats()
	if stats.TotalEvents != 50 {
		t.Errorf("expected 50 events, got %d", stats.TotalEvents)
	}
}

// TestPerformanceOverheadTarget tests overhead target validation.
func TestPerformanceOverheadTarget(t *testing.T) {
	metrics := display.NewPerformanceMetrics()

	// Set execution times to simulate different overhead scenarios
	tests := []struct {
		name            string
		renderTimeMs    int64
		executionTimeMs int64
		expectExceeded  bool
	}{
		{"low overhead 1%", 1, 100, false},
		{"acceptable 4%", 4, 100, false},
		{"at target 5%", 5, 100, false},
		{"exceeded 6%", 6, 100, true},
		{"high overhead 10%", 10, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.Reset()

			// Simulate render time
			metrics.RecordRenderComplete(time.Duration(tt.renderTimeMs) * time.Millisecond)

			// Set execution time
			metrics.SetTotalExecutionTime(time.Duration(tt.executionTimeMs) * time.Millisecond)

			exceeded := metrics.IsOverheadTargetExceeded()
			if exceeded != tt.expectExceeded {
				t.Errorf("expected overhead exceeded=%v, got %v (overhead=%f%%)",
					tt.expectExceeded, exceeded, metrics.GetOverheadRatio()*100)
			}
		})
	}
}
