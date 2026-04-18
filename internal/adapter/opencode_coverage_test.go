package adapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// --- ResolveAdapter tests ---

func TestResolveAdapter_AllCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
	}{
		{"claude", "claude", "*adapter.ClaudeAdapter"},
		{"opencode", "opencode", "*adapter.OpenCodeAdapter"},
		{"codex", "codex", "*adapter.CodexAdapter"},
		{"gemini", "gemini", "*adapter.GeminiAdapter"},
		{"browser", "browser", "*adapter.BrowserAdapter"},
		{"default", "unknown-adapter", "*adapter.ProcessGroupRunner"},
		{"empty string", "", "*adapter.ProcessGroupRunner"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveAdapter(tt.input)
			gotType := reflect.TypeOf(got).String()
			if gotType != tt.wantType {
				t.Errorf("ResolveAdapter(%q) = %s, want %s", tt.input, gotType, tt.wantType)
			}
		})
	}
}

func TestResolveAdapter_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input    string
		wantType string
	}{
		{"Claude", "*adapter.ClaudeAdapter"},
		{"CLAUDE", "*adapter.ClaudeAdapter"},
		{"OpenCode", "*adapter.OpenCodeAdapter"},
		{"OPENCODE", "*adapter.OpenCodeAdapter"},
		{"CODEX", "*adapter.CodexAdapter"},
		{"Gemini", "*adapter.GeminiAdapter"},
		{"BROWSER", "*adapter.BrowserAdapter"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ResolveAdapter(tt.input)
			gotType := reflect.TypeOf(got).String()
			if gotType != tt.wantType {
				t.Errorf("ResolveAdapter(%q) = %s, want %s", tt.input, gotType, tt.wantType)
			}
		})
	}
}

// --- parseOutput tests ---

func TestParseOutput_EmptyInput(t *testing.T) {
	a := NewOpenCodeAdapter()
	result := a.parseOutput([]byte{})
	// tokens = len(data)/4 = 0
	if result.Tokens != 0 {
		t.Errorf("Tokens = %d, want 0", result.Tokens)
	}
	if result.ResultContent != "" {
		t.Errorf("ResultContent = %q, want empty", result.ResultContent)
	}
}

func TestParseOutput_WhitespaceOnly(t *testing.T) {
	a := NewOpenCodeAdapter()
	result := a.parseOutput([]byte("  \n  \n  "))
	// len("  \n  \n  ") = 8, /4 = 2
	if result.Tokens != 2 {
		t.Errorf("Tokens = %d, want 2 (len/4 fallback)", result.Tokens)
	}
}

func TestParseOutput_ZeroTokenFallback(t *testing.T) {
	a := NewOpenCodeAdapter()
	// result event with zero usage tokens and no step_finish
	data := []byte(`{"type":"result","subtype":"success","result":"done","usage":{"input_tokens":0,"output_tokens":0}}`)
	result := a.parseOutput(data)
	// tokens should fall back to len(data)/4
	expected := len(data) / 4
	if result.Tokens != expected {
		t.Errorf("Tokens = %d, want %d (len/4 fallback)", result.Tokens, expected)
	}
	if result.ResultContent != "done" {
		t.Errorf("ResultContent = %q, want %q", result.ResultContent, "done")
	}
}

func TestParseOutput_ResultContentFallbackToContent(t *testing.T) {
	a := NewOpenCodeAdapter()
	data := []byte(`{"type":"result","subtype":"success","content":"fallback-content","usage":{"input_tokens":10,"output_tokens":5}}`)
	result := a.parseOutput(data)
	if result.ResultContent != "fallback-content" {
		t.Errorf("ResultContent = %q, want %q", result.ResultContent, "fallback-content")
	}
}

func TestParseOutput_ResultPreferredOverContent(t *testing.T) {
	a := NewOpenCodeAdapter()
	data := []byte(`{"type":"result","subtype":"success","result":"primary","content":"fallback","usage":{"input_tokens":10,"output_tokens":5}}`)
	result := a.parseOutput(data)
	if result.ResultContent != "primary" {
		t.Errorf("ResultContent = %q, want %q", result.ResultContent, "primary")
	}
}

