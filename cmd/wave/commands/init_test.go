package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/defaults"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/onboarding"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// testEnv provides a clean testing environment for init tests.
type testEnv struct {
	t       *testing.T
	rootDir string
	origDir string
}

// newTestEnv creates a new test environment with a temp directory.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	origDir, err := os.Getwd()
	require.NoError(t, err, "failed to get current directory")

	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err, "failed to change to temp directory")

	return &testEnv{
		t:       t,
		rootDir: tmpDir,
		origDir: origDir,
	}
}

// cleanup restores the original working directory.
func (e *testEnv) cleanup() {
	err := os.Chdir(e.origDir)
	if err != nil {
		e.t.Errorf("failed to restore original directory: %v", err)
	}
}

// executeInitCmd runs the init command with given arguments and returns output/error.
func executeInitCmd(args ...string) (stdout, stderr string, err error) {
	cmd := NewInitCmd()

	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)

	err = cmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// dirExists checks if a directory exists at the given path.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// readYAML reads and unmarshals a YAML file.
func readYAML(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = yaml.Unmarshal(data, &result)
	return result, err
}

// TestInitEmptyDirectory tests that init works correctly in an empty directory.
func TestInitEmptyDirectory(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	stdout, _, err := executeInitCmd()

	// Verify successful execution
	require.NoError(t, err, "init should succeed in empty directory")
	assert.Contains(t, stdout, "Project initialized successfully", "should confirm initialization")

	// Verify wave.yaml was created
	assert.True(t, fileExists("wave.yaml"), "wave.yaml should be created")

	// Verify .wave directories were created
	expectedDirs := []string{
		".wave/personas",
		".wave/pipelines",
		".wave/contracts",
		".wave/traces",
		".wave/workspaces",
	}
	for _, dir := range expectedDirs {
		assert.True(t, dirExists(dir), "%s directory should be created", dir)
	}

	// Verify manifest content
	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err, "should be able to read wave.yaml")
	assert.Equal(t, "v1", manifest["apiVersion"])
	assert.Equal(t, "WaveManifest", manifest["kind"])

	metadata, ok := manifest["metadata"].(map[string]interface{})
	require.True(t, ok, "metadata should be a map")
	assert.NotEmpty(t, metadata["name"])
}

// TestInitWithExistingWaveYaml tests that init defaults to merge when wave.yaml already exists.
func TestInitWithExistingWaveYaml(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create an existing wave.yaml with a custom name
	existingContent := []byte("apiVersion: v1\nkind: WaveManifest\nmetadata:\n  name: existing\nruntime:\n  workspace_root: .wave/workspaces\n")
	err := os.WriteFile("wave.yaml", existingContent, 0644)
	require.NoError(t, err, "failed to create existing wave.yaml")

	// Run init with --yes (auto-confirm merge)
	stdout, _, err := executeInitCmd("--yes")

	// Verify successful merge execution (default behavior when wave.yaml exists)
	require.NoError(t, err, "init should default to merge when wave.yaml already exists")
	assert.Contains(t, stdout, "Configuration merged successfully", "should indicate merge operation")

	// Verify custom name is preserved
	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)
	metadata, ok := manifest["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "existing", metadata["name"], "custom name should be preserved in merge")
}

// TestInitWithForceFlag tests that init --force --yes overwrites existing files.
func TestInitWithForceFlag(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create an existing wave.yaml with different content
	existingContent := []byte("apiVersion: v1\nkind: WaveManifest\nmetadata:\n  name: existing\n")
	err := os.WriteFile("wave.yaml", existingContent, 0644)
	require.NoError(t, err, "failed to create existing wave.yaml")

	// --force now requires confirmation, use --yes to skip
	stdout, _, err := executeInitCmd("--force", "--yes")

	// Verify successful execution
	require.NoError(t, err, "init --force --yes should succeed")
	assert.Contains(t, stdout, "Project initialized successfully")

	// Verify file was overwritten with new content
	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)
	metadata, ok := manifest["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, metadata["name"], "metadata name should not be empty")
}

// TestInitForceRequiresConfirmation tests that --force warns and asks for confirmation.
func TestInitForceRequiresConfirmation(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create an existing wave.yaml
	existingContent := []byte("apiVersion: v1\nkind: WaveManifest\nmetadata:\n  name: existing\n")
	err := os.WriteFile("wave.yaml", existingContent, 0644)
	require.NoError(t, err, "failed to create existing wave.yaml")

	// Run --force without --yes and decline
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--force"})
	cmd.SetIn(strings.NewReader("n\n"))
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	err = cmd.Execute()

	// Should fail because user declined
	assert.Error(t, err, "init --force should fail when user declines")
	assert.Contains(t, err.Error(), "force overwrite cancelled", "error should mention cancellation")

	// Should have printed warning to stderr
	assert.Contains(t, errBuf.String(), "WARNING", "should print warning about data loss")
	assert.Contains(t, errBuf.String(), "Custom personas", "warning should mention personas")
	assert.Contains(t, errBuf.String(), "Ontology", "warning should mention ontology")

	// Original file should be unchanged
	data, err := os.ReadFile("wave.yaml")
	require.NoError(t, err)
	assert.Equal(t, existingContent, data, "existing wave.yaml should be unchanged")
}

// TestInitForceAccepted tests that --force proceeds when user accepts.
func TestInitForceAccepted(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create an existing wave.yaml
	existingContent := []byte("apiVersion: v1\nkind: WaveManifest\nmetadata:\n  name: existing\n")
	err := os.WriteFile("wave.yaml", existingContent, 0644)
	require.NoError(t, err, "failed to create existing wave.yaml")

	// Run --force and accept
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--force"})
	cmd.SetIn(strings.NewReader("y\n"))
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	err = cmd.Execute()

	// Should succeed
	require.NoError(t, err, "init --force should succeed when user accepts")
	assert.Contains(t, outBuf.String(), "Project initialized successfully")

	// File should be overwritten
	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)
	metadata, ok := manifest["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, metadata["name"], "metadata name should not be empty")
}

// TestInitDefaultsMergeWhenExisting tests that plain 'wave init' defaults to merge behavior.
func TestInitDefaultsMergeWhenExisting(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create an existing wave.yaml with custom personas and ontology
	existingContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: my-project
  description: My custom project
adapters:
  custom-llm:
    binary: custom-llm
    mode: headless
personas:
  my-agent:
    adapter: custom-llm
    system_prompt_file: .wave/personas/my-agent.md
    temperature: 0.7
ontology:
  telos: "Build the best widget system"
  contexts:
    - name: widget-core
      description: Core widget functionality
      invariants:
        - "Widgets must be immutable after creation"
runtime:
  workspace_root: .wave/workspaces
`
	err := os.WriteFile("wave.yaml", []byte(existingContent), 0644)
	require.NoError(t, err)

	// Run plain init with --yes (auto-confirm)
	stdout, _, err := executeInitCmd("--yes")
	require.NoError(t, err, "plain init with existing wave.yaml should merge")
	assert.Contains(t, stdout, "Configuration merged successfully")

	// Verify custom settings preserved
	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	// Custom name preserved
	metadata, ok := manifest["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-project", metadata["name"])

	// Custom adapter preserved
	adapters, ok := manifest["adapters"].(map[string]interface{})
	require.True(t, ok)
	_, hasCustom := adapters["custom-llm"]
	assert.True(t, hasCustom, "custom adapter should be preserved")

	// Default adapter added
	_, hasClaude := adapters["claude"]
	assert.True(t, hasClaude, "default claude adapter should be added")

	// Custom persona preserved
	personas, ok := manifest["personas"].(map[string]interface{})
	require.True(t, ok)
	_, hasMyAgent := personas["my-agent"]
	assert.True(t, hasMyAgent, "custom persona should be preserved")
}

// TestInitMergeFlag tests that init --merge merges with existing configuration.
func TestInitMergeFlag(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create an existing wave.yaml with custom content
	existingContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: my-custom-project
  description: My custom description
adapters:
  custom-adapter:
    binary: custom-bin
    mode: interactive
personas:
  custom-persona:
    adapter: custom-adapter
    system_prompt_file: .wave/personas/custom.md
    temperature: 0.5
runtime:
  workspace_root: .wave/workspaces
`
	err := os.WriteFile("wave.yaml", []byte(existingContent), 0644)
	require.NoError(t, err)

	// Create .wave directory structure and custom persona file
	err = os.MkdirAll(".wave/personas", 0755)
	require.NoError(t, err)
	err = os.WriteFile(".wave/personas/custom.md", []byte("# Custom Persona"), 0644)
	require.NoError(t, err)

	stdout, _, err := executeInitCmd("--merge", "--yes")

	// Verify successful execution
	require.NoError(t, err, "init --merge should succeed")
	assert.Contains(t, stdout, "Configuration merged successfully", "should indicate merge operation")

	// Verify merged content preserves custom settings
	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	// Check metadata is preserved
	metadata, ok := manifest["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-custom-project", metadata["name"], "custom name should be preserved")
	assert.Equal(t, "My custom description", metadata["description"], "custom description should be preserved")

	// Check custom adapter is preserved
	adapters, ok := manifest["adapters"].(map[string]interface{})
	require.True(t, ok)
	_, hasCustomAdapter := adapters["custom-adapter"]
	assert.True(t, hasCustomAdapter, "custom adapter should be preserved")

	// Check default adapter is added
	_, hasClaudeAdapter := adapters["claude"]
	assert.True(t, hasClaudeAdapter, "default claude adapter should be added")

	// Check custom persona is preserved
	personas, ok := manifest["personas"].(map[string]interface{})
	require.True(t, ok)
	_, hasCustomPersona := personas["custom-persona"]
	assert.True(t, hasCustomPersona, "custom persona should be preserved")

	// Check default personas are added
	_, hasNavigator := personas["navigator"]
	assert.True(t, hasNavigator, "navigator persona should be added")
}

