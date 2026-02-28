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
		PersonaConfigs: map[string]manifest.Persona{
			"navigator": {
				Description: "Strategic planner",
				Temperature: 0.3,
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Glob", "Grep"},
					Deny:         []string{"Bash(*)"},
				},
			},
		},
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

	// Verify personas section exists with model at persona level (not adapter level)
	personas, ok := m["personas"].(map[string]interface{})
	require.True(t, ok, "manifest must contain personas section")
	nav, ok := personas["navigator"].(map[string]interface{})
	require.True(t, ok, "personas must contain navigator")
	assert.Equal(t, "opus", nav["model"], "model should be at persona level")
	assert.Equal(t, "claude", nav["adapter"])
	assert.Equal(t, ".wave/personas/navigator.md", nav["system_prompt_file"])

	// Verify model is NOT at adapter level
	adapters, ok := m["adapters"].(map[string]interface{})
	require.True(t, ok)
	adapterCfg, ok := adapters["claude"].(map[string]interface{})
	require.True(t, ok)
	_, hasModel := adapterCfg["model"]
	assert.False(t, hasModel, "model should NOT be at adapter level")

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

func TestBuildManifest_HasPersonas(t *testing.T) {
	cfg := WizardConfig{
		Workspace: ".wave/workspaces",
		PersonaConfigs: map[string]manifest.Persona{
			"craftsman": {
				Description: "Implementation specialist",
				Temperature: 0.2,
				Model:       "sonnet",
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Write", "Edit", "Bash"},
					Deny:         []string{},
				},
			},
		},
	}
	result := &WizardResult{
		Adapter: "claude",
		Model:   "opus",
	}

	m := buildManifest(cfg, result)

	// Verify personas section
	personas, ok := m["personas"].(map[string]interface{})
	require.True(t, ok)

	craftsman, ok := personas["craftsman"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "claude", craftsman["adapter"])
	assert.Equal(t, "opus", craftsman["model"], "wizard model overrides persona default")
	assert.Equal(t, ".wave/personas/craftsman.md", craftsman["system_prompt_file"])
	assert.Equal(t, "Implementation specialist", craftsman["description"])
	assert.Equal(t, 0.2, craftsman["temperature"])

	perms, ok := craftsman["permissions"].(map[string]interface{})
	require.True(t, ok)
	assert.NotNil(t, perms["allowed_tools"])
	assert.NotNil(t, perms["deny"])
}

func TestBuildManifest_PersonaFallbackModel(t *testing.T) {
	cfg := WizardConfig{
		Workspace: ".wave/workspaces",
		PersonaConfigs: map[string]manifest.Persona{
			"navigator": {
				Description: "Planner",
				Model:       "sonnet",
			},
		},
	}
	result := &WizardResult{
		Adapter: "claude",
		Model:   "", // no wizard model
	}

	m := buildManifest(cfg, result)

	personas := m["personas"].(map[string]interface{})
	nav := personas["navigator"].(map[string]interface{})
	assert.Equal(t, "sonnet", nav["model"], "should fall back to persona config model")
}

func TestBuildManifest_NoPersonas(t *testing.T) {
	cfg := WizardConfig{
		Workspace: ".wave/workspaces",
	}
	result := &WizardResult{
		Adapter: "claude",
	}

	m := buildManifest(cfg, result)
	_, ok := m["personas"]
	assert.False(t, ok, "should not have personas section when no persona configs")
}
