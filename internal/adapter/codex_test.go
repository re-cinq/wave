package adapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexAdapter_BuildArgs(t *testing.T) {
	a := &CodexAdapter{codexPath: "/usr/bin/codex"}

	tests := []struct {
		name string
		cfg  AdapterRunConfig
		want []string
	}{
		{
			name: "prompt only",
			cfg:  AdapterRunConfig{Prompt: "hello world"},
			want: []string{"hello world", "--quiet"},
		},
		{
			name: "with model",
			cfg:  AdapterRunConfig{Prompt: "test", Model: "o3"},
			want: []string{"test", "--model", "o3", "--quiet"},
		},
		{
			name: "empty prompt",
			cfg:  AdapterRunConfig{},
			want: []string{"--quiet"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := a.buildArgs(tt.cfg)
			if len(got) != len(tt.want) {
				t.Fatalf("buildArgs() returned %d args, want %d: %v vs %v", len(got), len(tt.want), got, tt.want)
			}
			for i, arg := range got {
				if arg != tt.want[i] {
					t.Errorf("arg[%d] = %q, want %q", i, arg, tt.want[i])
				}
			}
		})
	}
}

func TestCodexAdapter_PrepareWorkspace(t *testing.T) {
	a := &CodexAdapter{codexPath: "/usr/bin/codex"}
	dir := t.TempDir()

	err := a.prepareWorkspace(dir, AdapterRunConfig{
		SystemPrompt: "You are a test assistant",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Check AGENTS.md was written
	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "You are a test assistant" {
		t.Errorf("AGENTS.md content = %q, want %q", string(data), "You are a test assistant")
	}
}

func TestCodexAdapter_PrepareWorkspaceNoPrompt(t *testing.T) {
	a := &CodexAdapter{codexPath: "/usr/bin/codex"}
	dir := t.TempDir()

	err := a.prepareWorkspace(dir, AdapterRunConfig{})
	if err != nil {
		t.Fatal(err)
	}
	// No AGENTS.md should be written when SystemPrompt is empty
}

func TestCodexAdapter_PrepareWorkspace_DenyRules(t *testing.T) {
	a := &CodexAdapter{codexPath: "/usr/bin/codex"}
	dir := t.TempDir()

	err := a.prepareWorkspace(dir, AdapterRunConfig{
		SystemPrompt: "You are a test assistant",
		DenyTools:    []string{"Bash(*)", "Write(*)"},
		AllowedDomains: []string{"api.example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "Denied Tools") {
		t.Error("AGENTS.md should contain denied tools section")
	}
	if !strings.Contains(content, "Bash(*)") {
		t.Error("AGENTS.md should contain denied tool Bash(*)")
	}
	if !strings.Contains(content, "Network Access") {
		t.Error("AGENTS.md should contain network access section")
	}
	if !strings.Contains(content, "api.example.com") {
		t.Error("AGENTS.md should contain allowed domain")
	}
}

func TestParseCodexStreamLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantOK   bool
		wantType string
	}{
		{
			name:     "message event",
			line:     `{"type":"message","content":"hello"}`,
			wantOK:   true,
			wantType: "text",
		},
		{
			name:     "function_call event",
			line:     `{"type":"function_call","name":"read_file"}`,
			wantOK:   true,
			wantType: "tool_use",
		},
		{
			name:   "unknown event type",
			line:   `{"type":"unknown"}`,
			wantOK: false,
		},
		{
			name:   "empty line",
			line:   "",
			wantOK: false,
		},
		{
			name:   "malformed json",
			line:   "not json",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt, ok := parseCodexStreamLine([]byte(tt.line))
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && evt.Type != tt.wantType {
				t.Errorf("type = %q, want %q", evt.Type, tt.wantType)
			}
		})
	}
}