func TestParseOutput_UsageTokensOverrideStepFinish(t *testing.T) {
	a := NewOpenCodeAdapter()
	// step_finish with total=50, then result with usage total=200 (> 50)
	data := []byte(`{"type":"step_finish","part":{"tokens":{"total":50,"input":30,"output":20}}}
{"type":"result","subtype":"success","content":"ok","usage":{"input_tokens":150,"output_tokens":50}}`)
	result := a.parseOutput(data)
	// usage tokens (150+50=200) > step_finish tokens (50), so should be 200
	if result.Tokens != 200 {
		t.Errorf("Tokens = %d, want 200", result.Tokens)
	}
}

func TestParseOutput_UsageTokensDoNotOverrideHigherStepFinish(t *testing.T) {
	a := NewOpenCodeAdapter()
	// step_finish with total=500, then result with usage total=30 (< 500)
	data := []byte(`{"type":"step_finish","part":{"tokens":{"total":500,"input":300,"output":200}}}
{"type":"result","subtype":"success","content":"ok","usage":{"input_tokens":20,"output_tokens":10}}`)
	result := a.parseOutput(data)
	if result.Tokens != 500 {
		t.Errorf("Tokens = %d, want 500", result.Tokens)
	}
}

func TestParseOutput_TextEventAsResultContent(t *testing.T) {
	a := NewOpenCodeAdapter()
	data := []byte(`{"type":"text","part":{"text":"hello from text"}}`)
	result := a.parseOutput(data)
	if result.ResultContent != "hello from text" {
		t.Errorf("ResultContent = %q, want %q", result.ResultContent, "hello from text")
	}
}

func TestParseOutput_TextEventMessageContentFallback(t *testing.T) {
	a := NewOpenCodeAdapter()
	data := []byte(`{"type":"text","part":{"text":""},"message":{"content":[{"type":"text","text":"from message"}]}}`)
	result := a.parseOutput(data)
	if result.ResultContent != "from message" {
		t.Errorf("ResultContent = %q, want %q", result.ResultContent, "from message")
	}
}

func TestParseOutput_TextEventSkippedWhenResultExists(t *testing.T) {
	a := NewOpenCodeAdapter()
	// result event first, then text event — text should be skipped because resultContent is already set
	data := []byte(`{"type":"result","subtype":"success","result":"primary-result","usage":{"input_tokens":10,"output_tokens":5}}
{"type":"text","part":{"text":"should be ignored"}}`)
	result := a.parseOutput(data)
	if result.ResultContent != "primary-result" {
		t.Errorf("ResultContent = %q, want %q", result.ResultContent, "primary-result")
	}
}

func TestParseOutput_MultipleResultEvents(t *testing.T) {
	a := NewOpenCodeAdapter()
	data := []byte(`{"type":"result","subtype":"error","result":"first","usage":{"input_tokens":10,"output_tokens":5}}
{"type":"result","subtype":"success","result":"second","usage":{"input_tokens":20,"output_tokens":10}}`)
	result := a.parseOutput(data)
	if result.ResultContent != "second" {
		t.Errorf("ResultContent = %q, want %q (later result should overwrite)", result.ResultContent, "second")
	}
	if result.Subtype != "success" {
		t.Errorf("Subtype = %q, want %q", result.Subtype, "success")
	}
}

// --- parseOpenCodeStreamLine additional coverage ---

func TestParseOpenCodeStreamLine_ToolCallType(t *testing.T) {
	line := []byte(`{"type":"tool_call","sessionID":"s1","part":{"type":"tool_call","tool":"Write","input":"test.go"}}`)
	evt, ok := parseOpenCodeStreamLine(line)
	if !ok {
		t.Fatal("expected ok=true for tool_call event")
	}
	if evt.Type != "tool_use" {
		t.Errorf("Type = %q, want %q", evt.Type, "tool_use")
	}
	if evt.ToolName != "Write" {
		t.Errorf("ToolName = %q, want %q", evt.ToolName, "Write")
	}
}

