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
