package commands

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// T084: Test helpers for list command tests

// listTestHelper provides common utilities for list command tests.
type listTestHelper struct {
	t       *testing.T
	tmpDir  string
	origDir string
}

// newListTestHelper creates a new test helper with a temporary directory.
func newListTestHelper(t *testing.T) *listTestHelper {
	t.Helper()
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err, "failed to get current directory")

	return &listTestHelper{
		t:       t,
		tmpDir:  tmpDir,
		origDir: origDir,
	}
}

// chdir changes to the temporary directory.
func (h *listTestHelper) chdir() {
	h.t.Helper()
	err := os.Chdir(h.tmpDir)
	require.NoError(h.t, err, "failed to change to temp directory")
}

// restore returns to the original directory.
func (h *listTestHelper) restore() {
	h.t.Helper()
	_ = os.Chdir(h.origDir)
}

// writeFile writes content to a file in the temp directory.
func (h *listTestHelper) writeFile(relPath, content string) {
	h.t.Helper()
	fullPath := filepath.Join(h.tmpDir, relPath)
	dir := filepath.Dir(fullPath)
	err := os.MkdirAll(dir, 0755)
	require.NoError(h.t, err, "failed to create directory: %s", dir)
	err = os.WriteFile(fullPath, []byte(content), 0644)
	require.NoError(h.t, err, "failed to write file: %s", relPath)
}

// executeListCmd runs the list command with given arguments and returns output/error.
func executeListCmd(args ...string) (stdout, stderr string, err error) {
	cmd := NewListCmd()

	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)

	// Capture stdout since list command uses fmt.Printf directly
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	return buf.String(), errBuf.String(), err
}

// sampleManifest returns a sample wave.yaml content for testing.
func sampleManifest() string {
	return `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
  description: Test project for list commands
adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
  opencode:
    binary: opencode
    mode: headless
    output_format: json
personas:
  navigator:
    adapter: claude
    description: Explores the codebase
    system_prompt_file: personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools:
        - Read
        - Glob
        - Grep
      deny: []
  craftsman:
    adapter: claude
    description: Implements code changes
    system_prompt_file: personas/craftsman.md
    temperature: 0.7
    permissions:
      allowed_tools:
        - Read
        - Write
        - Edit
        - Bash
      deny:
        - Bash(rm -rf /*)
  auditor:
    adapter: opencode
    description: Reviews code for security
    system_prompt_file: personas/auditor.md
    temperature: 0.1
    permissions:
      allowed_tools:
        - Read
        - Grep
      deny:
        - Write(*)
        - Edit(*)
runtime:
  workspace_root: .wave/workspaces
  default_timeout_minutes: 30
`
}

// samplePipeline returns a sample pipeline YAML for testing.
func samplePipeline(name, description string, stepCount int) string {
	var steps strings.Builder
	steps.WriteString("steps:\n")
	for i := 1; i <= stepCount; i++ {
		steps.WriteString("  - id: step")
		steps.WriteString(string(rune('0' + i)))
		steps.WriteString("\n    persona: navigator\n")
		steps.WriteString("    exec:\n      type: prompt\n      source: \"Task ")
		steps.WriteString(string(rune('0' + i)))
		steps.WriteString("\"\n")
	}
	return `kind: WavePipeline
metadata:
  name: ` + name + `
  description: ` + description + `
` + steps.String()
}

// T085: Test for list pipelines output format

func TestListCmd_Pipelines_TableFormat(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create wave.yaml
	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	// Create some pipelines
	h.writeFile(".wave/pipelines/feature.yaml", samplePipeline("feature", "Feature development pipeline", 3))
	h.writeFile(".wave/pipelines/hotfix.yaml", samplePipeline("hotfix", "Quick fix pipeline", 2))

	stdout, _, err := executeListCmd("pipelines")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Pipelines")
	assert.Contains(t, stdout, "feature")
	assert.Contains(t, stdout, "hotfix")
	assert.Contains(t, stdout, "Feature development pipeline")
	assert.Contains(t, stdout, "Quick fix pipeline")
}

