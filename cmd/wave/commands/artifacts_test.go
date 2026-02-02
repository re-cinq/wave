package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// artifactsTestEnv provides a testing environment for artifacts tests
type artifactsTestEnv struct {
	t       *testing.T
	rootDir string
	origDir string
}

// newArtifactsTestEnv creates a new test environment with a temp directory
func newArtifactsTestEnv(t *testing.T) *artifactsTestEnv {
	t.Helper()

	origDir, err := os.Getwd()
	require.NoError(t, err, "failed to get current directory")

	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err, "failed to change to temp directory")

	return &artifactsTestEnv{
		t:       t,
		rootDir: tmpDir,
		origDir: origDir,
	}
}

// cleanup restores the original working directory
func (e *artifactsTestEnv) cleanup() {
	err := os.Chdir(e.origDir)
	if err != nil {
		e.t.Errorf("failed to restore original directory: %v", err)
	}
}

// createWorkspaceStructure creates the base .wave/workspaces directory
func (e *artifactsTestEnv) createWorkspaceStructure() {
	e.t.Helper()
	err := os.MkdirAll(".wave/workspaces", 0755)
	require.NoError(e.t, err, "failed to create workspaces directory")
}

// createPipelineWorkspace creates a pipeline workspace with step directories
func (e *artifactsTestEnv) createPipelineWorkspace(pipelineName string, steps []string) string {
	e.t.Helper()

	pipelineDir := filepath.Join(".wave", "workspaces", pipelineName)
	for _, step := range steps {
		stepDir := filepath.Join(pipelineDir, step)
		err := os.MkdirAll(stepDir, 0755)
		require.NoError(e.t, err, "failed to create step directory: %s", step)
	}

	return pipelineDir
}

// createArtifact creates a test artifact file in a step directory
func (e *artifactsTestEnv) createArtifact(pipeline, step, name, content string) string {
	e.t.Helper()

	artifactPath := filepath.Join(".wave", "workspaces", pipeline, step, name)
	dir := filepath.Dir(artifactPath)
	err := os.MkdirAll(dir, 0755)
	require.NoError(e.t, err, "failed to create artifact directory")

	err = os.WriteFile(artifactPath, []byte(content), 0644)
	require.NoError(e.t, err, "failed to write artifact: %s", name)

	return artifactPath
}

// executeArtifactsCmd runs the artifacts command with given arguments and returns output/error
func executeArtifactsCmd(args ...string) (stdout, stderr string, err error) {
	cmd := NewArtifactsCmd()

	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)

	// Capture stdout since artifacts command uses fmt.Printf directly
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

// Test: List artifacts from workspace
func TestArtifactsCmd_ListFromWorkspace(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"investigate", "plan"})

	// Create test artifacts
	env.createArtifact("debug", "investigate", "analysis.md", "# Analysis\n\nThis is an analysis.")
	env.createArtifact("debug", "plan", "plan.md", "# Plan\n\nThis is a plan.")

	stdout, _, err := executeArtifactsCmd()

	require.NoError(t, err)
	assert.Contains(t, stdout, "investigate")
	assert.Contains(t, stdout, "plan")
	assert.Contains(t, stdout, "analysis.md")
	assert.Contains(t, stdout, "plan.md")
	assert.Contains(t, stdout, "markdown")
}

// Test: Filter by step ID
func TestArtifactsCmd_StepFilter(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"investigate", "plan"})

	env.createArtifact("debug", "investigate", "analysis.md", "# Analysis")
	env.createArtifact("debug", "plan", "plan.md", "# Plan")

	stdout, _, err := executeArtifactsCmd("--step", "investigate")

	require.NoError(t, err)
	assert.Contains(t, stdout, "investigate")
	assert.Contains(t, stdout, "analysis.md")
	assert.NotContains(t, stdout, "plan.md")
}

// Test: JSON format output
func TestArtifactsCmd_JSONFormat(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"investigate"})

	env.createArtifact("debug", "investigate", "analysis.md", "# Analysis content")

	stdout, _, err := executeArtifactsCmd("--format", "json")

	require.NoError(t, err)

	// Verify valid JSON
	var output ArtifactsOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err, "output should be valid JSON")

	assert.Equal(t, "debug", output.RunID)
	require.Len(t, output.Artifacts, 1)
	assert.Equal(t, "investigate", output.Artifacts[0].Step)
	assert.Equal(t, "analysis.md", output.Artifacts[0].Name)
	assert.Equal(t, "markdown", output.Artifacts[0].Type)
	assert.True(t, output.Artifacts[0].Size > 0)
	assert.True(t, output.Artifacts[0].Exists)
}

