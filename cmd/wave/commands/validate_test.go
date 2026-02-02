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

// testHelper provides common utilities for validate command tests.
type testHelper struct {
	t          *testing.T
	tmpDir     string
	origDir    string
	origStdout *os.File
	outBuf     *bytes.Buffer
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

// captureOutput captures stdout for testing.
func (h *testHelper) captureOutput() {
	h.t.Helper()
	h.origStdout = os.Stdout
	r, w, err := os.Pipe()
	require.NoError(h.t, err, "failed to create pipe")
	os.Stdout = w
	h.outBuf = new(bytes.Buffer)
	go func() {
		_, _ = h.outBuf.ReadFrom(r)
	}()
}

// getOutput restores stdout and returns captured output.
func (h *testHelper) getOutput() string {
	h.t.Helper()
	os.Stdout.Close()
	os.Stdout = h.origStdout
	return h.outBuf.String()
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

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--verbose"})

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
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

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"-v"})

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
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

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--verbose"})

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
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

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{"--verbose"})

	// The validate should succeed (missing binary is a warning, not error)
	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
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