func TestListCmd_Pipelines_ShowsStepCount(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	// Create pipeline with 3 steps
	h.writeFile(".wave/pipelines/test.yaml", samplePipeline("test", "Test pipeline", 3))

	stdout, _, err := executeListCmd("pipelines")

	require.NoError(t, err)
	assert.Contains(t, stdout, "3 steps", "output should show step count")
}

func TestListCmd_Pipelines_ShowsStepIDs(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	h.writeFile(".wave/pipelines/test.yaml", `kind: WavePipeline
metadata:
  name: test
  description: Test pipeline
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze"
  - id: implement
    persona: craftsman
    exec:
      type: prompt
      source: "Implement"
  - id: review
    persona: auditor
    exec:
      type: prompt
      source: "Review"
`)

	stdout, _, err := executeListCmd("pipelines")

	require.NoError(t, err)
	// Check that step IDs are shown in order with arrow connector
	assert.Contains(t, stdout, "analyze")
	assert.Contains(t, stdout, "implement")
	assert.Contains(t, stdout, "review")
}

func TestListCmd_Pipelines_NoPipelinesDirectory(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	// Don't create .wave/pipelines directory
	stdout, _, err := executeListCmd("pipelines")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Pipelines")
	assert.Contains(t, stdout, "(none found", "should indicate no pipelines found")
}

func TestListCmd_Pipelines_InvalidYAML(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	// Create invalid pipeline file
	h.writeFile(".wave/pipelines/broken.yaml", "{ invalid: yaml: content")

	stdout, _, err := executeListCmd("pipelines")

	require.NoError(t, err)
	assert.Contains(t, stdout, "broken")
	assert.Contains(t, stdout, "error parsing", "should indicate parsing error")
}

func TestListCmd_Pipelines_SortedAlphabetically(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	// Create pipelines in non-alphabetical order
	h.writeFile(".wave/pipelines/zebra.yaml", samplePipeline("zebra", "Z pipeline", 1))
	h.writeFile(".wave/pipelines/alpha.yaml", samplePipeline("alpha", "A pipeline", 1))
	h.writeFile(".wave/pipelines/middle.yaml", samplePipeline("middle", "M pipeline", 1))

	stdout, _, err := executeListCmd("pipelines")

	require.NoError(t, err)
	// Check that alpha appears before middle, and middle before zebra
	alphaIdx := strings.Index(stdout, "alpha")
	middleIdx := strings.Index(stdout, "middle")
	zebraIdx := strings.Index(stdout, "zebra")

	assert.True(t, alphaIdx < middleIdx, "alpha should appear before middle")
	assert.True(t, middleIdx < zebraIdx, "middle should appear before zebra")
}

// T086: Test for list personas output format

func TestListCmd_Personas_TableFormat(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	stdout, _, err := executeListCmd("personas")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Personas")
	assert.Contains(t, stdout, "navigator")
	assert.Contains(t, stdout, "craftsman")
	assert.Contains(t, stdout, "auditor")
}

func TestListCmd_Personas_ShowsAdapter(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	stdout, _, err := executeListCmd("personas")

	require.NoError(t, err)
	assert.Contains(t, stdout, "adapter: claude")
	assert.Contains(t, stdout, "adapter: opencode")
}

func TestListCmd_Personas_ShowsTemperature(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	stdout, _, err := executeListCmd("personas")

	require.NoError(t, err)
	assert.Contains(t, stdout, "temp: 0.1")
	assert.Contains(t, stdout, "temp: 0.7")
}

func TestListCmd_Personas_ShowsDescription(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	stdout, _, err := executeListCmd("personas")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Explores the codebase")
	assert.Contains(t, stdout, "Implements code changes")
	assert.Contains(t, stdout, "Reviews code for security")
}

