package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
)

// testEmitter collects events for assertions.
type testEmitter struct {
	events []event.Event
}

func (e *testEmitter) Emit(ev event.Event) {
	e.events = append(e.events, ev)
}

func (e *testEmitter) hasState(state string) bool {
	for _, ev := range e.events {
		if ev.State == state {
			return true
		}
	}
	return false
}

func TestCompositionExecutor_SubPipeline(t *testing.T) {
	// This test verifies that executeSubPipeline resolves input templates.
	// Full sub-pipeline execution requires pipeline files on disk, so we
	// test the template resolution path.
	ctx := NewTemplateContext("test-input", "/tmp")
	ctx.SetStepOutput("prior", []byte(`{"value": "hello"}`))

	step := &Step{
		ID:          "sub",
		SubPipeline: "nonexistent-pipeline",
		SubInput:    "{{prior.output.value}}",
	}

	// Resolve input
	input, err := resolveStepInputForTest(step, ctx)
	if err != nil {
		t.Fatalf("failed to resolve input: %v", err)
	}
	if input != "hello" {
		t.Errorf("expected %q, got %q", "hello", input)
	}
}

func resolveStepInputForTest(step *Step, ctx *TemplateContext) (string, error) {
	if step.SubInput != "" {
		return ResolveTemplate(step.SubInput, ctx)
	}
	return ctx.Input, nil
}

func TestCompositionExecutor_BranchDispatch(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		cases    map[string]string
		expected string
		wantErr  bool
	}{
		{
			name:     "exact match",
			value:    "high",
			cases:    map[string]string{"high": "hotfix", "low": "backlog"},
			expected: "hotfix",
		},
		{
			name:     "default fallback",
			value:    "unknown",
			cases:    map[string]string{"high": "hotfix", "default": "backlog"},
			expected: "backlog",
		},
		{
			name:    "no match no default",
			value:   "unknown",
			cases:   map[string]string{"high": "hotfix"},
			wantErr: true,
		},
		{
			name:     "skip case",
			value:    "low",
			cases:    map[string]string{"high": "hotfix", "low": "skip"},
			expected: "skip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewTemplateContext("", "/tmp")
			ctx.SetStepOutput("analyze", []byte(`{"severity": "`+tt.value+`"}`))

			step := &Step{
				ID: "branch-step",
				Branch: &BranchConfig{
					On:    "{{analyze.output.severity}}",
					Cases: tt.cases,
				},
			}

			// Resolve the branch condition
			resolved, err := ResolveTemplate(step.Branch.On, ctx)
			if err != nil {
				t.Fatalf("template resolution failed: %v", err)
			}

			pipelineName, ok := step.Branch.Cases[resolved]
			if !ok {
				pipelineName, ok = step.Branch.Cases["default"]
			}

			if tt.wantErr {
				if ok {
					t.Errorf("expected no match, got %q", pipelineName)
				}
				return
			}

			if !ok {
				t.Fatal("expected match but got none")
			}
			if pipelineName != tt.expected {
				t.Errorf("expected pipeline %q, got %q", tt.expected, pipelineName)
			}
		})
	}
}

func TestCompositionExecutor_IterateItems(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	ctx.SetStepOutput("scope", []byte(`{"child_issues": [{"url": "issue/1"}, {"url": "issue/2"}, {"url": "issue/3"}]}`))

	// Resolve the iterate.over expression
	itemsJSON, err := ResolveTemplate("{{scope.output.child_issues}}", ctx)
	if err != nil {
		t.Fatalf("failed to resolve iterate.over: %v", err)
	}

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		t.Fatalf("failed to unmarshal items: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Verify each item's URL can be extracted
	for i, item := range items {
		ctx.Item = item
		url, err := ResolveTemplate("{{item.url}}", ctx)
		if err != nil {
			t.Fatalf("item %d: failed to resolve url: %v", i, err)
		}
		expected := "issue/" + string(rune('1'+i))
		if url != expected {
			t.Errorf("item %d: expected %q, got %q", i, expected, url)
		}
	}
}

func TestCompositionExecutor_LoopTermination(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")

	// Simulate loop iterations checking a condition
	for i := 0; i < 5; i++ {
		ctx.Iteration = i

		// Simulate condition becoming true at iteration 2
		if i >= 2 {
			ctx.SetStepOutput("check", []byte(`{"status": "true"}`))
		} else {
			ctx.SetStepOutput("check", []byte(`{"status": "false"}`))
		}

		result, err := ResolveTemplate("{{check.output.status}}", ctx)
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}

		if i >= 2 && result != "true" {
			t.Errorf("iteration %d: expected true, got %q", i, result)
		}
		if i < 2 && result != "false" {
			t.Errorf("iteration %d: expected false, got %q", i, result)
		}
	}
}

func TestCompositionExecutor_Aggregate_Concat(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output", "merged.txt")

	ctx := NewTemplateContext("", "/tmp")
	ctx.SetStepOutput("step1", []byte("result-1"))
	ctx.SetStepOutput("step2", []byte("result-2"))

	step := &Step{
		ID: "agg",
		Aggregate: &AggregateConfig{
			From:     "{{step1.output}} {{step2.output}}",
			Into:     outputPath,
			Strategy: "concat",
		},
	}

	// Resolve the from template
	resolved, err := ResolveTemplate(step.Aggregate.From, ctx)
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}
	if resolved != "result-1 result-2" {
		t.Errorf("expected %q, got %q", "result-1 result-2", resolved)
	}

	// Write to file (simulating aggregate behavior)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outputPath, []byte(resolved), 0644); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "result-1 result-2" {
		t.Errorf("file content mismatch: got %q", string(data))
	}
}

func TestMergeJSONArrays(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "flat array passthrough",
			input: `[1, 2, 3]`,
			want:  `[1, 2, 3]`,
		},
		{
			name:  "array of arrays",
			input: `[[1, 2], [3, 4]]`,
			want:  `[1,2,3,4]`,
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeJSONArrays(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateCompositionTemplates(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "scope", SubPipeline: "gh-scope", SubInput: "{{input}}"},
			{ID: "implement", SubPipeline: "speckit-flow", SubInput: "{{scope.output.url}}",
				Iterate: &IterateConfig{Over: "{{scope.output.child_issues}}", Mode: "sequential"}},
			{ID: "bad-ref", SubPipeline: "test", SubInput: "{{nonexistent.output.field}}"},
		},
	}

	errors := ValidateCompositionTemplates(p)
	if len(errors) == 0 {
		t.Fatal("expected validation errors for nonexistent step reference")
	}

	// Should have exactly one error for "nonexistent"
	found := false
	for _, e := range errors {
		if containsSubstr(e, "nonexistent") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error about 'nonexistent', got: %v", errors)
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCompositionExecutor_Execute_Gate_Auto(t *testing.T) {
	emitter := &testEmitter{}
	m := &manifest.Manifest{}

	ce := NewCompositionExecutor(nil, emitter, nil, m, "test", ".wave/pipelines", false)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "gate-test"},
		Steps: []Step{
			{
				ID:   "approve",
				Gate: &GateConfig{Type: "approval", Auto: true},
			},
		},
	}

	ctx := context.Background()
	err := ce.Execute(ctx, p, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !emitter.hasState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}
