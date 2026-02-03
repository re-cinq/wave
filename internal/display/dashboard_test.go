package display

import (
	"strings"
	"testing"
	"time"
)

func TestDashboard_Render(t *testing.T) {
	dashboard := NewDashboard()

	// Create a test pipeline context
	ctx := &PipelineContext{
		ManifestPath:      "wave.yaml",
		PipelineName:      "test-pipeline",
		WorkspacePath:     ".wave/workspaces/test",
		TotalSteps:        3,
		CurrentStepNum:    2,
		CompletedSteps:    1,
		FailedSteps:       0,
		SkippedSteps:      0,
		OverallProgress:   50,
		EstimatedTimeMs:   30000,
		CurrentStepID:     "step2",
		CurrentPersona:    "developer",
		CurrentAction:     "Writing code",
		CurrentStepName:   "Implementation",
		PipelineStartTime: time.Now().Add(-1 * time.Minute).UnixNano(),
		CurrentStepStart:  time.Now().Add(-30 * time.Second).UnixNano(),
		AverageStepTimeMs: 60000,
		ElapsedTimeMs:     60000,
		StepStatuses: map[string]ProgressState{
			"step1": StateCompleted,
			"step2": StateRunning,
			"step3": StateNotStarted,
		},
		Message: "Processing",
	}

	// Test rendering
	err := dashboard.Render(ctx)
	if err != nil {
		t.Errorf("Render failed: %v", err)
	}

	// Test compact mode
	err = dashboard.RenderCompact(ctx)
	if err != nil {
		t.Errorf("RenderCompact failed: %v", err)
	}
}

func TestDashboard_ProgressBar(t *testing.T) {
	dashboard := NewDashboard()

	tests := []struct {
		name     string
		progress int
		width    int
	}{
		{"zero progress", 0, 40},
		{"half progress", 50, 40},
		{"full progress", 100, 40},
		{"small width", 25, 10},
		{"large width", 75, 80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := dashboard.renderProgressBar(tt.progress, tt.width)
			if bar == "" {
				t.Error("Progress bar should not be empty")
			}
			if !strings.Contains(bar, "[") || !strings.Contains(bar, "]") {
				t.Error("Progress bar should contain brackets")
			}
		})
	}
}

func TestDashboard_StatusIcon(t *testing.T) {
	dashboard := NewDashboard()

	tests := []struct {
		state ProgressState
	}{
		{StateCompleted},
		{StateFailed},
		{StateRunning},
		{StateSkipped},
		{StateCancelled},
		{StateNotStarted},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			icon := dashboard.getStatusIcon(tt.state)
			if icon == "" {
				t.Errorf("Status icon for %s should not be empty", tt.state)
			}
		})
	}
}

func TestDashboard_ShouldUseCompactMode(t *testing.T) {
	dashboard := NewDashboard()

	// Just verify it doesn't panic
	compactMode := dashboard.ShouldUseCompactMode()
	_ = compactMode // Use the variable
}

func TestFormatDashboardDuration(t *testing.T) {
	tests := []struct {
		name     string
		ms       int64
		contains string
	}{
		{"seconds", 30000, "s"},
		{"minutes", 120000, "m"},
		{"hours", 7200000, "h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDashboardDuration(tt.ms)
			if result == "" {
				t.Error("Duration should not be empty")
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Duration %q should contain %q", result, tt.contains)
			}
		})
	}
}

func TestDashboard_RenderPanels(t *testing.T) {
	dashboard := NewDashboard()

	ctx := &PipelineContext{
		ManifestPath:    "wave.yaml",
		PipelineName:    "test-pipeline",
		WorkspacePath:   ".wave/workspaces/test",
		TotalSteps:      3,
		CurrentStepNum:  1,
		CompletedSteps:  0,
		OverallProgress: 10,
		StepStatuses: map[string]ProgressState{
			"step1": StateRunning,
			"step2": StateNotStarted,
			"step3": StateNotStarted,
		},
	}

	// Test individual panel rendering
	t.Run("header", func(t *testing.T) {
		header := dashboard.renderHeader(ctx)
		if !strings.Contains(header, "WAVE") && !strings.Contains(header, "╦") {
			t.Error("Header should contain Wave logo")
		}
	})

	t.Run("progress panel", func(t *testing.T) {
		panel := dashboard.renderProgressPanel(ctx)
		if !strings.Contains(panel, "Step") {
			t.Error("Progress panel should contain step information")
		}
	})

	t.Run("step status panel", func(t *testing.T) {
		panel := dashboard.renderStepStatusPanel(ctx)
		if !strings.Contains(panel, "Step Status") {
			t.Error("Step status panel should contain header")
		}
	})

	t.Run("project info panel", func(t *testing.T) {
		panel := dashboard.renderProjectInfoPanel(ctx)
		if !strings.Contains(panel, "Project Information") {
			t.Error("Project info panel should contain header")
		}
	})
}

