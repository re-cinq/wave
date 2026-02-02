package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/recinq/wave/internal/pipeline"
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
skill_mounts: []
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

// TestDoCommand_SaveWritesPipelineFile verifies that --save flag writes
// a valid pipeline YAML file
func TestDoCommand_SaveWritesPipelineFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestManifest(t, tmpDir, []string{"navigator", "craftsman"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		Save:     "my-pipeline",
		DryRun:   true,
		Mock:     true,
	}

	output := captureOutput(t, func() {
		err := runDo("build feature Y", opts)
		require.NoError(t, err)
	})

	// Verify save confirmation in output
	assert.Contains(t, output, "Pipeline saved")
	assert.Contains(t, output, "my-pipeline.yaml")

	// Verify the pipeline file was created
	pipelinePath := filepath.Join(tmpDir, ".wave/pipelines/my-pipeline.yaml")
	assert.FileExists(t, pipelinePath)

	// Read and validate the saved pipeline
	data, err := os.ReadFile(pipelinePath)
	require.NoError(t, err)

	var p pipeline.Pipeline
	err = yaml.Unmarshal(data, &p)
	require.NoError(t, err)

	assert.Equal(t, "WavePipeline", p.Kind)
	assert.Equal(t, "adhoc", p.Metadata.Name)
	assert.Len(t, p.Steps, 2)
	assert.Equal(t, "navigate", p.Steps[0].ID)
	assert.Equal(t, "execute", p.Steps[1].ID)
	assert.Equal(t, "navigator", p.Steps[0].Persona)
	assert.Equal(t, "craftsman", p.Steps[1].Persona)
}

