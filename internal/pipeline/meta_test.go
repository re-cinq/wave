package pipeline

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
)

// mockMetaRunner implements adapter.AdapterRunner for testing.
type mockMetaRunner struct {
	response   string
	tokensUsed int
	err        error
}

func (m *mockMetaRunner) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &adapter.AdapterResult{
		Stdout:     io.NopCloser(strings.NewReader(m.response)),
		ExitCode:   0,
		TokensUsed: m.tokensUsed,
	}, nil
}

func TestValidateGeneratedPipeline(t *testing.T) {
	tests := []struct {
		name    string
		p       *Pipeline
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil pipeline",
			p:       nil,
			wantErr: true,
			errMsg:  "pipeline is nil",
		},
		{
			name:    "empty steps",
			p:       &Pipeline{Kind: "WavePipeline", Steps: []Step{}},
			wantErr: true,
			errMsg:  "pipeline has no steps",
		},
		{
			name: "invalid kind",
			p: &Pipeline{
				Kind:  "InvalidKind",
				Steps: []Step{{ID: "test", Persona: "navigator"}},
			},
			wantErr: true,
			errMsg:  "invalid pipeline kind",
		},
		{
			name: "first step not navigator",
			p: &Pipeline{
				Kind: "WavePipeline",
				Steps: []Step{
					{
						ID:       "impl",
						Persona:  "implementer",
						Memory:   MemoryConfig{Strategy: "fresh"},
						Handover: HandoverConfig{Contract: ContractConfig{Type: "test_suite"}},
					},
				},
			},
			wantErr: true,
			errMsg:  "first step must use",
		},
		{
			name: "missing handover contract",
			p: &Pipeline{
				Kind: "WavePipeline",
				Steps: []Step{
					{
						ID:      "nav",
						Persona: "navigator",
						Memory:  MemoryConfig{Strategy: "fresh"},
						// Missing handover contract
					},
				},
			},
			wantErr: true,
			errMsg:  "missing handover.contract",
		},
		{
			name: "non-fresh memory strategy",
			p: &Pipeline{
				Kind: "WavePipeline",
				Steps: []Step{
					{
						ID:       "nav",
						Persona:  "navigator",
						Memory:   MemoryConfig{Strategy: "persistent"},
						Handover: HandoverConfig{Contract: ContractConfig{Type: "json_schema"}},
					},
				},
			},
			wantErr: true,
			errMsg:  "must use memory.strategy='fresh'",
		},
		{
			name: "valid pipeline",
			p: &Pipeline{
				Kind: "WavePipeline",
				Metadata: PipelineMetadata{
					Name: "test-pipeline",
				},
				Steps: []Step{
					{
						ID:       "nav",
						Persona:  "navigator",
						Memory:   MemoryConfig{Strategy: "fresh"},
						Handover: HandoverConfig{Contract: ContractConfig{Type: "json_schema"}},
					},
					{
						ID:           "impl",
						Persona:      "implementer",
						Dependencies: []string{"nav"},
						Memory:       MemoryConfig{Strategy: "fresh"},
						Handover:     HandoverConfig{Contract: ContractConfig{Type: "test_suite"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "circular dependency",
			p: &Pipeline{
				Kind: "WavePipeline",
				Steps: []Step{
					{
						ID:           "nav",
						Persona:      "navigator",
						Dependencies: []string{"impl"},
						Memory:       MemoryConfig{Strategy: "fresh"},
						Handover:     HandoverConfig{Contract: ContractConfig{Type: "json_schema"}},
					},
					{
						ID:           "impl",
						Persona:      "implementer",
						Dependencies: []string{"nav"},
						Memory:       MemoryConfig{Strategy: "fresh"},
						Handover:     HandoverConfig{Contract: ContractConfig{Type: "test_suite"}},
					},
				},
			},
			wantErr: true,
			errMsg:  "cycle detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGeneratedPipeline(tt.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGeneratedPipeline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message %q does not contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestValidatePipelineYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "invalid yaml syntax",
			yaml:    "not: valid: yaml:",
			wantErr: true,
			errMsg:  "invalid YAML syntax",
		},
		{
			name: "missing name",
			yaml: `
kind: WavePipeline
metadata:
  description: test
steps:
  - id: nav
    persona: navigator
    exec:
      type: prompt
`,
			wantErr: true,
			errMsg:  "metadata.name is required",
		},
		{
			name: "no steps",
			yaml: `
kind: WavePipeline
metadata:
  name: test
steps: []
`,
			wantErr: true,
			errMsg:  "at least one step is required",
		},
		{
			name: "step missing id",
			yaml: `
kind: WavePipeline
metadata:
  name: test
steps:
  - persona: navigator
    exec:
      type: prompt
`,
			wantErr: true,
			errMsg:  "missing required field: id",
		},
		{
			name: "step missing persona",
			yaml: `
kind: WavePipeline
metadata:
  name: test
steps:
  - id: nav
    exec:
      type: prompt
`,
			wantErr: true,
			errMsg:  "missing required field: persona",
		},
		{
			name: "valid pipeline yaml",
			yaml: `
kind: WavePipeline
metadata:
  name: test-pipeline
steps:
  - id: nav
    persona: navigator
    exec:
      type: prompt
      source: test
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePipelineYAML([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePipelineYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message %q does not contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestMetaPipelineExecutor_CheckDepthLimit(t *testing.T) {
	runner := &mockMetaRunner{}
	executor := NewMetaPipelineExecutor(runner)

	m := &manifest.Manifest{
		Runtime: manifest.Runtime{
			MetaPipeline: manifest.MetaConfig{
				MaxDepth: 3,
			},
		},
	}

	config := executor.getMetaConfig(m)

	// Depth 0 should be allowed
	executor.currentDepth = 0
	if err := executor.checkDepthLimit(config); err != nil {
		t.Errorf("depth 0 should be allowed: %v", err)
	}

	// Depth 2 should be allowed
	executor.currentDepth = 2
	if err := executor.checkDepthLimit(config); err != nil {
		t.Errorf("depth 2 should be allowed: %v", err)
	}

	// Depth 3 should be blocked (equal to max)
	executor.currentDepth = 3
	if err := executor.checkDepthLimit(config); err == nil {
		t.Error("depth 3 should be blocked")
	}

	// Depth 4 should be blocked
	executor.currentDepth = 4
	if err := executor.checkDepthLimit(config); err == nil {
		t.Error("depth 4 should be blocked")
	}
}

func TestMetaPipelineExecutor_CheckTokenLimit(t *testing.T) {
	runner := &mockMetaRunner{}
	executor := NewMetaPipelineExecutor(runner)

	m := &manifest.Manifest{
		Runtime: manifest.Runtime{
			MetaPipeline: manifest.MetaConfig{
				MaxTotalTokens: 100000,
			},
		},
	}

	config := executor.getMetaConfig(m)

	// Under limit
	executor.totalTokensUsed = 50000
	if err := executor.checkTokenLimit(config); err != nil {
		t.Errorf("50000 tokens should be allowed: %v", err)
	}

	// Over limit
	executor.totalTokensUsed = 150000
	if err := executor.checkTokenLimit(config); err == nil {
		t.Error("150000 tokens should be blocked")
	}
}

func TestMetaPipelineExecutor_CheckStepLimit(t *testing.T) {
	runner := &mockMetaRunner{}
	executor := NewMetaPipelineExecutor(runner)

	m := &manifest.Manifest{
		Runtime: manifest.Runtime{
			MetaPipeline: manifest.MetaConfig{
				MaxTotalSteps: 20,
			},
		},
	}

	config := executor.getMetaConfig(m)

	// Under limit
	executor.totalStepsUsed = 10
	if err := executor.checkStepLimit(config); err != nil {
		t.Errorf("10 steps should be allowed: %v", err)
	}

	// Over limit
	executor.totalStepsUsed = 25
	if err := executor.checkStepLimit(config); err == nil {
		t.Error("25 steps should be blocked")
	}
}

func TestMetaPipelineExecutor_CreateChildMetaExecutor(t *testing.T) {
	runner := &mockMetaRunner{}
	parent := NewMetaPipelineExecutor(runner)
	parent.currentDepth = 1
	parent.totalStepsUsed = 5
	parent.totalTokensUsed = 10000
	parent.parentPipelineID = "parent"

	child := parent.CreateChildMetaExecutor()

	if child.currentDepth != 2 {
		t.Errorf("child depth = %d, want 2", child.currentDepth)
	}
	if child.totalStepsUsed != 5 {
		t.Errorf("child totalStepsUsed = %d, want 5", child.totalStepsUsed)
	}
	if child.totalTokensUsed != 10000 {
		t.Errorf("child totalTokensUsed = %d, want 10000", child.totalTokensUsed)
	}
	if child.parentPipelineID != "parent:meta:1" {
		t.Errorf("child parentPipelineID = %q, want 'parent:meta:1'", child.parentPipelineID)
	}
}

func TestMetaPipelineExecutor_SyncFromChild(t *testing.T) {
	runner := &mockMetaRunner{}
	parent := NewMetaPipelineExecutor(runner)
	parent.totalStepsUsed = 5
	parent.totalTokensUsed = 10000

	child := parent.CreateChildMetaExecutor()
	child.totalStepsUsed = 15
	child.totalTokensUsed = 50000

	parent.SyncFromChild(child)

	if parent.totalStepsUsed != 15 {
		t.Errorf("parent totalStepsUsed = %d, want 15", parent.totalStepsUsed)
	}
	if parent.totalTokensUsed != 50000 {
		t.Errorf("parent totalTokensUsed = %d, want 50000", parent.totalTokensUsed)
	}
}

func TestExtractYAML(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "plain yaml",
			input:  "kind: WavePipeline\nmetadata:\n  name: test",
			expect: "kind: WavePipeline\nmetadata:\n  name: test",
		},
		{
			name:   "yaml in code block",
			input:  "Here's the pipeline:\n```yaml\nkind: WavePipeline\nmetadata:\n  name: test\n```\nDone.",
			expect: "kind: WavePipeline\nmetadata:\n  name: test",
		},
		{
			name:   "yaml in generic code block",
			input:  "```\nkind: WavePipeline\nmetadata:\n  name: test\n```",
			expect: "kind: WavePipeline\nmetadata:\n  name: test",
		},
		{
			name:   "yaml preceded by text",
			input:  "Here is the generated pipeline:\n\nkind: WavePipeline\nmetadata:\n  name: test",
			expect: "kind: WavePipeline\nmetadata:\n  name: test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractYAML(tt.input)
			if got != tt.expect {
				t.Errorf("extractYAML() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestMetaPipelineExecutor_GetTimeout(t *testing.T) {
	runner := &mockMetaRunner{}
	executor := NewMetaPipelineExecutor(runner)

	// Default timeout
	m := &manifest.Manifest{}
	timeout := executor.getTimeout(m)
	if timeout != DefaultMetaTimeout {
		t.Errorf("default timeout = %v, want %v", timeout, DefaultMetaTimeout)
	}

	// Custom timeout
	m = &manifest.Manifest{
		Runtime: manifest.Runtime{
			MetaPipeline: manifest.MetaConfig{
				TimeoutMin: 60,
			},
		},
	}
	timeout = executor.getTimeout(m)
	if timeout != 60*time.Minute {
		t.Errorf("custom timeout = %v, want %v", timeout, 60*time.Minute)
	}
}

func TestMetaPipelineExecutor_GetPipelineID(t *testing.T) {
	runner := &mockMetaRunner{}

	// Without parent
	executor := NewMetaPipelineExecutor(runner)
	executor.currentDepth = 2
	id := executor.getPipelineID()
	if id != "meta:2" {
		t.Errorf("pipeline ID = %q, want 'meta:2'", id)
	}

	// With parent
	executor.parentPipelineID = "my-parent"
	id = executor.getPipelineID()
	if id != "my-parent:meta:2" {
		t.Errorf("pipeline ID = %q, want 'my-parent:meta:2'", id)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		expect string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is a longer string", 10, "this is..."},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.expect {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expect)
		}
	}
}