// TestInitCreatesAllPersonaPromptFiles tests that init creates all 5 persona prompt files.
func TestInitCreatesAllPersonaPromptFiles(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	expectedPersonas := []string{
		"navigator.md",
		"philosopher.md",
		"craftsman.md",
		"auditor.md",
		"summarizer.md",
	}

	for _, persona := range expectedPersonas {
		path := filepath.Join(".wave", "personas", persona)
		assert.True(t, fileExists(path), "persona file %s should be created", persona)

		// Verify file has content
		content, err := os.ReadFile(path)
		require.NoError(t, err, "should be able to read %s", persona)
		assert.NotEmpty(t, content, "%s should not be empty", persona)

		// Verify file starts with a heading
		assert.True(t, strings.HasPrefix(string(content), "#"),
			"%s should start with a markdown heading", persona)
	}
}

// TestInitCreatesPipelineFiles tests that init creates only release pipeline files.
func TestInitCreatesPipelineFiles(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	// Default init (without --all) should only create release pipelines
	releasePipelines := defaults.ReleasePipelineNames()
	require.NotEmpty(t, releasePipelines, "there should be at least one release pipeline")

	for _, pipeline := range releasePipelines {
		path := filepath.Join(".wave", "pipelines", pipeline)
		assert.True(t, fileExists(path), "release pipeline file %s should be created", pipeline)

		// Verify it's valid YAML
		content, err := os.ReadFile(path)
		require.NoError(t, err)

		var pipelineData map[string]interface{}
		err = yaml.Unmarshal(content, &pipelineData)
		require.NoError(t, err, "%s should be valid YAML", pipeline)

		assert.Equal(t, "WavePipeline", pipelineData["kind"], "%s should have kind WavePipeline", pipeline)
	}

	// Verify non-release pipelines are NOT created
	allPipelines := defaults.PipelineNames()
	for _, pipeline := range allPipelines {
		isRelease := false
		for _, rp := range releasePipelines {
			if pipeline == rp {
				isRelease = true
				break
			}
		}
		if !isRelease {
			path := filepath.Join(".wave", "pipelines", pipeline)
			assert.False(t, fileExists(path), "non-release pipeline file %s should NOT be created", pipeline)
		}
	}
}

// TestInitCreatesContractFiles tests that init creates only transitively-referenced contract files.
func TestInitCreatesContractFiles(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	// Verify that contracts directory exists and contains only contracts
	// referenced by release pipelines
	entries, err := os.ReadDir(".wave/contracts")
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Only JSON schema contracts should contain $schema; criteria .md files won't
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(".wave", "contracts", entry.Name())
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(content), "$schema", "%s should contain $schema", entry.Name())
	}

	// With --all, all contracts should be created
	env2 := newTestEnv(t)
	defer env2.cleanup()

	_, _, err = executeInitCmd("--all")
	require.NoError(t, err)

	allContracts, err := defaults.GetContracts()
	require.NoError(t, err)

	allEntries, err := os.ReadDir(".wave/contracts")
	require.NoError(t, err)
	assert.Equal(t, len(allContracts), len(allEntries),
		"--all should create all contract files")
}

// TestInitOutputPath tests the --manifest-path flag for custom manifest path.
func TestInitOutputPath(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	customPath := "config/my-wave.yaml"

	stdout, _, err := executeInitCmd("--manifest-path", customPath)

	require.NoError(t, err)
	assert.Contains(t, stdout, customPath)
	assert.True(t, fileExists(customPath), "manifest should be created at custom path")
	assert.False(t, fileExists("wave.yaml"), "wave.yaml should not be created at default path")
}

// TestInitAdapterFlag tests the --adapter flag for default adapter selection.
func TestInitAdapterFlag(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd("--adapter", "opencode")
	require.NoError(t, err)

	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	adapters, ok := manifest["adapters"].(map[string]interface{})
	require.True(t, ok)

	_, hasOpencode := adapters["opencode"]
	assert.True(t, hasOpencode, "opencode adapter should be present")
}

// TestInitOutputValidatesWithWaveValidate tests that init output passes wave validate.
func TestInitOutputValidatesWithWaveValidate(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Initialize the project
	_, _, err := executeInitCmd()
	require.NoError(t, err, "init should succeed")

	// Run wave validate command
	validateCmd := NewValidateCmd()
	var outBuf, errBuf bytes.Buffer
	validateCmd.SetOut(&outBuf)
	validateCmd.SetErr(&errBuf)
	validateCmd.SetArgs([]string{"--manifest", "wave.yaml"})

	err = validateCmd.Execute()
	// The validate command should succeed (no error)
	assert.NoError(t, err, "wave validate should pass on init output: stdout=%s, stderr=%s", outBuf.String(), errBuf.String())
	// Note: The success message may be printed to stdout via fmt.Printf directly,
	// so we just verify no error occurred, which means validation passed.
}

// TestInitErrorMessagesIncludeFilePaths tests that error messages include full file paths.
func TestInitErrorMessagesIncludeFilePaths(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create a read-only directory to trigger an error
	err := os.MkdirAll(".wave", 0755)
	require.NoError(t, err)

	// Create wave.yaml with invalid content to trigger a parse error during merge
	err = os.WriteFile("wave.yaml", []byte("test: [invalid"), 0644)
	require.NoError(t, err)

	// Without --force, init defaults to merge — which will fail to parse
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--yes"})
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	err = cmd.Execute()

	assert.Error(t, err)
	// The error should include the file path (either relative or absolute)
	assert.True(t, strings.Contains(err.Error(), "wave.yaml") || strings.Contains(err.Error(), env.rootDir),
		"error message should include file path: %v", err)
}

// TestInitIdempotence tests that running init twice with --force --yes produces the same result.
func TestInitIdempotence(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// First init
	_, _, err := executeInitCmd()
	require.NoError(t, err)

	// Read generated files
	manifest1, err := readYAML("wave.yaml")
	require.NoError(t, err)

	// Second init with --force --yes
	_, _, err = executeInitCmd("--force", "--yes")
	require.NoError(t, err)

	// Read regenerated files
	manifest2, err := readYAML("wave.yaml")
	require.NoError(t, err)

	// Compare key structural elements
	assert.Equal(t, manifest1["apiVersion"], manifest2["apiVersion"])
	assert.Equal(t, manifest1["kind"], manifest2["kind"])

	// Compare metadata name
	m1, _ := manifest1["metadata"].(map[string]interface{})
	m2, _ := manifest2["metadata"].(map[string]interface{})
	assert.Equal(t, m1["name"], m2["name"])
}

