package display

import (
	"testing"
)

func TestProgressState_Constants(t *testing.T) {
	// Verify all progress states have expected values
	tests := []struct {
		state ProgressState
		want  string
	}{
		{StateNotStarted, "not_started"},
		{StateRunning, "running"},
		{StateCompleted, "completed"},
		{StateFailed, "failed"},
		{StateSkipped, "skipped"},
		{StateCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.state) != tt.want {
				t.Errorf("ProgressState = %q, want %q", tt.state, tt.want)
			}
		})
	}
}

func TestColorPalette_Structure(t *testing.T) {
	palette := ColorPalette{
		Primary:    "\033[36m",
		Success:    "\033[32m",
		Warning:    "\033[33m",
		Error:      "\033[31m",
		Muted:      "\033[37m",
		Background: "\033[40m",
		Reset:      "\033[0m",
	}

	if palette.Primary == "" {
		t.Error("Primary should not be empty")
	}
	if palette.Reset == "" {
		t.Error("Reset should not be empty")
	}
}

func TestTerminalCapabilities_Structure(t *testing.T) {
	caps := TerminalCapabilities{
		IsTTY:             true,
		Width:             120,
		Height:            40,
		SupportsANSI:      true,
		SupportsColor:     true,
		Supports256Colors: true,
		SupportsUnicode:   true,
		SupportsAlternate: true,
		HasMouseSupport:   false,
		ColorScheme:       "dark",
	}

	if !caps.IsTTY {
		t.Error("IsTTY should be true")
	}
	if caps.Width != 120 {
		t.Errorf("Width = %d, want 120", caps.Width)
	}
}

func TestPipelineContext_Structure(t *testing.T) {
	ctx := PipelineContext{
		ManifestPath:      "wave.yaml",
		PipelineName:      "test-pipeline",
		WorkspacePath:     "/tmp/workspace",
		TotalSteps:        3,
		CurrentStepNum:    2,
		CompletedSteps:    1,
		FailedSteps:       0,
		SkippedSteps:      0,
		OverallProgress:   50,
		EstimatedTimeMs:   30000,
		CurrentStepID:     "step-2",
		CurrentPersona:    "developer",
		CurrentAction:     "Coding",
		CurrentStepName:   "Implementation",
		PipelineStartTime: 1234567890,
		CurrentStepStart:  1234567900,
		AverageStepTimeMs: 60000,
		ElapsedTimeMs:     120000,
		StepStatuses:      map[string]ProgressState{"step-1": StateCompleted},
		StepOrder:         []string{"step-1", "step-2", "step-3"},
		StepDurations:     map[string]int64{"step-1": 60000},
		DeliverablesByStep: map[string][]string{
			"step-1": {"output.txt"},
		},
		Message: "Processing",
		Error:   "",
	}

	if ctx.PipelineName != "test-pipeline" {
		t.Errorf("PipelineName = %q, want %q", ctx.PipelineName, "test-pipeline")
	}
	if ctx.TotalSteps != 3 {
		t.Errorf("TotalSteps = %d, want 3", ctx.TotalSteps)
	}
	if len(ctx.StepOrder) != 3 {
		t.Errorf("StepOrder length = %d, want 3", len(ctx.StepOrder))
	}
}

func TestPipelineContext_StepPersonas(t *testing.T) {
	ctx := PipelineContext{
		StepOrder: []string{"step-1", "step-2", "step-3"},
		StepStatuses: map[string]ProgressState{
			"step-1": StateCompleted,
			"step-2": StateRunning,
			"step-3": StateNotStarted,
		},
		StepPersonas: map[string]string{
			"step-1": "navigator",
			"step-2": "implementer",
			"step-3": "reviewer",
		},
	}

	// Verify mapping
	if ctx.StepPersonas["step-1"] != "navigator" {
		t.Errorf("StepPersonas[step-1] = %q, want %q", ctx.StepPersonas["step-1"], "navigator")
	}
	if ctx.StepPersonas["step-2"] != "implementer" {
		t.Errorf("StepPersonas[step-2] = %q, want %q", ctx.StepPersonas["step-2"], "implementer")
	}
	if ctx.StepPersonas["step-3"] != "reviewer" {
		t.Errorf("StepPersonas[step-3] = %q, want %q", ctx.StepPersonas["step-3"], "reviewer")
	}

	// Verify length
	if len(ctx.StepPersonas) != 3 {
		t.Errorf("StepPersonas length = %d, want 3", len(ctx.StepPersonas))
	}
}

func TestPipelineContext_StepPersonasInStructLiteral(t *testing.T) {
	// Verify StepPersonas works alongside all other fields in PipelineContext
	ctx := PipelineContext{
		ManifestPath:      "wave.yaml",
		PipelineName:      "test-pipeline",
		WorkspacePath:     "/tmp/workspace",
		TotalSteps:        3,
		CurrentStepNum:    2,
		CompletedSteps:    1,
		FailedSteps:       0,
		SkippedSteps:      0,
		OverallProgress:   50,
		CurrentStepID:     "step-2",
		CurrentPersona:    "implementer",
		PipelineStartTime: 1234567890,
		CurrentStepStart:  1234567900,
		StepStatuses:      map[string]ProgressState{"step-1": StateCompleted},
		StepOrder:         []string{"step-1", "step-2", "step-3"},
		StepDurations:     map[string]int64{"step-1": 60000},
		StepPersonas: map[string]string{
			"step-1": "navigator",
			"step-2": "implementer",
			"step-3": "reviewer",
		},
		DeliverablesByStep: map[string][]string{
			"step-1": {"output.txt"},
		},
	}

	if ctx.StepPersonas["step-1"] != "navigator" {
		t.Errorf("StepPersonas[step-1] = %q, want %q", ctx.StepPersonas["step-1"], "navigator")
	}
	if ctx.PipelineName != "test-pipeline" {
		t.Errorf("PipelineName = %q, want %q", ctx.PipelineName, "test-pipeline")
	}
}
