package commands

import (
	"testing"

	"github.com/recinq/wave/internal/tui"
	"github.com/stretchr/testify/assert"
)

func TestIsInteractive_NonTTY(t *testing.T) {
	// In test environment, stdin is not a TTY (it's a pipe).
	// Unless WAVE_FORCE_TTY is set, isInteractive should return false.
	t.Setenv("WAVE_FORCE_TTY", "")
	assert.False(t, isInteractive(), "should not be interactive in test environment")
}

func TestIsInteractive_ForceOn(t *testing.T) {
	t.Setenv("WAVE_FORCE_TTY", "1")
	assert.True(t, isInteractive(), "WAVE_FORCE_TTY=1 should force interactive mode")
}

func TestIsInteractive_ForceOff(t *testing.T) {
	t.Setenv("WAVE_FORCE_TTY", "0")
	assert.False(t, isInteractive(), "WAVE_FORCE_TTY=0 should disable interactive mode")
}

func TestApplySelection_PipelineAndInput(t *testing.T) {
	opts := RunOptions{}
	debug := false
	sel := &tui.Selection{
		Pipeline: "feature",
		Input:    "add user auth",
	}
	applySelection(&opts, sel, &debug)

	assert.Equal(t, "feature", opts.Pipeline)
	assert.Equal(t, "add user auth", opts.Input)
	assert.False(t, debug)
}

func TestApplySelection_AllFlags(t *testing.T) {
	opts := RunOptions{}
	debug := false
	sel := &tui.Selection{
		Pipeline: "debug",
		Input:    "fix nil pointer",
		Flags:    []string{"--verbose", "--output json", "--dry-run", "--mock", "--debug"},
	}
	applySelection(&opts, sel, &debug)

	assert.Equal(t, "debug", opts.Pipeline)
	assert.Equal(t, "fix nil pointer", opts.Input)
	assert.True(t, opts.Output.Verbose)
	assert.Equal(t, OutputFormatJSON, opts.Output.Format)
	assert.True(t, opts.DryRun)
	assert.True(t, opts.Mock)
	assert.True(t, debug)
}

func TestApplySelection_EmptyInput(t *testing.T) {
	opts := RunOptions{Input: "existing input"}
	debug := false
	sel := &tui.Selection{
		Pipeline: "hotfix",
		Input:    "",
	}
	applySelection(&opts, sel, &debug)

	assert.Equal(t, "hotfix", opts.Pipeline)
	assert.Equal(t, "existing input", opts.Input, "empty selection input should not overwrite existing")
}

func TestApplySelection_NoFlags(t *testing.T) {
	opts := RunOptions{}
	debug := false
	sel := &tui.Selection{
		Pipeline: "refactor",
	}
	applySelection(&opts, sel, &debug)

	assert.Equal(t, "refactor", opts.Pipeline)
	assert.False(t, opts.DryRun)
	assert.False(t, opts.Mock)
	assert.False(t, opts.Output.Verbose)
	assert.False(t, debug)
}

func TestNewRunCmd_NoPipelineNonTTY(t *testing.T) {
	// When no pipeline is provided and not on a TTY, should return error.
	t.Setenv("WAVE_FORCE_TTY", "0")
	cmd := NewRunCmd()
	// Set root command to avoid persistent flag issues.
	cmd.Root().PersistentFlags().String("output", "auto", "")
	cmd.Root().PersistentFlags().Bool("verbose", false, "")
	cmd.Root().PersistentFlags().Bool("debug", false, "")
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline name is required")
}

func TestNewRunCmd_FullArgsBypassTUI(t *testing.T) {
	// When full args are provided, TUI should not be invoked at all.
	// This test verifies args are correctly parsed and execution proceeds
	// past the TUI check directly to manifest/pipeline loading.
	t.Setenv("WAVE_FORCE_TTY", "0")
	cmd := NewRunCmd()
	cmd.Root().PersistentFlags().String("output", "auto", "")
	cmd.Root().PersistentFlags().Bool("verbose", false, "")
	cmd.Root().PersistentFlags().Bool("debug", false, "")

	// Use a nonexistent pipeline — will fail at manifest/pipeline load, not TUI.
	cmd.SetArgs([]string{"nonexistent-pipeline", "some input"})
	err := cmd.Execute()
	assert.Error(t, err)
	// Error should NOT be about "pipeline name is required" — that would mean TUI logic ran.
	assert.NotContains(t, err.Error(), "pipeline name is required")
}

func TestPipelinesDir(t *testing.T) {
	assert.Equal(t, ".wave/pipelines", pipelinesDir())
}
