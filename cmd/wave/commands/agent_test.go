package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentCmd_Structure(t *testing.T) {
	cmd := NewAgentCmd()
	assert.Equal(t, "agent", cmd.Use)

	subcommands := cmd.Commands()
	names := make([]string, len(subcommands))
	for i, c := range subcommands {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "inspect")
	assert.Contains(t, names, "export")
}

func TestBuildAgentListItems_Sorted(t *testing.T) {
	m := &manifest.Manifest{
		Personas: map[string]manifest.Persona{
			"zebra":     {Adapter: "claude-code", Model: "sonnet"},
			"alpha":     {Adapter: "claude-code", Model: "opus"},
			"navigator": {Adapter: "claude-code"},
		},
	}

	items := buildAgentListItems(m)
	require.Len(t, items, 3)
	assert.Equal(t, "alpha", items[0].Name)
	assert.Equal(t, "navigator", items[1].Name)
	assert.Equal(t, "zebra", items[2].Name)
	assert.Equal(t, "opus", items[0].Model)
}

func TestPersonaToAgentMarkdown_Basic(t *testing.T) {
	spec := adapter.PersonaSpec{
		Model:        "sonnet",
		AllowedTools: []string{"Read", "Glob", "Grep"},
	}

	md := adapter.PersonaToAgentMarkdown(spec, "# Base Protocol", "# Navigator\nYou are a navigator.", "", "")

	assert.True(t, strings.HasPrefix(md, "---\n"))
	assert.Contains(t, md, "model: sonnet")
	assert.Contains(t, md, "permissionMode: bypassPermissions")
	assert.Contains(t, md, "  - Read")
	assert.Contains(t, md, "  - Glob")
	assert.Contains(t, md, "  - Grep")
	assert.Contains(t, md, "# Base Protocol")
	assert.Contains(t, md, "# Navigator")
}

func TestPersonaToAgentMarkdown_NoDeny(t *testing.T) {
	spec := adapter.PersonaSpec{Model: "opus"}

	md := adapter.PersonaToAgentMarkdown(spec, "", "System prompt", "", "")

	assert.NotContains(t, md, "disallowedTools")
	assert.Contains(t, md, "model: opus")
	assert.Contains(t, md, "System prompt")
}

func TestPersonaToAgentMarkdown_WithDeny(t *testing.T) {
	spec := adapter.PersonaSpec{DenyTools: []string{"Bash(rm*)"}}

	md := adapter.PersonaToAgentMarkdown(spec, "", "", "", "")

	assert.Contains(t, md, "disallowedTools:")
	assert.Contains(t, md, "  - Bash(rm*)")
}

func TestPersonaToAgentMarkdown_WithRestrictions(t *testing.T) {
	spec := adapter.PersonaSpec{}

	md := adapter.PersonaToAgentMarkdown(spec, "", "", "## Contract\nMust output JSON", "## Restrictions\nNo network")

	assert.Contains(t, md, "## Contract")
	assert.Contains(t, md, "## Restrictions")
}

func TestLoadManifestForAgent_MissingFile(t *testing.T) {
	_, err := loadManifestForAgent("/nonexistent/wave.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest file not found")
}

func TestLoadManifestForAgent_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "wave.yaml")
	require.NoError(t, os.WriteFile(path, []byte("{{invalid"), 0644))

	_, err := loadManifestForAgent(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestLoadManifestForAgent_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "wave.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`apiVersion: v1
kind: manifest
metadata:
  name: test
personas:
  navigator:
    adapter: claude-code
    model: sonnet
`), 0644))

	m, err := loadManifestForAgent(path)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Contains(t, m.Personas, "navigator")
}

func TestAgentExportCmd_WritesFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create manifest
	manifestPath := filepath.Join(tmpDir, "wave.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`apiVersion: v1
kind: manifest
metadata:
  name: test
personas:
  navigator:
    adapter: claude-code
    description: "Test navigator"
    system_prompt_file: "personas/navigator.md"
    model: sonnet
    permissions:
      allowed_tools:
        - Read
`), 0644))

	// Create persona files
	personaDir := filepath.Join(tmpDir, "personas")
	require.NoError(t, os.MkdirAll(personaDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(personaDir, "navigator.md"), []byte("# Navigator\nYou navigate."), 0644))

	// Create .agents/personas/base-protocol.md
	wavePersonaDir := filepath.Join(tmpDir, ".agents", "personas")
	require.NoError(t, os.MkdirAll(wavePersonaDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(wavePersonaDir, "base-protocol.md"), []byte("# Base Protocol"), 0644))

	// Run from tmpDir so relative paths resolve
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	outputPath := filepath.Join(tmpDir, "nav.agent.md")
	cmd := NewAgentCmd()
	rootCmd := &cobra.Command{}
	rootCmd.PersistentFlags().String("manifest", manifestPath, "")
	rootCmd.AddCommand(cmd)
	rootCmd.SetArgs([]string{"agent", "export", "--export-path", outputPath, "navigator"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "model: sonnet")
	assert.Contains(t, string(data), "permissionMode: bypassPermissions")
	assert.Contains(t, string(data), "# Navigator")
}
