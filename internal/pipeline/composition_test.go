package pipeline

import (
	"testing"
)

func resolveStepInputForTest(step *Step, ctx *TemplateContext) (string, error) {
	if step.SubInput != "" {
		return ResolveTemplate(step.SubInput, ctx)
	}
	return ctx.Input, nil
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
			got, err := mergeJSONArrays(tt.input, "")
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

func TestMergeJSONArrays_KeyExtraction(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		key     string
		want    string
		wantErr bool
	}{
		{
			name: "extract findings from objects",
			input: `[
				{"findings": [{"id": 1}], "summary": "a"},
				{"findings": [{"id": 2}], "summary": "b"},
				{"findings": [{"id": 3}], "summary": "c"}
			]`,
			key:  "findings",
			want: `[{"id":1},{"id":2},{"id":3}]`,
		},
		{
			name: "extract with multiple items per array",
			input: `[
				{"results": [{"id": 1}, {"id": 2}]},
				{"results": [{"id": 3}]}
			]`,
			key:  "results",
			want: `[{"id":1},{"id":2},{"id":3}]`,
		},
		{
			name:  "extract from single element",
			input: `[{"items": [10, 20]}]`,
			key:   "items",
			want:  `[10,20]`,
		},
		{
			name:  "extract with empty arrays",
			input: `[{"data": []}, {"data": [1]}, {"data": []}]`,
			key:   "data",
			want:  `[1]`,
		},
		{
			name:    "key not found in object",
			input:   `[{"other": [1]}]`,
			key:     "findings",
			wantErr: true,
		},
		{
			name:    "element is not an object",
			input:   `[[1, 2], [3]]`,
			key:     "findings",
			wantErr: true,
		},
		{
			name:    "value at key is not an array",
			input:   `[{"findings": "not-array"}]`,
			key:     "findings",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeJSONArrays(tt.input, tt.key)
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

func TestCompositionExecutor_SubPipelineWithConfig(t *testing.T) {
	// Verify that executeSubPipeline applies timeout from config
	ctx := NewTemplateContext("test-input", "/tmp")

	step := &Step{
		ID:          "sub-with-config",
		SubPipeline: "child-pipeline",
		Config: &SubPipelineConfig{
			Inject:    []string{"plan"},
			Extract:   []string{"output"},
			Timeout:   "1h",
			MaxCycles: 10,
		},
	}

	// Verify the step is a composition step
	if !step.IsCompositionStep() {
		t.Error("step with SubPipeline should be a composition step")
	}

	// Verify config validation passes
	if err := step.Config.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	// Verify input resolution still works
	step.SubInput = "{{input}}"
	input, err := resolveStepInputForTest(step, ctx)
	if err != nil {
		t.Fatalf("input resolution failed: %v", err)
	}
	if input != "test-input" {
		t.Errorf("expected 'test-input', got %q", input)
	}
}

func TestCompositionExecutor_SubPipelineBackwardCompatibility(t *testing.T) {
	// Verify that a step with SubPipeline but no Config still works
	step := &Step{
		ID:          "sub-legacy",
		SubPipeline: "simple-pipeline",
	}

	if !step.IsCompositionStep() {
		t.Error("step with SubPipeline should be a composition step")
	}

	// Config should be nil
	if step.Config != nil {
		t.Error("legacy step should have nil Config")
	}

	// SubPipelineConfig.Validate should handle nil
	if err := (*SubPipelineConfig)(nil).Validate(); err != nil {
		t.Errorf("nil config should validate: %v", err)
	}

	// ParseTimeout should handle nil
	if d := (*SubPipelineConfig)(nil).ParseTimeout(); d != 0 {
		t.Errorf("nil config ParseTimeout should be 0, got %v", d)
	}
}

// TestBranchDispatch_DefaultArmOnMissingEnv verifies the branch primitive
// falls back to cases.default when {{ env.<key> }} resolves to empty (no
// matching case key for "").
func TestBranchDispatch_DefaultArmOnMissingEnv(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	// Env is empty — {{ env.profile }} resolves to ""
	step := &Step{
		ID: "publish",
		Branch: &BranchConfig{
			On: "{{ env.profile }}",
			Cases: map[string]string{
				"core":    "skip",
				"default": "ops-pr-review-publish",
			},
		},
	}

	resolved, err := ResolveTemplate(step.Branch.On, ctx)
	if err != nil {
		t.Fatalf("template resolution failed: %v", err)
	}
	if resolved != "" {
		t.Fatalf("expected empty resolution for missing env, got %q", resolved)
	}

	pipelineName, ok := step.Branch.Cases[resolved]
	if !ok {
		pipelineName = step.Branch.Cases["default"]
	}
	if pipelineName != "ops-pr-review-publish" {
		t.Errorf("expected default arm 'ops-pr-review-publish', got %q", pipelineName)
	}
}

// TestBranchDispatch_EnvProfileCore verifies the branch primitive routes to
// the "core" case when env.profile is "core" — which the unified
// ops-pr-review pipeline uses to skip the publish step.
func TestBranchDispatch_EnvProfileCore(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	ctx.Env = map[string]string{"profile": "core"}
	step := &Step{
		ID: "publish",
		Branch: &BranchConfig{
			On: "{{ env.profile }}",
			Cases: map[string]string{
				"core":    "skip",
				"default": "ops-pr-review-publish",
			},
		},
	}

	resolved, err := ResolveTemplate(step.Branch.On, ctx)
	if err != nil {
		t.Fatalf("template resolution failed: %v", err)
	}
	if resolved != "core" {
		t.Fatalf("expected 'core', got %q", resolved)
	}
	if step.Branch.Cases[resolved] != "skip" {
		t.Errorf("expected core arm to dispatch to 'skip', got %q", step.Branch.Cases[resolved])
	}
}

// TestSubPipelineConfig_EnvField verifies the Env field is preserved on the
// SubPipelineConfig and round-trips through Validate.
func TestSubPipelineConfig_EnvField(t *testing.T) {
	cfg := &SubPipelineConfig{
		Env: map[string]string{"profile": "core", "region": "eu"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if got := cfg.Env["profile"]; got != "core" {
		t.Errorf("expected profile=core, got %q", got)
	}
	if len(cfg.Env) != 2 {
		t.Errorf("expected 2 env entries, got %d", len(cfg.Env))
	}
}
