package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeminiAdapter_BuildArgs(t *testing.T) {
	a := NewGeminiAdapter()

	tests := []struct {
		name string
		cfg  AdapterRunConfig
		want []string
	}{
		{
			name: "basic prompt",
			cfg:  AdapterRunConfig{Prompt: "implement the feature"},
			want: []string{"--yolo", "--output-format", "stream-json", "-p", "implement the feature"},
		},
		{
			name: "with model",
			cfg:  AdapterRunConfig{Prompt: "fix the bug", Model: "gemini-pro"},
			want: []string{"--model", "gemini-pro", "--yolo", "--output-format", "stream-json", "-p", "fix the bug"},
		},
		{
			name: "no prompt no model",
			cfg:  AdapterRunConfig{},
			want: []string{"--yolo", "--output-format", "stream-json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := a.buildArgs(tt.cfg)
			assert.Equal(t, tt.want, args)
		})
	}
}

func TestGeminiAdapter_ParseOutput(t *testing.T) {
	a := NewGeminiAdapter()

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
			output:      `{"type":"result","usage":{"input_tokens":200,"output_tokens":80},"content":"completed"}`,
			wantIn:      200,
			wantOut:     80,
			wantContent: "completed",
		},
		{
			name:        "plain text output",
			output:      "plain text response",
			wantContent: "plain text response",
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

func TestParseGeminiStreamLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantOK  bool
		wantEvt StreamEvent
	}{
		{
			name:    "tool_use event",
			line:    `{"type":"tool_use","name":"WriteFile","input":"/tmp/bar"}`,
			wantOK:  true,
			wantEvt: StreamEvent{Type: "tool_use", ToolName: "WriteFile", ToolInput: "/tmp/bar"},
		},
		{
			name:    "text event",
			line:    `{"type":"text","content":"thinking about solution"}`,
			wantOK:  true,
			wantEvt: StreamEvent{Type: "text", Content: "thinking about solution"},
		},
		{
			name:    "result event",
			line:    `{"type":"result","usage":{"input_tokens":200,"output_tokens":80},"content":"done"}`,
			wantOK:  true,
			wantEvt: StreamEvent{Type: "result", TokensIn: 200, TokensOut: 80, Content: "done"},
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
			assert.Equal(t, tt.wantOK, ok)
			if ok {
				assert.Equal(t, tt.wantEvt.Type, evt.Type)
				assert.Equal(t, tt.wantEvt.ToolName, evt.ToolName)
			}
		})
	}
}

func TestClassifyGeminiFailure(t *testing.T) {
	assert.Equal(t, "timeout", classifyGeminiFailure(124))
	assert.Equal(t, "timeout", classifyGeminiFailure(137))
	assert.Equal(t, "general_error", classifyGeminiFailure(1))
	assert.Equal(t, "general_error", classifyGeminiFailure(255))
}

func TestGeminiAdapter_PrepareWorkspace(t *testing.T) {
	a := NewGeminiAdapter()
	tmpDir := t.TempDir()

	cfg := AdapterRunConfig{
		SystemPrompt: "You are a helpful assistant",
	}

	err := a.prepareWorkspace(tmpDir, cfg)
	assert.NoError(t, err)
}