func TestParseOpenCodeStreamLine_ToolNameFallback(t *testing.T) {
	// Part.Tool is empty, Part.Name is set
	line := []byte(`{"type":"tool","sessionID":"s1","part":{"type":"tool","name":"Grep","input":"pattern"}}`)
	evt, ok := parseOpenCodeStreamLine(line)
	if !ok {
		t.Fatal("expected ok=true for tool event with Name fallback")
	}
	if evt.ToolName != "Grep" {
		t.Errorf("ToolName = %q, want %q", evt.ToolName, "Grep")
	}
}

func TestParseOpenCodeStreamLine_ToolInputTruncation(t *testing.T) {
	tests := []struct {
		name      string
		inputLen  int
		wantLen   int
		truncated bool
	}{
		{"exactly 100", 100, 100, false},
		{"101 truncated", 101, 100, true},
		{"200 truncated", 200, 100, true},
		{"short input", 10, 10, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.Repeat("x", tt.inputLen)
			line := []byte(`{"type":"tool","sessionID":"s1","part":{"type":"tool","tool":"Read","input":"` + input + `"}}`)
			evt, ok := parseOpenCodeStreamLine(line)
			if !ok {
				t.Fatal("expected ok=true")
			}
			if len(evt.ToolInput) != tt.wantLen {
				t.Errorf("ToolInput len = %d, want %d", len(evt.ToolInput), tt.wantLen)
			}
		})
	}
}

func TestParseOpenCodeStreamLine_TextTruncationBoundary(t *testing.T) {
	tests := []struct {
		name    string
		textLen int
		wantLen int
	}{
		{"exactly 200", 200, 200},
		{"201 truncated", 201, 200},
		{"199 not truncated", 199, 199},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := strings.Repeat("b", tt.textLen)
			line := []byte(`{"type":"text","sessionID":"s1","part":{"text":"` + text + `"}}`)
			evt, ok := parseOpenCodeStreamLine(line)
			if !ok {
				t.Fatal("expected ok=true")
			}
			if len(evt.Content) != tt.wantLen {
				t.Errorf("Content len = %d, want %d", len(evt.Content), tt.wantLen)
			}
		})
	}
}

func TestParseOpenCodeStreamLine_StepFinishTokenFallback(t *testing.T) {
	// Usage fields are zero, should fall back to Part.Tokens
	line := []byte(`{"type":"step_finish","sessionID":"s1","part":{"tokens":{"total":500,"input":300,"output":200}}}`)
	evt, ok := parseOpenCodeStreamLine(line)
	if !ok {
		t.Fatal("expected ok=true for step_finish event")
	}
	if evt.TokensIn != 300 {
		t.Errorf("TokensIn = %d, want 300", evt.TokensIn)
	}
	if evt.TokensOut != 200 {
		t.Errorf("TokensOut = %d, want 200", evt.TokensOut)
	}
}

func TestParseOpenCodeStreamLine_StepFinishUsagePreferred(t *testing.T) {
	// Usage fields are non-zero, should be used instead of Part.Tokens
	line := []byte(`{"type":"step_finish","sessionID":"s1","usage":{"input_tokens":100,"output_tokens":50},"part":{"tokens":{"total":500,"input":300,"output":200}}}`)
	evt, ok := parseOpenCodeStreamLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if evt.TokensIn != 100 {
		t.Errorf("TokensIn = %d, want 100 (from usage)", evt.TokensIn)
	}
	if evt.TokensOut != 50 {
		t.Errorf("TokensOut = %d, want 50 (from usage)", evt.TokensOut)
	}
}

func TestParseOpenCodeStreamLine_ResultContentFallback(t *testing.T) {
	// Result is empty, should fall back to Content
	line := []byte(`{"type":"result","usage":{"input_tokens":10,"output_tokens":5},"content":"fallback","subtype":"success"}`)
	evt, ok := parseOpenCodeStreamLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if evt.Content != "fallback" {
		t.Errorf("Content = %q, want %q", evt.Content, "fallback")
	}
}

func TestParseOpenCodeStreamLine_ResultPreferredOverContent(t *testing.T) {
	line := []byte(`{"type":"result","usage":{"input_tokens":10,"output_tokens":5},"result":"primary","content":"fallback","subtype":"success"}`)
	evt, ok := parseOpenCodeStreamLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if evt.Content != "primary" {
		t.Errorf("Content = %q, want %q", evt.Content, "primary")
	}
}

