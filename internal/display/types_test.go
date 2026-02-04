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

func TestAnimationType_Constants(t *testing.T) {
	tests := []struct {
		animType AnimationType
		want     string
	}{
		{AnimationDots, "dots"},
		{AnimationLine, "line"},
		{AnimationBars, "bars"},
		{AnimationSpinner, "spinner"},
		{AnimationClock, "clock"},
		{AnimationBouncingBar, "bouncing_bar"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.animType) != tt.want {
				t.Errorf("AnimationType = %q, want %q", tt.animType, tt.want)
			}
		})
	}
}

func TestDefaultDisplayConfig(t *testing.T) {
	config := DefaultDisplayConfig()

	if !config.Enabled {
		t.Error("Default Enabled should be true")
	}
	if config.AnimationType != AnimationSpinner {
		t.Errorf("Default AnimationType = %q, want %q", config.AnimationType, AnimationSpinner)
	}
	if config.RefreshRate != 10 {
		t.Errorf("Default RefreshRate = %d, want 10", config.RefreshRate)
	}
	if !config.ShowDetails {
		t.Error("Default ShowDetails should be true")
	}
	if !config.ShowArtifacts {
		t.Error("Default ShowArtifacts should be true")
	}
	if config.CompactMode {
		t.Error("Default CompactMode should be false")
	}
	if config.ColorMode != "auto" {
		t.Errorf("Default ColorMode = %q, want %q", config.ColorMode, "auto")
	}
	if config.ColorTheme != "default" {
		t.Errorf("Default ColorTheme = %q, want %q", config.ColorTheme, "default")
	}
	if config.AsciiOnly {
		t.Error("Default AsciiOnly should be false")
	}
	if config.MaxHistoryLines != 100 {
		t.Errorf("Default MaxHistoryLines = %d, want 100", config.MaxHistoryLines)
	}
	if !config.EnableTimestamps {
		t.Error("Default EnableTimestamps should be true")
	}
	if config.VerboseOutput {
		t.Error("Default VerboseOutput should be false")
	}
	if !config.AnimationEnabled {
		t.Error("Default AnimationEnabled should be true")
	}
	if !config.ShowLogo {
		t.Error("Default ShowLogo should be true")
	}
	if !config.ShowMetrics {
		t.Error("Default ShowMetrics should be true")
	}
}

func TestDisplayConfig_Validate_RefreshRate(t *testing.T) {
	tests := []struct {
		name        string
		refreshRate int
		want        int
	}{
		{"too low", 0, 1},
		{"negative", -5, 1},
		{"minimum valid", 1, 1},
		{"normal", 30, 30},
		{"maximum valid", 60, 60},
		{"too high", 100, 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DisplayConfig{RefreshRate: tt.refreshRate}
			config.Validate()
			if config.RefreshRate != tt.want {
				t.Errorf("After Validate, RefreshRate = %d, want %d", config.RefreshRate, tt.want)
			}
		})
	}
}

func TestDisplayConfig_Validate_MaxHistoryLines(t *testing.T) {
	tests := []struct {
		name            string
		maxHistoryLines int
		want            int
	}{
		{"zero", 0, 100},
		{"negative", -10, 100},
		{"valid", 50, 50},
		{"large", 1000, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DisplayConfig{MaxHistoryLines: tt.maxHistoryLines, RefreshRate: 10}
			config.Validate()
			if config.MaxHistoryLines != tt.want {
				t.Errorf("After Validate, MaxHistoryLines = %d, want %d", config.MaxHistoryLines, tt.want)
			}
		})
	}
}

func TestDisplayConfig_Validate_ColorMode(t *testing.T) {
	tests := []struct {
		name      string
		colorMode string
		want      string
	}{
		{"auto", "auto", "auto"},
		{"on", "on", "on"},
		{"off", "off", "off"},
		{"invalid", "invalid", "auto"},
		{"empty", "", "auto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DisplayConfig{ColorMode: tt.colorMode, RefreshRate: 10, MaxHistoryLines: 100}
			config.Validate()
			if config.ColorMode != tt.want {
				t.Errorf("After Validate, ColorMode = %q, want %q", config.ColorMode, tt.want)
			}
		})
	}
}

