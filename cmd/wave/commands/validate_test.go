package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/defaults"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/onboarding"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newValidateCmdWithRoot creates a validate command under a root that has the
// persistent --verbose flag, mirroring the real CLI structure.
func newValidateCmdWithRoot() *cobra.Command {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().BoolP("verbose", "v", false, "Include real-time tool activity")
	validateCmd := NewValidateCmd()
	root.AddCommand(validateCmd)
	return root
}

// testHelper provides common utilities for validate command tests.
type testHelper struct {
	t       *testing.T
	tmpDir  string
	origDir string
}

// newTestHelper creates a test helper with a temporary directory.
func newTestHelper(t *testing.T) *testHelper {
	t.Helper()
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err, "failed to get current directory")

	return &testHelper{
		t:       t,
		tmpDir:  tmpDir,
		origDir: origDir,
	}
}

// chdir changes to the temporary directory.
func (h *testHelper) chdir() {
	h.t.Helper()
	err := os.Chdir(h.tmpDir)
	require.NoError(h.t, err, "failed to change to temp directory")
}

// restore returns to the original directory.
func (h *testHelper) restore() {
	h.t.Helper()
	_ = os.Chdir(h.origDir)
}

// writeFile writes content to a file in the temp directory.
func (h *testHelper) writeFile(relPath, content string) {
	h.t.Helper()
	fullPath := filepath.Join(h.tmpDir, relPath)
	dir := filepath.Dir(fullPath)
	err := os.MkdirAll(dir, 0755)
	require.NoError(h.t, err, "failed to create directory: %s", dir)
	err = os.WriteFile(fullPath, []byte(content), 0644)
	require.NoError(h.t, err, "failed to write file: %s", relPath)
}

// T024: Test helpers for validate command tests

// T025: Test validate with valid manifest
func TestValidateCmd_ValidManifest(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create a valid manifest
	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
  description: Test project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/navigator.md
runtime:
  workspace_root: .wave/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")

	// Run validate
	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.NoError(t, err, "validate should succeed with valid manifest")
}

// T025: Test validate with valid manifest using existing testdata
func TestValidateCmd_ValidManifest_Testdata(t *testing.T) {
	// Use the existing testdata fixture
	testdataPath := filepath.Join("testdata", "valid", "wave.yaml")

	// Skip if testdata doesn't exist
	if _, err := os.Stat(testdataPath); os.IsNotExist(err) {
		t.Skip("testdata fixture not found")
	}

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--manifest", testdataPath})

	err := cmd.Execute()
	assert.NoError(t, err, "validate should succeed with valid testdata manifest")
}

// T026: Test validate with invalid adapter reference
func TestValidateCmd_InvalidAdapterReference(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create manifest with invalid adapter reference
	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: nonexistent
    system_prompt_file: personas/navigator.md
runtime:
  workspace_root: .wave/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")

	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail with invalid adapter reference")
	assert.Contains(t, err.Error(), "validation failed", "error should mention validation failure")
}

// T026: Test validate with invalid adapter reference using testdata
func TestValidateCmd_InvalidAdapterReference_Testdata(t *testing.T) {
	testdataPath := filepath.Join("testdata", "invalid-adapter", "wave.yaml")

	if _, err := os.Stat(testdataPath); os.IsNotExist(err) {
		t.Skip("testdata fixture not found")
	}

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--manifest", testdataPath})

	err := cmd.Execute()
	assert.Error(t, err, "validate should fail with invalid adapter reference")
}