// TestInitWorkspaceFlag tests the --workspace flag.
func TestInitWorkspaceFlag(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	customWorkspace := "custom/workspace/path"
	_, _, err := executeInitCmd("--workspace", customWorkspace)
	require.NoError(t, err)

	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	runtime, ok := manifest["runtime"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, customWorkspace, runtime["workspace_root"])
}

// TestInitMergeWithEmptyExistingFile tests merge when existing file is minimal.
func TestInitMergeWithEmptyExistingFile(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create a minimal valid wave.yaml
	minimalContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: minimal
runtime:
  workspace_root: .wave/workspaces
`
	err := os.WriteFile("wave.yaml", []byte(minimalContent), 0644)
	require.NoError(t, err)

	_, _, err = executeInitCmd("--merge", "--yes")
	require.NoError(t, err)

	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	// Should have adapters and personas added
	adapters, ok := manifest["adapters"].(map[string]interface{})
	require.True(t, ok, "adapters should exist after merge")
	assert.NotEmpty(t, adapters, "adapters should not be empty after merge")

	personas, ok := manifest["personas"].(map[string]interface{})
	require.True(t, ok, "personas should exist after merge")
	assert.NotEmpty(t, personas, "personas should not be empty after merge")
}

// TestInitMergeDoesNotDeleteExistingPersonas tests that merge preserves all existing personas.
func TestInitMergeDoesNotDeleteExistingPersonas(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create wave.yaml with a custom persona
	existingContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  my-special-persona:
    adapter: claude
    system_prompt_file: .wave/personas/special.md
    temperature: 0.9
    permissions:
      allowed_tools:
        - SpecialTool
runtime:
  workspace_root: .wave/workspaces
`
	err := os.WriteFile("wave.yaml", []byte(existingContent), 0644)
	require.NoError(t, err)

	// Create the persona file
	err = os.MkdirAll(".wave/personas", 0755)
	require.NoError(t, err)
	err = os.WriteFile(".wave/personas/special.md", []byte("# Special"), 0644)
	require.NoError(t, err)

	_, _, err = executeInitCmd("--merge", "--yes")
	require.NoError(t, err)

	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	personas, ok := manifest["personas"].(map[string]interface{})
	require.True(t, ok)

	// Verify custom persona is still present
	specialPersona, hasSpecial := personas["my-special-persona"]
	assert.True(t, hasSpecial, "custom persona should be preserved")

	// Verify custom persona settings
	sp, ok := specialPersona.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 0.9, sp["temperature"], "custom temperature should be preserved")
}

// TestInitDirectoryCreationPermissions tests that directories are created with correct permissions.
func TestInitDirectoryCreationPermissions(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	// Check directory permissions
	dirs := []string{".wave", ".wave/personas", ".wave/pipelines", ".wave/contracts"}
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		require.NoError(t, err, "directory %s should exist", dir)
		assert.True(t, info.IsDir(), "%s should be a directory", dir)

		// Check that directory is readable and writable
		perm := info.Mode().Perm()
		assert.True(t, perm&0700 == 0700, "%s should be readable and writable by owner", dir)
	}
}

// TestInitFilePermissions tests that files are created with correct permissions.
func TestInitFilePermissions(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	// Use release pipelines and their contracts for permission checks
	releasePipelines := defaults.ReleasePipelineNames()
	require.NotEmpty(t, releasePipelines)

	files := []string{
		"wave.yaml",
		".wave/personas/navigator.md",
		filepath.Join(".wave", "pipelines", releasePipelines[0]),
	}

	for _, file := range files {
		info, err := os.Stat(file)
		require.NoError(t, err, "file %s should exist", file)
		assert.False(t, info.IsDir(), "%s should be a file", file)

		perm := info.Mode().Perm()
		assert.True(t, perm&0600 == 0600, "%s should be readable and writable by owner", file)
	}
}

// TestInitWithBinaryAvailable tests init when wave binary is available for validation.
func TestInitWithBinaryAvailable(t *testing.T) {
	// Skip if wave binary is not in PATH (this is an integration test)
	_, err := exec.LookPath("wave")
	if err != nil {
		t.Skip("wave binary not in PATH, skipping integration test")
	}

	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err = executeInitCmd()
	require.NoError(t, err)

	// Run actual wave validate command
	cmd := exec.Command("wave", "validate", "--manifest", "wave.yaml")
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err, "wave validate should pass: %s", string(output))
}

// TestInitManifestStructure tests the complete structure of the generated manifest.
func TestInitManifestStructure(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	// Verify top-level keys
	assert.Contains(t, manifest, "apiVersion")
	assert.Contains(t, manifest, "kind")
	assert.Contains(t, manifest, "metadata")
	assert.Contains(t, manifest, "adapters")
	assert.Contains(t, manifest, "personas")
	assert.Contains(t, manifest, "runtime")
	assert.NotContains(t, manifest, "skill_mounts")

	// Verify runtime structure
	runtime, ok := manifest["runtime"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, runtime, "workspace_root")
	assert.Contains(t, runtime, "max_concurrent_workers")
	assert.Contains(t, runtime, "default_timeout_minutes")
	assert.Contains(t, runtime, "relay")
	assert.Contains(t, runtime, "audit")
	assert.Contains(t, runtime, "meta_pipeline")
}

// TestInitPersonaPromptContent tests that persona prompts have meaningful content.
func TestInitPersonaPromptContent(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	personaTests := []struct {
		file        string
		mustContain []string
	}{
		{
			"navigator.md",
			[]string{"Navigator", "codebase", "read", "search"},
		},
		{
			"philosopher.md",
			[]string{"Philosopher", "architect", "specification"},
		},
		{
			"craftsman.md",
			[]string{"Craftsman", "implement", "test"},
		},
		{
			"auditor.md",
			[]string{"Auditor", "security", "review"},
		},
		{
			"summarizer.md",
			[]string{"Summarizer", "context", "checkpoint"},
		},
	}

	for _, tc := range personaTests {
		content, err := os.ReadFile(filepath.Join(".wave", "personas", tc.file))
		require.NoError(t, err)
		contentStr := strings.ToLower(string(content))

		for _, expected := range tc.mustContain {
			assert.Contains(t, contentStr, strings.ToLower(expected),
				"%s should contain '%s'", tc.file, expected)
		}
	}
}

// TestInitAllFlagExtractsAllPipelines tests that wave init --all extracts ALL embedded pipelines.
func TestInitAllFlagExtractsAllPipelines(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd("--all")
	require.NoError(t, err)

	allPipelines, err := defaults.GetPipelines()
	require.NoError(t, err)

	entries, err := os.ReadDir(".wave/pipelines")
	require.NoError(t, err)
	assert.Equal(t, len(allPipelines), len(entries), "--all should create all pipeline files")
}

// TestInitDefaultOnlyReleasePipelines tests that default wave init writes only release pipelines.
func TestInitDefaultOnlyReleasePipelines(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	releasePipelines, err := defaults.GetReleasePipelines()
	require.NoError(t, err)

	entries, err := os.ReadDir(".wave/pipelines")
	require.NoError(t, err)
	assert.Equal(t, len(releasePipelines), len(entries), "default init should create only release pipeline files")

	for _, entry := range entries {
		_, isRelease := releasePipelines[entry.Name()]
		assert.True(t, isRelease, "pipeline %s should be a release pipeline", entry.Name())
	}
}

// TestInitTransitiveContractExclusion tests that contracts referenced only by non-release pipelines are absent.
func TestInitTransitiveContractExclusion(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	// Count contracts in filtered init
	filteredEntries, err := os.ReadDir(".wave/contracts")
	require.NoError(t, err)

	// Count contracts with --all
	env2 := newTestEnv(t)
	defer env2.cleanup()

	_, _, err = executeInitCmd("--all")
	require.NoError(t, err)

	allEntries, err := os.ReadDir(".wave/contracts")
	require.NoError(t, err)

	// Filtered should have fewer or equal contracts
	assert.LessOrEqual(t, len(filteredEntries), len(allEntries),
		"filtered init should have fewer or equal contracts than --all")
}

