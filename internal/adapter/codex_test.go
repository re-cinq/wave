package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodexAdapter_BuildArgs(t *testing.T) {
	a := NewCodexAdapter()

	tests := []struct {
		name string
		cfg  AdapterRunConfig
		want []string
	}{
		{
			name: "basic prompt",
			cfg:  AdapterRunConfig{Prompt: "implement the feature"},
			want: []string{"--full-auto", "implement the feature"},
		},
		{
			name: "with model",
			cfg:  AdapterRunConfig{Prompt: "fix the bug", Model: "gpt-4o"},
			want: []string{"--full-auto", "--model", "gpt-4o", "fix the bug"},
		},
		{
			name: "no prompt",
			cfg:  AdapterRunConfig{},
			want: []string{"--full-auto"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := a.buildArgs(tt.cfg)
			assert.Equal(t, tt.want, args)
		})
	}
}

func TestCodexAdapter_ParseOutput(t *testing.T) {
	a := NewCodexAdapter()

	tests := []struct {
		name        string
		output      string
		wantIn      int
		wantOut     int
		wantContent string
	}{
		{
			name:   "empty output",
			output: "",
		},
		{
			name:        "result event with tokens",
			output:      `{"type":"result","usage":{"input_tokens":100,"output_tokens":50},"content":"done"}`,
			wantIn:      100,
			wantOut:     50,
			wantContent: "done",
		},
		{
			name:   "non-json output",
			output: "plain text output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.parseOutput(tt.output)
			assert.Equal(t, tt.wantIn, result.TokensIn)
			assert.Equal(t, tt.wantOut, result.TokensOut)
			if tt.wantContent != "" {
				assert.Equal(t, tt.wantContent, result.ResultContent)
			}
		})
	}
}

func TestParseCodexStreamLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantOK  bool
		wantEvt StreamEvent
	}{
		{
			name:    "function_call event",
			line:    `{"type":"function_call","name":"ReadFile","arguments":"/tmp/foo"}`,
			wantOK:  true,
			wantEvt: StreamEvent{Type: "tool_use", ToolName: "ReadFile", ToolInput: "/tmp/foo"},
		},
		{
			name:    "message event",
			line:    `{"type":"message","content":"analyzing code"}`,
			wantOK:  true,
			wantEvt: StreamEvent{Type: "text", Content: "analyzing code"},
		},
		{
			name:    "result event",
			line:    `{"type":"result","usage":{"input_tokens":100,"output_tokens":50},"content":"ok"}`,
			wantOK:  true,
			wantEvt: StreamEvent{Type: "result", TokensIn: 100, TokensOut: 50, Content: "ok"},
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
		{
			name:   "unknown type",
			line:   `{"type":"unknown"}`,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt, ok := parseCodexStreamLine([]byte(tt.line))
			assert.Equal(t, tt.wantOK, ok)
			if ok {
				assert.Equal(t, tt.wantEvt.Type, evt.Type)
				assert.Equal(t, tt.wantEvt.ToolName, evt.ToolName)
			}
		})
	}
}

func TestClassifyCodexFailure(t *testing.T) {
	assert.Equal(t, "timeout", classifyCodexFailure(124))
	assert.Equal(t, "timeout", classifyCodexFailure(137))
	assert.Equal(t, "general_error", classifyCodexFailure(1))
	assert.Equal(t, "general_error", classifyCodexFailure(255))
}

func TestCodexAdapter_PrepareWorkspace(t *testing.T) {
	a := NewCodexAdapter()
	tmpDir := t.TempDir()

	cfg := AdapterRunConfig{
		SystemPrompt: "You are a helpful assistant",
	}

	err := a.prepareWorkspace(tmpDir, cfg)
	assert.NoError(t, err)
}
