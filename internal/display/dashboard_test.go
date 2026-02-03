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
		header := dashboard.renderHeader()
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