func TestListCmd_Personas_ShowsPermissionSummary(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	stdout, _, err := executeListCmd("personas")

	require.NoError(t, err)
	// T089: Check for permission summary (allowed tools count and deny count)
	assert.True(t,
		strings.Contains(stdout, "allow:") || strings.Contains(stdout, "tools:"),
		"output should show permission summary")
}

func TestListCmd_Personas_NoPersonasDefined(t *testing.T) {
	h := newListTestHelper(t)
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
personas: {}
runtime:
  workspace_root: .wave/workspaces
`)

	stdout, _, err := executeListCmd("personas")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Personas")
	assert.Contains(t, stdout, "(none defined)")
}

func TestListCmd_Personas_SortedAlphabetically(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	stdout, _, err := executeListCmd("personas")

	require.NoError(t, err)
	// Check that personas are sorted alphabetically
	auditorIdx := strings.Index(stdout, "auditor")
	craftsmanIdx := strings.Index(stdout, "craftsman")
	navigatorIdx := strings.Index(stdout, "navigator")

	assert.True(t, auditorIdx < craftsmanIdx, "auditor should appear before craftsman")
	assert.True(t, craftsmanIdx < navigatorIdx, "craftsman should appear before navigator")
}

// T087: Test for list adapters with binary availability check

func TestListCmd_Adapters_TableFormat(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	stdout, _, err := executeListCmd("adapters")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Adapters")
	assert.Contains(t, stdout, "claude")
	assert.Contains(t, stdout, "opencode")
}

func TestListCmd_Adapters_ShowsBinary(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	stdout, _, err := executeListCmd("adapters")

	require.NoError(t, err)
	assert.Contains(t, stdout, "binary: claude")
	assert.Contains(t, stdout, "binary: opencode")
}

func TestListCmd_Adapters_ShowsMode(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	stdout, _, err := executeListCmd("adapters")

	require.NoError(t, err)
	assert.Contains(t, stdout, "mode: headless")
}

func TestListCmd_Adapters_ShowsOutputFormat(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	stdout, _, err := executeListCmd("adapters")

	require.NoError(t, err)
	assert.Contains(t, stdout, "format: json")
}

func TestListCmd_Adapters_ShowsBinaryAvailability(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Use a non-existent binary to test availability check
	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  fake-adapter:
    binary: definitely-not-a-real-binary-xyz123
    mode: headless
    output_format: json
personas: {}
runtime:
  workspace_root: .wave/workspaces
`)

	stdout, _, err := executeListCmd("adapters")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Adapters")
	assert.Contains(t, stdout, "fake-adapter")
	// Should indicate binary is not found
	assert.True(t,
		strings.Contains(stdout, "not found") || strings.Contains(stdout, "unavailable") || strings.Contains(stdout, "[X]"),
		"output should indicate binary is not available")
}

func TestListCmd_Adapters_ShowsBinaryAvailable(t *testing.T) {
	// Skip if no common binary is available
	_, err := exec.LookPath("ls")
	if err != nil {
		t.Skip("ls command not available, skipping test")
	}

	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Use a binary that definitely exists (ls on Unix)
	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  ls-adapter:
    binary: ls
    mode: headless
    output_format: json
personas: {}
runtime:
  workspace_root: .wave/workspaces
`)

	stdout, _, err := executeListCmd("adapters")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Adapters")
	assert.Contains(t, stdout, "ls-adapter")
}

func TestListCmd_Adapters_NoAdaptersDefined(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters: {}
personas: {}
runtime:
  workspace_root: .wave/workspaces
`)

	stdout, _, err := executeListCmd("adapters")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Adapters")
	assert.Contains(t, stdout, "(none defined)")
}

// Test contracts listing

func TestListCmd_Contracts_TableFormat(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile(".wave/contracts/navigation.json", `{"type": "object", "properties": {"files": {"type": "array"}}}`)
	h.writeFile(".wave/contracts/output.json", `{"type": "object", "properties": {"result": {"type": "string"}}}`)

	stdout, _, err := executeListCmd("contracts")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Contracts")
	assert.Contains(t, stdout, "navigation")
	assert.Contains(t, stdout, "output")
	assert.Contains(t, stdout, "json-schema")
}

