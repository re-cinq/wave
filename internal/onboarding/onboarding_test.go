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
	waveDir := filepath.Join(dir, ".agents")
	outputPath := filepath.Join(dir, "wave.yaml")

	// Create .agents/pipelines for pipeline discovery
	pipelinesDir := filepath.Join(waveDir, "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	// Create a sample Go project marker
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	cfg := WizardConfig{
		WaveDir:     waveDir,
		Interactive: false,
		Adapter:     "claude",
		Workspace:   ".agents/workspaces",
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
	assert.Equal(t, ".agents/personas/navigator.md", nav["system_prompt_file"])

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
	waveDir := filepath.Join(dir, ".agents")
	outputPath := filepath.Join(dir, "wave.yaml")

	require.NoError(t, os.MkdirAll(filepath.Join(waveDir, "pipelines"), 0755))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

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
		Workspace:   ".agents/workspaces",
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
	waveDir := filepath.Join(dir, ".agents")
	outputPath := filepath.Join(dir, "wave.yaml")

	require.NoError(t, os.MkdirAll(filepath.Join(waveDir, "pipelines"), 0755))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	assert.False(t, IsOnboarded(waveDir))

	cfg := WizardConfig{
		WaveDir:     waveDir,
		Interactive: false,
		Adapter:     "claude",
		Workspace:   ".agents/workspaces",
		OutputPath:  outputPath,
	}

	_, err := RunWizard(cfg)
	require.NoError(t, err)

	assert.True(t, IsOnboarded(waveDir))
}

func TestBuildManifest_HasPersonas(t *testing.T) {
	cfg := WizardConfig{
		Workspace: ".agents/workspaces",
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
	assert.Equal(t, ".agents/personas/craftsman.md", craftsman["system_prompt_file"])
	assert.Equal(t, "Implementation specialist", craftsman["description"])
	assert.Equal(t, 0.2, craftsman["temperature"])

	perms, ok := craftsman["permissions"].(map[string]interface{})
	require.True(t, ok)
	assert.NotNil(t, perms["allowed_tools"])
	assert.NotNil(t, perms["deny"])
}

func TestBuildManifest_PersonaFallbackModel(t *testing.T) {
	cfg := WizardConfig{
		Workspace: ".agents/workspaces",
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

func TestBuildManifest_WithSkills(t *testing.T) {
	cfg := WizardConfig{
		Workspace: ".agents/workspaces",
	}
	result := &WizardResult{
		Adapter: "claude",
		Skills:  []string{"golang", "spec-kit", "agentic-coding"},
	}

	m := buildManifest(cfg, result)

	skills, ok := m["skills"]
	require.True(t, ok, "manifest must contain skills key when skills are present")
	skillsList, ok := skills.([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"golang", "spec-kit", "agentic-coding"}, skillsList)
}

func TestBuildManifest_NoSkills(t *testing.T) {
	cfg := WizardConfig{
		Workspace: ".agents/workspaces",
	}
	result := &WizardResult{
		Adapter: "claude",
		Skills:  []string{},
	}

	m := buildManifest(cfg, result)

	_, ok := m["skills"]
	assert.False(t, ok, "manifest should not have skills key when skills list is empty")
}

func TestBuildManifest_NoPersonas(t *testing.T) {
	cfg := WizardConfig{
		Workspace: ".agents/workspaces",
	}
	result := &WizardResult{
		Adapter: "claude",
	}

	m := buildManifest(cfg, result)
	_, ok := m["personas"]
	assert.False(t, ok, "should not have personas section when no persona configs")
}

func TestBuildManifest_TokenScopes_WritePersona(t *testing.T) {
	cfg := WizardConfig{
		Workspace: ".agents/workspaces",
		PersonaConfigs: map[string]manifest.Persona{
			"craftsman": {
				Description: "Implementation specialist",
				Temperature: 0.2,
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

	personas := m["personas"].(map[string]interface{})
	craftsman := personas["craftsman"].(map[string]interface{})
	scopes, ok := craftsman["token_scopes"]
	require.True(t, ok, "write-capable persona must have token_scopes in generated manifest")
	assert.Equal(t, []string{"issues:read", "pulls:write", "repos:write"}, scopes)
}

func TestBuildManifest_TokenScopes_ReadOnlyPersona(t *testing.T) {
	cfg := WizardConfig{
		Workspace: ".agents/workspaces",
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
	result := &WizardResult{
		Adapter: "claude",
		Model:   "opus",
	}

	m := buildManifest(cfg, result)

	personas := m["personas"].(map[string]interface{})
	nav := personas["navigator"].(map[string]interface{})
	scopes, ok := nav["token_scopes"]
	require.True(t, ok, "read-only persona must have read token_scopes in generated manifest")
	assert.Equal(t, []string{"issues:read", "pulls:read"}, scopes)
}

func TestBuildManifest_TokenScopes_NoForgeTools(t *testing.T) {
	cfg := WizardConfig{
		Workspace: ".agents/workspaces",
		PersonaConfigs: map[string]manifest.Persona{
			"analyst": {
				Description: "Data analyst with no forge tools",
				Temperature: 0.1,
				Permissions: manifest.Permissions{
					AllowedTools: []string{},
					Deny:         []string{"Bash(*)", "Write(*)", "Edit(*)"},
				},
			},
		},
	}
	result := &WizardResult{
		Adapter: "claude",
		Model:   "opus",
	}

	m := buildManifest(cfg, result)

	personas := m["personas"].(map[string]interface{})
	analyst := personas["analyst"].(map[string]interface{})
	_, ok := analyst["token_scopes"]
	assert.False(t, ok, "persona with no forge-relevant tools must not have token_scopes")
}

func TestInferTokenScopes_BashTools(t *testing.T) {
	pcfg := manifest.Persona{
		Permissions: manifest.Permissions{
			AllowedTools: []string{"Read", "Write", "Edit", "Bash"},
		},
	}
	scopes := inferTokenScopes(pcfg)
	assert.Equal(t, []string{"issues:read", "pulls:write", "repos:write"}, scopes)
}

func TestInferTokenScopes_ReadOnlyTools(t *testing.T) {
	pcfg := manifest.Persona{
		Permissions: manifest.Permissions{
			AllowedTools: []string{"Read", "Glob", "Grep"},
		},
	}
	scopes := inferTokenScopes(pcfg)
	assert.Equal(t, []string{"issues:read", "pulls:read"}, scopes)
}

func TestInferTokenScopes_NoRelevantTools(t *testing.T) {
	pcfg := manifest.Persona{
		Permissions: manifest.Permissions{
			AllowedTools: []string{},
		},
	}
	scopes := inferTokenScopes(pcfg)
	assert.Nil(t, scopes)
}
