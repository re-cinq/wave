package adapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeAllowedTools(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "bare tools unchanged",
			input: []string{"Read", "Write", "Edit", "Bash"},
			want:  []string{"Read", "Write", "Edit", "Bash"},
		},
		{
			name:  "Write scoped entries normalized to bare Write",
			input: []string{"Read", "Write(output/*)", "Write(artifact.json)"},
			want:  []string{"Read", "Write"},
		},
		{
			name:  "deduplicates after normalization",
			input: []string{"Write(output/*)", "Write(artifact.json)", "Read"},
			want:  []string{"Write", "Read"},
		},
		{
			name:  "preserves Bash scoped entries",
			input: []string{"Bash(go test*)", "Bash(git log*)", "Read"},
			want:  []string{"Bash(go test*)", "Bash(git log*)", "Read"},
		},
		{
			name:  "mixed scoped and bare",
			input: []string{"Read", "Glob", "Grep", "WebSearch", "Write(output/*)", "Write"},
			want:  []string{"Read", "Glob", "Grep", "WebSearch", "Write"},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  nil,
		},
		{
			name:  "bare Write preserved",
			input: []string{"Write"},
			want:  []string{"Write"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAllowedTools(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("normalizeAllowedTools(%v) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("normalizeAllowedTools(%v)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSettingsJSONFormat(t *testing.T) {
	adapter := NewClaudeAdapter()
	tmpDir := t.TempDir()

	cfg := AdapterRunConfig{
		Persona:       "test",
		WorkspacePath: tmpDir,
		Model:         "sonnet",
		Temperature:   0.5,
		AllowedTools:  []string{"Read", "Write(output/*)", "Glob"},
	}

	if err := adapter.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace failed: %v", err)
	}

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}

	// Verify it's valid JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v", err)
	}

	// Verify permissions.allow field exists (not allowed_tools)
	if _, ok := raw["allowed_tools"]; ok {
		t.Error("settings.json should not have 'allowed_tools' field")
	}

	perms, ok := raw["permissions"].(map[string]interface{})
	if !ok {
		t.Fatal("settings.json missing 'permissions' object")
	}

	allow, ok := perms["allow"].([]interface{})
	if !ok {
		t.Fatal("settings.json missing 'permissions.allow' array")
	}

	// Verify Write(output/*) was normalized to Write
	allowStrs := make([]string, len(allow))
	for i, v := range allow {
		allowStrs[i] = v.(string)
	}

	expected := []string{"Read", "Write", "Glob"}
	if len(allowStrs) != len(expected) {
		t.Fatalf("permissions.allow = %v, want %v", allowStrs, expected)
	}
	for i, want := range expected {
		if allowStrs[i] != want {
			t.Errorf("permissions.allow[%d] = %q, want %q", i, allowStrs[i], want)
		}
	}
}

func TestBuildArgsNormalizesAllowedTools(t *testing.T) {
	adapter := NewClaudeAdapter()
	cfg := AdapterRunConfig{
		AllowedTools: []string{"Read", "Write(output/*)", "Write(artifact.json)", "Glob"},
		Prompt:       "test",
	}

	args := adapter.buildArgs(cfg)

	// Find the --allowedTools value
	var allowedToolsArg string
	for i, arg := range args {
		if arg == "--allowedTools" && i+1 < len(args) {
			allowedToolsArg = args[i+1]
			break
		}
	}

	if allowedToolsArg == "" {
		t.Fatal("--allowedTools not found in args")
	}

	expected := "Read,Write,Glob"
	if allowedToolsArg != expected {
		t.Errorf("--allowedTools = %q, want %q", allowedToolsArg, expected)
	}
}