func TestListCmd_Contracts_ShowsUsage(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile(".wave/contracts/navigation.json", `{"type": "object"}`)

	// Create pipeline that uses the contract
	pipelineWithContract := `apiVersion: v1
kind: WavePipeline
metadata:
  name: test-pipeline
  description: Test pipeline with contract
steps:
  - id: navigate
    persona: navigator
    contract:
      schema_path: .wave/contracts/navigation.json
`
	h.writeFile(".wave/pipelines/test-pipeline.yaml", pipelineWithContract)

	stdout, _, err := executeListCmd("contracts")

	require.NoError(t, err)
	assert.Contains(t, stdout, "navigation")
	assert.Contains(t, stdout, "used by")
	assert.Contains(t, stdout, "test-pipeline")
	assert.Contains(t, stdout, "navigate")
	assert.Contains(t, stdout, "navigator")
}

func TestListCmd_Contracts_ShowsUnused(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile(".wave/contracts/unused.json", `{"type": "object"}`)

	stdout, _, err := executeListCmd("contracts")

	require.NoError(t, err)
	assert.Contains(t, stdout, "unused")
	assert.Contains(t, stdout, "(unused)")
}

func TestListCmd_Contracts_NoContractsDirectory(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())

	stdout, _, err := executeListCmd("contracts")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Contracts")
	assert.Contains(t, stdout, "(none found")
}

// Test list all (no filter)

func TestListCmd_All_ShowsEverything(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")
	h.writeFile(".wave/pipelines/test.yaml", samplePipeline("test", "Test pipeline", 2))
	h.writeFile(".wave/contracts/test.json", `{"type": "object"}`)

	stdout, _, err := executeListCmd()

	require.NoError(t, err)
	assert.Contains(t, stdout, "Adapters")
	assert.Contains(t, stdout, "Pipelines")
	assert.Contains(t, stdout, "Personas")
	assert.Contains(t, stdout, "Contracts")
}

// Test with missing manifest

func TestListCmd_MissingManifest(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Don't create wave.yaml
	stdout, _, err := executeListCmd("personas")

	require.NoError(t, err)
	assert.Contains(t, stdout, "manifest not found", "should indicate manifest not found")
}

func TestListCmd_CustomManifestPath(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile("config/custom-wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	stdout, _, err := executeListCmd("--manifest", "config/custom-wave.yaml", "personas")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Personas")
	assert.Contains(t, stdout, "navigator")
}

// Test with testdata fixtures

func TestListCmd_Pipelines_WithTestdataFixtures(t *testing.T) {
	// Get the path to testdata
	testdataPath := filepath.Join("testdata", "pipelines")

	if _, err := os.Stat(testdataPath); os.IsNotExist(err) {
		t.Skip("testdata/pipelines not found")
	}

	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Copy testdata pipelines to .wave/pipelines
	h.writeFile("wave.yaml", sampleManifest())
	h.writeFile("personas/navigator.md", "# Navigator")
	h.writeFile("personas/craftsman.md", "# Craftsman")
	h.writeFile("personas/auditor.md", "# Auditor")

	// Read and copy simple.yaml
	simpleContent, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "pipelines", "simple.yaml"))
	if err == nil {
		h.writeFile(".wave/pipelines/simple.yaml", string(simpleContent))
	}

	stdout, _, err := executeListCmd("pipelines")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Pipelines")
}

func TestListCmd_Personas_WithTestdataFixtures(t *testing.T) {
	// Get the path to testdata
	testdataPath := filepath.Join("testdata", "valid", "wave.yaml")

	if _, err := os.Stat(testdataPath); os.IsNotExist(err) {
		t.Skip("testdata/valid/wave.yaml not found")
	}

	stdout, _, err := executeListCmd("--manifest", testdataPath, "personas")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Personas")
	assert.Contains(t, stdout, "navigator")
	assert.Contains(t, stdout, "craftsman")
}

