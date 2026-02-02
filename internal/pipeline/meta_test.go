package pipeline

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
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
			got := extractYAMLLegacy(tt.input)
			if got != tt.expect {
				t.Errorf("extractYAMLLegacy() = %q, want %q", got, tt.expect)
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

// =============================================================================
// T097: Test for meta-pipeline depth limit enforcement
// =============================================================================

func TestMetaPipelineDepthLimitEnforcement(t *testing.T) {
	tests := []struct {
		name         string
		maxDepth     int
		currentDepth int
		wantErr      bool
		errContains  string
	}{
		{
			name:         "depth 0 with max 3 allowed",
			maxDepth:     3,
			currentDepth: 0,
			wantErr:      false,
		},
		{
			name:         "depth 1 with max 3 allowed",
			maxDepth:     3,
			currentDepth: 1,
			wantErr:      false,
		},
		{
			name:         "depth 2 with max 3 allowed",
			maxDepth:     3,
			currentDepth: 2,
			wantErr:      false,
		},
		{
			name:         "depth 3 equals max 3 blocked",
			maxDepth:     3,
			currentDepth: 3,
			wantErr:      true,
			errContains:  "depth limit reached",
		},
		{
			name:         "depth 4 exceeds max 3 blocked",
			maxDepth:     3,
			currentDepth: 4,
			wantErr:      true,
			errContains:  "depth limit reached",
		},
		{
			name:         "depth 0 with max 1 allowed",
			maxDepth:     1,
			currentDepth: 0,
			wantErr:      false,
		},
		{
			name:         "depth 1 with max 1 blocked",
			maxDepth:     1,
			currentDepth: 1,
			wantErr:      true,
			errContains:  "depth limit reached",
		},
		{
			name:         "uses default max depth when 0",
			maxDepth:     0, // Should use DefaultMaxDepth (3)
			currentDepth: 2,
			wantErr:      false,
		},
		{
			name:         "default max depth blocks at 3",
			maxDepth:     0, // Should use DefaultMaxDepth (3)
			currentDepth: 3,
			wantErr:      true,
			errContains:  "depth limit reached",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &mockMetaRunner{}
			executor := NewMetaPipelineExecutor(runner)
			executor.currentDepth = tt.currentDepth

			m := &manifest.Manifest{
				Runtime: manifest.Runtime{
					MetaPipeline: manifest.MetaConfig{
						MaxDepth: tt.maxDepth,
					},
				},
			}

			config := executor.getMetaConfig(m)
			err := executor.checkDepthLimit(config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestMetaPipelineDepthLimitErrorMessage tests that depth limit errors have helpful messages (T100)
func TestMetaPipelineDepthLimitErrorMessage(t *testing.T) {
	tests := []struct {
		name            string
		maxDepth        int
		currentDepth    int
		parentPipeline  string
		expectedParts   []string
		unexpectedParts []string
	}{
		{
			name:           "error includes current and max depth",
			maxDepth:       3,
			currentDepth:   3,
			parentPipeline: "",
			expectedParts: []string{
				"current=3",
				"max=3",
				"depth limit",
			},
		},
		{
			name:           "error includes parent pipeline context",
			maxDepth:       2,
			currentDepth:   2,
			parentPipeline: "root-pipeline",
			expectedParts: []string{
				"current=2",
				"max=2",
				"root-pipeline",
				"call stack",
			},
		},
		{
			name:           "error includes suggestion for resolution",
			maxDepth:       3,
			currentDepth:   5,
			parentPipeline: "parent:meta:1",
			expectedParts: []string{
				"increase runtime.meta_pipeline.max_depth",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &mockMetaRunner{}
			executor := NewMetaPipelineExecutor(runner,
				WithMetaDepth(tt.currentDepth),
				WithParentPipeline(tt.parentPipeline),
			)

			m := &manifest.Manifest{
				Runtime: manifest.Runtime{
					MetaPipeline: manifest.MetaConfig{
						MaxDepth: tt.maxDepth,
					},
				},
			}

			config := executor.getMetaConfig(m)
			err := executor.checkDepthLimit(config)

			if err == nil {
				t.Fatal("expected error but got nil")
			}

			errMsg := err.Error()
			for _, part := range tt.expectedParts {
				if !strings.Contains(errMsg, part) {
					t.Errorf("error message %q should contain %q", errMsg, part)
				}
			}
			for _, part := range tt.unexpectedParts {
				if strings.Contains(errMsg, part) {
					t.Errorf("error message %q should NOT contain %q", errMsg, part)
				}
			}
		})
	}
}

// TestMetaPipelineNestedDepthTracking tests depth tracking across nested executions
func TestMetaPipelineNestedDepthTracking(t *testing.T) {
	runner := &mockMetaRunner{}

	// Simulate a chain of nested meta-pipelines
	root := NewMetaPipelineExecutor(runner)
	if root.currentDepth != 0 {
		t.Errorf("root depth = %d, want 0", root.currentDepth)
	}

	child1 := root.CreateChildMetaExecutor()
	if child1.currentDepth != 1 {
		t.Errorf("child1 depth = %d, want 1", child1.currentDepth)
	}

	child2 := child1.CreateChildMetaExecutor()
	if child2.currentDepth != 2 {
		t.Errorf("child2 depth = %d, want 2", child2.currentDepth)
	}

	child3 := child2.CreateChildMetaExecutor()
	if child3.currentDepth != 3 {
		t.Errorf("child3 depth = %d, want 3", child3.currentDepth)
	}

	// child3 should fail depth check with default max of 3
	m := &manifest.Manifest{}
	config := child3.getMetaConfig(m)
	err := child3.checkDepthLimit(config)
	if err == nil {
		t.Error("child3 should fail depth check with default max of 3")
	}
}

// =============================================================================
// T098: Test for meta-pipeline validation of generated pipelines
// =============================================================================

func TestMetaPipelineValidation(t *testing.T) {
	tests := []struct {
		name        string
		pipeline    *Pipeline
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil pipeline fails validation",
			pipeline:    nil,
			wantErr:     true,
			errContains: "pipeline is nil",
		},
		{
			name: "pipeline with no steps fails",
			pipeline: &Pipeline{
				Kind:     "WavePipeline",
				Metadata: PipelineMetadata{Name: "test"},
				Steps:    []Step{},
			},
			wantErr:     true,
			errContains: "no steps",
		},
		{
			name: "first step must be navigator",
			pipeline: &Pipeline{
				Kind:     "WavePipeline",
				Metadata: PipelineMetadata{Name: "test"},
				Steps: []Step{
					{
						ID:       "impl",
						Persona:  "implementer",
						Memory:   MemoryConfig{Strategy: "fresh"},
						Handover: HandoverConfig{Contract: ContractConfig{Type: "test_suite"}},
					},
				},
			},
			wantErr:     true,
			errContains: "first step must use",
		},
		{
			name: "all steps must have handover contract",
			pipeline: &Pipeline{
				Kind:     "WavePipeline",
				Metadata: PipelineMetadata{Name: "test"},
				Steps: []Step{
					{
						ID:       "nav",
						Persona:  "navigator",
						Memory:   MemoryConfig{Strategy: "fresh"},
						Handover: HandoverConfig{Contract: ContractConfig{Type: "json_schema"}},
					},
					{
						ID:       "impl",
						Persona:  "implementer",
						Memory:   MemoryConfig{Strategy: "fresh"},
						Handover: HandoverConfig{}, // Missing contract
					},
				},
			},
			wantErr:     true,
			errContains: "missing handover.contract",
		},
		{
			name: "all steps must use fresh memory strategy",
			pipeline: &Pipeline{
				Kind:     "WavePipeline",
				Metadata: PipelineMetadata{Name: "test"},
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
						Memory:       MemoryConfig{Strategy: "persistent"}, // Wrong strategy
						Handover:     HandoverConfig{Contract: ContractConfig{Type: "test_suite"}},
					},
				},
			},
			wantErr:     true,
			errContains: "must use memory.strategy='fresh'",
		},
		{
			name: "circular dependencies detected",
			pipeline: &Pipeline{
				Kind:     "WavePipeline",
				Metadata: PipelineMetadata{Name: "test"},
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
			wantErr:     true,
			errContains: "cycle",
		},
		{
			name: "invalid pipeline kind",
			pipeline: &Pipeline{
				Kind:     "InvalidKind",
				Metadata: PipelineMetadata{Name: "test"},
				Steps: []Step{
					{
						ID:       "nav",
						Persona:  "navigator",
						Memory:   MemoryConfig{Strategy: "fresh"},
						Handover: HandoverConfig{Contract: ContractConfig{Type: "json_schema"}},
					},
				},
			},
			wantErr:     true,
			errContains: "invalid pipeline kind",
		},
		{
			name: "valid pipeline passes all checks",
			pipeline: &Pipeline{
				Kind:     "WavePipeline",
				Metadata: PipelineMetadata{Name: "valid-pipeline"},
				Steps: []Step{
					{
						ID:       "navigate",
						Persona:  "navigator",
						Memory:   MemoryConfig{Strategy: "fresh"},
						Handover: HandoverConfig{Contract: ContractConfig{Type: "json_schema"}},
					},
					{
						ID:           "implement",
						Persona:      "implementer",
						Dependencies: []string{"navigate"},
						Memory:       MemoryConfig{Strategy: "fresh"},
						Handover:     HandoverConfig{Contract: ContractConfig{Type: "test_suite"}},
					},
					{
						ID:           "review",
						Persona:      "reviewer",
						Dependencies: []string{"implement"},
						Memory:       MemoryConfig{Strategy: "fresh"},
						Handover:     HandoverConfig{Contract: ContractConfig{Type: "approval"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing dependency reference",
			pipeline: &Pipeline{
				Kind:     "WavePipeline",
				Metadata: PipelineMetadata{Name: "test"},
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
						Dependencies: []string{"nonexistent"},
						Memory:       MemoryConfig{Strategy: "fresh"},
						Handover:     HandoverConfig{Contract: ContractConfig{Type: "test_suite"}},
					},
				},
			},
			wantErr:     true,
			errContains: "nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGeneratedPipeline(tt.pipeline)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestMetaPipelineValidationBeforeExecution tests that validation happens before execution
func TestMetaPipelineValidationBeforeExecution(t *testing.T) {
	// Create a mock that returns invalid pipeline YAML
	invalidYAML := `kind: WavePipeline
metadata:
  name: invalid-pipeline
steps:
  - id: impl
    persona: implementer
    memory:
      strategy: fresh
    handover:
      contract:
        type: test_suite
`
	runner := &mockMetaRunner{
		response:   invalidYAML,
		tokensUsed: 1000,
	}

	// Create an executor that tracks if child execution was attempted
	executionAttempted := false
	mockChildExecutor := &mockPipelineExecutor{
		executeFunc: func(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string) error {
			executionAttempted = true
			return nil
		},
	}

	executor := NewMetaPipelineExecutor(runner,
		WithChildExecutor(mockChildExecutor),
	)

	m := createTestMetaManifest()

	ctx := context.Background()
	_, err := executor.Execute(ctx, "test task", m)

	// Should fail validation (first step not navigator)
	if err == nil {
		t.Error("expected validation error")
	}
	if !strings.Contains(err.Error(), "first step must use") {
		t.Errorf("expected navigator error, got: %v", err)
	}

	// Child executor should NOT have been called
	if executionAttempted {
		t.Error("child executor should not be called when validation fails")
	}
}

// =============================================================================
// T099: Test for meta-pipeline failure trace preservation
// =============================================================================

func TestMetaPipelineFailureTracePreservation(t *testing.T) {
	tests := []struct {
		name              string
		setupExecutor     func() (*MetaPipelineExecutor, *testMetaEventCollector)
		task              string
		expectError       bool
		expectTraceFields []string
	}{
		{
			name: "depth limit error includes call stack",
			setupExecutor: func() (*MetaPipelineExecutor, *testMetaEventCollector) {
				runner := &mockMetaRunner{}
				collector := newTestMetaEventCollector()
				executor := NewMetaPipelineExecutor(runner,
					WithMetaDepth(3),
					WithParentPipeline("root:meta:0:child:meta:1:grandchild:meta:2"),
					WithMetaEmitter(collector),
				)
				return executor, collector
			},
			task:        "test task",
			expectError: true,
			expectTraceFields: []string{
				"root",
				"meta",
				"depth",
			},
		},
		{
			name: "validation error includes generated pipeline info",
			setupExecutor: func() (*MetaPipelineExecutor, *testMetaEventCollector) {
				// Returns invalid pipeline (first step not navigator)
				invalidYAML := `kind: WavePipeline
metadata:
  name: invalid
steps:
  - id: impl
    persona: implementer
    memory:
      strategy: fresh
    handover:
      contract:
        type: test`
				runner := &mockMetaRunner{response: invalidYAML, tokensUsed: 500}
				collector := newTestMetaEventCollector()
				executor := NewMetaPipelineExecutor(runner, WithMetaEmitter(collector))
				return executor, collector
			},
			task:        "test task",
			expectError: true,
			expectTraceFields: []string{
				"validation",
				"first step",
			},
		},
		{
			name: "philosopher error preserves context",
			setupExecutor: func() (*MetaPipelineExecutor, *testMetaEventCollector) {
				runner := &mockMetaRunner{err: errors.New("philosopher persona unavailable")}
				collector := newTestMetaEventCollector()
				executor := NewMetaPipelineExecutor(runner,
					WithMetaDepth(1),
					WithParentPipeline("parent-pipeline"),
					WithMetaEmitter(collector),
				)
				return executor, collector
			},
			task:        "generate pipeline",
			expectError: true,
			expectTraceFields: []string{
				"philosopher",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, collector := tt.setupExecutor()
			m := createTestMetaManifest()

			ctx := context.Background()
			result, err := executor.Execute(ctx, tt.task, m)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}

				errMsg := err.Error()
				for _, field := range tt.expectTraceFields {
					if !strings.Contains(strings.ToLower(errMsg), strings.ToLower(field)) {
						t.Errorf("error message %q should contain %q", errMsg, field)
					}
				}

				// Check that failure events were emitted
				events := collector.GetEvents()
				hasMetaStarted := false
				for _, e := range events {
					if e.State == "meta_started" {
						hasMetaStarted = true
						break
					}
				}
				// Only check for started event if we got past depth check
				if executor.currentDepth < DefaultMaxDepth && !hasMetaStarted {
					t.Error("meta_started event should be emitted before failure")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Error("expected non-nil result")
				}
			}
		})
	}
}

// TestMetaPipelinePreservesGeneratedPipelineOnFailure tests T101
func TestMetaPipelinePreservesGeneratedPipelineOnFailure(t *testing.T) {
	validYAML := `kind: WavePipeline
metadata:
  name: test-pipeline
steps:
  - id: nav
    persona: navigator
    memory:
      strategy: fresh
    handover:
      contract:
        type: json_schema
  - id: impl
    persona: implementer
    dependencies: [nav]
    memory:
      strategy: fresh
    handover:
      contract:
        type: test_suite`

	tests := []struct {
		name                    string
		yaml                    string
		childExecutorErr        error
		wantGeneratedPreserved  bool
		wantResultNonNil        bool
	}{
		{
			name:                   "preserves pipeline when child executor fails",
			yaml:                   validYAML,
			childExecutorErr:       errors.New("child execution failed"),
			wantGeneratedPreserved: true,
			wantResultNonNil:       true,
		},
		{
			name:                   "preserves pipeline on success",
			yaml:                   validYAML,
			childExecutorErr:       nil,
			wantGeneratedPreserved: true,
			wantResultNonNil:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &mockMetaRunner{
				response:   tt.yaml,
				tokensUsed: 1000,
			}

			mockChildExecutor := &mockPipelineExecutor{
				executeFunc: func(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string) error {
					return tt.childExecutorErr
				},
			}

			collector := newTestMetaEventCollector()
			executor := NewMetaPipelineExecutor(runner,
				WithChildExecutor(mockChildExecutor),
				WithMetaEmitter(collector),
			)

			m := createTestMetaManifest()
			ctx := context.Background()

			result, err := executor.Execute(ctx, "test task", m)

			if tt.childExecutorErr != nil {
				if err == nil {
					t.Error("expected error from child executor")
				}
			}

			if tt.wantResultNonNil {
				if result == nil {
					t.Fatal("expected non-nil result even on failure")
				}
			}

			if tt.wantGeneratedPreserved && result != nil {
				if result.GeneratedPipeline == nil {
					t.Error("GeneratedPipeline should be preserved in result")
				} else {
					if result.GeneratedPipeline.Metadata.Name != "test-pipeline" {
						t.Errorf("GeneratedPipeline name = %q, want 'test-pipeline'",
							result.GeneratedPipeline.Metadata.Name)
					}
					if len(result.GeneratedPipeline.Steps) != 2 {
						t.Errorf("GeneratedPipeline steps = %d, want 2",
							len(result.GeneratedPipeline.Steps))
					}
				}
			}
		})
	}
}

// TestMetaPipelineFailureTraceIncludesCallStack tests call stack preservation
func TestMetaPipelineFailureTraceIncludesCallStack(t *testing.T) {
	runner := &mockMetaRunner{}

	// Build a chain of nested executors
	root := NewMetaPipelineExecutor(runner)
	root.parentPipelineID = "root-pipeline"

	child1 := root.CreateChildMetaExecutor()
	child2 := child1.CreateChildMetaExecutor()
	child3 := child2.CreateChildMetaExecutor()

	// child3 should be at depth 3, which equals default max
	m := &manifest.Manifest{}
	config := child3.getMetaConfig(m)

	err := child3.checkDepthLimit(config)
	if err == nil {
		t.Fatal("expected depth limit error")
	}

	errMsg := err.Error()

	// Should include depth info
	if !strings.Contains(errMsg, "current=3") {
		t.Errorf("error should include current depth, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "max=3") {
		t.Errorf("error should include max depth, got: %s", errMsg)
	}

	// Should include call stack
	if !strings.Contains(errMsg, "call stack") {
		t.Errorf("error should include call stack, got: %s", errMsg)
	}

	// Call stack should show the chain
	if !strings.Contains(errMsg, "root-pipeline") {
		t.Errorf("error should include root pipeline in call stack, got: %s", errMsg)
	}
}

// =============================================================================
// Helper types and functions for meta-pipeline tests
// =============================================================================

// mockPipelineExecutor is a mock implementation of PipelineExecutor for testing
type mockPipelineExecutor struct {
	executeFunc func(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string) error
}

func (m *mockPipelineExecutor) Execute(ctx context.Context, p *Pipeline, man *manifest.Manifest, input string) error {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, p, man, input)
	}
	return nil
}

func (m *mockPipelineExecutor) Resume(ctx context.Context, pipelineID string, fromStep string) error {
	return nil
}

func (m *mockPipelineExecutor) GetStatus(pipelineID string) (*PipelineStatus, error) {
	return nil, nil
}

// testMetaEventCollector collects events emitted during meta-pipeline execution
type testMetaEventCollector struct {
	events []event.Event
}

func newTestMetaEventCollector() *testMetaEventCollector {
	return &testMetaEventCollector{
		events: make([]event.Event, 0),
	}
}

func (c *testMetaEventCollector) Emit(e event.Event) {
	c.events = append(c.events, e)
}

func (c *testMetaEventCollector) GetEvents() []event.Event {
	return c.events
}

func (c *testMetaEventCollector) HasEventWithState(state string) bool {
	for _, e := range c.events {
		if e.State == state {
			return true
		}
	}
	return false
}

// createTestMetaManifest creates a manifest suitable for meta-pipeline testing
func createTestMetaManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test-project"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"philosopher": {
				Adapter:     "claude",
				Temperature: 0.7,
			},
			"navigator": {
				Adapter:     "claude",
				Temperature: 0.1,
			},
			"implementer": {
				Adapter:     "claude",
				Temperature: 0.3,
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     "/tmp/test-workspace",
			DefaultTimeoutMin: 5,
			MetaPipeline: manifest.MetaConfig{
				MaxDepth:       3,
				MaxTotalSteps:  20,
				MaxTotalTokens: 100000,
			},
		},
	}
}