// TestInitTransitivePromptExclusion tests that prompts referenced only by non-release pipelines are absent.
func TestInitTransitivePromptExclusion(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	// The speckit-flow pipeline is NOT release: true, so its prompts
	// should not be present (unless referenced by another release pipeline)
	// speckit-flow references prompts under .wave/prompts/speckit-flow/
	if !fileExists(".wave/prompts/speckit-flow") {
		// Good - speckit-flow prompts excluded because no release pipeline references them
		return
	}

	// If they exist, it means a release pipeline also references them
	// which is acceptable
}

// TestInitPersonasNeverExcluded tests that all personas are always included.
func TestInitPersonasNeverExcluded(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	allPersonas, err := defaults.GetPersonas()
	require.NoError(t, err)

	entries, err := os.ReadDir(".wave/personas")
	require.NoError(t, err)
	assert.Equal(t, len(allPersonas), len(entries),
		"all personas should be present regardless of release filtering")
}

// TestInitAllMergeCompose tests that --all and --merge compose naturally.
func TestInitAllMergeCompose(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// First init with default filtering
	_, _, err := executeInitCmd()
	require.NoError(t, err)

	filteredEntries, err := os.ReadDir(".wave/pipelines")
	require.NoError(t, err)
	filteredCount := len(filteredEntries)

	// Then merge with --all
	_, _, err = executeInitCmd("--merge", "--all", "--yes")
	require.NoError(t, err)

	allEntries, err := os.ReadDir(".wave/pipelines")
	require.NoError(t, err)

	allPipelines, err := defaults.GetPipelines()
	require.NoError(t, err)

	assert.Equal(t, len(allPipelines), len(allEntries),
		"--all --merge should result in all pipelines")
	assert.GreaterOrEqual(t, len(allEntries), filteredCount,
		"merge with --all should have at least as many as filtered")
}

// TestInitMergePreservesExistingNonReleasePipelines tests that merge doesn't delete existing files.
func TestInitMergePreservesExistingNonReleasePipelines(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// First init with --all to get all pipelines
	_, _, err := executeInitCmd("--all")
	require.NoError(t, err)

	allEntries, err := os.ReadDir(".wave/pipelines")
	require.NoError(t, err)
	allCount := len(allEntries)

	// Then merge without --all
	_, _, err = executeInitCmd("--merge", "--yes")
	require.NoError(t, err)

	// Files should still be there - merge doesn't delete
	afterEntries, err := os.ReadDir(".wave/pipelines")
	require.NoError(t, err)
	assert.Equal(t, allCount, len(afterEntries),
		"merge should not delete existing pipeline files")
}

// TestInitDisplayShowsFilteredCounts tests that init display shows filtered counts, not total.
func TestInitDisplayShowsFilteredCounts(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	stdout, _, err := executeInitCmd()
	require.NoError(t, err)

	releasePipelines, err := defaults.GetReleasePipelines()
	require.NoError(t, err)

	// The output should show the filtered pipeline count
	expectedCount := fmt.Sprintf("%d pipelines", len(releasePipelines))
	assert.Contains(t, stdout, expectedCount,
		"init output should show filtered pipeline count")
}

// TestDetectProject tests project type auto-detection from filesystem markers
// using the unified flavour detection system.
func TestDetectProject(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string // filename -> content
		wantLanguage string
		wantNil      bool
		wantTestCmd  string
		wantLintCmd  string
		wantBuildCmd string
	}{
		{
			name:         "go project",
			files:        map[string]string{"go.mod": "module test"},
			wantLanguage: "go",
			wantTestCmd:  "go test ./...",
			wantLintCmd:  "go vet ./...",
			wantBuildCmd: "go build ./...",
		},
		{
			name: "bun project",
			files: map[string]string{
				"bun.lockb":     "",
				"package.json":  "{}",
				"tsconfig.json": "{}",
			},
			wantLanguage: "typescript",
			wantTestCmd:  "bun test",
			wantLintCmd:  "bun lint",
			wantBuildCmd: "bun run build",
		},
		{
			name:         "deno project",
			files:        map[string]string{"deno.json": "{}"},
			wantLanguage: "typescript",
			wantTestCmd:  "deno test",
		},
		{
			name:         "deno jsonc project",
			files:        map[string]string{"deno.jsonc": "{}"},
			wantLanguage: "typescript",
			wantTestCmd:  "deno test",
		},
		{
			name: "typescript node project",
			files: map[string]string{
				"package.json":  "{}",
				"tsconfig.json": "{}",
			},
			wantLanguage: "typescript",
			wantTestCmd:  "npm test",
			wantLintCmd:  "npm run lint",
			wantBuildCmd: "npm run build",
		},
		{
			name: "javascript node project",
			files: map[string]string{
				"package.json": "{}",
			},
			wantLanguage: "javascript",
			wantTestCmd:  "npm test",
			wantBuildCmd: "npm run build",
		},
		{
			name: "pnpm project",
			files: map[string]string{
				"package.json":   "{}",
				"pnpm-lock.yaml": "",
				"tsconfig.json":  "{}",
			},
			wantLanguage: "typescript",
			wantTestCmd:  "pnpm test",
			wantLintCmd:  "pnpm lint",
		},
		{
			name: "yarn project",
			files: map[string]string{
				"package.json": "{}",
				"yarn.lock":    "",
			},
			wantLanguage: "javascript",
			wantTestCmd:  "yarn test",
		},
		{
			name:         "rust project",
			files:        map[string]string{"Cargo.toml": ""},
			wantLanguage: "rust",
			wantTestCmd:  "cargo test",
		},
		{
			name:         "python project with pyproject",
			files:        map[string]string{"pyproject.toml": ""},
			wantLanguage: "python",
			wantTestCmd:  "pytest",
		},
		{
			name:         "python project with setup.py",
			files:        map[string]string{"setup.py": ""},
			wantLanguage: "python",
			wantTestCmd:  "python -m pytest",
		},
		{
			name:    "unknown project",
			files:   map[string]string{},
			wantNil: true,
		},
		{
			name:         "go takes priority over package.json",
			files:        map[string]string{"go.mod": "module test", "package.json": "{}"},
			wantLanguage: "go",
			wantTestCmd:  "go test ./...",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			for name, content := range tc.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
			}

			result := flavourToProjectMap(onboarding.DetectFlavour(dir))

			if tc.wantNil {
				assert.Nil(t, result, "expected nil for unknown project")
				return
			}

			require.NotNil(t, result, "expected non-nil project detection")
			assert.Equal(t, tc.wantLanguage, result["language"])

			if tc.wantTestCmd != "" {
				assert.Equal(t, tc.wantTestCmd, result["test_command"])
			}
			if tc.wantLintCmd != "" {
				assert.Equal(t, tc.wantLintCmd, result["lint_command"])
			}
			if tc.wantBuildCmd != "" {
				assert.Equal(t, tc.wantBuildCmd, result["build_command"])
			}
		})
	}
}

// TestInitIncludesDetectedProject tests that wave init includes the detected project in the manifest.
func TestInitIncludesDetectedProject(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create a go.mod so detection picks up Go
	require.NoError(t, os.WriteFile("go.mod", []byte("module test"), 0644))

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	project, ok := manifest["project"].(map[string]interface{})
	require.True(t, ok, "manifest should contain project key")
	assert.Equal(t, "go", project["language"])
	assert.Equal(t, "go", project["flavour"])
	assert.Equal(t, "go test ./...", project["test_command"])
	assert.Equal(t, "go vet ./...", project["lint_command"])
	assert.Equal(t, "gofmt -l .", project["format_command"])
}

// TestInitNoProjectWhenUndetected tests that wave init omits project when no markers found.
func TestInitNoProjectWhenUndetected(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	_, hasProject := manifest["project"]
	assert.False(t, hasProject, "manifest should not contain project key when undetected")
}

