package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// Persona Permission Model Tests
//
// These tests verify the permission model for personas defined in wave.yaml,
// focusing on artifact creation permissions and deny pattern enforcement.
// =============================================================================

// TestPersonaPermission_ToolPatternParsing tests that permission patterns are
// correctly parsed from the manifest format.
func TestPersonaPermission_ToolPatternParsing(t *testing.T) {
	m := createTestManifestWithPersonas(t)

	reviewer := m.GetPersona("reviewer")
	if reviewer == nil {
		t.Fatal("reviewer persona not found")
	}

	// Verify the pattern structure in reviewer permissions
	expectedAllowPatterns := map[string]bool{
		"Read":                         true,
		"Glob":                         true,
		"Grep":                         true,
		"Write(.agents/artifact.json)": true,
		"Write(.agents/artifacts/*)":   true,
		"Bash(go test*)":               true,
		"Bash(npm test*)":              true,
	}

	for _, tool := range reviewer.Permissions.AllowedTools {
		if !expectedAllowPatterns[tool] {
			// Not an error, just log unexpected patterns
			t.Logf("found additional allowed tool: %s", tool)
		}
	}

	// Check key patterns exist
	foundArtifactJson := false
	foundArtifactsDir := false
	for _, tool := range reviewer.Permissions.AllowedTools {
		if tool == "Write(.agents/artifact.json)" {
			foundArtifactJson = true
		}
		if tool == "Write(.agents/artifacts/*)" {
			foundArtifactsDir = true
		}
	}

	if !foundArtifactJson {
		t.Error("reviewer should have Write(artifact.json) pattern")
	}
	if !foundArtifactsDir {
		t.Error("reviewer should have Write(.agents/artifacts/*) pattern")
	}
}