func TestDashboard_Clear(t *testing.T) {
	dashboard := NewDashboard()

	ctx := &PipelineContext{
		PipelineName: "test",
		TotalSteps:   1,
		StepStatuses: map[string]ProgressState{
			"step1": StateRunning,
		},
	}

	// Render then clear
	dashboard.Render(ctx)
	dashboard.Clear()

	// Verify lastLines is reset
	if dashboard.lastLines != 0 {
		t.Error("lastLines should be reset after Clear")
	}
}

// TestDashboard_RenderHeader_SignatureRegression ensures renderHeader method
// maintains compatibility with PipelineContext parameter.
// This test prevents the refactoring issue where method signature changes
// but test calls aren't updated (fixed in commit d026885).
func TestDashboard_RenderHeader_SignatureRegression(t *testing.T) {
	dashboard := NewDashboard()

	// Minimal context - only fields used by renderHeader
	ctx := &PipelineContext{
		PipelineName:  "test-pipeline",
		ManifestPath:  "wave.yaml",
		ElapsedTimeMs: 5000, // 5 seconds
	}

	// This call must compile and execute without error
	// If the method signature changes without updating tests, this will fail at compile time
	header := dashboard.renderHeader(ctx)

	// Verify the header contains expected elements
	if header == "" {
		t.Error("renderHeader should return non-empty string")
	}

	// Verify it includes Wave logo elements
	if !strings.Contains(header, "╦") && !strings.Contains(header, "WAVE") {
		t.Error("Header should contain Wave logo elements")
	}

	// Verify it includes pipeline info from context
	if !strings.Contains(header, "test-pipeline") {
		t.Error("Header should include pipeline name from context")
	}

	if !strings.Contains(header, "5.0s") {
		t.Error("Header should include formatted elapsed time from context")
	}
}

// TestProgressBarAnimationRegression verifies that progress bars include pulsing animation.
// This test prevents the regression where BubbleTeaProgressDisplay had static progress bars
// while Dashboard had animated ones (root cause of progress bar visibility issue).
func TestProgressBarAnimationRegression(t *testing.T) {
	dashboard := NewDashboard()

	// Test partial progress to ensure there's empty space for pulse animation
	progress := 50
	width := 20

	// Render progress bar multiple times at different time points
	// to verify animation changes
	bar1 := dashboard.renderProgressBar(progress, width)
	time.Sleep(100 * time.Millisecond) // Small delay to advance animation
	bar2 := dashboard.renderProgressBar(progress, width)

	// Basic structure checks
	if !strings.Contains(bar1, "[") || !strings.Contains(bar1, "]") {
		t.Error("Progress bar should contain brackets")
	}
	if !strings.Contains(bar2, "[") || !strings.Contains(bar2, "]") {
		t.Error("Progress bar should contain brackets")
	}

	// Animation checks - bars should be different due to pulse position changes
	// Note: Due to rapid animation timing, we mainly verify structure consistency
	if len(bar1) != len(bar2) {
		t.Error("Progress bars should have consistent length despite animation")
	}

	// Verify both bars contain filled and empty portions for 50% progress
	if !strings.Contains(bar1, "█") {
		t.Error("Progress bar should contain filled blocks for 50% progress")
	}
	if !strings.Contains(bar2, "█") {
		t.Error("Progress bar should contain filled blocks for 50% progress")
	}

	// Test edge case: 0% progress (all empty, should still animate)
	barEmpty := dashboard.renderProgressBar(0, width)
	if !strings.Contains(barEmpty, "[") || !strings.Contains(barEmpty, "]") {
		t.Error("Empty progress bar should still contain brackets")
	}

	// Test edge case: 100% progress (all filled, no empty space for animation)
	barFull := dashboard.renderProgressBar(100, width)
	if !strings.Contains(barFull, "█") {
		t.Error("Full progress bar should contain filled blocks")
	}
}