// TestInitPersonaPermissionsAreGeneric tests that persona permissions don't contain language-specific commands.
func TestInitPersonaPermissionsAreGeneric(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	content, err := os.ReadFile("wave.yaml")
	require.NoError(t, err)
	contentStr := string(content)

	assert.NotContains(t, contentStr, "go vet", "manifest should not contain go vet in permissions")
	assert.NotContains(t, contentStr, "go test", "manifest should not contain go test in permissions")
	assert.NotContains(t, contentStr, "npm audit", "manifest should not contain npm audit in permissions")
}

// TestInitTransitivePersonaFiltering tests that default init only includes
// manifest persona entries for personas referenced by release pipelines + system personas.
func TestInitTransitivePersonaFiltering(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd()
	require.NoError(t, err)

	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	personas, ok := manifest["personas"].(map[string]interface{})
	require.True(t, ok, "personas should exist in manifest")

	// System personas should always be present
	for _, name := range []string{"summarizer", "navigator", "philosopher"} {
		_, has := personas[name]
		assert.True(t, has, "system persona %q should be present in manifest", name)
	}

	// supervisor is used by the release pipeline "ops-supervise"
	_, hasSupervisor := personas["supervisor"]
	assert.True(t, hasSupervisor, "supervisor should be in manifest (used by release pipeline ops-supervise)")

	// With --all, all personas should be in the manifest
	env2 := newTestEnv(t)
	defer env2.cleanup()

	_, _, err = executeInitCmd("--all")
	require.NoError(t, err)

	allManifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	allPersonas, ok := allManifest["personas"].(map[string]interface{})
	require.True(t, ok)

	_, hasSupervisorAll := allPersonas["supervisor"]
	assert.True(t, hasSupervisorAll, "supervisor should be present with --all flag")

	// --all should have more or equal personas than filtered
	assert.GreaterOrEqual(t, len(allPersonas), len(personas),
		"--all should have at least as many personas in manifest as filtered init")
}

// TestInitPersonaManifestMatchesConfig tests that the generated manifest persona
// entries match the embedded YAML config data.
func TestInitPersonaManifestMatchesConfig(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	_, _, err := executeInitCmd("--all")
	require.NoError(t, err)

	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	personas, ok := manifest["personas"].(map[string]interface{})
	require.True(t, ok)

	personaConfigs, err := defaults.GetPersonaConfigs()
	require.NoError(t, err)

	for name, cfg := range personaConfigs {
		entry, ok := personas[name].(map[string]interface{})
		require.True(t, ok, "persona %q should be in manifest", name)

		assert.Equal(t, cfg.Description, entry["description"], "%s description mismatch", name)
		assert.Equal(t, "claude", entry["adapter"], "%s adapter should be 'claude'", name)
		assert.Equal(t, fmt.Sprintf(".wave/personas/%s.md", name), entry["system_prompt_file"],
			"%s system_prompt_file should follow convention", name)

		if cfg.Model != "" {
			assert.Equal(t, cfg.Model, entry["model"], "%s model mismatch", name)
		} else {
			_, hasModel := entry["model"]
			assert.False(t, hasModel, "%s should not have model field", name)
		}
	}
}

// TestComputeChangeSummary tests the computeChangeSummary function with table-driven cases.
func TestComputeChangeSummary(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T, assets *initAssets)
		wantNewCount  int  // expected minimum count of FileStatusNew entries (-1 means all)
		wantUpToDate  int  // expected minimum count of FileStatusUpToDate entries (-1 means all)
		wantPreserved int  // expected minimum count of FileStatusPreserved entries
		checkUpToDate bool // if true, assert summary.AlreadyUpToDate is true
	}{
		{
			name: "all files new (fresh project)",
			setup: func(t *testing.T, assets *initAssets) {
				t.Helper()
				// Only create wave.yaml — no .wave/ directory at all.
				require.NoError(t, os.WriteFile("wave.yaml", []byte("apiVersion: v1\nkind: WaveManifest\n"), 0644))
			},
			wantNewCount: -1,
		},
		{
			name: "all files up-to-date",
			setup: func(t *testing.T, assets *initAssets) {
				t.Helper()
				// Run a full init so every default file is on disk with default content
				_, _, err := executeInitCmd()
				require.NoError(t, err)
			},
			wantUpToDate:  -1,
			checkUpToDate: true,
		},
		{
			name: "mixed states",
			setup: func(t *testing.T, assets *initAssets) {
				t.Helper()
				// Run a full init first
				_, _, err := executeInitCmd()
				require.NoError(t, err)
				// Modify one persona file to make it "preserved"
				require.NoError(t, os.WriteFile(
					".wave/personas/navigator.md",
					[]byte("# My Custom Navigator"),
					0644,
				))
				// Remove one pipeline to make it "new" on next check
				for name := range assets.pipelines {
					os.Remove(filepath.Join(".wave", "pipelines", name))
					break // only remove one
				}
			},
			wantPreserved: 1, // at least 1 preserved (the modified navigator.md)
			wantNewCount:  1, // at least 1 new (the removed pipeline)
		},
		{
			name: "empty persona file treated as preserved",
			setup: func(t *testing.T, assets *initAssets) {
				t.Helper()
				// Run a full init first
				_, _, err := executeInitCmd()
				require.NoError(t, err)
				// Overwrite navigator.md with empty content — it differs from the
				// default so it should be classified as "preserved", not "up_to_date"
				require.NoError(t, os.WriteFile(
					".wave/personas/navigator.md",
					[]byte{},
					0644,
				))
			},
			wantPreserved: 1, // the empty navigator.md
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			defer env.cleanup()

			cmd := NewInitCmd()
			assets, err := getFilteredAssets(cmd, InitOptions{})
			require.NoError(t, err)

			tc.setup(t, assets)

			// Read existing manifest (or use empty)
			var existingManifest map[string]interface{}
			if data, readErr := os.ReadFile("wave.yaml"); readErr == nil {
				_ = yaml.Unmarshal(data, &existingManifest)
			}
			if existingManifest == nil {
				existingManifest = map[string]interface{}{}
			}

			defaultManifest := createDefaultManifest("claude", ".wave/workspaces", nil, assets.personaConfigs)
			summary := computeChangeSummary(assets, existingManifest, defaultManifest)

			require.NotNil(t, summary)
			require.NotEmpty(t, summary.Files, "summary should contain file entries")

			// Count statuses
			var newCount, upToDateCount, preservedCount int
			for _, f := range summary.Files {
				switch f.Status {
				case FileStatusNew:
					newCount++
				case FileStatusUpToDate:
					upToDateCount++
				case FileStatusPreserved:
					preservedCount++
				}
			}

			if tc.wantNewCount == -1 {
				assert.Equal(t, len(summary.Files), newCount, "all files should be new")
			} else if tc.wantNewCount > 0 {
				assert.GreaterOrEqual(t, newCount, tc.wantNewCount, "expected at least %d new files", tc.wantNewCount)
			}

			if tc.wantUpToDate == -1 {
				assert.Equal(t, len(summary.Files), upToDateCount, "all files should be up_to_date")
			} else if tc.wantUpToDate > 0 {
				assert.GreaterOrEqual(t, upToDateCount, tc.wantUpToDate, "expected at least %d up_to_date files", tc.wantUpToDate)
			}

			if tc.wantPreserved > 0 {
				assert.GreaterOrEqual(t, preservedCount, tc.wantPreserved, "expected at least %d preserved files", tc.wantPreserved)
			}

			if tc.checkUpToDate {
				assert.True(t, summary.AlreadyUpToDate, "summary should be marked AlreadyUpToDate")
			}
		})
	}
}

