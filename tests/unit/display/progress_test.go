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

// TestDefaultDisplayConfig verifies default configuration values.
func TestDefaultDisplayConfig(t *testing.T) {
	config := display.DefaultDisplayConfig()

	if !config.Enabled {
		t.Error("expected Enabled to be true by default")
	}

	if config.RefreshRate != 10 {
		t.Errorf("expected RefreshRate 10, got %d", config.RefreshRate)
	}

	if config.ColorMode != "auto" {
		t.Errorf("expected ColorMode 'auto', got %q", config.ColorMode)
	}

	if config.ColorTheme != "default" {
		t.Errorf("expected ColorTheme 'default', got %q", config.ColorTheme)
	}

	if config.AsciiOnly {
		t.Error("expected AsciiOnly to be false by default")
	}

	if config.VerboseOutput {
		t.Error("expected VerboseOutput to be false by default")
	}

	if !config.AnimationEnabled {
		t.Error("expected AnimationEnabled to be true by default")
	}
}

// TestDisplayConfigValidation tests configuration validation.
func TestDisplayConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		config   display.DisplayConfig
		validate func(*testing.T, *display.DisplayConfig)
	}{
		{
			name: "clamp refresh rate below minimum",
			config: display.DisplayConfig{
				RefreshRate: -5,
			},
			validate: func(t *testing.T, cfg *display.DisplayConfig) {
				if cfg.RefreshRate != 1 {
					t.Errorf("expected RefreshRate to be clamped to 1, got %d", cfg.RefreshRate)
				}
			},
		},
		{
			name: "clamp refresh rate above maximum",
			config: display.DisplayConfig{
				RefreshRate: 100,
			},
			validate: func(t *testing.T, cfg *display.DisplayConfig) {
				if cfg.RefreshRate != 60 {
					t.Errorf("expected RefreshRate to be clamped to 60, got %d", cfg.RefreshRate)
				}
			},
		},
		{
			name: "fix invalid color mode",
			config: display.DisplayConfig{
				ColorMode: "invalid",
			},
			validate: func(t *testing.T, cfg *display.DisplayConfig) {
				if cfg.ColorMode != "auto" {
					t.Errorf("expected ColorMode to be 'auto', got %q", cfg.ColorMode)
				}
			},
		},
		{
			name: "fix invalid color theme",
			config: display.DisplayConfig{
				ColorTheme: "neon-pink",
			},
			validate: func(t *testing.T, cfg *display.DisplayConfig) {
				if cfg.ColorTheme != "default" {
					t.Errorf("expected ColorTheme to be 'default', got %q", cfg.ColorTheme)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config
			config.Validate()
			tt.validate(t, &config)
		})
	}
}

// TestGetColorSchemeByName tests color scheme selection.
func TestGetColorSchemeByName(t *testing.T) {
	tests := []struct {
		name     string
		theme    string
		expected display.ColorPalette
	}{
		{"default", "default", display.DefaultColorScheme},
		{"dark", "dark", display.DarkColorScheme},
		{"light", "light", display.LightColorScheme},
		{"high_contrast", "high_contrast", display.HighContrastColorScheme},
		{"invalid defaults to default", "invalid", display.DefaultColorScheme},
		{"empty defaults to default", "", display.DefaultColorScheme},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := display.GetColorSchemeByName(tt.theme)
			if scheme.Primary != tt.expected.Primary {
				t.Errorf("expected Primary %q, got %q", tt.expected.Primary, scheme.Primary)
			}
			if scheme.Success != tt.expected.Success {
				t.Errorf("expected Success %q, got %q", tt.expected.Success, scheme.Success)
			}
		})
	}
}

// TestColorPaletteProperties verifies color palette properties.
func TestColorPaletteProperties(t *testing.T) {
	tests := []struct {
		name    string
		palette display.ColorPalette
	}{
		{"default", display.DefaultColorScheme},
		{"dark", display.DarkColorScheme},
		{"light", display.LightColorScheme},
		{"high_contrast", display.HighContrastColorScheme},
		{"ascii_only", display.AsciiOnlyColorScheme},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// All schemes should have a reset code (except ASCII-only)
			if tt.name != "ascii_only" && tt.palette.Reset == "" {
				t.Error("expected Reset code to be non-empty")
			}

			// ASCII-only should have all empty strings
			if tt.name == "ascii_only" {
				if tt.palette.Primary != "" || tt.palette.Success != "" || tt.palette.Error != "" {
					t.Error("expected all ASCII-only colors to be empty strings")
				}
			}
		})
	}
}

// TestStepProgress tests step progress structure.
func TestStepProgress(t *testing.T) {
	step := display.StepProgress{
		StepID:        "test-step",
		Name:          "Test Step",
		State:         display.StateRunning,
		Persona:       "navigator",
		Message:       "Running tests",
		Progress:      50,
		CurrentAction: "executing",
		Artifacts:     []string{"artifact1", "artifact2"},
		StartTime:     1000000,
		TokensUsed:    1500,
		DurationMs:    5000,
	}

	if step.StepID != "test-step" {
		t.Errorf("expected StepID 'test-step', got %q", step.StepID)
	}

	if step.Progress != 50 {
		t.Errorf("expected Progress 50, got %d", step.Progress)
	}

	if len(step.Artifacts) != 2 {
		t.Errorf("expected 2 artifacts, got %d", len(step.Artifacts))
	}

	if step.EndTime != nil {
		t.Error("expected EndTime to be nil for running step")
	}
}

// TestPipelineProgress tests pipeline progress structure.
func TestPipelineProgress(t *testing.T) {
	pipeline := display.PipelineProgress{
		PipelineID:     "test-pipeline-001",
		PipelineName:   "Test Pipeline",
		State:          display.StateRunning,
		TotalSteps:     5,
		CompletedSteps: 2,
		CurrentStep:    3,
		Progress:       40,
		Steps:          make(map[string]*display.StepProgress),
		StartTime:      1000000,
		Message:        "Pipeline in progress",
	}

	if pipeline.TotalSteps != 5 {
		t.Errorf("expected TotalSteps 5, got %d", pipeline.TotalSteps)
	}

	if pipeline.CompletedSteps != 2 {
		t.Errorf("expected CompletedSteps 2, got %d", pipeline.CompletedSteps)
	}

	if pipeline.Progress != 40 {
		t.Errorf("expected Progress 40, got %d", pipeline.Progress)
	}

	if pipeline.Steps == nil {
		t.Error("expected Steps map to be initialized")
	}

	if pipeline.EndTime != nil {
		t.Error("expected EndTime to be nil for running pipeline")
	}
}

// TestPipelineContext tests pipeline context structure.
func TestPipelineContext(t *testing.T) {
	ctx := display.PipelineContext{
		ManifestPath:      "wave.yaml",
		PipelineName:      "test-pipeline",
		WorkspacePath:     ".wave/workspaces/test",
		TotalSteps:        10,
		CurrentStepNum:    3,
		CompletedSteps:    2,
		FailedSteps:       0,
		SkippedSteps:      0,
		OverallProgress:   20,
		CurrentStepID:     "step-3",
		CurrentPersona:    "craftsman",
		CurrentAction:     "building",
		CurrentStepName:   "Step 3",
		PipelineStartTime: 1000000,
		CurrentStepStart:  1005000,
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

	if ctx.CurrentPersona != "craftsman" {
		t.Errorf("expected CurrentPersona 'craftsman', got %q", ctx.CurrentPersona)
	}

	if ctx.StepStatuses == nil {
		t.Error("expected StepStatuses map to be initialized")
	}
}
