package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// setupTestManifest writes a minimal wave.yaml with the given personas.
func setupTestManifest(t *testing.T, dir string, personas []string) {
	t.Helper()

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
  description: Test project for do command
adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
personas:
`
	for _, p := range personas {
		manifestContent += `  ` + p + `:
    adapter: claude
    system_prompt_file: personas/` + p + `.md
    temperature: 0.7
    permissions:
      allowed_tools:
        - Read
        - Write
      deny: []
`
	}

	manifestContent += `runtime:
  workspace_root: .agents/workspaces
  default_timeout_minutes: 30
`

	// Create personas directory and files
	personasDir := filepath.Join(dir, "personas")
	require.NoError(t, os.MkdirAll(personasDir, 0755))

	for _, p := range personas {
		promptContent := "You are a " + p + " persona for testing."
		require.NoError(t, os.WriteFile(filepath.Join(personasDir, p+".md"), []byte(promptContent), 0644))
	}

	manifestPath := filepath.Join(dir, "wave.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0644))
}
