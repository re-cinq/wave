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
kind: Manifest
metadata:
  name: test-wave
  description: Test manifest
adapters:
  - binary: claude
    mode: agent
    outputFormat: json
  - binary: opencode
    mode: agent
personas:
  - adapter: claude
    systemPromptFile: prompts/dev.txt
    temperature: 0.7
runtime:
  workspaceRoot: ./workspace
  maxConcurrentWorkers: 4
  defaultTimeoutMin: 10
skillMounts:
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
	if manifest.Kind != "Manifest" {
		t.Errorf("Expected Kind 'Manifest', got '%s'", manifest.Kind)
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
kind: Manifest
metadata:
  description: No name provided
runtime:
  workspaceRoot: ./workspace
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
kind: Manifest
metadata:
  name: test
personas:
  - adapter: missing-adapter
    systemPromptFile: prompts/dev.txt
runtime:
  workspaceRoot: ./workspace
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
kind: Manifest
metadata:
  name: test
adapters:
  - binary: claude
    mode: agent
personas:
  - adapter: claude
    systemPromptFile: nonexistent.txt
runtime:
  workspaceRoot: ./workspace
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
kind: Manifest
metadata:
  name: test
runtime: {}
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	_, err := Load(manifestPath)
	if err == nil {
		t.Error("Expected validation error for missing workspaceRoot")
	}
}

func TestValidateEmptyAdapterBinary(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	manifestContent := `apiVersion: v1
kind: Manifest
metadata:
  name: test
adapters:
  - binary: ""
    mode: agent
runtime:
  workspaceRoot: ./workspace
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
kind: Manifest
metadata:
  name: test
adapters:
  - binary: claude
    mode: agent
personas:
  - adapter: ""
    systemPromptFile: prompts/dev.txt
runtime:
  workspaceRoot: ./workspace
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
		Adapters: []Adapter{
			{Binary: "claude"},
			{Binary: "opencode"},
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
		Personas: []Persona{
			{Adapter: "claude"},
			{Adapter: "opencode"},
		},
	}

	persona := m.GetPersona("claude")
	if persona == nil {
		t.Error("Expected to find claude persona")
	}
	if persona.Adapter != "claude" {
		t.Errorf("Expected persona adapter 'claude', got '%s'", persona.Adapter)
	}

	notFound := m.GetPersona("nonexistent")
	if notFound != nil {
		t.Error("Expected nil for nonexistent persona")
	}
}

func TestPersonaGetSystemPromptPath(t *testing.T) {
	p := &Persona{SystemPromptFile: "prompts/dev.txt"}

	absPath := p.GetSystemPromptPath("/workspace")
	if absPath != "/workspace/prompts/dev.txt" {
		t.Errorf("Expected '/workspace/prompts/dev.txt', got '%s'", absPath)
	}

	p2 := &Persona{SystemPromptFile: "/absolute/path.txt"}
	absPath2 := p2.GetSystemPromptPath("/workspace")
	if absPath2 != "/absolute/path.txt" {
		t.Errorf("Expected '/absolute/path.txt', got '%s'", absPath2)
	}
}

func TestRuntimeGetDefaultTimeout(t *testing.T) {
	r := &Runtime{DefaultTimeoutMin: 10}
	timeout := r.GetDefaultTimeout()
	if timeout.Minutes() != 10 {
		t.Errorf("Expected 10 minutes, got %v", timeout)
	}

	r2 := &Runtime{}
	timeout2 := r2.GetDefaultTimeout()
	if timeout2.Minutes() != 5 {
		t.Errorf("Expected default 5 minutes, got %v", timeout2)
	}
}
