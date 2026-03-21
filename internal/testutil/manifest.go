package testutil

import "github.com/recinq/wave/internal/manifest"

// CreateTestManifest creates a manifest for testing with navigator and craftsman personas.
func CreateTestManifest(workspaceRoot string) *manifest.Manifest {
	return &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test-project"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:          "claude",
				SystemPromptFile: "",
				Temperature:      0.1,
			},
			"craftsman": {
				Adapter:          "claude",
				SystemPromptFile: "",
				Temperature:      0.7,
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     workspaceRoot,
			DefaultTimeoutMin: 5,
		},
	}
}