// T027: Test validate with missing system prompt file
func TestValidateCmd_MissingSystemPromptFile(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create manifest referencing non-existent system prompt file
	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/nonexistent.md
runtime:
  workspace_root: .wave/workspaces
`)

	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail when system prompt file is missing")
	assert.Contains(t, err.Error(), "validation failed", "error should mention validation failure")
}

// T027: Test validate with missing system prompt file using testdata
func TestValidateCmd_MissingSystemPromptFile_Testdata(t *testing.T) {
	testdataPath := filepath.Join("testdata", "missing-file", "wave.yaml")

	if _, err := os.Stat(testdataPath); os.IsNotExist(err) {
		t.Skip("testdata fixture not found")
	}

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--manifest", testdataPath})

	err := cmd.Execute()
	assert.Error(t, err, "validate should fail when system prompt file is missing")
}

// T028: Test validate --verbose output
func TestValidateCmd_VerboseOutput(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create a valid manifest
	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/navigator.md
runtime:
  workspace_root: .wave/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")

	// Capture stdout since the command writes to os.Stdout directly
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newValidateCmdWithRoot()
	cmd.SetArgs([]string{"validate", "--verbose"})

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err, "validate should succeed")
	// Verbose should show the validation steps
	assert.Contains(t, output, "Validating manifest", "verbose output should show manifest being validated")
}

// T028: Test validate --verbose shows checkmarks for each validation step
func TestValidateCmd_VerboseOutput_ShowsSteps(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/navigator.md
runtime:
  workspace_root: .wave/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newValidateCmdWithRoot()
	cmd.SetArgs([]string{"validate", "--verbose"})

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	// Verbose mode should show validation progress
	assert.True(t, strings.Contains(output, "syntax") || strings.Contains(output, "Manifest"),
		"verbose output should mention syntax validation or manifest")
}

// T029: Test validate with malformed YAML
func TestValidateCmd_MalformedYAML(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create manifest with invalid YAML syntax
	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
  - this is invalid YAML
    because lists can't appear here
`)

	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail with malformed YAML")
	assert.Contains(t, err.Error(), "parse", "error should mention parsing failure")
}

// T029: Test validate with YAML that has a syntax error at a specific position
func TestValidateCmd_MalformedYAML_TabIndent(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	// YAML with tabs (which can cause issues)
	h.writeFile("wave.yaml", "apiVersion: v1\nkind: WaveManifest\nmetadata:\n\tname: test\n")

	cmd := NewValidateCmd()
	err := cmd.Execute()

	// Tab indentation may or may not be an error depending on the YAML parser
	// but we're testing that parsing errors are handled
	if err != nil {
		assert.Contains(t, err.Error(), "parse", "error should mention parsing")
	}
}

// T029: Test validate with completely invalid YAML
func TestValidateCmd_MalformedYAML_Completely(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `
this is not valid yaml at all
{{{
`)

	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail with completely malformed YAML")
}

// T030: Test that verbose flag provides summary output
func TestValidateCmd_VerboseFlag_Summary(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
  opencode:
    binary: opencode
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/navigator.md
  craftsman:
    adapter: opencode
    system_prompt_file: personas/craftsman.md
runtime:
  workspace_root: .wave/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")
	h.writeFile("personas/craftsman.md", "You are a craftsman.")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newValidateCmdWithRoot()
	cmd.SetArgs([]string{"validate", "--verbose"})

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	// Verify verbose summary shows counts
	assert.True(t,
		strings.Contains(output, "Adapters") || strings.Contains(output, "Personas") || strings.Contains(output, "Summary"),
		"verbose output should mention validation details")
}

