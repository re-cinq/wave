package onboarding

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRunWizard_NonInteractive(t *testing.T) {
	dir := t.TempDir()
	waveDir := filepath.Join(dir, ".wave")
	outputPath := filepath.Join(dir, "wave.yaml")

	// Create .wave/pipelines for pipeline discovery
	pipelinesDir := filepath.Join(waveDir, "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	// Create a sample Go project marker
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	cfg := WizardConfig{
		WaveDir:     waveDir,
		Interactive: false,
		Adapter:     "claude",
		Workspace:   ".wave/workspaces",
		OutputPath:  outputPath,
	}

	result, err := RunWizard(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify results
	assert.Equal(t, "claude", result.Adapter)
	assert.Equal(t, "go test ./...", result.TestCommand)
	assert.Equal(t, "go", result.Language)
	assert.Equal(t, "opus", result.Model) // default model for claude

	// Verify manifest was written
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var m map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &m))
	assert.Equal(t, "v1", m["apiVersion"])
	assert.Equal(t, "WaveManifest", m["kind"])

	// Verify project section
	project, ok := m["project"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "go", project["language"])

	// Verify onboarding marked complete
	assert.True(t, IsOnboarded(waveDir))
}

func TestRunWizard_Reconfigure(t *testing.T) {
	dir := t.TempDir()
	waveDir := filepath.Join(dir, ".wave")
	outputPath := filepath.Join(dir, "wave.yaml")

	require.NoError(t, os.MkdirAll(filepath.Join(waveDir, "pipelines"), 0755))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	existing := &manifest.Manifest{
		Project: &manifest.Project{
			Language:    "go",
			TestCommand: "make test",
			LintCommand: "golangci-lint run",
		},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude"},
		},
	}

	cfg := WizardConfig{
		WaveDir:     waveDir,
		Interactive: false,
		Reconfigure: true,
		Existing:    existing,
		Adapter:     "claude",
		Workspace:   ".wave/workspaces",
		OutputPath:  outputPath,
	}

	result, err := RunWizard(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should use existing values from reconfigure
	assert.Equal(t, "make test", result.TestCommand)
	assert.Equal(t, "golangci-lint run", result.LintCommand)
	assert.Equal(t, "claude", result.Adapter)
}

func TestRunWizard_MarksOnboarded(t *testing.T) {
	dir := t.TempDir()
	waveDir := filepath.Join(dir, ".wave")
	outputPath := filepath.Join(dir, "wave.yaml")

	require.NoError(t, os.MkdirAll(filepath.Join(waveDir, "pipelines"), 0755))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origDir)

	assert.False(t, IsOnboarded(waveDir))

	cfg := WizardConfig{
		WaveDir:     waveDir,
		Interactive: false,
		Adapter:     "claude",
		Workspace:   ".wave/workspaces",
		OutputPath:  outputPath,
	}

	_, err := RunWizard(cfg)
	require.NoError(t, err)

	assert.True(t, IsOnboarded(waveDir))
}
