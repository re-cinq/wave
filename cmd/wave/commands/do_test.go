package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions

// setupTestManifest creates a test manifest file with required personas
func setupTestManifest(t *testing.T, dir string, personas []string) string {
	t.Helper()

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
  description: Test project for do command
adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
personas:
`
	for _, p := range personas {
		manifestContent += `  ` + p + `:
    adapter: claude
    system_prompt_file: personas/` + p + `.md
    temperature: 0.7
    permissions:
      allowed_tools:
        - Read
        - Write
      deny: []
`
	}

	manifestContent += `runtime:
  workspace_root: .wave/workspaces
  default_timeout_minutes: 30
`

	// Create personas directory and files
	personasDir := filepath.Join(dir, "personas")
	require.NoError(t, os.MkdirAll(personasDir, 0755))

	for _, p := range personas {
		promptContent := "You are a " + p + " persona for testing."
		require.NoError(t, os.WriteFile(filepath.Join(personasDir, p+".md"), []byte(promptContent), 0644))
	}

	manifestPath := filepath.Join(dir, "wave.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0644))

	return manifestPath
}

// captureOutput captures stdout during test execution
func captureOutput(t *testing.T, f func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// TestDoCommand_GeneratesTwoStepPipeline verifies that `wave do` generates
// a two-step pipeline with navigate and execute steps
func TestDoCommand_GeneratesTwoStepPipeline(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestManifest(t, tmpDir, []string{"navigator", "craftsman"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		DryRun:   true,
		Mock:     true,
	}

	output := captureOutput(t, func() {
		err := runDo("implement feature X", opts)
		require.NoError(t, err)
	})

	// Verify output mentions two-step pipeline structure
	assert.Contains(t, output, "navigate")
	assert.Contains(t, output, "execute")
	assert.Contains(t, output, "Step")
}

// TestDoCommand_PersonaOverride verifies that --persona flag overrides
// the default execute persona
func TestDoCommand_PersonaOverride(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestManifest(t, tmpDir, []string{"navigator", "craftsman", "architect"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		Persona:  "architect",
		DryRun:   true,
		Mock:     true,
	}

	output := captureOutput(t, func() {
		err := runDo("design system architecture", opts)
		require.NoError(t, err)
	})

	// Verify the architect persona is used for execute step
	assert.Contains(t, output, "architect")
}

// TestDoCommand_DryRunOutput verifies that --dry-run prints pipeline
// structure without executing
func TestDoCommand_DryRunOutput(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestManifest(t, tmpDir, []string{"navigator", "craftsman"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		DryRun:   true,
		Mock:     true,
	}

	output := captureOutput(t, func() {
		err := runDo("test task description", opts)
		require.NoError(t, err)
	})

	// Verify dry-run output structure
	assert.Contains(t, output, "Ad-hoc pipeline")
	assert.Contains(t, output, "navigate")
	assert.Contains(t, output, "execute")
	assert.Contains(t, output, "Input:")
	assert.Contains(t, output, "test task description")
	assert.Contains(t, output, "Steps:")
	assert.Contains(t, output, "Workspace:")

	// Verify no workspace was created (dry-run should not create files)
	_, err = os.Stat(filepath.Join(tmpDir, ".wave/workspaces/adhoc"))
	assert.True(t, os.IsNotExist(err), "workspace should not be created in dry-run mode")
}

// TestDoCommand_MissingManifestError verifies that missing manifest
// produces a clear error message with helpful guidance
func TestDoCommand_MissingManifestError(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to test directory (no manifest)
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		DryRun:   true,
		Mock:     true,
	}

	err = runDo("test task", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest file not found")
	assert.Contains(t, err.Error(), "wave init")
}

// TestDoCommand_InvalidManifestError verifies that invalid manifest YAML
// produces a clear error message with helpful guidance
func TestDoCommand_InvalidManifestError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid manifest
	invalidManifest := `apiVersion: v1
kind: WaveManifest
metadata:
  - invalid yaml structure
`
	manifestPath := filepath.Join(tmpDir, "wave.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(invalidManifest), 0644))

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		DryRun:   true,
		Mock:     true,
	}

	err = runDo("test task", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
	assert.Contains(t, err.Error(), "valid YAML")
}

// TestDoCommand_MissingPersonaError verifies that referencing a missing
// persona produces a clear error
func TestDoCommand_MissingPersonaError(t *testing.T) {
	tmpDir := t.TempDir()
	// Only create navigator, not craftsman (default execute persona)
	setupTestManifest(t, tmpDir, []string{"navigator"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		DryRun:   true,
		Mock:     true,
	}

	err = runDo("test task", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "persona")
	assert.Contains(t, err.Error(), "craftsman")
}

// TestDoCommand_MissingNavigatorPersonaError verifies that missing navigator
// persona produces a clear error
func TestDoCommand_MissingNavigatorPersonaError(t *testing.T) {
	tmpDir := t.TempDir()
	// Only create craftsman, not navigator
	setupTestManifest(t, tmpDir, []string{"craftsman"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		DryRun:   true,
		Mock:     true,
	}

	err = runDo("test task", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "navigator")
}

// TestDoCommand_CustomPersonaOverride verifies that --persona correctly
// overrides the default craftsman persona
func TestDoCommand_CustomPersonaOverride(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestManifest(t, tmpDir, []string{"navigator", "craftsman", "debugger"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		Persona:  "debugger",
		DryRun:   true,
		Mock:     true,
	}

	output := captureOutput(t, func() {
		err := runDo("debug issue", opts)
		require.NoError(t, err)
	})

	// Verify the debugger persona is used for execute step
	assert.Contains(t, output, "debugger")
}

// TestNewDoCmd verifies the command structure and flags
func TestNewDoCmd(t *testing.T) {
	cmd := NewDoCmd()

	assert.Equal(t, "do [task description]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Verify flags exist
	flags := cmd.Flags()

	personaFlag := flags.Lookup("persona")
	require.NotNil(t, personaFlag)
	assert.Equal(t, "", personaFlag.DefValue)

	manifestFlag := flags.Lookup("manifest")
	require.NotNil(t, manifestFlag)
	assert.Equal(t, "wave.yaml", manifestFlag.DefValue)

	mockFlag := flags.Lookup("mock")
	require.NotNil(t, mockFlag)
	assert.Equal(t, "false", mockFlag.DefValue)

	dryRunFlag := flags.Lookup("dry-run")
	require.NotNil(t, dryRunFlag)
	assert.Equal(t, "false", dryRunFlag.DefValue)

	// Verify --save and --meta flags no longer exist (moved to wave meta command)
	assert.Nil(t, flags.Lookup("save"), "save flag should not exist on do command")
	assert.Nil(t, flags.Lookup("meta"), "meta flag should not exist on do command")
}

// TestDoCommand_MultiWordTaskDescription verifies that multi-word task
// descriptions are properly joined
func TestDoCommand_MultiWordTaskDescription(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestManifest(t, tmpDir, []string{"navigator", "craftsman"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		DryRun:   true,
		Mock:     true,
	}

	output := captureOutput(t, func() {
		err := runDo("implement a new feature with multiple components", opts)
		require.NoError(t, err)
	})

	// Verify the full task description appears in output
	assert.Contains(t, output, "implement a new feature with multiple components")
}

// TestDoCommand_EmptyInput verifies that empty input is handled (by cobra)
func TestDoCommand_EmptyInput(t *testing.T) {
	cmd := NewDoCmd()

	// Cobra should reject empty args due to MinimumNArgs(1)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "at least") || strings.Contains(err.Error(), "requires"))
}

