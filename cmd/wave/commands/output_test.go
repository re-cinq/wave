package commands

import (
	"testing"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
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