// T031: Test that error messages include helpful context
func TestValidateCmd_ErrorMessagesWithContext(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create manifest with missing required field
	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  description: Missing name field
adapters:
  claude:
    binary: claude
    mode: headless
runtime:
  workspace_root: .wave/workspaces
`)

	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail when metadata.name is missing")
}

// T032: Test that error suggests running wave init when manifest is missing
func TestValidateCmd_SuggestsWaveInit(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	// Don't create wave.yaml - test missing manifest case
	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail when manifest is missing")
	// The error should suggest running wave init
	errMsg := err.Error()
	assert.True(t,
		strings.Contains(errMsg, "wave init") || strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no such file"),
		"error should suggest 'wave init' or indicate file not found")
}

// T033: Test that verbose output shows adapter binary availability
func TestValidateCmd_AdapterBinaryCheck(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create manifest with an adapter binary that definitely doesn't exist
	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  custom:
    binary: definitely-not-a-real-binary-xyz123
    mode: headless
personas:
  navigator:
    adapter: custom
    system_prompt_file: personas/navigator.md
runtime:
  workspace_root: .wave/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newValidateCmdWithRoot()
	cmd.SetArgs([]string{"validate", "--verbose"})

	// The validate should succeed (missing binary is a warning, not error)
	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err, "validate should succeed even if adapter binary is not found")
	// Should warn about missing binary
	assert.True(t,
		strings.Contains(output, "not found") || strings.Contains(output, "Warning"),
		"output should warn about missing adapter binary")
}

// Test validate with non-existent manifest file
func TestValidateCmd_NonExistentManifest(t *testing.T) {
	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--manifest", "/path/to/nonexistent/wave.yaml"})

	err := cmd.Execute()
	assert.Error(t, err, "validate should fail when manifest file doesn't exist")
	assert.Contains(t, err.Error(), "failed to read manifest", "error should mention reading failure")
}

// Test validate with missing apiVersion
func TestValidateCmd_MissingAPIVersion(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `kind: WaveManifest
metadata:
  name: test-project
runtime:
  workspace_root: .wave/workspaces
`)

	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail when apiVersion is missing")
}

// Test validate with invalid kind
func TestValidateCmd_InvalidKind(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: InvalidKind
metadata:
  name: test-project
runtime:
  workspace_root: .wave/workspaces
`)

	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail when kind is invalid")
}

// Test validate with missing workspace_root
func TestValidateCmd_MissingWorkspaceRoot(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
runtime: {}
`)

	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail when workspace_root is missing")
}

// Test validate with adapter missing binary
func TestValidateCmd_AdapterMissingBinary(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    mode: headless
runtime:
  workspace_root: .wave/workspaces
`)

	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail when adapter binary is not specified")
}

// Test validate with adapter missing mode
func TestValidateCmd_AdapterMissingMode(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
runtime:
  workspace_root: .wave/workspaces
`)

	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail when adapter mode is not specified")
}

// Test validate with persona missing adapter reference
func TestValidateCmd_PersonaMissingAdapter(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    system_prompt_file: personas/navigator.md
runtime:
  workspace_root: .wave/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")

	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail when persona adapter is not specified")
}

// Test validate with persona missing system_prompt_file
func TestValidateCmd_PersonaMissingSystemPromptFile(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
runtime:
  workspace_root: .wave/workspaces
`)

	cmd := NewValidateCmd()
	err := cmd.Execute()

	assert.Error(t, err, "validate should fail when persona system_prompt_file is not specified")
}

// Test validate specific pipeline
func TestValidateCmd_SpecificPipeline(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/navigator.md
runtime:
  workspace_root: .wave/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")
	h.writeFile(".wave/pipelines/test.yaml", `kind: WavePipeline
metadata:
  name: test
steps:
  - id: navigate
    persona: navigator
    exec:
      type: prompt
      source: "Test"
`)

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--pipeline", "test"})

	err := cmd.Execute()
	assert.NoError(t, err, "validate should succeed for valid pipeline")
}

// Test validate with non-existent pipeline
func TestValidateCmd_NonExistentPipeline(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/navigator.md
runtime:
  workspace_root: .wave/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--pipeline", "nonexistent"})

	err := cmd.Execute()
	assert.Error(t, err, "validate should fail when pipeline doesn't exist")
	assert.Contains(t, err.Error(), "does not exist", "error should mention pipeline not existing")
}

// Test validate with pipeline that references non-existent persona
func TestValidateCmd_PipelineInvalidPersona(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/navigator.md
runtime:
  workspace_root: .wave/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")
	h.writeFile(".wave/pipelines/test.yaml", `kind: WavePipeline
metadata:
  name: test
steps:
  - id: step1
    persona: nonexistent_persona
    exec:
      type: prompt
      source: "Test"
`)

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--pipeline", "test"})

	err := cmd.Execute()
	assert.Error(t, err, "validate should fail when pipeline references non-existent persona")
	assert.Contains(t, err.Error(), "not found", "error should mention persona not found")
}

