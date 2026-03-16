package adapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCodePrepareWorkspace_ModelResolution(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		wantProvider string
		wantModel    string
	}{
		{
			name:         "explicit prefix openai/gpt-4o",
			model:        "openai/gpt-4o",
			wantProvider: "openai",
			wantModel:    "gpt-4o",
		},
		{
			name:         "inferred prefix gpt-4o",
			model:        "gpt-4o",
			wantProvider: "openai",
			wantModel:    "gpt-4o",
		},
		{
			name:         "empty model uses defaults",
			model:        "",
			wantProvider: "anthropic",
			wantModel:    "claude-sonnet-4-20250514",
		},
		{
			name:         "multi-slash splits on first slash only",
			model:        "provider/org/model",
			wantProvider: "provider",
			wantModel:    "org/model",
		},
		{
			name:         "explicit google prefix",
			model:        "google/gemini-pro",
			wantProvider: "google",
			wantModel:    "gemini-pro",
		},
		{
			name:         "inferred anthropic from claude prefix",
			model:        "claude-sonnet-4-20250514",
			wantProvider: "anthropic",
			wantModel:    "claude-sonnet-4-20250514",
		},
		{
			name:         "unknown model without prefix defaults to anthropic",
			model:        "my-custom-model",
			wantProvider: "anthropic",
			wantModel:    "my-custom-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			a := NewOpenCodeAdapter()
			cfg := AdapterRunConfig{
				Model: tt.model,
			}

			if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
				t.Fatalf("prepareWorkspace returned unexpected error: %v", err)
			}

			configPath := filepath.Join(tmpDir, ".opencode", "config.json")
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("failed to read config.json: %v", err)
			}

			var config map[string]interface{}
			if err := json.Unmarshal(data, &config); err != nil {
				t.Fatalf("failed to unmarshal config.json: %v", err)
			}

			gotProvider, _ := config["provider"].(string)
			gotModel, _ := config["model"].(string)

			if gotProvider != tt.wantProvider {
				t.Errorf("provider = %q, want %q", gotProvider, tt.wantProvider)
			}
			if gotModel != tt.wantModel {
				t.Errorf("model = %q, want %q", gotModel, tt.wantModel)
			}
		})
	}
}

func TestOpenCodePrepareWorkspace_CreatesConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	a := NewOpenCodeAdapter()
	cfg := AdapterRunConfig{
		Model:       "openai/gpt-4o",
		Temperature: 0.7,
	}

	if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace returned unexpected error: %v", err)
	}

	configPath := filepath.Join(tmpDir, ".opencode", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("expected config.json to exist at %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config.json: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("config.json is not valid JSON: %v", err)
	}

	if _, ok := config["provider"]; !ok {
		t.Error("config.json missing required field: provider")
	}
	if _, ok := config["model"]; !ok {
		t.Error("config.json missing required field: model")
	}
	if _, ok := config["temperature"]; !ok {
		t.Error("config.json missing required field: temperature")
	}
}

func TestOpenCodePrepareWorkspace_SystemPromptWritesAgentsMd(t *testing.T) {
	tmpDir := t.TempDir()
	a := NewOpenCodeAdapter()
	cfg := AdapterRunConfig{
		SystemPrompt: "You are a helpful assistant.",
	}

	if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace returned unexpected error: %v", err)
	}

	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("expected AGENTS.md to exist: %v", err)
	}

	if string(data) != cfg.SystemPrompt {
		t.Errorf("AGENTS.md content = %q, want %q", string(data), cfg.SystemPrompt)
	}
}

func TestOpenCodePrepareWorkspace_CreatesSettingsDir(t *testing.T) {
	tmpDir := t.TempDir()
	a := NewOpenCodeAdapter()
	cfg := AdapterRunConfig{}

	if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace returned unexpected error: %v", err)
	}

	settingsDir := filepath.Join(tmpDir, ".opencode")
	info, err := os.Stat(settingsDir)
	if err != nil {
		t.Fatalf("expected .opencode directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected .opencode to be a directory")
	}
}
