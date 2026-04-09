package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestManifestWithPhilosopher creates a test manifest with philosopher persona
// required for meta-pipeline testing
func setupTestManifestWithPhilosopher(t *testing.T, dir string) string {
	t.Helper()

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
  description: Test project for meta-pipeline
adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/navigator.md
    temperature: 0.7
    permissions:
      allowed_tools:
        - Read
      deny: []
  craftsman:
    adapter: claude
    system_prompt_file: personas/craftsman.md
    temperature: 0.7
    permissions:
      allowed_tools:
        - Read
        - Write
      deny: []
  philosopher:
    adapter: claude
    system_prompt_file: personas/philosopher.md
    temperature: 0.7
    permissions:
      allowed_tools:
        - Read
      deny: []
runtime:
  workspace_root: .wave/workspaces
  default_timeout_minutes: 30
  meta_pipeline:
    max_depth: 2
    max_total_steps: 20
    max_total_tokens: 500000
    timeout_minutes: 60
`

	// Create personas directory and files
	personasDir := filepath.Join(dir, "personas")
	require.NoError(t, os.MkdirAll(personasDir, 0755))

	personas := []string{"navigator", "craftsman", "philosopher"}
	for _, p := range personas {
		promptContent := "You are a " + p + " persona for testing."
		require.NoError(t, os.WriteFile(filepath.Join(personasDir, p+".md"), []byte(promptContent), 0644))
	}

	manifestPath := filepath.Join(dir, "wave.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0644))

	return manifestPath
}

// TestNewMetaCmd verifies the command structure and flags
func TestNewMetaCmd(t *testing.T) {
	cmd := NewMetaCmd()

	assert.Equal(t, "meta [task description]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "philosopher")
	assert.Contains(t, cmd.Long, "multi-step")

	// Verify flags exist
	flags := cmd.Flags()

	saveFlag := flags.Lookup("save")
	require.NotNil(t, saveFlag)
	assert.Equal(t, "", saveFlag.DefValue)

	manifestFlag := flags.Lookup("manifest")
	require.NotNil(t, manifestFlag)
	assert.Equal(t, "wave.yaml", manifestFlag.DefValue)

	mockFlag := flags.Lookup("mock")
	require.NotNil(t, mockFlag)
	assert.Equal(t, "false", mockFlag.DefValue)

	dryRunFlag := flags.Lookup("dry-run")
	require.NotNil(t, dryRunFlag)
	assert.Equal(t, "false", dryRunFlag.DefValue)

	modelFlag := flags.Lookup("model")
	require.NotNil(t, modelFlag)
	assert.Equal(t, "", modelFlag.DefValue)
}

// TestMetaCommand_MissingManifestError verifies that missing manifest
// produces a clear error message with helpful guidance
func TestMetaCommand_MissingManifestError(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to test directory (no manifest)
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	opts := MetaOptions{
		Manifest: "wave.yaml",
		DryRun:   true,
		Mock:     true,
	}

	err = runMeta("test task", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest file not found")
	var cliErr *CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Equal(t, CodeManifestMissing, cliErr.Code)
	assert.Contains(t, cliErr.Suggestion, "wave init")
}

// TestMetaCommand_InvalidManifestError verifies that invalid manifest YAML
// produces a clear error message with helpful guidance
func TestMetaCommand_InvalidManifestError(t *testing.T) {
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
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	opts := MetaOptions{
		Manifest: "wave.yaml",
		DryRun:   true,
		Mock:     true,
	}

	err = runMeta("test task", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
	var cliErr2 *CLIError
	require.ErrorAs(t, err, &cliErr2)
	assert.Equal(t, CodeManifestInvalid, cliErr2.Code)
	assert.Contains(t, cliErr2.Suggestion, "valid YAML")
}

// TestMetaCommand_MissingPhilosopher verifies that missing philosopher
// persona produces a clear error
func TestMetaCommand_MissingPhilosopher(t *testing.T) {
	tmpDir := t.TempDir()
	// Setup manifest without philosopher persona
	setupTestManifest(t, tmpDir, []string{"navigator", "craftsman"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	opts := MetaOptions{
		Manifest: "wave.yaml",
		DryRun:   true,
		Mock:     true,
	}

	err = runMeta("implement feature X", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "philosopher")
}

// TestMetaCommand_EmptyInput verifies that empty input is handled (by cobra)
func TestMetaCommand_EmptyInput(t *testing.T) {
	cmd := NewMetaCmd()

	// Cobra should reject empty args due to MinimumNArgs(1)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
}

// TestMetaCommand_HelpText verifies the help text mentions key features
func TestMetaCommand_HelpText(t *testing.T) {
	cmd := NewMetaCmd()

	// Verify examples are present in the help
	assert.Contains(t, cmd.Long, "wave meta \"implement user authentication\"")
	assert.Contains(t, cmd.Long, "--dry-run")
	assert.Contains(t, cmd.Long, "--save")
}

// TestSaveMetaPipelinePath verifies save path logic
func TestSaveMetaPipelinePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	minimalPipeline := &pipeline.Pipeline{
		Kind:     "WavePipeline",
		Metadata: pipeline.PipelineMetadata{Name: "test"},
		Steps: []pipeline.Step{
			{ID: "nav", Persona: "navigator"},
		},
	}

	tests := []struct {
		name         string
		savePath     string
		expectedFile string
	}{
		{
			name:         "bare name gets .wave/pipelines prefix and .yaml extension",
			savePath:     "my-pipeline",
			expectedFile: ".wave/pipelines/my-pipeline.yaml",
		},
		{
			name:         "name with .yaml does not get double extension",
			savePath:     "my-pipeline.yaml",
			expectedFile: ".wave/pipelines/my-pipeline.yaml",
		},
		{
			name:         "path with / is used as-is",
			savePath:     "custom/dir/my-pipeline.yaml",
			expectedFile: "custom/dir/my-pipeline.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := saveMetaPipeline(minimalPipeline, tc.savePath)
			require.NoError(t, err)

			_, statErr := os.Stat(filepath.Join(tmpDir, tc.expectedFile))
			assert.NoError(t, statErr, "expected file at %s", tc.expectedFile)
		})
	}
}