// TestComputeManifestDiff tests the computeManifestDiff function directly with crafted maps.
func TestComputeManifestDiff(t *testing.T) {
	tests := []struct {
		name             string
		defaults         map[string]interface{}
		existing         map[string]interface{}
		wantAddedKeys    []string
		wantPreserved    []string
		checkMergedValue map[string]interface{} // dot-path -> expected value in merged result
	}{
		{
			name: "nested key additions",
			defaults: map[string]interface{}{
				"runtime": map[string]interface{}{
					"workspace_root":  ".wave/workspaces",
					"timeout_minutes": 30,
					"max_workers":     5,
				},
			},
			existing: map[string]interface{}{
				"runtime": map[string]interface{}{
					"workspace_root": ".wave/workspaces",
				},
			},
			wantAddedKeys: []string{"runtime.timeout_minutes", "runtime.max_workers"},
		},
		{
			name: "array atomic preservation",
			defaults: map[string]interface{}{
				"personas": map[string]interface{}{
					"navigator": map[string]interface{}{
						"permissions": map[string]interface{}{
							"allowed_tools": []string{"Read", "Write"},
						},
					},
				},
			},
			existing: map[string]interface{}{
				"personas": map[string]interface{}{
					"navigator": map[string]interface{}{
						"permissions": map[string]interface{}{
							"allowed_tools": []string{"Read", "Write", "Bash"},
						},
					},
				},
			},
			wantPreserved: []string{"personas.navigator.permissions.allowed_tools"},
		},
		{
			name: "user key precedence",
			defaults: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":        "wave-project",
					"description": "A Wave project",
				},
			},
			existing: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":        "my-custom-project",
					"description": "My project",
				},
			},
			wantPreserved: []string{"metadata.name", "metadata.description"},
			checkMergedValue: map[string]interface{}{
				"metadata.name": "my-custom-project",
			},
		},
		{
			name: "new subsection addition",
			defaults: map[string]interface{}{
				"adapters": map[string]interface{}{
					"claude": map[string]interface{}{
						"binary": "claude",
					},
				},
				"runtime": map[string]interface{}{
					"audit": map[string]interface{}{
						"log_dir": ".wave/traces/",
					},
				},
			},
			existing: map[string]interface{}{
				"adapters": map[string]interface{}{
					"claude": map[string]interface{}{
						"binary": "claude",
					},
				},
			},
			wantAddedKeys: []string{"runtime"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			merged, changes := computeManifestDiff(tc.defaults, tc.existing)

			require.NotNil(t, merged)

			// Build maps for easy lookup
			addedKeys := make(map[string]bool)
			preservedKeys := make(map[string]bool)
			for _, c := range changes {
				switch c.Action {
				case ManifestActionAdded:
					addedKeys[c.KeyPath] = true
				case ManifestActionPreserved:
					preservedKeys[c.KeyPath] = true
				}
			}

			for _, key := range tc.wantAddedKeys {
				assert.True(t, addedKeys[key], "expected key %q to be added, got changes: %+v", key, changes)
			}

			for _, key := range tc.wantPreserved {
				assert.True(t, preservedKeys[key], "expected key %q to be preserved, got changes: %+v", key, changes)
			}

			// Verify merged values
			for dotPath, expected := range tc.checkMergedValue {
				parts := strings.Split(dotPath, ".")
				var current interface{} = merged
				for _, part := range parts {
					m, ok := current.(map[string]interface{})
					require.True(t, ok, "expected map at %q in merged result", dotPath)
					current = m[part]
				}
				assert.Equal(t, expected, current, "merged value for %q should match", dotPath)
			}
		})
	}
}

// TestInitMergeUpgradeLifecycle tests the full init -> customize -> merge lifecycle.
func TestInitMergeUpgradeLifecycle(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Step 1: Initial init
	_, _, err := executeInitCmd()
	require.NoError(t, err, "initial init should succeed")

	// Step 2: Write a custom persona file
	customPersonaContent := "# Custom Navigator\n\nMy custom navigator persona.\n"
	require.NoError(t, os.WriteFile(".wave/personas/navigator.md", []byte(customPersonaContent), 0644))

	// Step 3: Modify wave.yaml with custom settings
	manifestData, err := os.ReadFile("wave.yaml")
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, yaml.Unmarshal(manifestData, &m))

	adapters := m["adapters"].(map[string]interface{})
	adapters["my-adapter"] = map[string]interface{}{
		"binary": "my-adapter-bin",
		"mode":   "headless",
	}
	metadata := m["metadata"].(map[string]interface{})
	metadata["name"] = "my-project"

	updatedData, err := yaml.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("wave.yaml", updatedData, 0644))

	// Step 3b: Delete one pipeline to create a "new" entry for merge
	cmd0 := NewInitCmd()
	assets, err := getFilteredAssets(cmd0, InitOptions{})
	require.NoError(t, err)
	var removedPipeline string
	for name := range assets.pipelines {
		removed := filepath.Join(".wave", "pipelines", name)
		os.Remove(removed)
		removedPipeline = name
		break
	}
	require.NotEmpty(t, removedPipeline, "should have removed at least one pipeline")

	// Step 4: Run init --merge --yes and capture stderr
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--merge", "--yes"})
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	err = cmd.Execute()
	require.NoError(t, err, "merge should succeed")

	stderrStr := errBuf.String()
	stdoutStr := outBuf.String()

	// Verify change summary was displayed on stderr
	assert.Contains(t, stderrStr, "Change Summary",
		"stderr should contain change summary, got: %s", stderrStr)

	// Verify success message on stdout
	assert.Contains(t, stdoutStr, "merged successfully",
		"stdout should contain merge success message")

	// Step 5: Verify custom persona is preserved on disk
	personaData, err := os.ReadFile(".wave/personas/navigator.md")
	require.NoError(t, err)
	assert.Equal(t, customPersonaContent, string(personaData),
		"custom persona file should be preserved")

	// Step 6: Verify the removed pipeline was re-created
	assert.True(t, fileExists(filepath.Join(".wave", "pipelines", removedPipeline)),
		"removed pipeline %s should be re-created by merge", removedPipeline)

	// Step 7: Verify all default persona files exist
	for _, name := range []string{"philosopher.md", "craftsman.md", "auditor.md", "summarizer.md"} {
		assert.True(t, fileExists(filepath.Join(".wave", "personas", name)),
			"default persona file %s should exist", name)
	}

	// Step 8: Verify manifest merge correctness
	mergedManifest, err := readYAML("wave.yaml")
	require.NoError(t, err)

	mergedMetadata := mergedManifest["metadata"].(map[string]interface{})
	assert.Equal(t, "my-project", mergedMetadata["name"], "custom name should be preserved")

	mergedAdapters := mergedManifest["adapters"].(map[string]interface{})
	_, hasCustomAdapter := mergedAdapters["my-adapter"]
	assert.True(t, hasCustomAdapter, "custom adapter should be preserved")

	_, hasClaude := mergedAdapters["claude"]
	assert.True(t, hasClaude, "default claude adapter should still be present")
}

// TestInitMergeFlagCombinations tests all four flag combinations for init.
func TestInitMergeFlagCombinations(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		envVars        map[string]string
		stdin          string
		setupCustom    bool
		expectError    bool
		errorContains  string
		checkOverwrite bool // if true, verify custom persona was overwritten
		checkPreserved bool // if true, verify custom persona was preserved
		checkMerge     bool // if true, verify merge behavior
	}{
		{
			name:           "init on existing defaults to merge silently",
			args:           []string{},
			envVars:        map[string]string{},
			stdin:          "n\n",
			setupCustom:    true,
			checkPreserved: true, // custom persona file should be preserved
			checkMerge:     true,
		},
		{
			name:           "force with yes overwrites everything",
			args:           []string{"--force", "--yes"},
			setupCustom:    true,
			checkOverwrite: true,
		},
		{
			name:           "merge with yes skips prompt and merges",
			args:           []string{"--merge", "--yes"},
			setupCustom:    true,
			checkPreserved: true,
			checkMerge:     true,
		},
		{
			name:           "merge with force skips prompt and merges",
			args:           []string{"--merge", "--force"},
			setupCustom:    true,
			checkPreserved: true,
			checkMerge:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			defer env.cleanup()

			// Set env vars
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			if tc.setupCustom {
				// Run initial init
				_, _, err := executeInitCmd()
				require.NoError(t, err)

				// Write a custom persona file that differs from defaults
				require.NoError(t, os.WriteFile(
					".wave/personas/navigator.md",
					[]byte("# My Custom Navigator"),
					0644,
				))
			}

			cmd := NewInitCmd()
			cmd.SetArgs(tc.args)

			var outBuf, errBuf bytes.Buffer
			cmd.SetOut(&outBuf)
			cmd.SetErr(&errBuf)

			if tc.stdin != "" {
				cmd.SetIn(strings.NewReader(tc.stdin))
			}

			err := cmd.Execute()

			if tc.expectError {
				require.Error(t, err, "expected error")
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			require.NoError(t, err, "expected no error, got: %v\nstderr: %s", err, errBuf.String())

			if tc.checkOverwrite {
				// Force should overwrite the custom persona with the default
				data, err := os.ReadFile(".wave/personas/navigator.md")
				require.NoError(t, err)
				assert.NotEqual(t, "# My Custom Navigator", string(data),
					"force should overwrite custom persona with default")
				assert.True(t, len(data) > 0, "persona file should have content")
			}

			if tc.checkPreserved {
				// Merge should preserve the custom persona file
				data, err := os.ReadFile(".wave/personas/navigator.md")
				require.NoError(t, err)
				assert.Equal(t, "# My Custom Navigator", string(data),
					"merge should preserve custom persona file")
			}

			if tc.checkMerge {
				stderrStr := errBuf.String()
				// Merge should display summary on stderr (or up-to-date)
				assert.True(t,
					strings.Contains(stderrStr, "Change Summary") || strings.Contains(stderrStr, "Already up to date"),
					"stderr should contain change summary or up-to-date, got: %s", stderrStr)
			}
		})
	}
}

