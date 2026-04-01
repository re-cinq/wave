package adapter

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func TestOpenCodeBuildArgs_ModelPassthrough(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  []string
	}{
		{
			name:  "with provider/model",
			model: "zai-coding-plan/glm-5-turbo",
			want:  []string{"run", "--model", "zai-coding-plan/glm-5-turbo", "--format", "json"},
		},
		{
			name:  "with short model name",
			model: "gpt-4o",
			want:  []string{"run", "--model", "gpt-4o", "--format", "json"},
		},
		{
			name:  "empty model no flag",
			model: "",
			want:  []string{"run", "--format", "json"},
		},
		{
			name:  "default model no flag",
			model: "default",
			want:  []string{"run", "--format", "json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewOpenCodeAdapter()
			cfg := AdapterRunConfig{Model: tt.model}
			got := a.buildArgs(cfg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenCodePrepareWorkspace_NoConfigFile(t *testing.T) {
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
	if _, err := os.Stat(configPath); err == nil {
		t.Errorf("expected config.json to NOT exist at %s (model passed via CLI args)", configPath)
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

func TestOpenCodePrepareWorkspace_NoSettingsDir(t *testing.T) {
	tmpDir := t.TempDir()
	a := NewOpenCodeAdapter()
	cfg := AdapterRunConfig{}

	if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace returned unexpected error: %v", err)
	}

	settingsDir := filepath.Join(tmpDir, ".opencode")
	if _, err := os.Stat(settingsDir); err == nil {
		t.Errorf("expected .opencode directory to NOT exist (model passed via CLI args)")
	}
}

// --- parseOpenCodeStreamLine unit tests ---

func TestParseOpenCodeStreamLine_SystemEvent(t *testing.T) {
	line := []byte(`{"type":"system","message":"initialising"}`)
	evt, ok := parseOpenCodeStreamLine(line)
	if !ok {
		t.Fatal("expected ok=true for system event")
	}
	if evt.Type != "system" {
		t.Errorf("Type = %q, want %q", evt.Type, "system")
	}
}

func TestParseOpenCodeStreamLine_TextEvent(t *testing.T) {
	line := []byte(`{"type":"text","sessionID":"s1","part":{"id":"p1","sessionID":"s1","messageID":"m1","type":"text","text":"Hello world"}}`)
	evt, ok := parseOpenCodeStreamLine(line)
	if !ok {
		t.Fatal("expected ok=true for text event")
	}
	if evt.Type != "text" {
		t.Errorf("Type = %q, want %q", evt.Type, "text")
	}
	if evt.Content != "Hello world" {
		t.Errorf("Content = %q, want %q", evt.Content, "Hello world")
	}
}

func TestParseOpenCodeStreamLine_TextTruncated(t *testing.T) {
	longText := make([]byte, 300)
	for i := range longText {
		longText[i] = 'a'
	}
	line := fmt.Appendf(nil, `{"type":"text","sessionID":"s1","part":{"id":"p1","sessionID":"s1","messageID":"m1","type":"text","text":"%s"}}`, longText)
	evt, ok := parseOpenCodeStreamLine(line)
	if !ok {
		t.Fatal("expected ok=true for long text event")
	}
	if len(evt.Content) > 200 {
		t.Errorf("Content length %d exceeds 200 chars — should be truncated", len(evt.Content))
	}
}

func TestParseOpenCodeStreamLine_EmptyText(t *testing.T) {
	line := []byte(`{"type":"text","sessionID":"s1","part":{"id":"p1","sessionID":"s1","messageID":"m1","type":"text","text":""}}`)
	_, ok := parseOpenCodeStreamLine(line)
	if ok {
		t.Error("expected ok=false for text event with empty text")
	}
}

func TestParseOpenCodeStreamLine_ToolEvent(t *testing.T) {
	line := []byte(`{"type":"tool","sessionID":"s1","part":{"id":"p1","sessionID":"s1","messageID":"m1","type":"tool","tool":"Read","input":"internal/pipeline/executor.go"}}`)
	evt, ok := parseOpenCodeStreamLine(line)
	if !ok {
		t.Fatal("expected ok=true for tool event")
	}
	if evt.Type != "tool_use" {
		t.Errorf("Type = %q, want %q", evt.Type, "tool_use")
	}
	if evt.ToolName != "Read" {
		t.Errorf("ToolName = %q, want %q", evt.ToolName, "Read")
	}
	if evt.ToolInput != "internal/pipeline/executor.go" {
		t.Errorf("ToolInput = %q, want %q", evt.ToolInput, "internal/pipeline/executor.go")
	}
}

func TestParseOpenCodeStreamLine_ToolEventMissingToolName(t *testing.T) {
	line := []byte(`{"type":"tool","sessionID":"s1","part":{"id":"p1","sessionID":"s1","messageID":"m1","type":"tool"}}`)
	_, ok := parseOpenCodeStreamLine(line)
	if ok {
		t.Error("expected ok=false for tool event without tool name")
	}
}

func TestParseOpenCodeStreamLine_ResultEvent(t *testing.T) {
	line := []byte(`{"type":"result","usage":{"input_tokens":100,"output_tokens":50},"content":"done","subtype":"success"}`)
	evt, ok := parseOpenCodeStreamLine(line)
	if !ok {
		t.Fatal("expected ok=true for result event")
	}
	if evt.Type != "result" {
		t.Errorf("Type = %q, want %q", evt.Type, "result")
	}
	if evt.TokensIn != 100 {
		t.Errorf("TokensIn = %d, want 100", evt.TokensIn)
	}
	if evt.TokensOut != 50 {
		t.Errorf("TokensOut = %d, want 50", evt.TokensOut)
	}
	if evt.Subtype != "success" {
		t.Errorf("Subtype = %q, want %q", evt.Subtype, "success")
	}
}

func TestParseOpenCodeStreamLine_UnknownTypeSkipped(t *testing.T) {
	line := []byte(`{"type":"unknown_future_type","data":"whatever"}`)
	_, ok := parseOpenCodeStreamLine(line)
	if ok {
		t.Error("expected ok=false for unknown event type")
	}
}

func TestParseOpenCodeStreamLine_MalformedJSON(t *testing.T) {
	cases := [][]byte{
		[]byte(`not json at all`),
		[]byte(`{"type":"result"`), // truncated
		[]byte(`{}`),               // empty object — no type
		[]byte(``),                 // empty line
		[]byte(`   `),              // whitespace only
	}
	for _, line := range cases {
		_, ok := parseOpenCodeStreamLine(line)
		if ok {
			t.Errorf("expected ok=false for malformed/empty line %q", line)
		}
	}
}

// --- Integration-style tests using a fake opencode binary ---

// opencodeFakeBinary writes fake helper scripts that simulate opencode's
// NDJSON output to a temporary directory, returns the path.
func writeFakeOpencode(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()

	// Write a shell script that acts as opencode.
	scriptPath := filepath.Join(dir, "opencode")
	content := "#!/bin/sh\n" + script + "\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write fake opencode script: %v", err)
	}
	return scriptPath
}

// TestOpenCodeRun_StreamEventsEmittedDuringExecution verifies that
// OnStreamEvent is called for each valid NDJSON line during execution,
// not just after the process exits.
func TestOpenCodeRun_StreamEventsEmittedDuringExecution(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	// The fake opencode emits three NDJSON lines to stdout.
	ndjson := `{"type":"system","message":"init"}
{"type":"tool","sessionID":"s1","part":{"id":"p1","sessionID":"s1","messageID":"m1","type":"tool","tool":"Read","input":"main.go"}}
{"type":"step_finish","sessionID":"s1","part":{"id":"p2","sessionID":"s1","messageID":"m2","type":"step-finish","reason":"stop","cost":0,"tokens":{"total":10022,"input":9927,"output":24,"reasoning":17,"cache":{"read":71,"write":0}}}}`

	escapedNDJSON := fmt.Sprintf(`printf '%s\n'`, ndjson)
	fakePath := writeFakeOpencode(t, escapedNDJSON)

	a := &OpenCodeAdapter{opencodePath: fakePath}

	var mu sync.Mutex
	var received []StreamEvent

	cfg := AdapterRunConfig{
		WorkspacePath: t.TempDir(),
		OnStreamEvent: func(evt StreamEvent) {
			mu.Lock()
			received = append(received, evt)
			mu.Unlock()
		},
	}

	result, err := a.Run(t.Context(), cfg)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Run returned nil result")
	}

	mu.Lock()
	got := len(received)
	mu.Unlock()

	// Expect system + tool_use + result = 3 events.
	if got != 3 {
		t.Errorf("received %d events, want 3; events: %+v", got, received)
	}

	mu.Lock()
	defer mu.Unlock()

	if received[0].Type != "system" {
		t.Errorf("event[0].Type = %q, want %q", received[0].Type, "system")
	}
	if received[1].Type != "tool_use" {
		t.Errorf("event[1].Type = %q, want %q", received[1].Type, "tool_use")
	}
	if received[2].Type != "result" {
		t.Errorf("event[2].Type = %q, want %q", received[2].Type, "result")
	}
}

// TestOpenCodeRun_MalformedLinesSkipped verifies that malformed NDJSON lines
// in the output do not crash the adapter and do not produce spurious events.
func TestOpenCodeRun_MalformedLinesSkipped(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	// Mix of valid, invalid, and empty lines.
	fakePath := writeFakeOpencode(t,
		`printf 'not json\n{"type":"system"}\ntruncated{"type":\n{"type":"result","usage":{"input_tokens":1,"output_tokens":1},"content":"ok","subtype":"success"}\n'`)

	a := &OpenCodeAdapter{opencodePath: fakePath}

	var mu sync.Mutex
	var received []StreamEvent

	cfg := AdapterRunConfig{
		WorkspacePath: t.TempDir(),
		OnStreamEvent: func(evt StreamEvent) {
			mu.Lock()
			received = append(received, evt)
			mu.Unlock()
		},
	}

	result, err := a.Run(t.Context(), cfg)
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Run returned nil result")
	}

	mu.Lock()
	defer mu.Unlock()

	// Only system and result events should fire; malformed lines are skipped.
	for _, evt := range received {
		if evt.Type == "" {
			t.Errorf("received empty-type event from malformed line: %+v", evt)
		}
	}
}

// TestOpenCodeRun_FullOutputCapturedForArtifactExtraction verifies that the
// full stdout is still available via result.Stdout after streaming, so the
// executor can extract artifacts from the complete output.
func TestOpenCodeRun_FullOutputCapturedForArtifactExtraction(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	wantContent := `{"type":"result","usage":{"input_tokens":20,"output_tokens":10},"content":"artifact-content","subtype":"success"}`
	fakePath := writeFakeOpencode(t, fmt.Sprintf(`printf '%s\n'`, wantContent))

	a := &OpenCodeAdapter{opencodePath: fakePath}

	cfg := AdapterRunConfig{
		WorkspacePath: t.TempDir(),
	}

	result, err := a.Run(t.Context(), cfg)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Run returned nil result")
	}
	if result.Stdout == nil {
		t.Fatal("result.Stdout is nil")
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(result.Stdout); err != nil {
		t.Fatalf("failed to read result.Stdout: %v", err)
	}

	got := buf.String()
	if !bytes.Contains([]byte(got), []byte(wantContent)) {
		t.Errorf("result.Stdout does not contain expected content\ngot:  %q\nwant: %q", got, wantContent)
	}
}