func TestDisplayConfig_Validate_ColorTheme(t *testing.T) {
	tests := []struct {
		name       string
		colorTheme string
		want       string
	}{
		{"default", "default", "default"},
		{"dark", "dark", "dark"},
		{"light", "light", "light"},
		{"high_contrast", "high_contrast", "high_contrast"},
		{"invalid", "neon", "default"},
		{"empty", "", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DisplayConfig{ColorTheme: tt.colorTheme, RefreshRate: 10, MaxHistoryLines: 100, ColorMode: "auto"}
			config.Validate()
			if config.ColorTheme != tt.want {
				t.Errorf("After Validate, ColorTheme = %q, want %q", config.ColorTheme, tt.want)
			}
		})
	}
}

func TestDisplayConfig_Validate_AnimationType(t *testing.T) {
	tests := []struct {
		name          string
		animationType AnimationType
		want          AnimationType
	}{
		{"dots", AnimationDots, AnimationDots},
		{"spinner", AnimationSpinner, AnimationSpinner},
		{"line", AnimationLine, AnimationLine},
		{"bars", AnimationBars, AnimationBars},
		{"clock", AnimationClock, AnimationClock},
		{"bouncing_bar", AnimationBouncingBar, AnimationBouncingBar},
		{"invalid", AnimationType("invalid"), AnimationSpinner},
		{"empty", AnimationType(""), AnimationSpinner},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DisplayConfig{
				AnimationType:    tt.animationType,
				RefreshRate:      10,
				MaxHistoryLines:  100,
				ColorMode:        "auto",
				ColorTheme:       "default",
				AnimationEnabled: true,
			}
			config.Validate()
			if config.AnimationType != tt.want {
				t.Errorf("After Validate, AnimationType = %q, want %q", config.AnimationType, tt.want)
			}
		})
	}
}

func TestDisplayConfig_Validate_AnimationDisabled(t *testing.T) {
	config := DisplayConfig{
		AnimationType:    AnimationSpinner,
		AnimationEnabled: false,
		RefreshRate:      10,
		MaxHistoryLines:  100,
		ColorMode:        "auto",
		ColorTheme:       "default",
	}
	config.Validate()

	if config.AnimationType != AnimationDots {
		t.Errorf("When AnimationEnabled=false, AnimationType should be %q, got %q",
			AnimationDots, config.AnimationType)
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

func TestStepProgress_Structure(t *testing.T) {
	sp := StepProgress{
		StepID:        "step-1",
		Name:          "Test Step",
		State:         StateRunning,
		Persona:       "developer",
		Message:       "Processing",
		Progress:      50,
		CurrentAction: "Writing code",
		Artifacts:     []string{"file1.go", "file2.go"},
		StartTime:     1234567890,
		EndTime:       nil,
		Error:         "",
		TokensUsed:    1000,
		DurationMs:    5000,
	}

	if sp.StepID != "step-1" {
		t.Errorf("StepID = %q, want %q", sp.StepID, "step-1")
	}
	if sp.State != StateRunning {
		t.Errorf("State = %v, want %v", sp.State, StateRunning)
	}
}

func TestPipelineProgress_Structure(t *testing.T) {
	pp := PipelineProgress{
		PipelineID:     "pipeline-1",
		PipelineName:   "Test Pipeline",
		State:          StateRunning,
		TotalSteps:     5,
		CompletedSteps: 2,
		CurrentStep:    3,
		Progress:       40,
		Steps:          make(map[string]*StepProgress),
		StartTime:      1234567890,
		EndTime:        nil,
		Message:        "In progress",
		Error:          "",
	}

	if pp.PipelineID != "pipeline-1" {
		t.Errorf("PipelineID = %q, want %q", pp.PipelineID, "pipeline-1")
	}
	if pp.TotalSteps != 5 {
		t.Errorf("TotalSteps = %d, want 5", pp.TotalSteps)
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

func TestProgressRenderer_Interface(t *testing.T) {
	// This test verifies the interface can be implemented
	var _ ProgressRenderer = (*mockProgressRenderer)(nil)
}

type mockProgressRenderer struct{}

func (m *mockProgressRenderer) Render(progress *PipelineProgress) error {
	return nil
}

func (m *mockProgressRenderer) Clear() error {
	return nil
}

func (m *mockProgressRenderer) Close() error {
	return nil
}