// TestDoCommand_SaveWithAbsolutePath verifies that --save with an absolute
// path writes to that location directly
func TestDoCommand_SaveWithAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestManifest(t, tmpDir, []string{"navigator", "craftsman"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	customPath := filepath.Join(tmpDir, "custom/location/pipeline.yaml")
	opts := DoOptions{
		Manifest: "wave.yaml",
		Save:     customPath,
		DryRun:   true,
		Mock:     true,
	}

	output := captureOutput(t, func() {
		err := runDo("build feature", opts)
		require.NoError(t, err)
	})

	// Verify save confirmation with custom path
	assert.Contains(t, output, "Pipeline saved")
	assert.Contains(t, output, customPath)

	// Verify the pipeline file was created at the custom path
	assert.FileExists(t, customPath)
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
		Save:     "debug-pipeline",
		DryRun:   true,
		Mock:     true,
	}

	captureOutput(t, func() {
		err := runDo("debug issue", opts)
		require.NoError(t, err)
	})

	// Read the saved pipeline and verify persona
	pipelinePath := filepath.Join(tmpDir, ".wave/pipelines/debug-pipeline.yaml")
	data, err := os.ReadFile(pipelinePath)
	require.NoError(t, err)

	var p pipeline.Pipeline
	err = yaml.Unmarshal(data, &p)
	require.NoError(t, err)

	// Find execute step and verify persona
	var executeStep *pipeline.Step
	for i := range p.Steps {
		if p.Steps[i].ID == "execute" {
			executeStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, executeStep)
	assert.Equal(t, "debugger", executeStep.Persona)
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

// TestDoCommand_PipelineStepsHaveDependencies verifies that the generated
// pipeline has proper step dependencies
func TestDoCommand_PipelineStepsHaveDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestManifest(t, tmpDir, []string{"navigator", "craftsman"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		Save:     "test-deps",
		DryRun:   true,
		Mock:     true,
	}

	captureOutput(t, func() {
		err := runDo("test dependencies", opts)
		require.NoError(t, err)
	})

	// Read the saved pipeline
	pipelinePath := filepath.Join(tmpDir, ".wave/pipelines/test-deps.yaml")
	data, err := os.ReadFile(pipelinePath)
	require.NoError(t, err)

	var p pipeline.Pipeline
	err = yaml.Unmarshal(data, &p)
	require.NoError(t, err)

	// Find execute step and verify it depends on navigate
	var executeStep *pipeline.Step
	for i := range p.Steps {
		if p.Steps[i].ID == "execute" {
			executeStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, executeStep)
	assert.Contains(t, executeStep.Dependencies, "navigate")
}

// TestDoCommand_PipelineHasArtifactInjection verifies that the execute step
// has artifact injection from navigate step
func TestDoCommand_PipelineHasArtifactInjection(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestManifest(t, tmpDir, []string{"navigator", "craftsman"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		Save:     "test-artifacts",
		DryRun:   true,
		Mock:     true,
	}

	captureOutput(t, func() {
		err := runDo("test artifacts", opts)
		require.NoError(t, err)
	})

	// Read the saved pipeline
	pipelinePath := filepath.Join(tmpDir, ".wave/pipelines/test-artifacts.yaml")
	data, err := os.ReadFile(pipelinePath)
	require.NoError(t, err)

	var p pipeline.Pipeline
	err = yaml.Unmarshal(data, &p)
	require.NoError(t, err)

	// Find execute step and verify artifact injection
	var executeStep *pipeline.Step
	for i := range p.Steps {
		if p.Steps[i].ID == "execute" {
			executeStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, executeStep)
	assert.NotEmpty(t, executeStep.Memory.InjectArtifacts)

	// Verify artifact comes from navigate step
	found := false
	for _, ref := range executeStep.Memory.InjectArtifacts {
		if ref.Step == "navigate" {
			found = true
			break
		}
	}
	assert.True(t, found, "execute step should inject artifacts from navigate step")
}

// TestDoCommand_SaveAddsYamlExtension verifies that .yaml extension is
// added when missing
func TestDoCommand_SaveAddsYamlExtension(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestManifest(t, tmpDir, []string{"navigator", "craftsman"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		Save:     "no-extension",
		DryRun:   true,
		Mock:     true,
	}

	captureOutput(t, func() {
		err := runDo("test extension", opts)
		require.NoError(t, err)
	})

	// Verify .yaml was added
	pipelinePath := filepath.Join(tmpDir, ".wave/pipelines/no-extension.yaml")
	assert.FileExists(t, pipelinePath)
}

// TestDoCommand_VerifyWorkspaceConfig verifies that workspace configuration
// in generated pipeline is correct
func TestDoCommand_VerifyWorkspaceConfig(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestManifest(t, tmpDir, []string{"navigator", "craftsman"})

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	require.NoError(t, os.Chdir(tmpDir))

	opts := DoOptions{
		Manifest: "wave.yaml",
		Save:     "test-workspace",
		DryRun:   true,
		Mock:     true,
	}

	captureOutput(t, func() {
		err := runDo("test workspace", opts)
		require.NoError(t, err)
	})

	// Read the saved pipeline
	pipelinePath := filepath.Join(tmpDir, ".wave/pipelines/test-workspace.yaml")
	data, err := os.ReadFile(pipelinePath)
	require.NoError(t, err)

	var p pipeline.Pipeline
	err = yaml.Unmarshal(data, &p)
	require.NoError(t, err)

	// Verify navigate step has readonly mount
	var navigateStep *pipeline.Step
	for i := range p.Steps {
		if p.Steps[i].ID == "navigate" {
			navigateStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, navigateStep)
	require.NotEmpty(t, navigateStep.Workspace.Mount)

	foundReadonly := false
	for _, mount := range navigateStep.Workspace.Mount {
		if mount.Mode == "readonly" {
			foundReadonly = true
			break
		}
	}
	assert.True(t, foundReadonly, "navigate step should have readonly mount")

	// Verify execute step has readwrite mount
	var executeStep *pipeline.Step
	for i := range p.Steps {
		if p.Steps[i].ID == "execute" {
			executeStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, executeStep)
	require.NotEmpty(t, executeStep.Workspace.Mount)

	foundReadwrite := false
	for _, mount := range executeStep.Workspace.Mount {
		if mount.Mode == "readwrite" {
			foundReadwrite = true
			break
		}
	}
	assert.True(t, foundReadwrite, "execute step should have readwrite mount")
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