// Table-driven tests

func TestListCmd_FilterOptions(t *testing.T) {
	tests := []struct {
		name         string
		filter       string
		wantContains []string
	}{
		{
			name:         "pipelines filter",
			filter:       "pipelines",
			wantContains: []string{"Pipelines"},
		},
		{
			name:         "personas filter",
			filter:       "personas",
			wantContains: []string{"Personas"},
		},
		{
			name:         "adapters filter",
			filter:       "adapters",
			wantContains: []string{"Adapters"},
		},
		{
			name:         "contracts filter",
			filter:       "contracts",
			wantContains: []string{"Contracts"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newListTestHelper(t)
			h.chdir()
			defer h.restore()

			h.writeFile("wave.yaml", sampleManifest())
			h.writeFile("personas/navigator.md", "# Navigator")
			h.writeFile("personas/craftsman.md", "# Craftsman")
			h.writeFile("personas/auditor.md", "# Auditor")
			h.writeFile(".wave/pipelines/test.yaml", samplePipeline("test", "Test", 1))
			h.writeFile(".wave/contracts/test.json", `{"type": "object"}`)

			stdout, _, err := executeListCmd(tc.filter)

			require.NoError(t, err)
			for _, want := range tc.wantContains {
				assert.Contains(t, stdout, want)
			}
		})
	}
}

