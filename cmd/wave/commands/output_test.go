package commands

import (
	"testing"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOutputFormat(t *testing.T) {
	tests := []struct {
		format  string
		wantErr bool
	}{
		{"auto", false},
		{"json", false},
		{"text", false},
		{"quiet", false},
		{"invalid", true},
		{"", true},
		{"JSON", true}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			err := ValidateOutputFormat(tt.format)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateEmitter_JSONFormat(t *testing.T) {
	cfg := OutputConfig{Format: OutputFormatJSON, Verbose: false}
	result := CreateEmitter(cfg, "test", "test", nil, &manifest.Manifest{})
	defer result.Cleanup()

	assert.NotNil(t, result.Emitter, "json format should return an emitter")
	assert.Nil(t, result.Progress, "json format should not have a progress display")
}

func TestCreateEmitter_TextFormat(t *testing.T) {
	cfg := OutputConfig{Format: OutputFormatText, Verbose: false}
	result := CreateEmitter(cfg, "test", "test", nil, &manifest.Manifest{})
	defer result.Cleanup()

	assert.NotNil(t, result.Emitter)
	assert.NotNil(t, result.Progress, "text format should have a progress display")

	_, ok := result.Progress.(*display.ThrottledProgressEmitter)
	assert.True(t, ok, "text format should use ThrottledProgressEmitter wrapping BasicProgressDisplay")
}

func TestCreateEmitter_TextFormatVerbose(t *testing.T) {
	cfg := OutputConfig{Format: OutputFormatText, Verbose: true}
	result := CreateEmitter(cfg, "test", "test", nil, &manifest.Manifest{})
	defer result.Cleanup()

	assert.NotNil(t, result.Emitter)
	assert.NotNil(t, result.Progress)

	_, ok := result.Progress.(*display.ThrottledProgressEmitter)
	assert.True(t, ok, "text verbose format should use ThrottledProgressEmitter")
}

func TestCreateEmitter_QuietFormat(t *testing.T) {
	cfg := OutputConfig{Format: OutputFormatQuiet, Verbose: false}
	result := CreateEmitter(cfg, "test", "test", nil, &manifest.Manifest{})
	defer result.Cleanup()

	assert.NotNil(t, result.Emitter)
	assert.NotNil(t, result.Progress, "quiet format should have a progress display")

	_, ok := result.Progress.(*display.ThrottledProgressEmitter)
	assert.True(t, ok, "quiet format should use ThrottledProgressEmitter wrapping QuietProgressDisplay")
}

func TestCreateEmitter_AutoFormatWithSteps(t *testing.T) {
	cfg := OutputConfig{Format: OutputFormatAuto, Verbose: false}
	steps := []pipeline.Step{
		{ID: "step1", Persona: "navigator"},
		{ID: "step2", Persona: "craftsman"},
	}
	result := CreateEmitter(cfg, "test-pipeline", "test-pipeline", steps, &manifest.Manifest{})
	defer result.Cleanup()

	assert.NotNil(t, result.Emitter)
	require.NotNil(t, result.Cleanup)
}

func TestCreateEmitter_AutoFormatForceTTY(t *testing.T) {
	t.Setenv("WAVE_FORCE_TTY", "1")

	cfg := OutputConfig{Format: OutputFormatAuto, Verbose: false}
	steps := []pipeline.Step{
		{ID: "step1", Persona: "navigator"},
	}
	result := CreateEmitter(cfg, "test", "test", steps, &manifest.Manifest{})
	defer result.Cleanup()

	assert.NotNil(t, result.Emitter)
	// When TTY is forced, auto mode should use ThrottledProgressEmitter wrapping BubbleTea
	_, isThrottled := result.Progress.(*display.ThrottledProgressEmitter)
	assert.True(t, isThrottled, "auto mode with WAVE_FORCE_TTY=1 should use ThrottledProgressEmitter")
}

func TestCreateEmitter_AutoFormatForceNonTTY(t *testing.T) {
	t.Setenv("WAVE_FORCE_TTY", "0")

	cfg := OutputConfig{Format: OutputFormatAuto, Verbose: false}
	steps := []pipeline.Step{
		{ID: "step1", Persona: "navigator"},
	}
	result := CreateEmitter(cfg, "test", "test", steps, &manifest.Manifest{})
	defer result.Cleanup()

	assert.NotNil(t, result.Emitter)
	// When non-TTY is forced, auto mode should use ThrottledProgressEmitter wrapping BasicProgressDisplay
	_, isThrottled := result.Progress.(*display.ThrottledProgressEmitter)
	assert.True(t, isThrottled, "auto mode with WAVE_FORCE_TTY=0 should use ThrottledProgressEmitter")
}

func TestCreateEmitter_NilSteps(t *testing.T) {
	cfg := OutputConfig{Format: OutputFormatText, Verbose: false}
	result := CreateEmitter(cfg, "test", "test", nil, &manifest.Manifest{})
	defer result.Cleanup()

	assert.NotNil(t, result.Emitter)
}

func TestOutputFormatConstants(t *testing.T) {
	assert.Equal(t, "auto", OutputFormatAuto)
	assert.Equal(t, "json", OutputFormatJSON)
	assert.Equal(t, "text", OutputFormatText)
	assert.Equal(t, "quiet", OutputFormatQuiet)
}

func TestCreateEmitter_TextFormatUsesThrottle(t *testing.T) {
	cfg := OutputConfig{Format: OutputFormatText, Verbose: false}
	result := CreateEmitter(cfg, "test-pipeline", "test-pipeline", []pipeline.Step{}, nil)
	defer result.Cleanup()

	if _, ok := result.Progress.(*display.ThrottledProgressEmitter); !ok {
		t.Errorf("text format should use ThrottledProgressEmitter, got %T", result.Progress)
	}
}

func TestCreateEmitter_QuietFormatUsesThrottle(t *testing.T) {
	cfg := OutputConfig{Format: OutputFormatQuiet, Verbose: false}
	result := CreateEmitter(cfg, "test-pipeline", "test-pipeline", []pipeline.Step{}, nil)
	defer result.Cleanup()

	if _, ok := result.Progress.(*display.ThrottledProgressEmitter); !ok {
		t.Errorf("quiet format should use ThrottledProgressEmitter, got %T", result.Progress)
	}
}

func TestCreateEmitter_JSONFormatNoThrottle(t *testing.T) {
	cfg := OutputConfig{Format: OutputFormatJSON, Verbose: false}
	result := CreateEmitter(cfg, "test-pipeline", "test-pipeline", []pipeline.Step{}, nil)
	defer result.Cleanup()

	if result.Progress != nil {
		t.Errorf("json format should have nil Progress, got %T", result.Progress)
	}
}

func TestResolveOutputConfig_JsonAlone(t *testing.T) {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("quiet", false, "")
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().StringP("output", "o", "auto", "")
	root.PersistentFlags().BoolP("verbose", "v", false, "")
	root.PersistentFlags().BoolP("debug", "d", false, "")
	root.PersistentFlags().Bool("no-tui", false, "")

	_ = root.PersistentFlags().Set("json", "true")

	rf, err := ResolveOutputConfig(root)
	require.NoError(t, err)
	assert.Equal(t, OutputFormatJSON, rf.Output.Format)
	assert.True(t, rf.NoTUI, "json implies no TUI")
}

func TestResolveOutputConfig_QuietAlone(t *testing.T) {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("quiet", false, "")
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().StringP("output", "o", "auto", "")
	root.PersistentFlags().BoolP("verbose", "v", false, "")
	root.PersistentFlags().BoolP("debug", "d", false, "")
	root.PersistentFlags().Bool("no-tui", false, "")

	_ = root.PersistentFlags().Set("quiet", "true")

	rf, err := ResolveOutputConfig(root)
	require.NoError(t, err)
	assert.Equal(t, OutputFormatQuiet, rf.Output.Format)
	assert.True(t, rf.NoTUI, "quiet implies no TUI")
}

func TestResolveOutputConfig_JsonOutputConflict(t *testing.T) {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("quiet", false, "")
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().StringP("output", "o", "auto", "")
	root.PersistentFlags().BoolP("verbose", "v", false, "")
	root.PersistentFlags().BoolP("debug", "d", false, "")
	root.PersistentFlags().Bool("no-tui", false, "")

	_ = root.PersistentFlags().Set("json", "true")
	_ = root.PersistentFlags().Set("output", "text")

	_, err := ResolveOutputConfig(root)
	require.Error(t, err)
	var cliErr *CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Equal(t, CodeFlagConflict, cliErr.Code)
}

func TestResolveOutputConfig_QuietOutputConflict(t *testing.T) {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("quiet", false, "")
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().StringP("output", "o", "auto", "")
	root.PersistentFlags().BoolP("verbose", "v", false, "")
	root.PersistentFlags().BoolP("debug", "d", false, "")
	root.PersistentFlags().Bool("no-tui", false, "")

	_ = root.PersistentFlags().Set("quiet", "true")
	_ = root.PersistentFlags().Set("output", "json")

	_, err := ResolveOutputConfig(root)
	require.Error(t, err)
	var cliErr *CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Equal(t, CodeFlagConflict, cliErr.Code)
}

func TestResolveOutputConfig_JsonQuietNoConflict(t *testing.T) {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("quiet", false, "")
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().StringP("output", "o", "auto", "")
	root.PersistentFlags().BoolP("verbose", "v", false, "")
	root.PersistentFlags().BoolP("debug", "d", false, "")
	root.PersistentFlags().Bool("no-tui", false, "")

	_ = root.PersistentFlags().Set("json", "true")
	_ = root.PersistentFlags().Set("quiet", "true")

	rf, err := ResolveOutputConfig(root)
	require.NoError(t, err)
	assert.Equal(t, OutputFormatJSON, rf.Output.Format, "json takes precedence over quiet")
}

func TestResolveOutputConfig_NoColor(t *testing.T) {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("quiet", false, "")
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().StringP("output", "o", "auto", "")
	root.PersistentFlags().BoolP("verbose", "v", false, "")
	root.PersistentFlags().BoolP("debug", "d", false, "")
	root.PersistentFlags().Bool("no-tui", false, "")

	_ = root.PersistentFlags().Set("no-color", "true")

	rf, err := ResolveOutputConfig(root)
	require.NoError(t, err)
	assert.True(t, rf.Output.NoColor)
}

func TestResolveOutputConfig_QuietVerbose(t *testing.T) {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("quiet", false, "")
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().StringP("output", "o", "auto", "")
	root.PersistentFlags().BoolP("verbose", "v", false, "")
	root.PersistentFlags().BoolP("debug", "d", false, "")
	root.PersistentFlags().Bool("no-tui", false, "")

	_ = root.PersistentFlags().Set("quiet", "true")
	_ = root.PersistentFlags().Set("verbose", "true")

	rf, err := ResolveOutputConfig(root)
	require.NoError(t, err)
	assert.Equal(t, OutputFormatQuiet, rf.Output.Format)
	assert.False(t, rf.Output.Verbose, "quiet should win over verbose")
}

func TestResolveFormat_RootJsonOverridesLocal(t *testing.T) {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("quiet", false, "")
	root.PersistentFlags().StringP("output", "o", "auto", "")

	_ = root.PersistentFlags().Set("json", "true")

	result := ResolveFormat(root, "table")
	assert.Equal(t, "json", result)
}

func TestResolveFormat_RootQuietOverridesLocal(t *testing.T) {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("quiet", false, "")
	root.PersistentFlags().StringP("output", "o", "auto", "")

	_ = root.PersistentFlags().Set("quiet", "true")

	result := ResolveFormat(root, "json")
	assert.Equal(t, "quiet", result)
}

func TestResolveFormat_DefaultPreservesLocal(t *testing.T) {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("quiet", false, "")
	root.PersistentFlags().StringP("output", "o", "auto", "")

	result := ResolveFormat(root, "table")
	assert.Equal(t, "table", result)
}

func TestResolveFormat_OutputTextMapsToTable(t *testing.T) {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("quiet", false, "")
	root.PersistentFlags().StringP("output", "o", "auto", "")

	_ = root.PersistentFlags().Set("output", "text")

	result := ResolveFormat(root, "json")
	assert.Equal(t, "table", result)
}
