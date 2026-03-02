package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunCmdSilencesUsageOnPipelineError verifies that cobra's automatic usage
// text is suppressed when runRun returns an error. Argument validation errors
// (e.g., invalid output format) should still show usage text because SilenceUsage
// is set only after validation passes.
func TestRunCmdSilencesUsageOnPipelineError(t *testing.T) {
	// Create a temporary directory with a minimal wave.yaml so the manifest
	// loads successfully but no pipelines exist, causing runRun to fail.
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")
	err := os.WriteFile(manifestPath, []byte("metadata:\n  name: test\nadapters:\n  claude:\n    binary: claude\n    mode: headless\n"), 0644)
	require.NoError(t, err)

	// Ensure non-interactive mode (no TTY prompts).
	t.Setenv("WAVE_FORCE_TTY", "0")

	cmd := NewRunCmd()

	// Register persistent flags that the root command normally provides,
	// since in tests there is no parent root command.
	cmd.PersistentFlags().String("output", "auto", "")
	cmd.PersistentFlags().Bool("verbose", false, "")

	// SilenceUsage should be false initially — cobra's default.
	assert.False(t, cmd.SilenceUsage, "SilenceUsage should be false before execution")

	// Execute with a pipeline name that won't be found but passes arg validation.
	// Use --force to skip onboarding check.
	cmd.SetArgs([]string{
		"nonexistent-pipeline",
		"--manifest", manifestPath,
		"--force",
	})

	// Silence errors to avoid cobra printing to stderr during tests.
	cmd.SilenceErrors = true

	err = cmd.Execute()
	assert.Error(t, err, "should fail because the pipeline doesn't exist")

	// After RunE executes past the argument validation, SilenceUsage must be true
	// so cobra does not append usage/help text to the error output.
	assert.True(t, cmd.SilenceUsage, "SilenceUsage should be true after runRun error")
	assert.True(t, cmd.SilenceErrors, "SilenceErrors should be true after runRun error")
}
