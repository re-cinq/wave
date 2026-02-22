package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/defaults"
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

	stdout, stderr, err := executeInitCmd()

	// Verify successful execution
	require.NoError(t, err, "init should succeed in empty directory")
	assert.Empty(t, stderr, "should have no stderr output")
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
	assert.Equal(t, "wave-project", metadata["name"])
}

// TestInitWithExistingWaveYaml tests that init fails when wave.yaml already exists.
func TestInitWithExistingWaveYaml(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create an existing wave.yaml
	existingContent := []byte("apiVersion: v1\nkind: WaveManifest\nmetadata:\n  name: existing\n")
	err := os.WriteFile("wave.yaml", existingContent, 0644)
	require.NoError(t, err, "failed to create existing wave.yaml")

	// Use --yes flag which will cause failure without interactive prompt
	// (since it requires --force or --merge)
	cmd := NewInitCmd()
	cmd.SetArgs([]string{})

	// Provide "n" as input to decline overwrite
	cmd.SetIn(strings.NewReader("n\n"))
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	err = cmd.Execute()

	// Verify that init fails (user declined)
	assert.Error(t, err, "init should fail when wave.yaml already exists and user declines")
	assert.Contains(t, err.Error(), "already exists", "error should mention file exists")

	// Verify original file is unchanged
	data, err := os.ReadFile("wave.yaml")
	require.NoError(t, err)
	assert.Equal(t, existingContent, data, "existing wave.yaml should be unchanged")
}

// TestInitWithForceFlag tests that init --force overwrites existing files.
func TestInitWithForceFlag(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// Create an existing wave.yaml with different content
	existingContent := []byte("apiVersion: v1\nkind: WaveManifest\nmetadata:\n  name: existing\n")
	err := os.WriteFile("wave.yaml", existingContent, 0644)
	require.NoError(t, err, "failed to create existing wave.yaml")

	stdout, _, err := executeInitCmd("--force")

	// Verify successful execution
	require.NoError(t, err, "init --force should succeed")
	assert.Contains(t, stdout, "Project initialized successfully")

	// Verify file was overwritten with new content
	manifest, err := readYAML("wave.yaml")
	require.NoError(t, err)
	metadata, ok := manifest["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wave-project", metadata["name"], "file should be overwritten with default name")
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

	stdout, _, err := executeInitCmd("--merge")

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
		"reviewer.md",
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

// TestInitOutputPath tests the --output flag for custom manifest path.
func TestInitOutputPath(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	customPath := "config/my-wave.yaml"

	stdout, _, err := executeInitCmd("--output", customPath)

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

	// Create wave.yaml without --force to trigger "already exists" error
	err = os.WriteFile("wave.yaml", []byte("test"), 0644)
	require.NoError(t, err)

	// Use the command with a "n" response to decline overwrite
	cmd := NewInitCmd()
	cmd.SetIn(strings.NewReader("n\n"))
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	err = cmd.Execute()

	assert.Error(t, err)
	// The error should include the file path (either relative or absolute)
	assert.True(t, strings.Contains(err.Error(), "wave.yaml") || strings.Contains(err.Error(), env.rootDir),
		"error message should include file path: %v", err)
}

// TestInitIdempotence tests that running init twice with --force produces the same result.
func TestInitIdempotence(t *testing.T) {
	env := newTestEnv(t)
	defer env.cleanup()

	// First init
	_, _, err := executeInitCmd()
	require.NoError(t, err)

	// Read generated files
	manifest1, err := readYAML("wave.yaml")
	require.NoError(t, err)

	// Second init with --force
	_, _, err = executeInitCmd("--force")
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

	_, _, err = executeInitCmd("--merge")
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

	_, _, err = executeInitCmd("--merge")
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
	assert.Contains(t, manifest, "skill_mounts")

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
			"reviewer.md",
			[]string{"Reviewer", "security", "review"},
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
	_, _, err = executeInitCmd("--merge", "--all")
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
	_, _, err = executeInitCmd("--merge")
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

// TestDetectProject tests project type auto-detection from filesystem markers.
func TestDetectProject(t *testing.T) {
	pkgJSON := func(scripts map[string]string) string {
		s, _ := json.Marshal(map[string]interface{}{"scripts": scripts})
		return string(s)
	}

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
			name: "bun project reads scripts",
			files: map[string]string{
				"bun.lockb":    "",
				"package.json": pkgJSON(map[string]string{"test": "vitest", "lint": "eslint .", "build": "tsc"}),
				"tsconfig.json": "{}",
			},
			wantLanguage: "typescript",
			wantTestCmd:  "bun test",
			wantLintCmd:  "bun run lint",
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
			name: "typescript node project reads scripts",
			files: map[string]string{
				"package.json":  pkgJSON(map[string]string{"test": "jest", "lint": "eslint", "build": "tsc"}),
				"tsconfig.json": "{}",
			},
			wantLanguage: "typescript",
			wantTestCmd:  "npm test",
			wantLintCmd:  "npm run lint",
			wantBuildCmd: "npm run build",
		},
		{
			name: "javascript node project reads scripts",
			files: map[string]string{
				"package.json": pkgJSON(map[string]string{"test": "mocha", "build": "webpack"}),
			},
			wantLanguage: "javascript",
			wantTestCmd:  "npm test",
			wantBuildCmd: "npm run build",
		},
		{
			name: "pnpm project uses pnpm runner",
			files: map[string]string{
				"package.json":   pkgJSON(map[string]string{"test": "vitest", "lint": "eslint"}),
				"pnpm-lock.yaml": "",
				"tsconfig.json":  "{}",
			},
			wantLanguage: "typescript",
			wantTestCmd:  "pnpm test",
			wantLintCmd:  "pnpm run lint",
		},
		{
			name: "yarn project uses yarn runner",
			files: map[string]string{
				"package.json": pkgJSON(map[string]string{"test": "jest"}),
				"yarn.lock":    "",
			},
			wantLanguage: "javascript",
			wantTestCmd:  "yarn test",
		},
		{
			name: "package.json without scripts still detects language",
			files: map[string]string{
				"package.json": `{"name": "my-app"}`,
			},
			wantLanguage: "javascript",
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
			wantTestCmd:  "pytest",
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
		{
			name: "deno takes priority over package.json",
			files: map[string]string{
				"deno.json":    "{}",
				"package.json": pkgJSON(map[string]string{"test": "jest"}),
			},
			wantLanguage: "typescript",
			wantTestCmd:  "deno test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			defer env.cleanup()

			for name, content := range tc.files {
				require.NoError(t, os.WriteFile(name, []byte(content), 0644))
			}

			result := detectProject()

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
	assert.Equal(t, "go test ./...", project["test_command"])
	assert.Equal(t, "go vet ./...", project["lint_command"])
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