func TestParseOpenCodeStreamLine_BareToolJSON(t *testing.T) {
	// JSON with "tool" key but no "type" field — should trigger extractToolTarget
	input, _ := json.Marshal(map[string]string{"file_path": "/tmp/test.go"})
	line := []byte(`{"tool":"Read","input":` + string(input) + `}`)
	evt, ok := parseOpenCodeStreamLine(line)
	if !ok {
		t.Fatal("expected ok=true for bare tool JSON")
	}
	if evt.Type != "tool_use" {
		t.Errorf("Type = %q, want %q", evt.Type, "tool_use")
	}
	if evt.ToolName != "Read" {
		t.Errorf("ToolName = %q, want %q", evt.ToolName, "Read")
	}
}

func TestParseOpenCodeStreamLine_BareToolJSONNoTool(t *testing.T) {
	// JSON without "type" or "tool" key — should return false
	line := []byte(`{"foo":"bar","baz":123}`)
	_, ok := parseOpenCodeStreamLine(line)
	if ok {
		t.Error("expected ok=false for JSON without type or tool key")
	}
}

// --- prepareWorkspace additional coverage ---

func TestPrepareWorkspace_PersonaFileRead(t *testing.T) {
	tmpDir := t.TempDir()
	a := NewOpenCodeAdapter()

	// Create persona file
	personaDir := filepath.Join(".agents", "personas")
	if err := os.MkdirAll(personaDir, 0755); err != nil {
		t.Fatalf("failed to create persona dir: %v", err)
	}
	personaContent := "You are a test persona."
	if err := os.WriteFile(filepath.Join(personaDir, "test-persona.md"), []byte(personaContent), 0644); err != nil {
		t.Fatalf("failed to write persona file: %v", err)
	}
	defer os.RemoveAll(".agents/personas")

	cfg := AdapterRunConfig{
		Persona: "test-persona",
		// SystemPrompt is empty, so persona path is used
	}

	if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace returned error: %v", err)
	}

	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("expected AGENTS.md to exist: %v", err)
	}
	if string(data) != personaContent {
		t.Errorf("AGENTS.md = %q, want %q", string(data), personaContent)
	}
}

func TestPrepareWorkspace_MissingPersona_NoError(t *testing.T) {
	tmpDir := t.TempDir()
	a := NewOpenCodeAdapter()

	cfg := AdapterRunConfig{
		Persona: "nonexistent-persona",
	}

	if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace returned error: %v", err)
	}

	// AGENTS.md should NOT exist because persona file doesn't exist
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if _, err := os.Stat(agentsPath); err == nil {
		t.Error("expected AGENTS.md to NOT exist when persona file is missing")
	}
}

func TestPrepareWorkspace_NonWritableDir(t *testing.T) {
	tmpDir := t.TempDir()
	a := NewOpenCodeAdapter()

	// Make directory non-writable
	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Skipf("cannot change permissions: %v", err)
	}
	defer func() { _ = os.Chmod(tmpDir, 0755) }()

	cfg := AdapterRunConfig{
		SystemPrompt: "should fail",
	}

	err := a.prepareWorkspace(tmpDir, cfg)
	if err == nil {
		t.Error("expected error when writing to non-writable directory")
	}
}

// --- buildArgs additional coverage ---

func TestBuildArgs_WithPrompt(t *testing.T) {
	a := NewOpenCodeAdapter()
	cfg := AdapterRunConfig{
		Prompt: "fix the bug in main.go",
	}
	got := a.buildArgs(cfg)
	want := []string{"run", "--format", "json", "--", "fix the bug in main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgs() = %v, want %v", got, want)
	}
}

func TestBuildArgs_ModelAndPrompt(t *testing.T) {
	a := NewOpenCodeAdapter()
	cfg := AdapterRunConfig{
		Model:  "gpt-4o",
		Prompt: "explain this code",
	}
	got := a.buildArgs(cfg)
	want := []string{"run", "--model", "gpt-4o", "--format", "json", "--", "explain this code"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgs() = %v, want %v", got, want)
	}
}
