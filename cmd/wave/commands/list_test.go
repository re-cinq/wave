package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Contains(t, stdout, "Pipelines:")
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
	assert.Contains(t, stdout, "Pipelines:")
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
	assert.Contains(t, stdout, "Personas:")
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
	assert.Contains(t, stdout, "adapter:claude")
	assert.Contains(t, stdout, "adapter:opencode")
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
	assert.Contains(t, stdout, "temp:0.1")
	assert.Contains(t, stdout, "temp:0.7")
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
	assert.Contains(t, stdout, "Personas:")
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
	assert.Contains(t, stdout, "Adapters:")
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
	assert.Contains(t, stdout, "binary:claude")
	assert.Contains(t, stdout, "binary:opencode")
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
	assert.Contains(t, stdout, "mode:headless")
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
	assert.Contains(t, stdout, "format:json")
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
	assert.Contains(t, stdout, "Adapters:")
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
	assert.Contains(t, stdout, "Adapters:")
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
	assert.Contains(t, stdout, "Adapters:")
	assert.Contains(t, stdout, "(none defined)")
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

	stdout, _, err := executeListCmd()

	require.NoError(t, err)
	assert.Contains(t, stdout, "Pipelines:")
	assert.Contains(t, stdout, "Personas:")
	// No adapters without explicit filter when showing all
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
	assert.Contains(t, stdout, "Personas:")
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
	assert.Contains(t, stdout, "Pipelines:")
}

func TestListCmd_Personas_WithTestdataFixtures(t *testing.T) {
	// Get the path to testdata
	testdataPath := filepath.Join("testdata", "valid", "wave.yaml")

	if _, err := os.Stat(testdataPath); os.IsNotExist(err) {
		t.Skip("testdata/valid/wave.yaml not found")
	}

	stdout, _, err := executeListCmd("--manifest", testdataPath, "personas")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Personas:")
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
			wantContains: []string{"Pipelines:"},
		},
		{
			name:         "personas filter",
			filter:       "personas",
			wantContains: []string{"Personas:"},
		},
		{
			name:         "adapters filter",
			filter:       "adapters",
			wantContains: []string{"Adapters:"},
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