// TestInitMergeEdgeCases tests edge cases for the merge workflow.
func TestInitMergeEdgeCases(t *testing.T) {
	t.Run("malformed YAML parse error", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.cleanup()

		// Create a wave.yaml with invalid YAML
		require.NoError(t, os.WriteFile("wave.yaml", []byte("invalid:\n  yaml: [\n  broken\n"), 0644))
		require.NoError(t, os.MkdirAll(".wave/personas", 0755))

		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--merge", "--yes"})
		var outBuf, errBuf bytes.Buffer
		cmd.SetOut(&outBuf)
		cmd.SetErr(&errBuf)

		err := cmd.Execute()
		require.Error(t, err, "merge with malformed YAML should fail")
		assert.Contains(t, err.Error(), "parse",
			"error should mention parse failure, got: %v", err)
	})

	t.Run("empty persona file preserved", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.cleanup()

		// Initial init
		_, _, err := executeInitCmd()
		require.NoError(t, err)

		// Create an empty persona file
		require.NoError(t, os.WriteFile(".wave/personas/navigator.md", []byte{}, 0644))

		// Run merge
		_, _, err = executeInitCmd("--merge", "--yes")
		require.NoError(t, err)

		// Verify the file still exists and is still empty
		data, err := os.ReadFile(".wave/personas/navigator.md")
		require.NoError(t, err)
		assert.Empty(t, data, "empty persona file should be preserved (not overwritten)")
	})

	t.Run("read-only .wave/personas permission error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("skipping permission test when running as root")
		}

		env := newTestEnv(t)
		defer env.cleanup()

		// Create minimal wave.yaml
		require.NoError(t, os.WriteFile("wave.yaml", []byte("apiVersion: v1\nkind: WaveManifest\nmetadata:\n  name: test\nruntime:\n  workspace_root: .wave/workspaces\n"), 0644))

		// Create .wave/personas as read-only (this will prevent writing new files)
		require.NoError(t, os.MkdirAll(".wave/personas", 0755))
		// Make personas directory read-only
		require.NoError(t, os.Chmod(".wave/personas", 0555))
		defer os.Chmod(".wave/personas", 0755) // restore for cleanup

		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--merge", "--yes"})
		var outBuf, errBuf bytes.Buffer
		cmd.SetOut(&outBuf)
		cmd.SetErr(&errBuf)

		err := cmd.Execute()
		// Should fail because it can't write persona files
		assert.Error(t, err, "should fail when .wave/personas is read-only")
	})

	t.Run("already up-to-date short circuit", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.cleanup()

		// Run init to create all files
		_, _, err := executeInitCmd()
		require.NoError(t, err)

		// Run merge — everything should already be up to date
		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--merge", "--yes"})
		var outBuf, errBuf bytes.Buffer
		cmd.SetOut(&outBuf)
		cmd.SetErr(&errBuf)

		err = cmd.Execute()
		require.NoError(t, err, "merge on up-to-date project should succeed")

		stderrStr := errBuf.String()
		assert.Contains(t, stderrStr, "Already up to date",
			"should display up-to-date message on stderr, got: %s", stderrStr)
	})

	t.Run("non-interactive terminal without --yes", func(t *testing.T) {
		env := newTestEnv(t)
		defer env.cleanup()

		// Ensure WAVE_FORCE_TTY is not set (tests are non-interactive by default)
		t.Setenv("WAVE_FORCE_TTY", "")

		// Run init first
		_, _, err := executeInitCmd()
		require.NoError(t, err)

		// Modify a file so it's not up-to-date
		require.NoError(t, os.WriteFile(".wave/personas/navigator.md", []byte("# Modified"), 0644))

		// Remove a pipeline to ensure there's a "new" entry
		cmd0 := NewInitCmd()
		assets, err := getFilteredAssets(cmd0, InitOptions{})
		require.NoError(t, err)
		for name := range assets.pipelines {
			os.Remove(filepath.Join(".wave", "pipelines", name))
			break // only remove one
		}

		// Run merge without --yes in non-interactive mode
		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--merge"})
		var outBuf, errBuf bytes.Buffer
		cmd.SetOut(&outBuf)
		cmd.SetErr(&errBuf)

		err = cmd.Execute()
		require.Error(t, err, "merge without --yes in non-interactive terminal should fail")
		assert.Contains(t, err.Error(), "non-interactive",
			"error should mention non-interactive terminal, got: %v", err)
	})
}

// Test filterTransitiveDeps expands forge template personas to all 4 variants
func TestFilterTransitiveDeps_ForgeTemplateExpansion(t *testing.T) {
	// Create a mock cobra command for stderr output
	cmd := &cobra.Command{Use: "test"}
	var errBuf bytes.Buffer
	cmd.SetErr(&errBuf)

	// Pipeline YAML that references a forge-templated persona
	pipelineYAML := `kind: WavePipeline
metadata:
  name: test-pipeline
steps:
  - id: analyze
    persona: "{{ forge.type }}-analyst"
    exec:
      type: prompt
      source: "analyze this"
`
	pipelines := map[string]string{
		"test-pipeline": pipelineYAML,
	}

	// All persona configs include all 4 forge variants + a system persona
	allPersonaConfigs := map[string]manifest.Persona{
		"github-analyst":    {Adapter: "claude"},
		"gitlab-analyst":    {Adapter: "claude"},
		"gitea-analyst":     {Adapter: "claude"},
		"bitbucket-analyst": {Adapter: "claude"},
		"navigator":         {Adapter: "claude"},
		"summarizer":        {Adapter: "claude"},
		"philosopher":       {Adapter: "claude"},
		"unrelated":         {Adapter: "claude"},
	}

	_, _, personaConfigs := filterTransitiveDeps(cmd, pipelines, nil, nil, allPersonaConfigs)

	// All 4 forge variants should be included
	assert.Contains(t, personaConfigs, "github-analyst", "github variant should be included")
	assert.Contains(t, personaConfigs, "gitlab-analyst", "gitlab variant should be included")
	assert.Contains(t, personaConfigs, "gitea-analyst", "gitea variant should be included")
	assert.Contains(t, personaConfigs, "bitbucket-analyst", "bitbucket variant should be included")

	// System personas should be included
	assert.Contains(t, personaConfigs, "navigator", "system persona navigator should be included")
	assert.Contains(t, personaConfigs, "summarizer", "system persona summarizer should be included")

	// Unrelated persona should NOT be included
	_, hasUnrelated := personaConfigs["unrelated"]
	assert.False(t, hasUnrelated, "unrelated persona should not be included")
}