func TestListCmd_PersonaPermissionVariants(t *testing.T) {
	tests := []struct {
		name          string
		personaConfig string
		wantContains  string
	}{
		{
			name: "read-only permissions",
			personaConfig: `
  readonly:
    adapter: claude
    description: Read-only persona
    system_prompt_file: personas/readonly.md
    temperature: 0.1
    permissions:
      allowed_tools:
        - Read
        - Glob
        - Grep
      deny:
        - Write(*)
        - Edit(*)
        - Bash(*)
`,
			wantContains: "readonly",
		},
		{
			name: "full permissions",
			personaConfig: `
  admin:
    adapter: claude
    description: Full access persona
    system_prompt_file: personas/admin.md
    temperature: 0.5
    permissions:
      allowed_tools:
        - Read
        - Write
        - Edit
        - Bash
        - Glob
        - Grep
      deny: []
`,
			wantContains: "admin",
		},
		{
			name: "no permissions specified",
			personaConfig: `
  basic:
    adapter: claude
    description: Basic persona
    system_prompt_file: personas/basic.md
    temperature: 0.3
`,
			wantContains: "basic",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newListTestHelper(t)
			h.chdir()
			defer h.restore()

			manifest := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:` + tc.personaConfig + `
runtime:
  workspace_root: .wave/workspaces
`
			h.writeFile("wave.yaml", manifest)
			h.writeFile("personas/readonly.md", "# Read-only")
			h.writeFile("personas/admin.md", "# Admin")
			h.writeFile("personas/basic.md", "# Basic")

			stdout, _, err := executeListCmd("personas")

			require.NoError(t, err)
			assert.Contains(t, stdout, tc.wantContains)
		})
	}
}

// ====================================================================
// Task 10: Tests for list runs subcommand
// ====================================================================

// executeListRunsCmd runs the list runs command with given arguments
func executeListRunsCmd(args ...string) (stdout, stderr string, err error) {
	// Prepend "runs" to args
	fullArgs := append([]string{"runs"}, args...)
	return executeListCmd(fullArgs...)
}

// Test list runs command exists
func TestListRunsCmd_Exists(t *testing.T) {
	listCmd := NewListCmd()

	// "runs" should be a valid argument
	assert.Contains(t, listCmd.ValidArgs, "runs", "runs should be a valid argument")
	assert.Contains(t, listCmd.Long, "runs", "help should mention runs")
}

// Test list runs flags exist
func TestListRunsCmd_FlagsExist(t *testing.T) {
	listCmd := NewListCmd()
	flags := listCmd.Flags()

	limitFlag := flags.Lookup("limit")
	assert.NotNil(t, limitFlag, "limit flag should exist")
	assert.Equal(t, "10", limitFlag.DefValue, "default limit should be 10")

	pipelineFlag := flags.Lookup("run-pipeline")
	assert.NotNil(t, pipelineFlag, "run-pipeline flag should exist")

	statusFlag := flags.Lookup("run-status")
	assert.NotNil(t, statusFlag, "run-status flag should exist")

	formatFlag := flags.Lookup("format")
	assert.NotNil(t, formatFlag, "format flag should exist")
}

// Test list runs with no data
func TestListRunsCmd_NoData(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	stdout, _, err := executeListRunsCmd()

	require.NoError(t, err)
	assert.Contains(t, stdout, "Recent Pipeline Runs")
	assert.Contains(t, stdout, "no runs found")
}

// Test list runs with workspace fallback
func TestListRunsCmd_WorkspaceFallback(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create some workspaces (but no state database)
	h.writeFile(".wave/workspaces/pipeline-1/marker.txt", "test")
	h.writeFile(".wave/workspaces/pipeline-2/marker.txt", "test")

	stdout, _, err := executeListRunsCmd()

	require.NoError(t, err)
	assert.Contains(t, stdout, "Recent Pipeline Runs")
	// Should show workspaces as runs
	assert.True(t,
		strings.Contains(stdout, "pipeline-1") || strings.Contains(stdout, "pipeline-2"),
		"should show workspace-based runs")
}

// Test list runs JSON output
func TestListRunsCmd_JSONFormat(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create a workspace for the fallback
	h.writeFile(".wave/workspaces/test-pipeline/marker.txt", "test")

	stdout, _, err := executeListRunsCmd("--format", "json")

	require.NoError(t, err)

	// Should be valid JSON
	var output ListOutput
	err = json.Unmarshal([]byte(stdout), &output)
	assert.NoError(t, err, "output should be valid JSON")
	assert.NotNil(t, output.Runs, "runs should be in output")
}

// Test list runs with --limit flag
func TestListRunsCmd_Limit(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create several workspaces
	for i := 1; i <= 5; i++ {
		h.writeFile(fmt.Sprintf(".wave/workspaces/pipeline-%d/marker.txt", i), "test")
	}

	stdout, _, err := executeListRunsCmd("--limit", "2")

	require.NoError(t, err)

	// Count the number of data rows (excluding header)
	lines := strings.Split(stdout, "\n")
	dataLines := 0
	for _, line := range lines {
		if strings.Contains(line, "pipeline-") {
			dataLines++
		}
	}
	assert.LessOrEqual(t, dataLines, 2, "should respect limit")
}

// Test list runs with --run-pipeline filter
func TestListRunsCmd_PipelineFilter(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create workspaces
	h.writeFile(".wave/workspaces/target-pipeline/marker.txt", "test")
	h.writeFile(".wave/workspaces/other-pipeline/marker.txt", "test")

	stdout, _, err := executeListRunsCmd("--run-pipeline", "target-pipeline")

	require.NoError(t, err)
	assert.Contains(t, stdout, "target-pipeline")
	assert.NotContains(t, stdout, "other-pipeline")
}

// Test list runs table format header
func TestListRunsCmd_TableHeader(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create a workspace
	h.writeFile(".wave/workspaces/test-run/marker.txt", "test")

	stdout, _, err := executeListRunsCmd()

	require.NoError(t, err)
	assert.Contains(t, stdout, "RUN_ID")
	assert.Contains(t, stdout, "PIPELINE")
	assert.Contains(t, stdout, "STATUS")
	assert.Contains(t, stdout, "STARTED")
	assert.Contains(t, stdout, "DURATION")
}

// Test formatDuration function
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{500 * time.Millisecond, "500ms"},
		{5 * time.Second, "5.0s"},
		{90 * time.Second, "1m30s"},
		{65 * time.Minute, "1h5m"},
		{2*time.Hour + 30*time.Minute, "2h30m"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatDuration(tc.duration)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Test list runs via main list command with "runs" argument
func TestListCmd_RunsFilter(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	stdout, _, err := executeListCmd("runs")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Recent Pipeline Runs")
}

// Test list runs with database (if available)
func TestListRunsCmd_WithDatabase(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create a minimal state database with pipeline_run table
	dbDir := ".wave"
	err := os.MkdirAll(dbDir, 0755)
	require.NoError(t, err)

	dbPath := filepath.Join(dbDir, "state.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Create table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pipeline_run (
			run_id TEXT PRIMARY KEY,
			pipeline_name TEXT NOT NULL,
			status TEXT NOT NULL,
			started_at INTEGER NOT NULL,
			completed_at INTEGER
		)
	`)
	require.NoError(t, err)

	// Insert test data
	now := time.Now().Unix()
	_, err = db.Exec(`
		INSERT INTO pipeline_run (run_id, pipeline_name, status, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?)
	`, "run-123", "test-pipeline", "completed", now-3600, now)
	require.NoError(t, err)

	stdout, _, err := executeListRunsCmd()

	require.NoError(t, err)
	assert.Contains(t, stdout, "run-123")
	assert.Contains(t, stdout, "test-pipeline")
	assert.Contains(t, stdout, "completed")
}

// Test list runs with status filter on database
func TestListRunsCmd_StatusFilter(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create database with multiple runs
	dbDir := ".wave"
	err := os.MkdirAll(dbDir, 0755)
	require.NoError(t, err)

	dbPath := filepath.Join(dbDir, "state.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pipeline_run (
			run_id TEXT PRIMARY KEY,
			pipeline_name TEXT NOT NULL,
			status TEXT NOT NULL,
			started_at INTEGER NOT NULL,
			completed_at INTEGER
		)
	`)
	require.NoError(t, err)

	now := time.Now().Unix()
	_, err = db.Exec(`INSERT INTO pipeline_run VALUES (?, ?, ?, ?, ?)`,
		"run-1", "pipeline-a", "completed", now-3600, now)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO pipeline_run VALUES (?, ?, ?, ?, ?)`,
		"run-2", "pipeline-b", "failed", now-7200, now-3600)
	require.NoError(t, err)

	// Filter by status
	stdout, _, err := executeListRunsCmd("--run-status", "completed")

	require.NoError(t, err)
	assert.Contains(t, stdout, "run-1")
	assert.NotContains(t, stdout, "run-2")
}

// Test JSON output structure
func TestListRunsCmd_JSONStructure(t *testing.T) {
	h := newListTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create database with a run
	dbDir := ".wave"
	err := os.MkdirAll(dbDir, 0755)
	require.NoError(t, err)

	dbPath := filepath.Join(dbDir, "state.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pipeline_run (
			run_id TEXT PRIMARY KEY,
			pipeline_name TEXT NOT NULL,
			status TEXT NOT NULL,
			started_at INTEGER NOT NULL,
			completed_at INTEGER
		)
	`)
	require.NoError(t, err)

	now := time.Now().Unix()
	_, err = db.Exec(`INSERT INTO pipeline_run VALUES (?, ?, ?, ?, ?)`,
		"test-run", "test-pipeline", "completed", now-60, now)
	require.NoError(t, err)

	stdout, _, err := executeListRunsCmd("--format", "json")

	require.NoError(t, err)

	var output ListOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err)

	require.Len(t, output.Runs, 1)
	assert.Equal(t, "test-run", output.Runs[0].RunID)
	assert.Equal(t, "test-pipeline", output.Runs[0].Pipeline)
	assert.Equal(t, "completed", output.Runs[0].Status)
	assert.NotEmpty(t, output.Runs[0].StartedAt)
}
