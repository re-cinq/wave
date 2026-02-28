package onboarding

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyStep(t *testing.T) {
	step := &DependencyStep{}
	assert.Equal(t, "Dependency Verification", step.Name())

	cfg := &WizardConfig{Interactive: false}
	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Data)

	deps, ok := result.Data["dependencies"].([]DependencyStatus)
	require.True(t, ok)
	assert.NotEmpty(t, deps)

	// Should check for gh and adapter binaries
	var foundGH bool
	for _, dep := range deps {
		if dep.Name == "GitHub CLI" {
			foundGH = true
			assert.NotEmpty(t, dep.InstallURL)
		}
	}
	assert.True(t, foundGH, "should check for GitHub CLI")
}

func TestTestConfigStep_GoProject(t *testing.T) {
	// Create a temp dir with go.mod to simulate a Go project
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644))

	// Change to temp dir for detection
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	step := &TestConfigStep{}
	assert.Equal(t, "Test Command Configuration", step.Name())

	cfg := &WizardConfig{Interactive: false}
	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "go test ./...", result.Data["test_command"])
	assert.Equal(t, "go vet ./...", result.Data["lint_command"])
	assert.Equal(t, "go build ./...", result.Data["build_command"])
	assert.Equal(t, "go", result.Data["language"])
}

func TestTestConfigStep_Reconfigure(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	step := &TestConfigStep{}
	cfg := &WizardConfig{
		Interactive: false,
		Reconfigure: true,
		Existing: &manifest.Manifest{
			Project: &manifest.Project{
				Language:    "go",
				TestCommand: "make test",
				LintCommand: "golangci-lint run",
			},
		},
	}

	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Reconfigure should use existing values
	assert.Equal(t, "make test", result.Data["test_command"])
	assert.Equal(t, "golangci-lint run", result.Data["lint_command"])
	assert.Equal(t, "go", result.Data["language"])
}

func TestTestConfigStep_UnknownProject(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	step := &TestConfigStep{}
	cfg := &WizardConfig{Interactive: false}

	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	// No project detected — empty strings
	assert.Equal(t, "", result.Data["test_command"])
	assert.Equal(t, "", result.Data["language"])
}

func TestAdapterConfigStep(t *testing.T) {
	step := &AdapterConfigStep{}
	assert.Equal(t, "Adapter Configuration", step.Name())

	cfg := &WizardConfig{Interactive: false, Adapter: "claude"}
	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "claude", result.Data["adapter"])
}

func TestAdapterConfigStep_DefaultAdapter(t *testing.T) {
	step := &AdapterConfigStep{}
	cfg := &WizardConfig{Interactive: false}

	result, err := step.Run(cfg)
	require.NoError(t, err)

	assert.Equal(t, "claude", result.Data["adapter"])
}

func TestAdapterConfigStep_Reconfigure(t *testing.T) {
	step := &AdapterConfigStep{}
	cfg := &WizardConfig{
		Interactive: false,
		Reconfigure: true,
		Existing: &manifest.Manifest{
			Adapters: map[string]manifest.Adapter{
				"opencode": {Binary: "opencode"},
			},
		},
	}

	result, err := step.Run(cfg)
	require.NoError(t, err)

	assert.Equal(t, "opencode", result.Data["adapter"])
}

func TestModelSelectionStep(t *testing.T) {
	step := &ModelSelectionStep{}
	assert.Equal(t, "Model Selection", step.Name())

	cfg := &WizardConfig{Interactive: false, Adapter: "claude"}
	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should default to first model for claude
	assert.Equal(t, "opus", result.Data["model"])
}

func TestModelSelectionStep_UnknownAdapter(t *testing.T) {
	step := &ModelSelectionStep{}
	cfg := &WizardConfig{Interactive: false, Adapter: "unknown"}

	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Skipped, "unknown adapter should not skip")
	assert.Equal(t, "", result.Data["model"], "unknown adapter non-interactive should return empty model")
}

func TestPipelineSelectionStep_NoPipelinesDir(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	step := &PipelineSelectionStep{}
	assert.Equal(t, "Pipeline Selection", step.Name())

	cfg := &WizardConfig{Interactive: false}
	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	pipelines, ok := result.Data["pipelines"].([]string)
	require.True(t, ok)
	assert.Empty(t, pipelines)
}

func TestPipelineSelectionStep_WithPipelines(t *testing.T) {
	dir := t.TempDir()
	pipelinesDir := filepath.Join(dir, ".wave", "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	// Create test pipeline files
	releasePipeline := `kind: Pipeline
metadata:
  name: test-release
  description: A release pipeline
  release: true
  category: stable
steps:
  - id: step1
    persona: craftsman
`
	expPipeline := `kind: Pipeline
metadata:
  name: test-experimental
  description: An experimental pipeline
  category: experimental
steps:
  - id: step1
    persona: craftsman
`
	require.NoError(t, os.WriteFile(filepath.Join(pipelinesDir, "test-release.yaml"), []byte(releasePipeline), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(pipelinesDir, "test-experimental.yaml"), []byte(expPipeline), 0644))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	step := &PipelineSelectionStep{}
	cfg := &WizardConfig{Interactive: false}

	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	pipelines, ok := result.Data["pipelines"].([]string)
	require.True(t, ok)
	// Non-interactive should select only release/stable pipelines
	assert.Contains(t, pipelines, "test-release")
	assert.NotContains(t, pipelines, "test-experimental")
}

func TestPipelineSelectionStep_AllFlag(t *testing.T) {
	dir := t.TempDir()
	pipelinesDir := filepath.Join(dir, ".wave", "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	releasePipeline := `kind: Pipeline
metadata:
  name: test-release
  release: true
steps:
  - id: step1
    persona: craftsman
`
	expPipeline := `kind: Pipeline
metadata:
  name: test-experimental
steps:
  - id: step1
    persona: craftsman
`
	require.NoError(t, os.WriteFile(filepath.Join(pipelinesDir, "test-release.yaml"), []byte(releasePipeline), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(pipelinesDir, "test-experimental.yaml"), []byte(expPipeline), 0644))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	step := &PipelineSelectionStep{}
	cfg := &WizardConfig{Interactive: false, All: true}

	result, err := step.Run(cfg)
	require.NoError(t, err)

	pipelines, ok := result.Data["pipelines"].([]string)
	require.True(t, ok)
	assert.Contains(t, pipelines, "test-release")
	assert.Contains(t, pipelines, "test-experimental")
}

func TestDetectProjectType(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected string // expected language
	}{
		{
			name:     "Go project",
			files:    map[string]string{"go.mod": "module test"},
			expected: "go",
		},
		{
			name:     "Rust project",
			files:    map[string]string{"Cargo.toml": "[package]"},
			expected: "rust",
		},
		{
			name:     "Python project",
			files:    map[string]string{"pyproject.toml": "[project]"},
			expected: "python",
		},
		{
			name:     "Node project",
			files:    map[string]string{"package.json": "{}"},
			expected: "javascript",
		},
		{
			name:     "Deno project",
			files:    map[string]string{"deno.json": "{}"},
			expected: "typescript",
		},
		{
			name:     "Unknown project",
			files:    map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
			}

			origDir, _ := os.Getwd()
			require.NoError(t, os.Chdir(dir))
			defer os.Chdir(origDir)

			result := detectProjectType()
			if tt.expected == "" {
				assert.Empty(t, result["language"])
			} else {
				assert.Equal(t, tt.expected, result["language"])
			}
		})
	}
}