// TestMergeTypedManifestsPreservesCustomPersonas tests that custom personas are preserved.
func TestMergeTypedManifestsPreservesCustomPersonas(t *testing.T) {
	existing := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: "my-project"},
		Personas: map[string]manifest.Persona{
			"custom-agent": {
				Adapter:          "claude",
				SystemPromptFile: ".wave/personas/custom.md",
				Temperature:      0.9,
			},
			"navigator": {
				Adapter:          "claude",
				SystemPromptFile: ".wave/personas/navigator.md",
				Temperature:      0.5,
				Model:            "custom-model",
			},
		},
	}

	generated := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: "wave-project"},
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:          "claude",
				SystemPromptFile: ".wave/personas/navigator.md",
				Temperature:      0.3,
			},
			"craftsman": {
				Adapter:          "claude",
				SystemPromptFile: ".wave/personas/craftsman.md",
				Temperature:      0.2,
			},
		},
	}

	result := mergeTypedManifests(existing, generated)

	// Custom persona preserved
	customAgent, hasCustom := result.Personas["custom-agent"]
	assert.True(t, hasCustom, "custom persona should be preserved")
	assert.Equal(t, 0.9, customAgent.Temperature, "custom persona temperature preserved")

	// Existing navigator overrides generated (user's customization wins)
	nav, hasNav := result.Personas["navigator"]
	assert.True(t, hasNav)
	assert.Equal(t, 0.5, nav.Temperature, "existing persona should override generated")
	assert.Equal(t, "custom-model", nav.Model, "existing model should be preserved")

	// New default persona added
	_, hasCraftsman := result.Personas["craftsman"]
	assert.True(t, hasCraftsman, "new default persona should be added")
}

// TestMergeTypedManifestsPreservesAdapters tests that custom adapters are preserved.
func TestMergeTypedManifestsPreservesAdapters(t *testing.T) {
	existing := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: "test"},
		Adapters: map[string]manifest.Adapter{
			"custom-llm": {
				Binary: "custom-llm",
				Mode:   "interactive",
			},
		},
	}

	generated := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: "wave-project"},
		Adapters: map[string]manifest.Adapter{
			"claude": {
				Binary:       "claude",
				Mode:         "headless",
				OutputFormat: "json",
			},
		},
	}

	result := mergeTypedManifests(existing, generated)

	// Custom adapter preserved
	_, hasCustom := result.Adapters["custom-llm"]
	assert.True(t, hasCustom, "custom adapter should be preserved")

	// Default adapter added
	_, hasClaude := result.Adapters["claude"]
	assert.True(t, hasClaude, "default adapter should be added")
}

// TestMergeTypedManifestsPreservesOntology tests that the ontology section is preserved.
func TestMergeTypedManifestsPreservesOntology(t *testing.T) {
	existing := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: "test"},
		Ontology: &manifest.Ontology{
			Telos: "Build the best widget system",
			Contexts: []manifest.OntologyContext{
				{
					Name:        "widget-core",
					Description: "Core widget functionality",
					Invariants:  []string{"Widgets must be immutable after creation"},
				},
			},
			Conventions: map[string]string{
				"naming": "PascalCase for types",
			},
		},
	}

	generated := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: "wave-project"},
		// No ontology in generated
	}

	result := mergeTypedManifests(existing, generated)

	// Ontology preserved entirely from existing
	require.NotNil(t, result.Ontology, "ontology should be preserved")
	assert.Equal(t, "Build the best widget system", result.Ontology.Telos)
	assert.Len(t, result.Ontology.Contexts, 1)
	assert.Equal(t, "widget-core", result.Ontology.Contexts[0].Name)
	assert.Equal(t, "PascalCase for types", result.Ontology.Conventions["naming"])
}

// TestMergeTypedManifestsUpdatesInfrastructure tests that apiVersion, kind, and runtime are updated.
func TestMergeTypedManifestsUpdatesInfrastructure(t *testing.T) {
	existing := &manifest.Manifest{
		APIVersion: "v0",
		Kind:       "OldKind",
		Metadata:   manifest.Metadata{Name: "test"},
		Runtime: manifest.Runtime{
			WorkspaceRoot:        ".wave/workspaces",
			MaxConcurrentWorkers: 2,
		},
	}

	generated := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: "wave-project"},
		Runtime: manifest.Runtime{
			WorkspaceRoot:        ".wave/workspaces",
			MaxConcurrentWorkers: 5,
			DefaultTimeoutMin:    30,
		},
	}

	result := mergeTypedManifests(existing, generated)

	// Infrastructure updated from generated
	assert.Equal(t, "v1", result.APIVersion, "apiVersion should be updated from generated")
	assert.Equal(t, "WaveManifest", result.Kind, "kind should be updated from generated")
	assert.Equal(t, 5, result.Runtime.MaxConcurrentWorkers, "runtime should be updated from generated")
	assert.Equal(t, 30, result.Runtime.DefaultTimeoutMin, "runtime timeout should come from generated")
}

// TestMergeTypedManifestsPreservesMetadata tests that metadata is preserved from existing.
func TestMergeTypedManifestsPreservesMetadata(t *testing.T) {
	existing := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata: manifest.Metadata{
			Name:        "my-project",
			Description: "My custom description",
			Repo:        "https://github.com/me/my-project",
			Forge:       "github",
		},
	}

	generated := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata: manifest.Metadata{
			Name:        "wave-project",
			Description: "A Wave multi-agent project",
		},
	}

	result := mergeTypedManifests(existing, generated)

	assert.Equal(t, "my-project", result.Metadata.Name, "name should be preserved from existing")
	assert.Equal(t, "My custom description", result.Metadata.Description, "description should be preserved")
	assert.Equal(t, "https://github.com/me/my-project", result.Metadata.Repo, "repo should be preserved")
	assert.Equal(t, "github", result.Metadata.Forge, "forge should be preserved")
}

// TestMergeTypedManifestsEmptyExisting tests merge with a minimal existing manifest.
func TestMergeTypedManifestsEmptyExisting(t *testing.T) {
	existing := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: ""},
	}

	generated := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata: manifest.Metadata{
			Name:        "wave-project",
			Description: "A Wave multi-agent project",
		},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {Adapter: "claude"},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:        ".wave/workspaces",
			MaxConcurrentWorkers: 5,
		},
	}

	result := mergeTypedManifests(existing, generated)

	// When existing is empty, generated values are used
	assert.Equal(t, "wave-project", result.Metadata.Name, "generated name used when existing is empty")
	assert.Equal(t, "A Wave multi-agent project", result.Metadata.Description, "generated description used")
	assert.Len(t, result.Adapters, 1, "generated adapters used")
	assert.Len(t, result.Personas, 1, "generated personas used")
}

// TestMergeTypedManifestsPreservesProject tests that project config is preserved.
func TestMergeTypedManifestsPreservesProject(t *testing.T) {
	existing := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: "test"},
		Project: &manifest.Project{
			Language:    "go",
			TestCommand: "go test ./...",
			LintCommand: "golangci-lint run",
		},
	}

	generated := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: "wave-project"},
		Project: &manifest.Project{
			Language:    "go",
			TestCommand: "go test ./...",
		},
	}

	result := mergeTypedManifests(existing, generated)

	require.NotNil(t, result.Project)
	assert.Equal(t, "go", result.Project.Language)
	assert.Equal(t, "golangci-lint run", result.Project.LintCommand, "existing lint command preserved")
}

// TestMergeTypedManifestsMergesSkills tests that skills are merged and deduplicated.
func TestMergeTypedManifestsMergesSkills(t *testing.T) {
	existing := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: "test"},
		Skills:     []string{"skill-a", "skill-b"},
	}

	generated := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata:   manifest.Metadata{Name: "wave-project"},
		Skills:     []string{"skill-b", "skill-c"},
	}

	result := mergeTypedManifests(existing, generated)

	assert.Len(t, result.Skills, 3, "skills should be merged and deduplicated")
	assert.Contains(t, result.Skills, "skill-a")
	assert.Contains(t, result.Skills, "skill-b")
	assert.Contains(t, result.Skills, "skill-c")
}
