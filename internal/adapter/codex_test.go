package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodexAdapter_BuildArgs(t *testing.T) {
	a := NewCodexAdapter()
	base := []string{"exec", "--json", "--skip-git-repo-check", "--ephemeral", "--color", "never", "--dangerously-bypass-approvals-and-sandbox"}

	tests := []struct {
		name string
		cfg  AdapterRunConfig
		want []string
	}{
		{
			name: "prompt is read from stdin, never argv",
			cfg:  AdapterRunConfig{Prompt: "implement the feature"},
			want: append(append([]string{}, base...), "-"),
		},
		{
			name: "with model",
			cfg:  AdapterRunConfig{Prompt: "fix the bug", Model: "gpt-6.2-sol"},
			want: append(append([]string{}, base...), "--model", "gpt-6.2-sol", "-"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, a.buildArgs(tt.cfg))
		})
	}
}

// codexTranscript is a trimmed real `codex exec --json` stream (codex-cli 0.171.0).
const codexTranscript = `{"type":"thread.started","thread_id":"01a20f61"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"I'll run echo hi."}}
{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"zsh -lc 'echo hi'","aggregated_output":"","exit_code":null,"status":"in_progress"}}
{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"zsh -lc 'echo hi'","aggregated_output":"hi\n","exit_code":0,"status":"completed"}}
{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"DONE"}}
{"type":"turn.completed","usage":{"input_tokens":35144,"cached_input_tokens":31232,"output_tokens":41}}
`

func TestCodexAdapter_ParseOutput(t *testing.T) {
	a := NewCodexAdapter()

	t.Run("empty output", func(t *testing.T) {
		r := a.parseOutput("")
		assert.Zero(t, r.TokensUsed)
		assert.Empty(t, r.FailureReason)
	})

	t.Run("real transcript", func(t *testing.T) {
		r := a.parseOutput(codexTranscript)
		assert.Equal(t, 35144, r.TokensIn)
		assert.Equal(t, 41, r.TokensOut)
		assert.Equal(t, 35185, r.TokensUsed)
		assert.Equal(t, "DONE", r.ResultContent, "last agent_message is the result")
		assert.Empty(t, r.FailureReason)
	})

	t.Run("turn failed", func(t *testing.T) {
		r := a.parseOutput(`{"type":"turn.failed","error":{"message":"Rate limit reached for model"}}`)
		assert.Equal(t, FailureReasonRateLimit, r.FailureReason)
		assert.Equal(t, "Rate limit reached for model", r.ResultContent)
	})

	t.Run("non-json output", func(t *testing.T) {
		r := a.parseOutput("plain text output")
		assert.Zero(t, r.TokensUsed)
	})
}

func TestParseCodexStreamLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantOK  bool
		wantEvt StreamEvent
	}{
		{
			name:    "command started",
			line:    `{"type":"item.started","item":{"type":"command_execution","command":"go test ./..."}}`,
			wantOK:  true,
			wantEvt: StreamEvent{Type: "tool_use", ToolName: "Bash", ToolInput: "go test ./..."},
		},
		{
			name:    "agent message",
			line:    `{"type":"item.completed","item":{"type":"agent_message","text":"analyzing code"}}`,
			wantOK:  true,
			wantEvt: StreamEvent{Type: "text", Content: "analyzing code"},
		},
		{
			name:    "turn completed",
			line:    `{"type":"turn.completed","usage":{"input_tokens":100,"output_tokens":50}}`,
			wantOK:  true,
			wantEvt: StreamEvent{Type: "result", Subtype: "success", TokensIn: 100, TokensOut: 50},
		},
		{
			name:   "thread started is not surfaced",
			line:   `{"type":"thread.started","thread_id":"x"}`,
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
			assert.Equal(t, tt.wantOK, ok)
			if ok {
				assert.Equal(t, tt.wantEvt, evt)
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
