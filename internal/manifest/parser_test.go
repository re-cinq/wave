package manifest

import (
	"os"
	"path/filepath"
	"testing"
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
skill_mounts:
  - path: ./skills
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
		t.Error("Expected to find claude adapter")
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
  craftsman:
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
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte("# Persona"), 0644)
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

	// Craftsman persona sandbox
	impl := m.GetPersona("craftsman")
	if impl == nil {
		t.Fatal("craftsman persona not found")
	}
	if impl.Sandbox == nil {
		t.Fatal("craftsman sandbox config not found")
	}
	if len(impl.Sandbox.AllowedDomains) != 4 {
		t.Errorf("expected 4 craftsman domains, got %d", len(impl.Sandbox.AllowedDomains))
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
	os.WriteFile(manifestPath, []byte(manifestContent), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "prompts"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "prompts", "nav.md"), []byte("# Nav"), 0644)

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

func TestManifestSkillsConfig(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-skills
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
skills:
  speckit:
    install: "uv tool install specify-cli"
    init: "specify init"
    check: "specify --version"
  bmad:
    install: "npx bmad-method install --yes"
    check: "ls .claude/commands/bmad.*.md"
    commands_glob: ".claude/commands/bmad.*.md"
`
	os.WriteFile(manifestPath, []byte(manifestContent), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "prompts"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "prompts", "nav.md"), []byte("# Nav"), 0644)

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Failed to load manifest with skills: %v", err)
	}

	if len(m.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(m.Skills))
	}

	speckit, ok := m.Skills["speckit"]
	if !ok {
		t.Fatal("speckit skill not found")
	}
	if speckit.Install != "uv tool install specify-cli" {
		t.Errorf("unexpected speckit install: %s", speckit.Install)
	}
	if speckit.Init != "specify init" {
		t.Errorf("unexpected speckit init: %s", speckit.Init)
	}
	if speckit.Check != "specify --version" {
		t.Errorf("unexpected speckit check: %s", speckit.Check)
	}

	bmad, ok := m.Skills["bmad"]
	if !ok {
		t.Fatal("bmad skill not found")
	}
	if bmad.CommandsGlob != ".claude/commands/bmad.*.md" {
		t.Errorf("unexpected bmad commands_glob: %s", bmad.CommandsGlob)
	}
}

func TestManifestSkillsMissingCheck(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-skills-invalid
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
skills:
  speckit:
    install: "uv tool install specify-cli"
`
	os.WriteFile(manifestPath, []byte(manifestContent), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "prompts"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "prompts", "nav.md"), []byte("# Nav"), 0644)

	_, err := Load(manifestPath)
	if err == nil {
		t.Error("expected validation error for skill missing check command")
	}
}

func TestManifestSkillsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-no-skills
runtime:
  workspace_root: ./workspace
`
	os.WriteFile(manifestPath, []byte(manifestContent), 0644)

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Failed to load manifest without skills: %v", err)
	}

	if m.Skills != nil && len(m.Skills) != 0 {
		t.Errorf("expected nil or empty skills, got %d", len(m.Skills))
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
		t.Error("Expected to find navigator persona")
	}
	if persona.Adapter != "claude" {
		t.Errorf("Expected persona adapter 'claude', got '%s'", persona.Adapter)
	}

	notFound := m.GetPersona("nonexistent")
	if notFound != nil {
		t.Error("Expected nil for nonexistent persona")
	}
}