// TestLoadWaveYAML_PersonaPermissions loads the actual wave.yaml and verifies
// persona permissions are correctly defined.
func TestLoadWaveYAML_PersonaPermissions(t *testing.T) {
	// Try to find wave.yaml relative to the test
	waveYAMLPath := findWaveYAML(t)
	if waveYAMLPath == "" {
		t.Skip("wave.yaml not found, skipping integration test")
		return
	}

	manifest, err := Load(waveYAMLPath)
	if err != nil {
		t.Fatalf("failed to load wave.yaml: %v", err)
	}

	// Verify expected personas exist
	expectedPersonas := []string{
		"implementer",
		"reviewer",
		"navigator",
		"auditor",
		"craftsman",
		"philosopher",
		"planner",
	}

	for _, name := range expectedPersonas {
		persona := manifest.GetPersona(name)
		if persona == nil {
			t.Errorf("expected persona '%s' not found in wave.yaml", name)
		}
	}

	// Verify implementer has Write permission. The wildcard "*" grant
	// (current default) implies Write among everything else.
	implementer := manifest.GetPersona("implementer")
	if implementer != nil {
		hasWrite := false
		for _, tool := range implementer.Permissions.AllowedTools {
			if tool == "*" || strings.HasPrefix(tool, "Write") {
				hasWrite = true
				break
			}
		}
		if !hasWrite {
			t.Error("implementer in wave.yaml should have Write permission")
		}
	}

	// Verify reviewer has Write permission. The wildcard "*" grant
	// (current default) covers the previously-narrow artifact paths.
	reviewer := manifest.GetPersona("reviewer")
	if reviewer != nil {
		hasArtifactWrite := false
		for _, tool := range reviewer.Permissions.AllowedTools {
			if tool == "*" || tool == "Write(.agents/artifact.json)" || tool == "Write(.agents/artifacts/*)" {
				hasArtifactWrite = true
				break
			}
		}
		if !hasArtifactWrite {
			t.Error("reviewer in wave.yaml should have Write permission (or wildcard)")
		}

		// Deny rules are intentionally empty — security is enforced via bubblewrap
		// sandbox and CLAUDE.md behavioral restrictions (issue #282).
		if len(reviewer.Permissions.Deny) != 0 {
			t.Errorf("reviewer in wave.yaml should have empty deny list (security via sandbox), got: %v", reviewer.Permissions.Deny)
		}
	}

	// Verify navigator has empty deny list — security via sandbox, not deny rules
	navigator := manifest.GetPersona("navigator")
	if navigator != nil {
		// Deny rules are intentionally empty — security is enforced via bubblewrap
		// sandbox and CLAUDE.md behavioral restrictions (issue #282).
		if len(navigator.Permissions.Deny) != 0 {
			t.Errorf("navigator in wave.yaml should have empty deny list (security via sandbox), got: %v", navigator.Permissions.Deny)
		}
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// createTestManifestWithPersonas creates a test manifest with personas matching
// the expected wave.yaml configuration.
func createTestManifestWithPersonas(t *testing.T) *Manifest {
	t.Helper()

	return &Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata: Metadata{
			Name:        "test-wave",
			Description: "Test manifest for permission tests",
		},
		Adapters: map[string]Adapter{
			"claude": {
				Binary: "claude",
				Mode:   "headless",
			},
		},
		Personas: map[string]Persona{
			"implementer": {
				Adapter:     "claude",
				Description: "Code execution and artifact generation for pipeline steps",
				Permissions: Permissions{
					AllowedTools: []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
					Deny:         []string{"Bash(rm -rf /*)", "Bash(sudo *)"},
				},
			},
			"reviewer": {
				Adapter:     "claude",
				Description: "Quality review, validation, and assessment",
				Permissions: Permissions{
					AllowedTools: []string{
						"Read",
						"Glob",
						"Grep",
						"Write(.agents/artifact.json)",
						"Write(.agents/artifacts/*)",
						"Bash(go test*)",
						"Bash(npm test*)",
					},
					Deny: []string{"Write(*.go)", "Write(*.ts)", "Write(*.py)", "Write(*.rs)", "Edit(*)", "Bash(rm *)", "Bash(git push*)", "Bash(git commit*)"},
				},
			},
			"navigator": {
				Adapter:     "claude",
				Description: "Read-only codebase exploration and analysis",
				Permissions: Permissions{
					AllowedTools: []string{
						"Read",
						"Glob",
						"Grep",
						"Bash(git log*)",
						"Bash(git status*)",
					},
					Deny: []string{"Write(*)", "Edit(*)", "Bash(git commit*)", "Bash(git push*)"},
				},
			},
			"auditor": {
				Adapter:     "claude",
				Description: "Security review and quality assurance",
				Permissions: Permissions{
					AllowedTools: []string{
						"Read",
						"Grep",
						"Bash(go vet*)",
						"Bash(npm audit*)",
					},
					Deny: []string{"Write(*)", "Edit(*)"},
				},
			},
			"craftsman": {
				Adapter:     "claude",
				Description: "Code implementation and testing",
				Permissions: Permissions{
					AllowedTools: []string{"Read", "Write", "Edit", "Bash"},
					Deny:         []string{"Bash(rm -rf /*)"},
				},
			},
			"philosopher": {
				Adapter:     "claude",
				Description: "Architecture design and specification",
				Permissions: Permissions{
					AllowedTools: []string{"Read", "Write(.agents/specs/*)"},
					Deny:         []string{"Bash(*)"},
				},
			},
			"planner": {
				Adapter:     "claude",
				Description: "Task breakdown and project planning",
				Permissions: Permissions{
					AllowedTools: []string{"Read", "Glob", "Grep"},
					Deny:         []string{"Write(*)", "Edit(*)", "Bash(*)"},
				},
			},
		},
		Runtime: Runtime{
			WorkspaceRoot: ".agents/workspaces",
		},
	}
}

// findWaveYAML attempts to find the wave.yaml file relative to the test.
func findWaveYAML(t *testing.T) string {
	t.Helper()

	// Try common relative paths from test execution location
	candidates := []string{
		"../../wave.yaml",
		"../../../wave.yaml",
		"wave.yaml",
	}

	// Also try from current working directory
	if cwd, err := os.Getwd(); err == nil {
		// Walk up from cwd looking for wave.yaml
		dir := cwd
		for i := 0; i < 5; i++ {
			candidate := filepath.Join(dir, "wave.yaml")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
			dir = filepath.Dir(dir)
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}