// Test: Export to new directory
func TestArtifactsCmd_ExportToNewDirectory(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"investigate", "plan"})

	env.createArtifact("debug", "investigate", "analysis.md", "# Analysis content here")
	env.createArtifact("debug", "plan", "plan.md", "# Plan content")

	exportDir := filepath.Join(env.rootDir, "exported")

	stdout, _, err := executeArtifactsCmd("--export", exportDir)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Exported")
	assert.Contains(t, stdout, "2 artifact(s)")

	// Verify exported files exist
	assert.FileExists(t, filepath.Join(exportDir, "investigate", "analysis.md"))
	assert.FileExists(t, filepath.Join(exportDir, "plan", "plan.md"))

	// Verify content was copied correctly
	content, err := os.ReadFile(filepath.Join(exportDir, "investigate", "analysis.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Analysis content here", string(content))
}

// Test: Export to existing directory
func TestArtifactsCmd_ExportToExistingDirectory(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"investigate"})

	env.createArtifact("debug", "investigate", "analysis.md", "# Analysis")

	// Create existing directory with a file
	exportDir := filepath.Join(env.rootDir, "existing-export")
	err := os.MkdirAll(exportDir, 0755)
	require.NoError(t, err)

	existingFile := filepath.Join(exportDir, "existing.txt")
	err = os.WriteFile(existingFile, []byte("existing content"), 0644)
	require.NoError(t, err)

	stdout, _, err := executeArtifactsCmd("--export", exportDir)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Exported")

	// Verify exported file exists
	assert.FileExists(t, filepath.Join(exportDir, "investigate", "analysis.md"))

	// Verify existing file still exists
	assert.FileExists(t, existingFile)
}

// Test: Handle missing artifacts (warn but don't fail)
func TestArtifactsCmd_MissingArtifactsWarning(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"investigate"})

	// Create an artifact then delete it (simulate missing file)
	artifactPath := env.createArtifact("debug", "investigate", "analysis.md", "# Analysis")
	os.Remove(artifactPath)

	// Create another valid artifact
	env.createArtifact("debug", "investigate", "valid.md", "# Valid")

	exportDir := filepath.Join(env.rootDir, "export")

	stdout, _, err := executeArtifactsCmd("--export", exportDir)

	// Should not fail
	require.NoError(t, err)
	assert.Contains(t, stdout, "Exported")

	// Valid artifact should be exported
	assert.FileExists(t, filepath.Join(exportDir, "investigate", "valid.md"))
}

// Test: Handle name collisions in export
func TestArtifactsCmd_ExportNameCollisions(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()

	// Create two step directories with same artifact name
	env.createPipelineWorkspace("debug", []string{"step1", "step2"})
	env.createArtifact("debug", "step1", "output.md", "# Step 1 output")
	env.createArtifact("debug", "step2", "output.md", "# Step 2 output")

	exportDir := filepath.Join(env.rootDir, "export")

	stdout, _, err := executeArtifactsCmd("--export", exportDir)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Exported")
	assert.Contains(t, stdout, "2 artifact(s)")

	// Both files should exist in their step subdirectories
	assert.FileExists(t, filepath.Join(exportDir, "step1", "output.md"))
	assert.FileExists(t, filepath.Join(exportDir, "step2", "output.md"))

	// Verify content is correct for each
	content1, _ := os.ReadFile(filepath.Join(exportDir, "step1", "output.md"))
	content2, _ := os.ReadFile(filepath.Join(exportDir, "step2", "output.md"))
	assert.Equal(t, "# Step 1 output", string(content1))
	assert.Equal(t, "# Step 2 output", string(content2))
}

// Test: No artifacts found
func TestArtifactsCmd_NoArtifacts(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	// Create pipeline with no artifacts
	env.createPipelineWorkspace("empty", []string{"step1"})

	stdout, _, err := executeArtifactsCmd()

	require.NoError(t, err)
	assert.Contains(t, stdout, "No artifacts found")
}

// Test: No workspaces at all
func TestArtifactsCmd_NoWorkspaces(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	// Don't create any workspace structure

	stdout, _, err := executeArtifactsCmd()

	require.NoError(t, err)
	assert.Contains(t, stdout, "No artifacts found")
}

// Test: Different artifact types
func TestArtifactsCmd_DifferentArtifactTypes(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"step1"})

	// Create different types of artifacts
	env.createArtifact("debug", "step1", "analysis.md", "# Markdown")
	env.createArtifact("debug", "step1", "config.yaml", "key: value")
	env.createArtifact("debug", "step1", "data.json", "{}")
	env.createArtifact("debug", "step1", "notes.txt", "text notes")
	env.createArtifact("debug", "step1", "debug.log", "log entries")

	stdout, _, err := executeArtifactsCmd()

	require.NoError(t, err)
	assert.Contains(t, stdout, "markdown")
	assert.Contains(t, stdout, "yaml")
	assert.Contains(t, stdout, "json")
	assert.Contains(t, stdout, "text")
	assert.Contains(t, stdout, "log")
}

