package adapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeminiAdapter_BuildArgs(t *testing.T) {
	a := &GeminiAdapter{geminiPath: "/usr/bin/gemini"}

	tests := []struct {
		name string
		cfg  AdapterRunConfig
		want []string
	}{
		{
			name: "prompt only",
			cfg:  AdapterRunConfig{Prompt: "hello world"},
			want: []string{"-p", "hello world"},
		},
		{
			name: "with model",
			cfg:  AdapterRunConfig{Prompt: "test", Model: "gemini-2.0-flash"},
			want: []string{"-p", "test", "--model", "gemini-2.0-flash"},
		},
		{
			name: "empty prompt",
			cfg:  AdapterRunConfig{},
			want: nil,
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

func TestGeminiAdapter_PrepareWorkspace(t *testing.T) {
	a := &GeminiAdapter{geminiPath: "/usr/bin/gemini"}
	dir := t.TempDir()

	err := a.prepareWorkspace(dir, AdapterRunConfig{
		SystemPrompt: "You are a test assistant",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Check GEMINI.md was written
	data, err := readTestFile(t, dir, "GEMINI.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "You are a test assistant" {
		t.Errorf("GEMINI.md content = %q, want %q", string(data), "You are a test assistant")
	}
}

func TestGeminiAdapter_PrepareWorkspaceNoPrompt(t *testing.T) {
	a := &GeminiAdapter{geminiPath: "/usr/bin/gemini"}
	dir := t.TempDir()

	err := a.prepareWorkspace(dir, AdapterRunConfig{})
	if err != nil {
		t.Fatal(err)
	}
	// No GEMINI.md should be written when SystemPrompt is empty
}

func TestParseGeminiStreamLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantOK   bool
		wantType string
	}{
		{
			name:     "text event",
			line:     `{"type":"text","content":"hello"}`,
			wantOK:   true,
			wantType: "text",
		},
		{
			name:     "tool_use event",
			line:     `{"type":"tool_use","name":"read_file"}`,
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
			evt, ok := parseGeminiStreamLine([]byte(tt.line))
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && evt.Type != tt.wantType {
				t.Errorf("type = %q, want %q", evt.Type, tt.wantType)
			}
		})
	}
}

func TestGeminiAdapter_PrepareWorkspace_DenyRules(t *testing.T) {
	a := &GeminiAdapter{geminiPath: "/usr/bin/gemini"}
	dir := t.TempDir()

	err := a.prepareWorkspace(dir, AdapterRunConfig{
		SystemPrompt: "You are a test assistant",
		DenyTools:    []string{"Bash(*)", "Write(*)"},
		AllowedDomains: []string{"api.example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "GEMINI.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "Denied Tools") {
		t.Error("GEMINI.md should contain denied tools section")
	}
	if !strings.Contains(content, "Bash(*)") {
		t.Error("GEMINI.md should contain denied tool Bash(*)")
	}
	if !strings.Contains(content, "Network Access") {
		t.Error("GEMINI.md should contain network access section")
	}
	if !strings.Contains(content, "api.example.com") {
		t.Error("GEMINI.md should contain allowed domain")
	}
}

// readTestFile is a helper for reading files in test directories.
func readTestFile(t *testing.T, dir, name string) ([]byte, error) {
	t.Helper()
	return os.ReadFile(filepath.Join(dir, name))
}
