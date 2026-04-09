package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadValidManifest(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-wave
  description: Test manifest
adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
  opencode:
    binary: opencode
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: prompts/dev.txt
    temperature: 0.7
runtime:
  workspace_root: ./workspace
  max_concurrent_workers: 4
  default_timeout_minutes: 10
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	promptPath := filepath.Join(tmpDir, "prompts", "dev.txt")
	if err := os.MkdirAll(filepath.Dir(promptPath), 0755); err != nil {
		t.Fatalf("Failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(promptPath, []byte("You are a developer."), 0644); err != nil {
		t.Fatalf("Failed to write prompt file: %v", err)
	}

	manifest, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	if manifest.APIVersion != "v1" {
		t.Errorf("Expected APIVersion 'v1', got '%s'", manifest.APIVersion)
	}
	if manifest.Kind != "WaveManifest" {
		t.Errorf("Expected Kind 'WaveManifest', got '%s'", manifest.Kind)
	}
	if manifest.Metadata.Name != "test-wave" {
		t.Errorf("Expected name 'test-wave', got '%s'", manifest.Metadata.Name)
	}
	if len(manifest.Adapters) != 2 {
		t.Errorf("Expected 2 adapters, got %d", len(manifest.Adapters))
	}
	if len(manifest.Personas) != 1 {
		t.Errorf("Expected 1 persona, got %d", len(manifest.Personas))
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	invalidContent := `apiVersion: v1
metadata:
  name: test
  - invalid yaml
`
	if err := os.WriteFile(manifestPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	_, err := Load(manifestPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestValidateMissingMetadataName(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  description: No name provided
runtime:
  workspace_root: ./workspace
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	_, err := Load(manifestPath)
	if err == nil {
		t.Error("Expected validation error for missing metadata.name")
	}
}

func TestValidateMissingAdapter(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test
personas:
  navigator:
    adapter: missing-adapter
    system_prompt_file: prompts/dev.txt
runtime:
  workspace_root: ./workspace
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	promptPath := filepath.Join(tmpDir, "prompts", "dev.txt")
	if err := os.MkdirAll(filepath.Dir(promptPath), 0755); err != nil {
		t.Fatalf("Failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(promptPath, []byte("You are a developer."), 0644); err != nil {
		t.Fatalf("Failed to write prompt file: %v", err)
	}

	_, err := Load(manifestPath)
	if err == nil {
		t.Error("Expected validation error for missing adapter reference")
	}
}

func TestValidateMissingSystemPromptFile(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: nonexistent.txt
runtime:
  workspace_root: ./workspace
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	_, err := Load(manifestPath)
	if err == nil {
		t.Error("Expected validation error for missing system prompt file")
	}
}

func TestValidateMissingWorkspaceRoot(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test
runtime: {}
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	_, err := Load(manifestPath)
	if err == nil {
		t.Error("Expected validation error for missing workspace_root")
	}
}

func TestValidateEmptyAdapterBinary(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test
adapters:
  claude:
    binary: ""
    mode: headless
runtime:
  workspace_root: ./workspace
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	_, err := Load(manifestPath)
	if err == nil {
		t.Error("Expected validation error for empty adapter binary")
	}
}

func TestValidateEmptyPersonaAdapter(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: ""
    system_prompt_file: prompts/dev.txt
runtime:
  workspace_root: ./workspace
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	promptPath := filepath.Join(tmpDir, "prompts", "dev.txt")
	if err := os.MkdirAll(filepath.Dir(promptPath), 0755); err != nil {
		t.Fatalf("Failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(promptPath, []byte("You are a developer."), 0644); err != nil {
		t.Fatalf("Failed to write prompt file: %v", err)
	}

	_, err := Load(manifestPath)
	if err == nil {
		t.Error("Expected validation error for empty persona adapter")
	}
}

func TestManifestGetAdapter(t *testing.T) {
	m := &Manifest{
		Adapters: map[string]Adapter{
			"claude":   {Binary: "claude"},
			"opencode": {Binary: "opencode"},
		},
	}

	adapter := m.GetAdapter("claude")
	if adapter == nil {
		t.Fatal("Expected to find claude adapter")
	}
	if adapter.Binary != "claude" {
		t.Errorf("Expected adapter binary 'claude', got '%s'", adapter.Binary)
	}

	notFound := m.GetAdapter("nonexistent")
	if notFound != nil {
		t.Error("Expected nil for nonexistent adapter")
	}
}

func TestManifestSandboxConfig(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-sandbox
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: prompts/nav.md
    permissions:
      allowed_tools: ["Read", "Glob", "Grep"]
      deny: ["Write(*)", "Edit(*)"]
    sandbox:
      allowed_domains:
        - api.anthropic.com
  implementer:
    adapter: claude
    system_prompt_file: prompts/impl.md
    permissions:
      allowed_tools: ["Read", "Write", "Edit", "Bash", "Glob", "Grep"]
    sandbox:
      allowed_domains:
        - api.anthropic.com
        - github.com
        - "*.github.com"
        - proxy.golang.org
runtime:
  workspace_root: ./workspace
  sandbox:
    enabled: true
    default_allowed_domains:
      - api.anthropic.com
      - github.com
    env_passthrough:
      - ANTHROPIC_API_KEY
      - GH_TOKEN
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Create prompt files
	for _, f := range []string{"prompts/nav.md", "prompts/impl.md"} {
		p := filepath.Join(tmpDir, f)
		_ = os.MkdirAll(filepath.Dir(p), 0755)
		_ = os.WriteFile(p, []byte("# Persona"), 0644)
	}

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	// Runtime sandbox
	if !m.Runtime.Sandbox.Enabled {
		t.Error("expected runtime.sandbox.enabled = true")
	}
	if len(m.Runtime.Sandbox.DefaultAllowedDomains) != 2 {
		t.Errorf("expected 2 default domains, got %d", len(m.Runtime.Sandbox.DefaultAllowedDomains))
	}
	if len(m.Runtime.Sandbox.EnvPassthrough) != 2 {
		t.Errorf("expected 2 env passthrough vars, got %d", len(m.Runtime.Sandbox.EnvPassthrough))
	}
	if m.Runtime.Sandbox.EnvPassthrough[0] != "ANTHROPIC_API_KEY" {
		t.Errorf("expected ANTHROPIC_API_KEY, got %s", m.Runtime.Sandbox.EnvPassthrough[0])
	}

	// Navigator persona sandbox
	nav := m.GetPersona("navigator")
	if nav == nil {
		t.Fatal("navigator persona not found")
	}
	if nav.Sandbox == nil {
		t.Fatal("navigator sandbox config not found")
	}
	if len(nav.Sandbox.AllowedDomains) != 1 {
		t.Errorf("expected 1 navigator domain, got %d", len(nav.Sandbox.AllowedDomains))
	}
	if nav.Sandbox.AllowedDomains[0] != "api.anthropic.com" {
		t.Errorf("expected api.anthropic.com, got %s", nav.Sandbox.AllowedDomains[0])
	}

	// Navigator permissions
	if len(nav.Permissions.Deny) != 2 {
		t.Errorf("expected 2 navigator deny rules, got %d", len(nav.Permissions.Deny))
	}

	// Implementer persona sandbox
	impl := m.GetPersona("implementer")
	if impl == nil {
		t.Fatal("implementer persona not found")
	}
	if impl.Sandbox == nil {
		t.Fatal("implementer sandbox config not found")
	}
	if len(impl.Sandbox.AllowedDomains) != 4 {
		t.Errorf("expected 4 implementer domains, got %d", len(impl.Sandbox.AllowedDomains))
	}
}

func TestManifestNoSandbox(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-no-sandbox
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: prompts/nav.md
runtime:
  workspace_root: ./workspace
`
	_ = os.WriteFile(manifestPath, []byte(manifestContent), 0644)
	_ = os.MkdirAll(filepath.Join(tmpDir, "prompts"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "prompts", "nav.md"), []byte("# Nav"), 0644)

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	// Runtime sandbox should have zero values
	if m.Runtime.Sandbox.Enabled {
		t.Error("expected sandbox.enabled = false by default")
	}
	if len(m.Runtime.Sandbox.DefaultAllowedDomains) != 0 {
		t.Error("expected no default domains")
	}

	// Persona sandbox should be nil
	nav := m.GetPersona("navigator")
	if nav.Sandbox != nil {
		t.Error("expected nil persona sandbox when not configured")
	}
}

func TestManifestGetPersona(t *testing.T) {
	m := &Manifest{
		Personas: map[string]Persona{
			"navigator": {Adapter: "claude"},
			"craftsman": {Adapter: "opencode"},
		},
	}

	persona := m.GetPersona("navigator")
	if persona == nil {
		t.Fatal("Expected to find navigator persona")
	}
	if persona.Adapter != "claude" {
		t.Errorf("Expected persona adapter 'claude', got '%s'", persona.Adapter)
	}

	notFound := m.GetPersona("nonexistent")
	if notFound != nil {
		t.Error("Expected nil for nonexistent persona")
	}
}

func TestManifestSkillsYAMLRoundTrip(t *testing.T) {
	t.Run("skills populated", func(t *testing.T) {
		yamlStr := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test
skills:
  - speckit
  - golang
runtime:
  workspace_root: ./workspace
`
		var m Manifest
		if err := yaml.Unmarshal([]byte(yamlStr), &m); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		want := []string{"speckit", "golang"}
		if !reflect.DeepEqual(m.Skills, want) {
			t.Errorf("Skills = %v, want %v", m.Skills, want)
		}

		// Marshal back to YAML
		out, err := yaml.Marshal(&m)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		// Unmarshal again and verify identical
		var m2 Manifest
		if err := yaml.Unmarshal(out, &m2); err != nil {
			t.Fatalf("round-trip unmarshal error: %v", err)
		}

		if !reflect.DeepEqual(m2.Skills, want) {
			t.Errorf("round-trip Skills = %v, want %v", m2.Skills, want)
		}
	})

	t.Run("no skills key", func(t *testing.T) {
		yamlStr := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test
runtime:
  workspace_root: ./workspace
`
		var m Manifest
		if err := yaml.Unmarshal([]byte(yamlStr), &m); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if m.Skills != nil {
			t.Errorf("Skills should be nil when key is absent, got %v", m.Skills)
		}
	})
}

func TestPersonaSkillsYAMLRoundTrip(t *testing.T) {
	t.Run("persona with skills", func(t *testing.T) {
		yamlStr := `adapter: claude
system_prompt_file: prompts/nav.md
skills:
  - speckit
`
		var p Persona
		if err := yaml.Unmarshal([]byte(yamlStr), &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		want := []string{"speckit"}
		if !reflect.DeepEqual(p.Skills, want) {
			t.Errorf("Skills = %v, want %v", p.Skills, want)
		}

		// Round-trip
		out, err := yaml.Marshal(&p)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		var p2 Persona
		if err := yaml.Unmarshal(out, &p2); err != nil {
			t.Fatalf("round-trip unmarshal error: %v", err)
		}

		if !reflect.DeepEqual(p2.Skills, want) {
			t.Errorf("round-trip Skills = %v, want %v", p2.Skills, want)
		}
	})

	t.Run("persona with multiple skills", func(t *testing.T) {
		yamlStr := `adapter: claude
system_prompt_file: prompts/impl.md
skills:
  - speckit
  - golang
  - testing
`
		var p Persona
		if err := yaml.Unmarshal([]byte(yamlStr), &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		want := []string{"speckit", "golang", "testing"}
		if !reflect.DeepEqual(p.Skills, want) {
			t.Errorf("Skills = %v, want %v", p.Skills, want)
		}

		// Round-trip
		out, err := yaml.Marshal(&p)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		var p2 Persona
		if err := yaml.Unmarshal(out, &p2); err != nil {
			t.Fatalf("round-trip unmarshal error: %v", err)
		}

		if !reflect.DeepEqual(p2.Skills, want) {
			t.Errorf("round-trip Skills = %v, want %v", p2.Skills, want)
		}
	})

	t.Run("persona without skills key", func(t *testing.T) {
		yamlStr := `adapter: claude
system_prompt_file: prompts/nav.md
`
		var p Persona
		if err := yaml.Unmarshal([]byte(yamlStr), &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if p.Skills != nil {
			t.Errorf("Skills should be nil when key is absent, got %v", p.Skills)
		}
	})
}

func TestSkillsNullAndEmptyParsing(t *testing.T) {
	t.Run("manifest skills null", func(t *testing.T) {
		yamlStr := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test
skills: null
runtime:
  workspace_root: ./workspace
`
		var m Manifest
		if err := yaml.Unmarshal([]byte(yamlStr), &m); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if m.Skills != nil {
			t.Errorf("skills: null should parse to nil, got %v", m.Skills)
		}
	})

	t.Run("manifest skills empty list", func(t *testing.T) {
		yamlStr := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test
skills: []
runtime:
  workspace_root: ./workspace
`
		var m Manifest
		if err := yaml.Unmarshal([]byte(yamlStr), &m); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if len(m.Skills) != 0 {
			t.Errorf("skills: [] should parse to empty slice, got %v", m.Skills)
		}
	})

	t.Run("manifest skills missing key", func(t *testing.T) {
		yamlStr := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test
runtime:
  workspace_root: ./workspace
`
		var m Manifest
		if err := yaml.Unmarshal([]byte(yamlStr), &m); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if m.Skills != nil {
			t.Errorf("missing skills key should parse to nil, got %v", m.Skills)
		}
	})

	t.Run("persona skills null", func(t *testing.T) {
		yamlStr := `adapter: claude
system_prompt_file: prompts/nav.md
skills: null
`
		var p Persona
		if err := yaml.Unmarshal([]byte(yamlStr), &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if p.Skills != nil {
			t.Errorf("skills: null should parse to nil, got %v", p.Skills)
		}
	})

	t.Run("persona skills empty list", func(t *testing.T) {
		yamlStr := `adapter: claude
system_prompt_file: prompts/nav.md
skills: []
`
		var p Persona
		if err := yaml.Unmarshal([]byte(yamlStr), &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if len(p.Skills) != 0 {
			t.Errorf("skills: [] should parse to empty slice, got %v", p.Skills)
		}
	})

	t.Run("persona skills missing key", func(t *testing.T) {
		yamlStr := `adapter: claude
system_prompt_file: prompts/nav.md
`
		var p Persona
		if err := yaml.Unmarshal([]byte(yamlStr), &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if p.Skills != nil {
			t.Errorf("missing skills key should parse to nil, got %v", p.Skills)
		}
	})

	// Pipeline scope: use a generic map to verify the "skills" YAML key behavior
	// since the Pipeline struct lives in the pipeline package (tested separately there).
	for _, tt := range []struct {
		name      string
		yamlStr   string
		wantNil   bool
		wantEmpty bool
	}{
		{
			name:    "pipeline skills null",
			yamlStr: "kind: WavePipeline\nmetadata:\n  name: test\nskills: null\ninput:\n  source: cli\nsteps: []\n",
			wantNil: true,
		},
		{
			name:      "pipeline skills empty list",
			yamlStr:   "kind: WavePipeline\nmetadata:\n  name: test\nskills: []\ninput:\n  source: cli\nsteps: []\n",
			wantEmpty: true,
		},
		{
			name:    "pipeline skills missing key",
			yamlStr: "kind: WavePipeline\nmetadata:\n  name: test\ninput:\n  source: cli\nsteps: []\n",
			wantNil: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var raw map[string]interface{}
			if err := yaml.Unmarshal([]byte(tt.yamlStr), &raw); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			skills, exists := raw["skills"]
			if tt.wantNil {
				if exists && skills != nil {
					t.Errorf("expected skills to be nil/absent, got %v", skills)
				}
			}
			if tt.wantEmpty {
				if !exists {
					t.Fatal("expected skills key to exist")
				}
				sl, ok := skills.([]interface{})
				if !ok {
					t.Fatalf("expected skills to be a slice, got %T", skills)
				}
				if len(sl) != 0 {
					t.Errorf("expected empty skills slice, got %v", sl)
				}
			}
		})
	}
}

// mockSkillStore implements manifest.SkillStore for testing.
type mockSkillStore struct {
	skills map[string]bool
}

func (m *mockSkillStore) Read(name string) (interface{}, error) {
	if m.skills[name] {
		return nil, nil
	}
	return nil, fmt.Errorf("not found")
}

// writeTestManifest creates a valid manifest YAML in tmpDir with optional skills and persona skills.
// Returns the path to the manifest file.
func writeTestManifest(t *testing.T, tmpDir string, globalSkills []string, personaSkills map[string][]string) string {
	t.Helper()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	// Build personas YAML
	var personasYAML string
	promptFiles := []string{"prompts/dev.txt"}
	if len(personaSkills) > 0 {
		personasYAML = "personas:\n"
		for pName, skills := range personaSkills {
			promptFile := fmt.Sprintf("prompts/%s.txt", pName)
			promptFiles = append(promptFiles, promptFile)
			personasYAML += fmt.Sprintf("  %s:\n    adapter: claude\n    system_prompt_file: %s\n", pName, promptFile)
			if len(skills) > 0 {
				personasYAML += "    skills:\n"
				for _, s := range skills {
					personasYAML += fmt.Sprintf("      - %s\n", s)
				}
			}
		}
	} else {
		personasYAML = "personas:\n  navigator:\n    adapter: claude\n    system_prompt_file: prompts/dev.txt\n"
	}

	// Build global skills YAML
	skillsYAML := ""
	if len(globalSkills) > 0 {
		skillsYAML = "skills:\n"
		for _, s := range globalSkills {
			skillsYAML += fmt.Sprintf("  - %s\n", s)
		}
	}

	content := fmt.Sprintf(`apiVersion: v1
kind: WaveManifest
metadata:
  name: test-skills
adapters:
  claude:
    binary: claude
    mode: headless
%s%sruntime:
  workspace_root: ./workspace
`, skillsYAML, personasYAML)

	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	// Create all prompt files
	for _, f := range promptFiles {
		p := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatalf("Failed to create prompt dir: %v", err)
		}
		if err := os.WriteFile(p, []byte("# Persona prompt"), 0644); err != nil {
			t.Fatalf("Failed to write prompt file: %v", err)
		}
	}

	return manifestPath
}

func TestLoadWithSkillStore(t *testing.T) {
	t.Run("valid global skills pass validation", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := writeTestManifest(t, tmpDir, []string{"speckit", "golang"}, nil)
		store := &mockSkillStore{skills: map[string]bool{"speckit": true, "golang": true}}

		m, err := LoadWithSkillStore(manifestPath, store)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(m.Skills) != 2 {
			t.Errorf("expected 2 global skills, got %d", len(m.Skills))
		}
	})

	t.Run("invalid global skill produces error", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := writeTestManifest(t, tmpDir, []string{"nonexistent-skill"}, nil)
		store := &mockSkillStore{skills: map[string]bool{"speckit": true}}

		_, err := LoadWithSkillStore(manifestPath, store)
		if err == nil {
			t.Fatal("expected error for nonexistent skill, got nil")
		}
		if !strings.Contains(err.Error(), "nonexistent-skill") {
			t.Errorf("expected error to mention skill name, got: %v", err)
		}
		if !strings.Contains(err.Error(), "global") {
			t.Errorf("expected error to mention global scope, got: %v", err)
		}
	})

	t.Run("persona with invalid skill produces scoped error", func(t *testing.T) {
		tmpDir := t.TempDir()
		personaSkills := map[string][]string{
			"craftsman": {"missing-skill"},
		}
		manifestPath := writeTestManifest(t, tmpDir, nil, personaSkills)
		store := &mockSkillStore{skills: map[string]bool{"speckit": true}}

		_, err := LoadWithSkillStore(manifestPath, store)
		if err == nil {
			t.Fatal("expected error for missing persona skill, got nil")
		}
		if !strings.Contains(err.Error(), "persona:craftsman") {
			t.Errorf("expected error to contain 'persona:craftsman' scope, got: %v", err)
		}
		if !strings.Contains(err.Error(), "missing-skill") {
			t.Errorf("expected error to mention skill name, got: %v", err)
		}
	})

	t.Run("errors aggregated across global and persona scopes", func(t *testing.T) {
		tmpDir := t.TempDir()
		personaSkills := map[string][]string{
			"navigator": {"persona-bad-skill"},
		}
		manifestPath := writeTestManifest(t, tmpDir, []string{"global-bad-skill"}, personaSkills)
		store := &mockSkillStore{skills: map[string]bool{}}

		_, err := LoadWithSkillStore(manifestPath, store)
		if err == nil {
			t.Fatal("expected error for multiple invalid skills, got nil")
		}
		errMsg := err.Error()
		if !strings.Contains(errMsg, "global-bad-skill") {
			t.Errorf("expected error to mention global-bad-skill, got: %v", err)
		}
		if !strings.Contains(errMsg, "persona-bad-skill") {
			t.Errorf("expected error to mention persona-bad-skill, got: %v", err)
		}
	})

	t.Run("absent skills field produces no error", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := writeTestManifest(t, tmpDir, nil, nil)
		store := &mockSkillStore{skills: map[string]bool{}}

		m, err := LoadWithSkillStore(manifestPath, store)
		if err != nil {
			t.Fatalf("expected no error for absent skills, got: %v", err)
		}
		if m == nil {
			t.Fatal("expected manifest to be returned")
		}
	})

	t.Run("nil store skips existence checks", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := writeTestManifest(t, tmpDir, []string{"any-skill"}, nil)

		m, err := LoadWithSkillStore(manifestPath, nil)
		if err != nil {
			t.Fatalf("expected no error with nil store, got: %v", err)
		}
		if m == nil {
			t.Fatal("expected manifest to be returned")
		}
		if len(m.Skills) != 1 {
			t.Errorf("expected 1 skill in manifest, got %d", len(m.Skills))
		}
	})
}

// makeTestPersonaEnv creates a temp directory with a system prompt file and returns
// the adapters map, personas map, basePath, and filePath suitable for calling
// validatePersonasListWithFile directly.
func makeTestPersonaEnv(t *testing.T, tokenScopes []string) (map[string]Persona, map[string]Adapter, string, string) {
	t.Helper()
	tmpDir := t.TempDir()

	promptFile := "prompts/persona.md"
	promptPath := filepath.Join(tmpDir, promptFile)
	if err := os.MkdirAll(filepath.Dir(promptPath), 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(promptPath, []byte("# Persona"), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	adapters := map[string]Adapter{
		"claude": {Binary: "claude", Mode: "headless"},
	}
	personas := map[string]Persona{
		"navigator": {
			Adapter:          "claude",
			SystemPromptFile: promptFile,
			TokenScopes:      tokenScopes,
		},
	}

	filePath := filepath.Join(tmpDir, "wave.yaml")
	return personas, adapters, tmpDir, filePath
}

func TestValidatePersonaTokenScopes(t *testing.T) {
	t.Run("valid token_scopes are accepted", func(t *testing.T) {
		validCases := []struct {
			name   string
			scopes []string
		}{
			{"single read scope", []string{"issues:read"}},
			{"single write scope", []string{"pulls:write"}},
			{"single admin scope", []string{"repos:admin"}},
			{"multiple valid scopes", []string{"issues:read", "pulls:write", "repos:admin"}},
			{"all canonical resources", []string{"issues:read", "pulls:read", "repos:read", "actions:read", "packages:read"}},
			{"scope with env var override", []string{"issues:write@GH_TOKEN"}},
			{"write and admin scopes", []string{"actions:write", "packages:admin"}},
		}

		for _, tc := range validCases {
			t.Run(tc.name, func(t *testing.T) {
				personas, adapters, basePath, filePath := makeTestPersonaEnv(t, tc.scopes)
				errs := validatePersonasListWithFile(personas, adapters, basePath, filePath)
				if len(errs) != 0 {
					t.Errorf("expected no errors for scopes %v, got: %v", tc.scopes, errs)
				}
			})
		}
	})

	t.Run("invalid token_scopes are rejected", func(t *testing.T) {
		invalidCases := []struct {
			name            string
			scopes          []string
			expectedInError string
		}{
			{
				name:            "bare word without colon",
				scopes:          []string{"invalid"},
				expectedInError: "invalid",
			},
			{
				name:            "leading colon missing resource",
				scopes:          []string{":read"},
				expectedInError: ":read",
			},
			{
				name:            "unknown permission level",
				scopes:          []string{"issues:unknown"},
				expectedInError: "issues:unknown",
			},
			{
				name:            "trailing colon missing permission",
				scopes:          []string{"issues:"},
				expectedInError: "issues:",
			},
			{
				name:            "empty string scope",
				scopes:          []string{""},
				expectedInError: "token_scopes",
			},
			{
				name:            "completely invalid format",
				scopes:          []string{"no-colon-at-all"},
				expectedInError: "no-colon-at-all",
			},
		}

		for _, tc := range invalidCases {
			t.Run(tc.name, func(t *testing.T) {
				personas, adapters, basePath, filePath := makeTestPersonaEnv(t, tc.scopes)
				errs := validatePersonasListWithFile(personas, adapters, basePath, filePath)
				if len(errs) == 0 {
					t.Errorf("expected validation error for scopes %v, got none", tc.scopes)
					return
				}
				errMsg := errs[0].Error()
				if !strings.Contains(errMsg, tc.expectedInError) {
					t.Errorf("expected error to contain %q, got: %s", tc.expectedInError, errMsg)
				}
			})
		}
	})

	t.Run("missing token_scopes field is accepted (backward compat)", func(t *testing.T) {
		nilScopesCases := []struct {
			name   string
			scopes []string
		}{
			{"nil scopes", nil},
			{"empty scopes slice", []string{}},
		}

		for _, tc := range nilScopesCases {
			t.Run(tc.name, func(t *testing.T) {
				personas, adapters, basePath, filePath := makeTestPersonaEnv(t, tc.scopes)
				errs := validatePersonasListWithFile(personas, adapters, basePath, filePath)
				if len(errs) != 0 {
					t.Errorf("expected no errors for absent token_scopes, got: %v", errs)
				}
			})
		}
	})

	t.Run("error references persona name and field", func(t *testing.T) {
		personas, adapters, basePath, filePath := makeTestPersonaEnv(t, []string{"bad-scope"})
		errs := validatePersonasListWithFile(personas, adapters, basePath, filePath)
		if len(errs) == 0 {
			t.Fatal("expected at least one validation error")
		}
		errMsg := errs[0].Error()
		if !strings.Contains(errMsg, "token_scopes") {
			t.Errorf("expected error to reference 'token_scopes' field, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "navigator") {
			t.Errorf("expected error to reference persona name 'navigator', got: %s", errMsg)
		}
	})

	t.Run("multiple invalid scopes produce multiple errors", func(t *testing.T) {
		personas, adapters, basePath, filePath := makeTestPersonaEnv(t, []string{"bad-one", "also:bad"})
		errs := validatePersonasListWithFile(personas, adapters, basePath, filePath)
		if len(errs) < 2 {
			t.Errorf("expected at least 2 errors for 2 invalid scopes, got %d: %v", len(errs), errs)
		}
	})
}

func TestManifestSandboxBackendFields(t *testing.T) {
	t.Run("parse backend and docker_image fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := filepath.Join(tmpDir, "wave.yaml")

		content := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-docker-sandbox
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: prompts/nav.md
runtime:
  workspace_root: ./workspace
  sandbox:
    enabled: true
    backend: docker
    docker_image: custom:latest
    default_allowed_domains:
      - api.anthropic.com
    env_passthrough:
      - ANTHROPIC_API_KEY
`
		if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write manifest: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(tmpDir, "prompts"), 0755); err != nil {
			t.Fatalf("Failed to create prompts dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "prompts", "nav.md"), []byte("# Nav"), 0644); err != nil {
			t.Fatalf("Failed to write persona prompt: %v", err)
		}

		m, err := Load(manifestPath)
		if err != nil {
			t.Fatalf("Failed to load manifest: %v", err)
		}

		if m.Runtime.Sandbox.Backend != "docker" {
			t.Errorf("expected backend 'docker', got %q", m.Runtime.Sandbox.Backend)
		}
		if m.Runtime.Sandbox.DockerImage != "custom:latest" {
			t.Errorf("expected docker_image 'custom:latest', got %q", m.Runtime.Sandbox.DockerImage)
		}
		if got := m.Runtime.Sandbox.ResolveBackend(); got != "docker" {
			t.Errorf("ResolveBackend() = %q, want 'docker'", got)
		}
		if got := m.Runtime.Sandbox.GetDockerImage(); got != "custom:latest" {
			t.Errorf("GetDockerImage() = %q, want 'custom:latest'", got)
		}
	})

	t.Run("backward compat without backend field", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := filepath.Join(tmpDir, "wave.yaml")

		content := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-legacy-sandbox
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: prompts/nav.md
runtime:
  workspace_root: ./workspace
  sandbox:
    enabled: true
    default_allowed_domains:
      - api.anthropic.com
`
		if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write manifest: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(tmpDir, "prompts"), 0755); err != nil {
			t.Fatalf("Failed to create prompts dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "prompts", "nav.md"), []byte("# Nav"), 0644); err != nil {
			t.Fatalf("Failed to write persona prompt: %v", err)
		}

		m, err := Load(manifestPath)
		if err != nil {
			t.Fatalf("Failed to load manifest: %v", err)
		}

		if m.Runtime.Sandbox.Backend != "" {
			t.Errorf("expected empty backend for legacy manifest, got %q", m.Runtime.Sandbox.Backend)
		}
		if m.Runtime.Sandbox.DockerImage != "" {
			t.Errorf("expected empty docker_image for legacy manifest, got %q", m.Runtime.Sandbox.DockerImage)
		}
		if got := m.Runtime.Sandbox.ResolveBackend(); got != "bubblewrap" {
			t.Errorf("ResolveBackend() = %q, want 'bubblewrap' for legacy enabled=true", got)
		}
		if got := m.Runtime.Sandbox.GetDockerImage(); got != "ubuntu:24.04" {
			t.Errorf("GetDockerImage() = %q, want 'ubuntu:24.04' default", got)
		}
	})

	t.Run("fields correctly populated from YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := filepath.Join(tmpDir, "wave.yaml")

		content := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-fields
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: prompts/nav.md
runtime:
  workspace_root: ./workspace
  sandbox:
    enabled: false
    backend: bubblewrap
    docker_image: debian:bookworm
    default_allowed_domains:
      - example.com
    env_passthrough:
      - HOME
`
		if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write manifest: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(tmpDir, "prompts"), 0755); err != nil {
			t.Fatalf("Failed to create prompts dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "prompts", "nav.md"), []byte("# Nav"), 0644); err != nil {
			t.Fatalf("Failed to write persona prompt: %v", err)
		}

		m, err := Load(manifestPath)
		if err != nil {
			t.Fatalf("Failed to load manifest: %v", err)
		}

		s := m.Runtime.Sandbox
		if s.Enabled != false {
			t.Error("expected enabled=false")
		}
		if s.Backend != "bubblewrap" {
			t.Errorf("expected backend 'bubblewrap', got %q", s.Backend)
		}
		if s.DockerImage != "debian:bookworm" {
			t.Errorf("expected docker_image 'debian:bookworm', got %q", s.DockerImage)
		}
		if len(s.DefaultAllowedDomains) != 1 || s.DefaultAllowedDomains[0] != "example.com" {
			t.Errorf("unexpected default_allowed_domains: %v", s.DefaultAllowedDomains)
		}
		if len(s.EnvPassthrough) != 1 || s.EnvPassthrough[0] != "HOME" {
			t.Errorf("unexpected env_passthrough: %v", s.EnvPassthrough)
		}
		// backend supersedes enabled
		if got := s.ResolveBackend(); got != "bubblewrap" {
			t.Errorf("ResolveBackend() = %q, want 'bubblewrap'", got)
		}
	})
}

func TestManifestServerConfig(t *testing.T) {
	t.Run("server section parsed correctly", func(t *testing.T) {
		yamlStr := `apiVersion: wave/v1alpha1
kind: Manifest
metadata:
  name: test
  description: test manifest
runtime:
  workspace_root: .wave/workspaces
server:
  bind: "0.0.0.0:9090"
  max_concurrent: 10
  auth:
    mode: jwt
    jwt_secret: "${WAVE_JWT_SECRET}"
  tls:
    enabled: true
    cert: "/path/to/cert.pem"
    key: "/path/to/key.pem"
`
		var m Manifest
		if err := yaml.Unmarshal([]byte(yamlStr), &m); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if m.Server == nil {
			t.Fatal("expected Server to be non-nil")
		}
		if m.Server.Bind != "0.0.0.0:9090" {
			t.Errorf("expected Bind '0.0.0.0:9090', got %q", m.Server.Bind)
		}
		if m.Server.MaxConcurrent != 10 {
			t.Errorf("expected MaxConcurrent 10, got %d", m.Server.MaxConcurrent)
		}
		if m.Server.Auth.Mode != "jwt" {
			t.Errorf("expected Auth.Mode 'jwt', got %q", m.Server.Auth.Mode)
		}
		if m.Server.Auth.JWTSecret != "${WAVE_JWT_SECRET}" {
			t.Errorf("expected Auth.JWTSecret '${WAVE_JWT_SECRET}', got %q", m.Server.Auth.JWTSecret)
		}
		if !m.Server.TLS.Enabled {
			t.Error("expected TLS.Enabled to be true")
		}
		if m.Server.TLS.Cert != "/path/to/cert.pem" {
			t.Errorf("expected TLS.Cert '/path/to/cert.pem', got %q", m.Server.TLS.Cert)
		}
		if m.Server.TLS.Key != "/path/to/key.pem" {
			t.Errorf("expected TLS.Key '/path/to/key.pem', got %q", m.Server.TLS.Key)
		}
	})

	t.Run("no server section yields nil", func(t *testing.T) {
		yamlStr := `apiVersion: wave/v1alpha1
kind: Manifest
metadata:
  name: test
  description: test manifest
runtime:
  workspace_root: .wave/workspaces
`
		var m Manifest
		if err := yaml.Unmarshal([]byte(yamlStr), &m); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if m.Server != nil {
			t.Errorf("expected Server to be nil when not configured, got %+v", m.Server)
		}
	})
}