// Test: JSON output validity with multiple artifacts
func TestArtifactsCmd_JSONMultipleArtifacts(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"step1", "step2"})

	env.createArtifact("debug", "step1", "file1.md", "content 1")
	env.createArtifact("debug", "step1", "file2.json", "{}")
	env.createArtifact("debug", "step2", "file3.yaml", "key: val")

	stdout, _, err := executeArtifactsCmd("--format", "json")

	require.NoError(t, err)

	var output ArtifactsOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err, "output should be valid JSON")

	assert.Len(t, output.Artifacts, 3)

	// Check all artifacts have required fields
	for _, a := range output.Artifacts {
		assert.NotEmpty(t, a.Step)
		assert.NotEmpty(t, a.Name)
		assert.NotEmpty(t, a.Type)
		assert.NotEmpty(t, a.Path)
		assert.True(t, a.Exists)
	}
}

// Test: Step filter with JSON output
func TestArtifactsCmd_StepFilterWithJSON(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"step1", "step2"})

	env.createArtifact("debug", "step1", "file1.md", "content 1")
	env.createArtifact("debug", "step2", "file2.md", "content 2")

	stdout, _, err := executeArtifactsCmd("--step", "step1", "--format", "json")

	require.NoError(t, err)

	var output ArtifactsOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err)

	// Should only have step1 artifact
	assert.Len(t, output.Artifacts, 1)
	assert.Equal(t, "step1", output.Artifacts[0].Step)
	assert.Equal(t, "file1.md", output.Artifacts[0].Name)
}

// Test: Export with step filter
func TestArtifactsCmd_ExportWithStepFilter(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"step1", "step2"})

	env.createArtifact("debug", "step1", "file1.md", "step1 content")
	env.createArtifact("debug", "step2", "file2.md", "step2 content")

	exportDir := filepath.Join(env.rootDir, "export")

	stdout, _, err := executeArtifactsCmd("--step", "step1", "--export", exportDir)

	require.NoError(t, err)
	assert.Contains(t, stdout, "1 artifact(s)")

	// Only step1 artifact should be exported
	assert.FileExists(t, filepath.Join(exportDir, "step1", "file1.md"))
	assert.NoDirExists(t, filepath.Join(exportDir, "step2"))
}

// Test: Command flags existence
func TestNewArtifactsCmdFlags(t *testing.T) {
	cmd := NewArtifactsCmd()

	assert.Equal(t, "artifacts [run-id]", cmd.Use)
	assert.Contains(t, cmd.Short, "artifact")

	flags := cmd.Flags()

	stepFlag := flags.Lookup("step")
	assert.NotNil(t, stepFlag, "step flag should exist")

	exportFlag := flags.Lookup("export")
	assert.NotNil(t, exportFlag, "export flag should exist")

	formatFlag := flags.Lookup("format")
	assert.NotNil(t, formatFlag, "format flag should exist")

	manifestFlag := flags.Lookup("manifest")
	assert.NotNil(t, manifestFlag, "manifest flag should exist")
}

// Test: Format size helper function (artifacts context)
func TestArtifactsCmd_FormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatSize(tc.bytes)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Test: Table output format
func TestArtifactsCmd_TableOutputFormat(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"investigate"})

	env.createArtifact("debug", "investigate", "analysis.md", "# Analysis content")

	stdout, _, err := executeArtifactsCmd()

	require.NoError(t, err)

	// Verify table headers
	assert.Contains(t, stdout, "STEP")
	assert.Contains(t, stdout, "ARTIFACT")
	assert.Contains(t, stdout, "TYPE")
	assert.Contains(t, stdout, "SIZE")
	assert.Contains(t, stdout, "PATH")
}

// Test: Subdirectory artifacts are found
func TestArtifactsCmd_SubdirectoryArtifacts(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"step1"})

	// Create artifact in a subdirectory
	subDir := filepath.Join(".wave", "workspaces", "debug", "step1", "output")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(subDir, "result.md"), []byte("# Result"), 0644)
	require.NoError(t, err)

	stdout, _, err := executeArtifactsCmd()

	require.NoError(t, err)
	assert.Contains(t, stdout, "result.md")
}

// Test: Non-artifact files are ignored
func TestArtifactsCmd_NonArtifactFilesIgnored(t *testing.T) {
	env := newArtifactsTestEnv(t)
	defer env.cleanup()

	env.createWorkspaceStructure()
	env.createPipelineWorkspace("debug", []string{"step1"})

	// Create non-artifact files
	stepDir := filepath.Join(".wave", "workspaces", "debug", "step1")
	os.WriteFile(filepath.Join(stepDir, "binary.exe"), []byte{0x00, 0x01}, 0644)
	os.WriteFile(filepath.Join(stepDir, "image.png"), []byte{0x89, 0x50}, 0644)
	os.WriteFile(filepath.Join(stepDir, "source.go"), []byte("package main"), 0644)

	// Create one valid artifact
	env.createArtifact("debug", "step1", "readme.md", "# Readme")

	stdout, _, err := executeArtifactsCmd("--format", "json")

	require.NoError(t, err)

	var output ArtifactsOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err)

	// Should only find the markdown file
	assert.Len(t, output.Artifacts, 1)
	assert.Equal(t, "readme.md", output.Artifacts[0].Name)
}