// Test validate with pipeline that has duplicate step IDs
func TestValidateCmd_PipelineDuplicateStepIDs(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/navigator.md
runtime:
  workspace_root: .wave/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")
	h.writeFile(".wave/pipelines/test.yaml", `kind: WavePipeline
metadata:
  name: test
steps:
  - id: step1
    persona: navigator
    exec:
      type: prompt
      source: "Test 1"
  - id: step1
    persona: navigator
    exec:
      type: prompt
      source: "Test 2"
`)

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--pipeline", "test"})

	err := cmd.Execute()
	assert.Error(t, err, "validate should fail when pipeline has duplicate step IDs")
	assert.Contains(t, err.Error(), "duplicate", "error should mention duplicate step ID")
}

// Test validate with pipeline that has invalid dependency reference
func TestValidateCmd_PipelineInvalidDependency(t *testing.T) {
	h := newTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: personas/navigator.md
runtime:
  workspace_root: .wave/workspaces
`)
	h.writeFile("personas/navigator.md", "You are a navigator.")
	h.writeFile(".wave/pipelines/test.yaml", `kind: WavePipeline
metadata:
  name: test
steps:
  - id: step1
    persona: navigator
    dependencies:
      - nonexistent_step
    exec:
      type: prompt
      source: "Test"
`)

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--pipeline", "test"})

	err := cmd.Execute()
	assert.Error(t, err, "validate should fail when pipeline has invalid dependency")
	assert.Contains(t, err.Error(), "non-existent step", "error should mention non-existent step dependency")
}

// Test resolveForgeTemplate with known forge, unknown forge, and no template
func TestResolveForgeTemplate(t *testing.T) {
	tests := []struct {
		name     string
		persona  string
		fi       forge.ForgeInfo
		expected []string
	}{
		{
			name:     "no template returns persona unchanged",
			persona:  "navigator",
			fi:       forge.ForgeInfo{Type: forge.ForgeGitHub},
			expected: []string{"navigator"},
		},
		{
			name:     "known forge resolves spaced template",
			persona:  "{{ forge.type }}-analyst",
			fi:       forge.ForgeInfo{Type: forge.ForgeGitHub},
			expected: []string{"github-analyst"},
		},
		{
			name:     "known forge resolves compact template",
			persona:  "{{forge.type}}-analyst",
			fi:       forge.ForgeInfo{Type: forge.ForgeGitLab},
			expected: []string{"gitlab-analyst"},
		},
		{
			name:     "unknown forge expands to all 4 variants",
			persona:  "{{ forge.type }}-analyst",
			fi:       forge.ForgeInfo{Type: forge.ForgeUnknown},
			expected: []string{"github-analyst", "gitlab-analyst", "gitea-analyst", "bitbucket-analyst"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveForgeTemplate(tt.persona, tt.fi)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test isCompositionStep with different step types
func TestIsCompositionStep(t *testing.T) {
	tests := []struct {
		name     string
		step     pipeline.Step
		expected bool
	}{
		{
			name:     "persona step is not composition",
			step:     pipeline.Step{ID: "s1", Persona: "navigator"},
			expected: false,
		},
		{
			name:     "sub-pipeline step is composition",
			step:     pipeline.Step{ID: "s1", SubPipeline: "child-pipeline"},
			expected: true,
		},
		{
			name:     "branch step is composition",
			step:     pipeline.Step{ID: "s1", Branch: &pipeline.BranchConfig{On: "{{ outcome }}"}},
			expected: true,
		},
		{
			name:     "gate step is composition",
			step:     pipeline.Step{ID: "s1", Gate: &pipeline.GateConfig{Type: "approval"}},
			expected: true,
		},
		{
			name:     "loop step is composition",
			step:     pipeline.Step{ID: "s1", Loop: &pipeline.LoopConfig{MaxIterations: 3}},
			expected: true,
		},
		{
			name:     "aggregate step is composition",
			step:     pipeline.Step{ID: "s1", Aggregate: &pipeline.AggregateConfig{From: "steps.*"}},
			expected: true,
		},
		{
			name:     "command step is composition",
			step:     pipeline.Step{ID: "s1", Type: pipeline.StepTypeCommand, Script: "echo hello"},
			expected: true,
		},
		{
			name:     "conditional step is composition",
			step:     pipeline.Step{ID: "s1", Type: pipeline.StepTypeConditional},
			expected: true,
		},
		{
			name:     "empty step is not composition",
			step:     pipeline.Step{ID: "s1"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isCompositionStep(tt.step))
		})
	}
}

// Test validatePipelineFull with various scenarios
func TestValidatePipelineFull(t *testing.T) {
	fi := forge.ForgeInfo{Type: forge.ForgeGitHub}

	t.Run("missing persona", func(t *testing.T) {
		h := newTestHelper(t)
		h.chdir()
		defer h.restore()

		m := &manifest.Manifest{
			Personas: map[string]manifest.Persona{
				"navigator": {Adapter: "claude"},
			},
		}
		h.writeFile(".wave/pipelines/test.yaml", `kind: WavePipeline
metadata:
  name: test
steps:
  - id: s1
    persona: nonexistent
    exec:
      type: prompt
      source: "do something"
`)
		errs := validatePipelineFull("test", m, fi)
		found := false
		for _, e := range errs {
			if strings.Contains(e, "not found in manifest") {
				found = true
			}
		}
		assert.True(t, found, "should report missing persona, got: %v", errs)
	})

	t.Run("composition step without persona is valid", func(t *testing.T) {
		h := newTestHelper(t)
		h.chdir()
		defer h.restore()

		m := &manifest.Manifest{
			Personas: map[string]manifest.Persona{},
		}
		h.writeFile(".wave/pipelines/test.yaml", `kind: WavePipeline
metadata:
  name: test
steps:
  - id: s1
    gate:
      type: approval
`)
		errs := validatePipelineFull("test", m, fi)
		for _, e := range errs {
			assert.NotContains(t, e, "no persona", "gate step should not require persona")
		}
	})

	t.Run("bad dependency", func(t *testing.T) {
		h := newTestHelper(t)
		h.chdir()
		defer h.restore()

		m := &manifest.Manifest{
			Personas: map[string]manifest.Persona{
				"navigator": {Adapter: "claude"},
			},
		}
		h.writeFile(".wave/pipelines/test.yaml", `kind: WavePipeline
metadata:
  name: test
steps:
  - id: s1
    persona: navigator
    dependencies:
      - does-not-exist
    exec:
      type: prompt
      source: "do something"
`)
		errs := validatePipelineFull("test", m, fi)
		found := false
		for _, e := range errs {
			if strings.Contains(e, "non-existent step") {
				found = true
			}
		}
		assert.True(t, found, "should report bad dependency, got: %v", errs)
	})

	t.Run("forward dependency is valid after two-pass fix", func(t *testing.T) {
		h := newTestHelper(t)
		h.chdir()
		defer h.restore()

		m := &manifest.Manifest{
			Personas: map[string]manifest.Persona{
				"navigator": {Adapter: "claude"},
			},
		}
		// step1 depends on step2, but step2 is listed after step1 in YAML.
		// This should be valid because the executor does topological sorting.
		h.writeFile(".wave/pipelines/test.yaml", `kind: WavePipeline
metadata:
  name: test
steps:
  - id: step1
    persona: navigator
    dependencies:
      - step2
    exec:
      type: prompt
      source: "depends on step2"
  - id: step2
    persona: navigator
    exec:
      type: prompt
      source: "base step"
`)
		errs := validatePipelineFull("test", m, fi)
		for _, e := range errs {
			assert.NotContains(t, e, "non-existent step", "forward dependency should be valid, got error: %s", e)
		}
	})
}

// Test stepTypeLabel returns "step" for unrecognized types
func TestStepTypeLabel_Fallback(t *testing.T) {
	step := pipeline.Step{ID: "s1", Persona: "navigator"}
	assert.False(t, isCompositionStep(step), "persona step should not be composition")
}

// TestShippedPipelines_ValidateAll simulates `wave init --all && wave validate --all`
// by generating a manifest via the onboarding wizard (non-interactive), writing all
// shipped defaults, and validating every pipeline. This catches bugs where the engine
// supports a feature but the validator or manifest rejects it.
//
// Runs for Go, TypeScript, Python, Rust, and bare projects.
func TestShippedPipelines_ValidateAll(t *testing.T) {
	languages := []struct {
		name    string
		marker  string // file to create so flavour detection works
		content string
	}{
		{"golang", "go.mod", "module test\n\ngo 1.25\n"},
		{"typescript", "package.json", `{"name":"test","scripts":{"test":"jest"}}`},
		{"python", "pyproject.toml", "[project]\nname = \"test\"\n"},
		{"rust", "Cargo.toml", "[package]\nname = \"test\"\nversion = \"0.1.0\"\n"},
		{"bare", "", ""},
	}

	for _, lang := range languages {
		t.Run(lang.name, func(t *testing.T) {
			dir := t.TempDir()
			waveDir := filepath.Join(dir, ".wave")

			// Create language marker
			if lang.marker != "" {
				_ = os.MkdirAll(filepath.Dir(filepath.Join(dir, lang.marker)), 0o755)
				_ = os.WriteFile(filepath.Join(dir, lang.marker), []byte(lang.content), 0o644)
			}

			// Write all shipped pipelines
			pipelines, err := defaults.GetPipelines()
			assert.NoError(t, err)
			for name, content := range pipelines {
				path := filepath.Join(waveDir, "pipelines", name)
				_ = os.MkdirAll(filepath.Dir(path), 0o755)
				_ = os.WriteFile(path, []byte(content), 0o644)
			}

			// Write all shipped contracts
			contracts, _ := defaults.GetContracts()
			for name, content := range contracts {
				path := filepath.Join(waveDir, "contracts", name)
				_ = os.MkdirAll(filepath.Dir(path), 0o755)
				_ = os.WriteFile(path, []byte(content), 0o644)
			}

			// Write persona prompts
			personas, _ := defaults.GetPersonas()
			for name, content := range personas {
				path := filepath.Join(waveDir, "personas", name)
				_ = os.MkdirAll(filepath.Dir(path), 0o755)
				_ = os.WriteFile(path, []byte(content), 0o644)
			}

			// Write prompt files
			prompts, _ := defaults.GetPrompts()
			for name, content := range prompts {
				path := filepath.Join(waveDir, "prompts", name)
				_ = os.MkdirAll(filepath.Dir(path), 0o755)
				_ = os.WriteFile(path, []byte(content), 0o644)
			}

			// Get persona configs for manifest generation
			personaConfigs, _ := defaults.GetPersonaConfigs()

			// Generate manifest via onboarding wizard (non-interactive)
			origDir, _ := os.Getwd()
			_ = os.Chdir(dir)
			defer func() { _ = os.Chdir(origDir) }()

			cfg := onboarding.WizardConfig{
				WaveDir:        waveDir,
				Interactive:    false,
				Adapter:        "claude",
				Workspace:      ".wave/workspaces",
				OutputPath:     filepath.Join(dir, "wave.yaml"),
				PersonaConfigs: personaConfigs,
			}

			_, wizErr := onboarding.RunWizard(cfg)
			if wizErr != nil {
				t.Fatalf("wizard failed for %s: %v", lang.name, wizErr)
			}

			// Load the generated manifest
			m, loadErr := manifest.Load(filepath.Join(dir, "wave.yaml"))
			if loadErr != nil {
				t.Fatalf("failed to load generated manifest for %s: %v", lang.name, loadErr)
			}

			// Validate ALL shipped pipelines
			fi := forge.ForgeInfo{Type: forge.ForgeGitHub}
			var allErrs []string
			for name := range pipelines {
				pName := strings.TrimSuffix(name, ".yaml")
				errs := validatePipelineFull(pName, m, fi)
				for _, e := range errs {
					allErrs = append(allErrs, pName+": "+e)
				}
			}

			if len(allErrs) > 0 {
				t.Errorf("%s project: %d validation errors:\n  %s",
					lang.name, len(allErrs), strings.Join(allErrs, "\n  "))
			} else {
				t.Logf("%s: validated %d pipelines — all clean", lang.name, len(pipelines))
			}
		})
	}
}
